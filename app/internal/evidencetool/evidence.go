package evidencetool

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// WriteEvidence marshals report as indented JSON and writes it to path,
// creating the parent directory and rejecting an empty path.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-secret-store-smoke,
// cmd/ocr-worker-platform-smoke (identical writeEvidence bodies; the report
// is passed separately from its EvidencePath field).
func WriteEvidence(path string, report any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("evidence path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
