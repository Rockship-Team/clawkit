package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// InterestRate represents a bank deposit rate.
type InterestRate struct {
	Bank      string  `json:"bank"`
	Term      string  `json:"term"`
	RatePct   float64 `json:"rate_pct"`
	Type      string  `json:"type"` // "online" or "counter"
	SourceURL string  `json:"source_url,omitempty"`
}

func runRates() {
	var allRates []InterestRate

	// Strategy 1: Crawl laisuat.vn
	rates, err := crawlLaiSuatVN()
	if err != nil {
		info("laisuat.vn failed: " + err.Error())
	} else {
		info(fmt.Sprintf("laisuat.vn: found %d rate entries", len(rates)))
		allRates = append(allRates, rates...)
	}

	// Strategy 2: Crawl thebank.vn savings articles
	rates2, err := crawlTheBankRates()
	if err != nil {
		info("thebank.vn rates failed: " + err.Error())
	} else {
		info(fmt.Sprintf("thebank.vn: found %d rate entries", len(rates2)))
		allRates = append(allRates, rates2...)
	}

	// Strategy 3: Crawl individual bank pages
	cfg := getConfig()
	for _, bp := range cfg.BankRatePages {
		info("crawling rates from " + bp.Bank)
		rates, err := crawlBankRatePage(bp.Bank, bp.URL)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  found %d entries", len(rates)))
		allRates = append(allRates, rates...)
	}

	info(fmt.Sprintf("total rate entries: %d", len(allRates)))
	jsonPrint(allRates)
}

// crawlLaiSuatVN extracts rates from laisuat.vn which typically has a comparison table.
func crawlLaiSuatVN() ([]InterestRate, error) {
	doc, err := fetchDoc("https://laisuat.vn")
	if err != nil {
		return nil, err
	}

	var rates []InterestRate

	// laisuat.vn has <table> elements with bank rates.
	tables := findElemsByTag(doc, "table")
	for _, table := range tables {
		rates = append(rates, extractRatesFromTable(table, "laisuat.vn")...)
	}

	return rates, nil
}

// crawlTheBankRates extracts rate info from thebank.vn articles.
func crawlTheBankRates() ([]InterestRate, error) {
	doc, err := fetchDoc("https://thebank.vn/blog/tag/lai-suat-gui-tiet-kiem")
	if err != nil {
		return nil, err
	}

	var rates []InterestRate

	// Look for tables inside the page
	tables := findElemsByTag(doc, "table")
	for _, table := range tables {
		rates = append(rates, extractRatesFromTable(table, "thebank.vn")...)
	}

	return rates, nil
}

// crawlBankRatePage extracts rates from a specific bank's rate page.
func crawlBankRatePage(bank, url string) ([]InterestRate, error) {
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	var rates []InterestRate

	// Find all tables and try to extract rates
	tables := findElemsByTag(doc, "table")
	for _, table := range tables {
		entries := extractRatesFromTable(table, url)
		// Override bank name since we know which bank
		for i := range entries {
			if entries[i].Bank == "" {
				entries[i].Bank = bank
			}
		}
		rates = append(rates, entries...)
	}

	return rates, nil
}

