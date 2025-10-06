# Windows Service Testing Guide

## Prerequisites

- Windows system (service support is Windows-only)
- Administrator privileges (required for service installation)
- Built `compliance-client.exe`

## Service Management Commands

### 1. Install Service

Install the compliance client as a Windows service:

```powershell
# Run as Administrator
.\compliance-client.exe --install-service

# Install with custom config path
.\compliance-client.exe --install-service --config "C:\ComplianceToolkit\client.yaml"
```

**Expected Output:**
```
Service ComplianceToolkitClient installed successfully
Start with: sc start ComplianceToolkitClient
```

**What Happens:**
- Service registered with Windows Service Control Manager (SCM)
- Service set to start automatically on system boot
- Event log source created for logging
- Recovery actions configured (restart on failure after 30s, 60s, 120s)
- Service depends on TCP/IP (network required)

### 2. Start Service

Start the installed service:

```powershell
# Using our tool
.\compliance-client.exe --start-service

# Or using Windows sc command
sc start ComplianceToolkitClient

# Or using PowerShell
Start-Service ComplianceToolkitClient
```

**Expected Output:**
```
Service ComplianceToolkitClient started successfully
```

**What Happens:**
- Service starts in background
- Logs to Windows Event Log (Application log)
- Runs compliance reports according to schedule in config
- Submits reports to server or saves locally

### 3. Check Service Status

View current service status:

```powershell
# Using our tool
.\compliance-client.exe --service-status

# Or using Windows sc command
sc query ComplianceToolkitClient

# Or using PowerShell
Get-Service ComplianceToolkitClient
```

**Expected Output:**
```
Service: ComplianceToolkitClient
Display Name: Compliance Toolkit Client
Description: Automated compliance scanning and reporting client for Windows registry security checks
State: Running
Start Type: Automatic
Executable: C:\path\to\compliance-client.exe --config C:\path\to\client.yaml
```

### 4. Stop Service

Stop the running service:

```powershell
# Using our tool
.\compliance-client.exe --stop-service

# Or using Windows sc command
sc stop ComplianceToolkitClient

# Or using PowerShell
Stop-Service ComplianceToolkitClient
```

**Expected Output:**
```
Service ComplianceToolkitClient stopped successfully
```

### 5. Uninstall Service

Remove the service from the system:

```powershell
# Run as Administrator
.\compliance-client.exe --uninstall-service
```

**Expected Output:**
```
Stopping service...
Service stopped
Service ComplianceToolkitClient uninstalled successfully
```

**What Happens:**
- Service stopped if running
- Service unregistered from SCM
- Event log source removed

## Viewing Service Logs

Service events are logged to Windows Event Log:

```powershell
# View all events from our service
Get-EventLog -LogName Application -Source ComplianceToolkitClient -Newest 20

# View only errors
Get-EventLog -LogName Application -Source ComplianceToolkitClient -EntryType Error

# Monitor in real-time
Get-EventLog -LogName Application -Source ComplianceToolkitClient -Newest 1 -After (Get-Date).AddMinutes(-5)
```

**Event Types:**
- **Information (ID 1)**: Service start, stop, normal operations
- **Warning (ID 1)**: Non-fatal issues, timeouts
- **Error (ID 1)**: Failures, exceptions

## Testing Scenarios

### Scenario 1: Basic Installation and Operation

```powershell
# 1. Install service
.\compliance-client.exe --install-service --config "client.yaml"

# 2. Verify installation
.\compliance-client.exe --service-status

# 3. Start service
.\compliance-client.exe --start-service

# 4. Wait 30 seconds, then check logs
Start-Sleep -Seconds 30
Get-EventLog -LogName Application -Source ComplianceToolkitClient -Newest 5

# 5. Stop service
.\compliance-client.exe --stop-service

# 6. Uninstall
.\compliance-client.exe --uninstall-service
```

### Scenario 2: Service Recovery Testing

Test automatic restart on failure:

```powershell
# 1. Install and start service
.\compliance-client.exe --install-service --config "client.yaml"
.\compliance-client.exe --start-service

# 2. Simulate crash by killing process
$process = Get-Process | Where-Object { $_.Name -eq "compliance-client" }
Stop-Process -Id $process.Id -Force

# 3. Wait 35 seconds (recovery delay is 30s)
Start-Sleep -Seconds 35

# 4. Check if service restarted
.\compliance-client.exe --service-status

# Should show: State: Running
```

### Scenario 3: Scheduled Reports

Test scheduled report execution:

```powershell
# 1. Create config with schedule enabled
@"
client:
  id: "TEST-CLIENT-001"
  hostname: "$(hostname)"
  enabled: true

server:
  url: ""  # Standalone mode for testing

reports:
  config_path: "../../configs/reports"
  output_path: "../../output/reports"
  save_local: true
  reports:
    - "NIST_800_171_compliance.json"

schedule:
  enabled: true
  cron: "*/2 * * * *"  # Every 2 minutes for testing

retry:
  max_attempts: 3
  initial_backoff: 30s
  max_backoff: 5m
  backoff_multiplier: 2.0

cache:
  enabled: true
  path: "../../cache/submissions"
  max_size_mb: 100

logging:
  level: "debug"
  format: "text"
  output_path: "../../output/logs/service.log"
"@ | Out-File -FilePath "test-service.yaml" -Encoding UTF8

# 2. Install service with test config
.\compliance-client.exe --install-service --config "test-service.yaml"

# 3. Start service
.\compliance-client.exe --start-service

# 4. Wait 5 minutes and check for reports
Start-Sleep -Seconds 300
Get-ChildItem "..\..\output\reports" | Sort-Object LastWriteTime -Descending | Select-Object -First 3

# 5. Check service logs
Get-Content "..\..\output\logs\service.log" -Tail 20

# 6. Cleanup
.\compliance-client.exe --stop-service
.\compliance-client.exe --uninstall-service
Remove-Item "test-service.yaml"
```

