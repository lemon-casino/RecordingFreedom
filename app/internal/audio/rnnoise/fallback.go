//go:build !rnnoise_dynamic && (!cgo || !rnnoise_native)

package rnnoise

import "errors"

type Suppressor struct{}

func Available() bool {
	return false
}

func RequiredSampleRate() int {
	return 48000
}

func FrameSize() int {
	return 480
}

func New(float64) (*Suppressor, error) {
	return nil, errors.New("rnnoise native DSP requires a build with rnnoise_dynamic and a packaged native module")
}

func (s *Suppressor) Name() string {
	return "rnnoise-unavailable"
}

func (s *Suppressor) ProcessFrame([]float32) error {
	return errors.New("rnnoise native DSP requires a build with rnnoise_dynamic and a packaged native module")
}

func (s *Suppressor) Reset() error {
	return nil
}

func (s *Suppressor) Close() {}
