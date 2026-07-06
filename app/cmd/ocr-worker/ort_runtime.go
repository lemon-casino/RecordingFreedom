package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/lemon-casino/RecordingFreedom/app/internal/ocr"
)

const ortAPIVersion = 23

const (
	ortAPIGetErrorMessageIndex                = 2
	ortAPICreateEnvIndex                      = 3
	ortAPICreateSessionFromArrayIndex         = 8
	ortAPIRunIndex                            = 9
	ortAPICreateSessionOptionsIndex           = 10
	ortAPISessionGetInputCountIndex           = 30
	ortAPISessionGetOutputCountIndex          = 31
	ortAPISessionGetInputTypeInfoIndex        = 33
	ortAPISessionGetOutputTypeInfoIndex       = 34
	ortAPISessionGetInputNameIndex            = 36
	ortAPISessionGetOutputNameIndex           = 37
	ortAPICreateTensorWithDataAsOrtValueIndex = 49
	ortAPIGetTensorMutableDataIndex           = 51
	ortAPICastTypeInfoToTensorInfoIndex       = 55
	ortAPIGetTensorElementTypeIndex           = 60
	ortAPIGetDimensionsCountIndex             = 61
	ortAPIGetDimensionsIndex                  = 62
	ortAPIGetTensorShapeElementCountIndex     = 64
	ortAPIGetTensorTypeAndShapeIndex          = 65
	ortAPICreateCpuMemoryInfoIndex            = 69
	ortAPIAllocatorFreeIndex                  = 76
	ortAPIGetAllocatorWithDefaultOptionsIndex = 78
	ortAPIReleaseEnvIndex                     = 92
	ortAPIReleaseStatusIndex                  = 93
	ortAPIReleaseMemoryInfoIndex              = 94
	ortAPIReleaseSessionIndex                 = 95
	ortAPIReleaseValueIndex                   = 96
	ortAPIReleaseTypeInfoIndex                = 98
	ortAPIReleaseTensorTypeAndShapeInfoIndex  = 99
	ortAPIReleaseSessionOptionsIndex          = 100
)

const (
	onnxTensorElementDataTypeFloat = 1
	ortArenaAllocator              = 0
	ortMemTypeDefault              = 0
)

type ortRuntimeProbe struct {
	Version    string
	APIVersion int
}

var probeONNXRuntimeLibrary = probeONNXRuntimeLibraryWithPurego
var loadONNXModelBundle = loadONNXModelBundleWithPurego
var runONNXDetection = runONNXDetectionWithPurego

type ortApiBase struct {
	GetApi           uintptr
	GetVersionString uintptr
}

func probeONNXRuntimeLibraryWithPurego(libraryPath string) (ortRuntimeProbe, error) {
	runtimeHandle, err := openONNXRuntime(libraryPath)
	if err != nil {
		return ortRuntimeProbe{}, err
	}
	defer runtimeHandle.close()

	return ortRuntimeProbe{Version: runtimeHandle.version, APIVersion: runtimeHandle.apiVersion}, nil
}

type ortRuntimeHandle struct {
	libraryHandle uintptr
	api           ortAPI
	version       string
	apiVersion    int
	env           uintptr
	options       uintptr
}

