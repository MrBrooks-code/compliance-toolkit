# Available Reports Reference

**Version:** 1.1.0
**Last Updated:** 2025-01-05

Complete reference of all compliance and diagnostic reports available in the Compliance Toolkit.

---

## Table of Contents

1. [Overview](#overview)
2. [Security & Compliance Reports](#security--compliance-reports)
3. [System Inventory Reports](#system-inventory-reports)
4. [Network & Configuration Reports](#network--configuration-reports)
5. [Performance & Diagnostics Reports](#performance--diagnostics-reports)
6. [How to Run Reports](#how-to-run-reports)

---

## Overview

The Compliance Toolkit includes **7 comprehensive reports** covering:

- ✅ Security compliance (NIST 800-171, FIPS 140-2)
- ✅ System inventory and software tracking
- ✅ Network configuration analysis
- ✅ User preferences and settings
- ✅ Performance diagnostics

### Report Categories

| Category | Reports | Purpose |
|----------|---------|---------|
| **Security & Compliance** | 2 | NIST 800-171, FIPS 140-2 validation |
| **System Inventory** | 2 | OS info, software inventory |
| **Network Security** | 1 | Network configuration, DNS, proxy |
| **User Configuration** | 1 | User preferences, desktop settings |
| **Performance** | 1 | Performance diagnostics, optimization |

---

## Security & Compliance Reports

### 1. NIST 800-171 Security Compliance Report

**File:** `NIST_800_171_compliance.json`
**Version:** 2.0.0
**Category:** Security & Compliance

#### Description
Validates NIST 800-171 Rev 2 security controls for Controlled Unclassified Information (CUI) protection.

#### Compliance Checks (13 total)

| Check | Description | Registry Path |
|-------|-------------|---------------|
| **uac_enabled** | User Account Control Status | `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System` |
| **uac_consent_prompt_admin** | UAC Consent Prompt for Admins | `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System` |
| **windows_defender_enabled** | Real-Time Protection Status | `HKLM\SOFTWARE\Microsoft\Windows Defender\Real-Time Protection` |
| **firewall_domain_profile** | Firewall Domain Profile | `HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\DomainProfile` |
| **firewall_standard_profile** | Firewall Standard Profile | `HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile` |
| **firewall_public_profile** | Firewall Public Profile | `HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\PublicProfile` |
| **auto_update_enabled** | Windows Auto Update | `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU` |
| **smb_v1_enabled** | SMBv1 Protocol (should be disabled) | `HKLM\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters` |
| **lsa_protection** | LSA Protection/Credential Guard | `HKLM\SYSTEM\CurrentControlSet\Control\Lsa` |
| **remote_desktop_enabled** | Remote Desktop Status | `HKLM\SYSTEM\CurrentControlSet\Control\Terminal Server` |
| **nla_required** | Network Level Authentication for RDP | `HKLM\SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp` |
| **secure_boot_enabled** | Secure Boot Configuration | `HKLM\SYSTEM\CurrentControlSet\Control\SecureBoot\State` |
| **bitlocker_status** | BitLocker Encryption Policies | `HKLM\SOFTWARE\Policies\Microsoft\FVE` |

#### Use Cases
- Government contractor compliance (DFARS 252.204-7012)
- CUI protection validation
- Security baseline verification
- Pre-audit preparation

#### Run Command
```bash
ComplianceToolkit.exe -report=NIST_800_171_compliance.json -quiet
```

---

### 2. FIPS 140-2 Compliance Report

**File:** `fips_140_2_compliance.json`
**Version:** 1.0.0
**Category:** Security & Compliance

#### Description
Federal Information Processing Standard (FIPS) 140-2 cryptographic module validation for government systems.

#### Compliance Checks (35 total)

**FIPS Algorithm Policy (2 checks)**
- FIPS Algorithm Policy Enabled
- FIPS Algorithm MDU Enabled

**TLS Configuration (6 checks)**
- TLS 1.2 Client/Server Enabled
- TLS 1.2 Not Disabled by Default (Client/Server)
- TLS 1.0 Disabled (Client/Server)
- TLS 1.1 Disabled (Client/Server)

**Legacy Protocol Disabled (4 checks)**
- SSL 2.0 Disabled (Client/Server)
- SSL 3.0 Disabled (Client/Server)

**Cipher Suites (9 checks)**
- Weak Ciphers Disabled: Triple DES 168, RC4 (128/64/56/40)
- Strong Ciphers Enabled: AES-128, AES-256 (FIPS Approved)

**Hash Algorithms (5 checks)**
- MD5 Disabled
- SHA, SHA-256, SHA-384, SHA-512 Enabled (FIPS Approved)

**Key Exchange (3 checks)**
- PKCS, ECDH, Diffie-Hellman Enabled

**Additional Configurations (6 checks)**
- Cryptography Next Generation (CNG)
- EFS Algorithm Configuration
- BitLocker Encryption Method
- RDP Encryption Level
- RDP Security Layer

#### Use Cases
- Federal government compliance
- DOD systems validation
- Cryptographic standards enforcement
- Security certification requirements

#### Run Command
```bash
ComplianceToolkit.exe -report=fips_140_2_compliance.json -quiet
```

---

## System Inventory Reports

### 3. System Information Report

**File:** `system_info.json`
**Version:** 1.0.0
**Category:** System Inventory

#### Description
Comprehensive Windows system configuration and version information for IT asset management.

#### Information Gathered (11 items)

| Item | Description | Registry Path |
|------|-------------|---------------|
| **os_product_name** | Windows Product Name | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **os_edition** | Windows Edition ID | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **os_build_number** | Windows Build Number | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **os_build_lab** | Windows Build Lab String | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **os_install_date** | Windows Installation Date | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **computer_name** | Computer Name | `HKLM\SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName` |
| **system_root** | Windows System Root Directory | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **registered_owner** | Registered Owner | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **registered_organization** | Registered Organization | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |
| **processor_architecture** | Processor Architecture | `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment` |
| **all_version_info** | All Windows Version Values | `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion` |

#### Use Cases
- IT asset inventory
- License compliance tracking
- System documentation
- Hardware/software audits

#### Run Command
```bash
ComplianceToolkit.exe -report=system_info.json -quiet
```

---

### 4. Software Inventory Report

**File:** `software_inventory.json`
**Version:** 1.1.0
**Category:** Software Asset Management

#### Description
Installed software applications, versions, and development tools inventory for license compliance.

#### Software Tracked (14 categories)

**Development Frameworks**
- .NET Framework Versions
- .NET Core/5+ Runtime Versions
- PowerShell Version
- Visual Studio Installations

**Office & Productivity**
- Microsoft Office Version
- Microsoft Office Product Name

**Web Browsers**
- Google Chrome Version
- Microsoft Edge Version

**Development Tools**
- Java Runtime Version
- Python Installation Paths
- Node.js Version
- Git Installation Path
- Docker Desktop Version

**System Features**
- Windows Subsystem for Linux (WSL) Version

#### Use Cases
- Software license compliance
- Development environment audits
- Version tracking
- Security vulnerability assessment

#### Run Command
```bash
ComplianceToolkit.exe -report=software_inventory.json -quiet
```

---

## Network & Configuration Reports

### 5. Network Configuration Report

**File:** `network_config.json`
**Version:** 1.0.0
**Category:** Network Security

#### Description
Network settings, DNS, proxy, and connectivity configuration for security analysis.

#### Configuration Checks (13 items)

| Check | Description | Registry Path |
|-------|-------------|---------------|
| **computer_hostname** | Computer Hostname | `HKLM\SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName` |
| **dns_hostname** | DNS Hostname | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters` |
| **dns_domain** | DNS Domain Name | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters` |
| **dns_servers** | DNS Server List | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters` |
| **dhcp_enabled** | DHCP Enabled Status | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters` |
| **tcp_parameters** | All TCP/IP Parameters | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters` |
| **proxy_enabled** | Internet Proxy Enabled | `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings` |
| **proxy_server** | Proxy Server Address | `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings` |
| **proxy_override** | Proxy Override List | `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings` |
| **network_location_awareness** | Network Location Settings | `HKLM\SOFTWARE\Policies\Microsoft\Windows\NetworkConnectivityStatusIndicator` |
| **ipv6_enabled** | IPv6 Configuration | `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters` |
| **wifi_auto_connect** | WiFi Auto-Connect Setting | `HKLM\SOFTWARE\Microsoft\WcmSvc\wifinetworkmanager\config` |
| **network_discovery** | Network Discovery Status | `HKLM\SYSTEM\CurrentControlSet\Control\Network\NewNetworkWindowOff` |

#### Use Cases
- Network security audits
- Connectivity troubleshooting
- Proxy configuration verification
- DNS security validation

#### Run Command
```bash
ComplianceToolkit.exe -report=network_config.json -quiet
```

---

### 6. User Settings Report

**File:** `user_settings.json`
**Version:** 1.0.0
**Category:** User Configuration

#### Description
User preferences, desktop configuration, and personalization settings.

#### Settings Tracked (18 items)

**Desktop & Themes**
- Desktop Wallpaper Path
- Current Windows Theme
- Dark Mode for Apps
- Dark Mode for System

**Screen Saver**
- Screen Saver Active Status
- Screen Saver Timeout
- Screen Saver Password Protection

**Windows Explorer**
- All Explorer Settings
- Show File Extensions
- Show Hidden Files

**User Preferences**
- Taskbar Position
- Default Web Browser
- Mouse Pointer Speed
- Keyboard Repeat Delay

**Startup & Environment**
- User Startup Programs
- User Environment Variables
- User Time Zone

**Accessibility**
- Accessibility/Ease of Access Settings

#### Use Cases
- User experience standardization
- Desktop policy compliance
- Accessibility requirement verification
- Troubleshooting user issues

#### Run Command
```bash
ComplianceToolkit.exe -report=user_settings.json -quiet
```

---

## Performance & Diagnostics Reports

### 7. Performance Diagnostics Report

**File:** `performance_diagnostics.json`
**Version:** 1.0.0
**Category:** Performance & Optimization

#### Description
System performance settings, memory management, and optimization configuration.

#### Diagnostics Covered (16 items)

**Memory Management**
- Virtual Memory (Page File) Configuration
- Clear Page File at Shutdown
- Large System Cache for Network Services

**Boot & Startup**
- System Startup Delay
- Boot Optimization Settings
- Programs that Run at Boot
- Automatic Shell Restart on Crash

**Performance Features**
- Prefetch and Superfetch Status
- Processor Scheduling Priority
- Visual Effects Performance Settings
- Menu Show Delay

**Error Handling**
- System Crash Dump Configuration
- Memory Dump File Location
- Windows Error Reporting Status

**Disk Performance**
- Disk I/O Timeout Value

#### Use Cases
- Performance troubleshooting
- System optimization
- Resource usage analysis
- Boot time diagnostics

#### Run Command
```bash
ComplianceToolkit.exe -report=performance_diagnostics.json -quiet
```

---

## How to Run Reports

### Interactive Mode

```bash
# Launch toolkit
ComplianceToolkit.exe

# Select [1] Run Reports
# Choose report number
# View generated HTML report
```

### Command Line Mode

**Single Report:**
```bash
ComplianceToolkit.exe -report=NIST_800_171_compliance.json
```

**All Reports:**
```bash
ComplianceToolkit.exe -report=all -quiet
```

**With Custom Output:**
```bash
ComplianceToolkit.exe -report=all -output=C:\Compliance\Reports -quiet
```

**List Available Reports:**
```bash
ComplianceToolkit.exe -list
```

### Scheduled Execution

**Windows Task Scheduler:**
```
Program: C:\ComplianceTool\ComplianceToolkit.exe
Arguments: -report=all -quiet
Schedule: Daily at 2:00 AM
```

**PowerShell:**
```powershell
& "C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet
if ($LASTEXITCODE -ne 0) {
    Write-Error "Compliance scan failed!"
}
```

**Batch Script:**
```batch
@echo off
"C:\ComplianceTool\ComplianceToolkit.exe" -report=all -quiet
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Scan failed!
)
```

---

## Report Output

### Generated Files

For each report, the toolkit generates:

**HTML Report:**
```
output/reports/NIST_800-171_Security_Compliance_Report_YYYYMMDD_HHMMSS.html
```

**JSON Evidence Log:**
```
output/evidence/NIST_800_171_compliance_evidence_YYYYMMDD_HHMMSS.json
```

**Application Log:**
```
output/logs/toolkit_YYYYMMDD.log
```

### Report Features

- ✅ Interactive charts (Chart.js)
- ✅ Dark mode support
- ✅ Search and filter functionality
- ✅ Collapsible detail sections
- ✅ Sortable result tables
- ✅ Professional styling (Bulma CSS)
- ✅ Print-friendly layout

---

## Next Steps

- ✅ **Run Your First Report**: See [Quick Start Guide](../user-guide/QUICKSTART.md)
- ✅ **Automate Reports**: See [CLI Usage Guide](../user-guide/CLI_USAGE.md)
- ✅ **Evidence Logging**: See [Evidence Reference](EVIDENCE.md)
- ✅ **Create Custom Reports**: See [Adding Reports](../developer-guide/ADDING_REPORTS.md)

---

*Reports Reference v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
