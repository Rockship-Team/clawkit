package main

import "testing"

func TestOutreachFingerprintStableAndUnique(t *testing.T) {
	a := outreachFingerprint("default", "linkedin", "connection_request_sent", "https://www.linkedin.com/in/foo/")
	b := outreachFingerprint("default", "linkedin", "connection_request_sent", "https://www.linkedin.com/in/foo/")
	if a != b {
		t.Errorf("same inputs must produce the same fingerprint, got %q vs %q", a, b)
	}

	diffKey := outreachFingerprint("default", "linkedin", "connection_request_sent", "https://www.linkedin.com/in/bar/")
	if a == diffKey {
		t.Errorf("different profile URLs must not collide: %q", a)
	}

	diffType := outreachFingerprint("default", "linkedin", "connection_request_received", "https://www.linkedin.com/in/foo/")
	if a == diffType {
		t.Errorf("different event types must not collide: %q", a)
	}

	diffOrg := outreachFingerprint("acme", "linkedin", "connection_request_sent", "https://www.linkedin.com/in/foo/")
	if a == diffOrg {
		t.Errorf("different org_id must not collide: %q", a)
	}
}

func TestParseCard(t *testing.T) {
	cases := []struct {
		name         string
		lines        []string
		wantName     string
		wantHeadline string
	}{
		{
			name:         "sent invitation card",
			lines:        []string{"Violet Tran", "Branch Manager at TPBank", "Sent 4 hours ago", "Withdraw"},
			wantName:     "Violet Tran",
			wantHeadline: "Branch Manager at TPBank",
		},
		{
			name:         "received invitation card with duplicated name and mutual connections line",
			lines:        []string{"Dang Le Hoang Vinh", "Dang Le Hoang Vinh", "Data & BI Analyst", "Trung and 43 other mutual connections", "Ignore", "Accept"},
			wantName:     "Dang Le Hoang Vinh",
			wantHeadline: "Data & BI Analyst",
		},
		{
			name:         "no headline available",
			lines:        []string{"Someone", "Sent 4 hours ago", "Withdraw"},
			wantName:     "Someone",
			wantHeadline: "",
		},
	}
	for _, tc := range cases {
		gotName, gotHeadline := parseCard(tc.lines)
		if gotName != tc.wantName {
			t.Errorf("%s: name = %q, want %q", tc.name, gotName, tc.wantName)
		}
		if gotHeadline != tc.wantHeadline {
			t.Errorf("%s: headline = %q, want %q", tc.name, gotHeadline, tc.wantHeadline)
		}
	}
}

func TestParseMessageCard(t *testing.T) {
	cases := []struct {
		name         string
		lines        []string
		wantName     string
		wantSnippet  string
		wantOutbound bool
	}{
		{
			name:         "outbound with status line",
			lines:        []string{"Status is reachable", "Vy Nguyễn", "4:03 PM", "4:03 PM", "You: Chào chị Vy, rất vui được kết nối"},
			wantName:     "Vy Nguyễn",
			wantSnippet:  "Chào chị Vy, rất vui được kết nối",
			wantOutbound: true,
		},
		{
			name:         "outbound with trailing press-return line",
			lines:        []string{"Chien Thang Le", "4:00 PM", "4:00 PM", "You: Chào anh Chiến Thắng", ". Press return to go to conversation details"},
			wantName:     "Chien Thang Le",
			wantSnippet:  "Chào anh Chiến Thắng",
			wantOutbound: true,
		},
		{
			name:         "inbound reply (no You: prefix)",
			lines:        []string{"Status is online", "Someone", "9:15 AM", "9:15 AM", "Cảm ơn anh, để em xem qua rồi phản hồi nhé"},
			wantName:     "Someone",
			wantSnippet:  "Cảm ơn anh, để em xem qua rồi phản hồi nhé",
			wantOutbound: false,
		},
	}
	for _, tc := range cases {
		gotName, gotSnippet, gotOutbound := parseMessageCard(tc.lines)
		if gotName != tc.wantName {
			t.Errorf("%s: name = %q, want %q", tc.name, gotName, tc.wantName)
		}
		if gotSnippet != tc.wantSnippet {
			t.Errorf("%s: snippet = %q, want %q", tc.name, gotSnippet, tc.wantSnippet)
		}
		if gotOutbound != tc.wantOutbound {
			t.Errorf("%s: outbound = %v, want %v", tc.name, gotOutbound, tc.wantOutbound)
		}
	}
}
