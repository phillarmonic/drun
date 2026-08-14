<#
.SYNOPSIS
    drun installer for Windows (PowerShell).

.DESCRIPTION
    Downloads the xdrun executable from GitHub releases, installs it into a
    user-local directory (creating it if needed), and ensures that directory is
    on the user's PATH so "xdrun" can be invoked from any terminal.

.PARAMETER Version
    Release tag to install (e.g. v1.0.0). Defaults to the latest release.

.PARAMETER InstallDir
    Installation directory. Defaults to $env:LOCALAPPDATA\Programs\xdrun.

.EXAMPLE
    irm https://raw.githubusercontent.com/phillarmonic/drun/master/install.ps1 | iex

.EXAMPLE
    # Install a specific version to a custom directory
    .\install.ps1 -Version v1.0.0 -InstallDir C:\Tools\xdrun
#>

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = ""
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
            # 32-bit PowerShell on a 64-bit OS reports x86; check the real arch.
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
        $response = Invoke-RestMethod -Uri "$GitHubApi/releases/latest" -Headers @{ "User-Agent" = "drun-installer" }
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
        Invoke-RestMethod -Uri "$GitHubApi/releases/tags/$Ver" -Headers @{ "User-Agent" = "drun-installer" } | Out-Null
    }
    catch {
        Write-Err "Version $Ver not found"
        Write-Err "Available releases: $GitHubReleases"
        exit 1
    }
    Write-Success "Version $Ver found"
}

# Download and install the executable.
function Install-Binary {
    param([string]$Ver, [string]$Arch, [string]$TargetDir)

    $releaseBinary = "xdrun-windows-$Arch.exe"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Ver/$releaseBinary"

    # Ensure the install directory exists.
    if (-not (Test-Path -LiteralPath $TargetDir)) {
        Write-Info "Creating install directory: $TargetDir"
        try {
            New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
        }
        catch {
            Write-Err "Failed to create install directory: $TargetDir"
            exit 1
        }
    }

    $tempFile = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName() + ".exe")
    $targetPath = Join-Path $TargetDir $BinaryName

    Write-Info "Downloading $releaseBinary..."
    Write-Info "URL: $downloadUrl"
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -Headers @{ "User-Agent" = "drun-installer" } -UseBasicParsing
    }
    catch {
        Write-Err "Failed to download binary from $downloadUrl"
        Write-Err "Please check if the release exists: $GitHubReleases/tag/$Ver"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    # Verify the binary runs.
    Write-Info "Verifying binary..."
    try {
        & $tempFile --version | Out-Null
    }
    catch {
        Write-Err "Downloaded binary failed verification"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    try {
        Move-Item -LiteralPath $tempFile -Destination $targetPath -Force
    }
    catch {
        Write-Err "Failed to install binary to $TargetDir"
        if (Test-Path -LiteralPath $tempFile) { Remove-Item -LiteralPath $tempFile -Force }
        exit 1
    }

    Write-Success "Installed $BinaryName to $targetPath"
    return $targetPath
}

# Ensure the install directory is on the user's PATH.
function Add-ToUserPath {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ([string]::IsNullOrEmpty($userPath)) { $userPath = "" }

    # Compare entries case-insensitively, ignoring trailing slashes.
    $normalizedDir = $Dir.TrimEnd('\')
    $entries = $userPath.Split(';') | Where-Object { $_ -ne "" }
    $alreadyPresent = $entries | Where-Object { $_.TrimEnd('\') -ieq $normalizedDir }

    if ($alreadyPresent) {
        Write-Success "$Dir is already in your PATH"
    }
    else {
        Write-Info "Adding $Dir to your user PATH..."
        $newPath = if ($userPath.TrimEnd(';') -eq "") { $Dir } else { $userPath.TrimEnd(';') + ";" + $Dir }
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Success "Added $Dir to your user PATH"
        Write-Info "Restart your terminal for the PATH change to take effect"
    }

    # Make xdrun available in the current session too.
    if (($env:PATH.Split(';') | Where-Object { $_.TrimEnd('\') -ieq $normalizedDir }).Count -eq 0) {
        $env:PATH = "$env:PATH;$Dir"
    }
}

# Main
Write-Host ""
Write-Host "drun installer (Windows)" -ForegroundColor Cyan
Write-Host "========================"
Write-Host ""

$arch = Get-ReleaseArch
Write-Info "Detected platform: windows/$arch"

# Default install directory.
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\xdrun"
}

# Resolve version.
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = Get-LatestVersion
    Write-Info "Installing latest version: $Version"
}
else {
    Test-VersionFormat $Version
    Test-VersionExists $Version
    Write-Info "Installing specified version: $Version"
}

$installedPath = Install-Binary -Ver $Version -Arch $arch -TargetDir $InstallDir
Add-ToUserPath -Dir $InstallDir

Write-Host ""
Write-Success "xdrun CLI installation completed successfully!"
Write-Host ""

Write-Info "Installed version:"
& $installedPath --version
Write-Host ""

Write-Info "Get started with:"
Write-Info "  xdrun --help"
Write-Info "  xdrun --init"
Write-Host ""
Write-Info "Documentation: https://github.com/$Repo"
Write-Info "Examples: https://github.com/$Repo/tree/master/examples"
Write-Host ""
Write-Info "To uninstall: Remove-Item `"$installedPath`""
