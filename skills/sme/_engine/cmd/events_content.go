package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// eventGenContent reads a markdown template for the event's type, fills
// placeholders with event metadata + user-provided vars, and prints the
// rendered content. Multi-section templates (event description, caption,
// email announcement) are returned as a single string so the caller can
// chunk / post manually.
//
//	sme-cli event gen-content <event_id> [--var key=value ...]
func eventGenContent(args []string) {
	if len(args) == 0 {
		errOut("usage: event gen-content <event_id> [--var key=value ...]")
	}
	eventID := args[0]
	overrides := map[string]string{}
	for i := 1; i < len(args); i++ {
		if args[i] == "--var" && i+1 < len(args) {
			kv := strings.SplitN(args[i+1], "=", 2)
			if len(kv) == 2 {
				overrides[kv[0]] = kv[1]
			}
			i++
		}
	}

	// Fetch event from COSMO
	raw, code, err := cosmoRequest("GET", "/v1/events/"+eventID, nil)
	if err != nil {
		errOut("fetch event: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("event fetch failed HTTP %d: %s", code, string(raw)))
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut("parse event: " + err.Error())
	}

	md, _ := resp.Data["metadata"].(map[string]interface{})
	typeID, _ := md["event_type_id"].(string)
	if typeID == "" {
		typeID = "workshop"
	}

	tpl, tplPath, err := loadEventTemplate(typeID)
	if err != nil {
		errOut(fmt.Sprintf("load template for type %q: %v — make sure references/templates/%s-default.md exists", typeID, err, typeID))
	}

	vars := buildEventVars(resp.Data, md, overrides)
	rendered, missing := fillEventTemplate(tpl, vars)

	out := map[string]interface{}{
		"event_id":         eventID,
		"event_type":       typeID,
		"template_path":    tplPath,
		"rendered_content": rendered,
	}
	if len(missing) > 0 {
		out["missing_placeholders"] = missing
		out["hint"] = "Some placeholders are still empty in the output. Run again with --var key=value for each missing key (or update event metadata and retry)."
	}
	okOut(out)
}

// loadEventTemplate returns the content of references/templates/<type>-default.md
// relative to the installed skill dir, or the repo root during dev.
func loadEventTemplate(typeID string) (string, string, error) {
	candidates := []string{
		// Installed location (~/.openclaw/workspace/skills/events/references/templates/)
		filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace", "skills", "events", "references", "templates", typeID+"-default.md"),
		filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace", "skills", "sme-events", "references", "templates", typeID+"-default.md"),
		// Dev / repo location
		filepath.Join("skills", "sme", "events", "references", "templates", typeID+"-default.md"),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			// Strip leading YAML frontmatter if present
			content := string(data)
			if strings.HasPrefix(content, "---\n") {
				if end := strings.Index(content[4:], "\n---"); end >= 0 {
					content = strings.TrimSpace(content[4+end+4:])
				}
			}
			return content, p, nil
		}
	}
	return "", "", fmt.Errorf("template not found in any of: %v", candidates)
}

// buildEventVars merges event fields, metadata, and user overrides into
// a single map used by fillEventTemplate.
func buildEventVars(event, md map[string]interface{}, overrides map[string]string) map[string]string {
	vars := map[string]string{}
	setIfString(vars, "title", event["title"])
	setIfString(vars, "date_ict", event["date"])
	setIfString(vars, "venue", event["venue"])
	setIfInt(vars, "capacity", event["capacity"])
	setIfString(vars, "time_display", event["time_display"])
	if md != nil {
		setIfString(vars, "duration", md["duration"])
		setIfString(vars, "audience", md["audience"])
		setIfString(vars, "takeaways", md["takeaways"])
		setIfString(vars, "agenda", md["agenda"])
		setIfString(vars, "what_to_bring", md["what_to_bring"])
		setIfString(vars, "speakers", md["speakers"])
		setIfString(vars, "zoom_url", md["zoom_url"])
		setIfString(vars, "luma_url", md["luma_url"])
		setIfInt(vars, "price_vnd", md["price_vnd"])
	}
	if urls, ok := event["external_urls"].(map[string]interface{}); ok {
		if v, ok := urls["luma_url"].(string); ok && v != "" && vars["luma_url"] == "" {
			vars["luma_url"] = v
		}
		if v, ok := urls["zoom_url"].(string); ok && v != "" && vars["zoom_url"] == "" {
			vars["zoom_url"] = v
		}
	}

	// Payment info from local sme-cli config (org.payment_info)
	c := loadConnections()
	if c.Org.PaymentInfo != "" {
		vars["payment_info"] = c.Org.PaymentInfo
	}
	// Brand name / signature defaults
	if c.Org.Name != "" {
		vars["brand_name"] = c.Org.Name
		vars["signature"] = "Best regards,\n\n" + c.Org.Name + " team"
	} else {
		vars["brand_name"] = "Rockship"
		vars["signature"] = "Best regards,\n\nRockship team"
	}

	// Derived fields (short versions for social captions)
	if vars["date_ict"] != "" {
		vars["date_short"] = deriveDateShort(vars["date_ict"])
		vars["time"] = deriveTime(vars["date_ict"])
	}
	if vars["title"] != "" {
		vars["title_short"] = deriveShortTitle(vars["title"])
	}
	if vars["takeaways"] != "" {
		vars["takeaways_short"] = deriveShortText(vars["takeaways"], 80)
	}

	// User overrides beat everything else
	for k, v := range overrides {
		vars[k] = v
	}
	return vars
}

