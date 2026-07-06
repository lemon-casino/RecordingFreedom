param(
    [Parameter(Mandatory = $true)]
    [string]$InstallerPath,

    [ValidateSet("x64", "arm64")]
    [string]$Architecture = "x64",

    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

function Resolve-FullPath([string]$Path) {
    $executionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($Path)
}

function Require-File([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Required installed file is missing: $Path"
    }
    $item = Get-Item -LiteralPath $Path
    if ($item.Length -le 0) {
        throw "Required installed file is empty: $Path"
    }
    return $item
}

function Assert-FileContains([string]$Path, [string[]]$Needles) {
    $content = Get-Content -LiteralPath $Path -Raw
    foreach ($needle in $Needles) {
        if (-not $content.Contains($needle)) {
            throw "File $Path is missing required text: $needle"
        }
    }
}

$installer = Resolve-FullPath $InstallerPath
if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) {
    throw "Windows installer does not exist: $installer"
}
if ((Get-Item -LiteralPath $installer).Length -le 0) {
    throw "Windows installer is empty: $installer"
}

$ownsInstallDir = $false
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-installer-" + [System.Guid]::NewGuid().ToString("N"))
    $ownsInstallDir = $true
}
$InstallDir = Resolve-FullPath $InstallDir
if (Test-Path -LiteralPath $InstallDir) {
    Remove-Item -LiteralPath $InstallDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

try {
    $arguments = @("/S", "/D=$InstallDir")
    $process = Start-Process -FilePath $installer -ArgumentList $arguments -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Windows installer exited with code $($process.ExitCode)"
    }

    $appPath = Join-Path $InstallDir "RecordingFreedom.exe"
    if (-not (Test-Path -LiteralPath $appPath -PathType Leaf)) {
        $appPath = Join-Path $InstallDir "recordingfreedom.exe"
    }
    Require-File $appPath | Out-Null

    $toolsDir = Join-Path $InstallDir "tools"
    $ffmpegPath = Join-Path $toolsDir "ffmpeg.exe"
    $ffprobePath = Join-Path $toolsDir "ffprobe.exe"
    $rnnoisePath = Join-Path $toolsDir "rnnoise.dll"
    $noticePath = Join-Path $toolsDir "THIRD_PARTY_FFMPEG.txt"
    $thirdPartyNoticesPath = Join-Path $toolsDir "THIRD_PARTY_NOTICES.txt"
    $goArch = if ($Architecture -eq "arm64") { "arm64" } else { "amd64" }
    $ocrWorkerPath = Join-Path $toolsDir "ocr-worker\windows-$goArch\rf-ocr-worker.exe"
    $onnxRuntimeDir = Join-Path $toolsDir "onnxruntime\windows-$goArch"
    $onnxRuntimeDll = Join-Path $onnxRuntimeDir "onnxruntime.dll"
    $onnxRuntimeProviders = Join-Path $onnxRuntimeDir "onnxruntime_providers_shared.dll"
    $onnxRuntimeNotice = Join-Path $onnxRuntimeDir "THIRD_PARTY_ONNXRUNTIME.txt"
    Require-File $ffmpegPath | Out-Null
    Require-File $ffprobePath | Out-Null
    Require-File $rnnoisePath | Out-Null
    Require-File $noticePath | Out-Null
    Require-File $thirdPartyNoticesPath | Out-Null
    Require-File $ocrWorkerPath | Out-Null
    Require-File $onnxRuntimeDll | Out-Null
    Require-File $onnxRuntimeProviders | Out-Null
    Require-File $onnxRuntimeNotice | Out-Null
    Assert-FileContains -Path $thirdPartyNoticesPath -Needles @(
        "@excalidraw/excalidraw",
        "License: MIT",
        "Copyright (c) 2020 Excalidraw",
        "THE SOFTWARE IS PROVIDED `"AS IS`""
    )
    Require-File (Join-Path $InstallDir "uninstall.exe") | Out-Null

    $version = & $ffmpegPath -version 2>$null | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($version) -or $version -notmatch "ffmpeg") {
        throw "Installed ffmpeg.exe did not report a valid version"
    }
    $ocrCapabilitiesText = & $ocrWorkerPath --capabilities --runtime-dir $onnxRuntimeDir 2>$null | Select-Object -First 1
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($ocrCapabilitiesText)) {
        throw "Installed OCR worker did not report capabilities"
    }
    $ocrCapabilities = $ocrCapabilitiesText | ConvertFrom-Json
    if (-not $ocrCapabilities.runtimeAvailable) {
        throw "Installed OCR worker did not detect bundled ONNX Runtime at $onnxRuntimeDir"
    }

    Write-Host "Windows installer verified: $installer"
} finally {
    $uninstaller = Join-Path $InstallDir "uninstall.exe"
    if (Test-Path -LiteralPath $uninstaller -PathType Leaf) {
        $uninstall = Start-Process -FilePath $uninstaller -ArgumentList @("/S") -Wait -PassThru
        if ($uninstall.ExitCode -ne 0) {
            Write-Warning "Uninstaller exited with code $($uninstall.ExitCode)"
        }
    }
    if ($ownsInstallDir -and (Test-Path -LiteralPath $InstallDir)) {
        Remove-Item -LiteralPath $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
