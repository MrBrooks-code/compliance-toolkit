# ComplianceToolkit CLI Usage Guide

**Version:** 1.1.0
**Last Updated:** 2025-01-04

---

## Overview

The Compliance Toolkit supports both **interactive** and **non-interactive** (CLI) modes. The CLI mode is designed for scheduled tasks, automation, and scripting scenarios.

---

## Interactive Mode (Default)

Run without any arguments to launch the interactive menu:

```bash
ComplianceToolkit.exe
```

This displays the ASCII menu interface for manual operation.

---

## Non-Interactive Mode (CLI)

### Available Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-report` | string | "" | Report to run (filename or "all") |
| `-list` | bool | false | List available reports and exit |
| `-quiet` | bool | false | Suppress non-essential output (for scheduled runs) |
| `-output` | string | "output/reports" | Output directory for HTML reports |
| `-evidence` | string | "output/evidence" | Evidence logs directory |
| `-logs` | string | "output/logs" | Application logs directory |
| `-timeout` | duration | 10s | Registry operation timeout |
| `-h` or `-help` | bool | false | Show help message |

---

## Common Usage Examples

### 1. List Available Reports

```bash
ComplianceToolkit.exe -list
```

**Output:**
```
Available Reports:
==================
  - NIST_800_171_compliance.json
    Title:    NIST 800-171 Security Compliance Report
    Category: Security & Compliance
    Version:  2.0.0

  - fips_140_2_compliance.json
    Title:    FIPS 140-2 Compliance Report
    Category: Security & Compliance
    Version:  1.0.0

  ...

To run a specific report:
  ComplianceToolkit.exe -report=<report-name.json>

To run all reports:
  ComplianceToolkit.exe -report=all
```

### 2. Run a Single Report

```bash
ComplianceToolkit.exe -report=NIST_800_171_compliance.json
```

**Output:**
```
Running: NIST 800-171 Security Compliance Report
======================
  Results: 11 successful, 2 errors
  HTML Report: output\reports\NIST_800_171_Security_Compliance_Report_20251004_120530.html
  Evidence Log: output/evidence/NIST_800_171_compliance_evidence_20251004_120530.json
======================
Report completed successfully!
Report saved to: output/reports
Evidence saved to: output/evidence
```

### 3. Run All Reports

```bash
ComplianceToolkit.exe -report=all
```

**Output:**
```
Running all reports...
======================

▶ Running: NIST 800-171 Security Compliance Report
  ✅ SUCCESS

▶ Running: FIPS 140-2 Compliance Report
  ✅ SUCCESS

▶ Running: System Information Report
  ✅ SUCCESS

...

======================
All reports completed successfully!
Reports saved to: output/reports
Evidence saved to: output/evidence
```

### 4. Quiet Mode (For Scheduled Tasks)

```bash
ComplianceToolkit.exe -report=all -quiet
```

**Output:** (minimal to none, only errors logged to file)

This mode is ideal for scheduled tasks where you don't want console output.

### 5. Custom Output Directories

```bash
ComplianceToolkit.exe -report=fips_140_2_compliance.json -output=C:\Compliance\Reports -evidence=C:\Compliance\Evidence
```

### 6. Increase Timeout for Slow Systems

```bash
ComplianceToolkit.exe -report=all -timeout=30s
```

---

## Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success - All reports executed successfully |
| 1 | Failure - One or more reports failed or error occurred |

Use exit codes in scripts to detect failures:

```bash
ComplianceToolkit.exe -report=all -quiet
if %ERRORLEVEL% NEQ 0 (
    echo "Compliance scan failed!"
    exit /b 1
)
```

---

## Scheduled Tasks (Windows Task Scheduler)

### Create a Daily Compliance Scan

**1. Open Task Scheduler:**
```
Win + R → taskschd.msc → Enter
```

**2. Create New Task:**
- **Name:** "Daily Compliance Scan"
- **Description:** "Automated NIST 800-171 and FIPS 140-2 compliance scanning"
- **Security Options:** "Run whether user is logged on or not"
- **Run with highest privileges:** ✓ (Checked)

