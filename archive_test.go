package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndExtractTarGz(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir()
	skillDir := filepath.Join(srcDir, "test-skill")
	os.MkdirAll(filepath.Join(skillDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(skillDir, "subdir", "data.txt"), []byte("hello"), 0644)

	// Create archive
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	err := createTarGz(skillDir, archivePath)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Verify archive exists
	fi, err := os.Stat(archivePath)
	if err != nil {
		t.Fatal("archive not created")
	}
	if fi.Size() == 0 {
		t.Fatal("archive is empty")
	}

	// Extract
	destDir := filepath.Join(t.TempDir(), "extracted")
	os.MkdirAll(destDir, 0755)
	err = extractTarGz(archivePath, destDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Verify extracted files
	data, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatal("SKILL.md not extracted")
	}
	if string(data) != "# Test" {
		t.Errorf("SKILL.md content mismatch: %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(destDir, "subdir", "data.txt"))
	if err != nil {
		t.Fatal("subdir/data.txt not extracted")
	}
	if string(data) != "hello" {
		t.Errorf("data.txt content mismatch: %q", string(data))
	}
}
