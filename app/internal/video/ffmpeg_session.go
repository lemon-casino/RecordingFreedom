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
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	EnvFFmpegPath             = "RECORDINGFREEDOM_FFMPEG_PATH"
	ffmpegStopTimeout         = 10 * time.Second
	ffmpegFinalizeTimeout     = 30 * time.Second
	ffmpegSegmentDirectory    = "cache"
	ffmpegVideoSegmentSubdir  = "ffmpeg-video"
	ffmpegSegmentListFileName = "segments.txt"
)

type ffmpegInputArgsBuilder func(CaptureConfig) ([]string, error)

type ffmpegDesktopSession struct {
	config    CaptureConfig
	ffmpeg    string
	inputArgs ffmpegInputArgsBuilder

	diagnostics Diagnostics
	segments    []ffmpegSegment
	active      *ffmpegProcess
	paused      bool
	started     bool
	stopped     bool
	totalActive time.Duration

	mu sync.Mutex
}

type ffmpegSegment struct {
	Path    string
	Started time.Time
	Stopped time.Time
	Bytes   int64
}

type ffmpegProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stderr  *bytes.Buffer
	path    string
	started time.Time
}

func FFmpegAvailability() (string, bool, string) {
	path, err := ResolveFFmpegPath()
	if err == nil {
		return path, true, fmt.Sprintf("FFmpeg executable found at %s", path)
	}
	return "", false, err.Error()
}

func ResolveFFmpegPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(EnvFFmpegPath)); configured != "" {
		return validateExecutablePath(configured, EnvFFmpegPath)
	}

	names := ffmpegExecutableNames()
	candidates := make([]string, 0, 8)
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(base, name),
				filepath.Join(base, "tools", name),
				filepath.Join(base, "bin", name),
			)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(cwd, name),
				filepath.Join(cwd, "tools", name),
				filepath.Join(cwd, "bin", name),
			)
		}
	}
	for _, candidate := range candidates {
		if path, err := validateExecutablePath(candidate, "bundled ffmpeg"); err == nil {
			return path, nil
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("FFmpeg executable was not found; set %s or bundle ffmpeg beside the app under tools/", EnvFFmpegPath)
}

func ffmpegExecutableNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"ffmpeg.exe", "ffmpeg"}
	}
	return []string{"ffmpeg"}
}

func validateExecutablePath(path string, source string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is empty", source)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s %q is not readable: %w", source, path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s %q is a directory, not an ffmpeg executable", source, path)
	}
	return path, nil
}

func newFFmpegDesktopSession(config CaptureConfig, inputArgs ffmpegInputArgsBuilder) (*ffmpegDesktopSession, error) {
	config = NormalizeCaptureConfig(config)
	if config.OutputPath == "" {
		return nil, errors.New("FFmpeg output path is required")
	}
	if inputArgs == nil {
		return nil, errors.New("FFmpeg input args builder is required")
	}
	ffmpegPath, err := ResolveFFmpegPath()
	if err != nil {
		return nil, err
	}
	diagnostics := NewDiagnostics(config)
	diagnostics.Screen.Path = filepath.Base(config.OutputPath)
	diagnostics.Screen.Clock = "media-timestamp"
	diagnostics.Screen.Width = captureWidth(config)
	diagnostics.Screen.Height = captureHeight(config)
	diagnostics.Messages = append(diagnostics.Messages, "FFmpeg desktop capture writer initialized.")
	return &ffmpegDesktopSession{
		config:      config,
		ffmpeg:      ffmpegPath,
		inputArgs:   inputArgs,
		diagnostics: diagnostics,
	}, nil
}

func captureWidth(config CaptureConfig) int {
	if config.SourceGeometry != nil && config.SourceGeometry.Width > 0 {
		return evenOutputDimension(config.SourceGeometry.Width)
	}
	return 0
}

func captureHeight(config CaptureConfig) int {
	if config.SourceGeometry != nil && config.SourceGeometry.Height > 0 {
		return evenOutputDimension(config.SourceGeometry.Height)
	}
	return 0
}

func evenOutputDimension(value int) int {
	if value <= 0 {
		return 0
	}
	if value%2 == 0 {
		return value
	}
	return value + 1
}

func (s *ffmpegDesktopSession) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("FFmpeg desktop capture session is stopped")
	}
	if s.started {
		return errors.New("FFmpeg desktop capture session is already started")
	}
	if err := os.MkdirAll(filepath.Dir(s.config.OutputPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.segmentDir(), 0o755); err != nil {
		return err
	}
	if err := s.startSegmentLocked(ctx); err != nil {
		return err
	}
	s.started = true
	s.diagnostics.Messages = append(s.diagnostics.Messages, "FFmpeg segment 0 started.")
	return nil
}

