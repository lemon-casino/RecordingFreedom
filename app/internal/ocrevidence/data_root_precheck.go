package ocrevidence

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const (
	dataRootMaxEvidenceSpan       = 6 * time.Hour
	dataRootAppEventWindowPadding = 2 * time.Hour
)

type DataRootPrecheckReport struct {
	DataRoot              string                     `json:"dataRoot"`
	CheckComplete         bool                       `json:"checkComplete"`
	AppLogLines           int                        `json:"appLogLines"`
	JobEventLines         int                        `json:"jobEventLines"`
	ResultFiles           int                        `json:"resultFiles"`
	Session               DataRootPrecheckSession    `json:"session"`
	RunWindow             DataRootPrecheckRunWindow  `json:"runWindow"`
	Sources               []DataRootPrecheckSource   `json:"sources"`
	Annotation            DataRootAnnotationPrecheck `json:"annotation"`
	MissingRequirements   []string                   `json:"missingRequirements,omitempty"`
	UnexpectedSourceKinds []string                   `json:"unexpectedSourceKinds,omitempty"`
	Files                 DataRootPrecheckFiles      `json:"files"`
}

type DataRootPrecheckRunWindow struct {
	ResultStart            time.Time `json:"resultStart,omitempty"`
	ResultEnd              time.Time `json:"resultEnd,omitempty"`
	ResultSpanSeconds      int64     `json:"resultSpanSeconds,omitempty"`
	MaxSpanSeconds         int64     `json:"maxSpanSeconds"`
	AppEventStart          time.Time `json:"appEventStart,omitempty"`
	AppEventEnd            time.Time `json:"appEventEnd,omitempty"`
	AppEventPaddingSeconds int64     `json:"appEventPaddingSeconds"`
}

type DataRootPrecheckFiles struct {
	AppLogs     []string `json:"appLogs,omitempty"`
	JobEvents   string   `json:"jobEvents,omitempty"`
	ResultsRoot string   `json:"resultsRoot,omitempty"`
}

type DataRootPrecheckSession struct {
	SessionID       string    `json:"sessionId,omitempty"`
	Start           time.Time `json:"start,omitempty"`
	End             time.Time `json:"end,omitempty"`
	DurationSeconds int64     `json:"durationSeconds,omitempty"`
	StartLog        string    `json:"startLog,omitempty"`
	EndLog          string    `json:"endLog,omitempty"`
}

type DataRootPrecheckSource struct {
	SourceKind         string    `json:"sourceKind"`
	SourceID           string    `json:"sourceId,omitempty"`
	ResultID           string    `json:"resultId,omitempty"`
	ResultCreatedAt    time.Time `json:"resultCreatedAt,omitempty"`
	ResultReady        bool      `json:"resultReady"`
	ImageReady         bool      `json:"imageReady"`
	JobQueued          bool      `json:"jobQueued"`
	JobReady           bool      `json:"jobReady"`
	AppQueueRequest    bool      `json:"appQueueRequest"`
	AppOpenResult      bool      `json:"appOpenResult"`
	AppReadResultImage bool      `json:"appReadResultImage"`
	ClientPreview      bool      `json:"clientPreview"`
	ClientRendered     bool      `json:"clientRendered"`
	Missing            []string  `json:"missing,omitempty"`
}

type DataRootAnnotationPrecheck struct {
	ShowPackageDir         bool     `json:"showPackageDir"`
	SaveCapturePackageDirs []string `json:"saveCapturePackageDirs,omitempty"`
	BackgroundQueueSources []string `json:"backgroundQueueSources,omitempty"`
	MatchingPackage        bool     `json:"matchingPackage"`
	Missing                []string `json:"missing,omitempty"`
}

type dataRootAppLogEvent struct {
	Timestamp string            `json:"timestamp,omitempty"`
	Component string            `json:"component"`
	Event     string            `json:"event"`
	Fields    map[string]string `json:"fields,omitempty"`
	LogPath   string            `json:"-"`
}

type dataRootJobEvent struct {
	Event      string      `json:"event,omitempty"`
	SourceKind string      `json:"sourceKind,omitempty"`
	SourceID   string      `json:"sourceId,omitempty"`
	Status     string      `json:"status,omitempty"`
	ResultID   string      `json:"resultId,omitempty"`
	Result     *ocr.Result `json:"result,omitempty"`
}

