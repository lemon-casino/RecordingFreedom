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
	writeWorkflow(t, root, "app/build/windows/Taskfile.yml", workflowFixture("app/build/windows/Taskfile.yml"))
	writeWorkflow(t, root, "scripts/verify-windows-portable.ps1", workflowFixture("scripts/verify-windows-portable.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-preview-release.ps1", workflowFixture("scripts/verify-windows-preview-release.ps1"))

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
	writeWorkflow(t, root, "app/build/windows/Taskfile.yml", workflowFixture("app/build/windows/Taskfile.yml"))
	writeWorkflow(t, root, "scripts/verify-windows-portable.ps1", workflowFixture("scripts/verify-windows-portable.ps1"))
	writeWorkflow(t, root, "scripts/verify-windows-preview-release.ps1", workflowFixture("scripts/verify-windows-preview-release.ps1"))

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
	builder.WriteString("RNNOISE_TAG: rnnoise_native\n")
	builder.WriteString("CGO_ENABLED=1 wails3 build -tags\n")
	builder.WriteString("CGO_ENABLED=1 wails3 task windows:build EXTRA_TAGS=\"${RNNOISE_TAG}\" CGO_ENABLED=1\n")
	builder.WriteString("gtk3,${RNNOISE_TAG}\n")
	builder.WriteString("-require-rnnoise\n")
	builder.WriteString("pacman -S --noconfirm --needed mingw-w64-x86_64-gcc\n")
	builder.WriteString("./scripts/ensure-windows-ffmpeg.ps1\n")
	builder.WriteString("-require-video -require-rnnoise\n")
	if strings.Contains(name, "release.yml") {
		builder.WriteString("RNNoise native DSP is compiled into release artifacts\n")
		builder.WriteString("tools/ffmpeg.exe\n")
		builder.WriteString("tools/ffprobe.exe\n")
		builder.WriteString("tools/THIRD_PARTY_FFMPEG.txt\n")
		builder.WriteString("./scripts/verify-windows-portable.ps1\n")
	}
	if strings.Contains(name, "Taskfile.yml") {
		builder.WriteString("-ldflags=\"-w -s -H windowsgui\"\n")
	}
	if strings.Contains(name, "verify-windows-portable.ps1") {
		builder.WriteString("Assert-PEMetadata\n")
		builder.WriteString("ExpectedSubsystem 2\n")
		builder.WriteString("0x8664\n")
		builder.WriteString("recordingfreedom.exe\n")
	}
	if strings.Contains(name, "verify-windows-preview-release.ps1") {
		builder.WriteString("api.github.com/repos\n")
		builder.WriteString("SHA256SUMS-windows-x64\n")
		builder.WriteString("Get-FileHash -Algorithm SHA256\n")
		builder.WriteString("verify-windows-portable.ps1\n")
	}
	return builder.String()
}
