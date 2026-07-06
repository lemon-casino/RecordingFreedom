package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

func TestCapabilitiesReportNoFakeRecognition(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--capabilities"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--capabilities) code = %d stderr=%s", code, stderr.String())
	}
	var capabilities ocr.WorkerCapabilities
	if err := json.Unmarshal(stdout.Bytes(), &capabilities); err != nil {
		t.Fatalf("Unmarshal(capabilities) error = %v; output=%s", err, stdout.String())
	}
	if capabilities.SchemaVersion != 1 || capabilities.ProtocolVersion != protocolVersion {
		t.Fatalf("capabilities = %#v, want schema 1 and protocol %s", capabilities, protocolVersion)
	}
	if capabilities.SupportsRecognize {
		t.Fatalf("supportsRecognize = true without a loadable runtime")
	}
	if capabilities.RuntimeAvailable {
		t.Fatalf("runtimeAvailable = true, want false without runtime dir")
	}
}

func TestCapabilitiesRejectFakeRuntimeLibrary(t *testing.T) {
	runtimeDir := writeTestRuntime(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--capabilities", "--runtime-dir", runtimeDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--capabilities --runtime-dir) code = %d stderr=%s", code, stderr.String())
	}
	var capabilities ocr.WorkerCapabilities
	if err := json.Unmarshal(stdout.Bytes(), &capabilities); err != nil {
		t.Fatalf("Unmarshal(capabilities) error = %v; output=%s", err, stdout.String())
	}
	if capabilities.RuntimeAvailable {
		t.Fatalf("runtimeAvailable = true for fake runtime; capabilities=%#v", capabilities)
	}
	if capabilities.RuntimeLibrary == "" {
		t.Fatalf("runtimeLibrary is empty; capabilities=%#v", capabilities)
	}
	if capabilities.RuntimeError == "" {
		t.Fatalf("runtimeError is empty for fake runtime; capabilities=%#v", capabilities)
	}
	if capabilities.SupportsRecognize {
		t.Fatalf("supportsRecognize = true with an unloadable runtime")
	}
}

func TestCapabilitiesReportLoadableRuntimeWithProbe(t *testing.T) {
	withFakeRuntimeProbe(t)
	runtimeDir := writeTestRuntime(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--capabilities", "--runtime-dir", runtimeDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(--capabilities --runtime-dir) code = %d stderr=%s", code, stderr.String())
	}
	var capabilities ocr.WorkerCapabilities
	if err := json.Unmarshal(stdout.Bytes(), &capabilities); err != nil {
		t.Fatalf("Unmarshal(capabilities) error = %v; output=%s", err, stdout.String())
	}
	if !capabilities.RuntimeAvailable || capabilities.RuntimeVersion != "1.23.2-test" || capabilities.RuntimeAPIVersion != ortAPIVersion {
		t.Fatalf("capabilities=%#v, want loadable runtime metadata", capabilities)
	}
	if !capabilities.SupportsRecognize {
		t.Fatalf("supportsRecognize = false with a loadable runtime; capabilities=%#v", capabilities)
	}
}

func TestRecognizeReturnsOCRResult(t *testing.T) {
	withFakeRuntimeProbe(t)
	closedCount := withFakeModelBundle(t, nil)
	withFakeRecognitionRun(t)
	modelDir := writeTestModel(t, true)
	runtimeDir := writeTestRuntime(t)
	imagePath := writeSolidPNG(t, t.TempDir(), 128, 64, colorForProtocolTest())
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(runtimeDir) + `}}` + "\n" +
			`{"id":"recognize-1","method":"recognize","params":{"imagePath":` + quoteJSON(imagePath) + `,"maxSide":64}}` + "\n" +
			`{"id":"release-1","method":"release"}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	decoder := json.NewDecoder(&stdout)
	var initResponse protocolResponse
	if err := decoder.Decode(&initResponse); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if initResponse.ID != "init-1" || !initResponse.OK {
		t.Fatalf("init response = %#v, want ok", initResponse)
	}
	if initResponse.Init == nil || len(initResponse.Init.Models) != 3 || initResponse.Init.Models[0].Kind != "det" {
		t.Fatalf("init summary = %#v, want three model session summaries", initResponse.Init)
	}
	if len(initResponse.Init.Models[0].Inputs) != 1 || initResponse.Init.Models[0].Inputs[0].Name != "x" || initResponse.Init.Models[0].Inputs[0].ElementType != "float32" {
		t.Fatalf("det input summary = %#v, want tensor metadata", initResponse.Init.Models[0].Inputs)
	}
	var recognizeResponse protocolResponse
	if err := decoder.Decode(&recognizeResponse); err != nil {
		t.Fatalf("Decode(recognize) error = %v", err)
	}
	if recognizeResponse.ID != "recognize-1" || !recognizeResponse.OK || recognizeResponse.Result == nil {
		t.Fatalf("recognize response = %#v, want OCR result", recognizeResponse)
	}
	if recognizeResponse.Preprocess == nil || recognizeResponse.Preprocess.InputWidth != 64 || recognizeResponse.Preprocess.InputHeight != 32 {
		t.Fatalf("recognize preprocess summary = %#v, want 64x32", recognizeResponse.Preprocess)
	}
	if recognizeResponse.Inference == nil || recognizeResponse.Inference.Kind != "det" || recognizeResponse.Inference.OutputName != "fetch_name_0" {
		t.Fatalf("recognize inference summary = %#v, want det output summary", recognizeResponse.Inference)
	}
	if recognizeResponse.Result.PlainText != "RecordingFreedom" || len(recognizeResponse.Result.Blocks) != 1 {
		t.Fatalf("recognize result = %#v, want one RecordingFreedom block", recognizeResponse.Result)
	}
	var releaseResponse protocolResponse
	if err := decoder.Decode(&releaseResponse); err != nil {
		t.Fatalf("Decode(release) error = %v", err)
	}
	if releaseResponse.ID != "release-1" || !releaseResponse.OK {
		t.Fatalf("release response = %#v, want ok", releaseResponse)
	}
	if *closedCount != 1 {
		t.Fatalf("model bundle close count = %d, want 1 after release", *closedCount)
	}
}

func TestInitRejectsMissingRuntime(t *testing.T) {
	modelDir := writeTestModel(t, true)
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(filepath.Join(t.TempDir(), "missing-runtime")) + `}}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	var response protocolResponse
	if err := json.NewDecoder(&stdout).Decode(&response); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if response.OK || response.Error == nil || response.Error.Code != "onnx_runtime_missing" {
		t.Fatalf("init response = %#v, want onnx_runtime_missing", response)
	}
}

