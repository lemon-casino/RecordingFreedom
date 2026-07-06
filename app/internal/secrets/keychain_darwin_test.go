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
	if status.Backend != keychainFallbackName || status.Dir == "" {
		t.Fatalf("Status() = %#v, want macOS Keychain backend with explicit fallback", status)
	}
}

func TestMacOSKeychainUnavailableErrorsPermitFallback(t *testing.T) {
	if !isKeychainUnavailable(keychainStatusError{operation: "save secret", status: -25307}) {
		t.Fatal("missing default keychain should permit local-file fallback")
	}
	if !isKeychainUnavailable(keychainStatusError{operation: "save secret", status: -25308}) {
		t.Fatal("headless interaction denial should permit local-file fallback")
	}
	if isKeychainUnavailable(keychainStatusError{operation: "save secret", status: -25299}) {
		t.Fatal("unexpected Keychain errors should not silently downgrade to local-file fallback")
	}
}
