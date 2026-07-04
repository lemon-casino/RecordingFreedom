//go:build rnnoise_dynamic

package rnnoise

import (
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
