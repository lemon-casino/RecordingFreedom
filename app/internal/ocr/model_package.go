package ocr

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	modelInstallStagingPrefix = ".install-"
	modelInstallBackupPrefix  = ".backup-"
)

func (s *Service) InstallModelPackage(sourcePath string) (ModelInfo, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return ModelInfo{}, errors.New("OCR model package path is required")
	}
	absoluteSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return ModelInfo{}, err
	}
	info, err := os.Stat(absoluteSource)
	if err != nil {
		return ModelInfo{}, err
	}
	root, err := s.modelRoot()
	if err != nil {
		return ModelInfo{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ModelInfo{}, err
	}
	staging, err := os.MkdirTemp(root, modelInstallStagingPrefix)
	if err != nil {
		return ModelInfo{}, err
	}
	stagingDone := false
	defer func() {
		if !stagingDone {
			_ = os.RemoveAll(staging)
		}
	}()

	if info.IsDir() {
		if err := copyModelPackageDir(absoluteSource, staging); err != nil {
			return ModelInfo{}, err
		}
	} else if strings.EqualFold(filepath.Ext(absoluteSource), ".zip") {
		if err := extractModelPackageZip(absoluteSource, staging); err != nil {
			return ModelInfo{}, err
		}
	} else {
		return ModelInfo{}, fmt.Errorf("OCR model package %q must be a directory or .zip file", sourcePath)
	}

	packageDir, manifest, err := locateModelPackageManifest(staging)
	if err != nil {
		return ModelInfo{}, err
	}
	fallback, ok := findDefaultModel(manifest.ID)
	if !ok {
		return ModelInfo{}, fmt.Errorf("unknown OCR model %q in package manifest", manifest.ID)
	}
	installed, missing, verificationErr := verifyModelDir(packageDir, fallback)
	if !installed {
		return ModelInfo{}, fmt.Errorf("OCR model package %q is missing required files: %s", manifest.ID, strings.Join(missing, ", "))
	}
	if verificationErr != "" {
		return ModelInfo{}, fmt.Errorf("OCR model package %q failed verification: %s", manifest.ID, verificationErr)
	}
	if err := installVerifiedModelDir(root, packageDir, manifest.ID); err != nil {
		return ModelInfo{}, err
	}
	if err := os.RemoveAll(staging); err != nil {
		return ModelInfo{}, err
	}
	stagingDone = true
	return s.modelInfo(fallback, activeModelIDFromState(s))
}

func (s *Service) modelRoot() (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", modelRootDir, ocrModelDir), nil
}

func locateModelPackageManifest(root string) (string, ModelManifest, error) {
	candidates := []string{filepath.Join(root, "manifest.json")}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", ModelManifest{}, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(root, entry.Name(), "manifest.json"))
		}
	}
	for _, candidate := range candidates {
		manifest, err := readModelManifest(candidate)
		if err == nil {
			return filepath.Dir(candidate), manifest, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", ModelManifest{}, err
		}
	}
	return "", ModelManifest{}, errors.New("OCR model package manifest.json was not found")
}

func copyModelPackageDir(source string, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if !safeModelPackageRelativePath(rel) {
			return fmt.Errorf("unsafe OCR model package path %q", rel)
		}
		destination := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("OCR model package path %q is a symlink", rel)
		}
		return copyFile(path, destination)
	})
}

func extractModelPackageZip(source string, target string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		name := filepath.FromSlash(strings.TrimSpace(file.Name))
		if name == "" {
			continue
		}
		if !safeModelPackageRelativePath(name) {
			return fmt.Errorf("unsafe OCR model package path %q", file.Name)
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("OCR model package path %q is a symlink", file.Name)
		}
		destination := filepath.Join(target, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeStreamToFile(src, destination, file.FileInfo().Mode().Perm())
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func installVerifiedModelDir(root string, source string, modelID string) error {
	target := filepath.Join(root, modelID)
	backup := filepath.Join(root, modelInstallBackupPrefix+modelID)
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	if fileExists(target) {
		return fmt.Errorf("OCR model target %q exists and is not a directory", target)
	}
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, target)
		}
		return err
	}
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	return nil
}

func safeModelPackageRelativePath(path string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." || filepath.IsAbs(path) {
		return false
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func copyFile(source string, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return writeStreamToFile(src, target, info.Mode().Perm())
}

func writeStreamToFile(reader io.Reader, target string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, reader)
	return err
}

func activeModelIDFromState(s *Service) string {
	state, err := s.LoadState()
	if err != nil {
		return defaultActiveModelID()
	}
	return state.ActiveModelID
}