type dataRootEvidenceWindow struct {
	valid    bool
	appStart time.Time
	appEnd   time.Time
}

type dataRootAppEventMatch struct {
	Found        bool
	Timestamp    time.Time
	TimestampRaw string
	TimestampOK  bool
}

func AuditDataRoot(dataRoot string) (DataRootPrecheckReport, error) {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return DataRootPrecheckReport{}, errors.New("data root is required")
	}
	absRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return DataRootPrecheckReport{}, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return DataRootPrecheckReport{}, err
	}
	if !info.IsDir() {
		return DataRootPrecheckReport{}, fmt.Errorf("data root %q is not a directory", absRoot)
	}
	report := DataRootPrecheckReport{
		DataRoot:      absRoot,
		CheckComplete: true,
		Files: DataRootPrecheckFiles{
			JobEvents:   filepath.Join(absRoot, "data", "ocr", "evidence", "ocr-job-events.jsonl"),
			ResultsRoot: filepath.Join(absRoot, "data", "ocr", "results"),
		},
	}

	appLogs, appEvents, appLineCount, appErr := readDataRootAppLogs(absRoot)
	report.Files.AppLogs = appLogs
	report.AppLogLines = appLineCount
	if appErr != nil {
		report.MissingRequirements = append(report.MissingRequirements, appErr.Error())
	}
	jobEvents, jobLineCount, jobErr := readDataRootJobEvents(report.Files.JobEvents)
	report.JobEventLines = jobLineCount
	if jobErr != nil {
		report.MissingRequirements = append(report.MissingRequirements, jobErr.Error())
	}
	results, resultCount, resultErr := readDataRootResults(report.Files.ResultsRoot)
	report.ResultFiles = resultCount
	if resultErr != nil {
		report.MissingRequirements = append(report.MissingRequirements, resultErr.Error())
	}

	latest := latestResultsByKind(results)
	runWindow, evidenceWindow, windowMissing := dataRootRunWindow(latest)
	if len(windowMissing) > 0 {
		report.MissingRequirements = append(report.MissingRequirements, windowMissing...)
	}
	session, sessionMissing := dataRootEvidenceSession(appEvents, runWindow)
	report.Session = session
	if len(sessionMissing) > 0 {
		report.MissingRequirements = append(report.MissingRequirements, sessionMissing...)
	}
	if session.SessionID != "" {
		runWindow.AppEventStart = session.Start
		runWindow.AppEventEnd = session.End
		evidenceWindow = dataRootEvidenceWindow{
			valid:    true,
			appStart: session.Start,
			appEnd:   session.End,
		}
	}
	report.RunWindow = runWindow
	required := map[ocr.SourceKind]bool{}
	for _, kind := range RequiredSourceKinds {
		required[kind] = true
	}
	for kind := range latest {
		if !required[kind] {
			report.UnexpectedSourceKinds = append(report.UnexpectedSourceKinds, string(kind))
		}
	}
	sort.Strings(report.UnexpectedSourceKinds)

	for _, kind := range RequiredSourceKinds {
		source := auditDataRootSource(absRoot, kind, latest[kind], appEvents, jobEvents, evidenceWindow)
		if len(source.Missing) > 0 {
			report.MissingRequirements = append(report.MissingRequirements, source.Missing...)
		}
		report.Sources = append(report.Sources, source)
	}
	report.Annotation = auditDataRootAnnotation(appEvents, evidenceWindow)
	if len(report.Annotation.Missing) > 0 {
		report.MissingRequirements = append(report.MissingRequirements, report.Annotation.Missing...)
	}
	if len(report.UnexpectedSourceKinds) > 0 {
		report.MissingRequirements = append(report.MissingRequirements, "unexpected OCR result sourceKind: "+strings.Join(report.UnexpectedSourceKinds, ", "))
	}
	report.CheckComplete = len(report.MissingRequirements) == 0
	return report, nil
}

