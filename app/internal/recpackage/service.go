package recpackage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/audio"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CreateMock(videoDir string, req CreateMockRequest) (Package, error) {
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	if req.Status == "" {
		req.Status = StatusRecording
	}

	id, packageDir, err := reservePackageDir(videoDir, SessionID(req.CreatedAt))
	if err != nil {
		return Package{}, err
	}

	mockMediaPath := filepath.Join(packageDir, MockScreenFile)
	if err := os.WriteFile(mockMediaPath, []byte("RecordingFreedom mock recording package. This is not real video.\n"), 0o644); err != nil {
		return Package{}, err
	}

	recording := recordingprofile.Normalize(req.Recording)
	manifestAudio := normalizeAudio(req.Audio)
	manifestCamera := normalizeCamera(req.Camera)
	manifest := Manifest{
		SchemaVersion: 1,
		App:           AppName,
		CreatedAt:     req.CreatedAt,
		Status:        req.Status,
		Media: ManifestMedia{
			ScreenVideoPath: MockScreenFile,
		},
		Source:    req.Source,
		Recording: recording,
		Audio:     manifestAudio,
		Camera:    manifestCamera,
		Diagnostics: ManifestDiagnostics{
			Mock:    true,
			Message: "UI shell package only; native capture is implemented in later milestones.",
			Sync:    mockSyncDiagnostics(req.CreatedAt, MockScreenFile, recording, manifestAudio, manifestCamera),
		},
	}

	manifestPath := filepath.Join(packageDir, ManifestFile)
	if err := s.WriteManifest(manifestPath, manifest); err != nil {
		return Package{}, err
	}

	return Package{
		ID:           id,
		Dir:          packageDir,
		ManifestPath: manifestPath,
		Manifest:     manifest,
	}, nil
}

func (s *Service) CreateNative(videoDir string, req CreateNativeRequest) (RecordingWritePlan, error) {
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	if req.Status == "" {
		req.Status = StatusRecording
	}
	if strings.TrimSpace(req.Backend) == "" {
		req.Backend = "native"
	}

	id, packageDir, err := reservePackageDir(videoDir, SessionID(req.CreatedAt))
	if err != nil {
		return RecordingWritePlan{}, err
	}
	for _, dir := range []string{CacheDir, ExportsDir} {
		if err := os.MkdirAll(filepath.Join(packageDir, dir), 0o755); err != nil {
			return RecordingWritePlan{}, err
		}
	}

	manifestCamera := normalizeCamera(req.Camera)
	manifestAudio := normalizeAudio(req.Audio)
	media := ManifestMedia{
		ScreenVideoPath: ScreenVideoFile,
	}
	if manifestAudio.System {
		media.SystemAudioPath = SystemAudioFile
		media.SystemAudioStorage = AudioStorageSidecar
	}
	if manifestAudio.Microphone {
		media.MicrophoneAudioPath = MicrophoneAudioFile
		media.MicrophoneAudioStorage = AudioStorageSidecar
	}
	if manifestCamera.Enabled {
		media.WebcamVideoPath = WebcamVideoFile
	}
	manifest := Manifest{
		SchemaVersion: 1,
		App:           AppName,
		CreatedAt:     req.CreatedAt,
		Status:        req.Status,
		Media:         media,
		Source:        req.Source,
		Recording:     recordingprofile.Normalize(req.Recording),
		Audio:         manifestAudio,
		Camera:        manifestCamera,
		Diagnostics: ManifestDiagnostics{
			Message: fmt.Sprintf("Native backend %q initialized the package; media writers must fill package-relative paths.", req.Backend),
		},
	}

	manifestPath := filepath.Join(packageDir, ManifestFile)
	if err := s.WriteManifest(manifestPath, manifest); err != nil {
		return RecordingWritePlan{}, err
	}

	pkg := Package{
		ID:           id,
		Dir:          packageDir,
		ManifestPath: manifestPath,
		Manifest:     manifest,
	}
	return RecordingWritePlan{
		Package:              pkg,
		ScreenVideoPath:      filepath.Join(packageDir, ScreenVideoFile),
		SystemAudioPath:      optionalAbsPackagePath(packageDir, manifest.Media.SystemAudioPath),
		MicrophoneAudioPath:  optionalAbsPackagePath(packageDir, manifest.Media.MicrophoneAudioPath),
		WebcamVideoPath:      optionalAbsPackagePath(packageDir, manifest.Media.WebcamVideoPath),
		AudioDiagnosticsPath: filepath.Join(packageDir, AudioDiagnosticsFile),
		VideoDiagnosticsPath: filepath.Join(packageDir, VideoDiagnosticsFile),
		CacheDir:             filepath.Join(packageDir, CacheDir),
		ExportsDir:           filepath.Join(packageDir, ExportsDir),
	}, nil
}

