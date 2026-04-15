// Package installer implements clawkit's install, update, list, status,
// and package commands.
package installer

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rockship-co/clawkit/internal/archive"
	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
	"github.com/rockship-co/clawkit/skills"
)

// embeddedRegistry is the registry.json shipped with the binary. It is the
// authoritative source when running the globally-installed CLI (no network,
// no local file). Remote and local sources are treated as optional overrides.
//
//go:embed registry.json
var embeddedRegistry []byte

const (
	// remoteRegistryURL is the GitHub raw content URL for registry.json.
	remoteRegistryURL = "https://raw.githubusercontent.com/Rockship-Team/clawkit/main/registry.json"
	// remoteSkillBaseURL is the GitHub Releases URL for skill packages.
	remoteSkillBaseURL = "https://github.com/Rockship-Team/clawkit/releases/latest/download"
)

// SetupPrompt defines an interactive prompt shown during clawkit install.
type SetupPrompt struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
}

// SkillInfo describes a skill in the registry.
type SkillInfo struct {
	Version      string        `json:"version"`
	Description  string        `json:"description"`
	RequiresBins []string      `json:"requires_bins,omitempty"`
	SetupPrompts []SetupPrompt `json:"setup_prompts,omitempty"`
	Exclude []string      `json:"exclude,omitempty"`
}

// Registry holds the available skills manifest.
type Registry struct {
	Skills map[string]SkillInfo `json:"skills"`
}

// GetSkill returns a skill by name from the registry.
func (r *Registry) GetSkill(name string) (*SkillInfo, bool) {
	skill, ok := r.Skills[name]
	return &skill, ok
}

// loadRegistry returns the skills registry. The embedded registry.json is
// always available and is the authoritative baseline. Remote and local
// registry files are treated as optional overrides: remote lets us ship
// registry updates without rebuilding the binary, and local supports dev
// mode (skills added to ./skills but not yet pushed).
func loadRegistry() (*Registry, error) {
	var reg Registry
	if err := json.Unmarshal(embeddedRegistry, &reg); err != nil {
		return nil, fmt.Errorf("invalid embedded registry.json: %w", err)
	}

	// Optional remote override — ignored on failure (private repo, offline, etc.).
	if data, err := fetchRemoteRegistry(); err == nil {
		var remote Registry
		if json.Unmarshal(data, &remote) == nil {
			for name, skill := range remote.Skills {
				reg.Skills[name] = skill
			}
		}
	}

	// Optional local override (dev mode: ./registry.json or config dir).
	if data, err := loadLocalRegistry(); err == nil {
		var local Registry
		if json.Unmarshal(data, &local) == nil {
			for name, skill := range local.Skills {
				reg.Skills[name] = skill
			}
		}
	}

	return &reg, nil
}

func fetchRemoteRegistry() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteRegistryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry fetch failed: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func loadLocalRegistry() ([]byte, error) {
	if _, err := os.Stat("registry.json"); err == nil {
		return os.ReadFile("registry.json")
	}
	path := filepath.Join(config.GetConfigDir(), "registry.json")
	return os.ReadFile(path)
}

// downloadSkill installs a skill's files into targetDir. Sources in priority
// order:
//  1. Local ./skills/<name> (dev mode: developer is in the repo).
//  2. Embedded skills shipped with the binary (works for npm-installed CLI).
//  3. Remote GitHub Releases (.tar.gz) — useful when the repo is public and
//     we want to ship registry/skill updates without rebuilding the binary.
// alwaysExclude are files that should never be copied to the installed skill
// directory. config.json is the dev-time metadata (the installer writes its
// own clawkit.json instead).
// Note: bootstrap-files/ IS copied (needed by LockdownWorkspace) then
// deleted by CmdInstall after lockdown applies them to workspace root.
var alwaysExclude = []string{"config.json"}

func downloadSkill(skillName, targetDir string, excludePatterns ...[]string) error {
	patterns := append([]string{}, alwaysExclude...)
	if len(excludePatterns) > 0 {
		patterns = append(patterns, excludePatterns[0]...)
	}

	// 1. Local (dev mode) — search skills/<name> or skills/<vertical>/<name>.
	if localDir := findLocalSkill(skillName); localDir != "" {
		ui.Info("Installing from local source")
		return copyDir(localDir, targetDir, patterns)
	}

	// 2. Embedded — search the skill across verticals in the embedded FS.
	if embeddedPath := skills.FindSkill(skillName); embeddedPath != "" {
		ui.Info("Installing from embedded skills")
		return copyEmbeddedSkill(embeddedPath, targetDir, patterns)
	}

	// 3. Remote GitHub Release.
	dlURL := fmt.Sprintf("%s/%s.tar.gz", remoteSkillBaseURL, skillName)
	ui.Info("Downloading %s...", skillName)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("skill package not found at %s (HTTP %d)", dlURL, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "clawkit-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("download incomplete: %w", err)
	}

	return archive.ExtractTarGz(tmpFile.Name(), targetDir)
}

// findLocalSkill searches for a skill in the local skills/ directory.
// Supports both flat (skills/<name>) and grouped (skills/<vertical>/<name>) layouts.
func findLocalSkill(skillName string) string {
	// Try flat first.
	flat := filepath.Join("skills", skillName)
	if _, err := os.Stat(filepath.Join(flat, "SKILL.md")); err == nil {
		return flat
	}
	// Search one level of nesting: skills/<vertical>/<name>.
	entries, err := os.ReadDir("skills")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		nested := filepath.Join("skills", e.Name(), skillName)
		if _, err := os.Stat(filepath.Join(nested, "SKILL.md")); err == nil {
			return nested
		}
	}
	return ""
}

// shouldExclude checks whether relPath matches any of the exclude patterns.
// Supports tsconfig-style globs:
//   - "cmd"           — matches the directory (and everything inside it)
//   - "*.tmp"         — matches *.tmp at any depth
//   - "**/*.test.go"  — matches *.test.go at any depth
//   - "**/test"       — matches any path component named "test"
//   - "tools/crawl"   — matches that exact prefix
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

// matchGlob matches a path against a single glob pattern with ** support.
func matchGlob(path, pattern string) bool {
	// Handle ** prefix: "**/<rest>" matches <rest> against any suffix.
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Match against the full path and every sub-path.
		parts := strings.Split(path, "/")
		for i := range parts {
			sub := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, sub); matched {
				return true
			}
		}
		return false
	}

	// No slash in pattern → treat as component-level match (like tsconfig).
	// "cmd" matches "cmd", "cmd/main.go"; "*.tmp" matches "foo.tmp", "a/b/foo.tmp".
	if !strings.Contains(pattern, "/") {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		return false
	}

	// Pattern has slashes → match against full path.
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	// Also try as prefix so "tools/crawl" matches "tools/crawl/main.go".
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}
	return false
}

// copyEmbeddedSkill walks skills.FS under skillName and writes every file
// into targetDir, preserving the relative directory structure.
// Files/dirs matching excludePatterns are skipped.
func copyEmbeddedSkill(skillName, targetDir string, excludePatterns []string) error {
	return fs.WalkDir(skills.FS, skillName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// path is like "finance-tracker/SKILL.md". Strip the skillName prefix
		// so files land directly in targetDir.
		relPath, err := filepath.Rel(skillName, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(targetDir, 0755)
		}
		if shouldExclude(relPath, excludePatterns) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		dest := filepath.Join(targetDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := skills.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}
		return os.WriteFile(dest, data, 0644)
	})
}

// copyDir recursively copies a directory tree.
// Files/dirs matching excludePatterns are skipped.
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
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
