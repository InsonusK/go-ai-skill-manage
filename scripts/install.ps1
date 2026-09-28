[CmdletBinding()]
param(
    [string]$InstallDir = $env:AISM_INSTALL_DIR
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repository = 'InsonusK/go-ai-skill-manage'
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\aism'
}

$architecture = if ($env:PROCESSOR_ARCHITEW6432) {
    $env:PROCESSOR_ARCHITEW6432
} else {
    $env:PROCESSOR_ARCHITECTURE
}
if ($architecture -ne 'AMD64') {
    throw "Unsupported architecture: $architecture (only amd64 is published)"
}

$headers = @{
    Accept                 = 'application/vnd.github+json'
    'X-GitHub-Api-Version' = '2022-11-28'
    'User-Agent'           = 'ai-skill-manager-installer'
}
$release = Invoke-RestMethod `
    -Uri "https://api.github.com/repos/$repository/releases/latest" `
    -Headers $headers
$tag = [string]$release.tag_name
if (-not $tag.StartsWith('v') -or $tag.Length -eq 1) {
    throw "Could not determine the latest release (received tag '$tag')"
}

$version = $tag.Substring(1)
$assetName = "ai-skill-manager_${version}_windows_amd64.exe"
$checksumsName = "ai-skill-manager_${version}_checksums.txt"
$binaryAsset = @($release.assets) | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
$checksumsAsset = @($release.assets) | Where-Object { $_.name -eq $checksumsName } | Select-Object -First 1
if ($null -eq $binaryAsset -or $null -eq $checksumsAsset) {
    throw "Release $tag does not contain the expected Windows binary and checksums"
}

$tempDir = Join-Path ([IO.Path]::GetTempPath()) ("aism-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $binaryPath = Join-Path $tempDir $assetName
    $checksumsPath = Join-Path $tempDir $checksumsName

    Write-Host "Downloading ai-skill-manager $version for windows/amd64..."
    Invoke-WebRequest -UseBasicParsing -Uri $binaryAsset.browser_download_url -OutFile $binaryPath
    Invoke-WebRequest -UseBasicParsing -Uri $checksumsAsset.browser_download_url -OutFile $checksumsPath

    $checksumPattern = '^([a-fA-F0-9]{64})\s+\*?' + [regex]::Escape($assetName) + '$'
    $expectedHash = $null
    foreach ($line in Get-Content -LiteralPath $checksumsPath) {
        if ($line -match $checksumPattern) {
            $expectedHash = $Matches[1].ToLowerInvariant()
            break
        }
    }
    if ($null -eq $expectedHash) {
        throw "Checksum for $assetName is missing"
    }

    $actualHash = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "Checksum verification failed for $assetName"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $destination = Join-Path $InstallDir 'aism.exe'
    Copy-Item -Force -LiteralPath $binaryPath -Destination $destination

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $pathEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $pathIsConfigured = $pathEntries | Where-Object {
        $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\')
    }
    if (-not $pathIsConfigured) {
        $newUserPath = (@($pathEntries) + $InstallDir) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
    }

    $processPathEntries = @($env:Path -split ';')
    $processPathIsConfigured = $processPathEntries | Where-Object {
        $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\')
    }
    if (-not $processPathIsConfigured) {
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Host "Installed $destination"
    & $destination --version
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
