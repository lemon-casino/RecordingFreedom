package recordingprofile

const (
	QualityStandard = "standard"
	QualityBalanced = "balanced"
	QualityHigh     = "high"

	DefaultQuality          = QualityBalanced
	DefaultFPS              = 30
	DefaultCaptureCursor    = true
	DefaultCountdownSeconds = 0
	MaxCountdownSeconds     = 10
)

type Profile struct {
	Quality          string `json:"quality"`
	FPS              int    `json:"fps"`
	CaptureCursor    bool   `json:"captureCursor"`
	CountdownSeconds int    `json:"countdownSeconds"`
}

func Default() Profile {
	return Profile{
		Quality:          DefaultQuality,
		FPS:              DefaultFPS,
		CaptureCursor:    DefaultCaptureCursor,
		CountdownSeconds: DefaultCountdownSeconds,
	}
}

func Normalize(profile Profile) Profile {
	if IsZero(profile) {
		return Default()
	}
	if !ValidQuality(profile.Quality) {
		profile.Quality = DefaultQuality
	}
	if !ValidFPS(profile.FPS) {
		profile.FPS = DefaultFPS
	}
	if profile.CountdownSeconds < 0 {
		profile.CountdownSeconds = DefaultCountdownSeconds
	}
	if profile.CountdownSeconds > MaxCountdownSeconds {
		profile.CountdownSeconds = MaxCountdownSeconds
	}
	return profile
}

func IsZero(profile Profile) bool {
	return profile.Quality == "" &&
		profile.FPS == 0 &&
		!profile.CaptureCursor &&
		profile.CountdownSeconds == 0
}

func ValidQuality(quality string) bool {
	switch quality {
	case QualityStandard, QualityBalanced, QualityHigh:
		return true
	default:
		return false
	}
}

func ValidFPS(fps int) bool {
	switch fps {
	case 24, 30, 60:
		return true
	default:
		return false
	}
}
