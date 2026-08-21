# Self-Improving Agent with Obsidian Knowledge Vault — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate Hermes Agent self-improvement patterns (skill creation, persistent memory, session search) with Obsidian as a knowledge layer into the clawkit skill ecosystem, enabling OpenClaw agents that learn from interactions and organize knowledge in a human-readable vault.

**Architecture:** Three layers — (1) a Go CLI `vault-cli` that reads/writes Obsidian vaults as the knowledge data layer, (2) two new clawkit skills (`knowledge-vault` for knowledge management, `agent-learner` for self-improvement loops), and (3) a set of OpenClaw workspace template files (AGENTS.md, bootstrap) that inject learning nudges and memory protocols into the agent's system prompt. All persistent state lives in markdown files (Obsidian-compatible) and SQLite (session search). No external dependencies beyond Go stdlib + modernc.org/sqlite.

**Tech Stack:** Go 1.22 (CLI), SQLite via modernc.org/sqlite (sessions), Obsidian vault (markdown files on disk), OpenClaw workspace templates (markdown)

---

## Where Changes Go: clawkit vs OpenClaw

This plan touches **both** systems. Here's the boundary:

| Change                               | Where                                     | Why                                                     |
| ------------------------------------ | ----------------------------------------- | ------------------------------------------------------- |
| `vault-cli` Go binary                | clawkit (`skills/tools/vault-cli/`)       | Shared CLI for reading/writing Obsidian vaults          |
| `knowledge-vault` skill              | clawkit (`skills/tools/knowledge-vault/`) | Skill that organizes knowledge via vault-cli            |
| `agent-learner` skill                | clawkit (`skills/tools/agent-learner/`)   | Skill that implements self-improvement loop             |
| Session persistence (SQLite + FTS5)  | clawkit (`vault-cli session` commands)    | Store/search past conversations                         |
| Learning nudge templates             | clawkit (`templates/bootstrap-files/`)    | AGENTS.md additions for memory + learning prompts       |
| Memory protocol (MEMORY.md, USER.md) | OpenClaw workspace convention             | Files that OpenClaw reads at session start              |
| Turn-counter nudge system            | OpenClaw runtime change (future)          | Periodic injection of "save what you learned" reminders |
| Multi-skill loading                  | OpenClaw runtime change (future)          | Allow knowledge-vault alongside domain skills           |

**This plan implements everything in clawkit.** OpenClaw runtime changes are noted as future work — the skills will work today within OpenClaw's existing single-skill model, and will unlock further capabilities when OpenClaw adds turn counters and multi-skill support.

---

## File Structure

### New files to create

```
skills/tools/vault-cli/
  cmd/
    go.mod                    # Module: vault-cli, deps: modernc.org/sqlite
    main.go                   # CLI entry: note, search, memory, session, learn
    store.go                  # Vault path discovery, file I/O, helpers
    note.go                   # note add|get|list|search|link|tag
    memory.go                 # memory get|set|replace|remove|show
    session.go                # session save|search|list|summarize
    learn.go                  # learn save-skill|patch-skill|list-skills
    frontmatter.go            # YAML frontmatter parser (hand-rolled, no deps)
    search.go                 # FTS5 index builder + BM25 search
  config/
    vault.example.json        # Example vault configuration

skills/tools/knowledge-vault/
  SKILL.md                    # AI prompt for knowledge management
  config.json                 # Clawkit metadata

skills/tools/agent-learner/
  SKILL.md                    # AI prompt for self-improvement loop
  config.json                 # Clawkit metadata

templates/bootstrap-files/
  AGENTS.md                   # Updated with memory + learning protocols (modify existing)
```

### Existing files to modify

```
skills/skills.go:21           # Add `all:tools` to //go:embed if not present
```

---

## Task 1: Frontmatter Parser

**Files:**

- Create: `skills/tools/vault-cli/cmd/frontmatter.go`
- Create: `skills/tools/vault-cli/cmd/frontmatter_test.go`

This is a hand-rolled YAML frontmatter parser consistent with clawkit's zero-dependency approach.

- [ ] **Step 1: Write the failing test**

```go
// skills/tools/vault-cli/cmd/frontmatter_test.go
package main

import (
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMeta map[string]string
		wantBody string
	}{
		{
			name:     "basic frontmatter",
			input:    "---\ntitle: Hello\ntags: project\n---\n\nBody text here.",
			wantMeta: map[string]string{"title": "Hello", "tags": "project"},
			wantBody: "Body text here.",
		},
		{
			name:     "no frontmatter",
			input:    "Just body text.",
			wantMeta: nil,
			wantBody: "Just body text.",
		},
		{
			name:     "empty frontmatter",
			input:    "---\n---\n\nBody.",
			wantMeta: map[string]string{},
			wantBody: "Body.",
		},
		{
			name:     "quoted values",
			input:    "---\ntitle: \"Hello World\"\nstatus: active\n---\n\nContent.",
			wantMeta: map[string]string{"title": "Hello World", "status": "active"},
			wantBody: "Content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, body := parseFrontmatter(tt.input)
			if tt.wantMeta == nil && meta != nil {
				t.Errorf("expected nil meta, got %v", meta)
			}
			if tt.wantMeta != nil {
				for k, v := range tt.wantMeta {
					if meta[k] != v {
						t.Errorf("meta[%s] = %q, want %q", k, meta[k], v)
					}
				}
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestBuildFrontmatter(t *testing.T) {
	meta := map[string]string{"title": "Test", "status": "active"}
	body := "Some content."
	result := buildNote(meta, body)
	// Should contain ---\n...---\n\nSome content.
	if len(result) == 0 {
		t.Error("empty result")
	}
	// Round-trip
	rmeta, rbody := parseFrontmatter(result)
	if rmeta["title"] != "Test" {
		t.Errorf("round-trip title = %q", rmeta["title"])
	}
	if rbody != body {
		t.Errorf("round-trip body = %q", rbody)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skills/tools/vault-cli/cmd && go test -run TestParseFrontmatter -v`
