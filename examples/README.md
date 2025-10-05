# Compliance Toolkit - Example Scripts

This directory contains example scripts for automating the Compliance Toolkit.

---

## Files

### 1. `scheduled_compliance_scan.bat`
**Windows Batch Script** for Task Scheduler

**Usage:**
```batch
scheduled_compliance_scan.bat
```

**Features:**
- Runs all compliance reports in quiet mode
- Logs results to `output/logs/scheduled_scan.log`
- Returns proper exit codes for monitoring
- Optional email notification on failure

**Best For:**
- Simple scheduled tasks
- Windows systems without PowerShell

---

### 2. `scheduled_compliance_scan.ps1`
**PowerShell Script** with advanced features

**Usage:**
```powershell
# Basic usage
.\scheduled_compliance_scan.ps1

# With archiving
.\scheduled_compliance_scan.ps1 -ArchiveReports -ArchiveBase "C:\Compliance\Archive"

# With email notifications
.\scheduled_compliance_scan.ps1 `
    -EmailTo "admin@company.com" `
    -EmailFrom "compliance@company.com" `
    -SmtpServer "smtp.company.com"

# Run specific report
.\scheduled_compliance_scan.ps1 -ReportName "NIST_800_171_compliance.json"

# Combine all features
.\scheduled_compliance_scan.ps1 `
    -ReportName "all" `
    -ArchiveReports `
    -ArchiveBase "\\fileserver\compliance\archive" `
    -EmailTo "security-team@company.com" `
    -SmtpServer "smtp.company.com"
```

**Parameters:**
- `-ToolkitPath`: Path to ComplianceToolkit.exe (default: `.\ComplianceToolkit.exe`)
- `-ReportName`: Report to run (default: `all`)
- `-ArchiveReports`: Enable archiving of reports
- `-ArchiveBase`: Archive directory (default: `.\archive`)
- `-EmailTo`: Email address for notifications
- `-EmailFrom`: From address (default: `compliance@company.com`)
- `-SmtpServer`: SMTP server for email

**Features:**
- Automatic report archiving by date
- Cleanup of old archives (>90 days)
- Email notifications (success and failure)
- Comprehensive logging
- Error handling

**Best For:**
- Enterprise environments
- Advanced automation
- Compliance teams needing notifications

---

## Windows Task Scheduler Setup

### Method 1: Using Batch Script

1. Open Task Scheduler (`taskschd.msc`)
2. Create Basic Task
3. Configure:
   - **Name:** "Daily Compliance Scan"
   - **Trigger:** Daily at 2:00 AM
   - **Action:** Start a program
   - **Program:** `C:\Path\To\ComplianceToolkit.exe`
   - **Arguments:** `-report=all -quiet`
   - **Start in:** `C:\Path\To\`
4. Settings:
   - ✓ Run whether user is logged on or not
   - ✓ Run with highest privileges

### Method 2: Using PowerShell Script

1. Open Task Scheduler (`taskschd.msc`)
2. Create Basic Task
3. Configure:
   - **Name:** "Advanced Compliance Scan with Archive"
   - **Trigger:** Weekly on Sunday at 3:00 AM
   - **Action:** Start a program
   - **Program:** `powershell.exe`
   - **Arguments:** `-ExecutionPolicy Bypass -File "C:\Path\To\examples\scheduled_compliance_scan.ps1" -ArchiveReports`
   - **Start in:** `C:\Path\To\examples\`
4. Settings:
   - ✓ Run whether user is logged on or not
   - ✓ Run with highest privileges

---

## Testing Scripts

### Test Batch Script
```batch
cd examples
scheduled_compliance_scan.bat
echo Exit Code: %ERRORLEVEL%
```

### Test PowerShell Script
```powershell
cd examples
.\scheduled_compliance_scan.ps1 -Verbose
echo "Exit Code: $LASTEXITCODE"
```

---

## Monitoring Script Execution

### View Scheduled Task History
1. Open Task Scheduler
2. Select your task
3. Click "History" tab
4. Review execution results

### View Application Logs
```powershell
# View latest log file
Get-Content ..\output\logs\scheduled_scan.log -Tail 50
```

### Check for Failed Scans
```powershell
# Search for errors in logs
Select-String -Path ..\output\logs\*.log -Pattern "ERROR"
```

---

## Customization

### Change Report Type

**Batch:**
```batch
SET REPORT=NIST_800_171_compliance.json
"%TOOLKIT_EXE%" -report=%REPORT% -quiet
```

**PowerShell:**
```powershell
.\scheduled_compliance_scan.ps1 -ReportName "fips_140_2_compliance.json"
```

### Archive to Network Share

**PowerShell:**
```powershell
.\scheduled_compliance_scan.ps1 `
    -ArchiveReports `
    -ArchiveBase "\\fileserver\compliance\archive"
```

### Increase Timeout for Slow Systems

**Batch:**
```batch
"%TOOLKIT_EXE%" -report=all -quiet -timeout=60s
```

**PowerShell:**
```powershell
Start-Process -FilePath $ToolkitExe -ArgumentList "-report=all -quiet -timeout=60s"
```

---

## Troubleshooting

### Script Doesn't Run from Task Scheduler

**Check:**
1. User account has admin privileges
2. "Run with highest privileges" is checked
3. Paths are absolute (not relative)
4. Script execution policy allows PowerShell scripts

**Fix PowerShell Execution Policy:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Email Notifications Not Sending

**Check:**
1. SMTP server is correct
2. Network connectivity to SMTP server
3. Firewall allows outbound SMTP (port 25/587)
4. Email credentials if required

**Test SMTP:**
```powershell
Send-MailMessage -To "test@company.com" -From "test@company.com" -Subject "Test" -Body "Test" -SmtpServer "smtp.company.com"
```

### Reports Not Archived

**Check:**
1. Archive directory exists and is writable
2. `-ArchiveReports` switch is specified
3. Network share is accessible

**Test Archive Path:**
```powershell
New-Item -ItemType Directory -Path "C:\Archive\Test" -Force
Test-Path "C:\Archive\Test"  # Should return True
```

---

## Security Considerations

### 1. **Store Credentials Securely**
Don't hardcode SMTP credentials in scripts. Use Windows Credential Manager:

```powershell
# Store credential
$Cred = Get-Credential
$Cred | Export-Clixml -Path "$env:USERPROFILE\smtp_cred.xml"

# Use in script
$Cred = Import-Clixml -Path "$env:USERPROFILE\smtp_cred.xml"
Send-MailMessage -Credential $Cred ...
```

### 2. **Restrict Script Permissions**
```powershell
# Allow only Administrators to execute
$Acl = Get-Acl scheduled_compliance_scan.ps1
$Acl.SetAccessRuleProtection($true, $false)
$Rule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators", "FullControl", "Allow")
$Acl.SetAccessRule($Rule)
Set-Acl scheduled_compliance_scan.ps1 $Acl
```

### 3. **Use Service Accounts**
Run scheduled tasks with dedicated service accounts instead of personal accounts.

---

## Next Steps

1. **Test manually** before scheduling
2. **Review generated reports** to ensure accuracy
3. **Set up monitoring** for failed tasks
4. **Configure email notifications** for compliance team
5. **Archive old reports** regularly to save disk space

---

**For more information, see:**
- [CLI_USAGE.md](../Documentation/CLI_USAGE.md) - Complete CLI documentation
- [PROJECT_STATUS.md](../Documentation/PROJECT_STATUS.md) - Project overview
- [QUICK_REFERENCE.md](../Documentation/QUICK_REFERENCE.md) - Quick start guide
