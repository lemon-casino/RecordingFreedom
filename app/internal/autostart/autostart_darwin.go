//go:build darwin

package autostart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Enable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	path, err := launchAgentPath(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create macOS LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(launchAgentPlist(config)), 0o644); err != nil {
		return fmt.Errorf("write macOS LaunchAgent: %w", err)
	}
	return nil
}

func Disable(config Config) error {
	config, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	path, err := launchAgentPath(config)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove macOS LaunchAgent: %w", err)
	}
	return nil
}

func launchAgentPath(config Config) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for macOS LaunchAgent: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", config.AppID+".plist"), nil
}
