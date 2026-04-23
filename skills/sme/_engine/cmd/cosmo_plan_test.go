package main

import (
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		c    planContact
		want string
	}{
		{"proposal hot day 4", planContact{BusinessStage: "PROPOSAL", IdleDays: 4}, "PROPOSAL_HOT"},
		{"proposal hot day 2", planContact{BusinessStage: "PROPOSAL", IdleDays: 2}, "PROPOSAL_HOT"},
		{"proposal hot day 1", planContact{BusinessStage: "PROPOSAL", IdleDays: 1}, ""}, // too fresh
		{"proposal stuck day 10", planContact{BusinessStage: "PROPOSAL", IdleDays: 10}, "PROPOSAL_STUCK"},
		{"proposal ghost day 20", planContact{BusinessStage: "PROPOSAL", IdleDays: 20}, "PROPOSAL_GHOST"},
		{"qualified idle 5", planContact{BusinessStage: "QUALIFIED", IdleDays: 5}, "QUALIFIED_OPEN"},
		{"qualified idle 1", planContact{BusinessStage: "QUALIFIED", IdleDays: 1}, ""},
		{"engaged warm", planContact{BusinessStage: "ENGAGED", IdleDays: 7, Interactions30d: 2}, "ENGAGED_WARM"},
		{"engaged cold idle 20", planContact{BusinessStage: "ENGAGED", IdleDays: 20}, "ENGAGED_COLD"},
		{"engaged cold NO_REPLY", planContact{BusinessStage: "ENGAGED", IdleDays: 3, ConversationState: "NO_REPLY"}, "ENGAGED_COLD"},
		{"new event", planContact{BusinessStage: "", Source: "openclaw_event", IdleDays: 1, Email: "x@y", HasEmail: true}, "NEW_EVENT"},
		{"new apollo full", planContact{BusinessStage: "NEW", Source: "apollo_io", HasEmail: true}, "NEW_APOLLO_FULL"},
		{"new apollo linkedin only", planContact{BusinessStage: "NEW", Source: "apollo_io", HasLinkedIn: true}, "NEW_APOLLO_LINKEDIN"},
		{"new no channel", planContact{BusinessStage: "", Source: "manual"}, "NEW_NO_CHANNEL"},
		{"won recent skip", planContact{BusinessStage: "WON", IdleDays: 10}, ""},
		{"won 30+d checkin", planContact{BusinessStage: "WON", IdleDays: 31}, "WON_CHECKIN"},
		{"lost recent skip", planContact{BusinessStage: "LOST", IdleDays: 30}, ""},
		{"lost 60+d revive", planContact{BusinessStage: "LOST", IdleDays: 60}, "LOST_REVIVE"},
	}
	for _, tc := range cases {
		got := classify(tc.c, planContext{})
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBuildPlanCellsFiltersByMode(t *testing.T) {
	contacts := []planContact{
		{BusinessStage: "PROPOSAL", IdleDays: 4},                                        // HOT
		{BusinessStage: "QUALIFIED", IdleDays: 5},                                       // QUALIFIED_OPEN
		{BusinessStage: "", Source: "openclaw_event", Email: "x@y", HasEmail: true},     // NEW_EVENT
		{BusinessStage: "ENGAGED", IdleDays: 20},                                        // ENGAGED_COLD
		{BusinessStage: "WON", IdleDays: 40},                                            // WON_CHECKIN
	}
	now := time.Now()

	morning := buildPlanCells(contacts, planContext{}, now, "morning")
	hasCell := func(cells []planCell, id string) bool {
		for _, c := range cells {
			if c.ID == id {
				return true
			}
		}
		return false
	}
	if !hasCell(morning, "PROPOSAL_HOT") {
		t.Error("morning should include PROPOSAL_HOT")
	}
	if !hasCell(morning, "QUALIFIED_OPEN") {
		t.Error("morning should include QUALIFIED_OPEN")
	}
	if hasCell(morning, "NEW_EVENT") {
		t.Error("morning should NOT include NEW_EVENT (too early for cold outreach)")
	}
	if hasCell(morning, "ENGAGED_COLD") {
		t.Error("morning should NOT include ENGAGED_COLD")
	}
	if hasCell(morning, "WON_CHECKIN") {
		t.Error("morning should NOT include WON_CHECKIN")
	}

	all := buildPlanCells(contacts, planContext{}, now, "all")
	if !hasCell(all, "NEW_EVENT") {
		t.Error("all-mode should include NEW_EVENT")
	}
	if !hasCell(all, "WON_CHECKIN") {
		t.Error("all-mode should include WON_CHECKIN")
	}

	// Priority sort: PROPOSAL_HOT (1) before QUALIFIED_OPEN (5)
	if len(morning) >= 2 && morning[0].ID != "PROPOSAL_HOT" {
		t.Errorf("morning[0] = %q, want PROPOSAL_HOT", morning[0].ID)
	}
}

func TestBuildPlanCellsLimitsContacts(t *testing.T) {
	// 10 PROPOSAL_HOT contacts, sorted by idle days descending
	contacts := make([]planContact, 10)
	for i := range contacts {
		contacts[i] = planContact{
			ID:            "c" + string(rune('0'+i)),
			Name:          "C" + string(rune('0'+i)),
			BusinessStage: "PROPOSAL",
			IdleDays:      i + 2,
		}
	}
	cells := buildPlanCells(contacts, planContext{}, time.Now(), "morning")
	var hot *planCell
	for i := range cells {
		if cells[i].ID == "PROPOSAL_HOT" {
			hot = &cells[i]
		}
	}
	if hot == nil {
		t.Fatal("no PROPOSAL_HOT cell returned")
	}
	// Only 6 contacts have idle 2-7 (HOT range)
	if hot.Count != 6 {
		t.Errorf("Count = %d, want 6 (idle days 2-7 in range 2..11)", hot.Count)
	}
	if len(hot.Contacts) > 5 {
		t.Errorf("morning limits contacts shown to 5, got %d", len(hot.Contacts))
	}
	// Sorted idle-days desc — first should have highest idle
	if len(hot.Contacts) >= 2 && hot.Contacts[0].IdleDays < hot.Contacts[1].IdleDays {
		t.Error("contacts not sorted by idle days descending")
	}
}

func TestDaysSince(t *testing.T) {
	now := time.Now().UTC()
	threeDaysAgo := now.Add(-72 * time.Hour).Format(time.RFC3339)
	if d := daysSince(threeDaysAgo); d != 3 {
		t.Errorf("daysSince(3d ago) = %d, want 3", d)
	}
	if d := daysSince(""); d != 9999 {
		t.Errorf("daysSince(empty) = %d, want 9999", d)
	}
	if d := daysSince("not-a-date"); d != 9999 {
		t.Errorf("daysSince(garbage) = %d, want 9999", d)
	}
}

func TestClassifyMeetingTomorrow(t *testing.T) {
	c := planContact{ID: "c1", BusinessStage: "QUALIFIED", IdleDays: 5}
	ctx := planContext{
		MeetingsTomorrow: map[string][]planMeeting{
			"c1": {{ID: "m1", Title: "demo", ContactID: "c1"}},
		},
	}
	if got := classify(c, ctx); got != "MEETING_TOMORROW" {
		t.Errorf("got %q, want MEETING_TOMORROW", got)
	}
}

func TestClassifyCampaignSentNoReply(t *testing.T) {
	c := planContact{ID: "c1", BusinessStage: "ENGAGED", IdleDays: 7, Interactions30d: 1}
	ctx := planContext{
		ActiveCampaigns: map[string][]planCampaignRef{
			"c1": {{ID: "camp1", Status: "active", Sent: 50, Reply: 0}},
		},
	}
	if got := classify(c, ctx); got != "CAMPAIGN_SENT_NO_REPLY" {
		t.Errorf("got %q, want CAMPAIGN_SENT_NO_REPLY", got)
	}
}

func TestClassifyProposalOverridesCampaign(t *testing.T) {
	// Campaign context should NOT override PROPOSAL_HOT — urgency rule
	c := planContact{ID: "c1", BusinessStage: "PROPOSAL", IdleDays: 4}
	ctx := planContext{
		ActiveCampaigns: map[string][]planCampaignRef{
			"c1": {{ID: "camp1", Status: "active", Sent: 50, Reply: 0}},
		},
	}
	if got := classify(c, ctx); got != "PROPOSAL_HOT" {
		t.Errorf("got %q, want PROPOSAL_HOT (proposal urgency wins)", got)
	}
}

func TestEveningModeLimitsSmaller(t *testing.T) {
	contacts := make([]planContact, 10)
	for i := range contacts {
		contacts[i] = planContact{BusinessStage: "PROPOSAL", IdleDays: i + 2}
	}
	cells := buildPlanCells(contacts, planContext{}, time.Now(), "evening")
	for _, c := range cells {
		if len(c.Contacts) > 3 {
			t.Errorf("evening mode should limit to 3 per cell, got %d in %s", len(c.Contacts), c.ID)
		}
	}
}
