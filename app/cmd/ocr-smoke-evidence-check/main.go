package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	screenshotEvidenceFileName = "screenshot-ocr-real-worker-smoke.json"
	whiteboardEvidenceFileName = "whiteboard-ocr-real-worker-smoke.json"
	defaultExpectedModelID     = "ppocrv5-mobile-zh-en"
)

var requiredScenarios = map[string]string{
	"region":                  "region-screenshot",
	"full":                    "full-screenshot",
	"window":                  "window-screenshot",
	"focused-window":          "focused-window-screenshot",
	"scrolling":               "scrolling-screenshot",
	"scrolling-long":          "scrolling-screenshot",
	"region-queued-cache-hit": "region-screenshot",
}

var requiredWhiteboardScenarios = map[string]string{
	"whiteboard-selection-real-worker": "whiteboard-selection",
}

type report struct {
	OK          bool          `json:"ok"`
	GeneratedAt time.Time     `json:"generatedAt"`
	EvidenceDir string        `json:"evidenceDir"`
	Checks      []checkResult `json:"checks"`
}

type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type evidenceDocument struct {
	SchemaVersion int             `json:"schemaVersion"`
	GeneratedAt   string          `json:"generatedAt"`
	Entries       []evidenceEntry `json:"entries"`
}

type evidenceEntry struct {
	Scenario              string     `json:"scenario"`
	Mode                  string     `json:"mode"`
	SourceKind            string     `json:"sourceKind"`
	SourceID              string     `json:"sourceId"`
	SourceImagePath       string     `json:"sourceImagePath"`
	ElementID             string     `json:"elementId"`
	ScreenshotPath        string     `json:"screenshotPath"`
	ResultID              string     `json:"resultId"`
	ResultImage           string     `json:"resultImage"`
	EvidenceImage         string     `json:"evidenceImage"`
	EvidenceOverlay       string     `json:"evidenceOverlay"`
	ImageWidth            int        `json:"imageWidth"`
	ImageHeight           int        `json:"imageHeight"`
	ModelID               string     `json:"modelId"`
	Language              string     `json:"language"`
	PlainText             string     `json:"plainText"`
	BlockCount            int        `json:"blockCount"`
	Blocks                []ocrBlock `json:"blocks"`
	CacheHitWithoutWorker bool       `json:"cacheHitWithoutWorker"`
	CachedResultID        string     `json:"cachedResultId"`
	QueuedCacheHit        bool       `json:"queuedCacheHit"`
	QueuedCacheResultID   string     `json:"queuedCacheResultId"`
}

type ocrBlock struct {
	ID         string     `json:"id"`
	Text       string     `json:"text"`
	Confidence float64    `json:"confidence"`
	Box        []ocrPoint `json:"box"`
	LineIndex  int        `json:"lineIndex"`
}

type ocrPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func main() {
	var evidenceDir string
	var expectedModelID string
	flag.StringVar(&evidenceDir, "evidence-dir", "", "OCR smoke evidence directory")
	flag.StringVar(&expectedModelID, "expected-model", defaultExpectedModelID, "expected OCR model id in the smoke evidence")
	flag.Parse()

	result, err := runWithExpectedModel(evidenceDir, expectedModelID)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode OCR smoke evidence report: %v\n", err)
		os.Exit(1)
	}
	if !result.OK {
		os.Exit(1)
	}
}

func run(evidenceDir string) (report, error) {
	return runWithExpectedModel(evidenceDir, defaultExpectedModelID)
}

