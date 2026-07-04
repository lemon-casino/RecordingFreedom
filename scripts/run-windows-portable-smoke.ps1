[CmdletBinding()]
param(
    [string]$PortableDir = "",
    [string]$DataDir = "",
    [string]$Duration = "6s",
    [string]$PauseAfter = "2s",
    [string]$PauseDuration = "1s",
    [string]$AnnotationDuration = "5s",
    [int]$AnnotationSegments = 5,
    [switch]$RunAnnotationLong,
    [string[]]$LongAnnotationDurations = @("1m", "5m"),
    [int]$LongAnnotationSegments = 60,
    [switch]$SkipAllScreens,
    [switch]$SkipRegion,
    [switch]$SkipWindow,
    [switch]$SkipPause,
    [switch]$SkipSystemAudio,
    [switch]$SkipMicrophone,
    [switch]$SkipAudioOnly,
    [switch]$SkipRNNoise,
    [switch]$ContinueOnError
)

$ErrorActionPreference = "Stop"

if ($AnnotationSegments -lt 2) {
    throw "AnnotationSegments must be at least 2."
}
if ($LongAnnotationSegments -lt 2) {
    throw "LongAnnotationSegments must be at least 2."
}

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Path))
}

function Get-ScriptDirectory {
    if ($PSScriptRoot) {
        return $PSScriptRoot
    }
    if ($MyInvocation.MyCommand.Path) {
        return Split-Path -Parent $MyInvocation.MyCommand.Path
    }
    return (Get-Location).Path
}

function Resolve-PortableDirectory {
    param([string]$Configured)
    if ($Configured.Trim() -ne "") {
        return Resolve-FullPath $Configured
    }

    $scriptDir = Get-ScriptDirectory
    $parentDir = Split-Path -Parent $scriptDir
    $candidates = @($scriptDir, $parentDir, (Get-Location).Path)
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath (Join-Path $candidate "recordingfreedom.exe"))) {
            return [System.IO.Path]::GetFullPath($candidate)
        }
    }
    throw "Could not find recordingfreedom.exe. Run this script from the portable root, portable tools directory, or pass -PortableDir."
}

function Require-File {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Required portable file is missing: $Path"
    }
    $item = Get-Item -LiteralPath $Path
    if ($item.Length -le 0) {
        throw "Required portable file is empty: $Path"
    }
    return $item.FullName
}

function New-Step {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )
    [pscustomobject]@{
        Name = $Name
        Path = $Path
        Arguments = $Arguments
    }
}

function Get-SafeName {
    param([Parameter(Mandatory = $true)][string]$Name)
    return ($Name.ToLowerInvariant() -replace "[^a-z0-9._-]+", "-").Trim("-")
}

function Invoke-SmokeStep {
    param(
        [Parameter(Mandatory = $true)]$Step,
        [Parameter(Mandatory = $true)][string]$ReportDir
    )

    $safeName = Get-SafeName $Step.Name
    $stdoutPath = Join-Path $ReportDir "$safeName.stdout.txt"
    $stderrPath = Join-Path $ReportDir "$safeName.stderr.txt"
    $startedAt = Get-Date
    Write-Host "==> $($Step.Name)"
    Write-Host "    $($Step.Path) $($Step.Arguments -join ' ')"

    & $Step.Path @($Step.Arguments) 1> $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    $endedAt = Get-Date
    $ok = $exitCode -eq 0

    if ($ok) {
        Write-Host "    OK"
    } else {
        Write-Host "    FAILED with exit code $exitCode"
        if (Test-Path -LiteralPath $stderrPath) {
            Get-Content -LiteralPath $stderrPath | Select-Object -Last 20 | ForEach-Object { Write-Host "    $_" }
        }
    }

    [pscustomobject]@{
        name = $Step.Name
        ok = $ok
        exitCode = $exitCode
        startedAt = $startedAt.ToUniversalTime().ToString("o")
        endedAt = $endedAt.ToUniversalTime().ToString("o")
        durationMs = [int64]($endedAt - $startedAt).TotalMilliseconds
        executable = $Step.Path
        arguments = $Step.Arguments
        stdout = $stdoutPath
        stderr = $stderrPath
    }
}

$portableRoot = Resolve-PortableDirectory $PortableDir
$toolsDir = Join-Path $portableRoot "tools"
$appPath = Require-File (Join-Path $portableRoot "recordingfreedom.exe")
$ffmpegPath = Require-File (Join-Path $toolsDir "ffmpeg.exe")
$ffprobePath = Require-File (Join-Path $toolsDir "ffprobe.exe")
$doctorPath = Require-File (Join-Path $toolsDir "desktop-doctor.exe")
$videoSmokePath = Require-File (Join-Path $toolsDir "video-smoke.exe")
$audioSmokePath = Require-File (Join-Path $toolsDir "audio-smoke.exe")
$pipSmokePath = Require-File (Join-Path $toolsDir "pip-export-smoke.exe")
$annotationSmokePath = Require-File (Join-Path $toolsDir "annotation-export-smoke.exe")
$annotationEvidenceCheckPath = Require-File (Join-Path $toolsDir "annotation-overlay-evidence-check.exe")

if ($DataDir.Trim() -eq "") {
    $DataDir = Join-Path $portableRoot "data-smoke"
}
$DataDir = Resolve-FullPath $DataDir
$reportDir = Join-Path $DataDir "portable-smoke-report"
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null

$env:RECORDINGFREEDOM_FFMPEG_PATH = $ffmpegPath

