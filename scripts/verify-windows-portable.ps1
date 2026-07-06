[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ZipPath,

    [ValidateSet("x64", "arm64")]
    [string]$Architecture = "x64",

    [string]$OcrModelPackagePath = "",

    [switch]$AllowMissingFFprobe,
    [switch]$SkipExecutableCheck
)

$ErrorActionPreference = "Stop"

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Path))
}

function Resolve-OptionalFullPath {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return ""
    }
    return Resolve-FullPath $Path
}

function Normalize-ZipEntryName {
    param([Parameter(Mandatory = $true)][string]$Name)
    return $Name.Replace("\", "/").TrimStart("/")
}

function Get-EntryByName {
    param(
        [Parameter(Mandatory = $true)]$Entries,
        [Parameter(Mandatory = $true)][string]$Name
    )
    $normalized = Normalize-ZipEntryName $Name
    return $Entries | Where-Object { (Normalize-ZipEntryName $_.FullName).ToLowerInvariant() -eq $normalized.ToLowerInvariant() } | Select-Object -First 1
}

function Require-Entry {
    param(
        [Parameter(Mandatory = $true)]$Entries,
        [Parameter(Mandatory = $true)][string]$Name
    )
    $entry = Get-EntryByName -Entries $Entries -Name $Name
    if ($null -eq $entry) {
        throw "Windows portable zip is missing required entry: $Name"
    }
    if ($entry.Length -le 0) {
        throw "Windows portable zip entry is empty: $Name"
    }
    return $entry
}

function Is-WindowsHost {
    return $IsWindows -or $env:OS -eq "Windows_NT"
}

function Can-ExecuteWindowsTarget {
    param([Parameter(Mandatory = $true)][string]$Architecture)
    if (-not (Is-WindowsHost)) {
        return $false
    }
    $hostArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    if ($Architecture -eq "arm64") {
        return $hostArch -eq "arm64"
    }
    return $hostArch -eq "x64" -or $hostArch -eq "amd64" -or $hostArch -eq "arm64"
}

function Invoke-VersionCommand {
    param([Parameter(Mandatory = $true)][string]$Path)
    $output = & $Path -version 2>$null
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "$Path -version failed with exit code $exitCode"
    }
    return ($output | Select-Object -First 1)
}

function Get-PEMetadata {
    param([Parameter(Mandatory = $true)][string]$Path)

    $stream = [System.IO.File]::Open($Path, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, [System.IO.FileShare]::Read)
    try {
        $reader = [System.IO.BinaryReader]::new($stream)
        try {
            $stream.Seek(0, [System.IO.SeekOrigin]::Begin) | Out-Null
            $mzSignature = $reader.ReadUInt16()
            if ($mzSignature -ne 0x5A4D) {
                throw "$Path is not a PE executable: missing MZ signature"
            }

            $stream.Seek(0x3C, [System.IO.SeekOrigin]::Begin) | Out-Null
            $peOffset = $reader.ReadInt32()
            if ($peOffset -le 0 -or $peOffset -gt ($stream.Length - 24)) {
                throw "$Path is not a PE executable: invalid PE header offset $peOffset"
            }

            $stream.Seek($peOffset, [System.IO.SeekOrigin]::Begin) | Out-Null
            $peSignature = $reader.ReadUInt32()
            if ($peSignature -ne 0x00004550) {
                throw "$Path is not a PE executable: missing PE signature"
            }

            $machine = $reader.ReadUInt16()
            $stream.Seek($peOffset + 20, [System.IO.SeekOrigin]::Begin) | Out-Null
            $optionalHeaderSize = $reader.ReadUInt16()
            if ($optionalHeaderSize -lt 70) {
                throw "$Path has an unexpectedly small PE optional header"
            }

            $optionalHeaderOffset = $peOffset + 24
            $stream.Seek($optionalHeaderOffset, [System.IO.SeekOrigin]::Begin) | Out-Null
            $optionalMagic = $reader.ReadUInt16()
            if ($optionalMagic -ne 0x010B -and $optionalMagic -ne 0x020B) {
                throw "$Path has an unsupported PE optional header magic 0x$($optionalMagic.ToString('X4'))"
            }

            $stream.Seek($optionalHeaderOffset + 68, [System.IO.SeekOrigin]::Begin) | Out-Null
            $subsystem = $reader.ReadUInt16()

            [pscustomobject]@{
                Machine = $machine
                OptionalMagic = $optionalMagic
                Subsystem = $subsystem
            }
        } finally {
            $reader.Dispose()
        }
    } finally {
        $stream.Dispose()
    }
}