func auditDataRootSource(dataRoot string, kind ocr.SourceKind, result ocr.Result, appEvents []dataRootAppLogEvent, jobEvents []dataRootJobEvent, window dataRootEvidenceWindow) DataRootPrecheckSource {
	source := DataRootPrecheckSource{SourceKind: string(kind)}
	if result.SourceKind != "" {
		source.ResultReady = true
		source.ResultID = result.ID
		source.SourceID = result.SourceID
		source.ResultCreatedAt = result.CreatedAt
		source.ImageReady = dataRootImageExists(dataRoot, result.ImagePath)
	}
	if !source.ResultReady {
		source.Missing = append(source.Missing, "missing OCR result sourceKind="+string(kind))
		return source
	}
	if result.CreatedAt.IsZero() {
		source.Missing = append(source.Missing, "missing OCR result createdAt sourceKind="+string(kind)+" resultId="+source.ResultID)
	}
	if !source.ImageReady {
		source.Missing = append(source.Missing, "missing OCR result image sourceKind="+string(kind)+" resultId="+source.ResultID)
	}
	source.JobQueued = dataRootJobQueued(jobEvents, result)
	source.JobReady = dataRootJobReady(jobEvents, result)
	queueRequest := dataRootAppEvent(appEvents, "ocr", "queue-request", result, false, window)
	openResult := dataRootAppEvent(appEvents, "ocr", "open-result", result, true, window)
	readResultImage := dataRootAppEvent(appEvents, "ocr", "read-result-image", result, true, window)
	clientPreview := dataRootAppEvent(appEvents, "client.ocr-result", "preview-loaded", result, true, window)
	clientRendered := dataRootAppEvent(appEvents, "client.ocr-result", "rendered", result, true, window)
	source.AppQueueRequest = queueRequest.Found
	source.AppOpenResult = openResult.Found
	source.AppReadResultImage = readResultImage.Found
	source.ClientPreview = clientPreview.Found
	source.ClientRendered = clientRendered.Found
	if !source.JobQueued {
		source.Missing = append(source.Missing, "missing ocr-job-events queued sourceKind="+string(kind)+" sourceId="+result.SourceID)
	}
	if !source.JobReady {
		source.Missing = append(source.Missing, "missing ocr-job-events ready sourceKind="+string(kind)+" sourceId="+result.SourceID+" resultId="+result.ID)
	}
	source.Missing = append(source.Missing, dataRootAppEventMissing("app-log", "queue-request", kind, result, false, queueRequest, window)...)
	source.Missing = append(source.Missing, dataRootAppEventMissing("app-log", "open-result", kind, result, true, openResult, window)...)
	source.Missing = append(source.Missing, dataRootAppEventMissing("app-log", "read-result-image", kind, result, true, readResultImage, window)...)
	source.Missing = append(source.Missing, dataRootAppEventMissing("client", "preview-loaded", kind, result, true, clientPreview, window)...)
	source.Missing = append(source.Missing, dataRootAppEventMissing("client", "rendered", kind, result, true, clientRendered, window)...)
	return source
}

func dataRootRunWindow(latest map[ocr.SourceKind]ocr.Result) (DataRootPrecheckRunWindow, dataRootEvidenceWindow, []string) {
	report := DataRootPrecheckRunWindow{
		MaxSpanSeconds:         int64(dataRootMaxEvidenceSpan.Seconds()),
		AppEventPaddingSeconds: int64(dataRootAppEventWindowPadding.Seconds()),
	}
	missing := []string{}
	var start time.Time
	var end time.Time
	validCount := 0
	for _, kind := range RequiredSourceKinds {
		result := latest[kind]
		if result.SourceKind == "" {
			continue
		}
		if result.CreatedAt.IsZero() {
			continue
		}
		if validCount == 0 || result.CreatedAt.Before(start) {
			start = result.CreatedAt
		}
		if validCount == 0 || result.CreatedAt.After(end) {
			end = result.CreatedAt
		}
		validCount++
	}
	if validCount == 0 {
		return report, dataRootEvidenceWindow{}, missing
	}
	report.ResultStart = start
	report.ResultEnd = end
	report.ResultSpanSeconds = int64(end.Sub(start).Seconds())
	report.AppEventStart = start.Add(-dataRootAppEventWindowPadding)
	report.AppEventEnd = end.Add(dataRootAppEventWindowPadding)
	if end.Sub(start) > dataRootMaxEvidenceSpan {
		missing = append(missing, "OCR result createdAt span exceeds evidence window max: "+end.Sub(start).String()+" > "+dataRootMaxEvidenceSpan.String())
	}
	return report, dataRootEvidenceWindow{
		valid:    true,
		appStart: report.AppEventStart,
		appEnd:   report.AppEventEnd,
	}, missing
}

