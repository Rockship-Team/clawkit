// Package installer — workspace allowlist management.
//
// When a user installs a skill via `clawkit install <skill>`, the skill is
// added to the agent's allowlist so it appears in <available_skills>. When
// the last skill is uninstalled, the allowlist is cleared.
package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
)

// SetupWorkspace adds a newly installed skill to the allowlist.
func SetupWorkspace(skillsDir, skillName string) {
	priorSkills := listInstalledSkills(skillsDir, skillName)
	allSkills := append(priorSkills, skillName)
	if err := setSkillsAllowlist(allSkills...); err != nil {
		ui.Warn("Could not update agents.defaults.skills: %v", err)
		ui.Info("Run manually: openclaw config set agents.defaults.skills '%s'", mustMarshalJSON(allSkills))
	} else {
		ui.Ok("Set agents.defaults.skills = %s", mustMarshalJSON(allSkills))
	}
}

// listInstalledSkills returns names of installed skills (excluding excludeName).
func listInstalledSkills(skillsDir, excludeName string) []string {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == excludeName {
			continue
		}
		if _, err := os.Stat(filepath.Join(skillsDir, e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	return names
}

// RemoveFromWorkspace removes a skill from the allowlist. If it was the last
// skill, clears the allowlist entirely.
func RemoveFromWorkspace(skillsDir, skillName string) {
	remaining := listInstalledSkills(skillsDir, skillName)

	if len(remaining) == 0 {
		if err := ClearSkillsAllowlist(); err != nil {
			ui.Warn("Could not clear skill allowlist: %v", err)
			ui.Info("Run manually: openclaw config unset agents.defaults.skills")
		} else {
			ui.Ok("Cleared agents.defaults.skills")
		}
		return
	}

	if err := setSkillsAllowlist(remaining...); err != nil {
		ui.Warn("Could not update agents.defaults.skills: %v", err)
	} else {
		ui.Ok("Updated agents.defaults.skills = %s", mustMarshalJSON(remaining))
	}
}

// setSkillsAllowlist runs `openclaw config set agents.defaults.skills [names...]`.
func setSkillsAllowlist(names ...string) error {
	openclawBin, err := exec.LookPath("openclaw")
	if err != nil {
		return fmt.Errorf("openclaw binary not found on PATH: %w", err)
	}

	val, err := json.Marshal(names)
	if err != nil {
		return err
	}

	cmd := exec.Command(openclawBin, "config", "set", "agents.defaults.skills", string(val))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw config set failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// ClearSkillsAllowlist removes the agents.defaults.skills config entry.
func ClearSkillsAllowlist() error {
	openclawBin, err := exec.LookPath("openclaw")
	if err != nil {
		return fmt.Errorf("openclaw binary not found on PATH: %w", err)
	}
	cmd := exec.Command(openclawBin, "config", "unset", "agents.defaults.skills")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw config unset failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// ResolveWorkspaceDir returns the OpenClaw workspace directory.
func ResolveWorkspaceDir() string {
	return filepath.Dir(config.GetSkillsDir())
}

func mustMarshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}
