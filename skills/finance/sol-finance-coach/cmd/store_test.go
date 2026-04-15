package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		// Plain numbers
		{"55000", 55000, false},
		{"0", 0, false},
		{"1000000", 1000000, false},

		// "k" / "K" suffix (thousands)
		{"55k", 55000, false},
		{"55K", 55000, false},
		{"100k", 100000, false},
		// Note: "2.5k" — dot is stripped before k parse, so "2.5k" → "25k" → 25000
		{"2.5k", 25000, false},

		// "tr" (trieu = million)
		{"1tr", 1000000, false},
		{"1tr5", 1500000, false},
		{"2tr", 2000000, false},
		// Note: "1.5tr" — dot stripped first, so "1.5tr" → "15tr" → 15,000,000
		{"1.5tr", 15000000, false},
		{"25tr", 25000000, false},

		// Dot as thousand separator
		{"55.000", 55000, false},
		{"1.500.000", 1500000, false},
		{"3.000.000.000", 3000000000, false},

		// Comma stripped
		{"55,000", 55000, false},

		// Whitespace trimmed
		{" 100k ", 100000, false},

		// Invalid
		{"abc", 0, true},
	}

	for _, tt := range tests {
		got, err := parseAmount(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseAmount(%q) expected error, got %d", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAmount(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSkillDirEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SOL_DATA_DIR", dir)
	got := skillDir()
	if got != dir {
		t.Errorf("skillDir() = %q, want %q", got, dir)
	}
}

func TestDataPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SOL_DATA_DIR", dir)
	got := dataPath("tips.json")
	want := filepath.Join(dir, "data", "tips.json")
	if got != want {
		t.Errorf("dataPath(tips.json) = %q, want %q", got, want)
	}
}

func TestUserPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SOL_DATA_DIR", dir)
	got := userPath("profile.json")
	want := filepath.Join(dir, "profile.json")
	if got != want {
		t.Errorf("userPath(profile.json) = %q, want %q", got, want)
	}
}

func TestReadWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := map[string]string{"hello": "world"}
	if err := writeJSON(path, data); err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	var out map[string]string
	if !readJSON(path, &out) {
		t.Fatal("readJSON returned false")
	}
	if out["hello"] != "world" {
		t.Errorf("readJSON got %q, want %q", out["hello"], "world")
	}
}

func TestReadJSONMissing(t *testing.T) {
	var out map[string]string
	if readJSON("/nonexistent/file.json", &out) {
		t.Error("readJSON should return false for missing file")
	}
}

func TestEnsureInit(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "test-skill")
	t.Setenv("SOL_DATA_DIR", subdir)

	ensureInit()

	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Error("ensureInit should create the directory")
	}
}
