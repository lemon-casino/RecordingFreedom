package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	capsuleWindowWidth           = 860
	capsuleWindowCollapsedHeight = 166
)

func createCapsuleWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "capsule-recorder",
		Title:           "RecordingFreedom",
		Width:           capsuleWindowWidth,
		Height:          capsuleWindowCollapsedHeight,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnEscape:    true,
		HideOnFocusLost: true,
		BackgroundType:  application.BackgroundTypeTransparent,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
		Linux: application.LinuxWindow{
			Icon: appIcon,
		},
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
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
		Linux: application.LinuxWindow{
			Icon: appIcon,
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

func createRegionOverlayWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "region-overlay",
		Title:            "RecordingFreedom Region Selector",
		Width:            1280,
		Height:           720,
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		Hidden:           true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				Hide:               true,
				HideTitle:          true,
				FullSizeContent:    true,
			},
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
			WindowLevel: application.MacWindowLevelFloating,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:                   true,
			DisableFramelessWindowDecorations: true,
		},
		Linux: application.LinuxWindow{
			Icon:                appIcon,
			WindowIsTranslucent: true,
		},
		URL: "/region-overlay",
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

func createScreenIndicatorWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "screen-indicator",
		Title:            "RecordingFreedom Screen Indicator",
		Width:            screenIndicatorMaxWidth,
		Height:           screenIndicatorMaxHeight,
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		Hidden:           true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				Hide:               true,
				HideTitle:          true,
				FullSizeContent:    true,
			},
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
			WindowLevel: application.MacWindowLevelFloating,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:                   true,
			DisableFramelessWindowDecorations: true,
		},
		Linux: application.LinuxWindow{
			Icon:                appIcon,
			WindowIsTranslucent: true,
		},
		URL: "/screen-indicator",
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
	tray.SetIcon(appIcon)
	tray.SetDarkModeIcon(appIcon)

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
