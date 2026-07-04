[CmdletBinding()]
param(
    [ValidateSet("x64", "arm64")]
    [string]$Architecture = "arm64",

    [string]$OutputPath = "",

    [string]$Compiler = ""
)

$ErrorActionPreference = "Stop"

function Resolve-RepoRoot {
    $current = (Get-Location).Path
    while ($true) {
        if (Test-Path -LiteralPath (Join-Path $current "app\internal\audio\rnnoise\native\likely_voice_enhancer.c")) {
            return $current
        }
        $parent = Split-Path -Parent $current
        if ($parent -eq $current -or [string]::IsNullOrWhiteSpace($parent)) {
            throw "Could not find RecordingFreedom repository root from $(Get-Location)"
        }
        $current = $parent
    }
}

function Resolve-CommandPath {
    param([Parameter(Mandatory = $true)][string[]]$Names)
    foreach ($name in $Names) {
        $command = Get-Command $name -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($null -ne $command) {
            return $command.Source
        }
    }
    return ""
}

function Assert-PEMachine {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][UInt16]$ExpectedMachine
    )

    $stream = [System.IO.File]::Open($Path, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, [System.IO.FileShare]::Read)
    try {
        $reader = [System.IO.BinaryReader]::new($stream)
        try {
            if ($reader.ReadUInt16() -ne 0x5A4D) {
                throw "$Path is not a PE image"
            }
            $stream.Seek(0x3C, [System.IO.SeekOrigin]::Begin) | Out-Null
            $peOffset = $reader.ReadInt32()
            $stream.Seek($peOffset, [System.IO.SeekOrigin]::Begin) | Out-Null
            if ($reader.ReadUInt32() -ne 0x00004550) {
                throw "$Path is missing the PE signature"
            }
            $machine = $reader.ReadUInt16()
            if ($machine -ne $ExpectedMachine) {
                throw "$Path has PE machine 0x$($machine.ToString('X4')), expected 0x$($ExpectedMachine.ToString('X4'))"
            }
        } finally {
            $reader.Dispose()
        }
    } finally {
        $stream.Dispose()
    }
}

$root = Resolve-RepoRoot
$nativeDir = Join-Path $root "app\internal\audio\rnnoise\native"
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $OutputPath = Join-Path $root "app\tools\rnnoise.dll"
}
$OutputPath = [System.IO.Path]::GetFullPath($OutputPath)
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $OutputPath) | Out-Null

if ([string]::IsNullOrWhiteSpace($Compiler)) {
    if (-not [string]::IsNullOrWhiteSpace($env:CC)) {
        $Compiler = $env:CC
    } elseif ($Architecture -eq "arm64") {
        $Compiler = Resolve-CommandPath @("clang", "aarch64-w64-mingw32-clang", "gcc")
    } else {
        $Compiler = Resolve-CommandPath @("gcc", "clang", "x86_64-w64-mingw32-gcc")
    }
}
if ([string]::IsNullOrWhiteSpace($Compiler)) {
    throw "No C compiler found. Install MSYS2 MINGW64/CLANGARM64 gcc or clang before building rnnoise.dll."
}

$sources = @(
    "likely_voice_enhancer.c",
    "denoise.c",
    "rnn.c",
    "rnn_data.c",
    "pitch.c",
    "celt_lpc.c",
    "kiss_fft.c"
) | ForEach-Object { Join-Path $nativeDir $_ }

$args = @(
    "-shared",
    "-O2",
    "-std=c99",
    "-D_GNU_SOURCE",
    "-DRNNOISE_BUILD",
    "-DDLL_EXPORT",
    "-DLIKELY_VOICE_ENHANCER_BUILD_DLL",
    "-I$nativeDir"
)
if ($Architecture -eq "arm64" -and (Split-Path -Leaf $Compiler).ToLowerInvariant().Contains("clang")) {
    $args += "--target=aarch64-w64-mingw32"
}
$args += $sources
$args += "-lm"
$args += @("-o", $OutputPath)

Write-Host "Building RNNoise DLL: $Compiler $($args -join ' ')"
& $Compiler @args
if ($LASTEXITCODE -ne 0) {
    throw "RNNoise DLL build failed with exit code $LASTEXITCODE"
}

$expectedMachine = if ($Architecture -eq "arm64") { [UInt16]0xAA64 } else { [UInt16]0x8664 }
Assert-PEMachine -Path $OutputPath -ExpectedMachine $expectedMachine
Write-Host "RNNoise DLL built: $OutputPath"
