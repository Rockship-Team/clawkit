package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	Register(&ZaloOA{})
}

// ZaloOA implements OAuth for Zalo Official Account.
// Unlike Zalo Personal (QR login), OA uses standard OAuth redirect flow
// because it requires developer App ID and App Secret.
type ZaloOA struct{}

func (z *ZaloOA) Name() string    { return "zalo_oa" }
func (z *ZaloOA) Display() string { return "Zalo Official Account" }

func (z *ZaloOA) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Zalo Official Account               ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	appID := PromptInput("  Zalo OA App ID")
	appSecret := PromptInput("  Zalo OA App Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/oa/permission?app_id=%s&redirect_uri=%s",
		appID, url.QueryEscape(redirectURI),
	)

	fmt.Println()
	fmt.Println("  Opening browser for Zalo OA authorization...")
	fmt.Println("  If browser doesn't open, visit:")
	fmt.Printf("  %s\n\n", authURL)

	OpenBrowser(authURL)

	code, err := WaitForCallback()
	if err != nil {
		return nil, err
	}

	tokens, err := exchangeZaloOACode(code, appID, appSecret)
	if err != nil {
		return nil, err
	}

	tokens["zalo_oa_app_id"] = appID
	tokens["zalo_oa_app_secret"] = appSecret
	return tokens, nil
}

// exchangeZaloOACode exchanges an authorization code for OA access/refresh tokens.
func exchangeZaloOACode(code, appID, appSecret string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data := url.Values{
		"code":       {code},
		"app_id":     {appID},
		"grant_type": {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth.zaloapp.com/v4/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", appSecret)

	resp, err := http.DefaultClient.Do(req)
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
