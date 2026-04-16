package main

import (
	"os"
)

// FeedbackEntry stores a user feedback rating.
type FeedbackEntry struct {
	Score   int    `json:"score"`
	Comment string `json:"comment,omitempty"`
	Type    string `json:"type,omitempty"`
	At      string `json:"at"`
}

func loadFeedback() []FeedbackEntry {
	var fb []FeedbackEntry
	readJSON(userPath("feedback.json"), &fb)
	return fb
}

func saveFeedback(fb []FeedbackEntry) error {
	return writeJSON(userPath("feedback.json"), fb)
}

func cmdFeedback(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: feedback rate|stats")
		os.Exit(1)
	}

	switch args[0] {
	case "rate":
		if len(args) < 2 {
			errOut("usage: feedback rate <score> [comment]")
			os.Exit(1)
		}
		score := atoi(args[1])
		if score < 1 || score > 5 {
			errOut("score must be 1-5")
			os.Exit(1)
		}
		comment := ""
		if len(args) > 2 {
			comment = args[2]
		}
		fb := loadFeedback()
		fb = append(fb, FeedbackEntry{
			Score:   score,
			Comment: comment,
			Type:    "rating",
			At:      vnNow().Format("2006-01-02T15:04:05-07:00"),
		})
		if err := saveFeedback(fb); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"score": score, "comment": comment})

	case "suggest":
		if len(args) < 2 {
			errOut("usage: feedback suggest <text>")
			os.Exit(1)
		}
		fb := loadFeedback()
		fb = append(fb, FeedbackEntry{
			Comment: args[1],
			Type:    "feature_request",
			At:      vnNow().Format("2006-01-02T15:04:05-07:00"),
		})
		if err := saveFeedback(fb); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"suggestion": args[1]})

	case "referral":
		if len(args) < 2 {
			errOut("usage: feedback referral <code>")
			os.Exit(1)
		}
		fb := loadFeedback()
		fb = append(fb, FeedbackEntry{
			Comment: args[1],
			Type:    "referral",
			At:      vnNow().Format("2006-01-02T15:04:05-07:00"),
		})
		if err := saveFeedback(fb); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		// Generate referral code for user if not set
		p := loadProfile()
		if p.ReferralCode == "" {
			p.ReferralCode = "SOL-" + args[1]
			saveProfile(p)
		}
		okOut(map[string]interface{}{"referral_code": args[1], "user_code": p.ReferralCode})

	case "stats":
		fb := loadFeedback()
		total := len(fb)
		sum := 0
		promoters := 0  // 4-5
		detractors := 0 // 1-2
		for _, f := range fb {
			sum += f.Score
			if f.Score >= 4 {
				promoters++
			} else if f.Score <= 2 {
				detractors++
			}
		}
		avg := 0.0
		if total > 0 {
			avg = float64(sum) / float64(total)
		}
		okOut(map[string]interface{}{
			"total":      total,
			"average":    avg,
			"promoters":  promoters,
			"detractors": detractors,
		})

	default:
		errOut("unknown feedback command: " + args[0])
		os.Exit(1)
	}
}

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
	var activeDeals []Deal
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
	var expiring []LoyaltyProgram
	thirtyDays := addDays(date, 30)
	for _, p := range lp {
		if p.Expiry != "" && p.Expiry <= thirtyDays && p.Expiry >= date {
			expiring = append(expiring, p)
		}
	}
	digest["expiring_loyalty"] = expiring

	// 4. Challenge status
	cs := loadChallengeState()
	if cs.ActiveID != "" {
		digest["active_challenge"] = map[string]interface{}{
			"id":     cs.ActiveID,
			"streak": cs.Streak,
		}
	}

	// 5. Spending summary for current month
	var txs []Transaction
	readJSON(userPath("transactions.json"), &txs)
	monthTotal := int64(0)
	monthPrefix := date[:7]
	for _, t := range txs {
		if len(t.Date) >= 7 && t.Date[:7] == monthPrefix {
			monthTotal += t.Amount
		}
	}
	digest["month_spending"] = monthTotal

	// 6. User profile for personalization
	profile := loadProfile()
	digest["knowledge_level"] = profile.KnowledgeLevel

	// 7. Budget status
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

	// 8. Knowledge micro-lesson
	var kb []KnowledgeChunk
	if readJSON(dataPath("knowledge-base.json"), &kb) && len(kb) > 0 {
		idx := deterministicIndex(date+"kb", len(kb))
		digest["micro_lesson"] = kb[idx]
	}

	okOut(digest)
}
