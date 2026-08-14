<#
.SYNOPSIS
    drun updater for Windows (PowerShell).

.DESCRIPTION
    Windows cannot reliably replace a running executable in place, so this script
    updates xdrun from a separate process. It locates the existing xdrun.exe,
    compares the installed version against the latest GitHub release, and replaces
    the executable when a newer version is available.

.PARAMETER Version
    Release tag to update to (e.g. v1.0.0). Defaults to the latest release.

.PARAMETER InstallDir
    Directory containing xdrun.exe. Defaults to auto-detection via PATH, then
    $env:LOCALAPPDATA\Programs\xdrun.

.PARAMETER Force
    Reinstall even if the installed version already matches the target version.

.PARAMETER WaitForPid
    Process ID to wait for before replacing the executable. Used by
    `xdrun --self-update` so the running xdrun process can exit first.

.EXAMPLE
    irm https://raw.githubusercontent.com/phillarmonic/drun/master/update.ps1 | iex

.EXAMPLE
    .\update.ps1 -Version v1.0.0
#>

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [switch]$Force,
    [int]$WaitForPid = 0
)

$ErrorActionPreference = "Stop"

# Configuration
$Repo = "phillarmonic/drun"
$BinaryName = "xdrun.exe"
$GitHubApi = "https://api.github.com/repos/$Repo"
$GitHubReleases = "https://github.com/$Repo/releases"

# Logging helpers
function Write-Info    { param([string]$Message) Write-Host "i  $Message" -ForegroundColor Blue }
function Write-Success { param([string]$Message) Write-Host "OK $Message" -ForegroundColor Green }
function Write-Warn    { param([string]$Message) Write-Host "!  $Message" -ForegroundColor Yellow }
function Write-Err     { param([string]$Message) Write-Host "X  $Message" -ForegroundColor Red }

# Detect the processor architecture and map it to a release binary suffix.
function Get-ReleaseArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" {
            if ($env:PROCESSOR_ARCHITEW6432 -eq "AMD64") { return "amd64" }
            if ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64") { return "arm64" }
            Write-Err "Unsupported architecture: x86 (32-bit is not supported)"
            exit 1
        }
        default {
            Write-Err "Unsupported architecture: $arch"
            Write-Err "Supported architectures: amd64, arm64"
            exit 1
        }
    }
}

# Resolve the latest release tag from the GitHub API.
function Get-LatestVersion {
    Write-Info "Fetching latest release information..."
    try {
        $response = Invoke-RestMethod -Uri "$GitHubApi/releases/latest" -Headers @{ "User-Agent" = "drun-updater" }
    }
    catch {
        Write-Err "Failed to fetch release information from GitHub"
        Write-Err "Please check your internet connection or try again later"
        exit 1
    }

    $tag = $response.tag_name
    if ([string]::IsNullOrWhiteSpace($tag)) {
        Write-Err "Failed to parse latest version from GitHub API"
        exit 1
    }
    return $tag
}

# Validate a version string looks like a release tag.
function Test-VersionFormat {
    param([string]$Ver)
    if ($Ver -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$') {
        Write-Err "Invalid version format: $Ver"
        Write-Err "Expected format: v1.0.0 or v1.0.0-beta.1"
        exit 1
    }
}

# Confirm a specific release tag exists.
function Test-VersionExists {
    param([string]$Ver)
    Write-Info "Checking if version $Ver exists..."
    try {
        Invoke-RestMethod -Uri "$GitHubApi/releases/tags/$Ver" -Headers @{ "User-Agent" = "drun-updater" } | Out-Null
    }
    catch {
        Write-Err "Version $Ver not found"
        Write-Err "Available releases: $GitHubReleases"
        exit 1
    }
    Write-Success "Version $Ver found"
}

# Find the directory that currently holds xdrun.exe.
function Resolve-InstallDir {
    param([string]$Requested)

    if (-not [string]::IsNullOrWhiteSpace($Requested)) {
        return $Requested
    }

    # Prefer the xdrun already resolvable on PATH.
    $existing = Get-Command $BinaryName -ErrorAction SilentlyContinue
    if ($existing) {
        return (Split-Path -Parent $existing.Source)
    }

    # Fall back to the default install location used by install.ps1.
    return (Join-Path $env:LOCALAPPDATA "Programs\xdrun")
}

# Normalize a version string for comparison (strip leading v and whitespace).
function Format-Version {
    param([string]$Ver)
    return ($Ver -replace '^v', '').Trim()
}

