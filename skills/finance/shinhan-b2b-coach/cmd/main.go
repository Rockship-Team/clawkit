package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "company":
		cmdCompany(os.Args[2:])
	case "txn":
		cmdTxn(os.Args[2:])
	case "cashflow":
		cmdCashflow(os.Args[2:])
	case "ar":
		cmdAR(os.Args[2:])
	case "ap":
		cmdAP(os.Args[2:])
	case "discount":
		cmdDiscount(os.Args[2:])
	case "pricing":
		cmdPricing(os.Args[2:])
	case "health":
		cmdHealth(os.Args[2:])
	case "report":
		cmdReport(os.Args[2:])
	case "tax":
		cmdTax(os.Args[2:])
	case "recommend":
		cmdRecommend(os.Args[2:])
	case "banker":
		cmdBanker(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `b2b-cli — Shinhan B2B Finance Coach data tool

Commands:
  init         Initialize database with 9 tables
  company      Company profile management
  txn          Business transaction tracking
  cashflow     Cashflow forecasting
  ar           Accounts receivable management
  ap           Accounts payable management
  discount     Discount strategy engine
  pricing      Pricing analyzer
  health       Business health metrics
  report       Financial reports
  tax          Tax estimator
  recommend    Bank product recommendations
  banker       Banker/RM feed`)
}
