package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// cosmoDailyPlan builds a deterministic daily outreach plan.
//
//	sme-cli cosmo daily-plan [--mode morning|evening|all] [--max-pages N]
//
// Output is a JSON report grouping contacts into pipeline cells with an
// action hint per cell (subject/length/CTA — baked 2026 benchmarks).
// The caller (a skill) is expected to render this into a human-friendly
// chat message.
func cosmoDailyPlan(args []string) {
	mode := "all"
	maxPages := 8
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		case "--max-pages":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &maxPages)
				i++
			}
		}
	}
	switch mode {
	case "morning", "evening", "all":
	default:
		errOut("--mode must be morning|evening|all, got: " + mode)
	}

	warnings := []planWarning{}
	if w := checkEmailAgentHealth(); w != nil {
		warnings = append(warnings, *w)
	}

	contacts, total, err := fetchAllContacts(maxPages)
	if err != nil {
		errOut("fetch contacts: " + err.Error())
	}

	now := vnNow()
	ctx := fetchPlanContext()
	cells := buildPlanCells(contacts, ctx, now, mode)

	okOut(map[string]interface{}{
		"mode":           mode,
		"generated_at":   now.Format(time.RFC3339),
		"total_contacts": total,
		"loaded":         len(contacts),
		"cells":          cells,
		"warnings":       warnings,
	})
}

type planWarning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type planContact struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Company           string `json:"company,omitempty"`
	JobTitle          string `json:"job_title,omitempty"`
	Industry          string `json:"industry,omitempty"`
	Email             string `json:"email,omitempty"`
	Source            string `json:"source,omitempty"`
	BusinessStage     string `json:"business_stage,omitempty"`
	IdleDays          int    `json:"idle_days"`
	Interactions30d   int    `json:"interactions_30d"`
	NextStep          string `json:"next_step,omitempty"`
	LastOutcome       string `json:"last_outcome,omitempty"`
	ConversationState string `json:"conversation_state,omitempty"`
	HasEmail          bool   `json:"has_email"`
	HasLinkedIn       bool   `json:"has_linkedin"`
	HasPhone          bool   `json:"has_phone"`

	// EnrichmentStatus signals whether the contact has enough business
	// context for a personalized action hint. "enriched" if company AND
	// (job_title OR industry) are set; "partial" if one is set; "needed"
	// if neither.
	EnrichmentStatus string `json:"enrichment_status"`
}

type planCell struct {
	ID       string        `json:"id"`
	Emoji    string        `json:"emoji"`
	Name     string        `json:"name"`
	Priority int           `json:"priority"`
	Why      string        `json:"why"`
	Count    int           `json:"count"`
	Action   planAction    `json:"action"`
	Contacts []planContact `json:"contacts"`

	// EnrichmentSummary reports how many contacts in this cell have
	// enough context for a personalized action. Populated by
	// buildPlanCells.
	EnrichmentSummary struct {
		Enriched int `json:"enriched"`
		Partial  int `json:"partial"`
		Needed   int `json:"needed"`
	} `json:"enrichment_summary"`
}

type planAction struct {
	Playbook    string `json:"playbook,omitempty"`
	SubjectHint string `json:"subject_hint"`
	Length      string `json:"length"`
	CTA         string `json:"cta"`
	Example     string `json:"example,omitempty"`
}

// --- fetch + flatten ---

