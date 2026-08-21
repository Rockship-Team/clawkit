package main

import (
	"strings"
	"testing"
)

func TestParseTXTContacts(t *testing.T) {
	input := `# header comment
Alice Nguyen — alice@acme.com
Bob Tran - bob@example.com
  Charlie Pham — charlie@foo.co
David Le—david@no.space

# skip
nodelim no-delim-no-space
Eve — no-at-sign
`
	contacts, errs := parseTXTContacts(input, "test")
	if len(contacts) != 4 {
		t.Fatalf("want 4 contacts, got %d (%v)", len(contacts), contacts)
	}
	want := map[string]string{
		"Alice Nguyen":  "alice@acme.com",
		"Bob Tran":      "bob@example.com",
		"Charlie Pham":  "charlie@foo.co",
		"David Le":      "david@no.space",
	}
	for _, c := range contacts {
		n, _ := c["name"].(string)
		e, _ := c["email"].(string)
		if want[n] != e {
			t.Errorf("contact %q: want email %q, got %q", n, want[n], e)
		}
		if c["source"] != "test" {
			t.Errorf("contact %q: source not propagated", n)
		}
	}
	if len(errs) < 1 {
		t.Errorf("expected at least 1 parse error for invalid lines, got %d", len(errs))
	}
}

func TestParseLumaCSV(t *testing.T) {
	csv := `api_id,name,first_name,last_name,email,phone_number,approval_status,ticket_name,utm_source
evt_1,,Quan,Nguyen,quan@example.com,+84900000001,approved,VIP,telegram
evt_2,JOON,,,joon@foo.co,,pending,,
evt_3,No Email Row,,,,,,,,
evt_4,,Tuan,Pham,tuan@bar.io,,approved,Standard,
`
	contacts, errs := parseLumaCSV(strings.NewReader(csv), "evt")
	if len(contacts) != 3 {
		t.Fatalf("want 3 valid contacts, got %d (errs=%v)", len(contacts), errs)
	}
	// Row 1: first+last build
	if contacts[0]["name"] != "Quan Nguyen" {
		t.Errorf("row1 name: got %v", contacts[0]["name"])
	}
	if contacts[0]["phone"] != "+84900000001" {
		t.Errorf("row1 phone: got %v", contacts[0]["phone"])
	}
	custom1 := contacts[0]["custom_fields"].(map[string]string)
	if custom1["approval_status"] != "approved" || custom1["ticket_type"] != "VIP" || custom1["utm_source"] != "telegram" {
		t.Errorf("row1 custom: %v", custom1)
	}
	// Row 2: fallback to 'name' field
	if contacts[1]["name"] != "JOON" {
		t.Errorf("row2 name: got %v", contacts[1]["name"])
	}
	if _, has := contacts[1]["phone"]; has {
		t.Errorf("row2 should not have phone")
	}
	// Row 4 (index 2 after skipping invalid rows)
	if contacts[2]["name"] != "Tuan Pham" {
		t.Errorf("row4 name: got %v", contacts[2]["name"])
	}
}

func TestParseGenericCSV(t *testing.T) {
	csv := `Name,Email,Phone,Company,Notes
Alice,alice@a.com,+1-555,Acme,VIP
Bob,bob@b.co,,Beta,
NoEmail,,+1,Gamma,skip me
`
	contacts, _ := parseGenericCSV(strings.NewReader(csv), "gen")
	if len(contacts) != 2 {
		t.Fatalf("want 2, got %d", len(contacts))
	}
	if contacts[0]["name"] != "Alice" || contacts[0]["email"] != "alice@a.com" {
		t.Errorf("row1: %+v", contacts[0])
	}
	if contacts[0]["company"] != "Acme" {
		t.Errorf("row1 company: got %v", contacts[0]["company"])
	}
	cf := contacts[0]["custom_fields"].(map[string]string)
	if cf["notes"] != "VIP" {
		t.Errorf("row1 notes in custom_fields: %v", cf)
	}
}

func TestSplitTXTLine(t *testing.T) {
	cases := []struct {
		line, name, email string
		ok                bool
	}{
		{"Alice — alice@a.com", "Alice", "alice@a.com", true},
		{"Bob - bob@b.co", "Bob", "bob@b.co", true},
		{"Charlie—c@foo", "Charlie", "c@foo", true},
		{"Jean-Paul — jp@foo", "Jean-Paul", "jp@foo", true},
		{"trailingonly-", "trailingonly", "", true}, // lastIndex on "-" at end
		{"nodelimnospace", "", "", false},
	}
	for _, c := range cases {
		n, e, ok := splitTXTLine(c.line)
		if ok != c.ok {
			t.Errorf("splitTXTLine(%q) ok = %v, want %v", c.line, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if n != c.name || e != c.email {
			t.Errorf("splitTXTLine(%q) = (%q, %q), want (%q, %q)", c.line, n, e, c.name, c.email)
		}
	}
}
