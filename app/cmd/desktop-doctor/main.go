package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

const (
	checkReady       = "ready"
	checkWarning     = "warning"
	checkBlocked     = "blocked"
	checkQueued      = "queued"
	checkUnsupported = "unsupported"
	checkNotRequired = "not-required"
)

type options struct {
	dataRoot       string
	backend        string
	requireVideo   bool
	requireRNNoise bool
}

type doctorReport struct {
	OK                 bool          `json:"ok"`
	GeneratedAt        time.Time     `json:"generatedAt"`
	Platform           string        `json:"platform"`
	DataRoot           string        `json:"dataRoot,omitempty"`
	VideoDir           string        `json:"videoDir,omitempty"`
	Backend            string        `json:"backend,omitempty"`
	RequestedBackend   string        `json:"requestedBackend,omitempty"`
	RequireVideo       bool          `json:"requireVideo"`
	RequireRNNoise     bool          `json:"requireRnnoise"`
	FFmpegPath         string        `json:"ffmpegPath,omitempty"`
	FFmpegEnv          string        `json:"ffmpegEnv"`
	Checks             []doctorCheck `json:"checks"`
	BlockedCheckCount  int           `json:"blockedCheckCount"`
	RequiredCheckCount int           `json:"requiredCheckCount"`
}

type doctorCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Backend  string `json:"backend,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message,omitempty"`
}

func main() {
	var opts options
	flag.StringVar(&opts.dataRoot, "data-dir", "", "data root to inspect; defaults to the app-managed RecordingFreedom root")
	flag.StringVar(&opts.backend, "backend", "native", "backend request to resolve, for example native, screencapturekit, ffmpeg-desktop-capture, pipewire-portal, or mock-package")
	flag.BoolVar(&opts.requireVideo, "require-video", false, "exit non-zero unless the current platform can start real screen/window video recording")
	flag.BoolVar(&opts.requireRNNoise, "require-rnnoise", false, "exit non-zero unless this build enables native RNNoise microphone suppression")
	flag.Parse()

	report := run(opts)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "desktop-doctor failed to encode report: %v\n", err)
		os.Exit(1)
	}
	if (opts.requireVideo || opts.requireRNNoise) && report.BlockedCheckCount > 0 {
		os.Exit(1)
	}
}

func run(opts options) doctorReport {
	data := appdata.NewService(opts.dataRoot)
	report := doctorReport{
		GeneratedAt:      time.Now().UTC(),
		Platform:         runtime.GOOS,
		RequestedBackend: strings.TrimSpace(opts.backend),
		RequireVideo:     opts.requireVideo,
		RequireRNNoise:   opts.requireRNNoise,
		FFmpegEnv:        video.EnvFFmpegPath,
	}

	info, err := data.Info()
	if err != nil {
		report.add(blocked("app-data", "App Data", true, err.Error()))
	} else {
		report.DataRoot = info.RootDir
		report.VideoDir = info.VideoDir
		report.add(readyPath("app-data", "App Data", true, info.RootDir, "app data root is available"))
		report.add(dataVideoCheck(info.VideoDir))
	}

	storage, storageErr := data.StorageStatus()
	if storageErr != nil {
		report.add(blocked("storage", "Recording Storage", true, storageErr.Error()))
	} else {
		report.add(storageCheck(storage))
	}

	backend := recording.SelectBackend(recpackage.NewService(), runtime.GOOS, opts.backend)
	report.Backend = backend.ID()
	report.add(ready("backend", "Recording Backend", true, fmt.Sprintf("resolved backend %q from request %q", backend.ID(), opts.backend)))

	capabilities := capture.NewService().Capabilities()
	report.add(capabilityCheck(capabilities.SourceEnumeration, false))
	report.add(capabilityCheck(capabilities.ScreenRecording, opts.requireVideo))
	report.add(capabilityCheck(capabilities.WindowRecording, opts.requireVideo))
	report.add(capabilityCheck(capabilities.SystemAudio, false))
	report.add(capabilityCheck(capabilities.Microphone, false))
	report.add(capabilityCheck(capabilities.MicrophoneEnhancement, opts.requireRNNoise))
	report.add(capabilityCheck(capabilities.CameraSidecar, false))
	report.add(capabilityCheck(capabilities.PIPExport, false))
	report.add(capabilityCheck(capabilities.PackageRecovery, true))
	report.add(ffmpegCheck(opts.requireVideo && runtime.GOOS == "windows", &report))

	report.OK = true
	for _, check := range report.Checks {
		if check.Required {
			report.RequiredCheckCount++
		}
		if check.Required && check.Status == checkBlocked {
			report.BlockedCheckCount++
			report.OK = false
		}
	}
	return report
}

