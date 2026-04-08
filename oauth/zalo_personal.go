package oauth

import (
	"fmt"
	"os"
	"os/exec"
)

func init() {
	Register(&ZaloPersonal{})
}

// ZaloPersonal implements authentication for Zalo Personal Account
// via OpenClaw's built-in zca-js QR code login.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate runs OpenClaw's QR code login flow.
// The user scans the QR code with the Zalo mobile app to authorize.
// No App ID or App Secret needed — OpenClaw handles it via zca-js.
func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║   Zalo Personal — QR Code Login      ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  Scan the QR code below with your Zalo app to connect.")
	fmt.Println()

	// Delegate to OpenClaw CLI which handles QR display and zca-js auth.
	cmd := exec.Command("openclaw", "channels", "login", "--channel", "zalouser")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("zalo login failed: %w\n\n  Make sure OpenClaw is installed and running.", err)
	}

	fmt.Println()
	return map[string]string{"zalo_personal_connected": "true"}, nil
}
