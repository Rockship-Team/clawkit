package main

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func looksLikeUUID(s string) bool { return uuidRegexp.MatchString(s) }

// cmdProposal dispatches proposal subcommands.
//
//	sme-cli proposal pricing
//	sme-cli proposal generate <company> <contact_id> <tier> <outline_file>
func cmdProposal(args []string) {
	if len(args) == 0 {
		errOut("usage: proposal pricing|generate")
		return
	}
	switch args[0] {
	case "pricing":
		proposalPricing()
	case "generate":
		proposalGenerate(args[1:])
	default:
		errOut("unknown proposal command: " + args[0])
	}
}

type proposalTier struct {
	Name                  string   `json:"name"`
	PriceVND              int64    `json:"price_vnd"`
	AgentsMax             int      `json:"agents_max,omitempty"`
	AgentsUnlimited       bool     `json:"agents_unlimited,omitempty"`
	TransactionsPerMonth  int      `json:"transactions_per_month,omitempty"`
	TransactionsUnlimited bool     `json:"transactions_unlimited,omitempty"`
	Features              []string `json:"features"`
	BestFor               string   `json:"best_for"`
}

var proposalTiers = []proposalTier{
	{
		Name:                 "Starter",
		PriceVND:             15_000_000,
		AgentsMax:            3,
		TransactionsPerMonth: 10_000,
		Features: []string{
			"Up to 3 AI agents",
			"10,000 transactions/month",
			"Standard pipeline automation",
			"Email + basic support",
		},
		BestFor: "Small teams piloting AI automation (<20 employees)",
	},
	{
		Name:                 "Pro",
		PriceVND:             400_000_000,
		AgentsMax:            10,
		TransactionsPerMonth: 100_000,
		Features: []string{
			"Up to 10 AI agents",
			"100,000 transactions/month",
			"Advanced workflow customization",
			"Priority support",
			"Dedicated onboarding session",
		},
		BestFor: "Growth-stage companies scaling operations (20-200 employees)",
	},
	{
		Name:                  "Enterprise",
		PriceVND:              800_000_000,
		AgentsUnlimited:       true,
		TransactionsUnlimited: true,
		Features: []string{
			"Unlimited AI agents",
			"Unlimited transactions",
			"Full custom integrations",
			"SLA & dedicated CSM",
			"On-prem / VPC deployment option",
		},
		BestFor: "Enterprises with complex multi-department workflows (200+ employees)",
	},
}

var proposalAddOns = []string{
	"Custom Vietnamese NLP model tuning",
	"Additional fine-tuning modules on customer data",
	"Extended BI / analytics integration",
}

func proposalPricing() {
	okOut(map[string]interface{}{
		"tiers":   proposalTiers,
		"add_ons": proposalAddOns,
		"rules": []string{
			"Only three tiers exist: Starter, Pro, Enterprise. Never invent new tiers (no 'Enterprise Plus', no 'Premium', no 'Custom').",
			"If client budget > Enterprise (800M VND/year), recommend Enterprise and flag add-ons for BD to quote separately.",
			"Never modify or round the listed prices.",
			"Discount policy: 2-year 10%, 3-year 15%, referral 5%, startup 20%.",
		},
	})
}

func validProposalTier(name string) (string, bool) {
	for _, t := range proposalTiers {
		if strings.EqualFold(t.Name, name) {
			return t.Name, true
		}
	}
	return "", false
}