func runWithExpectedModel(evidenceDir string, expectedModelID string) (report, error) {
	if strings.TrimSpace(evidenceDir) == "" {
		return report{}, fmt.Errorf("-evidence-dir is required")
	}
	expectedModelID = strings.TrimSpace(expectedModelID)
	if expectedModelID == "" {
		return report{}, fmt.Errorf("-expected-model is required")
	}
	resolved, err := filepath.Abs(evidenceDir)
	if err != nil {
		return report{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return report{}, err
	}
	if !info.IsDir() {
		return report{}, fmt.Errorf("evidence directory %q is not a directory", resolved)
	}

	result := report{OK: true, GeneratedAt: time.Now().UTC(), EvidenceDir: resolved}
	doc, err := readEvidenceDocument(filepath.Join(resolved, screenshotEvidenceFileName))
	result.addCheck(screenshotEvidenceFileName, err)
	if err != nil {
		result.OK = false
		return result, nil
	}

	byScenario := entriesByScenario(&result, doc.Entries)
	for scenario, wantSourceKind := range requiredScenarios {
		entry, ok := byScenario[scenario]
		if !ok {
			result.addCheck("scenario "+scenario, fmt.Errorf("missing required OCR smoke scenario"))
			continue
		}
		result.addCheck("scenario "+scenario, validateScenario(resolved, scenario, wantSourceKind, expectedModelID, entry))
	}
	result.addCheck("cache hit without worker", validateAnyCacheHitWithoutWorker(byScenario))
	whiteboardDoc, err := readEvidenceDocument(filepath.Join(resolved, whiteboardEvidenceFileName))
	result.addCheck(whiteboardEvidenceFileName, err)
	if err == nil {
		whiteboardByScenario := entriesByScenario(&result, whiteboardDoc.Entries)
		for scenario, wantSourceKind := range requiredWhiteboardScenarios {
			entry, ok := whiteboardByScenario[scenario]
			if !ok {
				result.addCheck("whiteboard scenario "+scenario, fmt.Errorf("missing required OCR smoke scenario"))
				continue
			}
			result.addCheck("whiteboard scenario "+scenario, validateScenario(resolved, scenario, wantSourceKind, expectedModelID, entry))
			result.addCheck("whiteboard selection "+scenario, validateWhiteboardSelection(entry))
		}
	}
	for _, check := range result.Checks {
		if check.Status != "ready" {
			result.OK = false
			break
		}
	}
	return result, nil
}

func entriesByScenario(result *report, entries []evidenceEntry) map[string]evidenceEntry {
	byScenario := map[string]evidenceEntry{}
	for _, entry := range entries {
		scenario := strings.TrimSpace(entry.Scenario)
		if scenario == "" {
			scenario = strings.TrimSpace(entry.Mode)
		}
		if scenario == "" {
			result.addCheck("entry scenario", fmt.Errorf("entry with resultId %q is missing scenario and mode", entry.ResultID))
			continue
		}
		if _, exists := byScenario[scenario]; exists {
			result.addCheck("entry "+scenario, fmt.Errorf("duplicate scenario %q", scenario))
			continue
		}
		byScenario[scenario] = entry
	}
	return byScenario
}

func readEvidenceDocument(path string) (evidenceDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return evidenceDocument{}, err
	}
	var doc evidenceDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return evidenceDocument{}, err
	}
	if doc.SchemaVersion != 1 {
		return evidenceDocument{}, fmt.Errorf("schemaVersion = %d, want 1", doc.SchemaVersion)
	}
	if len(doc.Entries) == 0 {
		return evidenceDocument{}, fmt.Errorf("entries is empty")
	}
	if strings.TrimSpace(doc.GeneratedAt) != "" {
		if _, err := time.Parse(time.RFC3339Nano, doc.GeneratedAt); err != nil {
			return evidenceDocument{}, fmt.Errorf("generatedAt is not RFC3339Nano: %w", err)
		}
	}
	return doc, nil
}

