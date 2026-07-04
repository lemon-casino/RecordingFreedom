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

	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	maxAnnotationEventBatchBytes = 512 * 1024
	maxAnnotationEventBatchLines = 512
	annotationRegionTargetType   = "annotation-region"
	annotationRegionTargetID     = "annotation:region"
	annotationOverlayFrameInset  = 10
)

type AnnotationOverlayState struct {
	Mode            string                              `json:"mode,omitempty"`
	PackageDir      string                              `json:"packageDir,omitempty"`
	ManifestPath    string                              `json:"manifestPath,omitempty"`
	WindowBounds    RegionRect                          `json:"windowBounds"`
	CanvasBounds    RegionRect                          `json:"canvasBounds"`
	Target          recpackage.ManifestAnnotationTarget `json:"target"`
	CaptureExcluded bool                                `json:"captureExcluded"`
}

type AnnotationCaptureRequest struct {
	SceneJSON       string `json:"sceneJson"`
	SnapshotDataURL string `json:"snapshotDataUrl"`
	EventsJSONL     string `json:"eventsJsonl,omitempty"`
}

type AnnotationCaptureResult struct {
	PackageDir           string `json:"packageDir"`
	ScenePath            string `json:"scenePath"`
	EventsPath           string `json:"eventsPath"`
	SnapshotPath         string `json:"snapshotPath"`
	TimelineSnapshotPath string `json:"timelineSnapshotPath,omitempty"`
	Bytes                int64  `json:"bytes"`
}

type annotationOverlayDiagnosticEvent struct {
	SchemaVersion   int                                   `json:"schemaVersion"`
	Type            string                                `json:"type"`
	RecordedAt      string                                `json:"recordedAt"`
	SessionID       string                                `json:"sessionId,omitempty"`
	RecordingMode   string                                `json:"recordingMode,omitempty"`
	Status          string                                `json:"status,omitempty"`
	WindowBounds    RegionRect                            `json:"windowBounds"`
	CanvasBounds    RegionRect                            `json:"canvasBounds"`
	Target          recpackage.ManifestAnnotationTarget   `json:"target"`
	CaptureExcluded bool                                  `json:"captureExcluded"`
	HitRegions      *annotationOverlayHitRegionDiagnostic `json:"hitRegions,omitempty"`
}

type annotationOverlayHitRegionDiagnostic struct {
	Enabled          bool                     `json:"enabled"`
	Force            bool                     `json:"force,omitempty"`
	ViewportWidth    float64                  `json:"viewportWidth"`
	ViewportHeight   float64                  `json:"viewportHeight"`
	DevicePixelRatio float64                  `json:"devicePixelRatio"`
	Regions          []CapsuleWindowHitRegion `json:"regions"`
}

func (s *RecordingFreedomService) ShowAnnotationOverlay() (AnnotationOverlayState, error) {
	if s.app == nil || s.annotationOverlay == nil {
		return AnnotationOverlayState{}, errors.New("annotation overlay window is not configured")
	}
	state, err := s.annotationOverlayState()
	if err != nil {
		return AnnotationOverlayState{}, err
	}
	windowBounds := application.Rect{
		X:      state.WindowBounds.X,
		Y:      state.WindowBounds.Y,
		Width:  state.WindowBounds.Width,
		Height: state.WindowBounds.Height,
	}
	s.annotationOverlay.SetIgnoreMouseEvents(false)
	s.annotationOverlay.SetAlwaysOnTop(true)
	s.annotationOverlay.SetBounds(windowBounds)
	s.annotationOverlay.Show()
	s.annotationOverlay.SetBounds(windowBounds)
	s.annotationOverlay.Focus()
	state.CaptureExcluded = setWindowCaptureExcluded(s.annotationOverlay, false)
	if err := s.appendAnnotationOverlayDiagnostic("show", state, nil); err != nil {
		return state, err
	}
	s.broadcastAnnotationOverlayState(state)
	go s.rebroadcastAnnotationOverlayState(state, s.nextAnnotationToken())
	s.logEvent("annotation-overlay", "show", map[string]string{
		"packageDir": state.PackageDir,
		"targetType": state.Target.Type,
		"targetId":   state.Target.ID,
	})
	s.emitWhiteboardVisibility(true, "annotation")
	return state, nil
}

