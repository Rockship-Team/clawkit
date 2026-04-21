package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// cmdEvent dispatches event subcommands.
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
	case "sync-luma":
		errOut("sync-luma not yet implemented — follows in a later commit once LUMA_API_KEY flow is wired.")
	case "create-survey":
		errOut("create-survey not yet implemented — follows in a later commit once Google Forms OAuth is wired.")
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
			"Bulk import leads vao CRM qua sme-cli cosmo import-csv",
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

// eventList fetches /v1/events and returns upcoming/recent/all events.
func eventList(args []string) {
	filter := "all"
	for i := 0; i < len(args); i++ {
		if args[i] == "--filter" && i+1 < len(args) {
			filter = args[i+1]
			i++
		}
	}
	raw, code, err := cosmoRequest("GET", "/v1/events", nil)
	if err != nil {
		errOut("fetch events: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("events endpoint HTTP %d: %s", code, string(raw)))
	}
	var resp struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		errOut("parse events response: " + err.Error())
	}
	now := vnNow()
	var upcoming, recent, other []map[string]interface{}
	for _, e := range resp.Data {
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
		"total":    len(resp.Data),
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

// eventCreate POSTs a new event to /v1/events. Supports --type --title
// --date --venue --capacity --luma-url flags.
func eventCreate(args []string) {
	var typeID, title, date, venue, lumaURL string
	capacity := 0
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
		}
	}
	if title == "" || date == "" {
		errOut("usage: event create --type <type> --title <t> --date <ISO_DATE> [--venue] [--capacity] [--luma-url]")
	}
	if _, ok := findEventType(typeID); !ok {
		errOut(fmt.Sprintf("invalid --type %q — run `sme-cli event types` to see supported values", typeID))
	}

	payload := map[string]interface{}{
		"title":  title,
		"date":   date,
		"status": "published",
		"metadata": map[string]interface{}{
			"event_type_id": strings.ToLower(typeID),
		},
	}
	if venue != "" {
		payload["venue"] = venue
	}
	if capacity > 0 {
		payload["capacity"] = capacity
	}
	if lumaURL != "" {
		payload["external_urls"] = map[string]interface{}{"luma_url": lumaURL}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		errOut("encode payload: " + err.Error())
	}
	raw, code, err := cosmoRequest("POST", "/v1/events", body)
	if err != nil {
		errOut("create event: " + err.Error())
	}
	if code >= 400 {
		errOut(fmt.Sprintf("event create failed HTTP %d: %s", code, string(raw)))
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(raw, &resp)
	okOut(map[string]interface{}{
		"created":   resp,
		"next_step": fmt.Sprintf("Run `sme-cli event prep-checklist <event_id>` to see prep tasks for type %q.", typeID),
	})
}

// eventPrepChecklist returns the prep + day-of tasks for a given event.
// Reads the event's metadata.event_type_id, falls back to "workshop" if
// absent.
func eventPrepChecklist(args []string) {
	if len(args) == 0 {
		errOut("usage: event prep-checklist <event_id>")
	}
	eventID := args[0]
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
	et, ok := findEventType(typeID)
	if !ok {
		errOut(fmt.Sprintf("unknown event_type_id %q on event", typeID))
	}
	okOut(map[string]interface{}{
		"event_id":     eventID,
		"event_title":  resp.Data["title"],
		"event_date":   resp.Data["date"],
		"event_type":   et,
		"prep_tasks":   et.PrepTasks,
		"day_of_tasks": et.DayOfTasks,
	})
}

// eventPostActions returns post-event action suggestions + specific
// campaign handoff instructions.
func eventPostActions(args []string) {
	if len(args) == 0 {
		errOut("usage: event post-actions <event_id>")
	}
	eventID := args[0]
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
	et, ok := findEventType(typeID)
	if !ok {
		errOut(fmt.Sprintf("unknown event_type_id %q on event", typeID))
	}
	// Which playbook to use for post-event email campaign.
	playbook := "event_invite"
	switch typeID {
	case "webinar":
		playbook = "webinar_follow_up"
	case "workshop", "networking":
		playbook = "content_offering"
	}
	okOut(map[string]interface{}{
		"event_id":      eventID,
		"event_title":   resp.Data["title"],
		"post_tasks":    et.PostTasks,
		"survey_prompt": et.SurveyPrompt,
		"campaign_handoff": map[string]interface{}{
			"skill":    "sme-campaign",
			"playbook": playbook,
			"audience": fmt.Sprintf("All attendees of event %q", resp.Data["title"]),
			"command_hint": fmt.Sprintf(
				"Hand off to sme-campaign skill: create campaign for attendees with playbook %q.",
				playbook),
		},
		"survey_handoff": map[string]interface{}{
			"command": fmt.Sprintf("sme-cli event create-survey %s   (Phase 3, coming soon — uses Google Forms API)", eventID),
			"manual_fallback": "Until Phase 3 lands: create Google Form manually using `survey_prompt` as guide, then attach URL to event metadata.survey_url.",
		},
	})
}
