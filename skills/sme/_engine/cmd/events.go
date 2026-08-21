package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// cmdEvent dispatches event subcommands. Events are stored in the local
// SQLite database (events + event_registrations tables). Only attendee
// contacts are synced to COSMO CRM via the sme-crm gateway.
//
//	sme-cli event list [--filter upcoming|recent|all]
//	sme-cli event create --type <type> --title <t> --date <ISO> [--venue <v>] [--capacity <n>]
//	sme-cli event prep-checklist <event_id>
//	sme-cli event post-actions <event_id>
//	sme-cli event types
//	sme-cli event sync-luma                                (Phase 2)
//	sme-cli event create-survey <event_id>                 (Phase 3)
func cmdEvent(args []string) {
	if len(args) == 0 {
		errOut("usage: event list|create|prep-checklist|post-actions|types|sync-luma|create-survey")
		return
	}
	switch args[0] {
	case "list":
		eventList(args[1:])
	case "create":
		eventCreate(args[1:])
	case "prep-checklist":
		eventPrepChecklist(args[1:])
	case "post-actions":
		eventPostActions(args[1:])
	case "types":
		eventTypes()
	case "gen-content":
		eventGenContent(args[1:])
	case "save-links":
		eventSaveLinks(args[1:])
	case "set-payment-info":
		eventSetPaymentInfo(args[1:])
	case "process-registrations":
		eventProcessRegistrations(args[1:])
	case "register":
		eventRegister(args[1:])
	case "list-attendees":
		eventListAttendees(args[1:])
	case "confirm-payment":
		eventConfirmPayment(args[1:])
	case "check-in":
		eventCheckIn(args[1:])
	case "report":
		eventReport(args[1:])
	case "sync-luma":
		errOut("sync-luma not yet implemented — follows once LUMA_API_KEY flow is wired.")
	case "create-survey":
		errOut("create-survey not yet implemented — follows once Google Forms OAuth is wired.")
	default:
		errOut("unknown event command: " + args[0])
	}
}

// eventType defines a BD event archetype with prep + post-event guidance.
type eventType struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Emoji        string   `json:"emoji"`
	BestFor      string   `json:"best_for"`
	PrepTasks    []string `json:"prep_tasks"`    // 1-2 days before
	DayOfTasks   []string `json:"day_of_tasks"`  // on event day
	PostTasks    []string `json:"post_tasks"`    // after event
	SurveyPrompt string   `json:"survey_prompt"` // 1-line hint for feedback form
}

