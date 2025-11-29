package audio

import (
	"context"
	"log"
	"os/exec"
	"time"
)

// Player executes an external command to play wav files.
type Player struct {
	cmd    string
	args   []string
	logger *log.Logger
}

// NewPlayer constructs a Player.
func NewPlayer(cmd string, args []string, logger *log.Logger) Player {
	if logger == nil {
		logger = log.Default()
	}
	return Player{cmd: cmd, args: args, logger: logger}
}

// PlayWav runs the playback command with a timeout, logging start/end/errors.
func (p Player) PlayWav(path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fullArgs := append(append([]string{}, p.args...), path)
	p.logger.Printf("INFO: playback start cmd=%s args=%v", p.cmd, fullArgs)
	cmd := exec.CommandContext(ctx, p.cmd, fullArgs...)
	if err := cmd.Run(); err != nil {
		p.logger.Printf("ERROR: playback failed for %s: %v", path, err)
		return
	}
	p.logger.Printf("INFO: playback finished for %s", path)
}
