package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
)

const (
	whiteboardDirName          = "whiteboards"
	whiteboardSceneFileName    = "board-current.excalidraw"
	whiteboardExportDirName    = "exports"
	maxWhiteboardSceneBytes    = 25 * 1024 * 1024
	maxWhiteboardExportBytes   = 50 * 1024 * 1024
	whiteboardSceneContentType = "application/vnd.excalidraw+json"
	whiteboardPNGContentPrefix = "data:image/png;base64,"
	whiteboardSVGContentPrefix = "data:image/svg+xml;base64,"
)

type WhiteboardSceneRequest struct {
	SceneJSON string `json:"sceneJson"`
}

type WhiteboardSceneResult struct {
	Available   bool   `json:"available"`
	ScenePath   string `json:"scenePath"`
	SceneJSON   string `json:"sceneJson,omitempty"`
	Bytes       int64  `json:"bytes"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

type WhiteboardExportRequest struct {
	Format  string `json:"format"`
	DataURL string `json:"dataUrl,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type WhiteboardExportResult struct {
	Format     string `json:"format"`
	OutputPath string `json:"outputPath"`
	Bytes      int64  `json:"bytes"`
}

type WhiteboardSettingsPatchRequest struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	LastMode        *string `json:"lastMode,omitempty"`
	LastTool        *string `json:"lastTool,omitempty"`
	LastStrokeColor *string `json:"lastStrokeColor,omitempty"`
	LastStrokeWidth *string `json:"lastStrokeWidth,omitempty"`
	LastOpacity     *int    `json:"lastOpacity,omitempty"`
	CapturePolicy   *string `json:"capturePolicy,omitempty"`
}

type WhiteboardVisibilityEvent struct {
	Visible bool   `json:"visible"`
	Mode    string `json:"mode"`
}

func (s *RecordingFreedomService) ShowWhiteboardWindow() error {
	if s.whiteboardWindow == nil {
		return errors.New("whiteboard window is not configured")
	}
	s.whiteboardMu.Lock()
	s.whiteboardVisible = true
	s.whiteboardMu.Unlock()
	s.whiteboardWindow.SetAlwaysOnTop(true)
	s.whiteboardWindow.Show()
	s.whiteboardWindow.UnMinimise()
	s.whiteboardWindow.Focus()
	s.logEvent("whiteboard", "show", nil)
	s.emitWhiteboardVisibility(true, "whiteboard")
	return nil
}

func (s *RecordingFreedomService) HideWhiteboardWindow() error {
	if s.whiteboardWindow == nil {
		return errors.New("whiteboard window is not configured")
	}
	s.whiteboardMu.Lock()
	s.whiteboardVisible = false
	s.whiteboardMu.Unlock()
	s.whiteboardWindow.Hide()
	s.logEvent("whiteboard", "hide", nil)
	s.emitWhiteboardVisibility(false, "whiteboard")
	return nil
}

func (s *RecordingFreedomService) ToggleWhiteboardWindow() error {
	if s.whiteboardWindow == nil {
		return errors.New("whiteboard window is not configured")
	}
	s.whiteboardMu.Lock()
	visible := s.whiteboardVisible
	s.whiteboardMu.Unlock()
	if visible {
		return s.HideWhiteboardWindow()
	}
	return s.ShowWhiteboardWindow()
}

