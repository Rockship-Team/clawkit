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

// SkillInfo describes a skill in the registry.
type SkillInfo struct {
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	RequiresOAuth  []string `json:"requires_oauth"`
	RequiresBins   []string `json:"requires_bins,omitempty"`
	RequiresSkills []string `json:"requires_skills,omitempty"`
	SetupPrompts   []Prompt `json:"setup_prompts,omitempty"`
}

// Prompt defines a setup question asked during installation.
type Prompt struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default string `json:"default,omitempty"`
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
func downloadSkill(skillName, targetDir string) error {
	// 1. Local (dev mode).
	localDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(localDir); err == nil {
		ui.Info("Installing from local source")
		return copyDir(localDir, targetDir)
	}

	// 2. Embedded — check the skill exists in the embedded FS.
	if _, err := skills.FS.ReadDir(skillName); err == nil {
		ui.Info("Installing from embedded skills")
		return copyEmbeddedSkill(skillName, targetDir)
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

// copyEmbeddedSkill walks skills.FS under skillName and writes every file
// into targetDir, preserving the relative directory structure.
func copyEmbeddedSkill(skillName, targetDir string) error {
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
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
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
