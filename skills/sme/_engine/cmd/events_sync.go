package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// eventProcessRegistrations syncs new Luma registrants from Cosmo's
// /v2/emails/search into CRM + event metadata, branching on pricing:
//
//   - free (webinar): auto-add to CRM, tag event_<id>_registered,
//     mark ready for thank-you campaign.
//
//   - paid (workshop): queue into metadata.attendees.pending_payment;
//     wait for `event confirm-payment`.
//
//     sme-cli event process-registrations <event_id> [--since ISO]
func eventProcessRegistrations(args []string) {
	if len(args) == 0 {
		errOut("usage: event process-registrations <event_id> [--since ISO_DATE]")
	}
	eventID := args[0]
	since := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--since" && i+1 < len(args) {
			since = args[i+1]
			i++
		}
	}

	event, md, err := fetchEventWithMeta(eventID)
	if err != nil {
		errOut(err.Error())
	}
	pricing, _ := md["pricing_model"].(string)
	if pricing == "" {
		pricing = "free" // default
	}

	// lumaTitle is the subject-match key for this event's Luma notification
	// emails. Falls back to the event title for events created before the
	// luma_event_title field existed. Empty title = no filter (legacy behavior).
	lumaTitle, _ := md["luma_event_title"].(string)
	if lumaTitle == "" {
		lumaTitle, _ = event["title"].(string)
	}
	lumaTitleKey := strings.ToLower(strings.TrimSpace(lumaTitle))

	// Load previously processed gmail_message_ids from event metadata to dedupe.
	processedIDs := map[string]bool{}
	if prev, ok := md["processed_gmail_ids"].([]interface{}); ok {
		for _, v := range prev {
			if s, ok := v.(string); ok {
				processedIDs[s] = true
			}
		}
	}

	// Query Luma notification emails
	filter := map[string]interface{}{
		"from_email": "notifications@luma.com",
	}
	if since != "" {
		filter["since"] = since
	}
	searchBody, _ := json.Marshal(map[string]interface{}{
		"filter": filter,
		"limit":  100,
	})
	raw, code, err := cosmoRequest("POST", "/v2/emails/search", searchBody)
	if err != nil {
		errOut("search emails: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("email search failed HTTP %d: %s", code, string(raw)))
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity map[string]interface{} `json:"entity"`
			} `json:"list"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut("parse email search: " + err.Error())
	}

	newRegistrants := []map[string]string{}
	skipped := 0
	skippedOtherEvent := 0
	for _, item := range resp.Data.List {
		e := item.Entity
		if len(e) == 0 {
			e = item.Entity
		}
		gmailID, _ := e["gmail_message_id"].(string)
		if gmailID != "" && processedIDs[gmailID] {
			skipped++
			continue
		}
		subject, _ := e["subject"].(string)
		content, _ := e["content"].(string)
		// Multi-event guard: a Luma inbox can contain notifications for
		// several events. Only accept emails whose subject or body
		// mentions this event's luma title.
		if lumaTitleKey != "" {
			subjLower := strings.ToLower(subject)
			bodyLower := strings.ToLower(content)
			if !strings.Contains(subjLower, lumaTitleKey) && !strings.Contains(bodyLower, lumaTitleKey) {
				skippedOtherEvent++
				continue
			}
		}
		parsed := parseLumaRegistrant(subject, content)
		if parsed == nil {
			skipped++
			continue
		}
		parsed["gmail_message_id"] = gmailID
		newRegistrants = append(newRegistrants, parsed)
	}

	// Apply per pricing branch
	result := map[string]interface{}{
		"event_id":            eventID,
		"event_title":         event["title"],
		"luma_title_filter":   lumaTitle,
		"pricing_model":       pricing,
		"emails_scanned":      resp.Data.Total,
		"registrants_found":   len(newRegistrants),
		"skipped_duplicate":   skipped,
		"skipped_other_event": skippedOtherEvent,
		"action_taken":        "",
		"new_registrants":     newRegistrants,
		"campaign_handoff":    nil,
	}

	if len(newRegistrants) == 0 {
		result["action_taken"] = "no new registrants since last sync"
		okOut(result)
		return
	}

	// Update processed_gmail_ids so next run skips these
	for _, r := range newRegistrants {
		if id := r["gmail_message_id"]; id != "" {
			processedIDs[id] = true
		}
	}
	processedList := make([]string, 0, len(processedIDs))
	for id := range processedIDs {
		processedList = append(processedList, id)
	}
	md["processed_gmail_ids"] = processedList
	md["last_sync_at"] = time.Now().UTC().Format(time.RFC3339)

	switch pricing {
	case "free":
		createdIDs := applyFreeBranch(eventID, event, newRegistrants)
		result["action_taken"] = "free event: auto-added to CRM, ready for thank-you campaign"
		result["created_contact_ids"] = createdIDs
		result["campaign_handoff"] = map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"audience": fmt.Sprintf("%d new registrants of event %q", len(newRegistrants), event["title"]),
			"hint":     "Compose thank-you + reminder email via sme-campaign. Bot should call sme-cli cosmo api POST /v1/campaigns ... with these contact_ids.",
		}
	case "paid":
		pending := ensureAttendeesBucket(md, "pending_payment")
		for _, r := range newRegistrants {
			entry := map[string]interface{}{
				"name":             r["name"],
				"email":            r["email"],
				"registered_at":    time.Now().UTC().Format(time.RFC3339),
				"gmail_message_id": r["gmail_message_id"],
			}
			pending = append(pending, entry)
		}
		setAttendeesBucket(md, "pending_payment", pending)
		result["action_taken"] = "paid event: queued into pending_payment, waiting for confirm-payment"
		result["pending_count"] = len(pending)
		result["campaign_handoff"] = map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"variant":  "payment_request",
			"hint":     "Send confirmation-with-payment-instructions email using workshop template. Payment info from org.payment_info config.",
		}
	default:
		result["action_taken"] = fmt.Sprintf("unknown pricing_model %q — nothing done", pricing)
	}

	if err := patchEventMeta(eventID, md); err != nil {
		result["patch_error"] = err.Error()
	}
	okOut(result)
}

// parseLumaRegistrant extracts name + email from Luma notification emails.
// Luma uses several templates; we try a few patterns. Returns nil if we
// can't confidently extract both fields (safer than creating a half-empty
// CRM contact).
var (
	lumaNamePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)New guest\s+for[^:]*:\s*([^\r\n]+)`),
		regexp.MustCompile(`(?i)([A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+){0,3})\s+has\s+(registered|approved|joined)`),
		regexp.MustCompile(`(?i)Name:\s*([^\r\n<]+)`),
		regexp.MustCompile(`(?i)Guest:\s*([^\r\n<]+)`),
	}
	lumaEmailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

