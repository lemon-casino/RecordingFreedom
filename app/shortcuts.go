package main

import (
	"fmt"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

type ShortcutTriggeredEvent struct {
	Action      settings.ShortcutAction `json:"action"`
	Accelerator string                  `json:"accelerator"`
}

type ShortcutSettingsPatchRequest struct {
	ToggleRecording *string `json:"toggleRecording,omitempty"`
	TogglePause     *string `json:"togglePause,omitempty"`
	ToggleCamera    *string `json:"toggleCamera,omitempty"`
	OpenWhiteboard  *string `json:"openWhiteboard,omitempty"`
	OpenScreenshot  *string `json:"openScreenshot,omitempty"`
	PasteImage      *string `json:"pasteImage,omitempty"`
}

func (s *RecordingFreedomService) PatchShortcutSettings(patch ShortcutSettingsPatchRequest) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.loadSettingsForMutation()
	if err != nil {
		return settings.Settings{}, err
	}
	previousShortcuts := currentSettings.Shortcuts
	nextShortcuts := applyShortcutSettingsPatch(currentSettings.Shortcuts, patch)
	nextShortcuts, err = settings.ValidateShortcuts(nextShortcuts)
	if err != nil {
		return settings.Settings{}, err
	}
	if err := s.replaceGlobalShortcuts(nextShortcuts); err != nil {
		return settings.Settings{}, err
	}
	currentSettings.Shortcuts = nextShortcuts
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		_ = s.replaceGlobalShortcuts(previousShortcuts)
		return settings.Settings{}, err
	}
	s.logEvent("shortcuts", "patch", shortcutPatchFields(patch, saved.Shortcuts))
	s.emitSettingsChanged(saved)
	return saved, nil
}

func applyShortcutSettingsPatch(current settings.ShortcutSettings, patch ShortcutSettingsPatchRequest) settings.ShortcutSettings {
	if patch.ToggleRecording != nil {
		current.ToggleRecording = strings.TrimSpace(*patch.ToggleRecording)
	}
	if patch.TogglePause != nil {
		current.TogglePause = strings.TrimSpace(*patch.TogglePause)
	}
	if patch.ToggleCamera != nil {
		current.ToggleCamera = strings.TrimSpace(*patch.ToggleCamera)
	}
	if patch.OpenWhiteboard != nil {
		current.OpenWhiteboard = strings.TrimSpace(*patch.OpenWhiteboard)
	}
	if patch.OpenScreenshot != nil {
		current.OpenScreenshot = strings.TrimSpace(*patch.OpenScreenshot)
	}
	if patch.PasteImage != nil {
		current.PasteImage = strings.TrimSpace(*patch.PasteImage)
	}
	return current
}

func shortcutPatchFields(patch ShortcutSettingsPatchRequest, saved settings.ShortcutSettings) map[string]string {
	fields := map[string]string{
		"savedToggleRecording": saved.ToggleRecording,
		"savedTogglePause":     saved.TogglePause,
		"savedToggleCamera":    saved.ToggleCamera,
		"savedOpenWhiteboard":  saved.OpenWhiteboard,
		"savedOpenScreenshot":  saved.OpenScreenshot,
		"savedPasteImage":      saved.PasteImage,
	}
	if patch.ToggleRecording != nil {
		fields["toggleRecording"] = strings.TrimSpace(*patch.ToggleRecording)
	}
	if patch.TogglePause != nil {
		fields["togglePause"] = strings.TrimSpace(*patch.TogglePause)
	}
	if patch.ToggleCamera != nil {
		fields["toggleCamera"] = strings.TrimSpace(*patch.ToggleCamera)
	}
	if patch.OpenWhiteboard != nil {
		fields["openWhiteboard"] = strings.TrimSpace(*patch.OpenWhiteboard)
	}
	if patch.OpenScreenshot != nil {
		fields["openScreenshot"] = strings.TrimSpace(*patch.OpenScreenshot)
	}
	if patch.PasteImage != nil {
		fields["pasteImage"] = strings.TrimSpace(*patch.PasteImage)
	}
	return fields
}

func (s *RecordingFreedomService) replaceGlobalShortcuts(shortcuts settings.ShortcutSettings) error {
	if s == nil || s.app == nil {
		return nil
	}
	normalized, err := settings.ValidateShortcuts(shortcuts)
	if err != nil {
		return err
	}
	s.shortcutMu.Lock()
	defer s.shortcutMu.Unlock()

	previous := make(map[settings.ShortcutAction]string, len(s.registeredShortcuts))
	for action, accelerator := range s.registeredShortcuts {
		previous[action] = accelerator
		if unregisterErr := s.app.GlobalShortcut.Unregister(accelerator); unregisterErr != nil {
			s.logEvent("shortcuts", "unregister-error", map[string]string{
				"action":      string(action),
				"accelerator": accelerator,
				"error":       unregisterErr.Error(),
			})
		}
	}
	s.registeredShortcuts = map[settings.ShortcutAction]string{}

	if err := s.registerGlobalShortcutsLocked(normalized); err != nil {
		for _, accelerator := range s.registeredShortcuts {
			_ = s.app.GlobalShortcut.Unregister(accelerator)
		}
		s.registeredShortcuts = map[settings.ShortcutAction]string{}
		for action, accelerator := range previous {
			if registerErr := s.registerGlobalShortcutLocked(action, accelerator); registerErr != nil {
				s.logEvent("shortcuts", "restore-error", map[string]string{
					"action":      string(action),
					"accelerator": accelerator,
					"error":       registerErr.Error(),
				})
			}
		}
		return err
	}
	return nil
}

func (s *RecordingFreedomService) registerGlobalShortcutsLocked(shortcuts settings.ShortcutSettings) error {
	for _, binding := range settings.ShortcutBindings(shortcuts) {
		if strings.TrimSpace(binding.Accelerator) == "" {
			continue
		}
		if err := s.registerGlobalShortcutLocked(binding.Action, binding.Accelerator); err != nil {
			return err
		}
	}
	return nil
}

func (s *RecordingFreedomService) registerGlobalShortcutLocked(action settings.ShortcutAction, accelerator string) error {
	boundAction := action
	boundAccelerator := strings.TrimSpace(accelerator)
	if boundAccelerator == "" {
		return nil
	}
	if err := s.app.GlobalShortcut.Register(boundAccelerator, func() {
		s.emitShortcutTriggered(boundAction, boundAccelerator)
	}); err != nil {
		s.logEvent("shortcuts", "register-error", map[string]string{
			"action":      string(boundAction),
			"accelerator": boundAccelerator,
			"error":       err.Error(),
		})
		return fmt.Errorf("register %s shortcut %q: %w", boundAction, boundAccelerator, err)
	}
	s.registeredShortcuts[boundAction] = boundAccelerator
	s.logEvent("shortcuts", "registered", map[string]string{
		"action":      string(boundAction),
		"accelerator": boundAccelerator,
	})
	return nil
}

func (s *RecordingFreedomService) emitShortcutTriggered(action settings.ShortcutAction, accelerator string) {
	if s == nil || s.app == nil {
		return
	}
	s.logEvent("shortcuts", "triggered", map[string]string{
		"action":      string(action),
		"accelerator": accelerator,
	})
	s.app.Event.Emit("shortcut.triggered", ShortcutTriggeredEvent{
		Action:      action,
		Accelerator: accelerator,
	})
}
