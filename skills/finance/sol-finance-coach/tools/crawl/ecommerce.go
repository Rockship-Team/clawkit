package main

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// crawlEcommercePage extracts promotion deals from an e-commerce sale page.
func crawlEcommercePage(name, url string) ([]Deal, error) {
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	var deals []Deal

	// Look for promotion links/cards — e-commerce sites typically have
	// promo sections with titles and descriptions in divs/articles.
	articles := findAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		// Match common promo card patterns
		if n.Data == "a" || n.Data == "article" || n.Data == "div" {
			cls := getAttr(n, "class")
			lower := strings.ToLower(cls)
			return strings.Contains(lower, "promo") ||
				strings.Contains(lower, "campaign") ||
				strings.Contains(lower, "deal") ||
				strings.Contains(lower, "voucher") ||
				strings.Contains(lower, "banner")
		}
		return false
	})

	for _, a := range articles {
		text := strings.TrimSpace(textContent(a))
		if len(text) < 10 || len(text) > 500 {
			continue
		}

		// Try to extract link URL
		href := ""
		if a.Data == "a" {
			href = getAttr(a, "href")
		} else {
			link := findFirst(a, func(n *html.Node) bool { return isElem(n, "a") })
			if link != nil {
				href = getAttr(link, "href")
			}
		}

		// Determine category from text
		cat := categorizeEcommerceDeal(text)

		deals = append(deals, Deal{
			Source:      name,
			Description: text,
			Category:    cat,
			URL:         href,
		})
	}

	// Fallback: if no structured promos found, grab headings as deal titles
	if len(deals) == 0 {
		headings := findAll(doc, func(n *html.Node) bool {
			return isElem(n, "h2") || isElem(n, "h3") || isElem(n, "h4")
		})
		for _, h := range headings {
			text := strings.TrimSpace(textContent(h))
			if len(text) < 5 || len(text) > 300 {
				continue
			}
			// Skip navigation/footer headings
			lower := strings.ToLower(text)
			if strings.Contains(lower, "menu") || strings.Contains(lower, "footer") {
				continue
			}
			deals = append(deals, Deal{
				Source:      name,
				Description: text,
				Category:    categorizeEcommerceDeal(text),
			})
		}
	}

	return deals, nil
}

func categorizeEcommerceDeal(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "dien tu") || strings.Contains(lower, "laptop") || strings.Contains(lower, "phone"):
		return "shopping"
	case strings.Contains(lower, "thoi trang") || strings.Contains(lower, "giay") || strings.Contains(lower, "quan ao"):
		return "shopping"
	case strings.Contains(lower, "an") || strings.Contains(lower, "food") || strings.Contains(lower, "do uong"):
		return "food"
	case strings.Contains(lower, "du lich") || strings.Contains(lower, "travel") || strings.Contains(lower, "khach san"):
		return "travel"
	default:
		return "shopping"
	}
}

func crawlAllEcommerce() []Deal {
	cfg := getConfig()
	var all []Deal
	today := time.Now().Format("2006-01-02")

	for _, src := range cfg.EcommerceSales {
		info("crawling e-commerce deals from " + src.Name)
		deals, err := crawlEcommercePage(src.Name, src.URL)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		// Set today as crawl date for expiry tracking
		for i := range deals {
			if deals[i].Expiry == "" {
				deals[i].Expiry = today
			}
		}
		info(fmt.Sprintf("  found %d deals", len(deals)))
		all = append(all, deals...)
	}

	return all
}

func runEcommerce() {
	deals := crawlAllEcommerce()
	info(fmt.Sprintf("total e-commerce deals: %d", len(deals)))
	jsonPrint(deals)
}
