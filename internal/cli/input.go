package cli

import (
	"errors"
	"io"
	"os"
	"strings"
)

// ResolveText chooses text input based on flags and args.
// - If filePath is "-", read from stdin.
// - If filePath is a non-empty path, read the file contents.
// - Otherwise, use the joined args.
func ResolveText(filePath string, args []string, stdin io.Reader) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "-" {
		if stdin == nil {
			return "", errors.New("stdin unavailable")
		}
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return normalize(string(b))
	}

	if filePath != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return normalize(string(b))
	}

	if len(args) > 0 {
		return normalize(strings.Join(args, " "))
	}

	return "", errors.New("no input provided")
}

func normalize(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", errors.New("input is empty")
	}
	return trimmed, nil
}
