package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

type options struct {
	dataRoot  string
	event     string
	sessionID string
	timestamp string
}

func main() {
	var opts options
	flag.StringVar(&opts.dataRoot, "data-root", "", "RecordingFreedom data root; defaults to configured app data root")
	flag.StringVar(&opts.event, "event", "", "session event: start or end")
	flag.StringVar(&opts.sessionID, "session-id", "", "desktop evidence session id; generated for start when omitted")
	flag.StringVar(&opts.timestamp, "timestamp", "", "RFC3339Nano timestamp override for deterministic tests")
	flag.Parse()

	report, err := run(opts)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode ocr-desktop-evidence session report: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) (ocrevidence.SessionMarkerReport, error) {
	dataRoot, err := resolveDataRoot(opts.dataRoot)
	if err != nil {
		return ocrevidence.SessionMarkerReport{}, err
	}
	event, err := normalizeEvent(opts.event)
	if err != nil {
		return ocrevidence.SessionMarkerReport{}, err
	}
	timestamp := time.Time{}
	if strings.TrimSpace(opts.timestamp) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(opts.timestamp))
		if err != nil {
			return ocrevidence.SessionMarkerReport{}, fmt.Errorf("invalid -timestamp: %w", err)
		}
		timestamp = parsed
	}
	return ocrevidence.WriteSessionMarker(dataRoot, event, opts.sessionID, timestamp)
}

func resolveDataRoot(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value != "" {
		return value, nil
	}
	service := appdata.NewService("")
	return service.RootDir()
}

func normalizeEvent(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "start", ocrevidence.EvidenceSessionStartEvent:
		return ocrevidence.EvidenceSessionStartEvent, nil
	case "end", ocrevidence.EvidenceSessionEndEvent:
		return ocrevidence.EvidenceSessionEndEvent, nil
	default:
		return "", fmt.Errorf("-event must be start or end")
	}
}
