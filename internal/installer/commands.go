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
	"sort"
	"strings"
	"time"

	"github.com/rockship-co/clawkit/internal/archive"
	"github.com/rockship-co/clawkit/internal/config"
	"github.com/rockship-co/clawkit/internal/template"
	"github.com/rockship-co/clawkit/internal/ui"
)

// CmdList lists all available skills and groups with install status.
func CmdList() {
	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}

	skillsDir := config.GetSkillsDir()

	// Skills that belong to a group — rendered under the group heading.
	inGroup := make(map[string]string)
	for group, members := range reg.Groups {
		for _, m := range members {
			inGroup[m] = group
		}
	}

	if len(reg.Groups) > 0 {
		fmt.Println("Available groups:")
		fmt.Println()
		groupNames := make([]string, 0, len(reg.Groups))
		for name := range reg.Groups {
			groupNames = append(groupNames, name)
		}
		sort.Strings(groupNames)
		for _, name := range groupNames {
			members := reg.Groups[name]
			fmt.Printf("  %s (%d skills)\n", name, len(members))
			for _, m := range members {
				skill := reg.Skills[m]
				installed := ""
				if _, err := os.Stat(filepath.Join(skillsDir, m)); err == nil {
					installed = " [installed]"
				}
				fmt.Printf("    %-25s %s%s\n", m, truncate(skill.Description, 80), installed)
			}
			fmt.Println()
		}
	}

	fmt.Println("Available skills:")
	fmt.Println()
	flatNames := make([]string, 0, len(reg.Skills))
	for name := range reg.Skills {
		if _, ok := inGroup[name]; ok {
			continue
		}
		flatNames = append(flatNames, name)
	}
	sort.Strings(flatNames)
	for _, name := range flatNames {
		skill := reg.Skills[name]
		installed := ""
		if _, err := os.Stat(filepath.Join(skillsDir, name)); err == nil {
			installed = " [installed]"
		}
		fmt.Printf("  %-25s %s%s\n", name, truncate(skill.Description, 100), installed)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// CmdInstall resolves name as a flat skill or a group, then installs one
// or more skills. When name is a group, children selects a subset; empty
// children means install all members.
func CmdInstall(name string, children ...string) {
	skillsDir := config.Preflight()
	fmt.Println()

	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}

	targets, err := resolveInstallTargets(reg, name, children)
	if err != nil {
		ui.Fatal("%v", err)
	}

	stdinReader := bufio.NewReader(os.Stdin)
	for i, skillName := range targets {
		if i > 0 {
			fmt.Println()
		}
		installOne(skillsDir, reg, skillName, stdinReader)
	}
}

// resolveInstallTargets maps name + children into a flat list of skill
// names to install. Rules:
//   - name is a flat skill and no children → [name]
//   - name is a group and no children → all group members
//   - name is a group and children non-empty → those children (must all
//     be members of the group)
//   - name is a skill and children non-empty → error
func resolveInstallTargets(reg *Registry, name string, children []string) ([]string, error) {
	_, isSkill := reg.GetSkill(name)
	members := reg.GroupMembers(name)

	if len(children) > 0 {
		if members == nil {
			return nil, fmt.Errorf("'%s' is not a group; extra arguments %v are not allowed", name, children)
		}
		memberSet := make(map[string]bool, len(members))
		for _, m := range members {
			memberSet[m] = true
		}
		for _, c := range children {
			if !memberSet[c] {
				return nil, fmt.Errorf("'%s' is not a member of group '%s' (members: %s)", c, name, strings.Join(members, ", "))
			}
		}
		return children, nil
	}

	if members != nil {
		return members, nil
	}
	if isSkill {
		return []string{name}, nil
	}
	return nil, fmt.Errorf("'%s' not found. Run 'clawkit list' to see available skills and groups.", name)
}