function Assert-PEMetadata {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][UInt16]$ExpectedMachine,
        [int]$ExpectedSubsystem = -1
    )

    $metadata = Get-PEMetadata -Path $Path
    if ($metadata.Machine -ne $ExpectedMachine) {
        throw "$Path has PE machine 0x$($metadata.Machine.ToString('X4')), expected 0x$($ExpectedMachine.ToString('X4'))"
    }
    if ($ExpectedSubsystem -ge 0 -and $metadata.Subsystem -ne $ExpectedSubsystem) {
        throw "$Path has PE subsystem $($metadata.Subsystem), expected $ExpectedSubsystem"
    }
    Write-Host "PE verified: $Path machine=0x$($metadata.Machine.ToString('X4')) subsystem=$($metadata.Subsystem)"
}

function Assert-PowerShellScript {
    param([Parameter(Mandatory = $true)][string]$Path)

    $tokens = $null
    $errors = $null
    [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$errors) | Out-Null
    if ($errors.Count -gt 0) {
        $message = ($errors | ForEach-Object { $_.Message }) -join " | "
        throw "$Path has PowerShell parse errors: $message"
    }
    Write-Host "PowerShell script parsed: $Path"
}

function Assert-FileContains {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string[]]$Needles
    )

    $content = Get-Content -LiteralPath $Path -Raw
    foreach ($needle in $Needles) {
        if (-not $content.Contains($needle)) {
            throw "$Path is missing required smoke runner text: $needle"
        }
    }
    Write-Host "Smoke runner content verified: $Path"
}

function Resolve-OcrModelDirectory {
    param(
        [string]$PackagePath,
        [Parameter(Mandatory = $true)][string]$ScratchRoot
    )

    $resolvedPackage = Resolve-OptionalFullPath $PackagePath
    if ([string]::IsNullOrWhiteSpace($resolvedPackage)) {
        return ""
    }
    if (-not (Test-Path -LiteralPath $resolvedPackage)) {
        throw "OCR model package path does not exist: $resolvedPackage"
    }

    if ((Get-Item -LiteralPath $resolvedPackage).PSIsContainer) {
        $candidate = Join-Path $resolvedPackage "ppocrv5-mobile-zh-en"
        if (Test-Path -LiteralPath (Join-Path $candidate "manifest.json")) {
            return $candidate
        }
        if (Test-Path -LiteralPath (Join-Path $resolvedPackage "manifest.json")) {
            return $resolvedPackage
        }
        throw "OCR model package directory is missing ppocrv5-mobile-zh-en/manifest.json: $resolvedPackage"
    }

    if ([System.IO.Path]::GetExtension($resolvedPackage).ToLowerInvariant() -ne ".zip") {
        throw "OCR model package must be a .zip file or extracted model directory: $resolvedPackage"
    }

    $modelExtractDir = Join-Path $ScratchRoot "ocr-model-package"
    New-Item -ItemType Directory -Force -Path $modelExtractDir | Out-Null
    Expand-Archive -LiteralPath $resolvedPackage -DestinationPath $modelExtractDir -Force
    $modelDir = Join-Path $modelExtractDir "ppocrv5-mobile-zh-en"
    if (-not (Test-Path -LiteralPath (Join-Path $modelDir "manifest.json"))) {
        throw "OCR model package zip is missing ppocrv5-mobile-zh-en/manifest.json: $resolvedPackage"
    }
    foreach ($required in @("det.onnx", "cls.onnx", "rec.onnx", "keys.txt", "smoke.png", "smoke.expected.json")) {
        $requiredPath = Join-Path $modelDir $required
        if (-not (Test-Path -LiteralPath $requiredPath -PathType Leaf)) {
            throw "OCR model package is missing required smoke file: ppocrv5-mobile-zh-en/$required"
        }
    }
    return $modelDir
}

