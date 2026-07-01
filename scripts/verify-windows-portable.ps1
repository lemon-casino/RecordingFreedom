[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ZipPath,

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

function Invoke-VersionCommand {
    param([Parameter(Mandatory = $true)][string]$Path)
    $output = & $Path -version 2>$null
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "$Path -version failed with exit code $exitCode"
    }
    return ($output | Select-Object -First 1)
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
    Require-Entry -Entries $entries -Name "recordingfreedom.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/ffmpeg.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/THIRD_PARTY_FFMPEG.txt" | Out-Null
    if (-not $AllowMissingFFprobe) {
        Require-Entry -Entries $entries -Name "tools/ffprobe.exe" | Out-Null
    }
} finally {
    $archive.Dispose()
}

if (-not $SkipExecutableCheck -and (Is-WindowsHost)) {
    $extractDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-portable-" + [System.Guid]::NewGuid().ToString("N"))
    try {
        New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
        Expand-Archive -LiteralPath $zip -DestinationPath $extractDir -Force
        $ffmpegPath = Join-Path $extractDir "tools/ffmpeg.exe"
        if (-not (Test-Path -LiteralPath $ffmpegPath)) {
            throw "Extracted portable zip is missing tools/ffmpeg.exe"
        }
        $version = Invoke-VersionCommand -Path $ffmpegPath
        Write-Host $version

        if (-not $AllowMissingFFprobe) {
            $ffprobePath = Join-Path $extractDir "tools/ffprobe.exe"
            $probeVersion = Invoke-VersionCommand -Path $ffprobePath
            Write-Host $probeVersion
        }
    } finally {
        if ($extractDir -and $extractDir.StartsWith([System.IO.Path]::GetTempPath(), [System.StringComparison]::OrdinalIgnoreCase)) {
            Remove-Item -LiteralPath $extractDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host "Windows portable zip verified: $zip"
