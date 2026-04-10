// Package main is the entry point for the clawkit CLI.
//
//go:generate go run ../gen-registry
package main

import (
	"fmt"
	"os"

	"github.com/rockship-co/clawkit/internal/installer"
	"github.com/rockship-co/clawkit/internal/ui"
	"github.com/rockship-co/clawkit/oauth"
)

var version = "0.1.0"

func init() {
	// Wire the promptInput function into the oauth package.
	oauth.PromptInput = ui.PromptInput
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "list":
		installer.CmdList()
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit install <skill-name> [--skip-oauth]")
			os.Exit(1)
		}
		skipOAuth := false
		for _, arg := range os.Args[3:] {
			if arg == "--skip-oauth" {
				skipOAuth = true
			}
		}
		installer.CmdInstall(os.Args[2], skipOAuth)
	case "update":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit update <skill-name>")
			os.Exit(1)
		}
		installer.CmdUpdate(os.Args[2])
	case "status":
		installer.CmdStatus()
	case "package":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit package <skill-name>")
			fmt.Println("  Packages a skill from skills/ into a .tar.gz for distribution")
			os.Exit(1)
		}
		installer.CmdPackage(os.Args[2])
	case "version", "--version", "-v":
		fmt.Printf("clawkit v%s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`clawkit v%s - Rockship Skill Installer

Usage:
  clawkit <command> [arguments]

Commands:
  list                  List available skills
  install <skill>       Install a skill + run OAuth setup
  update  <skill>       Update an installed skill
  status                Show installed skills
  package <skill>       Package a skill for distribution (dev)
  version               Print version

Examples:
  clawkit list
  clawkit install shop-hoa
`, version)
}