func openONNXRuntime(libraryPath string) (*ortRuntimeHandle, error) {
	handle, err := openDynamicLibrary(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("load ONNX Runtime library: %w", err)
	}
	runtimeHandle := &ortRuntimeHandle{libraryHandle: handle}
	defer func() {
		if err != nil {
			runtimeHandle.close()
		}
	}()

	var ortGetAPIBase func() uintptr
	purego.RegisterLibFunc(&ortGetAPIBase, handle, "OrtGetApiBase")
	apiBasePtr := ortGetAPIBase()
	if apiBasePtr == 0 {
		err = errors.New("OrtGetApiBase returned null")
		return nil, err
	}

	apiBase := (*ortApiBase)(unsafe.Pointer(apiBasePtr))
	if apiBase.GetApi == 0 || apiBase.GetVersionString == 0 {
		err = errors.New("OrtApiBase is missing required function pointers")
		return nil, err
	}

	var getVersionString func() string
	purego.RegisterFunc(&getVersionString, apiBase.GetVersionString)
	version := getVersionString()
	if version == "" {
		err = errors.New("GetVersionString returned empty version")
		return nil, err
	}

	var getAPI func(uint32) uintptr
	purego.RegisterFunc(&getAPI, apiBase.GetApi)
	apiPtr := getAPI(ortAPIVersion)
	if apiPtr == 0 {
		err = fmt.Errorf("ONNX Runtime does not expose C API version %d", ortAPIVersion)
		return nil, err
	}

	api := ortAPI{ptr: apiPtr}
	env, err := api.createEnv()
	if err != nil {
		return nil, err
	}

	options, err := api.createSessionOptions()
	if err != nil {
		api.releaseEnv(env)
		return nil, err
	}

	runtimeHandle.api = api
	runtimeHandle.version = version
	runtimeHandle.apiVersion = ortAPIVersion
	runtimeHandle.env = env
	runtimeHandle.options = options
	return runtimeHandle, nil
}

func (h *ortRuntimeHandle) close() {
	if h == nil {
		return
	}
	if h.options != 0 {
		h.api.releaseSessionOptions(h.options)
		h.options = 0
	}
	if h.env != 0 {
		h.api.releaseEnv(h.env)
		h.env = 0
	}
	if h.libraryHandle != 0 {
		closeDynamicLibrary(h.libraryHandle)
		h.libraryHandle = 0
	}
}

type ortModelBundle struct {
	runtime                 *ortRuntimeHandle
	models                  []ortModelSession
	characters              []string
	textlineOrientationMode string
	closed                  bool
	onClose                 func()
}

type ortModelSession struct {
	Kind    string
	Path    string
	Session uintptr
	Inputs  []ortTensorMetadata
	Outputs []ortTensorMetadata
}

type ortTensorMetadata struct {
	Name            string
	ElementTypeCode int32
	ElementType     string
	Dimensions      []int64
}

type ortTensorOutput struct {
	Name  string
	Shape []int64
	Data  []float32
}

type ortInferenceSummary struct {
	Kind           string            `json:"kind"`
	OutputName     string            `json:"outputName"`
	Shape          []int64           `json:"shape"`
	Sample         []float32         `json:"sample,omitempty"`
	Min            float32           `json:"min"`
	Max            float32           `json:"max"`
	Mean           float32           `json:"mean"`
	CandidateCount int               `json:"candidateCount"`
	Candidates     []detCandidateBox `json:"candidates,omitempty"`
}

func loadONNXModelBundleWithPurego(libraryPath string, modelDir string) (*ortModelBundle, error) {
	manifest, err := readWorkerModelManifest(modelDir)
	if err != nil {
		return nil, err
	}
	if err := ocr.ValidateTextlineOrientationMode(manifest); err != nil {
		return nil, err
	}
	runtimeHandle, err := openONNXRuntime(libraryPath)
	if err != nil {
		return nil, err
	}
	bundle := &ortModelBundle{
		runtime:                 runtimeHandle,
		textlineOrientationMode: ocr.ModelTextlineOrientationMode(manifest),
	}
	defer func() {
		if err != nil {
			bundle.close()
		}
	}()

	for _, spec := range modelSessionSpecs(manifest) {
		session, sessionErr := runtimeHandle.api.loadModelSessionFromFile(runtimeHandle.env, runtimeHandle.options, filepath.Join(modelDir, spec.file))
		if sessionErr != nil {
			err = fmt.Errorf("%s: %w", spec.file, sessionErr)
			return nil, err
		}
		metadata, metadataErr := runtimeHandle.api.describeSession(session)
		if metadataErr != nil {
			runtimeHandle.api.releaseSession(session)
			err = fmt.Errorf("%s: %w", spec.file, metadataErr)
			return nil, err
		}
		bundle.models = append(bundle.models, ortModelSession{
			Kind:    spec.kind,
			Path:    filepath.Join(modelDir, spec.file),
			Session: session,
			Inputs:  metadata.inputs,
			Outputs: metadata.outputs,
		})
	}
	characters, err := loadOCRCharacters(filepath.Join(modelDir, "keys.txt"))
	if err != nil {
		return nil, err
	}
	bundle.characters = characters

	return bundle, nil
}

