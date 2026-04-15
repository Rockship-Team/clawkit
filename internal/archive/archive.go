// Package archive provides tar.gz and zip creation and extraction utilities.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	pathpkg "path"
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

// CreateTarGz creates a .tar.gz archive from sourceDir.
// Files/dirs matching excludePatterns are omitted from the archive.
func CreateTarGz(sourceDir, outputPath string, excludePatterns ...[]string) error {
	var patterns []string
	if len(excludePatterns) > 0 {
		patterns = excludePatterns[0]
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	gzw := gzip.NewWriter(outFile)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	baseName := filepath.Base(sourceDir)

	return filepath.Walk(sourceDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(sourceDir, path)
		if shouldExclude(relPath, patterns) {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("file info header: %w", err)
		}

		// Use forward slashes in tar headers regardless of OS (tar format requires /).
		header.Name = pathpkg.Join(baseName, filepath.ToSlash(relPath))

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("write header: %w", err)
		}

		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

// shouldExclude checks whether relPath matches any of the exclude patterns.
// Supports tsconfig-style globs:
//   - "cmd"           — matches the directory (and everything inside it)
//   - "*.tmp"         — matches *.tmp at any depth
//   - "**/*.test.go"  — matches *.test.go at any depth
//   - "**/test"       — matches any path component named "test"
//   - "tools/crawl"   — matches that exact prefix
func shouldExclude(relPath string, patterns []string) bool {
	if len(patterns) == 0 || relPath == "." {
		return false
	}
	normalized := filepath.ToSlash(relPath)
	for _, pattern := range patterns {
		if matchGlob(normalized, pattern) {
			return true
		}
	}
	return false
}

// matchGlob matches a path against a single glob pattern with ** support.
func matchGlob(path, pattern string) bool {
	// Handle ** prefix: "**/<rest>" matches <rest> against any suffix.
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		parts := strings.Split(path, "/")
		for i := range parts {
			sub := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(suffix, sub); matched {
				return true
			}
		}
		return false
	}

	// No slash in pattern → treat as component-level match.
	if !strings.Contains(pattern, "/") {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
		return false
	}

	// Pattern has slashes → match against full path.
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}
	return false
}
