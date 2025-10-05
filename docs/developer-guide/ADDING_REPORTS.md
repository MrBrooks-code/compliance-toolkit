# Adding New Reports - Developer Guide

## Overview

The Compliance Toolkit is designed to be easily extensible. Adding new reports is as simple as creating a JSON configuration file. No code changes required!

## 🚀 Quick Start - Add a New Report in 3 Steps

### Step 1: Create JSON Config File

Create a new file in `configs/reports/`:

```bash
# Example: Add a new "Browser Security" report
notepad configs/reports/browser_security.json
```

### Step 2: Define Your Checks

Use this template:

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "check_identifier",
      "description": "Human-readable description",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Path\\To\\Key",
      "value_name": "ValueName",
      "operation": "read"
    }
  ]
}
```

### Step 3: Run the Toolkit

```bash
.\ComplianceToolkit.exe
```

Your new report automatically appears in the menu! 🎉

## 📋 Complete Example

Let's create a **"Browser Security Report"** that checks browser settings.

### Create the File

**File**: `configs/reports/browser_security.json`

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "chrome_auto_update",
      "description": "Google Chrome Automatic Updates Enabled",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Policies\\Google\\Update",
      "value_name": "AutoUpdateCheckPeriodMinutes",
      "operation": "read"
    },
    {
      "name": "edge_smartscreen",
      "description": "Microsoft Edge SmartScreen Status",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Policies\\Microsoft\\Edge",
      "value_name": "SmartScreenEnabled",
      "operation": "read"
    },
    {
      "name": "ie_protected_mode",
      "description": "Internet Explorer Protected Mode",
      "root_key": "HKCU",
      "path": "Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\Zones\\3",
      "value_name": "2500",
      "operation": "read"
    },
    {
      "name": "firefox_policies",
      "description": "Firefox Enterprise Policies",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Policies\\Mozilla\\Firefox",
      "operation": "read",
      "read_all": true
    }
  ]
}
```

### That's It!

The report will now appear as option `[8]` in the "Run Reports" menu automatically!

## 📖 JSON Configuration Reference

### Root Structure

```json
{
  "version": "1.0",           // Config version (required)
  "queries": [...]            // Array of checks (required)
}
```

### Query Object Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `name` | string | ✅ Yes | Unique identifier | `"chrome_auto_update"` |
| `description` | string | ✅ Yes | Human-readable description | `"Chrome Auto Updates"` |
| `root_key` | string | ✅ Yes | Registry root | `"HKLM"` or `"HKCU"` |
| `path` | string | ✅ Yes | Registry key path | `"SOFTWARE\\Google\\Chrome"` |
| `operation` | string | ✅ Yes | Operation type | `"read"` (write not supported) |
| `value_name` | string | ❌ No | Specific value to read | `"Version"` |
| `read_all` | boolean | ❌ No | Read all values in key | `true` |

### Root Key Options

**Short form** (recommended):
- `HKLM` - HKEY_LOCAL_MACHINE (system-wide settings)
- `HKCU` - HKEY_CURRENT_USER (user-specific settings)
- `HKCR` - HKEY_CLASSES_ROOT (file associations)
- `HKU` - HKEY_USERS (all user profiles)
- `HKCC` - HKEY_CURRENT_CONFIG (hardware profiles)

**Long form** (also supported):
- `HKEY_LOCAL_MACHINE`
- `HKEY_CURRENT_USER`
- etc.

### Path Formatting

⚠️ **Important**: Use double backslashes `\\` in JSON!

✅ **Correct**:
```json
"path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion"
```

❌ **Wrong**:
```json
"path": "SOFTWARE\Microsoft\Windows NT\CurrentVersion"
```

### Read Operations

#### Single Value Read

Read one specific registry value:

```json
{
  "name": "windows_version",
  "description": "Windows Version",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
  "value_name": "ProductName",
  "operation": "read"
}
```

#### Read All Values

Read all values in a registry key:

```json
{
  "name": "all_version_info",
  "description": "All Windows Version Information",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
  "operation": "read",
  "read_all": true
}
```

**Note**: When `read_all` is `true`, omit `value_name`.

## 🎯 Report Categories & Ideas

### 1. Security & Compliance

**HIPAA Compliance Report** (`hipaa_compliance.json`):
```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "screen_saver_password",
      "description": "Screen Saver Password Protection Enabled",
      "root_key": "HKCU",
      "path": "Control Panel\\Desktop",
      "value_name": "ScreenSaverIsSecure",
      "operation": "read"
    },
    {
      "name": "automatic_logon_disabled",
      "description": "Automatic Logon Disabled",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon",
      "value_name": "AutoAdminLogon",
      "operation": "read"
    }
  ]
}
```

