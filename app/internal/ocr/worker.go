package ocr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	workerMethodInit           = "init"
	workerMethodRecognize      = "recognize"
	workerMethodRelease        = "release"
	defaultWorkerTimeout       = 120 * time.Second
	defaultCapabilitiesTimeout = 5 * time.Second
	defaultWorkerMaxSide       = 2400
)

type workerRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type workerInitParams struct {
	ModelDir   string `json:"modelDir"`
	RuntimeDir string `json:"runtimeDir"`
	Threads    int    `json:"threads"`
	Language   string `json:"language"`
}

type workerRecognizeParams struct {
	ImagePath   string     `json:"imagePath"`
	ImageSHA256 string     `json:"imageSha256"`
	ModelID     string     `json:"modelId"`
	Language    string     `json:"language"`
	SourceKind  SourceKind `json:"sourceKind"`
	SourceID    string     `json:"sourceId"`
	DetectAngle bool       `json:"detectAngle"`
	MaxSide     int        `json:"maxSide"`
}

type workerResponse struct {
	ID     string       `json:"id"`
	OK     bool         `json:"ok"`
	Result *Result      `json:"result,omitempty"`
	Error  *workerError `json:"error,omitempty"`
}

type workerError struct {
	Code        string `json:"code,omitempty"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable,omitempty"`
}

func (s *Service) requireRecognizeWorker() (WorkerCapabilities, error) {
	workerPath := s.workerPath()
	if strings.TrimSpace(workerPath) == "" {
		return WorkerCapabilities{}, errors.New("OCR worker path is empty")
	}
	if !fileExists(workerPath) {
		return WorkerCapabilities{}, fmt.Errorf("OCR worker is not available at %s", workerPath)
	}
	capabilities, err := s.queryWorkerCapabilities()
	if err != nil {
		return WorkerCapabilities{}, err
	}
	if !capabilities.SupportsRecognize {
		message := strings.TrimSpace(capabilities.Message)
		if message == "" {
			message = "OCR worker does not support image recognition"
		}
		return capabilities, errors.New(message)
	}
	return capabilities, nil
}

func (s *Service) queryWorkerCapabilities() (WorkerCapabilities, error) {
	workerPath := s.workerPath()
	if strings.TrimSpace(workerPath) == "" {
		return WorkerCapabilities{}, errors.New("OCR worker path is empty")
	}
	if !fileExists(workerPath) {
		return WorkerCapabilities{}, fmt.Errorf("OCR worker is not available at %s", workerPath)
	}
	args := []string{"--capabilities", "--runtime-dir", s.runtimeDir()}
	if len(s.workerCapabilitiesArgs) > 0 {
		args = append([]string(nil), s.workerCapabilitiesArgs...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultCapabilitiesTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, workerPath, args...)
	configureBackgroundCommand(cmd)
	cmd.Env = append(os.Environ(), s.workerEnv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return WorkerCapabilities{}, fmt.Errorf("OCR worker capability probe timed out after %s", defaultCapabilitiesTimeout)
	}
	if err != nil {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = strings.TrimSpace(stdout.String())
		}
		if text == "" {
			return WorkerCapabilities{}, err
		}
		return WorkerCapabilities{}, fmt.Errorf("%w: %s", err, text)
	}
	var capabilities WorkerCapabilities
	if err := json.Unmarshal(stdout.Bytes(), &capabilities); err != nil {
		return WorkerCapabilities{}, fmt.Errorf("OCR worker returned invalid capabilities JSON: %w", err)
	}
	if capabilities.SchemaVersion != 1 {
		return WorkerCapabilities{}, fmt.Errorf("unsupported OCR worker capabilities schema %d", capabilities.SchemaVersion)
	}
	if strings.TrimSpace(capabilities.ProtocolVersion) == "" {
		return WorkerCapabilities{}, errors.New("OCR worker capabilities missing protocolVersion")
	}
	return capabilities, nil
}

func (s *Service) runWorkerRecognize(req RecognizeRequest, imageSHA256 string, model ModelInfo) (Result, error) {
	workerPath := s.workerPath()
	if strings.TrimSpace(workerPath) == "" {
		return Result{}, errors.New("OCR worker path is empty")
	}
	timeout := s.workerTimeout
	if timeout <= 0 {
		timeout = defaultWorkerTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, workerPath, s.workerArgs...)
	configureBackgroundCommand(cmd)
	cmd.Env = append(os.Environ(), s.workerEnv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Result{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}
	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(bufio.NewReader(stdout))

	if err := sendWorkerRequest(encoder, decoder, workerRequest{
		ID:     "init-" + requestIDNow(),
		Method: workerMethodInit,
		Params: workerInitParams{
			ModelDir:   model.ModelDir,
			RuntimeDir: s.runtimeDir(),
			Threads:    0,
			Language:   req.Language,
		},
	}); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return Result{}, appendWorkerStderr(err, stderr.String())
	}

	recognizeID := "recognize-" + requestIDNow()
	response, err := exchangeWorkerRequest(encoder, decoder, workerRequest{
		ID:     recognizeID,
		Method: workerMethodRecognize,
		Params: workerRecognizeParams{
			ImagePath:   req.ImagePath,
			ImageSHA256: imageSHA256,
			ModelID:     model.ID,
			Language:    req.Language,
			SourceKind:  req.SourceKind,
			SourceID:    req.SourceID,
			DetectAngle: true,
			MaxSide:     defaultWorkerMaxSide,
		},
	})
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return Result{}, appendWorkerStderr(err, stderr.String())
	}
	_ = sendWorkerRequest(encoder, decoder, workerRequest{
		ID:     "release-" + requestIDNow(),
		Method: workerMethodRelease,
	})
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return Result{}, fmt.Errorf("OCR worker timed out after %s", timeout)
	}
	if waitErr != nil {
		return Result{}, appendWorkerStderr(waitErr, stderr.String())
	}
	if response.Result == nil {
		return Result{}, errors.New("OCR worker returned an empty recognition result")
	}
	return *response.Result, nil
}

func sendWorkerRequest(encoder *json.Encoder, decoder *json.Decoder, req workerRequest) error {
	_, err := exchangeWorkerRequest(encoder, decoder, req)
	return err
}

func exchangeWorkerRequest(encoder *json.Encoder, decoder *json.Decoder, req workerRequest) (workerResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return workerResponse{}, errors.New("worker request id is required")
	}
	if err := encoder.Encode(req); err != nil {
		return workerResponse{}, err
	}
	var response workerResponse
	if err := decoder.Decode(&response); err != nil {
		return workerResponse{}, err
	}
	if response.ID != req.ID {
		return workerResponse{}, fmt.Errorf("worker response id %q does not match request id %q", response.ID, req.ID)
	}
	if !response.OK {
		if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
			if response.Error.Code != "" {
				return response, fmt.Errorf("OCR worker error %s: %s", response.Error.Code, response.Error.Message)
			}
			return response, errors.New(response.Error.Message)
		}
		return response, errors.New("OCR worker returned an unknown error")
	}
	return response, nil
}

func appendWorkerStderr(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w; worker stderr: %s", err, stderr)
}

func requestIDNow() string {
	return fmt.Sprint(time.Now().UnixNano())
}
