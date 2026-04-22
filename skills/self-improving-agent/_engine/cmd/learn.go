package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// learnDir returns the skills directory inside the vault.
func learnDir() string {
	return filepath.Join(mustVaultPath(), "skills")
}

func cmdLearn(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli learn <save-skill|patch-skill|list|get> [args...]")
	}

	switch args[0] {
	case "save-skill":
		learnSaveSkill(args[1:])
	case "patch-skill":
		learnPatchSkill(args[1:])
	case "list":
		learnList()
	case "get":
		learnGet(args[1:])
	default:
		errOut(fmt.Sprintf("unknown learn subcommand: %s", args[0]))
	}
}

// learnSaveSkill creates a skill file with frontmatter.
// Usage: learn save-skill <name> <description> <procedure_body> [tags]
func learnSaveSkill(args []string) {
	if len(args) < 3 {
		errOut("usage: vault-cli learn save-skill <name> <description> <procedure_body> [tags]")
	}

	name := args[0]
	description := args[1]
	body := args[2]
	tags := ""
	if len(args) > 3 {
		tags = args[3]
	}

	meta := map[string]string{
		"name":        name,
		"description": description,
		"created":     vnToday(),
		"updated":     vnToday(),
	}
	if tags != "" {
		meta["tags"] = tags
	}

	content := buildNote(meta, body+"\n")

	vault := mustVaultPath()
	relPath := filepath.Join("skills", name+".md")
	if err := writeNote(vault, relPath, content); err != nil {
		errOut(fmt.Sprintf("cannot write skill: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status": "ok",
		"path":   relPath,
		"name":   name,
	})
}

// learnPatchSkill performs find-and-replace in a skill file, updating the `updated` date.
// Tries exact match first, falls back to whitespace-normalized fuzzy match.
// Usage: learn patch-skill <name> <find_text> <replace_text>
func learnPatchSkill(args []string) {
	if len(args) < 3 {
		errOut("usage: vault-cli learn patch-skill <name> <find_text> <replace_text>")
	}

	name := args[0]
	findText := args[1]
	replaceText := args[2]

	vault := mustVaultPath()
	relPath := filepath.Join("skills", name+".md")

	content, err := readNote(vault, relPath)
	if err != nil {
		errOut(fmt.Sprintf("cannot read skill: %v", err))
	}

	var newContent string

	// Try exact match first
	if strings.Contains(content, findText) {
		newContent = strings.Replace(content, findText, replaceText, 1)
	} else {
		// Fall back to whitespace-normalized fuzzy match
		normalizedFind := normalizeWhitespace(findText)
		// Build a regex that matches the find text with flexible whitespace
		escaped := regexp.QuoteMeta(normalizedFind)
		// Replace single spaces with \s+ to allow flexible whitespace matching
		pattern := strings.ReplaceAll(escaped, " ", `\s+`)
		re, err := regexp.Compile(pattern)
		if err != nil {
			errOut(fmt.Sprintf("cannot compile fuzzy pattern: %v", err))
		}

		loc := re.FindStringIndex(content)
		if loc == nil {
			errOut(fmt.Sprintf("text %q not found in skill (exact or fuzzy)", findText))
		}
		newContent = content[:loc[0]] + replaceText + content[loc[1]:]
	}

	// Update the `updated` frontmatter field
	newContent = updateFrontmatterField(newContent, "updated", vnToday())

	if err := writeNote(vault, relPath, newContent); err != nil {
		errOut(fmt.Sprintf("cannot write skill: %v", err))
	}

	jsonOut(map[string]interface{}{
		"status":  "ok",
		"path":    relPath,
		"patched": true,
	})
}

// normalizeWhitespace collapses all whitespace sequences to a single space and trims.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// learnList lists all skills with frontmatter.
func learnList() {
	vault := mustVaultPath()
	dir := filepath.Join(vault, "skills")
	notes, err := listNotes(dir)
	if err != nil {
		// Directory might not exist yet — return empty
		jsonOut(map[string]interface{}{
			"status": "ok",
			"count":  0,
			"skills": []interface{}{},
		})
		return
	}

	type skillSummary struct {
		Name        string            `json:"name"`
		Path        string            `json:"path"`
		Frontmatter map[string]string `json:"frontmatter,omitempty"`
	}

	var results []skillSummary
	for _, n := range notes {
		readPath := filepath.Join("skills", n)
		content, err := readNote(vault, readPath)
		if err != nil {
			continue
		}
		meta, _ := parseFrontmatter(content)
		skillName := strings.TrimSuffix(n, ".md")
		results = append(results, skillSummary{
			Name:        skillName,
			Path:        readPath,
			Frontmatter: meta,
		})
	}

	jsonOut(map[string]interface{}{
		"status": "ok",
		"count":  len(results),
		"skills": results,
	})
}

// learnGet reads one skill and returns its frontmatter + body.
// Usage: learn get <name>
func learnGet(args []string) {
	if len(args) < 1 {
		errOut("usage: vault-cli learn get <name>")
	}

	name := args[0]
	vault := mustVaultPath()
	relPath := filepath.Join("skills", name+".md")

	content, err := readNote(vault, relPath)
	if err != nil {
		errOut(fmt.Sprintf("cannot read skill: %v", err))
	}

	meta, body := parseFrontmatter(content)

	jsonOut(map[string]interface{}{
		"status":      "ok",
		"name":        name,
		"path":        relPath,
		"frontmatter": meta,
		"body":        body,
	})
}
