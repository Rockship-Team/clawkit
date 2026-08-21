package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"regexp"
	"strings"
	"time"
)

// eventProcessRegistrations pulls new Luma registration emails from Gmail
// (via the gog CLI) and inserts them into the local event_registrations
// table. For free events, it also pushes each attendee into COSMO CRM via
// the crm-gateway helper (cosmoUpsertContactForEvent) so the contact is
// tagged with the event id and can be fetched later. For paid events,
// attendees start in status='pending' until eventConfirmPayment is run.
//
//	sme-cli event process-registrations <event_id> [--since ISO]
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

	event := eventLoad(eventID)
	eventMap := eventRowToMap(event)
	pricing, _ := event["pricing_model"].(string)
	if pricing == "" {
		pricing = "free"
	}
	typeID, _ := event["event_type_id"].(string)

	// lumaTitle is the subject-match key for this event's Luma notification
	// emails. Falls back to the event title for events created before the
	// luma_event_title field existed.
	lumaTitle, _ := event["luma_event_title"].(string)
	if lumaTitle == "" {
		lumaTitle, _ = event["title"].(string)
	}
	lumaTitleKey := strings.ToLower(strings.TrimSpace(lumaTitle))

	// Build Gmail query string. Luma sends from notifications@luma.com.
	query := "from:notifications@luma.com"
	if since != "" {
		query += " after:" + strings.ReplaceAll(strings.Split(since, "T")[0], "-", "/")
	} else {
		query += " newer_than:30d"
	}

	threads, err := gogGmailSearch(query, 100)
	if err != nil {
		errOut("gog gmail search: " + err.Error())
	}

	newRegistrants := []map[string]string{}
	skipped := 0
	skippedOtherEvent := 0
	for _, th := range threads {
		msgID := th.ID
		if msgID == "" {
			msgID, _ = th.RawID.(string)
		}
		if msgID == "" {
			skipped++
			continue
		}
		// Skip if already processed (raw_email_id recorded in event_registrations for this event).
		if registrationExistsByRawID(eventID, msgID) {
			skipped++
			continue
		}

		subject := th.Subject
		body := th.Snippet
		// If snippet is empty / truncated, pull full message body.
		if len(body) < 200 {
			if full, ferr := gogGmailGet(msgID); ferr == nil {
				if full.Subject != "" {
					subject = full.Subject
				}
				body = full.Body
			}
		}

		// Multi-event guard: subject/body must mention this event's Luma title.
		if lumaTitleKey != "" {
			if !strings.Contains(strings.ToLower(subject), lumaTitleKey) &&
				!strings.Contains(strings.ToLower(body), lumaTitleKey) {
				skippedOtherEvent++
				continue
			}
		}
		parsed := parseLumaRegistrant(subject, body)
		if parsed == nil {
			skipped++
			continue
		}
		parsed["gmail_message_id"] = msgID
		newRegistrants = append(newRegistrants, parsed)
	}

	result := map[string]interface{}{
		"event_id":            eventID,
		"event_title":         event["title"],
		"luma_title_filter":   lumaTitle,
		"pricing_model":       pricing,
		"emails_scanned":      len(threads),
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

	orgID := defaultOrgID()
	now := vnNowISO()
	createdContactIDs := []string{}
	initialStatus := "pending"
	if pricing == "free" {
		initialStatus = "free"
	}
	evTitle, _ := event["title"].(string)

	for _, r := range newRegistrants {
		regID := newID()
		_, execErr := exec(`INSERT INTO event_registrations (
			id, org_id, event_id, email, name, status, source, raw_email_id, registered_at, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			regID, orgID, eventID, r["email"], r["name"], initialStatus, "luma",
			r["gmail_message_id"], now, now, now)
		if execErr != nil {
			continue
		}
		if pricing == "free" {
			cosmoID, err := cosmoUpsertContactForEvent(r["name"], r["email"], eventID, evTitle, typeID, "registered")
			if err == nil && cosmoID != "" {
				createdContactIDs = append(createdContactIDs, cosmoID)
				exec(`UPDATE event_registrations SET cosmo_contact_id=?, updated_at=? WHERE id=?`, cosmoID, vnNowISO(), regID)
			}
		}
	}

	switch pricing {
	case "free":
		result["action_taken"] = "free event: inserted into event_registrations (status=free) + pushed to CRM with source tag"
		result["created_contact_ids"] = createdContactIDs
		result["campaign_handoff"] = map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"audience": fmt.Sprintf("%d new registrants of event %q (fetch via sme-crm by tag event_%s_%s)", len(newRegistrants), event["title"], typeID, eventID),
			"hint":     "Compose thank-you + reminder via sme-campaign. Audience is filterable by tag on sme-crm.",
		}
	case "paid":
		pendingCount := 0
		mustDB().QueryRow(`SELECT COUNT(*) FROM event_registrations WHERE event_id=? AND status='pending'`, eventID).Scan(&pendingCount)
		result["action_taken"] = "paid event: inserted into event_registrations (status=pending), waiting for confirm-payment"
		result["pending_count"] = pendingCount
		result["campaign_handoff"] = map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"variant":  "payment_request",
			"hint":     "Send confirmation-with-payment-instructions email using workshop template. Payment info from org.payment_info config.",
		}
	default:
		result["action_taken"] = fmt.Sprintf("unknown pricing_model %q — nothing done", pricing)
	}

	_ = eventMap
	okOut(result)
}

// registrationExistsByRawID returns true if a registration for (eventID,
// raw_email_id) already exists. Used for Gmail dedup across runs.
func registrationExistsByRawID(eventID, rawEmailID string) bool {
	if rawEmailID == "" {
		return false
	}
	var n int
	mustDB().QueryRow(`SELECT COUNT(*) FROM event_registrations WHERE event_id=? AND raw_email_id=?`, eventID, rawEmailID).Scan(&n)
	return n > 0
}

// gogGmailThread is a lightweight view of a Gmail thread/message from gog.
type gogGmailThread struct {
	ID      string      `json:"id"`
	RawID   interface{} `json:"messageId"`
	Subject string      `json:"subject"`
	From    string      `json:"from"`
	Snippet string      `json:"snippet"`
	Body    string      `json:"body"`
	Date    string      `json:"date"`
}

// gogGmailSearch runs `gog gmail search <query> --json --results-only` and
// returns the list of threads. Account is taken from cosmo.auth_email
// (shared login in this setup; replace with a dedicated gmail.account field
// if needed later).
func gogGmailSearch(query string, max int) ([]gogGmailThread, error) {
	acct := gogAccount()
	if acct == "" {
		return nil, fmt.Errorf("no gmail account configured (set cosmo.auth_email)")
	}
	if max <= 0 {
		max = 50
	}
	raw, err := gogRun([]string{"gmail", "search", query, "--account", acct, "--json", "--results-only", "--max", fmt.Sprintf("%d", max)})
	if err != nil {
		return nil, err
	}
	// Output may be an array or a {threads: [...]} object.
	var arr []gogGmailThread
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) >= 0 {
		return arr, nil
	}
	var envelope struct {
		Threads  []gogGmailThread `json:"threads"`
		Messages []gogGmailThread `json:"messages"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse gog output: %w; raw=%s", err, truncate(string(raw), 300))
	}
	if len(envelope.Threads) > 0 {
		return envelope.Threads, nil
	}
	return envelope.Messages, nil
}

// gogGmailGet fetches a single message with full body.
func gogGmailGet(msgID string) (*gogGmailThread, error) {
	acct := gogAccount()
	if acct == "" {
		return nil, fmt.Errorf("no gmail account configured")
	}
	raw, err := gogRun([]string{"gmail", "get", msgID, "--account", acct, "--json", "--results-only", "--format", "full"})
	if err != nil {
		return nil, err
	}
	var m gogGmailThread
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse gog get: %w", err)
	}
	return &m, nil
}

