package main

import (
	"os"
)

func cmdDigest(args []string) {
	if len(args) == 0 {
		errOut("usage: digest generate")
		os.Exit(1)
	}

	if args[0] != "generate" {
		errOut("unknown digest command: " + args[0])
		os.Exit(1)
	}

	// Assemble daily digest from multiple sources
	date := vnToday()
	digest := map[string]interface{}{
		"date": date,
	}

	// 1. Daily tip
	tips := loadTips()
	if len(tips) > 0 {
		idx := deterministicIndex(date, len(tips))
		digest["tip"] = tips[idx]
	}

	// 2. Active deals (top 3)
	deals := loadDeals()
	activeDeals := make([]Deal, 0, 3)
	for _, d := range deals {
		if d.Used {
			continue
		}
		if d.Expiry != "" && d.Expiry < date {
			continue
		}
		activeDeals = append(activeDeals, d)
		if len(activeDeals) >= 3 {
			break
		}
	}
	digest["deals"] = activeDeals

	// 3. Expiring loyalty points
	lp := loadLoyalty()
	expiring := make([]LoyaltyProgram, 0)
	thirtyDays := addDays(date, 30)
	for _, p := range lp {
		if p.Expiry != "" && p.Expiry <= thirtyDays && p.Expiry >= date {
			expiring = append(expiring, p)
		}
	}
	digest["expiring_loyalty"] = expiring

	// 4. Spending summary for current month
	txs := loadTransactions()
	monthTotal := int64(0)
	monthPrefix := date[:7]
	for _, t := range txs {
		if len(t.Date) >= 7 && t.Date[:7] == monthPrefix {
			monthTotal += t.Amount
		}
	}
	digest["month_spending"] = monthTotal

	// 5. User profile for personalization
	profile := loadProfile()
	digest["knowledge_level"] = profile.KnowledgeLevel

	// 6. Budget status
	if profile.MonthlyBudget > 0 {
		remaining := profile.MonthlyBudget - monthTotal
		pctUsed := int(monthTotal * 100 / profile.MonthlyBudget)
		digest["budget"] = map[string]interface{}{
			"monthly_budget": profile.MonthlyBudget,
			"spent":          monthTotal,
			"remaining":      remaining,
			"pct_used":       pctUsed,
		}
	}

	// 7. Knowledge micro-lesson
	var kb []KnowledgeChunk
	if readJSON(dataPath("knowledge-base.json"), &kb) && len(kb) > 0 {
		idx := deterministicIndex(date+"kb", len(kb))
		digest["micro_lesson"] = kb[idx]
	}

	okOut(digest)
}
