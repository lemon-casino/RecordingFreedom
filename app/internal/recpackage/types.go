package recpackage

import (
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const (
	AppName                = "RecordingFreedom"
	ManifestFile           = "manifest.json"
	AudioDiagnosticsFile   = "audio-diagnostics.json"
	VideoDiagnosticsFile   = "video-diagnostics.json"
	ScreenVideoFile        = "screen.mp4"
	AudioOnlyFile          = "audio.m4a"
	AudioOnlyWAVFile       = "audio.wav"
	SystemAudioFile        = "system-audio.wav"
	MicrophoneAudioFile    = "microphone.wav"
	WebcamVideoFile        = "webcam.mov"
	WindowsWebcamVideoFile = "webcam.mp4"
	MockScreenFile         = "screen.mock.txt"
	CacheDir               = "cache"
	ExportsDir             = "exports"
	PackageDirSuffix       = ".rfrec"
	StatusRecording        = "recording"
	StatusPaused           = "paused"
	StatusFinalizing       = "finalizing"
	StatusReady            = "ready"
	StatusRecoverable      = "recoverable"
	StatusFailed           = "failed"
	NoiseSuppressionOn     = "rnnoise"
	NoiseSuppressionOff    = "off"
	TimelineBaseMock       = "mock"
	TimelineBaseMedia      = "media-timestamp"
	TimelineBasePlatform   = "platform-start-timestamp"
	AudioStorageSidecar    = "sidecar"
	AudioStorageMuxed      = "muxed"
	RecordingModeScreen    = "screen"
	RecordingModeAudio     = "audio-only"
)

type Package struct {
	ID           string
	Dir          string
	ManifestPath string
	Manifest     Manifest
}

type RecordingWritePlan struct {
	Package              Package
	ScreenVideoPath      string
	SystemAudioPath      string
	MicrophoneAudioPath  string
	WebcamVideoPath      string
	AudioOnlyPath        string
	AudioDiagnosticsPath string
	VideoDiagnosticsPath string
	CacheDir             string
	ExportsDir           string
}

type Manifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	App           string                   `json:"app"`
	CreatedAt     time.Time                `json:"createdAt"`
	CompletedAt   *time.Time               `json:"completedAt,omitempty"`
	Status        string                   `json:"status"`
	RecordingMode string                   `json:"recordingMode"`
	Media         ManifestMedia            `json:"media"`
	Source        ManifestSource           `json:"source"`
	Recording     recordingprofile.Profile `json:"recording"`
	Audio         ManifestAudio            `json:"audio"`
	Camera        ManifestCamera           `json:"camera"`
	Diagnostics   ManifestDiagnostics      `json:"diagnostics"`
}

type ManifestMedia struct {
	ScreenVideoPath        string `json:"screenVideoPath,omitempty"`
	AudioPath              string `json:"audioPath,omitempty"`
	SystemAudioPath        string `json:"systemAudioPath,omitempty"`
	SystemAudioStorage     string `json:"systemAudioStorage,omitempty"`
	MicrophoneAudioPath    string `json:"microphoneAudioPath,omitempty"`
	MicrophoneAudioStorage string `json:"microphoneAudioStorage,omitempty"`
	WebcamVideoPath        string `json:"webcamVideoPath,omitempty"`
	WebcamStartOffsetMs    int    `json:"webcamStartOffsetMs,omitempty"`
}

type ManifestSource struct {
	Type     string                  `json:"type"`
	ID       string                  `json:"id"`
	Name     string                  `json:"name,omitempty"`
	Geometry *ManifestSourceGeometry `json:"geometry,omitempty"`
}

type ManifestSourceGeometry struct {
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DisplayIndex int    `json:"displayIndex,omitempty"`
	NativeID     string `json:"nativeId,omitempty"`
}

