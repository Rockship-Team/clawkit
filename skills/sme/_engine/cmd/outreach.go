package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// cmdOutreach dispatches LinkedIn outreach-tracking subcommands.
//
//	sme-cli outreach sync                          # scrape LinkedIn via CDP, record events
//	sme-cli outreach log-event --event-type X ...  # manual activity log
//	sme-cli outreach today                         # today's event counts
//	sme-cli outreach funnel [--days N]             # event counts over a range
//	sme-cli outreach pending                        # received invitations awaiting a decision
//	sme-cli outreach list [--event-type X] [--days N] [--limit N]  # raw event rows (name, note, time)
//	sme-cli outreach reply-context [--limit N]     # unclassified replies, for hand-off to sme-engagement
//	sme-cli outreach log-classification --event-id X --intent I --sentiment S --objection O [--contact-id ID] [--note N]
//	sme-cli outreach classified [--days N]         # replies already classified
//	sme-cli outreach stale [--days N]              # derived: connected-no-message / sent-no-reply / due-follow-up
func cmdOutreach(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: outreach sync|log-event|today|funnel|pending|list|reply-context|log-classification|classified|stale")
		os.Exit(1)
	}
	ensureOutreachTable()
	ensureOutreachClassificationTable()
	switch args[0] {
	case "sync":
		outreachSync(args[1:])
	case "log-event":
		outreachLogEvent(args[1:])
	case "today":
		outreachToday(args[1:])
	case "funnel":
		outreachFunnel(args[1:])
	case "pending":
		outreachPending(args[1:])
	case "list":
		outreachList(args[1:])
	case "reply-context":
		outreachReplyContext(args[1:])
	case "log-classification":
		outreachLogClassification(args[1:])
	case "classified":
		outreachClassified(args[1:])
	case "stale":
		outreachStale(args[1:])
	default:
		errOut("unknown outreach command: " + args[0])
	}
}


func ensureOutreachTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS outreach_events (
		id          TEXT PRIMARY KEY,
		org_id      TEXT NOT NULL DEFAULT 'default',
		channel     TEXT NOT NULL,
		event_type  TEXT NOT NULL,
		profile_url TEXT,
		name        TEXT,
		headline    TEXT,
		note        TEXT,
		source      TEXT NOT NULL DEFAULT 'manual',
		fingerprint TEXT NOT NULL,
		occurred_at TEXT NOT NULL,
		created_at  TEXT NOT NULL,
		UNIQUE(org_id, fingerprint)
	)`)
	// Additive-only migration (Phase 2 execution gate): a nullable campaign_id
	// so a manually-executed LinkedIn action (Campaign channel=linkedin has no
	// automated send — see campaign.go) can be traced back to the campaign
	// that produced it. Never NOT NULL/default, never backfilled — existing
	// rows are untouched, no new table, no risk to old data.
	ensureColumn("outreach_events", "campaign_id", "TEXT")
}

// ensureColumn adds a nullable column to an existing table if it isn't
// already there — checked via PRAGMA table_info rather than relying on a
// driver-specific "duplicate column" error string, so this is safe to call
// on every startup.
func ensureColumn(table, column, sqlType string) {
	rows, err := queryRows(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return
	}
	for _, r := range rows {
		if fmt.Sprint(r["name"]) == column {
			return
		}
	}
	mustDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, sqlType))
}

// ensureOutreachClassificationTable holds the Engagement Unified Taxonomy
// result for a given reply event. This is NOT a second taxonomy — intent/
// sentiment/objection are validated against the exact same enums
// engagement/SKILL.md defines (see validIntent/validSentiment/validObjection
// below), so outreach.go stores the classification but never invents it.
func ensureOutreachClassificationTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS outreach_reply_classifications (
		id            TEXT PRIMARY KEY,
		org_id        TEXT NOT NULL DEFAULT 'default',
		event_id      TEXT NOT NULL,
		contact_id    TEXT,
		intent        TEXT NOT NULL,
		sentiment     TEXT NOT NULL,
		objection     TEXT NOT NULL,
		note          TEXT,
		classified_at TEXT NOT NULL,
		created_at    TEXT NOT NULL,
		UNIQUE(org_id, event_id)
	)`)
}

func outreachFingerprint(orgID, channel, eventType, key string) string {
	sum := sha256.Sum256([]byte(orgID + "|" + channel + "|" + eventType + "|" + key))
	return hex.EncodeToString(sum[:])[:24]
}

// --- Manual logging -------------------------------------------------------

