package main

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Deal matches the skill's deals format.
type Deal struct {
	Source      string `json:"source"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Expiry      string `json:"expiry,omitempty"`
	URL         string `json:"url,omitempty"`
}

func runDeals(bankFilter string) {
	cfg := getConfig()
	var allDeals []Deal

	// Crawl bank promotions
	for _, bp := range cfg.BankPromoPages {
		if bankFilter != "" && !strings.EqualFold(bp.Bank, bankFilter) {
			continue
		}
		info("crawling deals from " + bp.Bank)
		deals, err := crawlPromoPage(bp.Bank, bp.URL)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  found %d deals", len(deals)))
		allDeals = append(allDeals, deals...)
	}

	// Crawl e-wallet promotions
	for _, wp := range cfg.WalletPromos {
		if bankFilter != "" && !strings.EqualFold(wp.Name, bankFilter) {
			continue
		}
		info("crawling deals from " + wp.Name)
		deals, err := crawlPromoPage(wp.Name, wp.URL)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  found %d deals", len(deals)))
		allDeals = append(allDeals, deals...)
	}

	info(fmt.Sprintf("total deals: %d", len(allDeals)))
	jsonPrint(allDeals)
}

// crawlPromoPage extracts promotions from a bank/wallet promotion listing page.
func crawlPromoPage(source, url string) ([]Deal, error) {
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	var deals []Deal

	// Strategy 1: Look for structured promo blocks (common pattern across bank sites).
	// Banks usually have cards/blocks with: title, description, image, expiry date.
	promoBlocks := findAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		classes := getAttr(n, "class")
		return strings.Contains(classes, "promo") ||
			strings.Contains(classes, "promotion") ||
			strings.Contains(classes, "offer") ||
			strings.Contains(classes, "deal") ||
			strings.Contains(classes, "uu-dai") ||
			strings.Contains(classes, "uudai") ||
			strings.Contains(classes, "campaign") ||
			strings.Contains(classes, "card-item") ||
			strings.Contains(classes, "post-item")
	})

	seen := map[string]bool{}
	for _, block := range promoBlocks {
		deal := extractDealFromBlock(source, url, block)
		if deal != nil && !seen[deal.Description] {
			seen[deal.Description] = true
			deals = append(deals, *deal)
		}
	}

	// Strategy 2: If no promo blocks found, look for article links
	if len(deals) == 0 {
		links := findAll(doc, func(n *html.Node) bool {
			if !isElem(n, "a") {
				return false
			}
			href := getAttr(n, "href")
			return strings.Contains(href, "uu-dai") || strings.Contains(href, "uudai") ||
				strings.Contains(href, "khuyen-mai") || strings.Contains(href, "promo")
		})

		for _, link := range links {
			title := strings.TrimSpace(textContent(link))
			href := getAttr(link, "href")
			if title == "" || len(title) < 10 || len(title) > 200 {
				continue
			}
			if seen[title] {
				continue
			}
			seen[title] = true

			deals = append(deals, Deal{
				Source:      source,
				Description: title,
				Category:    guessDealCategory(title),
				URL:         normalizeURL(url, href),
			})
		}
	}

	// Strategy 3: Look for any <h2>/<h3> inside the page with promo-like content
	if len(deals) == 0 {
		headings := findAll(doc, func(n *html.Node) bool {
			return isElem(n, "h2") || isElem(n, "h3")
		})
		for _, h := range headings {
			text := strings.TrimSpace(textContent(h))
			lower := strings.ToLower(text)
			isPromo := strings.Contains(lower, "ưu đãi") || strings.Contains(lower, "uu dai") ||
				strings.Contains(lower, "giảm") || strings.Contains(lower, "giam") ||
				strings.Contains(lower, "hoàn") || strings.Contains(lower, "hoan") ||
				strings.Contains(lower, "miễn phí") || strings.Contains(lower, "mien phi") ||
				strings.Contains(lower, "khuyến mãi") || strings.Contains(lower, "khuyen mai") ||
				strings.Contains(lower, "%")
			if !isPromo || len(text) < 10 || len(text) > 200 {
				continue
			}
			if seen[text] {
				continue
			}
			seen[text] = true

			deals = append(deals, Deal{
				Source:      source,
				Description: text,
				Category:    guessDealCategory(text),
				URL:         url,
			})
		}
	}

	return deals, nil
}

// extractDealFromBlock extracts a deal from a promo card/block element.
func extractDealFromBlock(source, pageURL string, block *html.Node) *Deal {
	// Find the title (first <h2>, <h3>, <h4>, or element with title/heading class)
	titleNode := findFirst(block, func(n *html.Node) bool {
		if isElem(n, "h2") || isElem(n, "h3") || isElem(n, "h4") {
			return true
		}
		classes := getAttr(n, "class")
		return strings.Contains(classes, "title") || strings.Contains(classes, "heading") || strings.Contains(classes, "name")
	})

	title := ""
	if titleNode != nil {
		title = strings.TrimSpace(textContent(titleNode))
	}
	if title == "" {
		// Fallback: use the full text of the block, truncated
		title = strings.TrimSpace(textContent(block))
		if len(title) > 150 {
			title = title[:150] + "..."
		}
	}

	if title == "" || len(title) < 10 {
		return nil
	}

	// Find expiry date
	expiry := extractExpiry(block)

	// Find link
	linkURL := pageURL
	link := findFirst(block, func(n *html.Node) bool {
		return isElem(n, "a") && getAttr(n, "href") != ""
	})
	if link != nil {
		linkURL = normalizeURL(pageURL, getAttr(link, "href"))
	}

	return &Deal{
		Source:      source,
		Description: title,
		Category:    guessDealCategory(title),
		Expiry:      expiry,
		URL:         linkURL,
	}
}

var dateRegex = regexp.MustCompile(`(\d{1,2})[/\-.](\d{1,2})[/\-.](\d{4})`)

// extractExpiry looks for date patterns in text that indicate deal expiry.
func extractExpiry(block *html.Node) string {
	text := textContent(block)

	// Look for "đến ngày", "hết hạn", "den ngay", "het han", "to" patterns near a date
	lower := strings.ToLower(text)
	idx := -1
	for _, keyword := range []string{"đến", "den", "hết hạn", "het han", "tới", "to ", "- "} {
		i := strings.Index(lower, keyword)
		if i >= 0 && (idx < 0 || i < idx) {
			idx = i
		}
	}

	// Extract date after the keyword
	searchText := text
	if idx >= 0 {
		searchText = text[idx:]
	}

	matches := dateRegex.FindStringSubmatch(searchText)
	if len(matches) >= 4 {
		day, month, year := matches[1], matches[2], matches[3]
		if len(day) == 1 {
			day = "0" + day
		}
		if len(month) == 1 {
			month = "0" + month
		}
		return year + "-" + month + "-" + day
	}

	return ""
}

func guessDealCategory(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "ăn") || strings.Contains(lower, "an ") ||
		strings.Contains(lower, "food") || strings.Contains(lower, "nhà hàng") ||
		strings.Contains(lower, "nha hang") || strings.Contains(lower, "grab") ||
		strings.Contains(lower, "baemin") || strings.Contains(lower, "shopeefood"):
		return "food"
	case strings.Contains(lower, "mua sắm") || strings.Contains(lower, "mua sam") ||
		strings.Contains(lower, "shopping") || strings.Contains(lower, "shopee") ||
		strings.Contains(lower, "lazada") || strings.Contains(lower, "tiki"):
		return "shopping"
	case strings.Contains(lower, "du lịch") || strings.Contains(lower, "du lich") ||
		strings.Contains(lower, "travel") || strings.Contains(lower, "bay") ||
		strings.Contains(lower, "khách sạn") || strings.Contains(lower, "khach san"):
		return "travel"
	case strings.Contains(lower, "giải trí") || strings.Contains(lower, "giai tri") ||
		strings.Contains(lower, "cgv") || strings.Contains(lower, "phim"):
		return "entertainment"
	case strings.Contains(lower, "điện") || strings.Contains(lower, "dien") ||
		strings.Contains(lower, "nước") || strings.Contains(lower, "nuoc") ||
		strings.Contains(lower, "hóa đơn") || strings.Contains(lower, "hoa don"):
		return "bills"
	default:
		return "general"
	}
}

func normalizeURL(baseURL, href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		// Extract scheme + host from base
		parts := strings.SplitN(baseURL, "//", 2)
		if len(parts) == 2 {
			hostParts := strings.SplitN(parts[1], "/", 2)
			return parts[0] + "//" + hostParts[0] + href
		}
	}
	return baseURL + "/" + href
}
