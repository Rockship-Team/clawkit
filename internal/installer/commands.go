package installer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rockship-co/clawkit/internal/archive"
	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/template"
	"github.com/rockship-co/clawkit/internal/ui"
	"github.com/rockship-co/clawkit/oauth"
)

// CmdList lists all available skills and their install status.
func CmdList() {
	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}

	fmt.Println("Available skills:")
	fmt.Println()
	for name, skill := range reg.Skills {
		installed := ""
		skillDir := filepath.Join(config.GetSkillsDir(), name)
		if _, err := os.Stat(skillDir); err == nil {
			installed = " [installed]"
		}
		fmt.Printf("  %-25s %s%s\n", name, skill.Description, installed)
		if len(skill.RequiresOAuth) > 0 {
			fmt.Printf("  %-25s requires: %s\n", "", strings.Join(skill.RequiresOAuth, ", "))
		}
	}
}

// CmdInstall installs a skill with OAuth setup and configuration.
func CmdInstall(skillName string, skipOAuth bool, profileName string) {
	shouldSkipOAuth := skipOAuth

	// Check platform is installed and get skills directory.
	skillsDir := config.Preflight()
	fmt.Println()

	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}

	skill, exists := reg.GetSkill(skillName)
	if !exists {
		ui.Fatal("Skill '%s' not found. Run 'clawkit list' to see available skills.", skillName)
	}

	fmt.Printf("▸ Installing %s v%s\n", skillName, skill.Version)
	fmt.Printf("  %s\n\n", skill.Description)

	// Check required skills — dependencies must be installed manually first.
	// We intentionally do not auto-install them so each skill's OAuth flow
	// stays isolated and the user has explicit control over what's installed.
	for _, depSkill := range skill.RequiresSkills {
		depDir := filepath.Join(skillsDir, depSkill)
		if _, err := os.Stat(depDir); os.IsNotExist(err) {
			fmt.Println()
			ui.Fatal(`Skill '%s' requires '%s' to be installed first.

  Install the dependency:
    clawkit install %s

  Then retry:
    clawkit install %s`, skillName, depSkill, depSkill, skillName)
		}
		// Verify the dependency completed its install — config.json is written
		// only at the end of a successful CmdInstall run.
		if _, err := config.LoadSkillConfig(depDir); err != nil {
			fmt.Println()
			ui.Fatal(`Skill '%s' is required but its installation is incomplete.

  Reinstall it:
    clawkit install %s`, depSkill, depSkill)
		}
		ui.Ok("Dependency '%s' is installed", depSkill)
	}

	// Shared stdin reader — must be created once and reused throughout CmdInstall
	// so bufio buffering doesn't silently consume lines meant for later prompts.
	stdinReader := bufio.NewReader(os.Stdin)

	// Check if already installed.
	targetDir := filepath.Join(skillsDir, skillName)
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("  Skill already installed at %s\n", targetDir)
		fmt.Print("  Overwrite? [y/N]: ")
		answer, _ := stdinReader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Cancelled.")
			return
		}
		if err := os.RemoveAll(targetDir); err != nil {
			ui.Fatal("Failed to remove existing skill: %v", err)
		}
	}

	// Download skill (remote) or copy (local dev).
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		ui.Fatal("Failed to create skill directory: %v", err)
	}
	err = downloadSkill(skillName, targetDir)
	if err != nil {
		os.RemoveAll(targetDir)
		ui.Fatal("Failed to install: %v", err)
	}
	ui.Ok("Skill files installed to %s", targetDir)

	// Apply profile overlay if specified.
	var profileValues map[string]string
	if profileName != "" {
		var perr error
		profileValues, perr = applyProfile(targetDir, profileName)
		if perr != nil {
			os.RemoveAll(targetDir)
			ui.Fatal("Profile '%s': %v", profileName, perr)
		}
		ui.Ok("Applied profile '%s'", profileName)
	}

	// Install required CLI binaries.
	if len(skill.RequiresBins) > 0 {
		fmt.Println()
		ui.Info("Installing required CLI tools...")
		installRequiredBins(skill.RequiresBins)
	}

	// Run OAuth for each required provider.
	if shouldSkipOAuth {
		ui.Warn("Skipping OAuth setup (--skip-oauth)")
	} else {
		for _, providerName := range skill.RequiresOAuth {
			fmt.Println()
			ui.Info("Setting up %s authorization...", providerName)
			if err := runOAuth(providerName, targetDir); err != nil {
				ui.Fatal("OAuth setup failed for %s: %v", providerName, err)
			}
			ui.Ok("Connected to %s", providerName)

			// Post-OAuth: write credentials and configure gog for gmail provider.
			if providerName == "gmail" {
				cfg, _ := config.LoadSkillConfig(targetDir)
				if cfg != nil && cfg.Tokens != nil {
					postOAuthGmail(targetDir, cfg.Tokens)
				}
			}
		}
	}

	// Lock down the workspace into this skill's persona:
	//   1. Remove any previously installed skill (1-skill-at-a-time model)
	//   2. Back up existing workspace MD files
	//   3. Copy workspace-overrides/* from the new skill to workspace root
	//   4. Delete generic assistant files (BOOTSTRAP.md, HEARTBEAT.md, TOOLS.md)
	//   5. Reset prior conversation sessions
	//   6. Set agents.defaults.skills = [<skillName>]
	LockdownWorkspace(skillsDir, targetDir, skillName)

	// Ensure image directories match catalog.
	if err := template.EnsureImageDirs(targetDir); err != nil {
		ui.Warn("Could not create image directories: %v", err)
	}

	// Load config early — schema init and profile merge both write to it.
	cfg, _ := config.LoadSkillConfig(targetDir)
	if cfg == nil {
		cfg = &config.SkillConfig{}
	}
	cfg.SkillName = skillName
	cfg.Profile = profileName
	cfg.Version = skill.Version
	cfg.OAuthDone = len(skill.RequiresOAuth) > 0

	// Initialize database from schema.json.
	if schemaErr := initSchema(targetDir, cfg, profileValues); schemaErr != nil {
		ui.Warn("Database init: %v", schemaErr)
	} else if cfg.DBTarget != "" {
		ui.Ok("Database initialized (%s)", cfg.DBTarget)
	}

	// Merge profile values into UserInputs for template placeholder substitution.
	if len(profileValues) > 0 {
		if cfg.UserInputs == nil {
			cfg.UserInputs = make(map[string]string)
		}
		for k, v := range profileValues {
			cfg.UserInputs[k] = v
		}
	}

	if err := config.SaveSkillConfig(targetDir, cfg); err != nil {
		ui.Fatal("Failed to save config: %v", err)
	}

	// Replace {key} placeholders in SKILL.md with user/profile input values
	// (e.g. {shop_name}, {emoji}) so the skill prompt is customized.
	if len(cfg.UserInputs) > 0 {
		if err := template.Process(targetDir, cfg.UserInputs); err != nil {
			ui.Warn("Template processing failed: %v", err)
		}
	}

	// Replace {key} placeholders in SKILL.md with OAuth token values
	// (e.g. {spreadsheet_id}, {gmail_account}) so the skill prompt is ready to use.
	if len(cfg.Tokens) > 0 {
		if err := template.ProcessTokens(targetDir, cfg.Tokens); err != nil {
			ui.Warn("Could not replace SKILL.md placeholders: %v", err)
		}
	}

	fmt.Println()
	ui.Ok("'%s' installed!", skillName)
	fmt.Printf("  Location: %s\n", targetDir)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("  1. Open SKILL.md: %s\n", filepath.Join(targetDir, "SKILL.md"))
	fmt.Println("  2. Restart the gateway: openclaw gateway restart")
}