func (s *ffmpegDesktopSession) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || s.stopped || s.paused {
		return nil
	}
	if err := s.stopActiveSegmentLocked("pause"); err != nil {
		return err
	}
	s.paused = true
	s.diagnostics.Messages = append(s.diagnostics.Messages, "FFmpeg capture paused after closing the active segment.")
	return nil
}

func (s *ffmpegDesktopSession) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || s.stopped || !s.paused {
		return nil
	}
	if err := s.startSegmentLocked(context.Background()); err != nil {
		return err
	}
	s.paused = false
	s.diagnostics.Messages = append(s.diagnostics.Messages, fmt.Sprintf("FFmpeg segment %d started after resume.", len(s.segments)))
	return nil
}

func (s *ffmpegDesktopSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	s.stopped = true
	var errs []error
	if s.active != nil {
		if err := s.stopActiveSegmentLocked("stop"); err != nil {
			errs = append(errs, err)
		}
	}
	if len(s.segments) == 0 {
		errs = append(errs, errors.New("FFmpeg desktop capture wrote no segments"))
	} else if err := s.finalizeLocked(); err != nil {
		errs = append(errs, err)
	}
	s.patchDiagnosticsLocked()
	if strings.TrimSpace(s.config.DiagnosticsPath) != "" {
		if err := WriteDiagnostics(s.config.DiagnosticsPath, s.diagnostics); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *ffmpegDesktopSession) Diagnostics() Diagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.diagnostics
	if s.active != nil {
		activeDuration := s.totalActive + time.Since(s.active.started)
		next.Screen.DurationMs = activeDuration.Milliseconds()
		next.Screen.EndOffsetMs = next.Screen.DurationMs
		next.Screen.FramesWritten = estimatedFrames(activeDuration, s.config.Profile.FPS)
	}
	return next
}

func (s *ffmpegDesktopSession) startSegmentLocked(ctx context.Context) error {
	input, err := s.inputArgs(s.config)
	if err != nil {
		return err
	}
	segmentPath := filepath.Join(s.segmentDir(), fmt.Sprintf("segment-%03d.mp4", len(s.segments)))
	args := []string{"-hide_banner", "-loglevel", "warning", "-y"}
	args = append(args, input...)
	args = append(args, s.encodingArgs(segmentPath)...)

	cmd := exec.CommandContext(ctx, s.ffmpeg, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open FFmpeg stdin: %w", err)
	}
	stderr := &bytes.Buffer{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("start FFmpeg desktop capture: %w", err)
	}
	s.active = &ffmpegProcess{
		cmd:     cmd,
		stdin:   stdin,
		stderr:  stderr,
		path:    segmentPath,
		started: time.Now(),
	}
	return nil
}

func (s *ffmpegDesktopSession) encodingArgs(outputPath string) []string {
	return []string{
		"-an",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", ffmpegCRF(s.config.Profile.Quality),
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outputPath,
	}
}

func ffmpegCRF(quality string) string {
	switch strings.TrimSpace(strings.ToLower(quality)) {
	case "standard":
		return "28"
	case "high":
		return "20"
	default:
		return "24"
	}
}

func (s *ffmpegDesktopSession) stopActiveSegmentLocked(reason string) error {
	active := s.active
	if active == nil {
		return nil
	}
	s.active = nil
	if active.stdin != nil {
		_, _ = io.WriteString(active.stdin, "q")
	}

	done := make(chan error, 1)
	go func() {
		done <- active.cmd.Wait()
	}()

	var waitErr error
	select {
	case waitErr = <-done:
	case <-time.After(ffmpegStopTimeout):
		if active.stdin != nil {
			_ = active.stdin.Close()
		}
		if active.cmd.Process != nil {
			_ = active.cmd.Process.Kill()
		}
		waitErr = fmt.Errorf("timed out stopping FFmpeg segment for %s", reason)
		<-done
	}
	if active.stdin != nil {
		_ = active.stdin.Close()
	}

	stopped := time.Now()
	duration := stopped.Sub(active.started)
	if duration > 0 {
		s.totalActive += duration
	}
	info, statErr := os.Stat(active.path)
	if statErr == nil && !info.IsDir() && info.Size() > 0 {
		s.segments = append(s.segments, ffmpegSegment{
			Path:    active.path,
			Started: active.started,
			Stopped: stopped,
			Bytes:   info.Size(),
		})
	}
	if message := ffmpegStderrMessage(active.stderr); message != "" {
		s.diagnostics.Messages = append(s.diagnostics.Messages, message)
	}
	if waitErr != nil {
		return fmt.Errorf("FFmpeg segment failed: %w%s", waitErr, stderrSuffix(active.stderr))
	}
	if statErr != nil {
		return fmt.Errorf("stat FFmpeg segment: %w", statErr)
	}
	if info == nil || info.IsDir() || info.Size() == 0 {
		return fmt.Errorf("FFmpeg segment %q is empty", active.path)
	}
	return nil
}

