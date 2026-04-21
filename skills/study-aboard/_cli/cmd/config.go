package main

import (
	"encoding/json"
	"os"
)

// Connections holds channel credentials and LLM config.
// Stored at ~/.openclaw/workspace/sa-data/connections.json
type Connections struct {
	LLM struct {
		Provider string `json:"provider"` // claude, gpt
		APIKey   string `json:"api_key"`
		Model    string `json:"model"`
	} `json:"llm"`

	Telegram struct {
		BotToken string `json:"bot_token"`
	} `json:"telegram"`

	Zalo struct {
		OAToken string `json:"oa_token"`
		OAID    string `json:"oa_id"`
	} `json:"zalo"`

	App struct {
		DefaultChannel string `json:"default_channel"` // telegram | zalo | web
		Timezone       string `json:"timezone"`        // Asia/Ho_Chi_Minh
	} `json:"app"`
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
	os.MkdirAll(saDir(), 0o755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath(), data, 0o644)
}

func cmdConfig(args []string) {
	if len(args) == 0 {
		errOut("usage: config show|set|get <key> [value]")
	}
	switch args[0] {
	case "show":
		c := loadConnections()
		if c.LLM.APIKey != "" {
			c.LLM.APIKey = "***"
		}
		if c.Telegram.BotToken != "" {
			c.Telegram.BotToken = "***"
		}
		if c.Zalo.OAToken != "" {
			c.Zalo.OAToken = "***"
		}
		okOut(map[string]interface{}{"connections": c, "path": cfgPath()})

	case "set":
		if len(args) < 3 {
			errOut("usage: config set <key> <value>")
		}
		c := loadConnections()
		key, val := args[1], args[2]
		switch key {
		case "llm.provider":
			c.LLM.Provider = val
		case "llm.api_key":
			c.LLM.APIKey = val
		case "llm.model":
			c.LLM.Model = val
		case "telegram.bot_token":
			c.Telegram.BotToken = val
		case "zalo.oa_token":
			c.Zalo.OAToken = val
		case "zalo.oa_id":
			c.Zalo.OAID = val
		case "app.default_channel":
			c.App.DefaultChannel = val
		case "app.timezone":
			c.App.Timezone = val
		default:
			errOut("unknown config key: " + key)
		}
		if err := saveConnections(c); err != nil {
			errOut("save failed: " + err.Error())
		}
		okOut(map[string]interface{}{"key": key, "set": true})

	case "get":
		if len(args) < 2 {
			errOut("usage: config get <key>")
		}
		c := loadConnections()
		var val string
		switch args[1] {
		case "llm.provider":
			val = c.LLM.Provider
		case "llm.model":
			val = c.LLM.Model
		case "app.default_channel":
			val = c.App.DefaultChannel
		case "app.timezone":
			val = c.App.Timezone
		default:
			errOut("unknown config key: " + args[1])
		}
		okOut(map[string]interface{}{"key": args[1], "value": val})

	default:
		errOut("unknown config command: " + args[0])
	}
}
