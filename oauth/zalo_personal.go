package oauth

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func init() {
	Register(&ZaloPersonal{})
}

// ZaloPersonal implements authentication for Zalo Personal Account
// via OpenClaw's interactive configure wizard with QR code login.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate runs OpenClaw's interactive configure flow.
// While configure runs, a mini HTTP server serves the QR image
// so users on headless servers can open it from their phone/laptop.
func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Zalo Personal — QR Code Login      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	// Check if already connected.
	if isZaloConnected() {
		fmt.Println("  ✓ Zalo Personal already connected")
		return map[string]string{"zalo_personal_connected": "true"}, nil
	}

	// Start QR server in background — serves /tmp/openclaw/*.png via HTTP
	// so users on headless servers can view QR from their phone/laptop.
	qrServer, qrURL := startQRServer()
	if qrServer != nil {
		defer qrServer.Shutdown(context.Background())
	}

	// Run interactive configure.
	fmt.Println("  Opening OpenClaw setup wizard...")
	fmt.Println("  Select 'Zalo (Personal Account)' and scan the QR code.")
	if qrURL != "" {
		fmt.Println()
		fmt.Printf("  ┌─────────────────────────────────────────────┐\n")
		fmt.Printf("  │ QR code will be available at:               │\n")
		fmt.Printf("  │ %s  │\n", qrURL)
		fmt.Printf("  │ Open this URL on your phone to scan.       │\n")
		fmt.Printf("  └─────────────────────────────────────────────┘\n")
	}
	fmt.Println()

	cmd := exec.Command("openclaw", "configure")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	// Verify connection after configure.
	fmt.Println()
	if isZaloConnected() {
		fmt.Println("  ✓ Zalo Personal connected")
		return map[string]string{"zalo_personal_connected": "true"}, nil
	}

	fmt.Println("  ⚠ Zalo not connected yet.")
	fmt.Println("  You can complete this later by running:")
	fmt.Println("    openclaw configure")
	fmt.Println()

	answer := PromptInput("  Continue installing skill anyway? [y/N]")
	if answer != "y" && answer != "Y" && answer != "yes" {
		return nil, fmt.Errorf("zalo setup skipped — run 'openclaw configure' to complete later")
	}
	return map[string]string{"zalo_personal_connected": "pending"}, nil
}

// startQRServer starts a temporary HTTP server that serves PNG files
// from /tmp/openclaw/. Returns the server and the URL to access it.
func startQRServer() (*http.Server, string) {
	qrDir := filepath.Join(os.TempDir(), "openclaw")
	if _, err := os.Stat(qrDir); os.IsNotExist(err) {
		os.MkdirAll(qrDir, 0755)
	}

	// Get machine's LAN IP for the URL.
	ip := getLANIP()
	port := "9877"

	mux := http.NewServeMux()

	// Serve all PNGs in the directory.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Find the newest QR PNG.
		qrPath := findQRImage()
		if qrPath == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			fmt.Fprint(w, `<html><body style="font-family:sans-serif;text-align:center;padding:40px">
				<h2>Waiting for QR code...</h2>
				<p>QR code will appear here after you select Zalo Personal in the setup wizard.</p>
				<script>setTimeout(()=>location.reload(), 3000)</script>
			</body></html>`)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif;text-align:center;padding:20px">
			<h2>Scan QR with Zalo app</h2>
			<img src="/qr.png" style="max-width:400px">
			<p style="color:#666">After scanning, go back to the terminal and confirm.</p>
		</body></html>`)
	})

	mux.HandleFunc("/qr.png", func(w http.ResponseWriter, r *http.Request) {
		qrPath := findQRImage()
		if qrPath == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, qrPath)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Try to start — if port is busy, skip silently.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, ""
	}

	go server.Serve(listener)

	url := fmt.Sprintf("http://%s:%s", ip, port)
	return server, url
}

// getLANIP returns the machine's LAN IP address.
func getLANIP() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 2*time.Second)
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

// isZaloConnected checks if zalouser has an authenticated session.
func isZaloConnected() bool {
	var out bytes.Buffer
	cmd := exec.Command("openclaw", "channels", "status", "--probe")
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	output := out.String()
	if !strings.Contains(strings.ToLower(output), "zalo") {
		return false
	}
	if strings.Contains(output, "Not authenticated") || strings.Contains(output, "not configured") {
		return false
	}
	return true
}

// findQRImage looks for the most recent QR PNG in /tmp/openclaw/.
func findQRImage() string {
	dir := filepath.Join(os.TempDir(), "openclaw")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var pngs []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			pngs = append(pngs, e)
		}
	}

	if len(pngs) == 0 {
		return ""
	}

	sort.Slice(pngs, func(i, j int) bool {
		fi, _ := pngs[i].Info()
		fj, _ := pngs[j].Info()
		if fi == nil || fj == nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})

	return filepath.Join(dir, pngs[0].Name())
}
