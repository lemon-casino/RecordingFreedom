[CmdletBinding()]
param(
    [string]$Repository = "lemon-casino/RecordingFreedom",
    [string]$TagName = "",
    [ValidateSet("all", "windows", "macos", "linux", "ocr-models")]
    [string[]]$Targets = @("all"),
    [string[]]$Architectures = @("x64", "arm64"),
    [string]$DownloadDir = "",
    [switch]$ChecksumOnly,
    [switch]$RunWindowsInstallers,
    [switch]$RequireContentVerification,
    [switch]$AllowMissingFFprobe,
    [switch]$SkipExecutableCheck,
    [string]$OcrDesktopEvidenceVisualDir = "",
    [string]$OcrDesktopEvidenceToolsDir = "",
    [string]$OcrDesktopEvidenceDataRoot = "",
    [string]$OcrDesktopEvidenceOutputDir = "",
    [string[]]$OcrDesktopEvidenceMustContain = @(),
    [switch]$OcrDesktopEvidenceRequireTranslation
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Normalize-Architectures {
    param([string[]]$Values)
    $normalized = New-Object System.Collections.Generic.List[string]
    foreach ($value in $Values) {
        foreach ($part in ([string]$value -split ",")) {
            $arch = $part.Trim().ToLowerInvariant()
            if ([string]::IsNullOrWhiteSpace($arch)) {
                continue
            }
            switch ($arch) {
                "x64" { $arch = "x64" }
                "amd64" { $arch = "x64" }
                "arm64" { $arch = "arm64" }
                "aarch64" { $arch = "arm64" }
                default {
                    throw "Unsupported release artifact architecture: $part. Expected x64 or arm64."
                }
            }
            if (-not $normalized.Contains($arch)) {
                $normalized.Add($arch)
            }
        }
    }
    if ($normalized.Count -eq 0) {
        throw "At least one release artifact architecture is required."
    }
    return [string[]]$normalized.ToArray()
}

$Architectures = Normalize-Architectures -Values $Architectures

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
        "User-Agent" = "RecordingFreedom-release-artifacts-verifier"
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
    $partialPath = "$OutFile.part"
    $lastError = $null
    for ($attempt = 1; $attempt -le 5; $attempt++) {
        Remove-Item -LiteralPath $partialPath -Force -ErrorAction SilentlyContinue
        try {
            Invoke-WebRequest -Uri $Url -Headers (Get-GitHubHeaders) -OutFile $partialPath
            if (-not (Test-Path -LiteralPath $partialPath -PathType Leaf)) {
                throw "download did not produce an output file"
            }
            if ((Get-Item -LiteralPath $partialPath).Length -le 0) {
                throw "download produced an empty output file"
            }
            Move-Item -LiteralPath $partialPath -Destination $OutFile -Force
            return
        } catch {
            $lastError = $_
            Remove-Item -LiteralPath $partialPath -Force -ErrorAction SilentlyContinue
            if ($attempt -ge 5) {
                break
            }
            $delaySeconds = [int][Math]::Min(60, 5 * [Math]::Pow(2, $attempt - 1))
            Write-Warning "Download failed for $Url (attempt $attempt/5): $($_.Exception.Message). Retrying in ${delaySeconds}s."
            Start-Sleep -Seconds $delaySeconds
        }
    }
    throw "Failed to download $Url after 5 attempts: $($lastError.Exception.Message)"
}

function Select-Release {
    param(
        [Parameter(Mandatory = $true)][string]$Repo,
        [Parameter(Mandatory = $true)][string]$Tag
    )
    if (-not [string]::IsNullOrWhiteSpace($Tag)) {
        $escapedTag = [System.Uri]::EscapeDataString($Tag)
        $release = Invoke-GitHubJson -Uri "https://api.github.com/repos/$Repo/releases/tags/$escapedTag"
        if ($release.draft) {
            throw "Release $Tag is a draft and cannot be verified as a published release."
        }
        return $release
    }

    $releases = @(Invoke-GitHubJson -Uri "https://api.github.com/repos/$Repo/releases?per_page=100")
    $release = $releases |
        Where-Object {
            -not $_.draft -and
            (@($_.assets) | Where-Object { $_.name -like "RecordingFreedom-windows-x64-*-portable.zip" } | Select-Object -First 1)
        } |
        Sort-Object -Property published_at -Descending |
        Select-Object -First 1
    if ($null -eq $release) {
        throw "No published release in $Repo contains a Windows x64 portable RecordingFreedom asset."
    }
    return $release
}

