# Compliance Toolkit - Automation Guide

**Version:** 1.1.0
**Last Updated:** 2025-01-05

Complete guide to automating compliance scans with Windows Task Scheduler, PowerShell, and batch scripts.

---

## Table of Contents

1. [Overview](#overview)
2. [Windows Task Scheduler](#windows-task-scheduler)
3. [PowerShell Scripts](#powershell-scripts)
4. [Batch Scripts](#batch-scripts)
5. [Monitoring & Alerting](#monitoring--alerting)
6. [Best Practices](#best-practices)

---

## Overview

The Compliance Toolkit supports full automation through:

- ✅ CLI flags for non-interactive execution
- ✅ Exit codes for success/failure detection
- ✅ Quiet mode for minimal output
- ✅ Custom output directories
- ✅ Scheduled task compatibility

---

## Windows Task Scheduler

### Quick Setup (5 Minutes)

**1. Open Task Scheduler:**
```
Win + R → taskschd.msc → Enter
```

**2. Create Basic Task:**
- Click "Create Basic Task"
- Name: `Daily Compliance Scan`
- Description: `Automated compliance check`

**3. Set Trigger:**
- Trigger: `Daily`
- Start: `2:00 AM`
- Recur every: `1 day`

**4. Set Action:**
- Action: `Start a program`
- Program/script: `C:\ComplianceTool\ComplianceToolkit.exe`
- Add arguments: `-report=all -quiet`
- Start in: `C:\ComplianceTool`

**5. Configure Settings:**
- ☑ Run whether user is logged on or not
- ☑ Run with highest privileges
- ☑ Wake the computer to run this task

**6. Finish:**
- Enter password for account
- Click "OK"

### Test the Task

```cmd
# Right-click task → Run
# Check: output\reports\ for new files
# Check: output\logs\ for execution logs
```

---

## PowerShell Scripts

### Basic Script

**File:** `run-compliance-scan.ps1`

\`\`\`powershell
# Basic compliance scan
$ToolkitPath = "C:\ComplianceTool\ComplianceToolkit.exe"

# Run scan
& $ToolkitPath -report=all -quiet

# Check exit code
if ($LASTEXITCODE -ne 0) {
    Write-Error "Compliance scan failed!"
    exit 1
}

Write-Host "Compliance scan completed successfully!" -ForegroundColor Green
\`\`\`

### Advanced Script with Archiving

**File:** `advanced-compliance-scan.ps1`

See `examples/scheduled_compliance_scan.ps1` for complete implementation.

**Features:**
- Automatic report archiving by date
- Email notifications (success/failure)
- Old report cleanup (>90 days)
- Comprehensive error handling
- Detailed logging

**Usage:**
\`\`\`powershell
# Basic usage
.\advanced-compliance-scan.ps1

# With archiving
.\advanced-compliance-scan.ps1 -ArchiveReports -ArchiveBase "C:\Compliance\Archive"

# With email notifications
.\advanced-compliance-scan.ps1 `
    -EmailTo "admin@company.com" `
    -EmailFrom "compliance@company.com" `
    -SmtpServer "smtp.company.com"

# Specific report
.\advanced-compliance-scan.ps1 -ReportName "NIST_800_171_compliance.json"

# All features combined
.\advanced-compliance-scan.ps1 `
    -ReportName "all" `
    -ArchiveReports `
    -ArchiveBase "\\fileserver\compliance\archive" `
    -EmailTo "security-team@company.com" `
    -SmtpServer "smtp.company.com"
\`\`\`

### Schedule PowerShell Script

**Task Scheduler Setup:**
- Program: `powershell.exe`
- Arguments: `-ExecutionPolicy Bypass -File "C:\Scripts\advanced-compliance-scan.ps1" -ArchiveReports`
- Start in: `C:\Scripts`

---

## Batch Scripts

### Basic Batch Script

**File:** `run-compliance.bat`

\`\`\`batch
@echo off
SET TOOLKIT=C:\ComplianceTool\ComplianceToolkit.exe

echo Running compliance scan...
"%TOOLKIT%" -report=all -quiet

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Scan failed!
    exit /b 1
)

echo Success!
exit /b 0
\`\`\`

### Advanced Batch Script

**File:** `scheduled-compliance.bat`

See `examples/scheduled_compliance_scan.bat` for complete implementation.

**Features:**
- Timestamped logging
- Error detection
- Optional email alerts
- Exit code handling

---

## Monitoring & Alerting

### Method 1: Email on Failure

**PowerShell:**
\`\`\`powershell
& "C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet

if ($LASTEXITCODE -ne 0) {
    Send-MailMessage `
        -To "admin@company.com" `
        -From "compliance@company.com" `
        -Subject "❌ Compliance Scan Failed" `
        -Body "Check logs at C:\ComplianceTool\output\logs" `
        -SmtpServer "smtp.company.com"
}
\`\`\`

### Method 2: Event Log Entry

**PowerShell:**
\`\`\`powershell
& "C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet

if ($LASTEXITCODE -ne 0) {
    Write-EventLog -LogName Application `
        -Source "ComplianceTool" `
        -EventId 1001 `
        -EntryType Error `
        -Message "Compliance scan failed"
}
\`\`\`

### Method 3: Slack/Teams Webhook

**PowerShell:**
\`\`\`powershell
$WebhookUrl = "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

& "C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet

if ($LASTEXITCODE -ne 0) {
    $Body = @{
        text = "❌ Compliance scan failed on $(hostname)"
    } | ConvertTo-Json

    Invoke-RestMethod -Uri $WebhookUrl -Method Post -Body $Body -ContentType 'application/json'
}
\`\`\`

---

## Best Practices

### 1. Use Quiet Mode for Scheduled Tasks

\`\`\`bash
# Good - minimal output
ComplianceToolkit.exe -report=all -quiet

# Bad - clutters scheduled task logs
ComplianceToolkit.exe -report=all
\`\`\`

### 2. Set Custom Output Directories

\`\`\`bash
# Organize by date or project
ComplianceToolkit.exe -report=all -quiet `
    -output=\\\\FileServer\\Compliance\\$(Get-Date -Format 'yyyy-MM-dd')
\`\`\`

### 3. Archive Old Reports

\`\`\`powershell
# Keep reports for 90 days
Get-ChildItem -Path "C:\\Compliance\\Archive" -Directory |
    Where-Object { $_.CreationTime -lt (Get-Date).AddDays(-90) } |
    Remove-Item -Recurse -Force
\`\`\`

### 4. Monitor Exit Codes

\`\`\`bash
ComplianceToolkit.exe -report=all -quiet
if %ERRORLEVEL% NEQ 0 (
    # Take action: email, log, alert
    echo Failure detected!
)
\`\`\`

### 5. Test Before Deploying

\`\`\`bash
# Always test manually first
ComplianceToolkit.exe -report=all -quiet
echo Exit Code: %ERRORLEVEL%
\`\`\`

### 6. Use Service Accounts

```
Create dedicated service account:
- Name: SVC_ComplianceScanner
- Permissions: Read registry, Write to output dirs
- No interactive login required
```

### 7. Centralize Reports

\`\`\`bash
# Write directly to network share
ComplianceToolkit.exe -report=all -quiet `
    -output=\\\\FileServer\\Compliance\\Reports `
    -evidence=\\\\FileServer\\Compliance\\Evidence
\`\`\`

---

## Example Schedules

### Daily Security Scan (2 AM)

\`\`\`
-report=NIST_800_171_compliance.json -quiet
\`\`\`

### Weekly Full Compliance (Sunday 3 AM)

\`\`\`
-report=all -quiet
\`\`\`

### Monthly Deep Audit (1st of month, 1 AM)

\`\`\`
-report=all -timeout=60s -quiet `
    -output=C:\\Compliance\\Monthly\\$(Get-Date -Format 'yyyy-MM')
\`\`\`

### Quarterly Executive Report

\`\`\`powershell
# First day of quarter
-report=all -quiet -output=C:\\Compliance\\Quarterly\\Q$(Get-Date -Format 'yyyy-Q')
# Then email reports to executives
\`\`\`

---

## Troubleshooting Automation

### Task Doesn't Run

**Check:**
- Account has admin privileges
- "Run with highest privileges" is checked
- Paths are absolute (not relative)
- Script execution policy allows PowerShell

**Fix:**
\`\`\`powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
\`\`\`

### Reports Not Generated

**Check:**
- `configs/reports/*.json` files exist
- Output directory is writable
- No permission errors in logs

**View Logs:**
\`\`\`bash
type C:\\ComplianceTool\\output\\logs\\toolkit_*.log
\`\`\`

### Email Notifications Not Sending

**Check:**
- SMTP server is correct
- Network connectivity
- Firewall allows SMTP (port 25/587)

**Test SMTP:**
\`\`\`powershell
Send-MailMessage -To "test@company.com" `
    -From "test@company.com" `
    -Subject "Test" `
    -Body "Test" `
    -SmtpServer "smtp.company.com"
\`\`\`

---

## Next Steps

- ✅ **Review Examples**: Check `examples/` directory
- ✅ **CLI Reference**: See [CLI Usage Guide](CLI_USAGE.md)
- ✅ **Quick Start**: See [CLI Quick Start](CLI_QUICKSTART.md)

---

*Automation Guide v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
