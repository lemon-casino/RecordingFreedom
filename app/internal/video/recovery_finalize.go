package video

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// recoverySegmentGlob matches the per-group segment files a desktop FFmpeg
// session writes under cache/ffmpeg-video/<output>/ (segment-GROUP-NNN.mp4).
const recoverySegmentGlob = "segment-*.mp4"

// RebuildScreenVideo rebuilds outputPath from the FFmpeg segments a desktop
// session wrote before a crash. The last segment of a killed process usually
// lacks its moov atom, so concat is retried with progressively fewer segments
// until it succeeds; the returned count reports how many segments the final
// output contains. A rebuild from zero segments returns os.ErrNotExist.
func RebuildScreenVideo(ffmpegPath string, outputPath string, segmentDir string) (int, error) {
	segments, err := listRecoverySegments(segmentDir)
	if err != nil {
		return 0, err
	}
	for count := len(segments); count > 0; count-- {
		err := concatSegments(ffmpegPath, outputPath, segments[:count])
		if err == nil {
			return count, nil
		}
		if count == 1 {
			_ = os.Remove(outputPath)
			return 0, fmt.Errorf("rebuild from %d segment(s) failed: %w", len(segments), err)
		}
	}
	return 0, os.ErrNotExist
}

func listRecoverySegments(segmentDir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(segmentDir, recoverySegmentGlob))
	if err != nil {
		return nil, err
	}
	slices.Sort(matches)
	segments := make([]string, 0, len(matches))
	for _, path := range matches {
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		if info.IsDir() || info.Size() == 0 {
			continue
		}
		segments = append(segments, path)
	}
	if len(segments) == 0 {
		return nil, os.ErrNotExist
	}
	return segments, nil
}

func concatSegments(ffmpegPath string, outputPath string, segments []string) error {
	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(segments) == 1 {
		if err := os.Rename(segments[0], outputPath); err == nil {
			return nil
		}
		return copyFile(segments[0], outputPath)
	}
	listPath, err := writeSegmentConcatList(outputPath, segments)
	if err != nil {
		return err
	}
	args := ffmpegConcatArgs(listPath, outputPath)
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegFinalizeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	configureBackgroundCommand(cmd)
	stderr := &strings.Builder{}
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg recovery concat failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func writeSegmentConcatList(outputPath string, segments []string) (string, error) {
	listPath := filepath.Join(filepath.Dir(outputPath), "recovery-segments.txt")
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString("file '")
		builder.WriteString(strings.ReplaceAll(filepath.ToSlash(segment), "'", "'\\''"))
		builder.WriteString("'\n")
	}
	if err := os.WriteFile(listPath, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}
	return listPath, nil
}

// SegmentRecoveryDir returns the FFmpeg segment cache directory a desktop
// session uses for outputPath inside packageDir.
func SegmentRecoveryDir(packageDir string, outputPath string) string {
	name := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))
	return filepath.Join(packageDir, ffmpegSegmentDirectory, ffmpegVideoSegmentSubdir, ffmpegSegmentOutputSubdir(name))
}
