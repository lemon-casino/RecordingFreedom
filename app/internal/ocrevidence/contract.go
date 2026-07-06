package ocrevidence

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

type VisualRequirement struct {
	Name            string   `json:"name"`
	Terms           []string `json:"terms"`
	Exclude         []string `json:"exclude,omitempty"`
	RecommendedFile string   `json:"recommendedFile"`
	Description     string   `json:"description"`
	MinWidth        int      `json:"minWidth,omitempty"`
	MinHeight       int      `json:"minHeight,omitempty"`
}

type VisualRequirementMatch struct {
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	Terms           []string `json:"terms,omitempty"`
	Exclude         []string `json:"exclude,omitempty"`
	RecommendedFile string   `json:"recommendedFile,omitempty"`
	Description     string   `json:"description,omitempty"`
	MinWidth        int      `json:"minWidth,omitempty"`
	MinHeight       int      `json:"minHeight,omitempty"`
}

type VisualFileDimension struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type CaptureStep struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	SourceKind            string   `json:"sourceKind,omitempty"`
	VisualRequirement     string   `json:"visualRequirement"`
	RecommendedVisualFile string   `json:"recommendedVisualFile"`
	Action                string   `json:"action"`
	RequiredLogEvents     []string `json:"requiredLogEvents"`
	Acceptance            []string `json:"acceptance"`
}

var RequiredSourceKinds = []ocr.SourceKind{
	ocr.SourceRegionScreenshot,
	ocr.SourceFullScreenshot,
	ocr.SourceScrollingScreenshot,
	ocr.SourcePinnedScreenshot,
	ocr.SourceWhiteboard,
	ocr.SourceWhiteboardSelection,
}

var RequiredCaptureSteps = []CaptureStep{
	{
		ID:                    "region-screenshot",
		Title:                 "Region screenshot OCR",
		SourceKind:            string(ocr.SourceRegionScreenshot),
		VisualRequirement:     "region screenshot capture",
		RecommendedVisualFile: "region-screenshot-capture.png",
		Action:                "Use the real desktop region screenshot flow, save the selected region, queue OCR from screenshot history, open the OCR result panel, and read the result image.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourceRegionScreenshot),
		Acceptance:            sourceKindAcceptance("region screenshot", "region-screenshot-capture.png"),
	},
	{
		ID:                    "full-screenshot",
		Title:                 "Full screen OCR",
		SourceKind:            string(ocr.SourceFullScreenshot),
		VisualRequirement:     "full screenshot capture",
		RecommendedVisualFile: "full-screen-capture.png",
		Action:                "Use the real desktop full-screen or all-screen screenshot flow, queue OCR from screenshot history, open the result panel, and read the result image.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourceFullScreenshot),
		Acceptance:            sourceKindAcceptance("full screen screenshot", "full-screen-capture.png"),
	},
	{
		ID:                    "scrolling-screenshot",
		Title:                 "Scrolling screenshot OCR",
		SourceKind:            string(ocr.SourceScrollingScreenshot),
		VisualRequirement:     "scrolling screenshot capture",
		RecommendedVisualFile: "scrolling-screenshot-capture.png",
		Action:                "Use the real scrolling screenshot flow. If the target cannot scroll, capture the explicit no-scroll fallback and record it as scrolling-screenshot. Queue OCR, open the result panel, and read the result image.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourceScrollingScreenshot),
		Acceptance:            sourceKindAcceptance("scrolling screenshot final image", "scrolling-screenshot-capture.png"),
	},
	{
		ID:                    "screenshot-history-ready",
		Title:                 "Screenshot history ready state",
		VisualRequirement:     "screenshot history ready OCR",
		RecommendedVisualFile: "screenshot-history-ready.png",
		Action:                "Open screenshot history after OCR finishes and capture the ready item with result/copy/pin/translation actions visible.",
		RequiredLogEvents:     []string{"ocr/queue-request", "ocr.job.ready"},
		Acceptance:            []string{"History item is ready, not queued or fake-ready.", "Result/copy/pin/translation actions are visible.", "The main capsule size is not expanded by the history panel."},
	},
	{
		ID:                    "ocr-result-floating-panel",
		Title:                 "OCR result floating panel",
		VisualRequirement:     "OCR result floating panel",
		RecommendedVisualFile: "ocr-result-floating-panel.png",
		Action:                "Open a real OCR result from the desktop app and capture the independent floating panel after preview image and polygons render.",
		RequiredLogEvents:     []string{"floating-panel/show kind=ocr-result", "client.ocr-result/preview-loaded", "client.ocr-result/rendered"},
		Acceptance:            []string{"The panel shows the real source image.", "OCR polygons align with the image.", "The main capsule remains at capsule size."},
	},
	{
		ID:                    "pinned-screenshot",
		Title:                 "Pinned screenshot OCR highlight",
		SourceKind:            string(ocr.SourcePinnedScreenshot),
		VisualRequirement:     "pinned screenshot OCR highlight",
		RecommendedVisualFile: "pinned-screenshot-ocr-highlight.png",
		Action:                "Pin a real screenshot, queue pinned OCR, enable highlight, resize or scale the pin window, open the result panel, and read the result image.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourcePinnedScreenshot),
		Acceptance:            sourceKindAcceptance("pinned screenshot window after resize", "pinned-screenshot-ocr-highlight.png"),
	},
	{
		ID:                    "whiteboard",
		Title:                 "Whiteboard OCR",
		SourceKind:            string(ocr.SourceWhiteboard),
		VisualRequirement:     "whiteboard OCR",
		RecommendedVisualFile: "whiteboard-ocr.png",
		Action:                "Open the real whiteboard, save/export a board snapshot, queue OCR, open the result panel, and read the result image.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourceWhiteboard),
		Acceptance:            sourceKindAcceptance("whiteboard OCR result", "whiteboard-ocr.png"),
	},
	{
		ID:                    "whiteboard-selection",
		Title:                 "Whiteboard selected image OCR",
		SourceKind:            string(ocr.SourceWhiteboardSelection),
		VisualRequirement:     "whiteboard selection OCR",
		RecommendedVisualFile: "whiteboard-selection-ocr.png",
		Action:                "Select a real image element inside the whiteboard, export that element as its own PNG, queue OCR as whiteboard-selection, open the result panel, and read the selected image result.",
		RequiredLogEvents:     sourceKindLogEvents(ocr.SourceWhiteboardSelection),
		Acceptance:            []string{"The OCR result uses the selected image element's own imagePath.", "The result panel preview is the selected image, not the whole board.", "Block coordinates align with the selected image.", "Capture `whiteboard-selection-ocr.png` from the real desktop window."},
	},
	{
		ID:                    "recording-annotation",
		Title:                 "Recording annotation OCR safety",
		VisualRequirement:     "recording annotation OCR safety",
		RecommendedVisualFile: "recording-annotation-ocr-safety.png",
		Action:                "Start a real recording, open annotation overlay, save a snapshot into the recording package, queue background OCR, open the result panel, and keep recording active until after the OCR event is visible.",
		RequiredLogEvents:     []string{"annotation-overlay/show packageDir", "annotation-overlay/save-capture packageDir bytes", "ocr/queue-request sourceKind=whiteboard priority=background sourceId=packageDir", "ocr.job.ready sourceKind=whiteboard"},
		Acceptance:            []string{"Recording remains active while OCR is queued.", "Mouse, drawing, and package writes are not blocked by OCR.", "OCR failure, if any, is isolated to OCR state and does not stop recording.", "Capture `recording-annotation-ocr-safety.png` from the real desktop window."},
	},
}

