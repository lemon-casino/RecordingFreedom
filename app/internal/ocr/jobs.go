package ocr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type jobState struct {
	id        string
	status    string
	cacheKey  string
	requests  []RecognizeRequest
	priority  int
	createdAt time.Time
	updatedAt time.Time
	cancelled bool
}

func (s *Service) Events() <-chan JobEvent {
	if s == nil {
		return nil
	}
	return s.jobEvents
}

func (s *Service) EnqueueRecognize(req RecognizeRequest) (JobSnapshot, error) {
	req, cacheKey, err := s.normalizeJobRequest(req)
	if err != nil {
		return JobSnapshot{}, err
	}
	now := s.now().UTC()

	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	s.ensureJobWorkerLocked()
	if existing := s.jobsByKey[cacheKey]; existing != nil && !existing.cancelled {
		merged := appendJobRequest(existing, req)
		if req.Force {
			existing.requests[0].Force = true
		}
		if rank := jobPriorityRank(req.Priority); rank < existing.priority {
			existing.priority = rank
		}
		existing.updatedAt = now
		s.emitJobEventLocked(jobEventForRequest(existing, req, ResultStatusQueued, "", nil, merged))
		return jobSnapshotForRequest(existing, req, merged), nil
	}

	job := &jobState{
		id:        "ocr-job-" + requestIDNow(),
		status:    ResultStatusQueued,
		cacheKey:  cacheKey,
		requests:  []RecognizeRequest{req},
		priority:  jobPriorityRank(req.Priority),
		createdAt: now,
		updatedAt: now,
	}
	s.jobQueue = append(s.jobQueue, job)
	s.jobsByID[job.id] = job
	s.jobsByKey[job.cacheKey] = job
	s.signalJobWorkerLocked()
	s.emitJobEventLocked(jobEventForRequest(job, req, ResultStatusQueued, "", nil, false))
	return jobSnapshotForRequest(job, req, false), nil
}

func (s *Service) CancelJob(jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("OCR job id is required")
	}
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	job := s.jobsByID[jobID]
	if job == nil {
		return fmt.Errorf("OCR job %q was not found", jobID)
	}
	job.cancelled = true
	job.status = ResultStatusCancelled
	job.updatedAt = s.now().UTC()
	s.removeQueuedJobLocked(job)
	if s.activeJob == job {
		s.activeJob = nil
	}
	delete(s.jobsByID, job.id)
	delete(s.jobsByKey, job.cacheKey)
	for _, req := range append([]RecognizeRequest(nil), job.requests...) {
		s.emitJobEventLocked(jobEventForRequest(job, req, ResultStatusCancelled, "", nil, false))
	}
	return nil
}

func (s *Service) normalizeJobRequest(req RecognizeRequest) (RecognizeRequest, string, error) {
	req = normalizeRecognizeRequest(req)
	if strings.TrimSpace(req.ImagePath) == "" {
		return RecognizeRequest{}, "", errors.New("OCR image path is required")
	}
	imagePath, err := filepath.Abs(req.ImagePath)
	if err != nil {
		return RecognizeRequest{}, "", err
	}
	info, err := os.Stat(imagePath)
	if err != nil {
		return RecognizeRequest{}, "", err
	}
	if info.IsDir() {
		return RecognizeRequest{}, "", fmt.Errorf("OCR image path %q is a directory", imagePath)
	}
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		state, err := s.LoadState()
		if err != nil {
			return RecognizeRequest{}, "", err
		}
		modelID = state.ActiveModelID
	}
	if strings.TrimSpace(modelID) == "" {
		return RecognizeRequest{}, "", errors.New("OCR model id is required")
	}
	sum, err := fileSHA256(imagePath)
	if err != nil {
		return RecognizeRequest{}, "", err
	}
	req.ImagePath = imagePath
	req.ModelID = modelID
	cacheKey := safeCachePart(sum) + "." + safeCachePart(modelID) + "." + safeCachePart(req.Language)
	return req, cacheKey, nil
}

func (s *Service) ensureJobWorkerLocked() {
	if s.jobStarted {
		return
	}
	if s.jobNotify == nil {
		s.jobNotify = make(chan struct{}, 1)
	}
	if s.jobEvents == nil {
		s.jobEvents = make(chan JobEvent, 128)
	}
	if s.jobsByID == nil {
		s.jobsByID = map[string]*jobState{}
	}
	if s.jobsByKey == nil {
		s.jobsByKey = map[string]*jobState{}
	}
	s.jobStarted = true
	go s.runJobWorker()
}

