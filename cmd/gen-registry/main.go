// Command gen-registry scans skills/*/SKILL.md and skills/*/config.json
// to generate registry.json. This is the canonical way to keep the registry
// in sync with skill definitions.
//
// SKILL.md provides OpenClaw-native fields: name, description.
// config.json provides clawkit-specific fields: version, requires_bins,
// setup_prompts, exclude.
//
// Usage:
//
//	go run ./cmd/gen-registry            # generate registry.json
//	go run ./cmd/gen-registry -check     # exit 1 if registry.json is outdated
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillConfig is the per-skill config.json file containing clawkit-specific
// metadata that does not belong in OpenClaw's SKILL.md frontmatter.
type SkillConfig struct {
	Version      string        `json:"version"`
	RequiresBins []string      `json:"requires_bins,omitempty"`
	SetupPrompts []SetupPrompt `json:"setup_prompts,omitempty"`
	Exclude []string      `json:"exclude,omitempty"`
}

// SetupPrompt defines an interactive prompt shown during clawkit install.
type SetupPrompt struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
}

// SkillEntry combines data from SKILL.md (name, description) and
// config.json (everything else) for a single skill.
type SkillEntry struct {
	Name        string
	Description string
	Config      SkillConfig
}

// RegistrySkill is the per-skill entry in registry.json.
type RegistrySkill struct {
	Version      string        `json:"version"`
	Description  string        `json:"description"`
	RequiresBins []string      `json:"requires_bins,omitempty"`
	SetupPrompts []SetupPrompt `json:"setup_prompts,omitempty"`
	Exclude []string      `json:"exclude,omitempty"`
}

// Registry is the top-level structure of registry.json.
type Registry struct {
	Skills map[string]RegistrySkill `json:"skills"`
}

// registryPath is where the canonical registry.json lives inside the module.
// The installer package embeds it via //go:embed, so it must stay alongside
// internal/installer/*.go source files.
const registryPath = "internal/installer/registry.json"

func main() {
	check := flag.Bool("check", false, "verify registry.json is up to date (exit 1 if not)")
	flag.Parse()

	skills, err := scanSkills("skills")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	reg := Registry{Skills: make(map[string]RegistrySkill, len(skills))}
	for _, s := range skills {
		reg.Skills[s.Name] = RegistrySkill{
			Version:      s.Config.Version,
			Description:  s.Description,
			RequiresBins: s.Config.RequiresBins,
			SetupPrompts: s.Config.SetupPrompts,
			Exclude: s.Config.Exclude,
		}
	}

	generated, err := marshalRegistry(reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling registry: %v\n", err)
		os.Exit(1)
	}

	if *check {
		existing, err := os.ReadFile(registryPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", registryPath, err)
			fmt.Fprintln(os.Stderr, "Run 'go run ./cmd/gen-registry' to generate it.")
			os.Exit(1)
		}
		if !bytes.Equal(existing, generated) {
			fmt.Fprintf(os.Stderr, "%s is outdated. Run 'go run ./cmd/gen-registry' to update.\n", registryPath)
			os.Exit(1)
		}
		fmt.Printf("%s is up to date.\n", registryPath)
		return
	}

	if err := os.WriteFile(registryPath, generated, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", registryPath, err)
		os.Exit(1)
	}
	fmt.Printf("%s generated with %d skills.\n", registryPath, len(reg.Skills))
}

// scanSkills reads all SKILL.md + config.json files under the skills directory.
// Supports both flat (skills/<name>/) and grouped (skills/<vertical>/<name>/) layouts.
func scanSkills(dir string) ([]SkillEntry, error) {
	var skills []SkillEntry

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}
		// Skip SKILL.md files inside profiles/ subdirectories.
		if strings.Contains(path, string(filepath.Separator)+"profiles"+string(filepath.Separator)) {
			return nil
		}
		skillDir := filepath.Dir(path)
		skillName := filepath.Base(skillDir)

		entry, err := loadSkillEntry(skillDir, skillName)
		if err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
		skills = append(skills, entry)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan skills directory: %w", err)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

// loadSkillEntry reads name+description from SKILL.md frontmatter and
// clawkit-specific config from config.json.
func loadSkillEntry(skillDir, dirName string) (SkillEntry, error) {
	// Parse SKILL.md for name and description only.
	name, desc, err := parseFrontmatter(filepath.Join(skillDir, "SKILL.md"), dirName)
	if err != nil {
		return SkillEntry{}, err
	}

	// Load config.json for clawkit-specific fields.
	cfg, err := loadSkillConfig(filepath.Join(skillDir, "config.json"))
	if err != nil {
		return SkillEntry{}, fmt.Errorf("config.json: %w", err)
	}

	return SkillEntry{
		Name:        name,
		Description: desc,
		Config:      cfg,
	}, nil
}

// loadSkillConfig reads and parses a config.json file.
func loadSkillConfig(path string) (SkillConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg SkillConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return SkillConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Version == "" {
		return cfg, fmt.Errorf("missing version in %s", path)
	}
	return cfg, nil
}

// parseFrontmatter extracts name and description from SKILL.md YAML frontmatter.
// Only these two OpenClaw-native fields are read from frontmatter; all
// clawkit-specific fields come from config.json instead.
func parseFrontmatter(path, dirName string) (name, description string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return "", "", fmt.Errorf("no frontmatter found")
	}

	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return "", "", fmt.Errorf("unterminated frontmatter")
	}
	fmBlock := content[4 : 4+end]

	name = dirName // default to directory name
	for _, line := range strings.Split(fmBlock, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = trimYAMLValue(line)
		} else if strings.HasPrefix(line, "description:") {
			description = trimYAMLValue(line)
		}
	}

	// Use directory name as registry key.
	name = dirName

	if description == "" {
		return "", "", fmt.Errorf("missing description in frontmatter")
	}

	return name, description, nil
}

// trimYAMLValue extracts the value after "key: value", handling quoted strings.
func trimYAMLValue(line string) string {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return ""
	}
	val := strings.TrimSpace(line[idx+1:])
	// Remove surrounding quotes.
	if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
		val = val[1 : len(val)-1]
	}
	return val
}

// marshalRegistry produces deterministic JSON output with sorted keys.
func marshalRegistry(reg Registry) ([]byte, error) {
	names := make([]string, 0, len(reg.Skills))
	for name := range reg.Skills {
		names = append(names, name)
	}
	sort.Strings(names)

	ordered := make([]struct {
		Name  string
		Skill RegistrySkill
	}, len(names))
	for i, name := range names {
		ordered[i].Name = name
		ordered[i].Skill = reg.Skills[name]
	}

	// Use custom encoder to avoid escaping &, <, >.
	var buf bytes.Buffer
	buf.WriteString("{\n  \"skills\": {\n")
	for i, entry := range ordered {
		var skillBuf bytes.Buffer
		enc := json.NewEncoder(&skillBuf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("    ", "  ")
		if err := enc.Encode(entry.Skill); err != nil {
			return nil, err
		}
		skillJSON := bytes.TrimRight(skillBuf.Bytes(), "\n")
		buf.WriteString(fmt.Sprintf("    %q: %s", entry.Name, string(skillJSON)))
		if i < len(ordered)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("  }\n}\n")
	return buf.Bytes(), nil
}