Expected: FAIL — `parseFrontmatter` not defined

- [ ] **Step 3: Write the implementation**

```go
// skills/tools/vault-cli/cmd/frontmatter.go
package main

import (
	"fmt"
	"sort"
	"strings"
)

// parseFrontmatter extracts YAML frontmatter and body from a markdown string.
// Returns nil meta if no frontmatter found.
func parseFrontmatter(content string) (map[string]string, string) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content
	}
	end := strings.Index(content[4:], "\n---\n")
	if end == -1 {
		// Check for ---\n at very end
		if strings.HasSuffix(content[4:], "\n---") {
			end = len(content[4:]) - 4
		} else {
			return nil, content
		}
	}

	yamlBlock := content[4 : 4+end]
	bodyStart := 4 + end + 5 // skip \n---\n
	if bodyStart > len(content) {
		bodyStart = len(content)
	}
	body := content[bodyStart:]
	// Trim leading newline from body
	body = strings.TrimPrefix(body, "\n")

	meta := map[string]string{}
	for _, line := range strings.Split(yamlBlock, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		meta[key] = val
	}

	return meta, body
}

// buildNote constructs a markdown note with frontmatter.
func buildNote(meta map[string]string, body string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	// Sort keys for deterministic output
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s: %s\n", k, meta[k])
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body)
	return sb.String()
}

// updateFrontmatterField modifies one field in existing frontmatter, preserving others.
func updateFrontmatterField(content, key, value string) string {
	meta, body := parseFrontmatter(content)
	if meta == nil {
		meta = map[string]string{}
	}
	meta[key] = value
	return buildNote(meta, body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skills/tools/vault-cli/cmd && go test -run TestParse -v && go test -run TestBuild -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skills/tools/vault-cli/cmd/frontmatter.go skills/tools/vault-cli/cmd/frontmatter_test.go
git commit -m "feat: add hand-rolled frontmatter parser for vault-cli"
```

---

## Task 2: Vault Store Layer (path discovery, file I/O)

**Files:**

- Create: `skills/tools/vault-cli/cmd/go.mod`
- Create: `skills/tools/vault-cli/cmd/main.go`
- Create: `skills/tools/vault-cli/cmd/store.go`

- [ ] **Step 1: Create go.mod**

```go
// skills/tools/vault-cli/cmd/go.mod
module vault-cli

go 1.22

require modernc.org/sqlite v1.34.5
```

- [ ] **Step 2: Write store.go with vault path discovery and file helpers**

```go
// skills/tools/vault-cli/cmd/store.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// vaultConfig holds the user's vault configuration.
type vaultConfig struct {
	VaultPath string `json:"vault_path"` // absolute path to Obsidian vault
	DBPath    string `json:"db_path"`    // SQLite for session search
}

func loadVaultConfig() vaultConfig {
	var cfg vaultConfig
	// Try skill dir first
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".openclaw", "workspace", "skills", "knowledge-vault", "vault-config.json"),
		filepath.Join(home, ".openclaw", "workspace", "sme-data", "vault-config.json"),
		"vault-config.json",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			json.Unmarshal(data, &cfg)
			break
		}
	}
	// Default vault path
	if cfg.VaultPath == "" {
		cfg.VaultPath = filepath.Join(home, "ObsidianVault")
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(home, ".openclaw", "workspace", "sme-data", "sessions.db")
	}
	return cfg
}

func mustVaultPath() string {
	cfg := loadVaultConfig()
	if _, err := os.Stat(cfg.VaultPath); os.IsNotExist(err) {
		os.MkdirAll(cfg.VaultPath, 0o755)
	}
	return cfg.VaultPath
}

func openSessionDB() (*sql.DB, error) {
	if db != nil {
		return db, nil
	}
	cfg := loadVaultConfig()
	os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755)
	var err error
	db, err = sql.Open("sqlite", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	// Create session tables
	db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY, title TEXT, skill_name TEXT,
		message_count INTEGER DEFAULT 0, created_at TEXT, ended_at TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL,
		role TEXT NOT NULL, content TEXT NOT NULL, created_at TEXT
	)`)
	db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		content, content=messages, content_rowid=id
	)`)
	// Triggers for FTS sync
	db.Exec(`CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
	END`)
	db.Exec(`CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', old.id, old.content);
	END`)
	return db, nil
}

// --- Output helpers ---

func jsonOut(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func okOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	jsonOut(out)
}

func errOut(msg string) {
	jsonOut(map[string]interface{}{"ok": false, "error": msg})
	os.Exit(1)
}

// --- Vault file operations ---

func readNote(notePath string) (string, error) {
	vault := mustVaultPath()
	full := filepath.Join(vault, notePath)
	if !strings.HasSuffix(full, ".md") {
		full += ".md"
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeNote(notePath string, content string) error {
	vault := mustVaultPath()
	full := filepath.Join(vault, notePath)
	if !strings.HasSuffix(full, ".md") {
		full += ".md"
	}
	os.MkdirAll(filepath.Dir(full), 0o755)
	return os.WriteFile(full, []byte(content), 0o644)
}

func listNotes(dir string) ([]string, error) {
	vault := mustVaultPath()
	root := filepath.Join(vault, dir)
	var notes []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".obsidian" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			rel, _ := filepath.Rel(vault, path)
			notes = append(notes, rel)
		}
		return nil
	})
	return notes, nil
}

// --- Wikilink and tag extraction ---

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
var tagRe = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9/_-]+)`)

func extractLinks(body string) []string {
	matches := wikiLinkRe.FindAllStringSubmatch(body, -1)
	links := make([]string, 0, len(matches))
	for _, m := range matches {
		links = append(links, m[1])
	}
	return links
}

