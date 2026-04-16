package main

import (
	"os"
	"strings"
	"time"
)

// LoyaltyProgram tracks a user's loyalty membership.
type LoyaltyProgram struct {
	UserID     string `json:"user_id,omitempty"`
	Program    string `json:"program"`
	Display    string `json:"display"`
	Points     int64  `json:"points"`
	Tier       string `json:"tier,omitempty"`
	Expiry     string `json:"expiry,omitempty"`
	NotifyDays int    `json:"notify_days,omitempty"`
	UpdatedAt  string `json:"updated_at"`
}

// Deal tracks an active deal/promotion.
type Deal struct {
	ID          int    `json:"id"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Category    string `json:"category"`
	DiscountPct int    `json:"discount_pct,omitempty"`
	MaxDiscount int64  `json:"max_discount,omitempty"`
	MinOrder    int64  `json:"min_order,omitempty"`
	Code        string `json:"code,omitempty"`
	URL         string `json:"url,omitempty"`
	Expiry      string `json:"expiry,omitempty"`
	Used        bool   `json:"used"`
	CreatedAt   string `json:"created_at"`
}

func loadLoyalty() []LoyaltyProgram {
	uid := currentUserID()
	all := loadAllLoyalty()
	filtered := make([]LoyaltyProgram, 0, len(all))
	for _, p := range all {
		if p.UserID == uid {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func loadAllLoyalty() []LoyaltyProgram {
	var lp []LoyaltyProgram
	readJSON(userPath("loyalty.json"), &lp)
	for i := range lp {
		if lp[i].UserID == "" {
			lp[i].UserID = "default"
		}
	}
	return lp
}

func saveLoyalty(lp []LoyaltyProgram) error {
	uid := currentUserID()
	all := loadAllLoyalty()
	merged := make([]LoyaltyProgram, 0, len(all)+len(lp))
	for _, p := range all {
		if p.UserID != uid {
			merged = append(merged, p)
		}
	}
	for i := range lp {
		if lp[i].UserID == "" {
			lp[i].UserID = uid
		}
		merged = append(merged, lp[i])
	}
	return writeJSON(userPath("loyalty.json"), merged)
}

func loadDeals() []Deal {
	var seeded []Deal
	readJSON(dataPath("deals.json"), &seeded)
	return seeded
}

func cmdLoyalty(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: loyalty add|list|update|expiring")
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		// loyalty add <program> <display> <points> [expiry]
		if len(args) < 4 {
			errOut("usage: loyalty add <program> <display> <points> [expiry]")
			os.Exit(1)
		}
		points, err := parseAmount(args[3])
		if err != nil {
			errOut("invalid points: " + args[3])
			os.Exit(1)
		}
		expiry := ""
		if len(args) > 4 {
			expiry = args[4]
		}
		lp := loadLoyalty()
		// Update if exists
		found := false
		for i := range lp {
			if lp[i].Program == args[1] {
				lp[i].Display = args[2]
				lp[i].Points = points
				lp[i].Expiry = expiry
				lp[i].UpdatedAt = vnNow().Format("2006-01-02T15:04:05-07:00")
				found = true
				break
			}
		}
		if !found {
			lp = append(lp, LoyaltyProgram{
				UserID:    currentUserID(),
				Program:   args[1],
				Display:   args[2],
				Points:    points,
				Expiry:    expiry,
				UpdatedAt: vnNow().Format("2006-01-02T15:04:05-07:00"),
			})
		}
		if err := saveLoyalty(lp); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"program": args[1], "points": points, "updated": found})

	case "list":
		lp := loadLoyalty()
		okOut(map[string]interface{}{"programs": lp, "count": len(lp)})

	case "update":
		// loyalty update <program> <points>
		if len(args) < 3 {
			errOut("usage: loyalty update <program> <points>")
			os.Exit(1)
		}
		points, err := parseAmount(args[2])
		if err != nil {
			errOut("invalid points: " + args[2])
			os.Exit(1)
		}
		lp := loadLoyalty()
		found := false
		for i := range lp {
			if lp[i].Program == args[1] {
				lp[i].Points = points
				lp[i].UpdatedAt = vnNow().Format("2006-01-02T15:04:05-07:00")
				found = true
				break
			}
		}
		if !found {
			errOut("program not found: " + args[1])
			os.Exit(1)
		}
		if err := saveLoyalty(lp); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"program": args[1], "points": points})

	case "expiring":
		lp := loadLoyalty()
		today := vnToday()
		var expiring []LoyaltyProgram
		for _, p := range lp {
			if p.Expiry == "" {
				continue
			}
			days := p.NotifyDays
			if days <= 0 {
				days = 30
			}
			if p.Expiry <= today {
				expiring = append(expiring, p)
			} else if p.Expiry <= addDays(today, days) {
				expiring = append(expiring, p)
			}
		}
		okOut(map[string]interface{}{"expiring": expiring, "count": len(expiring)})

	case "remove":
		if len(args) < 2 {
			errOut("usage: loyalty remove <program>")
			os.Exit(1)
		}
		lp := loadLoyalty()
		found := false
		var updated []LoyaltyProgram
		for _, p := range lp {
			if p.Program == args[1] {
				found = true
				continue
			}
			updated = append(updated, p)
		}
		if !found {
			errOut("program not found: " + args[1])
			os.Exit(1)
		}
		if err := saveLoyalty(updated); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"removed": args[1]})

	default:
		errOut("unknown loyalty command: " + args[0])
		os.Exit(1)
	}
}

func cmdDeals(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: deals list|match")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		category := ""
		if len(args) > 1 {
			category = args[1]
		}
		deals := loadDeals()
		today := vnToday()
		// Filter active (not expired, not used)
		active := make([]Deal, 0, len(deals))
		for _, d := range deals {
			if d.Used {
				continue
			}
			if d.Expiry != "" && d.Expiry < today {
				continue
			}
			if category != "" && d.Category != category {
				continue
			}
			active = append(active, d)
		}
		okOut(map[string]interface{}{"deals": active, "count": len(active)})

	case "match":
		deals := loadDeals()
		profile := loadProfile()
		today := vnToday()

		// Build preferred categories set from profile
		prefCats := map[string]bool{}
		if profile.DealCategories != "" {
			for _, c := range strings.Split(profile.DealCategories, ",") {
				prefCats[strings.TrimSpace(c)] = true
			}
		}

		// Build credit card set
		cardSet := map[string]bool{}
		if profile.CreditCards != "" {
			for _, c := range strings.Split(profile.CreditCards, ",") {
				cardSet[strings.TrimSpace(strings.ToLower(c))] = true
			}
		}

		type scoredDeal struct {
			Deal  Deal `json:"deal"`
			Score int  `json:"score"`
		}
		var scored []scoredDeal
		for _, d := range deals {
			if d.Used || (d.Expiry != "" && d.Expiry < today) {
				continue
			}
			s := 1 // base score for active deal
			// Bonus for matching preferred category
			if len(prefCats) > 0 && prefCats[d.Category] {
				s += 3
			}
			// Bonus for matching credit card source
			src := strings.ToLower(d.Source)
			for card := range cardSet {
				if strings.Contains(src, card) {
					s += 2
					break
				}
			}
			scored = append(scored, scoredDeal{Deal: d, Score: s})
		}
		// Sort by score descending
		for i := 0; i < len(scored); i++ {
			for j := i + 1; j < len(scored); j++ {
				if scored[j].Score > scored[i].Score {
					scored[i], scored[j] = scored[j], scored[i]
				}
			}
		}
		if len(scored) > 10 {
			scored = scored[:10]
		}
		matched := make([]Deal, 0, len(scored))
		for _, sd := range scored {
			matched = append(matched, sd.Deal)
		}
		okOut(map[string]interface{}{"matched": matched, "count": len(matched), "profile_cards": profile.CreditCards, "deal_categories": profile.DealCategories})

	default:
		errOut("unknown deals command: " + args[0])
		os.Exit(1)
	}
}

// addDays adds n days to a YYYY-MM-DD date string and returns YYYY-MM-DD.
func addDays(dateStr string, n int) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}
