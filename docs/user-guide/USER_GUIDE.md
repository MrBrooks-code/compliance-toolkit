# Compliance Toolkit - User Guide

**Version:** 1.1.0
**Last Updated:** 2025-01-05

---

## Table of Contents

1. [Introduction](#introduction)
2. [Launching the Toolkit](#launching-the-toolkit)
3. [Main Menu](#main-menu)
4. [Running Reports](#running-reports)
5. [Viewing Reports](#viewing-reports)
6. [Understanding Results](#understanding-results)
7. [Dark Mode](#dark-mode)
8. [Search and Filter](#search-and-filter)
9. [Evidence Logs](#evidence-logs)
10. [Troubleshooting](#troubleshooting)

---

## Introduction

The Compliance Toolkit scans your Windows registry for compliance violations and generates professional HTML reports with:

- Interactive charts and visualizations
- Expected vs. Actual value comparisons
- Detailed registry paths and values
- JSON evidence logs for audits
- Dark mode support
- Search and filter capabilities

---

## Launching the Toolkit

### Method 1: Double-Click
```
1. Navigate to installation folder (e.g., C:\ComplianceTool\)
2. Double-click ComplianceToolkit.exe
3. Interactive menu appears
```

### Method 2: Command Line
```cmd
cd C:\ComplianceTool
ComplianceToolkit.exe
```

### Requires Administrator?

Some compliance checks require administrator privileges. Right-click → "Run as administrator" for full access.

---

## Main Menu

```
╔══════════════════════════════════════════════════════════════════════╗
║                    COMPLIANCE TOOLKIT v1.1.0                         ║
║              Windows Registry Compliance Scanner                     ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                      ║
║  [1] Run Reports          - Execute compliance scans                 ║
║  [2] View HTML Reports    - Open generated reports in browser        ║
║  [3] View Evidence Logs   - View JSON audit trails                   ║
║  [4] View Log Files       - View application logs                    ║
║  [5] Configuration        - View current settings                    ║
║  [6] About                - Version and information                  ║
║                                                                      ║
║  [0] Exit                                                            ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝
```

---

## Running Reports

### Select [1] Run Reports

The toolkit will display all available compliance reports:

```
Available Reports:
==================
  [1] NIST 800-171 Security Compliance Report
      Category: Security & Compliance
      Version: 2.0.0
      13 compliance checks

  [2] FIPS 140-2 Compliance Report
      Category: Security & Compliance
      Version: 1.0.0
      35 cryptographic checks

  [3] System Information Report
      Category: System Inventory
      Version: 1.0.0

  [4] Software Inventory Report
  [5] Network Configuration Report
  [6] User Settings Report
  [7] Performance Diagnostics Report

  [8] Run All Reports

  [0] Back to Main Menu
```

### Run a Single Report

```
1. Select report number (e.g., [1] for NIST 800-171)
2. Toolkit scans registry keys
3. Report generates automatically
4. Success message shows save location
```

**Example Output:**
```
Running NIST 800-171 Security Compliance Report
======================

  ✅  [uac_enabled] Success
  ✅  [firewall_domain_profile] Success
  ⚠️  [windows_defender_enabled] Not found
  ✅  [firewall_standard_profile] Success
  ...

  📊  Results: 11 successful, 2 errors
  📄  HTML Report: output\reports\NIST_800-171_Security_Compliance_Report_20251005_120530.html
  📋  Evidence Log: output/evidence/NIST_800_171_compliance_evidence_20251005_120530.json

Report completed successfully!
```

### Run All Reports

```
Select [8] Run All Reports
- Executes all 7 reports sequentially
- Takes 1-2 minutes
- Generates all HTML and evidence files
```

---

## Viewing Reports

### Option 1: From Menu

```
1. Select [2] View HTML Reports
2. Choose report from list
3. Report opens in default browser
```

### Option 2: Manual

```
Navigate to: output\reports\
Open: NIST_800-171_Security_Compliance_Report_YYYYMMDD_HHMMSS.html
```

---

## Understanding Results

### Report Header

```
┌─────────────────────────────────────────────────┐
│ 🛡️ Compliance Toolkit - NIST 800-171...         │
│ NIST 800-171 security controls validation...    │
│                                                  │
│ [Security & Compliance] [NIST 800-171 Rev 2]    │
│ [v2.0.0]                                        │
│                                                  │
│ 📅 Last Updated: 2025-01-04  👤 Author: ...     │
└─────────────────────────────────────────────────┘
```

### KPI Dashboard

```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│   69%        │     13       │      9       │      4       │
│ Compliance   │    Total     │    Passed    │   Failed     │
│   Rate       │   Checks     │              │              │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

### Compliance Chart

Interactive donut chart showing pass/fail ratio.

### Registry Check Details

```
Name: uac_enabled
Description: User Account Control (UAC) Status
Status: ✅ Success

▼ Click to expand details

Registry Details:
  Root Key: HKLM
  Path: SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System
  Value Name: EnableLUA
  Operation: read
  Expected Value: 1 (Enabled)
  Actual Value: 1
```

---

## Dark Mode

### Toggle Dark Mode

```
🌙 Button in top-right corner
- Click to toggle dark/light mode
- Preference saved in browser
- Automatic on next visit
```

### Dark Mode Features

- Reduced eye strain for long sessions
- Professional dark theme
- High contrast text
- Subtle hover effects
- Consistent across all sections

---

## Search and Filter

### Search Bar

```
[Search registry keys, values, or descriptions...]
```

**Search by:**
- Check name (e.g., "uac_enabled")
- Description (e.g., "User Account Control")
- Registry path (e.g., "HKLM\SOFTWARE")
- Value name (e.g., "EnableLUA")
- Expected/actual values
- Error messages

**Example Searches:**
- `firewall` - All firewall checks
- `HKLM` - All HKEY_LOCAL_MACHINE checks
- `disabled` - All disabled settings
- `error` - All failed checks

### Status Filter

```
[All Status ▼]
- All Status
- Success Only
- Errors Only
```

**Combine with search:**
- Search: "defender" + Filter: "Errors Only"
- Shows only failed Windows Defender checks

---

## Evidence Logs

### View Evidence Logs

```
1. Select [3] View Evidence Logs
2. Choose log file
3. Opens in default JSON viewer/text editor
```

### Evidence Log Contents

```json
{
  "scan_metadata": {
    "report_type": "NIST_800_171_compliance",
    "scan_time": "2025-01-05T12:05:30Z",
    "toolkit_version": "1.1.0"
  },
  "machine_info": {
    "hostname": "DESKTOP-ABC123",
    "os_version": "Windows 10 Pro",
    "architecture": "amd64"
  },
  "scan_results": [
    {
      "check_name": "uac_enabled",
      "description": "User Account Control (UAC) Status",
      "registry_path": "HKLM\\SOFTWARE\\Microsoft\\...",
      "value_name": "EnableLUA",
      "actual_value": "1",
      "status": "success"
    }
  ],
  "compliance_summary": {
    "total_checks": 13,
    "passed": 9,
    "failed": 4,
    "compliance_rate": 69.23
  }
}
```

### Use Cases

- Compliance audits
- Historical tracking
- Automated processing
- Trend analysis

---

## Troubleshooting

### Issue: "configs/reports not found"

**Cause:** Report configurations missing

**Solution:**
```cmd
# Verify configs directory exists
dir C:\ComplianceTool\configs\reports\*.json

# If missing, copy from distribution
xcopy configs\reports C:\ComplianceTool\configs\reports\ /E /I
```

### Issue: "Access denied" errors

**Cause:** Insufficient permissions

**Solution:**
```cmd
# Right-click ComplianceToolkit.exe
# Select "Run as administrator"
```

### Issue: Some registry keys "Not found"

**Cause:** Normal - not all keys exist on all systems

**Solution:**
- This is expected behavior
- "Not found" may be compliant for some checks
- Review expected value to determine if compliant
- Check evidence log for details

### Issue: Report won't open in browser

**Cause:** No default browser or file association

**Solution:**
```cmd
# Manually navigate to:
explorer output\reports

# Or specify browser:
"C:\Program Files\Mozilla Firefox\firefox.exe" output\reports\latest.html
```

### Issue: Dark mode text unreadable

**Cause:** Browser cache showing old styles

**Solution:**
```
1. Press Ctrl+F5 (hard refresh)
2. Clear browser cache
3. Regenerate report
```

### Issue: Search not working

**Cause:** JavaScript disabled or old cached report

**Solution:**
```
1. Enable JavaScript in browser
2. Clear cache and regenerate report
3. Try different browser
```

---

## Tips & Best Practices

### 1. Regular Scans

```
- Run weekly for compliance monitoring
- Compare results over time
- Track compliance improvements
```

### 2. Archive Reports

```
# Create dated folders
mkdir C:\Compliance\Archive\2025-01-05
copy output\reports\*.html C:\Compliance\Archive\2025-01-05\
copy output\evidence\*.json C:\Compliance\Archive\2025-01-05\
```

### 3. Share Results

```
- Email HTML reports to stakeholders
- Print to PDF for documentation
- Include evidence logs in audit packages
```

### 4. Custom Output

```
# Use different output directories for projects
ComplianceToolkit.exe -report=all -output=C:\Project1\Reports
```

---

## Next Steps

- ✅ **Automation**: See [CLI Usage Guide](CLI_USAGE.md)
- ✅ **Scheduled Tasks**: See [CLI Quick Start](CLI_QUICKSTART.md)
- ✅ **Custom Reports**: See [Adding Reports](../developer-guide/ADDING_REPORTS.md)
- ✅ **Evidence Logging**: See [Evidence Reference](../reference/EVIDENCE.md)

---

**For more information:**
- [Installation Guide](INSTALLATION.md)
- [CLI Usage](CLI_USAGE.md)
- [Project Status](../PROJECT_STATUS.md)

---

*User Guide v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
