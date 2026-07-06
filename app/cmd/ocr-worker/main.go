package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const (
	protocolVersion = "ocr-jsonl-v1"
	workerName      = "rf-ocr-worker"
	workerVersion   = "0.1.0-protocol"
)

var requiredModelFiles = []string{"manifest.json", "det.onnx", "cls.onnx", "rec.onnx", "keys.txt"}

type protocolRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type protocolResponse struct {
	ID         string                  `json:"id"`
	OK         bool                    `json:"ok"`
	Result     *ocr.Result             `json:"result,omitempty"`
	Init       *initResult             `json:"init,omitempty"`
	Preprocess *imagePreprocessSummary `json:"preprocess,omitempty"`
	Inference  *ortInferenceSummary    `json:"inference,omitempty"`
	Error      *protocolError          `json:"error,omitempty"`
}

type protocolError struct {
	Code        string `json:"code,omitempty"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable,omitempty"`
}

type initParams struct {
	ModelDir   string `json:"modelDir"`
	RuntimeDir string `json:"runtimeDir"`
	Threads    int    `json:"threads"`
	Language   string `json:"language"`
}

type recognizeParams struct {
	ImagePath string `json:"imagePath"`
	MaxSide   int    `json:"maxSide"`
}

type workerState struct {
	initialized    bool
	modelDir       string
	runtimeDir     string
	runtimeLibrary string
	modelBundle    *ortModelBundle
}

type smokeResult struct {
	OK             bool     `json:"ok"`
	RuntimeDir     string   `json:"runtimeDir,omitempty"`
	ModelDir       string   `json:"modelDir,omitempty"`
	ImagePath      string   `json:"imagePath,omitempty"`
	MustContain    []string `json:"mustContain,omitempty"`
	PlainText      string   `json:"plainText,omitempty"`
	Blocks         int      `json:"blocks"`
	CandidateCount int      `json:"candidateCount"`
	Error          string   `json:"error,omitempty"`
}

type initResult struct {
	RuntimeVersion    string                `json:"runtimeVersion,omitempty"`
	RuntimeAPIVersion int                   `json:"runtimeApiVersion,omitempty"`
	Models            []modelSessionSummary `json:"models,omitempty"`
}

type modelSessionSummary struct {
	Kind    string          `json:"kind"`
	Path    string          `json:"path,omitempty"`
	Inputs  []tensorSummary `json:"inputs"`
	Outputs []tensorSummary `json:"outputs"`
}

type tensorSummary struct {
	Name            string  `json:"name"`
	ElementTypeCode int32   `json:"elementTypeCode"`
	ElementType     string  `json:"elementType"`
	Dimensions      []int64 `json:"dimensions"`
}

func (s *workerState) close() {
	if s == nil {
		return
	}
	if s.modelBundle != nil {
		s.modelBundle.close()
		s.modelBundle = nil
	}
	s.initialized = false
}

func (s *workerState) initResult() *initResult {
	if s == nil || s.modelBundle == nil {
		return nil
	}
	result := &initResult{}
	if s.modelBundle.runtime != nil {
		result.RuntimeVersion = s.modelBundle.runtime.version
		result.RuntimeAPIVersion = s.modelBundle.runtime.apiVersion
	}
	for _, model := range s.modelBundle.models {
		result.Models = append(result.Models, modelSessionSummary{
			Kind:    model.Kind,
			Path:    model.Path,
			Inputs:  tensorSummaries(model.Inputs),
			Outputs: tensorSummaries(model.Outputs),
		})
	}
	return result
}

func tensorSummaries(values []ortTensorMetadata) []tensorSummary {
	result := make([]tensorSummary, 0, len(values))
	for _, value := range values {
		result = append(result, tensorSummary{
			Name:            value.Name,
			ElementTypeCode: value.ElementTypeCode,
			ElementType:     value.ElementType,
			Dimensions:      append([]int64(nil), value.Dimensions...),
		})
	}
	return result
}

