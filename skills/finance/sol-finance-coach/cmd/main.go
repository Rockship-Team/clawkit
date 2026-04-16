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
	case "onboard":
		cmdOnboard(os.Args[2:])
	case "profile":
		cmdProfile(os.Args[2:])
	case "spend":
		cmdSpend(os.Args[2:])
	case "tips":
		cmdTips(os.Args[2:])
	case "cards":
		cmdCards(os.Args[2:])
	case "loyalty":
		cmdLoyalty(os.Args[2:])
	case "deals":
		cmdDeals(os.Args[2:])
	case "challenge":
		cmdChallenge(os.Args[2:])
	case "quiz":
		cmdQuiz(os.Args[2:])
	case "simulate":
		cmdSimulate(os.Args[2:])
	case "feedback":
		cmdFeedback(os.Args[2:])
	case "digest":
		cmdDigest(os.Args[2:])
	case "knowledge":
		cmdKnowledge(os.Args[2:])
	case "rates":
		cmdRates(os.Args[2:])
	case "invest":
		cmdInvest(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `sol-cli — SOL Finance Coach data tool

Commands:
  init                                  Initialize data directory
  onboard   status|complete             Onboarding flow
  profile   set|get|delete              User profile
  spend     add|report|last|undo|budget|compare  Spending tracker
  tips      random|daily|seasonal       Savings tips
  cards     list|recommend|compare      Credit card optimizer
  loyalty   add|list|update|expiring|remove  Loyalty tracker
  deals     add|list|match              Deal hunter
  challenge list|start|checkin|status|abandon  Gamification
  quiz      random|answer|stats         Financial quiz
  simulate  compound|loan|goal          Investment simulator
  feedback  rate|suggest|referral|stats  User feedback
  digest    generate                    Daily digest
  knowledge search|list|get             Financial knowledge base
  rates     list [bank] [term]          Interest rates lookup
  invest    list [type]                 Investment data lookup`)
}
