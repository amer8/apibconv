$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
Set-StrictMode -Version Latest

function Require-Env {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $Value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($Value)) {
        throw "Required environment variable '$Name' is not set."
    }
    return $Value
}

function Get-ArchiveInfo {
    $osName = Require-Env -Name "RUNNER_OS"
    $archName = Require-Env -Name "RUNNER_ARCH"

    switch ($osName) {
        "Linux" {
            $platform = "Linux"
            $extension = "tar.gz"
            $binaryName = "apibconv"
        }
        "macOS" {
            $platform = "Darwin"
            $extension = "tar.gz"
            $binaryName = "apibconv"
        }
        "Windows" {
            $platform = "Windows"
            $extension = "zip"
            $binaryName = "apibconv.exe"
        }
        default {
            throw "Unsupported runner OS '$osName'."
        }
    }

    switch ($archName) {
        "X64" { $arch = "x86_64" }
        "ARM64" { $arch = "arm64" }
        "ARM" {
            if ($osName -eq "Windows") {
                throw "Windows ARM runners are not currently supported."
            }
            $arch = "armv7"
        }
        default {
            throw "Unsupported runner architecture '$archName'."
        }
    }

    return @{
        Platform = $platform
        Extension = $extension
        BinaryName = $binaryName
        Arch = $arch
    }
}

function Resolve-FileUriPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url
    )

    try {
        $uri = [System.Uri]$Url
    } catch {
        throw "Invalid file URL '$Url'."
    }

    if (-not $uri.IsAbsoluteUri -or $uri.Scheme -ne "file") {
        throw "Expected a file URL, got '$Url'."
    }

    return $uri.LocalPath
}

function Download-File {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [Parameter(Mandatory = $true)]
        [string]$Destination
    )

    if ($Url.StartsWith("file://", [StringComparison]::OrdinalIgnoreCase)) {
        $sourcePath = Resolve-FileUriPath -Url $Url
        if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
            throw "Failed to download $Url (file not found)."
        }
        Copy-Item -LiteralPath $sourcePath -Destination $Destination -Force
        return
    }

    Invoke-WebRequest -Uri $Url -OutFile $Destination
}

function Try-Download-File {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [Parameter(Mandatory = $true)]
        [string]$Destination
    )

    if ($Url.StartsWith("file://", [StringComparison]::OrdinalIgnoreCase)) {
        $sourcePath = Resolve-FileUriPath -Url $Url
        if (Test-Path -LiteralPath $sourcePath -PathType Leaf) {
            Copy-Item -LiteralPath $sourcePath -Destination $Destination -Force
            return $true
        }

        Remove-Item -LiteralPath $Destination -Force -ErrorAction SilentlyContinue
        return $false
    }

    try {
        Download-File -Url $Url -Destination $Destination
        return $true
    } catch {
        $statusCode = $null
        if ($null -ne $_.Exception -and $null -ne $_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        if ($statusCode -eq 404) {
            return $false
        }
        throw
    }
}

function Get-ExpectedChecksum {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ChecksumPath,
        [Parameter(Mandatory = $true)]
        [string]$ArchiveName
    )

    foreach ($line in Get-Content -LiteralPath $ChecksumPath) {
        if ($line -match "^([A-Fa-f0-9]+)\s+\*?(.+)$" -and $matches[2] -eq $ArchiveName) {
            return $matches[1].ToLowerInvariant()
        }
    }

    throw "Could not find $ArchiveName in $ChecksumPath."
}

function Write-OutputValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    Add-Content -LiteralPath (Require-Env -Name "GITHUB_OUTPUT") -Value "$Name=$Value" -Encoding utf8
}

$version = Require-Env -Name "APIBCONV_VERSION"
if (-not $version.StartsWith("v")) {
    throw "Expected APIBCONV_VERSION to be a tag like v0.7.0, got '$version'."
}

$archiveInfo = Get-ArchiveInfo
$cleanVersion = $version.Substring(1)
$archiveName = "apibconv_${cleanVersion}_$($archiveInfo.Platform)_$($archiveInfo.Arch).$($archiveInfo.Extension)"
$checksumName = "${archiveName}_checksums.txt"
$bundleName = "${checksumName}.sigstore.json"
$baseUrl = "https://github.com/amer8/apibconv/releases/download/$version"
$assetBaseUrl = [Environment]::GetEnvironmentVariable("APIBCONV_ASSET_BASE_URL")
if ([string]::IsNullOrWhiteSpace($assetBaseUrl)) {
    $assetBaseUrl = $baseUrl
}
$tempRoot = Join-Path (Require-Env -Name "RUNNER_TEMP") "apibconv-$cleanVersion-$($archiveInfo.Platform)-$($archiveInfo.Arch)"
$extractDir = Join-Path $tempRoot "extract"
$installDir = Join-Path $tempRoot "bin"
$archivePath = Join-Path $tempRoot $archiveName
$checksumPath = Join-Path $tempRoot $checksumName
$bundlePath = Join-Path $tempRoot $bundleName
$installedBinaryPath = Join-Path $installDir $archiveInfo.BinaryName

Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
New-Item -ItemType Directory -Path $installDir -Force | Out-Null

Write-Host "Downloading $archiveName from $assetBaseUrl"
Download-File -Url "$assetBaseUrl/$archiveName" -Destination $archivePath
Download-File -Url "$assetBaseUrl/$checksumName" -Destination $checksumPath

$signatureVerified = $false
if (Try-Download-File -Url "$assetBaseUrl/$bundleName" -Destination $bundlePath) {
    $cosign = (Get-Command cosign -ErrorAction Stop).Source
    $certificateIdentity = "https://github.com/amer8/apibconv/.github/workflows/release.yml@refs/tags/$version"

    Write-Host "Verifying checksum signature with cosign"
    & $cosign verify-blob `
        --certificate-identity $certificateIdentity `
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" `
        --bundle $bundlePath `
        $checksumPath

    if ($LASTEXITCODE -ne 0) {
        throw "cosign verification failed for $checksumName."
    }

    $signatureVerified = $true
} else {
    Write-Warning "No Sigstore bundle found for $checksumName. Falling back to checksum-only verification."
}

$expectedChecksum = Get-ExpectedChecksum -ChecksumPath $checksumPath -ArchiveName $archiveName
$actualChecksum = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()

if ($actualChecksum -ne $expectedChecksum) {
    throw "Checksum mismatch for $archiveName. Expected $expectedChecksum, got $actualChecksum."
}

Write-Host "Verified checksum for $archiveName"

if ($archiveInfo.Extension -eq "zip") {
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
} else {
    & tar -xzf $archivePath -C $extractDir
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to extract $archiveName."
    }
}

$binary = Get-ChildItem -LiteralPath $extractDir -Recurse -File |
    Where-Object { $_.Name -eq $archiveInfo.BinaryName } |
    Select-Object -First 1

if (-not $binary) {
    throw "Could not find $($archiveInfo.BinaryName) after extracting $archiveName."
}

Move-Item -LiteralPath $binary.FullName -Destination $installedBinaryPath -Force

if ((Require-Env -Name "RUNNER_OS") -ne "Windows") {
    & chmod +x $installedBinaryPath
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to make $installedBinaryPath executable."
    }
}

Add-Content -LiteralPath (Require-Env -Name "GITHUB_PATH") -Value $installDir -Encoding utf8
Write-OutputValue -Name "binary-path" -Value $installedBinaryPath
Write-OutputValue -Name "version" -Value $version
Write-OutputValue -Name "signature-verified" -Value $signatureVerified.ToString().ToLowerInvariant()
