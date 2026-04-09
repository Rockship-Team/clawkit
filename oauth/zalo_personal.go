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

	// Step 1: Check if OpenClaw is available.
	if _, err := exec.LookPath("openclaw"); err != nil {
		fmt.Println("  ✗ OpenClaw CLI not found in PATH.")
		fmt.Println()
		fmt.Println("  Zalo Personal login requires OpenClaw. Install it first:")
		fmt.Println("    npm install -g openclaw")
		fmt.Println()
		fmt.Println("  After installing, run:")
		fmt.Println("    openclaw channels login --channel zalouser")
		fmt.Println()

		answer := PromptInput("  Skip Zalo setup and continue installing? [y/N]")
		if answer == "y" || answer == "Y" || answer == "yes" {
			return map[string]string{"zalo_personal_connected": "pending"}, nil
		}
		return nil, fmt.Errorf("openclaw not installed — run 'npm install -g openclaw'")
	}

	// Step 2: Check if already connected.
	if isZaloConnected() {
		fmt.Println("  ✓ Zalo Personal already connected")
		return map[string]string{"zalo_personal_connected": "true"}, nil
	}

	// Step 3: Ensure zalouser plugin is available.
	if !isZaloPluginInstalled() {
		fmt.Println("  ▸ Zalo Personal plugin not found. Installing...")
		installCmd := exec.Command("openclaw", "plugins", "install", "@openclaw/zalouser")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			fmt.Println()
			fmt.Println("  ⚠ Could not auto-install Zalo plugin.")
			fmt.Println("  Install manually:")
			fmt.Println("    openclaw plugins install @openclaw/zalouser")
			fmt.Println()

			answer := PromptInput("  Skip Zalo setup and continue installing? [y/N]")
			if answer == "y" || answer == "Y" || answer == "yes" {
				return map[string]string{"zalo_personal_connected": "pending"}, nil
			}
			return nil, fmt.Errorf("zalo plugin installation failed: %w", err)
		}
		fmt.Println("  ✓ Zalo Personal plugin installed")
		fmt.Println()
	}

	// Step 4: Clean old QR files to avoid showing a stale code.
	cleanOldQRFiles()

	// Step 5: Start login command in background (non-blocking).
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
	qrPath := waitForQRFile(5 * time.Minute)
	if qrPath == "" {
		fmt.Println("  ✗ QR code not generated within 5 minutes.")
		fmt.Println()
		fmt.Println("  Try running manually:")
		fmt.Println("    openclaw channels login --channel zalouser")
		fmt.Println()
		return nil, fmt.Errorf("QR code generation timed out")
	}

	// Step 5: Display QR code.
	fmt.Print("\033[F\033[2K") // Erase "Generating QR code..." line
	printQRToTerminal(qrPath)

	// Step 6: Wait for login command to complete (user scans QR).
	fmt.Println("  ▸ Waiting for you to scan...")
	fmt.Println("    Open Zalo app → Scan QR → Confirm login")
	fmt.Println()

	// Wait for the background login command to finish (means scan completed or timed out).
	var loginErr error
	select {
	case loginErr = <-loginDone:
	case <-time.After(3 * time.Minute):
		if loginCmd.Process != nil {
			loginCmd.Process.Kill()
		}
		loginErr = fmt.Errorf("timeout")
	}

	if loginErr != nil {
		fmt.Println("  ⚠ Zalo login did not complete.")
		fmt.Println("  The QR code may have expired.")
		fmt.Println()
		fmt.Println("  To retry:")
		fmt.Println("    openclaw channels login --channel zalouser")
		fmt.Println()

		answer := PromptInput("  Continue installing skill anyway? [y/N]")
		if answer == "y" || answer == "Y" || answer == "yes" {
			return map[string]string{"zalo_personal_connected": "pending"}, nil
		}
		return nil, fmt.Errorf("zalo setup skipped — run 'openclaw channels login --channel zalouser' to complete later")
	}

	// Step 7: Enable zalouser channel in OpenClaw config.
	fmt.Println("  ▸ Enabling Zalo channel...")
	enableCmds := [][]string{
		{"openclaw", "config", "set", "channels.zalouser.enabled", "true"},
		{"openclaw", "config", "set", "channels.zalouser.dmPolicy", "pairing"},
	}
	for _, args := range enableCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}

	fmt.Println("  ✓ Zalo Personal connected and enabled!")
	fmt.Println()
	return map[string]string{"zalo_personal_connected": "true"}, nil
}