func extractTags(body string) []string {
	matches := tagRe.FindAllStringSubmatch(body, -1)
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

// --- Time helpers ---

func vnNow() time.Time {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	if loc == nil {
		return time.Now().UTC().Add(7 * time.Hour)
	}
	return time.Now().In(loc)
}

func vnNowISO() string { return vnNow().Format("2006-01-02T15:04:05+07:00") }
func vnToday() string  { return vnNow().Format("2006-01-02") }

func newID() string {
	b := make([]byte, 16)
	n := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(n >> (i * 4))
		n = n*6364136223846793005 + 1442695040888963407
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
```

- [ ] **Step 3: Write main.go dispatcher**

```go
// skills/tools/vault-cli/cmd/main.go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
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
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `vault-cli — Obsidian knowledge vault for OpenClaw agents

  note     add|get|list|search|append    Read/write notes in the vault
  memory   get|set|replace|remove|show   Persistent agent memory (MEMORY.md)
  session  save|search|list              Conversation session persistence
  learn    save-skill|patch-skill|list   Self-improvement skill creation
  search   <query>                       Full-text search across vault + sessions`)
}
```

- [ ] **Step 4: Run `go mod tidy` and verify compilation**

Run: `cd skills/tools/vault-cli/cmd && go mod tidy && go build -o ../vault-cli .`
Expected: Compiles (will fail on undefined cmdNote etc. — add stubs next)

- [ ] **Step 5: Commit**

```bash
git add skills/tools/vault-cli/cmd/
git commit -m "feat: vault-cli store layer — vault path discovery, file I/O, session DB"
```

---

## Task 3: Note Commands (Obsidian CRUD)

**Files:**

- Create: `skills/tools/vault-cli/cmd/note.go`

- [ ] **Step 1: Write note.go**

```go
// skills/tools/vault-cli/cmd/note.go
package main

import (
	"os"
	"path/filepath"
	"strings"
)

func cmdNote(args []string) {
	if len(args) == 0 {
		errOut("usage: note add|get|list|search|append")
	}
	switch args[0] {
	case "add":
		// note add <path> <body> [key=value frontmatter pairs...]
		if len(args) < 3 {
			errOut("usage: note add <path> <body> [title=X] [tags=Y] [status=Z]")
		}
		notePath := args[1]
		body := args[2]
		meta := map[string]string{"created": vnToday()}
		for _, arg := range args[3:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				meta[parts[0]] = parts[1]
			}
		}
		content := buildNote(meta, body)
		if err := writeNote(notePath, content); err != nil {
			errOut("write failed: " + err.Error())
		}
		okOut(map[string]interface{}{"path": notePath, "links": extractLinks(body), "tags": extractTags(body)})

	case "get":
		if len(args) < 2 {
			errOut("usage: note get <path>")
		}
		content, err := readNote(args[1])
		if err != nil {
			errOut("read failed: " + err.Error())
		}
		meta, body := parseFrontmatter(content)
		okOut(map[string]interface{}{
			"path":       args[1],
			"frontmatter": meta,
			"body":       body,
			"links":      extractLinks(body),
			"tags":       extractTags(body),
		})

	case "list":
		dir := ""
		if len(args) > 1 {
			dir = args[1]
		}
		notes, err := listNotes(dir)
		if err != nil {
			errOut("list failed: " + err.Error())
		}
		// Include frontmatter summary for each note
		type noteInfo struct {
			Path  string            `json:"path"`
			Meta  map[string]string `json:"frontmatter,omitempty"`
		}
		var infos []noteInfo
		for _, n := range notes {
			content, _ := readNote(n)
			meta, _ := parseFrontmatter(content)
			infos = append(infos, noteInfo{n, meta})
		}
		okOut(map[string]interface{}{"notes": infos, "count": len(infos)})

	case "search":
		if len(args) < 2 {
			errOut("usage: note search <query>")
		}
		query := strings.ToLower(strings.Join(args[1:], " "))
		vault := mustVaultPath()
		type match struct {
			Path    string `json:"path"`
			Snippet string `json:"snippet"`
		}
		var matches []match
		filepath.WalkDir(vault, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				if d != nil && d.IsDir() && d.Name() == ".obsidian" {
					return filepath.SkipDir
				}
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)
			lower := strings.ToLower(content)
			idx := strings.Index(lower, query)
			if idx == -1 {
				return nil
			}
			// Extract snippet around match
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + len(query) + 100
			if end > len(content) {
				end = len(content)
			}
			rel, _ := filepath.Rel(vault, path)
			matches = append(matches, match{rel, strings.TrimSpace(content[start:end])})
			return nil
		})
		okOut(map[string]interface{}{"results": matches, "count": len(matches), "query": query})

	case "append":
		if len(args) < 3 {
			errOut("usage: note append <path> <text>")
		}
		existing, err := readNote(args[1])
		if err != nil {
			// Create new note if doesn't exist
			writeNote(args[1], args[2])
			okOut(map[string]interface{}{"path": args[1], "action": "created"})
			return
		}
		content := existing + "\n\n" + args[2]
		writeNote(args[1], content)
		okOut(map[string]interface{}{"path": args[1], "action": "appended"})

	default:
		errOut("unknown note command: " + args[0])
	}
}
```

- [ ] **Step 2: Compile and test manually**

Run:

```bash
cd skills/tools/vault-cli/cmd && go build -o ../vault-cli .
../vault-cli note add "test/hello" "This is a test note with [[link]] and #tag" title="Hello World"
../vault-cli note get "test/hello"
../vault-cli note search "test note"
../vault-cli note list "test"
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/vault-cli/cmd/note.go
git commit -m "feat: vault-cli note commands — add, get, list, search, append"
```

---

## Task 4: Memory Commands (Hermes-style MEMORY.md + USER.md)

**Files:**

- Create: `skills/tools/vault-cli/cmd/memory.go`

This implements the Hermes Agent pattern: capped persistent memory with `§`-delimited entries, deduplication, and consolidation pressure.

- [ ] **Step 1: Write memory.go**

```go
// skills/tools/vault-cli/cmd/memory.go
package main

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	memoryCap = 2200  // chars — forces consolidation
	userCap   = 1375  // chars
	separator = "\n§\n"
)

