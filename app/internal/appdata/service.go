package appdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	appName                          = "RecordingFreedom"
	EnvDataDir                       = "RECORDINGFREEDOM_DATA_DIR"
	rootPointerFile                  = "data-root.json"
	MinimumRecommendedVideoFreeBytes = 1 << 30
	StorageStatusReady               = "ready"
	StorageStatusWarning             = "warning"
	StorageStatusBlocked             = "blocked"
)

type Info struct {
	RootDir  string `json:"rootDir"`
	VideoDir string `json:"videoDir"`
}

type StorageStatus struct {
	RootDir                 string `json:"rootDir"`
	VideoDir                string `json:"videoDir"`
	Writable                bool   `json:"writable"`
	FreeSpaceKnown          bool   `json:"freeSpaceKnown"`
	AvailableBytes          uint64 `json:"availableBytes"`
	MinimumRecommendedBytes uint64 `json:"minimumRecommendedBytes"`
	Status                  string `json:"status"`
	Reason                  string `json:"reason,omitempty"`
}

type Service struct {
	mu             sync.RWMutex
	rootDir        string
	pointerBaseDir string
}

type rootPointer struct {
	SchemaVersion int       `json:"schemaVersion"`
	RootDir       string    `json:"rootDir"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

var executablePath = os.Executable

func NewService(rootDir string) *Service {
	return &Service{rootDir: rootDir}
}

func NewServiceWithPointerBase(rootDir string, pointerBaseDir string) *Service {
	return &Service{rootDir: rootDir, pointerBaseDir: pointerBaseDir}
}

func (s *Service) Info() (Info, error) {
	root, err := s.RootDir()
	if err != nil {
		return Info{}, err
	}
	video, err := s.VideoDir()
	if err != nil {
		return Info{}, err
	}
	return Info{RootDir: root, VideoDir: video}, nil
}

func (s *Service) StorageStatus() (StorageStatus, error) {
	root, err := s.RootDir()
	if err != nil {
		return blockedStorageStatus("", "", err.Error()), err
	}
	video := filepath.Join(root, "data", "video")
	status := StorageStatus{
		RootDir:                 root,
		VideoDir:                video,
		MinimumRecommendedBytes: MinimumRecommendedVideoFreeBytes,
		Status:                  StorageStatusReady,
	}
	if err := os.MkdirAll(video, 0o755); err != nil {
		status.Status = StorageStatusBlocked
		status.Reason = fmt.Sprintf("cannot create data/video: %v", err)
		return status, nil
	}
	if err := probeWritable(video); err != nil {
		status.Status = StorageStatusBlocked
		status.Reason = err.Error()
		return status, nil
	}
	status.Writable = true

	available, err := availableBytes(video)
	if err != nil {
		status.Status = StorageStatusWarning
		status.Reason = fmt.Sprintf("could not read free space for data/video: %v", err)
		return status, nil
	}
	status.FreeSpaceKnown = true
	status.AvailableBytes = available
	if available < MinimumRecommendedVideoFreeBytes {
		status.Status = StorageStatusWarning
		status.Reason = fmt.Sprintf("available space is below the recommended %d bytes for long recordings", MinimumRecommendedVideoFreeBytes)
	}
	return status, nil
}

func (s *Service) RootDir() (string, error) {
	s.mu.RLock()
	root := s.rootDir
	pointerBaseDir := s.pointerBaseDir
	s.mu.RUnlock()

	if root == "" {
		root = os.Getenv(EnvDataDir)
	}
	if root == "" {
		if pointedRoot, err := readRootPointer(pointerBaseDir); err == nil {
			root = pointedRoot
		}
	}
	if root == "" {
		root = defaultRootDir()
	}
	return ensureRoot(root)
}

func (s *Service) SetRootDir(nextRoot string) (Info, error) {
	if os.Getenv(EnvDataDir) != "" {
		return Info{}, fmt.Errorf("%s is set; unset it before changing the RecordingFreedom data root", EnvDataDir)
	}
	root, err := ensureRoot(nextRoot)
	if err != nil {
		return Info{}, err
	}
	if err := writeRootPointer(s.pointerBaseDir, root); err != nil {
		return Info{}, err
	}
	s.mu.Lock()
	s.rootDir = root
	s.mu.Unlock()
	return s.Info()
}

func ensureRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("data root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	if err := probeWritable(root); err != nil {
		return "", err
	}
	return root, nil
}

func blockedStorageStatus(root string, video string, reason string) StorageStatus {
	return StorageStatus{
		RootDir:                 root,
		VideoDir:                video,
		MinimumRecommendedBytes: MinimumRecommendedVideoFreeBytes,
		Status:                  StorageStatusBlocked,
		Reason:                  reason,
	}
}

func (s *Service) VideoDir() (string, error) {
	root, err := s.RootDir()
	if err != nil {
		return "", err
	}
	video := filepath.Join(root, "data", "video")
	if err := os.MkdirAll(video, 0o755); err != nil {
		return "", err
	}
	return video, nil
}

func readRootPointer(pointerBaseDir string) (string, error) {
	data, err := os.ReadFile(rootPointerPath(pointerBaseDir))
	if err != nil {
		return "", err
	}
	var pointer rootPointer
	if err := json.Unmarshal(data, &pointer); err != nil {
		return "", err
	}
	if pointer.SchemaVersion != 1 || strings.TrimSpace(pointer.RootDir) == "" {
		return "", errors.New("invalid data root pointer")
	}
	return pointer.RootDir, nil
}

func writeRootPointer(pointerBaseDir string, root string) error {
	pointerDir, err := pointerBase(pointerBaseDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pointerDir, 0o755); err != nil {
		return err
	}
	defaultRoot, err := filepath.Abs(defaultRootForPointer(pointerBaseDir))
	if err != nil {
		return err
	}
	if root == defaultRoot {
		if err := os.Remove(rootPointerPath(pointerBaseDir)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(rootPointer{
		SchemaVersion: 1,
		RootDir:       root,
		UpdatedAt:     time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rootPointerPath(pointerBaseDir), append(data, '\n'), 0o644)
}

func rootPointerPath(pointerBaseDir string) string {
	return filepath.Join(defaultRootForPointer(pointerBaseDir), rootPointerFile)
}

func pointerBase(pointerBaseDir string) (string, error) {
	return ensureRoot(defaultRootForPointer(pointerBaseDir))
}

func defaultRootForPointer(pointerBaseDir string) string {
	if pointerBaseDir != "" {
		return pointerBaseDir
	}
	return defaultRootDir()
}

func probeWritable(root string) error {
	probe, err := os.CreateTemp(root, ".recordingfreedom-write-test-*")
	if err != nil {
		return fmt.Errorf("data root %q is not writable: %w", root, err)
	}
	probePath := probe.Name()
	if _, err := probe.Write([]byte("ok\n")); err != nil {
		_ = probe.Close()
		_ = os.Remove(probePath)
		return fmt.Errorf("data root %q is not writable: %w", root, err)
	}
	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return fmt.Errorf("data root %q is not writable: %w", root, err)
	}
	if err := os.Remove(probePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func defaultRootDir() string {
	if executable, err := executablePath(); err == nil {
		if dir := strings.TrimSpace(filepath.Dir(executable)); dir != "" && dir != "." {
			return dir
		}
	}
	if workingDir, err := os.Getwd(); err == nil && strings.TrimSpace(workingDir) != "" {
		return workingDir
	}
	return filepath.Join(".", appName)
}
