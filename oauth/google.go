package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func init() {
	Register(&Google{})
}

// Google implements OAuth for Google services (Gmail, Sheets, Calendar, etc.)
type Google struct{}

func (g *Google) Name() string    { return "google" }
func (g *Google) Display() string { return "Google Account" }

func (g *Google) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s                ║\n", g.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()

	clientID := PromptInput("  Google Client ID")
	clientSecret := PromptInput("  Google Client Secret")
	scopes := PromptInput("  Scopes (e.g., https://www.googleapis.com/auth/gmail.send)")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
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
	return tokens, nil
}

func exchangeGoogleCode(code, clientID, clientSecret, redirectURI string) (map[string]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := client.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)

	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("google auth error: %s - %v", errMsg, result["error_description"])
	}

	tokens := map[string]string{}
	if at, ok := result["access_token"].(string); ok {
		tokens["access_token"] = at
	}
	if rt, ok := result["refresh_token"].(string); ok {
		tokens["refresh_token"] = rt
	}
	return tokens, nil
}
