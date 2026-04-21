package main

import (
	"bytes"
	"encoding/json"
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

// apolloPost calls an Apollo.io endpoint using the required X-Api-Key
// header and a JSON body.
func apolloPost(apiPath string, body interface{}) {
	c := loadConnections()
	if c.Apollo.APIKey == "" {
		errOut("apollo.api_key not set — run: sme-cli config set apollo.api_key <key>")
	}

	raw, err := json.Marshal(body)
	if err != nil {
		errOut("failed to prepare the Apollo request")
	}

	req, err := http.NewRequest("POST", "https://api.apollo.io"+apiPath, bytes.NewReader(raw))
	if err != nil {
		errOut("failed to prepare the Apollo request")
	}
	req.Header.Set("X-Api-Key", c.Apollo.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		errOut("failed to reach Apollo — check your internet connection")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	rawJSONPassthrough(respBody, resp.StatusCode)
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