func (r *doctorReport) add(check doctorCheck) {
	r.Checks = append(r.Checks, check)
}

func dataVideoCheck(videoDir string) doctorCheck {
	cleaned := filepath.Clean(videoDir)
	if filepath.Base(cleaned) != "video" || filepath.Base(filepath.Dir(cleaned)) != "data" {
		return blocked("data-video", "Recording Output Directory", true, fmt.Sprintf("recordings must stay under data/video, got %q", videoDir))
	}
	return readyPath("data-video", "Recording Output Directory", true, videoDir, "recordings stay under the app-managed data/video directory")
}

func storageCheck(storage appdata.StorageStatus) doctorCheck {
	message := storage.Reason
	if message == "" {
		message = fmt.Sprintf("video dir %s is writable", storage.VideoDir)
	}
	switch storage.Status {
	case appdata.StorageStatusReady:
		return readyPath("storage", "Recording Storage", true, storage.VideoDir, message)
	case appdata.StorageStatusWarning:
		return warningPath("storage", "Recording Storage", true, storage.VideoDir, message)
	default:
		return doctorCheck{
			ID:       "storage",
			Label:    "Recording Storage",
			Status:   checkBlocked,
			Required: true,
			Path:     storage.VideoDir,
			Message:  message,
		}
	}
}

func capabilityCheck(capability capture.Capability, required bool) doctorCheck {
	status := string(capability.Status)
	switch capability.Status {
	case capture.StatusAvailable:
		status = checkReady
	case capture.StatusQueued:
		status = checkQueued
	case capture.StatusBlocked:
		status = checkBlocked
	case capture.StatusUnsupported:
		status = checkUnsupported
	}
	if required && status == checkQueued || required && status == checkUnsupported {
		status = checkBlocked
	}
	return doctorCheck{
		ID:       capability.ID,
		Label:    capability.Label,
		Status:   status,
		Required: required,
		Backend:  capability.Backend,
		Message:  capability.Reason,
	}
}

func ffmpegCheck(required bool, report *doctorReport) doctorCheck {
	if runtime.GOOS != "windows" {
		return doctorCheck{
			ID:       "ffmpeg",
			Label:    "FFmpeg Desktop Writer",
			Status:   checkNotRequired,
			Required: false,
			Message:  "FFmpeg gdigrab is only required for the current Windows desktop writer",
		}
	}
	path, ok, reason := video.FFmpegAvailability()
	if ok {
		if report != nil {
			report.FFmpegPath = path
		}
		return readyPath("ffmpeg", "FFmpeg Desktop Writer", required, path, reason)
	}
	return doctorCheck{
		ID:       "ffmpeg",
		Label:    "FFmpeg Desktop Writer",
		Status:   checkBlocked,
		Required: required,
		Message:  reason,
	}
}

func ready(id string, label string, required bool, message string) doctorCheck {
	return doctorCheck{ID: id, Label: label, Status: checkReady, Required: required, Message: message}
}

func readyPath(id string, label string, required bool, path string, message string) doctorCheck {
	check := ready(id, label, required, message)
	check.Path = path
	return check
}

func warningPath(id string, label string, required bool, path string, message string) doctorCheck {
	return doctorCheck{ID: id, Label: label, Status: checkWarning, Required: required, Path: path, Message: message}
}

func blocked(id string, label string, required bool, message string) doctorCheck {
	return doctorCheck{ID: id, Label: label, Status: checkBlocked, Required: required, Message: message}
}
