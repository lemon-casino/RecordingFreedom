package audio

import "errors"

func NewNativeCaptureSession(config CaptureConfig, suppressor NoiseSuppressor) (*CaptureSession, error) {
	config = normalizeCaptureConfig(config)
	sources, err := NewPlatformCaptureSources(config)
	if err != nil {
		return nil, err
	}
	sinks, err := NewWAVSinks(config)
	if err != nil {
		return nil, err
	}
	var enhancer *Enhancer
	if config.NoiseSuppression && config.Microphone.Enabled {
		enhancer = NewEnhancer(suppressor)
	}
	return NewCaptureSession(config, enhancer, sources, sinks)
}

func NewWAVSinks(config CaptureConfig) (map[StreamKind]CaptureSink, error) {
	config = normalizeCaptureConfig(config)
	sinks := map[StreamKind]CaptureSink{}
	if config.SystemAudio.Enabled {
		sink, err := NewWAVSink(string(StreamSystemAudio), config.SystemAudioOutputPath)
		if err != nil {
			return nil, err
		}
		sinks[StreamSystemAudio] = sink
	}
	if config.Microphone.Enabled {
		sink, err := NewWAVSink(string(StreamMicrophone), config.MicrophoneAudioPath)
		if err != nil {
			_ = closeSinks(sinks)
			return nil, err
		}
		sinks[StreamMicrophone] = sink
	}
	if len(sinks) == 0 {
		return nil, errors.New("no audio streams are enabled")
	}
	return sinks, nil
}
