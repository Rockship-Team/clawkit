// Package template handles SKILL.md placeholder substitution.
package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Process reads SKILL.md, replaces {key} placeholders with userInputs values,
// and writes back in place.
func Process(skillDir string, userInputs map[string]string) error {
	skillPath := filepath.Join(skillDir, "SKILL.md")

	data, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read SKILL.md: %w", err)
	}

	content := string(data)
	for key, value := range userInputs {
		if value != "" {
			content = strings.ReplaceAll(content, "{"+key+"}", value)
		}
	}

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}
	return nil
}
