param(
    [Parameter(Mandatory = $true)]
    [string]$InstallerPath,

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
    $noticePath = Join-Path $toolsDir "THIRD_PARTY_FFMPEG.txt"
    Require-File $ffmpegPath | Out-Null
    Require-File $ffprobePath | Out-Null
    Require-File $noticePath | Out-Null
    Require-File (Join-Path $InstallDir "uninstall.exe") | Out-Null

    $version = & $ffmpegPath -version 2>$null | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($version) -or $version -notmatch "ffmpeg") {
        throw "Installed ffmpeg.exe did not report a valid version"
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
