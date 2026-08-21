package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// writeTarGz builds a .tar.gz at path containing files keyed by their path
// inside the archive. If topDir is non-empty, it's prepended to every entry.
func writeTarGz(t *testing.T, path string, files map[string]string, topDir string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	for name, content := range files {
		entry := name
		if topDir != "" {
			entry = topDir + "/" + name
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:     entry,
			Size:     int64(len(content)),
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write content: %v", err)
		}
	}
	tw.Close()
	gzw.Close()
}

func TestExtractTarGzStripsTopLevelDir(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "nested.tar.gz")
	writeTarGz(t, archivePath, map[string]string{
		"SKILL.md":        "# Skill",
		"subdir/data.txt": "hello",
	}, "my-skill")

	destDir := t.TempDir()
	if err := ExtractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	for name, want := range map[string]string{
		"SKILL.md":        "# Skill",
		"subdir/data.txt": "hello",
	} {
		got, err := os.ReadFile(filepath.Join(destDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

func TestExtractTarGzFlatArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "flat.tar.gz")
	writeTarGz(t, archivePath, map[string]string{"gog": "binary"}, "")

	destDir := t.TempDir()
	if err := ExtractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("ExtractTarGz flat archive: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "gog"))
	if err != nil {
		t.Fatalf("gog binary not extracted: %v", err)
	}
	if string(data) != "binary" {
		t.Errorf("got %q, want %q", data, "binary")
	}
}

func TestExtractZipStripsTopLevelDir(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "nested.zip")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	for name, content := range map[string]string{
		"my-skill/SKILL.md":        "# Skill",
		"my-skill/subdir/data.txt": "hello",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("zip write: %v", err)
		}
	}
	zw.Close()
	f.Close()

	destDir := t.TempDir()
	if err := ExtractZip(archivePath, destDir); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	for name, want := range map[string]string{
		"SKILL.md":        "# Skill",
		"subdir/data.txt": "hello",
	} {
		got, err := os.ReadFile(filepath.Join(destDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