func (s *Service) PatchStatus(manifestPath string, status string, completedAt *time.Time) error {
	manifest, err := s.ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	manifest.Status = status
	if completedAt != nil && !completedAt.IsZero() {
		manifest.CompletedAt = completedAt
	}
	return s.WriteManifest(manifestPath, manifest)
}

func optionalAbsPackagePath(packageDir string, relativePath string) string {
	if strings.TrimSpace(relativePath) == "" {
		return ""
	}
	return filepath.Join(packageDir, filepath.Clean(relativePath))
}

func (s *Service) PatchSyncDiagnostics(manifestPath string, sync ManifestSyncDiagnostics) error {
	manifest, err := s.ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	manifest.Diagnostics.Sync = &sync
	return s.WriteManifest(manifestPath, manifest)
}

func (s *Service) ValidateReady(manifestPath string) error {
	manifest, err := s.ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	if err := validateManifest(manifest); err != nil {
		return err
	}
	packageDir := filepath.Dir(manifestPath)

	screenPath := strings.TrimSpace(manifest.Media.ScreenVideoPath)
	if manifest.Diagnostics.Mock {
		if filepath.Clean(screenPath) != MockScreenFile {
			return fmt.Errorf("mock package screenVideoPath must be %q, got %q", MockScreenFile, manifest.Media.ScreenVideoPath)
		}
		return requireReadablePackageFile(packageDir, "screenVideoPath", screenPath, true)
	}

	if err := requireReadablePackageFile(packageDir, "screenVideoPath", screenPath, false); err != nil {
		return err
	}
	if manifest.Camera.Enabled {
		if err := requireReadablePackageFile(packageDir, "webcamVideoPath", manifest.Media.WebcamVideoPath, false); err != nil {
			return err
		}
	}
	if manifest.Audio.System {
		if err := s.validateReadyAudioTrack(packageDir, "systemAudio", manifest.Media.SystemAudioPath, manifest.Media.SystemAudioStorage, manifest.Media.ScreenVideoPath); err != nil {
			return err
		}
	}
	if manifest.Audio.Microphone {
		if err := s.validateReadyAudioTrack(packageDir, "microphoneAudio", manifest.Media.MicrophoneAudioPath, manifest.Media.MicrophoneAudioStorage, manifest.Media.ScreenVideoPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateReadyAudioTrack(packageDir string, field string, audioPath string, storage string, screenPath string) error {
	switch storage {
	case AudioStorageSidecar:
		return requireReadablePackageFileMinSize(packageDir, field+"Path", audioPath, false, 45)
	case AudioStorageMuxed:
		if audioPath == "" {
			return fmt.Errorf("%sPath is required before muxed audio can be marked ready", field)
		}
		if filepath.Clean(audioPath) != filepath.Clean(screenPath) {
			return fmt.Errorf("%sPath %q must match screenVideoPath %q for muxed audio", field, audioPath, screenPath)
		}
		screenFile := filepath.Join(packageDir, filepath.Clean(screenPath))
		hasAudioTrack, err := mp4HasAudioTrack(screenFile)
		if err != nil {
			return fmt.Errorf("%s muxed track probe failed: %w", field, err)
		}
		if !hasAudioTrack {
			return fmt.Errorf("%s muxed track is missing from screenVideoPath %q", field, screenPath)
		}
		return nil
	default:
		return fmt.Errorf("%sStorage %q is not supported", field, storage)
	}
}

func (s *Service) Recover(videoDir string, packageDir string, completedAt time.Time) (RecoverySummary, error) {
	if completedAt.IsZero() {
		completedAt = time.Now()
	}
	packageDir, err := validatePackageDir(videoDir, packageDir)
	if err != nil {
		return RecoverySummary{}, err
	}
	if _, err := os.Stat(packageDir); err != nil {
		return RecoverySummary{}, err
	}

	manifestPath := filepath.Join(packageDir, ManifestFile)
	manifest, err := s.ReadManifest(manifestPath)
	if err != nil {
		screenPath, ok := findScreenMedia(packageDir)
		if !ok {
			return RecoverySummary{}, fmt.Errorf("cannot recover %q: manifest is unreadable and no non-empty screen media exists", packageDir)
		}
		manifest = recoveredManifest(packageDir, screenPath, completedAt)
	} else {
		if !isRecoverableStatus(manifest.Status) {
			return RecoverySummary{}, fmt.Errorf("package %q is not recoverable from status %q", packageDir, manifest.Status)
		}
		if !manifestHasReadableScreenMedia(packageDir, manifest) {
			return RecoverySummary{}, fmt.Errorf("cannot recover %q: manifest screen media is missing or empty", packageDir)
		}
		if manifest.Diagnostics.Message == "" {
			manifest.Diagnostics.Message = fmt.Sprintf("Recovered from manifest status %q.", manifest.Status)
		}
	}

	manifest.Status = StatusReady
	manifest.CompletedAt = &completedAt
	manifest.Diagnostics.Recovered = true
	if err := s.WriteManifest(manifestPath, manifest); err != nil {
		return RecoverySummary{}, err
	}
	return RecoverySummary{
		PackageDir:   packageDir,
		ManifestPath: manifestPath,
		Status:       StatusReady,
		Recoverable:  false,
		Reason:       "package recovered and marked ready",
	}, nil
}

func (s *Service) ReadManifest(manifestPath string) (Manifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (s *Service) WriteManifest(manifestPath string, manifest Manifest) error {
	manifest.Recording = recordingprofile.Normalize(manifest.Recording)
	manifest.Audio = normalizeAudio(manifest.Audio)
	manifest.Camera = normalizeCamera(manifest.Camera)
	manifest.Media = normalizeMedia(manifest.Media, manifest.Audio, manifest.Camera, manifest.Diagnostics.Mock)
	manifest.Diagnostics = normalizeDiagnostics(manifest)
	if err := validateManifest(manifest); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, append(data, '\n'), 0o644)
}

func (s *Service) Scan(videoDir string) ([]RecoverySummary, error) {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	summaries := make([]RecoverySummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), PackageDirSuffix) {
			continue
		}
		packageDir := filepath.Join(videoDir, entry.Name())
		manifestPath := filepath.Join(packageDir, ManifestFile)
		manifest, err := s.ReadManifest(manifestPath)
		if err != nil {
			summaries = append(summaries, missingManifestSummary(packageDir, manifestPath))
			continue
		}

		summary := RecoverySummary{
			PackageDir:   packageDir,
			ManifestPath: manifestPath,
			Status:       manifest.Status,
		}
		if isRecoverableStatus(manifest.Status) {
			summary.Status = StatusRecoverable
			summary.Recoverable = true
			summary.Reason = fmt.Sprintf("manifest status %q did not reach ready", manifest.Status)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func SessionID(t time.Time) string {
	return fmt.Sprintf("%s-%03d", t.Format("2006-01-02-15-04-05"), t.Nanosecond()/int(time.Millisecond))
}

func reservePackageDir(videoDir string, baseID string) (string, string, error) {
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		return "", "", err
	}
	for attempt := 0; attempt < 1000; attempt++ {
		id := baseID
		if attempt > 0 {
			id = fmt.Sprintf("%s-%03d", baseID, attempt)
		}
		packageDir := filepath.Join(videoDir, "recording-"+id+PackageDirSuffix)
		if err := os.Mkdir(packageDir, 0o755); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return "", "", err
		}
		return id, packageDir, nil
	}
	return "", "", fmt.Errorf("could not reserve a unique recording package directory for %q", baseID)
}

func normalizeAudio(manifestAudio ManifestAudio) ManifestAudio {
	if !manifestAudio.System {
		manifestAudio.SystemDeviceID = ""
	}
	if !manifestAudio.Microphone {
		manifestAudio.MicrophoneDeviceID = ""
		manifestAudio.MicrophoneNoiseSuppression = NoiseSuppressionOff
		manifestAudio.MicrophoneGain = 0
	}
	if manifestAudio.MicrophoneNoiseSuppression == "" {
		manifestAudio.MicrophoneNoiseSuppression = NoiseSuppressionOff
	}
	if manifestAudio.SampleRate == 0 {
		manifestAudio.SampleRate = audio.RNNoiseSampleRate
	}
	manifestAudio.SystemAudioIsNeverDenoised = true
	return manifestAudio
}

func normalizeCamera(camera ManifestCamera) ManifestCamera {
	if !camera.Enabled {
		camera.DeviceID = ""
		camera.PIPPreset = string(pip.PresetOff)
		return camera
	}
	if camera.PIPPreset == "" {
		camera.PIPPreset = string(pip.DefaultPreset)
	} else {
		camera.PIPPreset = string(pip.Normalize(camera.PIPPreset))
	}
	return camera
}

func normalizeMedia(media ManifestMedia, manifestAudio ManifestAudio, camera ManifestCamera, mock bool) ManifestMedia {
	if media.ScreenVideoPath == "" {
		media.ScreenVideoPath = ScreenVideoFile
	}
	if !manifestAudio.System {
		media.SystemAudioPath = ""
		media.SystemAudioStorage = ""
	} else if mock && media.SystemAudioPath == "" && media.SystemAudioStorage == "" {
		media.SystemAudioPath = ""
	} else {
		media.SystemAudioStorage = normalizeAudioStorage(media.SystemAudioStorage, media.SystemAudioPath)
		if media.SystemAudioStorage == AudioStorageMuxed {
			media.SystemAudioPath = media.ScreenVideoPath
		} else if media.SystemAudioPath == "" {
			media.SystemAudioPath = SystemAudioFile
		}
	}
	if !manifestAudio.Microphone {
		media.MicrophoneAudioPath = ""
		media.MicrophoneAudioStorage = ""
	} else if mock && media.MicrophoneAudioPath == "" && media.MicrophoneAudioStorage == "" {
		media.MicrophoneAudioPath = ""
	} else {
		media.MicrophoneAudioStorage = normalizeAudioStorage(media.MicrophoneAudioStorage, media.MicrophoneAudioPath)
		if media.MicrophoneAudioStorage == AudioStorageMuxed {
			media.MicrophoneAudioPath = media.ScreenVideoPath
		} else if media.MicrophoneAudioPath == "" {
			media.MicrophoneAudioPath = MicrophoneAudioFile
		}
	}
	if !camera.Enabled {
		media.WebcamVideoPath = ""
		media.WebcamStartOffsetMs = 0
	}
	if media.WebcamVideoPath == "" {
		media.WebcamStartOffsetMs = 0
	}
	return media
}

func normalizeDiagnostics(manifest Manifest) ManifestDiagnostics {
	diagnostics := manifest.Diagnostics
	if diagnostics.Sync == nil {
		return diagnostics
	}
	syncDiagnostics := *diagnostics.Sync
	if syncDiagnostics.TimelineBase == "" {
		syncDiagnostics.TimelineBase = TimelineBaseMedia
	}
	syncDiagnostics.Screen = normalizeTrackDiagnostics(syncDiagnostics.Screen, manifest.Media.ScreenVideoPath != "", manifest.Media.ScreenVideoPath)
	syncDiagnostics.SystemAudio = normalizeTrackDiagnostics(syncDiagnostics.SystemAudio, manifest.Audio.System, manifest.Media.SystemAudioPath)
	syncDiagnostics.Microphone = normalizeTrackDiagnostics(syncDiagnostics.Microphone, manifest.Audio.Microphone, manifest.Media.MicrophoneAudioPath)
	syncDiagnostics.Webcam = normalizeTrackDiagnostics(syncDiagnostics.Webcam, manifest.Camera.Enabled, manifest.Media.WebcamVideoPath)
	if syncDiagnostics.Webcam.Enabled && manifest.Media.WebcamStartOffsetMs != 0 && syncDiagnostics.Webcam.StartOffsetMs == 0 {
		syncDiagnostics.Webcam.StartOffsetMs = int64(manifest.Media.WebcamStartOffsetMs)
	}
	diagnostics.Sync = &syncDiagnostics
	return diagnostics
}

func normalizeAudioStorage(storage string, mediaPath string) string {
	storage = strings.TrimSpace(storage)
	switch storage {
	case AudioStorageMuxed:
		return AudioStorageMuxed
	case AudioStorageSidecar:
		return AudioStorageSidecar
	case "":
		if filepath.Clean(mediaPath) == ScreenVideoFile {
			return AudioStorageMuxed
		}
		return AudioStorageSidecar
	default:
		return storage
	}
}

func normalizeTrackDiagnostics(track ManifestTrackDiagnostics, sourceEnabled bool, defaultPath string) ManifestTrackDiagnostics {
	if !sourceEnabled {
		return ManifestTrackDiagnostics{}
	}
	if defaultPath != "" && track.Path == "" {
		track.Path = defaultPath
	}
	if defaultPath != "" || track.Path != "" {
		track.Enabled = true
	}
	return track
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion == 0 {
		return errors.New("manifest schemaVersion is required")
	}
	if manifest.App == "" {
		return errors.New("manifest app is required")
	}
	if manifest.Status == "" {
		return errors.New("manifest status is required")
	}
	if err := validatePackageRelativePath("screenVideoPath", manifest.Media.ScreenVideoPath); err != nil {
		return err
	}
	if err := validatePackageRelativePath("systemAudioPath", manifest.Media.SystemAudioPath); err != nil {
		return err
	}
	if err := validateAudioTrackStorage("systemAudioStorage", manifest.Media.SystemAudioStorage, manifest.Audio.System && !manifest.Diagnostics.Mock); err != nil {
		return err
	}
	if err := validatePackageRelativePath("microphoneAudioPath", manifest.Media.MicrophoneAudioPath); err != nil {
		return err
	}
	if err := validateAudioTrackStorage("microphoneAudioStorage", manifest.Media.MicrophoneAudioStorage, manifest.Audio.Microphone && !manifest.Diagnostics.Mock); err != nil {
		return err
	}
	if err := validatePackageRelativePath("webcamVideoPath", manifest.Media.WebcamVideoPath); err != nil {
		return err
	}
	if err := validateSyncDiagnostics(manifest.Diagnostics.Sync); err != nil {
		return err
	}
	return nil
}

func validateAudioTrackStorage(field string, value string, enabled bool) error {
	if !enabled && value == "" {
		return nil
	}
	switch value {
	case AudioStorageSidecar, AudioStorageMuxed:
		return nil
	case "":
		return fmt.Errorf("%s is required when audio is enabled", field)
	default:
		return fmt.Errorf("%s %q is not supported", field, value)
	}
}

func validateSyncDiagnostics(syncDiagnostics *ManifestSyncDiagnostics) error {
	if syncDiagnostics == nil {
		return nil
	}
	switch syncDiagnostics.TimelineBase {
	case TimelineBaseMock, TimelineBaseMedia, TimelineBasePlatform:
	default:
		return fmt.Errorf("diagnostics.sync.timelineBase %q is not supported", syncDiagnostics.TimelineBase)
	}
	if err := validatePackageRelativePath("diagnostics.sync.audioDiagnosticsPath", syncDiagnostics.AudioDiagnosticsPath); err != nil {
		return err
	}
	if err := validatePackageRelativePath("diagnostics.sync.videoDiagnosticsPath", syncDiagnostics.VideoDiagnosticsPath); err != nil {
		return err
	}
	if err := validateTrackDiagnostics("diagnostics.sync.screen", syncDiagnostics.Screen); err != nil {
		return err
	}
	if err := validateTrackDiagnostics("diagnostics.sync.systemAudio", syncDiagnostics.SystemAudio); err != nil {
		return err
	}
	if err := validateTrackDiagnostics("diagnostics.sync.microphone", syncDiagnostics.Microphone); err != nil {
		return err
	}
	if err := validateTrackDiagnostics("diagnostics.sync.webcam", syncDiagnostics.Webcam); err != nil {
		return err
	}
	for index, segment := range syncDiagnostics.PauseSegments {
		if segment.EndOffsetMs < segment.StartOffsetMs {
			return fmt.Errorf("diagnostics.sync.pauseSegments[%d] endOffsetMs must be greater than or equal to startOffsetMs", index)
		}
		if segment.DurationMs < 0 {
			return fmt.Errorf("diagnostics.sync.pauseSegments[%d] durationMs must be non-negative", index)
		}
	}
	return nil
}

func validateTrackDiagnostics(field string, track ManifestTrackDiagnostics) error {
	if err := validatePackageRelativePath(field+".path", track.Path); err != nil {
		return err
	}
	if track.DurationMs < 0 {
		return fmt.Errorf("%s.durationMs must be non-negative", field)
	}
	if track.DroppedFrames < 0 {
		return fmt.Errorf("%s.droppedFrames must be non-negative", field)
	}
	if track.DroppedSamples < 0 {
		return fmt.Errorf("%s.droppedSamples must be non-negative", field)
	}
	if track.AppendFailures < 0 {
		return fmt.Errorf("%s.appendFailures must be non-negative", field)
	}
	if track.SampleRate < 0 {
		return fmt.Errorf("%s.sampleRate must be non-negative", field)
	}
	if track.FrameRate < 0 {
		return fmt.Errorf("%s.frameRate must be non-negative", field)
	}
	return nil
}

func validatePackageRelativePath(field string, value string) error {
	if value == "" {
		return nil
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("%s must be package-relative, got absolute path %q", field, value)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s must stay inside the recording package, got %q", field, value)
	}
	return nil
}

func requireReadablePackageFile(packageDir string, field string, relativePath string, allowMockMarker bool) error {
	return requireReadablePackageFileMinSize(packageDir, field, relativePath, allowMockMarker, 1)
}

func requireReadablePackageFileMinSize(packageDir string, field string, relativePath string, allowMockMarker bool, minBytes int64) error {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		return fmt.Errorf("%s is required before package can be marked ready", field)
	}
	if err := validatePackageRelativePath(field, relativePath); err != nil {
		return err
	}
	cleaned := filepath.Clean(relativePath)
	if !allowMockMarker && isMockMarkerPath(cleaned) {
		return fmt.Errorf("%s %q is a mock marker, not readable media", field, relativePath)
	}
	target := filepath.Join(packageDir, cleaned)
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("%s %q is not readable: %w", field, relativePath, err)
	}
	if info.IsDir() || info.Size() < minBytes {
		return fmt.Errorf("%s %q is not readable media", field, relativePath)
	}
	return nil
}

