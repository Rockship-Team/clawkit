package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// runAll crawls all sources and writes results to the data/ directory.
func runAll() {
	dataDir := findDataDir()
	_ = os.MkdirAll(dataDir, 0o755)
	info("output directory: " + dataDir)

	// Cards
	info("=== Crawling credit cards ===")
	cards := crawlAllCards()
	writeDataFile(filepath.Join(dataDir, "credit-cards.json"), cards)

	// Deals
	info("=== Crawling deals ===")
	deals := crawlAllDeals()
	writeDataFile(filepath.Join(dataDir, "deals.json"), deals)

	// Loyalty
	info("=== Crawling loyalty programs ===")
	loyalty := crawlAllLoyalty()
	writeDataFile(filepath.Join(dataDir, "loyalty-catalog.json"), loyalty)

	info("=== Done! Files written to " + dataDir + " ===")
	info("Crawl data updated directly in final files:")
	info("  - credit-cards.json")
	info("  - deals.json")
	info("  - loyalty-catalog.json")
}

func crawlAllCards() []CreditCard {
	cfg := getConfig()
	var all []CreditCard
	for _, src := range cfg.CardSources {
		cards, err := crawlTheBankArticleList(src.URL, src.Category)
		if err != nil {
			info("  " + src.URL + ": " + err.Error())
			continue
		}
		all = append(all, cards...)
	}
	for _, bp := range cfg.BankCardPages {
		cards, err := crawlBankCardPage(bp.Bank, bp.URL)
		if err != nil {
			info("  " + bp.Bank + ": " + err.Error())
			continue
		}
		all = append(all, cards...)
	}
	return deduplicateCards(all)
}

func crawlAllDeals() []Deal {
	cfg := getConfig()
	var all []Deal
	for _, bp := range cfg.BankPromoPages {
		deals, err := crawlPromoPage(bp.Bank, bp.URL)
		if err == nil {
			all = append(all, deals...)
		}
	}
	for _, wp := range cfg.WalletPromos {
		deals, err := crawlPromoPage(wp.Name, wp.URL)
		if err == nil {
			all = append(all, deals...)
		}
	}
	return all
}

func crawlAllLoyalty() []LoyaltyCatalogEntry {
	cfg := getConfig()
	var all []LoyaltyCatalogEntry
	for _, src := range cfg.LoyaltyProgs {
		entry, err := crawlLoyaltyProgram(struct {
			Program string
			Display string
			Type    string
			URL     string
		}{src.Program, src.Display, src.Type, src.URL})
		if err != nil {
			all = append(all, LoyaltyCatalogEntry{
				Program:   src.Program,
				Display:   src.Display,
				Type:      src.Type,
				SourceURL: src.URL,
			})
			continue
		}
		all = append(all, *entry)
	}
	return all
}

func findDataDir() string {
	// Try relative path first (when run from tools/crawl/)
	candidates := []string{
		"../../data",
		"data",
		"skills/finance/sol-finance-coach/data",
		"skills/sol-finance-coach/data",
	}
	for _, d := range candidates {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			abs, _ := filepath.Abs(d)
			return abs
		}
	}
	// Fallback: create data/ in cwd
	os.MkdirAll("data", 0o755)
	abs, _ := filepath.Abs("data")
	return abs
}

func writeDataFile(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		info("failed to marshal: " + err.Error())
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		info("failed to write " + path + ": " + err.Error())
		return
	}
	info("wrote " + path)
}
