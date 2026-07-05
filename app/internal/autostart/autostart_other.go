//go:build !windows && !darwin && !linux

package autostart

import "runtime"

func Enable(config Config) error {
	_, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	return unsupportedPlatformError()
}

func Disable(config Config) error {
	_, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	return unsupportedPlatformError()
}

func unsupportedPlatformError() error {
	return &UnsupportedPlatformError{Platform: runtime.GOOS}
}

type UnsupportedPlatformError struct {
	Platform string
}

func (err *UnsupportedPlatformError) Error() string {
	return "start at login is not supported on " + err.Platform
}
