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
	for _, c := range cat.Categories {
		os.MkdirAll(filepath.Join(flowersDir, c.Folder), 0755)
	}
	for _, p := range cat.PriceTiers {
		os.MkdirAll(filepath.Join(flowersDir, fmt.Sprintf("price-%d", p)), 0755)
	}
	if cat.BestSeller {
		os.MkdirAll(filepath.Join(flowersDir, "best-seller"), 0755)
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

	// Replace user input placeholders.
	replacements := map[string]string{
		"{shopName}":               userInputs["shop_name"],
		"{notifyEmailFrom}":        userInputs["notify_email_from"],
		"{notifyEmailTo}":          userInputs["notify_email_to"],
		"{notifyEmailAppPassword}": userInputs["notify_email_app_password"],
	}

	for placeholder, value := range replacements {
		if value != "" {
			content = strings.ReplaceAll(content, placeholder, value)
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
