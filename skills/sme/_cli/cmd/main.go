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
	case "config":
		cmdConfig(os.Args[2:])
	// Accounting
	case "invoice":
		cmdInvoice(os.Args[2:])
	case "payment":
		cmdPayment(os.Args[2:])
	case "bank":
		cmdBank(os.Args[2:])
	case "cashflow":
		cmdCashflow(os.Args[2:])
	case "expense":
		cmdExpense(os.Args[2:])
	// Tax
	case "tax":
		cmdTax(os.Args[2:])
	// Legal
	case "legal":
		cmdLegal(os.Args[2:])
	// HR
	case "employee":
		cmdEmployee(os.Args[2:])
	case "payroll":
		cmdPayroll(os.Args[2:])
	case "leave":
		cmdLeave(os.Args[2:])
	// Sales
	case "contact":
		cmdContact(os.Args[2:])
	case "lead":
		cmdLead(os.Args[2:])
	case "quote":
		cmdQuote(os.Args[2:])
	case "order":
		cmdOrder(os.Args[2:])
	// Ops
	case "task":
		cmdTask(os.Args[2:])
	case "document":
		cmdDocument(os.Args[2:])
	case "license":
		cmdLicense(os.Args[2:])
	// BI
	case "dashboard":
		cmdDashboard(os.Args[2:])
	case "report":
		cmdReport(os.Args[2:])
	// BD / Proposals (COSMO CRM + Apollo + local PDF rendering)
	case "cosmo":
		cmdCosmo(os.Args[2:])
	case "apollo":
		cmdApollo(os.Args[2:])
	case "proposal":
		cmdProposal(os.Args[2:])
	case "channel":
		cmdChannel(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `sme-cli — OpenClaw SME Vietnam

  init                             Create SQLite database
  config show|set|get              Remote connections (MISA, LLM)

Accounting:  invoice, payment, bank, cashflow, expense
Tax:         tax calendar|pit|vat|deadlines
Legal:       legal licenses|add-license
HR:          employee, payroll, leave
Sales:       contact, lead, quote, order
Ops:         task, document, license
BI:          dashboard, report
BD:          cosmo, apollo, proposal
Channel:     channel send-file|send-message`)
}
