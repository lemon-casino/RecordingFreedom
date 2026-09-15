// Package fsutil centralizes the atomic file-write pattern shared by
// settings, secrets, whiteboard, annotation, and export writers.
package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const (
	TransientAccessAttempts = 20
	TransientAccessDelay    = 10 * time.Millisecond
)

// RetryTransientAccess reruns op while Windows reports access denied,
// sharing, or lock violations (antivirus, search indexer). Other platforms
// and other errors return immediately.
func RetryTransientAccess(op func() error) error {
	var lastErr error
	for attempt := 0; attempt < TransientAccessAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !IsTransientFileAccessError(err) {
			return err
		}
		time.Sleep(TransientAccessDelay)
	}
	return lastErr
}

// WriteFileAtomic writes data to path through a tmp file and a rename so a
// crash mid-write cannot leave a truncated target. os.Rename atomically
// replaces an existing target on both Windows and POSIX.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := RetryTransientAccess(func() error { return os.Rename(tmp, path) }); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// IsTransientFileAccessError reports whether err is a Windows access denied,
// sharing, or lock violation that typically clears within milliseconds.
func IsTransientFileAccessError(err error) bool {
	if runtime.GOOS != "windows" || err == nil {
		return false
	}
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	switch errno {
	case 5, 32, 33:
		return true
	default:
		return false
	}
}
