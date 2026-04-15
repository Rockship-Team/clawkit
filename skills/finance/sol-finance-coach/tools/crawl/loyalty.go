package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// LoyaltyCatalogEntry describes a loyalty program's earning/redemption rules.
type LoyaltyCatalogEntry struct {
	Program        string            `json:"program"`
	Display        string            `json:"display"`
	Type           string            `json:"type"` // "airline", "bank", "wallet", "retail"
	EarningRules   []string          `json:"earning_rules"`
	RedemptionInfo []string          `json:"redemption_info"`
	ExpiryPolicy   string            `json:"expiry_policy"`
	Tips           []string          `json:"tips"`
	SourceURL      string            `json:"source_url,omitempty"`
	Partners       map[string]string `json:"partners,omitempty"`
}

// Known loyalty program sources.
var loyaltySources = []struct {
	Program string
	Display string
	Type    string
	URL     string
}{
	{"lotusmiles", "Vietnam Airlines Lotusmiles", "airline", "https://www.vietnamairlines.com/vn/vi/lotusmiles/about"},
	{"bamboo_club", "Bamboo Airways Club", "airline", "https://www.bambooairways.com/bamboo-club"},
	{"grab_rewards", "GrabRewards", "wallet", "https://www.grab.com/vn/rewards/"},
	{"shopee_coins", "Shopee Xu", "wallet", "https://shopee.vn/m/shopee-coins"},
	{"momo_rewards", "MoMo Rewards", "wallet", "https://momo.vn"},
	{"zalopay_rewards", "ZaloPay Rewards", "wallet", "https://zalopay.vn"},
	{"the_coffee_house", "The Coffee House Rewards", "retail", "https://www.thecoffeehouse.com"},
	{"highlands", "Highlands Coffee Rewards", "retail", "https://highlandscoffee.com.vn"},
	{"cgv", "CGV Cinema Club", "retail", "https://www.cgv.vn"},
}

func runLoyalty() {
	cfg := getConfig()
	var catalog []LoyaltyCatalogEntry

	for _, src := range cfg.LoyaltyProgs {
		info("crawling loyalty program: " + src.Display)
		entry, err := crawlLoyaltyProgram(struct {
			Program string
			Display string
			Type    string
			URL     string
		}{src.Program, src.Display, src.Type, src.URL})
		if err != nil {
			info("  failed: " + err.Error())
			catalog = append(catalog, LoyaltyCatalogEntry{
				Program:   src.Program,
				Display:   src.Display,
				Type:      src.Type,
				SourceURL: src.URL,
			})
			continue
		}
		catalog = append(catalog, *entry)
	}

	info(fmt.Sprintf("total loyalty programs: %d", len(catalog)))
	jsonPrint(catalog)
}

func crawlLoyaltyProgram(src struct {
	Program string
	Display string
	Type    string
	URL     string
}) (*LoyaltyCatalogEntry, error) {
	doc, err := fetchDoc(src.URL)
	if err != nil {
		return nil, err
	}

	entry := &LoyaltyCatalogEntry{
		Program:   src.Program,
		Display:   src.Display,
		Type:      src.Type,
		SourceURL: src.URL,
		Partners:  make(map[string]string),
	}

	// Extract earning rules from the page
	entry.EarningRules = extractLoyaltySection(doc, []string{
		"tích điểm", "tich diem", "earn", "kiếm", "kiem",
		"tích lũy", "tich luy", "accumulate", "cách tích", "cach tich",
	})

	// Extract redemption info
	entry.RedemptionInfo = extractLoyaltySection(doc, []string{
		"đổi điểm", "doi diem", "redeem", "quy đổi", "quy doi",
		"sử dụng điểm", "su dung diem", "đổi quà", "doi qua",
		"đổi thưởng", "doi thuong",
	})

	// Extract expiry policy
	entry.ExpiryPolicy = extractExpiryPolicy(doc)

	// Extract partner info
	entry.Partners = extractPartners(doc)

	return entry, nil
}

