package oauth

import (
	"fmt"
	"net/url"
)

func init() {
	Register(&Gmail{})
}

// Gmail implements OAuth for Gmail via gog CLI.
// After obtaining tokens, it writes credential.json and configures gog.
type Gmail struct{}

func (g *Gmail) Name() string    { return "gmail" }
func (g *Gmail) Display() string { return "Gmail (Google OAuth2)" }

// gmailScopes are the default scopes for full Gmail + Calendar + Drive + Contacts + Sheets + Docs access.
const gmailScopes = "https://mail.google.com/ https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/drive https://www.googleapis.com/auth/contacts https://www.googleapis.com/auth/spreadsheets https://www.googleapis.com/auth/documents"

func (g *Gmail) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s            ║\n", g.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()
	fmt.Println("  Tạo OAuth2 credentials tại Google Cloud Console:")
	fmt.Println("  https://console.cloud.google.com/apis/credentials")
	fmt.Println()
	fmt.Println("  Redirect URI: http://localhost:9876/callback")
	fmt.Println()

	clientID := PromptInput("  Google Client ID")
	clientSecret := PromptInput("  Google Client Secret")
	email := PromptInput("  Gmail address (e.g. you@gmail.com)")

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

	tokens["google_client_id"] = clientID
	tokens["google_client_secret"] = clientSecret
	tokens["gmail_account"] = email
	return tokens, nil
}
