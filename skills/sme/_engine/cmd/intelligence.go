package main

import (
	"fmt"
	"strings"
)

// cmdIntelligence dispatches sme-intelligence — pre-conversation, account-
// level intelligence: who to target, why, why now, likely pain.
//
//	sme-cli intelligence account <company_name_or_domain> [--org-id APOLLO_ORG_ID]
//
// This is a FACADE, not a new scoring engine (REUSE FIRST, Phase 2B): it
// orchestrates the capabilities that already exist —
//   - apolloSearchCompanyData/apolloSearchPeopleData (apollo.go) for
//     account/contact research
//   - cosmoCalculateScores/cosmoRelationshipScoreData (cosmo_ai.go) for ICP
//     and relationship scores, IF the account already has a COSMO contact
//   - effectiveProposalTiers (proposal.go) as the "relevant Rockship
//     offering" reference — no separate offerings config is created
//
// It creates no new persistent entity and does not create/mutate CRM
// records — this is a read-only research view, same discipline as
// sme-opportunity (Phase 2A).
func cmdIntelligence(args []string) {
	if len(args) == 0 {
		errOut("usage: intelligence account <company_name_or_domain> [--org-id APOLLO_ORG_ID]")
		return
	}
	switch args[0] {
	case "account":
		intelligenceAccount(args[1:])
	default:
		errOut("unknown intelligence command: " + args[0])
	}
}

// buyingSignal is a single account-level (pre-conversation) buying signal —
// NOT to be confused with sme-engagement's conversation-level intent/
// sentiment/objection taxonomy. Every field here must trace to real data;
// intelligenceExtractSignals never invents a signal without a live Apollo
// field backing it.
type buyingSignal struct {
	SignalType string `json:"signal_type"`
	Evidence   string `json:"evidence"`
	Source     string `json:"source"`
	ObservedAt string `json:"observed_at,omitempty"`
	Relevance  string `json:"relevance"`
	Confidence string `json:"confidence"`
}

// painHypothesis is explicitly a HYPOTHESIS — never presented as fact. Every
// hypothesis links back to the buyingSignal evidence that produced it.
type painHypothesis struct {
	Hypothesis         string `json:"hypothesis"`
	SupportingEvidence string `json:"supporting_evidence"`
	RelevantOffering   string `json:"relevant_offering"`
	Confidence         string `json:"confidence"`
}

type qualificationResult struct {
	Level      string   `json:"level"` // high|medium|low|insufficient_data
	Reason     string   `json:"reason"`
	Evidence   []string `json:"evidence"`
	Confidence string   `json:"confidence"`
}

