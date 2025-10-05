# Compliance Toolkit - Installation Guide

**Version:** 1.1.0
**Last Updated:** 2025-01-04

---

## Quick Installation

### Minimum Files Required

To deploy the Compliance Toolkit to a new system, you need:

```
C:\ComplianceTool\
├── ComplianceToolkit.exe          # Main executable
└── configs\
    └── reports\                   # Report configuration files
        ├── NIST_800_171_compliance.json
        ├── fips_140_2_compliance.json
        ├── system_info.json
        ├── software_inventory.json
        ├── network_config.json
        ├── user_settings.json
        └── performance_diagnostics.json
```

**Note:** Output directories (`output/`, `logs/`, `evidence/`) are created automatically.

---

## Installation Methods

### Method 1: Simple Copy (Recommended)

1. **Create installation directory:**
   ```cmd
   mkdir C:\ComplianceTool
   ```

2. **Copy files:**
   ```cmd
   copy ComplianceToolkit.exe C:\ComplianceTool\
   xcopy configs C:\ComplianceTool\configs\ /E /I
   ```

3. **Verify installation:**
   ```cmd
   cd C:\ComplianceTool
   ComplianceToolkit.exe -list
   ```

4. **Run your first report:**
   ```cmd
   ComplianceToolkit.exe -report=system_info.json
   ```

---

### Method 2: ZIP Distribution

1. **Create a ZIP package:**
   - Include `ComplianceToolkit.exe`
   - Include entire `configs/` directory
   - Optional: Include `examples/` directory for automation scripts
   - Optional: Include `Documentation/` for reference

2. **Deploy:**
   - Extract ZIP to `C:\ComplianceTool\`
   - Verify `configs/reports/` contains JSON files
   - Run `ComplianceToolkit.exe -list` to test

---

### Method 3: Installer Script (PowerShell)

Create `install.ps1`:

```powershell
# Compliance Toolkit Installer
$InstallPath = "C:\ComplianceTool"

Write-Host "Installing Compliance Toolkit to $InstallPath..." -ForegroundColor Cyan

# Create directories
New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
New-Item -ItemType Directory -Path "$InstallPath\configs\reports" -Force | Out-Null

# Copy files
Copy-Item "ComplianceToolkit.exe" -Destination $InstallPath -Force
Copy-Item "configs\reports\*.json" -Destination "$InstallPath\configs\reports\" -Force

# Optional: Copy examples
if (Test-Path "examples") {
    Copy-Item "examples" -Destination "$InstallPath\examples\" -Recurse -Force
}

# Verify
Write-Host "`nVerifying installation..." -ForegroundColor Yellow
& "$InstallPath\ComplianceToolkit.exe" -list

Write-Host "`nInstallation complete!" -ForegroundColor Green
Write-Host "Location: $InstallPath" -ForegroundColor Yellow
Write-Host "`nTo run: cd $InstallPath && ComplianceToolkit.exe" -ForegroundColor Cyan
```

**Run installer:**
```powershell
.\install.ps1
```

---

## Path Resolution

The toolkit automatically searches for `configs/reports/` in these locations (in order):

1. **Current working directory** - `.\configs\reports\`
2. **Next to executable** - `C:\ComplianceTool\configs\reports\`
3. **One level up from executable** - `C:\configs\reports\` (if exe is in subfolder)

**This means:**
- ✅ You can run from any directory: `C:\ComplianceTool\ComplianceToolkit.exe -list`
- ✅ You can add to PATH and run globally: `ComplianceToolkit.exe -list`
- ✅ Works with Task Scheduler from any location

---

## Output Directory Locations

### Default Locations

By default, output is created relative to the executable:

```
C:\ComplianceTool\
├── ComplianceToolkit.exe
├── configs\
├── output\
│   ├── reports\      # HTML reports
│   ├── evidence\     # JSON evidence logs
│   └── logs\         # Application logs
```

### Custom Output Directories

Override defaults with CLI flags:

```cmd
ComplianceToolkit.exe -report=all ^
    -output=C:\Compliance\Reports ^
    -evidence=C:\Compliance\Evidence ^
    -logs=C:\Compliance\Logs
