//go:build !windows

package video

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}