func validateScenario(evidenceDir string, scenario string, wantSourceKind string, expectedModelID string, entry evidenceEntry) error {
	if entry.SourceKind != wantSourceKind {
		return fmt.Errorf("sourceKind = %q, want %q", entry.SourceKind, wantSourceKind)
	}
	for name, value := range map[string]string{
		"sourceId":       entry.SourceID,
		"resultId":       entry.ResultID,
		"screenshotPath": entry.ScreenshotPath,
		"resultImage":    entry.ResultImage,
		"modelId":        entry.ModelID,
		"language":       entry.Language,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if entry.ModelID != expectedModelID {
		return fmt.Errorf("modelId = %q, want %s smoke model", entry.ModelID, expectedModelID)
	}
	if entry.Language != "zh-en" {
		return fmt.Errorf("language = %q, want zh-en", entry.Language)
	}
	for _, text := range []string{"RecordingFreedom", "文字识别"} {
		if !strings.Contains(entry.PlainText, text) {
			return fmt.Errorf("plainText is missing %q", text)
		}
	}
	if entry.ImageWidth <= 0 || entry.ImageHeight <= 0 {
		return fmt.Errorf("image dimensions must be positive, got %dx%d", entry.ImageWidth, entry.ImageHeight)
	}
	if scenario == "scrolling-long" && entry.ImageHeight <= 2400 {
		return fmt.Errorf("scrolling-long imageHeight = %d, want > 2400 to prove tile OCR path", entry.ImageHeight)
	}
	if entry.BlockCount != len(entry.Blocks) || entry.BlockCount == 0 {
		return fmt.Errorf("blockCount = %d and len(blocks) = %d, want matching non-zero values", entry.BlockCount, len(entry.Blocks))
	}
	if err := validateBlocks(entry); err != nil {
		return err
	}
	if scenario == "region-queued-cache-hit" && (!entry.QueuedCacheHit || strings.TrimSpace(entry.QueuedCacheResultID) == "") {
		return fmt.Errorf("region-queued-cache-hit must prove queuedCacheHit with queuedCacheResultId")
	}
	if err := validateEvidenceImage(evidenceDir, entry.EvidenceImage, entry.ImageWidth, entry.ImageHeight); err != nil {
		return fmt.Errorf("evidenceImage: %w", err)
	}
	if err := validateEvidenceImage(evidenceDir, entry.EvidenceOverlay, entry.ImageWidth, entry.ImageHeight); err != nil {
		return fmt.Errorf("evidenceOverlay: %w", err)
	}
	return nil
}

func validateWhiteboardSelection(entry evidenceEntry) error {
	if strings.TrimSpace(entry.ElementID) == "" {
		return fmt.Errorf("elementId is required for whiteboard-selection evidence")
	}
	if strings.TrimSpace(entry.SourceImagePath) == "" {
		return fmt.Errorf("sourceImagePath is required for whiteboard-selection evidence")
	}
	if strings.TrimSpace(entry.ResultImage) == "" {
		return fmt.Errorf("resultImage is required for whiteboard-selection evidence")
	}
	return nil
}

func validateAnyCacheHitWithoutWorker(entries map[string]evidenceEntry) error {
	for scenario, entry := range entries {
		if entry.CacheHitWithoutWorker && strings.TrimSpace(entry.CachedResultID) != "" {
			return nil
		}
		if entry.CacheHitWithoutWorker || strings.TrimSpace(entry.CachedResultID) != "" {
			return fmt.Errorf("scenario %q has partial cache hit proof; cacheHitWithoutWorker and cachedResultId are both required", scenario)
		}
	}
	return fmt.Errorf("no scenario proves cacheHitWithoutWorker with cachedResultId")
}

func validateBlocks(entry evidenceEntry) error {
	for index, block := range entry.Blocks {
		if strings.TrimSpace(block.ID) == "" {
			return fmt.Errorf("block %d is missing id", index)
		}
		if strings.TrimSpace(block.Text) == "" {
			return fmt.Errorf("block %d is missing text", index)
		}
		if block.Confidence <= 0 {
			return fmt.Errorf("block %s confidence = %v, want > 0", block.ID, block.Confidence)
		}
		if len(block.Box) < 4 {
			return fmt.Errorf("block %s has %d box points, want at least 4", block.ID, len(block.Box))
		}
		for pointIndex, point := range block.Box {
			if point.X < -0.5 || point.Y < -0.5 || point.X > float64(entry.ImageWidth)+0.5 || point.Y > float64(entry.ImageHeight)+0.5 {
				return fmt.Errorf("block %s point %d = %.2f,%.2f outside image %dx%d", block.ID, pointIndex, point.X, point.Y, entry.ImageWidth, entry.ImageHeight)
			}
		}
	}
	return nil
}

func validateEvidenceImage(evidenceDir string, path string, wantWidth int, wantHeight int) error {
	resolved, err := evidencePath(evidenceDir, path)
	if err != nil {
		return err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return err
	}
	if config.Width != wantWidth || config.Height != wantHeight {
		return fmt.Errorf("%s dimensions = %dx%d, want %dx%d", resolved, config.Width, config.Height, wantWidth, wantHeight)
	}
	return nil
}

func evidencePath(evidenceDir string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return evidencePathInsideDir(evidenceDir, path)
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative path %q escapes evidence directory", path)
	}
	return evidencePathInsideDir(evidenceDir, filepath.Join(evidenceDir, clean))
}

func evidencePathInsideDir(evidenceDir string, path string) (string, error) {
	absDir, err := filepath.Abs(evidenceDir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if absPath != absDir && !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("evidence path %q must stay inside evidence directory %q", path, evidenceDir)
	}
	return absPath, nil
}

func (r *report) addCheck(name string, err error) {
	if err != nil {
		r.Checks = append(r.Checks, checkResult{Name: name, Status: "blocked", Message: err.Error()})
		return
	}
	r.Checks = append(r.Checks, checkResult{Name: name, Status: "ready"})
}