var eventTypes_ = []eventType{
	{
		ID: "workshop", Name: "Workshop (hands-on)", Emoji: "🎓",
		BestFor: "Hands-on training / deep-dive cho 10-30 nguoi tham gia",
		PrepTasks: []string{
			"Venue + AV: kiem tra mic, projector, extension cord",
			"Tai lieu: in handout, exercises, name tag",
			"Attendee list: confirm so luong, liet ke dietary preferences",
			"Facilitator: brief lai flow, timing per section, fallback plan",
			"Logistics: snack/drink, sign-in sheet",
		},
		DayOfTasks: []string{
			"Arrive 60p truoc de setup",
			"Test AV + rehearsal demo san",
			"Sign-in sheet ready + name tag de san",
		},
		PostTasks: []string{
			"Gui thank-you email kem slide + recording (<24h)",
			"Feedback form qua Google Forms",
			"Log attendee vao CRM + tag 'workshop_{event_id}'",
			"Video edit + share on social",
		},
		SurveyPrompt: "Danh gia workshop: noi dung, facilitator, logistic, NPS recommend",
	},
	{
		ID: "webinar", Name: "Webinar (online)", Emoji: "💻",
		BestFor: "Webinar online cho 50-500 nguoi, lead gen rong",
		PrepTasks: []string{
			"Platform (Zoom/GoToWebinar): test audio + video 1 ngay truoc",
			"Slides final: animations, transitions, demo recording backup",
			"Speaker rehearsal: Q&A prep, timing, hand-off giua speaker",
			"Reminder emails: gui 1 tuan / 1 ngay / 1h truoc",
			"Backup host: 1 nguoi moderate chat + tech support",
		},
		DayOfTasks: []string{
			"Joint 30p truoc + check tech",
			"Monitor chat, assign moderator",
			"Record session",
		},
		PostTasks: []string{
			"Gui recording link + deck (<24h)",
			"Feedback form qua Google Forms",
			"Log attendees vao CRM, tag 'webinar_{event_id}'",
			"Trigger nurture campaign qua sme-campaign: playbook webinar_follow_up",
		},
		SurveyPrompt: "Danh gia webinar: content relevance, speaker quality, length, would join again",
	},
	{
		ID: "networking", Name: "Networking Event", Emoji: "🤝",
		BestFor: "Gathering 30-100 nguoi, relationship building",
		PrepTasks: []string{
			"Venue: layout, F&B, AV cho 1-2 phut intro",
			"Name tags: in theo company ho bao khi register",
			"Intro script: 2 phut welcome + ice breaker",
			"Photographer: ban ngoai de capture candid shots",
			"Sign-in: co table de capture info + business card",
		},
		DayOfTasks: []string{
			"Check sign-in flow, name tag table",
			"Brief team de kich hoat chuyen chuyen thang ai",
			"Nho chu de vang de don dinh them cap moi",
		},
		PostTasks: []string{
			"Gui intro email cho tung contact moi (signal-led, mention cuoc gap cu the)",
			"Add contact vao CRM, tag 'networking_{event_id}'",
			"Photos share vao Telegram/Slack group co link lu-ma",
			"Follow-up sequence qua sme-campaign cho hot leads",
		},
		SurveyPrompt: "Danh gia networking: venue, attendee quality, useful connections",
	},
	{
		ID: "demo-day", Name: "Demo Day (sales)", Emoji: "🎯",
		BestFor: "Demo cho 5-10 prospect, close-oriented",
		PrepTasks: []string{
			"Demo environment: moi tester chay 1 lan, backup screenshots",
			"Sales deck: customize theo industry cua tung prospect",
			"Prospect research: doc LinkedIn + funding + recent news",
			"Leave-behind: 1-page product sheet + pricing",
			"Q&A prep: 5 cau hoi kho nhat + counter-argument",
		},
		DayOfTasks: []string{
			"Test demo env 60p truoc",
			"Arrive som, chuan bi setup",
			"Record demo neu khach cho phep",
		},
		PostTasks: []string{
			"Follow-up email <24h: recap demo + next step",
			"Send proposal qua sme-proposal neu interested",
			"Log interaction + move business_stage sang QUALIFIED neu match",
			"Chuan bi demo 2 neu can",
		},
		SurveyPrompt: "Demo match nhu cau khong? Concern nao con ton? Next step mong muon?",
	},
	{
		ID: "conference-booth", Name: "Conference Booth", Emoji: "🏢",
		BestFor: "Booth o industry conference, lead capture rong",
		PrepTasks: []string{
			"Booth assets: ship roll-up, branding, demo laptop",
			"Talking points: 30s elevator pitch + 2p demo script",
			"Lead capture: Luma form / QR → link auto-add vao CRM",
			"Swag: stickers, T-shirts, notebook",
			"Team rotation: schedule ai staff booth, hour by hour",
		},
		DayOfTasks: []string{
			"Setup booth trong slot setup cua conference",
			"Scan QR hoac capture info cua ai ghe vao",
			"Categorize lead theo interest (hot / warm / cold) sau moi conversation",
		},
		PostTasks: []string{
			"Bulk import leads vao CRM qua sme-crm (delegate tu sme-cli event process-registrations)",
			"Batch follow-up campaign qua sme-campaign: playbook event_invite",
			"Phan loai theo interest level → uu tien outreach cho hot leads",
			"Debrief team: what worked, conversion expected",
		},
		SurveyPrompt: "N/A — khong survey attendees o booth; danh gia noi bo team hieu qua booth",
	},
	{
		ID: "internal-kickoff", Name: "Internal Kickoff", Emoji: "🎬",
		BestFor: "Kickoff project / quarter internal, 10-50 stakeholder",
		PrepTasks: []string{
			"Agenda: vision + milestones + owners + timeline",
			"Stakeholder list: confirm attendance, pre-read gi",
			"Action owners: assign truoc, khong cho on-the-fly",
			"Meeting room + conference cam cho remote",
			"Pre-read: slide tom tat context, gui 24h truoc",
		},
		DayOfTasks: []string{
			"Time-box agenda, co facilitator track time",
			"Scribe ghi decision + action items live",
			"Parking lot: cau hoi nao off-topic ghi lai de follow-up",
		},
		PostTasks: []string{
			"Decisions log + action items to owners trong 24h",
			"Retro schedule: book 30p check-in sau 2 tuan",
			"Update project tracking (Linear/Notion/etc.)",
		},
		SurveyPrompt: "Kickoff co clear? Owner assignment co doi? Con unclear gi khong?",
	},
}

