[CmdletBinding()]
param(
    [string]$DestinationDir = "",
    [string]$Uri = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip",
    [string]$Sha256Uri = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip.sha256",
    [switch]$Force
)

$ErrorActionPreference = "Stop"

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Path))
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if ([string]::IsNullOrWhiteSpace($DestinationDir)) {
    $DestinationDir = Join-Path $repoRoot "app/tools"
} else {
    $DestinationDir = Resolve-FullPath $DestinationDir
}

New-Item -ItemType Directory -Force -Path $DestinationDir | Out-Null
$ffmpegPath = Join-Path $DestinationDir "ffmpeg.exe"

if ((Test-Path -LiteralPath $ffmpegPath) -and -not $Force) {
    $version = (& $ffmpegPath -version 2>$null | Select-Object -First 1)
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Using existing FFmpeg: $ffmpegPath"
        Write-Host $version
        exit 0
    }
    Write-Host "Existing FFmpeg is not executable, refreshing: $ffmpegPath"
}

$workDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-ffmpeg-" + [System.Guid]::NewGuid().ToString("N"))
$zipPath = Join-Path $workDir "ffmpeg.zip"
$shaPath = Join-Path $workDir "ffmpeg.zip.sha256"
$extractDir = Join-Path $workDir "extract"

try {
    New-Item -ItemType Directory -Force -Path $workDir, $extractDir | Out-Null

    Write-Host "Downloading FFmpeg from $Uri"
    Invoke-WebRequest -Uri $Uri -OutFile $zipPath

    Write-Host "Downloading FFmpeg checksum from $Sha256Uri"
    Invoke-WebRequest -Uri $Sha256Uri -OutFile $shaPath

    $checksumText = Get-Content -Raw -LiteralPath $shaPath
    $match = [regex]::Match($checksumText, "[A-Fa-f0-9]{64}")
    if (-not $match.Success) {
        throw "Could not parse SHA256 checksum from $Sha256Uri"
    }
    $expectedHash = $match.Value.ToUpperInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath).Hash.ToUpperInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "FFmpeg SHA256 mismatch. Expected $expectedHash, got $actualHash"
    }

    Expand-Archive -LiteralPath $zipPath -DestinationPath $extractDir -Force

    $ffmpeg = Get-ChildItem -Path $extractDir -Recurse -Filter "ffmpeg.exe" |
        Where-Object { $_.FullName -match "[\\/]bin[\\/]ffmpeg\.exe$" } |
        Select-Object -First 1
    if ($null -eq $ffmpeg) {
        $ffmpeg = Get-ChildItem -Path $extractDir -Recurse -Filter "ffmpeg.exe" | Select-Object -First 1
    }
    if ($null -eq $ffmpeg) {
        throw "Downloaded FFmpeg archive did not contain ffmpeg.exe"
    }

    Copy-Item -LiteralPath $ffmpeg.FullName -Destination $ffmpegPath -Force

    $ffprobe = Get-ChildItem -Path $extractDir -Recurse -Filter "ffprobe.exe" |
        Where-Object { $_.FullName -match "[\\/]bin[\\/]ffprobe\.exe$" } |
        Select-Object -First 1
    if ($null -ne $ffprobe) {
        Copy-Item -LiteralPath $ffprobe.FullName -Destination (Join-Path $DestinationDir "ffprobe.exe") -Force
    }

    $noticePath = Join-Path $DestinationDir "THIRD_PARTY_FFMPEG.txt"
    $notice = @"
RecordingFreedom bundled FFmpeg dependency

Source: $Uri
Checksum source: $Sha256Uri
SHA256: $actualHash
RetrievedAtUtc: $((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))

FFmpeg is provided by its upstream/build distribution and is governed by
the license terms shipped by that distribution and by the FFmpeg project.
Review FFmpeg licensing before publishing a public, signed release.
"@
    Set-Content -LiteralPath $noticePath -Value $notice -Encoding UTF8

    $version = (& $ffmpegPath -version | Select-Object -First 1)
    if ($LASTEXITCODE -ne 0) {
        throw "Bundled ffmpeg.exe failed to execute"
    }
    Write-Host "Bundled FFmpeg ready: $ffmpegPath"
    Write-Host $version
} finally {
    Remove-Item -LiteralPath $workDir -Recurse -Force -ErrorAction SilentlyContinue
}