type modelSessionSpec struct {
	kind string
	file string
}

func modelSessionSpecs(manifest ocr.ModelManifest) []modelSessionSpec {
	specs := []modelSessionSpec{{kind: "det", file: "det.onnx"}}
	if ocr.ModelTextlineOrientationMode(manifest) == ocr.TextlineOrientationCLS {
		specs = append(specs, modelSessionSpec{kind: "cls", file: "cls.onnx"})
	}
	specs = append(specs, modelSessionSpec{kind: "rec", file: "rec.onnx"})
	return specs
}

func (b *ortModelBundle) close() {
	if b == nil || b.closed {
		return
	}
	b.closed = true
	if b.runtime != nil {
		for i := len(b.models) - 1; i >= 0; i-- {
			b.runtime.api.releaseSession(b.models[i].Session)
			b.models[i].Session = 0
		}
		b.runtime.close()
	}
	if b.onClose != nil {
		b.onClose()
	}
}

func (b *ortModelBundle) runDetection(input imagePreprocessResult) (ortInferenceSummary, error) {
	if b == nil || b.runtime == nil {
		return ortInferenceSummary{}, errors.New("model bundle is not initialized")
	}
	model := b.findModel("det")
	if model == nil {
		return ortInferenceSummary{}, errors.New("det model is not loaded")
	}
	output, err := b.runtime.api.runFloat32Session(model.Session, firstTensorName(model.Inputs), firstTensorName(model.Outputs), input.Tensor, input.Summary.Shape)
	if err != nil {
		return ortInferenceSummary{}, err
	}
	summary := summarizeORTOutput("det", output)
	candidates, err := detectCandidateBoxes(output, input.Summary)
	if err != nil {
		return ortInferenceSummary{}, err
	}
	summary.CandidateCount = len(candidates)
	summary.Candidates = candidates
	return summary, nil
}

func runONNXDetectionWithPurego(bundle *ortModelBundle, input imagePreprocessResult) (ortInferenceSummary, error) {
	if bundle == nil {
		return ortInferenceSummary{}, errors.New("model bundle is not initialized")
	}
	return bundle.runDetection(input)
}

func (b *ortModelBundle) findModel(kind string) *ortModelSession {
	if b == nil {
		return nil
	}
	for i := range b.models {
		if b.models[i].Kind == kind {
			return &b.models[i]
		}
	}
	return nil
}

func firstTensorName(values []ortTensorMetadata) string {
	if len(values) == 0 {
		return ""
	}
	return values[0].Name
}

func summarizeORTOutput(kind string, output ortTensorOutput) ortInferenceSummary {
	summary := ortInferenceSummary{
		Kind:       kind,
		OutputName: output.Name,
		Shape:      append([]int64(nil), output.Shape...),
	}
	if len(output.Data) == 0 {
		return summary
	}
	summary.Min = output.Data[0]
	summary.Max = output.Data[0]
	var total float64
	for _, value := range output.Data {
		if value < summary.Min {
			summary.Min = value
		}
		if value > summary.Max {
			summary.Max = value
		}
		total += float64(value)
	}
	summary.Mean = float32(total / float64(len(output.Data)))
	sampleCount := len(output.Data)
	if sampleCount > 8 {
		sampleCount = 8
	}
	summary.Sample = append([]float32(nil), output.Data[:sampleCount]...)
	return summary
}

type ortSessionMetadata struct {
	inputs  []ortTensorMetadata
	outputs []ortTensorMetadata
}

type ortAPI struct {
	ptr uintptr
}