func TestInitRejectsUnloadableRuntime(t *testing.T) {
	modelDir := writeTestModel(t, true)
	runtimeDir := writeTestRuntime(t)
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(runtimeDir) + `}}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	var response protocolResponse
	if err := json.NewDecoder(&stdout).Decode(&response); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if response.OK || response.Error == nil || response.Error.Code != "onnx_runtime_unavailable" {
		t.Fatalf("init response = %#v, want onnx_runtime_unavailable", response)
	}
}

func TestInitRejectsModelSessionFailure(t *testing.T) {
	withFakeRuntimeProbe(t)
	withFakeModelBundle(t, errors.New("rec.onnx: invalid graph"))
	modelDir := writeTestModel(t, true)
	runtimeDir := writeTestRuntime(t)
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(runtimeDir) + `}}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	var response protocolResponse
	if err := json.NewDecoder(&stdout).Decode(&response); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if response.OK || response.Error == nil || response.Error.Code != "model_session_failed" || !strings.Contains(response.Error.Message, "invalid graph") {
		t.Fatalf("init response = %#v, want model_session_failed with detail", response)
	}
}

func TestInitRejectsMissingModelFile(t *testing.T) {
	modelDir := writeTestModel(t, false)
	runtimeDir := writeTestRuntime(t)
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(runtimeDir) + `}}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	var response protocolResponse
	if err := json.NewDecoder(&stdout).Decode(&response); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if response.OK || response.Error == nil || response.Error.Code != "model_file_missing" {
		t.Fatalf("init response = %#v, want model_file_missing", response)
	}
}

func TestInitAllowsMissingClsWhenTextlineOrientationDisabled(t *testing.T) {
	modelDir := writeTestModelWithoutCls(t)
	if err := validateModelDir(modelDir); err != nil {
		t.Fatalf("validateModelDir(no cls, orientation none) error = %#v", err)
	}
	specs := modelSessionSpecs(ocr.ModelManifest{TextlineOrientation: &ocr.ModelTextlineOrientation{Mode: ocr.TextlineOrientationNone}})
	if len(specs) != 2 || specs[0].kind != "det" || specs[1].kind != "rec" {
		t.Fatalf("modelSessionSpecs(no orientation) = %#v, want det and rec only", specs)
	}
}

func TestRecognizeRejectsMissingImage(t *testing.T) {
	withFakeRuntimeProbe(t)
	withFakeModelBundle(t, nil)
	withFakeRecognitionRun(t)
	modelDir := writeTestModel(t, true)
	runtimeDir := writeTestRuntime(t)
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	input := strings.NewReader(
		`{"id":"init-1","method":"init","params":{"modelDir":` + quoteJSON(modelDir) + `,"runtimeDir":` + quoteJSON(runtimeDir) + `}}` + "\n" +
			`{"id":"recognize-1","method":"recognize","params":{"imagePath":` + quoteJSON(missingImage) + `}}` + "\n",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run(protocol) code = %d stderr=%s", code, stderr.String())
	}
	decoder := json.NewDecoder(&stdout)
	var initResponse protocolResponse
	if err := decoder.Decode(&initResponse); err != nil {
		t.Fatalf("Decode(init) error = %v", err)
	}
	if !initResponse.OK {
		t.Fatalf("init response = %#v, want ok", initResponse)
	}
	var recognizeResponse protocolResponse
	if err := decoder.Decode(&recognizeResponse); err != nil {
		t.Fatalf("Decode(recognize) error = %v", err)
	}
	if recognizeResponse.OK || recognizeResponse.Error == nil || recognizeResponse.Error.Code != "image_preprocess_failed" {
		t.Fatalf("recognize response = %#v, want image_preprocess_failed", recognizeResponse)
	}
}

func withFakeRecognitionRun(t *testing.T) {
	t.Helper()
	previous := runONNXRecognition
	runONNXRecognition = func(_ *ortModelBundle, input imagePreprocessResult) (ocrRecognitionOutput, error) {
		if len(input.Tensor) == 0 {
			return ocrRecognitionOutput{}, errors.New("empty tensor")
		}
		inference := ortInferenceSummary{
			Kind:       "det",
			OutputName: "fetch_name_0",
			Shape:      []int64{1, 1, 16, 32},
			Sample:     []float32{0, 0.25, 0.5},
			Min:        0,
			Max:        0.5,
			Mean:       0.25,
		}
		return ocrRecognitionOutput{
			Inference: inference,
			Result: &ocr.Result{
				Width:     input.Summary.OriginalWidth,
				Height:    input.Summary.OriginalHeight,
				PlainText: "RecordingFreedom",
				Blocks: []ocr.Block{{
					ID:         "b1",
					Text:       "RecordingFreedom",
					Confidence: 0.98,
					Box: []ocr.Point{
						{X: 1, Y: 2},
						{X: 10, Y: 2},
						{X: 10, Y: 8},
						{X: 1, Y: 8},
					},
				}},
			},
		}, nil
	}
	t.Cleanup(func() {
		runONNXRecognition = previous
	})
}

func writeTestModel(t *testing.T, complete bool) string {
	t.Helper()
	dir := t.TempDir()
	files := []string{"det.onnx", "cls.onnx", "rec.onnx", "keys.txt"}
	if !complete {
		files = []string{"det.onnx", "cls.onnx", "keys.txt"}
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	writeTestModelManifest(t, dir, ocr.TextlineOrientationCLS, []string{"det.onnx", "cls.onnx", "rec.onnx", "keys.txt"})
	return dir
}

func writeTestModelWithoutCls(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"det.onnx", "rec.onnx", "keys.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	writeTestModelManifest(t, dir, ocr.TextlineOrientationNone, []string{"det.onnx", "rec.onnx", "keys.txt"})
	return dir
}

func writeTestModelManifest(t *testing.T, dir string, orientationMode string, files []string) {
	t.Helper()
	manifest := ocr.ModelManifest{
		SchemaVersion: 1,
		ID:            "test-model",
		Name:          "Test OCR Model",
		Channel:       "stable",
		Engine:        "onnxruntime",
		Language:      []string{"zh", "en"},
		Version:       "test",
		TextlineOrientation: &ocr.ModelTextlineOrientation{
			Mode: orientationMode,
		},
	}
	for _, name := range files {
		manifest.Files = append(manifest.Files, ocr.ModelFile{Name: name})
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
}

func writeTestRuntime(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, runtimeLibraryCandidates()[0]), []byte("runtime"), 0o644); err != nil {
		t.Fatalf("WriteFile(runtime) error = %v", err)
	}
	return dir
}

func withFakeRuntimeProbe(t *testing.T) {
	t.Helper()
	previous := probeONNXRuntimeLibrary
	probeONNXRuntimeLibrary = func(string) (ortRuntimeProbe, error) {
		return ortRuntimeProbe{Version: "1.23.2-test", APIVersion: ortAPIVersion}, nil
	}
	t.Cleanup(func() {
		probeONNXRuntimeLibrary = previous
	})
}

func withFakeModelBundle(t *testing.T, loadErr error) *int {
	t.Helper()
	closedCount := 0
	previous := loadONNXModelBundle
	loadONNXModelBundle = func(string, string) (*ortModelBundle, error) {
		if loadErr != nil {
			return nil, loadErr
		}
		return &ortModelBundle{
			models: []ortModelSession{
				{Kind: "det", Inputs: []ortTensorMetadata{{Name: "x", ElementType: "float32", Dimensions: []int64{1, 3, 640, 640}}}, Outputs: []ortTensorMetadata{{Name: "y", ElementType: "float32", Dimensions: []int64{1, 1, 160, 160}}}},
				{Kind: "cls", Inputs: []ortTensorMetadata{{Name: "x", ElementType: "float32", Dimensions: []int64{1, 3, 48, 192}}}, Outputs: []ortTensorMetadata{{Name: "y", ElementType: "float32", Dimensions: []int64{1, 2}}}},
				{Kind: "rec", Inputs: []ortTensorMetadata{{Name: "x", ElementType: "float32", Dimensions: []int64{1, 3, 48, 320}}}, Outputs: []ortTensorMetadata{{Name: "y", ElementType: "float32", Dimensions: []int64{1, 80, 6625}}}},
			},
			onClose: func() {
				closedCount++
			},
		}, nil
	}
	t.Cleanup(func() {
		loadONNXModelBundle = previous
	})
	return &closedCount
}

func colorForProtocolTest() color.NRGBA {
	return color.NRGBA{R: 255, G: 0, B: 0, A: 255}
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
