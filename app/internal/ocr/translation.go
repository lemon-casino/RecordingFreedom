package ocr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	translationProviderDisabled         = "disabled"
	translationProviderDeepL            = "deepl"
	translationProviderOpenAICompatible = "openai-compatible"
	translationPromptVersion            = "ocr-translation-v1"
	defaultTranslationTimeout           = 30 * time.Second
)

type translationSegment struct {
	BlockID string
	Text    string
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

type openAITranslatedBlock struct {
	BlockID    string `json:"blockId"`
	Translated string `json:"translated"`
}

type deepLResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

func (s *Service) Translate(req TranslateRequest) (TranslationResult, error) {
	req = normalizeTranslateRequest(req)
	if strings.TrimSpace(req.OcrResultID) == "" {
		return TranslationResult{}, errors.New("OCR result id is required")
	}
	if req.Provider == "" || req.Provider == translationProviderDisabled {
		return TranslationResult{}, errors.New("OCR translation provider is disabled")
	}
	result, err := s.ReadResult(req.OcrResultID)
	if err != nil {
		return TranslationResult{}, err
	}
	segments, err := translationSegments(result, req.BlockIDs)
	if err != nil {
		return TranslationResult{}, err
	}
	if len(segments) == 0 {
		return TranslationResult{}, errors.New("OCR result has no text blocks to translate")
	}
	if !req.Force {
		cached, ok, err := s.readCachedTranslation(req, segments)
		if err != nil {
			return TranslationResult{}, err
		}
		if ok {
			return cached, nil
		}
	}

	var translated []TranslationBlock
	switch req.Provider {
	case translationProviderDeepL:
		translated, err = s.translateWithDeepL(req, segments)
	case translationProviderOpenAICompatible:
		translated, err = s.translateWithOpenAICompatible(req, segments)
	default:
		err = fmt.Errorf("unsupported OCR translation provider %q", req.Provider)
	}
	if err != nil {
		return TranslationResult{}, err
	}
	if err := validateTranslationBlocks(segments, translated); err != nil {
		return TranslationResult{}, err
	}
	output := TranslationResult{
		OcrResultID:    req.OcrResultID,
		Provider:       req.Provider,
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Model:          req.Model,
		PromptVersion:  translationPromptVersion,
		Blocks:         translated,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.writeTranslation(req, segments, output); err != nil {
		return TranslationResult{}, err
	}
	return output, nil
}

func normalizeTranslateRequest(req TranslateRequest) TranslateRequest {
	req.OcrResultID = strings.TrimSpace(req.OcrResultID)
	req.Provider = strings.TrimSpace(req.Provider)
	req.SourceLanguage = strings.TrimSpace(req.SourceLanguage)
	req.TargetLanguage = strings.TrimSpace(req.TargetLanguage)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.APIKey = strings.TrimSpace(req.APIKey)
	req.Model = strings.TrimSpace(req.Model)
	if req.Provider == "" {
		req.Provider = translationProviderDisabled
	}
	if req.SourceLanguage == "" {
		req.SourceLanguage = "auto"
	}
	if req.TargetLanguage == "" {
		req.TargetLanguage = "zh-CN"
	}
	return req
}

func translationSegments(result Result, blockIDs []string) ([]translationSegment, error) {
	wanted := map[string]bool{}
	for _, id := range blockIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = true
		}
	}
	segments := make([]translationSegment, 0, len(result.Blocks))
	for _, block := range result.Blocks {
		if len(wanted) > 0 && !wanted[block.ID] {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		segments = append(segments, translationSegment{BlockID: block.ID, Text: text})
	}
	if len(wanted) > 0 && len(segments) != len(wanted) {
		return nil, errors.New("one or more OCR block ids were not found")
	}
	return segments, nil
}

func (s *Service) translateWithDeepL(req TranslateRequest, segments []translationSegment) ([]TranslationBlock, error) {
	if req.BaseURL == "" {
		return nil, errors.New("DeepL translation base URL is required")
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return nil, errors.New("DeepL API key is required")
	}
	form := url.Values{}
	form.Set("auth_key", req.APIKey)
	form.Set("target_lang", deeplLanguage(req.TargetLanguage))
	if req.SourceLanguage != "" && req.SourceLanguage != "auto" {
		form.Set("source_lang", deeplLanguage(req.SourceLanguage))
	}
	for _, segment := range segments {
		form.Add("text", segment.Text)
	}
	httpReq, err := http.NewRequest(http.MethodPost, req.BaseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseBody, err := doTranslationHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}
	var response deepLResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("DeepL translation response is invalid: %w", err)
	}
	if len(response.Translations) != len(segments) {
		return nil, fmt.Errorf("DeepL translation returned %d segments for %d OCR blocks", len(response.Translations), len(segments))
	}
	blocks := make([]TranslationBlock, 0, len(segments))
	for index, segment := range segments {
		blocks = append(blocks, TranslationBlock{
			BlockID:    segment.BlockID,
			Source:     segment.Text,
			Translated: response.Translations[index].Text,
		})
	}
	return blocks, nil
}