func (s *RecordingFreedomService) LoadWhiteboardScene() (WhiteboardSceneResult, error) {
	path, err := s.whiteboardScenePath()
	if err != nil {
		return WhiteboardSceneResult{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return WhiteboardSceneResult{
			Available:   false,
			ScenePath:   path,
			ContentType: whiteboardSceneContentType,
		}, nil
	}
	if err != nil {
		return WhiteboardSceneResult{}, err
	}
	if info.IsDir() {
		return WhiteboardSceneResult{}, fmt.Errorf("whiteboard scene path %q is a directory", path)
	}
	if info.Size() > maxWhiteboardSceneBytes {
		return WhiteboardSceneResult{}, fmt.Errorf("whiteboard scene %q is too large", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return WhiteboardSceneResult{}, err
	}
	if len(data) > 0 && !json.Valid(data) {
		return WhiteboardSceneResult{}, fmt.Errorf("whiteboard scene %q is not valid JSON", path)
	}
	return WhiteboardSceneResult{
		Available:   len(data) > 0,
		ScenePath:   path,
		SceneJSON:   string(data),
		Bytes:       info.Size(),
		UpdatedAt:   info.ModTime().UTC().Format(time.RFC3339Nano),
		ContentType: whiteboardSceneContentType,
	}, nil
}

func (s *RecordingFreedomService) SaveWhiteboardScene(req WhiteboardSceneRequest) (WhiteboardSceneResult, error) {
	sceneJSON := strings.TrimSpace(req.SceneJSON)
	if sceneJSON == "" {
		return WhiteboardSceneResult{}, errors.New("whiteboard scene JSON is required")
	}
	if len(sceneJSON) > maxWhiteboardSceneBytes {
		return WhiteboardSceneResult{}, fmt.Errorf("whiteboard scene exceeds %d bytes", maxWhiteboardSceneBytes)
	}
	if !json.Valid([]byte(sceneJSON)) {
		return WhiteboardSceneResult{}, errors.New("whiteboard scene JSON is invalid")
	}
	path, err := s.whiteboardScenePath()
	if err != nil {
		return WhiteboardSceneResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return WhiteboardSceneResult{}, err
	}
	if err := writeFileAtomic(path, append([]byte(sceneJSON), '\n'), 0o644); err != nil {
		return WhiteboardSceneResult{}, err
	}
	s.logEvent("whiteboard", "save-scene", map[string]string{
		"path":  path,
		"bytes": fmt.Sprint(len(sceneJSON)),
	})
	return s.LoadWhiteboardScene()
}

func (s *RecordingFreedomService) SaveWhiteboardExport(req WhiteboardExportRequest) (WhiteboardExportResult, error) {
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format != "png" && format != "svg" && format != "excalidraw" {
		return WhiteboardExportResult{}, fmt.Errorf("unsupported whiteboard export format %q", req.Format)
	}
	data, err := decodeWhiteboardExportPayload(format, req)
	if err != nil {
		return WhiteboardExportResult{}, err
	}
	if len(data) == 0 {
		return WhiteboardExportResult{}, errors.New("whiteboard export payload is empty")
	}
	if len(data) > maxWhiteboardExportBytes {
		return WhiteboardExportResult{}, fmt.Errorf("whiteboard export exceeds %d bytes", maxWhiteboardExportBytes)
	}
	dir, err := s.whiteboardExportDir()
	if err != nil {
		return WhiteboardExportResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return WhiteboardExportResult{}, err
	}
	path := filepath.Join(dir, "whiteboard-"+time.Now().Format("20060102-150405")+"."+format)
	if err := writeFileAtomic(path, data, 0o644); err != nil {
		return WhiteboardExportResult{}, err
	}
	s.logEvent("whiteboard", "save-export", map[string]string{
		"path":   path,
		"format": format,
		"bytes":  fmt.Sprint(len(data)),
	})
	return WhiteboardExportResult{Format: format, OutputPath: path, Bytes: int64(len(data))}, nil
}

func (s *RecordingFreedomService) PatchWhiteboardSettings(patch WhiteboardSettingsPatchRequest) (settings.Settings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	currentSettings, err := s.settings.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	if patch.Enabled != nil {
		currentSettings.Whiteboard.Enabled = *patch.Enabled
	}
	if patch.LastMode != nil {
		currentSettings.Whiteboard.LastMode = strings.TrimSpace(*patch.LastMode)
	}
	if patch.LastTool != nil {
		currentSettings.Whiteboard.LastTool = strings.TrimSpace(*patch.LastTool)
	}
	if patch.LastStrokeColor != nil {
		currentSettings.Whiteboard.LastStrokeColor = strings.TrimSpace(*patch.LastStrokeColor)
	}
	if patch.LastStrokeWidth != nil {
		currentSettings.Whiteboard.LastStrokeWidth = strings.TrimSpace(*patch.LastStrokeWidth)
	}
	if patch.LastOpacity != nil {
		currentSettings.Whiteboard.LastOpacity = *patch.LastOpacity
	}
	if patch.CapturePolicy != nil {
		currentSettings.Whiteboard.CapturePolicy = strings.TrimSpace(*patch.CapturePolicy)
	}
	saved, err := s.settings.Save(currentSettings)
	if err != nil {
		return settings.Settings{}, err
	}
	s.logEvent("whiteboard", "patch-settings", map[string]string{
		"enabled":         fmt.Sprint(saved.Whiteboard.Enabled),
		"lastMode":        saved.Whiteboard.LastMode,
		"lastTool":        saved.Whiteboard.LastTool,
		"lastStrokeColor": saved.Whiteboard.LastStrokeColor,
		"lastStrokeWidth": saved.Whiteboard.LastStrokeWidth,
		"lastOpacity":     fmt.Sprint(saved.Whiteboard.LastOpacity),
		"capturePolicy":   saved.Whiteboard.CapturePolicy,
	})
	s.emitSettingsChanged(saved)
	return saved, nil
}

func decodeWhiteboardExportPayload(format string, req WhiteboardExportRequest) ([]byte, error) {
	if req.Payload != "" {
		data := []byte(req.Payload)
		if format == "excalidraw" && !json.Valid(data) {
			return nil, errors.New("whiteboard excalidraw export payload is invalid JSON")
		}
		return data, nil
	}
	dataURL := strings.TrimSpace(req.DataURL)
	if dataURL == "" {
		return nil, errors.New("whiteboard export dataUrl or payload is required")
	}
	prefix := whiteboardPNGContentPrefix
	if format == "svg" {
		prefix = whiteboardSVGContentPrefix
	} else if format == "excalidraw" {
		return nil, errors.New("whiteboard excalidraw export requires JSON payload")
	}
	if !strings.HasPrefix(dataURL, prefix) {
		return nil, fmt.Errorf("whiteboard %s export has unexpected data URL prefix", format)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *RecordingFreedomService) whiteboardScenePath() (string, error) {
	dir, err := s.whiteboardDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, whiteboardSceneFileName), nil
}

func (s *RecordingFreedomService) whiteboardExportDir() (string, error) {
	dir, err := s.whiteboardDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, whiteboardExportDirName), nil
}

func (s *RecordingFreedomService) whiteboardDir() (string, error) {
	if s == nil || s.appData == nil {
		return "", errors.New("app data service is not initialized")
	}
	root, err := s.appData.RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", whiteboardDirName), nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
