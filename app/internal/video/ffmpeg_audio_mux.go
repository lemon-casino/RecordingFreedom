package video

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ffmpegAudioMuxTimeout = 2 * time.Minute
	audioMuxMinBytes      = 45
)

type AudioMuxInput struct {
	Path  string
	Label string
}

type AudioMuxConfig struct {
	VideoPath string
	Inputs    []AudioMuxInput
	Timeout   time.Duration
}

type AudioOnlyMuxConfig struct {
	OutputPath string
	Inputs     []AudioMuxInput
	Timeout    time.Duration
}

type AudioMuxResult struct {
	VideoPath string
	Inputs    []AudioMuxInput
}

type AudioOnlyMuxResult struct {
	OutputPath string
	Inputs     []AudioMuxInput
}

func MuxAudioIntoMP4(config AudioMuxConfig) (AudioMuxResult, error) {
	config.VideoPath = strings.TrimSpace(config.VideoPath)
	if config.VideoPath == "" {
		return AudioMuxResult{}, errors.New("audio mux video path is required")
	}
	if err := requireReadableFile(config.VideoPath, 1); err != nil {
		return AudioMuxResult{}, fmt.Errorf("audio mux video input: %w", err)
	}
	inputs, err := validatedAudioMuxInputs(config.Inputs)
	if err != nil {
		return AudioMuxResult{}, err
	}
	if len(inputs) == 0 {
		return AudioMuxResult{VideoPath: config.VideoPath}, nil
	}
	ffmpegPath, err := ResolveFFmpegPath()
	if err != nil {
		return AudioMuxResult{}, err
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = ffmpegAudioMuxTimeout
	}

	dir := filepath.Dir(config.VideoPath)
	tmp, err := os.CreateTemp(dir, ".screen-audio-mux-*.mp4")
	if err != nil {
		return AudioMuxResult{}, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)

	args := ffmpegAudioMuxArgs(config.VideoPath, inputs, tmpPath)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	configureBackgroundCommand(cmd)
	stderr := &bytes.Buffer{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpPath)
		return AudioMuxResult{}, fmt.Errorf("FFmpeg audio mux failed: %w%s", err, stderrSuffix(stderr))
	}
	if err := requireReadableFile(tmpPath, 1); err != nil {
		_ = os.Remove(tmpPath)
		return AudioMuxResult{}, fmt.Errorf("audio mux output: %w", err)
	}
	if err := replaceFileAtomically(config.VideoPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return AudioMuxResult{}, err
	}
	return AudioMuxResult{VideoPath: config.VideoPath, Inputs: inputs}, nil
}

func MuxAudioOnlyToM4A(config AudioOnlyMuxConfig) (AudioOnlyMuxResult, error) {
	config.OutputPath = strings.TrimSpace(config.OutputPath)
	if config.OutputPath == "" {
		return AudioOnlyMuxResult{}, errors.New("audio-only mux output path is required")
	}
	inputs, err := validatedAudioMuxInputs(config.Inputs)
	if err != nil {
		return AudioOnlyMuxResult{}, err
	}
	if len(inputs) == 0 {
		return AudioOnlyMuxResult{}, errors.New("audio-only mux requires at least one readable audio input")
	}
	ffmpegPath, err := ResolveFFmpegPath()
	if err != nil {
		return AudioOnlyMuxResult{}, err
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = ffmpegAudioMuxTimeout
	}

	dir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AudioOnlyMuxResult{}, err
	}
	tmp, err := os.CreateTemp(dir, ".audio-only-mux-*.m4a")
	if err != nil {
		return AudioOnlyMuxResult{}, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)

	args := ffmpegAudioOnlyMuxArgs(inputs, tmpPath)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	configureBackgroundCommand(cmd)
	stderr := &bytes.Buffer{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpPath)
		return AudioOnlyMuxResult{}, fmt.Errorf("FFmpeg audio-only mux failed: %w%s", err, stderrSuffix(stderr))
	}
	if err := requireReadableFile(tmpPath, 1); err != nil {
		_ = os.Remove(tmpPath)
		return AudioOnlyMuxResult{}, fmt.Errorf("audio-only mux output: %w", err)
	}
	if err := installFileAtomically(config.OutputPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return AudioOnlyMuxResult{}, err
	}
	return AudioOnlyMuxResult{OutputPath: config.OutputPath, Inputs: inputs}, nil
}

