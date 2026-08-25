package main

import "testing"

// TestIntelligenceExtractSignalsNoEvidenceNoSignal proves a zero-valued org
// (no real Apollo data — the common shape when a field is genuinely absent)
// produces zero signals, never a fabricated one.
func TestIntelligenceExtractSignalsNoEvidenceNoSignal(t *testing.T) {
	got := intelligenceExtractSignals(apolloOrg{Name: "Empty Co"})
	if len(got) != 0 {
		t.Fatalf("expected 0 signals for an org with no evidence, got %d: %+v", len(got), got)
	}
}

// TestIntelligenceExtractSignalsHeadcountGrowthThreshold proves growth below
// the 5% threshold is treated as noise (not a signal) — Apollo returns 0 for
// both "no growth" and "field not tracked", so a small positive value isn't
// reliable enough evidence to assert a real hiring/expansion signal.
func TestIntelligenceExtractSignalsHeadcountGrowthThreshold(t *testing.T) {
	below := intelligenceExtractSignals(apolloOrg{Name: "X", HeadcountSixMonthGrowth: 0.01})
	if len(below) != 0 {
		t.Fatalf("1%% growth should not produce a signal, got %+v", below)
	}
	above := intelligenceExtractSignals(apolloOrg{Name: "X", HeadcountTwelveMonthGrowth: 0.12})
	if len(above) != 1 || above[0].SignalType != "headcount_growth" {
		t.Fatalf("12%% growth should produce exactly 1 headcount_growth signal, got %+v", above)
	}
}

// TestIntelligenceExtractSignalsPublicCompany proves a literal, checkable
// fact (a ticker symbol) always produces a signal with high confidence in
// the FACT itself, even though its business relevance is low/generic.
func TestIntelligenceExtractSignalsPublicCompany(t *testing.T) {
	got := intelligenceExtractSignals(apolloOrg{Name: "X", PubliclyTradedSymbol: "ABC", PubliclyTradedExchange: "hose"})
	if len(got) != 1 || got[0].SignalType != "public_company_activity" || got[0].Confidence != "high" {
		t.Fatalf("expected 1 high-confidence public_company_activity signal, got %+v", got)
	}
}

// TestIntelligencePainHypothesesTraceToEvidence proves every hypothesis
// carries the exact signal evidence that produced it, and an unmapped
// signal type (public_company_activity alone) produces no hypothesis —
// never a generic filler guess.
func TestIntelligencePainHypothesesTraceToEvidence(t *testing.T) {
	signals := []buyingSignal{
		{SignalType: "headcount_growth", Evidence: "grew 12%"},
		{SignalType: "public_company_activity", Evidence: "ticker ABC"},
	}
	pains := intelligencePainHypotheses(signals)
	if len(pains) != 1 {
		t.Fatalf("expected exactly 1 hypothesis (public_company_activity has no template), got %d: %+v", len(pains), pains)
	}
	if pains[0].SupportingEvidence != "grew 12%" {
		t.Fatalf("hypothesis must carry its source evidence verbatim, got %q", pains[0].SupportingEvidence)
	}
}

// TestIntelligenceTargetPersonasDedupAndMatch proves duplicate titles
// collapse to one entry and typical-buyer-persona matching is exact
// (case-insensitive), never a fuzzy/invented match.
func TestIntelligenceTargetPersonasDedupAndMatch(t *testing.T) {
	people := []apolloPerson{
		{Title: "ceo"},
		{Title: "CEO"}, // duplicate (case-insensitive title text, same role)
		{Title: "Staff Accountant"},
	}
	got := intelligenceTargetPersonas(people)
	if len(got) != 2 {
		t.Fatalf("expected 2 deduped-by-exact-title entries, got %d: %+v", len(got), got)
	}
	foundCEOMatch := false
	for _, p := range got {
		if p["title"] == "ceo" {
			if m, _ := p["matches_typical_buyer_persona"].(bool); m {
				foundCEOMatch = true
			}
		}
		if p["title"] == "Staff Accountant" {
			if m, _ := p["matches_typical_buyer_persona"].(bool); m {
				t.Fatalf("Staff Accountant must not match the typical buyer persona list")
			}
		}
	}
	if !foundCEOMatch {
		t.Fatal("expected \"ceo\" to match the typical buyer persona list case-insensitively")
	}
}

// TestIntelligenceQualificationNeverDisqualifiesOnUnknown proves a cold
// account (no existing CRM contact, unknown ICP score) with no signals or
// persona evidence lands at insufficient_data, NOT some invented negative
// "disqualified" level — unknown must never be treated as a disqualifier.
func TestIntelligenceQualificationNeverDisqualifiesOnUnknown(t *testing.T) {
	got := intelligenceQualification(false, "unknown", nil, nil)
	if got.Level != "insufficient_data" {
		t.Fatalf("expected insufficient_data for a cold account with no evidence, got %q", got.Level)
	}
}

// TestIntelligenceQualificationHighRequiresRealICPScore proves "high" is
// only reachable once COSMO has actually computed an ICP score for an
// existing contact — not merely because a contact exists.
func TestIntelligenceQualificationHighRequiresRealICPScore(t *testing.T) {
	withoutScore := intelligenceQualification(true, "unknown", nil, nil)
	if withoutScore.Level == "high" {
		t.Fatalf("existing contact alone (icp_score still unknown) must not reach \"high\", got %+v", withoutScore)
	}
	withScore := intelligenceQualification(true, "42", nil, nil)
	if withScore.Level != "high" {
		t.Fatalf("existing contact + real icp_score should reach \"high\", got %+v", withScore)
	}
}