func installOne(skillsDir string, reg *Registry, skillName string, stdinReader *bufio.Reader) {
	skill, exists := reg.GetSkill(skillName)
	if !exists {
		ui.Fatal("Skill '%s' not found in registry.", skillName)
	}

	ui.Info("Installing %s v%s", skillName, skill.Version)
	fmt.Printf("  %s\n\n", skill.Description)

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

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		ui.Fatal("Failed to create skill directory: %v", err)
	}
	if err := downloadSkill(skillName, targetDir, skill.Exclude); err != nil {
		os.RemoveAll(targetDir)
		ui.Fatal("Failed to install: %v", err)
	}
	ui.Ok("Skill files installed to %s", targetDir)

	if len(skill.RequiresBins) > 0 {
		fmt.Println()
		ui.Info("Installing required CLI tools...")
		installRequiredBins(skill.RequiresBins)
	}

	userInputs := make(map[string]string)
	if len(skill.SetupPrompts) > 0 {
		fmt.Println()
		for _, p := range skill.SetupPrompts {
			prompt := p.Label
			if p.Placeholder != "" {
				prompt = fmt.Sprintf("%s [%s]", p.Label, p.Placeholder)
			}
			fmt.Printf("  %s: ", prompt)
			line, _ := stdinReader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "" {
				line = p.Placeholder
			}
			userInputs[p.Key] = line
		}
	}

	SetupWorkspace(skillsDir, skillName)

	if len(userInputs) > 0 {
		if err := template.Process(targetDir, userInputs); err != nil {
			os.RemoveAll(targetDir)
			ui.Fatal("Template processing failed: %v", err)
		}
	}

	if n, err := applyBootstrap(targetDir, skillsDir); err != nil {
		ui.Warn("Could not apply bootstrap files: %v", err)
	} else if n > 0 {
		ui.Ok("Applied %d bootstrap file(s) to workspace", n)
	}

	cfg := &config.SkillConfig{
		Version:    skill.Version,
		UserInputs: userInputs,
	}
	if err := config.SaveSkillConfig(targetDir, cfg); err != nil {
		ui.Fatal("Failed to save config: %v", err)
	}

	fmt.Println()
	ui.Ok("'%s' installed!", skillName)
	fmt.Printf("  Location: %s\n", targetDir)
}

// applyBootstrap copies top-level .md files from the skill's _bootstrap/
// directory into the workspace root (parent of skillsDir), overwriting
// any existing files there. Returns the number of files copied.
func applyBootstrap(skillDir, skillsDir string) (int, error) {
	bootstrapDir := filepath.Join(skillDir, "_bootstrap")
	entries, err := os.ReadDir(bootstrapDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	workspaceDir := filepath.Dir(skillsDir)
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		src := filepath.Join(bootstrapDir, e.Name())
		dst := filepath.Join(workspaceDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return count, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return count, fmt.Errorf("write %s: %w", e.Name(), err)
		}
		count++
	}
	return count, nil
}

// CmdUpdate resolves name as a flat skill or a group, then updates one or
// more installed skills, re-baking stored user_inputs into the new SKILL.md.
func CmdUpdate(name string, children ...string) {
	reg, err := loadRegistry()
	if err != nil {
		ui.Fatal("%v", err)
	}
	targets, err := resolveInstallTargets(reg, name, children)
	if err != nil {
		ui.Fatal("%v", err)
	}
	for i, skillName := range targets {
		if i > 0 {
			fmt.Println()
		}
		updateOne(reg, skillName)
	}
}

