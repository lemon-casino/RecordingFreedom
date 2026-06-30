package devices

type CaptureSourceType string

const (
	SourceScreen      CaptureSourceType = "screen"
	SourceWindow      CaptureSourceType = "window"
	SourceApplication CaptureSourceType = "application"
)

type SourceCapability string

const (
	CapabilityEnumerated   SourceCapability = "enumerated"
	CapabilityUnavailable  SourceCapability = "unavailable"
	CapabilityNativeQueued SourceCapability = "native-backend-queued"
)

type CaptureSource struct {
	ID                string            `json:"id"`
	Type              CaptureSourceType `json:"type"`
	Name              string            `json:"name"`
	Subtitle          string            `json:"subtitle"`
	Width             int               `json:"width,omitempty"`
	Height            int               `json:"height,omitempty"`
	NativeID          string            `json:"nativeId,omitempty"`
	DisplayIndex      int               `json:"displayIndex,omitempty"`
	ProcessID         int               `json:"processId,omitempty"`
	Available         bool              `json:"available"`
	Capability        SourceCapability  `json:"capability"`
	UnavailableReason string            `json:"unavailableReason,omitempty"`
}

type MediaDeviceType string

const (
	DeviceSystemAudio MediaDeviceType = "system-audio"
	DeviceMicrophone  MediaDeviceType = "microphone"
	DeviceCamera      MediaDeviceType = "camera"
)

type MediaDevice struct {
	ID                string           `json:"id"`
	Type              MediaDeviceType  `json:"type"`
	Name              string           `json:"name"`
	Subtitle          string           `json:"subtitle"`
	NativeID          string           `json:"nativeId,omitempty"`
	IsDefault         bool             `json:"isDefault"`
	Available         bool             `json:"available"`
	Capability        SourceCapability `json:"capability"`
	UnavailableReason string           `json:"unavailableReason,omitempty"`
	RNNoiseEligible   bool             `json:"rnnoiseEligible,omitempty"`
	SidecarEligible   bool             `json:"sidecarEligible,omitempty"`
}

type AudioEnhancement struct {
	Engine            string           `json:"engine"`
	AppliesTo         string           `json:"appliesTo"`
	Available         bool             `json:"available"`
	Capability        SourceCapability `json:"capability"`
	UnavailableReason string           `json:"unavailableReason,omitempty"`
}

type MediaInventory struct {
	SystemAudio []MediaDevice    `json:"systemAudio"`
	Microphones []MediaDevice    `json:"microphones"`
	Cameras     []MediaDevice    `json:"cameras"`
	Enhancement AudioEnhancement `json:"enhancement"`
}
