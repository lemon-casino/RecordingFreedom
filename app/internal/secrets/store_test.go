//go:build !darwin

package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

func TestStoreSavesLoadsAndDeletesSecret(t *testing.T) {
	store := NewStore(appdata.NewService(t.TempDir()))
	store.now = func() time.Time { return time.Date(2026, 7, 6, 1, 2, 3, 0, time.UTC) }

	if err := store.Save("ocr.translation.api-key", "secret-value"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, ok, err := store.Load("ocr.translation.api-key")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok || got != "secret-value" {
		t.Fatalf("Load() = %q %v, want stored secret", got, ok)
	}
	status, err := store.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(status.Dir, "ocr.translation.api-key"+fileSuffix))
	if err != nil {
		t.Fatalf("ReadFile(secret) error = %v", err)
	}
	if strings.Contains(string(data), "secret-value") {
		t.Fatalf("secret file contains raw secret literal: %s", data)
	}
	if err := store.Delete("ocr.translation.api-key"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got, ok, err := store.Load("ocr.translation.api-key"); err != nil || ok || got != "" {
		t.Fatalf("Load(after delete) = %q %v %v, want missing", got, ok, err)
	}
}
