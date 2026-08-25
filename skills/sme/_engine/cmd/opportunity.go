package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// cmdOpportunity dispatches the lightweight Opportunity view.
//
//	sme-cli opportunity view <contact_id_or_search_query>
//	sme-cli opportunity risk-list [--max-pages N]
//
// This is a PURE READ/AGGREGATION layer over data that already exists in
// COSMO (business_stage, next_step, relationship stats) and over the
// deterministic pipeline-cell classification already built in cosmo_plan.go
// (buildPlanCells/classify). It creates NO new persistent entity — no
// "opportunities" table, no new source of truth. It also folds in the
// LOW_PRIORITY signal reminder/SKILL.md previously computed ad hoc at
// render time (see mergedRisk below), so that logic lives in exactly one
// place instead of being duplicated between cosmo_plan.go and prose in
// reminder/SKILL.md.
func cmdOpportunity(args []string) {
	if len(args) == 0 {
		errOut("usage: opportunity view <contact_id_or_query>|risk-list")
		return
	}
	switch args[0] {
	case "view":
		opportunityView(args[1:])
	case "risk-list":
		opportunityRiskList(args[1:])
	default:
		errOut("unknown opportunity command: " + args[0])
	}
}

// opportunityContact is a resolved COSMO contact plus the classification
// data resolveContact/buildOpportunityView need. It reuses planContact
// (cosmo_plan.go) directly rather than re-declaring the same fields.
type opportunityContact struct {
	planContact
	StageLabel     string
	Interactions90d int
	Replies90d     int
}

