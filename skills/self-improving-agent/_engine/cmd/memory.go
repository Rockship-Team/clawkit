package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	memoryCap = 2200
	userCap   = 1375
	separator = "\n§\n"
)

// memoryPath returns the path to a memory file in the OpenClaw workspace.
func memoryPath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		errOut(fmt.Sprintf("cannot get home dir: %v", err))
	}
	return filepath.Join(home, ".openclaw", "workspace", name)
}

// loadMemory reads a memory file, splits by §, deduplicates, and trims empty entries.
func loadMemory(name string) []string {
	p := memoryPath(name)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}

	raw := strings.Split(string(data), "§")
	seen := make(map[string]bool)
	var entries []string
	for _, e := range raw {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		entries = append(entries, e)
	}
	return entries
}

// saveMemory joins entries with § separator and writes the file.
func saveMemory(name string, entries []string) error {
	p := memoryPath(name)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := strings.Join(entries, separator)
	return os.WriteFile(p, []byte(content), 0644)
}

// memoryCharCount calculates total characters including separators.
func memoryCharCount(entries []string) int {
	if len(entries) == 0 {
		return 0
	}
	total := 0
	for _, e := range entries {
		total += len(e)
	}
	// Add separator lengths between entries
	total += len(separator) * (len(entries) - 1)
	return total
}

// capForFile returns the character cap for the given file.
func capForFile(name string) int {
	if name == "USER.md" {
		return userCap
	}
	return memoryCap
}

func cmdMemory(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli memory <show|get|set|replace|remove> [args...]")
	}

	switch args[0] {
	case "show":
		memoryShow()
	case "get":
		memoryGet(args[1:])
	case "set":
		memorySet(args[1:])
	case "replace":
		memoryReplace(args[1:])
	case "remove":
		memoryRemove(args[1:])
	default:
		errOut(fmt.Sprintf("unknown memory subcommand: %s", args[0]))
	}
}

// memoryShow displays both MEMORY.md and USER.md with char counts and caps.
func memoryShow() {
	memEntries := loadMemory("MEMORY.md")
	userEntries := loadMemory("USER.md")

	jsonOut(map[string]interface{}{
		"status": "ok",
		"memory": map[string]interface{}{
			"entries": memEntries,
			"chars":   memoryCharCount(memEntries),
			"cap":     memoryCap,
		},
		"user": map[string]interface{}{
			"entries": userEntries,
			"chars":   memoryCharCount(userEntries),
			"cap":     userCap,
		},
	})
}

// memoryGet shows entries + chars + cap for one file.
func memoryGet(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli memory get <MEMORY.md|USER.md>")
	}
	name := args[0]
	entries := loadMemory(name)
	cap := capForFile(name)

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"file":    name,
		"entries": entries,
		"chars":   memoryCharCount(entries),
		"cap":     cap,
	})
}

// memorySet adds an entry, rejects duplicates, rejects if over cap.
func memorySet(args []string) {
	if len(args) < 2 {
		errOut("usage: vault-cli memory set <file> <entry>")
	}
	name := args[0]
	entry := strings.TrimSpace(args[1])
	if entry == "" {
		errOut("entry cannot be empty")
	}

	entries := loadMemory(name)
	cap := capForFile(name)

	// Check for duplicate
	for _, e := range entries {
		if e == entry {
			errOut("duplicate entry already exists")
		}
	}

	// Check if adding would exceed cap
	newEntries := append(entries, entry)
	if memoryCharCount(newEntries) > cap {
		errOut(fmt.Sprintf("over cap (%d/%d chars). Remove or replace an entry first", memoryCharCount(newEntries), cap))
	}

	if err := saveMemory(name, newEntries); err != nil {
		errOut(fmt.Sprintf("cannot save memory: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status": "ok",
		"file":   name,
		"chars":  memoryCharCount(newEntries),
		"cap":    cap,
	})
}

// memoryReplace finds the entry containing old_substring and replaces it.
func memoryReplace(args []string) {
	if len(args) < 3 {
		errOut("usage: vault-cli memory replace <file> <old_substring> <new_entry>")
	}
	name := args[0]
	oldSub := args[1]
	newEntry := strings.TrimSpace(args[2])

	entries := loadMemory(name)
	cap := capForFile(name)

	found := -1
	for i, e := range entries {
		if strings.Contains(e, oldSub) {
			found = i
			break
		}
	}
	if found < 0 {
		errOut(fmt.Sprintf("no entry containing %q", oldSub))
	}

	entries[found] = newEntry

	if memoryCharCount(entries) > cap {
		errOut(fmt.Sprintf("replacement would exceed cap (%d/%d chars)", memoryCharCount(entries), cap))
	}

	if err := saveMemory(name, entries); err != nil {
		errOut(fmt.Sprintf("cannot save memory: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status":   "ok",
		"file":     name,
		"replaced": true,
		"chars":    memoryCharCount(entries),
		"cap":      cap,
	})
}

// memoryRemove removes the first entry containing the given substring.
func memoryRemove(args []string) {
	if len(args) < 2 {
		errOut("usage: vault-cli memory remove <file> <substring>")
	}
	name := args[0]
	sub := args[1]

	entries := loadMemory(name)
	cap := capForFile(name)

	found := -1
	for i, e := range entries {
		if strings.Contains(e, sub) {
			found = i
			break
		}
	}
	if found < 0 {
		errOut(fmt.Sprintf("no entry containing %q", sub))
	}

	entries = append(entries[:found], entries[found+1:]...)

	if err := saveMemory(name, entries); err != nil {
		errOut(fmt.Sprintf("cannot save memory: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"file":    name,
		"removed": true,
		"chars":   memoryCharCount(entries),
		"cap":     cap,
	})
}
