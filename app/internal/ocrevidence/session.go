package ocrevidence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	EvidenceSessionComponent  = "ocr-desktop-evidence"
	EvidenceSessionStartEvent = "session-start"
	EvidenceSessionEndEvent   = "session-end"
)

type SessionMarkerReport struct {
	OK        bool      `json:"ok"`
	DataRoot  string    `json:"dataRoot"`
	SessionID string    `json:"sessionId"`
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	LogPath   string    `json:"logPath"`
}

func NewSessionID(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return fmt.Sprintf("ocr-evidence-%s-%d", now.UTC().Format("20060102T150405.000000000Z"), os.Getpid())
}

func WriteSessionMarker(dataRoot string, event string, sessionID string, timestamp time.Time) (SessionMarkerReport, error) {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return SessionMarkerReport{}, errors.New("data root is required")
	}
	event = strings.TrimSpace(event)
	if event != EvidenceSessionStartEvent && event != EvidenceSessionEndEvent {
		return SessionMarkerReport{}, fmt.Errorf("event must be %q or %q", EvidenceSessionStartEvent, EvidenceSessionEndEvent)
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	} else {
		timestamp = timestamp.UTC()
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		if event != EvidenceSessionStartEvent {
			return SessionMarkerReport{}, errors.New("session id is required for session-end")
		}
		sessionID = NewSessionID(timestamp)
	}
	if strings.ContainsAny(sessionID, "\r\n\t") {
		return SessionMarkerReport{}, errors.New("session id must be a single-line value")
	}
	absRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return SessionMarkerReport{}, err
	}
	logDir := filepath.Join(absRoot, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return SessionMarkerReport{}, err
	}
	logPath := filepath.Join(logDir, "recordingfreedom-"+timestamp.Format("2006-01-02")+".log")
	line, err := json.Marshal(struct {
		Timestamp string            `json:"timestamp"`
		Component string            `json:"component"`
		Event     string            `json:"event"`
		Fields    map[string]string `json:"fields"`
	}{
		Timestamp: timestamp.Format(time.RFC3339Nano),
		Component: EvidenceSessionComponent,
		Event:     event,
		Fields: map[string]string{
			"sessionId": sessionID,
			"goos":      runtime.GOOS,
			"goarch":    runtime.GOARCH,
		},
	})
	if err != nil {
		return SessionMarkerReport{}, err
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return SessionMarkerReport{}, err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		_ = file.Close()
		return SessionMarkerReport{}, err
	}
	if err := file.Close(); err != nil {
		return SessionMarkerReport{}, err
	}
	return SessionMarkerReport{
		OK:        true,
		DataRoot:  absRoot,
		SessionID: sessionID,
		Event:     event,
		Timestamp: timestamp,
		LogPath:   logPath,
	}, nil
}
