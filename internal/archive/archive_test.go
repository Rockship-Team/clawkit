package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

// TestExtractTarGzFlatArchive verifies flat archives (no top-level dir) are extracted correctly.
func TestExtractTarGzFlatArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "flat.tar.gz")

	// Build a flat tar.gz: files at root, no top-level directory.
	f, _ := os.Create(archivePath)
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	content := []byte("flat binary content")
	tw.WriteHeader(&tar.Header{Name: "gog", Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg})
	tw.Write(content)
	tw.Close()
	gzw.Close()
	f.Close()

	destDir := t.TempDir()
	if err := ExtractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("ExtractTarGz flat archive: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "gog"))
	if err != nil {
		t.Fatalf("flat archive: gog binary not extracted: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("flat archive: got %q, want %q", string(data), string(content))
	}
}

// TestExtractZip verifies zip extraction for both flat and nested archives.
func TestExtractZip(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string]string // path in zip → content
		want    map[string]string // expected path in destDir → content
	}{
		{
			name:    "nested (with top-level dir)",
			entries: map[string]string{"gogcli_0.1.0_windows_amd64/gog.exe": "exe content"},
			want:    map[string]string{"gog.exe": "exe content"},
		},
		{
			name:    "flat (no top-level dir)",
			entries: map[string]string{"gog.exe": "exe flat"},
			want:    map[string]string{"gog.exe": "exe flat"},
		},
		{
			name:    "multiple files nested",
			entries: map[string]string{"top/gog.exe": "exe", "top/README.md": "readme"},
			want:    map[string]string{"gog.exe": "exe", "README.md": "readme"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "test.zip")
			zf, _ := os.Create(archivePath)
			zw := zip.NewWriter(zf)
			for name, content := range tt.entries {
				w, _ := zw.Create(name)
				w.Write([]byte(content))
			}
			zw.Close()
			zf.Close()

			destDir := t.TempDir()
			if err := ExtractZip(archivePath, destDir); err != nil {
				t.Fatalf("ExtractZip() error: %v", err)
			}
			for relPath, wantContent := range tt.want {
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
