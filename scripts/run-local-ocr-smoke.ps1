[CmdletBinding()]
param(
    [string]$Model = "ppocrv5-mobile-zh-en",
    [string]$OutputRoot = "",
    [string]$RuntimeRoot = "",
    [switch]$IncludeCandidates,
    [switch]$ForceRuntime
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Path))
}

function Require-Command {
    param([Parameter(Mandatory = $true)][string]$Name)
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        throw "Required command '$Name' was not found in PATH"
    }
    return $command.Source
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$appDir = Join-Path $repoRoot "app"
if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path $repoRoot "release-out"
} else {
    $OutputRoot = Resolve-FullPath $OutputRoot
}
if ([string]::IsNullOrWhiteSpace($RuntimeRoot)) {
    $RuntimeRoot = Join-Path $appDir "tools\onnxruntime"
} else {
    $RuntimeRoot = Resolve-FullPath $RuntimeRoot
}

Require-Command "go" | Out-Null

$goos = (& go env GOOS).Trim()
$goarch = (& go env GOARCH).Trim()
if ($goos -ne "windows") {
    throw "run-local-ocr-smoke.ps1 currently validates the Windows local worker path only. Detected GOOS=$goos."
}
if ($goarch -ne "amd64" -and $goarch -ne "arm64") {
    throw "Unsupported Windows GOARCH for OCR smoke: $goarch"
}

$runtimeDir = Join-Path $RuntimeRoot "windows-$goarch"
$runtimeLibrary = Join-Path $runtimeDir "onnxruntime.dll"
if ($ForceRuntime -or -not (Test-Path -LiteralPath $runtimeLibrary -PathType Leaf)) {
    $runtimeArch = if ($goarch -eq "arm64") { "arm64" } else { "x64" }
    & (Join-Path $repoRoot "scripts\ensure-windows-onnxruntime.ps1") -DestinationRoot $RuntimeRoot -Architecture $runtimeArch -Force:$ForceRuntime
    if ($LASTEXITCODE -ne 0) {
        throw "ensure-windows-onnxruntime.ps1 failed with exit code $LASTEXITCODE"
    }
}
if (-not (Test-Path -LiteralPath $runtimeLibrary -PathType Leaf)) {
    throw "ONNX Runtime library is missing after prepare step: $runtimeLibrary"
}

$workerDir = Join-Path $appDir "tools\ocr-worker\windows-$goarch"
$workerPath = Join-Path $workerDir "rf-ocr-worker.exe"
New-Item -ItemType Directory -Force -Path $workerDir | Out-Null
Push-Location $appDir
try {
    & go build -trimpath -ldflags "-w -s" -o $workerPath ".\cmd\ocr-worker"
    if ($LASTEXITCODE -ne 0) {
        throw "go build cmd/ocr-worker failed with exit code $LASTEXITCODE"
    }

    $modelOutputDir = Join-Path $OutputRoot "ocr-models"
    $packageArgs = @(".\cmd\ocr-model-package", "-model", $Model, "-output", $modelOutputDir, "-force")
    if ($IncludeCandidates) {
        $packageArgs += "-include-candidates"
    }
    & go run @packageArgs
    if ($LASTEXITCODE -ne 0) {
        throw "ocr-model-package failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}

$modelZip = Get-ChildItem -LiteralPath (Join-Path $OutputRoot "ocr-models") -Filter "$Model-*.zip" |
    Sort-Object LastWriteTimeUtc -Descending |
    Select-Object -First 1
if ($null -eq $modelZip) {
    throw "Generated OCR model package was not found under $(Join-Path $OutputRoot 'ocr-models')"
}

$smokeDir = Join-Path $OutputRoot "ocr-model-smoke"
if (Test-Path -LiteralPath $smokeDir) {
    Remove-Item -LiteralPath $smokeDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $smokeDir | Out-Null
Expand-Archive -LiteralPath $modelZip.FullName -DestinationPath $smokeDir -Force

$modelDir = Join-Path $smokeDir $Model
if (-not (Test-Path -LiteralPath $modelDir -PathType Container)) {
    throw "Extracted OCR model directory was not found: $modelDir"
}

$smokeOutput = & $workerPath --smoke --runtime-dir $runtimeDir --model-dir $modelDir
if ($LASTEXITCODE -ne 0) {
    throw "rf-ocr-worker --smoke failed with exit code $LASTEXITCODE`n$smokeOutput"
}
$smoke = $smokeOutput | ConvertFrom-Json
if (-not $smoke.ok) {
    throw "rf-ocr-worker smoke did not report ok=true: $smokeOutput"
}
foreach ($text in @("RecordingFreedom", "文字识别")) {
    if (-not ([string]$smoke.plainText).Contains($text)) {
        throw "rf-ocr-worker smoke output is missing '$text': $smokeOutput"
    }
}

$previousScreenshotSmoke = $env:RF_OCR_SCREENSHOT_SMOKE
$previousWhiteboardSmoke = $env:RF_OCR_WHITEBOARD_SMOKE
$previousWorkerPath = $env:RF_OCR_WORKER_PATH
$previousRuntimeDir = $env:RF_OCR_RUNTIME_DIR
$previousModelPackage = $env:RF_OCR_MODEL_PACKAGE
$previousEvidenceDir = $env:RF_OCR_EVIDENCE_DIR
$evidenceDir = Join-Path $OutputRoot "ocr-smoke-evidence"
New-Item -ItemType Directory -Force -Path $evidenceDir | Out-Null
try {
    $env:RF_OCR_SCREENSHOT_SMOKE = "1"
    $env:RF_OCR_WHITEBOARD_SMOKE = "1"
    $env:RF_OCR_WORKER_PATH = $workerPath
    $env:RF_OCR_RUNTIME_DIR = $runtimeDir
    $env:RF_OCR_MODEL_PACKAGE = $modelZip.FullName
    $env:RF_OCR_EVIDENCE_DIR = $evidenceDir
    Push-Location $appDir
    try {
        & go test . -run "Test(ScreenshotOCRRealWorkerSmoke|WhiteboardSelectionOCRRealWorkerSmoke)" -count=1 -v
        if ($LASTEXITCODE -ne 0) {
            throw "real screenshot/whiteboard OCR smoke tests failed with exit code $LASTEXITCODE"
        }
        & go run ".\cmd\ocr-smoke-evidence-check" -evidence-dir $evidenceDir -expected-model $Model
        if ($LASTEXITCODE -ne 0) {
            throw "OCR smoke evidence check failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
} finally {
    $env:RF_OCR_SCREENSHOT_SMOKE = $previousScreenshotSmoke
    $env:RF_OCR_WHITEBOARD_SMOKE = $previousWhiteboardSmoke
    $env:RF_OCR_WORKER_PATH = $previousWorkerPath
    $env:RF_OCR_RUNTIME_DIR = $previousRuntimeDir
    $env:RF_OCR_MODEL_PACKAGE = $previousModelPackage
    $env:RF_OCR_EVIDENCE_DIR = $previousEvidenceDir
}

Write-Host "OCR local smoke passed"
Write-Host "Worker: $workerPath"
Write-Host "Runtime: $runtimeDir"
Write-Host "ModelPackage: $($modelZip.FullName)"
Write-Host "Evidence: $(Join-Path $evidenceDir 'screenshot-ocr-real-worker-smoke.json')"
Write-Host "WhiteboardEvidence: $(Join-Path $evidenceDir 'whiteboard-ocr-real-worker-smoke.json')"
Write-Host "PlainText:"
Write-Host $smoke.plainText