func parseLumaRegistrant(subject, content string) map[string]string {
	// Email — ignore notifications@luma.com itself
	var email string
	for _, m := range lumaEmailPattern.FindAllString(content, -1) {
		if strings.Contains(m, "luma.com") || strings.Contains(m, "no-reply") {
			continue
		}
		email = strings.ToLower(m)
		break
	}
	if email == "" {
		return nil
	}
	// Name — try subject first (often "New guest for <Event>: <Name>"),
	// then body patterns.
	var name string
	for _, re := range lumaNamePatterns {
		if m := re.FindStringSubmatch(subject); len(m) >= 2 {
			name = strings.TrimSpace(m[1])
			break
		}
		if m := re.FindStringSubmatch(content); len(m) >= 2 {
			name = strings.TrimSpace(m[1])
			break
		}
	}
	if name == "" {
		// Fallback: derive from email local-part
		name = strings.SplitN(email, "@", 2)[0]
	}
	return map[string]string{"name": name, "email": email}
}

// applyFreeBranch creates contacts in CRM for each registrant + tags.
// Returns created contact IDs.
func applyFreeBranch(eventID string, event map[string]interface{}, registrants []map[string]string) []string {
	var createdIDs []string
	tag := "event_" + eventID + "_registered"
	typeID, _ := event["metadata"].(map[string]interface{})["event_type_id"].(string)
	if typeID != "" {
		tag += "," + typeID + "_" + currentQuarter()
	}
	for _, r := range registrants {
		body, _ := json.Marshal(map[string]interface{}{
			"name":   r["name"],
			"email":  r["email"],
			"source": "event_" + eventID,
			"tags":   map[string]string{"event": tag},
		})
		raw, code, err := cosmoRequest("POST", "/v1/contacts", body)
		if err != nil || code >= 400 {
			continue
		}
		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue
		}
		if resp.Data.ID != "" {
			createdIDs = append(createdIDs, resp.Data.ID)
		}
	}
	return createdIDs
}

func currentQuarter() string {
	now := vnNow()
	q := (int(now.Month())-1)/3 + 1
	return fmt.Sprintf("q%d_%d", q, now.Year())
}

