package main

import (
	"encoding/json"
	"os"
)

// Connections holds remote service credentials.
// Loaded from ~/.openclaw/workspace/sme-data/connections.json
type Connections struct {
	// MISA AMIS integration
	MISA struct {
		AppID       string `json:"app_id"`
		AccessToken string `json:"access_token"`
		OrgCode     string `json:"org_code"`
		BaseURL     string `json:"base_url"`
	} `json:"misa"`

	// LLM for OCR, RAG, content generation
	LLM struct {
		Provider string `json:"provider"` // claude, gpt, deepseek
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
	} `json:"llm"`

	// Zalo OA for notifications
	Zalo struct {
		OAID    string `json:"oa_id"`
		OAToken string `json:"oa_token"`
	} `json:"zalo"`

	// Remote PostgreSQL (optional, for multi-user / server mode)
	Postgres struct {
		DSN string `json:"dsn"` // postgres://user:pass@host:5432/db?sslmode=disable
	} `json:"postgres"`

	// Email (for campaigns, notifications)
	Email struct {
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
		Username string `json:"username"`
		Password string `json:"password"`
		FromName string `json:"from_name"`
	} `json:"email"`

	// Organization defaults
	Org struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		TaxCode     string `json:"tax_code"`
		PaymentInfo string `json:"payment_info"` // Bank info for paid events (STK/NH/ND/...)
	} `json:"org"`

	// COSMO CRM (Rockship)
	COSMO struct {
		APIKey    string `json:"api_key"`
		BaseURL   string `json:"base_url"`
		AuthEmail string `json:"auth_email"`
	} `json:"cosmo"`

	// Apollo.io
	Apollo struct {
		APIKey string `json:"api_key"`
	} `json:"apollo"`

	// LinkedIn (read-only, via a locally-running Chrome CDP endpoint)
	LinkedIn struct {
		CDPUrl     string `json:"cdp_url"`
		LastSyncAt string `json:"last_sync_at"`
	} `json:"linkedin"`

	// Proposal pricing — overrides the hardcoded defaults in proposal.go when
	// set. Empty/missing fields fall back to the exact current defaults, so a
	// missing or partial config never breaks proposal generation.
	Proposal struct {
		Tiers         []proposalTier `json:"tiers,omitempty"`
		AddOns        []string       `json:"add_ons,omitempty"`
		DiscountRules []string       `json:"discount_rules,omitempty"`
	} `json:"proposal"`
}

func loadConnections() Connections {
	var c Connections
	data, err := os.ReadFile(cfgPath())
	if err != nil {
		return c
	}
	json.Unmarshal(data, &c)
	return c
}

func saveConnections(c Connections) error {
	os.MkdirAll(smeDir(), 0o755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath(), data, 0o644)
}

func defaultOrgID() string {
	c := loadConnections()
	if c.Org.ID != "" {
		return c.Org.ID
	}
	return "default"
}

func cmdConfig(args []string) {
	if len(args) == 0 {
		errOut("usage: config show|set|get <key> [value]")
	}
	switch args[0] {
	case "show":
		c := loadConnections()
		// Mask secrets
		if c.MISA.AccessToken != "" {
			c.MISA.AccessToken = "***"
		}
		if c.LLM.APIKey != "" {
			c.LLM.APIKey = "***"
		}
		if c.Zalo.OAToken != "" {
			c.Zalo.OAToken = "***"
		}
		if c.Email.Password != "" {
			c.Email.Password = "***"
		}
		if c.COSMO.APIKey != "" {
			c.COSMO.APIKey = "***"
		}
		if c.Apollo.APIKey != "" {
			c.Apollo.APIKey = "***"
		}
		okOut(map[string]interface{}{"connections": c, "path": cfgPath()})

	case "set":
		if len(args) < 3 {
			errOut("usage: config set <section.key> <value>")
		}
		c := loadConnections()
		key, val := args[1], args[2]
		switch key {
		case "org.id":
			c.Org.ID = val
		case "org.name":
			c.Org.Name = val
		case "org.tax_code":
			c.Org.TaxCode = val
		case "misa.app_id":
			c.MISA.AppID = val
		case "misa.access_token":
			c.MISA.AccessToken = val
		case "misa.org_code":
			c.MISA.OrgCode = val
		case "misa.base_url":
			c.MISA.BaseURL = val
		case "llm.provider":
			c.LLM.Provider = val
		case "llm.api_key":
			c.LLM.APIKey = val
		case "llm.model":
			c.LLM.Model = val
		case "zalo.oa_id":
			c.Zalo.OAID = val
		case "zalo.oa_token":
			c.Zalo.OAToken = val
		case "postgres.dsn":
			c.Postgres.DSN = val
		case "email.smtp_host":
			c.Email.SMTPHost = val
		case "email.username":
			c.Email.Username = val
		case "email.password":
			c.Email.Password = val
		case "email.from_name":
			c.Email.FromName = val
		case "cosmo.api_key":
			c.COSMO.APIKey = val
		case "cosmo.base_url":
			c.COSMO.BaseURL = val
		case "cosmo.auth_email":
			c.COSMO.AuthEmail = val
		case "apollo.api_key":
			c.Apollo.APIKey = val
		case "linkedin.cdp_url":
			c.LinkedIn.CDPUrl = val
		case "proposal.tiers_json":
			var tiers []proposalTier
			if err := json.Unmarshal([]byte(val), &tiers); err != nil {
				errOut("proposal.tiers_json must be a JSON array of tiers: " + err.Error())
			}
			c.Proposal.Tiers = tiers
		case "proposal.add_ons_json":
			var addOns []string
			if err := json.Unmarshal([]byte(val), &addOns); err != nil {
				errOut("proposal.add_ons_json must be a JSON array of strings: " + err.Error())
			}
			c.Proposal.AddOns = addOns
		case "proposal.discount_rules_json":
			var rules []string
			if err := json.Unmarshal([]byte(val), &rules); err != nil {
				errOut("proposal.discount_rules_json must be a JSON array of strings: " + err.Error())
			}
			c.Proposal.DiscountRules = rules
		default:
			errOut("unknown config key: " + key)
		}
		if err := saveConnections(c); err != nil {
			errOut("save failed: " + err.Error())
		}
		okOut(map[string]interface{}{"key": key, "set": true})

	case "get":
		if len(args) < 2 {
			errOut("usage: config get <section.key>")
		}
		c := loadConnections()
		key := args[1]
		var val string
		switch key {
		case "org.id":
			val = c.Org.ID
		case "org.name":
			val = c.Org.Name
		case "misa.app_id":
			val = c.MISA.AppID
		case "llm.provider":
			val = c.LLM.Provider
		case "llm.model":
			val = c.LLM.Model
		case "cosmo.api_key":
			val = c.COSMO.APIKey
		case "cosmo.base_url":
			val = c.COSMO.BaseURL
		case "cosmo.auth_email":
			val = c.COSMO.AuthEmail
		case "apollo.api_key":
			val = c.Apollo.APIKey
		case "linkedin.cdp_url":
			val = c.LinkedIn.CDPUrl
		case "proposal.tiers_json":
			b, _ := json.Marshal(effectiveProposalTiers())
			val = string(b)
		case "proposal.add_ons_json":
			b, _ := json.Marshal(effectiveProposalAddOns())
			val = string(b)
		case "proposal.discount_rules_json":
			b, _ := json.Marshal(effectiveDiscountRules())
			val = string(b)
		default:
			errOut("unknown config key: " + key)
		}
		okOut(map[string]interface{}{"key": key, "value": val})

	default:
		errOut("unknown config command: " + args[0])
	}
}