var EvidenceChainRequirements = []string{
	"Every required sourceKind has exactly one exported OCR result JSON in results/.",
	"app-log.jsonl queue/open/read/client render events must match the exported result sourceKind, sourceId, and resultId.",
	"ocr-job-events.jsonl queued and ready events must match the exported result sourceKind, sourceId, and resultId.",
	"recording annotation background OCR sourceId must match the annotation-overlay save-capture packageDir.",
	"No duplicate sourceKind result is allowed in the evidence package.",
}

var EvidenceSessionRunbook = []string{
	"Use the same data-root for the app run, session markers, plan, export, and check.",
	"Before opening OCR surfaces or queueing OCR jobs, run ocr-desktop-evidence-session with -event start and save the returned sessionId.",
	"Capture every required desktop OCR scene inside that one running session.",
	"After the last visual screenshot and OCR result render, run ocr-desktop-evidence-session with -event end -session-id <sessionId>.",
	"Do not create session markers during export; export and check only consume markers from the real desktop run.",
}

var RequiredVisualEvidence = []VisualRequirement{
	{
		Name:            "region screenshot capture",
		Terms:           []string{"region"},
		RecommendedFile: "region-screenshot-capture.png",
		Description:     "Region screenshot selection with OCR-ready text content saved in screenshot history.",
		MinWidth:        320,
		MinHeight:       160,
	},
	{
		Name:            "full screenshot capture",
		Terms:           []string{"full", "screen"},
		RecommendedFile: "full-screen-capture.png",
		Description:     "Full screen or all-screen capture result opened from screenshot history.",
		MinWidth:        640,
		MinHeight:       360,
	},
	{
		Name:            "scrolling screenshot capture",
		Terms:           []string{"scrolling"},
		RecommendedFile: "scrolling-screenshot-capture.png",
		Description:     "Final scrolling screenshot or no-scroll fallback result after the native selection flow.",
		MinWidth:        360,
		MinHeight:       480,
	},
	{
		Name:            "OCR result floating panel",
		Terms:           []string{"ocr-result", "floating"},
		RecommendedFile: "ocr-result-floating-panel.png",
		Description:     "Independent OCR result floating panel showing the preview image and OCR polygons.",
		MinWidth:        360,
		MinHeight:       240,
	},
	{
		Name:            "screenshot history ready OCR",
		Terms:           []string{"history", "ready"},
		RecommendedFile: "screenshot-history-ready.png",
		Description:     "Screenshot history item in ready state with actions available for result, copy, pin, and translation guard/provider.",
		MinWidth:        360,
		MinHeight:       220,
	},
	{
		Name:            "pinned screenshot OCR highlight",
		Terms:           []string{"pinned", "ocr"},
		RecommendedFile: "pinned-screenshot-ocr-highlight.png",
		Description:     "Pinned screenshot window with OCR highlight enabled after resize or scale change.",
		MinWidth:        300,
		MinHeight:       200,
	},
	{
		Name:            "whiteboard OCR",
		Terms:           []string{"whiteboard", "ocr"},
		Exclude:         []string{"selection"},
		RecommendedFile: "whiteboard-ocr.png",
		Description:     "Whiteboard snapshot OCR result/highlight for the whole board, not a selected image element.",
		MinWidth:        480,
		MinHeight:       300,
	},
	{
		Name:            "whiteboard selection OCR",
		Terms:           []string{"whiteboard-selection", "ocr"},
		RecommendedFile: "whiteboard-selection-ocr.png",
		Description:     "Selected Excalidraw image element OCR result using its own exported image.",
		MinWidth:        360,
		MinHeight:       220,
	},
	{
		Name:            "recording annotation OCR safety",
		Terms:           []string{"annotation", "ocr"},
		RecommendedFile: "recording-annotation-ocr-safety.png",
		Description:     "Recording annotation overlay OCR queued in background while recording remains active.",
		MinWidth:        480,
		MinHeight:       300,
	},
}