func (a ortAPI) function(index int) (uintptr, error) {
	if a.ptr == 0 {
		return 0, errors.New("OrtApi pointer is null")
	}
	fn := *(*uintptr)(unsafe.Pointer(a.ptr + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	if fn == 0 {
		return 0, fmt.Errorf("OrtApi function pointer at index %d is null", index)
	}
	return fn, nil
}

func (a ortAPI) createEnv() (uintptr, error) {
	fn, err := a.function(ortAPICreateEnvIndex)
	if err != nil {
		return 0, err
	}
	var createEnv func(int32, string, *uintptr) uintptr
	purego.RegisterFunc(&createEnv, fn)
	var env uintptr
	status := createEnv(2, "recordingfreedom-ocr-worker", &env)
	if status != 0 {
		return 0, a.statusError("CreateEnv", status)
	}
	if env == 0 {
		return 0, errors.New("CreateEnv returned null env")
	}
	return env, nil
}

func (a ortAPI) createSessionOptions() (uintptr, error) {
	fn, err := a.function(ortAPICreateSessionOptionsIndex)
	if err != nil {
		return 0, err
	}
	var createSessionOptions func(*uintptr) uintptr
	purego.RegisterFunc(&createSessionOptions, fn)
	var options uintptr
	status := createSessionOptions(&options)
	if status != 0 {
		return 0, a.statusError("CreateSessionOptions", status)
	}
	if options == 0 {
		return 0, errors.New("CreateSessionOptions returned null options")
	}
	return options, nil
}

func (a ortAPI) loadModelSessionFromFile(env uintptr, options uintptr, modelPath string) (uintptr, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return 0, fmt.Errorf("read model: %w", err)
	}
	if len(data) == 0 {
		return 0, errors.New("model file is empty")
	}
	fn, err := a.function(ortAPICreateSessionFromArrayIndex)
	if err != nil {
		return 0, err
	}
	var createSessionFromArray func(uintptr, *byte, uintptr, uintptr, *uintptr) uintptr
	purego.RegisterFunc(&createSessionFromArray, fn)
	var session uintptr
	status := createSessionFromArray(env, &data[0], uintptr(len(data)), options, &session)
	goruntime.KeepAlive(data)
	if status != 0 {
		return 0, a.statusError("CreateSessionFromArray", status)
	}
	if session == 0 {
		return 0, errors.New("CreateSessionFromArray returned null session")
	}
	return session, nil
}

func (a ortAPI) describeSession(session uintptr) (ortSessionMetadata, error) {
	allocator, err := a.defaultAllocator()
	if err != nil {
		return ortSessionMetadata{}, err
	}
	inputCount, err := a.sessionCount(session, ortAPISessionGetInputCountIndex, "SessionGetInputCount")
	if err != nil {
		return ortSessionMetadata{}, err
	}
	outputCount, err := a.sessionCount(session, ortAPISessionGetOutputCountIndex, "SessionGetOutputCount")
	if err != nil {
		return ortSessionMetadata{}, err
	}
	inputs := make([]ortTensorMetadata, 0, inputCount)
	for i := uintptr(0); i < inputCount; i++ {
		info, err := a.sessionTensorMetadata(session, allocator, i, true)
		if err != nil {
			return ortSessionMetadata{}, fmt.Errorf("input %d: %w", i, err)
		}
		inputs = append(inputs, info)
	}
	outputs := make([]ortTensorMetadata, 0, outputCount)
	for i := uintptr(0); i < outputCount; i++ {
		info, err := a.sessionTensorMetadata(session, allocator, i, false)
		if err != nil {
			return ortSessionMetadata{}, fmt.Errorf("output %d: %w", i, err)
		}
		outputs = append(outputs, info)
	}
	if len(inputs) == 0 {
		return ortSessionMetadata{}, errors.New("session has no inputs")
	}
	if len(outputs) == 0 {
		return ortSessionMetadata{}, errors.New("session has no outputs")
	}
	return ortSessionMetadata{inputs: inputs, outputs: outputs}, nil
}

func (a ortAPI) runFloat32Session(session uintptr, inputName string, outputName string, inputData []float32, inputShape []int64) (ortTensorOutput, error) {
	if session == 0 {
		return ortTensorOutput{}, errors.New("session is null")
	}
	if inputName == "" {
		return ortTensorOutput{}, errors.New("input name is empty")
	}
	if outputName == "" {
		return ortTensorOutput{}, errors.New("output name is empty")
	}
	if len(inputData) == 0 {
		return ortTensorOutput{}, errors.New("input tensor is empty")
	}
	if len(inputShape) == 0 {
		return ortTensorOutput{}, errors.New("input tensor shape is empty")
	}
	inputValue, err := a.createFloat32Tensor(inputData, inputShape)
	if err != nil {
		return ortTensorOutput{}, err
	}
	defer a.releaseValue(inputValue)

	inputNameBytes := append([]byte(inputName), 0)
	outputNameBytes := append([]byte(outputName), 0)
	inputNames := []uintptr{uintptr(unsafe.Pointer(&inputNameBytes[0]))}
	outputNames := []uintptr{uintptr(unsafe.Pointer(&outputNameBytes[0]))}
	inputValues := []uintptr{inputValue}
	outputValues := []uintptr{0}

	fn, err := a.function(ortAPIRunIndex)
	if err != nil {
		return ortTensorOutput{}, err
	}
	var run func(uintptr, uintptr, *uintptr, *uintptr, uintptr, *uintptr, uintptr, *uintptr) uintptr
	purego.RegisterFunc(&run, fn)
	status := run(
		session,
		0,
		&inputNames[0],
		&inputValues[0],
		uintptr(len(inputValues)),
		&outputNames[0],
		uintptr(len(outputNames)),
		&outputValues[0],
	)
	goruntime.KeepAlive(inputData)
	goruntime.KeepAlive(inputShape)
	goruntime.KeepAlive(inputNameBytes)
	goruntime.KeepAlive(outputNameBytes)
	goruntime.KeepAlive(inputNames)
	goruntime.KeepAlive(outputNames)
	goruntime.KeepAlive(inputValues)
	if status != 0 {
		return ortTensorOutput{}, a.statusError("Run", status)
	}
	if outputValues[0] == 0 {
		return ortTensorOutput{}, errors.New("Run returned null output")
	}
	defer a.releaseValue(outputValues[0])

	shape, elementCount, err := a.valueShape(outputValues[0])
	if err != nil {
		return ortTensorOutput{}, err
	}
	if elementCount == 0 {
		return ortTensorOutput{}, errors.New("Run returned empty output tensor")
	}
	data, err := a.float32ValueData(outputValues[0], elementCount)
	if err != nil {
		return ortTensorOutput{}, err
	}
	return ortTensorOutput{Name: outputName, Shape: shape, Data: data}, nil
}

func (a ortAPI) createFloat32Tensor(data []float32, shape []int64) (uintptr, error) {
	memoryInfo, err := a.createCPUMemoryInfo()
	if err != nil {
		return 0, err
	}
	defer a.releaseMemoryInfo(memoryInfo)
	fn, err := a.function(ortAPICreateTensorWithDataAsOrtValueIndex)
	if err != nil {
		return 0, err
	}
	var createTensorWithData func(uintptr, *float32, uintptr, *int64, uintptr, int32, *uintptr) uintptr
	purego.RegisterFunc(&createTensorWithData, fn)
	var value uintptr
	status := createTensorWithData(
		memoryInfo,
		&data[0],
		uintptr(len(data)*4),
		&shape[0],
		uintptr(len(shape)),
		onnxTensorElementDataTypeFloat,
		&value,
	)
	goruntime.KeepAlive(data)
	goruntime.KeepAlive(shape)
	if status != 0 {
		return 0, a.statusError("CreateTensorWithDataAsOrtValue", status)
	}
	if value == 0 {
		return 0, errors.New("CreateTensorWithDataAsOrtValue returned null value")
	}
	return value, nil
}

func (a ortAPI) createCPUMemoryInfo() (uintptr, error) {
	fn, err := a.function(ortAPICreateCpuMemoryInfoIndex)
	if err != nil {
		return 0, err
	}
	var createCPUMemoryInfo func(int32, int32, *uintptr) uintptr
	purego.RegisterFunc(&createCPUMemoryInfo, fn)
	var memoryInfo uintptr
	status := createCPUMemoryInfo(ortArenaAllocator, ortMemTypeDefault, &memoryInfo)
	if status != 0 {
		return 0, a.statusError("CreateCpuMemoryInfo", status)
	}
	if memoryInfo == 0 {
		return 0, errors.New("CreateCpuMemoryInfo returned null memory info")
	}
	return memoryInfo, nil
}

func (a ortAPI) valueShape(value uintptr) ([]int64, uintptr, error) {
	tensorInfo, err := a.valueTensorInfo(value)
	if err != nil {
		return nil, 0, err
	}
	defer a.releaseTensorTypeAndShapeInfo(tensorInfo)
	elementType, err := a.tensorElementType(tensorInfo)
	if err != nil {
		return nil, 0, err
	}
	if elementType != onnxTensorElementDataTypeFloat {
		return nil, 0, fmt.Errorf("output tensor type = %s, want float32", onnxTensorElementTypeName(elementType))
	}
	shape, err := a.tensorDimensions(tensorInfo)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.tensorElementCount(tensorInfo)
	if err != nil {
		return nil, 0, err
	}
	return shape, count, nil
}

func (a ortAPI) valueTensorInfo(value uintptr) (uintptr, error) {
	fn, err := a.function(ortAPIGetTensorTypeAndShapeIndex)
	if err != nil {
		return 0, err
	}
	var getTensorTypeAndShape func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getTensorTypeAndShape, fn)
	var tensorInfo uintptr
	status := getTensorTypeAndShape(value, &tensorInfo)
	if status != 0 {
		return 0, a.statusError("GetTensorTypeAndShape", status)
	}
	if tensorInfo == 0 {
		return 0, errors.New("GetTensorTypeAndShape returned null tensor info")
	}
	return tensorInfo, nil
}

