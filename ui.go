package main

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

func info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s▸%s %s\n", colorCyan, colorReset, msg)
}

func ok(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, msg)
}

func warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠%s %s\n", colorYellow, colorReset, msg)
}

func fatal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✗%s %s\n", colorRed, colorReset, msg)
	os.Exit(1)
}

func promptInput(label string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
