// Package main is the entry point for the clawkit CLI.
//
//go:generate go run ../gen-registry
package main

import (
	"fmt"
	"os"

	"github.com/rockship-co/clawkit/internal/installer"
)

var version = "0.1.0"

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
			fmt.Println("Usage: clawkit install <skill-name> [--profile <name>]")
			os.Exit(1)
		}
		profileName := ""
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--profile":
				if i+1 < len(os.Args) {
					i++
					profileName = os.Args[i]
				} else {
					fmt.Println("--profile requires a name")
					os.Exit(1)
				}
			}
		}
		installer.CmdInstall(os.Args[2], profileName)
	case "update":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit update <skill-name>")
			os.Exit(1)
		}
		installer.CmdUpdate(os.Args[2])
	case "uninstall", "remove":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit uninstall <skill-name>")
			os.Exit(1)
		}
		installer.CmdUninstall(os.Args[2])
	case "status":
		installer.CmdStatus()
	case "web":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit web <skill-name>")
			os.Exit(1)
		}
		installer.CmdWeb(os.Args[2])
	case "package":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit package <skill-name>")
			fmt.Println("  Packages a skill from skills/ into a .tar.gz for distribution")
			os.Exit(1)
		}
		installer.CmdPackage(os.Args[2])
	case "dashboard":
		port := 7432
		for i, arg := range os.Args[2:] {
			if arg == "--port" && i+1 < len(os.Args[2:]) {
				if p, err := fmt.Sscan(os.Args[i+3], &port); p == 0 || err != nil {
					fmt.Println("Usage: clawkit dashboard [--port <number>]")
					os.Exit(1)
				}
			}
		}
		installer.CmdDashboard(port)
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
  install <skill> [--profile <name>]  Install a skill (locks workspace to its persona)
  uninstall <skill>     Uninstall a skill (restores prior workspace files)
  update  <skill>       Update an installed skill
  status                Show installed skills
	web <skill>           Serve the skill web UI at http://localhost:7432
  dashboard             Start web dashboard (default port 7432)
  package <skill>       Package a skill for distribution (dev)
  version               Print version

Note: clawkit enforces a 1-skill-at-a-time model. Installing a new skill
will prompt to remove any previously installed skill first.

Examples:
  clawkit list
  clawkit install shop-hoa
  clawkit install ecommerce-bot --profile shop-hoa
  clawkit uninstall shop-hoa
`, version)
}
