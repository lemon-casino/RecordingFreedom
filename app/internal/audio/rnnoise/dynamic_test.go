//go:build rnnoise_dynamic

package rnnoise

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDynamicLibraryCandidatesIncludeModuleToolsFromPackageDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	moduleRoot, err := findModuleRoot(wd)
	if err != nil {
		t.Fatalf("findModuleRoot(%q) error = %v", wd, err)
	}
	want := filepath.Join(moduleRoot, "tools", dynamicLibraryName())
	for _, candidate := range dynamicLibraryCandidates() {
		if samePath(candidate, want) {
			return
		}
	}
	t.Fatalf("dynamic library candidates do not include module tools path %q: %v", want, dynamicLibraryCandidates())
}

func TestDynamicSuppressorProcessesOneFrame(t *testing.T) {
	if !Available() {
		t.Fatal("dynamic build reported RNNoise as unavailable")
	}
	suppressor, err := New(1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer suppressor.Close()

	frame := make([]float32, FrameSize())
	for index := range frame {
		frame[index] = 0.01
	}
	if err := suppressor.ProcessFrame(frame); err != nil {
		t.Fatalf("ProcessFrame() error = %v", err)
	}
	if err := suppressor.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
}

func TestDynamicSuppressorReducesSteadyFanLikeNoise(t *testing.T) {
	if !Available() {
		t.Fatal("dynamic build reported RNNoise as unavailable")
	}

	const frameCount = 300
	input := syntheticFanNoise(frameCount * FrameSize())
	output := processSyntheticSignal(t, input)

	discard := 100 * FrameSize()
	inputRMS := signalRMS(input[discard:])
	outputRMS := signalRMS(output[discard:])
	t.Logf("steady fan-like noise RMS: input=%.6f output=%.6f ratio=%.3f", inputRMS, outputRMS, outputRMS/inputRMS)
	if outputRMS >= inputRMS*0.55 {
		t.Fatalf("steady fan-like noise RMS ratio = %.3f, want < 0.55", outputRMS/inputRMS)
	}
}

func TestDynamicSuppressorKeepsVoiceProminentOverSteadyFanNoise(t *testing.T) {
	if !Available() {
		t.Fatal("dynamic build reported RNNoise as unavailable")
	}

	const frameCount = 300
	sampleCount := frameCount * FrameSize()
	fan := syntheticFanNoise(sampleCount)
	voice := syntheticVoicedSpeech(sampleCount)
	mixed := make([]float32, sampleCount)
	for index := range mixed {
		mixed[index] = fan[index] + voice[index]
	}

	fanOutput := processSyntheticSignal(t, fan)
	mixedOutput := processSyntheticSignal(t, mixed)
	discard := 100 * FrameSize()
	fanOutputRMS := signalRMS(fanOutput[discard:])
	voiceInputRMS := signalRMS(voice[discard:])
	mixedOutputRMS := signalRMS(mixedOutput[discard:])
	t.Logf(
		"voice-over-fan RMS: voice-input=%.6f fan-output=%.6f mixed-output=%.6f prominence=%.3f retention=%.3f",
		voiceInputRMS,
		fanOutputRMS,
		mixedOutputRMS,
		mixedOutputRMS/fanOutputRMS,
		mixedOutputRMS/voiceInputRMS,
	)
	if mixedOutputRMS <= fanOutputRMS*3.0 {
		t.Fatalf("processed voice prominence = %.3f, want > 3.0", mixedOutputRMS/fanOutputRMS)
	}
	if mixedOutputRMS <= voiceInputRMS*0.35 {
		t.Fatalf("processed voice retention = %.3f, want > 0.35", mixedOutputRMS/voiceInputRMS)
	}
}

func TestDynamicSuppressorRetainsCleanVoiceLevel(t *testing.T) {
	if !Available() {
		t.Fatal("dynamic build reported RNNoise as unavailable")
	}

	const frameCount = 300
	voice := syntheticVoicedSpeech(frameCount * FrameSize())
	output := processSyntheticSignal(t, voice)
	discard := 100 * FrameSize()
	inputRMS := signalRMS(voice[discard:])
	outputRMS := signalRMS(output[discard:])
	retention := outputRMS / inputRMS
	t.Logf("clean voice RMS: input=%.6f output=%.6f retention=%.3f", inputRMS, outputRMS, retention)
	if retention <= 0.50 {
		t.Fatalf("clean voice retention = %.3f, want > 0.50", retention)
	}
	if signalPeak(output[discard:]) >= 1.0 {
		t.Fatal("processed clean voice clipped")
	}
}

func processSyntheticSignal(t *testing.T, input []float32) []float32 {
	t.Helper()
	suppressor, err := New(1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer suppressor.Close()

	output := append([]float32(nil), input...)
	for offset := 0; offset < len(output); offset += FrameSize() {
		if err := suppressor.ProcessFrame(output[offset : offset+FrameSize()]); err != nil {
			t.Fatalf("ProcessFrame(%d) error = %v", offset/FrameSize(), err)
		}
	}
	return output
}

func syntheticFanNoise(sampleCount int) []float32 {
	samples := make([]float32, sampleCount)
	var randomState uint32 = 0x12345678
	for index := range samples {
		seconds := float64(index) / 48000
		randomState = randomState*1664525 + 1013904223
		broadband := (float64(randomState>>8)/float64(1<<24) - 0.5) * 0.018
		samples[index] = float32(
			0.060*math.Sin(2*math.Pi*92*seconds) +
				0.034*math.Sin(2*math.Pi*184*seconds) +
				0.020*math.Sin(2*math.Pi*368*seconds) +
				0.012*math.Sin(2*math.Pi*736*seconds) +
				broadband,
		)
	}
	return samples
}

func syntheticVoicedSpeech(sampleCount int) []float32 {
	samples := make([]float32, sampleCount)
	var phase float64
	for index := range samples {
		seconds := float64(index) / 48000
		fundamental := 138.0 + 16.0*math.Sin(2*math.Pi*2.1*seconds)
		phase += 2 * math.Pi * fundamental / 48000
		envelope := 0.38 + 0.62*(0.5+0.5*math.Sin(2*math.Pi*3.4*seconds))
		samples[index] = float32(envelope * (0.060*math.Sin(phase) +
			0.042*math.Sin(2*phase) +
			0.030*math.Sin(3*phase) +
			0.020*math.Sin(5*phase) +
			0.014*math.Sin(8*phase)))
	}
	return samples
}

func signalRMS(samples []float32) float64 {
	var energy float64
	for _, sample := range samples {
		energy += float64(sample) * float64(sample)
	}
	if len(samples) == 0 {
		return 0
	}
	return math.Sqrt(energy / float64(len(samples)))
}

func signalPeak(samples []float32) float64 {
	var peak float64
	for _, sample := range samples {
		magnitude := math.Abs(float64(sample))
		if magnitude > peak {
			peak = magnitude
		}
	}
	return peak
}

func findModuleRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func samePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
