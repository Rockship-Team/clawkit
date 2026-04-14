package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcess(t *testing.T) {
	tests := []struct {
		name      string
		tmpl      string
		catalog   string
		inputs    map[string]string
		wantIn    []string
		wantNotIn []string
	}{
		{
			name:    "replaces user inputs and catalog section",
			tmpl:    "# {shop_name}\nFrom: {notify_email}\n{catalogSection}\n{baseDir}",
			catalog: `{"categories":[{"folder":"roses","label":"red roses"}],"price_tiers":[100000],"best_seller":true}`,
			inputs: map[string]string{
				"shop_name":    "Test Shop",
				"notify_email": "a@b.com",
			},
			wantIn:    []string{"Test Shop", "a@b.com", "`roses`", "`price-100000`", "`best-seller`", "{baseDir}"},
			wantNotIn: []string{"{shop_name}", "{notify_email}", "{catalogSection}"},
		},
		{
			name:    "empty inputs leave placeholders",
			tmpl:    "# {shop_name}",
			catalog: `{"categories":[],"price_tiers":[],"best_seller":false}`,
			inputs:  map[string]string{},
			wantIn:  []string{"{shop_name}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(tt.tmpl), 0644)
			os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(tt.catalog), 0644)

			if err := Process(dir, tt.inputs); err != nil {
				t.Fatalf("Process() error: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
			if err != nil {
				t.Fatal("SKILL.md not created")
			}
			content := string(data)

			for _, want := range tt.wantIn {
				if !strings.Contains(content, want) {
					t.Errorf("expected %q in output", want)
				}
			}
			for _, notWant := range tt.wantNotIn {
				if strings.Contains(content, notWant) {
					t.Errorf("unexpected %q in output", notWant)
				}
			}
		})
	}
}

func TestProcessNoTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := Process(dir, map[string]string{}); err != nil {
		t.Fatalf("should not error when no template: %v", err)
	}
}

func TestGenerateCatalogSection(t *testing.T) {
	tests := []struct {
		name    string
		catalog string
		wantIn  []string
	}{
		{
			name:    "full catalog",
			catalog: `{"categories":[{"folder":"hoa-hong","label":"roses"},{"folder":"hoa-lan","label":"orchids"}],"price_tiers":[300000,500000],"best_seller":true}`,
			wantIn:  []string{"`hoa-hong`", "`hoa-lan`", "`price-300000`", "`price-500000`", "`best-seller`"},
		},
		{
			name:    "no best seller",
			catalog: `{"categories":[{"folder":"a","label":"b"}],"price_tiers":[],"best_seller":false}`,
			wantIn:  []string{"`a`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(tt.catalog), 0644)

			section, err := GenerateCatalogSection(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.wantIn {
				if !strings.Contains(section, want) {
					t.Errorf("expected %q in section", want)
				}
			}
		})
	}
}

func TestEnsureImageDirs(t *testing.T) {
	dir := t.TempDir()
	catalog := `{"categories":[{"folder":"roses","label":"r"}],"price_tiers":[100000],"best_seller":true}`
	os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(catalog), 0644)

	if err := EnsureImageDirs(dir); err != nil {
		t.Fatal(err)
	}

	// No schema.json → defaults to "products" subdirectory.
	dirs := []string{"products/roses", "products/price-100000", "products/best-seller"}
	for _, d := range dirs {
		t.Run(d, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(dir, d)); os.IsNotExist(err) {
				t.Errorf("directory %s should have been created", d)
			}
		})
	}
}
