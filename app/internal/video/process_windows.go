//go:build windows

package video

import (
	"os/exec"
	"strconv"
	"syscall"
)

const windowsCreateNoWindow = 0x08000000

func configureBackgroundCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windowsCreateNoWindow,
	}
}

func terminateCommandProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	taskkill := exec.Command("taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	configureBackgroundCommand(taskkill)
	if err := taskkill.Run(); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