```

**For scheduled tasks:**
```cmd
ComplianceToolkit.exe -report=all -quiet ^
    -output=\\FileServer\Compliance\Reports ^
    -evidence=\\FileServer\Compliance\Evidence
```

---

## Installation Verification

### Test Basic Functionality

```cmd
# 1. List reports
ComplianceToolkit.exe -list

# 2. Run a simple report
ComplianceToolkit.exe -report=system_info.json

# 3. Check output
dir output\reports\*.html
dir output\evidence\*.json

# 4. View report in browser
start output\reports\System_Information_*.html
```

### Expected Output

```
Available Reports:
==================
  - NIST_800_171_compliance.json
    Title:    NIST 800-171 Security Compliance Report
    Category: Security & Compliance
    Version:  2.0.0

  - fips_140_2_compliance.json
    ...
```

If you see this, installation is successful! ✅

---

## Network Installation (Enterprise)

### Deploy to Multiple Machines

**1. Create network share:**
```cmd
\\FileServer\Software\ComplianceTool\
├── ComplianceToolkit.exe
└── configs\reports\*.json
```

**2. Local installation script (GPO/SCCM):**
```cmd
@echo off
REM Deploy from network share
xcopy \\FileServer\Software\ComplianceTool C:\ComplianceTool\ /E /I /Y

REM Create scheduled task
schtasks /create /tn "Daily Compliance Scan" ^
    /tr "C:\ComplianceTool\ComplianceToolkit.exe -report=all -quiet" ^
    /sc daily /st 02:00 /ru SYSTEM
```

---

## Adding to System PATH

To run `ComplianceToolkit.exe` from anywhere:

### Windows

**Method 1: GUI**
1. Right-click **This PC** → **Properties**
2. Click **Advanced system settings**
3. Click **Environment Variables**
4. Under **System variables**, find **Path** and click **Edit**
5. Click **New** and add: `C:\ComplianceTool`
6. Click **OK** on all dialogs

**Method 2: PowerShell (Admin)**
```powershell
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
[Environment]::SetEnvironmentVariable("Path", "$CurrentPath;C:\ComplianceTool", "Machine")
```

**Test:**
```cmd
# Open new command prompt
ComplianceToolkit.exe -list
```

---

## Permissions

### Required Permissions

- **Read:** Registry (HKEY_LOCAL_MACHINE, HKEY_CURRENT_USER)
- **Write:** Output directories
- **Execute:** ComplianceToolkit.exe

### Run as Administrator

Some compliance checks require admin privileges:

```cmd
# PowerShell (as Admin)
Start-Process "C:\ComplianceTool\ComplianceToolkit.exe" -Verb RunAs

# Or right-click exe → Run as administrator
```

### Service Account (Scheduled Tasks)

Create dedicated service account:
1. Create account: `ComplianceScanner`
2. Grant permissions:
   - Read registry access
   - Write to output directories
3. Run scheduled task as this account

---

## Upgrading

### Update Existing Installation

**1. Backup current version:**
```cmd
copy C:\ComplianceTool\ComplianceToolkit.exe C:\ComplianceTool\ComplianceToolkit_backup.exe
```

**2. Replace executable:**
```cmd
copy ComplianceToolkit.exe C:\ComplianceTool\ComplianceToolkit.exe /Y
```

**3. Update report configs (if changed):**
```cmd
xcopy configs\reports\*.json C:\ComplianceTool\configs\reports\ /Y
```

**4. Test:**
```cmd
C:\ComplianceTool\ComplianceToolkit.exe -list
```

---

## Uninstallation

### Complete Removal

```cmd
# 1. Remove scheduled tasks (if any)
schtasks /delete /tn "Daily Compliance Scan" /f

