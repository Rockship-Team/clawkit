package main

import (
	"fmt"
	"time"
)

// goalReviewCheck is the Phase 3C anti-noise gate for the proactive Goal
// Review cron (see reminder/SKILL.md "GOAL_REVIEW MODE"). It reuses the
// exact same progress/NBA computation as `goal view`/`goal next-action` —
// no second implementation of progress or bottleneck detection — and adds
// exactly one new thing: a stored snapshot of what was last NOTIFIED for
// this goal, so repeat cron runs can tell "nothing meaningful changed"
// from "this needs a briefing" without an LLM judgment call for the core
// gate. The orchestrator/reminder skill still writes the actual prose and
// may add memory/ActionLog context on top — this only decides whether to
// speak at all.
//
//	sme-cli goal review-check <id>
func goalReviewCheck(args []string) {
	if len(args) == 0 {
		errOut("usage: goal review-check <id>")
		return
	}
	g, err := loadGoal(args[0])
	if err != nil {
		errOut(err.Error())
		return
	}
	if g == nil {
		errOut("không tìm thấy goal id " + args[0])
		return
	}
	if g.Status != "active" {
		okOut(map[string]interface{}{
			"goal_id": g.ID, "notify": false,
			"message": "Goal này đã ở trạng thái '" + g.Status + "' — không cần review nữa.",
		})
		return
	}

	progress := computeGoalProgress(g)
	daysRemaining := -1
	if g.Deadline != "" {
		if dl, perr := time.Parse("2006-01-02", g.Deadline); perr == nil {
			daysRemaining = int(time.Until(dl).Hours() / 24)
		}
	}

	// Reuse the exact same NBA gathering goalNextAction uses — one signal
	// path, not a second one for the cron.
	signals := goalSignals{}
	if progress.OK && progress.Progress < int(g.TargetValue) {
		contacts, _, cErr := fetchAllContacts(goalMaxPages)
		if cErr != nil {
			errOut("fetch contacts: " + cErr.Error())
			return
		}
		ctx := fetchPlanContext()
		for _, c := range contacts {
			switch classify(c, ctx) {
			case "NEW_APOLLO_FULL", "NEW_APOLLO_LINKEDIN", "NEW_EVENT":
				signals.PoolAvailable++
			case "ENGAGED_WARM", "ENGAGED_COLD", "QUALIFIED_OPEN", "CAMPAIGN_SENT_NO_REPLY":
				signals.ActiveOutreach++
			case "PROPOSAL_STUCK", "PROPOSAL_GHOST":
				signals.ProposalStuck++
			}
		}
		classifications, _, _ := outreachClassifiedData(30)
		for _, r := range classifications {
			intent := toString(r["intent"])
			if intent == "interested" || intent == "requesting_info" {
				signals.InterestedReplies30d++
			}
		}
		funnelRows, _, fErr := outreachFunnelData(30)
		if fErr == nil {
			recs := analyticsRecommendations(analyticsChannelComparison(funnelRows))
			if len(recs) > 0 {
				signals.ChannelGapRecommendation = recs[0]
			}
		}
	}
	nba := decideGoalNBA(g, progress, signals)

	prev := loadGoalReviewState(g.ID)
	notify, reasons := decideGoalReviewNotify(prev, progress, nba, daysRemaining)

	out := map[string]interface{}{
		"goal_id": g.ID, "notify": notify, "reasons": reasons,
		"progress": structToMap(progress), "nba": structToMap(nba),
		"days_remaining": daysRemaining,
	}
	if notify {
		saveGoalReviewState(g.ID, progress, nba, daysRemaining, prev)
	}
	okOut(out)
}

// goalReviewState is the snapshot of what was last NOTIFIED (not merely
// last checked) for a goal — the cron calls review-check far more often
// than it should actually speak, so comparing against "last spoken about"
// rather than "last checked" is what makes the anti-noise gate work.
type goalReviewState struct {
	LastProgress          int
	LastBottleneck        string
	LastRecommendedAction string
	LastApprovalRequired  bool
	DeadlineWarned        bool
	GoalReachedNotified   bool
}

