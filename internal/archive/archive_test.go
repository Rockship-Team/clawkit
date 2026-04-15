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

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		relPath  string
		patterns []string
		want     bool
	}{
		// No patterns → never exclude.
		{"anything.go", nil, false},
		{"anything.go", []string{}, false},
		{".", []string{"*"}, false},

		// Exact file name (no slash → component match).
		{"PRD.md", []string{"PRD.md"}, true},
		{"SKILL.md", []string{"PRD.md"}, false},

		// Directory name as component match.
		{"cmd/main.go", []string{"cmd"}, true},
		{"cmd/sub/deep.go", []string{"cmd"}, true},
		{"tools/crawl/main.go", []string{"tools"}, true},
		{"data/tips.json", []string{"cmd"}, false},
		{"crawl", []string{"crawl"}, true},
		{"tools/crawl/main.go", []string{"crawl"}, true},

		// Wildcard component match (like tsconfig).
		{"store_test.go", []string{"*_test.go"}, true},
		{"cmd/simulator_test.go", []string{"*_test.go"}, true},
		{"cmd/store.go", []string{"*_test.go"}, false},

		// Multiple patterns.
		{"PRD.md", []string{"cmd", "PRD.md"}, true},
		{"cmd/main.go", []string{"cmd", "PRD.md"}, true},
		{"data/tips.json", []string{"cmd", "PRD.md"}, false},

		// ** glob patterns.
		{"src/utils/helper.go", []string{"**/*.go"}, true},
		{"main.go", []string{"**/*.go"}, true},
		{"src/utils/helper.ts", []string{"**/*.go"}, false},
		{"a/b/test", []string{"**/test"}, true},
		{"test", []string{"**/test"}, true},
		{"a/testing", []string{"**/test"}, false},

		// Path with slash → prefix match.
		{"tools/crawl/main.go", []string{"tools/crawl"}, true},
		{"tools/crawl", []string{"tools/crawl"}, true},
		{"tools/other/main.go", []string{"tools/crawl"}, false},
	}

	for _, tt := range tests {
		got := shouldExclude(tt.relPath, tt.patterns)
		if got != tt.want {
			t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.relPath, tt.patterns, got, tt.want)
		}
	}
}

func TestCreateTarGzWithExclude(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "test-skill")
	files := map[string]string{
		"SKILL.md":             "# Skill",
		"data/tips.json":       `[{"id":"t1"}]`,
		"cmd/main.go":          "package main",
		"cmd/store_test.go":    "package main",
		"tools/crawl/fetch.go": "package crawl",
		"PRD.md":               "# PRD",
		"CHECKLIST.md":         "# Checklist",
		"sol-cli":              "binary",
	}
	for relPath, content := range files {
		fullPath := filepath.Join(srcDir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	excludePatterns := []string{"cmd", "tools", "PRD.md", "CHECKLIST.md", "*_test.go"}

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	if err := CreateTarGz(srcDir, archivePath, excludePatterns); err != nil {
		t.Fatalf("CreateTarGz with exclude: %v", err)
	}

	destDir := filepath.Join(t.TempDir(), "extracted")
	os.MkdirAll(destDir, 0755)
	if err := ExtractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	for _, want := range []string{"SKILL.md", "data/tips.json", "sol-cli"} {
		if _, err := os.Stat(filepath.Join(destDir, want)); err != nil {
			t.Errorf("expected %s to be included, but missing", want)
		}
	}

	for _, excluded := range []string{"cmd/main.go", "cmd/store_test.go", "tools/crawl/fetch.go", "PRD.md", "CHECKLIST.md"} {
		if _, err := os.Stat(filepath.Join(destDir, excluded)); err == nil {
			t.Errorf("expected %s to be excluded, but found", excluded)
		}
	}
}
