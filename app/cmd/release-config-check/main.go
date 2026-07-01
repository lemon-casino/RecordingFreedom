package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type configCheck struct {
	File    string   `json:"file"`
	Name    string   `json:"name"`
	Needles []string `json:"needles"`
}

type checkResult struct {
	File    string `json:"file"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type report struct {
	OK          bool          `json:"ok"`
	GeneratedAt time.Time     `json:"generatedAt"`
	Root        string        `json:"root"`
	Checks      []checkResult `json:"checks"`
}

var releaseConfigChecks = []configCheck{
	{
		File: ".github/workflows/ci.yml",
		Name: "CI keeps RNNoise artifact gates",
		Needles: []string{
			"RNNOISE_TAG: rnnoise_native",
			"CGO_ENABLED=1 wails3 build -tags",
			"gtk3,${RNNOISE_TAG}",
			"-require-rnnoise",
			"pacman -S --noconfirm --needed mingw-w64-x86_64-gcc",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI keeps Windows FFmpeg video gate",
		Needles: []string{
			"./scripts/ensure-windows-ffmpeg.ps1",
			"-require-video -require-rnnoise",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release builds RNNoise artifacts",
		Needles: []string{
			"RNNOISE_TAG: rnnoise_native",
			"CGO_ENABLED=1 wails3 build -tags",
			"gtk3,${RNNOISE_TAG}",
			"-require-rnnoise",
			"RNNoise native DSP is compiled into release artifacts",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release stages verified Windows portable zip",
		Needles: []string{
			"./scripts/ensure-windows-ffmpeg.ps1",
			"tools/ffmpeg.exe",
			"tools/ffprobe.exe",
			"tools/THIRD_PARTY_FFMPEG.txt",
			"./scripts/verify-windows-portable.ps1",
		},
	},
}

func main() {
	var root string
	flag.StringVar(&root, "root", "", "repository root; defaults to walking up from the current directory")
	flag.Parse()

	result, err := run(root)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode release config report: %v\n", err)
		os.Exit(1)
	}
	if !result.OK {
		os.Exit(1)
	}
}

func run(root string) (report, error) {
	resolved, err := resolveRoot(root)
	if err != nil {
		return report{}, err
	}
	result := report{
		OK:          true,
		GeneratedAt: time.Now().UTC(),
		Root:        resolved,
		Checks:      make([]checkResult, 0, len(releaseConfigChecks)),
	}
	for _, check := range releaseConfigChecks {
		item := evaluateCheck(resolved, check)
		if item.Status != "ready" {
			result.OK = false
		}
		result.Checks = append(result.Checks, item)
	}
	return result, nil
}

func resolveRoot(root string) (string, error) {
	if strings.TrimSpace(root) != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, ".github", "workflows", "release.yml")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("could not find repository root from %q", wd)
		}
		wd = parent
	}
}

func evaluateCheck(root string, check configCheck) checkResult {
	path := filepath.Join(root, filepath.FromSlash(check.File))
	data, err := os.ReadFile(path)
	if err != nil {
		return checkResult{File: check.File, Name: check.Name, Status: "blocked", Message: err.Error()}
	}
	content := string(data)
	missing := make([]string, 0)
	for _, needle := range check.Needles {
		if !strings.Contains(content, needle) {
			missing = append(missing, needle)
		}
	}
	if len(missing) > 0 {
		return checkResult{
			File:    check.File,
			Name:    check.Name,
			Status:  "blocked",
			Message: "missing required release gate text: " + strings.Join(missing, " | "),
		}
	}
	return checkResult{File: check.File, Name: check.Name, Status: "ready"}
}
