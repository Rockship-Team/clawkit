package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessTemplate(t *testing.T) {
	// Setup: create temp skill dir with template + catalog
	tmpDir := t.TempDir()

	// Write SKILL.md.tmpl
	tmpl := `---
name: test-skill
---
# Shop {shopName}
Email: {notifyEmailFrom}
To: {notifyEmailTo}
Pass: {notifyEmailAppPassword}
Catalog:
{catalogSection}
BaseDir: {baseDir}
`
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md.tmpl"), []byte(tmpl), 0644)

	// Write catalog.json
	catalog := `{
  "categories": [{"folder": "roses", "label": "red roses"}],
  "price_tiers": [100000, 200000],
  "best_seller": true
}`
	os.WriteFile(filepath.Join(tmpDir, "catalog.json"), []byte(catalog), 0644)

	// Run
	inputs := map[string]string{
		"shop_name":                "Test Shop",
		"notify_email_from":       "from@test.com",
		"notify_email_to":         "to@test.com",
		"notify_email_app_password": "secret123",
	}

	err := processTemplate(tmpDir, inputs)
	if err != nil {
		t.Fatalf("processTemplate failed: %v", err)
	}

	// Verify SKILL.md created
	data, err := os.ReadFile(filepath.Join(tmpDir, "SKILL.md"))
	if err != nil {
		t.Fatal("SKILL.md not created")
	}
	content := string(data)

	// Verify replacements
	assertions := map[string]string{
		"Test Shop":     "shop name",
		"from@test.com": "email from",
		"to@test.com":   "email to",
		"secret123":     "app password",
		"`roses`":       "catalog category",
		"`price-100000`": "price tier",
		"`best-seller`": "best seller",
		"{baseDir}":     "baseDir preserved",
	}
	for expected, label := range assertions {
		if !strings.Contains(content, expected) {
			t.Errorf("%s: expected %q in output", label, expected)
		}
	}

	// Verify no leftover placeholders
	leftover := []string{"{shopName}", "{notifyEmailFrom}", "{notifyEmailTo}", "{notifyEmailAppPassword}", "{catalogSection}"}
	for _, p := range leftover {
		if strings.Contains(content, p) {
			t.Errorf("placeholder %s should have been replaced", p)
		}
	}

	// Verify .tmpl removed
	if _, err := os.Stat(filepath.Join(tmpDir, "SKILL.md.tmpl")); !os.IsNotExist(err) {
		t.Error("SKILL.md.tmpl should have been removed")
	}
}

func TestProcessTemplateNoTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	// No .tmpl file — should return nil (no-op)
	err := processTemplate(tmpDir, map[string]string{})
	if err != nil {
		t.Fatalf("should not error when no template: %v", err)
	}
}

func TestGenerateCatalogSection(t *testing.T) {
	tmpDir := t.TempDir()
	catalog := `{
  "categories": [
    {"folder": "hoa-hong", "label": "roses"},
    {"folder": "hoa-lan", "label": "orchids"}
  ],
  "price_tiers": [300000, 500000],
  "best_seller": true
}`
	os.WriteFile(filepath.Join(tmpDir, "catalog.json"), []byte(catalog), 0644)

	section, err := generateCatalogSection(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(section, "`hoa-hong`") {
		t.Error("missing hoa-hong category")
	}
	if !strings.Contains(section, "`hoa-lan`") {
		t.Error("missing hoa-lan category")
	}
	if !strings.Contains(section, "`price-300000`") {
		t.Error("missing price tier 300000")
	}
	if !strings.Contains(section, "`best-seller`") {
		t.Error("missing best-seller")
	}
}

func TestEnsureFlowerDirs(t *testing.T) {
	tmpDir := t.TempDir()
	catalog := `{
  "categories": [{"folder": "roses", "label": "r"}],
  "price_tiers": [100000],
  "best_seller": true
}`
	os.WriteFile(filepath.Join(tmpDir, "catalog.json"), []byte(catalog), 0644)

	err := ensureFlowerDirs(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	dirs := []string{"flowers/roses", "flowers/price-100000", "flowers/best-seller"}
	for _, d := range dirs {
		path := filepath.Join(tmpDir, d)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("directory %s should have been created", d)
		}
	}
}
