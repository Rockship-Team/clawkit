// Package main is the entry point for the clawkit CLI.
//
//go:generate go run ../gen-registry
package main

import (
	"fmt"
	"os"

	"github.com/rockship-co/clawkit/internal/installer"
)

// version is injected via -ldflags at build time (see Makefile).
// The default is only seen on `go install` / `go build` without ldflags.
var version = "dev"

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
			fmt.Println("Usage: clawkit install <name> [<member>...]")
			fmt.Println("  <name> is a flat skill, or a group to install all its members")
			fmt.Println("  extra args select specific members of a group")
			os.Exit(1)
		}
		installer.CmdInstall(os.Args[2], os.Args[3:]...)
	case "update":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit update <name> [<member>...]")
			os.Exit(1)
		}
		installer.CmdUpdate(os.Args[2], os.Args[3:]...)
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
  list                            List available skills and groups
  install <name> [<member>...]    Install a flat skill, a whole group, or
                                  selected members of a group
  update  <name> [<member>...]    Update (same resolution as install)
  uninstall <skill>               Uninstall a single skill
  status                Show installed skills
  web <skill>           Serve the skill web UI at http://localhost:7432
  dashboard             Start web dashboard (default port 7432)
  version               Print version
`, version)
}
