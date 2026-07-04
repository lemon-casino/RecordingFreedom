package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAcceptsRequiredReleaseGates(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, ".github/workflows/ci.yml", workflowFixture(".github/workflows/ci.yml"))
	writeWorkflow(t, root, ".github/workflows/release.yml", workflowFixture(".github/workflows/release.yml"))
	writeWorkflow(t, root, "app/build/darwin/Taskfile.yml", workflowFixture("app/build/darwin/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/Taskfile.yml", workflowFixture("app/build/windows/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/nsis/project.nsi", workflowFixture("app/build/windows/nsis/project.nsi"))
	writeWorkflow(t, root, "app/tools/THIRD_PARTY_NOTICES.txt", workflowFixture("app/tools/THIRD_PARTY_NOTICES.txt"))
	writeWorkflow(t, root, "scripts/verify-windows-installer.ps1", workflowFixture("scripts/verify-windows-installer.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-portable.ps1", workflowFixture("scripts/verify-windows-portable.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-preview-release.ps1", workflowFixture("scripts/verify-windows-preview-release.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-windows.ps1", workflowFixture("scripts/build-rnnoise-windows.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-unix.sh", workflowFixture("scripts/build-rnnoise-unix.sh"))
	writeWorkflow(t, root, "scripts/ensure-windows-ffmpeg.ps1", workflowFixture("scripts/ensure-windows-ffmpeg.ps1"))
	writeWorkflow(t, root, "scripts/ensure-unix-ffmpeg.sh", workflowFixture("scripts/ensure-unix-ffmpeg.sh"))
	writeWorkflow(t, root, "scripts/verify-macos-app-zip.sh", workflowFixture("scripts/verify-macos-app-zip.sh"))
	writeWorkflow(t, root, "scripts/verify-linux-portable.sh", workflowFixture("scripts/verify-linux-portable.sh"))
	writeWorkflow(t, root, "scripts/run-windows-portable-smoke.ps1", workflowFixture("scripts/run-windows-portable-smoke.ps1"))
	writeWorkflow(t, root, "app/frontend/tests/capsule-whiteboard-entry.spec.ts", workflowFixture("app/frontend/tests/capsule-whiteboard-entry.spec.ts"))
	writeWorkflow(t, root, "docs/12-annotation-overlay-platform-smoke.md", workflowFixture("docs/12-annotation-overlay-platform-smoke.md"))
	writeWorkflow(t, root, "app/cmd/annotation-overlay-evidence-check/main.go", workflowFixture("app/cmd/annotation-overlay-evidence-check/main.go"))
	writeWorkflow(t, root, "docs/README.md", workflowFixture("docs/README.md"))

	report, err := run(root)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.OK {
		t.Fatalf("report.OK = false: %#v", report.Checks)
	}
	if len(report.Checks) != len(releaseConfigChecks) {
		t.Fatalf("checks = %d, want %d", len(report.Checks), len(releaseConfigChecks))
	}
}

func TestRunRejectsMissingRNNoiseGate(t *testing.T) {
	root := t.TempDir()
	ci := strings.ReplaceAll(workflowFixture(".github/workflows/ci.yml"), "-require-rnnoise", "")
	writeWorkflow(t, root, ".github/workflows/ci.yml", ci)
	writeWorkflow(t, root, ".github/workflows/release.yml", workflowFixture(".github/workflows/release.yml"))
	writeWorkflow(t, root, "app/build/darwin/Taskfile.yml", workflowFixture("app/build/darwin/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/Taskfile.yml", workflowFixture("app/build/windows/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/nsis/project.nsi", workflowFixture("app/build/windows/nsis/project.nsi"))
	writeWorkflow(t, root, "app/tools/THIRD_PARTY_NOTICES.txt", workflowFixture("app/tools/THIRD_PARTY_NOTICES.txt"))
	writeWorkflow(t, root, "scripts/verify-windows-installer.ps1", workflowFixture("scripts/verify-windows-installer.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-portable.ps1", workflowFixture("scripts/verify-windows-portable.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-preview-release.ps1", workflowFixture("scripts/verify-windows-preview-release.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-windows.ps1", workflowFixture("scripts/build-rnnoise-windows.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-unix.sh", workflowFixture("scripts/build-rnnoise-unix.sh"))
	writeWorkflow(t, root, "scripts/ensure-windows-ffmpeg.ps1", workflowFixture("scripts/ensure-windows-ffmpeg.ps1"))
	writeWorkflow(t, root, "scripts/ensure-unix-ffmpeg.sh", workflowFixture("scripts/ensure-unix-ffmpeg.sh"))
	writeWorkflow(t, root, "scripts/verify-macos-app-zip.sh", workflowFixture("scripts/verify-macos-app-zip.sh"))
	writeWorkflow(t, root, "scripts/verify-linux-portable.sh", workflowFixture("scripts/verify-linux-portable.sh"))
	writeWorkflow(t, root, "scripts/run-windows-portable-smoke.ps1", workflowFixture("scripts/run-windows-portable-smoke.ps1"))
	writeWorkflow(t, root, "app/frontend/tests/capsule-whiteboard-entry.spec.ts", workflowFixture("app/frontend/tests/capsule-whiteboard-entry.spec.ts"))
	writeWorkflow(t, root, "docs/12-annotation-overlay-platform-smoke.md", workflowFixture("docs/12-annotation-overlay-platform-smoke.md"))
	writeWorkflow(t, root, "app/cmd/annotation-overlay-evidence-check/main.go", workflowFixture("app/cmd/annotation-overlay-evidence-check/main.go"))
	writeWorkflow(t, root, "docs/README.md", workflowFixture("docs/README.md"))

	report, err := run(root)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked when RNNoise gate is missing")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Status == "blocked" && strings.Contains(check.Message, "-require-rnnoise") {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocked RNNoise check not found: %#v", report.Checks)
	}
}

func TestRunRejectsMissingWhiteboardReleaseNotes(t *testing.T) {
	root := t.TempDir()
	release := strings.ReplaceAll(workflowFixture(".github/workflows/release.yml"), "Whiteboard / Excalidraw board and recording annotation", "")
	writeWorkflow(t, root, ".github/workflows/ci.yml", workflowFixture(".github/workflows/ci.yml"))
	writeWorkflow(t, root, ".github/workflows/release.yml", release)
	writeWorkflow(t, root, "app/build/darwin/Taskfile.yml", workflowFixture("app/build/darwin/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/Taskfile.yml", workflowFixture("app/build/windows/Taskfile.yml"))
	writeWorkflow(t, root, "app/build/windows/nsis/project.nsi", workflowFixture("app/build/windows/nsis/project.nsi"))
	writeWorkflow(t, root, "app/tools/THIRD_PARTY_NOTICES.txt", workflowFixture("app/tools/THIRD_PARTY_NOTICES.txt"))
	writeWorkflow(t, root, "scripts/verify-windows-installer.ps1", workflowFixture("scripts/verify-windows-installer.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-portable.ps1", workflowFixture("scripts/verify-windows-portable.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-preview-release.ps1", workflowFixture("scripts/verify-windows-preview-release.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-windows.ps1", workflowFixture("scripts/build-rnnoise-windows.ps1"))
	writeWorkflow(t, root, "scripts/build-rnnoise-unix.sh", workflowFixture("scripts/build-rnnoise-unix.sh"))
	writeWorkflow(t, root, "scripts/ensure-windows-ffmpeg.ps1", workflowFixture("scripts/ensure-windows-ffmpeg.ps1"))
	writeWorkflow(t, root, "scripts/ensure-unix-ffmpeg.sh", workflowFixture("scripts/ensure-unix-ffmpeg.sh"))
	writeWorkflow(t, root, "scripts/verify-macos-app-zip.sh", workflowFixture("scripts/verify-macos-app-zip.sh"))
	writeWorkflow(t, root, "scripts/verify-linux-portable.sh", workflowFixture("scripts/verify-linux-portable.sh"))
	writeWorkflow(t, root, "scripts/run-windows-portable-smoke.ps1", workflowFixture("scripts/run-windows-portable-smoke.ps1"))
	writeWorkflow(t, root, "app/frontend/tests/capsule-whiteboard-entry.spec.ts", workflowFixture("app/frontend/tests/capsule-whiteboard-entry.spec.ts"))
	writeWorkflow(t, root, "docs/12-annotation-overlay-platform-smoke.md", workflowFixture("docs/12-annotation-overlay-platform-smoke.md"))
	writeWorkflow(t, root, "app/cmd/annotation-overlay-evidence-check/main.go", workflowFixture("app/cmd/annotation-overlay-evidence-check/main.go"))
	writeWorkflow(t, root, "docs/README.md", workflowFixture("docs/README.md"))

	report, err := run(root)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if report.OK {
		t.Fatalf("report.OK = true, want blocked when whiteboard preview release notes are missing")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Status == "blocked" && strings.Contains(check.Message, "Whiteboard / Excalidraw board and recording annotation") {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocked whiteboard release notes check not found: %#v", report.Checks)
	}
}

func writeWorkflow(t *testing.T, root string, name string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func workflowFixture(name string) string {
	var builder strings.Builder
	builder.WriteString("RNNOISE_TAG: rnnoise_dynamic\n")
	builder.WriteString("CGO_ENABLED=1 wails3 build -tags\n")
	builder.WriteString("wails3 task windows:build ARCH=\"${{ matrix.arch }}\" EXTRA_TAGS=\"${tags}\"\n")
	builder.WriteString("gtk3,${RNNOISE_TAG}\n")
	builder.WriteString("Build RNNoise dynamic module\n")
	builder.WriteString("./scripts/build-rnnoise-windows.ps1\n")
	builder.WriteString("./scripts/build-rnnoise-unix.sh\n")
	builder.WriteString("-require-rnnoise\n")
	builder.WriteString("msys2/setup-msys2@v2\n")
	builder.WriteString("release: true\n")
	builder.WriteString("windows_msystem: MINGW64\n")
	builder.WriteString("mingw-w64-x86_64-gcc\n")
	builder.WriteString("windows_cc_subdir: mingw64\n")
	builder.WriteString("windows_cc_exe: gcc.exe\n")
	builder.WriteString("windows_msystem: CLANGARM64\n")
	builder.WriteString("mingw-w64-clang-aarch64-clang\n")
	builder.WriteString("windows_cc_subdir: clangarm64\n")
	builder.WriteString("windows_cc_exe: clang.exe\n")
	builder.WriteString("steps.msys2.outputs.msys2-location\n")
	builder.WriteString("-Compiler \"$env:CC\"\n")
	builder.WriteString("cgo_enabled: \"0\"\n")
	builder.WriteString("rnnoise_dynamic: \"true\"\n")
	builder.WriteString("./scripts/ensure-windows-ffmpeg.ps1 -Architecture \"${{ matrix.ffmpeg_arch }}\"\n")
	builder.WriteString("-require-video -require-rnnoise\n")
	builder.WriteString("Install Playwright Chromium\n")
	builder.WriteString("Run frontend e2e\n")
	builder.WriteString("npm run test:e2e\n")
	builder.WriteString("Clean frontend e2e artifacts before Go tests\n")
	builder.WriteString("rm -rf frontend/test-results frontend/playwright-report\n")
	builder.WriteString("set +e\n")
	builder.WriteString("tee go-test.log\n")
	builder.WriteString("set -e\n")
	builder.WriteString("GITHUB_STEP_SUMMARY\n")
	builder.WriteString("Go test failed\n")
	builder.WriteString("Build desktop smoke tools\n")
	builder.WriteString("desktop-doctor${ext}\n")
	builder.WriteString("video-smoke${ext}\n")
	builder.WriteString("audio-smoke${ext}\n")
	builder.WriteString("annotation-export-smoke${ext}\n")
	builder.WriteString("annotation-overlay-evidence-check${ext}\n")
	builder.WriteString("windows-x64\n")
	builder.WriteString("windows-arm64\n")
	builder.WriteString("macos-x64\n")
	builder.WriteString("macos-arm64\n")
	builder.WriteString("linux-x64\n")
	builder.WriteString("linux-arm64\n")
	builder.WriteString("windows-11-arm\n")
	builder.WriteString("macos-15-intel\n")
	builder.WriteString("ubuntu-24.04-arm\n")
	builder.WriteString("./scripts/ensure-unix-ffmpeg.sh\n")
	builder.WriteString("wails3 task darwin:package\n")
	builder.WriteString("wails3 task linux:build\n")
	builder.WriteString("recordingfreedom-${{ matrix.name }}\n")
	if strings.Contains(name, "release.yml") {
		builder.WriteString("RNNoise native DSP is built from source as a dynamic module for every desktop artifact\n")
		builder.WriteString("including Windows ARM64\n")
		builder.WriteString("tools/rnnoise.dll\n")
		builder.WriteString("Contents/MacOS/tools/librnnoise.dylib\n")
		builder.WriteString("tools/librnnoise.so\n")
		builder.WriteString("Whiteboard / Excalidraw board and recording annotation\n")
		builder.WriteString("opens the normal whiteboard\n")
		builder.WriteString("opens the annotation overlay\n")
		builder.WriteString("Whiteboard scene persistence and export\n")
		builder.WriteString("Recording annotation package flow\n")
		builder.WriteString("Completed cross-platform annotation overlay acceptance\n")
		builder.WriteString("real Windows/macOS/Linux evidence is still required\n")
		builder.WriteString("transparent click-through\n")
		builder.WriteString("multi-monitor/high-DPI geometry\n")
		builder.WriteString("real hand-drawn annotation packages\n")
		builder.WriteString("Install Windows installer tooling\n")
		builder.WriteString("Build Windows installer\n")
		builder.WriteString("ARG_WAILS_${{ matrix.nsis_arch }}_BINARY\n")
		builder.WriteString("setup.exe\n")
		builder.WriteString("tools/ffmpeg.exe\n")
		builder.WriteString("tools/ffprobe.exe\n")
		builder.WriteString("tools/rnnoise.dll\n")
		builder.WriteString("tools/THIRD_PARTY_FFMPEG.txt\n")
		builder.WriteString("tools/THIRD_PARTY_NOTICES.txt\n")
		builder.WriteString("tools/desktop-doctor.exe\n")
		builder.WriteString("tools/video-smoke.exe\n")
		builder.WriteString("tools/audio-smoke.exe\n")
		builder.WriteString("tools/annotation-export-smoke.exe\n")
		builder.WriteString("tools/annotation-overlay-evidence-check.exe\n")
		builder.WriteString("tools/run-windows-portable-smoke.ps1\n")
		builder.WriteString("run-windows-portable-smoke.ps1\n")
		builder.WriteString("./scripts/verify-windows-portable.ps1\n")
		builder.WriteString("./scripts/verify-windows-installer.ps1\n")
		builder.WriteString("RecordingFreedom-${{ matrix.name }}-${{ github.ref_name }}-app.zip\n")
		builder.WriteString("RecordingFreedom-${{ matrix.name }}-${{ github.ref_name }}-portable.tar.gz\n")
		builder.WriteString("./scripts/verify-macos-app-zip.sh\n")
		builder.WriteString("./scripts/verify-linux-portable.sh\n")
		builder.WriteString("Full desktop build matrix is enabled for Windows x64/arm64, macOS x64/arm64, and Linux x64/arm64\n")
		builder.WriteString("macOS app bundles include Contents/MacOS/recordingfreedom\n")
		builder.WriteString("Contents/MacOS/tools/librnnoise.dylib\n")
		builder.WriteString("Linux portable archives include recordingfreedom\n")
		builder.WriteString("tools/librnnoise.so\n")
	}
	if strings.Contains(name, "Taskfile.yml") {
		builder.WriteString("-ldflags=\"-w -s -H windowsgui\"\n")
	}
	if strings.Contains(name, "app/build/darwin/Taskfile.yml") {
		builder.WriteString("Contents/MacOS/tools\n")
		builder.WriteString("tools/ffmpeg\n")
		builder.WriteString("tools/ffprobe\n")
		builder.WriteString("tools/librnnoise.dylib\n")
		builder.WriteString("THIRD_PARTY_FFMPEG.txt\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
	}
	if strings.Contains(name, "app/build/windows/nsis/project.nsi") {
		builder.WriteString("ARG_RECORDINGFREEDOM_TOOLS_DIR\n")
		builder.WriteString("$INSTDIR\\tools\n")
		builder.WriteString("ffmpeg.exe\n")
		builder.WriteString("ffprobe.exe\n")
		builder.WriteString("rnnoise.dll\n")
		builder.WriteString("THIRD_PARTY_FFMPEG.txt\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
	}
	if strings.Contains(name, "verify-windows-installer.ps1") {
		builder.WriteString("RecordingFreedom.exe\n")
		builder.WriteString("recordingfreedom.exe\n")
		builder.WriteString("tools\n")
		builder.WriteString("ffmpeg.exe\n")
		builder.WriteString("ffprobe.exe\n")
		builder.WriteString("rnnoise.dll\n")
		builder.WriteString("THIRD_PARTY_FFMPEG.txt\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
		builder.WriteString("uninstall.exe\n")
	}
	if strings.Contains(name, "verify-windows-portable.ps1") {
		builder.WriteString("ValidateSet(\"x64\", \"arm64\")\n")
		builder.WriteString("Assert-PEMetadata\n")
		builder.WriteString("Assert-PowerShellScript\n")
		builder.WriteString("Assert-FileContains\n")
		builder.WriteString("ExpectedSubsystem 2\n")
		builder.WriteString("0x8664\n")
		builder.WriteString("0xAA64\n")
		builder.WriteString("recordingfreedom.exe\n")
		builder.WriteString("tools/rnnoise.dll\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
		builder.WriteString("@excalidraw/excalidraw\n")
		builder.WriteString("Copyright (c) 2020 Excalidraw\n")
		builder.WriteString("tools/desktop-doctor.exe\n")
		builder.WriteString("tools/video-smoke.exe\n")
		builder.WriteString("tools/audio-smoke.exe\n")
		builder.WriteString("tools/annotation-export-smoke.exe\n")
		builder.WriteString("tools/annotation-overlay-evidence-check.exe\n")
		builder.WriteString("tools/run-windows-portable-smoke.ps1\n")
	}
	if strings.Contains(name, "verify-windows-preview-release.ps1") {
		builder.WriteString("api.github.com/repos\n")
		builder.WriteString("SHA256SUMS-windows-x64\n")
		builder.WriteString("Get-FileHash -Algorithm SHA256\n")
		builder.WriteString("verify-windows-portable.ps1\n")
	}
	if strings.Contains(name, "ensure-windows-ffmpeg.ps1") {
		builder.WriteString("ValidateSet(\"x64\", \"arm64\")\n")
		builder.WriteString("ffmpeg-master-latest-win64-gpl.zip\n")
		builder.WriteString("ffmpeg-master-latest-winarm64-gpl.zip\n")
		builder.WriteString("Architecture:\n")
		builder.WriteString("THIRD_PARTY_FFMPEG.txt\n")
	}
	if strings.Contains(name, "build-rnnoise-windows.ps1") {
		builder.WriteString("ValidateSet(\"x64\", \"arm64\")\n")
		builder.WriteString("likely_voice_enhancer.c\n")
		builder.WriteString("rnnoise.dll\n")
		builder.WriteString("LIKELY_VOICE_ENHANCER_BUILD_DLL\n")
		builder.WriteString("Resolve-CompilerPath\n")
		builder.WriteString("C:\\msys64\\mingw64\\bin\\gcc.exe\n")
		builder.WriteString("C:\\msys64\\clangarm64\\bin\\clang.exe\n")
		builder.WriteString("Resolved RNNoise compiler\n")
		builder.WriteString("Assert-PEMachine\n")
		builder.WriteString("0x8664\n")
		builder.WriteString("0xAA64\n")
	}
	if strings.Contains(name, "build-rnnoise-unix.sh") {
		builder.WriteString("--platform linux|macos\n")
		builder.WriteString("likely_voice_enhancer.c\n")
		builder.WriteString("librnnoise.dylib\n")
		builder.WriteString("librnnoise.so\n")
		builder.WriteString("LIKELY_VOICE_ENHANCER_BUILD_DLL\n")
		builder.WriteString("Mach-O.*arm64\n")
		builder.WriteString("ELF 64-bit\n")
	}
	if strings.Contains(name, "ensure-unix-ffmpeg.sh") {
		builder.WriteString("n8.1.2-1\n")
		builder.WriteString("ffmpeg-osx-x64\n")
		builder.WriteString("ffprobe-osx-x64\n")
		builder.WriteString("ffmpeg-osx-arm64\n")
		builder.WriteString("ffprobe-osx-arm64\n")
		builder.WriteString("ffmpeg-linux-x64\n")
		builder.WriteString("ffprobe-linux-x64\n")
		builder.WriteString("ffmpeg-linux-arm64\n")
		builder.WriteString("ffprobe-linux-arm64\n")
		builder.WriteString("SHA256 mismatch\n")
		builder.WriteString("THIRD_PARTY_FFMPEG.txt\n")
	}
	if strings.Contains(name, "verify-macos-app-zip.sh") {
		builder.WriteString("RecordingFreedom.app\n")
		builder.WriteString("Contents/MacOS/recordingfreedom\n")
		builder.WriteString("Contents/MacOS/tools\n")
		builder.WriteString("CFBundleExecutable\n")
		builder.WriteString("ffmpeg\n")
		builder.WriteString("ffprobe\n")
		builder.WriteString("librnnoise.dylib\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
		builder.WriteString("expected_arch\n")
		builder.WriteString("x86_64\n")
		builder.WriteString("arm64\n")
	}
	if strings.Contains(name, "verify-linux-portable.sh") {
		builder.WriteString("RecordingFreedom-linux-*\n")
		builder.WriteString("recordingfreedom\n")
		builder.WriteString("tools/ffmpeg\n")
		builder.WriteString("tools/ffprobe\n")
		builder.WriteString("tools/librnnoise.so\n")
		builder.WriteString("recordingfreedom.desktop\n")
		builder.WriteString("THIRD_PARTY_NOTICES.txt\n")
		builder.WriteString("ELF 64-bit\n")
		builder.WriteString("expected_arch\n")
		builder.WriteString("ARM aarch64\n")
	}
	if strings.Contains(name, "run-windows-portable-smoke.ps1") {
		builder.WriteString("desktop-doctor.exe\n")
		builder.WriteString("video-smoke.exe\n")
		builder.WriteString("audio-smoke.exe\n")
		builder.WriteString("annotation-export-smoke.exe\n")
		builder.WriteString("annotation-overlay-evidence-check.exe\n")
		builder.WriteString("RunAnnotationLong\n")
		builder.WriteString("LongAnnotationDurations\n")
		builder.WriteString("LongAnnotationSegments\n")
		builder.WriteString("annotation-long-snapshots\n")
		builder.WriteString("annotation-long-element-pngs\n")
		builder.WriteString("-segments=\n")
		builder.WriteString("-timeline=element-pngs\n")
		builder.WriteString("-source-type=region\n")
		builder.WriteString("-source-type=window\n")
		builder.WriteString("RECORDINGFREEDOM_FFMPEG_PATH\n")
		builder.WriteString("-source-type=region\n")
		builder.WriteString("-source-type=window\n")
		builder.WriteString("-microphone\n")
		builder.WriteString("-system\n")
		builder.WriteString("-rnnoise\n")
	}
	if strings.Contains(name, "capsule-whiteboard-entry.spec.ts") {
		builder.WriteString("opens board before recording and annotation during video recording\n")
		builder.WriteString("remains available as a board during audio recording\n")
		builder.WriteString("expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')\n")
		builder.WriteString("expectWhiteboardLaunch(page, 'annotation', '/#/annotation-overlay')\n")
		builder.WriteString("Pause recording\n")
		builder.WriteString("Resume recording\n")
		builder.WriteString("is-recording-compact\n")
	}
	if strings.Contains(name, "app/tools/THIRD_PARTY_NOTICES.txt") {
		builder.WriteString("RNNoise\n")
		builder.WriteString("License: BSD-style license\n")
		builder.WriteString("Copyright (c) 2017, Mozilla\n")
		builder.WriteString("Neither the name of the Xiph.Org Foundation\n")
		builder.WriteString("@excalidraw/excalidraw\n")
		builder.WriteString("License: MIT\n")
		builder.WriteString("Copyright (c) 2020 Excalidraw\n")
		builder.WriteString("THE SOFTWARE IS PROVIDED \"AS IS\"\n")
	}
	if strings.Contains(name, "12-annotation-overlay-platform-smoke.md") {
		builder.WriteString("录制标注 Overlay 实机验收标准\n")
		builder.WriteString("annotation-export-smoke\n")
		builder.WriteString("annotation-overlay-evidence-check\n")
		builder.WriteString("不能替代真实桌面上的透明窗口\n")
		builder.WriteString("全屏、单屏、区域、锁定窗口\n")
		builder.WriteString("多屏幕\n")
		builder.WriteString("高 DPI\n")
		builder.WriteString("点击穿透\n")
		builder.WriteString(".rfrec/annotations/\n")
		builder.WriteString("annotations/overlay-diagnostics.jsonl\n")
		builder.WriteString("exports/recording.mp4\n")
		builder.WriteString("evidence/annotation-overlay\n")
	}
	if strings.Contains(name, "annotation-overlay-evidence-check/main.go") {
		builder.WriteString("-evidence-dir\n")
		builder.WriteString("manifest annotations.enabled is required\n")
		builder.WriteString("annotation target geometry does not match source geometry\n")
		builder.WriteString("validateEvidenceREADME\n")
		builder.WriteString("README.md is missing evidence records\n")
		builder.WriteString("artifact source\n")
		builder.WriteString("validatePlatformFile\n")
		builder.WriteString("platform.txt is missing display environment records\n")
		builder.WriteString("displayResolutionPattern\n")
		builder.WriteString("requiredScreenshotEvidence\n")
		builder.WriteString("requiredRecordingEvidence\n")
		builder.WriteString("requireEvidenceNamedFiles\n")
		builder.WriteString("is missing evidence files\n")
		builder.WriteString("validateAppLog\n")
		builder.WriteString("app-log.jsonl is missing required events\n")
		builder.WriteString("recording/start-request sourceType=\n")
		builder.WriteString("annotation-overlay/show targetType=\n")
		builder.WriteString("recpackage.AnnotationEventsFile\n")
		builder.WriteString("recpackage.AnnotationOverlayDiagnosticsFile\n")
		builder.WriteString("missing %s diagnostic event\n")
		builder.WriteString("1m recording package\n")
		builder.WriteString("5m recording package\n")
		builder.WriteString("requiredEvidenceSourceChecks\n")
		builder.WriteString("source all-screens package\n")
		builder.WriteString("source screen package\n")
		builder.WriteString("source region package\n")
		builder.WriteString("source window package\n")
		builder.WriteString("requireSourceType\n")
		builder.WriteString("diagnostics.sync.screen.durationMs\n")
		builder.WriteString("missing drawing hit-regions event with full canvas rect\n")
		builder.WriteString("missing pass-through hit-regions event without full canvas rect\n")
		builder.WriteString("hitRegionsContainCanvasRect\n")
		builder.WriteString("hitRegionsContainPill\n")
		builder.WriteString("validateAnnotationEvents\n")
		builder.WriteString("missing element-created/element-updated/element-deleted event\n")
		builder.WriteString("scene-snapshot has invalid snapshotPath\n")
		builder.WriteString("findAnnotationSnapshot\n")
		builder.WriteString("recpackage.ProbeMP4\n")
		builder.WriteString("MP4 video track is missing\n")
		builder.WriteString("diagnostics.sync.screen.durationMs\n")
		builder.WriteString("exportplan.DefaultOutputPath\n")
	}
	if strings.Contains(name, "docs/README.md") {
		builder.WriteString("12-annotation-overlay-platform-smoke.md\n")
		builder.WriteString("真实桌面验收矩阵\n")
		builder.WriteString("不能只靠 `annotation-export-smoke` 宣称完成\n")
	}
	return builder.String()
}