// proposalGenerate builds a branded HTML proposal from an approved outline
// and renders it to PDF locally using whichever headless browser / PDF
// engine is available on the host (chromium > chrome > wkhtmltopdf).
// Returns { pdf_path, proposal, next_action }.
func proposalGenerate(args []string) {
	if len(args) < 4 {
		errOut("usage: proposal generate <company> <contact_id> <tier> <outline_file>")
	}
	company := args[0]
	contactID := args[1]
	tierIn := args[2]
	outlinePath := args[3]

	tier, ok := validProposalTier(tierIn)
	if !ok {
		errOut(fmt.Sprintf("invalid tier %q — only Starter, Pro, Enterprise are allowed. Do not invent tiers.", tierIn))
	}
	if strings.TrimSpace(contactID) == "" {
		errOut("contact_id is required — run `sme-cli cosmo search-contact <company>` first. If no match, create via `sme-cli cosmo create-contact`, then pass the returned id.")
	}
	if !looksLikeUUID(contactID) {
		errOut(fmt.Sprintf("contact_id %q must be a UUID from COSMO CRM — Step 1 CRM check is mandatory. Run `sme-cli cosmo search-contact <company>` first.", contactID))
	}

	outlineBytes, err := os.ReadFile(outlinePath)
	if err != nil {
		errOut("outline file not found: " + outlinePath)
	}
	if len(outlineBytes) < 200 {
		errOut(fmt.Sprintf("outline file too small (%d bytes) — provide a real approved outline, not a placeholder.", len(outlineBytes)))
	}

	html := buildProposalHTML(company, contactID, tier, string(outlineBytes))
	ts := time.Now().Format("20060102_150405")
	safeCo := sanitizeFilename(company)
	htmlPath := filepath.Join(os.TempDir(), fmt.Sprintf("proposal_%s_%s.html", safeCo, ts))
	pdfPath := filepath.Join(os.TempDir(), fmt.Sprintf("proposal_%s_%s.pdf", safeCo, ts))

	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		errOut("write html: " + err.Error())
	}
	engine, err := renderPDF(htmlPath, pdfPath)
	if err != nil {
		errOut(err.Error())
	}

	okOut(map[string]interface{}{
		"proposal": map[string]interface{}{
			"company":      company,
			"contact_id":   contactID,
			"tier":         tier,
			"outline_path": outlinePath,
		},
		"pdf_path":    pdfPath,
		"html_path":   htmlPath,
		"engine":      engine,
		"status":      "completed",
		"next_action": fmt.Sprintf("Attach %s to the user's chat, then: sme-cli cosmo log-interaction %s proposal_sent && sme-cli cosmo api PATCH /v1/contacts/%s '{\"business_stage\":\"PROPOSAL\"}'", pdfPath, contactID, contactID),
	})
}

// --- PDF rendering (chromium-first, fallback chain) ---