// CmdUpdate updates an installed skill while preserving tokens and config.
func CmdUpdate(skillName string) {
	targetDir := filepath.Join(config.GetSkillsDir(), skillName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		ui.Fatal("Skill '%s' is not installed. Run 'clawkit install %s' first.", skillName, skillName)
	}

	// Load existing config to preserve tokens.
	existingCfg, err := config.LoadSkillConfig(targetDir)
	if err != nil {
		ui.Info("No existing config found, doing fresh install")
		CmdInstall(skillName, false, "")
		return
	}

	// Remove old files except config.json.
	entries, _ := os.ReadDir(targetDir)
	for _, e := range entries {
		if e.Name() != config.ConfigFileName {
			os.RemoveAll(filepath.Join(targetDir, e.Name()))
		}
	}

	// Download new skill files.
	if err := downloadSkill(skillName, targetDir); err != nil {
		ui.Fatal("Failed to update: %v", err)
	}

	// Re-apply profile overlay if the skill was installed with one.
	if existingCfg.Profile != "" {
		if _, err := applyProfile(targetDir, existingCfg.Profile); err != nil {
			ui.Warn("Could not re-apply profile '%s': %v", existingCfg.Profile, err)
		}
	}

	// Restore config.
	if err := config.SaveSkillConfig(targetDir, existingCfg); err != nil {
		ui.Warn("Could not restore config: %v", err)
	}

	// Re-process template with existing config values.
	if existingCfg.UserInputs != nil {
		template.EnsureImageDirs(targetDir)
		if err := template.Process(targetDir, existingCfg.UserInputs); err != nil {
			ui.Warn("Template processing failed: %v", err)
		}
	}

	// Re-inject OAuth tokens (spreadsheet_id, gmail_account, ...) into SKILL.md
	// so the updated prompt keeps the real values instead of {placeholders}.
	if len(existingCfg.Tokens) > 0 {
		if err := template.ProcessTokens(targetDir, existingCfg.Tokens); err != nil {
			ui.Warn("Could not re-inject token placeholders: %v", err)
		}
	}

	ui.Ok("'%s' updated to latest version", skillName)
	ui.Info("Restart the gateway to pick up changes: openclaw gateway restart")
}

