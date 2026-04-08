package oauth

import (
	"bytes"
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
// via OpenClaw's interactive configure wizard with QR code login.
type ZaloPersonal struct{}

func (z *ZaloPersonal) Name() string    { return "zalo_personal" }
func (z *ZaloPersonal) Display() string { return "Zalo Personal Account" }

// Authenticate runs OpenClaw's interactive configure flow.
// After OpenClaw generates the QR PNG, clawkit renders it as
// Unicode text in the terminal so users can scan from their phone —
// works everywhere: laptop, VPS, Docker, SSH.
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

	fmt.Println("  Opening OpenClaw setup wizard...")
	fmt.Println("  Select 'Zalo (Personal Account)' and follow the prompts.")
	fmt.Println()
	fmt.Println("  When the QR code is generated, it will be displayed here")
	fmt.Println("  in the terminal. Scan it with your Zalo app.")
	fmt.Println()

	// Watch for QR file in background and print it when it appears.
	qrDone := make(chan struct{})
	go watchAndPrintQR(qrDone)

	// Run interactive configure (blocking).
	cmd := exec.Command("openclaw", "configure")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	close(qrDone)

	// Verify connection.
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

// watchAndPrintQR polls for the QR PNG file and prints it to terminal once found.
func watchAndPrintQR(done chan struct{}) {
	// Wait a bit for openclaw configure to start and generate QR.
	for i := 0; i < 60; i++ { // poll for up to 60 seconds
		select {
		case <-done:
			return
		default:
		}

		qrPath := findQRImage()
		if qrPath != "" {
			printQRToTerminal(qrPath)
			return
		}

		// Sleep 1 second between polls.
		select {
		case <-done:
			return
		case <-make(chan struct{}):
		default:
		}
		// Simple poll delay.
		time.Sleep(1 * time.Second)
	}
}

// printQRToTerminal reads a QR PNG and renders it as Unicode text in terminal.
// Uses █ (black) and spaces (white), with ▀▄ half-blocks to fit 2 rows per line.
func printQRToTerminal(pngPath string) {
	f, err := os.Open(pngPath)
	if err != nil {
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	bounds := img.Bounds()
	w, h := bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y

	// Sample the image — QR codes are typically small, but the PNG might be scaled up.
	// Find the module size by checking the first black pixel run.
	moduleSize := detectModuleSize(img)
	if moduleSize < 1 {
		moduleSize = 1
	}

	// Number of QR modules.
	cols := w / moduleSize
	rows := h / moduleSize

	// Build a grid of black/white.
	grid := make([][]bool, rows)
	for r := 0; r < rows; r++ {
		grid[r] = make([]bool, cols)
		for c := 0; c < cols; c++ {
			// Sample center of each module.
			px := bounds.Min.X + c*moduleSize + moduleSize/2
			py := bounds.Min.Y + r*moduleSize + moduleSize/2
			grid[r][c] = isDark(img.At(px, py))
		}
	}

	fmt.Println()
	fmt.Println("  ┌" + strings.Repeat("──", cols+2) + "┐")

	// Print 2 rows at a time using half-block characters.
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
		line += " │"
		fmt.Println(line)
	}

	fmt.Println("  └" + strings.Repeat("──", cols+2) + "┘")
	fmt.Println()
	fmt.Println("  Scan the QR code above with your Zalo app.")
	fmt.Println()
}

// detectModuleSize finds the size of one QR module (in pixels) by scanning
// the top-left finder pattern.
func detectModuleSize(img image.Image) int {
	bounds := img.Bounds()
	// Scan from top-left, find first dark pixel, then count consecutive dark pixels.
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
				// First dark run found.
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
	// Convert to 8-bit and check luminance.
	lum := (r>>8)*299 + (g>>8)*587 + (b>>8)*114
	return lum < 128000 // ~50% threshold
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