func updateOne(reg *Registry, skillName string) {
	targetDir := filepath.Join(config.GetSkillsDir(), skillName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		ui.Warn("'%s' is not installed — skipping. Run 'clawkit install %s' first.", skillName, skillName)
		return
	}

	existingCfg, cfgErr := config.LoadSkillConfig(targetDir)
	if cfgErr != nil {
		ui.Info("No existing config for '%s', doing fresh install", skillName)
		CmdInstall(skillName)
		return
	}

	skill, ok := reg.GetSkill(skillName)
	if !ok {
		ui.Warn("'%s' not found in registry — skipping.", skillName)
		return
	}

	entries, _ := os.ReadDir(targetDir)
	for _, e := range entries {
		if e.Name() != config.ConfigFileName {
			os.RemoveAll(filepath.Join(targetDir, e.Name()))
		}
	}

	if err := downloadSkill(skillName, targetDir, skill.Exclude); err != nil {
		ui.Fatal("Failed to update '%s': %v", skillName, err)
	}

	if len(existingCfg.UserInputs) > 0 {
		if err := template.Process(targetDir, existingCfg.UserInputs); err != nil {
			ui.Warn("Template processing failed: %v", err)
		}
	}

	cfg := &config.SkillConfig{
		Version:    skill.Version,
		UserInputs: existingCfg.UserInputs,
	}
	if err := config.SaveSkillConfig(targetDir, cfg); err != nil {
		ui.Warn("Could not save config: %v", err)
	}

	ui.Ok("'%s' updated to v%s", skillName, skill.Version)
	ui.Info("Restart the gateway to pick up changes: openclaw gateway restart")
}

// CmdUninstall removes a skill and its allowlist entry.
func CmdUninstall(skillName string) {
	skillsDir := config.Preflight()
	fmt.Println()

	targetDir := filepath.Join(skillsDir, skillName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		ui.Fatal("Skill '%s' is not installed.", skillName)
	}

	ui.Info("Uninstalling %s", skillName)
	fmt.Printf("  This will:\n")
	fmt.Printf("    • Delete %s\n", targetDir)
	fmt.Printf("    • Remove '%s' from the skill allowlist\n\n", skillName)
	fmt.Print("  Continue? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("  Cancelled.")
		return
	}

	RemoveFromWorkspace(skillsDir, skillName)

	if err := os.RemoveAll(targetDir); err != nil {
		ui.Fatal("Could not remove skill directory: %v", err)
	}
	ui.Ok("Removed %s", targetDir)

	fmt.Println()
	ui.Ok("'%s' uninstalled", skillName)
	ui.Info("Restart the gateway to apply: openclaw gateway restart")
}

// CmdStatus shows all installed skills with their version.
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
		fmt.Printf("  %-25s v%s\n", e.Name(), cfg.Version)
	}
}

func fetchGogLatestVersion() (string, error) {
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
	parts := strings.Split(loc, "/")
	tag := parts[len(parts)-1]
	if tag == "" || tag[0] != 'v' {
		return "", fmt.Errorf("unexpected tag format: %s", tag)
	}
	return tag, nil
}

func installGogCLI() (string, error) {
	if path, err := exec.LookPath("gog"); err == nil {
		return path, nil
	}

	ui.Info("gog CLI not found. Installing from GitHub Releases...")

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	version, err := fetchGogLatestVersion()
	if err != nil {
		return "", fmt.Errorf("could not determine latest gog version: %w", err)
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	dlURL := fmt.Sprintf(
		"https://github.com/steipete/gogcli/releases/download/%s/gogcli_%s_%s_%s.%s",
		version, strings.TrimPrefix(version, "v"), goos, goarch, ext,
	)
	ui.Info("Downloading gog %s for %s/%s...", version, goos, goarch)

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

	binName := "gog"
	if goos == "windows" {
		binName = "gog.exe"
	}
	extractedBin := filepath.Join(tmpDir, binName)
	if _, err := os.Stat(extractedBin); os.IsNotExist(err) {
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

	installDir := installBinDir()
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return "", fmt.Errorf("create install dir %s: %w", installDir, err)
	}

	destPath := filepath.Join(installDir, binName)
	data, err := os.ReadFile(extractedBin)
	if err != nil {
		return "", fmt.Errorf("read extracted binary: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0o755); err != nil {
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

	ensureInPath(installDir, goos)

	path, err := exec.LookPath("gog")
	if err != nil {
		return destPath, nil
	}
	ui.Ok("Installed gog CLI to %s", path)
	return path, nil
}

func ensureInPath(dir, goos string) {
	if goos == "windows" {
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
	if f, err := os.OpenFile("/usr/local/bin/.clawkit-test", os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
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
