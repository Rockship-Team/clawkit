// Package installer — workspace lockdown for 1-skill-at-a-time model.
//
// When a user installs a skill via `clawkit install <skill>`, their entire
// OpenClaw workspace becomes dedicated to that skill:
//   - Default workspace MD files (AGENTS.md, SOUL.md, IDENTITY.md, USER.md,
//     BOOTSTRAP.md, HEARTBEAT.md, TOOLS.md) are replaced/removed so the agent
//     adopts the skill's persona.
//   - The agent's skill allowlist is set to only this skill.
//   - Any prior conversation sessions are cleared so the new persona takes
//     effect on the next message.
//   - If a different skill was previously installed, it is removed first.
//
// This matches the "1 skill at a time, exclusive" model: to switch skills,
// the user must uninstall the current one (clawkit does this automatically
// during install with a prompt).
package installer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/ui"
)

// WorkspaceOverridesDirName is the subdirectory inside a skill package that
// contains files to be copied into the user's workspace root on install.
// Files here replace the user's default workspace persona files (AGENTS.md,
// SOUL.md, IDENTITY.md, USER.md) so the agent adopts the skill's character.
const WorkspaceOverridesDirName = "workspace-overrides"

// genericWorkspaceFiles are the default OpenClaw assistant files that ship
// with a fresh `openclaw onboard` setup. They turn the agent into a
// generic personal assistant with heartbeat polling, journaling, and other
// behaviors that conflict with a dedicated skill persona. We delete them
// during lockdown so they stop being loaded as system prompt context.
var genericWorkspaceFiles = []string{
	"BOOTSTRAP.md",
	"HEARTBEAT.md",
	"TOOLS.md",
}

// LockdownWorkspace performs the full 1-skill-at-a-time lockdown:
//  1. Detects and removes any previously installed skill (with user prompt).
//  2. Backs up existing workspace MD files to .clawkit-backup/<timestamp>/.
//  3. Copies workspace-overrides/* from the new skill to workspace root.
//  4. Deletes generic assistant files (BOOTSTRAP.md, HEARTBEAT.md, TOOLS.md).
//  5. Resets existing conversation sessions.
//  6. Sets agents.defaults.skills = [<skillName>] via openclaw config.
//
// skillsDir is the OpenClaw skills directory (~/.openclaw/workspace/skills).
// skillDir is the path where the new skill was just installed.
// skillName is the skill being installed.
func LockdownWorkspace(skillsDir, skillDir, skillName string) {
	workspaceDir := filepath.Dir(skillsDir) // ~/.openclaw/workspace

	// Step 1: handle prior skill. If another skill is already installed,
	// ask user whether to remove it — the 1-skill-at-a-time model forbids
	// keeping both.
	if err := removePriorSkills(skillsDir, skillName); err != nil {
		ui.Warn("Could not remove prior skills: %v", err)
	}

	// Step 2: backup the user's current workspace MD files so they can
	// be restored via `clawkit uninstall` or manual recovery.
	backupDir, backupErr := backupWorkspaceFiles(workspaceDir)
	if backupErr != nil {
		ui.Warn("Could not back up workspace files: %v", backupErr)
	} else if backupDir != "" {
		ui.Info("Backed up workspace files to %s", backupDir)
	}

	// Step 3: copy workspace-overrides/* from the skill into workspace root.
	// This is how the skill stamps its persona onto the agent's system prompt.
	overridesDir := filepath.Join(skillDir, WorkspaceOverridesDirName)
	if _, err := os.Stat(overridesDir); err == nil {
		count, err := applyWorkspaceOverrides(overridesDir, workspaceDir)
		if err != nil {
			ui.Warn("Could not apply workspace overrides: %v", err)
		} else if count > 0 {
			ui.Ok("Applied %d workspace override file(s) from skill", count)
		}
	}

	// Step 4: delete generic OpenClaw assistant files. These files from
	// `openclaw onboard` turn the agent into a generic personal assistant
	// that conflicts with a dedicated skill persona.
	for _, f := range genericWorkspaceFiles {
		p := filepath.Join(workspaceDir, f)
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				ui.Warn("Could not remove %s: %v", f, err)
			}
		}
	}

	// Step 5: reset any existing agent sessions. Without this, an active
	// session's cached system prompt (from the old persona) would leak
	// into the first reply after install.
	if err := resetAgentSessions(workspaceDir); err != nil {
		ui.Warn("Could not reset sessions: %v", err)
	}

	// Step 6: set the agent skill allowlist to only this skill. This
	// filters <available_skills> in the LLM's system prompt so the agent
	// cannot invoke other skills it might otherwise try to use.
	if err := setSkillsAllowlist(skillName); err != nil {
		ui.Warn("Could not set agents.defaults.skills: %v", err)
		ui.Info("Run manually: openclaw config set agents.defaults.skills '[\"%s\"]'", skillName)
	} else {
		ui.Ok("Set agents.defaults.skills = [\"%s\"]", skillName)
	}
}

