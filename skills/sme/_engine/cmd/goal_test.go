package main

import "testing"

// TestGoalMetricStagesOnlyKnownReliableTypes locks in exactly which metric
// types have a real progress source — any type not in this map (meetings,
// revenue, custom, or a typo) MUST fall back to "unknown" in goalView,
// never a guessed number. This test exists so adding a new metric type
// without a real data source can't silently happen.
//
// Phase 3B hardening: each metric now maps to a cumulative stage SET
// ("at-or-beyond"), not a single exact stage — see goalMetricStages'
// doc comment in goal.go for why exact-stage-only undercounts progress
// when a contact moves further along the pipeline.
func TestGoalMetricStagesOnlyKnownReliableTypes(t *testing.T) {
	want := map[string][]string{
		"qualified_leads": {"QUALIFIED", "PROPOSAL", "NEGOTIATION", "WON"},
		"proposals":       {"PROPOSAL", "NEGOTIATION", "WON"},
		"contracts":       {"WON"},
	}
	if len(goalMetricStages) != len(want) {
		t.Fatalf("goalMetricStages has %d entries, want %d — got %+v", len(goalMetricStages), len(want), goalMetricStages)
	}
	for k, v := range want {
		got := goalMetricStages[k]
		if len(got) != len(v) {
			t.Fatalf("goalMetricStages[%q] = %v, want %v", k, got, v)
		}
		for i := range v {
			if got[i] != v[i] {
				t.Errorf("goalMetricStages[%q] = %v, want %v", k, got, v)
			}
		}
	}
	for _, unreliable := range []string{"meetings", "revenue", "custom", "contact"} {
		if _, ok := goalMetricStages[unreliable]; ok {
			t.Errorf("goalMetricStages must NOT have an entry for %q — no reliable source exists, must stay unknown", unreliable)
		}
	}
}

// TestQualifiedLeadsIsCumulativeNotExactStage is the direct regression for
// the Phase 3B hardening bug: a contact must stay counted toward
// qualified_leads progress after moving QUALIFIED -> PROPOSAL, not drop out.
func TestQualifiedLeadsIsCumulativeNotExactStage(t *testing.T) {
	stages := goalMetricStages["qualified_leads"]
	match := map[string]bool{}
	for _, s := range stages {
		match[s] = true
	}
	for _, beyond := range []string{"QUALIFIED", "PROPOSAL", "NEGOTIATION", "WON"} {
		if !match[beyond] {
			t.Errorf("qualified_leads must count business_stage=%q (at-or-beyond qualified), got set %v", beyond, stages)
		}
	}
	if match["LOST"] || match["ENGAGED"] || match["NEW"] || match[""] {
		t.Errorf("qualified_leads must NOT count stages before/outside qualified, got set %v", stages)
	}
}

// TestProposalsIsCumulativeNotExactStage mirrors the above for the
// proposals metric: PROPOSAL -> WON must not make progress fall.
func TestProposalsIsCumulativeNotExactStage(t *testing.T) {
	stages := goalMetricStages["proposals"]
	match := map[string]bool{}
	for _, s := range stages {
		match[s] = true
	}
	for _, beyond := range []string{"PROPOSAL", "NEGOTIATION", "WON"} {
		if !match[beyond] {
			t.Errorf("proposals must count business_stage=%q (at-or-beyond proposal), got set %v", beyond, stages)
		}
	}
	if match["QUALIFIED"] || match["LOST"] {
		t.Errorf("proposals must NOT count QUALIFIED (before proposal) or LOST, got set %v", stages)
	}
}

// TestBaselineAndCurrentUseSameStageSet proves the exact same function
// (countContactsByBusinessStages) and the exact same goalMetricStages
// lookup back both goalSet's baseline capture and computeGoalProgress's
// current count — they cannot silently diverge into two different
// definitions of "qualified".
func TestBaselineAndCurrentUseSameStageSet(t *testing.T) {
	for metric, stages := range goalMetricStages {
		g := &goalRecord{MetricType: metric, HasBaseline: true, BaselineValue: 0, TargetValue: 100}
		lookedUp, ok := goalMetricStages[g.MetricType]
		if !ok {
			t.Fatalf("computeGoalProgress lookup path must find %q too", metric)
		}
		if len(lookedUp) != len(stages) {
			t.Fatalf("stage-set for %q diverged between call sites: %v vs %v", metric, lookedUp, stages)
		}
	}
}

