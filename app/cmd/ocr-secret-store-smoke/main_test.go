package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesSecretStoreSmokeEvidenceWithoutRawSecret(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "data-root")
	evidenceDir := filepath.Join(t.TempDir(), "evidence")

	report, err := run(dataRoot, evidenceDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK || !report.Saved || !report.Loaded || !report.Deleted || report.LoadAfterDeleteFound || report.RawSecretInDataRoot {
		t.Fatalf("report = %#v, want saved/loaded/deleted without raw data root secret", report)
	}
	if report.SecretBackend == "" || report.SecretStatus == "" || report.SecretName == "" {
		t.Fatalf("secret backend evidence is incomplete: %#v", report)
	}
	data, err := os.ReadFile(filepath.Join(evidenceDir, evidenceFileName))
	if err != nil {
		t.Fatalf("ReadFile(evidence) error = %v", err)
	}
	if strings.Contains(string(data), "rf-secret-store-smoke-value") {
		t.Fatalf("evidence contains raw secret value: %s", data)
	}
	var decoded smokeReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("evidence JSON is invalid: %v", err)
	}
	if !decoded.OK || !decoded.Deleted || decoded.LoadAfterDeleteFound {
		t.Fatalf("decoded evidence = %#v, want deleted secret", decoded)
	}
}
