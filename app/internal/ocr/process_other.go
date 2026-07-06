//go:build !windows

package ocr

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) {}
