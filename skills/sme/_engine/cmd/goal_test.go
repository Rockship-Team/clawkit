package main

import "testing"

// TestGoalMetricStageOnlyKnownReliableTypes locks in exactly which metric
// types have a real progress source — any type not in this map (meetings,
// revenue, custom, or a typo) MUST fall back to "unknown" in goalView,
// never a guessed number. This test exists so adding a new metric type
// without a real data source can't silently happen.
func TestGoalMetricStageOnlyKnownReliableTypes(t *testing.T) {
	want := map[string]string{
		"qualified_leads": "QUALIFIED",
		"proposals":       "PROPOSAL",
		"contracts":       "WON",
	}
	if len(goalMetricStage) != len(want) {
		t.Fatalf("goalMetricStage has %d entries, want %d — got %+v", len(goalMetricStage), len(want), goalMetricStage)
	}
	for k, v := range want {
		if goalMetricStage[k] != v {
			t.Errorf("goalMetricStage[%q] = %q, want %q", k, goalMetricStage[k], v)
		}
	}
	for _, unreliable := range []string{"meetings", "revenue", "custom", "contacts"} {
		if _, ok := goalMetricStage[unreliable]; ok {
			t.Errorf("goalMetricStage must NOT have an entry for %q — no reliable source exists, must stay unknown", unreliable)
		}
	}
}
