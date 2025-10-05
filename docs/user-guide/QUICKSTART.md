# Compliance Toolkit - Quick Start Guide

## 🚀 Getting Started

### Build the Application

```bash
# Build the interactive CLI
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Run the Application

```bash
# Launch the interactive menu
.\ComplianceToolkit.exe
```

## 📋 Main Menu

When you launch the application, you'll see:

```
╔══════════════════════════════════════════════════════════════════════╗
║                                                                      ║
║   ╔═╗╔═╗╔╦╗╔═╗╦  ╦╔═╗╔╗╔╔═╗╔═╗  ╔╦╗╔═╗╔═╗╦  ╦╔═╦╔╦╗                  ║
║   ║  ║ ║║║║╠═╝║  ║╠═╣║║║║  ║╣    ║ ║ ║║ ║║  ╠╩╗║ ║                   ║
║   ╚═╝╚═╝╩ ╩╩  ╩═╝╩╩ ╩╝╚╝╚═╝╚═╝   ╩ ╚═╝╚═╝╩═╝╩ ╩╩ ╩                   ║
║                                                                      ║
║                 Windows Registry Compliance Scanner                  ║
║                          Version 1.0.0                               ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝

┌──────────────────────────────────────────────────────────────────────┐
│                            MAIN MENU                                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│       [1]  Run Reports                                               │
│       [2]  View HTML Reports                                         │
│       [3]  View Log Files                                            │
│       [4]  Configuration                                             │
│       [5]  About                                                     │
│                                                                      │
│       [0]  Exit                                                      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## 🎯 How to Use

### 1. Run Reports

**Option 1** → Select option `[1]` from main menu

You'll see 6 report types:

- 💻 **[1] System Information** - OS version, computer name, etc.
- 🔒 **[2] Security Audit** - UAC, Firewall, Windows Defender
- 📦 **[3] Software Inventory** - Installed applications
- 🌐 **[4] Network Configuration** - DNS, proxy, network settings
- 👤 **[5] User Settings** - Desktop, themes, preferences
- ⚡ **[6] Performance Diagnostics** - Memory, boot settings

- 🚀 **[7] Run ALL Reports** - Execute all 6 reports at once

**Example:**
```
➤ Select report: 1

⏳ Running system_info.json...

  ✅  [os_product_name] Success
  ✅  [os_edition] Success
  ✅  [os_build_number] Success
  ⚠️  [registered_organization] Not found
  ...

  📊  Results: 8 successful, 2 errors
  📄  HTML Report: output/reports/System_Information_20250104_123045.html

✅ SUCCESS: Report completed successfully!
ℹ INFO: Report saved to: output/reports
```

### 2. View HTML Reports

**Option 2** → Opens generated HTML reports in your browser

- Lists all HTML reports in `output/reports/`
- Select a report number to open it
- Report opens automatically in your default browser
- Beautiful, interactive HTML with color-coded results

**HTML Report Features:**
- ✅ Green checkmarks for successful reads
- ❌ Red X for errors/not found
- 📊 Statistics dashboard
- 🎨 Professional gradient design
- 📱 Responsive layout
- 🖨️ Print-friendly

### 3. View Log Files

**Option 3** → View application logs

- Shows all log files in `output/logs/`
- JSON-formatted structured logs
- Includes timestamps, operation durations, errors

**Log Location:**
```
output/logs/toolkit_20250104_123045.log
```

### 4. Configuration

**Option 4** → View current settings

Shows:
- Output directory path
- Logs directory path
- Operation timeout (default: 10 seconds)
- Log level (INFO/DEBUG)

### 5. About

**Option 5** → Information about the toolkit

- Version information
- Features overview
- Security guarantees

## 📁 Output Structure

After running reports, you'll have:

