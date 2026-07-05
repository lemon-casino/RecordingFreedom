//go:build windows

package autostart

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const windowsRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func Enable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Windows start at login registry key: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue(config.AppName, commandLine(config.programArguments())); err != nil {
		return fmt.Errorf("write Windows start at login registry value: %w", err)
	}
	return nil
}

func Disable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open Windows start at login registry key: %w", err)
	}
	defer key.Close()
	if err := key.DeleteValue(config.AppName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete Windows start at login registry value: %w", err)
	}
	return nil
}
