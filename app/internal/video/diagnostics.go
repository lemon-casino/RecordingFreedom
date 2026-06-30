package video

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

func WriteDiagnostics(path string, diagnostics Diagnostics) error {
	if path == "" {
		return errors.New("video diagnostics path is required")
	}
	if diagnostics.SchemaVersion == 0 {
		diagnostics.SchemaVersion = 1
	}
	if diagnostics.CreatedAt.IsZero() {
		diagnostics.CreatedAt = time.Now()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
