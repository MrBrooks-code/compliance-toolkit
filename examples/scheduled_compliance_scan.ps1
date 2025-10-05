# ==============================================================================
# Scheduled Compliance Scan - PowerShell Script
# ==============================================================================
#
# Purpose: Run compliance reports and archive results
# Usage:   Add to Windows Task Scheduler or run manually
#
# Task Scheduler Settings:
#   - Program/script: powershell.exe
#   - Arguments:      -ExecutionPolicy Bypass -File "C:\Path\To\scheduled_compliance_scan.ps1"
#   - Start in:       C:\Path\To\
#
# ==============================================================================

[CmdletBinding()]
param(
    [string]$ToolkitPath = ".\ComplianceToolkit.exe",
    [string]$ReportName = "all",
    [switch]$ArchiveReports,
    [string]$ArchiveBase = ".\archive",
    [string]$EmailTo = "",
    [string]$EmailFrom = "compliance@company.com",
    [string]$SmtpServer = ""
)

$ErrorActionPreference = "Stop"

# Configuration
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ToolkitExe = Join-Path $ScriptDir "..\ComplianceToolkit.exe"
$ReportsDir = Join-Path $ScriptDir "..\output\reports"
$EvidenceDir = Join-Path $ScriptDir "..\output\evidence"
$LogsDir = Join-Path $ScriptDir "..\output\logs"

# Timestamps
$Timestamp = Get-Date -Format "yyyy-MM-dd_HHmmss"
$DateStamp = Get-Date -Format "yyyy-MM-dd"

# Logging function
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $LogMessage = "[{0}] [{1}] {2}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Level, $Message
    Write-Host $LogMessage
    $LogFile = Join-Path $LogsDir "scheduled_scan_$DateStamp.log"
    Add-Content -Path $LogFile -Value $LogMessage
}

try {
    Write-Log "=================================================="
    Write-Log "Starting Scheduled Compliance Scan"
    Write-Log "Report: $ReportName"
    Write-Log "Toolkit: $ToolkitExe"
    Write-Log "=================================================="

    # Verify toolkit exists
    if (-not (Test-Path $ToolkitExe)) {
        throw "ComplianceToolkit.exe not found at: $ToolkitExe"
    }

    # Run compliance scan
    Write-Log "Executing compliance scan..."
    $Process = Start-Process -FilePath $ToolkitExe `
        -ArgumentList "-report=$ReportName -quiet" `
        -Wait `
        -PassThru `
        -NoNewWindow `
        -WorkingDirectory (Split-Path -Parent $ToolkitExe)

    $ExitCode = $Process.ExitCode

    if ($ExitCode -ne 0) {
        throw "Compliance scan failed with exit code: $ExitCode"
    }

    Write-Log "Compliance scan completed successfully!" "SUCCESS"

    # Count generated reports
    $HtmlReports = Get-ChildItem -Path $ReportsDir -Filter "*.html" -ErrorAction SilentlyContinue
    $EvidenceLogs = Get-ChildItem -Path $EvidenceDir -Filter "*.json" -ErrorAction SilentlyContinue

    Write-Log "Generated $($HtmlReports.Count) HTML reports"
    Write-Log "Generated $($EvidenceLogs.Count) evidence logs"

    # Archive reports if requested
    if ($ArchiveReports) {
        Write-Log "Archiving reports..."

        $ArchiveDir = Join-Path $ArchiveBase $DateStamp
        New-Item -ItemType Directory -Path $ArchiveDir -Force | Out-Null

        # Copy reports
        if ($HtmlReports) {
            Copy-Item -Path "$ReportsDir\*.html" -Destination $ArchiveDir -Force
            Write-Log "Archived HTML reports to: $ArchiveDir"
        }

        # Copy evidence
        if ($EvidenceLogs) {
            Copy-Item -Path "$EvidenceDir\*.json" -Destination $ArchiveDir -Force
            Write-Log "Archived evidence logs to: $ArchiveDir"
        }

        # Clean up old archives (older than 90 days)
        $OldArchives = Get-ChildItem -Path $ArchiveBase -Directory |
            Where-Object { $_.CreationTime -lt (Get-Date).AddDays(-90) }

        if ($OldArchives) {
            Write-Log "Removing $($OldArchives.Count) old archives (>90 days)"
            $OldArchives | Remove-Item -Recurse -Force
        }
    }

    # Send success email if configured
    if ($EmailTo -and $SmtpServer) {
        $Subject = "✅ Compliance Scan Successful - $DateStamp"
        $Body = @"
Compliance scan completed successfully!

Timestamp: $Timestamp
Reports Generated: $($HtmlReports.Count)
Evidence Logs: $($EvidenceLogs.Count)

Reports Directory: $ReportsDir
Evidence Directory: $EvidenceDir

Latest Reports:
$($HtmlReports | Select-Object -First 5 | ForEach-Object { "  - $($_.Name)" } | Out-String)
"@

        Send-MailMessage `
            -To $EmailTo `
            -From $EmailFrom `
            -Subject $Subject `
            -Body $Body `
            -SmtpServer $SmtpServer `
            -ErrorAction Stop

        Write-Log "Success notification sent to $EmailTo"
    }

    Write-Log "=================================================="
    Write-Log "Scheduled compliance scan completed successfully"
    Write-Log "=================================================="

    exit 0

} catch {
    $ErrorMessage = $_.Exception.Message
    Write-Log "ERROR: $ErrorMessage" "ERROR"
    Write-Log "Stack Trace: $($_.ScriptStackTrace)" "ERROR"

    # Send failure email if configured
    if ($EmailTo -and $SmtpServer) {
        $Subject = "❌ Compliance Scan Failed - $DateStamp"
        $Body = @"
Compliance scan FAILED!

Timestamp: $Timestamp
Error: $ErrorMessage

Please check the logs at: $LogsDir

Stack Trace:
$($_.ScriptStackTrace)
"@

        try {
            Send-MailMessage `
                -To $EmailTo `
                -From $EmailFrom `
                -Subject $Subject `
                -Body $Body `
                -SmtpServer $SmtpServer `
                -Priority High `
                -ErrorAction Stop

            Write-Log "Failure notification sent to $EmailTo" "ERROR"
        } catch {
            Write-Log "Failed to send email notification: $($_.Exception.Message)" "ERROR"
        }
    }

    Write-Log "=================================================="
    exit 1
}
