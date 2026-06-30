package preflight

import "github.com/lemon-casino/RecordingFreedom/app/internal/recording"

type Status string

const (
	StatusReady   Status = "ready"
	StatusWarning Status = "warning"
	StatusBlocked Status = "blocked"
)

type Check struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type Summary struct {
	Status            Status                 `json:"status"`
	Backend           string                 `json:"backend"`
	Message           string                 `json:"message"`
	Checks            []Check                `json:"checks"`
	NormalizedRequest recording.StartRequest `json:"normalizedRequest"`
}
