package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTextFromArgs(t *testing.T) {
	text, err := ResolveText("", []string{"hello", "world"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "hello world" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestResolveTextFromFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(fp, []byte("from file\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	text, err := ResolveText(fp, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "from file" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestResolveTextFromStdin(t *testing.T) {
	stdin := bytes.NewBufferString("from stdin")
	text, err := ResolveText("-", nil, stdin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "from stdin" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestResolveTextEmpty(t *testing.T) {
	if _, err := ResolveText("", nil, nil); err == nil {
		t.Fatalf("expected error for empty input")
	}
}
