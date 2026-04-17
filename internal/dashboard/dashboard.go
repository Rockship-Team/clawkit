// Package dashboard implements the clawkit dashboard HTTP server.
package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/rockship-co/clawkit/internal/config"
)

// SkillEntry is the JSON shape returned by /api/skills.
type SkillEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Installed   bool   `json:"installed"`
	OAuthDone   bool   `json:"oauth_done"`
}

// RegistrySkill is a minimal struct for deserialising registry.json.
type RegistrySkill struct {
	Version     string `json:"version"`
	Description string `json:"description"`
}

// Registry is a minimal struct for deserialising registry.json.
type Registry struct {
	Skills map[string]RegistrySkill `json:"skills"`
}

// FileNode represents a file or directory in the tree.
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"` // relative to skill dir
	IsDir    bool        `json:"is_dir"`
	Children []*FileNode `json:"children,omitempty"`
}

// JSONTableResponse is the table-friendly view of a JSON document.
type JSONTableResponse struct {
	Kind    string           `json:"kind"`
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

// Serve starts the dashboard HTTP server on the given port.
func Serve(port int, registryData []byte) error {
	entries, err := buildEntries(registryData)
	if err != nil {
		return fmt.Errorf("build skill entries: %w", err)
	}
	_ = entries

	mux := http.NewServeMux()

	// GET /api/skills — list all skills with install status
	mux.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		live, err := buildEntries(registryData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, live)
	})

	// GET /api/skills/{name}/tree — directory tree for an installed skill
	mux.HandleFunc("/api/skills/", func(w http.ResponseWriter, r *http.Request) {
		// path: /api/skills/{name}/tree  or  /api/skills/{name}/file?path=...
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/skills/"), "/", 2)
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		skillName := parts[0]
		action := parts[1]

		skillDir := filepath.Join(config.GetSkillsDir(), skillName)
		if _, err := os.Stat(skillDir); err != nil {
			http.Error(w, "skill not installed", http.StatusNotFound)
			return
		}

		switch {
		case action == "tree" && r.Method == http.MethodGet:
			handleTree(w, r, skillDir)
		case action == "table" && r.Method == http.MethodGet:
			handleFileTable(w, r, skillDir)
		case action == "file/table" && r.Method == http.MethodGet:
			handleFileTable(w, r, skillDir)
		case action == "file":
			handleFile(w, r, skillDir)
		case strings.HasPrefix(action, "db/"):
			handleDB(w, r, skillDir, strings.TrimPrefix(action, "db/"))
		case strings.HasPrefix(action, "fs/"):
			handleFS(w, r, skillDir, strings.TrimPrefix(action, "fs/"))
		default:
			http.NotFound(w, r)
		}
	})

	// B2B Finance Coach dashboard API
	mux.HandleFunc("/api/b2b/", func(w http.ResponseWriter, r *http.Request) {
		handleB2B(w, r)
	})

	// Serve B2B dashboard HTML
	mux.HandleFunc("/b2b", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, b2bPage)
	})

	// Serve dashboard HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, htmlPage)
	})

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("  Dashboard running at %s\n", url)
	fmt.Println("  Press Ctrl+C to stop.")
	fmt.Println()
	openBrowser(url)

	return http.Serve(ln, mux)
}

// handleTree returns a JSON directory tree for a skill.
func handleTree(w http.ResponseWriter, _ *http.Request, skillDir string) {
	root, err := buildTree(skillDir, skillDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, root)
}

// handleFile reads (GET) or writes (PUT) or uploads (POST multipart) a file.
func handleFile(w http.ResponseWriter, r *http.Request, skillDir string) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "missing ?path=", http.StatusBadRequest)
		return
	}

	// Security: prevent path traversal
	abs := filepath.Join(skillDir, filepath.FromSlash(relPath))
	if !strings.HasPrefix(abs, skillDir+string(os.PathSeparator)) && abs != skillDir {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// Detect content type
		ct := detectContentType(abs, data)
		w.Header().Set("Content-Type", ct)
		w.Write(data)

	case http.MethodPut:
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB limit
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})

	case http.MethodPost:
		// Multipart file upload — replaces the file at ?path=
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "parse multipart: "+err.Error(), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dst, err := os.Create(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFileTable returns a JSON document in a table-friendly shape.
func handleFileTable(w http.ResponseWriter, r *http.Request, skillDir string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "missing ?path=", http.StatusBadRequest)
		return
	}

	abs := filepath.Join(skillDir, filepath.FromSlash(relPath))
	if !strings.HasPrefix(abs, skillDir+string(os.PathSeparator)) && abs != skillDir {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	table, err := buildJSONTable(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, table)
}

func buildJSONTable(data []byte) (JSONTableResponse, error) {
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return JSONTableResponse{}, fmt.Errorf("parse json: %w", err)
	}

	switch value := parsed.(type) {
	case []any:
		return buildJSONArrayTable(value), nil
	case map[string]any:
		return buildJSONObjectTable(value), nil
	default:
		return JSONTableResponse{
			Kind:    "value",
			Columns: []string{"value"},
			Rows: []map[string]any{{
				"value": value,
			}},
		}, nil
	}
}

func buildJSONArrayTable(items []any) JSONTableResponse {
	if len(items) == 0 {
		return JSONTableResponse{Kind: "array", Columns: []string{"value"}, Rows: []map[string]any{}}
	}

	allObjects := true
	for _, item := range items {
		if _, ok := item.(map[string]any); !ok {
			allObjects = false
			break
		}
	}

	if allObjects {
		columns := orderedJSONKeys(items)
		if len(columns) == 0 {
			rows := make([]map[string]any, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]any{"value": item})
			}
			return JSONTableResponse{Kind: "array", Columns: []string{"value"}, Rows: rows}
		}
		rows := make([]map[string]any, 0, len(items))
		for _, item := range items {
			row := make(map[string]any, len(columns))
			obj := item.(map[string]any)
			for _, column := range columns {
				row[column] = obj[column]
			}
			rows = append(rows, row)
		}
		return JSONTableResponse{Kind: "array", Columns: columns, Rows: rows}
	}

	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{"value": item})
	}
	return JSONTableResponse{Kind: "array", Columns: []string{"value"}, Rows: rows}
}

func buildJSONObjectTable(item map[string]any) JSONTableResponse {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, map[string]any{
			"key":   key,
			"value": item[key],
		})
	}

	return JSONTableResponse{Kind: "object", Columns: []string{"key", "value"}, Rows: rows}
}