# 2. Remove from PATH (if added)
# See "Adding to System PATH" section

# 3. Delete installation directory
rmdir /S /Q C:\ComplianceTool

# 4. Optional: Delete output archives
rmdir /S /Q C:\Compliance\Archive
```

---

## Troubleshooting Installation

### Issue: "configs/reports not found"

**Cause:** Report configurations missing or in wrong location

**Solution:**
```cmd
# Verify configs directory exists
dir C:\ComplianceTool\configs\reports\*.json

# If missing, copy from distribution
xcopy configs\reports C:\ComplianceTool\configs\reports\ /E /I
```

### Issue: "Access denied" when creating output

**Cause:** Insufficient permissions for output directory

**Solution:**
```cmd
# Run as administrator OR
# Use custom output directory with write permissions
ComplianceToolkit.exe -report=all -output=C:\Users\YourName\ComplianceReports
```

### Issue: Executable won't run

**Cause:** Windows SmartScreen or antivirus blocking

**Solution:**
```cmd
# Check file properties
# Right-click exe → Properties → Unblock

# Or add to antivirus exclusions
# Windows Defender → Virus & threat protection → Exclusions → Add exclusion
```

### Issue: Reports generate but don't open

**Cause:** No default browser or file association

**Solution:**
```cmd
# Manually open report
explorer output\reports

# Or specify browser
"C:\Program Files\Mozilla Firefox\firefox.exe" output\reports\latest_report.html
```

---

## Deployment Checklist

- [ ] Copy `ComplianceToolkit.exe` to installation directory
- [ ] Copy `configs/reports/` directory with all JSON files
- [ ] Verify installation: `ComplianceToolkit.exe -list`
- [ ] Run test report: `ComplianceToolkit.exe -report=system_info.json`
- [ ] Check output: `dir output\reports\*.html`
- [ ] Set up scheduled task (if needed)
- [ ] Add to PATH (optional)
- [ ] Configure permissions (if running as service account)
- [ ] Document installation location for team

---

## Directory Structure Reference

### Minimal Installation
```
C:\ComplianceTool\
├── ComplianceToolkit.exe          # Required
└── configs\
    └── reports\                   # Required
        └── *.json                 # At least one report config
```

### Full Installation
```
C:\ComplianceTool\
├── ComplianceToolkit.exe          # Main executable
├── configs\
│   └── reports\                   # Report configurations
│       ├── NIST_800_171_compliance.json
│       ├── fips_140_2_compliance.json
│       └── ...
├── examples\                      # Automation scripts (optional)
│   ├── scheduled_compliance_scan.bat
│   └── scheduled_compliance_scan.ps1
├── Documentation\                 # Reference docs (optional)
│   ├── CLI_USAGE.md
│   ├── INSTALLATION.md (this file)
│   └── ...
└── output\                        # Created automatically
    ├── reports\                   # Generated HTML reports
    ├── evidence\                  # JSON evidence logs
    └── logs\                      # Application logs
```

---

## Next Steps

1. ✅ **Verify Installation** - Run `ComplianceToolkit.exe -list`
2. ✅ **Generate Test Report** - Run `ComplianceToolkit.exe -report=system_info.json`
3. ✅ **View Report** - Open `output/reports/*.html` in browser
4. ✅ **Set Up Automation** - See [CLI_USAGE.md](Documentation/CLI_USAGE.md)
5. ✅ **Schedule Scans** - See [examples/README.md](examples/README.md)

---

**For more information:**
- [CLI_USAGE.md](Documentation/CLI_USAGE.md) - Command-line usage
- [CLI_QUICKSTART.md](CLI_QUICKSTART.md) - 5-minute quick start
- [PROJECT_STATUS.md](PROJECT_STATUS.md) - Project overview

---

*Installation Guide v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-04*
