// Package template handles SKILL.md.tmpl processing and catalog generation.
package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Catalog defines product categories, price tiers, and best-seller flag
// for generating the catalog section in SKILL.md.
type Catalog struct {
	Categories []Category `json:"categories"`
	PriceTiers []int      `json:"price_tiers"`
	BestSeller bool       `json:"best_seller"`
}

// Category represents a product category with a folder name and label.
type Category struct {
	Folder string `json:"folder"`
	Label  string `json:"label"`
}

// LoadCatalog reads catalog.json from the skill directory.
func LoadCatalog(skillDir string) (*Catalog, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &cat, nil
}

// GenerateCatalogSection builds the catalog listing text from catalog.json.
func GenerateCatalogSection(skillDir string) (string, error) {
	cat, err := LoadCatalog(skillDir)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, c := range cat.Categories {
		lines = append(lines, fmt.Sprintf("- `%s` — %s", c.Folder, c.Label))
	}
	if len(cat.PriceTiers) > 0 {
		var prices []string
		for _, p := range cat.PriceTiers {
			prices = append(prices, fmt.Sprintf("`price-%d`", p))
		}
		lines = append(lines, fmt.Sprintf("- %s — ảnh theo giá", strings.Join(prices, ", ")))
	}
	if cat.BestSeller {
		lines = append(lines, "- `best-seller` — ảnh hoa bán chạy")
	}
	return strings.Join(lines, "\n"), nil
}

// EnsureFlowerDirs creates directories under flowers/ matching catalog.json.
func EnsureFlowerDirs(skillDir string) error {
	cat, err := LoadCatalog(skillDir)
	if err != nil {
		return nil // no catalog = nothing to do
	}
	flowersDir := filepath.Join(skillDir, "flowers")

	dirs := make([]string, 0, len(cat.Categories)+len(cat.PriceTiers)+1)
	for _, c := range cat.Categories {
		dirs = append(dirs, filepath.Join(flowersDir, c.Folder))
	}
	for _, p := range cat.PriceTiers {
		dirs = append(dirs, filepath.Join(flowersDir, fmt.Sprintf("price-%d", p)))
	}
	if cat.BestSeller {
		dirs = append(dirs, filepath.Join(flowersDir, "best-seller"))
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return nil
}

// Process reads SKILL.md, replaces placeholders with userInputs
// and the generated catalog section, and writes back in place.
func Process(skillDir string, userInputs map[string]string) error {
	skillPath := filepath.Join(skillDir, "SKILL.md")

	data, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no SKILL.md = no processing needed
		}
		return fmt.Errorf("read SKILL.md: %w", err)
	}

	content := string(data)

	// Replace user input placeholders ({key} → userInputs[key]).
	for key, value := range userInputs {
		if value != "" {
			content = strings.ReplaceAll(content, "{"+key+"}", value)
		}
	}

	// Generate and replace catalog section.
	catalogSection, err := GenerateCatalogSection(skillDir)
	if err == nil && catalogSection != "" {
		content = strings.ReplaceAll(content, "{catalogSection}", catalogSection)
	}

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}

	return nil
}

// snakeToCamel converts snake_case to camelCase (e.g. "agent_name" → "agentName").
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// ProcessTokens replaces {key} placeholders in SKILL.md, IDENTITY.md, and SOUL.md
// with values from tokens. Both snake_case and camelCase variants of each key are
// substituted so skills can use either convention. Keys not present in the file
// are silently skipped.
func ProcessTokens(skillDir string, tokens map[string]string) error {
	targets := []string{"SKILL.md", "IDENTITY.md", "SOUL.md"}
	for _, fname := range targets {
		fpath := filepath.Join(skillDir, fname)
		data, err := os.ReadFile(fpath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", fname, err)
		}
		content := string(data)
		for key, value := range tokens {
			if value == "" {
				continue
			}
			// Replace both {snake_case} and {camelCase} variants.
			content = strings.ReplaceAll(content, "{"+key+"}", value)
			if camel := snakeToCamel(key); camel != key {
				content = strings.ReplaceAll(content, "{"+camel+"}", value)
			}
		}
		if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", fname, err)
		}
	}
	return nil
}