func isMockMarkerPath(relativePath string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(relativePath))
	return cleaned == MockScreenFile || strings.HasSuffix(cleaned, ".mock.txt")
}

func validatePackageDir(videoDir string, packageDir string) (string, error) {
	if strings.TrimSpace(videoDir) == "" {
		return "", errors.New("videoDir is required")
	}
	if strings.TrimSpace(packageDir) == "" {
		return "", errors.New("packageDir is required")
	}
	videoRoot, err := filepath.Abs(videoDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(packageDir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(videoRoot, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("packageDir %q must be inside videoDir %q", packageDir, videoDir)
	}
	if !strings.HasSuffix(filepath.Base(target), PackageDirSuffix) {
		return "", fmt.Errorf("packageDir %q must end with %s", packageDir, PackageDirSuffix)
	}
	return target, nil
}

func missingManifestSummary(packageDir string, manifestPath string) RecoverySummary {
	summary := RecoverySummary{
		PackageDir:   packageDir,
		ManifestPath: manifestPath,
		Status:       StatusFailed,
		Reason:       "manifest is missing or unreadable",
	}
	if hasScreenMedia(packageDir) {
		summary.Status = StatusRecoverable
		summary.Recoverable = true
		summary.Reason = "manifest is missing but screen media exists"
	}
	return summary
}

func hasScreenMedia(packageDir string) bool {
	_, ok := findScreenMedia(packageDir)
	return ok
}

func findScreenMedia(packageDir string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(packageDir, "screen.*"))
	if err != nil {
		return "", false
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() && info.Size() > 0 {
			return filepath.Base(match), true
		}
	}
	return "", false
}

func manifestHasReadableScreenMedia(packageDir string, manifest Manifest) bool {
	if manifest.Media.ScreenVideoPath == "" {
		return false
	}
	if validatePackageRelativePath("screenVideoPath", manifest.Media.ScreenVideoPath) != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(packageDir, filepath.Clean(manifest.Media.ScreenVideoPath)))
	return err == nil && !info.IsDir() && info.Size() > 0
}

