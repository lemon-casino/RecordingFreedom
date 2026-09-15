package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/fsutil"
)

const secretsDir = "secrets"

type diskSecret struct {
	SchemaVersion int       `json:"schemaVersion"`
	Backend       string    `json:"backend"`
	Protected     string    `json:"protected"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func diskSave(s *Store, name string, secret string) error {
	dir, err := diskDir(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	protected, err := protect([]byte(secret))
	if err != nil {
		return err
	}
	payload := diskSecret{
		SchemaVersion: schemaVersion,
		Backend:       diskBackendName(),
		Protected:     base64.StdEncoding.EncodeToString(protected),
		UpdatedAt:     s.now().UTC(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path, err := diskPath(s, name)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o600)
}

func diskLoad(s *Store, name string) (string, bool, error) {
	path, err := diskPath(s, name)
	if err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	var payload diskSecret
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, err
	}
	if payload.SchemaVersion != schemaVersion || strings.TrimSpace(payload.Protected) == "" {
		return "", false, errors.New("secret file is invalid")
	}
	protected, err := base64.StdEncoding.DecodeString(payload.Protected)
	if err != nil {
		return "", false, err
	}
	plain, err := unprotect(protected)
	if err != nil {
		return "", false, err
	}
	secret := strings.TrimSpace(string(plain))
	return secret, secret != "", nil
}

func diskDelete(s *Store, name string) error {
	path, err := diskPath(s, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func diskPath(s *Store, name string) (string, error) {
	dir, err := diskDir(s)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, safeName(name)+fileSuffix), nil
}

func diskDir(s *Store) (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", secretsDir), nil
}
