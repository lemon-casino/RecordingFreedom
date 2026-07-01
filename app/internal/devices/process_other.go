//go:build !windows

package devices

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}
