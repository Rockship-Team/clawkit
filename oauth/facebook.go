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
	Register(&Facebook{})
}

// Facebook implements OAuth for Facebook (Pages, Messenger).
type Facebook struct{}

func (f *Facebook) Name() string    { return "facebook" }
func (f *Facebook) Display() string { return "Facebook Page" }

func (f *Facebook) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s                 ║\n", f.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()

	appID := PromptInput("  Facebook App ID")
	appSecret := PromptInput("  Facebook App Secret")
	scopes := PromptInput("  Permissions (e.g., pages_messaging,pages_read_engagement)")

	if scopes == "" {
		scopes = "pages_messaging,pages_read_engagement"
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://www.facebook.com/v19.0/dialog/oauth?client_id=%s&redirect_uri=%s&scope=%s&response_type=code",
		appID,
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
	)

	fmt.Println()
	fmt.Println("  Opening browser for Facebook authorization...")
	fmt.Println("  If browser doesn't open, visit:")
	fmt.Printf("  %s\n\n", authURL)

	OpenBrowser(authURL)

	code, err := WaitForCallback()
	if err != nil {
		return nil, err
	}

	tokens, err := exchangeFacebookCode(code, appID, appSecret, redirectURI)
	if err != nil {
		return nil, err
	}

	tokens["facebook_app_id"] = appID
	tokens["facebook_app_secret"] = appSecret
	return tokens, nil
}

func exchangeFacebookCode(code, appID, appSecret, redirectURI string) (map[string]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	tokenURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		appID, appSecret, url.QueryEscape(redirectURI), code,
	)

	resp, err := client.Get(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)

	if errMsg, ok := result["error"].(map[string]any); ok {
		return nil, fmt.Errorf("facebook auth error: %v", errMsg["message"])
	}

	tokens := map[string]string{}
	if at, ok := result["access_token"].(string); ok {
		tokens["access_token"] = at
	}
	return tokens, nil
}