func outreachLogEvent(args []string) {
	channel := "manual"
	eventType := ""
	name := ""
	headline := ""
	note := ""
	count := 1
	campaignID := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--channel":
			i++
			channel = args[i]
		case "--event-type":
			i++
			eventType = args[i]
		case "--name":
			i++
			name = args[i]
		case "--headline":
			i++
			headline = args[i]
		case "--note":
			i++
			note = args[i]
		case "--count":
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				count = n
			}
		case "--campaign-id":
			i++
			campaignID = args[i]
		}
	}

	if eventType == "" {
		errOut("cần --event-type (vd: cold_call, demo, meeting, event_attended, connection_accepted, ...)")
		return
	}

	orgID := defaultOrgID()
	now := vnNowISO()
	for n := 0; n < count; n++ {
		id := newID()
		_, err := mustDB().Exec(`
			INSERT INTO outreach_events
				(id, org_id, channel, event_type, profile_url, name, headline, note, source, fingerprint, occurred_at, created_at, campaign_id)
			VALUES (?, ?, ?, ?, '', ?, ?, ?, 'manual', ?, ?, ?, ?)
		`, id, orgID, channel, eventType, name, headline, note, id, now, now, nullableString(campaignID))
		if err != nil {
			errOut(err.Error())
			return
		}
	}

	okOut(map[string]interface{}{
		"channel": channel, "event_type": eventType, "count": count, "campaign_id": campaignID,
		"message": fmt.Sprintf("Đã ghi %d event '%s' (%s)", count, eventType, channel),
	})
}

// nullableString returns nil for an empty string so an optional column
// stores SQL NULL instead of an empty-string sentinel — keeps "no campaign
// linked" distinguishable from "linked to an empty-string id".
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// --- Reporting -------------------------------------------------------------

func outreachToday(args []string) {
	orgID := defaultOrgID()
	rows, err := queryRows(`
		SELECT channel, event_type, COUNT(*) as count
		FROM outreach_events
		WHERE org_id = ? AND substr(occurred_at, 1, 10) = ?
		GROUP BY channel, event_type
		ORDER BY channel, event_type
	`, orgID, vnToday())
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"date": vnToday(), "events": rows,
		"message": fmt.Sprintf("Hôm nay có %d loại hoạt động outreach", len(rows)),
	})
}

// outreachFunnelData is the data-returning core of `outreach funnel` —
// extracted so sme-analytics can reuse the exact same query (channel
// comparison / bottleneck detection must never duplicate this SQL or risk
// the two reports disagreeing on numbers).
func outreachFunnelData(days int) (rows []map[string]interface{}, since string, err error) {
	orgID := defaultOrgID()
	since = vnNow().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err = queryRows(`
		SELECT channel, event_type, COUNT(*) as count
		FROM outreach_events
		WHERE org_id = ? AND substr(occurred_at, 1, 10) >= ?
		GROUP BY channel, event_type
		ORDER BY channel, event_type
	`, orgID, since)
	return rows, since, err
}

func outreachFunnel(args []string) {
	days := 7
	for i := 0; i < len(args); i++ {
		if args[i] == "--days" && i+1 < len(args) {
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				days = n
			}
		}
	}
	rows, since, err := outreachFunnelData(days)
	if err != nil {
		errOut(err.Error())
		return
	}

	okOut(map[string]interface{}{
		"since": since, "days": days,
		"events": rows,
		"note":   "events là số đếm theo (channel, event_type). Chưa đủ dữ liệu accepted/reply để tính conversion rate tự động — xem outreach pending cho invitation đang chờ.",
		"message": fmt.Sprintf("%d ngày gần đây: %d loại hoạt động", days, len(rows)),
	})
}

func outreachPending(args []string) {
	orgID := defaultOrgID()
	rows, err := queryRows(`
		SELECT id, name, headline, profile_url, occurred_at
		FROM outreach_events
		WHERE org_id = ? AND event_type = 'connection_request_received'
		ORDER BY occurred_at DESC
		LIMIT 50
	`, orgID)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"pending_received_invitations": rows,
		"count":                        len(rows),
		"note":                         "Danh sách connection request người khác gửi cho mình, lấy từ lần sync gần nhất — chưa chắc còn pending thật sự nếu đã accept/ignore ngoài LinkedIn từ lần sync trước.",
	})
}