function Invoke-OcrStableSmoke {
    param(
        [Parameter(Mandatory = $true)][string]$WorkerPath,
        [Parameter(Mandatory = $true)][string]$RuntimeDir,
        [Parameter(Mandatory = $true)][string]$ModelDir
    )

    $smokeOutput = & $WorkerPath --smoke --runtime-dir $RuntimeDir --model-dir $ModelDir --must-contain RecordingFreedom --must-contain 文字识别 2>$null
    $exitCode = $LASTEXITCODE
    $smokeText = $smokeOutput | Select-Object -First 1
    if ($exitCode -ne 0 -or [string]::IsNullOrWhiteSpace($smokeText)) {
        throw "OCR worker stable model smoke failed for $WorkerPath"
    }
    $smoke = $smokeText | ConvertFrom-Json
    if (-not $smoke.ok) {
        throw "OCR worker stable model smoke returned ok=false: $($smoke.error)"
    }
    if ($smoke.blocks -le 0) {
        throw "OCR worker stable model smoke returned no text blocks"
    }
    $plainText = [string]$smoke.plainText
    foreach ($expected in @("RecordingFreedom", "文字识别")) {
        if (-not $plainText.Contains($expected)) {
            throw "OCR worker stable model smoke plainText is missing $expected"
        }
    }
    $smokePreview = $plainText -replace "`r?`n", " | "
    Write-Host "OCR worker stable model smoke verified: blocks=$($smoke.blocks) text=$smokePreview"
}

