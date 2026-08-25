package main

import (
	"encoding/json"
	"fmt"
)

// goalNextAction is the Phase 3B goal-aware Next Best Action — Level 3 in
// the NBA hierarchy (Level 0 hardcoded cell hints and Level 1 single-skill
// recommendations already exist in cosmo_plan.go/opportunity.go and are
// untouched). This is a THIN decision tree over data that already exists
// (fetchAllContacts/classify, outreach funnel/classifications) — no new
// scoring model, no AI planning engine. All branching logic lives in the
// pure decideGoalNBA below; this function only gathers the real signals
// and hands them over, so the decision tree itself is unit-testable
// without a DB/COSMO connection.
//
//	sme-cli goal next-action <goal_id>
func goalNextAction(args []string) {
	if len(args) == 0 {
		errOut("usage: goal next-action <id>")
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
			"goal_id": g.ID,
			"message": "Goal này đã ở trạng thái '" + g.Status + "' — không cần next-action nữa.",
		})
		return
	}

	progress := computeGoalProgress(g)
	signals := goalSignals{}
	if progress.OK && progress.Progress < int(g.TargetValue) {
		// Reuse the exact same cell classification opportunity.go/analytics.go
		// already run — one fetchAllContacts pass, not a second implementation.
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

	result := decideGoalNBA(g, progress, signals)
	logGoalNBA(g, result.RecommendedAction)
	okOut(structToMap(result))
}

// goalSignals are the real, already-computable inputs decideGoalNBA
// branches on — gathered once in goalNextAction, never re-derived twice.
type goalSignals struct {
	PoolAvailable            int    // NEW_APOLLO_FULL/NEW_APOLLO_LINKEDIN/NEW_EVENT — contactable, not yet in active outreach
	ActiveOutreach           int    // ENGAGED_WARM/COLD, QUALIFIED_OPEN, CAMPAIGN_SENT_NO_REPLY
	ProposalStuck            int    // PROPOSAL_STUCK/PROPOSAL_GHOST
	InterestedReplies30d     int    // classified replies, intent=interested/requesting_info, last 30 days
	ChannelGapRecommendation string // analyticsRecommendations()[0] if a real channel gap exists, else ""
}

// nbaResult is the one canonical NBA output shape — every branch fills the
// same fields so callers never have to special-case which "kind" of
// recommendation came back.
type nbaResult struct {
	GoalID             string      `json:"goal_id"`
	CurrentProgress    interface{} `json:"current_progress"`
	Remaining          interface{} `json:"remaining"`
	DetectedBottleneck string      `json:"detected_bottleneck"`
	RecommendedAction  string      `json:"recommended_action"`
	RecommendedSkill   string      `json:"recommended_skill"`
	Evidence           []string    `json:"evidence"`
	Confidence         string      `json:"confidence"`
	ApprovalRequired   bool        `json:"approval_required"`
	Reason             string      `json:"reason"`
}