func (s *Service) runJobWorker() {
	for {
		job := s.nextJob()
		if job == nil {
			notify := s.jobNotify
			if notify == nil {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			<-notify
			continue
		}
		s.runJob(job)
	}
}

func (s *Service) nextJob() *jobState {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if len(s.jobQueue) == 0 {
		return nil
	}
	bestIndex := 0
	for index := 1; index < len(s.jobQueue); index++ {
		if compareJobPriority(s.jobQueue[index], s.jobQueue[bestIndex]) < 0 {
			bestIndex = index
		}
	}
	job := s.jobQueue[bestIndex]
	s.jobQueue = append(s.jobQueue[:bestIndex], s.jobQueue[bestIndex+1:]...)
	if job.cancelled {
		return nil
	}
	now := s.now().UTC()
	job.status = ResultStatusRunning
	job.updatedAt = now
	s.activeJob = job
	for _, req := range append([]RecognizeRequest(nil), job.requests...) {
		s.emitJobEventLocked(jobEventForRequest(job, req, ResultStatusRunning, "", nil, false))
	}
	return job
}

func (s *Service) runJob(job *jobState) {
	if job == nil {
		return
	}
	requests := append([]RecognizeRequest(nil), job.requests...)
	if len(requests) == 0 {
		s.finishJob(job, nil, errors.New("OCR job has no requests"))
		return
	}
	runReq := requests[0]
	for _, req := range requests {
		if req.Force {
			runReq.Force = true
			break
		}
	}
	result, err := s.RecognizeImage(runReq)
	s.finishJob(job, &result, err)
}

func (s *Service) finishJob(job *jobState, result *Result, err error) {
	s.jobMu.Lock()
	cancelled := job.cancelled
	if s.activeJob == job {
		s.activeJob = nil
	}
	delete(s.jobsByID, job.id)
	delete(s.jobsByKey, job.cacheKey)
	job.updatedAt = s.now().UTC()
	if cancelled {
		s.jobMu.Unlock()
		return
	}
	if err != nil {
		job.status = ResultStatusFailed
		requests := append([]RecognizeRequest(nil), job.requests...)
		s.jobMu.Unlock()
		for _, req := range requests {
			s.emitJobEvent(jobEventForRequest(job, req, ResultStatusFailed, err.Error(), nil, false))
		}
		return
	}
	job.status = ResultStatusReady
	requests := append([]RecognizeRequest(nil), job.requests...)
	s.jobMu.Unlock()
	for _, req := range requests {
		target := cloneResultForRequest(result, req)
		if writeErr := s.WriteResult(target); writeErr != nil {
			s.emitJobEvent(jobEventForRequest(job, req, ResultStatusFailed, writeErr.Error(), nil, false))
			continue
		}
		s.emitJobEvent(jobEventForRequest(job, req, ResultStatusReady, "", &target, false))
	}
}

func appendJobRequest(job *jobState, req RecognizeRequest) bool {
	for _, existing := range job.requests {
		if sameJobRequestTarget(existing, req) {
			return false
		}
	}
	job.requests = append(job.requests, req)
	return true
}

func sameJobRequestTarget(left RecognizeRequest, right RecognizeRequest) bool {
	return left.SourceKind == right.SourceKind &&
		strings.TrimSpace(left.SourceID) == strings.TrimSpace(right.SourceID) &&
		strings.TrimSpace(left.ImagePath) == strings.TrimSpace(right.ImagePath) &&
		strings.TrimSpace(left.ModelID) == strings.TrimSpace(right.ModelID) &&
		strings.TrimSpace(left.Language) == strings.TrimSpace(right.Language)
}

func (s *Service) removeQueuedJobLocked(job *jobState) {
	for index, candidate := range s.jobQueue {
		if candidate == job {
			s.jobQueue = append(s.jobQueue[:index], s.jobQueue[index+1:]...)
			return
		}
	}
}

func (s *Service) signalJobWorkerLocked() {
	if s.jobNotify == nil {
		return
	}
	select {
	case s.jobNotify <- struct{}{}:
	default:
	}
}

func (s *Service) emitJobEventLocked(event JobEvent) {
	s.emitJobEvent(event)
}

func (s *Service) emitJobEvent(event JobEvent) {
	if s == nil || s.jobEvents == nil {
		return
	}
	select {
	case s.jobEvents <- event:
	default:
		go func() {
			s.jobEvents <- event
		}()
	}
}

func jobEventForRequest(job *jobState, req RecognizeRequest, status string, message string, result *Result, merged bool) JobEvent {
	if job == nil {
		return JobEvent{Status: status, Request: req, Error: message, Result: result, Merged: merged}
	}
	return JobEvent{
		JobID:     job.id,
		Status:    status,
		Request:   req,
		Result:    result,
		Error:     message,
		CacheKey:  job.cacheKey,
		Merged:    merged,
		CreatedAt: job.createdAt,
		UpdatedAt: job.updatedAt,
	}
}

func jobSnapshotForRequest(job *jobState, req RecognizeRequest, merged bool) JobSnapshot {
	return JobSnapshot{
		JobID:     job.id,
		Status:    job.status,
		CacheKey:  job.cacheKey,
		Request:   req,
		Merged:    merged,
		CreatedAt: job.createdAt,
		UpdatedAt: job.updatedAt,
	}
}

func cloneResultForRequest(result *Result, req RecognizeRequest) Result {
	if result == nil {
		return Result{}
	}
	target := *result
	target.SourceKind = req.SourceKind
	target.SourceID = req.SourceID
	target.ImagePath = req.ImagePath
	target.Language = req.Language
	if strings.TrimSpace(req.ModelID) != "" {
		target.ModelID = req.ModelID
	}
	if target.ID == "" || target.SourceKind != result.SourceKind || strings.TrimSpace(target.SourceID) != strings.TrimSpace(result.SourceID) {
		target.ID = "ocr_" + requestIDNow()
	}
	if target.CreatedAt.IsZero() {
		target.CreatedAt = time.Now().UTC()
	}
	return target
}

func compareJobPriority(left *jobState, right *jobState) int {
	if left.priority != right.priority {
		return left.priority - right.priority
	}
	if left.createdAt.Before(right.createdAt) {
		return -1
	}
	if left.createdAt.After(right.createdAt) {
		return 1
	}
	return 0
}

func jobPriorityRank(priority string) int {
	switch strings.TrimSpace(priority) {
	case JobPriorityInteractive:
		return 0
	case JobPriorityBackground:
		return 2
	default:
		return 1
	}
}
