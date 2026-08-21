// Package archive provides tar.gz and zip creation and extraction utilities.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxFileSize is the maximum size per file during extraction (100MB).
const maxFileSize = 100 * 1024 * 1024

// ExtractTarGz extracts a .tar.gz archive into destDir.
// Handles both flat archives (files at root) and archives with a top-level directory (stripped).
func ExtractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := header.Name
		// Strip top-level directory if present (e.g. "gogcli_0.1.0_linux_amd64/gog" → "gog").
		// If there is no top-level dir (flat archive), use name as-is.
		parts := strings.SplitN(name, "/", 2)
		var relPath string
		if len(parts) == 2 && parts[1] != "" {
			relPath = parts[1]
		} else if len(parts) == 1 && parts[0] != "" {
			relPath = parts[0]
		} else {
			continue
		}

		target := filepath.Join(destDir, relPath)

		// Security: prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid path in archive: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			_, err = io.Copy(outFile, io.LimitReader(tr, maxFileSize))
			outFile.Close()
			if err != nil {
				return fmt.Errorf("write file %s: %w", target, err)
			}
		}
	}
	return nil
}

// ExtractZip extracts a .zip archive into destDir.
// Handles both flat archives and archives with a top-level directory (stripped).
func ExtractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := f.Name
		// Strip top-level directory if present.
		parts := strings.SplitN(name, "/", 2)
		var relPath string
		if len(parts) == 2 && parts[1] != "" {
			relPath = parts[1]
		} else if len(parts) == 1 && parts[0] != "" {
			relPath = parts[0]
		} else {
			continue
		}

		target := filepath.Join(destDir, relPath)

		// Security: prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid path in zip: %s", name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", name, err)
		}
		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file %s: %w", target, err)
		}
		_, err = io.Copy(outFile, io.LimitReader(rc, maxFileSize))
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("write file %s: %w", target, err)
		}
	}
	return nil
}
