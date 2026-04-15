package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProfileYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "profile.yaml")

	content := `# This is a comment
shop_name: Shop Hoa Tuoi
emoji: "🌸"
locale: 'vi'

description: A flower shop
empty_val:
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vals, err := parseProfileYAML(path)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key, want string
	}{
		{"shop_name", "Shop Hoa Tuoi"},
		{"emoji", "🌸"},
		{"locale", "vi"},
		{"description", "A flower shop"},
		{"empty_val", ""},
	}
	for _, tc := range tests {
		if got := vals[tc.key]; got != tc.want {
			t.Errorf("key %q: got %q, want %q", tc.key, got, tc.want)
		}
	}

	if _, ok := vals["# This is a comment"]; ok {
		t.Error("comment line should not be parsed as a key")
	}
}

func TestApplyProfile(t *testing.T) {
	// Set up a mock skill directory with a profile.
	skillDir := t.TempDir()

	// Base catalog.json.
	baseCatalog := `{"categories":[]}`
	os.WriteFile(filepath.Join(skillDir, "catalog.json"), []byte(baseCatalog), 0644)

	// Create profile directory.
	profileDir := filepath.Join(skillDir, "profiles", "test-shop")
	os.MkdirAll(profileDir, 0755)

	// Profile catalog.json (should overwrite base).
	profileCatalog := `{"categories":[{"folder":"roses","label":"Roses"}]}`
	os.WriteFile(filepath.Join(profileDir, "catalog.json"), []byte(profileCatalog), 0644)

	// Profile values.
	os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte("shop_name: Test Shop\n"), 0644)

	// Profile bootstrap-files.
	overridesDir := filepath.Join(profileDir, "bootstrap-files")
	os.MkdirAll(overridesDir, 0755)
	os.WriteFile(filepath.Join(overridesDir, "IDENTITY.md"), []byte("# Test Shop"), 0644)

	// Run applyProfile.
	vals, err := applyProfile(skillDir, "test-shop")
	if err != nil {
		t.Fatal(err)
	}

	// Check profile values were parsed.
	if vals["shop_name"] != "Test Shop" {
		t.Errorf("shop_name: got %q, want %q", vals["shop_name"], "Test Shop")
	}

	// Check catalog.json was overlaid.
	data, err := os.ReadFile(filepath.Join(skillDir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != profileCatalog {
		t.Errorf("catalog.json not overlaid: got %q", string(data))
	}

	// Check bootstrap-files was overlaid.
	data, err = os.ReadFile(filepath.Join(skillDir, "bootstrap-files", "IDENTITY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Test Shop" {
		t.Errorf("IDENTITY.md not overlaid: got %q", string(data))
	}

	// Check profiles/ directory was cleaned up.
	if _, err := os.Stat(filepath.Join(skillDir, "profiles")); !os.IsNotExist(err) {
		t.Error("profiles/ directory should have been removed after overlay")
	}
}

func TestApplyProfileNotFound(t *testing.T) {
	skillDir := t.TempDir()

	_, err := applyProfile(skillDir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}
