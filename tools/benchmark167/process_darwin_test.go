package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMeasurementCancellationStopsChildHelper(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "child-pid")
	helper := filepath.Join(dir, "slow-helper")
	quoted := "'" + strings.ReplaceAll(pidPath, "'", "'\\''") + "'"
	if err := os.WriteFile(helper, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s' \"$$\" > %s\nexec /bin/sleep 60\n", quoted)), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", "-l", helper)
	configureMeasurementProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var data []byte
	var err error
	for ctx.Err() == nil {
		data, err = os.ReadFile(pidPath)
		if err == nil && len(data) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	started := time.Now()
	cancel()
	waitErr := cmd.Wait()
	if waitErr == nil || time.Since(started) > 3*time.Second {
		t.Fatalf("cancellation not enforced: error=%v elapsed=%v", waitErr, time.Since(started))
	}
	if err != nil || len(data) == 0 {
		t.Fatal("helper did not reach the cancellation probe")
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	// A reaping zombie is terminal and consumes no inference resources.
	state, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	if err == nil && !strings.HasPrefix(strings.TrimSpace(string(state)), "Z") {
		t.Fatalf("helper remains live after deadline, pid=%d", pid)
	}
}
