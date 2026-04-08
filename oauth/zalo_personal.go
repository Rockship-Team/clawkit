package oauth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
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
// via OpenClaw's `channels login` command with QR code scanning.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate runs the Zalo Personal login flow:
//  1. Check if already connected
//  2. Clean old QR files so we only detect the fresh one
//  3. Start `openclaw channels login --channel zalouser` in background
//  4. Poll for QR PNG file
//  5. Display QR (inline image or Unicode fallback)
//  6. Poll for successful connection
//  7. Return result
func (z *ZaloPersonal) Authenticate() (map[string]string, error) {
	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────┐")
	fmt.Println("  │   Zalo Personal — QR Code Login       │")
	fmt.Println("  └──────────────────────────────────────┘")
	fmt.Println()

	// Step 1: Check if already connected.
	if isZaloConnected() {
		fmt.Println("  ✓ Zalo Personal already connected")
		return map[string]string{"zalo_personal_connected": "true"}, nil
	}

	// Step 2: Ensure zalouser plugin is available.
	if !isZaloPluginInstalled() {
		fmt.Println("  ▸ Zalo Personal plugin not found. Installing...")
		installCmd := exec.Command("openclaw", "plugins", "install", "@openclaw/zalouser")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			fmt.Println()
			fmt.Println("  ✗ Failed to install Zalo plugin.")
			fmt.Println("  Install manually:")
			fmt.Println("    openclaw plugins install @openclaw/zalouser")
			fmt.Println()
			return nil, fmt.Errorf("zalo plugin installation failed: %w", err)
		}
		fmt.Println("  ✓ Zalo Personal plugin installed")
		fmt.Println()
	}

	// Step 3: Clean old QR files to avoid showing a stale code.
	cleanOldQRFiles()

	// Step 4: Start login command in background (non-blocking).
	fmt.Println("  ▸ Generating QR code...")
	loginCmd := exec.Command("openclaw", "channels", "login", "--channel", "zalouser")
	loginCmd.Stdin = nil
	loginCmd.Stdout = nil
	loginCmd.Stderr = nil
	if err := loginCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start openclaw login: %w", err)
	}

	// Ensure we clean up the background process when we're done.
	loginDone := make(chan error, 1)
	go func() {
		loginDone <- loginCmd.Wait()
	}()

	// Step 4: Wait for QR file to appear (poll up to 30s).
	qrPath := waitForQRFile(30 * time.Second)
	if qrPath == "" {
		fmt.Println("  ✗ QR code not generated within 30 seconds.")
		fmt.Println()
		fmt.Println("  Try running manually:")
		fmt.Println("    openclaw channels login --channel zalouser")
		fmt.Println()
		return nil, fmt.Errorf("QR code generation timed out")
	}

	// Step 5: Display QR code.
	fmt.Print("\033[F\033[2K") // Erase "Generating QR code..." line
	printQRToTerminal(qrPath)

	// Step 6: Wait for user to scan (poll connection status, up to 2 min).
	fmt.Println("  ▸ Waiting for you to scan...")
	fmt.Println("    Open Zalo app → Scan QR → Confirm login")
	fmt.Println()

	connected := waitForConnection(2 * time.Minute)

	// Clean up background login process.
	if loginCmd.Process != nil {
		loginCmd.Process.Kill()
	}

	if connected {
		fmt.Print("\033[F\033[2K\033[F\033[2K\033[F\033[2K") // Erase waiting lines
		fmt.Println("  ✓ Zalo Personal connected!")
		fmt.Println()
		return map[string]string{"zalo_personal_connected": "true"}, nil
	}

	fmt.Println("  ⚠ Could not verify connection within 2 minutes.")
	fmt.Println("  The QR code may have expired.")
	fmt.Println()
	fmt.Println("  To retry:")
	fmt.Println("    openclaw channels login --channel zalouser")
	fmt.Println()

	answer := PromptInput("  Continue installing skill anyway? [y/N]")
	if answer != "y" && answer != "Y" && answer != "yes" {
		return nil, fmt.Errorf("zalo setup skipped — run 'openclaw channels login --channel zalouser' to complete later")
	}
	return map[string]string{"zalo_personal_connected": "pending"}, nil
}

// cleanOldQRFiles removes previous QR PNGs so we only detect the fresh one.
func cleanOldQRFiles() {
	dir := filepath.Join(os.TempDir(), "openclaw")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// waitForQRFile polls for a new QR PNG file until timeout.
func waitForQRFile(timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if path := findQRImage(); path != "" {
			return path
		}
		time.Sleep(500 * time.Millisecond)
	}
	return ""
}

