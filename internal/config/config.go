// Package config handles OpenClaw detection, skill configuration persistence,
// and preflight checks.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rockship-co/clawkit/internal/ui"
)

// ConfigFileName is the per-skill config file written after installation.
const ConfigFileName = "config.json"

// SkillConfig is the per-skill config saved after installation.
type SkillConfig struct {
	SkillName  string            `json:"skill_name"`
	Version    string            `json:"version"`
	OAuthDone  bool              `json:"oauth_done"`
	Tokens     map[string]string `json:"tokens,omitempty"`
	UserInputs map[string]string `json:"user_inputs,omitempty"`
}

// LoadSkillConfig reads the config.json from a skill directory.
func LoadSkillConfig(skillDir string) (*SkillConfig, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, ConfigFileName))
	if err != nil {
		return nil, err
	}
	var cfg SkillConfig
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}

// SaveSkillConfig writes the config.json to a skill directory.
func SaveSkillConfig(skillDir string, cfg *SkillConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, ConfigFileName), data, 0600)
}

// detectOpenClaw checks if OpenClaw is installed.
// Returns binary path and skills directory.
func detectOpenClaw() (binary string, skillsDir string) {
	home, _ := os.UserHomeDir()
	skillsDir = filepath.Join(home, ".openclaw", "workspace", "skills")

	if path, err := exec.LookPath("openclaw"); err == nil {
		binary = path
	}

	if _, err := os.Stat(skillsDir); err == nil {
		return binary, skillsDir
	}

	// Binary found but no skills dir yet — still valid.
	if binary != "" {
		return binary, skillsDir
	}

	return "", ""
}

// GetSkillsDir returns the OpenClaw skills directory.
func GetSkillsDir() string {
	_, skillsDir := detectOpenClaw()
	if skillsDir != "" {
		return skillsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", "skills")
}

// GetConfigDir returns the clawkit config directory (~/.clawkit).
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawkit")
}

// Preflight checks that OpenClaw is installed before proceeding.
// Returns the skills directory path.
func Preflight() string {
	binary, skillsDir := detectOpenClaw()

	if binary == "" && skillsDir == "" {
		fmt.Println()
		ui.Fatal(`OpenClaw not found on this machine.

  clawkit requires OpenClaw to be installed.
  Skills will not work without the OpenClaw runtime.

  Install OpenClaw:
    curl -fsSL https://get.openclaw.ai | bash

  Documentation: https://docs.openclaw.ai/installation`)
	}

	if binary != "" {
		ui.Ok("Detected OpenClaw (%s)", binary)
	} else {
		ui.Warn("OpenClaw skills directory found but binary not in PATH")
	}
	ui.Info("Skills directory: %s", skillsDir)
	return skillsDir
}
