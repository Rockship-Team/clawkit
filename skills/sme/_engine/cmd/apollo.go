package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// cmdApollo dispatches Apollo.io subcommands.
//
//	sme-cli apollo search-company <name>
//	sme-cli apollo search-people <company> [seniorities]
//	sme-cli apollo enrich-person <full_name> <company_or_domain>
//
// Apollo's read-only search/enrichment API requires the X-Api-Key header
// (NOT "Authorization: Bearer") and POST endpoints under /v1/mixed_*.
// The legacy proposal-agent bash scripts used the wrong header and a
// write endpoint (/companies/follow), which Apollo accepted with a 200
// empty body — a silent failure. This Go port fixes both.
func cmdApollo(args []string) {
	if len(args) == 0 {
		errOut("usage: apollo search-company|search-people|enrich-person")
		return
	}
	switch args[0] {
	case "search-company":
		apolloSearchCompany(args[1:])
	case "search-people":
		apolloSearchPeople(args[1:])
	case "enrich-person":
		apolloEnrichPerson(args[1:])
	default:
		errOut("unknown apollo command: " + args[0])
	}
}

// apolloPostData is the data-returning form of an Apollo.io call — extracted
// so sme-intelligence can read fields out of the response instead of only
// printing raw JSON. apolloPost (used by every existing CLI command below)
// wraps this with its original print-and-exit-on-error behavior, unchanged.
func apolloPostData(apiPath string, body interface{}) ([]byte, int, error) {
	c := loadConnections()
	if c.Apollo.APIKey == "" {
		return nil, 0, fmt.Errorf("apollo.api_key not set — run: sme-cli config set apollo.api_key <key>")
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to prepare the Apollo request")
	}

	req, err := http.NewRequest("POST", "https://api.apollo.io"+apiPath, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to prepare the Apollo request")
	}
	req.Header.Set("X-Api-Key", c.Apollo.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to reach Apollo — check your internet connection")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// apolloPost calls an Apollo.io endpoint using the required X-Api-Key
// header and a JSON body.
func apolloPost(apiPath string, body interface{}) {
	respBody, code, err := apolloPostData(apiPath, body)
	if err != nil {
		errOut(err.Error())
	}
	rawJSONPassthrough(respBody, code)
}

func apolloSearchCompany(args []string) {
	if len(args) == 0 {
		errOut("usage: apollo search-company <company_name>")
	}
	apolloPost("/v1/mixed_companies/search", map[string]interface{}{
		"q_organization_name": args[0],
		"page":                1,
		"per_page":            10,
	})
}

// apolloOrg is the subset of /v1/mixed_companies/search's organization
// fields that carry real, checkable evidence — used by sme-intelligence for
// account-level buying signals. Fields confirmed live during the Phase 2B
// audit; anything not present in a given response stays at its zero value
// and MUST NOT be treated as a signal (zero ≠ "no growth", it can mean
// "field absent" — callers gate on presence via the raw map, not just these
// typed fields, see intelligenceExtractSignals).
type apolloOrg struct {
	ID                                    string  `json:"id"`
	Name                                  string  `json:"name"`
	PrimaryDomain                         string  `json:"primary_domain"`
	FoundedYear                           int     `json:"founded_year"`
	OrganizationRevenue                   float64 `json:"organization_revenue"`
	OrganizationRevenuePrinted            string  `json:"organization_revenue_printed"`
	HeadcountSixMonthGrowth               float64 `json:"organization_headcount_six_month_growth"`
	HeadcountTwelveMonthGrowth            float64 `json:"organization_headcount_twelve_month_growth"`
	HeadcountTwentyFourMonthGrowth        float64 `json:"organization_headcount_twenty_four_month_growth"`
	PubliclyTradedSymbol                 string  `json:"publicly_traded_symbol"`
	PubliclyTradedExchange               string  `json:"publicly_traded_exchange"`
	HasIntentSignalAccount               bool    `json:"has_intent_signal_account"`
	IntentStrength                       *string `json:"intent_strength"`
	OwnedByOrganization                  *struct {
		Name string `json:"name"`
	} `json:"owned_by_organization"`
}

// apolloSearchCompanyData is the data-returning form of `apollo
// search-company` — reused by sme-intelligence to build account-level
// buying signals instead of only printing raw JSON to the terminal.
func apolloSearchCompanyData(name string) ([]apolloOrg, error) {
	respBody, code, err := apolloPostData("/v1/mixed_companies/search", map[string]interface{}{
		"q_organization_name": name,
		"page":                1,
		"per_page":            10,
	})
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", code, string(respBody))
	}
	var resp struct {
		Organizations []apolloOrg `json:"organizations"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp.Organizations, nil
}

func apolloSearchPeople(args []string) {
	if len(args) == 0 {
		errOut("usage: apollo search-people <company_name> [seniorities]")
	}
	seniorities := []string{"c_suite", "vp"}
	if len(args) > 1 {
		seniorities = strings.Split(args[1], ",")
		for i := range seniorities {
			seniorities[i] = strings.TrimSpace(seniorities[i])
		}
	}
	apolloPost("/v1/mixed_people/api_search", map[string]interface{}{
		"q_organization_name": args[0],
		"person_seniorities":  seniorities,
		"page":                1,
		"per_page":            10,
	})
}

// apolloPerson is the subset of /v1/mixed_people/api_search fields
// sme-intelligence uses for target-persona detection. Names are obfuscated
// by Apollo at this search tier (full identity requires apollo enrich-person
// / /v1/people/match, a separate paid step — matches the existing "Contact
// Research" delegation in GTM_ARCHITECTURE.md, not duplicated here).
type apolloPerson struct {
	FirstName          string `json:"first_name"`
	LastNameObfuscated string `json:"last_name_obfuscated"`
	Title              string `json:"title"`
}

// apolloSearchPeopleData is the data-returning form of `apollo
// search-people` — used by sme-intelligence for target_personas.
func apolloSearchPeopleData(company string, seniorities []string) ([]apolloPerson, error) {
	respBody, code, err := apolloPostData("/v1/mixed_people/api_search", map[string]interface{}{
		"q_organization_name": company,
		"person_seniorities":  seniorities,
		"page":                1,
		"per_page":            10,
	})
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", code, string(respBody))
	}
	var resp struct {
		People []apolloPerson `json:"people"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	return resp.People, nil
}

func apolloEnrichPerson(args []string) {
	if len(args) < 2 {
		errOut("usage: apollo enrich-person <full_name> <company_or_domain>")
	}
	parts := strings.SplitN(args[0], " ", 2)
	first := parts[0]
	last := ""
	if len(parts) > 1 {
		last = parts[1]
	}
	body := map[string]interface{}{
		"first_name": first,
		"last_name":  last,
	}
	// Prefer domain when it looks like one; otherwise fall back to
	// organization name. Apollo accepts either.
	if strings.Contains(args[1], ".") {
		body["domain"] = args[1]
	} else {
		body["organization_name"] = args[1]
	}
	apolloPost("/v1/people/match", body)
}
