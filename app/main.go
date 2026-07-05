package main

import (
	"embed"
	"log"
	"runtime"

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
	application.RegisterEvent[ScreenshotPinEvent]("screenshot.pin")
	application.RegisterEvent[ScreenshotWhiteboardContext]("screenshot.whiteboard")
}

func main() {
	recordingFreedom := NewRecordingFreedomService()
	recordingFreedom.logEvent("app", "startup", map[string]string{"platform": runtime.GOOS})

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

	capsuleWindow := createCapsuleWindow(app)
	settingsWindow := createSettingsWindow(app)
	whiteboardWindow := createWhiteboardWindow(app)
	annotationOverlayWindow := createAnnotationOverlayWindow(app)
	annotationRendererWindow := createAnnotationRendererWindow(app)
	regionOverlayWindow := createRegionOverlayWindow(app)
	screenIndicatorWindow := createScreenIndicatorWindow(app)
	pipOverlayWindow := createPIPOverlayWindow(app)
	screenshotPinWindow := createScreenshotPinWindow(app)
	recordingFreedom.setCapsuleWindow(capsuleWindow)
	recordingFreedom.setSettingsWindow(settingsWindow)
	recordingFreedom.setWhiteboardWindow(whiteboardWindow)
	recordingFreedom.setAnnotationOverlayWindow(annotationOverlayWindow)
	recordingFreedom.setAnnotationRendererWindow(annotationRendererWindow)
	recordingFreedom.setRegionOverlayWindow(regionOverlayWindow)
	recordingFreedom.setScreenIndicatorWindow(screenIndicatorWindow)
	recordingFreedom.setPIPOverlayWindow(pipOverlayWindow)
	recordingFreedom.setScreenshotPinWindow(screenshotPinWindow)
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