// gogRun shells out to the gog binary and returns stdout.
func gogRun(args []string) ([]byte, error) {
	cmd := osexec.Command("gog", args...)
	cmd.Env = append(os.Environ(), "GOG_NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gog %s: %w — stderr=%s", strings.Join(args, " "), err, truncate(stderr.String(), 300))
	}
	return stdout.Bytes(), nil
}

// gogAccount returns the Gmail account used for Luma email sync. For now
// this reuses cosmo.auth_email since the bot logs in with the same inbox.
func gogAccount() string {
	c := loadConnections()
	return c.COSMO.AuthEmail
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
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
		name = strings.SplitN(email, "@", 2)[0]
	}
	return map[string]string{"name": name, "email": email}
}

func currentQuarter() string {
	now := vnNow()
	q := (int(now.Month())-1)/3 + 1
	return fmt.Sprintf("q%d_%d", q, now.Year())
}

// eventConfirmPayment moves registrations from status='pending' to
// status='confirmed', sets payment_confirmed_at, and pushes each attendee
// into COSMO CRM via the crm gateway so they're tagged as paid for this
// event.
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

	event := eventLoad(eventID)
	evTitle, _ := event["title"].(string)
	typeID, _ := event["event_type_id"].(string)

	var confirmed, alreadyConfirmed, notFound []string
	createdContactIDs := []string{}

	for _, email := range emails {
		row, _ := queryOne(`SELECT id, name, status, cosmo_contact_id FROM event_registrations WHERE event_id=? AND email=?`, eventID, email)
		if row == nil {
			notFound = append(notFound, email)
			continue
		}
		status, _ := row["status"].(string)
		if status == "confirmed" {
			alreadyConfirmed = append(alreadyConfirmed, email)
			continue
		}
		regID, _ := row["id"].(string)
		name, _ := row["name"].(string)
		now := vnNowISO()
		_, err := exec(`UPDATE event_registrations SET status='confirmed', payment_confirmed_at=?, updated_at=? WHERE id=?`, now, now, regID)
		if err != nil {
			continue
		}
		confirmed = append(confirmed, email)
		cosmoID, cerr := cosmoUpsertContactForEvent(name, email, eventID, evTitle, typeID, "paid")
		if cerr == nil && cosmoID != "" {
			createdContactIDs = append(createdContactIDs, cosmoID)
			existing, _ := row["cosmo_contact_id"].(string)
			if existing == "" {
				exec(`UPDATE event_registrations SET cosmo_contact_id=?, updated_at=? WHERE id=?`, cosmoID, vnNowISO(), regID)
			}
		}
	}

	okOut(map[string]interface{}{
		"event_id":             eventID,
		"event_title":          evTitle,
		"confirmed_emails":     confirmed,
		"already_confirmed":    alreadyConfirmed,
		"not_found_in_pending": notFound,
		"added_to_crm":         createdContactIDs,
		"campaign_handoff": map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": "event_invite",
			"variant":  "paid_confirmation",
			"audience": fmt.Sprintf("%d confirmed-paid attendees for %q", len(confirmed), evTitle),
			"hint":     "Send confirmation email with Zoom link + venue details. Use event.zoom_url.",
		},
	})
}