func memoryPath(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", name)
}

func loadMemory(name string) ([]string, string) {
	data, err := os.ReadFile(memoryPath(name))
	if err != nil {
		return nil, ""
	}
	raw := string(data)
	if raw == "" {
		return nil, ""
	}
	entries := strings.Split(raw, separator)
	// Deduplicate
	seen := map[string]bool{}
	var unique []string
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		unique = append(unique, e)
	}
	return unique, raw
}

func saveMemory(name string, entries []string) error {
	content := strings.Join(entries, separator)
	os.MkdirAll(filepath.Dir(memoryPath(name)), 0o755)
	return os.WriteFile(memoryPath(name), []byte(content), 0o644)
}

func memoryCharCount(entries []string) int {
	total := 0
	for _, e := range entries {
		total += len(e)
	}
	total += len(separator) * (len(entries) - 1)
	if total < 0 {
		total = 0
	}
	return total
}

func cmdMemory(args []string) {
	if len(args) == 0 {
		errOut("usage: memory get|set|replace|remove|show")
	}

	switch args[0] {
	case "show":
		// Show both MEMORY.md and USER.md
		memEntries, _ := loadMemory("MEMORY.md")
		userEntries, _ := loadMemory("USER.md")
		okOut(map[string]interface{}{
			"memory":      memEntries,
			"memory_chars": memoryCharCount(memEntries),
			"memory_cap":  memoryCap,
			"user":        userEntries,
			"user_chars":  memoryCharCount(userEntries),
			"user_cap":    userCap,
		})

	case "get":
		// memory get <MEMORY.md|USER.md>
		if len(args) < 2 {
			errOut("usage: memory get <MEMORY.md|USER.md>")
		}
		entries, _ := loadMemory(args[1])
		cap := memoryCap
		if args[1] == "USER.md" {
			cap = userCap
		}
		okOut(map[string]interface{}{
			"file":    args[1],
			"entries": entries,
			"chars":   memoryCharCount(entries),
			"cap":     cap,
		})

	case "set":
		// memory set <MEMORY.md|USER.md> <entry>
		if len(args) < 3 {
			errOut("usage: memory set <MEMORY.md|USER.md> <entry>")
		}
		file, entry := args[1], args[2]
		cap := memoryCap
		if file == "USER.md" {
			cap = userCap
		}

		entries, _ := loadMemory(file)

		// Duplicate check
		for _, e := range entries {
			if e == entry {
				errOut("duplicate entry — already exists")
			}
		}

		entries = append(entries, entry)
		newSize := memoryCharCount(entries)
		if newSize > cap {
			errOut("memory full (" + string(rune('0'+len(entries))) + " entries, " +
				strings.Repeat("0", 0) + " chars). Use 'memory replace' or 'memory remove' to make room.")
		}

		saveMemory(file, entries)
		okOut(map[string]interface{}{
			"file":  file,
			"added": entry,
			"chars": newSize,
			"cap":   cap,
		})

	case "replace":
		// memory replace <MEMORY.md|USER.md> <old_entry_substring> <new_entry>
		if len(args) < 4 {
			errOut("usage: memory replace <file> <old_substring> <new_entry>")
		}
		file, oldSub, newEntry := args[1], args[2], args[3]
		entries, _ := loadMemory(file)
		found := false
		for i, e := range entries {
			if strings.Contains(e, oldSub) {
				entries[i] = newEntry
				found = true
				break
			}
		}
		if !found {
			errOut("no entry matching: " + oldSub)
		}
		saveMemory(file, entries)
		okOut(map[string]interface{}{"file": file, "replaced": true})

	case "remove":
		// memory remove <MEMORY.md|USER.md> <entry_substring>
		if len(args) < 3 {
			errOut("usage: memory remove <file> <entry_substring>")
		}
		file, sub := args[1], args[2]
		entries, _ := loadMemory(file)
		var kept []string
		removed := false
		for _, e := range entries {
			if !removed && strings.Contains(e, sub) {
				removed = true
				continue
			}
			kept = append(kept, e)
		}
		if !removed {
			errOut("no entry matching: " + sub)
		}
		saveMemory(file, kept)
		okOut(map[string]interface{}{"file": file, "removed": true, "remaining": len(kept)})

	default:
		errOut("unknown memory command: " + args[0])
	}
}
```

- [ ] **Step 2: Compile and test**

Run:

```bash
go build -o ../vault-cli . && ../vault-cli memory set MEMORY.md "User prefers VND amounts with comma separators"
../vault-cli memory set MEMORY.md "Company tax code is 0123456789"
../vault-cli memory show
../vault-cli memory replace MEMORY.md "tax code" "Company tax code is 9876543210 (updated April 2026)"
../vault-cli memory show
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/vault-cli/cmd/memory.go
git commit -m "feat: vault-cli memory commands — capped persistent memory with dedup"
```

---

## Task 5: Session Commands (FTS5 search)

**Files:**

- Create: `skills/tools/vault-cli/cmd/session.go`

- [ ] **Step 1: Write session.go**

```go
// skills/tools/vault-cli/cmd/session.go
package main

import (
	"strings"
)

