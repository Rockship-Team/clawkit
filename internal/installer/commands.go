package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"encoding/json"
	"io"
	"net/http"
	"runtime"

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
func CmdInstall(skillName string, skipOAuth ...bool) {
	shouldSkipOAuth := len(skipOAuth) > 0 && skipOAuth[0]

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

	// Install required skills first.
	for _, depSkill := range skill.RequiresSkills {
		depDir := filepath.Join(skillsDir, depSkill)
		if _, err := os.Stat(depDir); os.IsNotExist(err) {
			fmt.Printf("  Skill '%s' requires '%s' — installing dependency first...\n\n", skillName, depSkill)
			CmdInstall(depSkill, shouldSkipOAuth)
			fmt.Println()
			fmt.Printf("▸ Resuming installation of %s...\n\n", skillName)
		} else {
			ui.Ok("Dependency '%s' already installed", depSkill)
		}
	}

	// Check if already installed.
	targetDir := filepath.Join(skillsDir, skillName)
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("  Skill already installed at %s\n", targetDir)
		fmt.Print("  Overwrite? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Cancelled.")
			return
		}
		os.RemoveAll(targetDir)
	}

	// Download skill (remote) or copy (local dev).
	os.MkdirAll(targetDir, 0755)
	err = downloadSkill(skillName, targetDir)
	if err != nil {
		os.RemoveAll(targetDir)
		ui.Fatal("Failed to install: %v", err)
	}
	ui.Ok("Skill files installed to %s", targetDir)

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

	// Move IDENTITY.md and SOUL.md to ~/.openclaw/workspace if present.
	moveWorkspaceFiles(targetDir)

	// Ensure flower directories match catalog.
	template.EnsureFlowerDirs(targetDir)

	// Initialize database if init_db.py exists.
	initDB := filepath.Join(targetDir, "init_db.py")
	if _, err := os.Stat(initDB); err == nil {
		pythonBin := findPython()
		if pythonBin == "" {
			ui.Warn("Python not found. Run manually: python3 %s", initDB)
		} else {
			cmd := exec.Command(pythonBin, initDB)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				ui.Warn("Database init failed: %v (you can run manually: %s %s)", err, pythonBin, initDB)
			} else {
				ui.Ok("Database initialized")
			}
		}
	}

	// Save config — load existing to preserve tokens written by runOAuth.
	cfg, _ := config.LoadSkillConfig(targetDir)
	if cfg == nil {
		cfg = &config.SkillConfig{}
	}
	cfg.SkillName = skillName
	cfg.Version = skill.Version
	cfg.OAuthDone = len(skill.RequiresOAuth) > 0
	if err := config.SaveSkillConfig(targetDir, cfg); err != nil {
		ui.Fatal("Failed to save config: %v", err)
	}

	fmt.Println()
	ui.Ok("'%s' installed!", skillName)
	fmt.Printf("  Location: %s\n", targetDir)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("  1. Edit config:  %s\n", filepath.Join(targetDir, "SKILL.md"))

	restartGateway()
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
		CmdInstall(skillName)
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

	// Restore config.
	config.SaveSkillConfig(targetDir, existingCfg)

	// Re-process template with existing config values.
	if existingCfg.UserInputs != nil {
		template.EnsureFlowerDirs(targetDir)
		if err := template.Process(targetDir, existingCfg.UserInputs); err != nil {
			ui.Warn("Template processing failed: %v", err)
		}
	}

	ui.Ok("'%s' updated to latest version", skillName)
	restartGateway()
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
		oauthStatus := ""
		if cfg.OAuthDone {
			oauthStatus = " [oauth: connected]"
		}
		fmt.Printf("  %-25s v%s%s\n", cfg.SkillName, cfg.Version, oauthStatus)
	}
}

// CmdPackage packages a skill from skills/ into a .tar.gz for distribution.
func CmdPackage(skillName string) {
	sourceDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		ui.Fatal("Skill '%s' not found in skills/ directory", skillName)
	}

	os.MkdirAll("dist", 0755)
	outputPath := filepath.Join("dist", skillName+".tar.gz")

	ui.Info("Packaging %s...", skillName)
	if err := archive.CreateTarGz(sourceDir, outputPath); err != nil {
		ui.Fatal("Failed to package: %v", err)
	}

	fi, _ := os.Stat(outputPath)
	sizeMB := float64(fi.Size()) / 1024 / 1024

	ui.Ok("Packaged: %s (%.1f MB)", outputPath, sizeMB)
	fmt.Println()
	fmt.Println("  Upload this file to GitHub Releases:")
	fmt.Printf("  gh release upload latest %s\n", outputPath)
}

// collectUserInputs prompts the user for each setup prompt.
func collectUserInputs(prompts []Prompt) map[string]string {
	inputs := map[string]string{}
	if len(prompts) == 0 {
		return inputs
	}

	fmt.Println()
	ui.Info("Skill configuration")
	reader := bufio.NewReader(os.Stdin)
	for _, prompt := range prompts {
		label := prompt.Label
		if prompt.Default != "" {
			label = fmt.Sprintf("%s [%s]", label, prompt.Default)
		}
		fmt.Printf("  %s: ", label)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer == "" && prompt.Default != "" {
			answer = prompt.Default
		}
		inputs[prompt.Key] = answer
	}
	return inputs
}