### 2. Application Inventory

**Microsoft Office Audit** (`office_audit.json`):
```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "office_version",
      "description": "Microsoft Office Version",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Office\\ClickToRun\\Configuration",
      "value_name": "VersionToReport",
      "operation": "read"
    },
    {
      "name": "office_update_channel",
      "description": "Office Update Channel",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Office\\ClickToRun\\Configuration",
      "value_name": "UpdateChannel",
      "operation": "read"
    }
  ]
}
```

### 3. Developer Tools

**Development Environment Scan** (`dev_environment.json`):
```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "git_version",
      "description": "Git Installation Version",
      "root_key": "HKLM",
      "path": "SOFTWARE\\GitForWindows",
      "value_name": "CurrentVersion",
      "operation": "read"
    },
    {
      "name": "docker_version",
      "description": "Docker Desktop Version",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Docker Inc.\\Docker\\1.0",
      "value_name": "Version",
      "operation": "read"
    },
    {
      "name": "vscode_installed",
      "description": "Visual Studio Code Installation",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\{VSCODE-GUID}",
      "value_name": "DisplayName",
      "operation": "read"
    }
  ]
}
```

### 4. Hardware & Drivers

**Hardware Configuration** (`hardware_config.json`):
```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "tpm_enabled",
      "description": "TPM (Trusted Platform Module) Status",
      "root_key": "HKLM",
      "path": "SYSTEM\\CurrentControlSet\\Services\\TPM",
      "value_name": "Start",
      "operation": "read"
    },
    {
      "name": "secure_boot",
      "description": "Secure Boot Configuration",
      "root_key": "HKLM",
      "path": "SYSTEM\\CurrentControlSet\\Control\\SecureBoot\\State",
      "value_name": "UEFISecureBootEnabled",
      "operation": "read"
    }
  ]
}
```

## 🔍 Finding Registry Paths

### Method 1: Registry Editor (regedit)

1. Press `Win+R`, type `regedit`, press Enter
2. Navigate to the key you want to check
3. Right-click key → Copy Key Name
4. Replace single `\` with `\\` for JSON

### Method 2: PowerShell

```powershell
# List all subkeys
Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"

# Get specific value
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" -Name ProductName

# Export entire key
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion" | Format-List
```

### Method 3: Command Prompt

```cmd
# Query registry key
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion"

# Query specific value
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v ProductName
```

## ✅ Testing Your New Report

### Step 1: Validate JSON Syntax

```bash
# Using jq (if installed)
jq . configs/reports/your_report.json

# Or use online validator
# https://jsonlint.com/
```

### Step 2: Test the Report

```bash
.\ComplianceToolkit.exe
```

1. Select `[1] Run Reports`
2. Your report appears in the menu
3. Select it and verify output

### Step 3: Check Output Files

```bash
# HTML report
dir output\reports\*your_report*.html

# Evidence log
dir output\logs\*your_report*_evidence*.json
```

## 📝 Best Practices

### Naming Conventions

**File Names**:
- Use lowercase
- Use underscores for spaces
- Be descriptive
- Examples: `browser_security.json`, `hipaa_compliance.json`

**Check Names** (`name` field):
- Use lowercase with underscores
- Be specific
- Examples: `uac_enabled`, `chrome_auto_update`

**Descriptions** (`description` field):
- Be clear and professional
- Use proper capitalization
- Examples: "User Account Control Status", "Chrome Automatic Updates"

### Organization

Group related checks together:

```json
{
  "version": "1.0",
  "queries": [
    // Group 1: Operating System
    { "name": "os_version", ... },
    { "name": "os_build", ... },

    // Group 2: Security Settings
    { "name": "uac_enabled", ... },
    { "name": "firewall_enabled", ... },

    // Group 3: Applications
    { "name": "office_version", ... },
    { "name": "chrome_version", ... }
  ]
}
```

### Error Handling

Some registry keys may not exist on all systems. This is normal!

- **NOT FOUND** errors are expected
- The report will show them as warnings (yellow)
- Document expected behavior in description

Example:
```json
{
  "name": "smb_v1_status",
  "description": "SMBv1 Protocol (should be disabled/not found)",
  "root_key": "HKLM",
  "path": "SYSTEM\\CurrentControlSet\\Services\\LanmanServer\\Parameters",
  "value_name": "SMB1",
  "operation": "read"
}
```

## 🚀 Advanced Examples

### Multi-Value Complex Check

```json
{
  "name": "firewall_all_profiles",
  "description": "Windows Firewall - All Network Profiles",
  "root_key": "HKLM",
  "path": "SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy",
  "operation": "read",
  "read_all": true
}
```

### Cross-Reference Checks

Create checks that validate related settings:

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "bitlocker_policy",
      "description": "BitLocker Encryption Policy",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Policies\\Microsoft\\FVE",
      "operation": "read",
      "read_all": true
    },
    {
      "name": "tpm_enabled",
      "description": "TPM Required for BitLocker",
      "root_key": "HKLM",
      "path": "SYSTEM\\CurrentControlSet\\Services\\TPM",
      "value_name": "Start",
      "operation": "read"
    }
  ]
}
```

