package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// skillDir returns ~/.openclaw/workspace/skills/shinhan-b2b-coach
// Override with B2B_DATA_DIR env var for testing.
func skillDir() string {
	if dir := os.Getenv("B2B_DATA_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		errOut("cannot find home directory: " + err.Error())
		os.Exit(1)
	}
	return filepath.Join(home, ".openclaw", "workspace", "skills", "shinhan-b2b-coach")
}

// dbPath returns the full path to the SQLite database file.
func dbPath() string {
	return filepath.Join(skillDir(), "b2b.db")
}

// openDB opens the SQLite database with WAL mode and foreign keys enabled.
func openDB() (*sql.DB, error) {
	os.MkdirAll(skillDir(), 0o755)
	db, err := sql.Open("sqlite", dbPath())
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// mustDB opens the database or exits with an error.
func mustDB() *sql.DB {
	db, err := openDB()
	if err != nil {
		errOut("cannot open database: " + err.Error())
		os.Exit(1)
	}
	return db
}

// jsonOut prints a JSON object to stdout.
func jsonOut(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// okOut prints a success JSON response with optional extra fields.
func okOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	jsonOut(out)
}

// errOut prints a JSON error to stdout and exits.
func errOut(msg string) {
	jsonOut(map[string]interface{}{"ok": false, "error": msg})
	os.Exit(1)
}

// vnNow returns current time in Asia/Ho_Chi_Minh timezone.
func vnNow() time.Time {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.Now().UTC().Add(7 * time.Hour)
	}
	return time.Now().In(loc)
}

// vnToday returns today's date string YYYY-MM-DD in VN timezone.
func vnToday() string {
	return vnNow().Format("2006-01-02")
}

// vnNowISO returns the current VN time in ISO 8601 format.
func vnNowISO() string {
	return vnNow().Format(time.RFC3339)
}

// newID generates a UUID v4-like string from a time-seeded random source.
func newID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// parseVND parses Vietnamese currency strings.
// Handles: 50tr, 1.5ty, 200k, 50.000.000, 1200000000
func parseVND(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ToLower(s)

	// Handle "ty" (tỷ = billion)
	if strings.Contains(s, "ty") {
		parts := strings.SplitN(s, "ty", 2)
		whole, _ := strconv.ParseFloat(strings.ReplaceAll(parts[0], ".", ""), 64)
		frac := 0.0
		if len(parts) > 1 && parts[1] != "" {
			f, _ := strconv.ParseFloat(parts[1], 64)
			if f > 0 && f < 10 {
				frac = f * 100000000
			} else {
				frac = f
			}
		}
		return int64(whole*1000000000 + frac)
	}

	// Handle "tr" (triệu = million)
	if strings.Contains(s, "tr") {
		parts := strings.SplitN(s, "tr", 2)
		whole, _ := strconv.ParseFloat(strings.ReplaceAll(parts[0], ".", ""), 64)
		frac := 0.0
		if len(parts) > 1 && parts[1] != "" {
			f, _ := strconv.ParseFloat(parts[1], 64)
			if f > 0 && f < 10 {
				frac = f * 100000
			} else {
				frac = f
			}
		}
		return int64(whole*1000000 + frac)
	}

	// Handle "k" (thousand)
	if strings.HasSuffix(s, "k") {
		s = s[:len(s)-1]
		s = strings.ReplaceAll(s, ".", "")
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return int64(v * 1000)
	}

	// Handle dot as thousand separator: "50.000.000" -> "50000000"
	if strings.Count(s, ".") >= 1 {
		parts := strings.Split(s, ".")
		allThousands := true
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) != 3 {
				allThousands = false
				break
			}
		}
		if allThousands && len(parts) > 1 {
			s = strings.ReplaceAll(s, ".", "")
		}
	}

	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// queryRows executes a query and returns results as a slice of maps.
func queryRows(db *sql.DB, q string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// queryOne executes a query and returns the first row as a map.
func queryOne(db *sql.DB, q string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := queryRows(db, q, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// exec executes a statement and returns the result.
func exec(db *sql.DB, q string, args ...interface{}) (sql.Result, error) {
	return db.Exec(q, args...)
}
