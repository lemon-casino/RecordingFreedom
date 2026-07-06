[CmdletBinding()]
param(
    [string]$DestinationRoot = "",
    [ValidateSet("x64", "arm64")]
    [string]$Architecture = "x64",
    [string]$ManifestPath = "",
    [switch]$Force
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

function Save-Url {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][string]$OutFile
    )

    $curl = Get-Command curl.exe -ErrorAction SilentlyContinue
    if ($null -ne $curl) {
        & $curl.Source -fL --retry 5 --retry-delay 5 --connect-timeout 30 --output $OutFile $Url
        if ($LASTEXITCODE -ne 0) {
            throw "curl failed to download $Url with exit code $LASTEXITCODE"
        }
        return
    }

    Invoke-WebRequest -Uri $Url -OutFile $OutFile
}

function Get-ManifestTarget {
    param(
        [Parameter(Mandatory = $true)]$Manifest,
        [Parameter(Mandatory = $true)][string]$TargetKey
    )
    $property = $Manifest.targets.PSObject.Properties | Where-Object { $_.Name -eq $TargetKey } | Select-Object -First 1
    if ($null -eq $property) {
        throw "ONNX Runtime manifest does not define target $TargetKey"
    }
    return $property.Value
}

function Copy-ArchiveFile {
    param(
        [Parameter(Mandatory = $true)][string]$ExtractDir,
        [Parameter(Mandatory = $true)][string]$FileName,
        [Parameter(Mandatory = $true)][string]$DestinationDir
    )
    $file = Get-ChildItem -Path $ExtractDir -Recurse -File -Filter $FileName |
        Where-Object { $_.FullName -match "[\\/]lib[\\/]" } |
        Select-Object -First 1
    if ($null -eq $file) {
        $file = Get-ChildItem -Path $ExtractDir -Recurse -File -Filter $FileName | Select-Object -First 1
    }
    if ($null -eq $file) {
        throw "Downloaded ONNX Runtime archive did not contain $FileName"
    }
    Copy-Item -LiteralPath $file.FullName -Destination (Join-Path $DestinationDir $FileName) -Force
}

function Copy-OptionalArchiveFile {
    param(
        [Parameter(Mandatory = $true)][string]$ExtractDir,
        [Parameter(Mandatory = $true)][string]$FileName,
        [Parameter(Mandatory = $true)][string]$DestinationDir
    )
    $file = Get-ChildItem -Path $ExtractDir -Recurse -File -Filter $FileName | Select-Object -First 1
    if ($null -ne $file) {
        Copy-Item -LiteralPath $file.FullName -Destination (Join-Path $DestinationDir $FileName) -Force
    }
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
if ([string]::IsNullOrWhiteSpace($ManifestPath)) {
    $ManifestPath = Join-Path $repoRoot "third_party\onnxruntime\manifest.json"
} else {
    $ManifestPath = Resolve-FullPath $ManifestPath
}
if ([string]::IsNullOrWhiteSpace($DestinationRoot)) {
    $DestinationRoot = Join-Path $repoRoot "app\tools\onnxruntime"
} else {
    $DestinationRoot = Resolve-FullPath $DestinationRoot
}

$manifest = Get-Content -Raw -LiteralPath $ManifestPath | ConvertFrom-Json
$goArch = if ($Architecture -eq "arm64") { "arm64" } else { "amd64" }
$targetKey = "windows-$goArch"
$target = Get-ManifestTarget -Manifest $manifest -TargetKey $targetKey
$targetDir = Join-Path $DestinationRoot $targetKey
$noticePath = Join-Path $targetDir "THIRD_PARTY_ONNXRUNTIME.txt"
$requiredLibrary = Join-Path $targetDir ([string]$target.requiredLibrary)

if ((Test-Path -LiteralPath $requiredLibrary -PathType Leaf) -and (Test-Path -LiteralPath $noticePath -PathType Leaf) -and -not $Force) {
    $noticeText = Get-Content -Raw -LiteralPath $noticePath
    if ($noticeText.Contains("Version: $($manifest.version)") -and $noticeText.Contains("Target: $targetKey")) {
        Write-Host "Using existing ONNX Runtime bundle: $targetDir"
        exit 0
    }
}

$workDir = Join-Path ([System.IO.Path]::GetTempPath()) ("recordingfreedom-onnxruntime-" + [System.Guid]::NewGuid().ToString("N"))
$archivePath = Join-Path $workDir ([string]$target.archiveName)
$extractDir = Join-Path $workDir "extract"

try {
    New-Item -ItemType Directory -Force -Path $workDir, $extractDir | Out-Null

    Write-Host "Downloading ONNX Runtime $($manifest.version) for $targetKey"
    Save-Url -Url ([string]$target.downloadUrl) -OutFile $archivePath

    $actualBytes = (Get-Item -LiteralPath $archivePath).Length
    if ($actualBytes -ne [int64]$target.archiveBytes) {
        throw "ONNX Runtime archive size mismatch for $($target.archiveName). Expected $($target.archiveBytes), got $actualBytes"
    }
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    if ($actualHash -ne ([string]$target.archiveSha256).ToLowerInvariant()) {
        throw "ONNX Runtime SHA256 mismatch for $($target.archiveName). Expected $($target.archiveSha256), got $actualHash"
    }

    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force

    if (Test-Path -LiteralPath $targetDir) {
        Remove-Item -LiteralPath $targetDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null

    foreach ($library in @($target.libraryFiles)) {
        Copy-ArchiveFile -ExtractDir $extractDir -FileName ([string]$library) -DestinationDir $targetDir
    }
    foreach ($name in @("LICENSE", "ThirdPartyNotices.txt", "VERSION_NUMBER", "GIT_COMMIT_ID")) {
        Copy-OptionalArchiveFile -ExtractDir $extractDir -FileName $name -DestinationDir $targetDir
    }

    $notice = @"
RecordingFreedom bundled ONNX Runtime dependency

Name: $($manifest.name)
Version: $($manifest.version)
Target: $targetKey
Source: $($target.downloadUrl)
Release: $($manifest.source)
License: $($manifest.license)
Archive: $($target.archiveName)
ArchiveSHA256: $actualHash
RetrievedAtUtc: $((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))

ONNX Runtime is provided by Microsoft under the license terms shipped in
this directory. Review LICENSE and ThirdPartyNotices.txt before publishing
a public, signed release.
"@
    Set-Content -LiteralPath $noticePath -Value $notice -Encoding UTF8

    if (-not (Test-Path -LiteralPath $requiredLibrary -PathType Leaf)) {
        throw "ONNX Runtime install did not create required library: $requiredLibrary"
    }
    Write-Host "Bundled ONNX Runtime ready: $targetDir"
} finally {
    Remove-Item -LiteralPath $workDir -Recurse -Force -ErrorAction SilentlyContinue
}