type runtimeCheck struct {
	dir        string
	library    string
	available  bool
	version    string
	apiVersion int
	code       string
	err        string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if hasFlag(args, "--capabilities") {
		if err := writeCapabilities(stdout, runtimeDirFromArgs(args)); err != nil {
			fmt.Fprintf(stderr, "write capabilities: %v\n", err)
			return 2
		}
		return 0
	}
	if hasFlag(args, "--smoke") {
		return runSmoke(args, stdout, stderr)
	}

	decoder := json.NewDecoder(bufio.NewReader(stdin))
	encoder := json.NewEncoder(stdout)
	state := workerState{}
	defer state.close()
	for {
		var req protocolRequest
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			fmt.Fprintf(stderr, "decode request: %v\n", err)
			return 2
		}
		if strings.TrimSpace(req.ID) == "" {
			writeProtocolResponse(encoder, protocolResponse{
				ID: "",
				OK: false,
				Error: &protocolError{
					Code:    "invalid_request",
					Message: "request id is required",
				},
			})
			continue
		}
		switch req.Method {
		case "init":
			nextState, initErr := validateInit(req.Params)
			if initErr != nil {
				writeProtocolResponse(encoder, protocolResponse{
					ID:    req.ID,
					OK:    false,
					Error: initErr,
				})
				continue
			}
			state.close()
			state = nextState
			writeProtocolResponse(encoder, protocolResponse{ID: req.ID, OK: true, Init: state.initResult()})
		case "recognize":
			if !state.initialized {
				writeProtocolResponse(encoder, protocolResponse{
					ID: req.ID,
					OK: false,
					Error: &protocolError{
						Code:        "model_not_loaded",
						Message:     "OCR model is not loaded. Send init before recognize.",
						Recoverable: true,
					},
				})
				continue
			}
			preprocessed, preprocessErr := validateAndPreprocessRecognize(req.Params)
			if preprocessErr != nil {
				writeProtocolResponse(encoder, protocolResponse{
					ID:    req.ID,
					OK:    false,
					Error: preprocessErr,
				})
				continue
			}
			recognition, recognitionErr := runONNXRecognition(state.modelBundle, preprocessed)
			if recognitionErr != nil {
				writeProtocolResponse(encoder, protocolResponse{
					ID:         req.ID,
					OK:         false,
					Preprocess: &preprocessed.Summary,
					Inference:  &recognition.Inference,
					Error: &protocolError{
						Code:        "ocr_recognition_failed",
						Message:     "OCR recognition failed: " + recognitionErr.Error(),
						Recoverable: true,
					},
				})
				continue
			}
			writeProtocolResponse(encoder, protocolResponse{
				ID:         req.ID,
				OK:         true,
				Result:     recognition.Result,
				Preprocess: &preprocessed.Summary,
				Inference:  &recognition.Inference,
			})
		case "release":
			state.close()
			writeProtocolResponse(encoder, protocolResponse{ID: req.ID, OK: true})
			return 0
		default:
			writeProtocolResponse(encoder, protocolResponse{
				ID: req.ID,
				OK: false,
				Error: &protocolError{
					Code:    "unknown_method",
					Message: "unknown method " + req.Method,
				},
			})
		}
	}
}

func runSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	modelDir := strings.TrimSpace(argValue(args, "--model-dir"))
	runtimeDir := runtimeDirFromArgs(args)
	imagePath := strings.TrimSpace(argValue(args, "--image"))
	if imagePath == "" && modelDir != "" {
		imagePath = filepath.Join(modelDir, "smoke.png")
	}
	mustContain := argValues(args, "--must-contain")
	if len(mustContain) == 0 {
		mustContain = []string{"RecordingFreedom", "文字识别"}
	}
	result := smokeResult{
		RuntimeDir:  runtimeDir,
		ModelDir:    modelDir,
		ImagePath:   imagePath,
		MustContain: mustContain,
	}
	write := func(code int) int {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintf(stderr, "write smoke result: %v\n", err)
			return 2
		}
		return code
	}
	if modelDir == "" {
		result.Error = "model directory is required"
		return write(2)
	}
	if imagePath == "" {
		result.Error = "smoke image is required"
		return write(2)
	}
	if err := validateModelDir(modelDir); err != nil {
		result.Error = err.Message
		return write(1)
	}
	runtimeStatus := checkRuntime(runtimeDir)
	if !runtimeStatus.available {
		result.Error = runtimeStatus.err
		return write(1)
	}
	bundle, err := loadONNXModelBundle(runtimeStatus.library, modelDir)
	if err != nil {
		result.Error = "model session preflight failed: " + err.Error()
		return write(1)
	}
	defer bundle.close()
	preprocessed, err := preprocessImageFile(imagePath, smokeMaxSide(args))
	if err != nil {
		result.Error = "image preprocess failed: " + err.Error()
		return write(1)
	}
	recognition, err := runONNXRecognition(bundle, preprocessed)
	if err != nil {
		result.CandidateCount = recognition.Inference.CandidateCount
		result.Error = "recognition failed: " + err.Error()
		return write(1)
	}
	if recognition.Result != nil {
		result.PlainText = recognition.Result.PlainText
		result.Blocks = len(recognition.Result.Blocks)
	}
	result.CandidateCount = recognition.Inference.CandidateCount
	for _, expected := range mustContain {
		if !strings.Contains(result.PlainText, expected) {
			result.Error = "smoke text missing required content: " + expected
			return write(1)
		}
	}
	result.OK = true
	return write(0)
}

