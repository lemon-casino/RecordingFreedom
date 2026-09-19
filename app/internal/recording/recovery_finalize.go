package recording

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
	"github.com/lemon-casino/RecordingFreedom/app/internal/video"
)

// recoveryHooks lets tests stub the FFmpeg-backed finalize steps.
type recoveryHooks struct {
	rebuildScreenVideo func(ffmpegPath string, outputPath string, segmentDir string) (int, error)
	muxScreenAudio     func(config video.AudioMuxConfig) (video.AudioMuxResult, error)
	muxAudioOnly       func(config video.AudioOnlyMuxConfig) (video.AudioOnlyMuxResult, error)
}

func defaultRecoveryHooks() recoveryHooks {
	return recoveryHooks{
		rebuildScreenVideo: func(ffmpegPath string, outputPath string, segmentDir string) (int, error) {
			if ffmpegPath == "" {
				resolved, err := video.ResolveFFmpegPath()
				if err != nil {
					return 0, err
				}
				ffmpegPath = resolved
			}
			return video.RebuildScreenVideo(ffmpegPath, outputPath, segmentDir)
		},
		muxScreenAudio: video.MuxAudioIntoMP4,
		muxAudioOnly:   video.MuxAudioOnlyToM4A,
	}
}

// finalizeCrashedPackage rebuilds the media a killed recording process never
// finalized: it concatenates cached FFmpeg segments into the screen track and
// muxes surviving audio sidecars into it (or into the audio-only output).
// Every step is best-effort; the returned notes describe what happened and
// feed the recovery summary. Media rebuilds happen before packages.Recover
// validates the package; manifest patches run only once the media exists.
func (s *Service) finalizeCrashedPackage(packageDir string) ([]string, error) {
	manifestPath := filepath.Join(packageDir, recpackage.ManifestFile)
	manifest, err := s.packages.ReadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	hooks := s.recoveryHooks

	notes := make([]string, 0, 3)
	switch manifest.RecordingMode {
	case recpackage.RecordingModeScreen:
		screenPath := filepath.Join(packageDir, filepath.FromSlash(manifest.Media.ScreenVideoPath))
		if !readableScreenMedia(screenPath, manifest.Diagnostics.Mock) {
			rebuildNote, rebuildErr := rebuildScreenTrack(hooks, packageDir, screenPath)
			notes = append(notes, rebuildNote)
			if rebuildErr != nil {
				return notes, rebuildErr
			}
		}
		muxNote, muxErr := muxCrashedScreenAudio(s.packages, hooks, manifestPath, packageDir, manifest, screenPath)
		if muxNote != "" {
			notes = append(notes, muxNote)
		}
		if muxErr != nil {
			notes = append(notes, "audio mux skipped: "+muxErr.Error())
		}
	case recpackage.RecordingModeAudio:
		rebuildNote, rebuildErr := rebuildCrashedAudioOnly(s.packages, hooks, manifestPath, packageDir, manifest)
		if rebuildNote != "" {
			notes = append(notes, rebuildNote)
		}
		if rebuildErr != nil {
			notes = append(notes, "audio-only rebuild skipped: "+rebuildErr.Error())
		}
	}
	return notes, nil
}

func rebuildScreenTrack(hooks recoveryHooks, packageDir string, screenPath string) (string, error) {
	segmentDir := video.SegmentRecoveryDir(packageDir, screenPath)
	count, err := hooks.rebuildScreenVideo("", screenPath, segmentDir)
	if err != nil {
		return "", fmt.Errorf("cannot rebuild screen segments from %s: %w", segmentDir, err)
	}
	return fmt.Sprintf("rebuilt %s from %d cached FFmpeg segment(s).", filepath.Base(screenPath), count), nil
}

func muxCrashedScreenAudio(packages *recpackage.Service, hooks recoveryHooks, manifestPath string, packageDir string, manifest recpackage.Manifest, screenPath string) (string, error) {
	inputs, muxSystem, muxMicrophone := crashedAudioSidecarInputs(packageDir, manifest)
	if len(inputs) == 0 {
		return "", nil
	}
	if !readableOutputFile(screenPath) {
		return "", errors.New("screen media is still missing")
	}
	if _, err := hooks.muxScreenAudio(video.AudioMuxConfig{VideoPath: screenPath, Inputs: inputs}); err != nil {
		return "", err
	}
	if _, err := packages.PatchScreenAudioMuxed(manifestPath, muxSystem, muxMicrophone); err != nil {
		return "muxed recovered audio into " + filepath.Base(screenPath) + ".", err
	}
	return "muxed recovered audio into " + filepath.Base(screenPath) + ".", nil
}

func rebuildCrashedAudioOnly(packages *recpackage.Service, hooks recoveryHooks, manifestPath string, packageDir string, manifest recpackage.Manifest) (string, error) {
	outputPath := filepath.Join(packageDir, filepath.FromSlash(manifest.Media.AudioPath))
	if readableOutputFile(outputPath) {
		return "", nil
	}
	inputs, muxSystem, muxMicrophone := crashedAudioSidecarInputs(packageDir, manifest)
	if len(inputs) == 0 {
		return "", errors.New("no readable audio sidecar to rebuild from")
	}
	if _, err := hooks.muxAudioOnly(video.AudioOnlyMuxConfig{OutputPath: outputPath, Inputs: inputs}); err != nil {
		return "", err
	}
	if _, err := packages.PatchAudioOnlyMuxed(manifestPath, muxSystem, muxMicrophone); err != nil {
		return "rebuilt " + filepath.Base(outputPath) + " from recovered audio sidecar(s).", err
	}
	return "rebuilt " + filepath.Base(outputPath) + " from recovered audio sidecar(s).", nil
}

func crashedAudioSidecarInputs(packageDir string, manifest recpackage.Manifest) ([]video.AudioMuxInput, bool, bool) {
	inputs := make([]video.AudioMuxInput, 0, 2)
	muxSystem := false
	muxMicrophone := false
	if manifest.Audio.System && manifest.Media.SystemAudioStorage == recpackage.AudioStorageSidecar {
		path := filepath.Join(packageDir, filepath.FromSlash(manifest.Media.SystemAudioPath))
		if readableAudioSidecar(path) {
			inputs = append(inputs, video.AudioMuxInput{Path: path, Label: "system"})
			muxSystem = true
		}
	}
	if manifest.Audio.Microphone && manifest.Media.MicrophoneAudioStorage == recpackage.AudioStorageSidecar {
		path := filepath.Join(packageDir, filepath.FromSlash(manifest.Media.MicrophoneAudioPath))
		if readableAudioSidecar(path) {
			inputs = append(inputs, video.AudioMuxInput{Path: path, Label: "microphone"})
			muxMicrophone = true
		}
	}
	return inputs, muxSystem, muxMicrophone
}

func readableOutputFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// readableScreenMedia reports whether screenPath holds media the recovery
// chain can trust: a non-empty file that probes as an MP4 with a video track.
// A file left half-written by a finalize that was killed mid-concat passes
// the byte-size check but has no moov atom yet, so treating it as present
// would fake a ready package; it must be rebuilt from cached segments
// instead. Mock packages keep the size-only check: their marker file is
// plain text by design.
func readableScreenMedia(path string, mock bool) bool {
	if !readableOutputFile(path) {
		return false
	}
	if mock {
		return true
	}
	probe, err := recpackage.ProbeMP4(path)
	return err == nil && probe.HasVideoTrack
}
