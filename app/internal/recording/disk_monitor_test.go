package recording

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

func TestRunDiskMonitorStopsBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, recpackage.ManifestFile)
	packages := recpackage.NewService()
	if err := packages.WriteManifest(manifestPath, recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		Status:        recpackage.StatusRecording,
		RecordingMode: recpackage.RecordingModeScreen,
	}); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	var stops atomic.Int32
	monitor := &diskMonitor{stop: make(chan struct{}), done: make(chan struct{})}
	go runDiskMonitor(monitor, diskMonitorOptions{
		packageDir:   dir,
		manifestPath: manifestPath,
		threshold:    1000,
		interval:     time.Millisecond,
		freeBytes: func(string) (uint64, error) {
			return 999, nil
		},
		packages: packages,
		onStop: func() {
			stops.Add(1)
		},
	})

	<-monitor.done
	deadline := time.Now().Add(2 * time.Second)
	for stops.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if stops.Load() != 1 {
		t.Fatalf("stop calls = %d, want 1", stops.Load())
	}
	manifest, err := packages.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Diagnostics.Message == "" {
		t.Fatal("low-disk annotation was not written")
	}
}

func TestRunDiskMonitorKeepsRunningAboveThreshold(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, recpackage.ManifestFile)
	packages := recpackage.NewService()
	if err := packages.WriteManifest(manifestPath, recpackage.Manifest{
		SchemaVersion: 1,
		App:           recpackage.AppName,
		Status:        recpackage.StatusRecording,
		RecordingMode: recpackage.RecordingModeScreen,
	}); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	var stops atomic.Int32
	monitor := &diskMonitor{stop: make(chan struct{}), done: make(chan struct{})}
	go runDiskMonitor(monitor, diskMonitorOptions{
		packageDir:   dir,
		manifestPath: manifestPath,
		threshold:    1000,
		interval:     time.Millisecond,
		freeBytes: func(string) (uint64, error) {
			return 5000, nil
		},
		packages: packages,
		onStop: func() {
			stops.Add(1)
		},
	})

	time.Sleep(30 * time.Millisecond)
	monitor.close()
	if stops.Load() != 0 {
		t.Fatalf("stop calls = %d, want 0 above threshold", stops.Load())
	}
}

func TestRunDiskMonitorIgnoresCheckerErrors(t *testing.T) {
	monitor := &diskMonitor{stop: make(chan struct{}), done: make(chan struct{})}
	var stops atomic.Int32
	go runDiskMonitor(monitor, diskMonitorOptions{
		packageDir:   t.TempDir(),
		manifestPath: filepath.Join(t.TempDir(), recpackage.ManifestFile),
		threshold:    1000,
		interval:     time.Millisecond,
		freeBytes: func(string) (uint64, error) {
			return 0, os.ErrPermission
		},
		onStop: func() {
			stops.Add(1)
		},
	})

	time.Sleep(20 * time.Millisecond)
	monitor.close()
	if stops.Load() != 0 {
		t.Fatalf("stop calls = %d, want 0 when the checker errors", stops.Load())
	}
}

func TestMinFreeDiskBytes(t *testing.T) {
	t.Setenv(EnvMinFreeDiskMB, "")
	if got := minFreeDiskBytes(); got != appdata.MinimumRecommendedVideoFreeBytes {
		t.Fatalf("default threshold = %d, want %d", got, appdata.MinimumRecommendedVideoFreeBytes)
	}
	t.Setenv(EnvMinFreeDiskMB, "256")
	if got := minFreeDiskBytes(); got != 256*1024*1024 {
		t.Fatalf("threshold = %d, want 256 MiB", got)
	}
	t.Setenv(EnvMinFreeDiskMB, "0")
	if got := minFreeDiskBytes(); got != 0 {
		t.Fatalf("threshold = %d, want 0 to disable", got)
	}
	t.Setenv(EnvMinFreeDiskMB, "not-a-number")
	if got := minFreeDiskBytes(); got != 0 {
		t.Fatalf("threshold = %d, want 0 for invalid input", got)
	}
}
