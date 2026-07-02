package appdata

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestVideoDirUsesDataVideoUnderRoot(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)

	videoDir, err := service.VideoDir()
	if err != nil {
		t.Fatalf("VideoDir() error = %v", err)
	}

	want := filepath.Join(root, "data", "video")
	if videoDir != want {
		t.Fatalf("VideoDir() = %q, want %q", videoDir, want)
	}
	if info, err := os.Stat(videoDir); err != nil || !info.IsDir() {
		t.Fatalf("video dir was not created: info=%v err=%v", info, err)
	}
}

func TestEnvRootTakesPrecedence(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvDataDir, root)

	service := NewService("")
	info, err := service.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	wantVideo := filepath.Join(root, "data", "video")
	if info.VideoDir != wantVideo {
		t.Fatalf("Info().VideoDir = %q, want %q", info.VideoDir, wantVideo)
	}
}

func TestDefaultRootDirUsesExecutableDirectory(t *testing.T) {
	exeDir := filepath.Join(t.TempDir(), "portable")
	fakeExe := filepath.Join(exeDir, "recordingfreedom.exe")
	originalExecutablePath := executablePath
	executablePath = func() (string, error) {
		return fakeExe, nil
	}
	t.Cleanup(func() {
		executablePath = originalExecutablePath
	})

	if got := defaultRootDir(); got != exeDir {
		t.Fatalf("defaultRootDir() = %q, want executable dir %q", got, exeDir)
	}
}

func TestDefaultRootDirFallsBackToWorkingDirectory(t *testing.T) {
	workingDir := t.TempDir()
	originalExecutablePath := executablePath
	executablePath = func() (string, error) {
		return "", errors.New("no executable")
	}
	t.Cleanup(func() {
		executablePath = originalExecutablePath
	})
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	if got := defaultRootDir(); got != workingDir {
		t.Fatalf("defaultRootDir() = %q, want working dir %q", got, workingDir)
	}
}

func TestStorageStatusProbesManagedVideoDir(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)

	status, err := service.StorageStatus()
	if err != nil {
		t.Fatalf("StorageStatus() error = %v", err)
	}
	if status.RootDir != root {
		t.Fatalf("root dir = %q, want %q", status.RootDir, root)
	}
	if status.VideoDir != filepath.Join(root, "data", "video") {
		t.Fatalf("video dir = %q, want data/video under root", status.VideoDir)
	}
	if !status.Writable {
		t.Fatalf("storage should be writable: %#v", status)
	}
	if status.Status != StorageStatusReady && status.Status != StorageStatusWarning {
		t.Fatalf("storage status = %q, want ready or warning: %#v", status.Status, status)
	}
	if status.MinimumRecommendedBytes != MinimumRecommendedVideoFreeBytes {
		t.Fatalf("minimum recommended bytes = %d, want %d", status.MinimumRecommendedBytes, MinimumRecommendedVideoFreeBytes)
	}
	if status.FreeSpaceKnown && status.AvailableBytes == 0 {
		t.Fatalf("free space is known but zero: %#v", status)
	}
}

func TestStorageStatusConcurrentProbesDoNotCollide(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)

	const workers = 16
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := service.StorageStatus()
			if err != nil {
				errs <- err
				return
			}
			if !status.Writable || status.Status == StorageStatusBlocked {
				errs <- os.ErrPermission
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent StorageStatus() probe failed: %v", err)
		}
	}
}

func TestSetRootDirPersistsPointerAndKeepsDataVideo(t *testing.T) {
	pointerBase := t.TempDir()
	customRoot := filepath.Join(t.TempDir(), "recording-root")
	service := NewServiceWithPointerBase("", pointerBase)

	info, err := service.SetRootDir(customRoot)
	if err != nil {
		t.Fatalf("SetRootDir() error = %v", err)
	}
	wantRoot, err := filepath.Abs(customRoot)
	if err != nil {
		t.Fatalf("Abs(customRoot) error = %v", err)
	}
	if info.RootDir != wantRoot {
		t.Fatalf("root = %q, want %q", info.RootDir, wantRoot)
	}
	if info.VideoDir != filepath.Join(wantRoot, "data", "video") {
		t.Fatalf("video dir = %q, want data/video under custom root", info.VideoDir)
	}
	if _, err := os.Stat(filepath.Join(pointerBase, rootPointerFile)); err != nil {
		t.Fatalf("data root pointer was not written: %v", err)
	}

	restarted := NewServiceWithPointerBase("", pointerBase)
	restartedInfo, err := restarted.Info()
	if err != nil {
		t.Fatalf("restarted Info() error = %v", err)
	}
	if restartedInfo.RootDir != wantRoot {
		t.Fatalf("restarted root = %q, want persisted pointer root %q", restartedInfo.RootDir, wantRoot)
	}
}

func TestSetRootDirRejectsEnvOverride(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	service := NewServiceWithPointerBase("", t.TempDir())

	if _, err := service.SetRootDir(t.TempDir()); err == nil {
		t.Fatalf("SetRootDir() accepted data root change while %s is set", EnvDataDir)
	}
}
