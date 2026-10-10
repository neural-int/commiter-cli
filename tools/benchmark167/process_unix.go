//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// /usr/bin/time owns a helper child. Cancel the whole measurement process
// group so a cycle timeout cannot leave model inference running unobserved.
func configureMeasurementProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
}
