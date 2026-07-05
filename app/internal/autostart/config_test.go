package autostart

import (
	"strings"
	"testing"
)

func TestDefaultConfigUsesMinimizedTrayArgument(t *testing.T) {
	config, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error = %v", err)
	}
	if config.AppName != AppName || config.AppID != AppID {
		t.Fatalf("default app identity = %q/%q, want %q/%q", config.AppName, config.AppID, AppName, AppID)
	}
	if config.Executable == "" {
		t.Fatal("default executable is empty")
	}
	if len(config.Args) != 1 || config.Args[0] != StartupArg {
		t.Fatalf("default args = %#v, want %q", config.Args, StartupArg)
	}
}

func TestCommandLineQuotesExecutableWithSpaces(t *testing.T) {
	got := commandLine([]string{`C:\Program Files\RecordingFreedom\RecordingFreedom.exe`, StartupArg})
	want := `"C:\Program Files\RecordingFreedom\RecordingFreedom.exe" --minimized-to-tray`
	if got != want {
		t.Fatalf("command line = %q, want %q", got, want)
	}
}

func TestLaunchAgentPlistIncludesProgramArguments(t *testing.T) {
	config, err := normalizeConfig(Config{
		AppName:    AppName,
		AppID:      AppID,
		Executable: `/Applications/RecordingFreedom & Tools.app/Contents/MacOS/RecordingFreedom`,
		Args:       []string{StartupArg},
	})
	if err != nil {
		t.Fatalf("normalizeConfig() error = %v", err)
	}
	plist := launchAgentPlist(config)
	for _, want := range []string{
		"<string>com.lemon-casino.recordingfreedom</string>",
		"/Applications/RecordingFreedom &amp; Tools.app/Contents/MacOS/RecordingFreedom",
		"<string>--minimized-to-tray</string>",
		"<key>RunAtLoad</key>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("plist missing %q:\n%s", want, plist)
		}
	}
}

func TestDesktopEntryQuotesExecPath(t *testing.T) {
	config, err := normalizeConfig(Config{
		AppName:    AppName,
		AppID:      AppID,
		Executable: `/opt/Recording Freedom/RecordingFreedom`,
		Args:       []string{StartupArg},
	})
	if err != nil {
		t.Fatalf("normalizeConfig() error = %v", err)
	}
	entry := desktopEntry(config)
	for _, want := range []string{
		"Type=Application",
		"Name=RecordingFreedom",
		`Exec="/opt/Recording Freedom/RecordingFreedom" --minimized-to-tray`,
		"X-GNOME-Autostart-enabled=true",
	} {
		if !strings.Contains(entry, want) {
			t.Fatalf("desktop entry missing %q:\n%s", want, entry)
		}
	}
}
