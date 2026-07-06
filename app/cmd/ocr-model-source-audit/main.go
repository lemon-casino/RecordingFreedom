package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultHFBase      = "https://huggingface.co"
	defaultEvidenceDir = "../release-out/ocr-model-source-audit"
)

type auditReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	OK            bool             `json:"ok"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	HFBase        string           `json:"hfBase"`
	EvidencePath  string           `json:"evidencePath"`
	Candidates    []candidateAudit `json:"candidates"`
}

type candidateAudit struct {
	ID               string             `json:"id"`
	Channel          string             `json:"channel"`
	ProposedTier     string             `json:"proposedTier"`
	ReadyForManifest bool               `json:"readyForManifest"`
	Sources          []sourceAudit      `json:"sources"`
	Compatibility    compatibilityAudit `json:"compatibility"`
}

type sourceAudit struct {
	Kind           string              `json:"kind"`
	Repository     string              `json:"repository"`
	URL            string              `json:"url"`
	Commit         string              `json:"commit,omitempty"`
	License        string              `json:"license,omitempty"`
	PipelineTag    string              `json:"pipelineTag,omitempty"`
	LibraryName    string              `json:"libraryName,omitempty"`
	UsedStorage    int64               `json:"usedStorage,omitempty"`
	RequiredFiles  []string            `json:"requiredFiles"`
	PresentFiles   []string            `json:"presentFiles"`
	MissingFiles   []string            `json:"missingFiles,omitempty"`
	FileHashes     []fileHashAudit     `json:"fileHashes,omitempty"`
	GeneratedKeys  *generatedKeysAudit `json:"generatedKeys,omitempty"`
	ModelName      string              `json:"modelName,omitempty"`
	PostProcess    string              `json:"postProcess,omitempty"`
	CharacterCount int                 `json:"characterCount,omitempty"`
	Error          string              `json:"error,omitempty"`
}

type fileHashAudit struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Bytes  int64  `json:"bytes,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Error  string `json:"error,omitempty"`
}

type generatedKeysAudit struct {
	Source         string `json:"source"`
	Type           string `json:"type"`
	CharacterCount int    `json:"characterCount"`
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
}

type compatibilityAudit struct {
	WorkerRequires          []string `json:"workerRequires"`
	DetectedDetONNX         bool     `json:"detectedDetOnnx"`
	DetectedRecONNX         bool     `json:"detectedRecOnnx"`
	DetectedOfficialClsONNX bool     `json:"detectedOfficialClsOnnx"`
	RecCharacterCount       int      `json:"recCharacterCount,omitempty"`
	ExpectedRecClasses      int      `json:"expectedRecClasses,omitempty"`
	ReadyForManifest        bool     `json:"readyForManifest"`
	Blockers                []string `json:"blockers,omitempty"`
}

type hfModel struct {
	ID          string `json:"id"`
	SHA         string `json:"sha"`
	PipelineTag string `json:"pipeline_tag"`
	LibraryName string `json:"library_name"`
	UsedStorage int64  `json:"usedStorage"`
	CardData    struct {
		License string `json:"license"`
	} `json:"cardData"`
	Siblings []struct {
		RFilename string `json:"rfilename"`
	} `json:"siblings"`
}

type candidateSpec struct {
	ID           string
	Channel      string
	ProposedTier string
	DetRepo      string
	RecRepo      string
}