func eventTypes() {
	okOut(map[string]interface{}{"types": eventTypes_})
}

func findEventType(id string) (eventType, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, t := range eventTypes_ {
		if t.ID == id {
			return t, true
		}
	}
	return eventType{}, false
}

// slugify makes a URL-friendly slug from an event title.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ', r == '-', r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// eventRowToMap converts a DB row from queryRows() into the JSON shape the
// campaign skill expects (mirroring the old COSMO /v1/events payload).
func eventRowToMap(row map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range row {
		out[k] = v
	}
	// Rebuild metadata from flat columns for compatibility with older code.
	md := map[string]interface{}{}
	if mdStr, ok := row["metadata"].(string); ok && mdStr != "" && mdStr != "{}" {
		_ = json.Unmarshal([]byte(mdStr), &md)
	}
	if v, ok := row["event_type_id"].(string); ok && v != "" {
		md["event_type_id"] = v
	}
	if v, ok := row["pricing_model"].(string); ok && v != "" {
		md["pricing_model"] = v
	}
	if v, ok := row["luma_event_title"].(string); ok && v != "" {
		md["luma_event_title"] = v
	}
	if v, ok := row["price_vnd"].(int64); ok && v > 0 {
		md["price_vnd"] = v
	}
	out["metadata"] = md
	// Expose luma_url / zoom_url under external_urls.
	urls := map[string]interface{}{}
	if v, ok := row["luma_url"].(string); ok && v != "" {
		urls["luma_url"] = v
	}
	if v, ok := row["zoom_url"].(string); ok && v != "" {
		urls["zoom_url"] = v
	}
	if len(urls) > 0 {
		out["external_urls"] = urls
	}
	// Registration count from event_registrations.
	if id, ok := row["id"].(string); ok && id != "" {
		count := 0
		mustDB().QueryRow(`SELECT COUNT(*) FROM event_registrations WHERE event_id=?`, id).Scan(&count)
		out["registration_count"] = count
	}
	return out
}

// eventList returns events split into upcoming/recent/all.
func eventList(args []string) {
	filter := "all"
	for i := 0; i < len(args); i++ {
		if args[i] == "--filter" && i+1 < len(args) {
			filter = args[i+1]
			i++
		}
	}
	orgID := defaultOrgID()
	rows, err := queryRows(`SELECT * FROM events WHERE org_id=? ORDER BY date DESC`, orgID)
	if err != nil {
		errOut("fetch events: " + err.Error())
	}
	now := vnNow()
	var upcoming, recent, other []map[string]interface{}
	for _, row := range rows {
		e := eventRowToMap(row)
		dateStr, _ := e["date"].(string)
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			other = append(other, e)
			continue
		}
		tICT := t.In(now.Location())
		switch {
		case tICT.After(now):
			e["_days_until"] = int(tICT.Sub(now).Hours() / 24)
			upcoming = append(upcoming, e)
		case now.Sub(tICT).Hours()/24 <= 7:
			e["_days_since"] = int(now.Sub(tICT).Hours() / 24)
			recent = append(recent, e)
		default:
			other = append(other, e)
		}
	}
	out := map[string]interface{}{
		"total":    len(rows),
		"upcoming": upcoming,
		"recent":   recent,
	}
	if filter == "all" {
		out["other"] = other
	}
	if filter == "upcoming" {
		delete(out, "recent")
	}
	if filter == "recent" {
		delete(out, "upcoming")
	}
	okOut(out)
}

