// Package installer implements clawkit's install, update, list, status,
// and package commands.
package installer

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rockship-co/clawkit/internal/archive"
	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
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

// downloadSkill downloads a skill package from GitHub Releases or copies from local.
func downloadSkill(skillName, targetDir string) error {
	// Try local first (dev mode).
	localDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(localDir); err == nil {
		ui.Info("Installing from local source")
		return copyDir(localDir, targetDir)
	}

	// Download from remote.
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