// extractLoyaltySection finds text content related to given keywords in headings,
// then collects list items or paragraphs under those headings.
func extractLoyaltySection(doc *html.Node, keywords []string) []string {
	var rules []string

	// Find headings containing keywords
	headings := findAll(doc, func(n *html.Node) bool {
		if !isElem(n, "h1") && !isElem(n, "h2") && !isElem(n, "h3") && !isElem(n, "h4") {
			return false
		}
		text := strings.ToLower(textContent(n))
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				return true
			}
		}
		return false
	})

	for _, h := range headings {
		// Collect list items and paragraphs after the heading
		for sib := h.NextSibling; sib != nil; sib = sib.NextSibling {
			if sib.Type != html.ElementNode {
				continue
			}
			// Stop at next heading
			if isElem(sib, "h1") || isElem(sib, "h2") || isElem(sib, "h3") || isElem(sib, "h4") {
				break
			}
			// Collect <li> items
			if isElem(sib, "ul") || isElem(sib, "ol") {
				for _, li := range findElemsByTag(sib, "li") {
					text := strings.TrimSpace(textContent(li))
					if text != "" && len(text) > 5 {
						rules = append(rules, text)
					}
				}
			}
			// Collect <p> content
			if isElem(sib, "p") {
				text := strings.TrimSpace(textContent(sib))
				if text != "" && len(text) > 10 {
					rules = append(rules, text)
				}
			}
		}
	}

	// If no structured content found, try to find list items with keywords in their text
	if len(rules) == 0 {
		allLis := findElemsByTag(doc, "li")
		for _, li := range allLis {
			text := textContent(li)
			lower := strings.ToLower(text)
			for _, kw := range keywords {
				if strings.Contains(lower, kw) {
					trimmed := strings.TrimSpace(text)
					if trimmed != "" && len(trimmed) > 10 && len(trimmed) < 300 {
						rules = append(rules, trimmed)
					}
					break
				}
			}
		}
	}

	// Limit to reasonable count
	if len(rules) > 10 {
		rules = rules[:10]
	}

	return rules
}

func extractExpiryPolicy(doc *html.Node) string {
	keywords := []string{
		"hết hạn", "het han", "expir", "thời hạn", "thoi han",
		"hiệu lực", "hieu luc", "valid",
	}

	// Look for text containing expiry keywords
	paragraphs := findElemsByTag(doc, "p")
	for _, p := range paragraphs {
		text := textContent(p)
		lower := strings.ToLower(text)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				trimmed := strings.TrimSpace(text)
				if len(trimmed) > 10 && len(trimmed) < 300 {
					return trimmed
				}
			}
		}
	}

	// Check list items too
	lis := findElemsByTag(doc, "li")
	for _, li := range lis {
		text := textContent(li)
		lower := strings.ToLower(text)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				trimmed := strings.TrimSpace(text)
				if len(trimmed) > 10 && len(trimmed) < 300 {
					return trimmed
				}
			}
		}
	}

	return ""
}

func extractPartners(doc *html.Node) map[string]string {
	partners := make(map[string]string)

	keywords := []string{
		"đối tác", "doi tac", "partner", "liên kết", "lien ket",
	}

	headings := findAll(doc, func(n *html.Node) bool {
		if !isElem(n, "h2") && !isElem(n, "h3") && !isElem(n, "h4") {
			return false
		}
		text := strings.ToLower(textContent(n))
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				return true
			}
		}
		return false
	})

	for _, h := range headings {
		for sib := h.NextSibling; sib != nil; sib = sib.NextSibling {
			if sib.Type != html.ElementNode {
				continue
			}
			if isElem(sib, "h2") || isElem(sib, "h3") {
				break
			}
			if isElem(sib, "ul") || isElem(sib, "ol") {
				for _, li := range findElemsByTag(sib, "li") {
					text := strings.TrimSpace(textContent(li))
					if text != "" {
						// Try to split "Partner: description" or just store the text
						parts := strings.SplitN(text, ":", 2)
						if len(parts) == 2 {
							partners[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
						} else {
							partners[text] = ""
						}
					}
				}
			}
		}
	}

	return partners
}
