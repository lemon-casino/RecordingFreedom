package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

func createCapsuleWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "capsule-recorder",
		Title:           "RecordingFreedom",
		Width:           980,
		Height:          420,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnEscape:    true,
		HideOnFocusLost: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

func createSettingsWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "settings",
		Title:         "RecordingFreedom Settings",
		Width:         920,
		Height:        720,
		MinWidth:      760,
		MinHeight:     560,
		Hidden:        true,
		HideOnEscape:  true,
		DisableResize: false,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(11, 15, 19),
		URL:              "/settings",
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

func configureSystemTray(app *application.App, recorderWindow *application.WebviewWindow, settingsWindow *application.WebviewWindow) {
	tray := app.SystemTray.New()
	tray.SetTooltip("RecordingFreedom")

	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(icons.SystrayLight)
		tray.SetDarkModeIcon(icons.SystrayDark)
	}

	menu := app.NewMenu()
	menu.Add("Show Recorder").OnClick(func(ctx *application.Context) {
		tray.ShowWindow()
	})
	menu.Add("Show Settings").OnClick(func(ctx *application.Context) {
		settingsWindow.Show()
		settingsWindow.Focus()
	})
	menu.Add("Hide Recorder").OnClick(func(ctx *application.Context) {
		tray.HideWindow()
	})
	menu.AddSeparator()
	menu.Add("Quit RecordingFreedom").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	tray.AttachWindow(recorderWindow).WindowOffset(10)
	tray.OnClick(func() {
		tray.ToggleWindow()
	})
	tray.OnRightClick(func() {
		tray.OpenMenu()
	})
}