func fetchAllContacts(maxPages int) ([]planContact, int, error) {
	var all []planContact
	total := 0
	for page := 1; page <= maxPages; page++ {
		body, _ := json.Marshal(map[string]interface{}{
			"query":    "",
			"page":     page,
			"pageSize": 25,
		})
		raw, code, err := cosmoRequest("POST", "/v2/contacts/search", body)
		if err != nil {
			return nil, 0, err
		}
		if code >= 400 {
			return nil, 0, fmt.Errorf("HTTP %d: %s", code, string(raw))
		}
		var resp struct {
			Data struct {
				Total int `json:"total"`
				List  []struct {
					Entity map[string]interface{} `json:"entity"`
				} `json:"list"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, 0, err
		}
		total = resp.Data.Total
		if len(resp.Data.List) == 0 {
			break
		}
		for _, item := range resp.Data.List {
			all = append(all, flattenContact(item.Entity))
		}
		if len(all) >= total {
			break
		}
	}
	return all, total, nil
}

func flattenContact(e map[string]interface{}) planContact {
	get := func(k string) string {
		if v, ok := e[k].(string); ok {
			return v
		}
		return ""
	}
	c := planContact{
		ID:            get("id"),
		Name:          get("name"),
		Company:       get("company"),
		JobTitle:      get("job_title"),
		Industry:      get("industry"),
		Email:         get("email"),
		Source:        get("source"),
		BusinessStage: strings.ToUpper(get("business_stage")),
		NextStep:      get("next_step"),
		LastOutcome:   get("last_outcome"),
	}
	hasCompany := strings.TrimSpace(c.Company) != ""
	hasRole := strings.TrimSpace(c.JobTitle) != "" || strings.TrimSpace(c.Industry) != ""
	switch {
	case hasCompany && hasRole:
		c.EnrichmentStatus = "enriched"
	case hasCompany || hasRole:
		c.EnrichmentStatus = "partial"
	default:
		c.EnrichmentStatus = "needed"
	}
	c.HasEmail = c.Email != "" && strings.Contains(c.Email, "@")
	c.HasLinkedIn = get("linkedin_url") != ""
	c.HasPhone = get("phone") != ""

	rel, _ := e["relationship"].(map[string]interface{})
	if v, ok := rel["interactions_30d"].(float64); ok {
		c.Interactions30d = int(v)
	}
	lastISO, _ := rel["last_interaction_at"].(string)
	if lastISO == "" {
		lastISO = get("created_at")
	}
	c.IdleDays = daysSince(lastISO)
	if conv, ok := e["outreach_decision"].(map[string]interface{}); ok {
		if s, ok := conv["conversation_state"].(string); ok {
			c.ConversationState = strings.ToUpper(s)
		}
	}
	return c
}

func daysSince(iso string) int {
	if iso == "" {
		return 9999
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.999999Z", iso)
	}
	if err != nil {
		return 9999
	}
	d := time.Since(t).Hours() / 24
	if d < 0 {
		return 0
	}
	return int(d)
}

// --- categorization ---

// planMeeting is a scheduled meeting relevant to a contact.
type planMeeting struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	ScheduledAt time.Time `json:"scheduled_at"`
	ContactID   string    `json:"contact_id"`
}

// planCampaignRef summarises a campaign the contact is currently inside.
type planCampaignRef struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Playbook     string  `json:"playbook"`
	Status       string  `json:"status"`
	Sent         int     `json:"sent"`
	Reply        int     `json:"reply"`
	ReplyRate    float64 `json:"reply_rate"`
	CreatedDaysAgo int   `json:"created_days_ago"`
}

// planContext carries aggregate lookups reused across many contacts.
type planContext struct {
	MeetingsTomorrow map[string][]planMeeting    // contact_id → meetings scheduled tomorrow (ICT)
	ActiveCampaigns  map[string][]planCampaignRef // contact_id → live campaigns with sent>0 and reply_rate=0
}

func fetchPlanContext() planContext {
	return planContext{
		MeetingsTomorrow: fetchTomorrowMeetings(),
		ActiveCampaigns:  fetchNoReplyCampaignMemberships(),
	}
}

func fetchTomorrowMeetings() map[string][]planMeeting {
	out := map[string][]planMeeting{}
	raw, code, err := cosmoRequest("GET", "/v1/outreach/meetings", nil)
	if err != nil || code >= 400 {
		return out
	}
	var resp struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return out
	}
	now := vnNow()
	tomorrowStart := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	tomorrowEnd := tomorrowStart.Add(24 * time.Hour)
	for _, m := range resp.Data {
		contactID, _ := m["contact_id"].(string)
		if contactID == "" {
			continue
		}
		timeStr, _ := m["time"].(string)
		if timeStr == "" {
			timeStr, _ = m["scheduled_at"].(string)
		}
		scheduledAt, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}
		scheduledICT := scheduledAt.In(now.Location())
		if scheduledICT.Before(tomorrowStart) || !scheduledICT.Before(tomorrowEnd) {
			continue
		}
		title, _ := m["title"].(string)
		id, _ := m["id"].(string)
		out[contactID] = append(out[contactID], planMeeting{
			ID:          id,
			Title:       title,
			ScheduledAt: scheduledICT,
			ContactID:   contactID,
		})
	}
	return out
}

// fetchNoReplyCampaignMemberships lists active campaigns that have sent
// emails but received zero replies, then fetches each campaign's list
// of contacts. Returns map from contact_id → campaigns.
func fetchNoReplyCampaignMemberships() map[string][]planCampaignRef {
	out := map[string][]planCampaignRef{}
	raw, code, err := cosmoRequest("POST", "/v1/campaigns/search", []byte(`{"filter_":{}}`))
	if err != nil || code >= 400 {
		return out
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity map[string]interface{} `json:"entity"`
				Sent   float64                `json:"sent"`
				Reply  float64                `json:"reply"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return out
	}
	now := vnNow()
	processed := 0
	const maxCampaigns = 10
	for _, c := range resp.Data.List {
		if processed >= maxCampaigns {
			break
		}
		status, _ := c.Entity["status"].(string)
		if strings.ToLower(status) != "active" {
			continue
		}
		if int(c.Sent) <= 0 || int(c.Reply) > 0 {
			continue // only want sent>0, reply=0 campaigns
		}
		listID, _ := c.Entity["list_contact_id"].(string)
		if listID == "" {
			continue
		}
		processed++
		ref := planCampaignRef{
			ID:       stringField(c.Entity, "id"),
			Name:     stringField(c.Entity, "name"),
			Playbook: stringField(c.Entity, "playbook"),
			Status:   status,
			Sent:     int(c.Sent),
			Reply:    int(c.Reply),
		}
		if createdAt := stringField(c.Entity, "created_at"); createdAt != "" {
			ref.CreatedDaysAgo = daysSince(createdAt)
		}
		// Fetch list contacts
		listRaw, listCode, listErr := cosmoRequest("GET", "/v1/list-contacts/"+listID, nil)
		if listErr != nil || listCode >= 400 {
			continue
		}
		var listResp struct {
			Data struct {
				Contacts []map[string]interface{} `json:"contacts"`
			} `json:"data"`
		}
		if err := json.Unmarshal(listRaw, &listResp); err != nil {
			continue
		}
		for _, contact := range listResp.Data.Contacts {
			cid, _ := contact["id"].(string)
			if cid == "" {
				continue
			}
			out[cid] = append(out[cid], ref)
		}
	}
	_ = now
	return out
}

func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// classify returns the pipeline cell ID for a contact. Top-down match.
func classify(c planContact, ctx planContext) string {
	// Urgent operational signals override stage-based classification.
	if _, ok := ctx.MeetingsTomorrow[c.ID]; ok {
		return "MEETING_TOMORROW"
	}
	if _, ok := ctx.ActiveCampaigns[c.ID]; ok {
		// Only bump into this cell if the stage-based flow wouldn't
		// already put them somewhere more urgent (PROPOSAL*). Otherwise
		// campaign-in-flight is the most actionable signal.
		if c.BusinessStage != "PROPOSAL" {
			return "CAMPAIGN_SENT_NO_REPLY"
		}
	}
	src := strings.ToLower(c.Source)
	switch c.BusinessStage {
	case "PROPOSAL":
		switch {
		case c.IdleDays <= 7 && c.IdleDays >= 2:
			return "PROPOSAL_HOT"
		case c.IdleDays >= 8 && c.IdleDays <= 14:
			return "PROPOSAL_STUCK"
		case c.IdleDays >= 15:
			return "PROPOSAL_GHOST"
		}
	case "QUALIFIED":
		if c.IdleDays >= 3 {
			return "QUALIFIED_OPEN"
		}
	case "ENGAGED":
		if c.ConversationState == "NO_REPLY" || c.IdleDays >= 15 {
			return "ENGAGED_COLD"
		}
		if c.Interactions30d > 0 && c.IdleDays >= 5 && c.IdleDays <= 14 {
			return "ENGAGED_WARM"
		}
	case "WON":
		if c.IdleDays >= 30 {
			return "WON_CHECKIN"
		}
	case "LOST":
		if c.IdleDays >= 60 {
			return "LOST_REVIVE"
		}
	case "", "NEW":
		switch {
		case strings.Contains(src, "event"):
			return "NEW_EVENT"
		case strings.Contains(src, "apollo"):
			if c.HasEmail {
				return "NEW_APOLLO_FULL"
			}
			if c.HasLinkedIn {
				return "NEW_APOLLO_LINKEDIN"
			}
			return "NEW_NO_CHANNEL"
		}
		if !c.HasEmail && !c.HasLinkedIn && !c.HasPhone {
			return "NEW_NO_CHANNEL"
		}
	}
	if c.LastOutcome == "meeting_done" || c.ConversationState == "POST_MEETING" {
		return "POST_MEETING"
	}
	return ""
}

// --- cell templates ---

var cellTemplates = map[string]planCell{
	"MEETING_TOMORROW": {
		ID: "MEETING_TOMORROW", Emoji: "📆", Name: "Meeting ngay mai — can prep", Priority: 1,
		Why: "Meeting sap toi can brief truoc. Prep thieu = meeting yeu",
		Action: planAction{
			SubjectHint: "N/A (prep noi bo)",
			Length:      "Internal brief: goals / 3 questions to ask / objection preps / demo / proposal draft",
			CTA:         "Review interaction history + chuan bi 3 slide topic ho quan tam nhat",
		},
	},
	"CAMPAIGN_SENT_NO_REPLY": {
		ID: "CAMPAIGN_SENT_NO_REPLY", Emoji: "📧", Name: "Da gui campaign, chua co phan hoi", Priority: 4,
		Why: "Campaign da gui nhung chua ai reply. 42% replies den tu follow-up, khong phai email dau",
		Action: planAction{
			Playbook:    "cold_outreach",
			SubjectHint: "Khac voi campaign subject — personal, short",
			Length:      "50-80 words, 1-1 not mass",
			CTA:         "Follow-up tay sau 3 ngay (NOT re-send campaign) — reference 1 detail cu the",
		},
	},
	"PROPOSAL_HOT": {
		ID: "PROPOSAL_HOT", Emoji: "🔥", Name: "Can follow-up gap", Priority: 1,
		Why: "Day 3 sweet spot cho touch #2 (+31% reply vs next-day, -11% neu chua day 1)",
		Action: planAction{
			Playbook:    "cold_outreach",
			SubjectHint: "{specific_detail} from our proposal — 6-10 words OR <=4 words signal-led",
			Length:      "50-125 words",
			CTA:         "1 trong 3: (a) 15-min call voi 3 slot thoi gian, (b) cau hoi cu the, (c) resource",
		},
	},
	"PROPOSAL_STUCK": {
		ID: "PROPOSAL_STUCK", Emoji: "⏳", Name: "Dang bi stuck", Priority: 2,
		Why: "Day 10 la cutoff 93% cumulative reply. Shift ask tu 'decision' sang 'still relevant?'",
		Action: planAction{
			Playbook:    "revive_dormant_leads",
			SubjectHint: "'{company} — paused?' / '{name}, different direction?'",
			Length:      "50-80 words",
			CTA:         "Offer lower-commitment (shorter scope / smaller kickoff / close file)",
		},
	},
	"PROPOSAL_GHOST": {
		ID: "PROPOSAL_GHOST", Emoji: "👻", Name: "Proposal ghost", Priority: 3,
		Why: "Post day 17 reply <1%. Quyet dinh revive 1 lan cuoi vs close file",
		Action: planAction{
			Playbook:    "content_offering",
			SubjectHint: "Signal-led value, KHONG 'checking in'",
			Length:      "<=50 words, chi share resource, no ask",
			CTA:         "Khong ask — chi value drop. Neu 5d im → move LOST",
		},
	},
	"POST_MEETING": {
		ID: "POST_MEETING", Emoji: "📝", Name: "Recap meeting chua gui", Priority: 4,
		Why: "Recap trong 24h = critical. >48h = te, reply rate rot",
		Action: planAction{
			SubjectHint: "'Recap — {meeting_topic}'",
			Length:      "50-125 words: 3 bullet recap + next step our side + next step their side + target date",
			CTA:         "Log interaction type=meeting, direction=outbound",
		},
	},
	"QUALIFIED_OPEN": {
		ID: "QUALIFIED_OPEN", Emoji: "💡", Name: "Cho meeting slot", Priority: 5,
		Why: "Qualified im >=3d = dang mat nhiet",
		Action: planAction{
			SubjectHint: "'{name}, demo Tue or Thu?'",
			Length:      "50-75 words",
			CTA:         "3 slot cu the (KHONG 'anh ranh khi nao'): vd Tue 2pm / Thu 10am / Fri 4pm ICT",
		},
	},
	"ENGAGED_WARM": {
		ID: "ENGAGED_WARM", Emoji: "🌱", Name: "Nurture warm", Priority: 6,
		Why: "Tuong tac recent, nurture de khong nguoi. Value-first, KHONG push",
		Action: planAction{
			Playbook:    "content_offering",
			SubjectHint: "'Case study — {their_industry}' / signal-led",
			Length:      "50-100 words",
			CTA:         "Share resource, KHONG push meeting hay proposal",
		},
	},
	"ENGAGED_COLD": {
		ID: "ENGAGED_COLD", Emoji: "🧊", Name: "Pattern interrupt", Priority: 7,
		Why: "Idle 15+d / NO_REPLY. Pattern interrupt subject +24.6% reply vs formulaic",
		Action: planAction{
			SubjectHint: "<=4 words unusual: 'Random thought' / '{Topic} question'",
			Length:      "<=50 words, low-pressure",
			CTA:         "'Not a pitch, just curious about X' — reference recent event their side. 7d no reply → DROPPED",
		},
	},
	"NEW_EVENT": {
		ID: "NEW_EVENT", Emoji: "🎫", Name: "Event attendee chua cham", Priority: 8,
		Why: "<50 recipients/batch = 5.8% reply (3x mass). Signal-led (booth convo) = 15-25% reply",
		Action: planAction{
			Playbook:    "event_invite",
			SubjectHint: "'{event} — {value_topic}?' — 6-10 words",
			Length:      "50-125 words: thank + 1 takeaway + 1 CTA",
			CTA:         "20-min discovery call voi calendar link. Segment <50/batch truoc khi send",
			Example:     "Send trong 24h cua event. Reference booth/session cu the. 3-day gap truoc follow-up 2 = +31% reply",
		},
	},
	"NEW_APOLLO_FULL": {
		ID: "NEW_APOLLO_FULL", Emoji: "🆕", Name: "Apollo cold, co email", Priority: 9,
		Why: "Signal-led Apollo data → 15-25% reply; generic blast → 1-3%. Cadence 3-7-7 (NOT 0/3/7)",
		Action: planAction{
			Playbook:    "cold_outreach",
			SubjectHint: "Signal tu Apollo (recent funding / tech stack / role)",
			Length:      "50-80 words per email",
			CTA:         "4 emails: Day 0 intro+signal+CTA, Day 3 different angle, Day 10 case study, Day 17 breakup",
		},
	},
	"NEW_APOLLO_LINKEDIN": {
		ID: "NEW_APOLLO_LINKEDIN", Emoji: "🔗", Name: "LinkedIn-only", Priority: 10,
		Why: "Khong co email, chi co LinkedIn. Connection request truoc, KHONG pitch",
		Action: planAction{
			SubjectHint: "N/A (LinkedIn message)",
			Length:      "<=300 chars connection note",
			CTA:         "'Saw your {specific_thing} — impressive. Would love to connect.' Sau accept 3-5d moi follow-up",
		},
	},
	"NEW_NO_CHANNEL": {
		ID: "NEW_NO_CHANNEL", Emoji: "❓", Name: "Thieu channel", Priority: 11,
		Why: "Khong email/linkedin/phone. Phai enrich truoc khi outreach",
		Action: planAction{
			SubjectHint: "N/A",
			Length:      "N/A",
			CTA:         "Chay 'sme-cli cosmo enrich <id>'. 1 tuan van ra trong → move LOST",
		},
	},
	"WON_CHECKIN": {
		ID: "WON_CHECKIN", Emoji: "🏆", Name: "Customer check-in", Priority: 12,
		Why: "Customer 30+d silent = renewal/upsell opportunity",
		Action: planAction{
			Playbook:    "upsell_existing_customers",
			SubjectHint: "'{name}, pulse check — {feature/project}'",
			Length:      "50-125 words",
			CTA:         "Pulse question + mention 1 new capability. KHONG hard sell",
		},
	},
	"LOST_REVIVE": {
		ID: "LOST_REVIVE", Emoji: "♻️", Name: "Revive LOST", Priority: 13,
		Why: "60+d tu LOST = market/team doi. Fresh value proposition",
		Action: planAction{
			Playbook:    "revive_dormant_leads",
			SubjectHint: "'We've changed — {specific_thing}'",
			Length:      "<=50 words",
			CTA:         "New case study / pricing / product capability + 1 CTA",
		},
	},
}

// --- build cells ---

var morningCells = []string{"MEETING_TOMORROW", "PROPOSAL_HOT", "PROPOSAL_STUCK", "PROPOSAL_GHOST", "POST_MEETING", "CAMPAIGN_SENT_NO_REPLY", "QUALIFIED_OPEN"}
var eveningCells = []string{"MEETING_TOMORROW", "POST_MEETING", "PROPOSAL_HOT", "QUALIFIED_OPEN"}

func buildPlanCells(contacts []planContact, ctx planContext, now time.Time, mode string) []planCell {
	buckets := make(map[string][]planContact)
	for _, c := range contacts {
		id := classify(c, ctx)
		if id == "" {
			continue
		}
		buckets[id] = append(buckets[id], c)
	}

	var allowlist map[string]bool
	switch mode {
	case "morning":
		allowlist = stringSet(morningCells)
	case "evening":
		allowlist = stringSet(eveningCells)
	}

	result := make([]planCell, 0, len(buckets))
	for id, list := range buckets {
		if allowlist != nil && !allowlist[id] {
			continue
		}
		tpl, ok := cellTemplates[id]
		if !ok {
			continue
		}
		sortByIdleDesc(list)
		// Tally enrichment over the FULL bucket before we truncate for display.
		for _, c := range buckets[id] {
			switch c.EnrichmentStatus {
			case "enriched":
				tpl.EnrichmentSummary.Enriched++
			case "partial":
				tpl.EnrichmentSummary.Partial++
			default:
				tpl.EnrichmentSummary.Needed++
			}
		}
		limit := 5
		if mode == "evening" {
			limit = 3
		}
		if len(list) > limit {
			list = list[:limit]
		}
		tpl.Count = len(buckets[id])
		tpl.Contacts = list
		result = append(result, tpl)
	}
	sortByPriority(result)
	return result
}

func sortByIdleDesc(list []planContact) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].IdleDays > list[j-1].IdleDays; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func sortByPriority(cells []planCell) {
	for i := 1; i < len(cells); i++ {
		for j := i; j > 0 && cells[j].Priority < cells[j-1].Priority; j-- {
			cells[j], cells[j-1] = cells[j-1], cells[j]
		}
	}
}

func stringSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// --- email agent health ---

func checkEmailAgentHealth() *planWarning {
	raw, code, err := cosmoRequest("POST", "/v1/agents/search", []byte(`{"filter_":{}}`))
	if err != nil || code >= 400 {
		return nil
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity struct {
					ValidCred bool   `json:"valid_cred"`
					Status    string `json:"status"`
					Name      string `json:"name"`
				} `json:"entity"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	for _, it := range resp.Data.List {
		if !it.Entity.ValidCred {
			return &planWarning{
				Type:    "email_agent_invalid_cred",
				Message: fmt.Sprintf("Email agent %q mat xac thuc (%s). Cac suggestion email campaign can reconnect truoc khi chay.", it.Entity.Name, it.Entity.Status),
			}
		}
	}
	return nil
}
