package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	defaultRegistryURL = "https://skills.rockship.co"
	configFileName     = "config.json"
)

type AppConfig struct {
	RegistryURL string `json:"registry_url"`
	SkillsDir   string `json:"skills_dir"`
}

// Platform represents a detected AI agent runtime
type Platform struct {
	Name      string // "PicoClaw" or "OpenClaw"
	Binary    string // full path to binary, empty if not in PATH
	SkillsDir string // path to skills directory
}

// detectPlatforms finds all installed AI agent runtimes.
// PicoClaw and OpenClaw are separate products that share the same SKILL.md format.
func detectPlatforms() []Platform {
	home, _ := os.UserHomeDir()

	candidates := []struct {
		name      string
		bin       string
		skillsDir string
	}{
		{"PicoClaw", "picoclaw", filepath.Join(home, ".picoclaw", "workspace", "skills")},
		{"OpenClaw", "openclaw", filepath.Join(home, ".openclaw", "workspace", "skills")},
	}

	var found []Platform
	for _, c := range candidates {
		p := Platform{Name: c.name}

		if path, err := exec.LookPath(c.bin); err == nil {
			p.Binary = path
		}

		if _, err := os.Stat(c.skillsDir); err == nil {
			p.SkillsDir = c.skillsDir
		} else if p.Binary != "" {
			// Binary exists but no skills dir — can create it
			p.SkillsDir = c.skillsDir
		}

		if p.Binary != "" || p.SkillsDir != "" {
			found = append(found, p)
		}
	}
	return found
}

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawkit")
}

// getSkillsDir returns the skills directory of the first detected platform (non-interactive).
func getSkillsDir() string {
	platforms := detectPlatforms()
	if len(platforms) > 0 {
		return platforms[0].SkillsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawkit", "skills")
}

// preflight checks for installed platforms and lets user choose if multiple found.
// Returns the selected platform's skills directory.
func preflight() string {
	platforms := detectPlatforms()

	if len(platforms) == 0 {
		fmt.Println()
		fatal(`No AI agent runtime found on this machine.

  clawkit requires one of the following to be installed:

  PicoClaw:
    curl -fsSL https://get.picoclaw.com | bash
    https://github.com/sipeed/picoclaw

  OpenClaw:
    curl -fsSL https://get.openclaw.ai | bash
    https://docs.openclaw.ai/installation`)
	}

	// If only one platform, use it
	if len(platforms) == 1 {
		p := platforms[0]
		if p.Binary != "" {
			ok("Detected %s (%s)", p.Name, p.Binary)
		} else {
			ok("Detected %s", p.Name)
		}
		info("Skills directory: %s", p.SkillsDir)
		return p.SkillsDir
	}

	// Multiple platforms found — let user choose
	fmt.Println("Multiple platforms detected:")
	for i, p := range platforms {
		status := p.SkillsDir
		if p.Binary != "" {
			status = p.Binary
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, p.Name, status)
	}
	fmt.Print("Choose platform [1]: ")
	choice := promptInput("")
	idx := 0
	if choice == "2" && len(platforms) > 1 {
		idx = 1
	}

	p := platforms[idx]
	ok("Using %s", p.Name)
	info("Skills directory: %s", p.SkillsDir)
	return p.SkillsDir
}

// SkillConfig is the per-skill config saved after installation
type SkillConfig struct {
	SkillName  string            `json:"skill_name"`
	Version    string            `json:"version"`
	OAuthDone  bool              `json:"oauth_done"`
	Tokens     map[string]string `json:"tokens,omitempty"`
	UserInputs map[string]string `json:"user_inputs,omitempty"`
}

func loadSkillConfig(skillDir string) (*SkillConfig, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, configFileName))
	if err != nil {
		return nil, err
	}
	var cfg SkillConfig
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}

func saveSkillConfig(skillDir string, cfg *SkillConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, configFileName), data, 0600)
}
