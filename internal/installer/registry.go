// Package installer implements clawkit's install, update, list, status,
// and package commands.
package installer

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/engine"
	"github.com/rockship-co/clawkit/internal/ui"
)

//go:embed registry.json
var embeddedRegistry []byte

// Environment variables set by the npm wrapper (bin/clawkit.js) to point the
// binary at the skill files that ship inside the npm package.
const (
	envSkillsDir = "CLAWKIT_SKILLS_DIR"
	envRegistry  = "CLAWKIT_REGISTRY"
)

// SetupPrompt defines an interactive prompt shown during clawkit install.
type SetupPrompt struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
}

// SkillInfo describes a skill in the registry.
type SkillInfo struct {
	Name           string        `json:"name,omitempty"`
	Description    string        `json:"description"`
	OS             []string      `json:"os,omitempty"`
	RequiresBins   []string      `json:"requires_bins,omitempty"`
	RequiresConfig []string      `json:"requires_config,omitempty"`
	Version        string        `json:"version"`
	SetupPrompts   []SetupPrompt `json:"setup_prompts,omitempty"`
}

// Registry holds the available skills manifest.
type Registry struct {
	Skills map[string]SkillInfo `json:"skills"`
	Groups map[string][]string  `json:"groups,omitempty"`
}

// GetSkill returns a skill by name from the registry.
func (r *Registry) GetSkill(name string) (*SkillInfo, bool) {
	skill, ok := r.Skills[name]
	return &skill, ok
}

// GroupMembers returns the member skill names for a group, or nil if name
// is not a known group.
func (r *Registry) GroupMembers(name string) []string {
	return r.Groups[name]
}

// GroupOf returns the group a skill belongs to, or "" if the skill is flat.
func (r *Registry) GroupOf(skillName string) string {
	for group, members := range r.Groups {
		for _, m := range members {
			if m == skillName {
				return group
			}
		}
	}
	return ""
}

// loadRegistry resolves the canonical registry. Priority:
//  1. Local override — ./registry.json (cwd) or ~/.clawkit/registry.json (user override)
//  2. Packaged registry — CLAWKIT_REGISTRY env (set by the npm wrapper)
//  3. Embedded fallback — small snapshot baked into the binary
func loadRegistry() (*Registry, error) {
	if data, err := loadLocalRegistry(); err == nil {
		var reg Registry
		if err := json.Unmarshal(data, &reg); err == nil {
			return &reg, nil
		}
	}

	if path := os.Getenv(envRegistry); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var reg Registry
			if err := json.Unmarshal(data, &reg); err == nil {
				return &reg, nil
			}
		}
	}

	var reg Registry
	if err := json.Unmarshal(embeddedRegistry, &reg); err != nil {
		return nil, fmt.Errorf("invalid embedded registry.json: %w", err)
	}
	return &reg, nil
}

func loadLocalRegistry() ([]byte, error) {
	if _, err := os.Stat("registry.json"); err == nil {
		return os.ReadFile("registry.json")
	}
	path := filepath.Join(config.GetConfigDir(), "registry.json")
	return os.ReadFile(path)
}

// alwaysExclude are files that should never be copied to the installed skill
// directory. config.json is the dev-time metadata (the installer writes its
// own clawkit.json instead). _engine/ and engine.json are engine metadata —
// they live under ~/.clawkit/engines/, not inside the installed skill dir.
// _bootstrap/ is applied directly to the workspace root at install time,
// never copied into the skill dir.
var alwaysExclude = []string{"config.json", engine.SourceDir, engine.SpecFile, bootstrapDir}

// bootstrapDir is the folder within a skill or group that holds persona .md
// files copied to the workspace root on install.
const bootstrapDir = "_bootstrap"