func validateAndPreprocessRecognize(raw json.RawMessage) (imagePreprocessResult, *protocolError) {
	var params recognizeParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return imagePreprocessResult{}, &protocolError{
				Code:        "invalid_recognize_params",
				Message:     "invalid recognize params: " + err.Error(),
				Recoverable: true,
			}
		}
	}
	preprocessed, err := preprocessImageFile(params.ImagePath, params.MaxSide)
	if err != nil {
		return imagePreprocessResult{}, &protocolError{
			Code:        "image_preprocess_failed",
			Message:     "image preprocess failed: " + err.Error(),
			Recoverable: true,
		}
	}
	if len(preprocessed.Tensor) == 0 {
		return imagePreprocessResult{}, &protocolError{
			Code:        "image_preprocess_failed",
			Message:     "image preprocess failed: empty tensor",
			Recoverable: true,
		}
	}
	return preprocessed, nil
}

func writeCapabilities(writer io.Writer, runtimeDir string) error {
	runtimeStatus := checkRuntime(runtimeDir)
	message := "OCR inference pipeline is available."
	if !runtimeStatus.available {
		message = runtimeStatus.err
	}
	return json.NewEncoder(writer).Encode(ocr.WorkerCapabilities{
		SchemaVersion:     1,
		Name:              workerName,
		Version:           workerVersion,
		ProtocolVersion:   protocolVersion,
		Engine:            "onnxruntime",
		ModelFormats:      []string{"onnx"},
		SupportsRecognize: runtimeStatus.available,
		RuntimeDir:        runtimeStatus.dir,
		RuntimeLibrary:    runtimeStatus.library,
		RuntimeAvailable:  runtimeStatus.available,
		RuntimeVersion:    runtimeStatus.version,
		RuntimeAPIVersion: runtimeStatus.apiVersion,
		RuntimeError:      runtimeStatus.err,
		Message:           message,
	})
}

func writeProtocolResponse(encoder *json.Encoder, response protocolResponse) {
	_ = encoder.Encode(response)
}

func validateInit(raw json.RawMessage) (workerState, *protocolError) {
	var params initParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return workerState{}, &protocolError{
				Code:    "invalid_init_params",
				Message: "invalid init params: " + err.Error(),
			}
		}
	}
	modelDir := strings.TrimSpace(params.ModelDir)
	if modelDir == "" {
		return workerState{}, &protocolError{
			Code:        "model_dir_required",
			Message:     "OCR model directory is required.",
			Recoverable: true,
		}
	}
	if err := validateModelDir(modelDir); err != nil {
		return workerState{}, err
	}
	runtimeStatus := checkRuntime(params.RuntimeDir)
	if !runtimeStatus.available {
		code := runtimeStatus.code
		if code == "" {
			code = "onnx_runtime_unavailable"
		}
		return workerState{}, &protocolError{
			Code:        code,
			Message:     runtimeStatus.err,
			Recoverable: true,
		}
	}
	modelBundle, err := loadONNXModelBundle(runtimeStatus.library, modelDir)
	if err != nil {
		return workerState{}, &protocolError{
			Code:        "model_session_failed",
			Message:     "OCR model session preflight failed: " + err.Error(),
			Recoverable: true,
		}
	}
	return workerState{
		initialized:    true,
		modelDir:       modelDir,
		runtimeDir:     runtimeStatus.dir,
		runtimeLibrary: runtimeStatus.library,
		modelBundle:    modelBundle,
	}, nil
}

