package evidencetool

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RequireNonEmptyFile verifies that path exists, is a file, and is non-empty.
// Source: cmd/annotation-overlay-evidence-check, cmd/ocr-desktop-evidence-check.
func RequireNonEmptyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%s is empty", path)
	}
	return nil
}

// RequireDirWithFile verifies that path is a directory containing at least one file.
// Source: cmd/annotation-overlay-evidence-check, cmd/ocr-desktop-evidence-check.
func RequireDirWithFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	hasFile := false
	err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			hasFile = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !hasFile {
		return fmt.Errorf("%s contains no files", path)
	}
	return nil
}

// EvidenceFileNames returns the slash-separated, lowercased relative names of
// all files below path.
// Source: cmd/annotation-overlay-evidence-check, cmd/ocr-desktop-evidence-check.
func EvidenceFileNames(path string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(path, func(itemPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, itemPath)
		if err != nil {
			return err
		}
		files = append(files, strings.ToLower(filepath.ToSlash(rel)))
		return nil
	})
	return files, err
}

// RequirePath resolves path (flagName is used in the empty-value error) to an
// absolute path.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-worker-platform-smoke (identical copies).
func RequirePath(path string, flagName string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", flagName)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// RequireFile resolves path and verifies it is not a directory.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-worker-platform-smoke.
func RequireFile(path string, flagName string) (string, error) {
	resolved, err := RequirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s must be a file: %s", flagName, resolved)
	}
	return resolved, nil
}

// RequireDir resolves path and verifies it is a directory.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-worker-platform-smoke.
func RequireDir(path string, flagName string) (string, error) {
	resolved, err := RequirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s must be a directory: %s", flagName, resolved)
	}
	return resolved, nil
}

// RequirePackagePath resolves path and verifies it is a directory or a .zip file.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-worker-platform-smoke.
func RequirePackagePath(path string, flagName string) (string, error) {
	resolved, err := RequirePath(path, flagName)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return resolved, nil
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".zip") {
		return "", fmt.Errorf("%s must be a .zip file or directory: %s", flagName, resolved)
	}
	return resolved, nil
}

// PrepareEvidenceDir trims path, resolves it to an absolute path, and creates
// the directory (and parents) when missing.
// Source: cmd/ocr-model-lifecycle-smoke, cmd/ocr-worker-platform-smoke.
func PrepareEvidenceDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("evidence dir is required")
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return "", err
	}
	return resolved, nil
}
