package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func smeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "workspace", "sme-data")
}

func dbPath() string  { return filepath.Join(smeDir(), "sme.db") }
func cfgPath() string { return filepath.Join(smeDir(), "connections.json") }

func openDB() (*sql.DB, error) {
	if db != nil {
		return db, nil
	}
	os.MkdirAll(smeDir(), 0o755)
	var err error
	db, err = sql.Open("sqlite", dbPath()+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA foreign_keys = ON")
	return db, nil
}

func mustDB() *sql.DB {
	d, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: open db: %v\n", err)
		os.Exit(1)
	}
	return d
}

// --- JSON output ---

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

// --- Time ---

func vnNow() time.Time {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	if loc == nil {
		return time.Now().UTC().Add(7 * time.Hour)
	}
	return time.Now().In(loc)
}

func vnToday() string  { return vnNow().Format("2006-01-02") }
func vnNowISO() string { return vnNow().Format("2006-01-02T15:04:05+07:00") }

// --- UUID ---

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

// --- VND parsing ---

func parseVND(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.TrimRight(s, "đdVNDvnd ")
	if strings.HasSuffix(s, "ty") {
		s = strings.TrimSuffix(s, "ty")
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e9)
	}
	if strings.HasSuffix(s, "tr") || strings.HasSuffix(s, "trieu") {
		s = strings.TrimSuffix(s, "trieu")
		s = strings.TrimSuffix(s, "tr")
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e6)
	}
	if strings.HasSuffix(s, "k") || strings.HasSuffix(s, "K") {
		s = s[:len(s)-1]
		var v float64
		fmt.Sscanf(s, "%f", &v)
		return int64(v * 1e3)
	}
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// --- Query helpers ---

func queryRows(q string, args ...interface{}) ([]map[string]interface{}, error) {
	d := mustDB()
	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := map[string]interface{}{}
		for i, c := range cols {
			if b, ok := vals[i].([]byte); ok {
				row[c] = string(b)
			} else {
				row[c] = vals[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func queryOne(q string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := queryRows(q, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func exec(q string, args ...interface{}) (sql.Result, error) {
	return mustDB().Exec(q, args...)
}