// eventReport aggregates stats for an event — registration count by
// status, capacity usage, days-until, and recommended next actions.
//
//	sme-cli event report <event_id>
func eventReport(args []string) {
	if len(args) == 0 {
		errOut("usage: event report <event_id>")
	}
	eventID := args[0]
	event := eventLoad(eventID)

	// Count registrations by status
	statusCounts := map[string]int{}
	statusRows, _ := queryRows(`SELECT status, COUNT(*) AS cnt FROM event_registrations WHERE event_id=? GROUP BY status`, eventID)
	for _, r := range statusRows {
		st, _ := r["status"].(string)
		switch v := r["cnt"].(type) {
		case int64:
			statusCounts[st] = int(v)
		case int:
			statusCounts[st] = v
		}
	}

	capacity := 0
	switch c := event["capacity"].(type) {
	case int64:
		capacity = int(c)
	case int:
		capacity = c
	}
	pricing, _ := event["pricing_model"].(string)
	registeredTotal := 0
	for _, n := range statusCounts {
		registeredTotal += n
	}

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
		actions = append(actions, "Gửi reminder email lần 1 (trước 1 ngày) — chạy sme-campaign cho nhóm confirmed/free")
		actions = append(actions, "Chuẩn bị AV + tài liệu + facilitator brief (xem prep-checklist)")
	case daysUntil == 0:
		actions = append(actions, "Gửi reminder email lần 2 (1-2h trước event)")
		actions = append(actions, "Check Zoom link + venue setup")
	case daysUntil < 0 && daysUntil >= -3:
		actions = append(actions, "Gửi thank-you email + feedback form cho attendees")
		actions = append(actions, "Log attendance, post-mortem với team")
	}
	pendingCnt := statusCounts["pending"]
	if pricing == "paid" && pendingCnt > 0 {
		actions = append([]string{
			fmt.Sprintf("%d đăng ký đang đợi confirm payment — nhắc user confirm bằng `sme-cli event confirm-payment <id> --emails ...`", pendingCnt),
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
			"free":            statusCounts["free"],
			"pending_payment": statusCounts["pending"],
			"confirmed_paid":  statusCounts["confirmed"],
			"checked_in":      statusCounts["checked_in"],
			"no_show":         statusCounts["no_show"],
		},
		"capacity_used":       capacityUsed,
		"recommended_actions": actions,
	})
}