func sourceKindLogEvents(kind ocr.SourceKind) []string {
	key := string(kind)
	return []string{
		"ocr/queue-request sourceKind=" + key,
		"ocr.job.queued sourceKind=" + key,
		"ocr.job.ready sourceKind=" + key,
		"ocr/open-result sourceKind=" + key,
		"ocr/read-result-image sourceKind=" + key,
		"client.ocr-result/preview-loaded sourceKind=" + key,
		"client.ocr-result/rendered sourceKind=" + key,
	}
}

func sourceKindAcceptance(label string, visualFile string) []string {
	return []string{
		"The real desktop source image is saved and bound to the OCR result.",
		"The result has ready status with at least one OCR block.",
		"The OCR result panel opens and reads the image for " + label + ".",
		"Preview polygons align with the final image pixels.",
		"Capture `" + visualFile + "` from the real desktop window.",
	}
}

func MatchVisualRequirements(files []string) ([]VisualRequirementMatch, error) {
	matches := make([]VisualRequirementMatch, 0, len(RequiredVisualEvidence))
	missing := []string{}
	for _, requirement := range RequiredVisualEvidence {
		path, ok := VisualRequirementMatchPath(files, requirement)
		if !ok {
			missing = append(missing, requirement.Name)
			continue
		}
		matches = append(matches, VisualRequirementMatch{
			Name:            requirement.Name,
			Path:            path,
			Terms:           append([]string(nil), requirement.Terms...),
			Exclude:         append([]string(nil), requirement.Exclude...),
			RecommendedFile: requirement.RecommendedFile,
			Description:     requirement.Description,
			MinWidth:        requirement.MinWidth,
			MinHeight:       requirement.MinHeight,
		})
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("visual evidence is missing files: %s", strings.Join(missing, ", "))
	}
	return matches, nil
}

func VisualDimensionFailures(matches []VisualRequirementMatch, dimensions []VisualFileDimension) []string {
	byPath := map[string]VisualFileDimension{}
	for _, dimension := range dimensions {
		path := strings.ToLower(filepath.ToSlash(strings.TrimSpace(dimension.Path)))
		if path == "" {
			continue
		}
		dimension.Path = path
		byPath[path] = dimension
	}
	failures := []string{}
	for _, match := range matches {
		if match.MinWidth <= 0 && match.MinHeight <= 0 {
			continue
		}
		path := strings.ToLower(filepath.ToSlash(strings.TrimSpace(match.Path)))
		dimension, ok := byPath[path]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: %s is missing image dimensions", match.Name, match.Path))
			continue
		}
		if dimension.Width < match.MinWidth || dimension.Height < match.MinHeight {
			failures = append(failures, fmt.Sprintf("%s: %s is %dx%d, requires at least %dx%d", match.Name, match.Path, dimension.Width, dimension.Height, match.MinWidth, match.MinHeight))
		}
	}
	return failures
}

func VisualRequirementMatchPath(files []string, requirement VisualRequirement) (string, bool) {
	for _, file := range files {
		value := strings.ToLower(filepath.ToSlash(file))
		if VisualFileMatchesRequirement(value, requirement) {
			return value, true
		}
	}
	return "", false
}

func VisualFileMatchesRequirement(file string, requirement VisualRequirement) bool {
	file = strings.ToLower(filepath.ToSlash(file))
	for _, excluded := range requirement.Exclude {
		excluded = strings.ToLower(strings.TrimSpace(excluded))
		if excluded != "" && strings.Contains(file, excluded) {
			return false
		}
	}
	for _, term := range requirement.Terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" && !strings.Contains(file, term) {
			return false
		}
	}
	return true
}
