package main

import "testing"

func countRow(channel, eventType string, count int) map[string]interface{} {
	return map[string]interface{}{"channel": channel, "event_type": eventType, "count": int64(count)}
}

// TestAnalyticsChannelComparisonReplyRateGating proves reply_rate is only
// computed when messages sent > 0 — a channel with replies but zero
// recorded "message_sent" events (denominator missing) must report
// "insufficient_data", never a fabricated or divide-by-zero rate.
func TestAnalyticsChannelComparisonReplyRateGating(t *testing.T) {
	rows := []map[string]interface{}{
		countRow("linkedin", "message_sent", 20),
		countRow("linkedin", "message_reply_received", 5),
		countRow("email", "message_reply_received", 3), // no message_sent recorded for email
	}
	got := analyticsChannelComparison(rows)
	byChannel := map[string]map[string]interface{}{}
	for _, c := range got {
		byChannel[c["channel"].(string)] = c
	}
	if byChannel["linkedin"]["reply_rate"] != "25.0%" {
		t.Fatalf("linkedin reply_rate = %v, want 25.0%%", byChannel["linkedin"]["reply_rate"])
	}
	if byChannel["email"]["reply_rate"] != "insufficient_data" {
		t.Fatalf("email reply_rate = %v, want insufficient_data (no message_sent denominator)", byChannel["email"]["reply_rate"])
	}
}

// TestAnalyticsRecommendationsRequiresVolumeAndGap proves no recommendation
// fires below the minimum-volume threshold, and none fires when the gap
// between channels is within normal noise (<10 points) — recommendations
// must be grounded in a real, meaningful difference, not the first channel
// that happens to have a lower number.
func TestAnalyticsRecommendationsRequiresVolumeAndGap(t *testing.T) {
	lowVolume := []map[string]interface{}{
		{"channel": "linkedin", "messages": 2, "reply_rate": "0.0%"},
		{"channel": "email", "messages": 3, "reply_rate": "66.6%"},
	}
	if got := analyticsRecommendations(lowVolume); len(got) != 0 {
		t.Fatalf("expected no recommendation below minVolume, got %v", got)
	}

	smallGap := []map[string]interface{}{
		{"channel": "linkedin", "messages": 20, "reply_rate": "25.0%"},
		{"channel": "email", "messages": 20, "reply_rate": "30.0%"},
	}
	if got := analyticsRecommendations(smallGap); len(got) != 0 {
		t.Fatalf("expected no recommendation for a <10pt gap, got %v", got)
	}

	bigGap := []map[string]interface{}{
		{"channel": "linkedin", "messages": 20, "reply_rate": "5.0%"},
		{"channel": "email", "messages": 20, "reply_rate": "40.0%"},
	}
	got := analyticsRecommendations(bigGap)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 recommendation for a big gap, got %v", got)
	}
}

// TestAnalyticsRecommendationsSingleChannelNoComparison proves a single
// channel (nothing to compare against) never produces a recommendation,
// even with a very low reply rate — there's no evidence it's actually bad
// without a comparison point.
func TestAnalyticsRecommendationsSingleChannelNoComparison(t *testing.T) {
	single := []map[string]interface{}{
		{"channel": "linkedin", "messages": 50, "reply_rate": "1.0%"},
	}
	if got := analyticsRecommendations(single); len(got) != 0 {
		t.Fatalf("expected no recommendation with only 1 comparable channel, got %v", got)
	}
}
