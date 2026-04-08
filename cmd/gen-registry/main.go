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
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Version       string        `json:"version"`
	RequiresOAuth []string      `json:"requires_oauth"`
	SetupPrompts  []SetupPrompt `json:"setup_prompts"`
}

// SetupPrompt defines a setup question asked during installation.
type SetupPrompt struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default string `json:"default,omitempty"`
}

// RegistrySkill is the per-skill entry in registry.json.
type RegistrySkill struct {
	Version       string        `json:"version"`
	Description   string        `json:"description"`
	RequiresOAuth []string      `json:"requires_oauth"`
	SetupPrompts  []SetupPrompt `json:"setup_prompts"`
}

// Registry is the top-level structure of registry.json.
type Registry struct {
	Skills map[string]RegistrySkill `json:"skills"`
}

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
			Version:       s.Version,
			Description:   s.Description,
			RequiresOAuth: s.RequiresOAuth,
			SetupPrompts:  s.SetupPrompts,
		}
	}

	generated, err := marshalRegistry(reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling registry: %v\n", err)
		os.Exit(1)
	}

	if *check {
		existing, err := os.ReadFile("registry.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading registry.json: %v\n", err)
			fmt.Fprintln(os.Stderr, "Run 'go run ./cmd/gen-registry' to generate it.")
			os.Exit(1)
		}
		if !bytes.Equal(existing, generated) {
			fmt.Fprintln(os.Stderr, "registry.json is outdated. Run 'go run ./cmd/gen-registry' to update.")
			os.Exit(1)
		}
		fmt.Println("registry.json is up to date.")
		return
	}

	if err := os.WriteFile("registry.json", generated, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing registry.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("registry.json generated with %d skills.\n", len(reg.Skills))
}

// scanSkills reads all skills/*/SKILL.md files and parses their frontmatter.
func scanSkills(dir string) ([]SkillFrontmatter, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	var skills []SkillFrontmatter
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(dir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); os.IsNotExist(err) {
			continue
		}

		fm, err := parseFrontmatter(skillMD, e.Name())
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", skillMD, err)
		}
		skills = append(skills, fm)
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
		Name:          dirName, // default to directory name
		RequiresOAuth: []string{},
		SetupPrompts:  []SetupPrompt{},
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
			// Check if inline array or block list
			inline := trimYAMLValue(line)
			if inline == "" || inline == "[]" {
				// Block list: read subsequent "  - value" lines
				fm.RequiresOAuth = []string{}
				for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "  - ") {
					i++
					fm.RequiresOAuth = append(fm.RequiresOAuth, strings.TrimSpace(strings.TrimPrefix(lines[i], "  - ")))
				}
			} else {
				// Inline: [val1, val2]
				inline = strings.Trim(inline, "[]")
				for _, v := range strings.Split(inline, ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						fm.RequiresOAuth = append(fm.RequiresOAuth, v)
					}
				}
			}
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