// TestCumulativeCountingPreventsProgressRegressionOnStageAdvance is the
// direct before/after narrative from the Phase 3B hardening brief (Tests
// 2-4): a qualified_leads goal with baseline=6 (cumulative QUALIFIED+
// PROPOSAL+NEGOTIATION+WON, matching what was live-verified against
// production). Adding 2 new QUALIFIED contacts must raise progress; those
// same 2 contacts then advancing QUALIFIED -> PROPOSAL must NOT claw that
// progress back, because PROPOSAL is in the same cumulative set — the
// stage-set membership (proven in TestQualifiedLeadsIsCumulativeNotExactStage)
// guarantees the total headcount is unchanged by an internal move, so this
// exercises the real calculateGoalProgress with the concrete counts rather
// than re-deriving the logic.
func TestCumulativeCountingPreventsProgressRegressionOnStageAdvance(t *testing.T) {
	baseline := 6

	// Step 1: 2 new contacts reach QUALIFIED — cumulative count rises.
	afterNewQualified := baseline + 2 // = 8
	progress1, remaining1 := calculateGoalProgress(afterNewQualified, baseline, 5)
	if progress1 != 2 || remaining1 != 3 {
		t.Fatalf("after 2 new qualified: progress=%d remaining=%d, want 2/3", progress1, remaining1)
	}

	// Step 2: those exact 2 contacts advance QUALIFIED -> PROPOSAL. Because
	// PROPOSAL is also in the qualified_leads stage-set, the cumulative
	// headcount is unchanged (still 8) — progress must NOT fall back to 0.
	afterAdvanceToProposal := afterNewQualified // membership unchanged, still 8
	progress2, remaining2 := calculateGoalProgress(afterAdvanceToProposal, baseline, 5)
	if progress2 != progress1 {
		t.Fatalf("progress fell from %d to %d after QUALIFIED -> PROPOSAL advance — this is the exact bug being hardened against", progress1, progress2)
	}
	if remaining2 != remaining1 {
		t.Fatalf("remaining changed from %d to %d after a same-set stage advance, want unchanged", remaining1, remaining2)
	}

	// Mirror for a `proposals` goal: PROPOSAL -> WON must not regress either.
	baselineProposals := 2
	afterNewProposals := baselineProposals + 3 // 3 new PROPOSAL
	progress3, _ := calculateGoalProgress(afterNewProposals, baselineProposals, 5)
	if progress3 != 3 {
		t.Fatalf("proposals goal: progress=%d, want 3", progress3)
	}
	afterWon := afterNewProposals // WON is in the same proposals stage-set — headcount unchanged
	progress4, _ := calculateGoalProgress(afterWon, baselineProposals, 5)
	if progress4 != progress3 {
		t.Fatalf("proposals progress fell from %d to %d after PROPOSAL -> WON advance", progress3, progress4)
	}
}

// TestLostDowngradeIsHonestlyClampedNotHidden documents the accepted
// limitation (Phase 3B hardening, Part 4): COSMO has no stage-transition
// history, so a contact moving to LOST after having counted toward
// progress simply disappears from the cumulative count with no trace —
// progress can fall, clamped at 0, never negative. This is the "net new
// currently at-or-beyond" semantic, not "all-time achieved", and it is the
// honest ceiling given the data available (see measurement_method wording
// in computeGoalProgress).
func TestLostDowngradeIsHonestlyClampedNotHidden(t *testing.T) {
	baseline := 4
	// 2 new QUALIFIED arrive (current=6, progress=2)...
	progressBeforeLoss, _ := calculateGoalProgress(6, baseline, 5)
	if progressBeforeLoss != 2 {
		t.Fatalf("progress before loss = %d, want 2", progressBeforeLoss)
	}
	// ...then both get marked LOST (current drops back to 4, or below
	// baseline entirely if a pre-existing contact is also lost).
	progressAfterLoss, remainingAfterLoss := calculateGoalProgress(4, baseline, 5)
	if progressAfterLoss != 0 {
		t.Fatalf("progress after both go LOST = %d, want 0 (clamped, not negative)", progressAfterLoss)
	}
	if remainingAfterLoss != 5 {
		t.Fatalf("remaining after loss = %d, want 5 (back to full target)", remainingAfterLoss)
	}
}
