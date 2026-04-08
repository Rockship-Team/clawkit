// Package ui provides terminal output helpers with colors and prompts.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// color wraps text in ANSI color codes if the terminal supports them.
func color(code, text string) string {
	if !colorEnabled {
		return text
	}
	return code + text + "\033[0m"
}

// symbols returns the appropriate icon set based on terminal capability.
func sym(unicode, ascii string) string {
	if colorEnabled {
		return unicode
	}
	return ascii
}

// Info prints an informational message with a cyan bullet.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color("\033[0;36m", sym("▸", ">")), msg)
}

// Ok prints a success message with a green checkmark.
func Ok(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color("\033[0;32m", sym("✓", "[OK]")), msg)
}

// Warn prints a warning message with a yellow triangle.
func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color("\033[0;33m", sym("⚠", "[WARN]")), msg)
}

// Fatal prints an error message and exits with code 1.
func Fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color("\033[0;31m", sym("✗", "[FAIL]")), msg)
	os.Exit(1)
}

// PromptInput reads a line of input from the user with the given label.
func PromptInput(label string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
