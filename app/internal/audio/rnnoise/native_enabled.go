//go:build cgo && rnnoise_native

package rnnoise

import nativeimpl "github.com/lemon-casino/RecordingFreedom/app/internal/audio/rnnoise/native"

type Suppressor = nativeimpl.Suppressor

func Available() bool {
	return nativeimpl.Available()
}

func RequiredSampleRate() int {
	return nativeimpl.RequiredSampleRate()
}

func FrameSize() int {
	return nativeimpl.FrameSize()
}

func New(outputGain float64) (*Suppressor, error) {
	return nativeimpl.New(outputGain)
}
