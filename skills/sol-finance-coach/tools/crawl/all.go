package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// runAll crawls all sources and writes the results directly to the data/ directory.
func runAll() {
	dataDir := findDataDir()
	info("output directory: " + dataDir)

	// Cards
	info("=== Crawling credit cards ===")
	cards := crawlAllCards()
	writeDataFile(filepath.Join(dataDir, "credit-cards-crawled.json"), cards)

	// Rates
	info("=== Crawling interest rates ===")
	rates := crawlAllRates()
	writeDataFile(filepath.Join(dataDir, "interest-rates.json"), rates)

	// Deals
	info("=== Crawling deals ===")
	deals := crawlAllDeals()
	writeDataFile(filepath.Join(dataDir, "deals-seed.json"), deals)

	// Loyalty
	info("=== Crawling loyalty programs ===")
	loyalty := crawlAllLoyalty()
	writeDataFile(filepath.Join(dataDir, "loyalty-catalog.json"), loyalty)

	info("=== Done! Files written to " + dataDir + " ===")
	info("Review the *-crawled.json files, then merge into the main data files:")
	info("  - credit-cards-crawled.json → review and merge into credit-cards.json")
	info("  - interest-rates.json → new data file for the skill")
	info("  - deals-seed.json → seed data for deals")
	info("  - loyalty-catalog.json → loyalty program reference catalog")
}

func crawlAllCards() []CreditCard {
	var all []CreditCard
	for _, src := range cardSources {
		cards, err := crawlTheBankArticleList(src.URL, src.Category)
		if err != nil {
			info("  " + src.URL + ": " + err.Error())
			continue
		}
		all = append(all, cards...)
	}
	for _, bp := range bankCardPages {
		cards, err := crawlBankCardPage(bp.Bank, bp.URL)
		if err != nil {
			info("  " + bp.Bank + ": " + err.Error())
			continue
		}
		all = append(all, cards...)
	}
	return deduplicateCards(all)
}

func crawlAllRates() []InterestRate {
	var all []InterestRate
	rates, err := crawlLaiSuatVN()
	if err == nil {
		all = append(all, rates...)
	}
	rates, err = crawlTheBankRates()
	if err == nil {
		all = append(all, rates...)
	}
	for _, bp := range bankRatePages {
		rates, err := crawlBankRatePage(bp.Bank, bp.URL)
		if err == nil {
			all = append(all, rates...)
		}
	}
	return all
}

func crawlAllDeals() []Deal {
	var all []Deal
	for _, bp := range bankPromoPages {
		deals, err := crawlPromoPage(bp.Bank, bp.URL)
		if err == nil {
			all = append(all, deals...)
		}
	}
	for _, wp := range walletPromoPages {
		deals, err := crawlPromoPage(wp.Name, wp.URL)
		if err == nil {
			all = append(all, deals...)
		}
	}
	return all
}

func crawlAllLoyalty() []LoyaltyCatalogEntry {
	var all []LoyaltyCatalogEntry
	for _, src := range loyaltySources {
		entry, err := crawlLoyaltyProgram(src)
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
