package oauth

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
// The user selects Zalo Personal, scans QR code, and confirms.
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

	// Run interactive configure — handles everything:
	// plugin selection, QR code, scan confirmation.
	fmt.Println("  Opening OpenClaw setup wizard...")
	fmt.Println("  Select 'Zalo (Personal Account)' and scan the QR code.")
	fmt.Println()

	cmd := exec.Command("openclaw", "configure")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run() // Don't fail on error — user might cancel or skip

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

	// If status contains zalouser but NOT "not configured" or "Not authenticated",
	// then it's connected.
	if !strings.Contains(strings.ToLower(output), "zalo") {
		return false
	}
	if strings.Contains(output, "Not authenticated") || strings.Contains(output, "not configured") {
		return false
	}
	return true
}