var candidates = []candidateSpec{
	{
		ID:           "ppocrv6-mobile-zh-en",
		Channel:      "latest",
		ProposedTier: "small",
		DetRepo:      "PaddlePaddle/PP-OCRv6_small_det_onnx",
		RecRepo:      "PaddlePaddle/PP-OCRv6_small_rec_onnx",
	},
	{
		ID:           "ppocrv6-medium-zh-en",
		Channel:      "quality",
		ProposedTier: "medium",
		DetRepo:      "PaddlePaddle/PP-OCRv6_medium_det_onnx",
		RecRepo:      "PaddlePaddle/PP-OCRv6_medium_rec_onnx",
	},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var evidenceDir string
	var hfBase string
	var timeout time.Duration
	var hashFiles bool
	flagSet := flag.NewFlagSet("ocr-model-source-audit", flag.ContinueOnError)
	flagSet.StringVar(&evidenceDir, "evidence-dir", defaultEvidenceDir, "directory for OCR model source audit evidence")
	flagSet.StringVar(&hfBase, "hf-base", defaultHFBase, "Hugging Face base URL")
	flagSet.DurationVar(&timeout, "timeout", 20*time.Second, "HTTP timeout per request")
	flagSet.BoolVar(&hashFiles, "hash-files", false, "download present PP-OCRv6 required files and record bytes/SHA256 evidence")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	evidenceDir, err := filepath.Abs(evidenceDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return err
	}
	evidencePath := filepath.Join(evidenceDir, "ppocrv6-source-audit.json")
	client := &http.Client{Timeout: timeout}
	hfBase = strings.TrimRight(strings.TrimSpace(hfBase), "/")
	if hfBase == "" {
		return errors.New("-hf-base is required")
	}

	report := auditReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC(),
		HFBase:        hfBase,
		EvidencePath:  evidencePath,
	}
	ctx := context.Background()
	for _, spec := range candidates {
		report.Candidates = append(report.Candidates, auditCandidate(ctx, client, hfBase, spec, hashFiles))
	}
	report.OK = true
	for _, candidate := range report.Candidates {
		if !candidate.ReadyForManifest {
			report.OK = false
			break
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(evidencePath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func auditCandidate(ctx context.Context, client *http.Client, hfBase string, spec candidateSpec, hashFiles bool) candidateAudit {
	det := auditSource(ctx, client, hfBase, "det", spec.DetRepo, []string{"inference.onnx", "inference.yml", "README.md"}, hashFiles)
	rec := auditSource(ctx, client, hfBase, "rec", spec.RecRepo, []string{"inference.onnx", "inference.yml", "README.md"}, hashFiles)
	compat := compatibilityAudit{
		WorkerRequires:    []string{"det.onnx", "rec.onnx", "keys.txt", "smoke.png", "smoke.expected.json", "cls.onnx when textlineOrientation.mode=cls"},
		DetectedDetONNX:   hasPresent(det, "inference.onnx"),
		DetectedRecONNX:   hasPresent(rec, "inference.onnx"),
		RecCharacterCount: rec.CharacterCount,
	}
	if compat.RecCharacterCount > 0 {
		compat.ExpectedRecClasses = compat.RecCharacterCount + 1
	}
	if !compat.DetectedDetONNX {
		compat.Blockers = append(compat.Blockers, "official detection repository does not expose inference.onnx")
	}
	if !compat.DetectedRecONNX {
		compat.Blockers = append(compat.Blockers, "official recognition repository does not expose inference.onnx")
	}
	compat.Blockers = append(compat.Blockers,
		"PP-OCRv6 det/rec ONNX repositories do not provide an official cls.onnx; the RecordingFreedom manifest must either set textlineOrientation.mode=none and pass worker smoke, or pin a verified compatible cls.onnx",
		"PP-OCRv6 recognition repositories embed character_dict in inference.yml; generated keys.txt bytes/SHA256 must be pinned in the RecordingFreedom manifest before packaging",
		"PP-OCRv6 det/rec preprocessing, postprocessing thresholds, and no-orientation recognition mode differ from PP-OCRv5 and must pass RecordingFreedom worker smoke before manifest publication",
	)
	compat.ReadyForManifest = len(compat.Blockers) == 0
	return candidateAudit{
		ID:               spec.ID,
		Channel:          spec.Channel,
		ProposedTier:     spec.ProposedTier,
		ReadyForManifest: compat.ReadyForManifest,
		Sources:          []sourceAudit{det, rec},
		Compatibility:    compat,
	}
}

func auditSource(ctx context.Context, client *http.Client, hfBase string, kind string, repo string, required []string, hashFiles bool) sourceAudit {
	result := sourceAudit{
		Kind:          kind,
		Repository:    repo,
		URL:           hfBase + "/" + repo,
		RequiredFiles: append([]string(nil), required...),
	}
	model, err := fetchHFModel(ctx, client, hfBase, repo)
	if err != nil {
		result.Error = err.Error()
		result.MissingFiles = append([]string(nil), required...)
		return result
	}
	result.Commit = model.SHA
	result.License = model.CardData.License
	result.PipelineTag = model.PipelineTag
	result.LibraryName = model.LibraryName
	result.UsedStorage = model.UsedStorage
	presentSet := map[string]bool{}
	for _, sibling := range model.Siblings {
		presentSet[sibling.RFilename] = true
	}
	for _, name := range required {
		if presentSet[name] {
			result.PresentFiles = append(result.PresentFiles, name)
		} else {
			result.MissingFiles = append(result.MissingFiles, name)
		}
	}
	sort.Strings(result.PresentFiles)
	sort.Strings(result.MissingFiles)
	if hashFiles {
		for _, name := range required {
			if !presentSet[name] {
				continue
			}
			url := hfBase + "/" + repo + "/resolve/main/" + name
			fileHash := fileHashAudit{Name: name, URL: url}
			bytes, sha, err := hashRemoteFile(ctx, client, url)
			if err != nil {
				fileHash.Error = err.Error()
			} else {
				fileHash.Bytes = bytes
				fileHash.SHA256 = sha
			}
			result.FileHashes = append(result.FileHashes, fileHash)
		}
	}

	yml, err := fetchText(ctx, client, hfBase+"/"+repo+"/resolve/main/inference.yml")
	if err != nil {
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result
	}
	result.ModelName = firstYAMLScalar(yml, "model_name")
	result.PostProcess = firstYAMLScalar(yml, "name")
	characters, dictErr := extractPaddleOCRCharacterDict([]byte(yml))
	if dictErr != nil {
		if kind == "rec" && result.Error == "" {
			result.Error = dictErr.Error()
		}
		return result
	}
	result.CharacterCount = len(characters)
	keys := []byte(strings.Join(characters, "\n") + "\n")
	sum := sha256.Sum256(keys)
	result.GeneratedKeys = &generatedKeysAudit{
		Source:         "inference.yml",
		Type:           "paddleocr-character-dict-keys",
		CharacterCount: len(characters),
		Bytes:          int64(len(keys)),
		SHA256:         hex.EncodeToString(sum[:]),
	}
	return result
}

func fetchHFModel(ctx context.Context, client *http.Client, hfBase string, repo string) (hfModel, error) {
	var model hfModel
	data, err := fetchBytes(ctx, client, hfBase+"/api/models/"+repo)
	if err != nil {
		return model, err
	}
	if err := json.Unmarshal(data, &model); err != nil {
		return model, err
	}
	return model, nil
}

func fetchText(ctx context.Context, client *http.Client, url string) (string, error) {
	data, err := fetchBytes(ctx, client, url)
	return string(data), err
}

func hashRemoteFile(ctx context.Context, client *http.Client, url string) (int64, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-source-audit/1")
	response, err := client.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, "", fmt.Errorf("GET %s failed with HTTP %s", url, response.Status)
	}
	hash := sha256.New()
	bytes, err := io.Copy(hash, response.Body)
	if err != nil {
		return 0, "", err
	}
	return bytes, hex.EncodeToString(hash.Sum(nil)), nil
}

func fetchBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "RecordingFreedom-ocr-model-source-audit/1")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s failed with HTTP %s", url, response.Status)
	}
	return io.ReadAll(io.LimitReader(response.Body, 2<<20))
}

