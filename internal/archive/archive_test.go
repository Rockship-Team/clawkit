package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndExtractTarGz(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string // relative path → content
	}{
		{
			name: "single file",
			files: map[string]string{
				"SKILL.md": "# Test Skill",
			},
		},
		{
			name: "nested directories",
			files: map[string]string{
				"SKILL.md":          "# Nested",
				"subdir/data.txt":   "hello",
				"subdir/deep/f.txt": "deep content",
			},
		},
		{
			name:  "empty directory only",
			files: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source.
			srcDir := filepath.Join(t.TempDir(), "test-skill")
			for relPath, content := range tt.files {
				fullPath := filepath.Join(srcDir, relPath)
				os.MkdirAll(filepath.Dir(fullPath), 0755)
				os.WriteFile(fullPath, []byte(content), 0644)
			}
			if len(tt.files) == 0 {
				os.MkdirAll(srcDir, 0755)
			}

			// Create archive.
			archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
			if err := CreateTarGz(srcDir, archivePath); err != nil {
				t.Fatalf("CreateTarGz() error: %v", err)
			}

			fi, _ := os.Stat(archivePath)
			if fi.Size() == 0 {
				t.Fatal("archive is empty")
			}

			// Extract.
			destDir := filepath.Join(t.TempDir(), "extracted")
			os.MkdirAll(destDir, 0755)
			if err := ExtractTarGz(archivePath, destDir); err != nil {
				t.Fatalf("ExtractTarGz() error: %v", err)
			}

			// Verify.
			for relPath, wantContent := range tt.files {
				data, err := os.ReadFile(filepath.Join(destDir, relPath))
				if err != nil {
					t.Errorf("file %s not extracted: %v", relPath, err)
					continue
				}
				if string(data) != wantContent {
					t.Errorf("file %s: got %q, want %q", relPath, string(data), wantContent)
				}
			}
		})
	}
}
