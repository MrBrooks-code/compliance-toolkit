# Evidence Logging Reference

**Version:** 1.1.0
**Last Updated:** 2025-01-05

Complete guide to compliance evidence logging, audit trails, and forensic documentation in the Compliance Toolkit.

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Evidence File Format](#evidence-file-format)
4. [Using Evidence for Compliance](#using-evidence-for-compliance)
5. [Automated Monitoring](#automated-monitoring)
6. [Querying Evidence Files](#querying-evidence-files)
7. [Security & Best Practices](#security--best-practices)

---

## Overview

The Compliance Toolkit automatically generates **comprehensive audit trail logs** for every scan. These JSON-formatted evidence files serve as compliance proof and can be submitted to auditors.

### What You Get

Every time you run a compliance report, the toolkit automatically creates:

1. **📋 Evidence JSON File** - Complete audit trail
2. **📄 HTML Report** - Visual report for stakeholders
3. **📊 Console Summary** - Immediate feedback

### Evidence File Benefits

✅ **Tamper-Evident**: JSON format with timestamps
✅ **Complete**: Every check is logged with full details
✅ **Machine-Readable**: Can be parsed by automated tools
✅ **Human-Readable**: JSON is easy to review
✅ **Timestamped**: Proves when scan was performed
✅ **Operator-Tracked**: Shows who ran the scan
✅ **Machine-Identified**: Full system details included

---

## Quick Start

### Step 1: Run a Scan

```bash
.\ComplianceToolkit.exe
```

Select a report (e.g., `[1] NIST 800-171 Security Compliance Report`)

### Step 2: See the Evidence Summary

After the scan completes, you'll see:

```
COMPLIANCE SCAN SUMMARY
═══════════════════════════════════════════════════════════
Scan ID:          SCAN_20250105_123045
Machine:          DESKTOP-ABC123 (AMD64)
OS:               Windows 10 Pro Professional (Build 22631)
Operator:         john.smith
Scan Time:        2025-01-05 12:30:45 MST
Duration:         17.8s

RESULTS:
  Total Checks:   13
  ✅ Passed:      10
  ❌ Failed:      0
  ⚠️  Not Found:  2
  🔴 Errors:      1

COMPLIANCE RATE:  90.91%

Evidence Log:     output/evidence/NIST_800_171_compliance_evidence_20250105_123045.json
═══════════════════════════════════════════════════════════
```

### Step 3: Find Your Evidence Files

```
output/
├── evidence/
│   ├── NIST_800_171_compliance_evidence_20250105_123045.json
│   ├── fips_140_2_compliance_evidence_20250105_123100.json
│   └── system_info_evidence_20250105_123200.json
├── reports/
│   └── (HTML reports)
└── logs/
    └── (Application logs)
```

---

## Evidence File Format

### Complete Structure

```json
{
  "scan_metadata": { ... },
  "machine_information": { ... },
  "scan_results": [ ... ],
  "compliance_summary": { ... }
}
```

### 1. Scan Metadata

Information about the scan itself:

```json
{
  "scan_metadata": {
    "report_type": "NIST_800_171_compliance",
    "scan_time": "2025-01-05T12:30:45.123456789-07:00",
    "toolkit_version": "1.1.0",
    "scan_id": "SCAN_20250105_123045",
    "operator": "john.smith",
    "duration_seconds": 17.863
  }
}
```

**Fields:**
- `report_type` - Which compliance report was run
- `scan_time` - ISO 8601 timestamp of scan start
- `toolkit_version` - Version of Compliance Toolkit
- `scan_id` - Unique identifier for this scan
- `operator` - Windows username of person running scan
- `duration_seconds` - How long the scan took

### 2. Machine Information

Complete system identification:

```json
{
  "machine_information": {
    "hostname": "DESKTOP-ABC123",
    "os_product_name": "Windows 10 Pro",
    "os_edition": "Professional",
    "os_build_number": "22631",
    "os_version": "10.0",
    "registered_owner": "John Smith",
    "registered_organization": "Acme Corporation",
    "install_date": "1609459200",
    "system_root": "C:\\Windows",
    "processor_architecture": "AMD64",
    "scan_timestamp": "2025-01-05T12:30:45.234567890-07:00"
  }
}
```

**Fields:**
- `hostname` - Computer name
- `os_product_name` - Windows product name
- `os_edition` - Windows edition
- `os_build_number` - Windows build number
- `os_version` - Windows version
- `registered_owner` - From Windows registration
- `registered_organization` - From Windows registration
- `install_date` - When Windows was installed (Unix timestamp)
- `system_root` - Windows installation directory
- `processor_architecture` - CPU architecture (AMD64, ARM64, etc.)
- `scan_timestamp` - When machine info was gathered

### 3. Scan Results

For **each registry check**, the log includes:

```json
{
  "scan_results": [
    {
      "check_name": "uac_enabled",
      "description": "User Account Control (UAC) Status",
      "root_key": "HKLM",
      "registry_path": "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System",
      "value_name": "EnableLUA",
      "operation": "read",
      "expected_value": "1 (Enabled)",
      "actual_value": "1",
      "status": "success",
      "timestamp": "2025-01-05T12:30:47.123456789-07:00"
    },
    {
      "check_name": "smb_v1_enabled",
      "description": "SMBv1 Protocol Status (should be disabled)",
      "root_key": "HKLM",
      "registry_path": "SYSTEM\\CurrentControlSet\\Services\\LanmanServer\\Parameters",
      "value_name": "SMB1",
      "operation": "read",
      "expected_value": "0 or not present (Disabled)",
      "actual_value": "",
      "status": "error",
      "error_message": "Registry key or value does not exist",
      "timestamp": "2025-01-05T12:30:49.345678901-07:00"
    }
  ]
}
```

**Fields per Check:**
- `check_name` - Unique identifier
- `description` - What the check validates
- `root_key` - Registry root (HKLM, HKCU, etc.)
- `registry_path` - Full path to registry key
- `value_name` - Specific registry value checked
- `operation` - Type of operation (read, read_all)
- `expected_value` - What should be found (if applicable)
- `actual_value` - What was actually found
- `status` - `success`, `error`, or `not_found`
- `error_message` - Error details (if status is error)
- `timestamp` - When this check was performed

### 4. Compliance Summary

Aggregated results:

```json
{
  "compliance_summary": {
    "total_checks": 13,
    "passed": 10,
    "failed": 0,
    "not_found": 2,
    "errors": 1,
    "compliance_rate": 76.92,
    "timestamp": "2025-01-05T12:31:02.987654321-07:00"
  }
}
```

**Fields:**
- `total_checks` - Number of compliance checks performed
- `passed` - Successful reads
- `failed` - Checks that failed validation
- `not_found` - Registry keys/values that don't exist
- `errors` - Checks that encountered errors
- `compliance_rate` - Percentage of successful checks
- `timestamp` - When summary was generated

---

## Using Evidence for Compliance

### For Audits

**Step 1: Run the compliance scan**
```bash
ComplianceToolkit.exe -report=NIST_800_171_compliance.json -quiet
```

**Step 2: Collect the evidence package**
- Evidence JSON: `output/evidence/NIST_800_171_compliance_evidence_*.json`
- HTML Report: `output/reports/NIST_800-171_Security_Compliance_Report_*.html`

**Step 3: Submit to auditor**
Both files together provide complete compliance documentation.

### Evidence Proves

✅ **What machine** was scanned (hostname, OS, build)
✅ **When** the scan was performed (ISO 8601 timestamp)
✅ **Who** performed the scan (Windows username)
✅ **Exactly what** was checked (registry paths and values)
✅ **What values** were found (actual registry data)
✅ **Compliance rate** percentage
✅ **Immutable audit trail** (JSON format with timestamps)

### Compliance Use Cases

#### SOC 2 Compliance
- Submit evidence files showing security configurations
- Prove UAC, Firewall, Windows Defender are enabled
- Document system hardening measures

#### HIPAA Compliance
- Evidence of workstation security settings
- Proof of access controls (UAC)
- Documentation of encryption status (BitLocker)

#### PCI DSS Compliance
- Security configuration validation
- Firewall status evidence
- System hardening documentation

#### NIST 800-171 / CMMC
- CUI protection controls validation
- Access control evidence
- Audit trail for compliance certification

#### Internal Audits
- Regular compliance scans
- Historical evidence trail
- Configuration drift detection

---

## Automated Monitoring

### Scheduled Scanning

Create a scheduled task to run daily scans and archive evidence:

**PowerShell Script** (`scheduled_compliance_scan.ps1`):

```powershell
# Run compliance scan
& "C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet

if ($LASTEXITCODE -ne 0) {
    Write-Error "Compliance scan failed!"

    # Send alert email
    Send-MailMessage `
        -To "security-team@company.com" `
        -From "compliance@company.com" `
        -Subject "Compliance Scan Failed" `
        -Body "Check logs at C:\ComplianceTool\output\logs" `
        -SmtpServer "smtp.company.com"

    exit 1
}

# Archive evidence files by date
$archiveDate = Get-Date -Format "yyyy-MM-dd"
$archivePath = "C:\Compliance\Archive\$archiveDate"

New-Item -ItemType Directory -Path $archivePath -Force | Out-Null

Copy-Item "C:\ComplianceTool\output\evidence\*.json" -Destination $archivePath -Force
Copy-Item "C:\ComplianceTool\output\reports\*.html" -Destination $archivePath -Force

Write-Host "Evidence archived to: $archivePath" -ForegroundColor Green
```

### Evidence Retention Policy

**Recommended retention:**

| Period | Location | Purpose |
|--------|----------|---------|
| **Last 30 days** | `output/evidence/` | Active monitoring |
| **31-365 days** | `archive/` | Audit preparation |
| **> 1 year** | Secure backup | Compliance history |

**Cleanup Script:**

```powershell
# Archive old evidence files (>30 days)
$archiveDate = (Get-Date).AddDays(-30)

Get-ChildItem output/evidence/*.json |
    Where-Object {$_.CreationTime -lt $archiveDate} |
    Move-Item -Destination "archive/" -Force

Write-Host "Old evidence files archived" -ForegroundColor Green
```

---

## Querying Evidence Files

### Using PowerShell

**Load and analyze evidence:**

```powershell
# Load evidence file
$evidence = Get-Content output/evidence/NIST_800_171_compliance_evidence_*.json | ConvertFrom-Json

# Check compliance rate
Write-Host "Compliance Rate: $($evidence.compliance_summary.compliance_rate)%" -ForegroundColor Cyan

# Display machine info
Write-Host "`nMachine: $($evidence.machine_information.hostname)"
Write-Host "OS: $($evidence.machine_information.os_product_name) Build $($evidence.machine_information.os_build_number)"
Write-Host "Operator: $($evidence.scan_metadata.operator)"

# Find failed/error checks
$issues = $evidence.scan_results | Where-Object {$_.status -ne "success"}

if ($issues) {
    Write-Host "`nIssues Found:" -ForegroundColor Yellow
    foreach ($issue in $issues) {
        Write-Host "  - $($issue.check_name): $($issue.status) - $($issue.description)"
        if ($issue.error_message) {
            Write-Host "    Error: $($issue.error_message)" -ForegroundColor Red
        }
    }
} else {
    Write-Host "`nNo issues found - All checks passed!" -ForegroundColor Green
}

# Export to CSV for analysis
$evidence.scan_results |
    Select-Object check_name, description, status, actual_value, expected_value |
    Export-Csv -Path "compliance_results.csv" -NoTypeInformation

Write-Host "`nResults exported to: compliance_results.csv"
```

**Compare compliance over time:**

```powershell
# Compare two evidence files
$old = Get-Content archive/2025-01-01/NIST_800_171_compliance_evidence_*.json | ConvertFrom-Json
$new = Get-Content output/evidence/NIST_800_171_compliance_evidence_*.json | ConvertFrom-Json

Write-Host "Compliance Rate Change:"
Write-Host "  Previous: $($old.compliance_summary.compliance_rate)%"
Write-Host "  Current:  $($new.compliance_summary.compliance_rate)%"

$diff = $new.compliance_summary.compliance_rate - $old.compliance_summary.compliance_rate
$color = if ($diff -ge 0) { "Green" } else { "Red" }
Write-Host "  Change:   $diff%" -ForegroundColor $color
```

### Using Python

**Load and analyze evidence:**

```python
import json
from datetime import datetime

# Load evidence
with open('output/evidence/NIST_800_171_compliance_evidence_20250105_123045.json') as f:
    evidence = json.load(f)

# Print summary
print(f"Compliance Rate: {evidence['compliance_summary']['compliance_rate']}%")
print(f"Machine: {evidence['machine_information']['hostname']}")
print(f"Operator: {evidence['scan_metadata']['operator']}")
print(f"Scan Time: {evidence['scan_metadata']['scan_time']}")

# Find issues
print("\nIssues Found:")
for result in evidence['scan_results']:
    if result['status'] != 'success':
        print(f"  - {result['check_name']}: {result['status']}")
        print(f"    {result['description']}")
        if 'error_message' in result:
            print(f"    Error: {result['error_message']}")

# Calculate statistics
total = evidence['compliance_summary']['total_checks']
passed = evidence['compliance_summary']['passed']
failed = evidence['compliance_summary']['failed']
errors = evidence['compliance_summary']['errors']

print(f"\nStatistics:")
print(f"  Total Checks: {total}")
print(f"  Passed: {passed} ({passed/total*100:.1f}%)")
print(f"  Failed: {failed} ({failed/total*100:.1f}%)")
print(f"  Errors: {errors} ({errors/total*100:.1f}%)")
```

**Trend analysis:**

```python
import json
import glob
from collections import defaultdict

# Load all evidence files
evidence_files = sorted(glob.glob('output/evidence/NIST_800_171_compliance_evidence_*.json'))

compliance_trend = []

for file in evidence_files:
    with open(file) as f:
        evidence = json.load(f)
        compliance_trend.append({
            'date': evidence['scan_metadata']['scan_time'],
            'rate': evidence['compliance_summary']['compliance_rate'],
            'passed': evidence['compliance_summary']['passed'],
            'total': evidence['compliance_summary']['total_checks']
        })

# Print trend
print("Compliance Trend:")
for entry in compliance_trend:
    print(f"  {entry['date']}: {entry['rate']}% ({entry['passed']}/{entry['total']})")
```

---

## Security & Best Practices

### Best Practices

1. **Store Securely**: Keep evidence files in access-controlled directories
2. **Backup Regularly**: Include evidence files in regular backup schedule
3. **Hash Verification**: Calculate SHA-256 hash for integrity verification
4. **Access Logging**: Monitor who accesses evidence files
5. **Encryption**: Encrypt archived evidence files for long-term storage

### Generating Hash for Integrity

**PowerShell:**
```powershell
Get-FileHash output/evidence/NIST_800_171_compliance_evidence_20250105_123045.json -Algorithm SHA256

# Output:
# Algorithm       Hash
# ---------       ----
# SHA256          A1B2C3D4E5F6...
```

**Linux/Git Bash:**
```bash
sha256sum output/evidence/NIST_800_171_compliance_evidence_20250105_123045.json
```

**Store hash with evidence:**
```powershell
$hash = (Get-FileHash $evidenceFile -Algorithm SHA256).Hash
Set-Content "$evidenceFile.sha256" -Value $hash
```

### Access Control

**Recommended permissions:**

```powershell
# Evidence directory - Read-only for most users
icacls "C:\ComplianceTool\output\evidence" /grant "BUILTIN\Administrators:(OI)(CI)F" /grant "BUILTIN\Users:(OI)(CI)R"

# Archive - Restricted access
icacls "C:\Compliance\Archive" /grant "BUILTIN\Administrators:(OI)(CI)F" /grant "Security-Team:(OI)(CI)R"
```

### Encryption

**Encrypt archived evidence:**

```powershell
# Compress and encrypt old evidence
$password = Read-Host "Enter encryption password" -AsSecureString
$archiveDate = Get-Date -Format "yyyy-MM-dd"

Compress-Archive -Path "output/evidence/*.json" -DestinationPath "archive/evidence_$archiveDate.zip"

# Use 7-Zip or similar for AES-256 encryption
& "C:\Program Files\7-Zip\7z.exe" a -p -mhe=on "archive/evidence_$archiveDate.7z" "archive/evidence_$archiveDate.zip"

Remove-Item "archive/evidence_$archiveDate.zip" -Force
```

---

## Troubleshooting

### "Could not create evidence log"

**Cause:** Permission denied or directory doesn't exist

**Solution:**
```bash
# Run as Administrator
# OR create directory manually:
mkdir output\evidence
```

### "Could not gather machine info"

**Cause:** Some registry keys not accessible

**Solution:** Run as Administrator for full machine details

### Missing values in evidence

**Cause:** Registry keys don't exist on this system

**Solution:** Normal behavior - status will show `error` or `not_found`

### JSON parse errors

**Cause:** Corrupted evidence file

**Solution:** Regenerate the report, verify file integrity with hash

---

## Summary

Every compliance scan automatically creates:

1. **📋 Evidence JSON** - Complete audit trail with machine info, operator, timestamps, and all registry values
2. **📄 HTML Report** - Visual report for stakeholders
3. **📊 Console Summary** - Immediate feedback

### Key Features

✅ **Audit-Ready**: Evidence files meet SOC 2, HIPAA, PCI DSS, and NIST 800-171 requirements
✅ **Automated**: No manual steps required - evidence generated automatically
✅ **Complete**: Every check logged with full context and timestamps
✅ **Tamper-Evident**: JSON format with cryptographic hashing support
✅ **Queryable**: PowerShell and Python examples for analysis
✅ **Archivable**: Built-in retention and archiving support

**Evidence files are production-ready for compliance audits!** 🎯

---

## Next Steps

- ✅ **Automate Scans**: See [Automation Guide](../user-guide/AUTOMATION.md)
- ✅ **CLI Usage**: See [CLI Usage Guide](../user-guide/CLI_USAGE.md)
- ✅ **Available Reports**: See [Reports Reference](REPORTS.md)

---

*Evidence Logging Reference v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
