@echo off
REM ==============================================================================
REM Scheduled Compliance Scan - Example Batch Script
REM ==============================================================================
REM
REM Purpose: Run all compliance reports on a schedule
REM Usage:   Add to Windows Task Scheduler
REM
REM Task Scheduler Settings:
REM   - Program/script: C:\Path\To\ComplianceToolkit.exe
REM   - Arguments:      -report=all -quiet
REM   - Start in:       C:\Path\To\
REM
REM ==============================================================================

SETLOCAL EnableDelayedExpansion

REM Configuration
SET TOOLKIT_DIR=%~dp0..
SET TOOLKIT_EXE=%TOOLKIT_DIR%\ComplianceToolkit.exe
SET LOG_FILE=%TOOLKIT_DIR%\output\logs\scheduled_scan.log

REM Create timestamp
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /format:list') do set datetime=%%I
set TIMESTAMP=%datetime:~0,8%_%datetime:~8,6%

echo ================================================== >> "%LOG_FILE%"
echo Scheduled Compliance Scan - %TIMESTAMP% >> "%LOG_FILE%"
echo ================================================== >> "%LOG_FILE%"

REM Change to toolkit directory
cd /d "%TOOLKIT_DIR%"

REM Run compliance scan (all reports in quiet mode)
echo Running compliance scan... >> "%LOG_FILE%"
"%TOOLKIT_EXE%" -report=all -quiet

REM Check exit code
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Compliance scan failed with exit code %ERRORLEVEL% >> "%LOG_FILE%"
    echo Timestamp: %TIMESTAMP% >> "%LOG_FILE%"
    echo ================================================== >> "%LOG_FILE%"

    REM Optional: Send email notification
    REM powershell -Command "Send-MailMessage -To 'admin@company.com' -From 'compliance@company.com' -Subject 'Compliance Scan Failed' -Body 'Check logs at %TOOLKIT_DIR%\output\logs' -SmtpServer 'smtp.company.com'"

    exit /b 1
)

echo Compliance scan completed successfully! >> "%LOG_FILE%"
echo Reports saved to: %TOOLKIT_DIR%\output\reports >> "%LOG_FILE%"
echo Evidence saved to: %TOOLKIT_DIR%\output\evidence >> "%LOG_FILE%"
echo ================================================== >> "%LOG_FILE%"
echo. >> "%LOG_FILE%"

exit /b 0
