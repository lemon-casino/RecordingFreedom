//go:build linux

package secrets

import (
	"errors"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

func TestLinuxSecretServiceAttributesIdentifyRecordingFreedomSecret(t *testing.T) {
	attrs := secretAttributes("ocr.translation.api-key")
	if attrs["application"] != "RecordingFreedom" || attrs["component"] != "ocr-translation" || attrs["account"] != "ocr.translation.api-key" {
		t.Fatalf("secret attributes = %#v, want app/component/account identity", attrs)
	}
}

func TestLinuxSecretServiceUnavailableErrorsPermitFallback(t *testing.T) {
	if !isSecretServiceUnavailable(dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown"}) {
		t.Fatal("ServiceUnknown should permit local-file fallback")
	}
	if !isSecretServiceUnavailable(errors.New("unable to autolaunch a dbus-daemon without a $DISPLAY for X11")) {
		t.Fatal("missing session bus should permit local-file fallback")
	}
	if isSecretServiceUnavailable(errors.New("Secret Service prompt was dismissed")) {
		t.Fatal("dismissed prompt should not silently downgrade to local-file fallback")
	}
}

func TestLinuxSecretServiceStatusReportsFallbackDirectory(t *testing.T) {
	status, err := NewStore(appdata.NewService(t.TempDir())).Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Backend != secretServiceFallbackName {
		t.Fatalf("Backend = %q, want %q", status.Backend, secretServiceFallbackName)
	}
	if strings.TrimSpace(status.Dir) == "" || strings.Contains(status.Dir, ";") {
		t.Fatalf("Dir = %q, want concrete fallback directory path", status.Dir)
	}
}
