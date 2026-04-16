package main

import (
	"os"
	"strings"
)

// InterestRate is a crawled bank interest rate entry.
type InterestRate struct {
	Bank      string  `json:"bank"`
	Term      string  `json:"term"`
	RatePct   float64 `json:"rate_pct"`
	Type      string  `json:"type,omitempty"`
	SourceURL string  `json:"source_url,omitempty"`
}

// InvestmentData is a crawled fund NAV or gold price entry.
type InvestmentData struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	ChangePct float64 `json:"change_pct,omitempty"`
	Date      string  `json:"date,omitempty"`
	SourceURL string  `json:"source_url,omitempty"`
}

func cmdRates(args []string) {
	if len(args) == 0 {
		errOut("usage: rates list [bank] [term]")
		os.Exit(1)
	}

	if args[0] != "list" {
		errOut("unknown rates command: " + args[0])
		os.Exit(1)
	}

	var rates []InterestRate
	if !readJSON(dataPath("interest-rates.json"), &rates) {
		errOut("no interest rate data found. Run data-refresh crawl first.")
		os.Exit(1)
	}

	bank := ""
	term := ""
	if len(args) > 1 {
		bank = strings.ToLower(args[1])
	}
	if len(args) > 2 {
		term = strings.ToLower(args[2])
	}

	var filtered []InterestRate
	for _, r := range rates {
		if bank != "" && !strings.Contains(strings.ToLower(r.Bank), bank) {
			continue
		}
		if term != "" && !strings.Contains(strings.ToLower(r.Term), term) {
			continue
		}
		filtered = append(filtered, r)
	}

	// Sort by rate descending
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].RatePct > filtered[i].RatePct {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	if len(filtered) > 20 {
		filtered = filtered[:20]
	}

	okOut(map[string]interface{}{"rates": filtered, "count": len(filtered)})
}

func cmdInvest(args []string) {
	if len(args) == 0 {
		errOut("usage: invest list [type]")
		os.Exit(1)
	}

	if args[0] != "list" {
		errOut("unknown invest command: " + args[0])
		os.Exit(1)
	}

	var data []InvestmentData
	if !readJSON(dataPath("investment-data.json"), &data) {
		errOut("no investment data found. Run data-refresh crawl first.")
		os.Exit(1)
	}

	itype := ""
	if len(args) > 1 {
		itype = strings.ToLower(args[1])
	}

	var filtered []InvestmentData
	for _, d := range data {
		if itype != "" && strings.ToLower(d.Type) != itype {
			continue
		}
		filtered = append(filtered, d)
	}

	okOut(map[string]interface{}{"data": filtered, "count": len(filtered)})
}
