package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	defaultRegistryURL = "https://skills.rockship.co"
	defaultSkillsDir   = ".clawkit/skills"
	configFileName     = "config.json"
)

type AppConfig struct {
	RegistryURL string `json:"registry_url"`
	SkillsDir   string `json:"skills_dir"`
}

func getSkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultSkillsDir)
}

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawkit")
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