// outreachList returns the raw event rows (name, headline/note, time) — the
// thing to call whenever a user wants to see WHO, not just a count. Never
// answer a "list ai đã..." question from conversation memory of an earlier
// tool call; this command reflects the current DB state.
func outreachList(args []string) {
	eventType := ""
	days := 1
	limit := 100
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--event-type":
			i++
			eventType = args[i]
		case "--days":
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				days = n
			}
		case "--limit":
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				limit = n
			}
		}
	}

	orgID := defaultOrgID()
	since := vnNow().AddDate(0, 0, -days+1).Format("2006-01-02")

	q := `
		SELECT event_type, name, headline, note, profile_url, occurred_at
		FROM outreach_events
		WHERE org_id = ? AND substr(occurred_at, 1, 10) >= ?`
	qargs := []interface{}{orgID, since}
	if eventType != "" {
		q += ` AND event_type = ?`
		qargs = append(qargs, eventType)
	}
	q += ` ORDER BY occurred_at DESC LIMIT ?`
	qargs = append(qargs, limit)

	rows, err := queryRows(q, qargs...)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"events": rows, "count": len(rows),
		"since": since, "event_type_filter": eventType,
	})
}

// --- LinkedIn sync via CDP ---------------------------------------------------

type cdpTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type cdpEnvelope struct {
	ID     int             `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type cdpSession struct {
	conn   *websocket.Conn
	nextID int
}

func cdpDial(cdpURL string) (*cdpSession, error) {
	resp, err := http.Get(strings.TrimRight(cdpURL, "/") + "/json")
	if err != nil {
		return nil, fmt.Errorf("không kết nối được CDP endpoint %s: %w", cdpURL, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var targets []cdpTarget
	if err := json.Unmarshal(b, &targets); err != nil {
		return nil, fmt.Errorf("phản hồi CDP /json không hợp lệ: %w", err)
	}
	var page *cdpTarget
	for i := range targets {
		if targets[i].Type == "page" {
			page = &targets[i]
			break
		}
	}
	if page == nil {
		return nil, fmt.Errorf("không tìm thấy tab Chrome nào đang mở")
	}
	conn, _, err := websocket.DefaultDialer.Dial(page.WebSocketDebuggerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("kết nối WebSocket CDP thất bại: %w", err)
	}
	return &cdpSession{conn: conn}, nil
}

func (s *cdpSession) call(method string, params map[string]interface{}) (json.RawMessage, error) {
	s.nextID++
	id := s.nextID
	paramsJSON, _ := json.Marshal(params)
	req := cdpEnvelope{ID: id, Method: method, Params: paramsJSON}
	b, _ := json.Marshal(req)
	if err := s.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return nil, err
	}
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		var resp cdpEnvelope
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue // notification or reply to an earlier call
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("CDP error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func (s *cdpSession) navigateAndWait(url string, waitMs int) error {
	if _, err := s.call("Page.enable", nil); err != nil {
		return err
	}
	if _, err := s.call("Page.navigate", map[string]interface{}{"url": url}); err != nil {
		return err
	}
	_, err := s.call("Runtime.evaluate", map[string]interface{}{
		"expression":   fmt.Sprintf("new Promise(r => setTimeout(r, %d))", waitMs),
		"awaitPromise": true,
	})
	return err
}

func (s *cdpSession) evalString(expr string) (string, error) {
	result, err := s.call("Runtime.evaluate", map[string]interface{}{
		"expression":    expr,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return "", err
	}
	var wrapped struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &wrapped); err != nil {
		return "", fmt.Errorf("không parse được kết quả JS: %w", err)
	}
	if wrapped.ExceptionDetails != nil {
		return "", fmt.Errorf("JS exception: %s", wrapped.ExceptionDetails.Text)
	}
	return wrapped.Result.Value, nil
}

// linkedinCard is one connection-invitation card scraped off a My Network page.
type linkedinCard struct {
	Href  string   `json:"href"`
	Lines []string `json:"lines"`
}

// scrapeJS finds every profile-link anchor, walks up to the nearest ancestor
// whose text contains marker (e.g. "Withdraw" or "Accept") and is short enough
// to be a single card (not the whole page), and returns its text lines.
const scrapeJS = `(function(){
  function cardFor(a, marker){
    let el = a;
    for (let i=0;i<8 && el; i++){
      el = el.parentElement;
      if (el && el.innerText && el.innerText.includes(marker) && el.innerText.length < 500) return el;
    }
    return null;
  }
  const seen = new Set();
  const out = [];
  document.querySelectorAll('a[href*="/in/"]').forEach(function(a){
    if (seen.has(a.href)) return;
    const card = cardFor(a, %q);
    if (!card) return;
    seen.add(a.href);
    const lines = card.innerText.split("\n").map(function(s){return s.trim();}).filter(Boolean);
    out.push({href: a.href, lines: lines});
  });
  return JSON.stringify(out);
})()`

func (s *cdpSession) scrapeCards(marker string) ([]linkedinCard, error) {
	raw, err := s.evalString(fmt.Sprintf(scrapeJS, marker))
	if err != nil {
		return nil, err
	}
	var cards []linkedinCard
	if err := json.Unmarshal([]byte(raw), &cards); err != nil {
		return nil, fmt.Errorf("không parse được danh sách card: %w", err)
	}
	return cards, nil
}

// parseCard extracts name/headline from the raw text lines of one card.
// Lines are messy (duplicated name, "Sent X ago", "N other mutual connections",
// button labels) — this keeps only what's reliably identifiable.
func parseCard(lines []string) (name, headline string) {
	skip := func(l string) bool {
		lo := strings.ToLower(l)
		return strings.HasPrefix(lo, "sent ") ||
			l == "Withdraw" || l == "Accept" || l == "Ignore" ||
			strings.Contains(l, "mutual connection")
	}
	for _, l := range lines {
		if skip(l) {
			continue
		}
		if name == "" {
			name = l
			continue
		}
		if l == name {
			continue // LinkedIn duplicates the name line for a11y
		}
		headline = l
		break
	}
	return name, headline
}

// messagingKickoffJS reads the LinkedIn Messaging conversation list (left
// pane). Rows are virtualized — only ~10-20 are rendered at a time — so this
// scrolls the list's own scroll container
// (.msg-conversations-container__conversations-list) step by step,
// accumulating cards, until it hits a row with no recognizable date/time
// label at all, or the list stops growing.
//
// GOTCHA #1 (found 2026-09-16, root cause of "only 3 outreach counted on a
// day with dozens sent"): LinkedIn does NOT show the word "Yesterday" in this
// UI — non-today rows are labeled with an actual date like "Sep 15" (short
// month + day, no year). An earlier version matched only "H:MM AM/PM" (today)
// or literally "Yesterday", so on any sync where the very first/most-recent
// row was already from a prior day, EVERY row failed the check and the scrape
// stopped after round 1 with zero results. Now accepts both label formats.
//
// GOTCHA #2 (found same day): running this as a single blocking
// `Runtime.evaluate` call with awaitPromise:true reliably HANGS past 60s+
// once the widened date acceptance means dozens of rounds/cards get
// processed — root cause not fully isolated, but a non-blocking
// kickoff-then-poll pattern (this function stores progress on
// `window.__scrapeState` and returns immediately; the Go caller polls a
// separate tiny synchronous eval until `done`) reliably completes the same
// work in ~8s. Do not go back to a single blocking evalString call here.
const messagingKickoffJS = `(function(){
  window.__scrapeState = {done: false};
  (async function(){
    const scroller = document.querySelector('.msg-conversations-container__conversations-list');
    const timeRe = /^\d{1,2}:\d{2}\s*(AM|PM)$/;
    const dateRe = /^([A-Z][a-z]{2})\s+(\d{1,2})$/;
    const now = new Date();
    const cutoffMs = now.getTime() - %d * 24 * 60 * 60 * 1000;
    // "Mon D" carries no year, so a naive Date parse of an old label can
    // silently roll to the WRONG year in either direction — resolve it the
    // same "assume this year, roll back one if that lands in the future" way
    // the Go side does (resolveCardDate), so the two cutoffs agree instead
    // of one trusting the other blindly.
    function resolveMs(label){
      const m = dateRe.exec(label);
      if (!m) return null;
      let d = new Date(now.getFullYear(), new Date(m[1] + ' 1, 2000').getMonth(), parseInt(m[2], 10));
      if (d.getTime() > now.getTime()) d.setFullYear(d.getFullYear() - 1);
      return d.getTime();
    }
    const seen = new Map();
    let hitOld = false, stableRounds = 0, lastSize = -1;
    for (let round = 0; round < 40 && !hitOld; round++) {
      document.querySelectorAll('.msg-conversation-listitem').forEach(function(card){
        const lines = card.innerText.split("\n").map(function(s){return s.trim();}).filter(Boolean);
        if (lines.length === 0) return;
        const key = lines.slice(0, 2).join("|");
        if (seen.has(key)) return;
        const label = lines.find(function(l){ return timeRe.test(l) || dateRe.test(l); });
        if (!label) { hitOld = true; return; }
        if (dateRe.test(label)) {
          const ms = resolveMs(label);
          if (ms === null || ms < cutoffMs) { hitOld = true; return; }
        }
        seen.set(key, lines);
      });
      if (seen.size === lastSize) { stableRounds++; if (stableRounds >= 3) break; } else { stableRounds = 0; }
      lastSize = seen.size;
      if (scroller) scroller.scrollTop += scroller.clientHeight;
      await new Promise(function(r){ setTimeout(r, 350); });
    }
    window.__scrapeState = {done: true, cards: Array.from(seen.values())};
  })();
  return "kicked off";
})()`

// scrapeMessagingCards kicks off messagingKickoffJS (fire-and-forget — see
// GOTCHA #2 above for why this must NOT be a single blocking eval) and polls
// a tiny synchronous expression every 500ms until window.__scrapeState.done,
// up to ~15s — comfortably above the ~8s this normally takes, while still
// leaving headroom under the 30s outer `timeout` wrapping the whole sync.
func (s *cdpSession) scrapeMessagingCards() ([][]string, error) {
	if _, err := s.evalString(fmt.Sprintf(messagingKickoffJS, maxMessageLookbackDays)); err != nil {
		return nil, err
	}
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		status, err := s.evalString(`JSON.stringify(!!(window.__scrapeState && window.__scrapeState.done))`)
		if err != nil {
			return nil, err
		}
		if status == "true" {
			break
		}
	}
	raw, err := s.evalString(`JSON.stringify((window.__scrapeState && window.__scrapeState.cards) || [])`)
	if err != nil {
		return nil, err
	}
	var cards [][]string
	if err := json.Unmarshal([]byte(raw), &cards); err != nil {
		return nil, fmt.Errorf("không parse được danh sách hội thoại: %w", err)
	}
	return cards, nil
}

var msgTimePattern = regexp.MustCompile(`^\d{1,2}:\d{2}\s*(AM|PM)$`)
var msgDateLabelPattern = regexp.MustCompile(`^[A-Z][a-z]{2}\s+\d{1,2}$`)

// maxMessageLookbackDays bounds resolveCardDate — a safety net for stragglers
// missed by the last sync, NOT a full inbox export. 4 days comfortably covers
// the worst real gap (Friday evening -> Monday morning over a weekend the
// Saturday check already narrows) plus a day of slack for a missed sync.
const maxMessageLookbackDays = 4

// resolveCardDate scans a card's lines for a recognizable LinkedIn label —
// either "H:MM AM/PM" (today) or "Mon D" (an explicit past date, LinkedIn's
// real format for anything not today — it does NOT show the word
// "Yesterday") — and resolves it to a concrete YYYY-MM-DD. "Mon D" has no
// year, so this assumes the current year and rolls back one year if that
// would land in the future (handles the Dec->Jan boundary). Returns
// ok=false if no label matched or the resolved date is further back than
// maxMessageLookbackDays, so a genuinely stale row still gets dropped rather
// than silently mis-dated.
func resolveCardDate(lines []string, now time.Time) (dateStr string, ok bool) {
	for _, l := range lines {
		if msgTimePattern.MatchString(l) {
			return now.Format("2006-01-02"), true
		}
		if msgDateLabelPattern.MatchString(l) {
			parsed, err := time.Parse("Jan 2", l)
			if err != nil {
				continue
			}
			candidate := time.Date(now.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location())
			if candidate.After(now) {
				candidate = candidate.AddDate(-1, 0, 0)
			}
			if now.Sub(candidate) > maxMessageLookbackDays*24*time.Hour {
				return "", false
			}
			return candidate.Format("2006-01-02"), true
		}
	}
	return "", false
}

// parseMessageCard extracts the participant name and last-message snippet from
// one conversation-list row. A "You:" prefix on the snippet means the last
// message was sent by us (outbound); its absence means they replied.
func parseMessageCard(lines []string) (name, snippet string, outbound bool) {
	skip := func(l string) bool {
		return strings.HasPrefix(l, "Status is") ||
			strings.HasPrefix(l, ". Press return") ||
			strings.HasPrefix(l, "Open the options list") ||
			msgTimePattern.MatchString(l) ||
			msgDateLabelPattern.MatchString(l)
	}
	for _, l := range lines {
		if skip(l) {
			continue
		}
		if name == "" {
			name = l
			continue
		}
		if l == name {
			continue
		}
		snippet = l
		break
	}
	if strings.HasPrefix(snippet, "You:") {
		return name, strings.TrimSpace(strings.TrimPrefix(snippet, "You:")), true
	}
	return name, snippet, false
}

func outreachSync(args []string) {
	conn := loadConnections()
	cdpURL := conn.LinkedIn.CDPUrl
	if cdpURL == "" {
		errOut("chưa cấu hình linkedin.cdp_url — chạy: sme-cli config set linkedin.cdp_url http://<host>:<port>")
		return
	}

	sess, err := cdpDial(cdpURL)
	if err != nil {
		errOut("LinkedIn sync unavailable: " + err.Error())
		return
	}
	defer sess.conn.Close()

	orgID := defaultOrgID()
	now := vnNowISO()
	summary := map[string]int{}
	var errs []string

	sync := func(url, marker, eventType string) {
		if err := sess.navigateAndWait(url, 3500); err != nil {
			errs = append(errs, eventType+": "+err.Error())
			return
		}
		cards, err := sess.scrapeCards(marker)
		if err != nil {
			errs = append(errs, eventType+": "+err.Error())
			return
		}
		for _, c := range cards {
			name, headline := parseCard(c.Lines)
			fp := outreachFingerprint(orgID, "linkedin", eventType, c.Href)
			res, err := mustDB().Exec(`
				INSERT INTO outreach_events
					(id, org_id, channel, event_type, profile_url, name, headline, note, source, fingerprint, occurred_at, created_at)
				VALUES (?, ?, 'linkedin', ?, ?, ?, ?, '', 'sync', ?, ?, ?)
				ON CONFLICT(org_id, fingerprint) DO NOTHING
			`, newID(), orgID, eventType, c.Href, name, headline, fp, now, now)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			if n, _ := res.RowsAffected(); n > 0 {
				summary[eventType]++
			}
		}
	}

	sync("https://www.linkedin.com/mynetwork/invitation-manager/sent/", "Withdraw", "connection_request_sent")
	sync("https://www.linkedin.com/mynetwork/invitation-manager/", "Accept", "connection_request_received")

	// Messaging: no stable per-conversation ID is available from the list view
	// (LinkedIn's DOM ids are framework-generated and change every page load),
	// so fingerprint is scoped per participant PER DAY instead of forever —
	// a new "sent"/"reply" signal for the same person on a later day is a new event.
	if err := sess.navigateAndWait("https://www.linkedin.com/messaging/", 3500); err != nil {
		errs = append(errs, "messaging: "+err.Error())
	} else if cards, err := sess.scrapeMessagingCards(); err != nil {
		errs = append(errs, "messaging: "+err.Error())
	} else {
		nowT := vnNow()
		for _, lines := range cards {
			eventDate, ok := resolveCardDate(lines, nowT)
			if !ok {
				continue // no recognizable/in-bounds date label — skip, don't guess
			}
			name, snippet, outbound := parseMessageCard(lines)
			if name == "" {
				continue
			}
			// Attribute to the REAL calendar date LinkedIn labeled the card
			// with, not always "now" — a card scraped this morning but
			// labeled "Sep 15" really happened Sep 15, and must be
			// fingerprinted/dated as such or it silently vanishes into
			// "today" and throws off both dedup and day-level totals.
			occurredAt := now
			if eventDate != vnToday() {
				occurredAt = eventDate + "T23:59:59+07:00"
			}
			eventType := "message_reply_received"
			if outbound {
				eventType = "message_sent"
			}
			fp := outreachFingerprint(orgID, "linkedin", eventType, name+"|"+eventDate)
			res, err := mustDB().Exec(`
				INSERT INTO outreach_events
					(id, org_id, channel, event_type, profile_url, name, headline, note, source, fingerprint, occurred_at, created_at)
				VALUES (?, ?, 'linkedin', ?, '', ?, '', ?, 'sync', ?, ?, ?)
				ON CONFLICT(org_id, fingerprint) DO NOTHING
			`, newID(), orgID, eventType, name, snippet, fp, occurredAt, now)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			if n, _ := res.RowsAffected(); n > 0 {
				summary[eventType]++
			}
		}
	}

	conn.LinkedIn.LastSyncAt = now
	saveConnections(conn)

	out := map[string]interface{}{
		"ok": len(errs) == 0, "synced_at": now, "new_events": summary,
	}
	if len(errs) > 0 {
		out["errors"] = errs
		out["note"] = "Một phần sync lỗi — số liệu chỉ tính tới thời điểm sync thành công gần nhất, KHÔNG coi phần lỗi là 0."
	}
	jsonOut(out)
}

// --- LinkedIn reply → Engagement Unified Taxonomy pipeline ------------------
//
// outreach.go's job here is ONLY to detect/retrieve the reply and pass it +
// contact context onward, then persist whatever normalized result comes
// back. It never classifies the reply itself and never defines its own
// vocabulary — sme-engagement remains the single owner of the taxonomy
// (engagement/SKILL.md "UNIFIED TAXONOMY"). The three enums below exist here
// only to REJECT a value that doesn't match that taxonomy, not to redefine it.

var validIntents = map[string]bool{
	"interested": true, "requesting_info": true, "scheduling_meeting": true,
	"declining": true, "unclear": true,
}
var validSentiments = map[string]bool{"positive": true, "neutral": true, "negative": true}
var validObjections = map[string]bool{
	"none": true, "price": true, "timing": true, "authority": true, "trust": true, "other": true,
}

type replyClassification struct {
	EventID      string `json:"event_id"`
	ContactID    string `json:"contact_id,omitempty"`
	Intent       string `json:"intent"`
	Sentiment    string `json:"sentiment"`
	Objection    string `json:"objection"`
	Note         string `json:"note,omitempty"`
	ClassifiedAt string `json:"classified_at"`
}

// outreachReplyContext returns LinkedIn replies not yet classified, so the
// agent can resolve the contact (via the existing `sme-cli cosmo
// search-contact <name>` — reused as-is, not reimplemented here) and
// classify with sme-engagement's Unified Taxonomy, then write the result
// back via log-classification.
func outreachReplyContext(args []string) {
	limit := 20
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" && i+1 < len(args) {
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				limit = n
			}
		}
	}
	orgID := defaultOrgID()
	rows, err := queryRows(`
		SELECT e.id as event_id, e.name, e.note as snippet, e.occurred_at
		FROM outreach_events e
		LEFT JOIN outreach_reply_classifications c
			ON c.org_id = e.org_id AND c.event_id = e.id
		WHERE e.org_id = ? AND e.event_type = 'message_reply_received' AND c.id IS NULL
		ORDER BY e.occurred_at DESC
		LIMIT ?
	`, orgID, limit)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"unclassified_replies": rows,
		"count":                len(rows),
		"next_step": "Với mỗi reply: (1) resolve contact qua `sme-cli cosmo search-contact <name>`, " +
			"(2) phân loại theo Unified Taxonomy của sme-engagement (intent/sentiment/objection) — " +
			"KHÔNG tự định nghĩa vocab khác, (3) ghi kết quả qua `sme-cli outreach log-classification`.",
	})
}