// CmdUninstall removes a skill and reverses the workspace lockdown applied
// during install: restores backed-up workspace MD files, clears the skill
// allowlist, resets sessions, and deletes the skill directory.
//
// After this runs, the user's OpenClaw returns to a generic-assistant state
// (their pre-install workspace files are restored) and is ready to install
// a different skill or run without any skill.
func CmdUninstall(skillName string) {
	skillsDir := config.Preflight()
	fmt.Println()

	targetDir := filepath.Join(skillsDir, skillName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		ui.Fatal("Skill '%s' is not installed.", skillName)
	}

	fmt.Printf("▸ Uninstalling %s\n", skillName)
	fmt.Printf("  This will:\n")
	fmt.Printf("    • Delete %s (including orders database and assets)\n", targetDir)
	fmt.Printf("    • Restore your workspace files from the most recent backup\n")
	fmt.Printf("    • Clear agents.defaults.skills config\n")
	fmt.Printf("    • Archive all conversation sessions\n\n")
	fmt.Print("  Continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("  Cancelled.")
		return
	}

	workspaceDir := ResolveWorkspaceDir()

	// Restore workspace MD files from the backup dir written at install time.
	if err := RestoreWorkspaceFromBackup(workspaceDir); err != nil {
		ui.Warn("Could not restore workspace from backup: %v", err)
	}

	// Clear the skill allowlist so the agent goes back to unrestricted mode.
	if err := ClearSkillsAllowlist(); err != nil {
		ui.Warn("Could not clear skill allowlist: %v", err)
		ui.Info("Run manually: openclaw config unset agents.defaults.skills")
	} else {
		ui.Ok("Cleared agents.defaults.skills")
	}

	// Reset sessions so stale conversation history doesn't confuse the next
	// install or plain-assistant usage.
	if err := resetAgentSessions(workspaceDir); err != nil {
		ui.Warn("Could not reset sessions: %v", err)
	}

	// Finally, remove the skill directory itself.
	if err := os.RemoveAll(targetDir); err != nil {
		ui.Fatal("Could not remove skill directory: %v", err)
	}
	ui.Ok("Removed %s", targetDir)

	fmt.Println()
	ui.Ok("'%s' uninstalled", skillName)
	ui.Info("Restart the gateway to apply: openclaw gateway restart")
}

