package exportplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

type Service struct {
	packages *recpackage.Service
}

func NewService(packages *recpackage.Service) *Service {
	if packages == nil {
		packages = recpackage.NewService()
	}
	return &Service{packages: packages}
}

func (s *Service) Plan(req Request) (Plan, error) {
	videoDir, packageDir, err := validatePackageLocation(req.VideoDir, req.PackageDir)
	if err != nil {
		return Plan{}, err
	}
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	manifest, err := s.packages.ReadManifest(manifestPath)
	if err != nil {
		return Plan{}, err
	}
	if manifest.Status != recpackage.StatusReady {
		return Plan{}, fmt.Errorf("recording package must be ready before export, got status %q", manifest.Status)
	}
	if manifest.Diagnostics.Mock && !req.AllowMock {
		return Plan{}, errors.New("mock recording packages cannot be exported as real media")
	}
	if req.RequireSync {
		if manifest.Diagnostics.Sync == nil {
			return Plan{}, errors.New("export requires diagnostics.sync for audio/video alignment")
		}
		if manifest.Diagnostics.Sync.TimelineBase == recpackage.TimelineBaseMock && !req.AllowMock {
			return Plan{}, errors.New("export requires real media sync diagnostics, got mock timeline")
		}
	}

	outputRel := strings.TrimSpace(req.OutputPath)
	if outputRel == "" {
		outputRel = DefaultOutputPath
	}
	if err := validatePackageRelativePath("outputPath", outputRel); err != nil {
		return Plan{}, err
	}
	screenInput, err := mediaInputPath(packageDir, "screenVideoPath", manifest.Media.ScreenVideoPath)
	if err != nil {
		return Plan{}, err
	}

	preset := pip.Normalize(manifest.Camera.PIPPreset)
	rect, err := pip.Layout(preset, canvasForPIP(req.Canvas, preset))
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		PackageDir:          packageDir,
		ManifestPath:        manifestPath,
		OutputPath:          filepath.Join(packageDir, filepath.Clean(outputRel)),
		ScreenInputPath:     screenInput,
		WebcamStartOffsetMs: manifest.Media.WebcamStartOffsetMs,
		PIPPreset:           string(preset),
		PIPRect:             rect,
	}
	if manifest.Diagnostics.Sync != nil {
		plan.TimelineBase = manifest.Diagnostics.Sync.TimelineBase
		audioDiagnosticsPath, err := optionalPackagePath(packageDir, "audioDiagnosticsPath", manifest.Diagnostics.Sync.AudioDiagnosticsPath)
		if err != nil {
			return Plan{}, err
		}
		videoDiagnosticsPath, err := optionalPackagePath(packageDir, "videoDiagnosticsPath", manifest.Diagnostics.Sync.VideoDiagnosticsPath)
		if err != nil {
			return Plan{}, err
		}
		plan.AudioDiagnosticsPath = audioDiagnosticsPath
		plan.VideoDiagnosticsPath = videoDiagnosticsPath
		for _, segment := range manifest.Diagnostics.Sync.PauseSegments {
			plan.PauseSegments = append(plan.PauseSegments, PauseSegmentPlan{
				StartOffsetMs: segment.StartOffsetMs,
				EndOffsetMs:   segment.EndOffsetMs,
				DurationMs:    segment.DurationMs,
			})
		}
	}

	if preset == pip.PresetOff || !manifest.Camera.Enabled {
		plan.PIPRect = pip.Rect{Visible: false}
		plan.WebcamStartOffsetMs = 0
		return plan, nil
	}
	webcamInput, err := mediaInputPath(packageDir, "webcamVideoPath", manifest.Media.WebcamVideoPath)
	if err != nil {
		return Plan{}, err
	}
	plan.WebcamInputPath = webcamInput
	if manifest.Media.WebcamStartOffsetMs == 0 {
		plan.Warnings = append(plan.Warnings, "webcamStartOffsetMs is zero; export will align webcam at screen start")
	}
	if req.Canvas.Width <= 0 || req.Canvas.Height <= 0 {
		return Plan{}, errors.New("canvas size is required for visible PIP export")
	}
	if err := ensureInside(videoDir, packageDir); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func validatePackageLocation(videoDir string, packageDir string) (string, string, error) {
	if strings.TrimSpace(videoDir) == "" {
		return "", "", errors.New("videoDir is required")
	}
	if strings.TrimSpace(packageDir) == "" {
		return "", "", errors.New("packageDir is required")
	}
	videoRoot, err := filepath.Abs(videoDir)
	if err != nil {
		return "", "", err
	}
	target, err := filepath.Abs(packageDir)
	if err != nil {
		return "", "", err
	}
	if err := ensureInside(videoRoot, target); err != nil {
		return "", "", err
	}
	if !strings.HasSuffix(filepath.Base(target), recpackage.PackageDirSuffix) {
		return "", "", fmt.Errorf("packageDir %q must end with %s", packageDir, recpackage.PackageDirSuffix)
	}
	return videoRoot, target, nil
}

func ensureInside(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("packageDir %q must be inside videoDir %q", target, root)
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

func mediaInputPath(packageDir string, field string, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("%s is required for export", field)
	}
	if err := validatePackageRelativePath(field, relativePath); err != nil {
		return "", err
	}
	target := filepath.Join(packageDir, filepath.Clean(relativePath))
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("%s %q is not readable: %w", field, relativePath, err)
	}
	if info.IsDir() || info.Size() == 0 {
		return "", fmt.Errorf("%s %q is not readable media", field, relativePath)
	}
	return target, nil
}

func optionalPackagePath(packageDir string, field string, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", nil
	}
	if err := validatePackageRelativePath(field, relativePath); err != nil {
		return "", err
	}
	return filepath.Join(packageDir, filepath.Clean(relativePath)), nil
}

func canvasForPIP(canvas pip.Size, preset pip.Preset) pip.Size {
	if preset == pip.PresetOff {
		return pip.Size{Width: 1, Height: 1}
	}
	return canvas
}