// outreachLogClassification persists the Engagement-computed classification
// for one reply event. It validates against the Unified Taxonomy enums so
// this table can never silently drift into a second vocabulary.
func outreachLogClassification(args []string) {
	var eventID, contactID, intent, sentiment, objection, note string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--event-id":
			i++
			eventID = args[i]
		case "--contact-id":
			i++
			contactID = args[i]
		case "--intent":
			i++
			intent = args[i]
		case "--sentiment":
			i++
			sentiment = args[i]
		case "--objection":
			i++
			objection = args[i]
		case "--note":
			i++
			note = args[i]
		}
	}
	if eventID == "" {
		errOut("cần --event-id (lấy từ `outreach reply-context`)")
		return
	}
	if !validIntents[intent] {
		errOut("intent không hợp lệ — phải là 1 trong: interested, requesting_info, scheduling_meeting, declining, unclear (Unified Taxonomy của sme-engagement)")
		return
	}
	if !validSentiments[sentiment] {
		errOut("sentiment không hợp lệ — phải là 1 trong: positive, neutral, negative")
		return
	}
	if !validObjections[objection] {
		errOut("objection không hợp lệ — phải là 1 trong: none, price, timing, authority, trust, other")
		return
	}

	orgID := defaultOrgID()
	now := vnNowISO()
	id := newID()
	_, err := mustDB().Exec(`
		INSERT INTO outreach_reply_classifications
			(id, org_id, event_id, contact_id, intent, sentiment, objection, note, classified_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(org_id, event_id) DO UPDATE SET
			contact_id=excluded.contact_id, intent=excluded.intent, sentiment=excluded.sentiment,
			objection=excluded.objection, note=excluded.note, classified_at=excluded.classified_at
	`, id, orgID, eventID, contactID, intent, sentiment, objection, note, now, now)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{
		"event_id": eventID, "contact_id": contactID,
		"intent": intent, "sentiment": sentiment, "objection": objection,
		"message": "Đã ghi classification. Stage KHÔNG tự đổi — sme-opportunity đánh giá readiness riêng, KHÔNG auto-jump Proposal.",
	})
}