// postOAuthGmail writes credential files into the skill dir and configures gog CLI.
// Flow:
//  1. Write credential.json (Google client creds format) → gog auth credentials set
//  2. Write token.json (refresh token) → gog auth tokens import
//  3. Set GOG_ACCOUNT in SKILL.md
func postOAuthGmail(skillDir string, tokens map[string]string) {
	clientID := tokens["google_client_id"]
	clientSecret := tokens["google_client_secret"]
	email := tokens["gmail_account"]
	refreshToken := tokens["refresh_token"]
	if clientID == "" || clientSecret == "" {
		return
	}

	// 1. Write credential.json — Google OAuth client credentials format.
	credData, _ := json.MarshalIndent(map[string]any{
		"installed": map[string]any{
			"client_id":     clientID,
			"client_secret": clientSecret,
			"auth_uri":      "https://accounts.google.com/o/oauth2/auth",
			"token_uri":     "https://oauth2.googleapis.com/token",
			"redirect_uris": []string{"http://localhost:9876/callback"},
		},
	}, "", "  ")
	credPath := filepath.Join(skillDir, "credential.json")
	if err := os.WriteFile(credPath, credData, 0600); err != nil {
		ui.Warn("Could not write credential.json: %v", err)
		return
	}
	ui.Ok("Saved credential.json")

	// 2. Write token.json — refresh token for gog import.
	if email != "" && refreshToken != "" {
		tokData, _ := json.MarshalIndent(map[string]string{
			"email":         email,
			"refresh_token": refreshToken,
		}, "", "  ")
		tokPath := filepath.Join(skillDir, "token.json")
		if err := os.WriteFile(tokPath, tokData, 0600); err != nil {
			ui.Warn("Could not write token.json: %v", err)
		} else {
			ui.Ok("Saved token.json")
		}
	}

	// 3. Install gog CLI if missing.
	gogBin, err := installGogCLI()
	if err != nil {
		ui.Warn("%v", err)
		ui.Info("Install manually: https://github.com/steipete/gogcli")
		return
	}

	// gogEnv returns os.Environ() plus GOG_KEYRING_BACKEND=file and GOG_KEYRING_PASSWORD=""
	// so gog never prompts for a passphrase interactively on any platform.
	gogEnv := append(os.Environ(),
		"GOG_KEYRING_BACKEND=file",
		"GOG_KEYRING_PASSWORD=",
	)

	// 4. gog auth credentials set — import client_id/secret into gog keyring.
	cmd := exec.Command(gogBin, "auth", "credentials", "set", credPath, "--force", "--no-input")
	cmd.Env = gogEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("gog auth credentials set failed: %v", err)
	} else {
		ui.Ok("Client credentials imported into gog")
	}

	// 5. gog auth tokens import — import refresh_token into gog keyring.
	tokPath := filepath.Join(skillDir, "token.json")
	if _, err := os.Stat(tokPath); err == nil {
		cmd = exec.Command(gogBin, "auth", "tokens", "import", tokPath, "--force", "--no-input")
		cmd.Env = gogEnv
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ui.Warn("gog auth tokens import failed: %v", err)
			ui.Info("Run manually: GOG_KEYRING_BACKEND=file GOG_KEYRING_PASSWORD= gog auth tokens import %s --force", tokPath)
		} else {
			ui.Ok("Refresh token imported into gog for %s", email)
		}
	}

	// 6. Persist keyring env vars to shell profile so gog works at runtime without prompts.
	writeGogEnvToShellProfile()
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
			exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run()
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
		f.WriteString(block)
		f.Close()
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
	req, err := http.NewRequest(http.MethodHead, "https://github.com/steipete/gogcli/releases/latest", nil)
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
	resp, err := http.Get(dlURL)
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
	os.MkdirAll(installDir, 0755)

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
			exec.Command("sudo", "chmod", "+x", destPath).Run()
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
	ui.Info("  echo 'export PATH=\"%s:$PATH\"' >> ~/.bashrc && source ~/.bashrc", dir)
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

// moveWorkspaceFiles moves IDENTITY.md and SOUL.md from the skill dir to
// ~/.openclaw/workspace if they exist.
func moveWorkspaceFiles(skillDir string) {
	home, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(home, ".openclaw", "workspace")
	files := []string{"IDENTITY.md", "SOUL.md"}
	moved := false
	for _, f := range files {
		src := filepath.Join(skillDir, f)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			ui.Warn("Could not create workspace dir: %v", err)
			continue
		}
		dst := filepath.Join(workspaceDir, f)
		if err := moveFile(src, dst); err != nil {
			ui.Warn("Could not move %s to workspace: %v", f, err)
			continue
		}
		ui.Ok("Moved %s → %s", f, dst)
		moved = true
	}
	if moved {
		ui.Info("Workspace files placed in %s", workspaceDir)
	}
}

// moveFile moves src to dst, falling back to copy+delete if os.Rename fails
// (e.g. cross-device on Linux, or dst already exists on Windows).
func moveFile(src, dst string) error {
	// Remove dst first so Windows rename doesn't fail on existing file.
	os.Remove(dst)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Fallback: copy then delete (handles cross-device / cross-drive moves).
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return err
	}
	return os.Remove(src)
}

// restartGateway attempts to restart the openclaw gateway.
// If openclaw is not found or the gateway is not running, it warns instead of failing.
func restartGateway() {
	openclaw, err := exec.LookPath("openclaw")
	if err != nil {
		ui.Warn("openclaw not found — restart gateway manually when ready")
		return
	}
	fmt.Println()
	ui.Info("Restarting openclaw gateway...")
	cmd := exec.Command(openclaw, "gateway", "restart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ui.Warn("Gateway restart failed: %v", err)
		ui.Info("Run manually: openclaw gateway restart")
		return
	}
	ui.Ok("Gateway restarted")
}

// findPython returns the path to a Python 3 interpreter.
// Tries "python3" first (macOS/Linux), falls back to "python" (Windows).
func findPython() string {
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
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
