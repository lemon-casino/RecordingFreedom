[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$VisualDir,
    [string]$EvidenceDir = "",
    [string]$DataRoot = "",
    [string]$PlatformFile = "",
    [string]$Version = "manual",
    [string]$Commit = "",
    [string]$Artifact = "manual desktop run",
    [string]$KnownFailures = "none",
    [string]$DisplayCount = "",
    [string]$DisplayResolution = "",
    [string]$DisplayScale = "unknown",
    [string[]]$MustContain = @(),
    [string]$ToolsDir = "",
    [switch]$RequireTranslation,
    [switch]$SkipTranslations
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

function Resolve-GitCommit {
    if (-not [string]::IsNullOrWhiteSpace($Commit)) {
        return $Commit
    }
    $git = Get-Command git -ErrorAction SilentlyContinue
    if ($null -eq $git) {
        return "unknown"
    }
    try {
        $value = (& git rev-parse --short HEAD 2>$null).Trim()
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            return $value
        }
    } catch {
        return "unknown"
    }
    return "unknown"
}

function Resolve-WindowsDisplayInfo {
    $count = $DisplayCount
    $resolution = $DisplayResolution
    $scale = $DisplayScale
    try {
        Add-Type -AssemblyName System.Windows.Forms -ErrorAction Stop
        $screens = [System.Windows.Forms.Screen]::AllScreens
        if ([string]::IsNullOrWhiteSpace($count)) {
            $count = [string]$screens.Count
        }
        if ([string]::IsNullOrWhiteSpace($resolution) -and $screens.Count -gt 0) {
            $resolution = ($screens | ForEach-Object { "$($_.Bounds.Width)x$($_.Bounds.Height)" }) -join ","
        }
    } catch {
        if ([string]::IsNullOrWhiteSpace($count)) {
            $count = "unknown"
        }
    }
    if ([string]::IsNullOrWhiteSpace($scale)) {
        $scale = "unknown"
    }
    return [pscustomobject]@{
        Count = $count
        Resolution = $resolution
        Scale = $scale
    }
}

function New-GeneratedPlatformFile {
    $display = Resolve-WindowsDisplayInfo
    if ([string]::IsNullOrWhiteSpace($display.Count) -or [string]::IsNullOrWhiteSpace($display.Resolution)) {
        throw "Could not infer display count/resolution. Pass -PlatformFile or -DisplayCount/-DisplayResolution/-DisplayScale."
    }
    $temp = [System.IO.Path]::GetTempFileName()
    $os = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction SilentlyContinue
    $versionText = if ($null -ne $os) { "$($os.Caption) $($os.Version) build $($os.BuildNumber)" } else { [System.Environment]::OSVersion.VersionString }
    $content = @(
        "operating system: windows",
        "version: $versionText",
        "architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)",
        "display count: $($display.Count)",
        "resolution: $($display.Resolution)",
        "scale: $($display.Scale)",
        ""
    ) -join [Environment]::NewLine
    Set-Content -LiteralPath $temp -Value $content -Encoding UTF8
    return $temp
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$appDir = Join-Path $repoRoot "app"

$usePackagedTools = -not [string]::IsNullOrWhiteSpace($ToolsDir)
$exportTool = ""
$checkTool = ""
$planTool = ""
$sessionTool = ""
if ($usePackagedTools) {
    $resolvedToolsDir = Resolve-FullPath $ToolsDir
    if (-not (Test-Path -LiteralPath $resolvedToolsDir -PathType Container)) {
        throw "RecordingFreedom tools directory was not found: $resolvedToolsDir"
    }
    $exportTool = Join-Path $resolvedToolsDir "ocr-desktop-evidence-export.exe"
    $checkTool = Join-Path $resolvedToolsDir "ocr-desktop-evidence-check.exe"
    $planTool = Join-Path $resolvedToolsDir "ocr-desktop-evidence-plan.exe"
    $sessionTool = Join-Path $resolvedToolsDir "ocr-desktop-evidence-session.exe"
    if (-not (Test-Path -LiteralPath $exportTool -PathType Leaf)) {
        throw "RecordingFreedom tools directory is missing ocr-desktop-evidence-export.exe: $resolvedToolsDir"
    }
    if (-not (Test-Path -LiteralPath $checkTool -PathType Leaf)) {
        throw "RecordingFreedom tools directory is missing ocr-desktop-evidence-check.exe: $resolvedToolsDir"
    }
    if (-not (Test-Path -LiteralPath $planTool -PathType Leaf)) {
        throw "RecordingFreedom tools directory is missing ocr-desktop-evidence-plan.exe: $resolvedToolsDir"
    }
    if (-not (Test-Path -LiteralPath $sessionTool -PathType Leaf)) {
        throw "RecordingFreedom tools directory is missing ocr-desktop-evidence-session.exe: $resolvedToolsDir"
    }
} else {
    if (-not (Test-Path -LiteralPath $appDir -PathType Container)) {
        throw "RecordingFreedom app directory was not found: $appDir"
    }
    Require-Command "go" | Out-Null
}

$resolvedVisualDir = Resolve-FullPath $VisualDir
if (-not (Test-Path -LiteralPath $resolvedVisualDir -PathType Container)) {
    throw "Visual evidence directory does not exist: $resolvedVisualDir"
}
if ([string]::IsNullOrWhiteSpace($DataRoot)) {
    throw "DataRoot is required. Use the same RecordingFreedom data root that was passed to ocr-desktop-evidence-session start/end for this real desktop run."
}
$resolvedDataRoot = Resolve-FullPath $DataRoot
if (-not (Test-Path -LiteralPath $resolvedDataRoot -PathType Container)) {
    throw "RecordingFreedom data root does not exist: $resolvedDataRoot"
}

if ([string]::IsNullOrWhiteSpace($EvidenceDir)) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $EvidenceDir = Join-Path $repoRoot "release-out\ocr-desktop-evidence\$stamp"
} else {
    $EvidenceDir = Resolve-FullPath $EvidenceDir
}

