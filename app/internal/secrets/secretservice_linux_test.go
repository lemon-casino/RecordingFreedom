//go:build linux

package secrets

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
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
