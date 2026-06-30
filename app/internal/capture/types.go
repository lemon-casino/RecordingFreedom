package capture

type Status string

const (
	StatusAvailable   Status = "available"
	StatusQueued      Status = "queued"
	StatusBlocked     Status = "blocked"
	StatusUnsupported Status = "unsupported"
)

type Permission string

const (
	PermissionNotRequired     Permission = "not-required"
	PermissionUnknown         Permission = "unknown"
	PermissionScreenRecording Permission = "screen-recording"
	PermissionMicrophone      Permission = "microphone"
	PermissionCamera          Permission = "camera"
)

type Capability struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Status     Status     `json:"status"`
	Backend    string     `json:"backend"`
	Permission Permission `json:"permission"`
	Reason     string     `json:"reason,omitempty"`
}

type Capabilities struct {
	Platform              string     `json:"platform"`
	SourceEnumeration     Capability `json:"sourceEnumeration"`
	ScreenRecording       Capability `json:"screenRecording"`
	WindowRecording       Capability `json:"windowRecording"`
	ApplicationRecording  Capability `json:"applicationRecording"`
	SystemAudio           Capability `json:"systemAudio"`
	Microphone            Capability `json:"microphone"`
	MicrophoneEnhancement Capability `json:"microphoneEnhancement"`
	CameraSidecar         Capability `json:"cameraSidecar"`
	PIPExport             Capability `json:"pipExport"`
	PackageRecovery       Capability `json:"packageRecovery"`
}
