//go:build !windows

package exporter

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}
