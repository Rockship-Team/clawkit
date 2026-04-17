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
		errOut("not implemented")
	case "ar":
		errOut("not implemented")
	case "ap":
		errOut("not implemented")
	case "discount":
		errOut("not implemented")
	case "pricing":
		errOut("not implemented")
	case "health":
		errOut("not implemented")
	case "report":
		errOut("not implemented")
	case "tax":
		errOut("not implemented")
	case "recommend":
		errOut("not implemented")
	case "banker":
		errOut("not implemented")
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
