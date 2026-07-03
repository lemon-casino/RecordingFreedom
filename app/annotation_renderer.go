package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	annotationRendererWindowWidth  = 960
	annotationRendererWindowHeight = 720
	annotationRenderTimeout        = 5 * time.Minute
	annotationRenderModeScenes     = "element-scenes"
	annotationRenderModePNGs       = "element-pngs"
)

type AnnotationRenderJob struct {
	ID                 string `json:"id"`
	PackageDir         string `json:"packageDir"`
	ScenePath          string `json:"scenePath"`
	RelativeScenePath  string `json:"relativeScenePath"`
	OutputPath         string `json:"outputPath"`
	RelativeOutputPath string `json:"relativeOutputPath"`
	SceneJSON          string `json:"sceneJson"`
	CanvasWidth        int    `json:"canvasWidth"`
	CanvasHeight       int    `json:"canvasHeight"`
	Index              int    `json:"index"`
	StartOffsetMs      int64  `json:"startOffsetMs,omitempty"`
	EndOffsetMs        int64  `json:"endOffsetMs,omitempty"`
}

type AnnotationRenderJobClaim struct {
	Available bool                 `json:"available"`
	Job       *AnnotationRenderJob `json:"job,omitempty"`
}

type AnnotationRenderJobResult struct {
	ID      string `json:"id"`
	DataURL string `json:"dataUrl,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AnnotationRenderBatchResult struct {
	Rendered int      `json:"rendered"`
	Failed   int      `json:"failed"`
	Warnings []string `json:"warnings,omitempty"`
}

type annotationRenderJobState struct {
	job     AnnotationRenderJob
	claimed bool
	done    bool
	err     string
}

type annotationRenderBatch struct {
	id         string
	packageDir string
	jobs       []*annotationRenderJobState
	done       chan AnnotationRenderBatchResult
	completed  bool
}

func (s *RecordingFreedomService) setAnnotationRendererWindow(window *application.WebviewWindow) {
	s.annotationRenderer = window
}

func (s *RecordingFreedomService) ensureAnnotationRenderedAssets(req ExportRecordingRequest, plan exportplan.Plan) (exportplan.Plan, error) {
	if plan.AnnotationRenderMode != annotationRenderModeScenes || len(plan.AnnotationElementScenes) == 0 {
		return plan, nil
	}
	result, err := s.renderAnnotationElementScenes(plan)
	if err != nil {
		return exportplan.Plan{}, err
	}
	if result.Failed > 0 {
		return exportplan.Plan{}, fmt.Errorf("annotation rendering failed for %d scene(s): %s", result.Failed, strings.Join(result.Warnings, "; "))
	}
	replanned, err := s.exportRecordingPlan(req, true)
	if err != nil {
		return exportplan.Plan{}, err
	}
	if replanned.AnnotationRenderMode != annotationRenderModePNGs || len(replanned.AnnotationSnapshots) == 0 {
		return exportplan.Plan{}, errors.New("annotation renderer completed but rendered PNG timeline is not available")
	}
	return replanned, nil
}

func (s *RecordingFreedomService) renderAnnotationElementScenes(plan exportplan.Plan) (AnnotationRenderBatchResult, error) {
	if s.annotationRenderer == nil {
		return AnnotationRenderBatchResult{}, errors.New("annotation renderer window is not configured")
	}
	resultCh, err := s.startAnnotationRenderBatch(plan.PackageDir, plan.AnnotationElementScenes)
	if err != nil {
		return AnnotationRenderBatchResult{}, err
	}
	s.annotationRenderer.SetAlwaysOnTop(false)
	s.annotationRenderer.SetBounds(application.Rect{
		X:      -32000,
		Y:      -32000,
		Width:  annotationRendererWindowWidth,
		Height: annotationRendererWindowHeight,
	})
	s.annotationRenderer.Show()
	s.annotationRenderer.ExecJS("window.dispatchEvent(new CustomEvent('rf-annotation-renderer-wake'));")
	s.logEvent("annotation-renderer", "batch-start", map[string]string{
		"packageDir": plan.PackageDir,
		"scenes":     fmt.Sprint(len(plan.AnnotationElementScenes)),
	})
	defer s.annotationRenderer.Hide()
	select {
	case result := <-resultCh:
		s.logEvent("annotation-renderer", "batch-complete", map[string]string{
			"rendered": fmt.Sprint(result.Rendered),
			"failed":   fmt.Sprint(result.Failed),
		})
		return result, nil
	case <-time.After(annotationRenderTimeout):
		s.cancelAnnotationRenderBatch("annotation rendering timed out")
		return AnnotationRenderBatchResult{}, errors.New("annotation rendering timed out")
	}
}

func (s *RecordingFreedomService) startAnnotationRenderBatch(packageDir string, scenes []exportplan.AnnotationElementScenePlan) (<-chan AnnotationRenderBatchResult, error) {
	if len(scenes) == 0 {
		done := make(chan AnnotationRenderBatchResult, 1)
		done <- AnnotationRenderBatchResult{}
		return done, nil
	}
	jobs := make([]*annotationRenderJobState, 0, len(scenes))
	for index, scene := range scenes {
		job, err := annotationRenderJobFromScene(packageDir, scene, index+1)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, &annotationRenderJobState{job: job})
	}
	batch := &annotationRenderBatch{
		id:         fmt.Sprintf("annotation-render-%d", time.Now().UnixNano()),
		packageDir: packageDir,
		jobs:       jobs,
		done:       make(chan AnnotationRenderBatchResult, 1),
	}
	s.annotationRenderMu.Lock()
	s.annotationRenderBatch = batch
	s.annotationRenderMu.Unlock()
	return batch.done, nil
}

func annotationRenderJobFromScene(packageDir string, scene exportplan.AnnotationElementScenePlan, index int) (AnnotationRenderJob, error) {
	if strings.TrimSpace(packageDir) == "" {
		return AnnotationRenderJob{}, errors.New("annotation render packageDir is required")
	}
	packageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return AnnotationRenderJob{}, err
	}
	scenePath, sceneRel, err := annotationRenderPackagePath(packageDir, scene.InputPath, scene.RelativePath, recpackage.AnnotationRenderDir, ".excalidraw")
	if err != nil {
		return AnnotationRenderJob{}, err
	}
	outputRel := strings.TrimSpace(scene.RenderRelativePath)
	if outputRel == "" {
		outputRel = filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, fmt.Sprintf("annotation-%06d.png", index)))
	}
	outputPath, outputRel, err := annotationRenderPackagePath(packageDir, scene.RenderInputPath, outputRel, recpackage.AnnotationRenderPNGDir, ".png")
	if err != nil {
		return AnnotationRenderJob{}, err
	}
	if scene.CanvasWidth <= 0 || scene.CanvasHeight <= 0 {
		return AnnotationRenderJob{}, fmt.Errorf("annotation scene %q has invalid canvas size %dx%d", sceneRel, scene.CanvasWidth, scene.CanvasHeight)
	}
	info, err := os.Stat(scenePath)
	if err != nil {
		return AnnotationRenderJob{}, err
	}
	if info.IsDir() || info.Size() <= 0 {
		return AnnotationRenderJob{}, fmt.Errorf("annotation scene %q is not readable", sceneRel)
	}
	if info.Size() > maxWhiteboardSceneBytes {
		return AnnotationRenderJob{}, fmt.Errorf("annotation scene %q exceeds %d bytes", sceneRel, maxWhiteboardSceneBytes)
	}
	data, err := os.ReadFile(scenePath)
	if err != nil {
		return AnnotationRenderJob{}, err
	}
	if !json.Valid(data) {
		return AnnotationRenderJob{}, fmt.Errorf("annotation scene %q is invalid JSON", sceneRel)
	}
	return AnnotationRenderJob{
		ID:                 fmt.Sprintf("annotation-render-%06d", index),
		PackageDir:         packageDir,
		ScenePath:          scenePath,
		RelativeScenePath:  sceneRel,
		OutputPath:         outputPath,
		RelativeOutputPath: outputRel,
		SceneJSON:          string(data),
		CanvasWidth:        scene.CanvasWidth,
		CanvasHeight:       scene.CanvasHeight,
		Index:              index,
		StartOffsetMs:      scene.StartOffsetMs,
		EndOffsetMs:        scene.EndOffsetMs,
	}, nil
}

func annotationRenderPackagePath(packageDir string, absoluteOrEmpty string, relative string, requiredDir string, requiredExt string) (string, string, error) {
	if strings.TrimSpace(relative) == "" && strings.TrimSpace(absoluteOrEmpty) == "" {
		return "", "", errors.New("annotation render path is required")
	}
	var target string
	var err error
	if strings.TrimSpace(absoluteOrEmpty) != "" {
		target, err = filepath.Abs(absoluteOrEmpty)
		if err != nil {
			return "", "", err
		}
	} else {
		if filepath.IsAbs(relative) {
			return "", "", fmt.Errorf("annotation render path must be package-relative, got %q", relative)
		}
		target = filepath.Join(packageDir, filepath.Clean(relative))
	}
	rel, err := filepath.Rel(packageDir, target)
	if err != nil {
		return "", "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("annotation render path %q must stay inside package %q", target, packageDir)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	prefix := filepath.ToSlash(requiredDir) + "/"
	if !strings.HasPrefix(rel, prefix) || strings.ToLower(filepath.Ext(rel)) != requiredExt {
		return "", "", fmt.Errorf("annotation render path %q must stay under %s and use %s", rel, requiredDir, requiredExt)
	}
	return target, rel, nil
}

func (s *RecordingFreedomService) ClaimAnnotationRenderJob() (AnnotationRenderJobClaim, error) {
	s.annotationRenderMu.Lock()
	defer s.annotationRenderMu.Unlock()
	batch := s.annotationRenderBatch
	if batch == nil || batch.completed {
		return AnnotationRenderJobClaim{Available: false}, nil
	}
	for _, state := range batch.jobs {
		if state.done || state.claimed {
			continue
		}
		state.claimed = true
		job := state.job
		return AnnotationRenderJobClaim{Available: true, Job: &job}, nil
	}
	return AnnotationRenderJobClaim{Available: false}, nil
}

func (s *RecordingFreedomService) CompleteAnnotationRenderJob(result AnnotationRenderJobResult) error {
	id := strings.TrimSpace(result.ID)
	if id == "" {
		return errors.New("annotation render job id is required")
	}
	s.annotationRenderMu.Lock()
	defer s.annotationRenderMu.Unlock()
	batch := s.annotationRenderBatch
	if batch == nil || batch.completed {
		return errors.New("annotation render batch is not active")
	}
	for _, state := range batch.jobs {
		if state.job.ID != id {
			continue
		}
		if state.done {
			return nil
		}
		if message := strings.TrimSpace(result.Error); message != "" {
			state.done = true
			state.err = message
			s.finishAnnotationRenderBatchLocked()
			return nil
		}
		data, err := decodeAnnotationRenderedPNG(result.DataURL)
		if err != nil {
			state.done = true
			state.err = err.Error()
			s.finishAnnotationRenderBatchLocked()
			return nil
		}
		if err := writeAnnotationRenderedPNG(state.job.OutputPath, data); err != nil {
			state.done = true
			state.err = err.Error()
			s.finishAnnotationRenderBatchLocked()
			return nil
		}
		state.done = true
		s.finishAnnotationRenderBatchLocked()
		return nil
	}
	return fmt.Errorf("annotation render job %q was not found", id)
}

func (s *RecordingFreedomService) cancelAnnotationRenderBatch(message string) {
	s.annotationRenderMu.Lock()
	defer s.annotationRenderMu.Unlock()
	batch := s.annotationRenderBatch
	if batch == nil || batch.completed {
		return
	}
	for _, state := range batch.jobs {
		if !state.done {
			state.done = true
			state.err = message
		}
	}
	s.finishAnnotationRenderBatchLocked()
}

func (s *RecordingFreedomService) finishAnnotationRenderBatchLocked() {
	batch := s.annotationRenderBatch
	if batch == nil || batch.completed {
		return
	}
	result := AnnotationRenderBatchResult{}
	allDone := true
	for _, state := range batch.jobs {
		if !state.done {
			allDone = false
			break
		}
		if state.err != "" {
			result.Failed++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", state.job.RelativeScenePath, state.err))
		} else {
			result.Rendered++
		}
	}
	if !allDone {
		return
	}
	batch.completed = true
	s.annotationRenderBatch = nil
	batch.done <- result
}

func decodeAnnotationRenderedPNG(dataURL string) ([]byte, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return nil, errors.New("annotation rendered PNG data URL is required")
	}
	if !strings.HasPrefix(dataURL, whiteboardPNGContentPrefix) {
		return nil, errors.New("annotation rendered image must be a PNG data URL")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, whiteboardPNGContentPrefix))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("annotation rendered PNG is empty")
	}
	if len(data) > maxWhiteboardExportBytes {
		return nil, fmt.Errorf("annotation rendered PNG exceeds %d bytes", maxWhiteboardExportBytes)
	}
	return data, nil
}

func writeAnnotationRenderedPNG(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
