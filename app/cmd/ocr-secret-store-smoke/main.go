package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	secretstore "github.com/lemon-casino/RecordingFreedom/app/internal/secrets"
)

const (
	evidenceFileName = "secret-store-smoke.json"
	secretNamePrefix = "ocr.translation.api-key.smoke"
)

type smokeReport struct {
	SchemaVersion        int       `json:"schemaVersion"`
	OK                   bool      `json:"ok"`
	GeneratedAt          time.Time `json:"generatedAt"`
	GOOS                 string    `json:"goos"`
	GOARCH               string    `json:"goarch"`
	DataRoot             string    `json:"dataRoot"`
	EvidencePath         string    `json:"evidencePath"`
	SecretBackend        string    `json:"secretBackend"`
	SecretStatus         string    `json:"secretStatus"`
	SecretName           string    `json:"secretName"`
	Saved                bool      `json:"saved"`
	Loaded               bool      `json:"loaded"`
	Deleted              bool      `json:"deleted"`
	LoadAfterDeleteFound bool      `json:"loadAfterDeleteFound"`
	RawSecretInDataRoot  bool      `json:"rawSecretInDataRoot"`
	ScannedFiles         int       `json:"scannedFiles"`
}

func main() {
	var dataRoot string
	var evidenceDir string
	flag.StringVar(&dataRoot, "data-dir", "", "data root for the smoke run; defaults to a temp directory inside the evidence directory")
	flag.StringVar(&evidenceDir, "evidence-dir", filepath.Join("..", "release-out", "ocr-secret-store-smoke"), "directory for secret store smoke evidence")
	flag.Parse()

	report, err := run(dataRoot, evidenceDir)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode OCR secret store smoke report: %v\n", err)
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func run(dataRoot string, evidenceDir string) (smokeReport, error) {
	evidenceDir = strings.TrimSpace(evidenceDir)
	if evidenceDir == "" {
		return smokeReport{}, errors.New("evidence dir is required")
	}
	evidenceDir, err := filepath.Abs(evidenceDir)
	if err != nil {
		return smokeReport{}, err
	}
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return smokeReport{}, err
	}
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		dataRoot, err = os.MkdirTemp(evidenceDir, "data-root-*")
		if err != nil {
			return smokeReport{}, err
		}
	}

	data := appdata.NewService(dataRoot)
	info, err := data.Info()
	if err != nil {
		return smokeReport{}, fmt.Errorf("app data info: %w", err)
	}
	store := secretstore.NewStore(data)
	status, err := store.Status()
	if err != nil {
		return smokeReport{}, fmt.Errorf("secret store status: %w", err)
	}

	secretName := fmt.Sprintf("%s.%d", secretNamePrefix, time.Now().UnixNano())
	secretValue := "rf-secret-store-smoke-value"
	if err := store.Save(secretName, secretValue); err != nil {
		return smokeReport{}, fmt.Errorf("save secret: %w", err)
	}
	loaded, ok, err := store.Load(secretName)
	if err != nil {
		_ = store.Delete(secretName)
		return smokeReport{}, fmt.Errorf("load secret: %w", err)
	}
	if !ok || loaded != secretValue {
		_ = store.Delete(secretName)
		return smokeReport{}, errors.New("secret store did not return the saved value")
	}
	rawFound, scanned, err := scanForRawSecret(info.RootDir, secretValue)
	if err != nil {
		_ = store.Delete(secretName)
		return smokeReport{}, err
	}
	if rawFound {
		_ = store.Delete(secretName)
		return smokeReport{}, errors.New("data root contains raw secret value")
	}
	if err := store.Delete(secretName); err != nil {
		return smokeReport{}, fmt.Errorf("delete secret: %w", err)
	}
	afterDelete, afterDeleteFound, err := store.Load(secretName)
	if err != nil {
		return smokeReport{}, fmt.Errorf("load secret after delete: %w", err)
	}
	if afterDeleteFound || afterDelete != "" {
		return smokeReport{}, errors.New("secret store returned value after delete")
	}

	report := smokeReport{
		SchemaVersion:        1,
		OK:                   true,
		GeneratedAt:          time.Now().UTC(),
		GOOS:                 runtime.GOOS,
		GOARCH:               runtime.GOARCH,
		DataRoot:             info.RootDir,
		EvidencePath:         filepath.Join(evidenceDir, evidenceFileName),
		SecretBackend:        status.Backend,
		SecretStatus:         status.Dir,
		SecretName:           secretName,
		Saved:                true,
		Loaded:               true,
		Deleted:              true,
		LoadAfterDeleteFound: false,
		RawSecretInDataRoot:  false,
		ScannedFiles:         scanned,
	}
	if err := writeEvidence(report); err != nil {
		return smokeReport{}, err
	}
	return report, nil
}

func scanForRawSecret(root string, secret string) (bool, int, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false, 0, errors.New("secret value is required")
	}
	needle := []byte(secret)
	scanned := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > 4*1024*1024 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		if bytes.Contains(data, needle) {
			return errRawSecretFound
		}
		return nil
	})
	if errors.Is(err, errRawSecretFound) {
		return true, scanned, nil
	}
	if err != nil {
		return false, scanned, err
	}
	return false, scanned, nil
}

var errRawSecretFound = errors.New("raw secret found")

func writeEvidence(report smokeReport) error {
	if strings.TrimSpace(report.EvidencePath) == "" {
		return errors.New("evidence path is required")
	}
	if err := os.MkdirAll(filepath.Dir(report.EvidencePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(report.EvidencePath, append(data, '\n'), 0o644)
}
