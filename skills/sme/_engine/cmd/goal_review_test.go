package main

import "testing"

// TestGoalReviewNotifyFirstRun verifies the very first review-check for a
// goal always notifies — there is no "last notified" snapshot to diff
// against yet.
func TestGoalReviewNotifyFirstRun(t *testing.T) {
	progress := goalProgressResult{OK: true, Progress: 2, Remaining: 3}
	nba := nbaResult{DetectedBottleneck: "none", RecommendedAction: "a"}
	notify, reasons := decideGoalReviewNotify(nil, progress, nba, 6)
	if !notify {
		t.Fatal("first-ever review-check must notify")
	}
	if len(reasons) == 0 {
		t.Fatal("must explain why it notified")
	}
}

// TestGoalReviewNoNotifyWhenNothingChanged is Phase 3C brief Test A:
// active goal, progress unchanged, bottleneck/NBA unchanged, no approval
// flip, deadline not newly close -> NO notification.
func TestGoalReviewNoNotifyWhenNothingChanged(t *testing.T) {
	prev := &goalReviewState{
		LastProgress: 2, LastBottleneck: "none — goal chưa đạt nhưng chưa rõ bottleneck",
		LastRecommendedAction: "a", LastApprovalRequired: false,
	}
	progress := goalProgressResult{OK: true, Progress: 2, Remaining: 3}
	nba := nbaResult{DetectedBottleneck: prev.LastBottleneck, RecommendedAction: "a", ApprovalRequired: false}
	notify, reasons := decideGoalReviewNotify(prev, progress, nba, 6)
	if notify {
		t.Fatalf("expected no notification when nothing changed, got reasons=%v", reasons)
	}
}

// TestGoalReviewNotifyOnMeaningfulProgressChange is Test B: a new
// interested reply shifts the NBA (bottleneck/recommended_action changes)
// -> must notify.
func TestGoalReviewNotifyOnMeaningfulProgressChange(t *testing.T) {
	prev := &goalReviewState{LastProgress: 0, LastBottleneck: "unknown — không phát hiện bottleneck rõ ràng từ dữ liệu hiện có"}
	progress := goalProgressResult{OK: true, Progress: 0, Remaining: 5}
	nba := nbaResult{
		DetectedBottleneck: "2 reply gần đây (30 ngày) có intent interested/requesting_info, nhưng progress goal vẫn = 0",
		RecommendedAction:  "Ưu tiên discovery/follow-up...",
	}
	notify, reasons := decideGoalReviewNotify(prev, progress, nba, 6)
	if !notify {
		t.Fatal("a new interested-reply bottleneck must trigger a notification")
	}
	if len(reasons) == 0 {
		t.Fatal("must cite the bottleneck change as the reason")
	}
}

// TestGoalReviewNoNotifyOnIrrelevantActivity is Test C: outreach/campaign
// activity increases but the goal's progress/bottleneck/NBA/approval are
// unaffected -> no notification. Modeled by holding every NBA-relevant
// field constant, since decideGoalNBA itself only reacts to signals that
// matter — this proves the review gate doesn't add its own extra noise
// on top.
func TestGoalReviewNoNotifyOnIrrelevantActivity(t *testing.T) {
	prev := &goalReviewState{LastProgress: 1, LastBottleneck: "none", LastRecommendedAction: "a"}
	progress := goalProgressResult{OK: true, Progress: 1, Remaining: 4}
	nba := nbaResult{DetectedBottleneck: "none", RecommendedAction: "a", ApprovalRequired: false}
	notify, _ := decideGoalReviewNotify(prev, progress, nba, 6)
	if notify {
		t.Fatal("unrelated activity increase with unchanged goal state must not trigger a notification")
	}
}

// TestGoalReviewNotifiesOnceOnGoalReached is Test D: goal reached must
// notify, but only once — a second review-check after GoalReachedNotified
// is already set must not notify again for the same reason (it may still
// notify for a genuinely new reason, but not re-announce "goal reached").
func TestGoalReviewNotifiesOnceOnGoalReached(t *testing.T) {
	progress := goalProgressResult{OK: true, Progress: 5, Remaining: 0}
	nba := nbaResult{Reason: "progress >= target", DetectedBottleneck: "none — goal đã đạt target", RecommendedAction: "complete"}

	notify1, reasons1 := decideGoalReviewNotify(nil, progress, nba, 6)
	if !notify1 {
		t.Fatal("goal-reached must notify the first time")
	}
	foundGoalReached := false
	for _, r := range reasons1 {
		if r == "goal đã đạt target" {
			foundGoalReached = true
		}
	}
	if !foundGoalReached {
		t.Fatalf("expected a goal-reached reason, got %v", reasons1)
	}

	// Simulate having saved state after that first notification.
	prev := &goalReviewState{
		LastProgress: 5, LastBottleneck: nba.DetectedBottleneck, LastRecommendedAction: nba.RecommendedAction,
		GoalReachedNotified: true,
	}
	notify2, reasons2 := decideGoalReviewNotify(prev, progress, nba, 6)
	if notify2 {
		t.Fatalf("goal-reached must not re-announce once already notified with nothing else changed, got %v", reasons2)
	}
}

