package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

func TestRunWritesStartAndEndMarkers(t *testing.T) {
	root := t.TempDir()
	start, err := run(options{
		dataRoot:  root,
		event:     "start",
		sessionID: "session-test",
		timestamp: "2026-07-06T08:00:00Z",
	})
	if err != nil {
		t.Fatalf("start run() error = %v", err)
	}
	if !start.OK || start.SessionID != "session-test" || start.Event != ocrevidence.EvidenceSessionStartEvent {
		t.Fatalf("start report = %#v", start)
	}
	end, err := run(options{
		dataRoot:  root,
		event:     "end",
		sessionID: "session-test",
		timestamp: "2026-07-06T08:05:00Z",
	})
	if err != nil {
		t.Fatalf("end run() error = %v", err)
	}
	if !end.OK || end.SessionID != "session-test" || end.Event != ocrevidence.EvidenceSessionEndEvent {
		t.Fatalf("end report = %#v", end)
	}
	logPath := filepath.Join(root, "logs", "recordingfreedom-2026-07-06.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", logPath, err)
	}
	content := string(data)
	for _, needle := range []string{
		`"component":"ocr-desktop-evidence"`,
		`"event":"session-start"`,
		`"event":"session-end"`,
		`"sessionId":"session-test"`,
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("session log missing %q:\n%s", needle, content)
		}
	}
}

func TestRunRequiresSessionIDForEnd(t *testing.T) {
	_, err := run(options{
		dataRoot:  t.TempDir(),
		event:     "end",
		timestamp: "2026-07-06T08:05:00Z",
	})
	if err == nil || !strings.Contains(err.Error(), "session id is required") {
		t.Fatalf("run() error = %v, want session id required", err)
	}
}