func (a ortAPI) tensorElementCount(tensorInfo uintptr) (uintptr, error) {
	fn, err := a.function(ortAPIGetTensorShapeElementCountIndex)
	if err != nil {
		return 0, err
	}
	var getTensorShapeElementCount func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getTensorShapeElementCount, fn)
	var count uintptr
	status := getTensorShapeElementCount(tensorInfo, &count)
	if status != 0 {
		return 0, a.statusError("GetTensorShapeElementCount", status)
	}
	return count, nil
}

func (a ortAPI) float32ValueData(value uintptr, elementCount uintptr) ([]float32, error) {
	fn, err := a.function(ortAPIGetTensorMutableDataIndex)
	if err != nil {
		return nil, err
	}
	var getTensorMutableData func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getTensorMutableData, fn)
	var dataPtr uintptr
	status := getTensorMutableData(value, &dataPtr)
	if status != 0 {
		return nil, a.statusError("GetTensorMutableData", status)
	}
	if dataPtr == 0 {
		return nil, errors.New("GetTensorMutableData returned null data")
	}
	if elementCount > uintptr(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("output tensor element count is too large: %d", elementCount)
	}
	values := unsafe.Slice((*float32)(unsafe.Pointer(dataPtr)), int(elementCount))
	out := make([]float32, len(values))
	copy(out, values)
	return out, nil
}