func dataRootEvidenceSession(events []dataRootAppLogEvent, runWindow DataRootPrecheckRunWindow) (DataRootPrecheckSession, []string) {
	type marker struct {
		sessionID string
		event     string
		timestamp time.Time
		logPath   string
	}
	markers := []marker{}
	missing := []string{}
	for _, event := range events {
		if event.Component != EvidenceSessionComponent {
			continue
		}
		if event.Event != EvidenceSessionStartEvent && event.Event != EvidenceSessionEndEvent {
			continue
		}
		sessionID := strings.TrimSpace(event.Fields["sessionId"])
		if sessionID == "" {
			missing = append(missing, "missing ocr-desktop-evidence "+event.Event+" sessionId")
			continue
		}
		timestamp, ok := dataRootEventTimestamp(event)
		if !ok {
			missing = append(missing, "missing app-log timestamp event="+event.Event+" component="+EvidenceSessionComponent+" sessionId="+sessionID)
			continue
		}
		markers = append(markers, marker{
			sessionID: sessionID,
			event:     event.Event,
			timestamp: timestamp,
			logPath:   event.LogPath,
		})
	}
	var selected DataRootPrecheckSession
	var latest DataRootPrecheckSession
	foundWrapping := false
	for _, start := range markers {
		if start.event != EvidenceSessionStartEvent {
			continue
		}
		for _, end := range markers {
			if end.event != EvidenceSessionEndEvent || end.sessionID != start.sessionID || end.timestamp.Before(start.timestamp) {
				continue
			}
			pair := DataRootPrecheckSession{
				SessionID:       start.sessionID,
				Start:           start.timestamp,
				End:             end.timestamp,
				DurationSeconds: int64(end.timestamp.Sub(start.timestamp).Seconds()),
				StartLog:        start.logPath,
				EndLog:          end.logPath,
			}
			if latest.SessionID == "" || pair.End.After(latest.End) {
				latest = pair
			}
			if dataRootSessionWrapsRunWindow(pair, runWindow) && (!foundWrapping || pair.End.After(selected.End)) {
				selected = pair
				foundWrapping = true
			}
		}
	}
	if selected.SessionID == "" {
		selected = latest
	}
	if selected.SessionID == "" {
		missing = append(missing, "missing ocr-desktop-evidence session-start/session-end markers")
		return selected, missing
	}
	if !runWindow.ResultStart.IsZero() && selected.Start.After(runWindow.ResultStart) {
		missing = append(missing, "ocr-desktop-evidence session starts after OCR result window sessionId="+selected.SessionID)
	}
	if !runWindow.ResultEnd.IsZero() && selected.End.Before(runWindow.ResultEnd) {
		missing = append(missing, "ocr-desktop-evidence session ends before OCR result window sessionId="+selected.SessionID)
	}
	return selected, missing
}

func dataRootSessionWrapsRunWindow(session DataRootPrecheckSession, runWindow DataRootPrecheckRunWindow) bool {
	if session.SessionID == "" {
		return false
	}
	if !runWindow.ResultStart.IsZero() && session.Start.After(runWindow.ResultStart) {
		return false
	}
	if !runWindow.ResultEnd.IsZero() && session.End.Before(runWindow.ResultEnd) {
		return false
	}
	return true
}

func dataRootAppEventMissing(prefix string, name string, kind ocr.SourceKind, result ocr.Result, requireResultID bool, match dataRootAppEventMatch, window dataRootEvidenceWindow) []string {
	if !match.Found {
		message := "missing " + prefix + " " + name + " sourceKind=" + string(kind) + " sourceId=" + result.SourceID
		if requireResultID {
			message += " resultId=" + result.ID
		}
		return []string{message}
	}
	if !match.TimestampOK {
		return []string{"missing app-log timestamp event=" + name + " sourceKind=" + string(kind) + " sourceId=" + result.SourceID + " resultId=" + result.ID}
	}
	if window.valid && (match.Timestamp.Before(window.appStart) || match.Timestamp.After(window.appEnd)) {
		return []string{"app-log timestamp outside OCR evidence window event=" + name + " sourceKind=" + string(kind) + " sourceId=" + result.SourceID + " resultId=" + result.ID + " timestamp=" + match.Timestamp.Format(time.RFC3339Nano)}
	}
	return nil
}

