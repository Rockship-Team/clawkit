package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SourceConfig holds all crawl source URLs, loaded from sources.json.
type SourceConfig struct {
	CardSources    []CardSource    `json:"card_sources"`
	BankCardPages  []BankPage      `json:"bank_card_pages"`
	BankPromoPages []BankPage      `json:"bank_promo_pages"`
	WalletPromos   []WalletPage    `json:"wallet_promo_pages"`
	LoyaltyProgs   []LoyaltySource `json:"loyalty_programs"`
}

type CardSource struct {
	URL      string `json:"url"`
	Category string `json:"category"`
}

type BankPage struct {
	Bank string `json:"bank"`
	Name string `json:"name,omitempty"` // for wallet pages
	URL  string `json:"url"`
}

type WalletPage struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type LoyaltySource struct {
	Program string `json:"program"`
	Display string `json:"display"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

var loadedConfig *SourceConfig

// getConfig loads sources.json from next to the binary, or falls back to hardcoded defaults.
func getConfig() *SourceConfig {
	if loadedConfig != nil {
		return loadedConfig
	}

	// Try to find sources.json next to the executable
	candidates := []string{
		"sources.json",
		filepath.Join("tools", "crawl", "sources.json"),
	}

	// Also try next to the binary itself
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "sources.json"))
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg SourceConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			info("warning: failed to parse " + path + ": " + err.Error())
			continue
		}
		info("loaded sources from " + path)
		loadedConfig = &cfg
		return loadedConfig
	}

	fatal("sources.json not found — place it next to the crawl binary or in tools/crawl/", nil)
	return nil // unreachable
}