func (a ortAPI) defaultAllocator() (uintptr, error) {
	fn, err := a.function(ortAPIGetAllocatorWithDefaultOptionsIndex)
	if err != nil {
		return 0, err
	}
	var getAllocatorWithDefaultOptions func(*uintptr) uintptr
	purego.RegisterFunc(&getAllocatorWithDefaultOptions, fn)
	var allocator uintptr
	status := getAllocatorWithDefaultOptions(&allocator)
	if status != 0 {
		return 0, a.statusError("GetAllocatorWithDefaultOptions", status)
	}
	if allocator == 0 {
		return 0, errors.New("GetAllocatorWithDefaultOptions returned null allocator")
	}
	return allocator, nil
}

func (a ortAPI) sessionCount(session uintptr, functionIndex int, operation string) (uintptr, error) {
	fn, err := a.function(functionIndex)
	if err != nil {
		return 0, err
	}
	var count func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&count, fn)
	var out uintptr
	status := count(session, &out)
	if status != 0 {
		return 0, a.statusError(operation, status)
	}
	return out, nil
}

func (a ortAPI) sessionTensorMetadata(session uintptr, allocator uintptr, index uintptr, input bool) (ortTensorMetadata, error) {
	name, err := a.sessionName(session, allocator, index, input)
	if err != nil {
		return ortTensorMetadata{}, err
	}
	typeInfo, err := a.sessionTypeInfo(session, index, input)
	if err != nil {
		return ortTensorMetadata{}, err
	}
	defer a.releaseTypeInfo(typeInfo)

	tensorInfo, err := a.tensorInfo(typeInfo)
	if err != nil {
		return ortTensorMetadata{}, err
	}
	elementType, err := a.tensorElementType(tensorInfo)
	if err != nil {
		return ortTensorMetadata{}, err
	}
	dimensions, err := a.tensorDimensions(tensorInfo)
	if err != nil {
		return ortTensorMetadata{}, err
	}
	return ortTensorMetadata{
		Name:            name,
		ElementTypeCode: elementType,
		ElementType:     onnxTensorElementTypeName(elementType),
		Dimensions:      dimensions,
	}, nil
}

