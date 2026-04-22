package main

import (
	"sort"
	"strings"
)

// parseFrontmatter extracts YAML frontmatter delimited by "---\n" lines.
// Returns (metadata map, body). Returns nil meta if no frontmatter found.
// Strips surrounding double quotes from values.
func parseFrontmatter(content string) (map[string]string, string) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content
	}

	rest := content[4:] // skip opening "---\n"

	// Handle empty frontmatter: "---\n---\n" or "---\n---" at EOF
	var frontmatterBlock string
	var body string

	if strings.HasPrefix(rest, "---\n") {
		// Empty frontmatter block
		frontmatterBlock = ""
		body = rest[4:]
	} else if rest == "---" {
		// Empty frontmatter at EOF with no trailing newline
		frontmatterBlock = ""
		body = ""
	} else {
		endIdx := strings.Index(rest, "\n---\n")
		if endIdx < 0 {
			// Check for frontmatter that ends at EOF (no trailing newline after ---)
			if strings.HasSuffix(rest, "\n---") {
				endIdx = len(rest) - 4 // length of "\n---"
				frontmatterBlock = rest[:endIdx]
				body = ""
			} else {
				return nil, content
			}
		} else {
			frontmatterBlock = rest[:endIdx]
			body = rest[endIdx+5:] // skip "\n---\n"
		}
	}

	meta := make(map[string]string)
	lines := strings.Split(frontmatterBlock, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])
		// Strip surrounding double quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			meta[key] = val
		}
	}

	if len(meta) == 0 {
		return meta, body
	}

	return meta, body
}

// buildNote constructs a markdown note with sorted frontmatter keys.
func buildNote(meta map[string]string, body string) string {
	if len(meta) == 0 {
		return body
	}

	var b strings.Builder
	b.WriteString("---\n")

	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(meta[k])
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString(body)

	return b.String()
}

// updateFrontmatterField modifies one field in the frontmatter, preserving others.
// If no frontmatter exists, one is created. If the key doesn't exist, it is added.
func updateFrontmatterField(content, key, value string) string {
	meta, body := parseFrontmatter(content)
	if meta == nil {
		meta = make(map[string]string)
	}
	meta[key] = value
	return buildNote(meta, body)
}