func (s *Service) translateWithOpenAICompatible(req TranslateRequest, segments []translationSegment) ([]TranslationBlock, error) {
	if req.BaseURL == "" {
		return nil, errors.New("OpenAI-compatible translation base URL is required")
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return nil, errors.New("OpenAI-compatible API key is required")
	}
	if req.Model == "" {
		return nil, errors.New("OpenAI-compatible translation model is required")
	}
	payload, err := json.Marshal(openAIChatRequest{
		Model: req.Model,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "Translate OCR text blocks. Return only a JSON array of objects with blockId and translated fields. Keep the same block ids and order.",
			},
			{
				Role:    "user",
				Content: openAITranslationPrompt(req, segments),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, openAIChatCompletionsURL(req.BaseURL), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	responseBody, err := doTranslationHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}
	var response openAIChatResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("OpenAI-compatible translation response is invalid: %w", err)
	}
	if len(response.Choices) == 0 {
		return nil, errors.New("OpenAI-compatible translation returned no choices")
	}
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	var translated []openAITranslatedBlock
	if err := json.Unmarshal([]byte(content), &translated); err != nil {
		return nil, fmt.Errorf("OpenAI-compatible translation content is not valid block JSON: %w", err)
	}
	blocks := make([]TranslationBlock, 0, len(translated))
	sourceByID := map[string]string{}
	for _, segment := range segments {
		sourceByID[segment.BlockID] = segment.Text
	}
	for _, block := range translated {
		blocks = append(blocks, TranslationBlock{
			BlockID:    block.BlockID,
			Source:     sourceByID[block.BlockID],
			Translated: block.Translated,
		})
	}
	return blocks, nil
}

func doTranslationHTTPRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: defaultTranslationTimeout}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("translation provider returned HTTP %d", response.StatusCode)
	}
	return body, nil
}

func validateTranslationBlocks(segments []translationSegment, blocks []TranslationBlock) error {
	if len(blocks) != len(segments) {
		return fmt.Errorf("translation returned %d segments for %d OCR blocks", len(blocks), len(segments))
	}
	for index, segment := range segments {
		block := blocks[index]
		if block.BlockID != segment.BlockID {
			return fmt.Errorf("translation block %d id = %q, want %q", index, block.BlockID, segment.BlockID)
		}
		if strings.TrimSpace(block.Translated) == "" {
			return fmt.Errorf("translation block %q is empty", block.BlockID)
		}
	}
	return nil
}

func (s *Service) readCachedTranslation(req TranslateRequest, segments []translationSegment) (TranslationResult, bool, error) {
	path, err := s.translationCachePath(req, segments)
	if err != nil {
		return TranslationResult{}, false, err
	}
	data, err := osReadFileIfExists(path)
	if err != nil || data == nil {
		return TranslationResult{}, false, err
	}
	var result TranslationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return TranslationResult{}, false, err
	}
	return result, true, nil
}

func (s *Service) writeTranslation(req TranslateRequest, segments []translationSegment, result TranslationResult) error {
	path, err := s.translationCachePath(req, segments)
	if err != nil {
		return err
	}
	return writeJSONFile(path, result)
}

func (s *Service) translationCachePath(req TranslateRequest, segments []translationSegment) (string, error) {
	dir, err := s.translationsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, translationCacheFileName(req, segments)), nil
}

func translationCacheFileName(req TranslateRequest, segments []translationSegment) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(req.OcrResultID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(req.Provider))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(req.BaseURL))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(req.SourceLanguage))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(req.TargetLanguage))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(req.Model))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(translationPromptVersion))
	for _, segment := range segments {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(segment.BlockID))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(segment.Text))
	}
	return safeCachePart(req.OcrResultID) + "." + safeCachePart(req.Provider) + "." + safeCachePart(req.TargetLanguage) + "." + hex.EncodeToString(hash.Sum(nil)) + ".json"
}

func osReadFileIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func openAITranslationPrompt(req TranslateRequest, segments []translationSegment) string {
	type promptBlock struct {
		BlockID string `json:"blockId"`
		Text    string `json:"text"`
	}
	blocks := make([]promptBlock, 0, len(segments))
	for _, segment := range segments {
		blocks = append(blocks, promptBlock{BlockID: segment.BlockID, Text: segment.Text})
	}
	payload := struct {
		SourceLanguage string        `json:"sourceLanguage"`
		TargetLanguage string        `json:"targetLanguage"`
		Blocks         []promptBlock `json:"blocks"`
	}{
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Blocks:         blocks,
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func openAIChatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func deeplLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" || language == "auto" {
		return ""
	}
	switch language {
	case "zh", "zh-cn", "zh-hans", "cn":
		return "ZH"
	}
	return strings.ToUpper(strings.ReplaceAll(language, "_", "-"))
}