// TestGoalReviewNotifiesOnApprovalRequiredFlip is Test E: an NBA that
// newly requires approval must notify, even if progress itself hasn't
// moved.
func TestGoalReviewNotifiesOnApprovalRequiredFlip(t *testing.T) {
	prev := &goalReviewState{LastProgress: 0, LastBottleneck: "b", LastRecommendedAction: "a", LastApprovalRequired: false}
	progress := goalProgressResult{OK: true, Progress: 0, Remaining: 5}
	nba := nbaResult{DetectedBottleneck: "b", RecommendedAction: "a", ApprovalRequired: true}
	notify, reasons := decideGoalReviewNotify(prev, progress, nba, 6)
	if !notify {
		t.Fatal("a newly-approval-required NBA must notify even with bottleneck/action text unchanged")
	}
	if len(reasons) == 0 {
		t.Fatal("must cite the approval requirement as the reason")
	}
}

// TestGoalReviewWarnsOnceForApproachingDeadline verifies the deadline
// proximity rule fires once (deadline_warned latches), not every single
// day once inside the window.
func TestGoalReviewWarnsOnceForApproachingDeadline(t *testing.T) {
	prev := &goalReviewState{LastProgress: 2, LastBottleneck: "b", LastRecommendedAction: "a", DeadlineWarned: false}
	progress := goalProgressResult{OK: true, Progress: 2, Remaining: 3}
	nba := nbaResult{DetectedBottleneck: "b", RecommendedAction: "a"}

	notify1, reasons1 := decideGoalReviewNotify(prev, progress, nba, 2)
	if !notify1 {
		t.Fatalf("deadline <= 3 days must notify the first time, got %v", reasons1)
	}

	prevWarned := &goalReviewState{LastProgress: 2, LastBottleneck: "b", LastRecommendedAction: "a", DeadlineWarned: true}
	notify2, reasons2 := decideGoalReviewNotify(prevWarned, progress, nba, 1)
	if notify2 {
		t.Fatalf("deadline proximity must not re-notify once already warned with nothing else changed, got %v", reasons2)
	}
}

// TestGoalNBAContinuityDedupesIdenticalDoneRecommendation is Phase 3C
// Part 10: NBA X suggested yesterday, marked done, no new evidence today
// -> must not be treated as a fresh priority.
func TestGoalNBAContinuityDedupesIdenticalDoneRecommendation(t *testing.T) {
	c := goalNBAContinuity{PreviouslySuggested: true, PreviousStatus: "done", SameAsLast: true}
	note := decideContinuityNote(c)
	if note == "" {
		t.Fatal("must explain that this is a repeat of an already-done recommendation")
	}
}

// TestGoalNBAContinuitySkippedStillSurfacesWithContext: a previously
// skipped recommendation may resurface, but with an honest note — not
// silently repeated as if it were new, and not silently suppressed either.
func TestGoalNBAContinuitySkippedStillSurfacesWithContext(t *testing.T) {
	c := goalNBAContinuity{PreviouslySuggested: true, PreviousStatus: "skipped", SameAsLast: true}
	note := decideContinuityNote(c)
	if note == "" {
		t.Fatal("must mention the prior skip")
	}
}

// TestGoalNBAContinuityNewEvidenceNoNote: when the recommendation text
// differs from the last one logged (new evidence/case), no continuity
// note should suppress or flag it — it's genuinely new.
func TestGoalNBAContinuityNewEvidenceNoNote(t *testing.T) {
	c := goalNBAContinuity{PreviouslySuggested: true, PreviousStatus: "done", SameAsLast: false}
	if note := decideContinuityNote(c); note != "" {
		t.Fatalf("a genuinely different recommendation must not carry a repeat note, got %q", note)
	}
}

// TestGoalNBAContinuityNoPriorSuggestion: first-ever NBA for a goal has
// no continuity concern at all.
func TestGoalNBAContinuityNoPriorSuggestion(t *testing.T) {
	c := goalNBAContinuity{}
	if note := decideContinuityNote(c); note != "" {
		t.Fatalf("no prior suggestion must not produce a continuity note, got %q", note)
	}
}
