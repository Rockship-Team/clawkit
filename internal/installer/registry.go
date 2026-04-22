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
	"github.com/rockship-co/clawkit/internal/runtime"
	"github.com/rockship-co/clawkit/internal/ui"
	"github.com/rockship-co/clawkit/skills"
)

//go:embed registry.json
var embeddedRegistry []byte

const (
	remoteRegistryURL  = "https://raw.githubusercontent.com/Rockship-Team/clawkit/main/registry.json"
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

func loadRegistry() (*Registry, error) {
	var reg Registry
	if err := json.Unmarshal(embeddedRegistry, &reg); err != nil {
		return nil, fmt.Errorf("invalid embedded registry.json: %w", err)
	}

	if data, err := fetchRemoteRegistry(); err == nil {
		var remote Registry
		if json.Unmarshal(data, &remote) == nil {
			for name, skill := range remote.Skills {
				reg.Skills[name] = skill
			}
			for name, members := range remote.Groups {
				if reg.Groups == nil {
					reg.Groups = make(map[string][]string)
				}
				reg.Groups[name] = members
			}
		}
	}

	if data, err := loadLocalRegistry(); err == nil {
		var local Registry
		if json.Unmarshal(data, &local) == nil {
			for name, skill := range local.Skills {
				reg.Skills[name] = skill
			}
			for name, members := range local.Groups {
				if reg.Groups == nil {
					reg.Groups = make(map[string][]string)
				}
				reg.Groups[name] = members
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

// alwaysExclude are files that should never be copied to the installed skill
// directory. _config.json is the dev-time metadata (the installer writes its
// own clawkit.json instead). _cli/ and _cli.json are runtime metadata — they
// live under ~/.clawkit/runtimes/, not inside the installed skill dir.
// _bootstrap/ is applied directly to the workspace root at install time,
// never copied into the skill dir.
var alwaysExclude = []string{"_config.json", runtime.CLIDir, runtime.SpecFile, "_bootstrap"}

// downloadSkill installs a skill's files into targetDir. Sources in priority
// order: local → embedded → remote .tar.gz. Returns the runtime key (group
// name for grouped skills, skill name for flat skills with a _cli/), or ""
// when no runtime applies.
func downloadSkill(skillName, targetDir string) (string, error) {
	if localDir := findLocalSkill(skillName); localDir != "" {
		ui.Info("Installing from local source")
		if err := copyDir(localDir, targetDir, alwaysExclude); err != nil {
			return "", err
		}
		return installLocalRuntime(skillName, localDir)
	}

	if embeddedPath := skills.FindSkill(skillName); embeddedPath != "" {
		ui.Info("Installing from embedded skills")
		if err := copyEmbeddedSkill(embeddedPath, targetDir, alwaysExclude); err != nil {
			return "", err
		}
		return installEmbeddedRuntime(skillName, embeddedPath)
	}

	dlURL := fmt.Sprintf("%s/%s.tar.gz", remoteSkillBaseURL, skillName)
	ui.Info("Downloading %s...", skillName)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("skill package not found at %s (HTTP %d)", dlURL, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "clawkit-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return "", fmt.Errorf("download incomplete: %w", err)
	}

	return "", archive.ExtractTarGz(tmpFile.Name(), targetDir)
}

// runtimeSource locates the directory that holds the _cli/ payload for a
// local skill. It returns (parentDir, key) where parentDir contains both
// _cli/ and _cli.json, and key is the runtime identifier. For a grouped
// skill skills/<group>/<skill>, the parent is skills/<group> and the key is
// <group>. For a flat skill skills/<name> that contains its own _cli/, the
// parent is skills/<name> and the key is <name>. Returns ("","") if no
// runtime applies.
func runtimeSource(skillName, localDir string) (parentDir, key string) {
	if info, err := os.Stat(filepath.Join(localDir, runtime.CLIDir)); err == nil && info.IsDir() {
		return localDir, skillName
	}
	parent := filepath.Dir(localDir)
	if info, err := os.Stat(filepath.Join(parent, runtime.CLIDir)); err == nil && info.IsDir() {
		return parent, filepath.Base(parent)
	}
	return "", ""
}

// runtimeEmbeddedSource is the embedded-FS equivalent of runtimeSource.
func runtimeEmbeddedSource(skillName, embeddedPath string) (parentPath, key string) {
	if info, err := fs.Stat(skills.FS, embeddedPath+"/"+runtime.CLIDir); err == nil && info.IsDir() {
		return embeddedPath, skillName
	}
	parent := filepath.ToSlash(filepath.Dir(embeddedPath))
	if parent == "." || parent == "" {
		return "", ""
	}
	if info, err := fs.Stat(skills.FS, parent+"/"+runtime.CLIDir); err == nil && info.IsDir() {
		return parent, filepath.Base(parent)
	}
	return "", ""
}

// installLocalRuntime installs the runtime for a local skill, if one applies,
// and returns the runtime key.
func installLocalRuntime(skillName, localDir string) (string, error) {
	parent, key := runtimeSource(skillName, localDir)
	if key == "" {
		return "", nil
	}
	spec, err := runtime.LoadSpec(parent)
	if err != nil {
		return "", err
	}
	if err := runtime.Install(key, filepath.Join(parent, runtime.CLIDir), spec); err != nil {
		return "", fmt.Errorf("install runtime %s: %w", key, err)
	}
	if err := runtime.LinkBins(key, spec.Bins); err != nil {
		return "", fmt.Errorf("link bins for runtime %s: %w", key, err)
	}
	return key, nil
}

// installEmbeddedRuntime is the embedded-FS equivalent of installLocalRuntime.
func installEmbeddedRuntime(skillName, embeddedPath string) (string, error) {
	parent, key := runtimeEmbeddedSource(skillName, embeddedPath)
	if key == "" {
		return "", nil
	}
	spec, err := runtime.LoadEmbeddedSpec(skills.FS, parent)
	if err != nil {
		return "", err
	}
	if err := runtime.InstallEmbedded(key, skills.FS, parent+"/"+runtime.CLIDir, spec); err != nil {
		return "", fmt.Errorf("install runtime %s: %w", key, err)
	}
	if err := runtime.LinkBins(key, spec.Bins); err != nil {
		return "", fmt.Errorf("link bins for runtime %s: %w", key, err)
	}
	return key, nil
}

// findLocalSkill searches for a skill in the local skills/ directory.
// Supports both flat (skills/<name>) and grouped (skills/<group>/<name>) layouts.
func findLocalSkill(skillName string) string {
	flat := filepath.Join("skills", skillName)
	if _, err := os.Stat(filepath.Join(flat, "SKILL.md")); err == nil {
		return flat
	}
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

func copyEmbeddedSkill(skillName, targetDir string, excludePatterns []string) error {
	return fs.WalkDir(skills.FS, skillName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
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
