package main

import "testing"

func baseGoal() *goalRecord {
	return &goalRecord{ID: "g1", GoalText: "5 qualified leads cho AI Automation", MetricType: "qualified_leads", TargetValue: 5}
}

// TestDecideGoalNBAUnmeasurable proves an unmeasurable metric never forces
// a bottleneck guess — Case: no reliable source / no baseline.
func TestDecideGoalNBAUnmeasurable(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: false, MeasurementMethod: "unknown — no source"}
	got := decideGoalNBA(g, progress, goalSignals{})
	if got.Confidence != "low" || got.RecommendedSkill != "" {
		t.Fatalf("unmeasurable progress must not force a skill/confidence, got %+v", got)
	}
	if got.DetectedBottleneck == "" || got.RecommendedAction == "" {
		t.Fatal("must still explain why, not just silently return empty")
	}
}

// TestCalculateGoalProgressCase1BaselineCorrectness verifies Test 1 from
// the Phase 3B brief: 4 QUALIFIED existed before the goal, none since
// (baseline=4, current=4) -> progress MUST be 0/5, never 4/5. Exercises
// the real calculateGoalProgress, not a re-typed copy.
func TestCalculateGoalProgressCase1BaselineCorrectness(t *testing.T) {
	progress, remaining := calculateGoalProgress(4, 4, 5)
	if progress != 0 {
		t.Fatalf("progress = %d, want 0 (4 baseline, 4 current — nothing new since goal creation)", progress)
	}
	if remaining != 5 {
		t.Fatalf("remaining = %d, want 5", remaining)
	}
}

// TestCalculateGoalProgressCase2Progress verifies Test 2: baseline=4,
// current=6 -> progress=2, remaining=3.
func TestCalculateGoalProgressCase2Progress(t *testing.T) {
	progress, remaining := calculateGoalProgress(6, 4, 5)
	if progress != 2 {
		t.Fatalf("progress = %d, want 2", progress)
	}
	if remaining != 3 {
		t.Fatalf("remaining = %d, want 3", remaining)
	}
}

// TestCalculateGoalProgressNeverNegative proves a contact leaving the
// stage (e.g. QUALIFIED -> LOST) can't produce a negative progress —
// clamped to 0, never reported as "went backwards".
func TestCalculateGoalProgressNeverNegative(t *testing.T) {
	progress, remaining := calculateGoalProgress(2, 4, 5) // current dropped below baseline
	if progress != 0 {
		t.Fatalf("progress = %d, want 0 (clamped, never negative)", progress)
	}
	if remaining != 5 {
		t.Fatalf("remaining = %d, want 5", remaining)
	}
}

// TestDecideGoalNBACase3GoalReached verifies Test 3: progress >= target ->
// NBA recommends completion/report, never more outreach, regardless of
// what bottleneck signals are also present.
func TestDecideGoalNBACase3GoalReached(t *testing.T) {
	g := &goalRecord{ID: "g1", GoalText: "t", MetricType: "qualified_leads", TargetValue: 2}
	progress := goalProgressResult{OK: true, CurrentValue: 6, Progress: 2, Remaining: 0, MeasurementMethod: "m"}
	// Even with a huge available pool (which would trigger Case A if
	// progress were still short), goal-reached must win.
	got := decideGoalNBA(g, progress, goalSignals{PoolAvailable: 100, ActiveOutreach: 0})
	if got.RecommendedSkill == "sme-intelligence + sme-campaign" {
		t.Fatal("must not recommend more outreach once goal is reached")
	}
	if got.Reason != "progress >= target" {
		t.Fatalf("reason = %q, want goal-reached path", got.Reason)
	}
	if got.Remaining != 0 {
		t.Fatalf("remaining = %v, want 0", got.Remaining)
	}
}

