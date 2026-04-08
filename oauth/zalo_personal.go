package oauth

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	Register(&ZaloPersonal{})
}

// ZaloPersonal implements authentication for Zalo Personal Account
// via OpenClaw's built-in zca-js QR code login.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate ensures the zalouser plugin is installed and enabled,
// then runs OpenClaw's QR code login flow.
func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Zalo Personal — QR Code Login      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Check if zalouser plugin is available.
	if !isZaloPluginInstalled() {
		fmt.Println("  Zalo plugin not found. Installing...")
		if err := installZaloPlugin(); err != nil {
			return nil, fmt.Errorf("failed to install zalo plugin: %w\n\n  You can install manually: openclaw plugins install @openclaw/zalouser", err)
		}
		fmt.Println("  ✓ Zalo plugin installed")
	} else {
		fmt.Println("  ✓ Zalo plugin found")
	}

	// Step 2: Run login — OpenClaw generates a QR PNG in /tmp/openclaw/.
	fmt.Println()
	fmt.Println("  Starting Zalo login...")

	cmd := exec.Command("openclaw", "channels", "login", "--channel", "zalouser")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("zalo login failed: %w", err)
	}

	// Step 3: Find and open the QR code image for the user to scan.
	qrPath := findQRImage()
	if qrPath != "" {
		fmt.Println()
		fmt.Println("  Opening QR code — scan it with your Zalo app.")
		OpenBrowser(qrPath)
		fmt.Println()
		fmt.Printf("  QR code saved at: %s\n", qrPath)
		fmt.Println("  If the image didn't open, open it manually and scan with Zalo.")
		fmt.Println()
		fmt.Println("  Waiting for you to scan... (press Enter after scanning)")
		PromptInput("")
	} else {
		fmt.Println()
		fmt.Println("  QR code image not found in /tmp/openclaw/.")
		fmt.Println("  Check the OpenClaw logs or run manually:")
		fmt.Println("    openclaw channels login --channel zalouser")
	}

	fmt.Println()
	return map[string]string{"zalo_personal_connected": "true"}, nil
}

// isZaloPluginInstalled checks if the zalouser channel is available.
func isZaloPluginInstalled() bool {
	var out bytes.Buffer
	cmd := exec.Command("openclaw", "channels", "status", "--probe")
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.Contains(out.String(), "zalouser")
}

// installZaloPlugin installs the @openclaw/zalouser plugin.
func installZaloPlugin() error {
	cmd := exec.Command("openclaw", "plugins", "install", "@openclaw/zalouser")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findQRImage looks for the most recent PNG file in /tmp/openclaw/.
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

	// Sort by modification time, newest first.
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
