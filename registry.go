package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	// For now, load from local registry.json in the repo
	// Later: fetch from registryURL
	registryPath := getLocalRegistryPath()

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load registry: %w\nMake sure registry.json exists at %s", err, registryPath)
	}

	var reg Registry
	err = json.Unmarshal(data, &reg)
	if err != nil {
		return nil, fmt.Errorf("invalid registry.json: %w", err)
	}
	return &reg, nil
}

func getLocalRegistryPath() string {
	// Check if running from the repo directory
	if _, err := os.Stat("registry.json"); err == nil {
		return "registry.json"
	}
	// Fallback to config dir
	return filepath.Join(getConfigDir(), "registry.json")
}

func (r *Registry) GetSkill(name string) (*SkillInfo, bool) {
	skill, ok := r.Skills[name]
	return &skill, ok
}
