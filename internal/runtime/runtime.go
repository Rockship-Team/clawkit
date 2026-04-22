// Package runtime manages shared skill runtimes under ~/.clawkit/runtimes/.
// A runtime is the contents of a skill's or group's _cli/ directory installed
// once per key and referenced by every skill that needs it. Binaries declared
// in _cli.json are symlinked into ~/.clawkit/bin (on PATH) so skills can
// invoke them by bare name, and data paths are preserved across re-installs
// so shared databases survive skill updates.
package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Spec is the shape of _cli.json.
type Spec struct {
	Exclude   []string `json:"exclude,omitempty"`    // paths inside _cli/ skipped on install (source, tests, etc.)
	DataPaths []string `json:"data_paths,omitempty"` // paths preserved across re-installs
	Bins      []string `json:"bins,omitempty"`       // names symlinked into ~/.clawkit/bin
}

// SpecFile is the name of the runtime metadata file, placed alongside _cli/.
const SpecFile = "_cli.json"

// CLIDir is the directory name that holds runtime payloads inside a skill
// or group.
const CLIDir = "_cli"

// Dir returns ~/.clawkit/runtimes/<key>.
func Dir(key string) string {
	return filepath.Join(rootDir(), "runtimes", key)
}

// BinDir returns ~/.clawkit/bin — the dir added to PATH.
func BinDir() string {
	return filepath.Join(rootDir(), "bin")
}

func rootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "clawkit")
	}
	return filepath.Join(home, ".clawkit")
}

// LoadSpec reads _cli.json next to parentDir/_cli/. Missing file returns a
// zero-value Spec and nil error.
func LoadSpec(parentDir string) (*Spec, error) {
	data, err := os.ReadFile(filepath.Join(parentDir, SpecFile))
	if err != nil {
		if os.IsNotExist(err) {
			return &Spec{}, nil
		}
		return nil, err
	}
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", SpecFile, err)
	}
	return &s, nil
}

// LoadEmbeddedSpec is the embedded-FS equivalent of LoadSpec.
func LoadEmbeddedSpec(embedFS fs.FS, parentPath string) (*Spec, error) {
	data, err := fs.ReadFile(embedFS, parentPath+"/"+SpecFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Spec{}, nil
		}
		return nil, err
	}
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", SpecFile, err)
	}
	return &s, nil
}

// Install copies srcCLIDir into ~/.clawkit/runtimes/<key>, skipping paths in
// spec.Exclude and preserving existing paths in spec.DataPaths.
func Install(key, srcCLIDir string, spec *Spec) error {
	dst := Dir(key)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}
	if err := pruneStale(dst, spec.DataPaths); err != nil {
		return fmt.Errorf("prune stale runtime entries: %w", err)
	}
	return filepath.Walk(srcCLIDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcCLIDir, path)
		if rel == "." {
			return nil
		}
		if isExcluded(rel, spec.Exclude) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			if isData(rel, spec.DataPaths) {
				if _, err := os.Stat(target); err == nil {
					return filepath.SkipDir
				}
			}
			return os.MkdirAll(target, 0o755)
		}
		if isData(rel, spec.DataPaths) {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// InstallEmbedded is the embedded-FS equivalent of Install. Since embed.FS
// does not preserve executable bits, any file whose basename matches
// spec.Bins is written with 0o755; everything else uses 0o644.
func InstallEmbedded(key string, embedFS fs.FS, srcCLIPath string, spec *Spec) error {
	dst := Dir(key)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}
	if err := pruneStale(dst, spec.DataPaths); err != nil {
		return fmt.Errorf("prune stale runtime entries: %w", err)
	}
	binSet := make(map[string]bool, len(spec.Bins))
	for _, b := range spec.Bins {
		binSet[b] = true
	}
	return fs.WalkDir(embedFS, srcCLIPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcCLIPath, path)
		if rel == "." {
			return nil
		}
		if isExcluded(rel, spec.Exclude) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if isData(rel, spec.DataPaths) {
				if _, err := os.Stat(target); err == nil {
					return fs.SkipDir
				}
			}
			return os.MkdirAll(target, 0o755)
		}
		if isData(rel, spec.DataPaths) {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		data, err := fs.ReadFile(embedFS, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if binSet[filepath.Base(rel)] {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
}

// LinkBins chmod +x each bin and symlinks it into ~/.clawkit/bin. On Windows,
// where symlinks often require admin, it copies instead.
func LinkBins(key string, bins []string) error {
	if len(bins) == 0 {
		return nil
	}
	src := Dir(key)
	binDir := BinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}
	for _, name := range bins {
		srcPath := filepath.Join(src, name)
		if runtime.GOOS == "windows" {
			if _, err := os.Stat(srcPath); err != nil {
				if _, err2 := os.Stat(srcPath + ".exe"); err2 == nil {
					srcPath += ".exe"
					name += ".exe"
				} else {
					return fmt.Errorf("bin %s not found in runtime %s", name, key)
				}
			}
		} else {
			if _, err := os.Stat(srcPath); err != nil {
				return fmt.Errorf("bin %s not found in runtime %s", name, key)
			}
			_ = os.Chmod(srcPath, 0o755)
		}
		dst := filepath.Join(binDir, name)
		_ = os.Remove(dst)
		if runtime.GOOS == "windows" {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("read bin %s: %w", name, err)
			}
			if err := os.WriteFile(dst, data, 0o755); err != nil {
				return fmt.Errorf("write bin %s: %w", dst, err)
			}
			continue
		}
		if err := os.Symlink(srcPath, dst); err != nil {
			return fmt.Errorf("symlink %s -> %s: %w", dst, srcPath, err)
		}
	}
	return nil
}

// Purge removes ~/.clawkit/runtimes/<key> and its bin symlinks.
func Purge(key string, bins []string) error {
	for _, name := range bins {
		_ = os.Remove(filepath.Join(BinDir(), name))
	}
	return os.RemoveAll(Dir(key))
}

// List returns the names of every runtime currently installed.
func List() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(rootDir(), "runtimes"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// pruneStale removes every top-level entry in dst whose path is not rooted
// at one of dataPaths. This lets re-installs reflect updated _cli.json
// excludes (e.g. dropping a previously-copied `cmd/` directory) while
// leaving user state untouched.
func pruneStale(dst string, dataPaths []string) error {
	entries, err := os.ReadDir(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	roots := make(map[string]bool, len(dataPaths))
	for _, p := range dataPaths {
		p = filepath.ToSlash(p)
		root := strings.SplitN(p, "/", 2)[0]
		if root != "" {
			roots[root] = true
		}
	}
	for _, e := range entries {
		if roots[e.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dst, e.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", e.Name(), err)
		}
	}
	return nil
}

func isExcluded(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		p = filepath.ToSlash(p)
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
	}
	return false
}

func isData(rel string, dataPaths []string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range dataPaths {
		p = filepath.ToSlash(p)
		if rel == p || strings.HasPrefix(rel, p+"/") || strings.HasPrefix(p, rel+"/") {
			return true
		}
	}
	return false
}
