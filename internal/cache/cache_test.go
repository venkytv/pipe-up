package cache

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnforceLimitEvictsOldest(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(dir, 10, logDiscard)

	paths := []string{
		filepath.Join(dir, "a.wav"),
		filepath.Join(dir, "b.wav"),
		filepath.Join(dir, "c.wav"),
	}
	sizes := []int{5, 5, 5}
	modTimes := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	for i, p := range paths {
		if err := os.WriteFile(p, make([]byte, sizes[i]), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if err := os.Chtimes(p, modTimes[i], modTimes[i]); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	// tmp file should be ignored
	if err := os.WriteFile(filepath.Join(dir, "ignore.wav.tmp"), []byte("tmp"), 0o644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}

	if err := manager.EnforceLimit(); err != nil {
		t.Fatalf("enforce: %v", err)
	}

	remaining, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var names []string
	for _, e := range remaining {
		names = append(names, e.Name())
	}

	// Expect only the two newest wav files to remain (b.wav, c.wav), tmp ignored.
	expected := map[string]bool{"b.wav": true, "c.wav": true, "ignore.wav.tmp": true}
	for _, n := range names {
		if !expected[n] {
			t.Fatalf("unexpected file remaining: %s", n)
		}
		delete(expected, n)
	}
	if len(expected) != 0 {
		t.Fatalf("expected files missing: %v", expected)
	}
}

// logDiscard is a logger that drops output; keeps Manager construction simple in tests.
var logDiscard = log.New(io.Discard, "", 0)
