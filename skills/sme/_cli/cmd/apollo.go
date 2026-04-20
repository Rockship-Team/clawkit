package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// cmdApollo dispatches Apollo.io subcommands.
//
//	sme-cli apollo search-company <name>
//	sme-cli apollo search-people <company> [seniorities]
//	sme-cli apollo enrich-person <name> <company>
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

func apolloGet(apiPath string) {
	c := loadConnections()
	if c.Apollo.APIKey == "" {
		errOut("apollo.api_key not set — run: sme-cli config set apollo.api_key <key>")
	}

	req, err := http.NewRequest("GET", "https://api.apollo.io"+apiPath, nil)
	if err != nil {
		errOut("failed to prepare the Apollo request")
	}
	req.Header.Set("Authorization", "Bearer "+c.Apollo.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		errOut("failed to reach Apollo — check your internet connection")
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	rawJSONPassthrough(raw, resp.StatusCode)
}

func apolloSearchCompany(args []string) {
	if len(args) == 0 {
		errOut("usage: apollo search-company <company_name>")
	}
	q := url.QueryEscape(args[0])
	apolloGet("/v1/companies/follow?q=" + q)
}

func apolloSearchPeople(args []string) {
	if len(args) == 0 {
		errOut("usage: apollo search-people <company_name> [seniorities]")
	}
	seniorities := "c_suite,vp"
	if len(args) > 1 {
		seniorities = args[1]
	}
	q := url.QueryEscape(args[0])
	s := url.QueryEscape(seniorities)
	apolloGet("/v1/people/search?q=" + q + "&seniorities=" + s)
}

func apolloEnrichPerson(args []string) {
	if len(args) < 2 {
		errOut("usage: apollo enrich-person <name> <company_or_domain>")
	}
	parts := strings.SplitN(args[0], " ", 2)
	first := parts[0]
	last := ""
	if len(parts) > 1 {
		last = parts[1]
	}
	path := fmt.Sprintf("/v1/people/enrich?first_name=%s&last_name=%s&company_domain=%s",
		url.QueryEscape(first), url.QueryEscape(last), url.QueryEscape(args[1]))
	apolloGet(path)
}