// outreachClassifiedData is the data-returning core of `outreach
// classified` — extracted so sme-goal's NBA (goal_nba.go) can reuse the
// exact same recent-classification query instead of a second copy.
func outreachClassifiedData(days int) (rows []map[string]interface{}, since string, err error) {
	orgID := defaultOrgID()
	since = vnNow().AddDate(0, 0, -days+1).Format("2006-01-02")
	rows, err = queryRows(`
		SELECT c.event_id, c.contact_id, c.intent, c.sentiment, c.objection, c.note, c.classified_at,
		       e.name, e.occurred_at
		FROM outreach_reply_classifications c
		JOIN outreach_events e ON e.org_id = c.org_id AND e.id = c.event_id
		WHERE c.org_id = ? AND substr(c.classified_at, 1, 10) >= ?
		ORDER BY c.classified_at DESC
	`, orgID, since)
	return rows, since, err
}

func outreachClassified(args []string) {
	days := 7
	for i := 0; i < len(args); i++ {
		if args[i] == "--days" && i+1 < len(args) {
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				days = n
			}
		}
	}
	rows, since, err := outreachClassifiedData(days)
	if err != nil {
		errOut(err.Error())
		return
	}
	okOut(map[string]interface{}{"classified": rows, "count": len(rows), "since": since})
}