**3. Trigger:**
- **Begin the task:** On a schedule
- **Daily** at **2:00 AM**
- **Recur every:** 1 day

**4. Action:**
- **Program/script:** `C:\Path\To\ComplianceToolkit.exe`
- **Arguments:** `-report=all -quiet`
- **Start in:** `C:\Path\To\` (directory containing the exe)

**5. Conditions:**
- **Start only if computer is on AC power:** ✓ (Optional)
- **Wake the computer to run this task:** ✓ (Optional)

**6. Settings:**
- **Allow task to be run on demand:** ✓
- **If the task fails, restart every:** 15 minutes (3 attempts)

---

## PowerShell Automation Example

### Run Weekly Compliance Report

```powershell
# weekly_compliance.ps1
$ErrorActionPreference = "Stop"

$TOOLKIT = "C:\Tools\ComplianceToolkit.exe"
$REPORT_DIR = "C:\Compliance\Reports"
$ARCHIVE_DIR = "C:\Compliance\Archive"

# Create archive directory with date
$DATE = Get-Date -Format "yyyy-MM-dd"
$ARCHIVE_PATH = Join-Path $ARCHIVE_DIR $DATE
New-Item -ItemType Directory -Path $ARCHIVE_PATH -Force | Out-Null

# Run compliance scan
Write-Host "Running compliance scan..." -ForegroundColor Cyan
& $TOOLKIT -report=all -output=$REPORT_DIR -quiet

if ($LASTEXITCODE -ne 0) {
    Write-Host "Compliance scan FAILED!" -ForegroundColor Red
    exit 1
}

# Archive reports
Write-Host "Archiving reports..." -ForegroundColor Green
Copy-Item "$REPORT_DIR\*.html" -Destination $ARCHIVE_PATH
Copy-Item "$REPORT_DIR\..\evidence\*.json" -Destination $ARCHIVE_PATH

Write-Host "Compliance scan completed successfully!" -ForegroundColor Green
Write-Host "Reports archived to: $ARCHIVE_PATH" -ForegroundColor Yellow
```

**Schedule in Task Scheduler:**
- **Program:** `powershell.exe`
- **Arguments:** `-ExecutionPolicy Bypass -File "C:\Scripts\weekly_compliance.ps1"`

---

## Batch Script Example

### Run Specific Report and Email Results

```batch
@echo off
REM compliance_scan.bat

SET TOOLKIT=C:\Tools\ComplianceToolkit.exe
SET REPORT=NIST_800_171_compliance.json
SET OUTPUT_DIR=C:\Compliance\Reports

echo Running %REPORT%...
"%TOOLKIT%" -report=%REPORT% -output=%OUTPUT_DIR%

if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Compliance scan failed!
    REM Send email notification (requires blat.exe or similar)
    REM blat.exe -to admin@company.com -subject "Compliance Scan Failed" -body "Check logs"
    exit /b 1
)

echo Compliance scan completed successfully!

REM Find the latest HTML report
for /f "delims=" %%f in ('dir /b /od "%OUTPUT_DIR%\*.html"') do set LATEST=%%f

echo Latest report: %OUTPUT_DIR%\%LATEST%

REM Open in browser (optional)
REM start "" "%OUTPUT_DIR%\%LATEST%"

exit /b 0
```

---

## Docker/Container Usage (Future)

For containerized environments, mount the output directories:

```bash
docker run -v /host/output:/app/output compliance-toolkit:latest \
  -report=all -quiet -output=/app/output/reports