func (s *RecordingFreedomService) HideAnnotationOverlay() error {
	if s.annotationOverlay == nil {
		return nil
	}
	s.nextAnnotationToken()
	_, _ = s.annotationHitRegions.Update(CapsuleWindowHitRegionsRequest{Enabled: false})
	_ = s.applyAnnotationOverlayHitRegions()
	s.annotationOverlay.Hide()
	s.logEvent("annotation-overlay", "hide", nil)
	s.emitWhiteboardVisibility(false, "annotation")
	return nil
}

func (s *RecordingFreedomService) SetAnnotationOverlayHitRegions(req CapsuleWindowHitRegionsRequest) error {
	_, changed := s.annotationHitRegions.Update(req)
	if !changed && !req.Force {
		return nil
	}
	if err := s.applyAnnotationOverlayHitRegions(); err != nil {
		return err
	}
	state, err := s.annotationOverlayState()
	if err == nil {
		if err := s.appendAnnotationOverlayDiagnostic("hit-regions", state, &req); err != nil {
			return err
		}
	}
	return nil
}

func (s *RecordingFreedomService) ShowAnnotationRegionSelector() (RegionSelectionSession, error) {
	if s.recorder == nil {
		return RegionSelectionSession{}, errors.New("recorder service is not initialized")
	}
	if s.app == nil || s.regionOverlay == nil {
		return RegionSelectionSession{}, errors.New("region overlay window is not configured")
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.RecordingMode != recpackage.RecordingModeScreen {
		return RegionSelectionSession{}, errors.New("annotation region selection requires an active screen recording")
	}
	bounds := s.annotationSelectionBounds(session.Manifest)
	displayCount := 0
	if s.app != nil {
		_, displayCount = regionOverlayBounds(s.app.Screen.GetAll())
	}
	selection := RegionSelectionSession{
		ID:            fmt.Sprintf("annotation-region-%d", time.Now().UnixNano()),
		Bounds:        regionRectFromAppRect(bounds),
		MinimumWidth:  minRegionWidth,
		MinimumHeight: minRegionHeight,
		DisplayCount:  displayCount,
		Purpose:       regionSelectionPurposeAnnotation,
	}

	s.regionMu.Lock()
	s.regionSession = &selection
	s.regionMu.Unlock()

	s.regionOverlay.SetIgnoreMouseEvents(false)
	s.regionOverlay.SetAlwaysOnTop(true)
	s.regionOverlay.SetBounds(bounds)
	s.regionOverlay.Show()
	s.regionOverlay.SetBounds(bounds)
	s.regionOverlay.Focus()
	if payload, err := json.Marshal(selection); err == nil {
		s.regionOverlay.ExecJS(fmt.Sprintf(
			"window.__RF_REGION_SESSION__=%s;window.dispatchEvent(new CustomEvent('rf-region-session',{detail:window.__RF_REGION_SESSION__}));",
			string(payload),
		))
	}
	s.logEvent("annotation-overlay", "region-selector-show", map[string]string{
		"sessionId": session.ID,
		"bounds":    fmt.Sprintf("%d,%d %dx%d", bounds.X, bounds.Y, bounds.Width, bounds.Height),
	})
	return selection, nil
}

func (s *RecordingFreedomService) CompleteAnnotationRegionSelection(req RegionSelectionRequest) (AnnotationOverlayState, error) {
	if s.recorder == nil {
		return AnnotationOverlayState{}, errors.New("recorder service is not initialized")
	}
	recordingSession, ok := s.recorder.ActiveSession()
	if !ok || recordingSession.RecordingMode != recpackage.RecordingModeScreen {
		return AnnotationOverlayState{}, errors.New("annotation region selection requires an active screen recording")
	}

	s.regionMu.Lock()
	selection := s.regionSession
	if selection != nil && selection.Purpose == regionSelectionPurposeAnnotation {
		s.regionSession = nil
	} else {
		selection = nil
	}
	s.regionMu.Unlock()
	if selection == nil {
		return AnnotationOverlayState{}, errors.New("no active annotation region selection session")
	}

	relative := normalizeRegionSelection(req)
	if relative.Width < selection.MinimumWidth || relative.Height < selection.MinimumHeight {
		return AnnotationOverlayState{}, fmt.Errorf("selected annotation region must be at least %d x %d", selection.MinimumWidth, selection.MinimumHeight)
	}
	absoluteDIP := application.Rect{
		X:      selection.Bounds.X + relative.X,
		Y:      selection.Bounds.Y + relative.Y,
		Width:  relative.Width,
		Height: relative.Height,
	}
	s.setAnnotationRegionDIP(recordingSession.ID, absoluteDIP)
	if s.regionOverlay != nil {
		s.regionOverlay.Hide()
	}
	s.logEvent("annotation-overlay", "region-selected", map[string]string{
		"sessionId": recordingSession.ID,
		"bounds":    fmt.Sprintf("%d,%d %dx%d", absoluteDIP.X, absoluteDIP.Y, absoluteDIP.Width, absoluteDIP.Height),
	})
	return s.ShowAnnotationOverlay()
}

func (s *RecordingFreedomService) ReselectAnnotationRegion() (RegionSelectionSession, error) {
	if s.recorder == nil {
		return RegionSelectionSession{}, errors.New("recorder service is not initialized")
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.RecordingMode != recpackage.RecordingModeScreen {
		return RegionSelectionSession{}, errors.New("annotation region selection requires an active screen recording")
	}
	if err := s.clearAnnotationCaptureForSession(session); err != nil {
		return RegionSelectionSession{}, err
	}
	s.clearAnnotationRegionDIP(session.ID)
	if err := s.HideAnnotationOverlay(); err != nil {
		return RegionSelectionSession{}, err
	}
	selection, err := s.ShowAnnotationRegionSelector()
	if err != nil {
		return RegionSelectionSession{}, err
	}
	s.logEvent("annotation-overlay", "region-reselect", map[string]string{
		"sessionId":  session.ID,
		"packageDir": session.PackageDir,
	})
	return selection, nil
}

func (s *RecordingFreedomService) LoadAnnotationCapture() (WhiteboardSceneResult, error) {
	if s.recorder == nil {
		return WhiteboardSceneResult{}, errors.New("recorder service is not initialized")
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.PackageDir == "" {
		return WhiteboardSceneResult{}, errors.New("annotation scene requires an active recording session")
	}
	if session.RecordingMode != recpackage.RecordingModeScreen {
		return WhiteboardSceneResult{}, errors.New("annotation scene requires an active screen recording")
	}
	path := filepath.Join(session.PackageDir, recpackage.AnnotationSceneFile)
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
		return WhiteboardSceneResult{}, fmt.Errorf("annotation scene path %q is a directory", path)
	}
	if info.Size() > maxWhiteboardSceneBytes {
		return WhiteboardSceneResult{}, fmt.Errorf("annotation scene %q is too large", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return WhiteboardSceneResult{}, err
	}
	if len(data) > 0 && !json.Valid(data) {
		return WhiteboardSceneResult{}, fmt.Errorf("annotation scene %q is not valid JSON", path)
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

func (s *RecordingFreedomService) SaveAnnotationCapture(req AnnotationCaptureRequest) (AnnotationCaptureResult, error) {
	if s.recorder == nil {
		return AnnotationCaptureResult{}, errors.New("recorder service is not initialized")
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.PackageDir == "" || session.Manifest == "" {
		return AnnotationCaptureResult{}, errors.New("annotation capture requires an active screen recording session")
	}
	sceneJSON := strings.TrimSpace(req.SceneJSON)
	if sceneJSON == "" {
		return AnnotationCaptureResult{}, errors.New("annotation scene JSON is required")
	}
	if len(sceneJSON) > maxWhiteboardSceneBytes {
		return AnnotationCaptureResult{}, fmt.Errorf("annotation scene exceeds %d bytes", maxWhiteboardSceneBytes)
	}
	if !json.Valid([]byte(sceneJSON)) {
		return AnnotationCaptureResult{}, errors.New("annotation scene JSON is invalid")
	}
	snapshot, err := decodeAnnotationSnapshot(req.SnapshotDataURL)
	if err != nil {
		return AnnotationCaptureResult{}, err
	}
	if len(snapshot) == 0 {
		return AnnotationCaptureResult{}, errors.New("annotation snapshot is empty")
	}
	if len(snapshot) > maxWhiteboardExportBytes {
		return AnnotationCaptureResult{}, fmt.Errorf("annotation snapshot exceeds %d bytes", maxWhiteboardExportBytes)
	}

	annotationsDir := filepath.Join(session.PackageDir, recpackage.AnnotationsDir)
	exportsDir := filepath.Join(session.PackageDir, recpackage.AnnotationExportsDir)
	snapshotsDir := filepath.Join(session.PackageDir, recpackage.AnnotationSnapshotsDir)
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return AnnotationCaptureResult{}, err
	}
	scenePath := filepath.Join(session.PackageDir, recpackage.AnnotationSceneFile)
	eventsPath := filepath.Join(session.PackageDir, recpackage.AnnotationEventsFile)
	snapshotPath := filepath.Join(session.PackageDir, recpackage.AnnotationSnapshotFile)
	nextSequence, err := nextAnnotationEventSequence(eventsPath)
	if err != nil {
		return AnnotationCaptureResult{}, err
	}
	timelineSnapshotRel := annotationTimelineSnapshotFile(nextSequence)
	timelineSnapshotPath := filepath.Join(session.PackageDir, timelineSnapshotRel)
	if err := os.MkdirAll(annotationsDir, 0o755); err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := writeFileAtomic(scenePath, append([]byte(sceneJSON), '\n'), 0o644); err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := writeFileAtomic(snapshotPath, snapshot, 0o644); err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := writeFileAtomic(timelineSnapshotPath, snapshot, 0o644); err != nil {
		return AnnotationCaptureResult{}, err
	}
	eventJSON, err := s.annotationEventJSON(session, recpackage.AnnotationSceneFile, timelineSnapshotRel)
	if err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := appendAnnotationEvent(eventsPath, req.EventsJSONL, eventJSON); err != nil {
		return AnnotationCaptureResult{}, err
	}

	state, err := s.annotationOverlayState()
	if err != nil {
		return AnnotationCaptureResult{}, err
	}
	if err := s.appendAnnotationOverlayDiagnostic("save-capture", state, nil); err != nil {
		return AnnotationCaptureResult{}, err
	}
	_, err = recpackage.NewService().PatchAnnotations(session.Manifest, recpackage.ManifestAnnotations{
		Enabled:         true,
		Mode:            "overlay",
		ScenePath:       recpackage.AnnotationSceneFile,
		EventsPath:      recpackage.AnnotationEventsFile,
		SnapshotPath:    recpackage.AnnotationSnapshotFile,
		DiagnosticsPath: recpackage.AnnotationOverlayDiagnosticsFile,
		CapturePolicy:   s.annotationCapturePolicy(),
		Target:          state.Target,
	})
	if err != nil {
		return AnnotationCaptureResult{}, err
	}
	s.logEvent("annotation-overlay", "save-capture", map[string]string{
		"packageDir": session.PackageDir,
		"bytes":      fmt.Sprint(len(sceneJSON) + len(snapshot)),
	})
	return AnnotationCaptureResult{
		PackageDir:           session.PackageDir,
		ScenePath:            scenePath,
		EventsPath:           eventsPath,
		SnapshotPath:         snapshotPath,
		TimelineSnapshotPath: timelineSnapshotPath,
		Bytes:                int64(len(sceneJSON) + len(snapshot)),
	}, nil
}

func (s *RecordingFreedomService) appendAnnotationOverlayDiagnostic(eventType string, state AnnotationOverlayState, hitRegions *CapsuleWindowHitRegionsRequest) error {
	if s == nil || s.recorder == nil {
		return nil
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.PackageDir == "" || session.RecordingMode != recpackage.RecordingModeScreen {
		return nil
	}
	if strings.TrimSpace(eventType) == "" {
		eventType = "overlay"
	}
	event := annotationOverlayDiagnosticEvent{
		SchemaVersion:   1,
		Type:            eventType,
		RecordedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		SessionID:       session.ID,
		RecordingMode:   session.RecordingMode,
		Status:          string(session.Status),
		WindowBounds:    state.WindowBounds,
		CanvasBounds:    state.CanvasBounds,
		Target:          state.Target,
		CaptureExcluded: state.CaptureExcluded,
	}
	if hitRegions != nil {
		event.HitRegions = &annotationOverlayHitRegionDiagnostic{
			Enabled:          hitRegions.Enabled,
			Force:            hitRegions.Force,
			ViewportWidth:    hitRegions.ViewportWidth,
			ViewportHeight:   hitRegions.ViewportHeight,
			DevicePixelRatio: hitRegions.DevicePixelRatio,
			Regions:          append([]CapsuleWindowHitRegion(nil), hitRegions.Regions...),
		}
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	diagnosticsPath := filepath.Join(session.PackageDir, recpackage.AnnotationOverlayDiagnosticsFile)
	if err := os.MkdirAll(filepath.Dir(diagnosticsPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(diagnosticsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *RecordingFreedomService) annotationCapturePolicy() string {
	if s == nil || s.settings == nil {
		return "export-compose"
	}
	current, err := s.settings.Load()
	if err != nil {
		return "export-compose"
	}
	if current.Whiteboard.CapturePolicy == "preview-only" {
		return "preview-only"
	}
	return "export-compose"
}

func (s *RecordingFreedomService) annotationEventJSON(session recording.Session, scenePath string, snapshotPath string) (string, error) {
	recordedAt := time.Now()
	wallOffsetMs := int64(0)
	if !session.StartedAt.IsZero() {
		wallOffsetMs = recordedAt.Sub(session.StartedAt).Milliseconds()
		if wallOffsetMs < 0 {
			wallOffsetMs = 0
		}
	}
	recordingOffsetMs := wallOffsetMs
	if s != nil && s.recorder != nil {
		if offset, ok := s.recorder.ActiveRecordingOffset(recordedAt); ok {
			recordingOffsetMs = offset
		}
	}
	payload, err := json.Marshal(map[string]any{
		"type":              "scene-snapshot",
		"sessionId":         session.ID,
		"recordingMode":     session.RecordingMode,
		"status":            session.Status,
		"recordedAt":        recordedAt.UTC().Format(time.RFC3339Nano),
		"wallOffsetMs":      wallOffsetMs,
		"recordingOffsetMs": recordingOffsetMs,
		"scenePath":         scenePath,
		"snapshotPath":      snapshotPath,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (s *RecordingFreedomService) annotationOverlayState() (AnnotationOverlayState, error) {
	if s.recorder == nil {
		return AnnotationOverlayState{}, errors.New("recorder service is not initialized")
	}
	session, ok := s.recorder.ActiveSession()
	if !ok || session.RecordingMode != recpackage.RecordingModeScreen {
		return AnnotationOverlayState{}, errors.New("annotation overlay requires an active screen recording")
	}
	manifest, err := recpackage.NewService().ReadManifest(session.Manifest)
	if err != nil {
		return AnnotationOverlayState{}, err
	}
	canvasBounds, target, ok := s.annotationTargetWindowBounds(session.ID, manifest)
	if !ok {
		return AnnotationOverlayState{}, errors.New("annotation overlay requires a selected annotation region")
	}
	windowBounds := annotationOverlayWindowBounds(canvasBounds)
	return AnnotationOverlayState{
		Mode:         "annotation",
		PackageDir:   session.PackageDir,
		ManifestPath: session.Manifest,
		WindowBounds: regionRectFromAppRect(windowBounds),
		CanvasBounds: RegionRect{
			X:      annotationOverlayFrameInset,
			Y:      annotationOverlayFrameInset,
			Width:  canvasBounds.Width,
			Height: canvasBounds.Height,
		},
		Target: target,
	}, nil
}

func annotationOverlayWindowBounds(canvasBounds application.Rect) application.Rect {
	if canvasBounds.Width <= 0 || canvasBounds.Height <= 0 {
		return canvasBounds
	}
	return application.Rect{
		X:      canvasBounds.X - annotationOverlayFrameInset,
		Y:      canvasBounds.Y - annotationOverlayFrameInset,
		Width:  canvasBounds.Width + annotationOverlayFrameInset*2,
		Height: canvasBounds.Height + annotationOverlayFrameInset*2,
	}
}

func (s *RecordingFreedomService) annotationWindowBounds(manifest recpackage.Manifest) application.Rect {
	if manifest.Annotations != nil && manifest.Annotations.Target.Geometry != nil && manifest.Annotations.Target.Geometry.Width > 0 && manifest.Annotations.Target.Geometry.Height > 0 {
		return application.Rect{
			X:      manifest.Annotations.Target.Geometry.X,
			Y:      manifest.Annotations.Target.Geometry.Y,
			Width:  manifest.Annotations.Target.Geometry.Width,
			Height: manifest.Annotations.Target.Geometry.Height,
		}
	}
	if s.app != nil {
		bounds, _ := regionOverlayBounds(s.app.Screen.GetAll())
		return bounds
	}
	return application.Rect{Width: 1280, Height: 720}
}

func (s *RecordingFreedomService) annotationTargetWindowBounds(sessionID string, manifest recpackage.Manifest) (application.Rect, recpackage.ManifestAnnotationTarget, bool) {
	if bounds, ok := s.annotationRegionDIPForSession(sessionID); ok {
		target := annotationRegionTargetFromRect(bounds)
		return bounds, target, true
	}
	if manifest.Annotations != nil && manifest.Annotations.Target.Geometry != nil && manifest.Annotations.Target.Geometry.Width > 0 && manifest.Annotations.Target.Geometry.Height > 0 {
		geometry := *manifest.Annotations.Target.Geometry
		target := manifest.Annotations.Target
		return application.Rect{X: geometry.X, Y: geometry.Y, Width: geometry.Width, Height: geometry.Height}, target, true
	}
	return application.Rect{}, recpackage.ManifestAnnotationTarget{}, false
}

func (s *RecordingFreedomService) annotationSelectionBounds(manifestPath string) application.Rect {
	if strings.TrimSpace(manifestPath) != "" {
		if manifest, err := recpackage.NewService().ReadManifest(manifestPath); err == nil {
			if manifest.Source.Geometry != nil && manifest.Source.Geometry.Width > 0 && manifest.Source.Geometry.Height > 0 {
				geometry := manifest.Source.Geometry
				return application.Rect{X: geometry.X, Y: geometry.Y, Width: geometry.Width, Height: geometry.Height}
			}
		}
	}
	if s.app != nil {
		bounds, _ := regionOverlayBounds(s.app.Screen.GetAll())
		return bounds
	}
	return application.Rect{Width: 1280, Height: 720}
}

func (s *RecordingFreedomService) clearAnnotationCaptureForSession(session recording.Session) error {
	if session.PackageDir == "" || session.Manifest == "" {
		return errors.New("annotation reset requires an active recording package")
	}
	annotationsDir, err := annotationPackageChildDir(session.PackageDir, recpackage.AnnotationsDir)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(annotationsDir); err != nil {
		return err
	}
	_, err = recpackage.NewService().ClearAnnotations(session.Manifest)
	return err
}

func annotationPackageChildDir(packageDir string, relative string) (string, error) {
	if strings.TrimSpace(packageDir) == "" {
		return "", errors.New("recording package directory is required")
	}
	if filepath.IsAbs(relative) || strings.TrimSpace(relative) == "" {
		return "", fmt.Errorf("package child path %q is not relative", relative)
	}
	cleanRelative := filepath.Clean(relative)
	if cleanRelative == "." || strings.HasPrefix(cleanRelative, ".."+string(filepath.Separator)) || cleanRelative == ".." {
		return "", fmt.Errorf("package child path %q escapes the recording package", relative)
	}
	packageAbs, err := filepath.Abs(packageDir)
	if err != nil {
		return "", err
	}
	childAbs, err := filepath.Abs(filepath.Join(packageAbs, cleanRelative))
	if err != nil {
		return "", err
	}
	relToPackage, err := filepath.Rel(packageAbs, childAbs)
	if err != nil {
		return "", err
	}
	if relToPackage == "." || strings.HasPrefix(relToPackage, ".."+string(filepath.Separator)) || relToPackage == ".." || filepath.IsAbs(relToPackage) {
		return "", fmt.Errorf("package child path %q escapes the recording package", relative)
	}
	return childAbs, nil
}

func (s *RecordingFreedomService) setAnnotationRegionDIP(sessionID string, bounds application.Rect) {
	s.annotationMu.Lock()
	defer s.annotationMu.Unlock()
	s.annotationSessionID = sessionID
	s.annotationRegionDIP = bounds
}

func (s *RecordingFreedomService) clearAnnotationRegionDIP(sessionID string) {
	s.annotationMu.Lock()
	defer s.annotationMu.Unlock()
	if sessionID == "" || s.annotationSessionID == sessionID {
		s.annotationSessionID = ""
		s.annotationRegionDIP = application.Rect{}
	}
}

func (s *RecordingFreedomService) annotationRegionDIPForSession(sessionID string) (application.Rect, bool) {
	s.annotationMu.Lock()
	defer s.annotationMu.Unlock()
	if sessionID == "" || s.annotationSessionID != sessionID {
		return application.Rect{}, false
	}
	if s.annotationRegionDIP.Width <= 0 || s.annotationRegionDIP.Height <= 0 {
		return application.Rect{}, false
	}
	return s.annotationRegionDIP, true
}

func annotationRegionTargetFromRect(bounds application.Rect) recpackage.ManifestAnnotationTarget {
	geometry := recpackage.ManifestSourceGeometry{
		X:      bounds.X,
		Y:      bounds.Y,
		Width:  bounds.Width,
		Height: bounds.Height,
	}
	return recpackage.ManifestAnnotationTarget{
		Type:     annotationRegionTargetType,
		ID:       annotationRegionTargetID,
		Geometry: &geometry,
	}
}

func (s *RecordingFreedomService) broadcastAnnotationOverlayState(state AnnotationOverlayState) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	script := fmt.Sprintf(
		"window.__RF_ANNOTATION_OVERLAY__=%s;window.dispatchEvent(new CustomEvent('rf-annotation-overlay',{detail:window.__RF_ANNOTATION_OVERLAY__}));",
		string(payload),
	)
	if s.annotationOverlay != nil {
		s.annotationOverlay.ExecJS(script)
	}
}

func (s *RecordingFreedomService) rebroadcastAnnotationOverlayState(state AnnotationOverlayState, token uint64) {
	for _, delay := range []time.Duration{120 * time.Millisecond, 500 * time.Millisecond} {
		time.Sleep(delay)
		if !s.isAnnotationTokenCurrent(token) {
			return
		}
		s.broadcastAnnotationOverlayState(state)
	}
}

func (s *RecordingFreedomService) nextAnnotationToken() uint64 {
	s.annotationMu.Lock()
	defer s.annotationMu.Unlock()
	s.annotationToken++
	return s.annotationToken
}

func (s *RecordingFreedomService) isAnnotationTokenCurrent(token uint64) bool {
	s.annotationMu.Lock()
	defer s.annotationMu.Unlock()
	return token != 0 && s.annotationToken == token
}

func decodeAnnotationSnapshot(dataURL string) ([]byte, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return nil, errors.New("annotation snapshot dataUrl is required")
	}
	if !strings.HasPrefix(dataURL, whiteboardPNGContentPrefix) {
		return nil, errors.New("annotation snapshot must be a PNG data URL")
	}
	return base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, whiteboardPNGContentPrefix))
}

func appendAnnotationEvent(path string, eventsJSONL string, fallbackEventJSON string) error {
	eventsJSONL = strings.TrimSpace(eventsJSONL)
	if eventsJSONL == "" && strings.TrimSpace(fallbackEventJSON) == "" {
		return errors.New("annotation event JSON is required")
	}
	normalized, err := normalizeAnnotationEvents(path, eventsJSONL, fallbackEventJSON)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(normalized); err != nil {
		return err
	}
	if !strings.HasSuffix(normalized, "\n") {
		_, err = file.WriteString("\n")
	}
	return err
}

func normalizeAnnotationEvents(path string, eventsJSONL string, fallbackEventJSON string) (string, error) {
	if len(eventsJSONL) > maxAnnotationEventBatchBytes {
		return "", fmt.Errorf("annotation events batch exceeds %d bytes", maxAnnotationEventBatchBytes)
	}
	defaults := map[string]any{}
	if err := json.Unmarshal([]byte(fallbackEventJSON), &defaults); err != nil {
		return "", fmt.Errorf("annotation fallback event JSON is invalid: %w", err)
	}
	lines := nonEmptyJSONLLines(eventsJSONL)
	if len(lines) == 0 {
		lines = []string{fallbackEventJSON}
	} else {
		lines = append([]string{fallbackEventJSON}, lines...)
	}
	if len(lines) > maxAnnotationEventBatchLines {
		return "", fmt.Errorf("annotation events batch has %d lines, max %d", len(lines), maxAnnotationEventBatchLines)
	}
	nextSequence, err := nextAnnotationEventSequence(path)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for index, line := range lines {
		event := map[string]any{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return "", fmt.Errorf("annotation event line %d is invalid JSON: %w", index+1, err)
		}
		if len(event) == 0 {
			return "", fmt.Errorf("annotation event line %d must be a JSON object", index+1)
		}
		for key, value := range defaults {
			if _, exists := event[key]; !exists {
				event[key] = value
			}
		}
		if annotationStringField(event, "type") == "" {
			event["type"] = "scene-snapshot"
		}
		event["schemaVersion"] = 1
		event["sequence"] = nextSequence
		if annotationStringField(event, "eventId") == "" {
			event["eventId"] = fmt.Sprintf("annotation-%06d", nextSequence)
		}
		data, err := json.Marshal(event)
		if err != nil {
			return "", fmt.Errorf("annotation event line %d cannot be encoded: %w", index+1, err)
		}
		builder.Write(data)
		builder.WriteByte('\n')
		nextSequence++
	}
	return builder.String(), nil
}

func annotationStringField(event map[string]any, key string) string {
	value, ok := event[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func nonEmptyJSONLLines(value string) []string {
	rawLines := strings.Split(value, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func nextAnnotationEventSequence(path string) (int, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return len(nonEmptyJSONLLines(string(data))) + 1, nil
}

func annotationTimelineSnapshotFile(sequence int) string {
	if sequence < 1 {
		sequence = 1
	}
	return filepath.ToSlash(filepath.Join(recpackage.AnnotationSnapshotsDir, fmt.Sprintf("annotation-%06d.png", sequence)))
}