$steps = New-Object System.Collections.Generic.List[object]
$doctorArgs = @("-data-dir", $DataDir, "-require-video")
if (-not $SkipRNNoise) {
    $doctorArgs += "-require-rnnoise"
}
$steps.Add((New-Step "desktop-doctor" $doctorPath $doctorArgs))
$steps.Add((New-Step "pip-export-pixels" $pipSmokePath @("-synthetic", "-data-dir", $DataDir, "-duration=1s", "-width=640", "-height=360", "-keep")))
$steps.Add((New-Step "annotation-export-snapshots" $annotationSmokePath @("-data-dir", $DataDir, "-duration=$AnnotationDuration", "-segments=$AnnotationSegments", "-timeline=snapshot-segments", "-source-type=region", "-source-x=-1280", "-source-y=48", "-source-display-index=2", "-source-native-id=annotation-export-region-display", "-keep")))
$steps.Add((New-Step "annotation-export-element-pngs" $annotationSmokePath @("-data-dir", $DataDir, "-duration=$AnnotationDuration", "-segments=$AnnotationSegments", "-timeline=element-pngs", "-source-type=window", "-source-id=window:annotation-export-smoke", "-keep")))
if ($RunAnnotationLong) {
    foreach ($longDuration in $LongAnnotationDurations) {
        if ($null -eq $longDuration) {
            continue
        }
        $durationValue = $longDuration.Trim()
        if ($durationValue -eq "") {
            continue
        }
        $durationName = Get-SafeName $durationValue
        if (-not $SkipRegion) {
            $steps.Add((New-Step "annotation-long-snapshots-$durationName" $annotationSmokePath @("-data-dir", $DataDir, "-duration=$durationValue", "-segments=$LongAnnotationSegments", "-timeline=snapshot-segments", "-source-type=region", "-source-x=-1280", "-source-y=48", "-source-display-index=2", "-source-native-id=annotation-long-region-display", "-keep")))
        }
        if (-not $SkipWindow) {
            $steps.Add((New-Step "annotation-long-element-pngs-$durationName" $annotationSmokePath @("-data-dir", $DataDir, "-duration=$durationValue", "-segments=$LongAnnotationSegments", "-timeline=element-pngs", "-source-type=window", "-source-id=window:annotation-long-smoke", "-keep")))
        }
    }
}
$steps.Add((New-Step "video-screen" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=screen", "-keep")))
if (-not $SkipAllScreens) {
    $steps.Add((New-Step "video-all-screens" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=all-screens", "-keep")))
}
if (-not $SkipRegion) {
    $steps.Add((New-Step "video-region" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=region", "-keep")))
}
if (-not $SkipWindow) {
    $steps.Add((New-Step "video-window" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=window", "-keep")))
}
if (-not $SkipPause) {
    $steps.Add((New-Step "video-pause-resume" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-pause-after=$PauseAfter", "-pause-duration=$PauseDuration", "-keep")))
}
if (-not $SkipSystemAudio) {
    $steps.Add((New-Step "video-system-audio" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=screen", "-system", "-keep")))
}
if (-not $SkipMicrophone) {
    $steps.Add((New-Step "video-microphone" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=screen", "-microphone", "-keep")))
}
if ((-not $SkipSystemAudio) -and (-not $SkipMicrophone)) {
    $steps.Add((New-Step "video-system-and-microphone" $videoSmokePath @("-data-dir", $DataDir, "-duration=$Duration", "-source-type=screen", "-system", "-microphone", "-keep")))
}
if (-not $SkipAudioOnly) {
    if (-not $SkipMicrophone) {
        $args = @("-root", $DataDir, "-duration=$Duration", "-microphone=true", "-system=false", "-keep")
        if (-not $SkipRNNoise) {
            $args += "-rnnoise"
        }
        $steps.Add((New-Step "audio-only-microphone" $audioSmokePath $args))
    }
    if (-not $SkipSystemAudio) {
        $steps.Add((New-Step "audio-only-system" $audioSmokePath @("-root", $DataDir, "-duration=$Duration", "-microphone=false", "-system=true", "-keep")))
    }
    if ((-not $SkipMicrophone) -and (-not $SkipSystemAudio)) {
        $args = @("-root", $DataDir, "-duration=$Duration", "-microphone=true", "-system=true", "-keep")
        if (-not $SkipRNNoise) {
            $args += "-rnnoise"
        }
        $steps.Add((New-Step "audio-only-system-and-microphone" $audioSmokePath $args))
    }
}

$results = New-Object System.Collections.Generic.List[object]
foreach ($step in $steps) {
    $result = Invoke-SmokeStep -Step $step -ReportDir $reportDir
    $results.Add($result)
    if (-not $result.ok -and -not $ContinueOnError) {
        break
    }
}

$ok = $true
foreach ($result in $results) {
    if (-not $result.ok) {
        $ok = $false
        break
    }
}

$summary = [pscustomobject]@{
    ok = $ok
    generatedAt = (Get-Date).ToUniversalTime().ToString("o")
    portableDir = $portableRoot
    app = $appPath
    ffmpeg = $ffmpegPath
    ffprobe = $ffprobePath
    annotationEvidenceCheck = $annotationEvidenceCheckPath
    dataDir = $DataDir
    reportDir = $reportDir
    duration = $Duration
    annotationDuration = $AnnotationDuration
    annotationSegments = $AnnotationSegments
    runAnnotationLong = [bool]$RunAnnotationLong
    longAnnotationDurations = $LongAnnotationDurations
    longAnnotationSegments = $LongAnnotationSegments
    continueOnError = [bool]$ContinueOnError
    steps = $results
}
$summaryPath = Join-Path $reportDir "summary.json"
$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $summaryPath -Encoding UTF8
Write-Host "Portable smoke report: $summaryPath"

if (-not $ok) {
    exit 1
}
