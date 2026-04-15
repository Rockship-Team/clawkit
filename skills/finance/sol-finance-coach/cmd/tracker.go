package main

import (
	"os"
	"strings"
	"time"
)

// LoyaltyProgram tracks a user's loyalty membership.
type LoyaltyProgram struct {
	Program   string `json:"program"`
	Display   string `json:"display"`
	Points    int64  `json:"points"`
	Expiry    string `json:"expiry,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

// Deal tracks an active deal/promotion.
type Deal struct {
	ID          int    `json:"id"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Expiry      string `json:"expiry,omitempty"`
	Used        bool   `json:"used"`
	CreatedAt   string `json:"created_at"`
}

func loadLoyalty() []LoyaltyProgram {
	var lp []LoyaltyProgram
	readJSON(userPath("loyalty.json"), &lp)
	return lp
}

func saveLoyalty(lp []LoyaltyProgram) error {
	return writeJSON(userPath("loyalty.json"), lp)
}

func loadDeals() []Deal {
	var deals []Deal
	readJSON(userPath("user_deals.json"), &deals)
	return deals
}

func saveDeals(deals []Deal) error {
	return writeJSON(userPath("user_deals.json"), deals)
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
		// Find programs expiring within 30 days
		var expiring []LoyaltyProgram
		for _, p := range lp {
			if p.Expiry == "" {
				continue
			}
			if p.Expiry <= today {
				// Already expired but still tracked
				expiring = append(expiring, p)
			} else if p.Expiry <= addDays(today, 30) {
				expiring = append(expiring, p)
			}
		}
		okOut(map[string]interface{}{"expiring": expiring, "count": len(expiring)})

	default:
		errOut("unknown loyalty command: " + args[0])
		os.Exit(1)
	}
}

func cmdDeals(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: deals add|list|match")
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		// deals add <source> <description> <category> [expiry]
		if len(args) < 4 {
			errOut("usage: deals add <source> <description> <category> [expiry]")
			os.Exit(1)
		}
		expiry := ""
		if len(args) > 4 {
			expiry = args[4]
		}
		deals := loadDeals()
		maxID := 0
		for _, d := range deals {
			if d.ID > maxID {
				maxID = d.ID
			}
		}
		deal := Deal{
			ID:          maxID + 1,
			Source:      args[1],
			Description: args[2],
			Category:    args[3],
			Expiry:      expiry,
			Used:        false,
			CreatedAt:   vnNow().Format("2006-01-02T15:04:05-07:00"),
		}
		deals = append(deals, deal)
		if err := saveDeals(deals); err != nil {
			errOut("failed to save: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"deal": deal})

	case "list":
		category := ""
		if len(args) > 1 {
			category = args[1]
		}
		deals := loadDeals()
		today := vnToday()
		// Filter active (not expired, not used)
		var active []Deal
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

		var matched []Deal
		for _, d := range deals {
			if d.Used || (d.Expiry != "" && d.Expiry < today) {
				continue
			}
			// Match by credit card source
			if profile.CreditCards != "" && strings.Contains(strings.ToLower(d.Source), strings.ToLower(profile.CreditCards)) {
				matched = append(matched, d)
				continue
			}
			// Include all active deals
			matched = append(matched, d)
		}
		if len(matched) > 10 {
			matched = matched[:10]
		}
		okOut(map[string]interface{}{"matched": matched, "count": len(matched), "profile_cards": profile.CreditCards})

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
