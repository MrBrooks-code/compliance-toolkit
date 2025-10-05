# JSON Configuration Guide

## Overview

The registry reader supports JSON-based configuration files for declarative registry operations. This allows you to define complex scanning operations without writing code.

## JSON Schema

### Root Structure

```json
{
  "version": "1.0",
  "queries": [
    // Array of query objects
  ]
}
```

### Query Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ Yes | Unique identifier for the query |
| `description` | string | ✅ Yes | Human-readable description |
| `root_key` | string | ✅ Yes | Registry root key (see below) |
| `path` | string | ✅ Yes | Registry key path |
| `operation` | string | ✅ Yes | Operation type: "read" or "write" |
| `value_name` | string | ❌ No | Specific value to read (omit for read_all) |
| `read_all` | boolean | ❌ No | Read all values in the key (default: false) |
| `write_type` | string | ❌ No | Type for write ops: "string", "dword", "qword", "binary", "multi_string" |
| `write_value` | any | ❌ No | Value to write (type depends on write_type) |

### Supported Root Keys

**Short Form** (recommended):
- `HKLM` - HKEY_LOCAL_MACHINE
- `HKCU` - HKEY_CURRENT_USER
- `HKCR` - HKEY_CLASSES_ROOT
- `HKU` - HKEY_USERS
- `HKCC` - HKEY_CURRENT_CONFIG

**Long Form** (also supported):
- `HKEY_LOCAL_MACHINE`
- `HKEY_CURRENT_USER`
- `HKEY_CLASSES_ROOT`
- `HKEY_USERS`
- `HKEY_CURRENT_CONFIG`

## Examples

### 1. Read Single Value

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "windows_product_name",
      "description": "Read Windows product name",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
      "value_name": "ProductName",
      "operation": "read"
    }
  ]
}
```

### 2. Read All Values from a Key

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "windows_version_info",
      "description": "Read all Windows version information",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
      "operation": "read",
      "read_all": true
    }
  ]
}
```

### 3. Multiple Queries

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "product_name",
      "description": "Windows product name",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
      "value_name": "ProductName",
      "operation": "read"
    },
    {
      "name": "build_number",
      "description": "Windows build number",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
      "value_name": "CurrentBuild",
      "operation": "read"
    },
    {
      "name": "user_wallpaper",
      "description": "Current user wallpaper path",
      "root_key": "HKCU",
      "path": "Control Panel\\Desktop",
      "value_name": "Wallpaper",
      "operation": "read"
    }
  ]
}
```

## Path Formatting Rules

### ✅ Correct Path Format

```json
"path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion"
```

**Important**: Use double backslashes (`\\`) in JSON strings to properly escape the path separator.

### ❌ Incorrect Path Format

```json
"path": "SOFTWARE\Microsoft\Windows NT\CurrentVersion"  // Single backslash - WRONG!
```

## Common Registry Paths

### System Information (HKLM)

```
SOFTWARE\Microsoft\Windows NT\CurrentVersion    - Windows version info
SOFTWARE\Microsoft\Windows\CurrentVersion       - Windows settings
SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName  - Computer name
SYSTEM\CurrentControlSet\Services               - Windows services
SOFTWARE\Classes                                - File associations
```

### User Information (HKCU)

```
Software\Microsoft\Windows\CurrentVersion\Explorer  - Explorer settings
Control Panel\Desktop                               - Desktop settings
Software\Microsoft\Windows\CurrentVersion\Run       - User startup programs
Environment                                         - User environment variables
```

## Pre-Built Report Configurations

See the `configs/reports/` directory for ready-to-use configurations:

1. **system_info.json** - Complete system information report
2. **security_audit.json** - Security configuration audit
3. **software_inventory.json** - Installed software inventory
4. **network_config.json** - Network configuration
5. **user_settings.json** - User preferences and settings

## Creating Your Own Configs

### Step 1: Start with Template

```json
{
  "version": "1.0",
  "queries": []
}
```

### Step 2: Add Queries

Use this template for each query:

```json
{
  "name": "your_query_name",
  "description": "What this query does",
  "root_key": "HKLM",
  "path": "Path\\To\\Registry\\Key",
  "value_name": "ValueName",
  "operation": "read"
}
```

### Step 3: Test Your Config

```bash
# Validate JSON syntax
cat your_config.json | jq .

# Test with the application
go run ./cmd/main.go
```

## Tips & Best Practices

### 1. Naming Conventions

**Good Names:**
- `windows_product_name`
- `installed_dotnet_versions`
- `user_timezone_setting`

**Bad Names:**
- `query1`
- `test`
- `asdf`

### 2. Descriptions

**Good:**
```json
"description": "Read Windows product name and edition from registry"
```

**Bad:**
```json
"description": "read stuff"
```

### 3. Organize by Category

Group related queries together:

```json
{
  "version": "1.0",
  "queries": [
    {
      "name": "os_product_name",
      "description": "Operating System: Product Name",
      ...
    },
    {
      "name": "os_build_number",
      "description": "Operating System: Build Number",
      ...
    },
    {
      "name": "net_hostname",
      "description": "Network: Computer Hostname",
      ...
    }
  ]
}
```

### 4. Use read_all for Exploration

When you don't know all value names:

```json
{
  "name": "explore_windows_version",
  "description": "Discover all Windows version values",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
  "operation": "read",
  "read_all": true
}
```

## Validation Checklist

Before running your config:

- [ ] Valid JSON syntax (use `jq` or JSON validator)
- [ ] All required fields present
- [ ] Paths use double backslashes (`\\`)
- [ ] Root keys are valid (HKLM, HKCU, etc.)
- [ ] Operation is "read" (write not implemented yet)
- [ ] Unique names for all queries
- [ ] Descriptive names and descriptions

## Error Handling

### Common Errors

**1. Invalid JSON Syntax**
```
Error: failed to parse config JSON: invalid character...
```
→ Fix: Check for missing commas, quotes, brackets

**2. Invalid Root Key**
```
Error: unknown root key: HKLM2
```
→ Fix: Use valid root key (HKLM, HKCU, HKCR, HKU, HKCC)

**3. Path Not Found**
```
Error: registry OpenKey failed for SOFTWARE\BadPath: The system cannot find the file specified
```
→ Fix: Verify the registry path exists using `regedit`

**4. Value Not Found**
```
Error: registry GetStringValue failed for ProductName2: The system cannot find the file specified
```
→ Fix: Check value name spelling and existence

## Advanced Patterns

### Pattern 1: Compliance Scanning

```json
{
  "name": "check_auto_update_enabled",
  "description": "Verify Windows Auto Update is enabled",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate\\AU",
  "value_name": "NoAutoUpdate",
  "operation": "read"
}
```

### Pattern 2: Software Inventory

```json
{
  "name": "installed_applications",
  "description": "List all installed applications",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
  "operation": "read",
  "read_all": true
}
```

### Pattern 3: Security Settings

```json
{
  "name": "uac_enabled",
  "description": "Check if UAC is enabled",
  "root_key": "HKLM",
  "path": "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System",
  "value_name": "EnableLUA",
  "operation": "read"
}
```

## Next Steps

1. Review the example reports in `configs/reports/`
2. Copy a template that matches your use case
3. Customize the queries for your needs
4. Test with `go run ./cmd/main.go`
5. Iterate and refine

## Resources

- **Registry Editor**: Run `regedit` to explore available keys
- **JSON Validator**: https://jsonlint.com/
- **Registry Documentation**: https://learn.microsoft.com/en-us/windows/win32/sysinfo/registry