func intelligenceAccount(args []string) {
	if len(args) == 0 {
		errOut("usage: intelligence account <company_name_or_domain> [--org-id APOLLO_ORG_ID]")
		return
	}
	query := args[0]
	orgID := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--org-id" && i+1 < len(args) {
			orgID = args[i+1]
			i++
		}
	}

	orgs, err := apolloSearchCompanyData(query)
	if err != nil {
		errOut(err.Error())
		return
	}

	if len(orgs) == 0 {
		okOut(map[string]interface{}{
			"account":         query,
			"found":           false,
			"icp_score":       "unknown",
			"relationship_score": "unknown",
			"signals":         []buyingSignal{},
			"pain_hypotheses": []painHypothesis{},
			"target_personas": []map[string]interface{}{},
			"qualification": qualificationResult{
				Level:      "insufficient_data",
				Reason:     "Không tìm thấy company khớp trên Apollo — không đủ dữ liệu để đánh giá",
				Evidence:   []string{},
				Confidence: "low",
			},
			"relevant_offerings": []string{},
			"confidence":         "low",
			"evidence":           []string{"Apollo search-company: 0 kết quả cho \"" + query + "\""},
			"recommended_angle":  "unknown — chưa có dữ liệu company",
		})
		return
	}

	var org apolloOrg
	if orgID != "" {
		found := false
		for _, o := range orgs {
			if o.ID == orgID {
				org = o
				found = true
				break
			}
		}
		if !found {
			errOut("org-id " + orgID + " không khớp kết quả search-company nào — gọi lại không kèm --org-id để xem danh sách candidate")
			return
		}
	} else if len(orgs) == 1 {
		org = orgs[0]
	} else {
		// Multiple candidates — never guess which one the user means (same
		// disambiguation discipline as resolveOpportunityContacts).
		candidates := make([]map[string]interface{}, 0, len(orgs))
		for _, o := range orgs {
			candidates = append(candidates, map[string]interface{}{
				"org_id": o.ID,
				"name":   o.Name,
				"domain": o.PrimaryDomain,
			})
		}
		okOut(map[string]interface{}{
			"ambiguous":  true,
			"account":    query,
			"candidates": candidates,
			"message":    fmt.Sprintf("%d company khớp trên Apollo — chọn 1 org_id rồi gọi lại `intelligence account \"%s\" --org-id <id>`, KHÔNG tự đoán.", len(orgs), query),
		})
		return
	}

	signals := intelligenceExtractSignals(org)
	pains := intelligencePainHypotheses(signals)

	people, err := apolloSearchPeopleData(org.Name, []string{"c_suite", "vp", "director"})
	if err != nil {
		people = nil // Apollo people-search failing must not block the rest of the report — target_personas just comes back empty/unknown.
	}
	personas := intelligenceTargetPersonas(people)

	// Existing COSMO relationship, if any — reuse the exact same search
	// path as sme-opportunity (cosmoContactsSearch + contactTextFilter),
	// never a new lookup mechanism.
	icpScore := "unknown"
	icpEvidence := "Không có contact hiện hữu trong CRM khớp company này, hoặc chưa có ICP segment nào được cấu hình"
	relScore := "unknown"
	var existingContactID string
	evidence := []string{}
	if matches, mErr := resolveOpportunityContacts(org.Name); mErr == nil && len(matches) > 0 {
		existingContactID = matches[0].ID
		evidence = append(evidence, fmt.Sprintf("CRM: đã có contact hiện hữu (%s, business_stage=%s)", matches[0].Name, firstNonEmpty(matches[0].BusinessStage, "unknown")))
		if scores, code, sErr := cosmoCalculateScores(existingContactID); sErr == nil && code < 400 {
			if data, ok := scores["data"].(map[string]interface{}); ok {
				if evaluated, _ := data["segments_evaluated"].(float64); evaluated > 0 {
					if p, ok := data["priority_score"].(float64); ok {
						icpScore = fmt.Sprintf("%.0f", p)
						icpEvidence = fmt.Sprintf("priority_score=%.0f (segments_evaluated=%.0f, segments_matched=%.0f)", p, evaluated, data["segments_matched"])
					}
				} else {
					icpEvidence = "COSMO chưa cấu hình ICP segment nào cho org này — priority_score hiện luôn = 0, KHÔNG có nghĩa là fit kém"
				}
			}
		}
		if rel, code, rErr := cosmoRelationshipScoreData(existingContactID); rErr == nil && code < 400 {
			if data, ok := rel["data"].(map[string]interface{}); ok {
				if hs, ok := data["health_score"].(float64); ok {
					relScore = fmt.Sprintf("%.0f", hs)
					evidence = append(evidence, fmt.Sprintf("relationship health_score=%.0f, interactions_90d=%.0f, replies_90d=%.0f", hs, data["interactions_90d"], data["replies_90d"]))
				}
			}
		}
	} else {
		evidence = append(evidence, "CRM: chưa có contact hiện hữu khớp company này (cold account)")
	}
	evidence = append(evidence, icpEvidence)
	for _, s := range signals {
		evidence = append(evidence, s.Evidence)
	}

	qual := intelligenceQualification(existingContactID != "", icpScore, signals, personas)

	tiers := effectiveProposalTiers()
	relevantOfferings := make([]string, 0, len(tiers))
	for _, t := range tiers {
		relevantOfferings = append(relevantOfferings, t.Name)
	}

	overallConfidence := "low"
	if len(signals) >= 2 || existingContactID != "" {
		overallConfidence = "medium"
	}
	if existingContactID != "" && icpScore != "unknown" {
		overallConfidence = "high"
	}

	okOut(map[string]interface{}{
		"account":             org.Name,
		"found":               true,
		"org_id":              org.ID,
		"domain":              org.PrimaryDomain,
		"icp_score":           icpScore,
		"relationship_score":  relScore,
		"existing_contact_id": existingContactID, // "" if none — never fabricated
		"signals":             signals,
		"pain_hypotheses":     pains,
		"target_personas":     personas,
		"qualification":       qual,
		"relevant_offerings":  relevantOfferings,
		"confidence":          overallConfidence,
		"evidence":            evidence,
		"recommended_angle":   intelligenceRecommendedAngle(pains, tiers),
	})
}