func (s *ffmpegDesktopSession) finalizeLocked() error {
	if err := os.RemoveAll(s.config.OutputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(s.segments) == 1 {
		if err := os.Rename(s.segments[0].Path, s.config.OutputPath); err == nil {
			s.diagnostics.Messages = append(s.diagnostics.Messages, "Single FFmpeg segment moved to screen.mp4.")
			return nil
		}
		if err := copyFile(s.segments[0].Path, s.config.OutputPath); err != nil {
			return err
		}
		s.diagnostics.Messages = append(s.diagnostics.Messages, "Single FFmpeg segment copied to screen.mp4.")
		return nil
	}

	listPath, err := s.writeConcatListLocked()
	if err != nil {
		return err
	}
	args := []string{
		"-hide_banner", "-loglevel", "warning", "-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		"-movflags", "+faststart",
		s.config.OutputPath,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegFinalizeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.ffmpeg, args...)
	stderr := &bytes.Buffer{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg concat finalize failed: %w%s", err, stderrSuffix(stderr))
	}
	s.diagnostics.Messages = append(s.diagnostics.Messages, fmt.Sprintf("Merged %d FFmpeg segments into screen.mp4.", len(s.segments)))
	return nil
}

func (s *ffmpegDesktopSession) writeConcatListLocked() (string, error) {
	listPath := filepath.Join(s.segmentDir(), ffmpegSegmentListFileName)
	var builder strings.Builder
	for _, segment := range s.segments {
		builder.WriteString("file '")
		builder.WriteString(strings.ReplaceAll(filepath.ToSlash(segment.Path), "'", "'\\''"))
		builder.WriteString("'\n")
	}
	if err := os.WriteFile(listPath, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}
	return listPath, nil
}

func (s *ffmpegDesktopSession) patchDiagnosticsLocked() {
	duration := s.totalActive
	s.diagnostics.Screen.Enabled = true
	s.diagnostics.Screen.Path = filepath.Base(s.config.OutputPath)
	s.diagnostics.Screen.Clock = "media-timestamp"
	if s.diagnostics.Screen.Width == 0 {
		s.diagnostics.Screen.Width = captureWidth(s.config)
	}
	if s.diagnostics.Screen.Height == 0 {
		s.diagnostics.Screen.Height = captureHeight(s.config)
	}
	s.diagnostics.Screen.FrameRate = s.config.Profile.FPS
	s.diagnostics.Screen.FramesWritten = estimatedFrames(duration, s.config.Profile.FPS)
	s.diagnostics.Screen.StartOffsetMs = 0
	s.diagnostics.Screen.EndOffsetMs = duration.Milliseconds()
	s.diagnostics.Screen.DurationMs = duration.Milliseconds()
	if len(s.segments) == 0 {
		s.diagnostics.Screen.Enabled = false
		s.diagnostics.Screen.Message = "FFmpeg desktop capture stopped before a segment was written."
		return
	}
	s.diagnostics.Screen.Message = fmt.Sprintf("FFmpeg desktop capture wrote %d segment(s).", len(s.segments))
}

func (s *ffmpegDesktopSession) segmentDir() string {
	return filepath.Join(filepath.Dir(s.config.OutputPath), ffmpegSegmentDirectory, ffmpegVideoSegmentSubdir)
}

func estimatedFrames(duration time.Duration, fps int) int64 {
	if duration <= 0 || fps <= 0 {
		return 0
	}
	return int64(duration.Seconds() * float64(fps))
}

func ffmpegStderrMessage(stderr *bytes.Buffer) string {
	if stderr == nil || stderr.Len() == 0 {
		return ""
	}
	message := strings.TrimSpace(stderr.String())
	if message == "" {
		return ""
	}
	if len(message) > 1200 {
		message = message[len(message)-1200:]
	}
	return "FFmpeg: " + message
}

func stderrSuffix(stderr *bytes.Buffer) string {
	message := ffmpegStderrMessage(stderr)
	if message == "" {
		return ""
	}
	return ": " + message
}

func copyFile(from string, to string) error {
	input, err := os.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}
