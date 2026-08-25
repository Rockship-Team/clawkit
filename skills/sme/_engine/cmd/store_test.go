package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSmeDirHonorsDataDirEnv is the fix for the isolation gap found during
// Phase 2A testing: smeDir() used to hardcode ~/.openclaw/workspace/sme-data
// with no override, so $SME_DATA_DIR (which only ever affected dataDir(),
// the static reference-JSON path) silently did nothing for sme.db/
// connections.json — any "isolated" test run actually wrote production data.
// This test proves smeDir()/dbPath()/cfgPath() now resolve under
// $SME_DATA_DIR when set, and fall back to the untouched default when unset.
func TestSmeDirHonorsDataDirEnv(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("SME_DATA_DIR", temp)

	if got := smeDir(); got != temp {
		t.Fatalf("smeDir() = %q, want %q (SME_DATA_DIR override ignored)", got, temp)
	}
	if got, want := dbPath(), filepath.Join(temp, "sme.db"); got != want {
		t.Fatalf("dbPath() = %q, want %q", got, want)
	}
	if got, want := cfgPath(), filepath.Join(temp, "connections.json"); got != want {
		t.Fatalf("cfgPath() = %q, want %q", got, want)
	}
}

// TestSmeDirDefaultUnchangedWhenEnvUnset locks in that clearing the env var
// restores the exact pre-fix default path — the fix must never change
// production behavior for the real deployed binary, which never sets
// SME_DATA_DIR.
func TestSmeDirDefaultUnchangedWhenEnvUnset(t *testing.T) {
	t.Setenv("SME_DATA_DIR", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".openclaw", "workspace", "sme-data")
	if got := smeDir(); got != want {
		t.Fatalf("smeDir() with no override = %q, want %q", got, want)
	}
}

// TestIsolatedDBWriteNeverReachesProductionPath is the integration-level
// proof the fix set out to deliver: a write made under an $SME_DATA_DIR
// override lands ONLY in that temp directory. It never opens, reads, or
// writes the real default path — even to "check" it — because that default
// resolves to a real, in-use file on this machine (confirmed during Phase 2A
// cleanup). Isolation is proven by construction (distinct temp dir) and by
// asserting the resolved production path string differs from the temp path,
// never by touching the production file itself.
func TestIsolatedDBWriteNeverReachesProductionPath(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("SME_DATA_DIR", temp)

	// Force a fresh connection under the override — openDB() caches the
	// package-level *sql.DB, so any previous handle (there shouldn't be one,
	// since no other test in this package touches the DB) must not leak in.
	oldDB := db
	db = nil
	t.Cleanup(func() { db = nil; db = oldDB })

	d, err := openDB()
	if err != nil {
		t.Fatalf("openDB() under override: %v", err)
	}
	if _, err := d.Exec(`CREATE TABLE isolation_probe (marker TEXT)`); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO isolation_probe (marker) VALUES (?)`, "phase2b-isolation-proof"); err != nil {
		t.Fatalf("insert probe row: %v", err)
	}

	// The write must be sitting in temp/sme.db on disk...
	tempDBFile := filepath.Join(temp, "sme.db")
	if _, err := os.Stat(tempDBFile); err != nil {
		t.Fatalf("expected sme.db at %q, stat error: %v", tempDBFile, err)
	}

	// ...and the path that resolves to WITHOUT the override must be a
	// different file entirely — proven by path inequality, never by
	// opening/reading/writing that path from this test.
	home, _ := os.UserHomeDir()
	prodPath := filepath.Join(home, ".openclaw", "workspace", "sme-data", "sme.db")
	if tempDBFile == prodPath {
		t.Fatalf("isolation broken: temp db path equals production path %q", prodPath)
	}
}
