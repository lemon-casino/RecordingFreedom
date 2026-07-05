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
		Name: "CI keeps RNNoise dynamic artifact gates",
		Needles: []string{
			"RNNOISE_TAG: rnnoise_dynamic",
			"Build RNNoise dynamic module",
			"./scripts/build-rnnoise-windows.ps1",
			"./scripts/build-rnnoise-unix.sh",
			"wails3 task windows:build ARCH=\"${{ matrix.arch }}\" EXTRA_TAGS=\"${tags}\"",
			"-require-rnnoise",
			"msys2/setup-msys2@v2",
			"release: true",
			"windows_msystem: MINGW64",
			"mingw-w64-x86_64-gcc",
			"windows_cc_subdir: mingw64",
			"windows_cc_exe: gcc.exe",
			"windows_msystem: CLANGARM64",
			"mingw-w64-clang-aarch64-clang",
			"windows_cc_subdir: clangarm64",
			"windows_cc_exe: clang.exe",
			"steps.msys2.outputs.msys2-location",
			"-Compiler \"$env:CC\"",
			"cgo_enabled: \"0\"",
			"rnnoise_dynamic: \"true\"",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI keeps Windows FFmpeg video gate",
		Needles: []string{
			"./scripts/ensure-windows-ffmpeg.ps1 -Architecture \"${{ matrix.ffmpeg_arch }}\"",
			"-require-video -require-rnnoise",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI runs frontend e2e gates",
		Needles: []string{
			"Install Playwright Chromium",
			"Run frontend e2e",
			"npm run test:e2e",
			"Clean frontend e2e artifacts before Go tests",
			"rm -rf frontend/test-results frontend/playwright-report",
			"set +e",
			"tee go-test.log",
			"set -e",
			"GITHUB_STEP_SUMMARY",
			"Go test failed",
		},
	},
	{
		File: "app/frontend/tests/capsule-whiteboard-entry.spec.ts",
		Name: "Frontend e2e fixes whiteboard dual-state entry",
		Needles: []string{
			"opens board before recording and annotation during video recording",
			"remains available as a board during audio recording",
			"expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')",
			"expectWhiteboardLaunch(page, 'annotation', '/#/annotation-overlay')",
			"Pause recording",
			"Resume recording",
			"is-recording-compact",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI builds desktop smoke tools",
		Needles: []string{
			"Build desktop smoke tools",
			"desktop-doctor${ext}",
			"video-smoke${ext}",
			"audio-smoke${ext}",
			"pip-export-smoke${ext}",
			"annotation-export-smoke${ext}",
			"annotation-overlay-evidence-check${ext}",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI runs PIP export pixel smoke",
		Needles: []string{
			"Prepare Linux FFmpeg tools for PIP export smoke",
			"Run PIP export pixel smoke",
			"./cmd/pip-export-smoke -synthetic -width 640 -height 360 -duration 1s",
			"./bin/pip-export-smoke${ext}",
			"-synthetic -width 640 -height 360 -duration 1s",
		},
	},
	{
		File: ".github/workflows/ci.yml",
		Name: "CI builds all desktop platforms",
		Needles: []string{
			"windows-x64",
			"windows-arm64",
			"macos-x64",
			"macos-arm64",
			"linux-x64",
			"linux-arm64",
			"windows-11-arm",
			"macos-15-intel",
			"ubuntu-24.04-arm",
			"./scripts/ensure-unix-ffmpeg.sh",
			"wails3 task darwin:package",
			"wails3 task linux:build",
			"recordingfreedom-${{ matrix.name }}",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release builds RNNoise dynamic artifacts",
		Needles: []string{
			"RNNOISE_TAG: rnnoise_dynamic",
			"Build RNNoise dynamic module",
			"./scripts/build-rnnoise-windows.ps1",
			"./scripts/build-rnnoise-unix.sh",
			"wails3 task windows:build ARCH=\"${{ matrix.arch }}\" EXTRA_TAGS=\"${tags}\"",
			"-require-rnnoise",
			"msys2/setup-msys2@v2",
			"release: true",
			"windows_msystem: MINGW64",
			"mingw-w64-x86_64-gcc",
			"windows_cc_subdir: mingw64",
			"windows_cc_exe: gcc.exe",
			"windows_msystem: CLANGARM64",
			"mingw-w64-clang-aarch64-clang",
			"windows_cc_subdir: clangarm64",
			"windows_cc_exe: clang.exe",
			"steps.msys2.outputs.msys2-location",
			"-Compiler \"$env:CC\"",
			"cgo_enabled: \"0\"",
			"rnnoise_dynamic: \"true\"",
			"RNNoise native DSP is built from source as a dynamic module for every desktop artifact",
			"including Windows ARM64",
			"tools/rnnoise.dll",
			"Contents/MacOS/tools/librnnoise.dylib",
			"tools/librnnoise.so",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release gate runs frontend e2e gates",
		Needles: []string{
			"Install Playwright Chromium",
			"Run frontend e2e",
			"npm run test:e2e",
			"Clean frontend e2e artifacts before Go tests",
			"rm -rf frontend/test-results frontend/playwright-report",
			"set +e",
			"tee go-test.log",
			"set -e",
			"GITHUB_STEP_SUMMARY",
			"Go test failed",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release notes distinguish whiteboard preview from overlay acceptance",
		Needles: []string{
			"Whiteboard / Excalidraw board and recording annotation",
			"opens the normal whiteboard",
			"opens the annotation overlay",
			"Whiteboard scene persistence and export",
			"Recording annotation package flow",
			"Completed cross-platform annotation overlay acceptance",
			"real Windows/macOS/Linux evidence is still required",
			"transparent click-through",
			"multi-monitor/high-DPI geometry",
			"real hand-drawn annotation packages",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release notes include smart screenshot and whiteboard history behavior",
		Needles: []string{
			"focused-window screenshot",
			"Smart screenshot and region assist",
			"RegionSmartCandidate",
			"mg-chao/snow-shot",
			"manual normal whiteboard save now writes the Excalidraw scene and a PNG snapshot into screenshot history",
			"mode=whiteboard",
			"recording annotation saves continue to write into the active .rfrec annotation package",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release stages verified Windows portable zip and installer",
		Needles: []string{
			"./scripts/ensure-windows-ffmpeg.ps1 -Architecture \"${{ matrix.ffmpeg_arch }}\"",
			"Install Windows installer tooling",
			"Build Windows installer",
			"ARG_WAILS_${{ matrix.nsis_arch }}_BINARY",
			"setup.exe",
			"tools/ffmpeg.exe",
			"tools/ffprobe.exe",
			"tools/rnnoise.dll",
			"tools/THIRD_PARTY_FFMPEG.txt",
			"tools/THIRD_PARTY_NOTICES.txt",
			"tools/desktop-doctor.exe",
			"tools/video-smoke.exe",
			"tools/audio-smoke.exe",
			"tools/pip-export-smoke.exe",
			"tools/annotation-export-smoke.exe",
			"tools/annotation-overlay-evidence-check.exe",
			"tools/run-windows-portable-smoke.ps1",
			"./scripts/verify-windows-portable.ps1",
			"./scripts/verify-windows-installer.ps1",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release stages full desktop platform artifacts",
		Needles: []string{
			"windows-x64",
			"windows-arm64",
			"macos-x64",
			"macos-arm64",
			"linux-x64",
			"linux-arm64",
			"windows-11-arm",
			"macos-15-intel",
			"ubuntu-24.04-arm",
			"./scripts/ensure-unix-ffmpeg.sh",
			"wails3 task darwin:package",
			"wails3 task linux:build",
			"RecordingFreedom-${{ matrix.name }}-${{ github.ref_name }}-app.zip",
			"RecordingFreedom-${{ matrix.name }}-${{ github.ref_name }}-portable.tar.gz",
			"./scripts/verify-macos-app-zip.sh",
			"./scripts/verify-linux-portable.sh",
			"Full desktop build matrix is enabled for Windows x64/arm64, macOS x64/arm64, and Linux x64/arm64",
			"macOS app bundles include Contents/MacOS/recordingfreedom",
			"Contents/MacOS/tools/librnnoise.dylib",
			"Linux portable archives include recordingfreedom",
			"tools/librnnoise.so",
		},
	},
	{
		File: "scripts/build-rnnoise-windows.ps1",
		Name: "Windows RNNoise dynamic module builder emits x64 and arm64 DLLs",
		Needles: []string{
			`ValidateSet("x64", "arm64")`,
			"likely_voice_enhancer.c",
			"rnnoise.dll",
			"LIKELY_VOICE_ENHANCER_BUILD_DLL",
			"Resolve-CompilerPath",
			`C:\msys64\mingw64\bin\gcc.exe`,
			`C:\msys64\clangarm64\bin\clang.exe`,
			"Resolved RNNoise compiler",
			"Assert-PEMachine",
			"0x8664",
			"0xAA64",
		},
	},
	{
		File: "scripts/build-rnnoise-unix.sh",
		Name: "Unix RNNoise dynamic module builder emits dylib and so modules",
		Needles: []string{
			"--platform linux|macos",
			"likely_voice_enhancer.c",
			"librnnoise.dylib",
			"librnnoise.so",
			"LIKELY_VOICE_ENHANCER_BUILD_DLL",
			"Mach-O.*arm64",
			"ELF 64-bit",
		},
	},
	{
		File: "scripts/ensure-windows-ffmpeg.ps1",
		Name: "Windows FFmpeg bundler selects x64 and arm64 tools",
		Needles: []string{
			`ValidateSet("x64", "arm64")`,
			"ffmpeg-master-latest-win64-gpl.zip",
			"ffmpeg-master-latest-winarm64-gpl.zip",
			"Architecture:",
			"THIRD_PARTY_FFMPEG.txt",
		},
	},
	{
		File: "scripts/ensure-unix-ffmpeg.sh",
		Name: "Unix FFmpeg bundler pins checksummed macOS and Linux tools",
		Needles: []string{
			"n8.1.2-1",
			"ffmpeg-osx-x64",
			"ffprobe-osx-x64",
			"ffmpeg-osx-arm64",
			"ffprobe-osx-arm64",
			"ffmpeg-linux-x64",
			"ffprobe-linux-x64",
			"ffmpeg-linux-arm64",
			"ffprobe-linux-arm64",
			"SHA256 mismatch",
			"THIRD_PARTY_FFMPEG.txt",
		},
	},
	{
		File: "scripts/verify-macos-app-zip.sh",
		Name: "macOS release verifier checks app bundle contents",
		Needles: []string{
			"RecordingFreedom.app",
			"Contents/MacOS/recordingfreedom",
			"Contents/MacOS/tools",
			"CFBundleExecutable",
			"ffmpeg",
			"ffprobe",
			"librnnoise.dylib",
			"THIRD_PARTY_NOTICES.txt",
			"expected_arch",
			"x86_64",
			"arm64",
		},
	},
	{
		File: "scripts/verify-linux-portable.sh",
		Name: "Linux release verifier checks portable archive contents",
		Needles: []string{
			"RecordingFreedom-linux-*",
			"recordingfreedom",
			"tools/ffmpeg",
			"tools/ffprobe",
			"tools/pip-export-smoke",
			"tools/librnnoise.so",
			"recordingfreedom.desktop",
			"THIRD_PARTY_NOTICES.txt",
			"ELF 64-bit",
			"expected_arch",
			"ARM aarch64",
		},
	},
	{
		File: "app/build/windows/nsis/project.nsi",
		Name: "Windows installer includes bundled FFmpeg tools",
		Needles: []string{
			"ARG_RECORDINGFREEDOM_TOOLS_DIR",
			`$INSTDIR\tools`,
			"ffmpeg.exe",
			"ffprobe.exe",
			"rnnoise.dll",
			"THIRD_PARTY_FFMPEG.txt",
			"THIRD_PARTY_NOTICES.txt",
		},
	},
	{
		File: "scripts/verify-windows-installer.ps1",
		Name: "Windows installer verifies installed FFmpeg layout",
		Needles: []string{
			"RecordingFreedom.exe",
			"recordingfreedom.exe",
			"tools",
			"ffmpeg.exe",
			"ffprobe.exe",
			"rnnoise.dll",
			"THIRD_PARTY_FFMPEG.txt",
			"THIRD_PARTY_NOTICES.txt",
			"uninstall.exe",
		},
	},
	{
		File: "app/build/darwin/Taskfile.yml",
		Name: "macOS app bundle requires bundled FFmpeg tools",
		Needles: []string{
			"Contents/MacOS/tools",
			"tools/ffmpeg",
			"tools/ffprobe",
			"tools/librnnoise.dylib",
			"THIRD_PARTY_FFMPEG.txt",
			"THIRD_PARTY_NOTICES.txt",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release builds desktop smoke tools",
		Needles: []string{
			"Build desktop smoke tools",
			"desktop-doctor${ext}",
			"video-smoke${ext}",
			"audio-smoke${ext}",
			"pip-export-smoke${ext}",
			"annotation-export-smoke${ext}",
			"annotation-overlay-evidence-check${ext}",
			"run-windows-portable-smoke.ps1",
		},
	},
	{
		File: ".github/workflows/release.yml",
		Name: "Release runs PIP export pixel smoke",
		Needles: []string{
			"Prepare Linux FFmpeg tools for PIP export smoke",
			"Run PIP export pixel smoke",
			"./cmd/pip-export-smoke -synthetic -width 640 -height 360 -duration 1s",
			"./bin/pip-export-smoke${ext}",
			"-synthetic -width 640 -height 360 -duration 1s",
			"FFmpeg PIP export verification that samples the final MP4 PIP pixels",
		},
	},
	{
		File: "app/build/windows/Taskfile.yml",
		Name: "Windows build keeps GUI subsystem",
		Needles: []string{
			`-ldflags="-w -s -H windowsgui"`,
		},
	},
	{
		File: "scripts/verify-windows-portable.ps1",
		Name: "Windows portable zip verifies architecture-specific GUI PE metadata",
		Needles: []string{
			`ValidateSet("x64", "arm64")`,
			"Assert-PEMetadata",
			"Assert-PowerShellScript",
			"Assert-FileContains",
			"ExpectedSubsystem 2",
			"0x8664",
			"0xAA64",
			"recordingfreedom.exe",
			"tools/rnnoise.dll",
			"THIRD_PARTY_NOTICES.txt",
			"@excalidraw/excalidraw",
			"Copyright (c) 2020 Excalidraw",
			"tools/desktop-doctor.exe",
			"tools/video-smoke.exe",
			"tools/audio-smoke.exe",
			"tools/pip-export-smoke.exe",
			"tools/annotation-export-smoke.exe",
			"tools/annotation-overlay-evidence-check.exe",
			"tools/run-windows-portable-smoke.ps1",
			"pip-export-pixels",
			"-synthetic",
		},
	},
	{
		File: "app/tools/THIRD_PARTY_NOTICES.txt",
		Name: "Third-party notices include RNNoise and Excalidraw licenses",
		Needles: []string{
			"RNNoise",
			"License: BSD-style license",
			"Copyright (c) 2017, Mozilla",
			"Neither the name of the Xiph.Org Foundation",
			"@excalidraw/excalidraw",
			"License: MIT",
			"Copyright (c) 2020 Excalidraw",
			`THE SOFTWARE IS PROVIDED "AS IS"`,
		},
	},
	{
		File: "scripts/run-windows-portable-smoke.ps1",
		Name: "Windows portable smoke runner executes real recording diagnostics",
		Needles: []string{
			"desktop-doctor.exe",
			"video-smoke.exe",
			"audio-smoke.exe",
			"pip-export-smoke.exe",
			"annotation-export-smoke.exe",
			"annotation-overlay-evidence-check.exe",
			"pip-export-pixels",
			"-synthetic",
			"RunAnnotationLong",
			"LongAnnotationDurations",
			"LongAnnotationSegments",
			"annotation-long-snapshots",
			"annotation-long-element-pngs",
			"-segments=",
			"-timeline=element-pngs",
			"-source-type=region",
			"-source-type=window",
			"RECORDINGFREEDOM_FFMPEG_PATH",
			"-source-type=region",
			"-source-type=window",
			"-microphone",
			"-system",
			"-rnnoise",
		},
	},
	{
		File: "docs/12-annotation-overlay-platform-smoke.md",
		Name: "Annotation overlay has real platform smoke standard",
		Needles: []string{
			"录制标注 Overlay 实机验收标准",
			"annotation-export-smoke",
			"annotation-overlay-evidence-check",
			"不能替代真实桌面上的透明窗口",
			"全屏、单屏、区域、锁定窗口",
			"多屏幕",
			"高 DPI",
			"点击穿透",
			".rfrec/annotations/",
			"annotations/overlay-diagnostics.jsonl",
			"exports/recording.mp4",
			"evidence/annotation-overlay",
		},
	},
	{
		File: "app/cmd/annotation-overlay-evidence-check/main.go",
		Name: "Annotation overlay evidence checker verifies real package artifacts",
		Needles: []string{
			"-evidence-dir",
			"manifest annotations.enabled is required",
			"annotation target geometry does not match source geometry",
			"validateEvidenceREADME",
			"README.md is missing evidence records",
			"artifact source",
			"validatePlatformFile",
			"platform.txt is missing display environment records",
			"displayResolutionPattern",
			"requiredScreenshotEvidence",
			"requiredRecordingEvidence",
			"requireEvidenceNamedFiles",
			"is missing evidence files",
			"validateAppLog",
			"app-log.jsonl is missing required events",
			"recording/start-request sourceType=",
			"annotation-overlay/show targetType=",
			"recpackage.AnnotationEventsFile",
			"recpackage.AnnotationOverlayDiagnosticsFile",
			"missing %s diagnostic event",
			"1m recording package",
			"5m recording package",
			"requiredEvidenceSourceChecks",
			"source all-screens package",
			"source screen package",
			"source region package",
			"source window package",
			"requireSourceType",
			"diagnostics.sync.screen.durationMs",
			"missing drawing hit-regions event with full canvas rect",
			"missing pass-through hit-regions event without full canvas rect",
			"hitRegionsContainCanvasRect",
			"hitRegionsContainPill",
			"validateAnnotationEvents",
			"missing element-created/element-updated/element-deleted event",
			"scene-snapshot has invalid snapshotPath",
			"findAnnotationSnapshot",
			"recpackage.ProbeMP4",
			"MP4 video track is missing",
			"diagnostics.sync.screen.durationMs",
			"exportplan.DefaultOutputPath",
		},
	},
	{
		File: "docs/README.md",
		Name: "Docs index links annotation overlay smoke standard",
		Needles: []string{
			"12-annotation-overlay-platform-smoke.md",
			"真实桌面验收矩阵",
			"不能只靠 `annotation-export-smoke` 宣称完成",
		},
	},
	{
		File: "scripts/verify-windows-preview-release.ps1",
		Name: "Windows preview release asset can be downloaded and verified",
		Needles: []string{
			"api.github.com/repos",
			"SHA256SUMS-windows-x64",
			"Get-FileHash -Algorithm SHA256",
			"verify-windows-portable.ps1",
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
