package oauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// (browser OAuth removed — gog auth add handles the browser flow itself,
// so clawkit only needs to collect client credentials and the account email.)

func init() {
	Register(&Gmail{})
}

// Gmail implements OAuth for Gmail via gog CLI.
// Flow: auto-detect client_secret_*.json (or prompt path / manual input) →
// prompt for account email → return. The actual browser OAuth is performed
// later by `gog auth add`, so the user only sees one browser window.
type Gmail struct{}

func (g *Gmail) Name() string    { return "gmail" }
func (g *Gmail) Display() string { return "Gmail (Google OAuth2)" }

// googleCredentialFile mirrors the structure of a downloaded client_secret_*.json.
type googleCredentialFile struct {
	Installed struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
	Web struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"web"`
}

func (g *Gmail) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s            ║\n", g.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()

	// Step 1: collect client_id / client_secret.
	credPath := findLatestGoogleCredential()
	if credPath != "" {
		fmt.Printf("  ✓ Tìm thấy credential file: %s\n", credPath)
		use := PromptInput("  Dùng file này? [Y/n]")
		if strings.EqualFold(strings.TrimSpace(use), "n") {
			credPath = ""
		}
	}

	var clientID, clientSecret string

	if credPath == "" {
		fmt.Println("  Chưa tìm thấy credential file trong Downloads.")
		fmt.Println("  1. Tạo Desktop OAuth client tại Google Cloud Console")
		fmt.Println("  2. Download file JSON (client_secret_*.json)")
		fmt.Println()
		fmt.Println("  → Nhập đường dẫn tới file JSON (để trống nếu muốn nhập Client ID/Secret thủ công)")
		OpenBrowser("https://console.cloud.google.com/apis/credentials")
		credPath = strings.TrimSpace(PromptInput("  Đường dẫn tới client_secret_*.json"))
		if strings.HasPrefix(credPath, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				credPath = filepath.Join(home, strings.TrimPrefix(credPath, "~"))
			}
		}
	}

	if credPath != "" {
		id, secret, err := parseGoogleCredential(credPath)
		if err != nil {
			return nil, fmt.Errorf("đọc credential file: %w", err)
		}
		clientID, clientSecret = id, secret
	} else {
		fmt.Println()
		fmt.Println("  Nhập credentials thủ công:")
		clientID = strings.TrimSpace(PromptInput("  Google Client ID"))
		clientSecret = strings.TrimSpace(PromptInput("  Google Client Secret"))
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("client_id và client_secret là bắt buộc")
		}
	}
	fmt.Printf("  ✓ Client ID: %s\n", maskMiddle(clientID))

	// Step 2: prompt for the Gmail account email. The browser OAuth itself
	// runs later via `gog auth add <email>`, keeping the flow to a single
	// browser window.
	fmt.Println()
	email := strings.TrimSpace(PromptInput("  Gmail address (e.g. you@gmail.com)"))
	if email == "" {
		return nil, fmt.Errorf("gmail address là bắt buộc")
	}

	return map[string]string{
		"google_client_id":     clientID,
		"google_client_secret": clientSecret,
		"gmail_account":        email,
		"credential_file":      credPath,
	}, nil
}

// findLatestGoogleCredential scans common Downloads locations (cross-platform)
// for client_secret_*.json files and returns the most recently modified one,
// or "" if none found.
func findLatestGoogleCredential() string {
	dirs := downloadDirs()
	var all []string
	for _, d := range dirs {
		matches, err := filepath.Glob(filepath.Join(d, "client_secret_*.json"))
		if err != nil {
			continue
		}
		all = append(all, matches...)
	}
	if len(all) == 0 {
		return ""
	}
	sort.Slice(all, func(i, j int) bool {
		fi, err1 := os.Stat(all[i])
		fj, err2 := os.Stat(all[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return all[0]
}

// downloadDirs returns likely Downloads directory locations for the current OS.
// Covers: ~/Downloads (all platforms), $XDG_DOWNLOAD_DIR (Linux), and
// OneDrive-redirected ~/OneDrive/Downloads (Windows).
func downloadDirs() []string {
	var dirs []string
	home, err := os.UserHomeDir()
	if err == nil {
		dirs = append(dirs, filepath.Join(home, "Downloads"))
		dirs = append(dirs, filepath.Join(home, "OneDrive", "Downloads"))
	}
	if xdg := os.Getenv("XDG_DOWNLOAD_DIR"); xdg != "" {
		dirs = append(dirs, xdg)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// parseGoogleCredential reads a Google OAuth client JSON file and extracts
// the client_id and client_secret from either the "installed" or "web" key.
func parseGoogleCredential(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	var f googleCredentialFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", "", fmt.Errorf("parse JSON: %w", err)
	}
	if f.Installed.ClientID != "" {
		return f.Installed.ClientID, f.Installed.ClientSecret, nil
	}
	if f.Web.ClientID != "" {
		return f.Web.ClientID, f.Web.ClientSecret, nil
	}
	return "", "", fmt.Errorf("không tìm thấy client_id trong file (cần 'installed' hoặc 'web')")
}

// maskMiddle masks the middle of a string for display.
func maskMiddle(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:6] + "..." + s[len(s)-4:]
}