func cmdSession(args []string) {
	if len(args) == 0 {
		errOut("usage: session save|search|list")
	}
	db, err := openSessionDB()
	if err != nil {
		errOut("open session db: " + err.Error())
	}

	switch args[0] {
	case "save":
		// session save <session_id> <title> <skill_name> <role> <content>
		if len(args) < 6 {
			errOut("usage: session save <session_id> <title> <skill> <role> <content>")
		}
		sid, title, skill, role, content := args[1], args[2], args[3], args[4], args[5]

		// Upsert session
		db.Exec(`INSERT INTO sessions (id, title, skill_name, message_count, created_at)
			VALUES (?, ?, ?, 0, ?)
			ON CONFLICT(id) DO UPDATE SET title=?, message_count=message_count+1`,
			sid, title, skill, vnNowISO(), title)

		// Insert message
		db.Exec("INSERT INTO messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)",
			sid, role, content, vnNowISO())

		// Update count
		db.Exec("UPDATE sessions SET message_count = (SELECT COUNT(*) FROM messages WHERE session_id = ?) WHERE id = ?", sid, sid)

		okOut(map[string]interface{}{"session_id": sid, "role": role, "saved": true})

	case "search":
		// session search <query> [limit]
		if len(args) < 2 {
			errOut("usage: session search <query> [limit]")
		}
		query := args[1]
		limit := "10"
		if len(args) > 2 {
			limit = args[2]
		}

		rows, err := db.Query(`
			SELECT m.session_id, s.title, s.skill_name, m.role, snippet(messages_fts, 0, '>>>', '<<<', '...', 32) as snippet, m.created_at
			FROM messages_fts mf
			JOIN messages m ON m.id = mf.rowid
			JOIN sessions s ON s.id = m.session_id
			WHERE messages_fts MATCH ?
			ORDER BY rank
			LIMIT `+limit, query)
		if err != nil {
			errOut("search failed: " + err.Error())
		}
		defer rows.Close()

		type result struct {
			SessionID string `json:"session_id"`
			Title     string `json:"title"`
			Skill     string `json:"skill"`
			Role      string `json:"role"`
			Snippet   string `json:"snippet"`
			At        string `json:"at"`
		}
		var results []result
		for rows.Next() {
			var r result
			rows.Scan(&r.SessionID, &r.Title, &r.Skill, &r.Role, &r.Snippet, &r.At)
			results = append(results, r)
		}
		okOut(map[string]interface{}{"results": results, "count": len(results), "query": query})

	case "list":
		limit := "20"
		if len(args) > 1 {
			limit = args[1]
		}
		rows, err := db.Query("SELECT id, title, skill_name, message_count, created_at FROM sessions ORDER BY created_at DESC LIMIT " + limit)
		if err != nil {
			errOut("list failed: " + err.Error())
		}
		defer rows.Close()

		type sess struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Skill    string `json:"skill"`
			Messages int    `json:"messages"`
			At       string `json:"at"`
		}
		var sessions []sess
		for rows.Next() {
			var s sess
			rows.Scan(&s.ID, &s.Title, &s.Skill, &s.Messages, &s.At)
			sessions = append(sessions, s)
		}
		okOut(map[string]interface{}{"sessions": sessions, "count": len(sessions)})

	default:
		errOut("unknown session command: " + args[0])
	}
}

func cmdSearch(args []string) {
	if len(args) == 0 {
		errOut("usage: search <query>")
	}
	query := strings.Join(args, " ")

	// Search both vault notes AND session messages
	type combinedResult struct {
		Source  string `json:"source"` // "vault" or "session"
		Path    string `json:"path,omitempty"`
		Session string `json:"session_id,omitempty"`
		Snippet string `json:"snippet"`
	}
	var results []combinedResult

	// 1. Vault search
	vault := mustVaultPath()
	lower := strings.ToLower(query)
	vaultWalk(vault, func(path, content string) {
		if idx := strings.Index(strings.ToLower(content), lower); idx >= 0 {
			start := idx - 40
			if start < 0 { start = 0 }
			end := idx + len(query) + 80
			if end > len(content) { end = len(content) }
			rel, _ := filepath.Rel(vault, path)
			results = append(results, combinedResult{Source: "vault", Path: rel, Snippet: strings.TrimSpace(content[start:end])})
		}
	})

	// 2. Session search
	db, err := openSessionDB()
	if err == nil {
		rows, err := db.Query(`
			SELECT m.session_id, snippet(messages_fts, 0, '>>>', '<<<', '...', 24)
			FROM messages_fts mf JOIN messages m ON m.id = mf.rowid
			WHERE messages_fts MATCH ? LIMIT 10`, query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var sid, snip string
				rows.Scan(&sid, &snip)
				results = append(results, combinedResult{Source: "session", Session: sid, Snippet: snip})
			}
		}
	}

	okOut(map[string]interface{}{"results": results, "count": len(results), "query": query})
}

func vaultWalk(vault string, fn func(path, content string)) {
	filepath.WalkDir(vault, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && d.Name() == ".obsidian" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, _ := os.ReadFile(path)
		fn(path, string(data))
		return nil
	})
}
```

- [ ] **Step 2: Compile and test**

Run:

```bash
go build -o ../vault-cli .
../vault-cli session save "s1" "Test PIT calculation" "sme-tax" "user" "Tinh thue cho luong 35 trieu"
../vault-cli session save "s1" "Test PIT calculation" "sme-tax" "assistant" "PIT = 1,638,750 VND"
../vault-cli session search "thue"
../vault-cli session list
../vault-cli search "thue"
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/vault-cli/cmd/session.go
git commit -m "feat: vault-cli session commands — FTS5 search across conversations"
```

---

## Task 6: Learn Commands (Self-Improving Skill Loop)

**Files:**

- Create: `skills/tools/vault-cli/cmd/learn.go`

This implements the Hermes Agent pattern: save successful task approaches as skill files, patch them when gaps are found.

- [ ] **Step 1: Write learn.go**

```go
// skills/tools/vault-cli/cmd/learn.go
package main

import (
	"os"
	"path/filepath"
	"strings"
)

func learnDir() string {
	return filepath.Join(mustVaultPath(), "skills")
}

