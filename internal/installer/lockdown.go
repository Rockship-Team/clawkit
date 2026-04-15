// Package installer — workspace setup for skill installation.
//
// When a user installs a skill via `clawkit install <skill>`:
//   - The first skill sets the workspace persona (AGENTS.md, SOUL.md, etc.)
//   - Subsequent skills are added to the allowlist without changing the persona.
//   - The agent's skill allowlist is updated to include all installed skills.
//
// This supports multi-skill workspaces: users can install multiple skills
// and the agent sees all of them in <available_skills>. The persona is set
// by the first skill installed (or the most recently installed skill that
// has bootstrap-files/).
package installer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
)

// BootstrapFilesDirName is the subdirectory inside a skill package that
// contains files to be copied into the user's workspace root on install.
// Files here replace the user's default workspace persona files (AGENTS.md,
// SOUL.md, IDENTITY.md, USER.md) so the agent adopts the skill's character.
const BootstrapFilesDirName = "bootstrap-files"

// genericWorkspaceFiles are the default OpenClaw assistant files that ship
// with a fresh `openclaw onboard` setup. They turn the agent into a
// generic personal assistant with heartbeat polling, journaling, and other
// behaviors that conflict with a dedicated skill persona. We delete them
// during the first skill install so they stop being loaded as system prompt context.
var genericWorkspaceFiles = []string{
	"BOOTSTRAP.md",
	"HEARTBEAT.md",
	"TOOLS.md",
}

// SetupWorkspace configures the workspace for the newly installed skill:
//  1. If this is the first skill, backs up workspace files and applies persona.
//  2. If other skills are already installed, skips persona changes.
//  3. Adds the skill to the allowlist (appends, does not replace).
//
// skillsDir is the OpenClaw skills directory (~/.openclaw/workspace/skills).
// skillDir is the path where the new skill was just installed.
// skillName is the skill being installed.
func SetupWorkspace(skillsDir, skillDir, skillName string) {
	workspaceDir := filepath.Dir(skillsDir) // ~/.openclaw/workspace

	// Check if other skills are already installed.
	priorSkills := listInstalledSkills(skillsDir, skillName)
	isFirstSkill := len(priorSkills) == 0

	if isFirstSkill {
		// First skill: backup workspace files and apply persona.
		backupDir, backupErr := backupWorkspaceFiles(workspaceDir)
		if backupErr != nil {
			ui.Warn("Could not back up workspace files: %v", backupErr)
		} else if backupDir != "" {
			ui.Info("Backed up workspace files to %s", backupDir)
		}

		// Apply bootstrap-files (persona) from the skill.
		overridesDir := filepath.Join(skillDir, BootstrapFilesDirName)
		if _, err := os.Stat(overridesDir); err == nil {
			count, err := applyBootstrapFiles(overridesDir, workspaceDir)
			if err != nil {
				ui.Warn("Could not apply bootstrap files: %v", err)
			} else if count > 0 {
				ui.Ok("Applied %d bootstrap file(s) from skill", count)
			}
		}

		// Delete generic OpenClaw assistant files.
		for _, f := range genericWorkspaceFiles {
			p := filepath.Join(workspaceDir, f)
			if _, err := os.Stat(p); err == nil {
				if err := os.Remove(p); err != nil {
					ui.Warn("Could not remove %s: %v", f, err)
				}
			}
		}
	} else {
		ui.Info("Adding skill to existing workspace (persona unchanged)")
	}

	// Update the allowlist to include all installed skills.
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
// skill, restores the original workspace files from backup.
func RemoveFromWorkspace(skillsDir, skillName string) {
	workspaceDir := filepath.Dir(skillsDir)

	// Find remaining skills after removal.
	remaining := listInstalledSkills(skillsDir, skillName)

	if len(remaining) == 0 {
		// Last skill being removed — restore original workspace.
		if err := RestoreWorkspaceFromBackup(workspaceDir); err != nil {
			ui.Warn("Could not restore workspace from backup: %v", err)
		}
		if err := ClearSkillsAllowlist(); err != nil {
			ui.Warn("Could not clear skill allowlist: %v", err)
			ui.Info("Run manually: openclaw config unset agents.defaults.skills")
		} else {
			ui.Ok("Cleared agents.defaults.skills")
		}
	} else {
		// Other skills remain — just update the allowlist.
		if err := setSkillsAllowlist(remaining...); err != nil {
			ui.Warn("Could not update agents.defaults.skills: %v", err)
		} else {
			ui.Ok("Updated agents.defaults.skills = %s", mustMarshalJSON(remaining))
		}
	}
}

// backupWorkspaceFiles copies workspace MD files to a timestamped backup
// directory before they get overwritten. Returns the backup dir path or
// empty string if nothing was backed up.
func backupWorkspaceFiles(workspaceDir string) (string, error) {
	candidates := []string{
		"AGENTS.md", "SOUL.md", "IDENTITY.md", "USER.md",
		"BOOTSTRAP.md", "HEARTBEAT.md", "TOOLS.md",
	}

	var toBackup []string
	for _, f := range candidates {
		p := filepath.Join(workspaceDir, f)
		if _, err := os.Stat(p); err == nil {
			toBackup = append(toBackup, f)
		}
	}
	if len(toBackup) == 0 {
		return "", nil
	}

	stamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	backupDir := filepath.Join(workspaceDir, ".clawkit-backup", stamp)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	for _, f := range toBackup {
		src := filepath.Join(workspaceDir, f)
		dst := filepath.Join(backupDir, f)
		if err := copyFile(src, dst); err != nil {
			return "", fmt.Errorf("backup %s: %w", f, err)
		}
	}
	return backupDir, nil
}

// applyBootstrapFiles copies every file directly inside overridesDir
// into workspaceDir, overwriting existing files. It does not recurse into
// subdirectories — only top-level files.
func applyBootstrapFiles(overridesDir, workspaceDir string) (int, error) {
	entries, err := os.ReadDir(overridesDir)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(overridesDir, e.Name())
		dst := filepath.Join(workspaceDir, e.Name())
		if err := copyFile(src, dst); err != nil {
			return count, fmt.Errorf("copy %s: %w", e.Name(), err)
		}
		count++
	}
	return count, nil
}

