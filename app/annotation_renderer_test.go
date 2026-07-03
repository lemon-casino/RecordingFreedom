package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/exportplan"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestAnnotationRenderJobFromSceneValidatesPackagePaths(t *testing.T) {
	packageDir := filepath.Join(t.TempDir(), "recording-render-test"+recpackage.PackageDirSuffix)
	scenePath := filepath.Join(packageDir, recpackage.AnnotationRenderDir, "scene-000001.excalidraw")
	if err := os.MkdirAll(filepath.Dir(scenePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(scene dir) error = %v", err)
	}
	if err := os.WriteFile(scenePath, []byte(`{"type":"excalidraw","elements":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile(scene) error = %v", err)
	}

	job, err := annotationRenderJobFromScene(packageDir, exportplan.AnnotationElementScenePlan{
		InputPath:          scenePath,
		RelativePath:       filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderDir, "scene-000001.excalidraw")),
		RenderRelativePath: filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, "annotation-000001.png")),
		CanvasWidth:        1280,
		CanvasHeight:       720,
	}, 1)
	if err != nil {
		t.Fatalf("annotationRenderJobFromScene() error = %v", err)
	}
	if job.RelativeOutputPath != "annotations/reconstructed/png/annotation-000001.png" || job.CanvasWidth != 1280 || job.SceneJSON == "" {
		t.Fatalf("job = %#v, want validated package render job", job)
	}

	_, err = annotationRenderJobFromScene(packageDir, exportplan.AnnotationElementScenePlan{
		InputPath:          scenePath,
		RelativePath:       filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderDir, "scene-000001.excalidraw")),
		RenderRelativePath: "../annotation.png",
		CanvasWidth:        1280,
		CanvasHeight:       720,
	}, 1)
	if err == nil || !strings.Contains(err.Error(), "inside package") {
		t.Fatalf("escaping render path error = %v, want rejection", err)
	}
}

func TestCompleteAnnotationRenderJobWritesPNG(t *testing.T) {
	packageDir := filepath.Join(t.TempDir(), "recording-render-test"+recpackage.PackageDirSuffix)
	outputPath := filepath.Join(packageDir, recpackage.AnnotationRenderPNGDir, "annotation-000001.png")
	service := &RecordingFreedomService{}
	batch := &annotationRenderBatch{
		id:         "batch",
		packageDir: packageDir,
		jobs: []*annotationRenderJobState{{
			job: AnnotationRenderJob{
				ID:                 "annotation-render-000001",
				OutputPath:         outputPath,
				RelativeScenePath:  filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderDir, "scene-000001.excalidraw")),
				RelativeOutputPath: filepath.ToSlash(filepath.Join(recpackage.AnnotationRenderPNGDir, "annotation-000001.png")),
			},
			claimed: true,
		}},
		done: make(chan AnnotationRenderBatchResult, 1),
	}
	service.annotationRenderBatch = batch
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png bytes"))

	if err := service.CompleteAnnotationRenderJob(AnnotationRenderJobResult{ID: "annotation-render-000001", DataURL: dataURL}); err != nil {
		t.Fatalf("CompleteAnnotationRenderJob() error = %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(output) error = %v", err)
	}
	if string(data) != "png bytes" {
		t.Fatalf("output data = %q, want png bytes", data)
	}
	result := <-batch.done
	if result.Rendered != 1 || result.Failed != 0 {
		t.Fatalf("batch result = %#v, want one rendered job", result)
	}
}