$zip = Resolve-FullPath $ZipPath
if (-not (Test-Path -LiteralPath $zip)) {
    throw "Windows portable zip does not exist: $zip"
}
$zipInfo = Get-Item -LiteralPath $zip
if ($zipInfo.Length -le 0) {
    throw "Windows portable zip is empty: $zip"
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($zip)
try {
    $entries = @($archive.Entries)
    $goArch = if ($Architecture -eq "arm64") { "arm64" } else { "amd64" }
    $ocrWorkerEntry = "tools/ocr-worker/windows-$goArch/rf-ocr-worker.exe"
    $onnxRuntimeDllEntry = "tools/onnxruntime/windows-$goArch/onnxruntime.dll"
    $onnxRuntimeProvidersEntry = "tools/onnxruntime/windows-$goArch/onnxruntime_providers_shared.dll"
    $onnxRuntimeNoticeEntry = "tools/onnxruntime/windows-$goArch/THIRD_PARTY_ONNXRUNTIME.txt"
    Require-Entry -Entries $entries -Name "recordingfreedom.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ffmpeg.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/THIRD_PARTY_FFMPEG.txt" | Out-Null
    Require-Entry -Entries $entries -Name "tools/rnnoise.dll" | Out-Null
    Require-Entry -Entries $entries -Name "tools/THIRD_PARTY_NOTICES.txt" | Out-Null
    Require-Entry -Entries $entries -Name $ocrWorkerEntry | Out-Null
    Require-Entry -Entries $entries -Name $onnxRuntimeDllEntry | Out-Null
    Require-Entry -Entries $entries -Name $onnxRuntimeProvidersEntry | Out-Null
    Require-Entry -Entries $entries -Name $onnxRuntimeNoticeEntry | Out-Null
    Require-Entry -Entries $entries -Name "tools/desktop-doctor.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/video-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/audio-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/pip-export-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/annotation-export-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/annotation-overlay-evidence-check.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-desktop-evidence-export.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-desktop-evidence-check.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-desktop-evidence-plan.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-desktop-evidence-session.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-translation-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ocr-secret-store-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/run-windows-portable-smoke.ps1" | Out-Null
    if (-not $AllowMissingFFprobe) {
        Require-Entry -Entries $entries -Name "tools/ffprobe.exe" | Out-Null
    }
} finally {
    $archive.Dispose()
}

if (-not $SkipExecutableCheck) {
    $expectedMachine = if ($Architecture -eq "arm64") { [UInt16]0xAA64 } else { [UInt16]0x8664 }
    $goArch = if ($Architecture -eq "arm64") { "arm64" } else { "amd64" }
    $extractDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-portable-" + [System.Guid]::NewGuid().ToString("N"))
    try {
        New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
        Expand-Archive -LiteralPath $zip -DestinationPath $extractDir -Force
        $appPath = Join-Path $extractDir "recordingfreedom.exe"
        if (-not (Test-Path -LiteralPath $appPath)) {
            throw "Extracted portable zip is missing recordingfreedom.exe"
        }
        Assert-PEMetadata -Path $appPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 2

        $ffmpegPath = Join-Path $extractDir "tools/ffmpeg.exe"
        if (-not (Test-Path -LiteralPath $ffmpegPath)) {
            throw "Extracted portable zip is missing tools/ffmpeg.exe"
        }
        Assert-PEMetadata -Path $ffmpegPath -ExpectedMachine $expectedMachine

        $rnnoisePath = Join-Path $extractDir "tools/rnnoise.dll"
        if (-not (Test-Path -LiteralPath $rnnoisePath)) {
            throw "Extracted portable zip is missing tools/rnnoise.dll"
        }
        Assert-PEMetadata -Path $rnnoisePath -ExpectedMachine $expectedMachine

        $ocrWorkerPath = Join-Path $extractDir "tools/ocr-worker/windows-$goArch/rf-ocr-worker.exe"
        if (-not (Test-Path -LiteralPath $ocrWorkerPath)) {
            throw "Extracted portable zip is missing tools/ocr-worker/windows-$goArch/rf-ocr-worker.exe"
        }
        Assert-PEMetadata -Path $ocrWorkerPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3
        $onnxRuntimeDir = Join-Path $extractDir "tools/onnxruntime/windows-$goArch"
        $onnxRuntimeDll = Join-Path $onnxRuntimeDir "onnxruntime.dll"
        if (-not (Test-Path -LiteralPath $onnxRuntimeDll)) {
            throw "Extracted portable zip is missing tools/onnxruntime/windows-$goArch/onnxruntime.dll"
        }
        Assert-PEMetadata -Path $onnxRuntimeDll -ExpectedMachine $expectedMachine
        $onnxRuntimeProviders = Join-Path $onnxRuntimeDir "onnxruntime_providers_shared.dll"
        if (-not (Test-Path -LiteralPath $onnxRuntimeProviders)) {
            throw "Extracted portable zip is missing tools/onnxruntime/windows-$goArch/onnxruntime_providers_shared.dll"
        }
        Assert-PEMetadata -Path $onnxRuntimeProviders -ExpectedMachine $expectedMachine
        if (Can-ExecuteWindowsTarget -Architecture $Architecture) {
            $ocrCapabilitiesOutput = & $ocrWorkerPath --capabilities --runtime-dir $onnxRuntimeDir 2>$null
            $ocrCapabilitiesExitCode = $LASTEXITCODE
            $ocrCapabilitiesText = $ocrCapabilitiesOutput | Select-Object -First 1
            if ($ocrCapabilitiesExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($ocrCapabilitiesText)) {
                throw "OCR worker --capabilities failed for $ocrWorkerPath"
            }
            $ocrCapabilities = $ocrCapabilitiesText | ConvertFrom-Json
            if (-not $ocrCapabilities.runtimeAvailable) {
                throw "OCR worker did not detect bundled ONNX Runtime at $onnxRuntimeDir"
            }
            if (-not [string]::IsNullOrWhiteSpace($OcrModelPackagePath)) {
                $ocrModelDir = Resolve-OcrModelDirectory -PackagePath $OcrModelPackagePath -ScratchRoot $extractDir
                Invoke-OcrStableSmoke -WorkerPath $ocrWorkerPath -RuntimeDir $onnxRuntimeDir -ModelDir $ocrModelDir
            }
        } else {
            Write-Host "Skipping OCR worker execution for Windows $Architecture on non-compatible host; PE metadata and bundled runtime files were verified."
            if (-not [string]::IsNullOrWhiteSpace($OcrModelPackagePath)) {
                Write-Host "Skipping OCR stable model smoke for Windows $Architecture on non-compatible host."
            }
        }

        $thirdPartyNoticesPath = Join-Path $extractDir "tools/THIRD_PARTY_NOTICES.txt"
        if (-not (Test-Path -LiteralPath $thirdPartyNoticesPath)) {
            throw "Extracted portable zip is missing tools/THIRD_PARTY_NOTICES.txt"
        }
        Assert-FileContains -Path $thirdPartyNoticesPath -Needles @(
            "@excalidraw/excalidraw",
            "License: MIT",
            "Copyright (c) 2020 Excalidraw",
            "THE SOFTWARE IS PROVIDED `"AS IS`""
        )

        if (-not $AllowMissingFFprobe) {
            $ffprobePath = Join-Path $extractDir "tools/ffprobe.exe"
            if (-not (Test-Path -LiteralPath $ffprobePath)) {
                throw "Extracted portable zip is missing tools/ffprobe.exe"
            }
            Assert-PEMetadata -Path $ffprobePath -ExpectedMachine $expectedMachine
        }

        $doctorPath = Join-Path $extractDir "tools/desktop-doctor.exe"
        if (-not (Test-Path -LiteralPath $doctorPath)) {
            throw "Extracted portable zip is missing tools/desktop-doctor.exe"
        }
        Assert-PEMetadata -Path $doctorPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $videoSmokePath = Join-Path $extractDir "tools/video-smoke.exe"
        if (-not (Test-Path -LiteralPath $videoSmokePath)) {
            throw "Extracted portable zip is missing tools/video-smoke.exe"
        }
        Assert-PEMetadata -Path $videoSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $audioSmokePath = Join-Path $extractDir "tools/audio-smoke.exe"
        if (-not (Test-Path -LiteralPath $audioSmokePath)) {
            throw "Extracted portable zip is missing tools/audio-smoke.exe"
        }
        Assert-PEMetadata -Path $audioSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $pipSmokePath = Join-Path $extractDir "tools/pip-export-smoke.exe"
        if (-not (Test-Path -LiteralPath $pipSmokePath)) {
            throw "Extracted portable zip is missing tools/pip-export-smoke.exe"
        }
        Assert-PEMetadata -Path $pipSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $annotationSmokePath = Join-Path $extractDir "tools/annotation-export-smoke.exe"
        if (-not (Test-Path -LiteralPath $annotationSmokePath)) {
            throw "Extracted portable zip is missing tools/annotation-export-smoke.exe"
        }
        Assert-PEMetadata -Path $annotationSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $annotationEvidenceCheckPath = Join-Path $extractDir "tools/annotation-overlay-evidence-check.exe"
        if (-not (Test-Path -LiteralPath $annotationEvidenceCheckPath)) {
            throw "Extracted portable zip is missing tools/annotation-overlay-evidence-check.exe"
        }
        Assert-PEMetadata -Path $annotationEvidenceCheckPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $ocrDesktopEvidenceExportPath = Join-Path $extractDir "tools/ocr-desktop-evidence-export.exe"
        if (-not (Test-Path -LiteralPath $ocrDesktopEvidenceExportPath)) {
            throw "Extracted portable zip is missing tools/ocr-desktop-evidence-export.exe"
        }
        Assert-PEMetadata -Path $ocrDesktopEvidenceExportPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $ocrDesktopEvidenceCheckPath = Join-Path $extractDir "tools/ocr-desktop-evidence-check.exe"
        if (-not (Test-Path -LiteralPath $ocrDesktopEvidenceCheckPath)) {
            throw "Extracted portable zip is missing tools/ocr-desktop-evidence-check.exe"
        }
        Assert-PEMetadata -Path $ocrDesktopEvidenceCheckPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $ocrDesktopEvidencePlanPath = Join-Path $extractDir "tools/ocr-desktop-evidence-plan.exe"
        if (-not (Test-Path -LiteralPath $ocrDesktopEvidencePlanPath)) {
            throw "Extracted portable zip is missing tools/ocr-desktop-evidence-plan.exe"
        }
        Assert-PEMetadata -Path $ocrDesktopEvidencePlanPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $ocrDesktopEvidenceSessionPath = Join-Path $extractDir "tools/ocr-desktop-evidence-session.exe"
        if (-not (Test-Path -LiteralPath $ocrDesktopEvidenceSessionPath)) {
            throw "Extracted portable zip is missing tools/ocr-desktop-evidence-session.exe"
        }
        Assert-PEMetadata -Path $ocrDesktopEvidenceSessionPath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $translationSmokePath = Join-Path $extractDir "tools/ocr-translation-smoke.exe"
        if (-not (Test-Path -LiteralPath $translationSmokePath)) {
            throw "Extracted portable zip is missing tools/ocr-translation-smoke.exe"
        }
        Assert-PEMetadata -Path $translationSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $secretStoreSmokePath = Join-Path $extractDir "tools/ocr-secret-store-smoke.exe"
        if (-not (Test-Path -LiteralPath $secretStoreSmokePath)) {
            throw "Extracted portable zip is missing tools/ocr-secret-store-smoke.exe"
        }
        Assert-PEMetadata -Path $secretStoreSmokePath -ExpectedMachine $expectedMachine -ExpectedSubsystem 3

        $portableSmokePath = Join-Path $extractDir "tools/run-windows-portable-smoke.ps1"
        if (-not (Test-Path -LiteralPath $portableSmokePath)) {
            throw "Extracted portable zip is missing tools/run-windows-portable-smoke.ps1"
        }
        Assert-PowerShellScript -Path $portableSmokePath
        Assert-FileContains -Path $portableSmokePath -Needles @(
            "desktop-doctor.exe",
            "video-smoke.exe",
            "audio-smoke.exe",
            "pip-export-smoke.exe",
            "annotation-export-smoke.exe",
            "annotation-overlay-evidence-check.exe",
            "ocr-desktop-evidence-export.exe",
            "ocr-desktop-evidence-check.exe",
            "ocr-desktop-evidence-plan.exe",
            "ocr-desktop-evidence-session.exe",
            "ocr-translation-smoke.exe",
            "ocr-secret-store-smoke.exe",
            "translation-smoke",
            "secret-store-smoke",
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
            "-rnnoise"
        )

        if (Is-WindowsHost) {
            $version = Invoke-VersionCommand -Path $ffmpegPath
            Write-Host $version

            if (-not $AllowMissingFFprobe) {
                $probeVersion = Invoke-VersionCommand -Path $ffprobePath
                Write-Host $probeVersion
            }
        }
    } finally {
        if ($extractDir -and $extractDir.StartsWith([System.IO.Path]::GetTempPath(), [System.StringComparison]::OrdinalIgnoreCase)) {
            Remove-Item -LiteralPath $extractDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host "Windows portable zip verified: $zip"