// renderPDF shells out to the first available engine and returns its name
// on success. No silent fallbacks beyond the chain — caller gets a single
// actionable error if none work.
func renderPDF(htmlPath, pdfPath string) (string, error) {
	absHTML, err := filepath.Abs(htmlPath)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	fileURL := "file://" + absHTML
	chromiumArgs := []string{
		"--headless=new", "--disable-gpu", "--no-sandbox",
		"--no-pdf-header-footer", "--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=3000",
		"--print-to-pdf=" + pdfPath, fileURL,
	}
	candidates := [][]string{
		append([]string{"chromium"}, chromiumArgs...),
		append([]string{"chromium-browser"}, chromiumArgs...),
		append([]string{"google-chrome"}, chromiumArgs...),
		append([]string{"google-chrome-stable"}, chromiumArgs...),
		{"wkhtmltopdf", "--quiet", htmlPath, pdfPath},
		{"pandoc", htmlPath, "-o", pdfPath},
	}
	var lastErr error
	for _, argv := range candidates {
		if _, err := osexec.LookPath(argv[0]); err != nil {
			continue
		}
		cmd := osexec.Command(argv[0], argv[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%s failed: %v — %s", argv[0], err, strings.TrimSpace(string(out)))
			continue
		}
		if info, err := os.Stat(pdfPath); err == nil && info.Size() > 0 {
			return argv[0], nil
		}
		lastErr = fmt.Errorf("%s produced no output", argv[0])
	}
	if lastErr != nil {
		return "", fmt.Errorf("no working PDF engine: %v", lastErr)
	}
	return "", fmt.Errorf("no PDF engine found on PATH — install chromium, wkhtmltopdf, or pandoc")
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "proposal"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "proposal"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// --- HTML builder ---

func buildProposalHTML(company, contactID, tier, outlineMD string) string {
	t, _ := findTier(tier)
	date := time.Now().Format("2006-01-02")
	body := mdToHTML(outlineMD)
	pricingBlock := renderPricingBlock(t)

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Proposal — ` + htmlEscape(company) + `</title>
<style>
  @page { size: A4; margin: 22mm 18mm; }
  body { font-family: -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
         color: #1f2937; line-height: 1.55; font-size: 11pt; }
  h1 { font-size: 22pt; margin: 0 0 4px; color: #0f172a; }
  h2 { font-size: 14pt; margin: 22px 0 8px; color: #0f172a; border-bottom: 2px solid #e5e7eb; padding-bottom: 4px; }
  h3 { font-size: 12pt; margin: 16px 0 6px; color: #334155; }
  p, li { margin: 4px 0; }
  ul, ol { margin: 6px 0 10px 20px; }
  code { font-family: Menlo, Consolas, monospace; background: #f3f4f6; padding: 1px 4px; border-radius: 3px; font-size: 10pt; }
  strong { color: #0f172a; }
  em { color: #374151; }
  .header { border-bottom: 3px solid #0f172a; padding-bottom: 12px; margin-bottom: 20px; }
  .meta { color: #6b7280; font-size: 10pt; margin-top: 6px; }
  .pricing { background: #f8fafc; border: 1px solid #cbd5e1; border-radius: 6px;
             padding: 14px 18px; margin: 20px 0; }
  .pricing .tier-name { font-size: 13pt; font-weight: 700; color: #0f172a; margin: 0 0 4px; }
  .pricing .price { font-size: 16pt; font-weight: 700; color: #0f766e; margin: 4px 0 10px; }
  .pricing ul { margin-left: 18px; }
  .footer { margin-top: 28px; padding-top: 12px; border-top: 1px solid #e5e7eb;
            color: #6b7280; font-size: 9pt; }
</style>
</head>
<body>
  <div class="header">
    <h1>Proposal for ` + htmlEscape(company) + `</h1>
    <div class="meta">Tier: <strong>` + htmlEscape(tier) + `</strong> · Date: ` + date + ` · Ref: ` + htmlEscape(contactID) + `</div>
  </div>

  ` + body + `

  ` + pricingBlock + `

  <div class="footer">
    Prepared by Rockship · ` + date + ` · Reference ` + htmlEscape(contactID) + `
  </div>
</body>
</html>`
}

func findTier(name string) (proposalTier, bool) {
	for _, t := range proposalTiers {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return proposalTier{}, false
}

func renderPricingBlock(t proposalTier) string {
	if t.Name == "" {
		return ""
	}
	feat := "<ul>"
	for _, f := range t.Features {
		feat += "<li>" + htmlEscape(f) + "</li>"
	}
	feat += "</ul>"
	return `<div class="pricing">
    <div class="tier-name">` + htmlEscape(t.Name) + ` Tier</div>
    <div class="price">` + formatVND(t.PriceVND) + ` VND / year</div>
    ` + feat + `
    <div style="color:#6b7280;font-size:10pt;">Best for: ` + htmlEscape(t.BestFor) + `</div>
  </div>`
}

func formatVND(n int64) string {
	s := fmt.Sprintf("%d", n)
	// insert thousands separator
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// --- Minimal markdown-to-HTML ---

// mdToHTML handles: headers, unordered lists, bold, italic, inline code,
// paragraphs. Enough for a proposal. Tables / images / nested blocks are
// out of scope — proposal outlines don't need them.
func mdToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var out strings.Builder
	inList := false
	var para []string

	flushPara := func() {
		if len(para) == 0 {
			return
		}
		text := strings.Join(para, " ")
		out.WriteString("<p>" + inlineMD(text) + "</p>\n")
		para = para[:0]
	}
	closeList := func() {
		if inList {
			out.WriteString("</ul>\n")
			inList = false
		}
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		trim := strings.TrimSpace(line)
		if trim == "" {
			flushPara()
			closeList()
			continue
		}
		// Headers
		if strings.HasPrefix(trim, "### ") {
			flushPara()
			closeList()
			out.WriteString("<h3>" + inlineMD(strings.TrimPrefix(trim, "### ")) + "</h3>\n")
			continue
		}
		if strings.HasPrefix(trim, "## ") {
			flushPara()
			closeList()
			out.WriteString("<h2>" + inlineMD(strings.TrimPrefix(trim, "## ")) + "</h2>\n")
			continue
		}
		if strings.HasPrefix(trim, "# ") {
			flushPara()
			closeList()
			out.WriteString("<h2>" + inlineMD(strings.TrimPrefix(trim, "# ")) + "</h2>\n")
			continue
		}
		// Unordered list
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") {
			flushPara()
			if !inList {
				out.WriteString("<ul>\n")
				inList = true
			}
			out.WriteString("<li>" + inlineMD(trim[2:]) + "</li>\n")
			continue
		}
		// Horizontal rule
		if trim == "---" || trim == "***" {
			flushPara()
			closeList()
			out.WriteString("<hr>\n")
			continue
		}
		closeList()
		para = append(para, trim)
	}
	flushPara()
	closeList()
	return out.String()
}

var (
	reBold   = regexp.MustCompile(`\*\*([^\*]+)\*\*`)
	reItalic = regexp.MustCompile(`\*([^\*]+)\*`)
	reCode   = regexp.MustCompile("`([^`]+)`")
)

func inlineMD(s string) string {
	s = htmlEscape(s)
	s = reBold.ReplaceAllString(s, "<strong>$1</strong>")
	s = reItalic.ReplaceAllString(s, "<em>$1</em>")
	s = reCode.ReplaceAllString(s, "<code>$1</code>")
	return s
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func htmlEscape(s string) string { return htmlEscaper.Replace(s) }
