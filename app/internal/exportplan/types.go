package exportplan

import "github.com/lemon-casino/RecordingFreedom/app/internal/pip"

const (
	DefaultOutputPath = "exports/recording.mp4"
)

type Request struct {
	VideoDir                string
	PackageDir              string
	OutputPath              string
	Canvas                  pip.Size
	RequireSync             bool
	AllowMock               bool
	IncludeAnnotations      *bool
	PrepareAnnotationAssets bool
}

type Plan struct {
	PackageDir              string                       `json:"packageDir"`
	ManifestPath            string                       `json:"manifestPath"`
	OutputPath              string                       `json:"outputPath"`
	ScreenInputPath         string                       `json:"screenInputPath"`
	WebcamInputPath         string                       `json:"webcamInputPath,omitempty"`
	WebcamStartOffsetMs     int                          `json:"webcamStartOffsetMs,omitempty"`
	AnnotationInputPath     string                       `json:"annotationInputPath,omitempty"`
	AnnotationEventsPath    string                       `json:"annotationEventsPath,omitempty"`
	AnnotationStartMs       int64                        `json:"annotationStartMs,omitempty"`
	AnnotationTimeline      string                       `json:"annotationTimeline,omitempty"`
	AnnotationSnapshots     []AnnotationSnapshotPlan     `json:"annotationSnapshots,omitempty"`
	AnnotationRenderMode    string                       `json:"annotationRenderMode,omitempty"`
	AnnotationElementScenes []AnnotationElementScenePlan `json:"annotationElementScenes,omitempty"`
	AnnotationSummary       *AnnotationTimelineSummary   `json:"annotationSummary,omitempty"`
	AnnotationsVisible      bool                         `json:"annotationsVisible"`
	PIPPreset               string                       `json:"pipPreset"`
	PIPConfig               pip.Config                   `json:"pipConfig"`
	PIPRect                 pip.Rect                     `json:"pipRect"`
	PIPLayout               pip.Placement                `json:"pipLayout"`
	TimelineBase            string                       `json:"timelineBase,omitempty"`
	AudioDiagnosticsPath    string                       `json:"audioDiagnosticsPath,omitempty"`
	VideoDiagnosticsPath    string                       `json:"videoDiagnosticsPath,omitempty"`
	PauseSegments           []PauseSegmentPlan           `json:"pauseSegments,omitempty"`
	Warnings                []string                     `json:"warnings,omitempty"`
}

type PauseSegmentPlan struct {
	StartOffsetMs int64 `json:"startOffsetMs"`
	EndOffsetMs   int64 `json:"endOffsetMs"`
	DurationMs    int64 `json:"durationMs"`
}

type AnnotationSnapshotPlan struct {
	InputPath     string `json:"inputPath"`
	RelativePath  string `json:"relativePath,omitempty"`
	StartOffsetMs int64  `json:"startOffsetMs"`
	EndOffsetMs   int64  `json:"endOffsetMs,omitempty"`
	DurationMs    int64  `json:"durationMs,omitempty"`
	Bytes         int64  `json:"bytes,omitempty"`
}

type AnnotationTimelineSummary struct {
	Mode                   string                          `json:"mode,omitempty"`
	EventCount             int                             `json:"eventCount,omitempty"`
	SnapshotCount          int                             `json:"snapshotCount,omitempty"`
	ExportedSnapshotCount  int                             `json:"exportedSnapshotCount,omitempty"`
	SkippedSnapshotCount   int                             `json:"skippedSnapshotCount,omitempty"`
	ElementEventCount      int                             `json:"elementEventCount,omitempty"`
	ElementTimelineMode    string                          `json:"elementTimelineMode,omitempty"`
	ElementKeyframeCount   int                             `json:"elementKeyframeCount,omitempty"`
	FinalElementCount      int                             `json:"finalElementCount,omitempty"`
	DeletedElementCount    int                             `json:"deletedElementCount,omitempty"`
	MissingElementPayloads int                             `json:"missingElementPayloads,omitempty"`
	StartOffsetMs          int64                           `json:"startOffsetMs,omitempty"`
	EndOffsetMs            int64                           `json:"endOffsetMs,omitempty"`
	EventFileBytes         int64                           `json:"eventFileBytes,omitempty"`
	SnapshotBytes          int64                           `json:"snapshotBytes,omitempty"`
	ElementTypeCounts      map[string]int                  `json:"elementTypeCounts,omitempty"`
	ElementPreviewFrames   []AnnotationElementKeyframePlan `json:"elementPreviewFrames,omitempty"`
}

type AnnotationElementKeyframePlan struct {
	Sequence           int    `json:"sequence,omitempty"`
	StartOffsetMs      int64  `json:"startOffsetMs,omitempty"`
	EventType          string `json:"eventType,omitempty"`
	ElementID          string `json:"elementId,omitempty"`
	ElementType        string `json:"elementType,omitempty"`
	ActiveElementCount int    `json:"activeElementCount,omitempty"`
	HasElementPayload  bool   `json:"hasElementPayload"`
}

type AnnotationElementScenePlan struct {
	InputPath           string `json:"inputPath"`
	RelativePath        string `json:"relativePath,omitempty"`
	RenderInputPath     string `json:"renderInputPath,omitempty"`
	RenderRelativePath  string `json:"renderRelativePath,omitempty"`
	StartOffsetMs       int64  `json:"startOffsetMs,omitempty"`
	EndOffsetMs         int64  `json:"endOffsetMs,omitempty"`
	DurationMs          int64  `json:"durationMs,omitempty"`
	CanvasWidth         int    `json:"canvasWidth,omitempty"`
	CanvasHeight        int    `json:"canvasHeight,omitempty"`
	ElementCount        int    `json:"elementCount,omitempty"`
	SourceEventSequence int    `json:"sourceEventSequence,omitempty"`
	Bytes               int64  `json:"bytes,omitempty"`
}
