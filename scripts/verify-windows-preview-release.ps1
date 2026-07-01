[CmdletBinding()]
param(
    [string]$Repository = "lemon-casino/RecordingFreedom",
    [string]$TagName = "",
    [string]$AssetPattern = "RecordingFreedom-windows-x64-*-portable.zip",
    [string]$ChecksumPattern = "SHA256SUMS-windows-x64*.txt",
    [string]$DownloadDir = "",
    [switch]$AllowMissingFFprobe,
    [switch]$SkipExecutableCheck
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

function Get-GitHubHeaders {
    $headers = @{
        "Accept" = "application/vnd.github+json"
        "User-Agent" = "RecordingFreedom-release-verifier"
        "X-GitHub-Api-Version" = "2022-11-28"
    }
    if (-not [string]::IsNullOrWhiteSpace($env:GITHUB_TOKEN)) {
        $headers["Authorization"] = "Bearer $env:GITHUB_TOKEN"
    }
    return $headers
}

function Invoke-GitHubJson {
    param([Parameter(Mandatory = $true)][string]$Uri)
    Invoke-RestMethod -Uri $Uri -Headers (Get-GitHubHeaders)
}

function Save-Url {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][string]$OutFile
    )
    New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($OutFile)) | Out-Null
    Invoke-WebRequest -Uri $Url -Headers (Get-GitHubHeaders) -OutFile $OutFile
}

function Select-Release {
    param(
        [Parameter(Mandatory = $true)][string]$Repo,
        [Parameter(Mandatory = $true)][string]$Tag,
        [Parameter(Mandatory = $true)][string]$Pattern
    )

    if (-not [string]::IsNullOrWhiteSpace($Tag)) {
        $escapedTag = [System.Uri]::EscapeDataString($Tag)
        $release = Invoke-GitHubJson -Uri "https://api.github.com/repos/$Repo/releases/tags/$escapedTag"
        if ($release.draft) {
            throw "Release $Tag is a draft and cannot be verified as a published preview."
        }
        return $release
    }

    $releases = @(Invoke-GitHubJson -Uri "https://api.github.com/repos/$Repo/releases?per_page=100")
    $release = $releases |
        Where-Object {
            -not $_.draft -and
            (@($_.assets) | Where-Object { $_.name -like $Pattern } | Select-Object -First 1)
        } |
        Sort-Object -Property published_at -Descending |
        Select-Object -First 1
    if ($null -eq $release) {
        throw "No published release in $Repo contains an asset matching $Pattern"
    }
    return $release
}

function Select-ReleaseAsset {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$Pattern
    )
    $asset = @($Release.assets) |
        Where-Object { $_.name -like $Pattern } |
        Sort-Object -Property name |
        Select-Object -First 1
    if ($null -eq $asset) {
        throw "Release $($Release.tag_name) does not contain an asset matching $Pattern"
    }
    return $asset
}

function Get-ExpectedSha256 {
    param(
        [Parameter(Mandatory = $true)][string]$ChecksumPath,
        [Parameter(Mandatory = $true)][string]$AssetName
    )
    $checksumText = Get-Content -Raw -LiteralPath $ChecksumPath
    $matchingLine = ($checksumText -split "`r?`n") |
        Where-Object { $_ -like "*$AssetName*" } |
        Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($matchingLine)) {
        throw "Checksum file $ChecksumPath does not contain an entry for $AssetName"
    }
    $match = [regex]::Match($matchingLine, "[A-Fa-f0-9]{64}")
    if (-not $match.Success) {
        throw "Checksum entry for $AssetName does not contain a SHA256 hash"
    }
    return $match.Value.ToUpperInvariant()
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$release = Select-Release -Repo $Repository -Tag $TagName -Pattern $AssetPattern
$zipAsset = Select-ReleaseAsset -Release $release -Pattern $AssetPattern
$checksumAsset = Select-ReleaseAsset -Release $release -Pattern $ChecksumPattern

if ([string]::IsNullOrWhiteSpace($DownloadDir)) {
    $safeTag = ($release.tag_name -replace "[^A-Za-z0-9_.-]", "-")
    $DownloadDir = Join-Path $repoRoot "release-out/windows-preview-$safeTag"
} else {
    $DownloadDir = Resolve-FullPath $DownloadDir
}

New-Item -ItemType Directory -Force -Path $DownloadDir | Out-Null
$zipPath = Join-Path $DownloadDir $zipAsset.name
$checksumPath = Join-Path $DownloadDir $checksumAsset.name

Write-Host "Verifying Windows release asset from $Repository@$($release.tag_name)"
Write-Host "Downloading $($zipAsset.name)"
Save-Url -Url $zipAsset.browser_download_url -OutFile $zipPath
Write-Host "Downloading $($checksumAsset.name)"
Save-Url -Url $checksumAsset.browser_download_url -OutFile $checksumPath

$expectedHash = Get-ExpectedSha256 -ChecksumPath $checksumPath -AssetName $zipAsset.name
$actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath).Hash.ToUpperInvariant()
if ($actualHash -ne $expectedHash) {
    throw "SHA256 mismatch for $($zipAsset.name). Expected $expectedHash, got $actualHash"
}
Write-Host "SHA256 verified: $actualHash"

$verifyScript = Join-Path $PSScriptRoot "verify-windows-portable.ps1"
$verifyArgs = @{ ZipPath = $zipPath }
if ($AllowMissingFFprobe) {
    $verifyArgs["AllowMissingFFprobe"] = $true
}
if ($SkipExecutableCheck) {
    $verifyArgs["SkipExecutableCheck"] = $true
}
& $verifyScript @verifyArgs

Write-Host "Verified files are available at $DownloadDir"