// openclawQRDirs returns all candidate directories where openclaw may write QR files.
// openclaw uses /tmp/openclaw on macOS/Linux regardless of $TMPDIR,
// and %TEMP%\openclaw on Windows.
func openclawQRDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}

	// Always include /tmp/openclaw (macOS + Linux default used by openclaw).
	add(filepath.Join("/tmp", "openclaw"))

	// Go's os.TempDir() — matches $TMPDIR on macOS (/var/folders/…) and %TEMP% on Windows.
	add(filepath.Join(os.TempDir(), "openclaw"))

	// Extra Windows fallbacks via environment variables.
	for _, env := range []string{"TEMP", "TMP", "USERPROFILE", "APPDATA", "LOCALAPPDATA"} {
		if v := os.Getenv(env); v != "" {
			add(filepath.Join(v, "openclaw"))
		}
	}

	return dirs
}

// cleanOldQRFiles removes previous QR PNGs so we only detect the fresh one.
func cleanOldQRFiles() {
	for _, dir := range openclawQRDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
				os.Remove(filepath.Join(dir, e.Name()))
			}
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
// Returns true if a supported terminal was detected AND image rendering is reliable.
// VSCode is intentionally excluded — its iTerm2 protocol support is inconsistent
// across versions and renders as blank; Unicode fallback works everywhere.
func printInlineImage(data []byte) bool {
	b64 := base64.StdEncoding.EncodeToString(data)

	// iTerm2 inline image protocol.
	// Supported by: iTerm2, WezTerm, Hyper, Tabby, mintty (NOT VSCode — too unreliable).
	// https://iterm2.com/documentation-images.html
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" ||
		os.Getenv("TERM_PROGRAM") == "WezTerm" ||
		os.Getenv("LC_TERMINAL") == "iTerm2" {
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
// horizontal rows to find the first substantial dark run (≥3 px).
// Tries multiple rows (skipping quiet-zone rows that are all-white) so it
// reliably hits the QR finder pattern even when the image has a thick border.
func detectModuleSize(img image.Image) int {
	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y

	// Scan rows at 20%, 25%, 30% … 50% of image height.
	for pct := 20; pct <= 50; pct += 5 {
		y := bounds.Min.Y + h*pct/100
		inDark := false
		darkStart := 0
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dark := isDark(img.At(x, y))
			if dark {
				if !inDark {
					inDark = true
					darkStart = x
				}
			} else {
				if inDark {
					runLen := x - darkStart
					// A module must be at least 2px and at most half the image width.
					if runLen >= 2 && runLen < w/2 {
						return runLen
					}
					inDark = false
				}
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

// isZaloConnected checks if zalouser channel is configured and enabled.
func isZaloConnected() bool {
	var out bytes.Buffer
	cmd := exec.Command("openclaw", "config", "get", "channels.zalouser.enabled")
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(out.String()) == "true"
}

// findQRImage looks for the most recent QR PNG across all candidate openclaw directories.
// Searches /tmp/openclaw first (macOS/Linux), then os.TempDir()/openclaw,
// then Windows %TEMP%/%APPDATA% variants.
func findQRImage() string {
	type entry struct {
		path    string
		modTime time.Time
	}
	var candidates []entry

	for _, dir := range openclawQRDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			candidates = append(candidates, entry{
				path:    filepath.Join(dir, e.Name()),
				modTime: fi.ModTime(),
			})
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	return candidates[0].path
}
