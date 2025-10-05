# ComplianceToolkit CLI - Quick Start Guide

**Version:** 1.1.0
**5-Minute Setup for Scheduled Compliance Scans**

---

## Quick Commands

```bash
# List available reports
ComplianceToolkit.exe -list

# Run a specific report
ComplianceToolkit.exe -report=NIST_800_171_compliance.json

# Run all reports (for scheduled tasks)
ComplianceToolkit.exe -report=all -quiet

# Show help
ComplianceToolkit.exe -h
```

---

## Scheduled Task Setup (Windows)

### Option 1: Simple Batch Script (5 minutes)

1. **Open Task Scheduler:**
   ```
   Win + R → taskschd.msc → Enter
   ```

2. **Create Basic Task:**
   - Name: `Daily Compliance Scan`
   - Trigger: `Daily at 2:00 AM`

3. **Action:**
   - Program: `C:\Path\To\ComplianceToolkit.exe`
   - Arguments: `-report=all -quiet`
   - Start in: `C:\Path\To\`

4. **Settings:**
   - ✓ Run with highest privileges
   - ✓ Run whether user is logged on or not

5. **Done!** Reports will run daily at 2 AM.

---

### Option 2: Advanced PowerShell Script (10 minutes)

Use the provided `examples/scheduled_compliance_scan.ps1` script for:
- ✅ Automatic archiving
- ✅ Email notifications
- ✅ Old report cleanup
- ✅ Better error handling

**Setup:**

1. **Copy script to your directory:**
   ```powershell
   copy examples\scheduled_compliance_scan.ps1 C:\Compliance\
   ```

2. **Create Task Scheduler entry:**
   - Program: `powershell.exe`
   - Arguments: `-ExecutionPolicy Bypass -File "C:\Compliance\scheduled_compliance_scan.ps1" -ArchiveReports`
   - Trigger: `Weekly on Sunday at 3:00 AM`

3. **Optional: Add email notifications:**
   ```powershell
   -EmailTo "admin@company.com" -SmtpServer "smtp.company.com"
   ```

---

## Verify It Works

### Test Manually First:
```bash
# Test in normal mode (with output)
ComplianceToolkit.exe -report=system_info.json

# Test in quiet mode (scheduled task mode)
ComplianceToolkit.exe -report=all -quiet
echo Exit Code: %ERRORLEVEL%
```

**Expected:** Exit code 0 = Success

### Check Results:
```bash
# View generated reports
dir output\reports\*.html

# View evidence logs
dir output\evidence\*.json

# View application logs
dir output\logs\*.log
```

---

## Common Schedules

### Daily Security Scan (2 AM)
```
-report=NIST_800_171_compliance.json -quiet
```

### Weekly Full Audit (Sunday 3 AM)
```
-report=all -quiet
```

### Monthly Deep Scan (1st of month)
```
-report=all -timeout=60s -quiet
```

---

## Monitoring

### Check Task Scheduler History:
1. Open Task Scheduler
2. Select your task
3. Click **History** tab
4. Look for "Task completed" messages

### Check Application Logs:
```powershell
# View latest log
Get-Content output\logs\toolkit_*.log -Tail 50

# Search for errors
Select-String -Path output\logs\*.log -Pattern "ERROR"
```

---

## Troubleshooting

### Task doesn't run?
- ✓ Check "Run with highest privileges"
- ✓ Use absolute paths (not relative)
- ✓ Verify user has admin rights

### Reports not generated?
- ✓ Check `output/logs/` for errors
- ✓ Verify `configs/reports/` exists
- ✓ Test manually first

### Permission errors?
```bash
# Run as Administrator
Start-Process ComplianceToolkit.exe -Verb RunAs
```

---

## Next Steps

1. ✅ **Test manually** - Verify reports generate correctly
2. ✅ **Set up schedule** - Add to Task Scheduler
3. ✅ **Monitor first run** - Check logs after first scheduled execution
4. ✅ **Archive old reports** - Set up cleanup script (optional)

**For complete documentation:**
- See [CLI_USAGE.md](Documentation/CLI_USAGE.md) - Full CLI reference
- See [examples/README.md](examples/README.md) - Example scripts
- See [PROJECT_STATUS.md](PROJECT_STATUS.md) - Project overview

---

**Need Help?**
Check the log files in `output/logs/` for detailed error messages.

---

*Quick Start Guide v1.0*
*ComplianceToolkit v1.1.0*
