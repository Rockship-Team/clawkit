package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// GitHub raw content URL for the registry
	remoteRegistryURL = "https://raw.githubusercontent.com/Rockship-Team/clawkit/main/registry.json"
	// GitHub releases URL for skill packages
	remoteSkillBaseURL = "https://github.com/Rockship-Team/clawkit/releases/latest/download"
)

type SkillInfo struct {
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	RequiresOAuth []string `json:"requires_oauth"`
	SetupPrompts  []Prompt `json:"setup_prompts,omitempty"`
}

type Prompt struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default string `json:"default,omitempty"`
}

type Registry struct {
	Skills map[string]SkillInfo `json:"skills"`
}

func loadRegistry() (*Registry, error) {
	// Try remote first
	data, err := fetchRemoteRegistry()
	if err != nil {
		// Fallback to local (for dev or offline)
		data, err = loadLocalRegistry()
		if err != nil {
			return nil, fmt.Errorf("cannot load registry (tried remote and local): %w", err)
		}
		info("Using local registry (offline mode)")
	}

	var reg Registry
	err = json.Unmarshal(data, &reg)
	if err != nil {
		return nil, fmt.Errorf("invalid registry.json: %w", err)
	}
	return &reg, nil
}

func fetchRemoteRegistry() ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(remoteRegistryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry fetch failed: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func loadLocalRegistry() ([]byte, error) {
	// Check current directory (dev mode)
	if _, err := os.Stat("registry.json"); err == nil {
		return os.ReadFile("registry.json")
	}
	// Check config dir
	path := filepath.Join(getConfigDir(), "registry.json")
	return os.ReadFile(path)
}

func (r *Registry) GetSkill(name string) (*SkillInfo, bool) {
	skill, ok := r.Skills[name]
	return &skill, ok
}

// downloadSkill downloads a skill package from GitHub Releases and extracts it.
// Falls back to local skills/ directory for development.
func downloadSkill(skillName, targetDir string) error {
	// Try local first (dev mode — running from repo)
	localDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(localDir); err == nil {
		info("Installing from local source")
		return copyDir(localDir, targetDir)
	}

	// Download from remote
	url := fmt.Sprintf("%s/%s.tar.gz", remoteSkillBaseURL, skillName)
	info("Downloading %s...", skillName)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("skill package not found at %s (HTTP %d)", url, resp.StatusCode)
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "clawkit-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("download incomplete: %w", err)
	}

	// Extract
	return extractTarGz(tmpFile.Name(), targetDir)
}