## Troubleshooting

### Service Won't Install

**Error:** "Access denied" or "Failed to connect to service manager"

**Solution:** Run PowerShell as Administrator

```powershell
# Right-click PowerShell → "Run as Administrator"
```

### Service Won't Start

**Error:** "Failed to start service"

**Possible Causes:**
1. **Config file not found**: Service can't locate `client.yaml`
   ```powershell
   # Use absolute path when installing
   .\compliance-client.exe --install-service --config "C:\full\path\to\client.yaml"
   ```

2. **Invalid config**: Check config syntax
   ```powershell
   .\compliance-client.exe --config "client.yaml" --once
   # If this fails, fix config before installing service
   ```

3. **Port conflict**: Another service using same port
   ```powershell
   # Check event log for details
   Get-EventLog -LogName Application -Source ComplianceToolkitClient -Newest 1
   ```

### Service Crashes

**Check Event Log:**
```powershell
Get-EventLog -LogName Application -Source ComplianceToolkitClient -EntryType Error -Newest 5 | Format-List
```

**Common Issues:**
- Network unreachable (server mode)
- Registry access denied (run as SYSTEM or Administrator)
- Disk full (check output paths)

### Service Stuck in "Stop Pending"

**Force kill:**
```powershell
# Get service PID
sc queryex ComplianceToolkitClient

# Kill process
taskkill /PID <PID> /F

# Then uninstall/reinstall
.\compliance-client.exe --uninstall-service
.\compliance-client.exe --install-service
```

## Service Configuration Best Practices

### 1. Use Absolute Paths

Service runs from `C:\Windows\System32`, so relative paths won't work:

**Bad:**
```yaml
reports:
  config_path: "configs/reports"  # Won't find this!
```

**Good:**
```yaml
reports:
  config_path: "C:\\ComplianceToolkit\\configs\\reports"
```

### 2. Enable File Logging

Service has no console output, use file logging:

```yaml
logging:
  level: "info"
  format: "text"
  output_path: "C:\\ComplianceToolkit\\logs\\service.log"  # Not "stdout"!
```

### 3. Set Appropriate Schedule

Don't spam the system:

```yaml
schedule:
  enabled: true
  cron: "0 2 * * *"  # Once daily at 2 AM (good)
  # NOT: "* * * * *"  # Every minute (bad!)
```

### 4. Configure Retry Logic

Service should be resilient:

```yaml
retry:
  max_attempts: 5  # More attempts for service
  initial_backoff: 60s  # Longer backoff for background operation
  retry_on_server_error: true
```

## Verification Checklist

After installation, verify:

- [ ] Service shows "Running" in `--service-status`
- [ ] Event log shows "service started successfully"
- [ ] Log file being written to configured path
- [ ] Reports generated in output directory (if scheduled)
- [ ] Service survives system reboot (start type: Automatic)
- [ ] Service restarts automatically after crash
- [ ] Service stops cleanly with `--stop-service`

## Production Deployment

### Deployment Steps

1. **Build release binary:**
   ```powershell
   go build -ldflags="-s -w" -o compliance-client.exe
   ```

2. **Copy to production location:**
   ```powershell
   Copy-Item compliance-client.exe C:\ComplianceToolkit\bin\
   ```

3. **Create production config:**
   ```powershell
   cd C:\ComplianceToolkit
   .\bin\compliance-client.exe --generate-config --config config\client.yaml
   # Edit config\client.yaml with production values
   ```

4. **Install service:**
   ```powershell
   cd C:\ComplianceToolkit\bin
   .\compliance-client.exe --install-service --config ..\config\client.yaml
   ```

5. **Start service:**
   ```powershell
   .\compliance-client.exe --start-service
   ```

6. **Verify operation:**
   ```powershell
   # Check status
   .\compliance-client.exe --service-status

   # Check event log
   Get-EventLog -LogName Application -Source ComplianceToolkitClient -Newest 10

   # Check log file
   Get-Content ..\logs\service.log -Tail 20

   # Wait for first scheduled run, then check reports
   Get-ChildItem ..\output\reports | Sort-Object LastWriteTime -Descending | Select-Object -First 5
   ```

### Unattended Installation Script

```powershell
# deploy-service.ps1
param(
    [string]$ConfigPath = "C:\ComplianceToolkit\config\client.yaml",
    [string]$ExePath = "C:\ComplianceToolkit\bin\compliance-client.exe"
)

# Verify running as admin
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as Administrator"
    exit 1
}

# Stop existing service if running
Write-Host "Checking for existing service..."
try {
    & $ExePath --stop-service 2>$null
    Write-Host "Stopped existing service"
} catch {
    Write-Host "No existing service running"
}

# Uninstall existing service if present
try {
    & $ExePath --uninstall-service 2>$null
    Write-Host "Uninstalled existing service"
} catch {
    Write-Host "No existing service found"
}

# Install service
Write-Host "Installing service..."
& $ExePath --install-service --config $ConfigPath
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to install service"
    exit 1
}

# Start service
Write-Host "Starting service..."
& $ExePath --start-service
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to start service"
    exit 1
}

# Verify
Write-Host "`nService status:"
& $ExePath --service-status

Write-Host "`nService deployed successfully!"
```

Usage:
```powershell
.\deploy-service.ps1 -ConfigPath "C:\ComplianceToolkit\config\client.yaml"
```
