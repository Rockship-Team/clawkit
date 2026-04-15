package installer

import (
	"bufio"
	"context"
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
	}
}

// CmdInstall installs a skill with OAuth setup and configuration.
func CmdInstall(skillName string, profileName string) {
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

	ui.Info("Installing %s v%s", skillName, skill.Version)
	fmt.Printf("  %s\n\n", skill.Description)

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
	err = downloadSkill(skillName, targetDir, skill.Exclude)
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

	// Lock down the workspace into this skill's persona:
	//   1. Remove any previously installed skill (1-skill-at-a-time model)
	//   2. Back up existing workspace MD files
	//   3. Copy bootstrap-files/* from the new skill to workspace root
	//   4. Delete generic assistant files (BOOTSTRAP.md, HEARTBEAT.md, TOOLS.md)
	//   5. Reset prior conversation sessions
	//   6. Set agents.defaults.skills = [<skillName>]
	LockdownWorkspace(skillsDir, targetDir, skillName)

	// Remove bootstrap-files from the installed skill directory — they've
	// already been applied to the workspace root by LockdownWorkspace and
	// are no longer needed inside the skill dir.
	os.RemoveAll(filepath.Join(targetDir, BootstrapFilesDirName))

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

	// Initialize database from schema.json.
	if schemaErr := initSchema(targetDir, cfg, profileValues); schemaErr != nil {
		os.RemoveAll(targetDir)
		ui.Fatal("Database init failed: %v", schemaErr)
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
			os.RemoveAll(targetDir)
			ui.Fatal("Template processing failed: %v", err)
		}
	}

	// Replace {key} placeholders in SKILL.md with OAuth token values
	// (e.g. {spreadsheet_id}, {gmail_account}) so the skill prompt is ready to use.
	if len(cfg.Tokens) > 0 {
		if err := template.ProcessTokens(targetDir, cfg.Tokens); err != nil {
			os.RemoveAll(targetDir)
			ui.Fatal("Could not replace SKILL.md placeholders: %v", err)
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
		CmdInstall(skillName, "")
		return
	}

	// Remove old files except clawkit.json (installed config).
	entries, _ := os.ReadDir(targetDir)
	for _, e := range entries {
		if e.Name() != config.ConfigFileName {
			os.RemoveAll(filepath.Join(targetDir, e.Name()))
		}
	}

	// Load registry to get exclude patterns.
	var excludePatterns []string
	if reg, regErr := loadRegistry(); regErr == nil {
		if skill, ok := reg.GetSkill(skillName); ok {
			excludePatterns = skill.Exclude
		}
	}

	// Download new skill files.
	if err := downloadSkill(skillName, targetDir, excludePatterns); err != nil {
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

	ui.Info("Uninstalling %s", skillName)
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

	// Load exclude patterns from registry if available.
	var excludePatterns []string
	if reg, regErr := loadRegistry(); regErr == nil {
		if skill, ok := reg.GetSkill(skillName); ok {
			excludePatterns = skill.Exclude
		}
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		ui.Fatal("Failed to create dist directory: %v", err)
	}
	outputPath := filepath.Join("dist", skillName+".tar.gz")

	ui.Info("Packaging %s...", skillName)
	if len(excludePatterns) > 0 {
		ui.Info("Excluding: %s", strings.Join(excludePatterns, ", "))
	}
	if err := archive.CreateTarGz(sourceDir, outputPath, excludePatterns); err != nil {
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
	shell := os.Getenv("SHELL")
	if strings.HasSuffix(shell, "/zsh") {
		ui.Info("  echo 'export PATH=\"%s:$PATH\"' >> ~/.zshrc && source ~/.zshrc", dir)
	} else if strings.HasSuffix(shell, "/bash") {
		ui.Info("  echo 'export PATH=\"%s:$PATH\"' >> ~/.bashrc && source ~/.bashrc", dir)
	} else {
		ui.Info("  export PATH=\"%s:$PATH\"", dir)
	}
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
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "clawkit", "bin")
		}
		return filepath.Join(home, "AppData", "Local", "clawkit", "bin")
	}
	// Unix: prefer /usr/local/bin if writable, else ~/.local/bin
	if f, err := os.OpenFile("/usr/local/bin/.clawkit-test", os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Close()
		os.Remove("/usr/local/bin/.clawkit-test")
		return "/usr/local/bin"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "clawkit", "bin")
	}
	return filepath.Join(home, ".local", "bin")
}