// intelligenceExtractSignals turns real Apollo organization fields into
// evidence-backed signals. A field being absent/zero-valued never produces
// a signal — Apollo returns 0 for both "no growth" and "field not tracked
// for this org", so only a threshold crossing (a real, checkable number) is
// treated as a signal, per the "no evidence -> no signal" rule.
func intelligenceExtractSignals(org apolloOrg) []buyingSignal {
	out := []buyingSignal{} // never nil — must marshal to [] , not null, when empty
	type window struct {
		label string
		v     float64
	}
	windows := []window{
		{"6 tháng", org.HeadcountSixMonthGrowth},
		{"12 tháng", org.HeadcountTwelveMonthGrowth},
		{"24 tháng", org.HeadcountTwentyFourMonthGrowth},
	}
	for _, w := range windows {
		if w.v >= 0.05 {
			out = append(out, buyingSignal{
				SignalType: "headcount_growth",
				Evidence:   fmt.Sprintf("%s: tăng %.1f%% nhân sự trong %s (Apollo)", org.Name, w.v*100, w.label),
				Source:     "Apollo",
				Relevance:  "medium",
				Confidence: "medium",
			})
		}
	}
	if org.PubliclyTradedSymbol != "" {
		out = append(out, buyingSignal{
			SignalType: "public_company_activity",
			Evidence:   fmt.Sprintf("Niêm yết công khai mã %s (%s)", org.PubliclyTradedSymbol, org.PubliclyTradedExchange),
			Source:     "Apollo",
			Relevance:  "low",
			Confidence: "high",
		})
	}
	if org.HasIntentSignalAccount {
		strength := "không rõ mức độ"
		if org.IntentStrength != nil && *org.IntentStrength != "" {
			strength = *org.IntentStrength
		}
		out = append(out, buyingSignal{
			SignalType: "buying_intent",
			Evidence:   fmt.Sprintf("Apollo phát hiện buying-intent signal (%s) — KHÔNG rõ đang tìm sản phẩm/vấn đề cụ thể gì", strength),
			Source:     "Apollo",
			Relevance:  "high",
			Confidence: "low",
		})
	}
	if org.OwnedByOrganization != nil && org.OwnedByOrganization.Name != "" {
		out = append(out, buyingSignal{
			SignalType: "ownership_structure",
			Evidence:   fmt.Sprintf("Thuộc tập đoàn %s", org.OwnedByOrganization.Name),
			Source:     "Apollo",
			Relevance:  "low",
			Confidence: "high",
		})
	}
	return out
}

// intelligencePainHypotheses maps signals to a DETERMINISTIC, pre-templated
// hypothesis (mirroring cosmo_plan.go's cellTemplates pattern) — never a
// free-text generation step. A signal with no mapped hypothesis template
// simply contributes no hypothesis (e.g. public_company_activity alone is
// too generic to hypothesize a specific pain from).
func intelligencePainHypotheses(signals []buyingSignal) []painHypothesis {
	out := []painHypothesis{}
	for _, s := range signals {
		switch s.SignalType {
		case "headcount_growth":
			out = append(out, painHypothesis{
				Hypothesis:         "Có thể đang chịu áp lực vận hành do mở rộng nhân sự nhanh — quy trình thủ công khó theo kịp quy mô (HYPOTHESIS, chưa xác nhận qua hội thoại)",
				SupportingEvidence: s.Evidence,
				RelevantOffering:   "Pro (growth-stage, xem opportunity/proposal pricing)",
				Confidence:         "medium",
			})
		case "buying_intent":
			out = append(out, painHypothesis{
				Hypothesis:         "Có thể đang chủ động tìm giải pháp liên quan (HYPOTHESIS — Apollo không tiết lộ đang tìm sản phẩm/vấn đề cụ thể gì, cần xác nhận qua hội thoại)",
				SupportingEvidence: s.Evidence,
				RelevantOffering:   "unknown — cần thêm evidence cụ thể trước khi gắn với 1 tier",
				Confidence:         "low",
			})
		case "ownership_structure":
			out = append(out, painHypothesis{
				Hypothesis:         "Có thể đang chuẩn hoá vận hành theo hệ thống tập đoàn mẹ (HYPOTHESIS, evidence yếu)",
				SupportingEvidence: s.Evidence,
				RelevantOffering:   "unknown",
				Confidence:         "low",
			})
		}
	}
	return out
}

