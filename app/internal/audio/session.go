package audio

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type CaptureSession struct {
	pipeline *Pipeline
	sources  []CaptureSource
	sinks    map[StreamKind]CaptureSink
	mu       sync.Mutex
	started  bool
	stopped  bool
}

func NewCaptureSession(config CaptureConfig, enhancer *Enhancer, sources []CaptureSource, sinks map[StreamKind]CaptureSink) (*CaptureSession, error) {
	pipeline, err := NewPipeline(config, enhancer)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errors.New("audio capture session requires at least one source")
	}
	if len(sinks) == 0 {
		return nil, errors.New("audio capture session requires at least one sink")
	}
	for _, source := range sources {
		if source == nil {
			return nil, errors.New("audio capture session source cannot be nil")
		}
		if sinks[source.Kind()] == nil {
			return nil, fmt.Errorf("audio capture session missing sink for %q", source.Kind())
		}
	}
	return &CaptureSession{pipeline: pipeline, sources: sources, sinks: sinks}, nil
}

func (s *CaptureSession) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("audio capture session is already started")
	}
	if s.stopped {
		s.mu.Unlock()
		return errors.New("audio capture session is stopped")
	}
	s.started = true
	s.mu.Unlock()

	var started []CaptureSource
	for _, source := range s.sources {
		if err := ctx.Err(); err != nil {
			_ = stopSources(started)
			return err
		}
		if err := source.Start(s.handleFrame); err != nil {
			_ = stopSources(started)
			return fmt.Errorf("start audio source %q: %w", source.ID(), err)
		}
		started = append(started, source)
	}
	return nil
}

func (s *CaptureSession) Pause() error {
	return eachSource(s.sources, func(source CaptureSource) error {
		return source.Pause()
	})
}

func (s *CaptureSession) Resume() error {
	return eachSource(s.sources, func(source CaptureSource) error {
		return source.Resume()
	})
}

func (s *CaptureSession) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	started := s.started
	s.stopped = true
	s.mu.Unlock()

	var sourceErr error
	if started {
		sourceErr = stopSources(s.sources)
	}
	sinkErr := closeSinks(s.sinks)
	diagnosticsErr := s.pipeline.WriteDiagnostics()
	return errors.Join(sourceErr, sinkErr, diagnosticsErr)
}

func (s *CaptureSession) Diagnostics() Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pipeline.Diagnostics()
}

func (s *CaptureSession) handleFrame(input TimedPCMBuffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	output, err := s.pipeline.Process(input)
	if err != nil {
		return err
	}
	sink := s.sinks[output.Buffer.Kind]
	if sink == nil {
		return fmt.Errorf("missing sink for processed audio stream %q", output.Buffer.Kind)
	}
	if err := sink.Append(output); err != nil {
		return err
	}
	return nil
}

func eachSource(sources []CaptureSource, fn func(CaptureSource) error) error {
	var err error
	for _, source := range sources {
		if source == nil {
			continue
		}
		err = errors.Join(err, fn(source))
	}
	return err
}

func stopSources(sources []CaptureSource) error {
	return eachSource(sources, func(source CaptureSource) error {
		return source.Stop()
	})
}

func closeSinks(sinks map[StreamKind]CaptureSink) error {
	var err error
	seen := map[string]bool{}
	for _, sink := range sinks {
		if sink == nil || seen[sink.ID()] {
			continue
		}
		seen[sink.ID()] = true
		err = errors.Join(err, sink.Close())
	}
	return err
}