// eventConfirmPayment moves attendees from pending_payment to
// confirmed_paid and adds them to CRM if not already there.
//
//	sme-cli event confirm-payment <event_id> --emails a@x.vn,b@y.vn
func eventConfirmPayment(args []string) {
	if len(args) == 0 {
		errOut("usage: event confirm-payment <event_id> --emails a@x.vn,b@y.vn")
	}
	eventID := args[0]
	var emails []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--emails" && i+1 < len(args) {
			for _, e := range strings.Split(args[i+1], ",") {
				e = strings.TrimSpace(strings.ToLower(e))
				if e != "" {
					emails = append(emails, e)
				}
			}
			i++
		}
	}
	if len(emails) == 0 {
		errOut("--emails required (comma-separated list)")
	}

	event, md, err := fetchEventWithMeta(eventID)
	if err != nil {
		errOut(err.Error())
	}

	pending := ensureAttendeesBucket(md, "pending_payment")
	confirmed := ensureAttendeesBucket(md, "confirmed_paid")
	var movedEmails, notFoundEmails, alreadyConfirmed []string
	newPending := []interface{}{}

	emailSet := map[string]bool{}
	for _, e := range emails {
		emailSet[e] = true
	}
	confirmedSet := map[string]bool{}
	for _, a := range confirmed {
		if m, ok := a.(map[string]interface{}); ok {
			if e, ok := m["email"].(string); ok {
				confirmedSet[strings.ToLower(e)] = true
			}
		}
	}

	for _, a := range pending {
		m, ok := a.(map[string]interface{})
		if !ok {
			newPending = append(newPending, a)
			continue
		}
		email, _ := m["email"].(string)
		email = strings.ToLower(email)
		if !emailSet[email] {
			newPending = append(newPending, a)
			continue
		}
		if confirmedSet[email] {
			alreadyConfirmed = append(alreadyConfirmed, email)
			continue
		}
		m["confirmed_at"] = time.Now().UTC().Format(time.RFC3339)
		confirmed = append(confirmed, m)
		movedEmails = append(movedEmails, email)
	}

	for e := range emailSet {
		if !contains(movedEmails, e) && !contains(alreadyConfirmed, e) {
			notFoundEmails = append(notFoundEmails, e)
		}
	}

	setAttendeesBucket(md, "pending_payment", newPending)
	setAttendeesBucket(md, "confirmed_paid", confirmed)

	// Add to CRM
	createdIDs := []string{}
	tag := "event_" + eventID + "_paid"
	for _, e := range movedEmails {
		// find registrant info from confirmed list
		var info map[string]interface{}
		for _, a := range confirmed {
			if m, ok := a.(map[string]interface{}); ok {
				if em, _ := m["email"].(string); strings.EqualFold(em, e) {
					info = m
					break
				}
			}
		}
		if info == nil {
			continue
		}
		body, _ := json.Marshal(map[string]interface{}{
			"name":   info["name"],
			"email":  info["email"],
			"source": "event_" + eventID,
			"tags":   map[string]string{"event": tag},
		})
		raw, code, err := cosmoRequest("POST", "/v1/contacts", body)
		if err != nil || code >= 400 {
			continue
		}
		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue
		}
		if resp.Data.ID != "" {
			createdIDs = append(createdIDs, resp.Data.ID)
		}
	}

	if err := patchEventMeta(eventID, md); err != nil {
		errOut("patch event: " + err.Error())
	}

	okOut(map[string]interface{}{
		"event_id":             eventID,
		"event_title":          event["title"],
		"confirmed_emails":     movedEmails,
		"already_confirmed":    alreadyConfirmed,
		"not_found_in_pending": notFoundEmails,
		"added_to_crm":         createdIDs,
		"campaign_handoff": map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"variant":  "paid_confirmation",
			"audience": fmt.Sprintf("%d confirmed-paid attendees for %q", len(movedEmails), event["title"]),
			"hint":     "Send confirmation email with Zoom link + venue details. Use event metadata.zoom_url.",
		},
	})
}

