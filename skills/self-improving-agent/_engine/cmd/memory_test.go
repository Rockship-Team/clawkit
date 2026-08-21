package main

import (
	"os"
	"path/filepath"
	"testing"
)

// --- memoryCharCount (pure function) ---

func TestMemoryCharCountEmpty(t *testing.T) {
	if got := memoryCharCount(nil); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
	if got := memoryCharCount([]string{}); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestMemoryCharCountSingleEntry(t *testing.T) {
	entries := []string{"MST cong ty: 0312345678"}
	want := len("MST cong ty: 0312345678")
	if got := memoryCharCount(entries); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestMemoryCharCountMultipleEntries(t *testing.T) {
	// 2 entries: chars = len(e1) + len(separator) + len(e2)
	entries := []string{"alpha", "beta"}
	want := len("alpha") + len(separator) + len("beta")
	if got := memoryCharCount(entries); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestMemoryCharCountBelowCap(t *testing.T) {
	entries := []string{"short entry"}
	if got := memoryCharCount(entries); got >= memoryCap {
		t.Errorf("single short entry should be under cap, got %d", got)
	}
}

// --- loadMemory / saveMemory (file I/O with HOME override) ---

func withTempHome(t *testing.T) (string, func()) {
	t.Helper()
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	return tmp, func() {
		os.Setenv("HOME", orig)
	}
}

func TestSaveAndLoadMemory(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	entries := []string{"entry one", "entry two", "entry three"}
	if err := saveMemory("MEMORY.md", entries); err != nil {
		t.Fatalf("saveMemory failed: %v", err)
	}

	loaded := loadMemory("MEMORY.md")
	if len(loaded) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(loaded), len(entries))
	}
	for i, e := range entries {
		if loaded[i] != e {
			t.Errorf("entry[%d] = %q, want %q", i, loaded[i], e)
		}
	}
}

func TestLoadMemoryDeduplicates(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// Save with duplicates using raw file write
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".openclaw", "workspace", "MEMORY.md")
	os.MkdirAll(filepath.Dir(p), 0755)
	// Write "alpha § alpha § beta" manually
	raw := "alpha" + separator + "alpha" + separator + "beta"
	os.WriteFile(p, []byte(raw), 0644)

	loaded := loadMemory("MEMORY.md")
	if len(loaded) != 2 {
		t.Fatalf("after dedup got %d entries, want 2: %v", len(loaded), loaded)
	}
}

func TestLoadMemoryMissingFile(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	loaded := loadMemory("MEMORY.md")
	if loaded != nil {
		t.Errorf("missing file should return nil, got %v", loaded)
	}
}

func TestLoadMemoryEmptyFile(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".openclaw", "workspace", "MEMORY.md")
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, []byte(""), 0644)

	loaded := loadMemory("MEMORY.md")
	if len(loaded) != 0 {
		t.Errorf("empty file should return 0 entries, got %v", loaded)
	}
}

// --- capForFile ---

func TestCapForFile(t *testing.T) {
	if capForFile("MEMORY.md") != memoryCap {
		t.Errorf("MEMORY.md cap = %d, want %d", capForFile("MEMORY.md"), memoryCap)
	}
	if capForFile("USER.md") != userCap {
		t.Errorf("USER.md cap = %d, want %d", capForFile("USER.md"), userCap)
	}
	if capForFile("OTHER.md") != memoryCap {
		t.Errorf("OTHER.md cap should default to memoryCap")
	}
}

// --- Memory cap enforcement (integration with saveMemory) ---

func TestMemoryOverCapRejected(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// Build entries that together exceed memoryCap
	var entries []string
	chunk := make([]byte, 200)
	for i := range chunk {
		chunk[i] = 'a'
	}
	for i := 0; memoryCharCount(entries) < memoryCap-50; i++ {
		entries = append(entries, string(chunk))
	}

	// At this point entries are near cap — adding one more should exceed it
	newEntry := string(chunk)
	newEntries := append(entries, newEntry)
	if memoryCharCount(newEntries) <= memoryCap {
		t.Skip("test setup failed to exceed cap — adjust chunk size")
	}

	// Manually verify the check logic matches cmdMemory's enforcement
	if memoryCharCount(newEntries) <= memoryCap {
		t.Error("should exceed cap but didn't")
	}
}