function Select-ReleaseAsset {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$Pattern
    )
    for ($attempt = 1; $attempt -le 12; $attempt++) {
        $asset = @($Release.assets) |
            Where-Object { $_.name -like $Pattern } |
            Sort-Object -Property name |
            Select-Object -First 1
        if ($null -ne $asset) {
            return $asset
        }
        if ($attempt -lt 12 -and -not [string]::IsNullOrWhiteSpace($Release.url)) {
            Write-Host "Waiting for release asset matching $Pattern to appear on $($Release.tag_name) (attempt $attempt/12)"
            Start-Sleep -Seconds 5
            $Release = Invoke-GitHubJson -Uri $Release.url
        }
    }
    throw "Release $($Release.tag_name) does not contain an asset matching $Pattern"
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

function Test-TargetSelected {
    param([Parameter(Mandatory = $true)][string]$Target)
    return ($Targets -contains "all") -or ($Targets -contains $Target)
}

function Get-ChecksumAssetPath {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $asset = Select-ReleaseAsset -Release $Release -Pattern $Pattern
    $path = Join-Path $Destination $asset.name
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        Write-Host "Downloading $($asset.name)"
        Save-Url -Url $asset.browser_download_url -OutFile $path
    }
    return $path
}

function Save-And-VerifyAsset {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$AssetPattern,
        [Parameter(Mandatory = $true)][string]$ChecksumPattern,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $asset = Select-ReleaseAsset -Release $Release -Pattern $AssetPattern
    $assetPath = Join-Path $Destination $asset.name
    if (-not (Test-Path -LiteralPath $assetPath -PathType Leaf)) {
        Write-Host "Downloading $($asset.name)"
        Save-Url -Url $asset.browser_download_url -OutFile $assetPath
    }
    $checksumPath = Get-ChecksumAssetPath -Release $Release -Pattern $ChecksumPattern -Destination $Destination
    $expectedHash = Get-ExpectedSha256 -ChecksumPath $checksumPath -AssetName $asset.name
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $assetPath).Hash.ToUpperInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "SHA256 mismatch for $($asset.name). Expected $expectedHash, got $actualHash"
    }
    Write-Host "SHA256 verified: $($asset.name) $actualHash"
    return [pscustomobject]@{
        Name = $asset.name
        Path = $assetPath
        Checksum = $checksumPath
        SHA256 = $actualHash
    }
}

function Require-ContentVerification {
    param(
        [Parameter(Mandatory = $true)][string]$AssetName,
        [Parameter(Mandatory = $true)][string]$Reason
    )
    if ($RequireContentVerification) {
        throw "Content verification for $AssetName is required but unavailable: $Reason"
    }
    Write-Warning "Skipping content verification for ${AssetName}: $Reason"
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

function Can-ExecuteUnixTarget {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("macos", "linux")]
        [string]$Platform,
        [Parameter(Mandatory = $true)]
        [string]$Architecture
    )
    if ($Platform -eq "macos" -and -not $IsMacOS) {
        return $false
    }
    if ($Platform -eq "linux" -and -not $IsLinux) {
        return $false
    }
    $hostArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    if ($Architecture -eq "arm64") {
        return $hostArch -eq "arm64" -or $hostArch -eq "aarch64"
    }
    return $hostArch -eq "x64" -or $hostArch -eq "amd64"
}

function Can-ExecuteMacOSTarget {
    param([Parameter(Mandatory = $true)][string]$Architecture)
    return Can-ExecuteUnixTarget -Platform "macos" -Architecture $Architecture
}

