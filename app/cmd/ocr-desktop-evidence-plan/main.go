package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/ocrevidence"
)

type options struct {
	visualDir string
	dataRoot  string
	outDir    string
	format    string
	check     bool
}

type planReport = ocrevidence.ChecklistReport

func main() {
	var opts options
	flag.StringVar(&opts.visualDir, "visual-dir", "", "directory containing real desktop visual evidence screenshots to precheck")
	flag.StringVar(&opts.dataRoot, "data-root", "", "RecordingFreedom data root to precheck for app-log, OCR job events, and result chain")
	flag.StringVar(&opts.outDir, "out-dir", "", "directory to write visual-capture-checklist.md and visual-capture-checklist.json")
	flag.StringVar(&opts.format, "format", "json", "stdout format: json or markdown")
	flag.BoolVar(&opts.check, "check", false, "exit non-zero when -visual-dir is missing any required visual evidence scene")
	flag.Parse()

	report, err := run(opts)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		os.Exit(1)
	}
	switch strings.ToLower(strings.TrimSpace(opts.format)) {
	case "markdown", "md":
		fmt.Print(markdownChecklist(report))
	case "", "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode OCR desktop evidence plan: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported -format %q; use json or markdown\n", opts.format)
		os.Exit(2)
	}
	if opts.check && !report.CheckComplete {
		os.Exit(1)
	}
}

func run(opts options) (planReport, error) {
	report := ocrevidence.NewChecklistReport(time.Now().UTC(), "", "", nil)
	if strings.TrimSpace(opts.visualDir) != "" {
		visualDir, err := filepath.Abs(opts.visualDir)
		if err != nil {
			return planReport{}, err
		}
		files, dimensions, err := scanVisualDir(visualDir)
		if err != nil {
			return planReport{}, err
		}
		report = ocrevidence.NewChecklistReportWithDimensions(report.GeneratedAt, visualDir, "", files, dimensions)
	}
	if strings.TrimSpace(opts.dataRoot) != "" {
		dataRoot, err := filepath.Abs(opts.dataRoot)
		if err != nil {
			return planReport{}, err
		}
		precheck, err := ocrevidence.AuditDataRoot(dataRoot)
		if err != nil {
			return planReport{}, err
		}
		report.DataRootPrecheck = &precheck
		if !precheck.CheckComplete {
			report.CheckComplete = false
		}
	}
	if strings.TrimSpace(opts.outDir) != "" {
		outDir, err := filepath.Abs(opts.outDir)
		if err != nil {
			return planReport{}, err
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return planReport{}, err
		}
		report.OutputDir = outDir
		report.MarkdownChecklistPath = filepath.Join(outDir, "visual-capture-checklist.md")
		report.JSONChecklistPath = filepath.Join(outDir, "visual-capture-checklist.json")
		if err := os.WriteFile(report.MarkdownChecklistPath, []byte(markdownChecklist(report)), 0o644); err != nil {
			return planReport{}, err
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return planReport{}, err
		}
		if err := os.WriteFile(report.JSONChecklistPath, append(data, '\n'), 0o644); err != nil {
			return planReport{}, err
		}
	}
	return report, nil
}

func scanVisualDir(root string) ([]string, []ocrevidence.VisualFileDimension, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("visual dir %q is not a directory", root)
	}
	files := []string{}
	dimensions := []ocrevidence.VisualFileDimension{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isIgnoredVisualMetadataFile(entry.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = strings.ToLower(filepath.ToSlash(rel))
		width, height, err := imageSize(path)
		if err != nil {
			return fmt.Errorf("visual evidence %s is not a decodable image: %w", rel, err)
		}
		files = append(files, rel)
		dimensions = append(dimensions, ocrevidence.VisualFileDimension{
			Path:   rel,
			Width:  width,
			Height: height,
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return files, dimensions, nil
}

func isIgnoredVisualMetadataFile(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".ds_store", "thumbs.db", "visual-manifest.json", "visual-capture-checklist.md", "visual-capture-checklist.json":
		return true
	default:
		return false
	}
}

func markdownChecklist(report planReport) string {
	return ocrevidence.MarkdownChecklist(report)
}

func imageSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions %dx%d", config.Width, config.Height)
	}
	return config.Width, config.Height, nil
}
