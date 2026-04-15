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
	BankRatePages  []BankPage      `json:"bank_rate_pages"`
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

	info("using built-in source URLs (no sources.json found)")
	loadedConfig = defaultConfig()
	return loadedConfig
}

func defaultConfig() *SourceConfig {
	return &SourceConfig{
		CardSources: []CardSource{
			{"https://thebank.vn/blog/tag/the-tin-dung-cashback", "cashback"},
			{"https://thebank.vn/blog/tag/the-tin-dung-mien-phi", "free"},
			{"https://thebank.vn/blog/tag/the-tin-dung-tich-diem", "miles"},
		},
		BankCardPages: func() []BankPage {
			var pages []BankPage
			for _, bp := range bankCardPages {
				pages = append(pages, BankPage{Bank: bp.Bank, URL: bp.URL})
			}
			return pages
		}(),
		BankRatePages: func() []BankPage {
			var pages []BankPage
			for _, bp := range bankRatePages {
				pages = append(pages, BankPage{Bank: bp.Bank, URL: bp.URL})
			}
			return pages
		}(),
		BankPromoPages: func() []BankPage {
			var pages []BankPage
			for _, bp := range bankPromoPages {
				pages = append(pages, BankPage{Bank: bp.Bank, URL: bp.URL})
			}
			return pages
		}(),
		WalletPromos: func() []WalletPage {
			var pages []WalletPage
			for _, wp := range walletPromoPages {
				pages = append(pages, WalletPage{Name: wp.Name, URL: wp.URL})
			}
			return pages
		}(),
		LoyaltyProgs: func() []LoyaltySource {
			var progs []LoyaltySource
			for _, ls := range loyaltySources {
				progs = append(progs, LoyaltySource{
					Program: ls.Program, Display: ls.Display,
					Type: ls.Type, URL: ls.URL,
				})
			}
			return progs
		}(),
	}
}
