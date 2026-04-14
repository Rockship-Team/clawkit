// Package oauth provides a pluggable OAuth system for clawkit.
// Each provider implements the Provider interface and registers itself.
package oauth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const CallbackPort = 9876

// Provider defines the interface that all OAuth providers must implement.
type Provider interface {
	// Name returns the provider identifier (e.g., "zalo_personal", "gmail")
	Name() string

	// Display returns a human-readable name (e.g., "Zalo Personal Account")
	Display() string

	// Authenticate runs the full OAuth flow and returns tokens.
	Authenticate() (map[string]string, error)
}

// registry holds all registered providers
var registry = map[string]Provider{}

// Register adds a provider to the registry. Call this from each provider's init().
func Register(p Provider) {
	registry[p.Name()] = p
}

// Get returns a provider by name, or error if not found.
func Get(name string) (Provider, error) {
	p, ok := registry[name]
	if !ok {
		available := make([]string, 0, len(registry))
		for k := range registry {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unsupported OAuth provider: %s (available: %v)", name, available)
	}
	return p, nil
}

// ListProviders returns all registered provider names.
func ListProviders() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}

// WaitForCallback starts a local HTTP server and waits for an OAuth callback.
// Shared by all providers that use browser-based redirect flows.
func WaitForCallback() (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", CallbackPort),
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Missing authorization code")
			errCh <- fmt.Errorf("no code in OAuth callback")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body style="font-family:sans-serif;text-align:center;padding:60px">
			<h2>Authorization successful!</h2>
			<p>You can close this tab and return to the terminal.</p>
		</body></html>`)
		codeCh <- code
	})

	// serverErrCh captures ListenAndServe startup failures (e.g. port in use).
	// Buffered so the goroutine never blocks after server.Shutdown returns.
	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- fmt.Errorf("callback server failed to start: %w (is port %d in use?)", err, CallbackPort)
		}
	}()

	fmt.Println("  Waiting for authorization...")

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx) //nolint:errcheck
	}

	select {
	case code := <-codeCh:
		shutdown()
		return code, nil
	case err := <-errCh:
		shutdown()
		return "", err
	case err := <-serverErrCh:
		return "", err
	case <-time.After(5 * time.Minute):
		shutdown()
		return "", fmt.Errorf("OAuth timeout - no response after 5 minutes")
	}
}

// OpenBrowser opens a URL in the default browser (cross-platform).
func OpenBrowser(url string) {
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
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		fmt.Fprintf(os.Stderr, "Open this URL manually: %s\n", url)
	}
}

// PromptInput reads a line of input from the user. Thin wrapper to avoid
// importing bufio in every provider file.
var PromptInput func(label string) string