// eventReport aggregates stats for an event — registration count, paid
// vs pending, capacity usage, days-until, and recommended next actions.
//
//	sme-cli event report <event_id>
func eventReport(args []string) {
	if len(args) == 0 {
		errOut("usage: event report <event_id>")
	}
	eventID := args[0]
	event, md, err := fetchEventWithMeta(eventID)
	if err != nil {
		errOut(err.Error())
	}

	pending := ensureAttendeesBucket(md, "pending_payment")
	confirmed := ensureAttendeesBucket(md, "confirmed_paid")
	freeReg := ensureAttendeesBucket(md, "free_registered")

	capacity := 0
	if c, ok := event["capacity"].(float64); ok {
		capacity = int(c)
	}
	pricing, _ := md["pricing_model"].(string)
	registeredTotal := len(pending) + len(confirmed) + len(freeReg)

	dateStr, _ := event["date"].(string)
	var daysUntil int
	var eventPhase string
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		daysUntil = int(time.Until(t).Hours() / 24)
		switch {
		case daysUntil < 0:
			eventPhase = fmt.Sprintf("post-event (%d ngày trước)", -daysUntil)
		case daysUntil == 0:
			eventPhase = "hôm nay"
		case daysUntil <= 3:
			eventPhase = fmt.Sprintf("%d ngày nữa — prep gấp", daysUntil)
		case daysUntil <= 7:
			eventPhase = fmt.Sprintf("%d ngày nữa — sẵn sàng", daysUntil)
		default:
			eventPhase = fmt.Sprintf("%d ngày nữa", daysUntil)
		}
	}

	var capacityUsed string
	if capacity > 0 {
		pct := float64(registeredTotal) * 100 / float64(capacity)
		capacityUsed = fmt.Sprintf("%d/%d (%.0f%%)", registeredTotal, capacity, pct)
	} else {
		capacityUsed = fmt.Sprintf("%d registered (capacity unset)", registeredTotal)
	}

	actions := []string{}
	switch {
	case daysUntil > 7:
		actions = append(actions, "Đang còn thời gian — push marketing/announcement nếu capacity <50%")
	case daysUntil >= 3 && daysUntil <= 7:
		actions = append(actions, "Kiểm tra lại đã publish Luma chưa, đẩy thêm social nếu capacity <70%")
	case daysUntil >= 1 && daysUntil < 3:
		actions = append(actions, "Gửi reminder email lần 1 (trước 1 ngày) — chạy sme-campaign cho nhóm confirmed/free_registered")
		actions = append(actions, "Chuẩn bị AV + tài liệu + facilitator brief (xem prep-checklist)")
	case daysUntil == 0:
		actions = append(actions, "Gửi reminder email lần 2 (1-2h trước event)")
		actions = append(actions, "Check Zoom link + venue setup")
	case daysUntil < 0 && daysUntil >= -3:
		actions = append(actions, "Gửi thank-you email + feedback form cho attendees")
		actions = append(actions, "Log attendance, post-mortem với team")
	}
	if pricing == "paid" && len(pending) > 0 {
		actions = append([]string{
			fmt.Sprintf("%d đăng ký đang đợi confirm payment — nhắc user confirm bằng `sme-cli event confirm-payment <id> --emails ...`", len(pending)),
		}, actions...)
	}

	okOut(map[string]interface{}{
		"event_id":      eventID,
		"event_title":   event["title"],
		"event_date":    dateStr,
		"event_phase":   eventPhase,
		"days_until":    daysUntil,
		"pricing_model": pricing,
		"registrations": map[string]interface{}{
			"total":           registeredTotal,
			"free_registered": len(freeReg),
			"pending_payment": len(pending),
			"confirmed_paid":  len(confirmed),
		},
		"capacity_used":       capacityUsed,
		"recommended_actions": actions,
	})
}

// --- helpers ---

func fetchEventWithMeta(eventID string) (event, md map[string]interface{}, err error) {
	raw, code, reqErr := cosmoRequest("GET", "/v1/events/"+eventID, nil)
	if reqErr != nil {
		return nil, nil, fmt.Errorf("fetch event: %w", reqErr)
	}
	if code >= 400 {
		return nil, nil, fmt.Errorf("event fetch HTTP %d: %s", code, string(raw))
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse event: %w", err)
	}
	event = resp.Data
	md, _ = event["metadata"].(map[string]interface{})
	if md == nil {
		md = map[string]interface{}{}
	}
	return event, md, nil
}

func patchEventMeta(eventID string, md map[string]interface{}) error {
	patch, _ := json.Marshal(map[string]interface{}{"metadata": md})
	raw, code, err := cosmoRequest("PATCH", "/v1/events/"+eventID, patch)
	if err != nil {
		return fmt.Errorf("patch event: %w", err)
	}
	if code >= 400 {
		return fmt.Errorf("patch HTTP %d: %s", code, string(raw))
	}
	return nil
}

func ensureAttendeesBucket(md map[string]interface{}, bucket string) []interface{} {
	att, _ := md["attendees"].(map[string]interface{})
	if att == nil {
		att = map[string]interface{}{}
		md["attendees"] = att
	}
	list, _ := att[bucket].([]interface{})
	return list
}

func setAttendeesBucket(md map[string]interface{}, bucket string, list []interface{}) {
	att, _ := md["attendees"].(map[string]interface{})
	if att == nil {
		att = map[string]interface{}{}
		md["attendees"] = att
	}
	att[bucket] = list
}

func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}
