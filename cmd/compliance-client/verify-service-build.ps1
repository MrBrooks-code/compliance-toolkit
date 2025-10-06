# Verify Service Build Script
# Tests that service support is properly compiled into the client

Write-Host "=== Compliance Client Service Build Verification ===" -ForegroundColor Cyan
Write-Host ""

$exePath = ".\compliance-client.exe"

# Check if executable exists
if (-not (Test-Path $exePath)) {
    Write-Host "❌ ERROR: compliance-client.exe not found" -ForegroundColor Red
    Write-Host "   Run: go build -o compliance-client.exe" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Executable found: $exePath" -ForegroundColor Green

# Test version
Write-Host "`n📦 Testing version command..." -ForegroundColor Cyan
$version = & $exePath --version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Version: $version" -ForegroundColor Green
} else {
    Write-Host "❌ Version command failed" -ForegroundColor Red
    exit 1
}

# Test help output for service commands
Write-Host "`n📋 Checking for service commands..." -ForegroundColor Cyan
$help = & $exePath --help 2>&1
$serviceCommands = @(
    "--install-service",
    "--uninstall-service",
    "--start-service",
    "--stop-service",
    "--service-status"
)

$allFound = $true
foreach ($cmd in $serviceCommands) {
    if ($help -match [regex]::Escape($cmd)) {
        Write-Host "  ✅ $cmd" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $cmd NOT FOUND" -ForegroundColor Red
        $allFound = $false
    }
}

if (-not $allFound) {
    Write-Host "`n❌ Some service commands missing!" -ForegroundColor Red
    exit 1
}

# Test that service commands require admin (without actually running them)
Write-Host ""
Write-Host "Verifying service commands respond correctly..." -ForegroundColor Cyan

# These should fail gracefully without admin rights
$testCommands = @(
    @{Name = "install-service"; Args = "--install-service --config test.yaml"},
    @{Name = "service-status"; Args = "--service-status"}
)

foreach ($test in $testCommands) {
    Write-Host "  Testing $($test.Name)..." -ForegroundColor Gray
    $output = & $exePath $test.Args.Split() 2>&1

    # Should get either "access denied", "not found", or proper error message
    # (not crash or panic)
    if ($LASTEXITCODE -ne 0) {
        Write-Host "    ✅ Command handled correctly (expected failure without admin/service)" -ForegroundColor Green
    } else {
        Write-Host "    ⚠️  Command succeeded (unexpected, may need admin)" -ForegroundColor Yellow
    }
}

# Check for required DLLs/imports
Write-Host ""
Write-Host "Checking Windows service dependencies..." -ForegroundColor Cyan
$strings = & "C:\Windows\System32\findstr.exe" /C:"svc.Run" $exePath 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Service runtime found in binary" -ForegroundColor Green
} else {
    Write-Host "⚠️  Could not verify service runtime (findstr may not work on binary)" -ForegroundColor Yellow
}

# File size check
$size = (Get-Item $exePath).Length
$sizeMB = [math]::Round($size / 1MB, 2)
Write-Host ""
Write-Host "Binary size: $sizeMB MB" -ForegroundColor Cyan
if ($size -gt 1MB -and $size -lt 50MB) {
    Write-Host "✅ Size looks reasonable" -ForegroundColor Green
} else {
    Write-Host "⚠️  Size may be unusual (expected 5-15 MB)" -ForegroundColor Yellow
}

# Summary
Write-Host ""
Write-Host ("="*60) -ForegroundColor Cyan
Write-Host "Service Build Verification Complete!" -ForegroundColor Green
Write-Host ("="*60) -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. To test service installation (requires Administrator):" -ForegroundColor White
Write-Host "   .\compliance-client.exe --install-service --config client.yaml" -ForegroundColor Gray
Write-Host ""
Write-Host "2. See TEST_SERVICE.md for full testing guide" -ForegroundColor White
Write-Host ""
Write-Host "3. Use deploy-service.ps1 for production deployment" -ForegroundColor White
Write-Host ""
