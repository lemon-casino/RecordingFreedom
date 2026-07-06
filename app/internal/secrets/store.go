package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
)

const (
	schemaVersion = 1
	fileSuffix    = ".secret.json"
)

type Store struct {
	appData *appdata.Service
	now     func() time.Time
}

type Status struct {
	Backend string `json:"backend"`
	Dir     string `json:"dir"`
}

func NewStore(appData *appdata.Service) *Store {
	return &Store{appData: appData, now: time.Now}
}

func (s *Store) Status() (Status, error) {
	return backendStatus(s)
}

func (s *Store) Save(name string, secret string) error {
	name = strings.TrimSpace(name)
	secret = strings.TrimSpace(secret)
	if name == "" {
		return errors.New("secret name is required")
	}
	if secret == "" {
		return s.Delete(name)
	}
	return backendSave(s, name, secret)
}

func (s *Store) Load(name string) (string, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, errors.New("secret name is required")
	}
	return backendLoad(s, name)
}

func (s *Store) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("secret name is required")
	}
	return backendDelete(s, name)
}

func (s *Store) rootDir() (string, error) {
	if s == nil || s.appData == nil {
		return "", errors.New("secret store app data is not initialized")
	}
	return s.appData.RootDir()
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		}
	}
	clean := strings.Trim(builder.String(), ".-_")
	if clean != "" && clean == name {
		return clean
	}
	sum := sha256.Sum256([]byte(name))
	if clean == "" {
		clean = "secret"
	}
	if len(clean) > 40 {
		clean = clean[:40]
	}
	return clean + "-" + hex.EncodeToString(sum[:8])
}