function Can-ExecuteLinuxTarget {
    param([Parameter(Mandatory = $true)][string]$Architecture)
    return Can-ExecuteUnixTarget -Platform "linux" -Architecture $Architecture
}

function Reset-ExtractionDirectory {
    param([Parameter(Mandatory = $true)][string]$Destination)
    $resolved = Resolve-FullPath $Destination
    if (Test-Path -LiteralPath $resolved) {
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $resolved | Out-Null
    return $resolved
}

function Expand-WindowsPortableToolsDir {
    param(
        [Parameter(Mandatory = $true)][string]$ZipPath,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $Destination = Reset-ExtractionDirectory -Destination $Destination
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $Destination -Force
    $toolsPath = Join-Path $Destination "tools"
    foreach ($tool in @("ocr-desktop-evidence-export.exe", "ocr-desktop-evidence-check.exe", "ocr-desktop-evidence-plan.exe", "ocr-desktop-evidence-session.exe")) {
        $toolPath = Join-Path $toolsPath $tool
        if (-not (Test-Path -LiteralPath $toolPath -PathType Leaf)) {
            throw "Windows portable OCR desktop evidence tools were not found after extraction: $toolPath"
        }
    }
    return $toolsPath
}

function Expand-MacOSAppToolsDir {
    param(
        [Parameter(Mandatory = $true)][string]$ZipPath,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $Destination = Reset-ExtractionDirectory -Destination $Destination
    $ditto = Get-Command ditto -ErrorAction SilentlyContinue
    if ($null -ne $ditto) {
        & $ditto.Source -x -k $ZipPath $Destination
        if ($LASTEXITCODE -ne 0) {
            throw "ditto failed to extract macOS app zip: $ZipPath"
        }
    } else {
        Expand-Archive -LiteralPath $ZipPath -DestinationPath $Destination -Force
    }
    $appPath = Get-ChildItem -LiteralPath $Destination -Recurse -Directory -Filter "RecordingFreedom.app" |
        Select-Object -First 1
    if ($null -eq $appPath) {
        throw "RecordingFreedom.app was not found after extracting macOS release asset: $ZipPath"
    }
    $toolsPath = Join-Path $appPath.FullName "Contents/MacOS/tools"
    foreach ($tool in @("ocr-desktop-evidence-export", "ocr-desktop-evidence-check", "ocr-desktop-evidence-plan", "ocr-desktop-evidence-session")) {
        $toolPath = Join-Path $toolsPath $tool
        if (-not (Test-Path -LiteralPath $toolPath -PathType Leaf)) {
            throw "macOS app OCR desktop evidence tools were not found after extraction: $toolPath"
        }
    }
    return $toolsPath
}

function Expand-LinuxPortableToolsDir {
    param(
        [Parameter(Mandatory = $true)][string]$ArchivePath,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $Destination = Reset-ExtractionDirectory -Destination $Destination
    $tar = Get-Command tar -ErrorAction SilentlyContinue
    if ($null -eq $tar) {
        throw "tar is required to extract Linux portable OCR desktop evidence tools."
    }
    & $tar.Source -xzf $ArchivePath -C $Destination
    if ($LASTEXITCODE -ne 0) {
        throw "tar failed to extract Linux portable archive: $ArchivePath"
    }
    $portableRoot = Get-ChildItem -LiteralPath $Destination -Directory -Filter "RecordingFreedom-linux-*" |
        Select-Object -First 1
    if ($null -eq $portableRoot) {
        throw "RecordingFreedom-linux-* root directory was not found after extracting Linux release asset: $ArchivePath"
    }
    $toolsPath = Join-Path $portableRoot.FullName "tools"
    foreach ($tool in @("ocr-desktop-evidence-export", "ocr-desktop-evidence-check", "ocr-desktop-evidence-plan", "ocr-desktop-evidence-session")) {
        $toolPath = Join-Path $toolsPath $tool
        if (-not (Test-Path -LiteralPath $toolPath -PathType Leaf)) {
            throw "Linux portable OCR desktop evidence tools were not found after extraction: $toolPath"
        }
    }
    return $toolsPath
}

function Invoke-PowerShellVerifier {
    param(
        [Parameter(Mandatory = $true)][string]$ScriptPath,
        [Parameter(Mandatory = $true)][hashtable]$Arguments
    )
    & $ScriptPath @Arguments
}

function Invoke-BashVerifier {
    param(
        [Parameter(Mandatory = $true)][string]$ScriptPath,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )
    $bash = Get-Command bash -ErrorAction SilentlyContinue
    if ($null -eq $bash) {
        return $false
    }
    & $bash.Source $ScriptPath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$ScriptPath failed with exit code $LASTEXITCODE"
    }
    return $true
}

function Invoke-OcrDesktopEvidenceExport {
    param(
        [Parameter(Mandatory = $true)][string]$RepoRoot,
        [Parameter(Mandatory = $true)][string]$VisualDir,
        [Parameter(Mandatory = $true)][string]$ToolsDir,
        [Parameter(Mandatory = $true)][string]$EvidenceDir,
        [string]$DataRoot = "",
        [string[]]$MustContain = @(),
        [switch]$RequireTranslation,
        [string]$Artifact = "release artifact desktop OCR evidence"
    )
    $visualPath = Resolve-FullPath $VisualDir
    if (-not (Test-Path -LiteralPath $visualPath -PathType Container)) {
        throw "OCR desktop evidence visual directory does not exist: $visualPath"
    }
    $toolsPath = Resolve-FullPath $ToolsDir
    if (-not (Test-Path -LiteralPath $toolsPath -PathType Container)) {
        throw "OCR desktop evidence tools directory does not exist: $toolsPath"
    }
    $evidencePath = Resolve-FullPath $EvidenceDir
    if ([string]::IsNullOrWhiteSpace($DataRoot)) {
        throw "OCR desktop evidence DataRoot is required when OCR desktop evidence verification is requested. Use the same data root that was passed to ocr-desktop-evidence-session start/end."
    }
    $dataRootPath = Resolve-FullPath $DataRoot
    if (-not (Test-Path -LiteralPath $dataRootPath -PathType Container)) {
        throw "OCR desktop evidence data root does not exist: $dataRootPath"
    }
    $windowsExportTool = Join-Path $toolsPath "ocr-desktop-evidence-export.exe"
    $windowsCheckTool = Join-Path $toolsPath "ocr-desktop-evidence-check.exe"
    $windowsPlanTool = Join-Path $toolsPath "ocr-desktop-evidence-plan.exe"
    $windowsSessionTool = Join-Path $toolsPath "ocr-desktop-evidence-session.exe"
    $unixExportTool = Join-Path $toolsPath "ocr-desktop-evidence-export"
    $unixCheckTool = Join-Path $toolsPath "ocr-desktop-evidence-check"
    $unixPlanTool = Join-Path $toolsPath "ocr-desktop-evidence-plan"
    $unixSessionTool = Join-Path $toolsPath "ocr-desktop-evidence-session"
    if ((Test-Path -LiteralPath $windowsExportTool -PathType Leaf) -and (Test-Path -LiteralPath $windowsCheckTool -PathType Leaf) -and (Test-Path -LiteralPath $windowsPlanTool -PathType Leaf) -and (Test-Path -LiteralPath $windowsSessionTool -PathType Leaf)) {
        $scriptPath = Join-Path $RepoRoot "scripts/export-ocr-desktop-evidence.ps1"
        $args = @{
            VisualDir = $visualPath
            EvidenceDir = $evidencePath
            DataRoot = $dataRootPath
            ToolsDir = $toolsPath
            Artifact = $Artifact
        }
        if ($RequireTranslation) {
            $args["RequireTranslation"] = $true
        }
        if ($MustContain.Count -gt 0) {
            $args["MustContain"] = $MustContain
        }
        Invoke-PowerShellVerifier -ScriptPath $scriptPath -Arguments $args
    } elseif ((Test-Path -LiteralPath $unixExportTool -PathType Leaf) -and (Test-Path -LiteralPath $unixCheckTool -PathType Leaf) -and (Test-Path -LiteralPath $unixPlanTool -PathType Leaf) -and (Test-Path -LiteralPath $unixSessionTool -PathType Leaf)) {
        $scriptPath = Join-Path $RepoRoot "scripts/export-ocr-desktop-evidence.sh"
        $args = @(
            "--visual-dir", $visualPath,
            "--evidence-dir", $evidencePath,
            "--data-root", $dataRootPath,
            "--tools-dir", $toolsPath,
            "--artifact", $Artifact
        )
        if ($RequireTranslation) {
            $args += "--require-translation"
        }
        foreach ($text in $MustContain) {
            if (-not [string]::IsNullOrWhiteSpace($text)) {
                $args += @("--must-contain", $text)
            }
        }
        $ok = Invoke-BashVerifier -ScriptPath $scriptPath -Arguments $args
        if (-not $ok) {
            throw "bash is required to run Unix OCR desktop evidence tools from $toolsPath"
        }
    } else {
        throw "OCR desktop evidence tools directory must contain ocr-desktop-evidence-export/check/plan/session executables: $toolsPath"
    }
    $checkReport = Join-Path $evidencePath "check-report.json"
    if (-not (Test-Path -LiteralPath $checkReport -PathType Leaf)) {
        throw "OCR desktop evidence check report was not produced: $checkReport"
    }
    return [pscustomobject]@{
        Target = "ocr-desktop-evidence"
        Name = "OCR desktop evidence"
        Path = $evidencePath
        ToolsDir = $toolsPath
        VisualDir = $visualPath
        CheckReport = $checkReport
        ContentVerified = $true
        ContentVerification = "export-ocr-desktop-evidence with release/package tools"
    }
}

function Verify-ModelPackageAsset {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    $model = Save-And-VerifyAsset -Release $Release `
        -AssetPattern "ppocrv5-mobile-zh-en-*.zip" `
        -ChecksumPattern "SHA256SUMS-ocr-models.txt" `
        -Destination $Destination
    $catalogAsset = Select-ReleaseAsset -Release $Release -Pattern "ocr-model-catalog.json"
    $catalogPath = Join-Path $Destination $catalogAsset.name
    if (-not (Test-Path -LiteralPath $catalogPath -PathType Leaf)) {
        Write-Host "Downloading $($catalogAsset.name)"
        Save-Url -Url $catalogAsset.browser_download_url -OutFile $catalogPath
    }
    if ((Get-Item -LiteralPath $catalogPath).Length -le 0) {
        throw "OCR model catalog is empty: $catalogPath"
    }
    $catalogText = Get-Content -Raw -LiteralPath $catalogPath
    foreach ($needle in @("ppocrv5-mobile-zh-en", $model.Name, '"sha256"', '"bytes"', '"package"')) {
        if (-not $catalogText.Contains($needle)) {
            throw "OCR model catalog $catalogPath is missing required text: $needle"
        }
    }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [System.IO.Compression.ZipFile]::OpenRead($model.Path)
    try {
        $entries = @($archive.Entries | ForEach-Object { $_.FullName.Replace("\", "/").TrimStart("/") })
        foreach ($entry in @(
            "ppocrv5-mobile-zh-en/manifest.json",
            "ppocrv5-mobile-zh-en/det.onnx",
            "ppocrv5-mobile-zh-en/cls.onnx",
            "ppocrv5-mobile-zh-en/rec.onnx",
            "ppocrv5-mobile-zh-en/keys.txt",
            "ppocrv5-mobile-zh-en/smoke.png",
            "ppocrv5-mobile-zh-en/smoke.expected.json"
        )) {
            if (-not ($entries -contains $entry)) {
                throw "OCR model package $($model.Name) is missing required entry: $entry"
            }
        }
    } finally {
        $archive.Dispose()
    }
    Write-Host "OCR model package verified: $($model.Name)"
    return [pscustomobject]@{
        Target = "ocr-models"
        Name = $model.Name
        Path = $model.Path
        SHA256 = $model.SHA256
        ContentVerified = $true
        ContentVerification = "model package zip entries and release catalog"
    }
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$release = Select-Release -Repo $Repository -Tag $TagName
if ([string]::IsNullOrWhiteSpace($DownloadDir)) {
    $safeTag = ($release.tag_name -replace "[^A-Za-z0-9_.-]", "-")
    $DownloadDir = Join-Path $repoRoot "release-out/release-artifacts-$safeTag"
} else {
    $DownloadDir = Resolve-FullPath $DownloadDir
}
New-Item -ItemType Directory -Force -Path $DownloadDir | Out-Null

$results = New-Object System.Collections.Generic.List[object]
Write-Host "Verifying RecordingFreedom release assets from $Repository@$($release.tag_name)"

$modelVerification = $null
$autoOcrDesktopEvidenceToolsDir = ""
$autoOcrDesktopEvidenceArtifact = ""
if (Test-TargetSelected "ocr-models") {
    $modelVerification = Verify-ModelPackageAsset -Release $release -Destination $DownloadDir
    $results.Add($modelVerification)
}

foreach ($arch in $Architectures) {
    if (Test-TargetSelected "windows") {
        $portable = Save-And-VerifyAsset -Release $release `
            -AssetPattern "RecordingFreedom-windows-$arch-*-portable.zip" `
            -ChecksumPattern "SHA256SUMS-windows-$arch*.txt" `
            -Destination $DownloadDir
        $contentVerified = $false
        $contentMessage = "checksum only"
        if (-not $ChecksumOnly) {
            $verifyScript = Join-Path $PSScriptRoot "verify-windows-portable.ps1"
            $verifyArgs = @{ ZipPath = $portable.Path; Architecture = $arch }
            if ($AllowMissingFFprobe) {
                $verifyArgs["AllowMissingFFprobe"] = $true
            }
            if ($SkipExecutableCheck) {
                $verifyArgs["SkipExecutableCheck"] = $true
            }
            if ($null -ne $modelVerification -and -not [string]::IsNullOrWhiteSpace($modelVerification.Path)) {
                $verifyArgs["OcrModelPackagePath"] = $modelVerification.Path
            }
            Invoke-PowerShellVerifier -ScriptPath $verifyScript -Arguments $verifyArgs
            $contentVerified = $true
            $contentMessage = "verify-windows-portable.ps1"
        }
        $results.Add([pscustomobject]@{
            Target = "windows-portable"
            Architecture = $arch
            Name = $portable.Name
            Path = $portable.Path
            SHA256 = $portable.SHA256
            ContentVerified = $contentVerified
            ContentVerification = $contentMessage
        })
        if (-not [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceVisualDir) -and
            [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceToolsDir) -and
            [string]::IsNullOrWhiteSpace($autoOcrDesktopEvidenceToolsDir) -and
            (Can-ExecuteWindowsTarget -Architecture $arch)) {
            $toolsExtractDir = Join-Path $DownloadDir (Join-Path "ocr-desktop-evidence-tools" "windows-$arch")
            $autoOcrDesktopEvidenceToolsDir = Expand-WindowsPortableToolsDir -ZipPath $portable.Path -Destination $toolsExtractDir
            $autoOcrDesktopEvidenceArtifact = $portable.Name
            Write-Host "OCR desktop evidence tools auto-extracted from $($portable.Name): $autoOcrDesktopEvidenceToolsDir"
        }

        $installer = Save-And-VerifyAsset -Release $release `
            -AssetPattern "RecordingFreedom-windows-$arch-*-setup.exe" `
            -ChecksumPattern "SHA256SUMS-windows-$arch*.txt" `
            -Destination $DownloadDir
        $installerVerified = $false
        $installerMessage = "checksum only; pass -RunWindowsInstallers to execute the NSIS installer verifier"
        if ($RunWindowsInstallers -and -not $ChecksumOnly) {
            if (-not ($IsWindows -or $env:OS -eq "Windows_NT")) {
                Require-ContentVerification -AssetName $installer.Name -Reason "Windows installer verification requires a Windows host."
            } else {
                Invoke-PowerShellVerifier -ScriptPath (Join-Path $PSScriptRoot "verify-windows-installer.ps1") -Arguments @{
                    InstallerPath = $installer.Path
                    Architecture = $arch
                }
                $installerVerified = $true
                $installerMessage = "verify-windows-installer.ps1"
            }
        }
        $results.Add([pscustomobject]@{
            Target = "windows-installer"
            Architecture = $arch
            Name = $installer.Name
            Path = $installer.Path
            SHA256 = $installer.SHA256
            ContentVerified = $installerVerified
            ContentVerification = $installerMessage
        })
    }

    if (Test-TargetSelected "macos") {
        $mac = Save-And-VerifyAsset -Release $release `
            -AssetPattern "RecordingFreedom-macos-$arch-*-app.zip" `
            -ChecksumPattern "SHA256SUMS-macos-$arch*.txt" `
            -Destination $DownloadDir
        $macVerified = $false
        $macMessage = "checksum only"
        if (-not $ChecksumOnly) {
            if ($IsMacOS) {
                $macArgs = @($mac.Path, $arch)
                if ($null -ne $modelVerification -and -not [string]::IsNullOrWhiteSpace($modelVerification.Path)) {
                    $macArgs += $modelVerification.Path
                }
                $ok = Invoke-BashVerifier -ScriptPath (Join-Path $PSScriptRoot "verify-macos-app-zip.sh") -Arguments $macArgs
                if ($ok) {
                    $macVerified = $true
                    $macMessage = "verify-macos-app-zip.sh"
                }
            } else {
                Require-ContentVerification -AssetName $mac.Name -Reason "macOS app bundle verification requires a macOS host."
                $macMessage = "skipped: non-macOS host"
            }
        }
        $results.Add([pscustomobject]@{
            Target = "macos-app"
            Architecture = $arch
            Name = $mac.Name
            Path = $mac.Path
            SHA256 = $mac.SHA256
            ContentVerified = $macVerified
            ContentVerification = $macMessage
        })
        if (-not [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceVisualDir) -and
            [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceToolsDir) -and
            [string]::IsNullOrWhiteSpace($autoOcrDesktopEvidenceToolsDir) -and
            (Can-ExecuteMacOSTarget -Architecture $arch)) {
            $toolsExtractDir = Join-Path $DownloadDir (Join-Path "ocr-desktop-evidence-tools" "macos-$arch")
            $autoOcrDesktopEvidenceToolsDir = Expand-MacOSAppToolsDir -ZipPath $mac.Path -Destination $toolsExtractDir
            $autoOcrDesktopEvidenceArtifact = $mac.Name
            Write-Host "OCR desktop evidence tools auto-extracted from macOS app $($mac.Name): $autoOcrDesktopEvidenceToolsDir"
        }
    }

    if (Test-TargetSelected "linux") {
        $linux = Save-And-VerifyAsset -Release $release `
            -AssetPattern "RecordingFreedom-linux-$arch-*-portable.tar.gz" `
            -ChecksumPattern "SHA256SUMS-linux-$arch*.txt" `
            -Destination $DownloadDir
        $linuxVerified = $false
        $linuxMessage = "checksum only"
        if (-not $ChecksumOnly) {
            $linuxArgs = @($linux.Path, $arch)
            if ($null -ne $modelVerification -and -not [string]::IsNullOrWhiteSpace($modelVerification.Path)) {
                $linuxArgs += $modelVerification.Path
            }
            $ok = Invoke-BashVerifier -ScriptPath (Join-Path $PSScriptRoot "verify-linux-portable.sh") -Arguments $linuxArgs
            if ($ok) {
                $linuxVerified = $true
                $linuxMessage = "verify-linux-portable.sh"
            } else {
                Require-ContentVerification -AssetName $linux.Name -Reason "bash is not available for Linux archive verification."
                $linuxMessage = "skipped: bash unavailable"
            }
        }
        $results.Add([pscustomobject]@{
            Target = "linux-portable"
            Architecture = $arch
            Name = $linux.Name
            Path = $linux.Path
            SHA256 = $linux.SHA256
            ContentVerified = $linuxVerified
            ContentVerification = $linuxMessage
        })
        if (-not [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceVisualDir) -and
            [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceToolsDir) -and
            [string]::IsNullOrWhiteSpace($autoOcrDesktopEvidenceToolsDir) -and
            (Can-ExecuteLinuxTarget -Architecture $arch)) {
            $toolsExtractDir = Join-Path $DownloadDir (Join-Path "ocr-desktop-evidence-tools" "linux-$arch")
            $autoOcrDesktopEvidenceToolsDir = Expand-LinuxPortableToolsDir -ArchivePath $linux.Path -Destination $toolsExtractDir
            $autoOcrDesktopEvidenceArtifact = $linux.Name
            Write-Host "OCR desktop evidence tools auto-extracted from Linux portable $($linux.Name): $autoOcrDesktopEvidenceToolsDir"
        }
    }
}

if (-not [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceVisualDir)) {
    $ocrDesktopEvidenceToolsDirForRun = $OcrDesktopEvidenceToolsDir
    $ocrDesktopEvidenceArtifact = "release artifact verification $($release.tag_name)"
    if ([string]::IsNullOrWhiteSpace($ocrDesktopEvidenceToolsDirForRun) -and -not [string]::IsNullOrWhiteSpace($autoOcrDesktopEvidenceToolsDir)) {
        $ocrDesktopEvidenceToolsDirForRun = $autoOcrDesktopEvidenceToolsDir
        $ocrDesktopEvidenceArtifact = "release artifact verification $($release.tag_name) from $autoOcrDesktopEvidenceArtifact"
    }
    if ([string]::IsNullOrWhiteSpace($ocrDesktopEvidenceToolsDirForRun)) {
        throw "-OcrDesktopEvidenceToolsDir is required when -OcrDesktopEvidenceVisualDir is provided and no compatible release package tools were auto-extracted for this host."
    }
    $ocrEvidenceDir = $OcrDesktopEvidenceOutputDir
    if ([string]::IsNullOrWhiteSpace($ocrEvidenceDir)) {
        $ocrEvidenceDir = Join-Path $DownloadDir "ocr-desktop-evidence"
    }
    $ocrEvidence = Invoke-OcrDesktopEvidenceExport `
        -RepoRoot $repoRoot `
        -VisualDir $OcrDesktopEvidenceVisualDir `
        -ToolsDir $ocrDesktopEvidenceToolsDirForRun `
        -EvidenceDir $ocrEvidenceDir `
        -DataRoot $OcrDesktopEvidenceDataRoot `
        -MustContain $OcrDesktopEvidenceMustContain `
        -RequireTranslation:$OcrDesktopEvidenceRequireTranslation `
        -Artifact $ocrDesktopEvidenceArtifact
    $results.Add($ocrEvidence)
}

$report = [pscustomobject]@{
    Repository = $Repository
    TagName = $release.tag_name
    ReleaseURL = $release.html_url
    DownloadDir = $DownloadDir
    ChecksumOnly = [bool]$ChecksumOnly
    RunWindowsInstallers = [bool]$RunWindowsInstallers
    RequireContentVerification = [bool]$RequireContentVerification
    OcrDesktopEvidenceRequested = -not [string]::IsNullOrWhiteSpace($OcrDesktopEvidenceVisualDir)
    GeneratedAt = (Get-Date).ToUniversalTime().ToString("o")
    Results = [object[]]$results.ToArray()
}
$reportPath = Join-Path $DownloadDir "release-artifact-verification.json"
$report | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $reportPath -Encoding UTF8

Write-Host "Release artifact verification report written to $reportPath"