func validateModelDir(modelDir string) *protocolError {
	info, err := os.Stat(modelDir)
	if err != nil {
		return &protocolError{
			Code:        "model_dir_unavailable",
			Message:     "OCR model directory is unavailable: " + err.Error(),
			Recoverable: true,
		}
	}
	if !info.IsDir() {
		return &protocolError{
			Code:        "model_dir_invalid",
			Message:     "OCR model path is not a directory: " + modelDir,
			Recoverable: true,
		}
	}
	manifest, manifestErr := readWorkerModelManifest(modelDir)
	if manifestErr != nil {
		return &protocolError{
			Code:        "model_file_missing",
			Message:     "OCR model manifest is unavailable: " + manifestErr.Error(),
			Recoverable: true,
		}
	}
	if err := ocr.ValidateTextlineOrientationMode(manifest); err != nil {
		return &protocolError{
			Code:        "model_manifest_invalid",
			Message:     "OCR model manifest is invalid: " + err.Error(),
			Recoverable: true,
		}
	}
	for _, name := range requiredWorkerModelFiles(manifest) {
		path := filepath.Join(modelDir, name)
		fileInfo, err := os.Stat(path)
		if err != nil {
			return &protocolError{
				Code:        "model_file_missing",
				Message:     "OCR model file is missing: " + name,
				Recoverable: true,
			}
		}
		if fileInfo.IsDir() {
			return &protocolError{
				Code:        "model_file_invalid",
				Message:     "OCR model file path is a directory: " + name,
				Recoverable: true,
			}
		}
	}
	return nil
}

func readWorkerModelManifest(modelDir string) (ocr.ModelManifest, error) {
	data, err := os.ReadFile(filepath.Join(modelDir, "manifest.json"))
	if err != nil {
		return ocr.ModelManifest{}, err
	}
	var manifest ocr.ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ocr.ModelManifest{}, err
	}
	if manifest.SchemaVersion != 1 {
		return ocr.ModelManifest{}, fmt.Errorf("unsupported OCR model manifest schema %d", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return ocr.ModelManifest{}, errors.New("OCR model manifest id is required")
	}
	return manifest, nil
}

func requiredWorkerModelFiles(manifest ocr.ModelManifest) []string {
	return append([]string{"manifest.json"}, ocr.RequiredModelFileNames(manifest)...)
}

func checkRuntime(runtimeDir string) runtimeCheck {
	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		return runtimeCheck{code: "onnx_runtime_missing", err: "ONNX Runtime directory is not provided."}
	}
	absoluteDir, err := filepath.Abs(runtimeDir)
	if err == nil {
		runtimeDir = absoluteDir
	}
	candidates := runtimeLibraryCandidates()
	for _, name := range candidates {
		path := filepath.Join(runtimeDir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			probe, probeErr := probeONNXRuntimeLibrary(path)
			if probeErr != nil {
				return runtimeCheck{
					dir:     runtimeDir,
					library: path,
					code:    "onnx_runtime_unavailable",
					err:     "ONNX Runtime library was found but could not be loaded: " + probeErr.Error(),
				}
			}
			return runtimeCheck{
				dir:        runtimeDir,
				library:    path,
				available:  true,
				version:    probe.Version,
				apiVersion: probe.APIVersion,
			}
		}
	}
	return runtimeCheck{
		dir:  runtimeDir,
		code: "onnx_runtime_missing",
		err:  "ONNX Runtime library was not found under " + runtimeDir + ". Expected one of: " + strings.Join(candidates, ", "),
	}
}

func runtimeLibraryCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"onnxruntime.dll"}
	case "darwin":
		return []string{"libonnxruntime.dylib"}
	case "linux":
		return []string{"libonnxruntime.so"}
	default:
		return []string{"onnxruntime"}
	}
}

func runtimeDirFromArgs(args []string) string {
	if value := strings.TrimSpace(argValue(args, "--runtime-dir")); value != "" {
		return value
	}
	return os.Getenv("RF_OCR_RUNTIME_DIR")
}

func smokeMaxSide(args []string) int {
	value := strings.TrimSpace(argValue(args, "--max-side"))
	if value == "" {
		return defaultImagePreprocessMaxSide
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultImagePreprocessMaxSide
	}
	return parsed
}

func argValue(args []string, target string) string {
	for i, arg := range args {
		if arg == target && i+1 < len(args) {
			return args[i+1]
		}
		if value, ok := strings.CutPrefix(arg, target+"="); ok {
			return value
		}
	}
	return ""
}

func argValues(args []string, target string) []string {
	values := make([]string, 0)
	for i, arg := range args {
		if arg == target && i+1 < len(args) {
			values = appendSplitArgValues(values, args[i+1])
			continue
		}
		if value, ok := strings.CutPrefix(arg, target+"="); ok {
			values = appendSplitArgValues(values, value)
		}
	}
	return values
}

func appendSplitArgValues(values []string, value string) []string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func hasFlag(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
