package installer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		}
	}

	// Run setup prompts (shop name, phone, etc.).
	userInputs := collectUserInputs(skill.SetupPrompts)

	// Handle custom flowers directory if provided.
	if flowersSource := userInputs["flowers_dir"]; flowersSource != "" {
		defaultFlowers := filepath.Join(targetDir, "flowers")
		os.RemoveAll(defaultFlowers)
		err = copyDir(flowersSource, defaultFlowers)
		if err != nil {
			ui.Warn("Could not copy custom images: %v", err)
			ui.Info("You can manually copy images to %s later", defaultFlowers)
		} else {
			ui.Ok("Custom product images installed")
		}
	}

	// Ensure flower directories match catalog.
	template.EnsureFlowerDirs(targetDir)

	// Process SKILL.md template.
	if err := template.Process(targetDir, userInputs); err != nil {
		ui.Fatal("Failed to process skill template: %v", err)
	}

	// Save config.
	cfg := &config.SkillConfig{
		SkillName:  skillName,
		Version:    skill.Version,
		OAuthDone:  len(skill.RequiresOAuth) > 0,
		UserInputs: userInputs,
	}
	if err := config.SaveSkillConfig(targetDir, cfg); err != nil {
		ui.Fatal("Failed to save config: %v", err)
	}

	fmt.Println()
	ui.Ok("'%s' is ready!", skillName)
	fmt.Printf("  Location: %s\n", targetDir)
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
