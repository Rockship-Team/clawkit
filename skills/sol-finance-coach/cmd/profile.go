package main

import (
	"os"
)

// Profile is the user's financial profile.
type Profile struct {
	Name           string `json:"name,omitempty"`
	Income         int64  `json:"income,omitempty"`
	MonthlyFixed   int64  `json:"monthly_fixed,omitempty"`
	Goal           string `json:"goal,omitempty"`
	RiskLevel      string `json:"risk_level,omitempty"`
	CreditCards    string `json:"credit_cards,omitempty"`
	KnowledgeLevel string `json:"knowledge_level,omitempty"`
	DailyTips      bool   `json:"daily_tips"`
	Onboarded      bool   `json:"onboarded"`
	CreatedAt      string `json:"created_at,omitempty"`
}

func loadProfile() Profile {
	var p Profile
	readJSON(userPath("profile.json"), &p)
	return p
}

func saveProfile(p Profile) error {
	return writeJSON(userPath("profile.json"), p)
}

func cmdProfile(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: profile set|get|delete")
		os.Exit(1)
	}

	switch args[0] {
	case "get":
		p := loadProfile()
		okOut(map[string]interface{}{"profile": p})

	case "set":
		if len(args) < 3 {
			errOut("usage: profile set <key> <value>")
			os.Exit(1)
		}
		key, val := args[1], args[2]
		p := loadProfile()
		if p.CreatedAt == "" {
			p.CreatedAt = vnNow().Format("2006-01-02T15:04:05-07:00")
		}
		switch key {
		case "name":
			p.Name = val
		case "income":
			v, err := parseAmount(val)
			if err != nil {
				errOut("invalid amount: " + val)
				os.Exit(1)
			}
			p.Income = v
		case "monthly_fixed":
			v, err := parseAmount(val)
			if err != nil {
				errOut("invalid amount: " + val)
				os.Exit(1)
			}
			p.MonthlyFixed = v
		case "goal":
			p.Goal = val
		case "risk_level":
			p.RiskLevel = val
		case "knowledge_level":
			p.KnowledgeLevel = val
		case "credit_cards":
			p.CreditCards = val
		case "daily_tips":
			p.DailyTips = val == "true" || val == "yes" || val == "1"
		default:
			errOut("unknown profile key: " + key)
			os.Exit(1)
		}
		if err := saveProfile(p); err != nil {
			errOut("failed to save profile: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"key": key, "value": val})

	case "delete":
		os.Remove(userPath("profile.json"))
		okOut(map[string]interface{}{"deleted": true})

	default:
		errOut("unknown profile command: " + args[0])
		os.Exit(1)
	}
}

func cmdOnboard(args []string) {
	ensureInit()
	if len(args) == 0 {
		errOut("usage: onboard status|complete")
		os.Exit(1)
	}

	switch args[0] {
	case "status":
		p := loadProfile()
		okOut(map[string]interface{}{"onboarded": p.Onboarded})

	case "complete":
		p := loadProfile()
		p.Onboarded = true
		if p.CreatedAt == "" {
			p.CreatedAt = vnNow().Format("2006-01-02T15:04:05-07:00")
		}
		if err := saveProfile(p); err != nil {
			errOut("failed to save profile: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"onboarded": true})

	default:
		errOut("unknown onboard command: " + args[0])
		os.Exit(1)
	}
}

func cmdInit() {
	dir := skillDir()
	os.MkdirAll(dir, 0o755)

	// Create empty user data files if they don't exist
	files := []struct {
		name    string
		content string
	}{
		{"profile.json", "{}"},
		{"transactions.json", "[]"},
		{"loyalty.json", "[]"},
		{"user_deals.json", "[]"},
		{"challenge_state.json", "{}"},
		{"quiz_state.json", `{"answered":[],"score":0,"streak":0}`},
		{"feedback.json", "[]"},
	}

	created := 0
	for _, f := range files {
		p := userPath(f.name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			os.WriteFile(p, []byte(f.content), 0o644)
			created++
		}
	}
	okOut(map[string]interface{}{"initialized": true, "files_created": created})
}
