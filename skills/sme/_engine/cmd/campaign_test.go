package main

import "testing"

// TestCampaignChannelDefaultsToEmail proves an unset/missing channel marker
// resolves to "email" — COSMO's campaign model has no channel concept and
// is implicitly email-only, so campaignActivate must never silently treat
// an unmarked campaign as "linkedin" (which would skip the Google-auth
// check entirely).
func TestCampaignChannelDefaultsToEmail(t *testing.T) {
	cases := []struct {
		name string
		view map[string]interface{}
	}{
		{"nil entity", map[string]interface{}{}},
		{"no cmetadata", map[string]interface{}{"entity": map[string]interface{}{}}},
		{"cmetadata but no client", map[string]interface{}{"entity": map[string]interface{}{
			"cmetadata": map[string]interface{}{},
		}}},
		{"client but no channel key", map[string]interface{}{"entity": map[string]interface{}{
			"cmetadata": map[string]interface{}{"client": map[string]interface{}{}},
		}}},
	}
	for _, tc := range cases {
		if got := campaignChannel(tc.view); got != "email" {
			t.Errorf("%s: campaignChannel() = %q, want %q", tc.name, got, "email")
		}
	}
}

// TestCampaignChannelReadsLinkedIn proves an explicitly set "linkedin"
// marker round-trips correctly — this is the ONLY path that should ever
// route campaignActivate away from the real COSMO status=active call.
func TestCampaignChannelReadsLinkedIn(t *testing.T) {
	view := map[string]interface{}{
		"entity": map[string]interface{}{
			"cmetadata": map[string]interface{}{
				"client": map[string]interface{}{"channel": "linkedin"},
			},
		},
	}
	if got := campaignChannel(view); got != "linkedin" {
		t.Fatalf("campaignChannel() = %q, want %q", got, "linkedin")
	}
}

// TestCampaignChannelFallsBackToTopLevel proves the fallback path (when the
// response isn't wrapped in an "entity" key) still resolves correctly —
// defensive against either GetByID response shape actually returned live.
func TestCampaignChannelFallsBackToTopLevel(t *testing.T) {
	view := map[string]interface{}{
		"cmetadata": map[string]interface{}{
			"client": map[string]interface{}{"channel": "linkedin"},
		},
	}
	if got := campaignChannel(view); got != "linkedin" {
		t.Fatalf("campaignChannel() = %q, want %q", got, "linkedin")
	}
}