```
lab3-registry-read/
├── ComplianceToolkit.exe        ← Your executable
├── output/
│   ├── reports/
│   │   ├── System_Information_20250104_123045.html
│   │   ├── Security_Audit_20250104_123050.html
│   │   ├── Software_Inventory_20250104_123055.html
│   │   └── ...
│   └── logs/
│       └── toolkit_20250104_123045.log
├── configs/
│   └── reports/
│       ├── system_info.json
│       ├── security_audit.json
│       └── ...
```

## 🎬 Typical Workflow

### First Time Use

1. **Build** the application:
   ```bash
   go build -o ComplianceToolkit.exe ./cmd/toolkit.go
   ```

2. **Run** the toolkit:
   ```bash
   .\ComplianceToolkit.exe
   ```

3. **Select [1]** - Run Reports

4. **Select [7]** - Run ALL Reports

5. **Wait** for all reports to complete (~30 seconds)

6. **Select [0]** - Back to Main Menu

7. **Select [2]** - View HTML Reports

8. **Select a report** to open in browser

9. **Review** the beautiful HTML report!

### Regular Use

1. Launch toolkit: `.\ComplianceToolkit.exe`
2. Select specific report you want (e.g., Security Audit)
3. View HTML report immediately
4. Check logs if needed
5. Exit when done

## ⚡ Quick Commands

If you prefer command-line over interactive menu:

```bash
# Build all tools
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
go build -o registryreader.exe ./cmd/main.go
go build -o report_runner.exe ./cmd/report_runner.go

# Run tests
go test ./pkg/... -v

# Run specific report (non-interactive)
go run ./cmd/report_runner.go -config configs/reports/system_info.json

# Generate JSON output
go run ./cmd/report_runner.go -json > report.json
```

## 💡 Tips & Tricks

### Tip 1: Run as Administrator
Some registry keys require admin access:
- Right-click Command Prompt → "Run as Administrator"
- Then run `.\ComplianceToolkit.exe`

### Tip 2: Schedule Reports
Create a batch file for scheduled scanning:

```batch
@echo off
cd D:\golang-labs\lab3-registry-read
ComplianceToolkit.exe
```

### Tip 3: Export Reports
HTML reports can be:
- Printed to PDF (Ctrl+P → Save as PDF)
- Emailed to compliance team
- Archived for audit trails

### Tip 4: Quick System Check
For fast system info:
1. Launch toolkit
2. Press `1` → `1` → `0` → `2` → `1`
3. System info opens in browser

## 🐛 Troubleshooting

### "Access Denied" Errors
**Solution:** Run as Administrator

### "Key Not Found" Errors
**Normal:** Not all registry keys exist on all systems
- Security audit may show missing keys (expected)
- Software inventory depends on installed software

### "Timeout" Errors
**Solution:** Increase timeout in configuration
- Default is 10 seconds
- Registry might be busy/slow system

### HTML Report Won't Open
**Solution:**
- Check `output/reports/` directory exists
- Manually open the HTML file
- Check browser isn't blocking local files

## 📊 Understanding Report Results

### ✅ Green Checkmark
- Registry key/value found
- Successfully read
- Value displayed

### ❌ Red X with "Not Found"
- Registry key doesn't exist (normal)
- Value doesn't exist in that key
- No action needed unless expected

### ❌ Red X with Error
- Permission denied (run as admin)
- Timeout (increase timeout)
- Malformed registry path (check config)

## 🎯 Next Steps

1. ✅ Run your first report
2. ✅ View the HTML output
3. ✅ Review the security audit
4. ✅ Check software inventory
5. ✅ Customize JSON configs for your needs

## 📚 Additional Resources

- `JSON_CONFIG_GUIDE.md` - Create custom reports
- `TESTING_GUIDE.md` - Comprehensive testing
- `IMPROVEMENTS.md` - Technical improvements
- `README.md` - Full documentation

## 🎉 You're Ready!

Launch the toolkit and start scanning:

```bash
.\ComplianceToolkit.exe
```

Enjoy your professional compliance toolkit! 🚀
