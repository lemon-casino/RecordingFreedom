package ocr

import "time"

const (
	StatusReady             = "ready"
	StatusNoModel           = "no-model"
	StatusModelInvalid      = "model-invalid"
	StatusWorkerAbsent      = "worker-absent"
	StatusWorkerUnavailable = "worker-unavailable"

	JobPriorityInteractive = "interactive"
	JobPriorityNormal      = "normal"
	JobPriorityBackground  = "background"

	ResultStatusNone      = "none"
	ResultStatusQueued    = "queued"
	ResultStatusRunning   = "running"
	ResultStatusReady     = "ready"
	ResultStatusFailed    = "failed"
	ResultStatusCancelled = "cancelled"
)

type SourceKind string

const (
	SourceRegionScreenshot        SourceKind = "region-screenshot"
	SourceFullScreenshot          SourceKind = "full-screenshot"
	SourceWindowScreenshot        SourceKind = "window-screenshot"
	SourceFocusedWindowScreenshot SourceKind = "focused-window-screenshot"
	SourceScrollingScreenshot     SourceKind = "scrolling-screenshot"
	SourcePinnedScreenshot        SourceKind = "pinned-screenshot"
	SourceWhiteboard              SourceKind = "whiteboard"
	SourceWhiteboardSelection     SourceKind = "whiteboard-selection"
	SourceImage                   SourceKind = "image"
)

type ModelFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256,omitempty"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type ModelSmoke struct {
	Image         string   `json:"image,omitempty"`
	Expected      string   `json:"expected,omitempty"`
	MustContain   []string `json:"mustContain,omitempty"`
	MaxDurationMS int      `json:"maxDurationMs,omitempty"`
}

type ModelSource struct {
	URL     string `json:"url,omitempty"`
	Commit  string `json:"commit,omitempty"`
	License string `json:"license,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

const (
	TextlineOrientationCLS  = "cls"
	TextlineOrientationNone = "none"
)

type ModelTextlineOrientation struct {
	Mode string `json:"mode,omitempty"`
}

type ModelPackageSource struct {
	URL    string `json:"url,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type ModelManifest struct {
	SchemaVersion       int                       `json:"schemaVersion"`
	ID                  string                    `json:"id"`
	Name                string                    `json:"name"`
	Channel             string                    `json:"channel"`
	Engine              string                    `json:"engine"`
	Language            []string                  `json:"language"`
	Version             string                    `json:"version"`
	Source              ModelSource               `json:"source,omitempty"`
	Package             ModelPackageSource        `json:"package,omitempty"`
	TextlineOrientation *ModelTextlineOrientation `json:"textlineOrientation,omitempty"`
	Files               []ModelFile               `json:"files"`
	Smoke               ModelSmoke                `json:"smoke,omitempty"`
}

type ModelInfo struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Channel           string   `json:"channel"`
	Engine            string   `json:"engine"`
	Language          []string `json:"language"`
	Version           string   `json:"version,omitempty"`
	SourceURL         string   `json:"sourceUrl,omitempty"`
	License           string   `json:"license,omitempty"`
	DownloadAvailable bool     `json:"downloadAvailable,omitempty"`
	DownloadBytes     int64    `json:"downloadBytes,omitempty"`
	Installed         bool     `json:"installed"`
	Verified          bool     `json:"verified"`
	Active            bool     `json:"active"`
	ModelDir          string   `json:"modelDir,omitempty"`
	SmokeImage        string   `json:"smokeImage,omitempty"`
	SmokeExpected     string   `json:"smokeExpected,omitempty"`
	SmokeAssetReady   bool     `json:"smokeAssetReady,omitempty"`
	SmokeError        string   `json:"smokeError,omitempty"`
	MissingFiles      []string `json:"missingFiles,omitempty"`
	VerificationError string   `json:"verificationError,omitempty"`
}

type State struct {
	SchemaVersion int       `json:"schemaVersion"`
	ActiveModelID string    `json:"activeModelId"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Status struct {
	Status             string              `json:"status"`
	ActiveModelID      string              `json:"activeModelId,omitempty"`
	Models             []ModelInfo         `json:"models"`
	WorkerPath         string              `json:"workerPath,omitempty"`
	RuntimeDir         string              `json:"runtimeDir,omitempty"`
	WorkerCapabilities *WorkerCapabilities `json:"workerCapabilities,omitempty"`
	Message            string              `json:"message,omitempty"`
}

const (
	ModelDownloadQueued    = "queued"
	ModelDownloadRunning   = "running"
	ModelDownloadInstalled = "installed"
	ModelDownloadFailed    = "failed"
	ModelDownloadCancelled = "cancelled"
)

type ModelDownloadSnapshot struct {
	ID              string     `json:"id"`
	ModelID         string     `json:"modelId"`
	Status          string     `json:"status"`
	DownloadedBytes int64      `json:"downloadedBytes"`
	TotalBytes      int64      `json:"totalBytes"`
	Percent         float64    `json:"percent"`
	Error           string     `json:"error,omitempty"`
	Model           *ModelInfo `json:"model,omitempty"`
	StartedAt       time.Time  `json:"startedAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ModelDownloadEvent struct {
	Snapshot ModelDownloadSnapshot `json:"snapshot"`
}