func cmdLearn(args []string) {
	if len(args) == 0 {
		errOut("usage: learn save-skill|patch-skill|list|get")
	}

	switch args[0] {
	case "save-skill":
		// learn save-skill <name> <description> <procedure_body> [tags]
		if len(args) < 4 {
			errOut("usage: learn save-skill <name> <description> <procedure>")
		}
		name, desc, body := args[1], args[2], args[3]
		tags := ""
		if len(args) > 4 {
			tags = args[4]
		}

		dir := learnDir()
		os.MkdirAll(dir, 0o755)
		skillPath := filepath.Join(dir, name+".md")

		meta := map[string]string{
			"name":        name,
			"description": desc,
			"created":     vnToday(),
			"updated":     vnToday(),
		}
		if tags != "" {
			meta["tags"] = tags
		}

		content := buildNote(meta, body)
		if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
			errOut("save failed: " + err.Error())
		}
		okOut(map[string]interface{}{"name": name, "path": "skills/" + name + ".md", "action": "created"})

	case "patch-skill":
		// learn patch-skill <name> <find_text> <replace_text>
		if len(args) < 4 {
			errOut("usage: learn patch-skill <name> <find> <replace>")
		}
		name, find, replace := args[1], args[2], args[3]
		skillPath := filepath.Join(learnDir(), name+".md")

		data, err := os.ReadFile(skillPath)
		if err != nil {
			errOut("skill not found: " + name)
		}

		content := string(data)
		if !strings.Contains(content, find) {
			// Fuzzy match: try trimming whitespace differences
			normalized := strings.Join(strings.Fields(content), " ")
			normalizedFind := strings.Join(strings.Fields(find), " ")
			if !strings.Contains(normalized, normalizedFind) {
				errOut("find text not found in skill: " + name)
			}
			// For fuzzy match, replace in original with whitespace-aware approach
			content = strings.Replace(content, find, replace, 1)
		} else {
			content = strings.Replace(content, find, replace, 1)
		}

		// Update the "updated" frontmatter field
		content = updateFrontmatterField(content, "updated", vnToday())

		os.WriteFile(skillPath, []byte(content), 0o644)
		okOut(map[string]interface{}{"name": name, "action": "patched"})

	case "list":
		dir := learnDir()
		var skills []map[string]string
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			data, _ := os.ReadFile(path)
			meta, _ := parseFrontmatter(string(data))
			if meta == nil {
				meta = map[string]string{}
			}
			meta["file"] = d.Name()
			skills = append(skills, meta)
			return nil
		})
		okOut(map[string]interface{}{"skills": skills, "count": len(skills)})

	case "get":
		if len(args) < 2 {
			errOut("usage: learn get <name>")
		}
		skillPath := filepath.Join(learnDir(), args[1]+".md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			errOut("skill not found: " + args[1])
		}
		meta, body := parseFrontmatter(string(data))
		okOut(map[string]interface{}{"name": args[1], "frontmatter": meta, "body": body})

	default:
		errOut("unknown learn command: " + args[0])
	}
}
```

- [ ] **Step 2: Compile and test**

Run:

```bash
go build -o ../vault-cli .
../vault-cli learn save-skill "pit-calculation" "How to calculate Vietnamese PIT step by step" "1. Get gross salary\n2. Calculate BHXH (8%), BHYT (1.5%), BHTN (1%)\n3. Subtract self deduction (11M) and dependents (4.4M each)\n4. Apply 7-bracket progressive tax\n5. Net = Gross - Insurance - PIT" "tax,payroll"
../vault-cli learn list
../vault-cli learn get "pit-calculation"
../vault-cli learn patch-skill "pit-calculation" "self deduction (11M)" "self deduction (11M per month)"
../vault-cli learn get "pit-calculation"
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/vault-cli/cmd/learn.go
git commit -m "feat: vault-cli learn commands — self-improving skill creation and patching"
```

---

## Task 7: Knowledge Vault Skill (SKILL.md + config.json)

**Files:**

- Create: `skills/tools/knowledge-vault/SKILL.md`
- Create: `skills/tools/knowledge-vault/config.json`

- [ ] **Step 1: Create SKILL.md**

```markdown
---
name: knowledge-vault
description: "Quan ly kien thuc doanh nghiep trong Obsidian vault — ghi chep, tim kiem, lien ket, to chuc thong tin."
metadata: { "openclaw": { "emoji": "🧠" } }
---

# Tro ly quan ly kien thuc — Knowledge Vault

Ban la tro ly quan ly kien thuc. Ban giup nguoi dung ghi chep, to chuc, va tim kiem thong tin
trong kho kien thuc (Obsidian vault). Moi ghi chep la 1 file markdown voi metadata (frontmatter)
va noi dung co lien ket (wikilinks).

## QUY TAC

- Moi thao tac goi qua `vault-cli <command> <args...>` tren 1 dong duy nhat.
- PHAI doc output va kiem tra "ok":true truoc khi bao ket qua.
- KHONG bia noi dung. Chi luu nhung gi user cung cap.
- Khi tao ghi chep, LUON them frontmatter (title, tags, created date).
- Lien ket giua cac ghi chep bang [[wikilink]]. Goi y lien ket khi phu hop.

## CONG CU

### Ghi chep (Note)

sme-cli note add <duong_dan> <noi_dung> [title=X] [tags=Y] [status=Z]
vault-cli note get <duong_dan>
vault-cli note list [thu_muc]
vault-cli note search <tu_khoa>
vault-cli note append <duong_dan> <them_noi_dung>

### Bo nho (Memory)

vault-cli memory show
vault-cli memory set <MEMORY.md|USER.md> <noi_dung>
vault-cli memory replace <file> <tu_khoa_cu> <noi_dung_moi>
vault-cli memory remove <file> <tu_khoa>

### Tim kiem toan bo (Full-text search)

vault-cli search <tu_khoa>

### Phien lam viec (Session)

vault-cli session save <id> <tieu_de> <skill> <role> <noi_dung>
vault-cli session search <tu_khoa>
vault-cli session list