// CmdStatus shows all installed skills with version and OAuth status.
func CmdStatus() {
	skillsDir := config.GetSkillsDir()
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		fmt.Println("No skills installed yet.")
		fmt.Println("Run 'clawkit list' to see available skills.")
		return
	}

	fmt.Println("Installed skills:")
	fmt.Println()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfg, err := config.LoadSkillConfig(filepath.Join(skillsDir, e.Name()))
		if err != nil {
			fmt.Printf("  %-25s (no config)\n", e.Name())
			continue
		}
		profileInfo := ""
		if cfg.Profile != "" {
			profileInfo = fmt.Sprintf(" [profile: %s]", cfg.Profile)
		}
		oauthStatus := ""
		if cfg.OAuthDone {
			oauthStatus = " [oauth: connected]"
		}
		fmt.Printf("  %-25s v%s%s%s\n", cfg.SkillName, cfg.Version, profileInfo, oauthStatus)
	}
}

// CmdPackage packages a skill from skills/ into a .tar.gz for distribution.
func CmdPackage(skillName string) {
	sourceDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		ui.Fatal("Skill '%s' not found in skills/ directory", skillName)
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		ui.Fatal("Failed to create dist directory: %v", err)
	}
	outputPath := filepath.Join("dist", skillName+".tar.gz")

	ui.Info("Packaging %s...", skillName)
	if err := archive.CreateTarGz(sourceDir, outputPath); err != nil {
		ui.Fatal("Failed to package: %v", err)
	}

	fi, err := os.Stat(outputPath)
	if err != nil {
		ui.Fatal("Failed to stat output: %v", err)
	}
	sizeMB := float64(fi.Size()) / 1024 / 1024

	ui.Ok("Packaged: %s (%.1f MB)", outputPath, sizeMB)
	fmt.Println()
	fmt.Println("  Upload this file to GitHub Releases:")
	fmt.Printf("  gh release upload latest %s\n", outputPath)
}