type ManifestAudio struct {
	System                     bool    `json:"system"`
	SystemDeviceID             string  `json:"systemDeviceId,omitempty"`
	Microphone                 bool    `json:"microphone"`
	MicrophoneDeviceID         string  `json:"microphoneDeviceId,omitempty"`
	MicrophoneNoiseSuppression string  `json:"microphoneNoiseSuppression"`
	MicrophoneGain             float64 `json:"microphoneGain,omitempty"`
	SampleRate                 int     `json:"sampleRate"`
	MockPipeline               bool    `json:"mockPipeline,omitempty"`
	SystemAudioIsNeverDenoised bool    `json:"systemAudioIsNeverDenoised"`
}

type ManifestCamera struct {
	Enabled   bool   `json:"enabled"`
	DeviceID  string `json:"deviceId,omitempty"`
	PIPPreset string `json:"pipPreset"`
}

type ManifestDiagnostics struct {
	Mock      bool                     `json:"mock,omitempty"`
	Recovered bool                     `json:"recovered,omitempty"`
	Message   string                   `json:"message,omitempty"`
	Sync      *ManifestSyncDiagnostics `json:"sync,omitempty"`
}

type ManifestSyncDiagnostics struct {
	TimelineBase          string                   `json:"timelineBase"`
	TimelineStartUnixNano int64                    `json:"timelineStartUnixNano,omitempty"`
	AudioDiagnosticsPath  string                   `json:"audioDiagnosticsPath,omitempty"`
	VideoDiagnosticsPath  string                   `json:"videoDiagnosticsPath,omitempty"`
	Screen                ManifestTrackDiagnostics `json:"screen"`
	SystemAudio           ManifestTrackDiagnostics `json:"systemAudio"`
	Microphone            ManifestTrackDiagnostics `json:"microphone"`
	Webcam                ManifestTrackDiagnostics `json:"webcam"`
	PauseSegments         []ManifestPauseSegment   `json:"pauseSegments,omitempty"`
}

type ManifestTrackDiagnostics struct {
	Enabled        bool   `json:"enabled"`
	Path           string `json:"path,omitempty"`
	Clock          string `json:"clock,omitempty"`
	StartOffsetMs  int64  `json:"startOffsetMs"`
	EndOffsetMs    int64  `json:"endOffsetMs"`
	DurationMs     int64  `json:"durationMs"`
	DroppedFrames  int64  `json:"droppedFrames,omitempty"`
	DroppedSamples int64  `json:"droppedSamples,omitempty"`
	AppendFailures int64  `json:"appendFailures,omitempty"`
	SampleRate     int    `json:"sampleRate,omitempty"`
	FrameRate      int    `json:"frameRate,omitempty"`
	Message        string `json:"message,omitempty"`
}

type ManifestPauseSegment struct {
	StartOffsetMs int64 `json:"startOffsetMs"`
	EndOffsetMs   int64 `json:"endOffsetMs"`
	DurationMs    int64 `json:"durationMs"`
}

type CreateMockRequest struct {
	CreatedAt time.Time
	Status    string
	Source    ManifestSource
	Recording recordingprofile.Profile
	Audio     ManifestAudio
	Camera    ManifestCamera
}

type CreateNativeRequest struct {
	CreatedAt       time.Time
	Status          string
	Backend         string
	Source          ManifestSource
	Recording       recordingprofile.Profile
	Audio           ManifestAudio
	Camera          ManifestCamera
	WebcamVideoPath string
}

type CreateAudioOnlyRequest struct {
	CreatedAt              time.Time
	Status                 string
	Backend                string
	Source                 ManifestSource
	Recording              recordingprofile.Profile
	Audio                  ManifestAudio
	AudioPath              string
	SystemAudioPath        string
	SystemAudioStorage     string
	MicrophoneAudioPath    string
	MicrophoneAudioStorage string
}

type RecoverySummary struct {
	PackageDir   string `json:"packageDir"`
	ManifestPath string `json:"manifestPath,omitempty"`
	Status       string `json:"status"`
	Recoverable  bool   `json:"recoverable"`
	Reason       string `json:"reason,omitempty"`
}