// extractRatesFromTable parses an HTML table looking for interest rate data.
// Assumes table has headers like "Kỳ hạn", "Lãi suất", bank names, etc.
func extractRatesFromTable(table *html.Node, sourceURL string) []InterestRate {
	var rates []InterestRate

	rows := findElemsByTag(table, "tr")
	if len(rows) < 2 {
		return nil
	}

	// Parse header row to find column indices
	headerCells := findElemsByTag(rows[0], "th")
	if len(headerCells) == 0 {
		headerCells = findElemsByTag(rows[0], "td")
	}

	termCol := -1
	bankCols := map[int]string{} // col index -> bank name
	rateCol := -1

	for i, cell := range headerCells {
		text := strings.ToLower(textContent(cell))
		if strings.Contains(text, "kỳ hạn") || strings.Contains(text, "ky han") ||
			strings.Contains(text, "term") {
			termCol = i
		} else if strings.Contains(text, "lãi suất") || strings.Contains(text, "lai suat") ||
			strings.Contains(text, "rate") || strings.Contains(text, "%") {
			rateCol = i
		} else {
			// Check if it's a bank name
			for _, b := range knownBanks {
				if strings.Contains(strings.ToLower(text), strings.ToLower(b)) {
					bankCols[i] = b
					break
				}
			}
		}
	}

	// Parse data rows
	for _, row := range rows[1:] {
		cells := findElemsByTag(row, "td")
		if len(cells) < 2 {
			continue
		}

		// Layout 1: term | bank1_rate | bank2_rate | ...
		if termCol >= 0 && len(bankCols) > 0 && termCol < len(cells) {
			term := normalizeTerm(textContent(cells[termCol]))
			if term == "" {
				continue
			}
			for col, bank := range bankCols {
				if col >= len(cells) {
					continue
				}
				rate := parseRate(textContent(cells[col]))
				if rate > 0 {
					rates = append(rates, InterestRate{
						Bank:      bank,
						Term:      term,
						RatePct:   rate,
						Type:      "counter",
						SourceURL: sourceURL,
					})
				}
			}
			continue
		}

		// Layout 2: bank | term | rate
		if len(cells) >= 3 {
			bank := extractBankName(textContent(cells[0]))
			term := normalizeTerm(textContent(cells[1]))
			rate := parseRate(textContent(cells[2]))
			if bank != "" && term != "" && rate > 0 {
				rates = append(rates, InterestRate{
					Bank:      bank,
					Term:      term,
					RatePct:   rate,
					Type:      "counter",
					SourceURL: sourceURL,
				})
			}
			continue
		}

		// Layout 3: term | rate (single bank page)
		if termCol >= 0 && rateCol >= 0 && termCol < len(cells) && rateCol < len(cells) {
			term := normalizeTerm(textContent(cells[termCol]))
			rate := parseRate(textContent(cells[rateCol]))
			if term != "" && rate > 0 {
				rates = append(rates, InterestRate{
					Term:      term,
					RatePct:   rate,
					Type:      "counter",
					SourceURL: sourceURL,
				})
			}
		}
	}

	return rates
}

var rateRegex = regexp.MustCompile(`(\d+[.,]?\d*)\s*%`)

func parseRate(s string) float64 {
	s = strings.TrimSpace(s)
	m := rateRegex.FindStringSubmatch(s)
	if len(m) < 2 {
		// Try parsing as plain number
		s = strings.ReplaceAll(s, ",", ".")
		s = strings.TrimSuffix(s, "%")
		s = strings.TrimSpace(s)
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		if v > 0 && v < 30 { // reasonable rate range
			return v
		}
		return 0
	}
	numStr := strings.ReplaceAll(m[1], ",", ".")
	v, err := strconv.ParseFloat(numStr, 64)
	if err != nil || v <= 0 || v > 30 {
		return 0
	}
	return v
}

func normalizeTerm(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	// Common Vietnamese term formats
	switch {
	case strings.Contains(lower, "không kỳ hạn") || strings.Contains(lower, "khong ky han"):
		return "khong ky han"
	case strings.Contains(lower, "1 tháng") || strings.Contains(lower, "1 thang") || s == "1T":
		return "1 thang"
	case strings.Contains(lower, "3 tháng") || strings.Contains(lower, "3 thang") || s == "3T":
		return "3 thang"
	case strings.Contains(lower, "6 tháng") || strings.Contains(lower, "6 thang") || s == "6T":
		return "6 thang"
	case strings.Contains(lower, "9 tháng") || strings.Contains(lower, "9 thang") || s == "9T":
		return "9 thang"
	case strings.Contains(lower, "12 tháng") || strings.Contains(lower, "12 thang") || strings.Contains(lower, "1 năm") || s == "12T":
		return "12 thang"
	case strings.Contains(lower, "18 tháng") || strings.Contains(lower, "18 thang") || s == "18T":
		return "18 thang"
	case strings.Contains(lower, "24 tháng") || strings.Contains(lower, "24 thang") || strings.Contains(lower, "2 năm") || s == "24T":
		return "24 thang"
	case strings.Contains(lower, "36 tháng") || strings.Contains(lower, "36 thang") || strings.Contains(lower, "3 năm"):
		return "36 thang"
	}

	// Try to extract number + thang/nam pattern
	numMatch := regexp.MustCompile(`(\d+)\s*(tháng|thang|T|t|năm|nam)`).FindStringSubmatch(s)
	if len(numMatch) >= 3 {
		n, _ := strconv.Atoi(numMatch[1])
		unit := strings.ToLower(numMatch[2])
		if unit == "năm" || unit == "nam" {
			n *= 12
		}
		if n > 0 && n <= 120 {
			return strconv.Itoa(n) + " thang"
		}
	}

	if s != "" {
		return s
	}
	return ""
}

func extractBankName(s string) string {
	for _, b := range knownBanks {
		if strings.Contains(strings.ToLower(s), strings.ToLower(b)) {
			return b
		}
	}
	return ""
}