func ensureGoalReviewStateTable() {
	mustDB().Exec(`CREATE TABLE IF NOT EXISTS goal_review_state (
		goal_id                 TEXT PRIMARY KEY,
		last_progress           INTEGER NOT NULL DEFAULT 0,
		last_bottleneck         TEXT NOT NULL DEFAULT '',
		last_recommended_action TEXT NOT NULL DEFAULT '',
		last_approval_required INTEGER NOT NULL DEFAULT 0,
		deadline_warned         INTEGER NOT NULL DEFAULT 0,
		goal_reached_notified   INTEGER NOT NULL DEFAULT 0,
		updated_at              TEXT NOT NULL
	)`)
}

func loadGoalReviewState(goalID string) *goalReviewState {
	ensureGoalReviewStateTable()
	row, err := queryOne(`SELECT * FROM goal_review_state WHERE goal_id = ?`, goalID)
	if err != nil || row == nil {
		return nil
	}
	return &goalReviewState{
		LastProgress:          int(toInt64(row["last_progress"])),
		LastBottleneck:        toString(row["last_bottleneck"]),
		LastRecommendedAction: toString(row["last_recommended_action"]),
		LastApprovalRequired:  toInt64(row["last_approval_required"]) != 0,
		DeadlineWarned:        toInt64(row["deadline_warned"]) != 0,
		GoalReachedNotified:   toInt64(row["goal_reached_notified"]) != 0,
	}
}

func saveGoalReviewState(goalID string, progress goalProgressResult, nba nbaResult, daysRemaining int, prev *goalReviewState) {
	ensureGoalReviewStateTable()
	deadlineWarned := prev != nil && prev.DeadlineWarned
	if daysRemaining >= 0 && daysRemaining <= 3 {
		deadlineWarned = true
	}
	goalReached := prev != nil && prev.GoalReachedNotified
	if nba.Reason == "progress >= target" {
		goalReached = true
	}
	approvalRequired := 0
	if nba.ApprovalRequired {
		approvalRequired = 1
	}
	deadlineWarnedInt := 0
	if deadlineWarned {
		deadlineWarnedInt = 1
	}
	goalReachedInt := 0
	if goalReached {
		goalReachedInt = 1
	}
	mustDB().Exec(`
		INSERT INTO goal_review_state (goal_id, last_progress, last_bottleneck, last_recommended_action, last_approval_required, deadline_warned, goal_reached_notified, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(goal_id) DO UPDATE SET
			last_progress = excluded.last_progress,
			last_bottleneck = excluded.last_bottleneck,
			last_recommended_action = excluded.last_recommended_action,
			last_approval_required = excluded.last_approval_required,
			deadline_warned = excluded.deadline_warned,
			goal_reached_notified = excluded.goal_reached_notified,
			updated_at = excluded.updated_at
	`, goalID, progress.Progress, nba.DetectedBottleneck, nba.RecommendedAction, approvalRequired, deadlineWarnedInt, goalReachedInt, vnNowISO())
}

// decideGoalReviewNotify is the PURE anti-noise gate (Phase 3C Part 4/11) —
// no DB/network calls, fully unit-testable. First match wins; if nothing
// matches, stay silent. This never re-derives progress/NBA logic — it only
// diffs the already-computed result against the last-notified snapshot.
func decideGoalReviewNotify(prev *goalReviewState, progress goalProgressResult, nba nbaResult, daysRemaining int) (bool, []string) {
	var reasons []string

	if prev == nil {
		reasons = append(reasons, "lần review đầu tiên cho goal này")
	} else {
		if progress.OK && progress.Progress != prev.LastProgress {
			reasons = append(reasons, fmt.Sprintf("progress thay đổi: %d → %d", prev.LastProgress, progress.Progress))
		}
		if nba.DetectedBottleneck != prev.LastBottleneck {
			reasons = append(reasons, "bottleneck/NBA thay đổi so với lần review trước")
		}
		if nba.ApprovalRequired && !prev.LastApprovalRequired {
			reasons = append(reasons, "có action mới cần approval")
		}
	}

	goalReachedNotified := prev != nil && prev.GoalReachedNotified
	if nba.Reason == "progress >= target" && !goalReachedNotified {
		reasons = append(reasons, "goal đã đạt target")
	}
	deadlineWarned := prev != nil && prev.DeadlineWarned
	if daysRemaining >= 0 && daysRemaining <= 3 && !deadlineWarned {
		reasons = append(reasons, fmt.Sprintf("deadline còn %d ngày", daysRemaining))
	}

	if len(reasons) == 0 {
		return false, []string{"không có thay đổi đáng chú ý kể từ lần review trước"}
	}
	return true, reasons
}