// setSkillsAllowlist runs `openclaw config set agents.defaults.skills [names...]`
// so the LLM sees these skills in <available_skills>.
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

// copyFile copies src to dst, creating parent directories as needed. If dst
// exists it is overwritten. Uses buffered io.Copy so it handles large files
// without blowing up memory.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// RestoreWorkspaceFromBackup is the reverse of SetupWorkspace: restore
// the most recent .clawkit-backup/<stamp>/ files to the workspace root.
// Called when the last skill is uninstalled.
func RestoreWorkspaceFromBackup(workspaceDir string) error {
	backupRoot := filepath.Join(workspaceDir, ".clawkit-backup")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to restore
		}
		return err
	}

	// Pick the latest backup (directories are timestamp-named and sort
	// lexicographically).
	var latest string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() > latest {
			latest = e.Name()
		}
	}
	if latest == "" {
		return nil
	}

	latestDir := filepath.Join(backupRoot, latest)
	files, err := os.ReadDir(latestDir)
	if err != nil {
		return err
	}

	// Remove the current bootstrap files first so restoration is
	// clean (no orphan files from the skill we're uninstalling).
	overrideNames := []string{"AGENTS.md", "SOUL.md", "IDENTITY.md", "USER.md"}
	for _, f := range overrideNames {
		_ = os.Remove(filepath.Join(workspaceDir, f))
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		src := filepath.Join(latestDir, f.Name())
		dst := filepath.Join(workspaceDir, f.Name())
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("restore %s: %w", f.Name(), err)
		}
	}
	ui.Ok("Restored workspace files from %s", latestDir)
	return nil
}

// ClearSkillsAllowlist removes the agents.defaults.skills config entry so
// the agent goes back to unrestricted skill access. Called when the last
// skill is uninstalled.
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

// ResolveWorkspaceDir returns the OpenClaw workspace directory for the
// current user — derived from config.GetSkillsDir() for consistency with
// preflight detection.
func ResolveWorkspaceDir() string {
	return filepath.Dir(config.GetSkillsDir())
}

// mustMarshalJSON marshals v to JSON or returns "[]" on error.
func mustMarshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}
