package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// readMessage returns the message from CLI arguments, falling back to stdin
// when no arguments are given and stdin is not an interactive terminal.
func readMessage(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	if f, ok := stdin.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
			return "", nil
		}
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("reading message from stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
