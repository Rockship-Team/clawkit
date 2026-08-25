package main

import "testing"

func TestOpportunityQualification(t *testing.T) {
	cases := []struct {
		stage string
		want  string
	}{
		{"", "not_started"},
		{"NEW", "not_started"},
		{"ENGAGED", "in_progress"},
		{"QUALIFIED", "qualified"},
		{"PROPOSAL", "qualified"},
		{"WON", "qualified"},
		{"LOST", "closed_lost"},
		{"DROPPED", "closed_lost"},
		{"SOMETHING_ELSE", "unknown"},
	}
	for _, tc := range cases {
		if got := opportunityQualification(tc.stage); got != tc.want {
			t.Errorf("opportunityQualification(%q) = %q, want %q", tc.stage, got, tc.want)
		}
	}
}

func TestOpportunityProposalReadinessNeverAutoJumps(t *testing.T) {
	// A single interested+positive reply must NOT report "ready".
	status, note := opportunityProposalReadiness("QUALIFIED", []replyClassification{
		{Intent: "interested", Sentiment: "positive"},
	})
	if status != "not_ready" {
		t.Errorf("interested+positive alone must not be ready, got %q", status)
	}
	if note == "" {
		t.Error("expected a note explaining why readiness is not yet established")
	}

	// Stage already at PROPOSAL/WON means readiness was already established.
	status, _ = opportunityProposalReadiness("PROPOSAL", nil)
	if status != "ready" {
		t.Errorf("stage=PROPOSAL should report ready, got %q", status)
	}
	status, _ = opportunityProposalReadiness("WON", nil)
	if status != "ready" {
		t.Errorf("stage=WON should report ready, got %q", status)
	}

	// No evidence at all.
	status, _ = opportunityProposalReadiness("QUALIFIED", nil)
	if status != "not_ready" {
		t.Errorf("no evidence should not be ready, got %q", status)
	}
}

func TestMergedRiskLowPriorityOverride(t *testing.T) {
	oc := opportunityContact{
		planContact:     planContact{ID: "c1", BusinessStage: "PROPOSAL", IdleDays: 4},
		Interactions90d: 4,
		Replies90d:      0,
	}
	risk := mergedRisk(oc, planContext{})
	if !risk.LowPriority {
		t.Errorf("expected LOW_PRIORITY override for stale outreach-in-progress contact, got %+v", risk)
	}
}

func TestMergedRiskKeepsBaseCellWhenNotStale(t *testing.T) {
	oc := opportunityContact{
		planContact:     planContact{ID: "c1", BusinessStage: "PROPOSAL", IdleDays: 4},
		Interactions90d: 4,
		Replies90d:      1, // has replied — not stale
	}
	risk := mergedRisk(oc, planContext{})
	if risk.LowPriority {
		t.Errorf("should not override when contact has replied, got %+v", risk)
	}
	if risk.Cell != "PROPOSAL_HOT" {
		t.Errorf("expected base cell PROPOSAL_HOT, got %q", risk.Cell)
	}
}

func TestMergedRiskUnknownWhenNoCellMatches(t *testing.T) {
	oc := opportunityContact{planContact: planContact{ID: "c1", BusinessStage: "QUALIFIED", IdleDays: 1}}
	risk := mergedRisk(oc, planContext{})
	if risk.Label != "unknown" {
		t.Errorf("expected unknown risk when no cell matches, got %+v", risk)
	}
}