func (a ortAPI) sessionName(session uintptr, allocator uintptr, index uintptr, input bool) (string, error) {
	functionIndex := ortAPISessionGetOutputNameIndex
	operation := "SessionGetOutputName"
	if input {
		functionIndex = ortAPISessionGetInputNameIndex
		operation = "SessionGetInputName"
	}
	fn, err := a.function(functionIndex)
	if err != nil {
		return "", err
	}
	var getName func(uintptr, uintptr, uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getName, fn)
	var namePtr uintptr
	status := getName(session, index, allocator, &namePtr)
	if status != 0 {
		return "", a.statusError(operation, status)
	}
	defer a.allocatorFree(allocator, namePtr)
	if namePtr == 0 {
		return "", fmt.Errorf("%s returned null name", operation)
	}
	name := nullTerminatedString(namePtr)
	if name == "" {
		return "", fmt.Errorf("%s returned empty name", operation)
	}
	return name, nil
}

func (a ortAPI) sessionTypeInfo(session uintptr, index uintptr, input bool) (uintptr, error) {
	functionIndex := ortAPISessionGetOutputTypeInfoIndex
	operation := "SessionGetOutputTypeInfo"
	if input {
		functionIndex = ortAPISessionGetInputTypeInfoIndex
		operation = "SessionGetInputTypeInfo"
	}
	fn, err := a.function(functionIndex)
	if err != nil {
		return 0, err
	}
	var getTypeInfo func(uintptr, uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getTypeInfo, fn)
	var typeInfo uintptr
	status := getTypeInfo(session, index, &typeInfo)
	if status != 0 {
		return 0, a.statusError(operation, status)
	}
	if typeInfo == 0 {
		return 0, fmt.Errorf("%s returned null type info", operation)
	}
	return typeInfo, nil
}

func (a ortAPI) tensorInfo(typeInfo uintptr) (uintptr, error) {
	fn, err := a.function(ortAPICastTypeInfoToTensorInfoIndex)
	if err != nil {
		return 0, err
	}
	var castTypeInfoToTensorInfo func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&castTypeInfoToTensorInfo, fn)
	var tensorInfo uintptr
	status := castTypeInfoToTensorInfo(typeInfo, &tensorInfo)
	if status != 0 {
		return 0, a.statusError("CastTypeInfoToTensorInfo", status)
	}
	if tensorInfo == 0 {
		return 0, errors.New("type info is not a tensor")
	}
	return tensorInfo, nil
}

func (a ortAPI) tensorElementType(tensorInfo uintptr) (int32, error) {
	fn, err := a.function(ortAPIGetTensorElementTypeIndex)
	if err != nil {
		return 0, err
	}
	var getTensorElementType func(uintptr, *int32) uintptr
	purego.RegisterFunc(&getTensorElementType, fn)
	var elementType int32
	status := getTensorElementType(tensorInfo, &elementType)
	if status != 0 {
		return 0, a.statusError("GetTensorElementType", status)
	}
	return elementType, nil
}