// postOAuthGmail writes credential files and configures gog CLI.
// Flow (clawkit already did browser OAuth to capture email via userinfo):
//  1. Copy original client_secret_*.json → ~/.openclaw/workspace/skills/gog/
//  2. gog auth credentials set → import client_id/secret
//  3. gog auth add <email> --services … → let gog run its own OAuth flow
//  4. gog auth list → verify
func postOAuthGmail(skillDir string, tokens map[string]string) {
	clientID := tokens["google_client_id"]
	clientSecret := tokens["google_client_secret"]
	email := tokens["gmail_account"]
	srcCredFile := tokens["credential_file"]
	if clientID == "" || clientSecret == "" {
		return
	}

	// 1. Copy the original client_secret_*.json into the openclaw workspace,
	//    preserving the filename so future reinstalls can find it.
	home, _ := os.UserHomeDir()
	gogWorkspace := filepath.Join(home, ".openclaw", "workspace", "skills", "gog")
	if err := os.MkdirAll(gogWorkspace, 0755); err != nil {
		ui.Warn("Could not create gog workspace dir: %v", err)
		return
	}

	var credPath string
	if srcCredFile != "" {
		if data, err := os.ReadFile(srcCredFile); err == nil {
			credPath = filepath.Join(gogWorkspace, filepath.Base(srcCredFile))
			if err := os.WriteFile(credPath, data, 0600); err != nil {
				ui.Warn("Could not copy credential file: %v", err)
				credPath = ""
			} else {
				ui.Ok("Saved %s → %s", filepath.Base(srcCredFile), credPath)
			}
		}
	}

	// Fallback: synthesize a client_secret_*.json from client_id/secret if we
	// don't have the original file (user pasted values manually). Use the
	// standard Google naming convention so it matches the auto-detect pattern.
	if credPath == "" {
		credData, _ := json.MarshalIndent(map[string]any{
			"installed": map[string]any{
				"client_id":     clientID,
				"client_secret": clientSecret,
				"auth_uri":      "https://accounts.google.com/o/oauth2/auth",
				"token_uri":     "https://oauth2.googleapis.com/token",
				"redirect_uris": []string{"http://localhost"},
			},
		}, "", "  ")
		// Google's convention: client_secret_<client_id>.json (client_id already
		// ends with .apps.googleusercontent.com).
		fname := fmt.Sprintf("client_secret_%s.json", clientID)
		credPath = filepath.Join(gogWorkspace, fname)
		if err := os.WriteFile(credPath, credData, 0600); err != nil {
			ui.Warn("Could not write %s: %v", fname, err)
			return
		}
		ui.Ok("Saved %s → %s", fname, credPath)
	}

	// 2. Install gog CLI if missing.
	gogBin, err := installGogCLI()
	if err != nil {
		ui.Warn("%v", err)
		ui.Info("Install manually: https://github.com/steipete/gogcli")
		return
	}

	// gogEnv includes GOG_KEYRING_BACKEND=file and GOG_KEYRING_PASSWORD=""
	// so gog never prompts for a keyring passphrase interactively.
	gogEnv := append(os.Environ(),
		"GOG_KEYRING_BACKEND=file",
		"GOG_KEYRING_PASSWORD=",
	)

	// 3. gog auth credentials set — import client_id/secret into gog.
	cmd := exec.Command(gogBin, "auth", "credentials", "set", credPath, "--force", "--no-input")
	cmd.Env = gogEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("gog auth credentials set failed: %v", err)
		return
	}
	ui.Ok("Client credentials imported into gog")

	// 4. gog auth add <email> --services … — let gog run its own OAuth flow.
	//    This opens a second browser window, but it's the canonical path and
	//    ensures the refresh token is persisted in gog's keyring correctly.
	if email == "" {
		ui.Warn("Missing email; skipping gog auth add")
		ui.Info("Run manually: gog auth add you@gmail.com --services gmail,calendar,drive,contacts,sheets,docs")
		return
	}
	services := "gmail,calendar,drive,contacts,sheets,docs"
	ui.Info("Running: gog auth add %s --services %s", email, services)
	cmd = exec.Command(gogBin, "auth", "add", email, "--services", services)
	cmd.Env = gogEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("gog auth add failed: %v", err)
		ui.Info("Run manually: GOG_KEYRING_BACKEND=file GOG_KEYRING_PASSWORD= gog auth add %s --services %s", email, services)
	} else {
		ui.Ok("gog auth add succeeded for %s", email)
	}

	// 5. gog auth list — verify.
	fmt.Println()
	cmd = exec.Command(gogBin, "auth", "list")
	cmd.Env = gogEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("gog auth list failed: %v", err)
	}

	// 6. Install a gog wrapper in ~/.openclaw/bin that bakes in GOG_KEYRING_*
	//    env vars. This removes the dependency on the user's shell profile,
	//    so any process (openclaw-tui, subprocess, cron, CI) that calls gog
	//    via PATH gets the correct keyring backend automatically.
	if err := installGogWrapper(gogBin); err != nil {
		ui.Warn("Could not install gog wrapper: %v", err)
		// Fallback to the old shell-profile approach.
		writeGogEnvToShellProfile()
	}
}

