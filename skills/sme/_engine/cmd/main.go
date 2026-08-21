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
	// BD / Proposals (COSMO CRM + Apollo + local PDF rendering)
	case "cosmo":
		cmdCosmo(os.Args[2:])
	case "apollo":
		cmdApollo(os.Args[2:])
	case "proposal":
		cmdProposal(os.Args[2:])
	case "event":
		cmdEvent(os.Args[2:])
	case "channel":
		cmdChannel(os.Args[2:])
	// Social
	case "social":
		cmdSocial(os.Args[2:])
	// KPI
	case "kpi":
		cmdKPI(os.Args[2:])
	// Self-improvement
	case "cron-health":
		cmdCronHealth(os.Args[2:])
	case "action-log":
		cmdActionLog(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `sme-cli — OpenClaw SME Vietnam

  init                             Create SQLite database
  config show|set|get              Remote connections (COSMO, Apollo, LLM)

BD:          cosmo, apollo, proposal, event
Channel:     channel send-file|send-message
Social:      social buckets|voice|formats|next-slot|draft|update|get|list|schedule|mark-posted|upcoming|delete
KPI:         kpi set|get|list|check
Self-improve: cron-health log|check|autofix|list
Measure:      action-log suggest|done|skip|rate|pending|auto-check`)
}
