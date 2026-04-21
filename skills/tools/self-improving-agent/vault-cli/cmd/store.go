package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// vaultConfig holds paths for the vault and session database.
type vaultConfig struct {
	VaultPath string `json:"vault_path"`
	DBPath    string `json:"db_path"`
}

// loadVaultConfig reads config from the OpenClaw workspace config file.
// Resolution order:
//  1. $VAULT_CONFIG env var (absolute path to vault-config.json)
//  2. vault-config.json in the current working directory
//  3. ~/.openclaw/workspace/skills/knowledge-vault/vault-config.json (legacy default)
//
// Auto-creates vault-config.json with defaults if missing.
func loadVaultConfig() vaultConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return vaultConfig{VaultPath: "ObsidianVault", DBPath: "sessions.db"}
	}

	// 1. Explicit env var
	configPath := os.Getenv("VAULT_CONFIG")

	// 2. vault-config.json next to the binary / current working dir
	if configPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(cwd, "vault-config.json")
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
			}
		}
	}

	// 3. Legacy default: knowledge-vault skill directory
	skillDir := filepath.Join(home, ".openclaw", "workspace", "skills", "knowledge-vault")
	if configPath == "" {
		configPath = filepath.Join(skillDir, "vault-config.json")
	} else {
		skillDir = filepath.Dir(configPath)
	}

	defaultCfg := vaultConfig{
		VaultPath: skillDir,
		DBPath:    filepath.Join(skillDir, "session.db"),
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Auto-create with defaults
		if mkErr := os.MkdirAll(skillDir, 0755); mkErr == nil {
			if encoded, encErr := json.MarshalIndent(defaultCfg, "", "  "); encErr == nil {
				_ = os.WriteFile(configPath, append(encoded, '\n'), 0644)
			}
		}
		return defaultCfg
	}

	var cfg vaultConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultCfg
	}

	cfg.VaultPath = expandHome(cfg.VaultPath, home)
	cfg.DBPath = expandHome(cfg.DBPath, home)

	if cfg.VaultPath == "" {
		cfg.VaultPath = skillDir
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(cfg.VaultPath, "session.db")
	}

	return cfg
}

// expandHome replaces a leading "~" with the user's home directory.
func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// mustVaultPath returns the vault path and creates the directory if it does not exist.
func mustVaultPath() string {
	cfg := loadVaultConfig()
	if err := os.MkdirAll(cfg.VaultPath, 0755); err != nil {
		errOut(fmt.Sprintf("cannot create vault directory: %v", err))
	}
	return cfg.VaultPath
}

// openSessionDB opens the SQLite session database with WAL mode enabled,
// and creates the required tables if they do not exist.
func openSessionDB() *sql.DB {
	cfg := loadVaultConfig()

	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		errOut(fmt.Sprintf("cannot create db directory: %v", err))
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		errOut(fmt.Sprintf("cannot open database: %v", err))
	}

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		errOut(fmt.Sprintf("cannot set WAL mode: %v", err))
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		title       TEXT NOT NULL DEFAULT '',
		started_at  TEXT NOT NULL,
		ended_at    TEXT,
		summary     TEXT,
		tags        TEXT DEFAULT '',
		metadata    TEXT DEFAULT '{}'
	);

	CREATE TABLE IF NOT EXISTS messages (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		role       TEXT NOT NULL,
		content    TEXT NOT NULL,
		timestamp  TEXT NOT NULL,
		metadata   TEXT DEFAULT '{}'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		content,
		content_rowid='rowid'
	);

	CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, content) VALUES ('delete', OLD.rowid, OLD.content);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, content) VALUES ('delete', OLD.rowid, OLD.content);
		INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
	END;
	`

	if _, err := db.Exec(schema); err != nil {
		errOut(fmt.Sprintf("cannot create tables: %v", err))
	}

	return db
}

// jsonOut writes a JSON response to stdout.
func jsonOut(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
		os.Exit(1)
	}
}

// okOut writes a success JSON response to stdout.
func okOut(msg string) {
	jsonOut(map[string]string{"status": "ok", "message": msg})
}

// errOut writes an error JSON response to stderr and exits with code 1.
func errOut(msg string) {
	result := map[string]string{"status": "error", "message": msg}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(1)
}

// readNote reads a markdown note from the vault by relative path.
func readNote(vaultPath, relPath string) (string, error) {
	fullPath := filepath.Join(vaultPath, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeNote writes a markdown note to the vault by relative path.
// Creates parent directories as needed.
func writeNote(vaultPath, relPath, content string) error {
	fullPath := filepath.Join(vaultPath, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// listNotes returns all .md files in the vault, skipping the .obsidian/ directory.
// Paths are returned relative to the vault root.
func listNotes(vaultPath string) ([]string, error) {
	var notes []string
	err := filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(vaultPath, path)

		// Skip .obsidian directory
		if info.IsDir() && info.Name() == ".obsidian" {
			return filepath.SkipDir
		}
		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && rel != "." {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			notes = append(notes, filepath.ToSlash(rel))
		}
		return nil
	})
	return notes, err
}

var linkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// extractLinks finds all Obsidian-style [[links]] in the content.
func extractLinks(content string) []string {
	matches := linkRe.FindAllStringSubmatch(content, -1)
	links := make([]string, 0, len(matches))
	for _, m := range matches {
		links = append(links, m[1])
	}
	return links
}

var tagRe = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9/_-]+)`)

// extractTags finds all #tags in the content.
func extractTags(content string) []string {
	matches := tagRe.FindAllStringSubmatch(content, -1)
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

// Vietnam timezone (UTC+7)
var vnLoc = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)

// vnNow returns the current time in Vietnam timezone.
func vnNow() time.Time {
	return time.Now().In(vnLoc)
}

// vnNowISO returns the current time in Vietnam timezone as ISO 8601 string.
func vnNowISO() string {
	return vnNow().Format(time.RFC3339)
}

// vnToday returns today's date in Vietnam timezone as YYYY-MM-DD.
func vnToday() string {
	return vnNow().Format("2006-01-02")
}

// newID generates a UUID v4-style identifier.
// Uses time-seeded random; not cryptographically secure.
func newID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// init ensures runtime.GOOS is available for cross-platform checks.
var _ = runtime.GOOS
