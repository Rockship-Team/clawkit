package main

import (
	"fmt"
	"strings"
)

func cmdNote(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli note <add|get|list|search|append> [args...]")
	}

	switch args[0] {
	case "add":
		noteAdd(args[1:])
	case "get":
		noteGet(args[1:])
	case "list":
		noteList(args[1:])
	case "search":
		noteSearch(args[1:])
	case "append":
		noteAppend(args[1:])
	default:
		errOut(fmt.Sprintf("unknown note subcommand: %s", args[0]))
	}
}

// noteAdd creates a note with frontmatter.
// Usage: note add <path> <body> [key=value...]
func noteAdd(args []string) {
	if len(args) < 2 {
		errOut("usage: vault-cli note add <path> <body> [key=value...]")
	}

	relPath := args[0]
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}
	body := args[1]

	meta := map[string]string{
		"created": vnToday(),
	}

	// Parse key=value pairs
	for _, kv := range args[2:] {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			errOut(fmt.Sprintf("invalid key=value pair: %s", kv))
		}
		meta[kv[:idx]] = kv[idx+1:]
	}

	content := buildNote(meta, body+"\n")

	vault := mustVaultPath()
	if err := writeNote(vault, relPath, content); err != nil {
		errOut(fmt.Sprintf("cannot write note: %v", err))
	}

	links := extractLinks(body)
	tags := extractTags(body)

	jsonOut(map[string]interface{}{
		"status": "ok",
		"path":   relPath,
		"links":  links,
		"tags":   tags,
	})
}

// noteGet reads a note and returns its frontmatter, body, links, and tags.
// Usage: note get <path>
func noteGet(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli note get <path>")
	}

	relPath := args[0]
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	vault := mustVaultPath()
	content, err := readNote(vault, relPath)
	if err != nil {
		errOut(fmt.Sprintf("cannot read note: %v", err))
	}

	meta, body := parseFrontmatter(content)
	links := extractLinks(body)
	tags := extractTags(body)

	result := map[string]interface{}{
		"status":      "ok",
		"path":        relPath,
		"frontmatter": meta,
		"body":        body,
		"links":       links,
		"tags":        tags,
	}
	jsonOut(result)
}

// noteList lists all notes with frontmatter summaries.
// Usage: note list [dir]
func noteList(args []string) {
	vault := mustVaultPath()

	dir := vault
	if len(args) > 0 {
		dir = vault + "/" + args[0]
	}

	notes, err := listNotes(dir)
	if err != nil {
		errOut(fmt.Sprintf("cannot list notes: %v", err))
	}

	type noteSummary struct {
		Path        string            `json:"path"`
		Frontmatter map[string]string `json:"frontmatter,omitempty"`
	}

	results := make([]noteSummary, 0, len(notes))
	for _, n := range notes {
		// If we're listing a subdirectory, we need to adjust the read path
		var readPath string
		if len(args) > 0 {
			readPath = args[0] + "/" + n
		} else {
			readPath = n
		}
		content, err := readNote(vault, readPath)
		if err != nil {
			continue
		}
		meta, _ := parseFrontmatter(content)
		results = append(results, noteSummary{
			Path:        readPath,
			Frontmatter: meta,
		})
	}

	jsonOut(map[string]interface{}{
		"status": "ok",
		"count":  len(results),
		"notes":  results,
	})
}

// noteSearch performs case-insensitive substring search across vault notes.
// Returns path + snippet (50 chars before, 100 after match).
// Usage: note search <query>
func noteSearch(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli note search <query>")
	}

	query := strings.ToLower(args[0])
	vault := mustVaultPath()
	notes, err := listNotes(vault)
	if err != nil {
		errOut(fmt.Sprintf("cannot list notes: %v", err))
	}

	type searchResult struct {
		Path    string `json:"path"`
		Snippet string `json:"snippet"`
	}

	var results []searchResult
	for _, n := range notes {
		content, err := readNote(vault, n)
		if err != nil {
			continue
		}
		lower := strings.ToLower(content)
		idx := strings.Index(lower, query)
		if idx < 0 {
			continue
		}

		// Build snippet: 50 chars before, 100 chars after match
		start := idx - 50
		if start < 0 {
			start = 0
		}
		end := idx + len(query) + 100
		if end > len(content) {
			end = len(content)
		}
		snippet := content[start:end]

		results = append(results, searchResult{
			Path:    n,
			Snippet: snippet,
		})
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"query":   args[0],
		"count":   len(results),
		"results": results,
	})
}

// noteAppend appends text to an existing note, or creates a new one.
// Usage: note append <path> <text>
func noteAppend(args []string) {
	if len(args) < 2 {
		errOut("usage: vault-cli note append <path> <text>")
	}

	relPath := args[0]
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}
	text := args[1]

	vault := mustVaultPath()
	existing, err := readNote(vault, relPath)
	if err != nil {
		// File doesn't exist — create new
		content := text + "\n"
		if err := writeNote(vault, relPath, content); err != nil {
			errOut(fmt.Sprintf("cannot write note: %v", err))
		}
		jsonOut(map[string]interface{}{
			"status":  "ok",
			"path":    relPath,
			"created": true,
		})
		return
	}

	// Append to existing content
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	content := existing + text + "\n"

	if err := writeNote(vault, relPath, content); err != nil {
		errOut(fmt.Sprintf("cannot write note: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status":   "ok",
		"path":     relPath,
		"appended": true,
	})
}
