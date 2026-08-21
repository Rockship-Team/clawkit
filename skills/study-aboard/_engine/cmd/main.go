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
	// Profile
	case "student":
		cmdStudent(os.Args[2:])
	// Applications & Deadlines
	case "application":
		cmdApplication(os.Args[2:])
	case "checklist":
		cmdChecklist(os.Args[2:])
	// School Matching
	case "university":
		cmdUniversity(os.Args[2:])
	// Study Plan
	case "plan":
		cmdPlan(os.Args[2:])
	// Essay Review
	case "essay":
		cmdEssay(os.Args[2:])
	// EC Strategy
	case "ec":
		cmdEC(os.Args[2:])
	// Pre-Departure
	case "visa":
		cmdVisa(os.Args[2:])
	// Offer Comparison
	case "offer":
		cmdOffer(os.Args[2:])
	// Financial Aid
	case "financial":
		cmdFinancial(os.Args[2:])
	// Cron Jobs
	case "cron":
		cmdCron(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `sa-cli — OpenClaw Study Abroad Bot

  init                             Create SQLite database (~/.openclaw/workspace/sa-data/sa.db)
  config show|set|get              Channel credentials, LLM config

Profile:     student query|save|list|update|scorecard
Deadlines:   application add|list|dashboard|update
             checklist get|update|notes
Schools:     university list|get|search|seed|match
Financial:   financial cost-compare
Study:       plan create|list|checkin
Essay:       essay submit|save-scores|list|get
EC:          ec add|list|update-tier
Visa:        visa get|update
Offers:      offer add|update|list|decide|compare
Cron:        cron list|cancel`)
}
