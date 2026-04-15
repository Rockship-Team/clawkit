package main

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// CreditCard matches the skill's data/credit-cards.json schema.
type CreditCard struct {
	ID          string `json:"id"`
	Bank        string `json:"bank"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	AnnualFee   string `json:"annual_fee"`
	Cashback    string `json:"cashback"`
	Rewards     string `json:"rewards"`
	MinIncome   int64  `json:"min_income"`
	InterestPct string `json:"interest_pct"`
	BestFor     string `json:"best_for"`
	SourceURL   string `json:"source_url,omitempty"`
}

// Card listing pages on thebank.vn (credit card comparison site).
var cardSources = []struct {
	URL      string
	Category string
}{
	{"https://thebank.vn/blog/tag/the-tin-dung-cashback", "cashback"},
	{"https://thebank.vn/blog/tag/the-tin-dung-mien-phi", "free"},
	{"https://thebank.vn/blog/tag/the-tin-dung-tich-diem", "miles"},
}

// Bank-specific card listing pages for direct crawling.
var bankCardPages = []struct {
	Bank string
	URL  string
}{
	{"Techcombank", "https://techcombank.com/ca-nhan/the/the-tin-dung"},
	{"VPBank", "https://vpbank.com.vn/ca-nhan/the/the-tin-dung"},
	{"TPBank", "https://tpbank.vn/ca-nhan/the-tin-dung.html"},
	{"ACB", "https://acb.com.vn/the/the-tin-dung"},
	{"MB", "https://www.mbbank.com.vn/the-tin-dung"},
	{"VIB", "https://vib.com.vn/vn/the-tin-dung"},
	{"Sacombank", "https://www.sacombank.com.vn/the-tin-dung"},
	{"BIDV", "https://bidv.com.vn/ca-nhan/the/the-tin-dung"},
	{"Vietcombank", "https://vietcombank.com.vn/the-tin-dung"},
	{"HDBank", "https://hdbank.com.vn/vi/ca-nhan/the-tin-dung"},
	{"OCB", "https://ocb.com.vn/ca-nhan/the-tin-dung"},
	{"SHB", "https://shb.com.vn/ca-nhan/the/the-tin-dung"},
}

func runCards() {
	cfg := getConfig()
	var allCards []CreditCard

	// Strategy 1: Crawl comparison articles (thebank.vn etc.)
	for _, src := range cfg.CardSources {
		info("crawling cards from " + src.URL + " (category: " + src.Category + ")")
		cards, err := crawlTheBankArticleList(src.URL, src.Category)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  found %d cards", len(cards)))
		allCards = append(allCards, cards...)
	}

	// Strategy 2: Crawl individual bank card pages
	for _, bp := range cfg.BankCardPages {
		info("crawling " + bp.Bank + " cards from " + bp.URL)
		cards, err := crawlBankCardPage(bp.Bank, bp.URL)
		if err != nil {
			info("  failed: " + err.Error())
			continue
		}
		info(fmt.Sprintf("  found %d cards", len(cards)))
		allCards = append(allCards, cards...)
	}

	// Deduplicate by ID
	allCards = deduplicateCards(allCards)
	info(fmt.Sprintf("total unique cards: %d", len(allCards)))

	jsonPrint(allCards)
}

// crawlTheBankArticleList extracts card summaries from thebank.vn tag pages.
// These pages list articles about cards — each article title contains bank + card name.
func crawlTheBankArticleList(url, category string) ([]CreditCard, error) {
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	var cards []CreditCard

	// thebank.vn uses article blocks with <h2> or <h3> titles containing card names.
	// Look for article links with card names in them.
	links := findAll(doc, func(n *html.Node) bool {
		if !isElem(n, "a") {
			return false
		}
		href := getAttr(n, "href")
		return strings.Contains(href, "the-tin-dung") || strings.Contains(href, "the-ngan-hang")
	})

	for _, link := range links {
		title := strings.TrimSpace(textContent(link))
		href := getAttr(link, "href")
		if title == "" || len(title) < 10 {
			continue
		}

		bank, name := extractBankAndCard(title)
		if bank == "" || name == "" {
			continue
		}

		card := CreditCard{
			ID:        slugify(bank + "_" + name),
			Bank:      bank,
			Name:      name,
			Category:  category,
			SourceURL: href,
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// crawlBankCardPage extracts card names from a bank's credit card listing page.
// Banks have varied HTML structures, so this uses a general heuristic approach.
func crawlBankCardPage(bank, url string) ([]CreditCard, error) {
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	var cards []CreditCard

	// Heuristic: find elements that look like card names.
	// Banks typically list cards in <h2>, <h3>, <h4> or <div class="card-name">.
	headings := findAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		// Look for heading tags
		if n.Data == "h2" || n.Data == "h3" || n.Data == "h4" {
			return true
		}
		// Look for elements with card-related classes
		classes := getAttr(n, "class")
		return strings.Contains(classes, "card-name") ||
			strings.Contains(classes, "product-name") ||
			strings.Contains(classes, "title-card") ||
			strings.Contains(classes, "card-title")
	})

	seen := map[string]bool{}
	for _, h := range headings {
		text := strings.TrimSpace(textContent(h))
		if text == "" || len(text) < 5 || len(text) > 100 {
			continue
		}
		// Filter: must look like a specific card product, not a FAQ heading.
		// Require a card network name OR a specific tier keyword.
		lower := strings.ToLower(text)

		hasNetwork := strings.Contains(lower, "visa") ||
			strings.Contains(lower, "mastercard") ||
			strings.Contains(lower, "jcb") ||
			strings.Contains(lower, "amex") ||
			strings.Contains(lower, "unionpay")

		hasTier := strings.Contains(lower, "platinum") ||
			strings.Contains(lower, "gold") ||
			strings.Contains(lower, "classic") ||
			strings.Contains(lower, "signature") ||
			strings.Contains(lower, "infinite") ||
			strings.Contains(lower, "cashback") ||
			strings.Contains(lower, "evo") ||
			strings.Contains(lower, "digi") ||
			strings.Contains(lower, "online plus") ||
			strings.Contains(lower, "stepup") ||
			strings.Contains(lower, "step up")

		// Reject FAQ/article-like headings
		isFAQ := strings.Contains(lower, "là gì") || strings.Contains(lower, "la gi") ||
			strings.Contains(lower, "bao nhiêu") || strings.Contains(lower, "bao nhieu") ||
			strings.Contains(lower, "hỗ trợ") || strings.Contains(lower, "ho tro") ||
			strings.Contains(lower, "hữu ích") || strings.Contains(lower, "huu ich") ||
			strings.Contains(lower, "đăng ký") || strings.Contains(lower, "dang ky") ||
			strings.Contains(lower, "danh sách") || strings.Contains(lower, "danh sach") ||
			strings.Contains(lower, "chức năng") || strings.Contains(lower, "chuc nang") ||
			strings.Contains(lower, "tốt nhất") || strings.Contains(lower, "tot nhat") ||
			strings.Contains(lower, "ưu đãi thẻ") || strings.Contains(lower, "uu dai the")

		if isFAQ || !(hasNetwork || hasTier) {
			continue
		}

		id := slugify(bank + "_" + text)
		if seen[id] {
			continue
		}
		seen[id] = true

		// Try to extract fee and benefits from nearby text
		fee, cashback, rewards := extractCardDetails(h)

		card := CreditCard{
			ID:        id,
			Bank:      bank,
			Name:      text,
			Category:  guessCardCategory(text, cashback, rewards),
			AnnualFee: fee,
			Cashback:  cashback,
			Rewards:   rewards,
			SourceURL: url,
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// extractBankAndCard tries to split "Thẻ Techcombank Visa Platinum" into bank + card name.
var knownBanks = []string{
	"Techcombank", "VPBank", "TPBank", "ACB", "MB", "VIB", "Sacombank",
	"BIDV", "Vietcombank", "HDBank", "OCB", "SHB", "SCB", "MSB",
	"VietinBank", "HSBC", "Shinhan", "Standard Chartered", "Citibank",
	"Eximbank", "SeABank", "LienVietPostBank", "NCB", "PVcomBank",
	"VietABank", "ABBank", "BacABank", "KienlongBank", "NamABank",
}

func extractBankAndCard(title string) (bank, name string) {
	for _, b := range knownBanks {
		if strings.Contains(strings.ToLower(title), strings.ToLower(b)) {
			bank = b
			// Remove bank name and common prefixes to get card name
			name = title
			name = regexp.MustCompile(`(?i)thẻ\s+(tín dụng\s+)?`).ReplaceAllString(name, "")
			name = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(b)+`\s*`).ReplaceAllString(name, "")
			name = strings.TrimSpace(name)
			if name == "" {
				name = title
			}
			return bank, b + " " + name
		}
	}
	return "", ""
}

// extractCardDetails looks at siblings and nearby elements for fee/benefit info.
func extractCardDetails(heading *html.Node) (fee, cashback, rewards string) {
	// Walk next siblings looking for text with keywords
	for sib := heading.NextSibling; sib != nil; sib = sib.NextSibling {
		if sib.Type != html.ElementNode {
			continue
		}
		text := strings.ToLower(textContent(sib))
		if fee == "" && (strings.Contains(text, "phí") || strings.Contains(text, "phi") ||
			strings.Contains(text, "thường niên") || strings.Contains(text, "thuong nien")) {
			fee = strings.TrimSpace(textContent(sib))
		}
		if cashback == "" && (strings.Contains(text, "cashback") || strings.Contains(text, "hoàn tiền") ||
			strings.Contains(text, "hoan tien") || strings.Contains(text, "hoàn") || strings.Contains(text, "%")) {
			cashback = strings.TrimSpace(textContent(sib))
		}
		if rewards == "" && (strings.Contains(text, "tích điểm") || strings.Contains(text, "tich diem") ||
			strings.Contains(text, "reward") || strings.Contains(text, "ưu đãi") || strings.Contains(text, "dặm")) {
			rewards = strings.TrimSpace(textContent(sib))
		}
		// Stop after a few siblings
		if fee != "" && cashback != "" && rewards != "" {
			break
		}
	}
	return
}

func guessCardCategory(name, cashback, rewards string) string {
	lower := strings.ToLower(name + " " + cashback + " " + rewards)
	switch {
	case strings.Contains(lower, "dặm") || strings.Contains(lower, "dam") ||
		strings.Contains(lower, "miles") || strings.Contains(lower, "lotusmiles") ||
		strings.Contains(lower, "airline"):
		return "miles"
	case strings.Contains(lower, "cashback") || strings.Contains(lower, "hoàn tiền") ||
		strings.Contains(lower, "hoan tien"):
		return "cashback"
	case strings.Contains(lower, "miễn phí") || strings.Contains(lower, "mien phi") ||
		strings.Contains(lower, "free"):
		return "free"
	case strings.Contains(lower, "platinum") || strings.Contains(lower, "signature") ||
		strings.Contains(lower, "infinite") || strings.Contains(lower, "world"):
		return "premium"
	default:
		return "general"
	}
}

func deduplicateCards(cards []CreditCard) []CreditCard {
	seen := map[string]bool{}
	var unique []CreditCard
	for _, c := range cards {
		if !seen[c.ID] {
			seen[c.ID] = true
			unique = append(unique, c)
		}
	}
	return unique
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	// Transliterate common Vietnamese chars
	replacer := strings.NewReplacer(
		"ă", "a", "â", "a", "đ", "d", "ê", "e", "ô", "o", "ơ", "o", "ư", "u",
		"á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ắ", "a", "ằ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"ấ", "a", "ầ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"é", "e", "è", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ế", "e", "ề", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"í", "i", "ì", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"ó", "o", "ò", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ố", "o", "ồ", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ớ", "o", "ờ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"ú", "u", "ù", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ứ", "u", "ừ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"ý", "y", "ỳ", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
	)
	s = replacer.Replace(s)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
