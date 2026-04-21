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
// Named "clawkit.json" in the installed skill directory (user view) to
// distinguish it from the source "config.json" used during development.
const ConfigFileName = "clawkit.json"

// SkillConfig is the per-skill clawkit.json saved after installation.
// Stores the installed version and the user_inputs collected from
// setup_prompts so updates can re-bake placeholders into the new SKILL.md.
type SkillConfig struct {
	Version    string            `json:"version"`
	UserInputs map[string]string `json:"user_inputs,omitempty"`
}

// LoadSkillConfig reads the clawkit.json from an installed skill directory.
func LoadSkillConfig(skillDir string) (*SkillConfig, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, ConfigFileName))
	if err != nil {
		return nil, err
	}
	var cfg SkillConfig
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}

// SaveSkillConfig writes the clawkit.json to an installed skill directory.
func SaveSkillConfig(skillDir string, cfg *SkillConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, ConfigFileName), data, 0600)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return home
}

func detectOpenClaw() (binary string, skillsDir string) {
	skillsDir = filepath.Join(homeDir(), ".openclaw", "workspace", "skills")

	if path, err := exec.LookPath("openclaw"); err == nil {
		binary = path
	}

	if _, err := os.Stat(skillsDir); err == nil {
		return binary, skillsDir
	}

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
	return filepath.Join(homeDir(), ".openclaw", "workspace", "skills")
}

// GetConfigDir returns the clawkit config directory.
func GetConfigDir() string {
	if cfgDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(cfgDir, "clawkit")
	}
	return filepath.Join(homeDir(), ".clawkit")
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
