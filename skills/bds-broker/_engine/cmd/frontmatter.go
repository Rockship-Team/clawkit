package main

import (
	"strings"
)

// parseFrontmatter extracts key-value pairs from YAML frontmatter (--- block).
func parseFrontmatter(content string) map[string]string {
	out := map[string]string{}
	if !strings.HasPrefix(content, "---") {
		return out
	}
	rest := content[3:]
	// skip newline after ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return out
	}
	block := rest[:end]
	for _, line := range strings.Split(block, "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return out
}

// setFrontmatter replaces or adds a key in YAML frontmatter.
func setFrontmatter(content, key, value string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	rest := content[3:]
	nl := ""
	if len(rest) > 0 && rest[0] == '\n' {
		nl = "\n"
		rest = rest[1:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return content
	}
	block := rest[:end]
	after := rest[end:]

	lines := strings.Split(block, "\n")
	found := false
	for i, line := range lines {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		if k == key {
			lines[i] = key + ": " + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, key+": "+value)
	}
	return "---" + nl + strings.Join(lines, "\n") + after
}
