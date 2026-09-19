package recording

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recordingprofile"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

type recoveryFixture struct {
	root        string
	packages    *recpackage.Service
	packageDir  string
	manifest    recpackage.Manifest
	manifestStr string
	screenPath  string
	segmentDir  string
}

func newRecoveryFixture(t *testing.T, withSidecars bool) *recoveryFixture {
	t.Helper()
	root := t.TempDir()
	videoDir := filepath.Join(root, "data", "video")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatalf("create video dir: %v", err)
	}
	packages := recpackage.NewService()
	plan, err := packages.CreateNative(videoDir, recpackage.CreateNativeRequest{
		CreatedAt: time.Now(),
		Backend:   "ffmpeg-desktop-capture",
		Source:    recpackage.ManifestSource{Type: "screen", ID: "screen:primary", Name: "Primary Display"},
		Recording: recordingprofile.Profile{Quality: recordingprofile.QualityStandard, FPS: 30, CaptureCursor: true},
		Audio: recpackage.ManifestAudio{
			System:                     withSidecars,
			Microphone:                 withSidecars,
			MicrophoneDeviceID:         "microphone:default",
			MicrophoneNoiseSuppression: recpackage.NoiseSuppressionOn,
		},
		Camera: recpackage.ManifestCamera{Enabled: false},
	})
	if err != nil {
		t.Fatalf("CreateNative() error = %v", err)
	}
	manifest, err := packages.ReadManifest(plan.Package.ManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if withSidecars {
		manifest.Media.SystemAudioStorage = recpackage.AudioStorageSidecar
		manifest.Media.SystemAudioPath = "audio/system.wav"
		manifest.Media.MicrophoneAudioStorage = recpackage.AudioStorageSidecar
		manifest.Media.MicrophoneAudioPath = "audio/microphone.wav"
		if err := packages.WriteManifest(plan.Package.ManifestPath, manifest); err != nil {
			t.Fatalf("WriteManifest(sidecar layout) error = %v", err)
		}
		audioDir := filepath.Join(plan.Package.Dir, "audio")
		if err := os.MkdirAll(audioDir, 0o755); err != nil {
			t.Fatalf("create audio dir: %v", err)
		}
		for _, name := range []string{"system.wav", "microphone.wav"} {
			if err := os.WriteFile(filepath.Join(audioDir, name), []byte(strings.Repeat("w", 64)), 0o644); err != nil {
				t.Fatalf("write sidecar: %v", err)
			}
		}
	}
	fixture := &recoveryFixture{
		root:        root,
		packages:    packages,
		packageDir:  plan.Package.Dir,
		manifest:    manifest,
		manifestStr: plan.Package.ManifestPath,
		screenPath:  filepath.Join(plan.Package.Dir, recpackage.ScreenVideoFile),
		segmentDir:  video.SegmentRecoveryDir(plan.Package.Dir, plan.ScreenVideoPath),
	}
	if err := os.MkdirAll(fixture.segmentDir, 0o755); err != nil {
		t.Fatalf("create segment dir: %v", err)
	}
	return fixture
}

func (f *recoveryFixture) service(hooks recoveryHooks) *Service {
	service := NewServiceWithBackend(appdata.NewService(f.root), NewMockBackend(recpackage.NewService()))
	service.recoveryHooks = hooks
	return service
}

func writeSegments(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Repeat(name, 16)), 0o644); err != nil {
			t.Fatalf("write segment %s: %v", name, err)
		}
	}
}

func TestRecoverPackageRebuildsSegmentsAndMuxesAudio(t *testing.T) {
	fixture := newRecoveryFixture(t, true)
	writeSegments(t, fixture.segmentDir, "segment-000-000.mp4", "segment-000-001.mp4")

	muxedWith := ""
	hooks := recoveryHooks{
		rebuildScreenVideo: func(ffmpegPath string, outputPath string, segmentDir string) (int, error) {
			if segmentDir != fixture.segmentDir {
				t.Errorf("rebuild segmentDir = %q, want %q", segmentDir, fixture.segmentDir)
			}
			if err := os.WriteFile(outputPath, []byte("recovered-video"), 0o644); err != nil {
				return 0, err
			}
			return 2, nil
		},
		muxScreenAudio: func(config video.AudioMuxConfig) (video.AudioMuxResult, error) {
			muxedWith = config.VideoPath
			return video.AudioMuxResult{}, nil
		},
	}
	service := fixture.service(hooks)

	summary, err := service.RecoverPackage(fixture.packageDir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("summary status = %q, want ready", summary.Status)
	}
	if muxedWith != fixture.screenPath {
		t.Fatalf("mux video path = %q, want %q", muxedWith, fixture.screenPath)
	}
	if !strings.Contains(summary.Reason, "rebuilt screen.mp4 from 2 cached FFmpeg segment(s)") {
		t.Fatalf("summary reason = %q, want rebuild note", summary.Reason)
	}
	if !strings.Contains(summary.Reason, "muxed recovered audio") {
		t.Fatalf("summary reason = %q, want mux note", summary.Reason)
	}
	manifest, err := fixture.packages.ReadManifest(fixture.manifestStr)
	if err != nil {
		t.Fatalf("ReadManifest() after recovery error = %v", err)
	}
	if manifest.Media.SystemAudioStorage != recpackage.AudioStorageMuxed {
		t.Fatalf("system storage = %q, want muxed", manifest.Media.SystemAudioStorage)
	}
	if manifest.Media.SystemAudioPath != recpackage.ScreenVideoFile {
		t.Fatalf("system audio path = %q, want %q", manifest.Media.SystemAudioPath, recpackage.ScreenVideoFile)
	}
	if manifest.Diagnostics.Recovered != true {
		t.Fatal("manifest was not marked recovered")
	}
}

