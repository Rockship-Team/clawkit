package main

import (
	"os"
)

// Profile is the user's financial profile.
type Profile struct {
	UserID         string `json:"user_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Income         int64  `json:"income,omitempty"`
	MonthlyFixed   int64  `json:"monthly_fixed,omitempty"`
	MonthlyBudget  int64  `json:"monthly_budget,omitempty"`
	Goal           string `json:"goal,omitempty"`
	RiskLevel      string `json:"risk_level,omitempty"`
	CreditCards    string `json:"credit_cards,omitempty"`
	KnowledgeLevel string `json:"knowledge_level,omitempty"`
	DailyTips      bool   `json:"daily_tips"`
	TipCategories  string `json:"tip_categories,omitempty"`
	DealCategories string `json:"deal_categories,omitempty"`
	ReferralCode   string `json:"referral_code,omitempty"`
	Onboarded      bool   `json:"onboarded"`
	CreatedAt      string `json:"created_at,omitempty"`
}

func currentUserID() string {
	if v := os.Getenv("SOL_USER_ID"); v != "" {
		return v
	}
	return "default"
}

func isZeroProfile(p Profile) bool {
	return p.Name == "" &&
		p.Income == 0 &&
		p.MonthlyFixed == 0 &&
		p.MonthlyBudget == 0 &&
		p.Goal == "" &&
		p.RiskLevel == "" &&
		p.CreditCards == "" &&
		p.KnowledgeLevel == "" &&
		!p.DailyTips &&
		p.TipCategories == "" &&
		p.DealCategories == "" &&
		p.ReferralCode == "" &&
		!p.Onboarded &&
		p.CreatedAt == ""
}

func loadAllProfiles() []Profile {
	var ps []Profile
	if readJSON(userPath("profile.json"), &ps) {
		for i := range ps {
			if ps[i].UserID == "" {
				ps[i].UserID = "default"
			}
		}
		return ps
	}

	// Backward compatibility: old profile.json as a single object.
	var legacy Profile
	if readJSON(userPath("profile.json"), &legacy) && !isZeroProfile(legacy) {
		legacy.UserID = "default"
		return []Profile{legacy}
	}

	return []Profile{}
}

func loadProfile() Profile {
	uid := currentUserID()
	for _, p := range loadAllProfiles() {
		if p.UserID == uid {
			return p
		}
	}
	return Profile{UserID: uid}
}

func saveProfile(p Profile) error {
	if p.UserID == "" {
		p.UserID = currentUserID()
	}
	ps := loadAllProfiles()
	for i := range ps {
		if ps[i].UserID == p.UserID {
			ps[i] = p
			return writeJSON(userPath("profile.json"), ps)
		}
	}
	ps = append(ps, p)
	return writeJSON(userPath("profile.json"), ps)
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
		okOut(map[string]interface{}{"profile": p, "user_id": currentUserID()})

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
		case "monthly_budget":
			v, err := parseAmount(val)
			if err != nil {
				errOut("invalid amount: " + val)
				os.Exit(1)
			}
			p.MonthlyBudget = v
		case "daily_tips":
			p.DailyTips = val == "true" || val == "yes" || val == "1"
		case "tip_categories":
			p.TipCategories = val
		case "deal_categories":
			p.DealCategories = val
		case "referral_code":
			p.ReferralCode = val
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
		uid := currentUserID()
		ps := loadAllProfiles()
		updated := make([]Profile, 0, len(ps))
		deleted := false
		for _, p := range ps {
			if p.UserID == uid {
				deleted = true
				continue
			}
			updated = append(updated, p)
		}
		if err := writeJSON(userPath("profile.json"), updated); err != nil {
			errOut("failed to save profile: " + err.Error())
			os.Exit(1)
		}
		okOut(map[string]interface{}{"deleted": deleted, "user_id": uid})

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
		okOut(map[string]interface{}{"onboarded": p.Onboarded, "created_at": p.CreatedAt})

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
	ensureInit()

	// Create empty runtime user data files under data/ if they don't exist.
	files := []struct {
		name    string
		content string
	}{
		{"profile.json", "[]"},
		{"transactions.json", "[]"},
		{"loyalty.json", "[]"},
	}

	created := 0
	for _, f := range files {
		p := userPath(f.name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			os.WriteFile(p, []byte(f.content), 0o644)
			created++
		}
	}
	okOut(map[string]interface{}{"initialized": true, "data_dir": dataPath(""), "files_created": created})
}
