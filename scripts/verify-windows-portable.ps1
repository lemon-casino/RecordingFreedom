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
    Require-Entry -Entries $entries -Name "tools/desktop-doctor.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/video-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/audio-smoke.exe" | Out-Null
    Require-Entry -Entries $entries -Name "tools/run-windows-portable-smoke.ps1" | Out-Null
    if (-not $AllowMissingFFprobe) {
        Require-Entry -Entries $entries -Name "tools/ffprobe.exe" | Out-Null
    }
} finally {
    $archive.Dispose()
}

if (-not $SkipExecutableCheck) {
    $extractDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-portable-" + [System.Guid]::NewGuid().ToString("N"))
    try {
        New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
        Expand-Archive -LiteralPath $zip -DestinationPath $extractDir -Force
        $appPath = Join-Path $extractDir "recordingfreedom.exe"
        if (-not (Test-Path -LiteralPath $appPath)) {
            throw "Extracted portable zip is missing recordingfreedom.exe"
        }
        Assert-PEMetadata -Path $appPath -ExpectedMachine 0x8664 -ExpectedSubsystem 2

        $ffmpegPath = Join-Path $extractDir "tools/ffmpeg.exe"
        if (-not (Test-Path -LiteralPath $ffmpegPath)) {
            throw "Extracted portable zip is missing tools/ffmpeg.exe"
        }
        Assert-PEMetadata -Path $ffmpegPath -ExpectedMachine 0x8664

        if (-not $AllowMissingFFprobe) {
            $ffprobePath = Join-Path $extractDir "tools/ffprobe.exe"
            if (-not (Test-Path -LiteralPath $ffprobePath)) {
                throw "Extracted portable zip is missing tools/ffprobe.exe"
            }
            Assert-PEMetadata -Path $ffprobePath -ExpectedMachine 0x8664
        }

        $doctorPath = Join-Path $extractDir "tools/desktop-doctor.exe"
        if (-not (Test-Path -LiteralPath $doctorPath)) {
            throw "Extracted portable zip is missing tools/desktop-doctor.exe"
        }
        Assert-PEMetadata -Path $doctorPath -ExpectedMachine 0x8664 -ExpectedSubsystem 3

        $videoSmokePath = Join-Path $extractDir "tools/video-smoke.exe"
        if (-not (Test-Path -LiteralPath $videoSmokePath)) {
            throw "Extracted portable zip is missing tools/video-smoke.exe"
        }
        Assert-PEMetadata -Path $videoSmokePath -ExpectedMachine 0x8664 -ExpectedSubsystem 3

        $audioSmokePath = Join-Path $extractDir "tools/audio-smoke.exe"
        if (-not (Test-Path -LiteralPath $audioSmokePath)) {
            throw "Extracted portable zip is missing tools/audio-smoke.exe"
        }
        Assert-PEMetadata -Path $audioSmokePath -ExpectedMachine 0x8664 -ExpectedSubsystem 3

        $portableSmokePath = Join-Path $extractDir "tools/run-windows-portable-smoke.ps1"
        if (-not (Test-Path -LiteralPath $portableSmokePath)) {
            throw "Extracted portable zip is missing tools/run-windows-portable-smoke.ps1"
        }

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