func setIfString(vars map[string]string, key string, v interface{}) {
	if s, ok := v.(string); ok && s != "" {
		vars[key] = s
	}
}

func setIfInt(vars map[string]string, key string, v interface{}) {
	switch n := v.(type) {
	case float64:
		if n != 0 {
			vars[key] = fmt.Sprintf("%d", int64(n))
		}
	case int:
		if n != 0 {
			vars[key] = fmt.Sprintf("%d", n)
		}
	}
}

var placeholderPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// fillEventTemplate replaces {{key}} with vars[key]; placeholders with no
// matching var are left as {{key}} and returned in the missing slice so
// the caller can surface them.
func fillEventTemplate(tpl string, vars map[string]string) (string, []string) {
	missing := map[string]bool{}
	out := placeholderPattern.ReplaceAllStringFunc(tpl, func(match string) string {
		key := placeholderPattern.FindStringSubmatch(match)[1]
		if v, ok := vars[key]; ok && v != "" {
			return v
		}
		missing[key] = true
		return match // leave as-is so the user sees it needs filling
	})
	list := make([]string, 0, len(missing))
	for k := range missing {
		list = append(list, k)
	}
	return out, list
}

// deriveDateShort turns "2026-05-15T14:00:00+07:00" into "15/5".
func deriveDateShort(iso string) string {
	// Find `YYYY-MM-DDThh:mm`
	if len(iso) < 10 {
		return iso
	}
	// Parse month/day without pulling time.Parse failure surface.
	y := iso[:4]
	m := strings.TrimLeft(iso[5:7], "0")
	d := strings.TrimLeft(iso[8:10], "0")
	if y == "" || m == "" || d == "" {
		return iso[:10]
	}
	return d + "/" + m
}

func deriveTime(iso string) string {
	if len(iso) < 16 {
		return ""
	}
	return iso[11:16]
}

func deriveShortTitle(title string) string {
	if len(title) <= 30 {
		return title
	}
	return strings.TrimSpace(title[:30]) + "..."
}

func deriveShortText(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxChars {
		return text
	}
	return strings.TrimSpace(text[:maxChars]) + "..."
}

// eventSaveLinks patches event metadata with zoom_url and/or luma_url.
//
//	sme-cli event save-links <event_id> [--zoom URL] [--luma URL]
func eventSaveLinks(args []string) {
	if len(args) == 0 {
		errOut("usage: event save-links <event_id> [--zoom URL] [--luma URL]")
	}
	eventID := args[0]
	var zoomURL, lumaURL string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--zoom":
			if i+1 < len(args) {
				zoomURL = args[i+1]
				i++
			}
		case "--luma":
			if i+1 < len(args) {
				lumaURL = args[i+1]
				i++
			}
		}
	}
	if zoomURL == "" && lumaURL == "" {
		errOut("usage: event save-links <event_id> [--zoom URL] [--luma URL] — provide at least one of --zoom or --luma")
	}

	raw, code, err := cosmoRequest("GET", "/v1/events/"+eventID, nil)
	if err != nil {
		errOut("fetch event: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("event fetch failed HTTP %d: %s", code, string(raw)))
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut("parse event: " + err.Error())
	}
	md, _ := resp.Data["metadata"].(map[string]interface{})
	if md == nil {
		md = map[string]interface{}{}
	}
	if zoomURL != "" {
		md["zoom_url"] = zoomURL
	}
	if lumaURL != "" {
		md["luma_url"] = lumaURL
	}
	urls, _ := resp.Data["external_urls"].(map[string]interface{})
	if urls == nil {
		urls = map[string]interface{}{}
	}
	if zoomURL != "" {
		urls["zoom_url"] = zoomURL
	}
	if lumaURL != "" {
		urls["luma_url"] = lumaURL
	}

	patch := map[string]interface{}{
		"metadata":      md,
		"external_urls": urls,
	}
	body, err := json.Marshal(patch)
	if err != nil {
		errOut("encode patch: " + err.Error())
	}
	patchRaw, patchCode, err := cosmoRequest("PATCH", "/v1/events/"+eventID, body)
	if err != nil {
		errOut("patch event: " + err.Error())
	}
	if patchCode >= 400 {
		errOut(fmt.Sprintf("event patch failed HTTP %d: %s", patchCode, string(patchRaw)))
	}
	okOut(map[string]interface{}{
		"event_id": eventID,
		"zoom_url": zoomURL,
		"luma_url": lumaURL,
		"saved":    true,
	})
}

// eventSetPaymentInfo stores bank / account info in local sme-cli config
// (org.payment_info). Reused across all paid events.
//
//	sme-cli event set-payment-info "STK 1234567 - Vietcombank - Rockship"
func eventSetPaymentInfo(args []string) {
	if len(args) == 0 {
		errOut("usage: event set-payment-info \"<full bank instructions in one line>\"\n\nExample: event set-payment-info \"STK 1234567 - Vietcombank - Rockship / Nội dung: WORKSHOP {event}\"")
	}
	info := strings.Join(args, " ")
	c := loadConnections()
	c.Org.PaymentInfo = info
	if err := saveConnections(c); err != nil {
		errOut("save config: " + err.Error())
	}
	okOut(map[string]interface{}{
		"key":     "org.payment_info",
		"set":     true,
		"preview": info,
	})
}
