//go:build !windows

package video

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}

func terminateCommandProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
