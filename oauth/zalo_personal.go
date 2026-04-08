package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	Register(&ZaloPersonal{})
}

// ZaloPersonal implements OAuth for Zalo Personal Account.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s      ║\n", z.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
	fmt.Println()

	appID := PromptInput("  Zalo App ID")
	appSecret := PromptInput("  Zalo App Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/permission?app_id=%s&redirect_uri=%s&state=clawkit",
		appID, url.QueryEscape(redirectURI),
	)

	fmt.Println()
	fmt.Println("  Opening browser for Zalo login...")
	fmt.Println("  If browser doesn't open, visit:")
	fmt.Printf("  %s\n\n", authURL)

	OpenBrowser(authURL)

	code, err := WaitForCallback()
	if err != nil {
		return nil, err
	}

	tokens, err := exchangeZaloCode(code, appID, appSecret)
	if err != nil {
		return nil, err
	}

	tokens["zalo_app_id"] = appID
	tokens["zalo_app_secret"] = appSecret
	return tokens, nil
}

// exchangeZaloCode exchanges an authorization code for access/refresh tokens.
// Shared between Zalo Personal and Zalo OA.
func exchangeZaloCode(code, appID, appSecret string) (map[string]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	data := url.Values{
		"code":       {code},
		"app_id":     {appID},
		"grant_type": {"authorization_code"},
	}

	req, err := http.NewRequest("POST", "https://oauth.zaloapp.com/v4/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", appSecret)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("zalo auth error: %s - %v", errMsg, result["error_description"])
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