func auditDataRootAnnotation(events []dataRootAppLogEvent, window dataRootEvidenceWindow) DataRootAnnotationPrecheck {
	report := DataRootAnnotationPrecheck{}
	savePackages := map[string]bool{}
	backgroundSources := map[string]bool{}
	for _, event := range events {
		if window.valid && !dataRootEventInWindow(event, window) {
			continue
		}
		switch {
		case event.Component == "annotation-overlay" && event.Event == "show" && strings.TrimSpace(event.Fields["packageDir"]) != "":
			report.ShowPackageDir = true
		case event.Component == "annotation-overlay" && event.Event == "save-capture":
			packageDir := strings.TrimSpace(event.Fields["packageDir"])
			if packageDir != "" && strings.TrimSpace(event.Fields["bytes"]) != "" {
				savePackages[packageDir] = true
			}
		case event.Component == "ocr" && event.Event == "queue-request":
			if strings.TrimSpace(event.Fields["sourceKind"]) == string(ocr.SourceWhiteboard) && strings.TrimSpace(event.Fields["priority"]) == ocr.JobPriorityBackground {
				if sourceID := strings.TrimSpace(event.Fields["sourceId"]); sourceID != "" {
					backgroundSources[sourceID] = true
				}
			}
		}
	}
	report.SaveCapturePackageDirs = sortedKeys(savePackages)
	report.BackgroundQueueSources = sortedKeys(backgroundSources)
	for packageDir := range savePackages {
		if backgroundSources[packageDir] {
			report.MatchingPackage = true
			break
		}
	}
	if !report.ShowPackageDir {
		report.Missing = append(report.Missing, "missing annotation-overlay/show packageDir")
	}
	if len(report.SaveCapturePackageDirs) == 0 {
		report.Missing = append(report.Missing, "missing annotation-overlay/save-capture packageDir bytes")
	}
	if len(report.BackgroundQueueSources) == 0 {
		report.Missing = append(report.Missing, "missing ocr/queue-request sourceKind=whiteboard priority=background sourceId")
	}
	if len(report.SaveCapturePackageDirs) > 0 && len(report.BackgroundQueueSources) > 0 && !report.MatchingPackage {
		report.Missing = append(report.Missing, "missing recording annotation background OCR sourceId matching save-capture packageDir")
	}
	return report
}

func readDataRootAppLogs(dataRoot string) ([]string, []dataRootAppLogEvent, int, error) {
	matches, err := filepath.Glob(filepath.Join(dataRoot, "logs", "recordingfreedom-*.log"))
	if err != nil {
		return nil, nil, 0, err
	}
	if len(matches) == 0 {
		return nil, nil, 0, errors.New("missing app logs under logs/recordingfreedom-*.log")
	}
	sort.Strings(matches)
	events := []dataRootAppLogEvent{}
	lines := 0
	for _, path := range matches {
		file, err := os.Open(path)
		if err != nil {
			return matches, nil, lines, err
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			lines++
			var event dataRootAppLogEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				_ = file.Close()
				return matches, nil, lines, fmt.Errorf("%s line %d is invalid JSON: %w", path, lines, err)
			}
			if event.Fields == nil {
				event.Fields = map[string]string{}
			}
			event.LogPath = path
			events = append(events, event)
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return matches, nil, lines, scanErr
		}
		if closeErr != nil {
			return matches, nil, lines, closeErr
		}
	}
	if lines == 0 {
		return matches, nil, 0, errors.New("app logs contain no JSONL events")
	}
	return matches, events, lines, nil
}

func readDataRootJobEvents(path string) ([]dataRootJobEvent, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("missing OCR job events file: %w", err)
	}
	defer file.Close()
	events := []dataRootJobEvent{}
	lines := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines++
		var event dataRootJobEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, lines, fmt.Errorf("%s line %d is invalid JSON: %w", path, lines, err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, lines, err
	}
	if lines == 0 {
		return nil, 0, errors.New("OCR job events file contains no JSONL events")
	}
	return events, lines, nil
}

