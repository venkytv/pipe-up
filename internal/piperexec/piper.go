package piperexec

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner executes the Piper CLI to synthesize audio.
type Runner struct {
	execPath string
	model    string
	flags    []string
	logger   *log.Logger
}

// New creates a Runner for the given executable, model, and flags.
func New(execPath, model string, flags []string, logger *log.Logger) Runner {
	if logger == nil {
		logger = log.Default()
	}
	return Runner{
		execPath: execPath,
		model:    model,
		flags:    flags,
		logger:   logger,
	}
}

// Synthesize runs piper with stdin text and writes output to outPath using a temp file.
func (r Runner) Synthesize(ctx context.Context, text, outPath string) error {
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)

	args := []string{
		"-m", r.model,
		"-f", tmpPath,
	}
	if len(r.flags) > 0 {
		args = append(args, r.flags...)
	}

	r.logger.Printf("INFO: invoking piper exec=%s args=%v", r.execPath, args)

	cmd := exec.CommandContext(ctx, r.execPath, args...)
	cmd.Stdin = strings.NewReader(text)

	start := time.Now()
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmpPath)
		r.logger.Printf("ERROR: piper failed after %s: %v (output: %s)", time.Since(start).Round(time.Millisecond), err, strings.TrimSpace(string(output)))
		return fmt.Errorf("piper exec failed: %w", err)
	}
	r.logger.Printf("INFO: piper completed in %s", time.Since(start).Round(time.Millisecond))

	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename piper output failed: %w", err)
	}
	return nil
}
