//go:build windows

package main

import "os/exec"

// The harness requires the macOS MLX helper and time -l. Keep repository
// builds portable; Windows cannot perform this measurement.
func configureMeasurementProcess(cmd *exec.Cmd) {}