## HANH VI

**Khi user muon ghi chep:** Hoi tieu de, noi dung, tags. Tao ghi chep voi frontmatter day du.
Goi y lien ket [[]] neu co ghi chep lien quan.

**Khi user muon tim:** Goi `vault-cli search` hoac `vault-cli note search`. Trinh bay ket qua
voi snippet va duong dan.

**Khi user chia se thong tin ca nhan/cong ty:** Luu vao memory (MEMORY.md cho thong tin chung,
USER.md cho thong tin ca nhan). Giu duoi gioi han ky tu — neu day, goi y xoa/ghep muc cu.

## VI DU

User: "Ghi lai: MST cong ty la 0312345678, dia chi 123 Nguyen Hue Q1"
→ vault-cli memory set MEMORY.md "MST cong ty: 0312345678. Dia chi: 123 Nguyen Hue Q1 TPHCM"
→ "Da luu vao bo nho. Minh se nho thong tin nay cho cac lan sau."

User: "Tao ghi chep ve cuoc hop hom nay"
→ vault-cli note add "meetings/2026-04-17-standup" "Noi dung cuoc hop..." title="Standup 17/04" tags="meeting,daily"
→ "Da tao ghi chep meetings/2026-04-17-standup.md"

User: "Tim tat ca ghi chep ve thue"
→ vault-cli search "thue"
→ Trinh bay ket qua tu vault va session history
```

- [ ] **Step 2: Create config.json**

```json
{
  "version": "1.0.0",
  "requires_bins": ["vault-cli"],
  "setup_prompts": [
    {
      "key": "vault_path",
      "label": "Duong dan Obsidian vault",
      "placeholder": "~/ObsidianVault"
    }
  ],
  "exclude": ["cmd", "*.go"]
}
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/knowledge-vault/
git commit -m "feat: knowledge-vault skill — Obsidian-integrated knowledge management"
```

---

## Task 8: Agent Learner Skill (SKILL.md + config.json)

**Files:**

- Create: `skills/tools/agent-learner/SKILL.md`
- Create: `skills/tools/agent-learner/config.json`

- [ ] **Step 1: Create SKILL.md**

```markdown
---
name: agent-learner
description: "Tu hoc va cai thien — luu quy trinh thanh cong, cap nhat khi phat hien thieu sot, tim kiem kinh nghiem cu."
metadata: { "openclaw": { "emoji": "🔄" } }
---

# Tro ly tu hoc — Agent Learner

Ban la tro ly AI co kha nang TU HOC tu kinh nghiem. Sau moi nhiem vu phuc tap,
ban luu lai cach lam thanh cong. Khi gap van de tuong tu, ban tim lai kinh nghiem cu.
Khi phat hien quy trinh thieu buoc, ban tu cap nhat.

## NGUYEN TAC TU HOC

### Khi nao luu kinh nghiem moi

- Hoan thanh nhiem vu phuc tap (5+ buoc)
- Giai quyet loi kho
- Phat hien quy trinh moi hieu qua
- User khen hoac xac nhan cach lam dung

### Khi nao cap nhat kinh nghiem cu

- Load skill cu nhung thieu buoc → cap nhat TRUOC khi ket thuc
- User sua lai cach lam → ghi nhan thay doi
- Phat hien edge case moi → bo sung

### Khi nao tim kinh nghiem cu

- Truoc khi bat dau nhiem vu moi → kiem tra co skill lien quan
- Gap loi → tim session cu da giai quyet tuong tu
- User hoi "minh da lam gi truoc day" → search session history

## CONG CU

### Luu quy trinh moi

vault-cli learn save-skill <ten> <mo_ta> <quy_trinh> [tags]

### Cap nhat quy trinh

vault-cli learn patch-skill <ten> <doan_cu> <doan_moi>

### Xem danh sach kinh nghiem

vault-cli learn list
vault-cli learn get <ten>

### Tim kiem kinh nghiem

vault-cli session search <tu_khoa>
vault-cli search <tu_khoa>

### Luu bo nho

vault-cli memory set MEMORY.md <thong_tin>
vault-cli memory replace MEMORY.md <cu> <moi>

## QUY TRINH TU HOC

### Sau moi nhiem vu phuc tap:

1. Tu hoi: "Nhiem vu nay co gi dang hoc?"
2. Neu co → goi `vault-cli learn save-skill` voi ten, mo ta, va quy trinh chi tiet
3. Bao user: "Minh da luu cach lam nay de lan sau lam nhanh hon."

### Truoc moi nhiem vu moi:

1. Goi `vault-cli learn list` de xem co skill lien quan
2. Neu co → goi `vault-cli learn get <ten>` de doc quy trinh
3. Lam theo quy trinh, cap nhat neu thieu

### Khi gap loi:

1. Goi `vault-cli session search <tu_khoa_loi>` de tim kinh nghiem cu
2. Neu tim thay → ap dung giai phap cu
3. Neu khong → giai quyet va luu skill moi

## NHAC NHO (Nudge)

Moi 10 luot hoi thoai, tu hoi:

- "Co thong tin nao can luu vao bo nho?"
- "Co quy trinh nao can ghi lai?"
- "Bo nho co gi can cap nhat?"

## QUY TAC

- Chi luu thong tin THUC SU huu ich cho tuong lai
- KHONG luu thong tin nhay cam (mat khau, token, CCCD)
- KHONG luu toan bo hoi thoai — chi rut gon thanh quy trinh
- Khi bo nho day (>2200 ky tu), PHAI ghep hoac xoa muc cu truoc khi them moi
- Moi skill file PHAI co: ten, mo ta, quy trinh buoc-buoc, ngay tao

## VI DU

### Luu skill moi:

