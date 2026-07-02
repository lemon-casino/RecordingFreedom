package exportplan

import "github.com/lemon-casino/RecordingFreedom/app/internal/pip"

const (
	DefaultOutputPath = "exports/recording.mp4"
)

type Request struct {
	VideoDir    string
	PackageDir  string
	OutputPath  string
	Canvas      pip.Size
	RequireSync bool
	AllowMock   bool
}

type Plan struct {
	PackageDir           string             `json:"packageDir"`
	ManifestPath         string             `json:"manifestPath"`
	OutputPath           string             `json:"outputPath"`
	ScreenInputPath      string             `json:"screenInputPath"`
	WebcamInputPath      string             `json:"webcamInputPath,omitempty"`
	WebcamStartOffsetMs  int                `json:"webcamStartOffsetMs,omitempty"`
	PIPPreset            string             `json:"pipPreset"`
	PIPConfig            pip.Config         `json:"pipConfig"`
	PIPRect              pip.Rect           `json:"pipRect"`
	PIPLayout            pip.Placement      `json:"pipLayout"`
	TimelineBase         string             `json:"timelineBase,omitempty"`
	AudioDiagnosticsPath string             `json:"audioDiagnosticsPath,omitempty"`
	VideoDiagnosticsPath string             `json:"videoDiagnosticsPath,omitempty"`
	PauseSegments        []PauseSegmentPlan `json:"pauseSegments,omitempty"`
	Warnings             []string           `json:"warnings,omitempty"`
}

type PauseSegmentPlan struct {
	StartOffsetMs int64 `json:"startOffsetMs"`
	EndOffsetMs   int64 `json:"endOffsetMs"`
	DurationMs    int64 `json:"durationMs"`
}
