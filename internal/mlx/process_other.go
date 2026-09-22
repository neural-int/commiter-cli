//go:build !darwin && !linux

package mlx

import (
	"os"
	"os/exec"
	"time"
)

// Non-Unix builds retain the protocol implementation. MLX itself is only
// supported on macOS; without process groups, CommandContext still reaps the
// direct child on cancellation.
func configureHelperProcess(command *exec.Cmd, _ time.Duration) {}

func cancelHelperProcess(command *exec.Cmd, _ time.Duration) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	return command.Process.Kill()
}

func cleanupHelperProcess(_ *exec.Cmd) {}
