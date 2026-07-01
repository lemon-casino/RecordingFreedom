package main

import (
	"embed"

	"log"

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
	application.RegisterEvent[AudioLevelEvent]("audio.level")
}

func main() {
	recordingFreedom := NewRecordingFreedomService()

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
	})
	recordingFreedom.setApp(app)

	capsuleWindow := createCapsuleWindow(app)
	settingsWindow := createSettingsWindow(app)
	regionOverlayWindow := createRegionOverlayWindow(app)
	screenIndicatorWindow := createScreenIndicatorWindow(app)
	recordingFreedom.setSettingsWindow(settingsWindow)
	recordingFreedom.setRegionOverlayWindow(regionOverlayWindow)
	recordingFreedom.setScreenIndicatorWindow(screenIndicatorWindow)
	initialLocale := settings.LocaleZhCN
	if currentSettings, err := recordingFreedom.settings.Load(); err == nil {
		initialLocale = currentSettings.Locale
	}
	recordingFreedom.setTrayLocaleUpdater(configureSystemTray(app, capsuleWindow, settingsWindow, initialLocale))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
