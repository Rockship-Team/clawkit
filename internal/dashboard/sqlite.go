package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// sqliteAvailable returns true if the sqlite3 CLI is on PATH.
func sqliteAvailable() bool {
	_, err := exec.LookPath("sqlite3")
	return err == nil
}

// runQuery executes a SQL statement against dbPath using sqlite3 CLI.
// For SELECT it returns []map[string]any; for DML returns {"changes": N}.
func runQuery(dbPath, sql string) (any, error) {
	if !sqliteAvailable() {
		return nil, fmt.Errorf("sqlite3 CLI not found — install sqlite3 to use the DB viewer")
	}

	isSelect := isSelectQuery(sql)

	var args []string
	if isSelect {
		args = []string{"-json", dbPath, sql}
	} else {
		// For DML: capture changes count
		wrappedSQL := sql + "; SELECT changes() as changes;"
		args = []string{"-json", dbPath, wrappedSQL}
	}

	cmd := exec.Command("sqlite3", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("sqlite3: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		if isSelect {
			return []map[string]any{}, nil
		}
		return map[string]any{"changes": 0}, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		return nil, fmt.Errorf("parse output: %w", err)
	}

	if !isSelect && len(rows) > 0 {
		// Last row is changes()
		last := rows[len(rows)-1]
		if ch, ok := last["changes"]; ok {
			return map[string]any{"changes": ch}, nil
		}
	}
	return rows, nil
}

// listTables returns all user tables in a SQLite database.
func listTables(dbPath string) ([]string, error) {
	result, err := runQuery(dbPath, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	rows, ok := result.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	tables := make([]string, 0, len(rows))
	for _, row := range rows {
		if name, ok := row["name"].(string); ok {
			tables = append(tables, name)
		}
	}
	return tables, nil
}

// tableColumns returns column info for a table.
func tableColumns(dbPath, table string) ([]map[string]any, error) {
	result, err := runQuery(dbPath, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, err
	}
	rows, ok := result.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	return rows, nil
}

func isSelectQuery(sql string) bool {
	upper := strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "PRAGMA") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "WITH")
}

// findDBFiles returns all .db/.sqlite files in a directory (non-recursive).
func findDBFiles(dir string) []string {
	entries, _ := os.ReadDir(dir)
	var result []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" {
			result = append(result, e.Name())
		}
	}
	return result
}

// handleDB routes /api/skills/{name}/db/...
func handleDB(w http.ResponseWriter, r *http.Request, skillDir string, action string) {
	switch {
	case action == "files" && r.Method == http.MethodGet:
		dbFiles := findDBFiles(skillDir)
		jsonOK(w, dbFiles)

	case action == "tables" && r.Method == http.MethodGet:
		dbFile := r.URL.Query().Get("db")
		if dbFile == "" {
			http.Error(w, "missing ?db=", http.StatusBadRequest)
			return
		}
		dbPath := safeDBPath(skillDir, dbFile)
		if dbPath == "" {
			http.Error(w, "invalid db path", http.StatusBadRequest)
			return
		}
		tables, err := listTables(dbPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, tables)

	case action == "columns" && r.Method == http.MethodGet:
		dbFile := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		if dbFile == "" || table == "" {
			http.Error(w, "missing ?db= or ?table=", http.StatusBadRequest)
			return
		}
		dbPath := safeDBPath(skillDir, dbFile)
		if dbPath == "" {
			http.Error(w, "invalid db path", http.StatusBadRequest)
			return
		}
		cols, err := tableColumns(dbPath, table)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, cols)

	case action == "query" && r.Method == http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		var req struct {
			DB  string `json:"db"`
			SQL string `json:"sql"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.DB == "" || req.SQL == "" {
			http.Error(w, "body must be {db, sql}", http.StatusBadRequest)
			return
		}
		dbPath := safeDBPath(skillDir, req.DB)
		if dbPath == "" {
			http.Error(w, "invalid db path", http.StatusBadRequest)
			return
		}
		result, err := runQuery(dbPath, req.SQL)
		if err != nil {
			jsonOK(w, map[string]any{"error": err.Error()})
			return
		}
		jsonOK(w, map[string]any{"result": result})

	default:
		http.NotFound(w, r)
	}
}

// safeDBPath validates that dbFile is just a filename (no path traversal)
// and returns the absolute path only if the file exists in skillDir.
func safeDBPath(skillDir, dbFile string) string {
	// Reject anything with path separators
	if strings.ContainsAny(dbFile, "/\\") {
		return ""
	}
	abs := filepath.Join(skillDir, dbFile)
	if _, err := os.Stat(abs); err != nil {
		return ""
	}
	return abs
}
