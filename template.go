package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Catalog struct {
	Categories []struct {
		Folder string `json:"folder"`
		Label  string `json:"label"`
	} `json:"categories"`
	PriceTiers []int `json:"price_tiers"`
	BestSeller bool  `json:"best_seller"`
}

func loadCatalog(skillDir string) (*Catalog, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "catalog.json"))
	if err != nil {
		return nil, err
	}
	var cat Catalog
	return &cat, json.Unmarshal(data, &cat)
}

func generateCatalogSection(skillDir string) (string, error) {
	cat, err := loadCatalog(skillDir)
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

func ensureFlowerDirs(skillDir string) error {
	cat, err := loadCatalog(skillDir)
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

func processTemplate(skillDir string, userInputs map[string]string) error {
	tmplPath := filepath.Join(skillDir, "SKILL.md.tmpl")
	outPath := filepath.Join(skillDir, "SKILL.md")

	data, err := os.ReadFile(tmplPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no template = no processing needed
		}
		return err
	}

	content := string(data)

	// Replace user input placeholders
	replacements := map[string]string{
		"{shopName}":               userInputs["shop_name"],
		"{notifyEmailFrom}":       userInputs["notify_email_from"],
		"{notifyEmailTo}":         userInputs["notify_email_to"],
		"{notifyEmailAppPassword}": userInputs["notify_email_app_password"],
	}

	for placeholder, value := range replacements {
		if value != "" {
			content = strings.ReplaceAll(content, placeholder, value)
		}
	}

	// Generate and replace catalog section
	catalogSection, err := generateCatalogSection(skillDir)
	if err == nil && catalogSection != "" {
		content = strings.ReplaceAll(content, "{catalogSection}", catalogSection)
	}

	// Write final SKILL.md
	err = os.WriteFile(outPath, []byte(content), 0644)
	if err != nil {
		return err
	}

	// Remove template file
	os.Remove(tmplPath)
	return nil
}
