package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintln(os.Stderr, `vault-cli — Obsidian vault manager for OpenClaw agents

Usage:
  vault-cli <command> [arguments...]

Commands:
  note      Create, read, update, list vault notes
  memory    Store and recall agent memories
  session   Manage chat sessions
  learn     Extract and store learnings
  search    Full-text search across vault and sessions`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "note":
		cmdNote(os.Args[2:])
	case "memory":
		cmdMemory(os.Args[2:])
	case "session":
		cmdSession(os.Args[2:])
	case "learn":
		cmdLearn(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	default:
		usage()
	}
}

func cmdNote(args []string) {
	errOut("not implemented")
}

func cmdMemory(args []string) {
	errOut("not implemented")
}

func cmdSession(args []string) {
	errOut("not implemented")
}

func cmdLearn(args []string) {
	errOut("not implemented")
}

func cmdSearch(args []string) {
	errOut("not implemented")
}
