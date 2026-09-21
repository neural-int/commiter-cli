//go:build darwin || linux

package mlx

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureHelperProcess(command *exec.Cmd, grace time.Duration) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return cancelHelperProcess(command, grace) }
	command.WaitDelay = defaultWaitDelay
}

func cancelHelperProcess(command *exec.Cmd, grace time.Duration) error {
	if command.Process == nil || command.Process.Pid <= 0 {
		return os.ErrProcessDone
	}
	err := syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	if err != nil {
		return err
	}
	time.Sleep(grace)
	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func cleanupHelperProcess(command *exec.Cmd) {
	if command.Process == nil || command.Process.Pid <= 0 {
		return
	}
	if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return
	}
}
