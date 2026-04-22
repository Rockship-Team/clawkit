// Command gen-registry scans skills/**/SKILL.md and skills/**/config.json
// to generate registry.json.
//
// SKILL.md frontmatter provides: name, description,
//
//	metadata.openclaw.os, metadata.openclaw.requires.bins,
//	metadata.openclaw.requires.config.
//
// config.json provides: version, setup_prompts, exclude.
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

// SkillConfig is the per-skill config.json file.
type SkillConfig struct {
	Version      string        `json:"version"`
	SetupPrompts []SetupPrompt `json:"setup_prompts,omitempty"`
	Exclude      []string      `json:"exclude,omitempty"`
}

// SetupPrompt defines an interactive prompt shown during clawkit install.
type SetupPrompt struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
}

// SkillEntry combines SKILL.md frontmatter and config.json for one skill.
type SkillEntry struct {
	Name           string
	Description    string
	OS             []string
	RequiresBins   []string
	RequiresConfig []string
	Config         SkillConfig
}

// RegistrySkill is the per-skill entry in registry.json.
type RegistrySkill struct {
	Description    string        `json:"description"`
	OS             []string      `json:"os,omitempty"`
	RequiresBins   []string      `json:"requires_bins,omitempty"`
	RequiresConfig []string      `json:"requires_config,omitempty"`
	Version        string        `json:"version"`
	SetupPrompts   []SetupPrompt `json:"setup_prompts,omitempty"`
	Exclude        []string      `json:"exclude,omitempty"`
}

// Registry is the top-level structure of registry.json.
// Groups lists the member skill names for each group directory — a group
// is any directory under skills/ that holds a shared _cli/ and one or more
// child directories containing SKILL.md.
type Registry struct {
	Skills map[string]RegistrySkill `json:"skills"`
	Groups map[string][]string      `json:"groups,omitempty"`
}

const registryPath = "internal/installer/registry.json"

func main() {
	check := flag.Bool("check", false, "verify registry.json is up to date (exit 1 if not)")
	flag.Parse()

	skills, err := scanSkills("skills")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	groups, err := scanGroups("skills")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	reg := Registry{
		Skills: make(map[string]RegistrySkill, len(skills)),
		Groups: groups,
	}
	for _, s := range skills {
		reg.Skills[s.Name] = RegistrySkill{
			Description:    s.Description,
			OS:             s.OS,
			RequiresBins:   s.RequiresBins,
			RequiresConfig: s.RequiresConfig,
			Version:        s.Config.Version,
			SetupPrompts:   s.Config.SetupPrompts,
			Exclude:        s.Config.Exclude,
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

// scanGroups walks dir and records every directory that holds a _cli/
// subdirectory plus one or more immediate children with SKILL.md. Keys are
// group directory names (basename); values are the member skill names.
func scanGroups(dir string) (map[string][]string, error) {
	out := make(map[string][]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Name() == "_cli" {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(path, "_cli")); statErr != nil {
			return nil
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil
		}
		var members []string
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "_cli" {
				continue
			}
			if _, skillErr := os.Stat(filepath.Join(path, e.Name(), "SKILL.md")); skillErr == nil {
				members = append(members, e.Name())
			}
		}
		if len(members) > 0 {
			sort.Strings(members)
			out[filepath.Base(path)] = members
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan groups: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// scanSkills reads every SKILL.md + config.json under dir. Skips _cli/
// directories (shared runtime code, not skills themselves).
func scanSkills(dir string) ([]SkillEntry, error) {
	var skills []SkillEntry

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "_cli" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() != "SKILL.md" {
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

func loadSkillEntry(skillDir, dirName string) (SkillEntry, error) {
	meta, err := parseFrontmatter(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return SkillEntry{}, err
	}

	cfg, err := loadSkillConfig(filepath.Join(skillDir, "config.json"))
	if err != nil {
		return SkillEntry{}, fmt.Errorf("config.json: %w", err)
	}

	if meta.description == "" {
		return SkillEntry{}, fmt.Errorf("missing description in SKILL.md frontmatter")
	}

	return SkillEntry{
		Name:           dirName,
		Description:    meta.description,
		OS:             meta.os,
		RequiresBins:   meta.requiresBins,
		RequiresConfig: meta.requiresConfig,
		Config:         cfg,
	}, nil
}

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

type skillMetadata struct {
	name           string
	description    string
	os             []string
	requiresBins   []string
	requiresConfig []string
}

// parseFrontmatter extracts name, description, and
// metadata.openclaw.{os, requires.bins, requires.config} from SKILL.md.
func parseFrontmatter(path string) (skillMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skillMetadata{}, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return skillMetadata{}, fmt.Errorf("no frontmatter found")
	}

	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return skillMetadata{}, fmt.Errorf("unterminated frontmatter")
	}
	fmBlock := content[4 : 4+end]

	flat, err := parseYAML(fmBlock)
	if err != nil {
		return skillMetadata{}, fmt.Errorf("parse frontmatter: %w", err)
	}

	meta := skillMetadata{}
	if v, ok := flat["name"]; ok && len(v) > 0 {
		meta.name = v[0]
	}
	if v, ok := flat["description"]; ok && len(v) > 0 {
		meta.description = v[0]
	}
	meta.os = flat["metadata.openclaw.os"]
	meta.requiresBins = flat["metadata.openclaw.requires.bins"]
	meta.requiresConfig = flat["metadata.openclaw.requires.config"]
	return meta, nil
}

// parseYAML parses a restricted subset of YAML: nested maps (indent-based),
// scalar values, and inline flow-style arrays ([a, b, c]). Every leaf value
// is returned as a slice keyed by its dotted path.
func parseYAML(block string) (map[string][]string, error) {
	out := make(map[string][]string)
	type frame struct {
		indent int
		key    string
	}
	var stack []frame

	for _, rawLine := range strings.Split(block, "\n") {
		line := strings.TrimRight(rawLine, " \t\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(trimmed)

		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		pathParts := make([]string, 0, len(stack)+1)
		for _, f := range stack {
			pathParts = append(pathParts, f.key)
		}
		pathParts = append(pathParts, key)
		fullPath := strings.Join(pathParts, ".")

		if val == "" {
			stack = append(stack, frame{indent: indent, key: key})
			continue
		}

		out[fullPath] = parseScalarOrArray(val)
	}

	return out, nil
}

// parseScalarOrArray turns a YAML value into a []string. Supports:
//   - flow arrays: [a, b, "c"]
//   - quoted scalars: "foo" / 'foo'
//   - bare scalars: foo
func parseScalarOrArray(val string) []string {
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		inner := strings.TrimSpace(val[1 : len(val)-1])
		if inner == "" {
			return []string{}
		}
		parts := strings.Split(inner, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			out = append(out, trimQuoted(strings.TrimSpace(p)))
		}
		return out
	}
	return []string{trimQuoted(val)}
}

func trimQuoted(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
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
	buf.WriteString("  }")

	if len(reg.Groups) > 0 {
		buf.WriteString(",\n  \"groups\": {\n")
		groupNames := make([]string, 0, len(reg.Groups))
		for name := range reg.Groups {
			groupNames = append(groupNames, name)
		}
		sort.Strings(groupNames)
		for i, name := range groupNames {
			members := reg.Groups[name]
			memberBuf, err := json.Marshal(members)
			if err != nil {
				return nil, err
			}
			buf.WriteString(fmt.Sprintf("    %q: %s", name, string(memberBuf)))
			if i < len(groupNames)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString("  }")
	}

	buf.WriteString("\n}\n")
	return buf.Bytes(), nil
}
