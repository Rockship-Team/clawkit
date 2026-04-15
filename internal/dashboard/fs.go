package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleFS routes /api/skills/{name}/fs/{action}
func handleFS(w http.ResponseWriter, r *http.Request, skillDir, action string) {
	switch action {
	case "mkdir":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct{ Path string `json:"path"` }
		if err := readJSON(r, &req); err != nil || req.Path == "" {
			http.Error(w, "body: {path}", http.StatusBadRequest)
			return
		}
		abs := safePath(skillDir, req.Path)
		if abs == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(abs, 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})

	case "touch":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct{ Path string `json:"path"` }
		if err := readJSON(r, &req); err != nil || req.Path == "" {
			http.Error(w, "body: {path}", http.StatusBadRequest)
			return
		}
		abs := safePath(skillDir, req.Path)
		if abs == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f, err := os.OpenFile(abs, os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		f.Close()
		jsonOK(w, map[string]string{"status": "ok"})

	case "rename":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := readJSON(r, &req); err != nil || req.From == "" || req.To == "" {
			http.Error(w, "body: {from, to}", http.StatusBadRequest)
			return
		}
		absFrom := safePath(skillDir, req.From)
		absTo := safePath(skillDir, req.To)
		if absFrom == "" || absTo == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(absTo), 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.Rename(absFrom, absTo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})

	case "delete":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Path  string   `json:"path"`
			Paths []string `json:"paths"`
		}
		if err := readJSON(r, &req); err != nil {
			http.Error(w, "body: {path} or {paths}", http.StatusBadRequest)
			return
		}
		targets := req.Paths
		if req.Path != "" {
			targets = append(targets, req.Path)
		}
		if len(targets) == 0 {
			http.Error(w, "body: {path} or {paths}", http.StatusBadRequest)
			return
		}
		for _, p := range targets {
			abs := safePath(skillDir, p)
			if abs == "" {
				http.Error(w, "invalid path: "+p, http.StatusBadRequest)
				return
			}
			if err := os.RemoveAll(abs); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		jsonOK(w, map[string]string{"status": "ok"})

	case "copy":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := readJSON(r, &req); err != nil || req.From == "" || req.To == "" {
			http.Error(w, "body: {from, to}", http.StatusBadRequest)
			return
		}
		absFrom := safePath(skillDir, req.From)
		absTo := safePath(skillDir, req.To)
		if absFrom == "" || absTo == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if err := copyAll(absFrom, absTo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})

	default:
		http.NotFound(w, r)
	}
}

// copyAll recursively copies src to dst.
func copyAll(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyAll(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// safePath validates relPath has no traversal and returns absolute path.
// Returns "" if invalid.
func safePath(skillDir, relPath string) string {
	// Clean the path to remove any .. components
	clean := filepath.Join(skillDir, filepath.FromSlash(relPath))
	if !strings.HasPrefix(clean, skillDir+string(os.PathSeparator)) && clean != skillDir {
		return ""
	}
	return clean
}

func readJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