func (a ortAPI) tensorDimensions(tensorInfo uintptr) ([]int64, error) {
	countFn, err := a.function(ortAPIGetDimensionsCountIndex)
	if err != nil {
		return nil, err
	}
	var getDimensionsCount func(uintptr, *uintptr) uintptr
	purego.RegisterFunc(&getDimensionsCount, countFn)
	var count uintptr
	status := getDimensionsCount(tensorInfo, &count)
	if status != 0 {
		return nil, a.statusError("GetDimensionsCount", status)
	}
	if count == 0 {
		return nil, nil
	}
	dimensions := make([]int64, count)
	dimensionsFn, err := a.function(ortAPIGetDimensionsIndex)
	if err != nil {
		return nil, err
	}
	var getDimensions func(uintptr, *int64, uintptr) uintptr
	purego.RegisterFunc(&getDimensions, dimensionsFn)
	status = getDimensions(tensorInfo, &dimensions[0], count)
	if status != 0 {
		return nil, a.statusError("GetDimensions", status)
	}
	return dimensions, nil
}

func (a ortAPI) allocatorFree(allocator uintptr, value uintptr) {
	if allocator == 0 || value == 0 {
		return
	}
	fn, err := a.function(ortAPIAllocatorFreeIndex)
	if err != nil {
		return
	}
	var allocatorFree func(uintptr, uintptr) uintptr
	purego.RegisterFunc(&allocatorFree, fn)
	status := allocatorFree(allocator, value)
	if status != 0 {
		a.releaseStatus(status)
	}
}

func (a ortAPI) releaseEnv(env uintptr) {
	a.releasePointer(ortAPIReleaseEnvIndex, env)
}

func (a ortAPI) releaseMemoryInfo(memoryInfo uintptr) {
	a.releasePointer(ortAPIReleaseMemoryInfoIndex, memoryInfo)
}

func (a ortAPI) releaseSession(session uintptr) {
	a.releasePointer(ortAPIReleaseSessionIndex, session)
}

func (a ortAPI) releaseValue(value uintptr) {
	a.releasePointer(ortAPIReleaseValueIndex, value)
}

func (a ortAPI) releaseSessionOptions(options uintptr) {
	a.releasePointer(ortAPIReleaseSessionOptionsIndex, options)
}

func (a ortAPI) releaseTypeInfo(typeInfo uintptr) {
	a.releasePointer(ortAPIReleaseTypeInfoIndex, typeInfo)
}

func (a ortAPI) releaseTensorTypeAndShapeInfo(tensorInfo uintptr) {
	a.releasePointer(ortAPIReleaseTensorTypeAndShapeInfoIndex, tensorInfo)
}

func (a ortAPI) releaseStatus(status uintptr) {
	a.releasePointer(ortAPIReleaseStatusIndex, status)
}

func (a ortAPI) releasePointer(index int, value uintptr) {
	if value == 0 {
		return
	}
	fn, err := a.function(index)
	if err != nil {
		return
	}
	var release func(uintptr)
	purego.RegisterFunc(&release, fn)
	release(value)
}

func (a ortAPI) statusError(operation string, status uintptr) error {
	defer a.releaseStatus(status)
	fn, err := a.function(ortAPIGetErrorMessageIndex)
	if err != nil {
		return fmt.Errorf("%s failed with unreadable OrtStatus: %w", operation, err)
	}
	var getErrorMessage func(uintptr) string
	purego.RegisterFunc(&getErrorMessage, fn)
	message := getErrorMessage(status)
	if message == "" {
		message = "unknown ONNX Runtime error"
	}
	return fmt.Errorf("%s failed: %s", operation, message)
}

func nullTerminatedString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var length uintptr
	for {
		if *(*byte)(unsafe.Pointer(ptr + length)) == 0 {
			break
		}
		length++
	}
	if length == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length))
}

func onnxTensorElementTypeName(code int32) string {
	switch code {
	case 1:
		return "float32"
	case 2:
		return "uint8"
	case 3:
		return "int8"
	case 4:
		return "uint16"
	case 5:
		return "int16"
	case 6:
		return "int32"
	case 7:
		return "int64"
	case 8:
		return "string"
	case 9:
		return "bool"
	case 10:
		return "float16"
	case 11:
		return "float64"
	case 12:
		return "uint32"
	case 13:
		return "uint64"
	case 14:
		return "complex64"
	case 15:
		return "complex128"
	case 16:
		return "bfloat16"
	default:
		return fmt.Sprintf("unknown(%d)", code)
	}
}
