package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// skillDir returns ~/.openclaw/workspace/skills/sol-finance-coach
// Override with SOL_DATA_DIR env var for testing.
func skillDir() string {
	if dir := os.Getenv("SOL_DATA_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		errOut("cannot find home directory: " + err.Error())
		os.Exit(1)
	}
	return filepath.Join(home, ".openclaw", "workspace", "skills", "sol-finance-coach")
}

// dataPath returns the full path to a file in the skill's data/ directory.
func dataPath(name string) string {
	return filepath.Join(skillDir(), "data", name)
}

// userPath returns the full path to a runtime user data file in data/.
func userPath(name string) string {
	return dataPath(name)
}

func ensureDataDirs() {
	os.MkdirAll(filepath.Join(skillDir(), "data"), 0o755)
}

func migrateFileIfNeeded(srcPath, dstPath string) {
	if _, err := os.Stat(dstPath); err == nil {
		return
	}
	if _, err := os.Stat(srcPath); err != nil {
		return
	}

	if err := os.Rename(srcPath, dstPath); err == nil {
		return
	}

	// Fallback copy for environments where rename may fail.
	from, err := os.Open(srcPath)
	if err != nil {
		return
	}
	defer from.Close()

	to, err := os.Create(dstPath)
	if err != nil {
		return
	}
	if _, err := io.Copy(to, from); err != nil {
		to.Close()
		return
	}
	to.Close()
	_ = os.Remove(srcPath)
}

// readJSON reads a JSON file into v. Returns false if file does not exist.
func readJSON(path string, v interface{}) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false
	}
	return true
}

// writeJSON writes v as indented JSON to path, creating parent dirs.
func writeJSON(path string, v interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// jsonOut prints a JSON object to stdout.
func jsonOut(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// errOut prints a JSON error to stdout.
func errOut(msg string) {
	jsonOut(map[string]interface{}{"ok": false, "error": msg})
}

// okOut prints a simple ok JSON.
func okOut(extra map[string]interface{}) {
	out := map[string]interface{}{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	jsonOut(out)
}

// vnNow returns current time in Asia/Ho_Chi_Minh.
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

// parseAmount parses Vietnamese amount strings: 55000, 55k, 55.000, 1.5tr, 1tr5
func parseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	// Handle "tr" (trieu = million)
	if strings.Contains(s, "tr") {
		s = strings.ReplaceAll(s, ".", "")
		parts := strings.SplitN(s, "tr", 2)
		whole, _ := strconv.ParseFloat(parts[0], 64)
		frac := 0.0
		if len(parts) > 1 && parts[1] != "" {
			f, _ := strconv.ParseFloat(parts[1], 64)
			// "1tr5" = 1.5 million
			if f > 0 && f < 10 {
				frac = f * 100000
			} else {
				frac = f
			}
		}
		return int64(whole*1000000 + frac), nil
	}

	// Handle "k" (thousand)
	if strings.HasSuffix(s, "k") || strings.HasSuffix(s, "K") {
		s = s[:len(s)-1]
		s = strings.ReplaceAll(s, ".", "")
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		return int64(v * 1000), nil
	}

	// Handle dot as thousand separator: "55.000" -> "55000"
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

	return strconv.ParseInt(s, 10, 64)
}

// ensureInit creates user data directory if needed.
func ensureInit() {
	os.MkdirAll(skillDir(), 0o755)
	ensureDataDirs()

	legacyUserFiles := []string{
		"profile.json",
		"transactions.json",
		"loyalty.json",
	}
	for _, name := range legacyUserFiles {
		migrateFileIfNeeded(filepath.Join(skillDir(), name), userPath(name))
		migrateFileIfNeeded(filepath.Join(skillDir(), "data", "user", name), userPath(name))
	}
}

func deterministicIndex(seed string, n int) int {
	if n <= 0 {
		return 0
	}
	h := sha256.Sum256([]byte(seed))
	return int(binary.BigEndian.Uint32(h[:4]) % uint32(n))
}
