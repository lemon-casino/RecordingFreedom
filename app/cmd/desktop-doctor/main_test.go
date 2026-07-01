package main

import (
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/capture"
)

func TestRunReportsManagedDataVideoAndBackend(t *testing.T) {
	report := run(options{dataRoot: t.TempDir(), backend: "native"})
	if !report.OK {
		t.Fatalf("doctor OK = false, checks = %#v", report.Checks)
	}
	if report.DataRoot == "" || report.VideoDir == "" || report.Backend == "" {
		t.Fatalf("report missing core fields: %#v", report)
	}
	if check := findCheck(report.Checks, "data-video"); check == nil || check.Status != checkReady || !check.Required {
		t.Fatalf("data-video check = %#v, want required ready", check)
	}
	if check := findCheck(report.Checks, "backend"); check == nil || check.Status != checkReady {
		t.Fatalf("backend check = %#v, want ready", check)
	}
}

func TestDataVideoCheckRejectsNonManagedPath(t *testing.T) {
	check := dataVideoCheck("/tmp/recordings")
	if check.Status != checkBlocked || !check.Required {
		t.Fatalf("dataVideoCheck() = %#v, want required blocked", check)
	}
}

func TestCapabilityCheckMakesRequiredQueuedCapabilityBlocked(t *testing.T) {
	check := capabilityCheck(capture.Capability{
		ID:      "screen-recording",
		Label:   "Screen Recording",
		Status:  capture.StatusQueued,
		Backend: "queued-backend",
		Reason:  "writer is queued",
	}, true)
	if check.Status != checkBlocked || !check.Required {
		t.Fatalf("capabilityCheck() = %#v, want required blocked", check)
	}
}

func TestRequireRNNoiseMarksUnavailableEnhancementAsBlocking(t *testing.T) {
	report := run(options{dataRoot: t.TempDir(), backend: "native", requireRNNoise: true})
	check := findCheck(report.Checks, "microphone-enhancement")
	if check == nil || !check.Required {
		t.Fatalf("microphone enhancement check = %#v, want required check", check)
	}
	if check.Status == checkQueued || check.Status == checkUnsupported {
		t.Fatalf("microphone enhancement status = %q, want required queued/unsupported promoted to blocked", check.Status)
	}
}

func findCheck(checks []doctorCheck, id string) *doctorCheck {
	for index := range checks {
		if checks[index].ID == id {
			return &checks[index]
		}
	}
	return nil
}
