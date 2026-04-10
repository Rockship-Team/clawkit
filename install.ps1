# clawkit installer for Windows
# Usage: irm https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "Rockship-Team/clawkit"
$Binary = "clawkit"
$InstallDir = Join-Path $env:LOCALAPPDATA "clawkit\bin"

function Write-Info($msg)  { Write-Host "> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)    { Write-Host "[OK] $msg" -ForegroundColor Green }
function Write-Fail($msg)  { Write-Host "[FAIL] $msg" -ForegroundColor Red; exit 1 }

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Fail "32-bit systems are not supported"
}

$Asset = "$Binary-windows-$Arch.exe"

Write-Host ""
Write-Host "  clawkit installer"
Write-Host "  -----------------"
Write-Host ""
Write-Info "Detected: windows/$Arch"

# Get latest release download URL
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/$Asset"
Write-Info "Downloading from: $DownloadUrl"

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$DestPath = Join-Path $InstallDir "$Binary.exe"
$TempFile = Join-Path $env:TEMP "$Binary-download.exe"

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing
} catch {
    Remove-Item $TempFile -ErrorAction SilentlyContinue

    # Fallback: build from source
    Write-Info "No pre-built binary found. Building from source..."

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Fail "Go is not installed. Install Go 1.22+ first: https://go.dev/dl/"
    }

    $CloneDir = Join-Path $env:TEMP "clawkit-build"
    if (Test-Path $CloneDir) { Remove-Item $CloneDir -Recurse -Force }

    git clone --depth 1 "https://github.com/$Repo.git" $CloneDir
    Push-Location $CloneDir
    try {
        $env:CGO_ENABLED = "0"
        go build -ldflags "-s -w" -o "$Binary.exe" ./cmd/clawkit
        $TempFile = Join-Path $CloneDir "$Binary.exe"
    } finally {
        Pop-Location
    }
}

# Install
Move-Item -Path $TempFile -Destination $DestPath -Force

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Info "Adding $InstallDir to user PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Verify
$Version = & $DestPath version 2>&1
Write-Ok "Installed: $Version"
Write-Host ""
Write-Host "  Get started:"
Write-Host "    clawkit list"
Write-Host "    clawkit install shop-hoa"
Write-Host ""
Write-Host "  NOTE: Restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
Write-Host ""
