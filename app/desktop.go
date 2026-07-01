package main

import (
	"sync"

	"github.com/lemon-casino/RecordingFreedom/app/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	capsuleWindowWidth           = 960
	capsuleWindowCollapsedHeight = 112
	capsuleWindowExpandedHeight  = 640
)

func createCapsuleWindow(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:           "capsule-recorder",
		Title:          "RecordingFreedom",
		Width:          capsuleWindowWidth,
		Height:         capsuleWindowCollapsedHeight,
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		HideOnEscape:   true,
		BackgroundType: application.BackgroundTypeTransparent,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
				Hide:               true,
				HideTitle:          true,
				FullSizeContent:    true,
			},
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:                   true,
			DisableFramelessWindowDecorations: true,
		},
		Linux: application.LinuxWindow{
			Icon:                appIcon,
			WindowIsTranslucent: true,
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
		URL:              "/#/settings",
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
		URL: "/#/region-overlay",
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
		URL: "/#/screen-indicator",
	})

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	return window
}

type trayMenuCopy struct {
	ShowRecorder string
	ShowSettings string
	HideRecorder string
	Quit         string
}

func trayCopy(locale settings.Locale) trayMenuCopy {
	switch locale {
	case settings.LocaleEN:
		return trayMenuCopy{
			ShowRecorder: "Show Recorder",
			ShowSettings: "Show Settings",
			HideRecorder: "Hide Recorder",
			Quit:         "Quit RecordingFreedom",
		}
	default:
		return trayMenuCopy{
			ShowRecorder: "显示录制窗口",
			ShowSettings: "显示设置",
			HideRecorder: "隐藏录制窗口",
			Quit:         "退出 RecordingFreedom",
		}
	}
}

func configureSystemTray(app *application.App, recorderWindow *application.WebviewWindow, settingsWindow *application.WebviewWindow, initialLocale settings.Locale) func(settings.Locale) {
	tray := app.SystemTray.New()
	tray.SetTooltip("RecordingFreedom")
	tray.SetIcon(appIcon)
	tray.SetDarkModeIcon(appIcon)

	var menuMu sync.Mutex
	applyMenu := func(locale settings.Locale) {
		copy := trayCopy(locale)
		menu := app.NewMenu()
		menu.Add(copy.ShowRecorder).OnClick(func(ctx *application.Context) {
			tray.ShowWindow()
		})
		menu.Add(copy.ShowSettings).OnClick(func(ctx *application.Context) {
			settingsWindow.Show()
			settingsWindow.Focus()
		})
		menu.Add(copy.HideRecorder).OnClick(func(ctx *application.Context) {
			tray.HideWindow()
		})
		menu.AddSeparator()
		menu.Add(copy.Quit).OnClick(func(ctx *application.Context) {
			app.Quit()
		})
		menuMu.Lock()
		defer menuMu.Unlock()
		tray.SetMenu(menu)
	}
	applyMenu(initialLocale)

	tray.AttachWindow(recorderWindow).WindowOffset(10)
	tray.OnClick(func() {
		tray.ToggleWindow()
	})
	tray.OnRightClick(func() {
		tray.OpenMenu()
	})

	return applyMenu
}