func recoveredManifest(packageDir string, screenPath string, completedAt time.Time) Manifest {
	return Manifest{
		SchemaVersion: 1,
		App:           AppName,
		CreatedAt:     recoveredCreatedAt(packageDir, completedAt),
		Status:        StatusReady,
		CompletedAt:   &completedAt,
		Media: ManifestMedia{
			ScreenVideoPath: screenPath,
		},
		Recording: recordingprofile.Default(),
		Audio:     normalizeAudio(ManifestAudio{}),
		Camera:    normalizeCamera(ManifestCamera{}),
		Diagnostics: ManifestDiagnostics{
			Recovered: true,
			Message:   "Recovered from screen media after missing or unreadable manifest.",
		},
	}
}

func recoveredCreatedAt(packageDir string, fallback time.Time) time.Time {
	info, err := os.Stat(packageDir)
	if err == nil && !info.ModTime().IsZero() {
		return info.ModTime()
	}
	return fallback
}

func isRecoverableStatus(status string) bool {
	switch status {
	case StatusRecording, StatusPaused, StatusFinalizing:
		return true
	default:
		return false
	}
}

func mockSyncDiagnostics(createdAt time.Time, screenPath string, recording recordingprofile.Profile, manifestAudio ManifestAudio, camera ManifestCamera) *ManifestSyncDiagnostics {
	return &ManifestSyncDiagnostics{
		TimelineBase:          TimelineBaseMock,
		TimelineStartUnixNano: createdAt.UnixNano(),
		Screen: ManifestTrackDiagnostics{
			Enabled:   true,
			Path:      screenPath,
			Clock:     TimelineBaseMock,
			FrameRate: recording.FPS,
			Message:   "Mock marker only; no screen samples were captured.",
		},
		SystemAudio: mockAudioTrack(manifestAudio.System, manifestAudio.SampleRate),
		Microphone:  mockAudioTrack(manifestAudio.Microphone, manifestAudio.SampleRate),
		Webcam: ManifestTrackDiagnostics{
			Enabled: camera.Enabled,
			Clock:   TimelineBaseMock,
			Message: "Mock package only; camera sidecar samples were not captured.",
		},
	}
}

func mockAudioTrack(enabled bool, sampleRate int) ManifestTrackDiagnostics {
	if !enabled {
		return ManifestTrackDiagnostics{}
	}
	return ManifestTrackDiagnostics{
		Enabled:    true,
		Clock:      TimelineBaseMock,
		SampleRate: sampleRate,
		Message:    "Mock package only; audio samples were not captured.",
	}
}