User: "Tinh luong cho 15 nhan vien xong roi"
Bot: [kiem tra — nhiem vu phuc tap, 10+ buoc]
→ vault-cli learn save-skill "payroll-monthly" "Quy trinh tinh bang luong hang thang" "1. Kiem tra NV active...\n2. Tinh BHXH/BHYT/BHTN...\n3. Tinh TNCN luy tien...\n4. Gui duyet..." "hr,payroll"
→ "Minh da luu quy trinh tinh luong. Lan sau se lam nhanh hon!"

### Tim kinh nghiem cu:

User: "Doi soat ngan hang lam sao?"
Bot: [kiem tra skills]
→ vault-cli learn list → tim thay "bank-reconciliation"
→ vault-cli learn get "bank-reconciliation"
→ "Minh da co kinh nghiem doi soat. Quy trinh: [buoc 1, 2, 3...]"

### Cap nhat skill:

Bot: [phat hien thieu buoc trong quy trinh payroll]
→ vault-cli learn patch-skill "payroll-monthly" "3. Tinh TNCN" "3. Kiem tra muc dong BHXH toi da (46.8tr)\n4. Tinh TNCN luy tien"
→ "Da cap nhat quy trinh payroll voi buoc kiem tra BHXH cap."
```

- [ ] **Step 2: Create config.json**

```json
{
  "version": "1.0.0",
  "requires_bins": ["vault-cli"],
  "setup_prompts": [],
  "exclude": ["cmd", "*.go"]
}
```

- [ ] **Step 3: Commit**

```bash
git add skills/tools/agent-learner/
git commit -m "feat: agent-learner skill — self-improving agent with skill creation and patching"
```

---

## Task 9: Update Workspace Templates with Learning Protocols

**Files:**

- Modify: `templates/bootstrap-files/AGENTS.md` (if exists, else create)

- [ ] **Step 1: Add memory + learning protocol to AGENTS.md**

Append these sections to the existing AGENTS.md template (or create if it doesn't exist):

```markdown
## Memory Protocol

You have persistent memory across sessions. At the start of each session:

1. Check `vault-cli memory show` for stored context
2. Check `vault-cli learn list` for saved procedures

### When to save memory:

- User shares company info (tax code, address, preferences) → `vault-cli memory set MEMORY.md "<info>"`
- User corrects you → `vault-cli memory set MEMORY.md "User prefers: <correction>"`
- You discover environment details → `vault-cli memory set MEMORY.md "<detail>"`

### When to save a skill:

- After completing a task with 5+ tool calls → `vault-cli learn save-skill`
- After fixing a tricky error → save the solution
- After discovering a non-obvious workflow → document it

### When to update a skill:

- If a loaded skill was missing steps → `vault-cli learn patch-skill` before finishing
- If user corrects a procedure → update the skill

### Memory limits:

- MEMORY.md: 2,200 characters max. Consolidate before adding.
- USER.md: 1,375 characters max. Only store user preferences.
- Prioritize what reduces future user corrections.
```

- [ ] **Step 2: Commit**

```bash
git add templates/bootstrap-files/AGENTS.md
git commit -m "feat: add memory and learning protocols to workspace templates"
```

---

## Task 10: Update skills.go Embed and Generate Registry

**Files:**

- Modify: `skills/skills.go:21`

- [ ] **Step 1: Ensure `all:tools` is in the embed directive**

Check if `skills/skills.go` line 21 includes `all:tools`. If not, add it:

```go
//go:embed all:ecommerce all:finance all:real-estate all:tools all:utilities
```

Note: The `sme` vertical may have been removed. Only add `all:tools` if `skills/tools/` directory exists with the new vault-cli and skills.

- [ ] **Step 2: Run make generate**

```bash
make generate
```

Expected: Registry includes `knowledge-vault` and `agent-learner`

- [ ] **Step 3: Build and test**

```bash
make build && make test && make check-generate
```

- [ ] **Step 4: Verify skills appear**

```bash
./clawkit list | grep -E "knowledge|learner"
```

Expected:

```
knowledge-vault    Quan ly kien thuc doanh nghiep trong Obsidian vault...
agent-learner      Tu hoc va cai thien — luu quy trinh thanh cong...
```

- [ ] **Step 5: Commit**

```bash
git add skills/skills.go internal/installer/registry.json
git commit -m "feat: register knowledge-vault and agent-learner skills"
```

---

## Future Work (OpenClaw Runtime Changes)

These are NOT in this plan but are needed for full self-improvement:

1. **Turn-counter nudge system** — OpenClaw injects "time to save what you learned" every N turns
2. **Multi-skill loading** — Allow `knowledge-vault` + `agent-learner` alongside domain skills (e.g., `sme-accounting`)
3. **Session auto-persistence** — OpenClaw saves every conversation to `vault-cli session save` automatically
4. **Memory injection at startup** — OpenClaw reads MEMORY.md + USER.md and injects into system prompt
5. **Skill index in prompt** — OpenClaw lists saved skills (from `vault-cli learn list`) in `<available_skills>` context

---

## Summary

| Task | What                                           | Files                     | Test              |
| ---- | ---------------------------------------------- | ------------------------- | ----------------- |
| 1    | Frontmatter parser                             | frontmatter.go + test     | Unit test         |
| 2    | Store layer (vault path, file I/O, session DB) | go.mod, main.go, store.go | Compile           |
| 3    | Note commands (Obsidian CRUD)                  | note.go                   | Manual CLI        |
| 4    | Memory commands (Hermes-style capped memory)   | memory.go                 | Manual CLI        |
| 5    | Session commands (FTS5 search)                 | session.go                | Manual CLI        |
| 6    | Learn commands (skill creation/patching)       | learn.go                  | Manual CLI        |
| 7    | Knowledge Vault skill (SKILL.md)               | SKILL.md, config.json     | Registry          |
| 8    | Agent Learner skill (SKILL.md)                 | SKILL.md, config.json     | Registry          |
| 9    | Workspace template update                      | AGENTS.md                 | Manual review     |
| 10   | Embed + registry                               | skills.go                 | make build + list |
