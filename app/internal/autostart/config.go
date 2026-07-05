package autostart

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	AppName    = "RecordingFreedom"
	AppID      = "com.lemon-casino.recordingfreedom"
	StartupArg = "--minimized-to-tray"
)

type Config struct {
	AppName    string
	AppID      string
	Executable string
	Args       []string
}

func DefaultConfig() (Config, error) {
	executable, err := os.Executable()
	if err != nil {
		return Config{}, fmt.Errorf("resolve executable for start at login: %w", err)
	}
	return normalizeConfig(Config{
		AppName:    AppName,
		AppID:      AppID,
		Executable: executable,
		Args:       []string{StartupArg},
	})
}

func SetEnabled(enabled bool) error {
	config, err := DefaultConfig()
	if err != nil {
		return err
	}
	if enabled {
		return Enable(config)
	}
	return Disable(config)
}

func normalizeConfig(config Config) (Config, error) {
	config.AppName = strings.TrimSpace(config.AppName)
	config.AppID = strings.TrimSpace(config.AppID)
	config.Executable = strings.TrimSpace(config.Executable)
	if config.AppName == "" {
		config.AppName = AppName
	}
	if config.AppID == "" {
		config.AppID = AppID
	}
	if config.Executable == "" {
		return Config{}, errors.New("start at login executable path is required")
	}
	args := make([]string, 0, len(config.Args))
	for _, arg := range config.Args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			args = append(args, arg)
		}
	}
	if len(args) == 0 {
		args = []string{StartupArg}
	}
	config.Args = args
	return config, nil
}

func (config Config) programArguments() []string {
	args := make([]string, 0, len(config.Args)+1)
	args = append(args, config.Executable)
	args = append(args, config.Args...)
	return args
}

func commandLine(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, quoteCommandPart(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandPart(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\n\r\"") {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func launchAgentPlist(config Config) string {
	args := config.programArguments()
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	builder.WriteString(`<plist version="1.0">` + "\n")
	builder.WriteString("<dict>\n")
	builder.WriteString("\t<key>Label</key>\n")
	builder.WriteString("\t<string>" + escapeXML(config.AppID) + "</string>\n")
	builder.WriteString("\t<key>ProgramArguments</key>\n")
	builder.WriteString("\t<array>\n")
	for _, arg := range args {
		builder.WriteString("\t\t<string>" + escapeXML(arg) + "</string>\n")
	}
	builder.WriteString("\t</array>\n")
	builder.WriteString("\t<key>RunAtLoad</key>\n")
	builder.WriteString("\t<true/>\n")
	builder.WriteString("</dict>\n")
	builder.WriteString("</plist>\n")
	return builder.String()
}

func desktopEntry(config Config) string {
	return strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Version=1.0",
		"Name=" + desktopValue(config.AppName),
		"Comment=Start RecordingFreedom minimized to tray",
		"Exec=" + desktopExec(config.programArguments()),
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n")
}

func desktopValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func desktopExec(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, quoteDesktopExecPart(arg))
	}
	return strings.Join(parts, " ")
}

func quoteDesktopExecPart(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\n\r\"\\") {
		return value
	}
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", " ", "\r", " ")
	return `"` + replacer.Replace(value) + `"`
}

func escapeXML(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}
