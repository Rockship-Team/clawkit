package ui

import (
	"os"
	"runtime"
	"strings"
)

// colorEnabled reports whether the terminal supports ANSI color codes.
var colorEnabled = detectColorSupport()

func detectColorSupport() bool {
	// Explicit opt-out.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Explicit opt-in.
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	// Not a terminal (piped output).
	if fi, err := os.Stdout.Stat(); err == nil {
		if fi.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	// Windows: modern terminals (Windows Terminal, VS Code) set WT_SESSION or TERM_PROGRAM.
	// Legacy cmd.exe does not support ANSI without enabling virtual terminal processing.
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") != "" {
			return true // Windows Terminal
		}
		if strings.Contains(os.Getenv("TERM_PROGRAM"), "vscode") {
			return true // VS Code terminal
		}
		// ConEmu, cmder, etc.
		if os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		// Default: assume no ANSI on Windows unless explicitly detected.
		return false
	}
	// macOS/Linux: virtually all terminals support ANSI.
	return true
}