// downloadSkill installs a skill's files into targetDir. Sources in priority
// order: local dev tree → packaged skills dir (CLAWKIT_SKILLS_DIR, set by the
// npm wrapper). Returns the engine key (group name for grouped skills, skill
// name for flat skills with an _engine/), the on-disk source directory
// (used by applyBootstrap), and an error. No network access is involved;
// skills ship as files inside the npm package.
func downloadSkill(_ *Registry, skillName, targetDir string) (string, string, func(), error) {
	noop := func() {}

	sourceDir := resolveSkillSource(skillName)
	if sourceDir == "" {
		return "", "", noop, fmt.Errorf("skill %q not found in local skills/ or %s", skillName, envSkillsDir)
	}

	ui.Info("Installing from %s", sourceDir)
	if err := copyDir(sourceDir, targetDir, alwaysExclude); err != nil {
		return "", "", noop, err
	}
	key, err := installEngine(skillName, sourceDir)
	return key, sourceDir, noop, err
}

// resolveSkillSource finds the on-disk directory that holds SKILL.md for
// skillName. It checks the dev tree (./skills/...) first, then the packaged
// skills dir supplied by the npm wrapper.
func resolveSkillSource(skillName string) string {
	if local := findLocalSkill(skillName); local != "" {
		return local
	}
	pkg := os.Getenv(envSkillsDir)
	if pkg == "" {
		return ""
	}
	return findSkillIn(pkg, skillName)
}

// engineSource locates the directory that holds the _engine/ payload for a
// skill. For a grouped skill skills/<group>/<skill>, the parent is
// skills/<group> and the key is <group>. For a flat skill skills/<name>
// that contains its own _engine/, the parent is skills/<name> and the key
// is <name>. Returns ("","") if no engine applies.
func engineSource(skillName, localDir string) (parentDir, key string) {
	if info, err := os.Stat(filepath.Join(localDir, engine.SourceDir)); err == nil && info.IsDir() {
		return localDir, skillName
	}
	parent := filepath.Dir(localDir)
	if info, err := os.Stat(filepath.Join(parent, engine.SourceDir)); err == nil && info.IsDir() {
		return parent, filepath.Base(parent)
	}
	return "", ""
}

// installEngine installs the engine for a skill (from either the dev tree
// or the packaged skills dir), if one applies, and returns the key.
func installEngine(skillName, localDir string) (string, error) {
	parent, key := engineSource(skillName, localDir)
	if key == "" {
		return "", nil
	}
	spec, err := engine.LoadSpec(parent)
	if err != nil {
		return "", err
	}
	if err := engine.Install(key, filepath.Join(parent, engine.SourceDir), spec); err != nil {
		return "", fmt.Errorf("install engine %s: %w", key, err)
	}
	if err := engine.LinkBins(key, spec.Bins); err != nil {
		return "", fmt.Errorf("link bins for engine %s: %w", key, err)
	}
	return key, nil
}

// findLocalSkill searches for a skill in the local skills/ directory (dev
// tree). Supports both flat (skills/<name>) and grouped
// (skills/<group>/<name>) layouts.
func findLocalSkill(skillName string) string {
	return findSkillIn("skills", skillName)
}

// findSkillIn searches root for a skill directory by the same flat-or-grouped
// rules as findLocalSkill. root is either the dev "skills" dir or the path
// supplied via CLAWKIT_SKILLS_DIR.
func findSkillIn(root, skillName string) string {
	flat := filepath.Join(root, skillName)
	if _, err := os.Stat(filepath.Join(flat, "SKILL.md")); err == nil {
		return flat
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		nested := filepath.Join(root, e.Name(), skillName)
		if _, err := os.Stat(filepath.Join(nested, "SKILL.md")); err == nil {
			return nested
		}
	}
	return ""
}

// shouldExclude checks whether relPath matches any of the exclude patterns.
func shouldExclude(relPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	normalized := filepath.ToSlash(relPath)
	for _, pattern := range patterns {
		if matchGlob(normalized, pattern) {
			return true
		}
	}
	return false
}

func matchGlob(path, pattern string) bool {
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		parts := strings.Split(path, "/")
		for i := range parts {
			sub := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, sub); matched {
				return true
			}
		}
		return false
	}

	if !strings.Contains(pattern, "/") {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		return false
	}

	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}
	return false
}

func copyDir(src, dst string, excludePatterns []string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		if shouldExclude(relPath, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
