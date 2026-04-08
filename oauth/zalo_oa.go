package oauth

import (
	"fmt"
	"net/url"
)

func init() {
	Register(&ZaloOA{})
}

// ZaloOA implements OAuth for Zalo Official Account.
type ZaloOA struct{}

func (z *ZaloOA) Name() string    { return "zalo_oa" }
func (z *ZaloOA) Display() string { return "Zalo Official Account" }

func (z *ZaloOA) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Printf("  ╔══════════════════════════════════════╗\n")
	fmt.Printf("  ║   %s       ║\n", z.Display())
	fmt.Printf("  ╚══════════════════════════════════════╝\n")
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

	// Reuse same Zalo token exchange (same API, different auth URL)
	tokens, err := exchangeZaloCode(code, appID, appSecret)
	if err != nil {
		return nil, err
	}

	tokens["zalo_oa_app_id"] = appID
	tokens["zalo_oa_app_secret"] = appSecret
	return tokens, nil
}