type WorkerCapabilities struct {
	SchemaVersion     int      `json:"schemaVersion"`
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ProtocolVersion   string   `json:"protocolVersion"`
	Engine            string   `json:"engine"`
	ModelFormats      []string `json:"modelFormats"`
	SupportsRecognize bool     `json:"supportsRecognize"`
	RuntimeDir        string   `json:"runtimeDir,omitempty"`
	RuntimeLibrary    string   `json:"runtimeLibrary,omitempty"`
	RuntimeAvailable  bool     `json:"runtimeAvailable"`
	RuntimeVersion    string   `json:"runtimeVersion,omitempty"`
	RuntimeAPIVersion int      `json:"runtimeApiVersion,omitempty"`
	RuntimeError      string   `json:"runtimeError,omitempty"`
	Message           string   `json:"message,omitempty"`
}

type RecognizeRequest struct {
	ImagePath  string     `json:"imagePath"`
	SourceKind SourceKind `json:"sourceKind"`
	SourceID   string     `json:"sourceId"`
	Language   string     `json:"language"`
	ModelID    string     `json:"modelId,omitempty"`
	Force      bool       `json:"force"`
	Priority   string     `json:"priority"`
}

type JobSnapshot struct {
	JobID     string           `json:"jobId"`
	Status    string           `json:"status"`
	CacheKey  string           `json:"cacheKey,omitempty"`
	Request   RecognizeRequest `json:"request"`
	Merged    bool             `json:"merged"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type JobEvent struct {
	JobID     string           `json:"jobId"`
	Status    string           `json:"status"`
	Request   RecognizeRequest `json:"request"`
	Result    *Result          `json:"result,omitempty"`
	Error     string           `json:"error,omitempty"`
	CacheKey  string           `json:"cacheKey,omitempty"`
	Merged    bool             `json:"merged"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Block struct {
	ID           string  `json:"id"`
	Text         string  `json:"text"`
	Confidence   float64 `json:"confidence"`
	Box          []Point `json:"box"`
	LineIndex    int     `json:"lineIndex"`
	LanguageHint string  `json:"languageHint,omitempty"`
}

type Result struct {
	ID          string     `json:"id"`
	SourceKind  SourceKind `json:"sourceKind"`
	SourceID    string     `json:"sourceId"`
	ImagePath   string     `json:"imagePath"`
	ImageSHA256 string     `json:"imageSha256"`
	ModelID     string     `json:"modelId"`
	Language    string     `json:"language"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Blocks      []Block    `json:"blocks"`
	PlainText   string     `json:"plainText"`
	CreatedAt   time.Time  `json:"createdAt"`
	DurationMS  int        `json:"durationMs"`
}

type TranslateRequest struct {
	OcrResultID    string   `json:"ocrResultId"`
	BlockIDs       []string `json:"blockIds,omitempty"`
	Provider       string   `json:"provider"`
	SourceLanguage string   `json:"sourceLanguage"`
	TargetLanguage string   `json:"targetLanguage"`
	BaseURL        string   `json:"baseUrl,omitempty"`
	APIKey         string   `json:"apiKey,omitempty"`
	Model          string   `json:"model,omitempty"`
	Force          bool     `json:"force"`
}

type TranslationBlock struct {
	BlockID    string `json:"blockId"`
	Source     string `json:"source"`
	Translated string `json:"translated"`
}

type TranslationResult struct {
	OcrResultID    string             `json:"ocrResultId"`
	Provider       string             `json:"provider"`
	SourceLanguage string             `json:"sourceLanguage"`
	TargetLanguage string             `json:"targetLanguage"`
	Model          string             `json:"model,omitempty"`
	PromptVersion  string             `json:"promptVersion,omitempty"`
	Blocks         []TranslationBlock `json:"blocks"`
	CreatedAt      time.Time          `json:"createdAt"`
}

type WhiteboardRequest struct {
	ImagePath string `json:"imagePath"`
	ElementID string `json:"elementId,omitempty"`
	SceneID   string `json:"sceneId,omitempty"`
	Language  string `json:"language,omitempty"`
	Force     bool   `json:"force"`
	Priority  string `json:"priority,omitempty"`
}
