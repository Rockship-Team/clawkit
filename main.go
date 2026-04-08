package main

import (
	"fmt"
	"os"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "list":
		cmdList()
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit install <skill-name>")
			os.Exit(1)
		}
		cmdInstall(os.Args[2])
	case "update":
		if len(os.Args) < 3 {
			fmt.Println("Usage: clawkit update <skill-name>")
			os.Exit(1)
		}
		cmdUpdate(os.Args[2])
	case "status":
		cmdStatus()
	case "version":
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
  version               Print version

Examples:
  clawkit list
  clawkit install shop-hoa-zalo
`, version)
}