// eventCreate inserts a new event into the local events table.
func eventCreate(args []string) {
	var typeID, title, date, venue, lumaURL, lumaTitle, pricing string
	capacity, priceVND := 0, 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				typeID = args[i+1]
				i++
			}
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--date":
			if i+1 < len(args) {
				date = args[i+1]
				i++
			}
		case "--venue":
			if i+1 < len(args) {
				venue = args[i+1]
				i++
			}
		case "--capacity":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &capacity)
				i++
			}
		case "--luma-url":
			if i+1 < len(args) {
				lumaURL = args[i+1]
				i++
			}
		case "--luma-title":
			if i+1 < len(args) {
				lumaTitle = args[i+1]
				i++
			}
		case "--pricing":
			if i+1 < len(args) {
				pricing = args[i+1] // "free" or "paid"
				i++
			}
		case "--price":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &priceVND)
				i++
			}
		}
	}
	if title == "" || date == "" {
		errOut("usage: event create --type <type> --title <t> --date <ISO_DATE> [--venue] [--capacity] [--luma-url] [--luma-title]")
	}
	if _, ok := findEventType(typeID); !ok {
		errOut(fmt.Sprintf("invalid --type %q — run `sme-cli event types` to see supported values", typeID))
	}

	if pricing == "" {
		switch strings.ToLower(typeID) {
		case "webinar":
			pricing = "free"
		case "workshop", "demo-day":
			pricing = "paid"
		default:
			pricing = "free"
		}
	}
	if lumaTitle == "" {
		lumaTitle = title
	}

	id := newID()
	orgID := defaultOrgID()
	slug := slugify(title)
	now := vnNowISO()

	_, err := exec(`INSERT INTO events (
		id, org_id, title, slug, event_type_id, date, venue, capacity, status,
		pricing_model, price_vnd, luma_url, luma_event_title, metadata, created_at, updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, orgID, title, slug, strings.ToLower(typeID), date, venue, capacity, "published",
		pricing, priceVND, lumaURL, lumaTitle, "{}", now, now)
	if err != nil {
		errOut("create event: " + err.Error())
	}

	created, _ := queryOne(`SELECT * FROM events WHERE id=?`, id)
	okOut(map[string]interface{}{
		"created":   eventRowToMap(created),
		"next_step": fmt.Sprintf("Run `sme-cli event prep-checklist %s` to see prep tasks for type %q.", id, typeID),
	})
}

// eventLoad reads an event by id from local SQLite. Returns (row, ok).
// Exits with error JSON if not found.
func eventLoad(eventID string) map[string]interface{} {
	orgID := defaultOrgID()
	row, err := queryOne(`SELECT * FROM events WHERE id=? AND org_id=?`, eventID, orgID)
	if err != nil {
		errOut("fetch event: " + err.Error())
	}
	if row == nil {
		errOut(fmt.Sprintf("event not found: %s", eventID))
	}
	return row
}

// eventPrepChecklist returns the prep + day-of tasks for a given event.
func eventPrepChecklist(args []string) {
	if len(args) == 0 {
		errOut("usage: event prep-checklist <event_id>")
	}
	eventID := args[0]
	row := eventLoad(eventID)
	typeID, _ := row["event_type_id"].(string)
	if typeID == "" {
		typeID = "workshop"
	}
	et, ok := findEventType(typeID)
	if !ok {
		errOut(fmt.Sprintf("unknown event_type_id %q on event", typeID))
	}
	okOut(map[string]interface{}{
		"event_id":     eventID,
		"event_title":  row["title"],
		"event_date":   row["date"],
		"event_type":   et,
		"prep_tasks":   et.PrepTasks,
		"day_of_tasks": et.DayOfTasks,
	})
}

// eventPostActions returns post-event action suggestions + campaign handoff.
func eventPostActions(args []string) {
	if len(args) == 0 {
		errOut("usage: event post-actions <event_id>")
	}
	eventID := args[0]
	row := eventLoad(eventID)
	typeID, _ := row["event_type_id"].(string)
	if typeID == "" {
		typeID = "workshop"
	}
	et, ok := findEventType(typeID)
	if !ok {
		errOut(fmt.Sprintf("unknown event_type_id %q on event", typeID))
	}
	playbook := "event_invite"
	switch typeID {
	case "webinar":
		playbook = "webinar_follow_up"
	case "workshop", "networking":
		playbook = "content_offering"
	}
	okOut(map[string]interface{}{
		"event_id":      eventID,
		"event_title":   row["title"],
		"post_tasks":    et.PostTasks,
		"survey_prompt": et.SurveyPrompt,
		"campaign_handoff": map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": playbook,
			"audience": fmt.Sprintf("All attendees of event %q", row["title"]),
			"command_hint": fmt.Sprintf(
				"Hand off to sme-campaign skill: create campaign for attendees with playbook %q. Audience fetched via sme-crm by tag event_%s_%s.",
				playbook, typeID, eventID),
		},
		"survey_handoff": map[string]interface{}{
			"command":         fmt.Sprintf("sme-cli event create-survey %s   (Phase 3, coming soon — uses Google Forms API)", eventID),
			"manual_fallback": "Until Phase 3 lands: create Google Form manually using `survey_prompt` as guide, then attach URL to event metadata.survey_url.",
		},
	})
}

// eventUpdate applies a partial UPDATE on the events table for the given
// whitelist of columns, filling updated_at. Returns the refreshed row.
func eventUpdate(eventID string, fields map[string]interface{}) map[string]interface{} {
	orgID := defaultOrgID()
	allowed := map[string]bool{
		"title":            true,
		"date":             true,
		"venue":            true,
		"capacity":         true,
		"status":           true,
		"pricing_model":    true,
		"price_vnd":        true,
		"luma_url":         true,
		"luma_event_title": true,
		"zoom_url":         true,
		"payment_info":     true,
		"thank_you_sent":   true,
		"metadata":         true,
	}
	var sets []string
	var args []interface{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		sets = append(sets, k+"=?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		row := eventLoad(eventID)
		return row
	}
	sets = append(sets, "updated_at=?")
	args = append(args, vnNowISO())
	args = append(args, eventID, orgID)
	q := fmt.Sprintf("UPDATE events SET %s WHERE id=? AND org_id=?", strings.Join(sets, ","))
	res, err := exec(q, args...)
	if err != nil {
		errOut("update event: " + err.Error())
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		errOut(fmt.Sprintf("event not found: %s", eventID))
	}
	row, _ := queryOne(`SELECT * FROM events WHERE id=?`, eventID)
	return row
}

// eventRegister inserts an attendee registration for an event and pushes
// the contact to CRM via the crm-gateway helper so it's tagged with
// source=event_<type>_<role> and source_id=<event_id>. Use for manual
// attendee adds outside the Luma sync path (e.g. walk-ins, direct signups).
//
//	sme-cli event register <event_id> --email <e> [--name <n>] [--paid]
//	                                  [--source manual|luma|walkin]
func eventRegister(args []string) {
	if len(args) == 0 {
		errOut("usage: event register <event_id> --email <e> [--name <n>] [--paid] [--source manual|luma|walkin]")
	}
	eventID := args[0]
	var email, name, sourceTag string
	paid := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--email":
			if i+1 < len(args) {
				email = strings.ToLower(strings.TrimSpace(args[i+1]))
				i++
			}
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--paid":
			paid = true
		case "--source":
			if i+1 < len(args) {
				sourceTag = args[i+1]
				i++
			}
		}
	}
	if email == "" {
		errOut("--email required")
	}
	if sourceTag == "" {
		sourceTag = "manual"
	}

	event := eventLoad(eventID)
	evTitle, _ := event["title"].(string)
	typeID, _ := event["event_type_id"].(string)
	pricing, _ := event["pricing_model"].(string)

	// Determine status: paid flag wins; else pricing model decides.
	// free event → "free", paid event → "pending" until confirm-payment.
	status := "pending"
	role := "registered"
	if paid {
		status = "confirmed"
		role = "paid"
	} else if pricing == "free" {
		status = "free"
		role = "registered"
	}

	orgID := defaultOrgID()
	now := vnNowISO()

	// Idempotency: if (event_id, email) already exists, update instead of insert.
	existing, _ := queryOne(`SELECT id, status, cosmo_contact_id FROM event_registrations WHERE event_id=? AND email=?`, eventID, email)
	var regID string
	if existing != nil {
		regID, _ = existing["id"].(string)
		sets := []string{"status=?", "updated_at=?"}
		vals := []interface{}{status, now}
		if paid {
			sets = append(sets, "payment_confirmed_at=?")
			vals = append(vals, now)
		}
		if name != "" {
			sets = append(sets, "name=?")
			vals = append(vals, name)
		}
		vals = append(vals, regID)
		exec(fmt.Sprintf(`UPDATE event_registrations SET %s WHERE id=?`, strings.Join(sets, ",")), vals...)
	} else {
		regID = newID()
		paidAt := interface{}(nil)
		if paid {
			paidAt = now
		}
		_, err := exec(`INSERT INTO event_registrations (
			id, org_id, event_id, email, name, status, source, registered_at, payment_confirmed_at, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			regID, orgID, eventID, email, name, status, sourceTag, now, paidAt, now, now)
		if err != nil {
			errOut("insert registration: " + err.Error())
		}
	}

	// Push to CRM.
	cosmoID, cerr := cosmoUpsertContactForEvent(name, email, eventID, evTitle, typeID, role)
	if cerr == nil && cosmoID != "" {
		exec(`UPDATE event_registrations SET cosmo_contact_id=?, updated_at=? WHERE id=?`, cosmoID, vnNowISO(), regID)
	}

	okOut(map[string]interface{}{
		"event_id":         eventID,
		"registration_id":  regID,
		"email":            email,
		"name":             name,
		"status":           status,
		"cosmo_contact_id": cosmoID,
		"crm_push_error":   errString(cerr),
	})
}