// typicalBuyerTitles is a small, static, documented allow-list — not a
// scoring model. Extending it is a config/doc change, not new logic.
var typicalBuyerTitles = []string{
	"CEO", "Founder", "Co-Founder", "Chairman", "COO", "CTO",
	"VP Operations", "Head of Operations", "Director of Operations", "General Manager",
}

func intelligenceTargetPersonas(people []apolloPerson) []map[string]interface{} {
	seen := map[string]bool{}
	out := []map[string]interface{}{}
	for _, p := range people {
		key := strings.ToLower(strings.TrimSpace(p.Title))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		matched := false
		for _, t := range typicalBuyerTitles {
			if strings.EqualFold(p.Title, t) {
				matched = true
				break
			}
		}
		out = append(out, map[string]interface{}{
			"title":                          p.Title,
			"matches_typical_buyer_persona": matched,
			"source":                        "Apollo",
		})
	}
	return out
}

// intelligenceQualification is deterministic and evidence-gated — never a
// trained/ML model. unknown/insufficient evidence is never conflated with
// "disqualified": B/A/T unknown for a cold account still allows a "low" or
// "medium" qualification if signal/persona evidence exists.
func intelligenceQualification(hasExistingContact bool, icpScore string, signals []buyingSignal, personas []map[string]interface{}) qualificationResult {
	personaMatch := false
	for _, p := range personas {
		if m, _ := p["matches_typical_buyer_persona"].(bool); m {
			personaMatch = true
			break
		}
	}
	notableSignals := 0
	for _, s := range signals {
		if s.Relevance == "high" || s.Relevance == "medium" {
			notableSignals++
		}
	}

	switch {
	case hasExistingContact && icpScore != "unknown":
		return qualificationResult{
			Level:      "high",
			Reason:     "Đã có contact hiện hữu trong CRM VÀ ICP score đã được COSMO tính (segments đã cấu hình)",
			Evidence:   []string{"existing_contact=true", "icp_score=" + icpScore},
			Confidence: "medium",
		}
	case personaMatch && notableSignals >= 1:
		return qualificationResult{
			Level:      "medium",
			Reason:     "Có persona điển hình khớp VÀ ít nhất 1 signal đáng chú ý (medium/high relevance)",
			Evidence:   []string{},
			Confidence: "medium",
		}
	case personaMatch || notableSignals >= 1 || hasExistingContact:
		return qualificationResult{
			Level:      "low",
			Reason:     "Chỉ có 1 trong các yếu tố (persona/signal/relationship hiện hữu) — chưa đủ để đánh giá cao hơn",
			Evidence:   []string{},
			Confidence: "low",
		}
	default:
		return qualificationResult{
			Level:      "insufficient_data",
			Reason:     "Tìm thấy company trên Apollo nhưng không có signal, persona điển hình, hay quan hệ CRM nào đáng chú ý",
			Evidence:   []string{},
			Confidence: "low",
		}
	}
}

// intelligenceRecommendedAngle points at which existing offering/hypothesis
// combination is most defensible — never freshly-generated marketing copy.
// With no pain hypotheses at all, it says so plainly instead of guessing.
func intelligenceRecommendedAngle(pains []painHypothesis, tiers []proposalTier) string {
	if len(pains) == 0 {
		return "Chưa có đủ signal để đề xuất angle cụ thể — cần thêm thông tin (qua hội thoại hoặc enrich thêm)"
	}
	best := pains[0]
	for _, p := range pains {
		if p.Confidence == "medium" && best.Confidence != "medium" {
			best = p
		}
	}
	return fmt.Sprintf("Dẫn dắt từ: %s (evidence: %s) → nhắc tới offering: %s", best.Hypothesis, best.SupportingEvidence, best.RelevantOffering)
}
