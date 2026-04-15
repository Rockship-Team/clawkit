package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// parseProfileYAML reads a key: value file and returns the values as a map.
// Supports single-line values and YAML block scalars (key: |).
// Lines starting with # and blank lines are skipped (outside blocks).
func parseProfileYAML(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile.yaml: %w", err)
	}

	values := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	var blockKey string
	var blockLines []string

	flushBlock := func() {
		if blockKey != "" {
			values[blockKey] = strings.TrimRight(strings.Join(blockLines, "\n"), "\n")
			blockKey = ""
			blockLines = nil
		}
	}

	for _, line := range lines {
		// Inside a block scalar: collect indented lines.
		if blockKey != "" {
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				blockLines = append(blockLines, strings.TrimSpace(line))
				continue
			}
			// Non-indented line ends the block.
			flushBlock()
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		if key == "" {
			continue
		}

		// Block scalar: "key: |"
		if val == "|" {
			blockKey = key
			blockLines = nil
			continue
		}

		// Strip surrounding quotes.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		values[key] = val
	}
	flushBlock()
	return values, nil
}

// applyProfile overlays a profile's files onto the base skill directory.
// It copies catalog.json, product images, and bootstrap-files from
// profiles/<profileName>/ over the base skill's files, then removes the
// profiles/ directory (only the selected profile's data is needed at runtime).
// Returns the profile's key-value pairs for template placeholder substitution.
func applyProfile(skillDir, profileName string) (map[string]string, error) {
	profileDir := filepath.Join(skillDir, "profiles", profileName)
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		available, _ := listProfiles(skillDir)
		if len(available) > 0 {
			return nil, fmt.Errorf("profile '%s' not found (available: %s)", profileName, strings.Join(available, ", "))
		}
		return nil, fmt.Errorf("profile '%s' not found and no profiles/ directory exists", profileName)
	}

	// Parse profile values.
	values := make(map[string]string)
	yamlPath := filepath.Join(profileDir, "profile.yaml")
	if fileExists(yamlPath) {
		parsed, err := parseProfileYAML(yamlPath)
		if err != nil {
			return nil, err
		}
		values = parsed
	}

	// Overlay catalog.json.
	if src := filepath.Join(profileDir, "catalog.json"); fileExists(src) {
		dst := filepath.Join(skillDir, "catalog.json")
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("overlay catalog.json: %w", err)
		}
	}

	// Overlay schema.json (with extend-merge support).
	if src := filepath.Join(profileDir, "schema.json"); fileExists(src) {
		dst := filepath.Join(skillDir, "schema.json")
		if err := applySchemaOverlay(dst, src); err != nil {
			return nil, fmt.Errorf("overlay schema.json: %w", err)
		}
	}

	// Overlay product images directory.
	// Try schema.json images_dir, then common names, then skip.
	for _, imgDir := range profileImagesDirs(skillDir) {
		src := filepath.Join(profileDir, imgDir)
		if dirExists(src) {
			dst := filepath.Join(skillDir, imgDir)
			if err := copyDir(src, dst, nil); err != nil {
				return nil, fmt.Errorf("overlay %s/: %w", imgDir, err)
			}
			break
		}
	}

	// Overlay bootstrap files.
	if src := filepath.Join(profileDir, "bootstrap-files"); dirExists(src) {
		dst := filepath.Join(skillDir, "bootstrap-files")
		if err := copyDir(src, dst, nil); err != nil {
			return nil, fmt.Errorf("overlay bootstrap-files/: %w", err)
		}
	}

	// Clean up — only the selected profile's data is needed at runtime.
	os.RemoveAll(filepath.Join(skillDir, "profiles"))

	return values, nil
}

// listProfiles returns the names of available profiles in a skill directory.
func listProfiles(skillDir string) ([]string, error) {
	profilesDir := filepath.Join(skillDir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// profileImagesDirs returns candidate image directory names to overlay,
// reading from schema.json if available, falling back to common names.
func profileImagesDirs(skillDir string) []string {
	s, _ := loadSchema(skillDir)
	if s != nil && s.ImagesDir != "" {
		return []string{s.ImagesDir}
	}
	return []string{"products", "flowers"}
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