// eventListAttendees returns all registrations for an event from the
// local event_registrations table (source of truth).
//
//	sme-cli event list-attendees <event_id> [--status pending|confirmed|free|checked_in|no_show]
func eventListAttendees(args []string) {
	if len(args) == 0 {
		errOut("usage: event list-attendees <event_id> [--status <status>]")
	}
	eventID := args[0]
	statusFilter := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--status" && i+1 < len(args) {
			statusFilter = args[i+1]
			i++
		}
	}
	event := eventLoad(eventID)
	q := `SELECT id, email, name, phone, status, source, cosmo_contact_id, registered_at, payment_confirmed_at, checked_in_at FROM event_registrations WHERE event_id=?`
	qargs := []interface{}{eventID}
	if statusFilter != "" {
		q += ` AND status=?`
		qargs = append(qargs, statusFilter)
	}
	q += ` ORDER BY registered_at DESC`
	rows, err := queryRows(q, qargs...)
	if err != nil {
		errOut("fetch attendees: " + err.Error())
	}
	byStatus := map[string]int{}
	for _, r := range rows {
		if s, ok := r["status"].(string); ok {
			byStatus[s]++
		}
	}
	okOut(map[string]interface{}{
		"event_id":    eventID,
		"event_title": event["title"],
		"total":       len(rows),
		"by_status":   byStatus,
		"attendees":   rows,
	})
}

// eventCheckIn marks an attendee as checked in on event day.
//
//	sme-cli event check-in <event_id> --email <e>
func eventCheckIn(args []string) {
	if len(args) == 0 {
		errOut("usage: event check-in <event_id> --email <e>")
	}
	eventID := args[0]
	var email string
	for i := 1; i < len(args); i++ {
		if args[i] == "--email" && i+1 < len(args) {
			email = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		}
	}
	if email == "" {
		errOut("--email required")
	}
	now := vnNowISO()
	res, err := exec(`UPDATE event_registrations SET status='checked_in', checked_in_at=?, updated_at=? WHERE event_id=? AND email=?`, now, now, eventID, email)
	if err != nil {
		errOut("check-in: " + err.Error())
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		errOut(fmt.Sprintf("no registration found for %s at event %s", email, eventID))
	}
	okOut(map[string]interface{}{
		"event_id":      eventID,
		"email":         email,
		"checked_in_at": now,
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// avoid sql.ErrNoRows unused import when compiling if helpers are trimmed.
var _ = sql.ErrNoRows
