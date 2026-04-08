// Package ui provides terminal output helpers with colors and prompts.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorCyan   = "\033[0;36m"
	colorYellow = "\033[0;33m"
	colorReset  = "\033[0m"
)

// Info prints an informational message with a cyan bullet.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s▸%s %s\n", colorCyan, colorReset, msg)
}

// Ok prints a success message with a green checkmark.
func Ok(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, msg)
}

// Warn prints a warning message with a yellow triangle.
func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠%s %s\n", colorYellow, colorReset, msg)
}

// Fatal prints an error message and exits with code 1.
func Fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✗%s %s\n", colorRed, colorReset, msg)
	os.Exit(1)
}

// PromptInput reads a line of input from the user with the given label.
func PromptInput(label string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