// installGogWrapper creates ~/.openclaw/bin/gog (or gog.cmd on Windows) that
// sets GOG_KEYRING_BACKEND=file and GOG_KEYRING_PASSWORD= before exec'ing the
// real gog binary. It then prepends ~/.openclaw/bin to the user's PATH so the
// wrapper takes precedence over any other gog binary on the system.
func installGogWrapper(realGogPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(home, ".openclaw", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		wrapperPath := filepath.Join(binDir, "gog.cmd")
		content := "@echo off\r\n" +
			"set GOG_KEYRING_BACKEND=file\r\n" +
			"set GOG_KEYRING_PASSWORD=\r\n" +
			fmt.Sprintf("\"%s\" %%*\r\n", realGogPath)
		if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
			return err
		}
		ui.Ok("Installed gog wrapper → %s", wrapperPath)
		prependOpenclawBinToPath(binDir)
		return nil
	}

	wrapperPath := filepath.Join(binDir, "gog")
	content := "#!/bin/sh\n" +
		"# clawkit-managed wrapper — sets gog keyring env vars before exec.\n" +
		"export GOG_KEYRING_BACKEND=file\n" +
		"export GOG_KEYRING_PASSWORD=\n" +
		fmt.Sprintf("exec %q \"$@\"\n", realGogPath)
	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return err
	}
	// Ensure executable bit (WriteFile mode may be masked by umask).
	_ = os.Chmod(wrapperPath, 0755)
	ui.Ok("Installed gog wrapper → %s", wrapperPath)

	prependOpenclawBinToPath(binDir)
	return nil
}

// prependOpenclawBinToPath adds ~/.openclaw/bin to the FRONT of the user's
// PATH in their shell profile (so the wrapper wins over any other gog binary).
// On Windows it prepends via PowerShell to the user PATH env var.
func prependOpenclawBinToPath(binDir string) {
	if runtime.GOOS == "windows" {
		psCmd := fmt.Sprintf(
			`$p=[Environment]::GetEnvironmentVariable('Path','User'); `+
				`if($p -notlike '*%s*'){[Environment]::SetEnvironmentVariable('Path',"%s;$p",'User')}`,
			binDir, binDir,
		)
		if err := exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run(); err == nil {
			ui.Ok("Prepended %s to user PATH (restart terminal to take effect)", binDir)
		}
		return
	}

	const marker = "# Added by clawkit — openclaw bin (gog wrapper)"
	block := fmt.Sprintf("\n%s\nexport PATH=\"%s:$PATH\"\n", marker, binDir)

	home, _ := os.UserHomeDir()
	profiles := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}

	wrote := false
	for _, p := range profiles {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, _ := os.ReadFile(p)
		if strings.Contains(string(data), marker) {
			continue // already added
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		if _, err := f.WriteString(block); err != nil {
			f.Close()
			ui.Warn("Failed to write to %s: %v", p, err)
			continue
		}
		if err := f.Close(); err != nil {
			ui.Warn("Failed to close %s: %v", p, err)
			continue
		}
		ui.Ok("Prepended %s to PATH in %s", binDir, p)
		wrote = true
	}

	if !wrote {
		// All profiles already have the marker, or none exist.
		if _, err := os.Stat(filepath.Join(home, ".zshrc")); os.IsNotExist(err) {
			ui.Info("Add to your shell profile: export PATH=\"%s:$PATH\"", binDir)
		}
	} else {
		ui.Info("Restart your terminal (or agent) to pick up the new PATH")
	}
}

// writeGogEnvToShellProfile appends GOG_KEYRING_* env vars to the user's shell profile
// so gog never prompts for a keyring passphrase at runtime.
// On Windows it writes to the user PATH via PowerShell instead.
func writeGogEnvToShellProfile() {
	if runtime.GOOS == "windows" {
		vars := [][]string{
			{"GOG_KEYRING_BACKEND", "file"},
			{"GOG_KEYRING_PASSWORD", ""},
		}
		for _, kv := range vars {
			psCmd := fmt.Sprintf(`[Environment]::SetEnvironmentVariable('%s','%s','User')`, kv[0], kv[1])
			if err := exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run(); err != nil {
				ui.Warn("Failed to set %s: %v", kv[0], err)
			}
		}
		ui.Ok("GOG_KEYRING_BACKEND and GOG_KEYRING_PASSWORD set in user environment (restart terminal)")
		return
	}

	const block = "\n# Added by clawkit — gog keyring (no passphrase prompt)\n" +
		"export GOG_KEYRING_BACKEND=file\n" +
		"export GOG_KEYRING_PASSWORD=\n"

	// Detect which shell profile to write to.
	home, _ := os.UserHomeDir()
	profiles := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}

	// Write only to profiles that already exist.
	wrote := false
	for _, p := range profiles {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, _ := os.ReadFile(p)
		if strings.Contains(string(data), "GOG_KEYRING_BACKEND") {
			continue // already set
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		if _, err := f.WriteString(block); err != nil {
			f.Close()
			ui.Warn("Failed to write to %s: %v", p, err)
			continue
		}
		if err := f.Close(); err != nil {
			ui.Warn("Failed to close %s: %v", p, err)
			continue
		}
		ui.Ok("Added GOG_KEYRING vars to %s", p)
		wrote = true
	}

	if !wrote {
		ui.Info("Add to your shell profile to use gog without passphrase prompts:")
		ui.Info("  export GOG_KEYRING_BACKEND=file")
		ui.Info("  export GOG_KEYRING_PASSWORD=")
	} else {
		ui.Info("Run: source ~/.bashrc  (or open a new terminal)")
	}
}