$platformPath = ""
$removeGeneratedPlatform = $false
if ([string]::IsNullOrWhiteSpace($PlatformFile)) {
    $platformPath = New-GeneratedPlatformFile
    $removeGeneratedPlatform = $true
} else {
    $platformPath = Resolve-FullPath $PlatformFile
}
if (-not (Test-Path -LiteralPath $platformPath -PathType Leaf)) {
    throw "Platform file does not exist: $platformPath"
}

$resolvedCommit = Resolve-GitCommit

try {
    $planArgs = @(
        "-visual-dir", $resolvedVisualDir,
        "-out-dir", $EvidenceDir,
        "-data-root", $resolvedDataRoot,
        "-check"
    )
    Write-Host "OCR desktop visual capture checklist will be written as visual-capture-checklist.md/json in $EvidenceDir"
    if ($usePackagedTools) {
        & $planTool @planArgs
    } else {
        Push-Location $appDir
        try {
            & go run ".\cmd\ocr-desktop-evidence-plan" @planArgs
        } finally {
            Pop-Location
        }
    }
    if ($LASTEXITCODE -ne 0) {
        throw "ocr-desktop-evidence-plan failed; capture every required real desktop visual scene before exporting evidence. Checklist files: visual-capture-checklist.md/json in $EvidenceDir"
    }

    $exportArgs = @(
        "-evidence-dir", $EvidenceDir,
        "-visual-dir", $resolvedVisualDir,
        "-platform-file", $platformPath,
        "-version", $Version,
        "-commit", $resolvedCommit,
        "-artifact", $Artifact,
        "-known-failures", $KnownFailures,
        "-data-root", $resolvedDataRoot
    )
    if ($SkipTranslations) {
        $exportArgs += "-include-translations=false"
    }
    if ($usePackagedTools) {
        & $exportTool @exportArgs
    } else {
        Push-Location $appDir
        try {
            & go run ".\cmd\ocr-desktop-evidence-export" @exportArgs
        } finally {
            Pop-Location
        }
    }
    if ($LASTEXITCODE -ne 0) {
        throw "ocr-desktop-evidence-export failed with exit code $LASTEXITCODE"
    }

    $checkArgs = @("-evidence-dir", $EvidenceDir)
    if ($RequireTranslation) {
        $checkArgs += "-require-translation"
    }
    foreach ($text in $MustContain) {
        if (-not [string]::IsNullOrWhiteSpace($text)) {
            $checkArgs += @("-must-contain", $text)
        }
    }
    $checkReportPath = Join-Path $EvidenceDir "check-report.json"
    if ($usePackagedTools) {
        $checkOutput = & $checkTool @checkArgs
    } else {
        Push-Location $appDir
        try {
            $checkOutput = & go run ".\cmd\ocr-desktop-evidence-check" @checkArgs
        } finally {
            Pop-Location
        }
    }
    $checkExitCode = $LASTEXITCODE
    $checkOutput | Set-Content -LiteralPath $checkReportPath -Encoding UTF8
    $checkOutput | Write-Output
    if ($checkExitCode -ne 0) {
        throw "ocr-desktop-evidence-check failed with exit code $checkExitCode; report saved to $checkReportPath"
    }
} finally {
    if ($removeGeneratedPlatform) {
        Remove-Item -LiteralPath $platformPath -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "OCR desktop evidence exported and checked: $EvidenceDir"