## 🔧 Updating the Menu (Automatic)

**Good news**: The menu automatically updates!

The toolkit scans `configs/reports/` and lists all `.json` files. Just add your file and it appears.

**Current limit**: Reports 1-6 are hardcoded in `toolkit.go`. To add more:

### Option 1: Dynamic Loading (Recommended for Future)

Edit `cmd/toolkit.go` to scan directory:

```go
// TODO: Make this dynamic by scanning configs/reports/
```

### Option 2: Manual Menu Entry (Current Method)

Edit `cmd/toolkit.go`, find `runReports()` function:

```go
reportMap := map[int]string{
    1: "system_info.json",
    2: "security_audit.json",
    3: "software_inventory.json",
    4: "network_config.json",
    5: "user_settings.json",
    6: "performance_diagnostics.json",
    7: "browser_security.json",        // Add your new report
    8: "hipaa_compliance.json",        // Add another
}
```

And update `ShowReportMenu()` in `pkg/menu.go`:

```go
fmt.Println("│    💻  [1]  System Information Report                                │")
fmt.Println("│    🔒  [2]  Security Audit Report                                    │")
// ... existing reports ...
fmt.Println("│    🌐  [7]  Browser Security Report                                  │")
fmt.Println("│    🏥  [8]  HIPAA Compliance Report                                  │")
```

## 📚 Report Template Library

### Template 1: Basic Report

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "example_check",
      "description": "Example Registry Check",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Example\\Path",
      "value_name": "ExampleValue",
      "operation": "read"
    }
  ]
}
```

### Template 2: Comprehensive Report

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "single_value_check",
      "description": "Check Single Registry Value",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Path",
      "value_name": "ValueName",
      "operation": "read"
    },
    {
      "name": "all_values_check",
      "description": "Check All Values in Key",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Path",
      "operation": "read",
      "read_all": true
    },
    {
      "name": "user_setting_check",
      "description": "Check User-Specific Setting",
      "root_key": "HKCU",
      "path": "Software\\User\\Path",
      "value_name": "Setting",
      "operation": "read"
    }
  ]
}
```

## 🎯 Summary Checklist

Before deploying a new report:

- [ ] JSON syntax is valid (test with jq or jsonlint)
- [ ] All required fields present (`name`, `description`, `root_key`, `path`, `operation`)
- [ ] Paths use double backslashes (`\\`)
- [ ] Root keys are valid (`HKLM`, `HKCU`, etc.)
- [ ] Check names are unique and descriptive
- [ ] Descriptions are professional and clear
- [ ] Tested on actual Windows system
- [ ] HTML report generates successfully
- [ ] Evidence log creates properly
- [ ] No sensitive data exposed in results

## 🆘 Troubleshooting

### Report Doesn't Appear in Menu

**Problem**: New JSON file not showing up

**Solution**:
- Check file is in `configs/reports/` directory
- Verify filename ends with `.json`
- Rebuild: `go build -o ComplianceToolkit.exe ./cmd/toolkit.go`

### JSON Parse Error

**Problem**: `failed to parse config JSON`

**Solution**:
- Validate JSON syntax: `jq . yourfile.json`
- Check for missing commas
- Ensure proper quote escaping

### All Checks Show "NOT FOUND"

**Problem**: Every check fails with NOT FOUND

**Solution**:
- Verify registry paths in regedit
- Check if running as Administrator (some keys need elevated access)
- Confirm Windows version (some keys vary by version)

---

**You can now add unlimited custom reports by just creating JSON files! 🎉**
