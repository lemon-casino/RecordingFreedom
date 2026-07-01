package audio

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type CaptureSession struct {
	pipeline      *Pipeline
	sources       []CaptureSource
	sinks         map[StreamKind]CaptureSink
	queueCapacity int
	queue         chan audioQueueItem
	workerDone    chan struct{}
	workerErr     error
	mu            sync.Mutex
	pipelineMu    sync.Mutex
	started       bool
	accepting     bool
	stopped       bool
}

type audioQueueItem struct {
	frame TimedPCMBuffer
	flush chan error
}

func NewCaptureSession(config CaptureConfig, enhancer *Enhancer, sources []CaptureSource, sinks map[StreamKind]CaptureSink) (*CaptureSession, error) {
	config = normalizeCaptureConfig(config)
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
	return &CaptureSession{pipeline: pipeline, sources: sources, sinks: sinks, queueCapacity: config.MaxQueuedFrames}, nil
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
	queue := make(chan audioQueueItem, s.queueCapacity)
	workerDone := make(chan struct{})
	s.started = true
	s.accepting = true
	s.queue = queue
	s.workerDone = workerDone
	s.mu.Unlock()
	go s.runWorker(queue, workerDone)

	var started []CaptureSource
	for _, source := range s.sources {
		if err := ctx.Err(); err != nil {
			_ = stopSources(started)
			_ = s.stopWorker()
			return err
		}
		if err := source.Start(s.handleFrame); err != nil {
			_ = stopSources(started)
			_ = s.stopWorker()
			return fmt.Errorf("start audio source %q: %w", source.ID(), err)
		}
		started = append(started, source)
	}
	return nil
}

func (s *CaptureSession) Pause() error {
	sourceErr := eachSource(s.sources, func(source CaptureSource) error {
		return source.Pause()
	})
	workerErr := s.waitForQueueDrain()
	s.pipelineMu.Lock()
	resetErr := s.pipeline.Reset()
	s.pipelineMu.Unlock()
	return errors.Join(sourceErr, workerErr, resetErr)
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
	s.accepting = false
	s.mu.Unlock()

	var sourceErr error
	if started {
		sourceErr = stopSources(s.sources)
	}
	workerErr := s.stopWorker()
	sinkErr := closeSinks(s.sinks)
	s.pipelineMu.Lock()
	diagnosticsErr := s.pipeline.WriteDiagnostics()
	s.pipelineMu.Unlock()

	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	return errors.Join(sourceErr, workerErr, sinkErr, diagnosticsErr)
}

func (s *CaptureSession) Diagnostics() Diagnostics {
	s.pipelineMu.Lock()
	defer s.pipelineMu.Unlock()
	return s.pipeline.Diagnostics()
}

func (s *CaptureSession) handleFrame(input TimedPCMBuffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting {
		return nil
	}
	if s.workerErr != nil {
		return s.workerErr
	}
	if s.queue == nil {
		return errors.New("audio capture worker is not running")
	}
	select {
	case s.queue <- audioQueueItem{frame: input}:
		depth := len(s.queue)
		capacity := cap(s.queue)
		s.pipelineMu.Lock()
		s.pipeline.RecordQueueDepth(depth, capacity)
		s.pipelineMu.Unlock()
		return nil
	default:
		s.pipelineMu.Lock()
		s.pipeline.RecordDroppedInput(input.Buffer.Kind, len(input.Buffer.Samples), "audio processing queue is full; dropped input frame to keep memory bounded")
		s.pipelineMu.Unlock()
		return nil
	}
}

func (s *CaptureSession) runWorker(queue <-chan audioQueueItem, workerDone chan<- struct{}) {
	defer close(workerDone)
	for item := range queue {
		if item.flush != nil {
			s.pipelineMu.Lock()
			s.pipeline.RecordQueueFlush()
			s.pipelineMu.Unlock()
			s.mu.Lock()
			err := s.workerErr
			s.mu.Unlock()
			item.flush <- err
			continue
		}

		input := item.frame
		s.mu.Lock()
		workerErr := s.workerErr
		s.mu.Unlock()

		if workerErr != nil {
			s.pipelineMu.Lock()
			s.pipeline.RecordDroppedInput(input.Buffer.Kind, len(input.Buffer.Samples), "audio processing worker already failed; dropped queued frame")
			s.pipelineMu.Unlock()
			continue
		}
		if err := s.processFrame(input); err != nil {
			s.mu.Lock()
			if s.workerErr == nil {
				s.workerErr = err
			}
			s.mu.Unlock()
		}
	}
}

func (s *CaptureSession) processFrame(input TimedPCMBuffer) error {
	s.pipelineMu.Lock()
	output, err := s.pipeline.Process(input)
	s.pipelineMu.Unlock()
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

func (s *CaptureSession) waitForQueueDrain() error {
	flush := make(chan error, 1)
	s.mu.Lock()
	queue := s.queue
	workerDone := s.workerDone
	workerErr := s.workerErr
	if queue == nil || workerErr != nil {
		s.mu.Unlock()
		return workerErr
	}
	s.mu.Unlock()

	queue <- audioQueueItem{flush: flush}
	if workerDone == nil {
		return <-flush
	}
	select {
	case err := <-flush:
		return err
	case <-workerDone:
		s.mu.Lock()
		err := s.workerErr
		s.mu.Unlock()
		return err
	}
}

func (s *CaptureSession) stopWorker() error {
	s.mu.Lock()
	queue := s.queue
	workerDone := s.workerDone
	if queue != nil {
		close(queue)
		s.queue = nil
		s.workerDone = nil
	}
	s.accepting = false
	s.mu.Unlock()

	if workerDone != nil {
		<-workerDone
	}
	s.mu.Lock()
	err := s.workerErr
	s.mu.Unlock()
	return err
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
