package main

import (
	"embed"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recording"
	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func init() {
	application.RegisterEvent[recording.StatusEvent]("recording.status")
	application.RegisterEvent[RegionSelectionResult]("capture.region.selected")
	application.RegisterEvent[settings.Settings]("settings.changed")
	application.RegisterEvent[WhiteboardVisibilityEvent]("whiteboard.visibility")
	application.RegisterEvent[AudioState]("audio.state")
	application.RegisterEvent[AudioLevelEvent]("audio.level")
	application.RegisterEvent[ShortcutTriggeredEvent]("shortcut.triggered")
	application.RegisterEvent[ScreenshotCapturedEvent]("screenshot.captured")
	application.RegisterEvent[ScreenshotHistoryResult]("screenshot.history.changed")
	application.RegisterEvent[ScreenshotPinEvent]("screenshot.pin")
	application.RegisterEvent[ScreenshotWhiteboardContext]("screenshot.whiteboard")
	application.RegisterEvent[FloatingPanelState]("floating.panel.changed")
	application.RegisterEvent[FloatingSelectState]("floating.select.changed")
	application.RegisterEvent[FloatingSelectChosenEvent]("floating.select.chosen")
	application.RegisterEvent[SourceControlState]("source.state.changed")
	application.RegisterEvent[ocr.Status]("ocr.status.changed")
	application.RegisterEvent[OcrModelEvent]("ocr.model.installed")
	application.RegisterEvent[OcrModelEvent]("ocr.model.failed")
	application.RegisterEvent[ocr.ModelDownloadEvent]("ocr.model.download.changed")
	application.RegisterEvent[OcrJobEvent]("ocr.job.queued")
	application.RegisterEvent[OcrJobEvent]("ocr.job.started")
	application.RegisterEvent[OcrJobEvent]("ocr.job.finished")
	application.RegisterEvent[OcrJobEvent]("ocr.job.failed")
	application.RegisterEvent[OcrJobEvent]("ocr.job.cancelled")
}

func main() {
	startMinimizedToTray := shouldStartMinimizedToTray(os.Args[1:])
	recordingFreedom := NewRecordingFreedomService()
	recordingFreedom.logEvent("app", "startup", map[string]string{
		"platform":           runtime.GOOS,
		"minimizedToTrayArg": boolLogValue(startMinimizedToTray),
	})

	app := application.New(application.Options{
		Name:        "RecordingFreedom",
		Description: "A modern capsule screen recorder built with Go, React, and Wails v3",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(recordingFreedom),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		Windows: application.WindowsOptions{
			WndProcInterceptor: recordingFreedom.capsuleWindowWndProcInterceptor,
		},
	})
	recordingFreedom.setApp(app)

	capsuleWindow := createCapsuleWindow(app, startMinimizedToTray)
	settingsWindow := createSettingsWindow(app)
	whiteboardWindow := createWhiteboardWindow(app)
	annotationOverlayWindow := createAnnotationOverlayWindow(app)
	annotationRendererWindow := createAnnotationRendererWindow(app)
	regionOverlayWindow := createRegionOverlayWindow(app)
	screenIndicatorWindow := createScreenIndicatorWindow(app)
	pipOverlayWindow := createPIPOverlayWindow(app)
	screenshotPinWindow := createScreenshotPinWindow(app)
	floatingPanelWindow := createFloatingPanelWindow(app)
	floatingSelectWindow := createFloatingSelectWindow(app)
	recordingFreedom.setCapsuleWindow(capsuleWindow)
	recordingFreedom.setSettingsWindow(settingsWindow)
	recordingFreedom.setWhiteboardWindow(whiteboardWindow)
	recordingFreedom.setAnnotationOverlayWindow(annotationOverlayWindow)
	recordingFreedom.setAnnotationRendererWindow(annotationRendererWindow)
	recordingFreedom.setRegionOverlayWindow(regionOverlayWindow)
	recordingFreedom.setScreenIndicatorWindow(screenIndicatorWindow)
	recordingFreedom.setPIPOverlayWindow(pipOverlayWindow)
	recordingFreedom.setScreenshotPinWindow(screenshotPinWindow)
	recordingFreedom.setFloatingPanelWindow(floatingPanelWindow)
	recordingFreedom.setFloatingSelectWindow(floatingSelectWindow)
	initialSettings := settings.Default()
	if currentSettings, err := recordingFreedom.settings.Load(); err == nil {
		initialSettings = currentSettings
	}
	recordingFreedom.setTrayLocaleUpdater(configureSystemTray(app, capsuleWindow, settingsWindow, initialSettings.Locale))
	if err := recordingFreedom.replaceGlobalShortcuts(initialSettings.Shortcuts); err != nil {
		recordingFreedom.logEvent("shortcuts", "startup-register-error", map[string]string{"error": err.Error()})
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func shouldStartMinimizedToTray(args []string) bool {
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "--minimized-to-tray", "--start-at-login":
			return true
		}
	}
	return false
}

func boolLogValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