// decideGoalNBA is the PURE decision tree (Cases A-E from the Phase 3B
// brief) — no DB/network calls, fully unit-testable. Priority order,
// first match wins:
//
//  1. progress unmeasurable -> honest "unknown", no bottleneck guess
//  2. progress >= target (Case E) -> recommend completion, never more outreach
//  3. proposal stuck (Case D) -> closer to goal than a new lead, highest priority gap
//  4. replies exist but progress=0 (Case C) -> qualification bottleneck
//  5. real channel reply-rate gap (Case B) -> targeting/message review
//  6. active pipeline thin vs. remaining gap, pool available (Case A) -> expand outreach
//  7. no case matches -> honest fallback, never forces a pick
func decideGoalNBA(g *goalRecord, progress goalProgressResult, s goalSignals) nbaResult {
	if !progress.OK {
		return nbaResult{
			GoalID: g.ID, CurrentProgress: "unknown", Remaining: "unknown",
			DetectedBottleneck: "unknown — không đo được progress cho metric_type này",
			RecommendedAction: "Chưa đo được progress cho goal này (metric \"" + g.MetricType + "\" không có nguồn dữ liệu tin cậy, hoặc thiếu baseline). " +
				"Không thể đưa NBA goal-aware — xem gợi ý chung qua `sme-cli cosmo daily-plan` hoặc `sme-cli analytics summary` thay vì goal-aware.",
			RecommendedSkill: "",
			Evidence:         []string{progress.MeasurementMethod},
			Confidence:       "low",
			ApprovalRequired: false,
			Reason:           "metric_type không có nguồn dữ liệu tin cậy hoặc thiếu baseline_value",
		}
	}

	if progress.Progress >= int(g.TargetValue) {
		return nbaResult{
			GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: 0,
			DetectedBottleneck: "none — goal đã đạt target",
			RecommendedAction: fmt.Sprintf(
				"Goal đã đạt target (%d/%d). Đề xuất xác nhận hoàn thành — KHÔNG tiếp tục tạo thêm outreach cho goal này. Gọi `sme-cli goal complete %s` nếu anh xác nhận.",
				progress.Progress, g.TargetValue, g.ID),
			RecommendedSkill: "orchestrator",
			Evidence:         []string{progress.MeasurementMethod},
			Confidence:       "high",
			ApprovalRequired: false,
			Reason:           "progress >= target",
		}
	}

	remaining := progress.Remaining

	// Case D — proposal bottleneck. Checked first among the "gap" cases: a
	// stuck deal already in PROPOSAL is closer to the goal than a brand new
	// lead, so it's the higher-value action to surface.
	if s.ProposalStuck > 0 {
		return nbaResult{
			GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: remaining,
			DetectedBottleneck: fmt.Sprintf("%d deal đang ở PROPOSAL nhưng bị stuck/ghost (>=8 ngày không phản hồi)", s.ProposalStuck),
			RecommendedAction: fmt.Sprintf(
				"Kiểm tra lại %d deal đang stuck ở proposal trước khi tìm lead mới — xem `sme-cli opportunity risk-list`, quyết định nudge lại hoặc chuyển LOST cho từng deal.",
				s.ProposalStuck),
			RecommendedSkill: "sme-opportunity",
			Evidence:         []string{fmt.Sprintf("PROPOSAL_STUCK/PROPOSAL_GHOST count = %d (cosmo_plan.go classify(), cùng cơ chế cosmo daily-plan)", s.ProposalStuck)},
			Confidence:       "medium",
			ApprovalRequired: false,
			Reason:           "proposal đang nghẽn — xử lý deal gần đích trước khi mở rộng thêm lead mới",
		}
	}

	// Case C — replies exist but qualification stalled: recent positive
	// intent classified, yet goal progress hasn't moved at all.
	if s.InterestedReplies30d >= 2 && progress.Progress == 0 {
		return nbaResult{
			GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: remaining,
			DetectedBottleneck: fmt.Sprintf("%d reply gần đây (30 ngày) có intent interested/requesting_info, nhưng progress goal vẫn = 0", s.InterestedReplies30d),
			RecommendedAction:  "Ưu tiên discovery/follow-up cho các contact vừa reply tích cực trước khi tìm lead mới — xem `sme-cli outreach classified --days 30`, chuyển sang sme-engagement để soạn follow-up/đặt meeting.",
			RecommendedSkill:   "sme-engagement",
			Evidence:           []string{fmt.Sprintf("%d classified reply intent=interested/requesting_info trong 30 ngày (outreach_reply_classifications), goal progress=0", s.InterestedReplies30d)},
			Confidence:         "medium",
			ApprovalRequired:   false,
			Reason:             "có tín hiệu quan tâm nhưng chưa chuyển thành qualified — nghẽn ở bước qualification, không phải thiếu lead",
		}
	}

	// Case B — reply rate gap between channels (reuse analytics.go's own
	// relative-comparison recommendation, never a second benchmark).
	if s.ChannelGapRecommendation != "" {
		return nbaResult{
			GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: remaining,
			DetectedBottleneck: "Reply rate chênh lệch đáng kể giữa các channel (xem evidence)",
			RecommendedAction:  s.ChannelGapRecommendation + " Đề xuất: sme-intelligence re-check targeting cho segment hiện tại, sme-marketing soạn lại messaging angle. CHƯA activate/gửi gì.",
			RecommendedSkill:   "sme-intelligence + sme-marketing",
			Evidence:           []string{s.ChannelGapRecommendation},
			Confidence:         "medium",
			ApprovalRequired:   true,
			Reason:             "reply rate thấp ở 1 kênh so với kênh khác — cần review targeting/message trước khi outreach thêm, không nên tiếp tục gửi theo cách cũ",
		}
	}

	// Case A — insufficient active pipeline relative to the remaining gap,
	// with an available untouched pool to draw from.
	if s.ActiveOutreach < remaining*2 && s.PoolAvailable > 0 {
		suggestCount := s.PoolAvailable
		if remaining*3 < suggestCount {
			suggestCount = remaining * 3
		}
		return nbaResult{
			GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: remaining,
			DetectedBottleneck: fmt.Sprintf("Còn thiếu %d %s, hiện chỉ %d contact đang active outreach, trong khi %d contact chưa được liên hệ (pool NEW_*)", remaining, g.MetricType, s.ActiveOutreach, s.PoolAvailable),
			RecommendedAction: fmt.Sprintf(
				"Mở rộng outreach sang khoảng %d contact tiếp theo trong pool chưa liên hệ (cell NEW_APOLLO_FULL/NEW_APOLLO_LINKEDIN/NEW_EVENT — xem `sme-cli cosmo daily-plan`). Chuẩn bị qua sme-intelligence (research) + sme-campaign (draft) — KHÔNG tự activate/gửi.",
				suggestCount),
			RecommendedSkill: "sme-intelligence + sme-campaign",
			Evidence:         []string{fmt.Sprintf("pool NEW_* = %d, active outreach (ENGAGED_WARM/COLD, QUALIFIED_OPEN, CAMPAIGN_SENT_NO_REPLY) = %d, remaining goal = %d", s.PoolAvailable, s.ActiveOutreach, remaining)},
			Confidence:       "medium",
			ApprovalRequired: true,
			Reason:           "active pipeline không đủ so với phần goal còn thiếu, nhưng có sẵn pool chưa khai thác",
		}
	}

	// Honest fallback — no case matched clearly, never force a pick.
	return nbaResult{
		GoalID: g.ID, CurrentProgress: progress.Progress, Remaining: remaining,
		DetectedBottleneck: "unknown — không phát hiện bottleneck rõ ràng từ dữ liệu hiện có",
		RecommendedAction:  "Chưa đủ tín hiệu để đề xuất 1 hành động cụ thể. Xem `sme-cli analytics summary` và `sme-cli opportunity risk-list` để tự đánh giá thêm.",
		RecommendedSkill:   "",
		Evidence: []string{fmt.Sprintf(
			"pool=%d active=%d proposal_stuck=%d interested_replies_30d=%d progress=%d/%d",
			s.PoolAvailable, s.ActiveOutreach, s.ProposalStuck, s.InterestedReplies30d, progress.Progress, g.TargetValue)},
		Confidence:       "low",
		ApprovalRequired: false,
		Reason:           "không case nào trong 4 bottleneck case khớp rõ ràng",
	}
}

// structToMap round-trips a typed result through JSON so okOut's
// map[string]interface{} convention (used everywhere else in this engine)
// stays uniform, without hand-duplicating every field name into a map
// literal at each call site.
func structToMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

// logGoalNBA records the recommendation through the EXACT same
// action_suggestions table `action-log suggest/done/rate` already use
// (Part 7 — traceability only, no new learning engine, no self-tuning off
// this data yet). contact_name is repurposed as a short goal label since
// this recommendation is goal-level, not contact-level — action-log's
// schema is reused as-is, not extended.
func logGoalNBA(g *goalRecord, action string) {
	ensureActionLogTable()
	label := g.GoalText
	if len(label) > 60 {
		label = label[:60] + "…"
	}
	actionSuggestRecord(g.ID, "Goal: "+label, action, "goal_nba")
}
