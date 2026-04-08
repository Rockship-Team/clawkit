package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const oauthCallbackPort = 9876

func runOAuthFlow(provider, skillDir string) error {
	switch provider {
	case "zalo_personal":
		return oauthZaloPersonal(skillDir)
	case "zalo_oa":
		return oauthZaloOA(skillDir)
	case "google_sheets", "gmail":
		return oauthGoogle(provider, skillDir)
	default:
		return fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// ── Zalo Personal OAuth ──

func oauthZaloPersonal(skillDir string) error {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Kết nối Zalo Personal Account      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	// For Zalo Personal, we need app_id from the skill's manifest
	// or prompt the user
	appID := promptInput("  Zalo App ID")
	appSecret := promptInput("  Zalo App Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", oauthCallbackPort)

	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/permission?app_id=%s&redirect_uri=%s&state=rockship",
		appID, url.QueryEscape(redirectURI),
	)

	fmt.Println()
	info("Opening browser for Zalo login...")
	fmt.Println("  If browser doesn't open, visit:")
	fmt.Printf("  %s\n", authURL)
	fmt.Println()

	openBrowser(authURL)

	// Wait for OAuth callback
	code, err := waitForOAuthCallback()
	if err != nil {
		return err
	}

	// Exchange code for tokens
	tokens, err := exchangeZaloCode(code, appID, appSecret, redirectURI)
	if err != nil {
		return err
	}

	// Save tokens to skill config
	cfg, _ := loadSkillConfig(skillDir)
	if cfg == nil {
		cfg = &SkillConfig{Tokens: map[string]string{}}
	}
	if cfg.Tokens == nil {
		cfg.Tokens = map[string]string{}
	}
	cfg.Tokens["zalo_access_token"] = tokens["access_token"]
	cfg.Tokens["zalo_refresh_token"] = tokens["refresh_token"]
	cfg.Tokens["zalo_app_id"] = appID
	cfg.Tokens["zalo_app_secret"] = appSecret
	cfg.OAuthDone = true

	return saveSkillConfig(skillDir, cfg)
}

// ── Zalo OA OAuth ──

func oauthZaloOA(skillDir string) error {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Kết nối Zalo Official Account      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	appID := promptInput("  Zalo OA App ID")
	appSecret := promptInput("  Zalo OA App Secret")

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", oauthCallbackPort)

	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/oa/permission?app_id=%s&redirect_uri=%s",
		appID, url.QueryEscape(redirectURI),
	)

	fmt.Println()
	info("Opening browser for Zalo OA authorization...")
	openBrowser(authURL)

	code, err := waitForOAuthCallback()
	if err != nil {
		return err
	}

	tokens, err := exchangeZaloCode(code, appID, appSecret, redirectURI)
	if err != nil {
		return err
	}

	cfg, _ := loadSkillConfig(skillDir)
	if cfg == nil {
		cfg = &SkillConfig{Tokens: map[string]string{}}
	}
	if cfg.Tokens == nil {
		cfg.Tokens = map[string]string{}
	}
	cfg.Tokens["zalo_oa_access_token"] = tokens["access_token"]
	cfg.Tokens["zalo_oa_refresh_token"] = tokens["refresh_token"]
	cfg.Tokens["zalo_oa_app_id"] = appID
	cfg.Tokens["zalo_oa_app_secret"] = appSecret
	cfg.OAuthDone = true

	return saveSkillConfig(skillDir, cfg)
}

// ── Google OAuth ──

func oauthGoogle(scope, skillDir string) error {
	fmt.Println()
	info("Google OAuth for %s - coming soon", scope)
	fmt.Println("  For now, please configure Google credentials manually.")
	return nil
}

// ── Shared OAuth helpers ──

func waitForOAuthCallback() (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", oauthCallbackPort),
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(400)
			fmt.Fprint(w, "Missing authorization code")
			errCh <- fmt.Errorf("no code in OAuth callback")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body style="font-family:sans-serif;text-align:center;padding:60px">
			<h2>Kết nối thành công!</h2>
			<p>Bạn có thể đóng tab này và quay lại terminal.</p>
		</body></html>`)
		codeCh <- code
	})

	go server.ListenAndServe()

	fmt.Println("  Waiting for authorization...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		server.Shutdown(context.Background())
		return "", err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return "", fmt.Errorf("OAuth timeout - no response after 5 minutes")
	}

	server.Shutdown(context.Background())
	return code, nil
}

func exchangeZaloCode(code, appID, appSecret, redirectURI string) (map[string]string, error) {
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

	var result map[string]interface{}
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

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}