// waitForConnection polls `openclaw channels status --probe` until connected or timeout.
func waitForConnection(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isZaloConnected() {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// printQRToTerminal displays a QR PNG in the terminal.
// Tries iTerm2/Kitty inline image protocol first (shows actual PNG),
// falls back to Unicode half-block rendering.
func printQRToTerminal(pngPath string) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return
	}

	fmt.Println()

	if printInlineImage(data) {
		fmt.Println()
		return
	}

	printQRUnicode(data)
}

// printInlineImage displays a PNG inline using terminal-specific escape sequences.
// Returns true if a supported terminal was detected.
func printInlineImage(data []byte) bool {
	b64 := base64.StdEncoding.EncodeToString(data)

	// iTerm2 inline image protocol.
	// Supported by: iTerm2, WezTerm, Hyper, VS Code terminal, Tabby, mintty.
	// https://iterm2.com/documentation-images.html
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" ||
		os.Getenv("TERM_PROGRAM") == "WezTerm" ||
		os.Getenv("LC_TERMINAL") == "iTerm2" ||
		os.Getenv("WT_SESSION") != "" ||
		strings.Contains(os.Getenv("TERM_PROGRAM"), "vscode") {
		fmt.Printf("  \033]1337;File=inline=1;width=30;preserveAspectRatio=1:%s\a\n", b64)
		return true
	}

	// Kitty graphics protocol.
	if os.Getenv("TERM") == "xterm-kitty" || os.Getenv("KITTY_PID") != "" {
		const chunkSize = 4096
		for i := 0; i < len(b64); i += chunkSize {
			end := i + chunkSize
			if end > len(b64) {
				end = len(b64)
			}
			chunk := b64[i:end]
			more := 1
			if end >= len(b64) {
				more = 0
			}
			if i == 0 {
				fmt.Printf("\033_Gf=100,a=T,m=%d;%s\033\\", more, chunk)
			} else {
				fmt.Printf("\033_Gm=%d;%s\033\\", more, chunk)
			}
		}
		fmt.Println()
		return true
	}

	return false
}

// printQRUnicode decodes a PNG and renders it as Unicode half-blocks in the terminal.
func printQRUnicode(data []byte) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}

	bounds := img.Bounds()
	w, h := bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y

	moduleSize := detectModuleSize(img)
	if moduleSize < 1 {
		moduleSize = 1
	}

	cols := w / moduleSize
	rows := h / moduleSize

	grid := make([][]bool, rows)
	for r := 0; r < rows; r++ {
		grid[r] = make([]bool, cols)
		for c := 0; c < cols; c++ {
			px := bounds.Min.X + c*moduleSize + moduleSize/2
			py := bounds.Min.Y + r*moduleSize + moduleSize/2
			grid[r][c] = isDark(img.At(px, py))
		}
	}

	fmt.Println("  ┌" + strings.Repeat("──", cols+2) + "┐")
	for r := 0; r < rows; r += 2 {
		line := "  │ "
		for c := 0; c < cols; c++ {
			top := grid[r][c]
			bot := false
			if r+1 < rows {
				bot = grid[r+1][c]
			}
			if top && bot {
				line += "█"
			} else if top && !bot {
				line += "▀"
			} else if !top && bot {
				line += "▄"
			} else {
				line += " "
			}
		}
		fmt.Println(line + " │")
	}
	fmt.Println("  └" + strings.Repeat("──", cols+2) + "┘")
	fmt.Println()
}

// detectModuleSize finds the size of one QR module (in pixels) by scanning
// the top-left finder pattern.
func detectModuleSize(img image.Image) int {
	bounds := img.Bounds()
	inDark := false
	darkStart := 0

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if isDark(img.At(x, bounds.Min.Y+bounds.Dy()/4)) {
			if !inDark {
				inDark = true
				darkStart = x
			}
		} else {
			if inDark {
				runLen := x - darkStart
				if runLen > 2 {
					return runLen
				}
				inDark = false
			}
		}
	}
	return 1
}

// isDark returns true if a pixel is dark (closer to black than white).
func isDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	lum := (r>>8)*299 + (g>>8)*587 + (b>>8)*114
	return lum < 128000
}

// isZaloPluginInstalled checks if the zalouser plugin is available in OpenClaw.
func isZaloPluginInstalled() bool {
	var out bytes.Buffer
	cmd := exec.Command("openclaw", "plugins", "list")
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	output := strings.ToLower(out.String())
	return strings.Contains(output, "zalouser") || strings.Contains(output, "zalo-personal")
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

// findQRImage looks for the most recent QR PNG in $TMPDIR/openclaw/.
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