// queryReplyClassificationsByContact is used by opportunity.go to fold
// LinkedIn-sourced classifications into the Opportunity view's evidence —
// the actual code-level link from Outreach → Engagement → Opportunity.
func queryReplyClassificationsByContact(orgID, contactID string) ([]replyClassification, error) {
	if contactID == "" {
		return nil, nil
	}
	rows, err := queryRows(`
		SELECT event_id, contact_id, intent, sentiment, objection, note, classified_at
		FROM outreach_reply_classifications
		WHERE org_id = ? AND contact_id = ?
		ORDER BY classified_at DESC
	`, orgID, contactID)
	if err != nil {
		return nil, err
	}
	var out []replyClassification
	for _, r := range rows {
		out = append(out, replyClassification{
			EventID:      fmt.Sprint(r["event_id"]),
			ContactID:    fmt.Sprint(r["contact_id"]),
			Intent:       fmt.Sprint(r["intent"]),
			Sentiment:    fmt.Sprint(r["sentiment"]),
			Objection:    fmt.Sprint(r["objection"]),
			Note:         fmt.Sprint(r["note"]),
			ClassifiedAt: fmt.Sprint(r["classified_at"]),
		})
	}
	return out, nil
}

// --- Stale / follow-up derived view -----------------------------------------
//
// Pure aggregation over outreach_events already collected by sync — no new
// events are ever created here, no schema beyond what ensureOutreachTable
// already defines.

