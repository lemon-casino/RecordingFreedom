//go:build linux

package autostart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Enable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	path, err := linuxAutostartPath(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create Linux autostart directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(desktopEntry(config)), 0o644); err != nil {
		return fmt.Errorf("write Linux autostart desktop entry: %w", err)
	}
	return nil
}

func Disable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	path, err := linuxAutostartPath(config)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove Linux autostart desktop entry: %w", err)
	}
	return nil
}

func linuxAutostartPath(config Config) (string, error) {
	configDir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configDir == "" {
		var err error
		configDir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve Linux config directory: %w", err)
		}
	}
	return filepath.Join(configDir, "autostart", strings.ToLower(config.AppName)+".desktop"), nil
}