func readDataRootResults(dir string) ([]ocr.Result, int, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("missing OCR results directory: %w", err)
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("%s is not a directory", dir)
	}
	results := []ocr.Result{}
	count := 0
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}
		count++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var result ocr.Result
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("%s is not an OCR result JSON: %w", path, err)
		}
		if result.SourceKind != "" {
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, count, err
	}
	if count == 0 {
		return nil, 0, errors.New("OCR results directory contains no result JSON files")
	}
	return results, count, nil
}

func latestResultsByKind(results []ocr.Result) map[ocr.SourceKind]ocr.Result {
	byKind := map[ocr.SourceKind]ocr.Result{}
	for _, result := range results {
		current, exists := byKind[result.SourceKind]
		if !exists || result.CreatedAt.After(current.CreatedAt) {
			byKind[result.SourceKind] = result
		}
	}
	return byKind
}

func dataRootImageExists(dataRoot string, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	candidate := value
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(dataRoot, filepath.FromSlash(value))
	}
	info, err := os.Stat(candidate)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func dataRootAppEvent(events []dataRootAppLogEvent, component string, name string, result ocr.Result, requireResultID bool, window dataRootEvidenceWindow) dataRootAppEventMatch {
	fallback := dataRootAppEventMatch{}
	for _, event := range events {
		if event.Component != component || event.Event != name {
			continue
		}
		if strings.TrimSpace(event.Fields["sourceKind"]) != string(result.SourceKind) {
			continue
		}
		if strings.TrimSpace(event.Fields["sourceId"]) != result.SourceID {
			continue
		}
		if requireResultID && strings.TrimSpace(event.Fields["resultId"]) != result.ID {
			continue
		}
		match := dataRootAppEventMatch{Found: true, TimestampRaw: strings.TrimSpace(event.Timestamp)}
		if match.TimestampRaw != "" {
			if timestamp, err := time.Parse(time.RFC3339Nano, match.TimestampRaw); err == nil {
				match.Timestamp = timestamp
				match.TimestampOK = true
				if !window.valid || (!timestamp.Before(window.appStart) && !timestamp.After(window.appEnd)) {
					return match
				}
				if !fallback.Found {
					fallback = match
				}
				continue
			}
		}
		if !fallback.Found {
			fallback = match
		}
	}
	return fallback
}

func dataRootEventTimestamp(event dataRootAppLogEvent) (time.Time, bool) {
	timestampRaw := strings.TrimSpace(event.Timestamp)
	if timestampRaw == "" {
		return time.Time{}, false
	}
	timestamp, err := time.Parse(time.RFC3339Nano, timestampRaw)
	return timestamp, err == nil
}

func dataRootEventInWindow(event dataRootAppLogEvent, window dataRootEvidenceWindow) bool {
	timestamp, ok := dataRootEventTimestamp(event)
	if !ok {
		return false
	}
	return !timestamp.Before(window.appStart) && !timestamp.After(window.appEnd)
}

func dataRootJobQueued(events []dataRootJobEvent, result ocr.Result) bool {
	for _, event := range events {
		if dataRootJobSourceKind(event) == string(result.SourceKind) && strings.TrimSpace(event.SourceID) == result.SourceID && strings.TrimSpace(event.Status) == ocr.ResultStatusQueued {
			return true
		}
	}
	return false
}

func dataRootJobReady(events []dataRootJobEvent, result ocr.Result) bool {
	for _, event := range events {
		if dataRootJobSourceKind(event) != string(result.SourceKind) || strings.TrimSpace(event.SourceID) != result.SourceID {
			continue
		}
		if strings.TrimSpace(event.Status) != ocr.ResultStatusReady {
			continue
		}
		if dataRootJobResultID(event) == result.ID {
			return true
		}
	}
	return false
}

func dataRootJobSourceKind(event dataRootJobEvent) string {
	sourceKind := strings.TrimSpace(event.SourceKind)
	if sourceKind == "" && event.Result != nil {
		sourceKind = string(event.Result.SourceKind)
	}
	return sourceKind
}

func dataRootJobResultID(event dataRootJobEvent) string {
	resultID := strings.TrimSpace(event.ResultID)
	if resultID == "" && event.Result != nil {
		resultID = strings.TrimSpace(event.Result.ID)
	}
	return resultID
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
