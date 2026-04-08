package oauth

import (
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
// via OpenClaw's interactive channel setup with QR code login.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate runs OpenClaw's interactive channel add flow which handles
// plugin installation, QR code generation, and Zalo login.
//
// The correct command is `openclaw channels add` (interactive) — NOT
// `openclaw channels login` which is unsupported for zalouser.
func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Zalo Personal — QR Code Login      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()

	// Remove stale extension that causes "duplicate plugin" errors.
	home, _ := os.UserHomeDir()
	staleExt := filepath.Join(home, ".openclaw", "extensions", "zalouser")
	if _, err := os.Stat(staleExt); err == nil {
		fmt.Println("  Removing stale zalouser extension...")
		os.RemoveAll(staleExt)
	}

	// Run interactive channel setup — this handles everything:
	// 1. Plugin selection (bundled vs npm)
	// 2. QR code generation → saved to /tmp/openclaw/openclaw-zalouser-qr-default.png
	// 3. User scans QR with Zalo app
	// 4. Config update
	fmt.Println("  Starting OpenClaw Zalo setup...")
	fmt.Println("  Follow the prompts below:")
	fmt.Println()

	cmd := exec.Command("openclaw", "channels", "add", "--channel", "zalouser")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Interactive add failed — show QR file location as fallback.
		qrPath := findQRImage()
		if qrPath != "" {
			fmt.Println()
			fmt.Printf("  QR code saved at: %s\n", qrPath)
			fmt.Println("  Open the file and scan with your Zalo app.")
			fmt.Println()
			PromptInput("  Press Enter after scanning")
			return map[string]string{"zalo_personal_connected": "true"}, nil
		}

		fmt.Println()
		fmt.Println("  ⚠ Automatic setup failed.")
		fmt.Println()
		fmt.Println("  To setup manually, run:")
		fmt.Println("    openclaw channels add")
		fmt.Println("  Then select 'Zalo (Personal Account)' and scan QR.")
		fmt.Println()

		answer := PromptInput("  Continue installing skill anyway? [y/N]")
		if answer != "y" && answer != "Y" && answer != "yes" {
			return nil, fmt.Errorf("zalo setup skipped — run 'openclaw channels add' to complete later")
		}
		return map[string]string{"zalo_personal_connected": "pending"}, nil
	}

	fmt.Println()
	fmt.Println("  ✓ Zalo Personal connected")
	return map[string]string{"zalo_personal_connected": "true"}, nil
}

// findQRImage looks for the zalouser QR code PNG in /tmp/openclaw/.
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
