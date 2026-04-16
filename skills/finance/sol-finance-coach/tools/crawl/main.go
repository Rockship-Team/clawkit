// crawl — enrich sol-finance-coach data by crawling Vietnamese financial sites.
//
// Usage:
//
//	crawl cards                     Credit card data from comparison sites
//	crawl rates                     Deposit/savings interest rates
//	crawl deals [--bank <name>]     Promotions from bank websites
//	crawl loyalty                   Loyalty program earning/redemption info
//	crawl all                       Run all crawlers and merge into data/
//
// Output goes to stdout as JSON. Redirect to update data files:
//
//	crawl cards > data/credit-cards.json
//	crawl rates > data/interest-rates.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "cards":
		runCards()
	case "rates":
		runRates()
	case "deals":
		bank := ""
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--bank" && i+1 < len(os.Args) {
				bank = os.Args[i+1]
				i++
			}
		}
		runDeals(bank)
	case "loyalty":
		runLoyalty()
	case "ecommerce":
		runEcommerce()
	case "investment":
		runInvestment()
	case "all":
		runAll()
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `crawl — enrich sol-finance-coach data from Vietnamese financial sites

Usage:
  crawl cards                     Credit card data from comparison sites
  crawl rates                     Deposit/savings interest rates
  crawl deals [--bank <name>]     Promotions from bank websites
  crawl loyalty                   Loyalty program earning/redemption info
  crawl ecommerce                 E-commerce sale deals (Shopee, Lazada, Tiki)
  crawl investment                Fund NAV + gold prices
  crawl all                       Run all crawlers, write to data/

Sources defined in sources.json (single source of truth).

Output: JSON to stdout. Pipe to data files:
  crawl cards > ../../data/credit-cards.json
  crawl rates > ../../data/interest-rates.json`)
}

func jsonPrint(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func fatal(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	}
	os.Exit(1)
}

func info(msg string) {
	fmt.Fprintf(os.Stderr, "[info] %s\n", msg)
}