// TestDecideGoalNBACase4InsufficientPipeline verifies Test 4: large gap,
// few active prospects, but a real pool available -> Case A (expand
// outreach via Intelligence/Campaign), approval required.
func TestDecideGoalNBACase4InsufficientPipeline(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: true, CurrentValue: 4, Progress: 0, Remaining: 5, MeasurementMethod: "m"}
	got := decideGoalNBA(g, progress, goalSignals{PoolAvailable: 20, ActiveOutreach: 1})
	if got.RecommendedSkill != "sme-intelligence + sme-campaign" {
		t.Fatalf("recommended_skill = %q, want sme-intelligence + sme-campaign", got.RecommendedSkill)
	}
	if !got.ApprovalRequired {
		t.Fatal("expanding outreach must require approval before send/activate")
	}
}

// TestDecideGoalNBACase5OutreachBottleneck verifies Test 5: enough
// targets/messages, low reply rate relative to another channel -> Case B.
func TestDecideGoalNBACase5OutreachBottleneck(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: true, CurrentValue: 4, Progress: 0, Remaining: 5, MeasurementMethod: "m"}
	got := decideGoalNBA(g, progress, goalSignals{
		PoolAvailable: 1, ActiveOutreach: 10, // pipeline looks fine on its own
		ChannelGapRecommendation: "Kênh linkedin có reply rate 5.0%% — thấp hơn email (40.0%%).",
	})
	if got.RecommendedSkill != "sme-intelligence + sme-marketing" {
		t.Fatalf("recommended_skill = %q, want targeting/messaging review", got.RecommendedSkill)
	}
	if !got.ApprovalRequired {
		t.Fatal("messaging change must require approval before send")
	}
}

// TestDecideGoalNBACase6QualificationBottleneck verifies Test 6: replies
// exist (interested/requesting_info) but goal progress is still 0 ->
// Case C (Engagement/discovery), takes priority over Case B/A.
func TestDecideGoalNBACase6QualificationBottleneck(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: true, CurrentValue: 4, Progress: 0, Remaining: 5, MeasurementMethod: "m"}
	got := decideGoalNBA(g, progress, goalSignals{
		InterestedReplies30d: 3, PoolAvailable: 20, ActiveOutreach: 1,
		ChannelGapRecommendation: "some gap that should be shadowed by Case C",
	})
	if got.RecommendedSkill != "sme-engagement" {
		t.Fatalf("recommended_skill = %q, want sme-engagement (Case C must win over B/A)", got.RecommendedSkill)
	}
}

// TestDecideGoalNBACase7ProposalBottleneck verifies Test 7: qualified
// exists, proposal progression weak -> Case D, highest priority among gap
// cases (wins even with interested replies / channel gap / thin pool also
// present).
func TestDecideGoalNBACase7ProposalBottleneck(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: true, CurrentValue: 4, Progress: 0, Remaining: 5, MeasurementMethod: "m"}
	got := decideGoalNBA(g, progress, goalSignals{
		ProposalStuck: 2, InterestedReplies30d: 3, PoolAvailable: 20, ActiveOutreach: 1,
		ChannelGapRecommendation: "some gap that should be shadowed by Case D",
	})
	if got.RecommendedSkill != "sme-opportunity" {
		t.Fatalf("recommended_skill = %q, want sme-opportunity (Case D must win over C/B/A)", got.RecommendedSkill)
	}
}

// TestDecideGoalNBANoClearSignalIsHonest verifies the fallback: healthy
// pipeline, no stuck proposals, no channel gap, no stalled qualification ->
// must NOT force any of Cases A-D, reports unknown/low confidence instead.
func TestDecideGoalNBANoClearSignalIsHonest(t *testing.T) {
	g := baseGoal()
	progress := goalProgressResult{OK: true, CurrentValue: 4, Progress: 0, Remaining: 5, MeasurementMethod: "m"}
	got := decideGoalNBA(g, progress, goalSignals{ActiveOutreach: 20, PoolAvailable: 0})
	if got.RecommendedSkill != "" || got.Confidence != "low" {
		t.Fatalf("expected honest low-confidence fallback, got %+v", got)
	}
}