// removePriorSkills finds other skill directories in skillsDir and prompts
// the user to remove them. Enforces the 1-skill-at-a-time model.
func removePriorSkills(skillsDir, currentSkillName string) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var prior []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == currentSkillName {
			continue
		}
		// Only treat as "installed skill" if it has a SKILL.md file.
		if _, err := os.Stat(filepath.Join(skillsDir, e.Name(), "SKILL.md")); err == nil {
			prior = append(prior, e.Name())
		}
	}

	if len(prior) == 0 {
		return nil
	}

	fmt.Println()
	ui.Warn("Another skill is already installed: %s", strings.Join(prior, ", "))
	fmt.Println("  clawkit enforces a 1-skill-at-a-time model — installing a new skill")
	fmt.Println("  requires removing the old one. The current skill's data directory")
	fmt.Println("  (including its orders database, OAuth tokens, and assets) will be")
	fmt.Println("  deleted from ~/.openclaw/workspace/skills/.")
	fmt.Print("  Remove the existing skill(s) and continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		ui.Fatal("Cancelled. Run `clawkit uninstall %s` manually, then retry.", prior[0])
	}

	for _, name := range prior {
		dir := filepath.Join(skillsDir, name)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w", name, err)
		}
		ui.Ok("Removed prior skill: %s", name)
	}
	return nil
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

// applyWorkspaceOverrides copies every file directly inside overridesDir
// into workspaceDir, overwriting existing files. It does not recurse into
// subdirectories — only top-level files.
func applyWorkspaceOverrides(overridesDir, workspaceDir string) (int, error) {
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

// resetAgentSessions clears active conversation sessions so the new persona
// takes effect on the next incoming message. It renames existing *.jsonl
// files to *.jsonl.reset.<timestamp>.bak and empties sessions.json.
//
// OpenClaw stores sessions at ~/.openclaw/agents/<agent>/sessions/. By
// default there's only the "main" agent. We iterate over all agent
// directories to handle non-default setups.
func resetAgentSessions(workspaceDir string) error {
	// agents dir sits next to workspace dir: ~/.openclaw/agents/
	openclawDir := filepath.Dir(workspaceDir) // ~/.openclaw
	agentsDir := filepath.Join(openclawDir, "agents")

	agents, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	stamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	touched := 0

	for _, agent := range agents {
		if !agent.IsDir() {
			continue
		}
		sessionsDir := filepath.Join(agentsDir, agent.Name(), "sessions")
		if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(sessionsDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			src := filepath.Join(sessionsDir, e.Name())
			dst := filepath.Join(sessionsDir, e.Name()+".reset."+stamp+".bak")
			if err := os.Rename(src, dst); err != nil {
				ui.Warn("Could not archive session %s: %v", e.Name(), err)
				continue
			}
			touched++
		}

		// Empty the sessions index file so OpenClaw treats this agent as
		// having zero active sessions on next startup.
		sessionsJSON := filepath.Join(sessionsDir, "sessions.json")
		if _, err := os.Stat(sessionsJSON); err == nil {
			// Back up then overwrite.
			_ = os.Rename(sessionsJSON, sessionsJSON+".reset."+stamp+".bak")
			if err := os.WriteFile(sessionsJSON, []byte("{}\n"), 0644); err != nil {
				return fmt.Errorf("write sessions.json: %w", err)
			}
		}
	}

	if touched > 0 {
		ui.Ok("Archived %d prior conversation session(s)", touched)
	}
	return nil
}

// setSkillsAllowlist runs `openclaw config set agents.defaults.skills [<name>]`
// so the LLM only sees this skill in <available_skills>. Returns an error if
// the openclaw CLI is not on PATH or the command fails.
func setSkillsAllowlist(skillName string) error {
	openclawBin, err := exec.LookPath("openclaw")
	if err != nil {
		return fmt.Errorf("openclaw binary not found on PATH: %w", err)
	}

	// Pass the value as a JSON array literal. openclaw config set accepts it.
	val, err := json.Marshal([]string{skillName})
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

// RestoreWorkspaceFromBackup is the reverse of LockdownWorkspace: restore
// the most recent .clawkit-backup/<stamp>/ files to the workspace root.
// Called by `clawkit uninstall`.
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

	// Remove the current workspace-override files first so restoration is
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
// the agent goes back to unrestricted skill access. Called by uninstall.
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
