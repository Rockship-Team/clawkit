package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdList() {
	reg, err := loadRegistry()
	if err != nil {
		fatal("%v", err)
	}

	fmt.Println("Available skills:")
	fmt.Println()
	for name, skill := range reg.Skills {
		installed := ""
		skillDir := filepath.Join(getSkillsDir(), name)
		if _, err := os.Stat(skillDir); err == nil {
			installed = " [installed]"
		}
		fmt.Printf("  %-25s %s%s\n", name, skill.Description, installed)
		if len(skill.RequiresOAuth) > 0 {
			fmt.Printf("  %-25s requires: %s\n", "", strings.Join(skill.RequiresOAuth, ", "))
		}
	}
}

func cmdInstall(skillName string, skipOAuth ...bool) {
	shouldSkipOAuth := len(skipOAuth) > 0 && skipOAuth[0]

	// Check platform is installed and get skills directory
	skillsDir := preflight()
	fmt.Println()

	reg, err := loadRegistry()
	if err != nil {
		fatal("%v", err)
	}

	skill, exists := reg.GetSkill(skillName)
	if !exists {
		fatal("Skill '%s' not found. Run 'clawkit list' to see available skills.", skillName)
	}

	fmt.Printf("▸ Installing %s v%s\n", skillName, skill.Version)
	fmt.Printf("  %s\n\n", skill.Description)

	// Check if already installed
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

	// Download skill (remote) or copy (local dev)
	os.MkdirAll(targetDir, 0755)
	err = downloadSkill(skillName, targetDir)
	if err != nil {
		os.RemoveAll(targetDir)
		fatal("Failed to install: %v", err)
	}
	ok("Skill files installed to %s", targetDir)

	// Run OAuth for each required provider
	if shouldSkipOAuth {
		warn("Skipping OAuth setup (--skip-oauth)")
	} else {
		for _, provider := range skill.RequiresOAuth {
			fmt.Println()
			info("Setting up %s authorization...", provider)
			err = runOAuthFlow(provider, targetDir)
			if err != nil {
				fatal("OAuth setup failed for %s: %v", provider, err)
			}
			ok("Connected to %s", provider)
		}
	}

	// Run setup prompts (tên shop, SĐT, etc.)
	userInputs := map[string]string{}
	if len(skill.SetupPrompts) > 0 {
		fmt.Println()
		info("Skill configuration")
		reader := bufio.NewReader(os.Stdin)
		for _, prompt := range skill.SetupPrompts {
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
			userInputs[prompt.Key] = answer
		}
	}

	// Handle custom flowers directory if provided
	if flowersSource := userInputs["flowers_dir"]; flowersSource != "" {
		defaultFlowers := filepath.Join(targetDir, "flowers")
		os.RemoveAll(defaultFlowers)
		err = copyDir(flowersSource, defaultFlowers)
		if err != nil {
			warn("Could not copy custom images: %v", err)
			info("You can manually copy images to %s later", defaultFlowers)
		} else {
			ok("Custom product images installed")
		}
	}

	// Ensure flower directories match catalog
	ensureFlowerDirs(targetDir)

	// Process SKILL.md template
	err = processTemplate(targetDir, userInputs)
	if err != nil {
		fatal("Failed to process skill template: %v", err)
	}

	// Save config
	cfg := &SkillConfig{
		SkillName:  skillName,
		Version:    skill.Version,
		OAuthDone:  len(skill.RequiresOAuth) > 0,
		UserInputs: userInputs,
	}
	err = saveSkillConfig(targetDir, cfg)
	if err != nil {
		fatal("Failed to save config: %v", err)
	}

	fmt.Println()
	ok("'%s' is ready!", skillName)
	fmt.Printf("  Location: %s\n", targetDir)
}

func cmdUpdate(skillName string) {
	targetDir := filepath.Join(getSkillsDir(), skillName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fatal("Skill '%s' is not installed. Run 'clawkit install %s' first.", skillName, skillName)
	}

	// Load existing config to preserve tokens
	existingCfg, err := loadSkillConfig(targetDir)
	if err != nil {
		info("No existing config found, doing fresh install")
		cmdInstall(skillName)
		return
	}

	// Remove old files except config.json
	entries, _ := os.ReadDir(targetDir)
	for _, e := range entries {
		if e.Name() != configFileName {
			os.RemoveAll(filepath.Join(targetDir, e.Name()))
		}
	}

	// Download new skill files
	err = downloadSkill(skillName, targetDir)
	if err != nil {
		fatal("Failed to update: %v", err)
	}

	// Restore config
	saveSkillConfig(targetDir, existingCfg)

	// Re-process template with existing config values
	if existingCfg.UserInputs != nil {
		ensureFlowerDirs(targetDir)
		err = processTemplate(targetDir, existingCfg.UserInputs)
		if err != nil {
			warn("Template processing failed: %v", err)
		}
	}

	ok("'%s' updated to latest version", skillName)
}

func cmdStatus() {
	skillsDir := getSkillsDir()
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		fmt.Println("No skills installed yet.")
		fmt.Println("Run 'rockship list' to see available skills.")
		return
	}

	fmt.Println("Installed skills:")
	fmt.Println()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfg, err := loadSkillConfig(filepath.Join(skillsDir, e.Name()))
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

func cmdPackage(skillName string) {
	sourceDir := filepath.Join("skills", skillName)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fatal("Skill '%s' not found in skills/ directory", skillName)
	}

	os.MkdirAll("dist", 0755)
	outputPath := filepath.Join("dist", skillName+".tar.gz")

	info("Packaging %s...", skillName)
	err := createTarGz(sourceDir, outputPath)
	if err != nil {
		fatal("Failed to package: %v", err)
	}

	// Show file size
	fi, _ := os.Stat(outputPath)
	sizeMB := float64(fi.Size()) / 1024 / 1024

	ok("Packaged: %s (%.1f MB)", outputPath, sizeMB)
	fmt.Println()
	fmt.Println("  Upload this file to GitHub Releases:")
	fmt.Printf("  gh release upload latest %s\n", outputPath)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