type staleEntry struct {
	Name        string `json:"name"`
	State       string `json:"state"`
	LastEvent   string `json:"last_event_type"`
	OccurredAt  string `json:"occurred_at"`
	DaysSince   int    `json:"days_since"`
}

// classifyStaleState is the pure decision function behind outreachStale —
// separated out so it's testable without a database.
func classifyStaleState(latestType string, hasMessage bool, daysSinceLatest, thresholdDays int) string {
	switch {
	case latestType == "connection_request_sent" && !hasMessage && daysSinceLatest >= thresholdDays:
		return "connected_no_message"
	case latestType == "message_sent" && daysSinceLatest >= thresholdDays:
		return "sent_no_reply"
	case latestType == "message_reply_received" && daysSinceLatest >= thresholdDays:
		return "due_follow_up"
	default:
		return ""
	}
}

func outreachStale(args []string) {
	days := 3
	for i := 0; i < len(args); i++ {
		if args[i] == "--days" && i+1 < len(args) {
			i++
			n, err := strconv.Atoi(args[i])
			if err == nil && n > 0 {
				days = n
			}
		}
	}
	orgID := defaultOrgID()
	rows, err := queryRows(`
		SELECT name, event_type, occurred_at
		FROM outreach_events
		WHERE org_id = ? AND name IS NOT NULL AND name != ''
		ORDER BY name, occurred_at DESC
	`, orgID)
	if err != nil {
		errOut(err.Error())
		return
	}

	type nameState struct {
		latestType string
		latestAt   string
		hasMessage bool
	}
	byName := map[string]*nameState{}
	order := []string{}
	for _, r := range rows {
		name := fmt.Sprint(r["name"])
		eventType := fmt.Sprint(r["event_type"])
		occurredAt := fmt.Sprint(r["occurred_at"])
		st, ok := byName[name]
		if !ok {
			st = &nameState{}
			byName[name] = st
			order = append(order, name)
		}
		if st.latestAt == "" {
			st.latestType = eventType
			st.latestAt = occurredAt
		}
		if eventType == "message_sent" || eventType == "message_reply_received" {
			st.hasMessage = true
		}
	}

	var entries []staleEntry
	for _, name := range order {
		st := byName[name]
		since := daysSince(strings.Replace(st.latestAt, " ", "T", 1))
		if state := classifyStaleState(st.latestType, st.hasMessage, since, days); state != "" {
			entries = append(entries, staleEntry{Name: name, State: state, LastEvent: st.latestType, OccurredAt: st.latestAt, DaysSince: since})
		}
	}

	okOut(map[string]interface{}{
		"days_threshold": days,
		"stale":          entries,
		"count":          len(entries),
		"note":           "Derived từ outreach_events hiện có — KHÔNG tạo event mới, KHÔNG tự gửi tin nhắn. Chỉ đề xuất, chờ user quyết định.",
	})
}