# Read the installed version by running the existing binary.
function Get-InstalledVersion {
    param([string]$BinaryPath)
    if (-not (Test-Path -LiteralPath $BinaryPath)) {
        return $null
    }
    try {
        $output = & $BinaryPath --version 2>$null
    }
    catch {
        return $null
    }
    if (-not $output) { return $null }

    # Extract the first vX.Y.Z (or X.Y.Z) token from the version output.
    $match = [regex]::Match(($output -join " "), 'v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?')
    if ($match.Success) {
        return $match.Value
    }
    return $null
}

# Download and replace the executable in place.
function Update-Binary {
    param([string]$Ver, [string]$Arch, [string]$TargetDir)

    $releaseBinary = "xdrun-windows-$Arch.exe"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Ver/$releaseBinary"
    $targetPath = Join-Path $TargetDir $BinaryName

    if (-not (Test-Path -LiteralPath $TargetDir)) {
        Write-Info "Creating install directory: $TargetDir"
        New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
    }

    $tempFile = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName() + ".exe")

    Write-Info "Downloading $releaseBinary..."
    Write-Info "URL: $downloadUrl"
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -Headers @{ "User-Agent" = "drun-updater" } -UseBasicParsing
    }
    catch {
        Write-Err "Failed to download binary from $downloadUrl"
        Write-Err "Please check if the release exists: $GitHubReleases/tag/$Ver"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    Write-Info "Verifying downloaded binary..."
    try {
        & $tempFile --version | Out-Null
    }
    catch {
        Write-Err "Downloaded binary failed verification"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    # Wait for the invoking xdrun process to exit so its executable is unlocked.
    if ($WaitForPid -gt 0) {
        $proc = Get-Process -Id $WaitForPid -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Info "Waiting for xdrun (PID $WaitForPid) to exit..."
            try { $proc.WaitForExit() } catch { }
            # Give Windows a moment to release the file handle.
            Start-Sleep -Milliseconds 500
        }
    }

    # Guard against replacing a binary that is currently running.
    if (Test-Path -LiteralPath $targetPath) {
        $running = Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and ($_.Path -ieq $targetPath) -and ($_.Id -ne $WaitForPid) }
        if ($running) {
            Write-Err "xdrun is currently running (PID $($running.Id -join ', '))."
            Write-Err "Close all xdrun processes and run this updater again."
            Remove-Item -LiteralPath $tempFile -Force
            exit 1
        }
    }

    try {
        Move-Item -LiteralPath $tempFile -Destination $targetPath -Force
    }
    catch {
        Write-Err "Failed to replace $targetPath"
        Write-Err "Make sure xdrun is not running and you have write access to $TargetDir"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    Write-Success "Updated $BinaryName at $targetPath"
    return $targetPath
}

# Main
Write-Host ""
Write-Host "drun updater (Windows)" -ForegroundColor Cyan
Write-Host "======================"
Write-Host ""

$arch = Get-ReleaseArch
Write-Info "Detected platform: windows/$arch"

$InstallDir = Resolve-InstallDir -Requested $InstallDir
$targetPath = Join-Path $InstallDir $BinaryName
Write-Info "Install directory: $InstallDir"

if (-not (Test-Path -LiteralPath $targetPath)) {
    Write-Warn "xdrun.exe was not found in $InstallDir"
    Write-Warn "Run the installer first:"
    Write-Warn "  irm https://raw.githubusercontent.com/$Repo/master/install.ps1 | iex"
    exit 1
}

# Resolve the target version.
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = Get-LatestVersion
}
else {
    Test-VersionFormat $Version
    Test-VersionExists $Version
}

$installedVersion = Get-InstalledVersion -BinaryPath $targetPath
if ($installedVersion) {
    Write-Info "Installed version: $installedVersion"
}
else {
    Write-Warn "Could not determine the installed version"
}
Write-Info "Target version: $Version"

if (-not $Force -and $installedVersion -and ((Format-Version $installedVersion) -eq (Format-Version $Version))) {
    Write-Success "xdrun is already up to date ($installedVersion)"
    exit 0
}

$updatedPath = Update-Binary -Ver $Version -Arch $arch -TargetDir $InstallDir

Write-Host ""
Write-Success "xdrun updated successfully!"
Write-Host ""

Write-Info "New version:"
& $updatedPath --version
Write-Host ""
Write-Info "Documentation: https://github.com/$Repo"