func validatedAudioMuxInputs(inputs []AudioMuxInput) ([]AudioMuxInput, error) {
	validated := make([]AudioMuxInput, 0, len(inputs))
	seen := map[string]bool{}
	for _, input := range inputs {
		path := strings.TrimSpace(input.Path)
		if path == "" {
			continue
		}
		cleaned, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if seen[cleaned] {
			continue
		}
		if err := requireReadableFile(cleaned, audioMuxMinBytes); err != nil {
			return nil, fmt.Errorf("audio mux input %q: %w", path, err)
		}
		seen[cleaned] = true
		validated = append(validated, AudioMuxInput{
			Path:  cleaned,
			Label: strings.TrimSpace(input.Label),
		})
	}
	return validated, nil
}

func ffmpegAudioMuxArgs(videoPath string, inputs []AudioMuxInput, outputPath string) []string {
	args := []string{"-hide_banner", "-loglevel", "warning", "-y", "-i", videoPath}
	for _, input := range inputs {
		args = append(args, "-i", input.Path)
	}
	args = append(args, "-map", "0:v:0", "-c:v", "copy")
	if len(inputs) == 1 {
		args = append(args, "-map", "1:a:0")
	} else {
		args = append(args, "-filter_complex", audioMixFilter(len(inputs)), "-map", "[mixed_audio]")
	}
	args = append(args,
		"-c:a", "aac",
		"-b:a", "192k",
		"-ar", "48000",
		"-ac", "2",
		"-shortest",
		"-movflags", "+faststart",
		outputPath,
	)
	return args
}

func ffmpegAudioOnlyMuxArgs(inputs []AudioMuxInput, outputPath string) []string {
	args := []string{"-hide_banner", "-loglevel", "warning", "-y"}
	for _, input := range inputs {
		args = append(args, "-i", input.Path)
	}
	if len(inputs) == 1 {
		args = append(args, "-map", "0:a:0")
	} else {
		args = append(args, "-filter_complex", audioOnlyMixFilter(len(inputs)), "-map", "[mixed_audio]")
	}
	args = append(args,
		"-vn",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ar", "48000",
		"-ac", "2",
		"-movflags", "+faststart",
		outputPath,
	)
	return args
}

func audioMixFilter(inputCount int) string {
	return audioMixFilterFrom(1, inputCount)
}

func audioOnlyMixFilter(inputCount int) string {
	return audioMixFilterFrom(0, inputCount)
}

func audioMixFilterFrom(firstInputIndex int, inputCount int) string {
	if inputCount <= 1 {
		return ""
	}
	parts := make([]string, 0, inputCount+1)
	labels := make([]string, 0, inputCount)
	for index := 0; index < inputCount; index++ {
		label := fmt.Sprintf("a%d", index)
		parts = append(parts, fmt.Sprintf("[%d:a]aresample=async=1:first_pts=0[%s]", firstInputIndex+index, label))
		labels = append(labels, "["+label+"]")
	}
	parts = append(parts, fmt.Sprintf("%samix=inputs=%d:duration=longest:dropout_transition=0,aresample=async=1:first_pts=0[mixed_audio]", strings.Join(labels, ""), inputCount))
	return strings.Join(parts, ";")
}

func requireReadableFile(path string, minBytes int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() || info.Size() < minBytes {
		return fmt.Errorf("%q is not readable media", path)
	}
	return nil
}

func replaceFileAtomically(target string, replacement string) error {
	backup := target + ".before-audio-mux"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("prepare audio mux replacement: %w", err)
	}
	if err := os.Rename(replacement, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install audio mux replacement: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

func installFileAtomically(target string, replacement string) error {
	if _, err := os.Stat(target); err == nil {
		return replaceFileAtomically(target, replacement)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat audio mux target: %w", err)
	}
	if err := os.Rename(replacement, target); err != nil {
		return fmt.Errorf("install audio mux output: %w", err)
	}
	return nil
}
