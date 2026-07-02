package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/pip"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
)

const fileName = "settings.json"

type Service struct {
	appData *appdata.Service
	now     func() time.Time
}

func NewService(appData *appdata.Service) *Service {
	return &Service{
		appData: appData,
		now:     time.Now,
	}
}

func (s *Service) Path() (string, error) {
	root, err := s.appData.RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, fileName), nil
}

func (s *Service) Load() (Settings, error) {
	path, err := s.Path()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Settings{}, err
	}

	var result Settings
	if err := json.Unmarshal(data, &result); err != nil {
		return Settings{}, err
	}
	return normalize(result), nil
}

func (s *Service) Save(next Settings) (Settings, error) {
	path, err := s.Path()
	if err != nil {
		return Settings{}, err
	}

	next = normalize(next)
	next.UpdatedAt = s.now().UTC()
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return Settings{}, err
	}
	if err := replaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return Settings{}, err
	}
	return next, nil
}

func Default() Settings {
	return Settings{
		SchemaVersion: SchemaVersion,
		Locale:        LocaleZhCN,
		Source: SourceSettings{
			LastSourceType: "screen",
		},
		Recording: recordingprofile.Default(),
		Audio: AudioSettings{
			System:             false,
			SystemDeviceID:     "system-audio:default",
			Microphone:         false,
			MicrophoneDeviceID: "microphone:default",
			NoiseSuppression:   false,
			MicrophoneGain:     1,
		},
		Camera: CameraSettings{
			Enabled:   false,
			DeviceID:  "camera:default",
			PIPPreset: string(pip.DefaultPreset),
			PIP:       pip.DefaultConfig(),
		},
		Window: WindowSettings{
			MinimizeToTray: true,
		},
	}
}

func normalize(value Settings) Settings {
	defaults := Default()
	if value.SchemaVersion == 0 {
		value.SchemaVersion = SchemaVersion
	}
	if value.Locale == "" || !validLocale(value.Locale) {
		value.Locale = defaults.Locale
	}
	if value.Source.LastSourceType == "" {
		value.Source.LastSourceType = defaults.Source.LastSourceType
	}
	value.Storage.DataRootDir = strings.TrimSpace(value.Storage.DataRootDir)
	value.Recording = recordingprofile.Normalize(value.Recording)
	if value.Audio.SystemDeviceID == "" {
		value.Audio.SystemDeviceID = defaults.Audio.SystemDeviceID
	}
	if value.Audio.MicrophoneDeviceID == "" {
		value.Audio.MicrophoneDeviceID = defaults.Audio.MicrophoneDeviceID
	}
	if value.Audio.MicrophoneGain <= 0 {
		value.Audio.MicrophoneGain = defaults.Audio.MicrophoneGain
	}
	if value.Camera.DeviceID == "" {
		value.Camera.DeviceID = defaults.Camera.DeviceID
	}
	value.Camera.PIP = pip.NormalizeConfigForPreset(value.Camera.PIPPreset, value.Camera.PIP)
	value.Camera.PIPPreset = string(value.Camera.PIP.Preset)
	return value
}

func validLocale(locale Locale) bool {
	switch locale {
	case LocaleZhCN, LocaleEN:
		return true
	default:
		return false
	}
}

func replaceFile(tmp string, target string) error {
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tmp, target)
}
