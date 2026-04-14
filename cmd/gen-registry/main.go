// Command gen-registry scans skills/*/SKILL.md frontmatter and generates
// registry.json. This is the canonical way to keep the registry in sync
// with skill definitions.
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

// SkillFrontmatter mirrors the YAML frontmatter in SKILL.md.
type SkillFrontmatter struct {
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Version        string        `json:"version"`
	RequiresOAuth  []string      `json:"requires_oauth"`
	RequiresBins   []string      `json:"requires_bins"`
	RequiresSkills []string      `json:"requires_skills"`
	SetupPrompts   []SetupPrompt `json:"setup_prompts"`
}

// SetupPrompt defines a setup question asked during installation.
type SetupPrompt struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default string `json:"default,omitempty"`
}

// RegistrySkill is the per-skill entry in registry.json.
type RegistrySkill struct {
	Version        string        `json:"version"`
	Description    string        `json:"description"`
	RequiresOAuth  []string      `json:"requires_oauth"`
	RequiresBins   []string      `json:"requires_bins,omitempty"`
	RequiresSkills []string      `json:"requires_skills,omitempty"`
	SetupPrompts   []SetupPrompt `json:"setup_prompts"`
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
			Version:        s.Version,
			Description:    s.Description,
			RequiresOAuth:  s.RequiresOAuth,
			RequiresBins:   s.RequiresBins,
			RequiresSkills: s.RequiresSkills,
			SetupPrompts:   s.SetupPrompts,
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

// scanSkills reads all SKILL.md files under the skills directory.
// Supports both flat (skills/<name>/SKILL.md) and grouped
// (skills/<vertical>/<name>/SKILL.md) layouts.
func scanSkills(dir string) ([]SkillFrontmatter, error) {
	var skills []SkillFrontmatter

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}
		skillDir := filepath.Dir(path)
		skillName := filepath.Base(skillDir)

		fm, err := parseFrontmatter(path, skillName)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		skills = append(skills, fm)
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

// parseFrontmatter extracts YAML frontmatter from a SKILL.md file.
// We parse manually to avoid adding a YAML dependency (keeping deps at zero).
func parseFrontmatter(path, dirName string) (SkillFrontmatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillFrontmatter{}, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return SkillFrontmatter{}, fmt.Errorf("no frontmatter found")
	}

	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return SkillFrontmatter{}, fmt.Errorf("unterminated frontmatter")
	}
	fmBlock := content[4 : 4+end]

	fm := SkillFrontmatter{
		Name: dirName, // default to directory name
	}

	lines := strings.Split(fmBlock, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "name:") {
			fm.Name = trimYAMLValue(line)
		} else if strings.HasPrefix(line, "description:") {
			fm.Description = trimYAMLValue(line)
		} else if strings.HasPrefix(line, "version:") {
			fm.Version = trimYAMLValue(line)
		} else if strings.HasPrefix(line, "requires_oauth:") {
			fm.RequiresOAuth, i = parseYAMLStringList(lines, i, trimYAMLValue(line))
		} else if strings.HasPrefix(line, "requires_bins:") {
			fm.RequiresBins, i = parseYAMLStringList(lines, i, trimYAMLValue(line))
		} else if strings.HasPrefix(line, "requires_skills:") {
			fm.RequiresSkills, i = parseYAMLStringList(lines, i, trimYAMLValue(line))
		} else if strings.HasPrefix(line, "setup_prompts:") {
			inline := trimYAMLValue(line)
			if inline == "[]" {
				fm.SetupPrompts = []SetupPrompt{}
				continue
			}
			// Block list of objects
			fm.SetupPrompts = []SetupPrompt{}
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "  - ") {
				i++
				prompt := SetupPrompt{}
				// First line: "  - key: value"
				first := strings.TrimPrefix(lines[i], "  - ")
				parsePromptField(&prompt, first)
				// Subsequent lines: "    label: value", "    default: value"
				for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    ") && !strings.HasPrefix(lines[i+1], "  - ") {
					i++
					parsePromptField(&prompt, strings.TrimSpace(lines[i]))
				}
				fm.SetupPrompts = append(fm.SetupPrompts, prompt)
			}
		}
	}

	// Use directory name as skill name for registry key
	fm.Name = dirName

	if fm.Version == "" {
		return fm, fmt.Errorf("missing version in frontmatter")
	}
	if fm.Description == "" {
		return fm, fmt.Errorf("missing description in frontmatter")
	}

	return fm, nil
}

// parseYAMLStringList parses a YAML string list field, supporting both inline
// ([val1, val2]) and block list (  - val) forms. Returns the parsed values and
// the updated line index.
func parseYAMLStringList(lines []string, i int, inline string) ([]string, int) {
	if inline == "" || inline == "[]" {
		// Block list: read subsequent "  - value" lines.
		var result []string
		for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "  - ") {
			i++
			result = append(result, strings.TrimSpace(strings.TrimPrefix(lines[i], "  - ")))
		}
		return result, i
	}
	// Inline: [val1, val2]
	var result []string
	for _, v := range strings.Split(strings.Trim(inline, "[]"), ",") {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result, i
}

func parsePromptField(p *SetupPrompt, field string) {
	if strings.HasPrefix(field, "key:") {
		p.Key = trimYAMLValue(field)
	} else if strings.HasPrefix(field, "label:") {
		p.Label = trimYAMLValue(field)
	} else if strings.HasPrefix(field, "default:") {
		p.Default = trimYAMLValue(field)
	}
}

// trimYAMLValue extracts the value after "key: value", handling quoted strings.
func trimYAMLValue(line string) string {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return ""
	}
	val := strings.TrimSpace(line[idx+1:])
	// Remove surrounding quotes
	if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
		val = val[1 : len(val)-1]
	}
	return val
}

// marshalRegistry produces deterministic JSON output with sorted keys.
func marshalRegistry(reg Registry) ([]byte, error) {
	// Sort skill names for deterministic output.
	names := make([]string, 0, len(reg.Skills))
	for name := range reg.Skills {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build ordered map for deterministic output.
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
		// Encoder adds trailing newline — trim it.
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
