package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func init() {
	Register(&Gmail{})
}

// Gmail implements OAuth for Gmail via gog CLI.
// After obtaining tokens, it writes credential.json and configures gog.
type Gmail struct{}

func (g *Gmail) Name() string    { return "gmail" }
func (g *Gmail) Display() string { return "Gmail (Google OAuth2)" }

// gmailScopes includes userinfo.email so we can auto-detect the account email after OAuth.
const gmailScopes = "https://mail.google.com/ https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/drive https://www.googleapis.com/auth/contacts https://www.googleapis.com/auth/spreadsheets https://www.googleapis.com/auth/documents https://www.googleapis.com/auth/userinfo.email"

func (g *Gmail) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s            ║\n", g.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()
	fmt.Println("  Đang mở Google Cloud Console để tạo OAuth credentials...")
	fmt.Println("  1. Chọn project (hoặc tạo mới)")
	fmt.Println("  2. Create Credentials → OAuth client ID → Desktop app")
	fmt.Println("  3. Copy Client ID và Client Secret dán vào bên dưới")
	fmt.Println()
	fmt.Println("  Redirect URI cần thêm vào OAuth app:")
	fmt.Println("  → http://localhost:9876/callback")
	fmt.Println()
	OpenBrowser("https://console.cloud.google.com/apis/credentials")

	clientID := PromptInput("  Google Client ID")
	clientSecret := PromptInput("  Google Client Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(gmailScopes),
	)

	fmt.Println()
	fmt.Println("  Opening browser for Google authorization...")
	fmt.Println("  If browser doesn't open, visit:")
	fmt.Printf("  %s\n\n", authURL)

	OpenBrowser(authURL)

	code, err := WaitForCallback()
	if err != nil {
		return nil, err
	}

	tokens, err := exchangeGoogleCode(code, clientID, clientSecret, redirectURI)
	if err != nil {
		return nil, err
	}

	// Auto-detect email from Google userinfo — no need to ask the user.
	email, err := fetchGoogleEmail(tokens["access_token"])
	if err != nil {
		fmt.Println("  Không lấy được email tự động.")
		email = PromptInput("  Gmail address (e.g. you@gmail.com)")
	} else {
		fmt.Printf("  ✓ Tài khoản: %s\n", email)
	}

	tokens["google_client_id"] = clientID
	tokens["google_client_secret"] = clientSecret
	tokens["gmail_account"] = email
	return tokens, nil
}

// fetchGoogleEmail calls the Google userinfo endpoint to get the authenticated account email.
func fetchGoogleEmail(accessToken string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var info struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil || info.Email == "" {
		return "", fmt.Errorf("no email in response")
	}
	return info.Email, nil
}
