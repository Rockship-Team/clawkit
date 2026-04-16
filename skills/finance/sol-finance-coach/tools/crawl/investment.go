package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InvestmentData represents a fund NAV or gold price data point.
type InvestmentData struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"` // "fund" or "gold"
	Value     float64 `json:"value"`
	Change    float64 `json:"change_pct,omitempty"`
	Date      string  `json:"date"`
	SourceURL string  `json:"source_url"`
}

var numberRe = regexp.MustCompile(`[\d,.]+`)

// crawlFundNAV extracts the latest NAV from a fund page (e.g. fmarket.vn).
func crawlFundNAV(src InvestmentSource) (*InvestmentData, error) {
	doc, err := fetchDoc(src.URL)
	if err != nil {
		return nil, err
	}

	// Look for NAV value — typically in a prominent number display
	// fmarket.vn shows NAV in large text near fund name
	body := textContent(doc)

	// Try to find patterns like "NAV: 12,345" or "12.345,67 VND"
	navPatterns := []string{
		`(?i)nav[:\s]+([0-9.,]+)`,
		`(?i)gia\s+tri[:\s]+([0-9.,]+)`,
		`(?i)gia\s+hien\s+tai[:\s]+([0-9.,]+)`,
	}

	for _, pat := range navPatterns {
		re := regexp.MustCompile(pat)
		m := re.FindStringSubmatch(body)
		if len(m) >= 2 {
			val := parseFloat(m[1])
			if val > 0 {
				return &InvestmentData{
					Name:      src.Name,
					Type:      src.Type,
					Value:     val,
					Date:      time.Now().Format("2006-01-02"),
					SourceURL: src.URL,
				}, nil
			}
		}
	}

	// Fallback: find first large number that looks like a price
	nums := numberRe.FindAllString(body, 50)
	for _, n := range nums {
		val := parseFloat(n)
		if val >= 1000 && val < 1000000 {
			return &InvestmentData{
				Name:      src.Name,
				Type:      src.Type,
				Value:     val,
				Date:      time.Now().Format("2006-01-02"),
				SourceURL: src.URL,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not extract NAV from %s", src.URL)
}

// crawlGoldPrice extracts gold price from SJC or PNJ pages.
func crawlGoldPrice(src InvestmentSource) (*InvestmentData, error) {
	body, err := fetchBody(src.URL)
	if err != nil {
		return nil, err
	}

	// Gold prices are typically displayed in millions (e.g. "95.500" = 95,500,000 VND per tael)
	goldPatterns := []string{
		`(?i)ban[:\s]+([0-9.,]+)`,
		`(?i)sell[:\s]+([0-9.,]+)`,
		`(?i)sjc[:\s]+([0-9.,]+)`,
		`(?i)vang[:\s]+([0-9.,]+)`,
	}

	for _, pat := range goldPatterns {
		re := regexp.MustCompile(pat)
		m := re.FindStringSubmatch(body)
		if len(m) >= 2 {
			val := parseFloat(m[1])
			if val > 0 {
				return &InvestmentData{
					Name:      src.Name,
					Type:      src.Type,
					Value:     val,
					Date:      time.Now().Format("2006-01-02"),
					SourceURL: src.URL,
				}, nil
			}
		}
	}

	// Fallback: find numbers in typical gold price range
	nums := numberRe.FindAllString(body, 100)
	for _, n := range nums {
		val := parseFloat(n)
		// Gold prices in VN are typically 50-200 (million per tael)
		if val >= 50 && val <= 500 {
			return &InvestmentData{
				Name:      src.Name,
				Type:      src.Type,
				Value:     val,
				Date:      time.Now().Format("2006-01-02"),
				SourceURL: src.URL,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not extract gold price from %s", src.URL)
}

func crawlAllInvestment() []InvestmentData {
	cfg := getConfig()
	var all []InvestmentData

	for _, src := range cfg.InvestmentData {
		info("crawling investment data: " + src.Name)
		var data *InvestmentData
		var err error

		switch src.Type {
		case "fund":
			data, err = crawlFundNAV(src)
		case "gold":
			data, err = crawlGoldPrice(src)
		default:
			info("  unknown type: " + src.Type)
			continue
		}

		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  %s = %.2f", data.Name, data.Value))
		all = append(all, *data)
	}

	return all
}

func runInvestment() {
	data := crawlAllInvestment()
	info(fmt.Sprintf("total investment data points: %d", len(data)))
	jsonPrint(data)
}

// parseFloat parses a VN-formatted number (e.g. "12.345,67" or "12,345.67").
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Detect VN format (dot as thousands separator, comma as decimal)
	dotCount := strings.Count(s, ".")
	commaCount := strings.Count(s, ",")

	if dotCount > 0 && commaCount > 0 {
		dotLast := strings.LastIndex(s, ".")
		commaLast := strings.LastIndex(s, ",")
		if commaLast > dotLast {
			// "12.345,67" → VN format
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			// "12,345.67" → US format
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if dotCount > 1 {
		// "12.345.678" → dots are thousands separators
		s = strings.ReplaceAll(s, ".", "")
	} else if commaCount == 1 && dotCount == 0 {
		// "12345,67" → comma is decimal
		s = strings.Replace(s, ",", ".", 1)
	} else if commaCount > 1 {
		// "12,345,678" → commas are thousands separators
		s = strings.ReplaceAll(s, ",", "")
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