// fetchGogLatestVersion gets the latest release tag from GitHub.
func fetchGogLatestVersion() (string, error) {
	// GitHub redirects /releases/latest to /releases/tag/vX.Y.Z
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/steipete/gogcli/releases/latest", nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect from GitHub releases/latest")
	}
	// Location: https://github.com/steipete/gogcli/releases/tag/v0.12.0
	parts := strings.Split(loc, "/")
	tag := parts[len(parts)-1]
	if tag == "" || tag[0] != 'v' {
		return "", fmt.Errorf("unexpected tag format: %s", tag)
	}
	return tag, nil
}

// installGogCLI installs the gog CLI binary from GitHub Releases.
// Supports macOS (arm64/amd64), Linux (arm64/amd64), and Windows (arm64/amd64).
func installGogCLI() (string, error) {
	if path, err := exec.LookPath("gog"); err == nil {
		return path, nil
	}

	ui.Info("gog CLI not found. Installing from GitHub Releases...")

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Fetch latest version tag from GitHub API.
	version, err := fetchGogLatestVersion()
	if err != nil {
		return "", fmt.Errorf("could not determine latest gog version: %w", err)
	}

	// Determine download URL.
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	dlURL := fmt.Sprintf(
		"https://github.com/steipete/gogcli/releases/download/%s/gogcli_%s_%s_%s.%s",
		version, strings.TrimPrefix(version, "v"), goos, goarch, ext,
	)
	ui.Info("Downloading gog %s for %s/%s...", version, goos, goarch)

	// Download to temp file.
	dlCtx, dlCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer dlCancel()

	dlReq, err := http.NewRequestWithContext(dlCtx, http.MethodGet, dlURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("download gog failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download gog failed: HTTP %d from %s", resp.StatusCode, dlURL)
	}

	tmpFile, err := os.CreateTemp("", "gogcli-*."+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("download incomplete: %w", err)
	}
	tmpFile.Close()

	// Extract to temp dir.
	tmpDir, err := os.MkdirTemp("", "gogcli-extract-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var extractErr error
	if goos == "windows" {
		extractErr = archive.ExtractZip(tmpFile.Name(), tmpDir)
	} else {
		extractErr = archive.ExtractTarGz(tmpFile.Name(), tmpDir)
	}
	if extractErr != nil {
		return "", fmt.Errorf("extract gog archive: %w", extractErr)
	}

	// Find the gog binary in extracted files.
	binName := "gog"
	if goos == "windows" {
		binName = "gog.exe"
	}
	extractedBin := filepath.Join(tmpDir, binName)
	if _, err := os.Stat(extractedBin); os.IsNotExist(err) {
		// Fallback: search recursively in case archive has a nested structure.
		found := ""
		_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && info.Name() == binName {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found == "" {
			return "", fmt.Errorf("gog binary not found in extracted archive at %s", tmpDir)
		}
		extractedBin = found
	}

	// Determine install directory per platform.
	installDir := installBinDir()
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("create install dir %s: %w", installDir, err)
	}

	destPath := filepath.Join(installDir, binName)
	data, err := os.ReadFile(extractedBin)
	if err != nil {
		return "", fmt.Errorf("read extracted binary: %w", err)
	}

	// Try direct write first, fall back to sudo on Unix.
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		if goos != "windows" {
			ui.Info("Need sudo to install to %s", installDir)
			cmd := exec.Command("sudo", "cp", extractedBin, destPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("install gog to %s failed: %w", destPath, err)
			}
			if err := exec.Command("sudo", "chmod", "+x", destPath).Run(); err != nil {
				ui.Warn("chmod +x %s failed: %v", destPath, err)
			}
		} else {
			return "", fmt.Errorf("install gog failed: %w", err)
		}
	}

	// Ensure installDir is in PATH so the binary is discoverable.
	ensureInPath(installDir, goos)

	path, err := exec.LookPath("gog")
	if err != nil {
		return destPath, nil // not in PATH yet (restart terminal), but absolute path works
	}
	ui.Ok("Installed gog CLI to %s", path)
	return path, nil
}