```

---

## Logging

All CLI executions are logged to `output/logs/toolkit_YYYYMMDD_HHMMSS.log` in JSON format.

**Example log entry:**
```json
{
  "time": "2025-01-04T12:05:30.123Z",
  "level": "INFO",
  "msg": "Report execution completed",
  "report": "NIST 800-171 Security Compliance Report",
  "success_count": 11,
  "error_count": 2,
  "html_report": "output/reports/NIST_800_171_Security_Compliance_Report_20251004_120530.html"
}
```

**View logs:**
```bash
type output\logs\toolkit_20251004_120530.log
```

Or use the interactive mode: **[4] View Log Files**

---

## Troubleshooting

### Issue: "No reports found in configs/reports/"

**Solution:** Ensure the `configs/reports/` directory exists and contains JSON report configurations.

### Issue: Exit code 1 even when report completes

**Solution:** Check the log files for errors. Some registry keys may not exist, which is normal. Only critical errors cause exit code 1.

### Issue: Permission denied errors

**Solution:** Run as Administrator for full registry access:
```bash
# PowerShell (as Admin)
Start-Process "ComplianceToolkit.exe" -ArgumentList "-report=all" -Verb RunAs
```

### Issue: Reports not generated in custom directory

**Solution:** Ensure the output directory exists or has write permissions:
```bash
mkdir C:\Compliance\Reports
ComplianceToolkit.exe -report=all -output=C:\Compliance\Reports
```

---

## Best Practices

### 1. **Always Use Quiet Mode for Scheduled Tasks**
```bash
ComplianceToolkit.exe -report=all -quiet
```
Reduces log noise and improves performance.

### 2. **Archive Old Reports Regularly**
Create a cleanup script to move reports older than 30 days to archive storage.

### 3. **Monitor Exit Codes**
Use exit codes in scripts to detect failures and trigger alerts:
```powershell
if ($LASTEXITCODE -ne 0) {
    Send-MailMessage -To "admin@company.com" -Subject "Compliance Scan Failed"
}
```

### 4. **Test Before Scheduling**
Always test CLI commands manually before adding to Task Scheduler:
```bash
ComplianceToolkit.exe -report=all -quiet
echo Exit Code: %ERRORLEVEL%
```

### 5. **Use Absolute Paths in Scheduled Tasks**
Don't rely on relative paths when running from Task Scheduler:
```
C:\Tools\ComplianceToolkit.exe -report=all -output=C:\Compliance\Reports
```

---

## Advanced Use Cases

### 1. Run Different Reports at Different Times

**Daily Security Scan (2 AM):**
```bash
ComplianceToolkit.exe -report=NIST_800_171_compliance.json -quiet
```

**Weekly Full Audit (Sunday 3 AM):**
```bash
ComplianceToolkit.exe -report=all -quiet
```

**Monthly Deep Scan (1st of month):**
```bash
ComplianceToolkit.exe -report=all -timeout=60s -quiet
```

### 2. Integration with Monitoring Tools

**Export to SIEM (Security Information and Event Management):**
```powershell
# Parse evidence JSON and forward to SIEM
$EVIDENCE = Get-Content "output/evidence/NIST_800_171_compliance_evidence_*.json" | ConvertFrom-Json
# Send to SIEM via REST API
Invoke-RestMethod -Uri "https://siem.company.com/api/events" -Method POST -Body ($EVIDENCE | ConvertTo-Json)
```

### 3. Compliance Trending Over Time

Store evidence logs in dated directories:
```bash
ComplianceToolkit.exe -report=all -evidence=C:\Compliance\Evidence\2025-01-04 -quiet
```

Then analyze trends using PowerShell or Python scripts.

---

## Summary

| Use Case | Command |
|----------|---------|
| List reports | `ComplianceToolkit.exe -list` |
| Run one report | `ComplianceToolkit.exe -report=NIST_800_171_compliance.json` |
| Run all reports | `ComplianceToolkit.exe -report=all` |
| Scheduled task | `ComplianceToolkit.exe -report=all -quiet` |
| Custom output | `ComplianceToolkit.exe -report=all -output=C:\Custom\Path` |
| Increase timeout | `ComplianceToolkit.exe -report=all -timeout=30s` |

---

**For interactive usage, see:** [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
**For adding new reports, see:** [ADDING_NEW_REPORTS.md](ADDING_NEW_REPORTS.md)
**For project status, see:** [PROJECT_STATUS.md](PROJECT_STATUS.md)

---

*Document Version: 1.0.0*
*Last Updated: 2025-01-04*
*CLI Support: v1.1.0+*