func TestRecoverPackageWithoutSegmentsReportsFinalizeFailure(t *testing.T) {
	fixture := newRecoveryFixture(t, true)
	hooks := recoveryHooks{
		rebuildScreenVideo: func(string, string, string) (int, error) {
			return 0, os.ErrNotExist
		},
	}
	service := fixture.service(hooks)

	_, err := service.RecoverPackage(fixture.packageDir)
	if err == nil {
		t.Fatal("RecoverPackage() succeeded without screen media or segments")
	}
	if !strings.Contains(err.Error(), "recovery finalize") {
		t.Fatalf("error = %v, want finalize context", err)
	}
}

func TestRecoverPackageLeavesCompleteScreenTrackUntouched(t *testing.T) {
	fixture := newRecoveryFixture(t, false)
	probeable := minimalMP4Data("vide")
	if err := os.WriteFile(fixture.screenPath, probeable, 0o644); err != nil {
		t.Fatalf("write screen: %v", err)
	}
	hooks := recoveryHooks{
		rebuildScreenVideo: func(string, string, string) (int, error) {
			t.Fatal("rebuild must not run when the screen track already exists")
			return 0, nil
		},
	}
	service := fixture.service(hooks)

	summary, err := service.RecoverPackage(fixture.packageDir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("summary status = %q, want ready", summary.Status)
	}
	data, err := os.ReadFile(fixture.screenPath)
	if err != nil || !bytes.Equal(data, probeable) {
		t.Fatalf("screen track was modified: %q (%v)", data, err)
	}
}

func TestRecoverPackageRebuildsTruncatedScreenTrack(t *testing.T) {
	fixture := newRecoveryFixture(t, false)
	probeable := minimalMP4Data("vide")
	// Only the ftyp box survived a finalize killed mid-concat: the file is
	// non-empty but has no moov, so recovery must rebuild, not mark ready.
	if err := os.WriteFile(fixture.screenPath, probeable[:16], 0o644); err != nil {
		t.Fatalf("write truncated screen: %v", err)
	}
	rebuilt := false
	hooks := recoveryHooks{
		rebuildScreenVideo: func(ffmpegPath string, outputPath string, segmentDir string) (int, error) {
			rebuilt = true
			if err := os.WriteFile(outputPath, probeable, 0o644); err != nil {
				return 0, err
			}
			return 1, nil
		},
	}
	service := fixture.service(hooks)

	summary, err := service.RecoverPackage(fixture.packageDir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if !rebuilt {
		t.Fatal("truncated screen track did not trigger a rebuild")
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("summary status = %q, want ready", summary.Status)
	}
}

func TestRecoverPackageAudioOnlyRebuildsM4A(t *testing.T) {
	fixture := newRecoveryFixture(t, true)
	manifest := fixture.manifest
	manifest.RecordingMode = recpackage.RecordingModeAudio
	manifest.Media.AudioPath = "audio/audio.m4a"
	manifest.Media.ScreenVideoPath = ""
	if err := fixture.packages.WriteManifest(fixture.manifestStr, manifest); err != nil {
		t.Fatalf("WriteManifest(audio-only) error = %v", err)
	}

	rebuilt := ""
	hooks := recoveryHooks{
		muxAudioOnly: func(config video.AudioOnlyMuxConfig) (video.AudioOnlyMuxResult, error) {
			rebuilt = config.OutputPath
			if err := os.WriteFile(config.OutputPath, []byte("recovered-audio"), 0o644); err != nil {
				return video.AudioOnlyMuxResult{}, err
			}
			return video.AudioOnlyMuxResult{}, nil
		},
	}
	service := fixture.service(hooks)

	summary, err := service.RecoverPackage(fixture.packageDir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("summary status = %q, want ready", summary.Status)
	}
	want := filepath.Join(fixture.packageDir, "audio", "audio.m4a")
	if rebuilt != want {
		t.Fatalf("audio-only output = %q, want %q", rebuilt, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("rebuilt m4a missing: %v", err)
	}
}

func TestRecoverPackageScreenMuxFailureStillRecoversVideo(t *testing.T) {
	fixture := newRecoveryFixture(t, true)
	if err := os.WriteFile(fixture.screenPath, minimalMP4Data("vide"), 0o644); err != nil {
		t.Fatalf("write screen: %v", err)
	}
	hooks := recoveryHooks{
		muxScreenAudio: func(video.AudioMuxConfig) (video.AudioMuxResult, error) {
			return video.AudioMuxResult{}, errors.New("ffmpeg exploded")
		},
	}
	service := fixture.service(hooks)

	summary, err := service.RecoverPackage(fixture.packageDir)
	if err != nil {
		t.Fatalf("RecoverPackage() error = %v", err)
	}
	if summary.Status != recpackage.StatusReady {
		t.Fatalf("summary status = %q, want ready", summary.Status)
	}
	if !strings.Contains(summary.Reason, "audio mux skipped") {
		t.Fatalf("summary reason = %q, want skipped-audio note", summary.Reason)
	}
}