func orderedJSONKeys(items []any) []string {
	keys := make([]string, 0)
	seen := make(map[string]struct{})
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemKeys := make([]string, 0, len(obj))
		for key := range obj {
			itemKeys = append(itemKeys, key)
		}
		sort.Strings(itemKeys)
		for _, key := range itemKeys {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}

func buildTree(root, dir string) (*FileNode, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(root, dir)
	if rel == "." {
		rel = ""
	}

	node := &FileNode{
		Name:  info.Name(),
		Path:  filepath.ToSlash(rel),
		IsDir: info.IsDir(),
	}

	if !info.IsDir() {
		return node, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return node, nil
	}

	// Dirs first, then files, each sorted alphabetically
	var dirs, files []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || e.Name() == "__pycache__" {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, e := range append(dirs, files...) {
		child, err := buildTree(root, filepath.Join(dir, e.Name()))
		if err == nil {
			node.Children = append(node.Children, child)
		}
	}
	return node, nil
}

func buildEntries(registryData []byte) ([]SkillEntry, error) {
	var reg Registry
	if err := json.Unmarshal(registryData, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	skillsDir := config.GetSkillsDir()

	// Only show skills that are installed on disk in the OpenClaw skills directory
	dirs, err := os.ReadDir(skillsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	names := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d.IsDir() {
			names = append(names, d.Name())
		}
	}
	sort.Strings(names)

	entries := make([]SkillEntry, 0, len(names))
	for _, name := range names {
		info := reg.Skills[name]
		entry := SkillEntry{
			Name:        name,
			Description: info.Description,
			Version:     info.Version,
			Installed:   true,
		}
		skillDir := filepath.Join(skillsDir, name)
		if cfg, err := config.LoadSkillConfig(skillDir); err == nil {
			entry.OAuthDone = cfg.OAuthDone
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func detectContentType(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".md", ".txt":
		return "text/plain; charset=utf-8"
	}
	// fallback to sniff
	ct := http.DetectContentType(data)
	return ct
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

const htmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Clawkit Dashboard</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background: #0f0f11;
  color: #e2e2e2;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
header {
  background: #18181b;
  border-bottom: 1px solid #27272a;
  padding: 14px 28px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
header h1 { font-size: 17px; font-weight: 600; }
.badge {
  margin-left: auto;
  background: #27272a;
  color: #a1a1aa;
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 999px;
}

/* ── Layout ── */
.app { display: flex; flex: 1; overflow: hidden; height: calc(100vh - 49px); }

/* ── Skill list sidebar ── */
.sidebar {
  width: 260px;
  flex-shrink: 0;
  border-right: 1px solid #27272a;
  overflow-y: auto;
  background: #111113;
}
.sidebar-header {
  padding: 14px 16px 10px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .08em;
  color: #52525b;
}
.skill-item {
  padding: 10px 16px;
  cursor: pointer;
  border-left: 2px solid transparent;
  transition: background .1s;
}
.skill-item:hover { background: #18181b; }
.skill-item.active { background: #1c1c1f; border-left-color: #6366f1; }
.skill-item-name { font-size: 13px; font-weight: 500; }
.skill-item-desc { font-size: 11px; color: #52525b; margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.skill-item-pills { display: flex; gap: 4px; margin-top: 4px; flex-wrap: wrap; }
.pill {
  font-size: 10px; padding: 1px 6px; border-radius: 999px; font-weight: 500;
}
.pill-installed { background: #14532d; color: #4ade80; }
.pill-oauth { background: #1e3a5f; color: #60a5fa; }
.pill-avail { background: #27272a; color: #71717a; }

/* ── Main panel ── */
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

/* ── Empty state ── */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #3f3f46;
  gap: 8px;
}
.empty-state .icon { font-size: 48px; }
.empty-state p { font-size: 14px; }

/* ── Skill detail header ── */
.detail-header {
  background: #18181b;
  border-bottom: 1px solid #27272a;
  padding: 14px 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.detail-header h2 { font-size: 15px; font-weight: 600; }
.detail-header .meta { font-size: 12px; color: #71717a; }

/* ── File explorer split ── */
.explorer { display: flex; flex: 1; overflow: hidden; }

.filetree {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid #27272a;
  overflow-y: auto;
  background: #111113;
  font-size: 12px;
}
.filetree-header {
  padding: 10px 12px 6px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .08em;
  color: #52525b;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.tree-node { user-select: none; }
.tree-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  border-radius: 4px;
  margin: 1px 4px;
}
.tree-row:hover { background: #1c1c1f; }
.tree-row.active { background: #27272a; color: #e2e2e2; }
.tree-icon { flex-shrink: 0; font-size: 11px; }
.tree-name { overflow: hidden; text-overflow: ellipsis; color: #a1a1aa; }
.tree-row.active .tree-name { color: #e2e2e2; }
.tree-children { /* indent handled via padding-left on row */ }

/* ── Editor / preview pane ── */
.editor-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.editor-toolbar {
  background: #18181b;
  border-bottom: 1px solid #27272a;
  padding: 8px 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  font-size: 12px;
  color: #71717a;
}
.editor-toolbar .path { flex: 1; font-family: monospace; font-size: 11px; }
.btn {
  background: #27272a;
  border: none;
  color: #e2e2e2;
  padding: 5px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background .1s;
}
.btn:hover { background: #3f3f46; }
.btn-primary { background: #4f46e5; color: #fff; }
.btn-primary:hover { background: #4338ca; }
.btn-danger { background: #7f1d1d; color: #fca5a5; }
.btn-danger:hover { background: #991b1b; }
textarea.editor {
  flex: 1;
  background: #0f0f11;
  color: #e2e2e2;
  border: none;
  outline: none;
  padding: 16px;
  font-family: "JetBrains Mono", "Fira Code", monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: none;
}
.image-preview {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  background: #0f0f11;
  padding: 24px;
}
.image-preview img { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 6px; }
.editor-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3f3f46;
  font-size: 13px;
}

/* ── Upload drop zone ── */
.drop-overlay {
  position: fixed; inset: 0; background: rgba(79,70,229,.15);
  border: 2px dashed #6366f1;
  display: none;
  align-items: center;
  justify-content: center;
  z-index: 99;
  pointer-events: none;
  font-size: 18px;
  color: #a5b4fc;
}
.drop-overlay.active { display: flex; }

/* ── Context Menu ── */
.ctx-menu {
  position: fixed; z-index: 300;
  background: #1c1c1f; border: 1px solid #3f3f46; border-radius: 8px;
  padding: 4px; min-width: 160px;
  box-shadow: 0 8px 24px rgba(0,0,0,.5);
  font-size: 13px;
}
.ctx-item {
  padding: 7px 12px; border-radius: 5px; cursor: pointer;
  display: flex; align-items: center; gap: 8px; color: #d4d4d8;
  transition: background .1s;
}
.ctx-item:hover { background: #27272a; }
.ctx-item.danger { color: #f87171; }
.ctx-item.danger:hover { background: #3f1111; }
.ctx-sep { height: 1px; background: #27272a; margin: 3px 0; }

/* ── Tree drag/drop ── */
.tree-row.drag-over { background: #312e81 !important; border-radius: 4px; }
.tree-row[draggable] { cursor: grab; }
.tree-row.dragging { opacity: .4; }

/* ── Inline rename ── */
.tree-rename-input {
  background: #27272a; border: 1px solid #4f46e5; border-radius: 4px;
  color: #e2e2e2; font-size: 12px; padding: 1px 6px; outline: none;
  width: 140px; font-family: inherit;
}
.tree-action-btn {
  background: none; border: none; cursor: pointer; padding: 2px 4px;
  color: #52525b; font-size: 12px; border-radius: 4px; line-height: 1;
  transition: color .1s, background .1s;
}
.tree-action-btn:hover { color: #e2e2e2; background: #27272a; }
.tree-action-btn.danger { color: #f87171; }
.tree-action-btn.danger:hover { background: #3f1111; }
.tree-row.multi-selected { background: #1e3a5f !important; }

/* ── Toast ── */
.toast {
  position: fixed; bottom: 24px; right: 24px;
  background: #18181b; border: 1px solid #27272a;
  color: #e2e2e2; padding: 10px 16px; border-radius: 8px;
  font-size: 13px; z-index: 100;
  opacity: 0; transform: translateY(8px);
  transition: opacity .2s, transform .2s;
  pointer-events: none;
}
.toast.show { opacity: 1; transform: translateY(0); }
.toast.ok { border-color: #14532d; color: #4ade80; }
.toast.err { border-color: #7f1d1d; color: #fca5a5; }

/* ── DB Viewer ── */
.db-viewer { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
.db-tabs {
  display: flex; align-items: center; gap: 4px;
  padding: 8px 12px; border-bottom: 1px solid #27272a;
  background: #111113; flex-shrink: 0; flex-wrap: wrap;
}
.db-tab {
  padding: 4px 12px; border-radius: 6px; font-size: 12px;
  cursor: pointer; background: #27272a; color: #a1a1aa; border: none;
  transition: background .1s;
}
.db-tab:hover { background: #3f3f46; color: #e2e2e2; }
.db-tab.active { background: #4f46e5; color: #fff; }
.db-tab-sql { background: #18181b; color: #71717a; margin-left: auto; }
.db-tab-sql.active { background: #065f46; color: #6ee7b7; }

.db-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.db-toolbar {
  padding: 8px 12px; display: flex; gap: 6px; align-items: center;
  background: #18181b; border-bottom: 1px solid #27272a; flex-shrink: 0;
}
.db-count { font-size: 11px; color: #52525b; margin-left: auto; }

.db-table-wrap { flex: 1; overflow: auto; }
table.db-table {
  width: 100%; border-collapse: collapse; font-size: 12px;
  min-width: max-content;
}
table.db-table th {
  background: #18181b; padding: 8px 12px; text-align: left;
  font-weight: 600; font-size: 11px; color: #71717a; text-transform: uppercase;
  letter-spacing: .05em; border-bottom: 1px solid #27272a;
  position: sticky; top: 0; white-space: nowrap;
}
table.db-table td {
  padding: 7px 12px; border-bottom: 1px solid #1c1c1f;
  max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  vertical-align: middle;
}
table.db-table tr:hover td { background: #18181b; }
table.db-table td.null { color: #3f3f46; font-style: italic; }
table.db-table td.act { width: 60px; text-align: center; white-space: nowrap; }
.row-btn {
  background: none; border: none; cursor: pointer; padding: 2px 4px;
  color: #52525b; font-size: 14px; transition: color .1s;
}
.row-btn:hover { color: #e2e2e2; }
.row-btn.del:hover { color: #f87171; }

/* SQL pane */
.sql-pane { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
.sql-pane textarea {
  flex: 1; background: #0f0f11; color: #e2e2e2; border: none; outline: none;
  padding: 14px 16px; font-family: monospace; font-size: 13px; line-height: 1.6; resize: none;
  border-bottom: 1px solid #27272a; max-height: 180px;
}
.sql-result { flex: 1; overflow: auto; }
table.sql-table {
  width: 100%; border-collapse: collapse; font-size: 12px; min-width: max-content;
}
table.sql-table th {
  background: #18181b; padding: 7px 12px; text-align: left;
  font-weight: 600; font-size: 11px; color: #71717a;
  border-bottom: 1px solid #27272a; position: sticky; top: 0;
}
table.sql-table td { padding: 6px 12px; border-bottom: 1px solid #1c1c1f; }
.sql-msg { padding: 12px 16px; font-size: 12px; color: #71717a; }

/* Row edit modal */
.modal-backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center; z-index: 200;
}
.modal {
  background: #18181b; border: 1px solid #27272a; border-radius: 12px;
  width: 480px; max-width: 95vw; max-height: 85vh;
  display: flex; flex-direction: column; overflow: hidden;
}
.modal-header {
  padding: 14px 20px; border-bottom: 1px solid #27272a;
  font-size: 14px; font-weight: 600; display: flex; justify-content: space-between; align-items: center;
}
.modal-body { padding: 16px 20px; overflow-y: auto; display: flex; flex-direction: column; gap: 10px; }
.modal-footer { padding: 12px 20px; border-top: 1px solid #27272a; display: flex; gap: 8px; justify-content: flex-end; }
.field-row { display: flex; flex-direction: column; gap: 3px; }
.field-label { font-size: 11px; color: #71717a; text-transform: uppercase; }
.field-input {
  background: #0f0f11; border: 1px solid #27272a; border-radius: 6px;
  color: #e2e2e2; padding: 7px 10px; font-size: 13px; outline: none;
  transition: border-color .1s; font-family: inherit;
}
.field-input:focus { border-color: #4f46e5; }
.close-btn { background: none; border: none; color: #71717a; cursor: pointer; font-size: 18px; line-height: 1; }
.close-btn:hover { color: #e2e2e2; }
</style>
</head>
<body>
<header>
  <span>⚙️</span>
  <h1>Clawkit Dashboard</h1>
  <span class="badge" id="badge">loading…</span>
  <a href="/b2b" style="margin-left:12px;padding:4px 12px;background:#0f4c81;color:#fff;border-radius:4px;font-size:11px;text-decoration:none;font-weight:500">B2B Dashboard</a>
</header>

<div class="app">
  <!-- Sidebar: skill list -->
  <aside class="sidebar">
    <div class="sidebar-header">Skills</div>
    <div id="skill-list"></div>
  </aside>

  <!-- Main content -->
  <div class="main" id="main">
    <div class="empty-state">
      <div class="icon">📦</div>
      <p>Select a skill to explore its files</p>
    </div>
  </div>
</div>

<div class="drop-overlay" id="drop-overlay">Drop image to upload</div>
<div class="toast" id="toast"></div>

<script>
// ── State ──
let skills = [];
let activeSkill = null;
let activeFilePath = null;  // relative path
let activeFileKind = null;
let activeJsonText = '';
let editorDirty = false;
let editorToolbarWired = false;

// ── Boot ──
async function boot() {
  await loadSkills();
  setInterval(loadSkills, 8000);
}

async function loadSkills() {
  const data = await apiFetch('/api/skills');
  if (!data) return;
  skills = data;
  renderSidebar();
  document.getElementById('badge').textContent =
    skills.filter(s => s.installed).length + ' installed';
}

// ── Sidebar ──
function renderSidebar() {
  const el = document.getElementById('skill-list');
  el.innerHTML = skills.map(s => {
    const active = activeSkill && activeSkill.name === s.name ? 'active' : '';
    const statusPill = s.installed
      ? '<span class="pill pill-installed">installed</span>'
      : '<span class="pill pill-avail">available</span>';
    const oauthPill = s.installed && s.oauth_done
      ? '<span class="pill pill-oauth">oauth ✓</span>' : '';
    return '<div class="skill-item ' + active + '" data-name="' + s.name + '">' +
      '<div class="skill-item-name">' + s.name + '</div>' +
      '<div class="skill-item-desc">' + (s.description || '') + '</div>' +
      '<div class="skill-item-pills">' + statusPill + oauthPill + '</div>' +
      '</div>';
  }).join('');

  el.querySelectorAll('.skill-item').forEach(item => {
    item.addEventListener('click', () => selectSkill(item.dataset.name));
  });
}

// ── Skill detail ──
async function selectSkill(name) {
  const skill = skills.find(s => s.name === name);
  if (!skill) return;
  activeSkill = skill;
  activeFilePath = null;
  editorDirty = false;
  renderSidebar(); // update active highlight

  const main = document.getElementById('main');
  if (!skill.installed) {
    main.innerHTML = '<div class="empty-state"><div class="icon">🔒</div><p>' + name + ' is not installed</p><p style="font-size:12px;color:#52525b">Run: clawkit install ' + name + '</p></div>';
    return;
  }

  // Show skeleton while loading tree
  main.innerHTML = buildDetailShell(skill);
  wireEditorToolbar();
  wireDragDrop();

  await loadTree(name);
}

function buildDetailShell(skill) {
  const oauthPill = skill.oauth_done ? '<span class="pill pill-oauth">oauth ✓</span>' : '';
  return '<div class="detail-header">' +
    '<h2>' + skill.name + '</h2>' +
    '<span class="pill pill-installed">v' + skill.version + '</span>' +
    oauthPill +
    '<span class="meta" style="margin-left:auto">' + (skill.description || '') + '</span>' +
    '</div>' +
    '<div class="explorer">' +
      '<div class="filetree">' +
        '<div class="filetree-header">Files' +
          '<span style="display:flex;gap:2px">' +
            '<button class="tree-action-btn" id="btn-new-file" title="New file">📄+</button>' +
            '<button class="tree-action-btn" id="btn-new-folder" title="New folder">📁+</button>' +
            '<label class="tree-action-btn" title="Upload files" style="cursor:pointer">⬆<input type="file" id="toolbar-upload-input" multiple style="display:none"></label>' +
            '<button class="tree-action-btn danger" id="btn-delete-selected" title="Delete selected" style="display:none">🗑 Delete</button>' +
          '</span>' +
        '</div>' +
        '<div id="tree-root"><div style="padding:12px;color:#52525b">Loading…</div></div>' +
      '</div>' +
      '<div class="editor-pane">' +
        '<div class="editor-toolbar">' +
          '<span class="path" id="editor-path">—</span>' +
          '<button class="btn" id="btn-json-toggle" style="display:none">Raw</button>' +
          '<label class="btn" id="btn-upload" style="display:none">Upload <input type="file" id="upload-input" style="display:none" multiple></label>' +
          '<button class="btn btn-primary" id="btn-save" style="display:none">Save</button>' +
        '</div>' +
        '<div class="editor-empty" id="editor-area">Select a file to view or edit</div>' +
      '</div>' +
    '</div>';
}

// ── File tree ──
function getOpenPaths() {
  const open = new Set();
  document.querySelectorAll('.tree-node[data-path]').forEach(wrap => {
    const children = wrap.querySelector('.tree-children');
    if (children && children.style.display !== 'none') {
      open.add(wrap.dataset.path);
    }
  });
  return open;
}

async function loadTree(skillName) {
  const openPaths = getOpenPaths();
  const tree = await apiFetch('/api/skills/' + skillName + '/tree');
  if (!tree) return;
  const root = document.getElementById('tree-root');
  if (!root) return;
  root.innerHTML = '';
  if (tree.children) {
    tree.children.forEach(node => root.appendChild(renderTreeNode(node, 0)));
  }
  // Restore expanded state
  if (openPaths.size > 0) {
    document.querySelectorAll('.tree-node[data-path]').forEach(wrap => {
      if (!openPaths.has(wrap.dataset.path)) return;
      const children = wrap.querySelector('.tree-children');
      const icon = wrap.querySelector('.tree-row > .tree-icon');
      if (children) children.style.display = 'block';
      if (icon && icon.textContent === '📁') icon.textContent = '📂';
    });
  }
}

function fileIcon(name) {
  const ext = name.split('.').pop().toLowerCase();
  if (['png','jpg','jpeg','gif','webp','svg'].includes(ext)) return '🖼';
  if (['md'].includes(ext)) return '📝';
  if (['json','yaml','yml','toml'].includes(ext)) return '{}';
  if (['py','go','js','ts','sh'].includes(ext)) return '📄';
  if (['db','sqlite'].includes(ext)) return '🗄';
  return '📄';
}

// ── File open/edit ──
async function openFile(relPath, rowEl) {
  // Mark active row
  document.querySelectorAll('.tree-row').forEach(r => r.classList.remove('active'));
  rowEl.classList.add('active');

  activeFilePath = relPath;
  activeFileKind = null;
  activeJsonText = '';
  editorDirty = false;

  document.getElementById('editor-path').textContent = relPath;

  const ext = relPath.split('.').pop().toLowerCase();
  const isJson = ext === 'json';
  const isImage = ['png','jpg','jpeg','gif','webp','svg'].includes(ext);
  const isDB = ['db','sqlite','sqlite3'].includes(ext);
  const isBinary = ['zip','tar','gz'].includes(ext);

  const area = document.getElementById('editor-area');
  const btnSave = document.getElementById('btn-save');
  const btnUpload = document.getElementById('btn-upload');
  const btnJsonToggle = document.getElementById('btn-json-toggle');

  btnJsonToggle.style.display = 'none';

  if (isDB) {
    activeFileKind = 'db';
    btnSave.style.display = 'none';
    btnUpload.style.display = 'none';
    openDBViewer(relPath);
    return;
  } else if (isJson) {
    activeFileKind = 'json';
    btnSave.style.display = 'none';
    btnUpload.style.display = 'none';
    btnJsonToggle.style.display = 'inline-block';
    btnJsonToggle.textContent = 'Raw';
    await openJsonTableView(relPath);
    return;
  } else if (isImage) {
    activeFileKind = 'image';
    btnSave.style.display = 'none';
    btnUpload.style.display = 'flex';
    const url = '/api/skills/' + activeSkill.name + '/file?path=' + encodeURIComponent(relPath) + '&t=' + Date.now();
    area.outerHTML; // keep reference
    document.getElementById('editor-area').replaceWith(
      Object.assign(document.createElement('div'), {
        id: 'editor-area', className: 'image-preview',
        innerHTML: '<img src="' + url + '" alt="' + relPath + '">'
      })
    );
  } else if (isBinary) {
    activeFileKind = 'binary';
    btnSave.style.display = 'none';
    btnUpload.style.display = 'none';
    setEditorHTML('<div class="editor-empty" id="editor-area">Binary file — cannot preview</div>');
  } else {
    activeFileKind = 'text';
    btnUpload.style.display = 'none';
    const resp = await fetch('/api/skills/' + activeSkill.name + '/file?path=' + encodeURIComponent(relPath));
    if (!resp.ok) { toast('Could not read file', true); return; }
    const text = await resp.text();
    const ta = document.createElement('textarea');
    ta.className = 'editor';
    ta.id = 'editor-area';
    ta.value = text;
    ta.addEventListener('input', () => {
      activeJsonText = ta.value;
      editorDirty = true;
      btnSave.style.display = 'inline-block';
    });
    setEditorHTML(null, ta);
    btnSave.style.display = 'none';
  }
}

async function openJsonTableView(relPath) {
  setEditorHTML('<div class="db-viewer" id="editor-area"><div class="editor-empty">Loading…</div></div>');

  const resp = await fetch('/api/skills/' + activeSkill.name + '/file/table?path=' + encodeURIComponent(relPath));
  if (!resp.ok) {
    setEditorHTML('<div class="editor-empty" id="editor-area">Could not load JSON table view</div>');
    toast(await resp.text(), true);
    return;
  }

  const data = await resp.json();
  renderJsonTableView(data);
}

function renderJsonTableView(data) {
  const area = document.getElementById('editor-area');
  if (!area) return;

  const columns = Array.isArray(data.columns) ? data.columns : [];
  const rows = Array.isArray(data.rows) ? data.rows : [];

  const head = columns.map(column => '<th>' + escH(column) + '</th>').join('');
  const body = rows.length > 0 ? rows.map(row => {
    const cells = columns.map(column => {
      const value = row ? row[column] : undefined;
      if (value === null || value === undefined) return '<td class="null">NULL</td>';
      const text = typeof value === 'object' ? JSON.stringify(value) : String(value);
      const short = text.length > 120 ? text.slice(0, 120) + '…' : text;
      return '<td title="' + escH(text) + '">' + escH(short) + '</td>';
    }).join('');
    return '<tr>' + cells + '</tr>';
  }).join('') : '<tr><td colspan="' + Math.max(columns.length, 1) + '" style="padding:20px;color:#52525b;text-align:center">No rows</td></tr>';

  area.innerHTML =
    '<div class="db-toolbar">' +
      '<span style="font-size:11px;color:#52525b">JSON table view</span>' +
      '<span class="db-count">' + rows.length + ' row(s)</span>' +
    '</div>' +
    '<div class="db-table-wrap"><table class="db-table"><thead><tr>' + head + '</tr></thead><tbody>' + body + '</tbody></table></div>';
}

async function showJsonRawView() {
  if (!activeSkill || !activeFilePath) return;

  const text = await loadRawJsonText(activeFilePath);
  if (text === null) return;
  activeJsonText = text;
  editorDirty = false;

  const ta = document.createElement('textarea');
  ta.className = 'editor';
  ta.id = 'editor-area';
  ta.value = text;
  ta.addEventListener('input', () => {
    activeJsonText = ta.value;
    editorDirty = true;
    document.getElementById('btn-save').style.display = 'inline-block';
  });
  setEditorHTML(null, ta);
  document.getElementById('btn-save').style.display = 'none';
}

async function loadRawJsonText(relPath) {
  const resp = await fetch('/api/skills/' + activeSkill.name + '/file?path=' + encodeURIComponent(relPath));
  if (!resp.ok) {
    toast('Could not read file', true);
    return null;
  }
  return resp.text();
}

function renderJsonFromText(text) {
  try {
    const data = buildJsonTableFromText(text);
    renderJsonTableView(data);
    return true;
  } catch (err) {
    setEditorHTML('<div class="editor-empty" id="editor-area">' + escH(err.message || 'Invalid JSON') + '</div>');
    return false;
  }
}

function buildJsonTableFromText(text) {
  const parsed = JSON.parse(text);

  if (Array.isArray(parsed)) {
    if (parsed.length === 0) {
      return { kind: 'array', columns: ['value'], rows: [] };
    }

    const allObjects = parsed.every(item => item && typeof item === 'object' && !Array.isArray(item));
    if (allObjects) {
      const columns = [];
      const seen = new Set();
      for (const item of parsed) {
        for (const key of Object.keys(item)) {
          if (seen.has(key)) continue;
          seen.add(key);
          columns.push(key);
        }
      }
      if (columns.length === 0) {
        return {
          kind: 'array',
          columns: ['value'],
          rows: parsed.map(item => ({ value: item })),
        };
      }
      const rows = parsed.map(item => {
        const row = {};
        for (const column of columns) {
          row[column] = item[column];
        }
        return row;
      });
      return { kind: 'array', columns, rows };
    }

    return {
      kind: 'array',
      columns: ['value'],
      rows: parsed.map(item => ({ value: item })),
    };
  }

  if (parsed && typeof parsed === 'object') {
    const columns = ['key', 'value'];
    const rows = Object.keys(parsed).sort().map(key => ({ key, value: parsed[key] }));
    return { kind: 'object', columns, rows };
  }

  return {
    kind: 'value',
    columns: ['value'],
    rows: [{ value: parsed }],
  };
}

function setEditorHTML(html, node) {
  const area = document.getElementById('editor-area');
  if (!area) return;
  if (node) {
    area.replaceWith(node);
  } else {
    const div = document.createElement('div');
    div.innerHTML = html;
    const child = div.firstChild;
    child.id = 'editor-area';
    area.replaceWith(child);
  }
}

// ── Toolbar actions ──
function wireEditorToolbar() {
  if (editorToolbarWired) return;
  editorToolbarWired = true;

  document.addEventListener('click', async e => {
    if (e.target.id === 'btn-save') await saveCurrentFile();
    if (e.target.id === 'btn-json-toggle') await toggleJsonView();
  }, { once: false });
}

async function toggleJsonView() {
  if (activeFileKind !== 'json' || !activeFilePath) return;

  const btnSave = document.getElementById('btn-save');
  const btnToggle = document.getElementById('btn-json-toggle');

  if (btnToggle.textContent === 'Raw') {
    await showJsonRawView();
    btnToggle.textContent = 'Table';
    btnSave.style.display = editorDirty ? 'inline-block' : 'none';
    return;
  }

  const ta = document.getElementById('editor-area');
  if (ta && ta.tagName === 'TEXTAREA') {
    activeJsonText = ta.value;
  }
  if (!renderJsonFromText(activeJsonText)) {
    return;
  }
  btnToggle.textContent = 'Raw';
  btnSave.style.display = 'none';
}

async function saveCurrentFile() {
  if (!activeSkill || !activeFilePath) return;
  const ta = document.getElementById('editor-area');
  if (!ta || ta.tagName !== 'TEXTAREA') return;
  const body = ta.value;
  const resp = await fetch(
    '/api/skills/' + activeSkill.name + '/file?path=' + encodeURIComponent(activeFilePath),
    { method: 'PUT', body }
  );
  if (resp.ok) {
    editorDirty = false;
    activeJsonText = body;
    document.getElementById('btn-save').style.display = 'none';
    toast('Saved ✓');
  } else {
    toast('Save failed', true);
  }
}

// ── Upload (click or drag-drop) ──
function wireUploadInput() {
  document.addEventListener('change', async e => {
    if (e.target.id === 'upload-input') {
      for (const file of e.target.files) {
        await uploadFileToPath(file, activeFilePath || file.name);
      }
      e.target.value = '';
    }
  });
}

// uploadFileTo uploads a File object to an exact relative path in the active skill.
async function uploadFileTo(file, target) {
  const fd = new FormData();
  fd.append('file', file);
  const resp = await fetch(
    '/api/skills/' + activeSkill.name + '/file?path=' + encodeURIComponent(target),
    { method: 'POST', body: fd }
  );
  if (resp.ok) {
    toast('Uploaded ' + file.name + ' ✓');
    const img = document.querySelector('#editor-area img');
    if (img && activeFilePath === target) {
      img.src = img.src.split('&t=')[0] + '&t=' + Date.now();
    }
  } else {
    toast('Upload failed: ' + file.name, true);
  }
}

// uploadFileToPath kept for click-upload button (uploads next to active file)
async function uploadFileToPath(file, destPath) {
  const dir = destPath.includes('/') ? destPath.substring(0, destPath.lastIndexOf('/')) : '';
  const target = dir ? dir + '/' + file.name : file.name;
  await uploadFileTo(file, target);
  await loadTree(activeSkill.name);
}

// ── Global drag-drop overlay (files dropped outside a folder row) ──
function wireDragDrop() {
  const overlay = document.getElementById('drop-overlay');
  let dragCount = 0;

  // Only show overlay when dragging real files from OS (not internal tree DnD)
  document.addEventListener('dragenter', e => {
    if (!activeSkill?.installed) return;
    if (e.dataTransfer.types.includes('Files')) { dragCount++; overlay.classList.add('active'); }
  });
  document.addEventListener('dragleave', e => {
    if (e.dataTransfer.types.includes('Files')) {
      dragCount--;
      if (dragCount <= 0) { dragCount = 0; overlay.classList.remove('active'); }
    }
  });
  document.addEventListener('dragover', e => e.preventDefault());
  document.addEventListener('drop', async e => {
    // If the drop landed on a folder row it was already handled — skip
    if (e.target.closest('.tree-row[data-is-dir="1"]')) return;
    e.preventDefault();
    dragCount = 0;
    overlay.classList.remove('active');
    if (!activeSkill || !e.dataTransfer.files.length) return;

    // Upload to selectedDir (last focused folder) or skill root
    const dir = selectedDir || '';
    for (const file of e.dataTransfer.files) {
      const target = dir ? dir + '/' + file.name : file.name;
      await uploadFileTo(file, target);
    }
    await loadTree(activeSkill.name);
  });
}

// ── Helpers ──
async function apiFetch(url) {
  try {
    const r = await fetch(url);
    if (!r.ok) return null;
    return r.json();
  } catch { return null; }
}

let toastTimer;
function toast(msg, isErr) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast show ' + (isErr ? 'err' : 'ok');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), 2500);
}

// ═══════════════════════════════════════════
// DB VIEWER
// ═══════════════════════════════════════════
let dbState = { file: null, table: null, tables: [], columns: [], rows: [], offset: 0, limit: 100 };

async function openDBViewer(relPath) {
  dbState = { file: relPath, table: null, tables: [], columns: [], rows: [], offset: 0, limit: 100 };
  setEditorHTML('<div class="db-viewer" id="editor-area"><div class="editor-empty">Loading…</div></div>');

  const tables = await dbFetch('tables', { db: relPath });
  if (!tables) return;
  dbState.tables = tables;

  renderDBViewer();
  if (tables.length > 0) await selectDBTable(tables[0]);
}

function renderDBViewer() {
  const area = document.getElementById('editor-area');
  if (!area) return;

  const tabsHTML = dbState.tables.map(t =>
    '<button class="db-tab' + (t === dbState.table ? ' active' : '') + '" data-table="' + t + '">' + t + '</button>'
  ).join('') + '<button class="db-tab db-tab-sql' + (dbState.table === '__sql__' ? ' active' : '') + '" data-table="__sql__">SQL ›</button>';

  area.innerHTML =
    '<div class="db-tabs" id="db-tabs">' + tabsHTML + '</div>' +
    '<div class="db-content" id="db-content"></div>';

  area.querySelectorAll('.db-tab').forEach(btn => {
    btn.addEventListener('click', () => {
      if (btn.dataset.table === '__sql__') {
        dbState.table = '__sql__';
        renderDBTabs();
        renderSQLPane();
      } else {
        selectDBTable(btn.dataset.table);
      }
    });
  });
}

function renderDBTabs() {
  document.querySelectorAll('.db-tab').forEach(b => {
    b.classList.toggle('active', b.dataset.table === dbState.table);
  });
}

async function selectDBTable(tableName) {
  dbState.table = tableName;
  dbState.offset = 0;
  renderDBTabs();

  const cols = await dbFetch('columns', { db: dbState.file, table: tableName });
  if (!cols) return;
  dbState.columns = cols;

  await loadTableRows();
}

async function loadTableRows() {
  const { file, table, limit, offset } = dbState;
  const sql = 'SELECT * FROM "' + table + '" LIMIT ' + limit + ' OFFSET ' + offset;
  const res = await dbQuery(file, sql);
  if (!res) return;
  dbState.rows = Array.isArray(res) ? res : [];
  renderTableView();
}

function renderTableView() {
  const content = document.getElementById('db-content');
  if (!content) return;
  const { columns, rows, table, offset } = dbState;

  const pkCol = columns.find(c => c.pk > 0);
  const pkName = pkCol ? pkCol.name : 'id';

  const colNames = columns.map(c => c.name);

  const thead = '<tr>' +
    colNames.map(c => '<th>' + escH(c) + '</th>').join('') +
    '<th>Actions</th></tr>';

  const tbody = rows.map((row, i) => {
    const cells = colNames.map(c => {
      const val = row[c];
      if (val === null || val === undefined) return '<td class="null">NULL</td>';
      const str = String(val);
      return '<td title="' + escH(str) + '">' + escH(str.length > 80 ? str.slice(0,80)+'…' : str) + '</td>';
    }).join('');
    const pk = row[pkName];
    return '<tr>' + cells +
      '<td class="act">' +
        '<button class="row-btn" title="Edit" data-idx="' + i + '">✏️</button>' +
        '<button class="row-btn del" title="Delete" data-pk="' + escH(String(pk)) + '" data-pkname="' + escH(pkName) + '">🗑</button>' +
      '</td></tr>';
  }).join('') || '<tr><td colspan="' + (colNames.length+1) + '" style="padding:20px;color:#52525b;text-align:center">No rows</td></tr>';

  content.innerHTML =
    '<div class="db-toolbar">' +
      '<button class="btn btn-primary" id="btn-add-row">＋ Add row</button>' +
      '<span class="db-count">' + rows.length + ' rows' + (offset > 0 ? ' (offset ' + offset + ')' : '') + '</span>' +
      (offset > 0 ? '<button class="btn" id="btn-prev">‹ Prev</button>' : '') +
      (rows.length === dbState.limit ? '<button class="btn" id="btn-next">Next ›</button>' : '') +
    '</div>' +
    '<div class="db-table-wrap"><table class="db-table"><thead>' + thead + '</thead><tbody>' + tbody + '</tbody></table></div>';

  content.querySelector('#btn-add-row')?.addEventListener('click', () => showRowModal(null));
  content.querySelector('#btn-prev')?.addEventListener('click', () => { dbState.offset -= dbState.limit; loadTableRows(); });
  content.querySelector('#btn-next')?.addEventListener('click', () => { dbState.offset += dbState.limit; loadTableRows(); });

  content.querySelectorAll('.row-btn[data-idx]').forEach(btn => {
    btn.addEventListener('click', () => showRowModal(rows[+btn.dataset.idx]));
  });
  content.querySelectorAll('.row-btn.del').forEach(btn => {
    btn.addEventListener('click', () => deleteRow(btn.dataset.pkname, btn.dataset.pk));
  });
}

function renderSQLPane() {
  const content = document.getElementById('db-content');
  if (!content) return;
  content.innerHTML =
    '<div class="sql-pane">' +
      '<div class="db-toolbar">' +
        '<button class="btn btn-primary" id="btn-run-sql">▶ Run</button>' +
        '<span style="font-size:11px;color:#52525b;margin-left:8px">Ctrl+Enter to run</span>' +
      '</div>' +
      '<textarea id="sql-input" placeholder="SELECT * FROM listings LIMIT 10;"></textarea>' +
      '<div class="sql-result" id="sql-result"><div class="sql-msg">Results will appear here</div></div>' +
    '</div>';

  const input = content.querySelector('#sql-input');
  content.querySelector('#btn-run-sql').addEventListener('click', () => runSQL());
  input.addEventListener('keydown', e => { if (e.ctrlKey && e.key === 'Enter') runSQL(); });
}

async function runSQL() {
  const input = document.getElementById('sql-input');
  if (!input) return;
  const sql = input.value.trim();
  if (!sql) return;
  const result = document.getElementById('sql-result');
  result.innerHTML = '<div class="sql-msg">Running…</div>';

  const res = await dbQuery(dbState.file, sql);
  if (res === null) { result.innerHTML = '<div class="sql-msg" style="color:#f87171">Query failed</div>'; return; }

  if (res && typeof res === 'object' && 'changes' in res) {
    result.innerHTML = '<div class="sql-msg" style="color:#4ade80">✓ ' + res.changes + ' row(s) affected</div>';
    if (dbState.table && dbState.table !== '__sql__') loadTableRows();
    return;
  }
  if (!Array.isArray(res) || res.length === 0) {
    result.innerHTML = '<div class="sql-msg">No results</div>';
    return;
  }
  const cols = Object.keys(res[0]);
  const thead = '<tr>' + cols.map(c => '<th>' + escH(c) + '</th>').join('') + '</tr>';
  const tbody = res.map(row =>
    '<tr>' + cols.map(c => '<td>' + escH(String(row[c] ?? 'NULL')) + '</td>').join('') + '</tr>'
  ).join('');
  result.innerHTML = '<table class="sql-table"><thead>' + thead + '</thead><tbody>' + tbody + '</tbody></table>';
}

// ── Row modal (add / edit) ──
function showRowModal(existingRow) {
  const { columns, file, table } = dbState;
  const isEdit = existingRow !== null;
  const title = isEdit ? 'Edit row' : 'Add row';

  const fields = columns
    .filter(c => !(isEdit === false && c.pk > 0))  // hide PK on add (autoincrement)
    .map(c => {
      const val = existingRow ? (existingRow[c.name] ?? '') : (c.dflt_value || '');
      return '<div class="field-row">' +
        '<label class="field-label">' + escH(c.name) + ' <span style="color:#3f3f46">(' + c.type + ')</span></label>' +
        '<input class="field-input" name="' + escH(c.name) + '" value="' + escH(String(val)) + '">' +
        '</div>';
    }).join('');

  const backdrop = document.createElement('div');
  backdrop.className = 'modal-backdrop';
  backdrop.innerHTML =
    '<div class="modal">' +
      '<div class="modal-header">' + title + '<button class="close-btn">✕</button></div>' +
      '<div class="modal-body">' + fields + '</div>' +
      '<div class="modal-footer">' +
        '<button class="btn" id="modal-cancel">Cancel</button>' +
        '<button class="btn btn-primary" id="modal-ok">' + (isEdit ? 'Save' : 'Insert') + '</button>' +
      '</div>' +
    '</div>';

  document.body.appendChild(backdrop);
  backdrop.querySelector('.close-btn').addEventListener('click', () => backdrop.remove());
  backdrop.querySelector('#modal-cancel').addEventListener('click', () => backdrop.remove());
  backdrop.querySelector('#modal-ok').addEventListener('click', async () => {
    const inputs = backdrop.querySelectorAll('.field-input');
    const values = {};
    inputs.forEach(inp => { values[inp.name] = inp.value; });

    let sql;
    if (isEdit) {
      const pkCol = columns.find(c => c.pk > 0);
      if (!pkCol) { toast('No primary key found', true); return; }
      const sets = Object.entries(values)
        .filter(([k]) => k !== pkCol.name)
        .map(([k, v]) => '"' + k + '" = ' + sqlVal(v)).join(', ');
      sql = 'UPDATE "' + table + '" SET ' + sets + ' WHERE "' + pkCol.name + '" = ' + sqlVal(String(existingRow[pkCol.name]));
    } else {
      const keys = Object.keys(values).map(k => '"' + k + '"').join(', ');
      const vals = Object.values(values).map(sqlVal).join(', ');
      sql = 'INSERT INTO "' + table + '" (' + keys + ') VALUES (' + vals + ')';
    }

    const res = await dbQuery(file, sql);
    if (res && 'changes' in res) {
      toast((isEdit ? 'Updated' : 'Inserted') + ' ✓');
      backdrop.remove();
      await loadTableRows();
    } else {
      toast('Operation failed', true);
    }
  });
}

async function deleteRow(pkName, pkValue) {
  if (!confirm('Delete this row?')) return;
  const sql = 'DELETE FROM "' + dbState.table + '" WHERE "' + pkName + '" = ' + sqlVal(pkValue);
  const res = await dbQuery(dbState.file, sql);
  if (res && 'changes' in res) {
    toast('Deleted ✓');
    await loadTableRows();
  } else {
    toast('Delete failed', true);
  }
}

// ── DB API helpers ──
async function dbFetch(action, params) {
  const qs = Object.entries(params).map(([k,v]) => k + '=' + encodeURIComponent(v)).join('&');
  const r = await fetch('/api/skills/' + activeSkill.name + '/db/' + action + '?' + qs);
  if (!r.ok) { toast(await r.text(), true); return null; }
  return r.json();
}

async function dbQuery(dbFile, sql) {
  const r = await fetch('/api/skills/' + activeSkill.name + '/db/query', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ db: dbFile, sql })
  });
  if (!r.ok) { toast(await r.text(), true); return null; }
  const data = await r.json();
  if (data.error) { toast(data.error, true); return null; }
  return data.result;
}

function sqlVal(v) {
  if (v === '' || v === 'NULL' || v === null || v === undefined) return 'NULL';
  if (!isNaN(v) && v.trim() !== '') return v;
  return "'" + v.replace(/'/g, "''") + "'";
}

function escH(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ── end DB VIEWER ──

// ═══════════════════════════════════════════
// FILE MANAGER — context menu, rename, dnd, mkdir
// ═══════════════════════════════════════════

// selectedDir: the currently "focused" directory for new file/folder
let selectedDir = '';
// selectedPaths: Set of paths selected for multi-delete
let selectedPaths = new Set();

function togglePathSelection(path, row) {
  if (selectedPaths.has(path)) {
    selectedPaths.delete(path);
    row.classList.remove('multi-selected');
  } else {
    selectedPaths.add(path);
    row.classList.add('multi-selected');
  }
  updateDeleteSelectionBtn();
}

function clearSelection() {
  selectedPaths.clear();
  document.querySelectorAll('.tree-row.multi-selected').forEach(r => r.classList.remove('multi-selected'));
  updateDeleteSelectionBtn();
}

function updateDeleteSelectionBtn() {
  let btn = document.getElementById('btn-delete-selected');
  if (!btn) return;
  if (selectedPaths.size > 0) {
    btn.style.display = '';
    btn.textContent = '🗑 Delete (' + selectedPaths.size + ')';
  } else {
    btn.style.display = 'none';
  }
}

// Override renderTreeNode to include dnd + context menu
function renderTreeNode(node, depth) {
  const wrap = document.createElement('div');
  wrap.className = 'tree-node';
  wrap.dataset.path = node.path;
  wrap.dataset.isDir = node.is_dir ? '1' : '0';

  const row = document.createElement('div');
  row.className = 'tree-row';
  row.style.paddingLeft = (8 + depth * 14) + 'px';
  row.dataset.path = node.path;
  row.dataset.isDir = node.is_dir ? '1' : '0';
  row.draggable = true;

  const icon = document.createElement('span');
  icon.className = 'tree-icon';
  const nameSpan = document.createElement('span');
  nameSpan.className = 'tree-name';
  nameSpan.textContent = node.name;

  row.appendChild(icon);
  row.appendChild(nameSpan);

  // Right-click → context menu
  row.addEventListener('contextmenu', e => {
    e.preventDefault();
    showContextMenu(e.clientX, e.clientY, node, row);
  });

  // Drag events
  row.addEventListener('dragstart', e => {
    e.dataTransfer.setData('text/plain', node.path);
    e.dataTransfer.effectAllowed = 'move';
    setTimeout(() => row.classList.add('dragging'), 0);
  });
  row.addEventListener('dragend', () => row.classList.remove('dragging'));

  if (node.is_dir) {
    icon.textContent = '📁';
    const children = document.createElement('div');
    children.className = 'tree-children';
    children.style.display = 'none';
    let open = false;

    row.addEventListener('click', e => {
      if (e.ctrlKey || e.metaKey) { togglePathSelection(node.path, row); return; }
      clearSelection();
      open = !open;
      icon.textContent = open ? '📂' : '📁';
      children.style.display = open ? 'block' : 'none';
      selectedDir = node.path; // track focused dir
    });

    // Drop target — use wrap so dragging into open children doesn't fire dragleave
    let dragCounter = 0;
    wrap.addEventListener('dragover', e => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = 'move'; });
    wrap.addEventListener('dragenter', e => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter++;
      row.classList.add('drag-over');
    });
    wrap.addEventListener('dragleave', e => {
      e.stopPropagation();
      dragCounter--;
      if (dragCounter <= 0) { dragCounter = 0; row.classList.remove('drag-over'); }
    });
    wrap.addEventListener('drop', async e => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter = 0;
      row.classList.remove('drag-over');

      // ── External files dropped from OS ──
      if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        for (const file of e.dataTransfer.files) {
          const target = node.path ? node.path + '/' + file.name : file.name;
          await uploadFileTo(file, target);
        }
        await loadTree(activeSkill.name);
        return;
      }

      // ── Internal tree DnD ──
      const srcPath = e.dataTransfer.getData('text/plain');
      if (srcPath && srcPath !== node.path && !node.path.startsWith(srcPath + '/')) {
        const srcName = srcPath.split('/').pop();
        const destPath = node.path ? node.path + '/' + srcName : srcName;
        moveNode(srcPath, destPath);
      }
    });

    wrap.appendChild(row);
    if (node.children) {
      node.children.forEach(child => children.appendChild(renderTreeNode(child, depth + 1)));
    }
    wrap.appendChild(children);
  } else {
    icon.textContent = fileIcon(node.name);
    row.addEventListener('click', e => {
      if (e.ctrlKey || e.metaKey) { togglePathSelection(node.path, row); return; }
      clearSelection();
      // track parent dir for new file context
      const parts = node.path.split('/');
      selectedDir = parts.length > 1 ? parts.slice(0,-1).join('/') : '';
      openFile(node.path, row);
    });
    wrap.appendChild(row);
  }
  return wrap;
}

// ── Context menu ──
function showContextMenu(x, y, node, row) {
  removeContextMenu();
  const menu = document.createElement('div');
  menu.className = 'ctx-menu';
  menu.id = 'ctx-menu';

  const items = [];
  if (node.is_dir) {
    items.push({ icon: '📄', label: 'New file', action: () => promptNewFile(node.path) });
    items.push({ icon: '📁', label: 'New folder', action: () => promptNewFolder(node.path) });
    items.push({ icon: '⬆', label: 'Upload files…', action: () => triggerUploadToDir(node.path) });
    items.push({ sep: true });
    items.push({ icon: '📋', label: 'Copy', action: () => clipboardCopy(node) });
    if (fsClipboard) {
      items.push({ icon: '📌', label: 'Paste into "' + node.name + '"', action: () => clipboardPaste(node.path) });
    }
    items.push({ sep: true });
  }
  items.push({ icon: '✏️', label: 'Rename', action: () => startInlineRename(row, node) });
  items.push({ sep: true });
  items.push({ icon: '🗑', label: 'Delete', action: () => deleteNode(node), danger: true });

  items.forEach(item => {
    if (item.sep) {
      const sep = document.createElement('div');
      sep.className = 'ctx-sep';
      menu.appendChild(sep);
      return;
    }
    const el = document.createElement('div');
    el.className = 'ctx-item' + (item.danger ? ' danger' : '');
    el.innerHTML = '<span>' + item.icon + '</span>' + item.label;
    el.addEventListener('click', () => { removeContextMenu(); item.action(); });
    menu.appendChild(el);
  });

  // Position
  menu.style.left = x + 'px';
  menu.style.top = y + 'px';
  document.body.appendChild(menu);

  // Adjust if out of viewport
  const rect = menu.getBoundingClientRect();
  if (rect.right > window.innerWidth) menu.style.left = (x - rect.width) + 'px';
  if (rect.bottom > window.innerHeight) menu.style.top = (y - rect.height) + 'px';
}

function removeContextMenu() {
  document.getElementById('ctx-menu')?.remove();
}
document.addEventListener('click', removeContextMenu);
document.addEventListener('keydown', e => { if (e.key === 'Escape') removeContextMenu(); });

// ── Inline rename ──
function startInlineRename(row, node) {
  const nameSpan = row.querySelector('.tree-name');
  const oldName = node.name;
  const input = document.createElement('input');
  input.className = 'tree-rename-input';
  input.value = oldName;
  nameSpan.replaceWith(input);
  input.focus();
  input.select();

  const commit = async () => {
    const newName = input.value.trim();
    if (!newName || newName === oldName) {
      input.replaceWith(nameSpan);
      return;
    }
    const dir = node.path.includes('/')
      ? node.path.substring(0, node.path.lastIndexOf('/'))
      : '';
    const newPath = dir ? dir + '/' + newName : newName;
    const ok = await fsOp('rename', { from: node.path, to: newPath });
    if (ok) {
      toast('Renamed ✓');
      await loadTree(activeSkill.name);
    } else {
      input.replaceWith(nameSpan);
    }
  };

  input.addEventListener('blur', commit);
  input.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
    if (e.key === 'Escape') { input.value = oldName; input.blur(); }
  });
}

// ── Move (drag-drop) ──
async function moveNode(srcPath, destPath) {
  if (srcPath === destPath) return;
  const ok = await fsOp('rename', { from: srcPath, to: destPath });
  if (ok) {
    toast('Moved ✓');
    await loadTree(activeSkill.name);
    if (activeFilePath === srcPath) activeFilePath = destPath;
  }
}

// ── Delete ──
async function deleteNode(node) {
  const label = node.is_dir ? 'folder "' + node.name + '" and all its contents' : 'file "' + node.name + '"';
  if (!confirm('Delete ' + label + '?')) return;
  const ok = await fsOp('delete', { path: node.path });
  if (ok) {
    toast('Deleted ✓');
    if (activeFilePath && (activeFilePath === node.path || activeFilePath.startsWith(node.path + '/'))) {
      activeFilePath = null;
      document.getElementById('editor-area')?.replaceWith(
        Object.assign(document.createElement('div'), { id: 'editor-area', className: 'editor-empty', textContent: 'Select a file to view or edit' })
      );
    }
    await loadTree(activeSkill.name);
  }
}

// ── Clipboard copy/paste ──
let fsClipboard = null; // { node }

function clipboardCopy(node) {
  fsClipboard = { node };
  toast('Copied "' + node.name + '" — right-click a folder to Paste');
}

async function clipboardPaste(destDir) {
  if (!fsClipboard) return;
  const src = fsClipboard.node;
  const destPath = destDir ? destDir + '/' + src.name : src.name;
  const ok = await fsOp('copy', { from: src.path, to: destPath });
  if (ok) {
    toast('Pasted "' + src.name + '" into "' + destDir + '" ✓');
    await loadTree(activeSkill.name);
  }
}

async function deleteSelected() {
  if (selectedPaths.size === 0) return;
  const paths = Array.from(selectedPaths);
  if (!confirm('Delete ' + paths.length + ' item(s)?')) return;
  const ok = await fsOp('delete', { paths });
  if (ok) {
    toast('Deleted ' + paths.length + ' item(s) ✓');
    if (activeFilePath && paths.some(p => activeFilePath === p || activeFilePath.startsWith(p + '/'))) {
      activeFilePath = null;
      document.getElementById('editor-area')?.replaceWith(
        Object.assign(document.createElement('div'), { id: 'editor-area', className: 'editor-empty', textContent: 'Select a file to view or edit' })
      );
    }
    clearSelection();
    await loadTree(activeSkill.name);
  }
}

// ── New file / folder prompts ──
function promptNewFile(inDir) {
  showInputModal('New file', 'Filename (e.g. notes.md)', '', async name => {
    if (!name) return;
    const path = inDir ? inDir + '/' + name : name;
    const ok = await fsOp('touch', { path });
    if (ok) { toast('Created ✓'); await loadTree(activeSkill.name); }
  });
}

function promptNewFolder(inDir) {
  showInputModal('New folder', 'Folder name', '', async name => {
    if (!name) return;
    const path = inDir ? inDir + '/' + name : name;
    const ok = await fsOp('mkdir', { path });
    if (ok) { toast('Created ✓'); await loadTree(activeSkill.name); }
  });
}

// Wire toolbar buttons
document.addEventListener('click', e => {
  if (e.target.id === 'btn-new-file' || e.target.closest('#btn-new-file')) {
    promptNewFile(selectedDir);
  }
  if (e.target.id === 'btn-new-folder' || e.target.closest('#btn-new-folder')) {
    promptNewFolder(selectedDir);
  }
  if (e.target.id === 'btn-delete-selected' || e.target.closest('#btn-delete-selected')) {
    deleteSelected();
  }
});

// ── Simple input modal ──
function showInputModal(title, placeholder, defaultVal, onConfirm) {
  const backdrop = document.createElement('div');
  backdrop.className = 'modal-backdrop';
  backdrop.innerHTML =
    '<div class="modal" style="width:360px">' +
      '<div class="modal-header">' + title + '<button class="close-btn">✕</button></div>' +
      '<div class="modal-body">' +
        '<input class="field-input" id="input-modal-val" placeholder="' + placeholder + '" value="' + escH(defaultVal) + '" style="width:100%">' +
      '</div>' +
      '<div class="modal-footer">' +
        '<button class="btn" id="input-modal-cancel">Cancel</button>' +
        '<button class="btn btn-primary" id="input-modal-ok">OK</button>' +
      '</div>' +
    '</div>';
  document.body.appendChild(backdrop);

  const input = backdrop.querySelector('#input-modal-val');
  input.focus(); input.select();

  const confirm = () => { const v = input.value.trim(); backdrop.remove(); onConfirm(v); };
  backdrop.querySelector('.close-btn').addEventListener('click', () => backdrop.remove());
  backdrop.querySelector('#input-modal-cancel').addEventListener('click', () => backdrop.remove());
  backdrop.querySelector('#input-modal-ok').addEventListener('click', confirm);
  input.addEventListener('keydown', e => { if (e.key === 'Enter') confirm(); if (e.key === 'Escape') backdrop.remove(); });
}

// ── Upload from file picker ──

// triggerUploadToDir opens native file picker and uploads to a specific dir
function triggerUploadToDir(dir) {
  const input = document.createElement('input');
  input.type = 'file';
  input.multiple = true;
  input.addEventListener('change', async () => {
    for (const file of input.files) {
      const target = dir ? dir + '/' + file.name : file.name;
      await uploadFileTo(file, target);
    }
    await loadTree(activeSkill.name);
  });
  input.click();
}

// Toolbar ⬆ button — uploads to selectedDir
document.addEventListener('change', async e => {
  if (e.target.id !== 'toolbar-upload-input') return;
  const dir = selectedDir || '';
  for (const file of e.target.files) {
    const target = dir ? dir + '/' + file.name : file.name;
    await uploadFileTo(file, target);
  }
  e.target.value = '';
  await loadTree(activeSkill.name);
});

// ── FS API helper ──
async function fsOp(action, body) {
  const r = await fetch('/api/skills/' + activeSkill.name + '/fs/' + action, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!r.ok) { toast(await r.text(), true); return false; }
  return true;
}

// ── end FILE MANAGER ──

wireUploadInput();
boot();
</script>
</body>
</html>`