func hasPresent(source sourceAudit, file string) bool {
	for _, present := range source.PresentFiles {
		if present == file {
			return true
		}
	}
	return false
}

func firstYAMLScalar(data string, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), `"'`)
		}
	}
	return ""
}

func countYAMLListItemsAfterKey(data string, key string) int {
	lines := strings.Split(data, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != key+":" {
			continue
		}
		keyIndent := leadingSpaces(line)
		count := 0
		for _, next := range lines[index+1:] {
			if strings.TrimSpace(next) == "" {
				continue
			}
			indent := leadingSpaces(next)
			trimmed := strings.TrimSpace(next)
			if strings.HasPrefix(trimmed, "- ") && indent >= keyIndent {
				count++
				continue
			}
			if indent <= keyIndent {
				break
			}
			if strings.HasPrefix(trimmed, "- ") {
				count++
			}
		}
		return count
	}
	return 0
}

func extractPaddleOCRCharacterDict(data []byte) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse PaddleOCR inference.yml: %w", err)
	}
	postProcess := yamlMappingValue(&root, "PostProcess")
	if postProcess == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess")
	}
	dict := yamlMappingValue(postProcess, "character_dict")
	if dict == nil {
		return nil, errors.New("PaddleOCR inference.yml missing PostProcess.character_dict")
	}
	if dict.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("PaddleOCR PostProcess.character_dict kind = %v, want sequence", dict.Kind)
	}
	characters := make([]string, 0, len(dict.Content))
	for _, item := range dict.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("PaddleOCR character_dict item kind = %v, want scalar", item.Kind)
		}
		characters = append(characters, item.Value)
	}
	return characters, nil
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return yamlMappingValue(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}
