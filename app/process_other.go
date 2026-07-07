//go:build !windows

package main

import (
	"errors"
	"os/exec"
)

func configureBackgroundCommand(cmd *exec.Cmd) {}

func openWindowsPath(path string) error {
	return errors.New("Windows shell open is unavailable on this platform")
}
