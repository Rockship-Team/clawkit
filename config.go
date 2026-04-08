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

	// Binary found but no skills dir yet — still valid
	if binary != "" {
		return binary, skillsDir
	}

	return "", ""
}

func getSkillsDir() string {
	_, skillsDir := detectOpenClaw()
	if skillsDir != "" {
		return skillsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", "skills")
}

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawkit")
}

// preflight checks that OpenClaw is installed before proceeding.
// Returns the skills directory.
func preflight() string {
	binary, skillsDir := detectOpenClaw()

	if binary == "" && skillsDir == "" {
		fmt.Println()
		fatal(`OpenClaw not found on this machine.

  clawkit requires OpenClaw to be installed.
  Skills will not work without the OpenClaw runtime.

  Install OpenClaw:
    curl -fsSL https://get.openclaw.ai | bash

  Documentation: https://docs.openclaw.ai/installation`)
	}

	if binary != "" {
		ok("Detected OpenClaw (%s)", binary)
	} else {
		warn("OpenClaw skills directory found but binary not in PATH")
	}
	info("Skills directory: %s", skillsDir)
	return skillsDir
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
