//go:build darwin

package secrets

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

func TestMacOSKeychainBackendStatus(t *testing.T) {
	store := NewStore(appdata.NewService(t.TempDir()))
	status, err := store.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Backend != "macos-keychain" || status.Dir != "macOS Keychain" {
		t.Fatalf("Status() = %#v, want macOS Keychain backend", status)
	}
}
