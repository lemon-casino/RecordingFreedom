package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesPPOCRv6SourceAuditEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		repo := strings.TrimPrefix(request.URL.Path, "/api/models/")
		if repo == request.URL.Path {
			parts := strings.Split(strings.TrimPrefix(request.URL.Path, "/"), "/resolve/main/")
			if len(parts) != 2 {
				http.NotFound(writer, request)
				return
			}
			repo = parts[0]
			if parts[1] != "inference.yml" {
				http.NotFound(writer, request)
				return
			}
			if strings.Contains(repo, "_rec_") {
				fmt.Fprint(writer, "Global:\n  model_name: PP-OCRv6_test_rec\nPostProcess:\n  name: CTCLabelDecode\n  character_dict:\n  - A\n  - B\n  - 文\n")
				return
			}
			fmt.Fprint(writer, "Global:\n  model_name: PP-OCRv6_test_det\nPostProcess:\n  name: DBPostProcess\n")
			return
		}
		response := map[string]any{
			"id":           repo,
			"sha":          "test-" + strings.ReplaceAll(repo, "/", "-"),
			"pipeline_tag": "image-to-text",
			"library_name": "PaddleOCR",
			"usedStorage":  int64(1234),
			"cardData": map[string]any{
				"license": "apache-2.0",
			},
			"siblings": []map[string]string{
				{"rfilename": ".gitattributes"},
				{"rfilename": "README.md"},
				{"rfilename": "inference.onnx"},
				{"rfilename": "inference.yml"},
			},
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	evidenceDir := t.TempDir()
	if err := run([]string{"-hf-base", server.URL, "-evidence-dir", evidenceDir}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(evidenceDir, "ppocrv6-source-audit.json"))
	if err != nil {
		t.Fatalf("ReadFile(evidence) error = %v", err)
	}
	var report auditReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("Unmarshal(evidence) error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want false until cls/keys/smoke blockers are resolved")
	}
	if len(report.Candidates) != 2 {
		t.Fatalf("candidates = %d, want latest and quality", len(report.Candidates))
	}
	latest := report.Candidates[0]
	if latest.ID != "ppocrv6-mobile-zh-en" || latest.Channel != "latest" || latest.ProposedTier != "small" {
		t.Fatalf("latest candidate = %#v", latest)
	}
	if !latest.Compatibility.DetectedDetONNX || !latest.Compatibility.DetectedRecONNX {
		t.Fatalf("det/rec detection = %#v, want ONNX sources present", latest.Compatibility)
	}
	if latest.Compatibility.RecCharacterCount != 3 || latest.Compatibility.ExpectedRecClasses != 4 {
		t.Fatalf("rec character audit = %#v, want character count and CTC classes", latest.Compatibility)
	}
	latestRec := latest.Sources[1]
	expectedKeys := []byte("A\nB\n文\n")
	if latestRec.GeneratedKeys == nil || latestRec.GeneratedKeys.CharacterCount != 3 || latestRec.GeneratedKeys.Bytes != int64(len(expectedKeys)) || latestRec.GeneratedKeys.SHA256 != sha256Hex(expectedKeys) {
		t.Fatalf("generated keys audit = %#v, want bytes and SHA256 for generated keys.txt", latestRec.GeneratedKeys)
	}
	blockers := strings.Join(latest.Compatibility.Blockers, "\n")
	for _, want := range []string{"cls.onnx", "keys.txt", "worker smoke"} {
		if !strings.Contains(blockers, want) {
			t.Fatalf("blockers = %q, want %q", blockers, want)
		}
	}
}

func TestRunCanHashPresentPPOCRv6SourceFiles(t *testing.T) {
	filePayloads := map[string][]byte{
		"inference.onnx": []byte("onnx-bytes"),
		"inference.yml":  []byte("Global:\n  model_name: PP-OCRv6_test_rec\nPostProcess:\n  name: CTCLabelDecode\n  character_dict:\n  - A\n"),
		"README.md":      []byte("# model card\n"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		repo := strings.TrimPrefix(request.URL.Path, "/api/models/")
		if repo == request.URL.Path {
			parts := strings.Split(strings.TrimPrefix(request.URL.Path, "/"), "/resolve/main/")
			if len(parts) != 2 {
				http.NotFound(writer, request)
				return
			}
			data, ok := filePayloads[parts[1]]
			if !ok {
				http.NotFound(writer, request)
				return
			}
			_, _ = writer.Write(data)
			return
		}
		response := map[string]any{
			"id":           repo,
			"sha":          "test-" + strings.ReplaceAll(repo, "/", "-"),
			"pipeline_tag": "image-to-text",
			"library_name": "PaddleOCR",
			"usedStorage":  int64(1234),
			"cardData": map[string]any{
				"license": "apache-2.0",
			},
			"siblings": []map[string]string{
				{"rfilename": "README.md"},
				{"rfilename": "inference.onnx"},
				{"rfilename": "inference.yml"},
			},
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	evidenceDir := t.TempDir()
	if err := run([]string{"-hf-base", server.URL, "-evidence-dir", evidenceDir, "-hash-files"}); err != nil {
		t.Fatalf("run(-hash-files) error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(evidenceDir, "ppocrv6-source-audit.json"))
	if err != nil {
		t.Fatalf("ReadFile(evidence) error = %v", err)
	}
	var report auditReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("Unmarshal(evidence) error = %v", err)
	}
	latestDet := report.Candidates[0].Sources[0]
	if len(latestDet.FileHashes) != 3 {
		t.Fatalf("file hashes = %#v, want hashes for required present files", latestDet.FileHashes)
	}
	hash := findFileHash(t, latestDet.FileHashes, "inference.onnx")
	if hash.Bytes != int64(len(filePayloads["inference.onnx"])) || hash.SHA256 != sha256Hex(filePayloads["inference.onnx"]) || hash.Error != "" {
		t.Fatalf("inference.onnx hash = %#v, want bytes and SHA256", hash)
	}
}

func TestCountYAMLListItemsAfterKey(t *testing.T) {
	data := "PostProcess:\n  character_dict:\n  - A\n  - B\n  other: value\n"
	if got := countYAMLListItemsAfterKey(data, "character_dict"); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func findFileHash(t *testing.T, hashes []fileHashAudit, name string) fileHashAudit {
	t.Helper()
	for _, hash := range hashes {
		if hash.Name == name {
			return hash
		}
	}
	t.Fatalf("missing file hash for %s: %#v", name, hashes)
	return fileHashAudit{}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