// ensureInPath adds dir to the user's persistent PATH if not already present.
// On Windows it updates the user environment via PowerShell.
// On Linux/macOS it warns the user if the dir is not already in PATH.
func ensureInPath(dir, goos string) {
	if goos == "windows" {
		// Add to Windows user PATH via PowerShell (safe: reads current value first).
		psCmd := fmt.Sprintf(
			`$p=[Environment]::GetEnvironmentVariable('Path','User'); `+
				`if($p -notlike '*%s*'){[Environment]::SetEnvironmentVariable('Path',"$p;%s",'User')}`,
			dir, dir,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
		if err := cmd.Run(); err == nil {
			ui.Ok("Added %s to user PATH (restart terminal to take effect)", dir)
		}
		return
	}
	// Unix: check if dir is already in PATH.
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if p == dir {
			return
		}
	}
	ui.Warn("gog installed to %s — add it to your PATH if not already:", dir)
	ui.Info("  bash:  echo 'export PATH=\"%s:$PATH\"' >> ~/.bashrc && source ~/.bashrc", dir)
	ui.Info("  zsh:   echo 'export PATH=\"%s:$PATH\"' >> ~/.zshrc  && source ~/.zshrc", dir)
}

// installRequiredBins installs external CLI binaries required by a skill.
// Each bin name maps to a dedicated installer function.
func installRequiredBins(bins []string) {
	for _, bin := range bins {
		switch bin {
		case "gog":
			if path, err := installGogCLI(); err != nil {
				ui.Warn("Could not install gog CLI: %v", err)
				ui.Info("Install manually: https://github.com/steipete/gogcli/releases")
			} else {
				ui.Ok("gog CLI ready at %s", path)
			}
		default:
			ui.Warn("Unknown required bin '%s' — install it manually", bin)
		}
	}
}

// installBinDir returns the appropriate directory for installing CLI binaries.
// macOS/Linux: /usr/local/bin (standard), falls back to ~/.local/bin
// Windows: %LOCALAPPDATA%\clawkit\bin
func installBinDir() string {
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "clawkit", "bin")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Local", "clawkit", "bin")
	}
	// Unix: prefer /usr/local/bin if writable, else ~/.local/bin
	if f, err := os.OpenFile("/usr/local/bin/.clawkit-test", os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Close()
		os.Remove("/usr/local/bin/.clawkit-test")
		return "/usr/local/bin"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

// runOAuth runs the OAuth flow for a provider and saves tokens.
func runOAuth(providerName, skillDir string) error {
	provider, err := oauth.Get(providerName)
	if err != nil {
		return err
	}

	tokens, err := provider.Authenticate()
	if err != nil {
		return err
	}

	cfg, _ := config.LoadSkillConfig(skillDir)
	if cfg == nil {
		cfg = &config.SkillConfig{Tokens: map[string]string{}}
	}
	if cfg.Tokens == nil {
		cfg.Tokens = map[string]string{}
	}

	for k, v := range tokens {
		cfg.Tokens[k] = v
	}
	cfg.OAuthDone = true

	return config.SaveSkillConfig(skillDir, cfg)
}
