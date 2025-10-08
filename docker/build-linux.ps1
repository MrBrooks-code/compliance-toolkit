# Build script for compliance-server Linux binary
# This script cross-compiles the Go application for Linux AMD64

$ErrorActionPreference = "Stop"

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Building compliance-server for Linux" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Navigate to project root (parent of docker directory)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $scriptDir "..")

# Clean previous builds
Write-Host "Cleaning previous builds..." -ForegroundColor Yellow
if (Test-Path "docker\bin\compliance-server") {
    Remove-Item "docker\bin\compliance-server" -Force
}

# Create bin directory if it doesn't exist
if (-not (Test-Path "docker\bin")) {
    New-Item -ItemType Directory -Path "docker\bin" | Out-Null
}

# Build for Linux AMD64
Write-Host "Building Linux AMD64 binary..." -ForegroundColor Yellow
$env:CGO_ENABLED = "1"
$env:GOOS = "linux"
$env:GOARCH = "amd64"

go build -ldflags="-s -w" -o docker\bin\compliance-server .\cmd\compliance-server

# Reset environment variables
$env:GOOS = ""
$env:GOARCH = ""
$env:CGO_ENABLED = ""

# Verify the binary was created
if (Test-Path "docker\bin\compliance-server") {
    Write-Host ""
    Write-Host "✓ Build successful!" -ForegroundColor Green
    Write-Host "Binary location: docker\bin\compliance-server" -ForegroundColor Green
    $binary = Get-Item "docker\bin\compliance-server"
    Write-Host "Size: $([math]::Round($binary.Length / 1MB, 2)) MB" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "✗ Build failed - binary not found" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Build complete!" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. cd docker" -ForegroundColor White
Write-Host "2. docker-compose -f docker-compose.binary.yml build" -ForegroundColor White
Write-Host "3. docker-compose -f docker-compose.binary.yml up -d" -ForegroundColor White
Write-Host ""