// resolveOpportunityContacts searches COSMO via the exact same endpoint
// fetchAllContacts/flattenContact already use (/v2/contacts/search), so
// contact parsing stays in one place. If query looks like a UUID, only an
// exact id match is returned (never a fuzzy guess); otherwise every match
// COSMO's search returns is returned for the caller to disambiguate.
func resolveOpportunityContacts(query string) ([]opportunityContact, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"query":    query,
		"page":     1,
		"pageSize": 25,
	})
	raw, code, err := cosmoRequest("POST", "/v2/contacts/search", body)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", code, string(raw))
	}
	var resp struct {
		Data struct {
			List []struct {
				Entity map[string]interface{} `json:"entity"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var out []opportunityContact
	wantID := looksLikeUUID(query)
	for _, item := range resp.Data.List {
		pc := flattenContact(item.Entity)
		if wantID && pc.ID != query {
			continue
		}
		oc := opportunityContact{planContact: pc}
		if sl, ok := item.Entity["stage_label"].(string); ok {
			oc.StageLabel = sl
		}
		if rel, ok := item.Entity["relationship"].(map[string]interface{}); ok {
			if v, ok := rel["interactions_90d"].(float64); ok {
				oc.Interactions90d = int(v)
			}
			if v, ok := rel["replies_90d"].(float64); ok {
				oc.Replies90d = int(v)
			}
		}
		out = append(out, oc)
	}
	return out, nil
}

// mergedRisk is the SINGLE place that merges cosmo_plan.go's cell-based
// pipeline risk with the LOW_PRIORITY override reminder/SKILL.md used to
// compute ad hoc in prose. It never touches buildPlanCells/classify's
// callers (cosmo daily-plan stays untouched and fast — this is a separate,
// on-demand code path, not part of the 10-minute PIPELINE_WATCH hot loop).
type riskResult struct {
	Cell        string `json:"cell"`
	Label       string `json:"label"`
	LowPriority bool   `json:"low_priority"`
	Note        string `json:"note"`
}

func mergedRisk(oc opportunityContact, ctx planContext) riskResult {
	cell := classify(oc.planContact, ctx)
	tpl, hasTpl := cellTemplates[cell]

	// LOW_PRIORITY override: >=3 net outbound (interactions minus replies)
	// in the last 90 days with zero replies is the same signal reminder's
	// render-time rule used, sourced entirely from data already fetched by
	// flattenContact/relationship — no extra COSMO calls.
	netOutbound := oc.Interactions90d - oc.Replies90d
	stale := oc.Replies90d == 0 && netOutbound >= 3
	outreachInProgress := cell == "PROPOSAL_HOT" || cell == "PROPOSAL_STUCK" || cell == "PROPOSAL_GHOST" ||
		cell == "ENGAGED_WARM" || cell == "ENGAGED_COLD" || cell == "QUALIFIED_OPEN" || cell == "CAMPAIGN_SENT_NO_REPLY"

	if stale && outreachInProgress {
		return riskResult{
			Cell:        cell,
			Label:       "LOW_PRIORITY — đã follow-up nhiều lần, im lặng",
			LowPriority: true,
			Note:        fmt.Sprintf("%d tương tác gần đây (90 ngày), 0 reply — deprioritize khỏi briefing tự động, xem qua khi hỏi trực tiếp", netOutbound),
		}
	}
	if !hasTpl {
		return riskResult{Cell: "", Label: "unknown", Note: "chưa có tín hiệu rủi ro rõ ràng cho contact này"}
	}
	return riskResult{Cell: cell, Label: tpl.Name, Note: tpl.Why}
}

// opportunityQualification is a deterministic mapping off business_stage —
// NOT a new scoring model. Real BANT/qualification-criteria scoring doesn't
// exist anywhere in COSMO today (confirmed during the Phase 2 audit), so
// this only reports what stage-progression already implies, never invents
// a score.
func opportunityQualification(stage string) string {
	switch stage {
	case "", "NEW":
		return "not_started"
	case "ENGAGED":
		return "in_progress"
	case "QUALIFIED", "PROPOSAL", "WON":
		return "qualified"
	case "LOST", "DROPPED":
		return "closed_lost"
	default:
		return "unknown"
	}
}

// opportunityProposalReadiness never auto-advances past what COSMO's own
// stage already reflects — it will not report "ready" off a single
// positive/interested signal. This mirrors the Phase 1.1 rule: interested +
// positive != proposal-ready.
func opportunityProposalReadiness(stage string, classifications []replyClassification) (string, string) {
	switch stage {
	case "PROPOSAL", "WON":
		return "ready", "stage hiện tại đã ở PROPOSAL/WON — proposal readiness đã được xác lập trước đó"
	}
	for _, c := range classifications {
		if c.Intent == "requesting_info" || (c.Intent == "interested" && c.Sentiment == "positive") {
			// Evidence exists that the contact engaged, but that alone is
			// NOT sufficient — explicit ask for a quote/proposal is required.
			return "not_ready", "có tín hiệu quan tâm (" + c.Intent + "/" + c.Sentiment + ") nhưng CHƯA có xác nhận khách muốn nhận báo giá/proposal cụ thể — cần thêm bước qualification/discovery"
		}
	}
	return "not_ready", "chưa có evidence nào cho thấy khách sẵn sàng nhận proposal"
}

func opportunityView(args []string) {
	if len(args) == 0 {
		errOut("usage: opportunity view <contact_id_or_query>")
		return
	}
	query := args[0]

	contacts, err := resolveOpportunityContacts(query)
	if err != nil {
		errOut("resolve contact: " + err.Error())
		return
	}
	if len(contacts) == 0 {
		okOut(map[string]interface{}{
			"found": false,
			"message": "Không tìm thấy contact nào khớp — kiểm tra lại tên/công ty/contact_id, KHÔNG suy đoán.",
		})
		return
	}
	if len(contacts) > 1 && !looksLikeUUID(query) {
		var candidates []map[string]string
		for _, c := range contacts {
			candidates = append(candidates, map[string]string{"id": c.ID, "name": c.Name, "company": c.Company})
		}
		okOut(map[string]interface{}{
			"found":      true,
			"ambiguous":  true,
			"candidates": candidates,
			"message":    fmt.Sprintf("%d contact khớp — chọn 1 contact_id cụ thể rồi gọi lại `opportunity view <id>`, KHÔNG tự đoán.", len(contacts)),
		})
		return
	}

	oc := contacts[0]
	ctx := fetchPlanContext()
	risk := mergedRisk(oc, ctx)

	orgID := defaultOrgID()
	classifications, _ := queryReplyClassificationsByContact(orgID, oc.ID)
	readiness, readinessNote := opportunityProposalReadiness(oc.BusinessStage, classifications)

	evidence := []string{
		fmt.Sprintf("business_stage (COSMO) = %q", oc.BusinessStage),
		fmt.Sprintf("idle_days = %d, interactions_30d = %d, interactions_90d = %d, replies_90d = %d",
			oc.IdleDays, oc.Interactions30d, oc.Interactions90d, oc.Replies90d),
	}
	if oc.LastOutcome != "" {
		evidence = append(evidence, "last_outcome (COSMO) = "+oc.LastOutcome)
	}
	if oc.ConversationState != "" {
		evidence = append(evidence, "conversation_state (COSMO) = "+oc.ConversationState)
	}
	for _, c := range classifications {
		evidence = append(evidence, fmt.Sprintf(
			"LinkedIn reply (%s) qua sme-outreach → engagement taxonomy: intent=%s, sentiment=%s, objection=%s",
			c.ClassifiedAt, c.Intent, c.Sentiment, c.Objection))
	}

	knownPain := "unknown — COSMO chưa có field pain/need riêng, chưa track được (xem Phase 2 audit)"

	recommended := "Chưa có action template cho cell này — xem trực tiếp qua sme-crm/sme-engagement."
	if tpl, ok := cellTemplates[risk.Cell]; ok {
		recommended = tpl.Action.CTA
	}

	okOut(map[string]interface{}{
		"found": true,
		"contact": map[string]interface{}{
			"id":      oc.ID,
			"name":    oc.Name,
			"company": oc.Company,
		},
		"current_stage": map[string]interface{}{
			"business_stage": oc.BusinessStage,
			"stage_label":    oc.StageLabel,
		},
		"qualification_status": opportunityQualification(oc.BusinessStage),
		"known_pain_need":      knownPain,
		"next_step":            firstNonEmpty(oc.NextStep, "unknown"),
		"risk":                 risk,
		"proposal_readiness": map[string]interface{}{
			"status": readiness,
			"note":   readinessNote,
		},
		"bant": map[string]string{
			"budget":    "unknown",
			"authority":  "unknown",
			"timeline":   "unknown",
			"note":       "COSMO không có field BANT — không suy luận, luôn trả unknown cho tới khi có evidence thật",
		},
		"evidence":           evidence,
		"recommended_action": recommended,
	})
}

// opportunityRiskList is the batch equivalent of mergedRisk, reused by
// reminder's explicit "ai đang low priority" / "stale leads" trigger instead
// of that skill maintaining its own separate render-time rule. Deliberately
// NOT called from cosmo_plan.go's daily-plan (which runs every 10 minutes
// via PIPELINE_WATCH) — this does one relationship-stats pass per contact
// and is meant to be invoked on demand only.
func opportunityRiskList(args []string) {
	maxPages := 8
	for i := 0; i < len(args); i++ {
		if args[i] == "--max-pages" && i+1 < len(args) {
			i++
			fmt.Sscanf(args[i], "%d", &maxPages)
		}
	}
	contacts, _, err := fetchAllContacts(maxPages)
	if err != nil {
		errOut("fetch contacts: " + err.Error())
		return
	}
	// fetchAllContacts already gives us relationship data via flattenContact
	// (Interactions30d), but not the 90d/replies_90d split opportunityContact
	// needs — re-search would be N+1 API calls, so risk-list uses the 30d
	// proxy (Interactions30d + ConversationState) instead of the 90d one
	// opportunity view uses. This is a coarser, cheaper signal appropriate
	// for a batch scan; opportunity view (single contact) uses the richer one.
	ctx := fetchPlanContext()
	var lowPriority []map[string]interface{}
	for _, c := range contacts {
		cell := classify(c, ctx)
		outreachInProgress := cell == "PROPOSAL_HOT" || cell == "PROPOSAL_STUCK" || cell == "PROPOSAL_GHOST" ||
			cell == "ENGAGED_WARM" || cell == "ENGAGED_COLD" || cell == "QUALIFIED_OPEN" || cell == "CAMPAIGN_SENT_NO_REPLY"
		stale := c.ConversationState == "NO_REPLY" && c.IdleDays >= 15
		if outreachInProgress && stale {
			lowPriority = append(lowPriority, map[string]interface{}{
				"id": c.ID, "name": c.Name, "company": c.Company,
				"original_cell": cell, "idle_days": c.IdleDays,
			})
		}
	}
	okOut(map[string]interface{}{
		"low_priority": lowPriority,
		"count":        len(lowPriority),
		"note":         "render-layer label — KHÔNG ghi DB, KHÔNG đổi business_stage. Dùng khi user hỏi explicit ('ai đang low priority'), KHÔNG render trong briefing tự động.",
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
