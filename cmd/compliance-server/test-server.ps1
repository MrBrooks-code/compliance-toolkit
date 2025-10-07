# Test Server Script
# Tests the compliance server endpoints

Write-Host "=== Compliance Server Test ===" -ForegroundColor Cyan
Write-Host ""

$serverUrl = "https://localhost:8443"
$apiKey = "test-api-key-12345"

# Ignore self-signed certificate warnings for testing
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}
Add-Type @"
    using System.Net;
    using System.Security.Cryptography.X509Certificates;
    public class TrustAllCertsPolicy : ICertificatePolicy {
        public bool CheckValidationResult(
            ServicePoint svcPoint, X509Certificate certificate,
            WebRequest request, int certificateProblem) {
            return true;
        }
    }
"@
[System.Net.ServicePointManager]::CertificatePolicy = New-Object TrustAllCertsPolicy

# Test 1: Health Check
Write-Host "Test 1: Health Check" -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$serverUrl/api/v1/health" -Method GET
    Write-Host "✅ Health check passed" -ForegroundColor Green
    Write-Host "   Status: $($response.status)" -ForegroundColor Gray
    Write-Host "   Version: $($response.version)" -ForegroundColor Gray
} catch {
    Write-Host "❌ Health check failed: $_" -ForegroundColor Red
}

Write-Host ""

# Test 2: Submit Compliance Report
Write-Host "Test 2: Submit Compliance Report" -ForegroundColor Yellow

$submission = @{
    submission_id = "test-submission-001"
    client_id = "TEST-CLIENT-001"
    hostname = "TEST-MACHINE"
    timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    report_type = "NIST 800-171 Security Compliance Report"
    report_version = "2.0.0"
    compliance = @{
        overall_status = "compliant"
        total_checks = 10
        passed_checks = 8
        failed_checks = 2
        warning_checks = 0
        error_checks = 0
        queries = @(
            @{
                name = "test_check"
                description = "Test compliance check"
                status = "pass"
                expected = "1"
                actual = "1"
            }
        )
    }
    evidence = @(
        @{
            query_name = "test_check"
            timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
            action = "registry_read"
            result = "success"
            details = @{
                path = "HKLM\Test"
            }
        }
    )
    system_info = @{
        os_version = "Windows 11 Pro"
        build_number = "22631"
        architecture = "amd64"
        ip_address = "192.168.1.100"
        mac_address = "00:11:22:33:44:55"
    }
}

try {
    $headers = @{
        "Authorization" = "Bearer $apiKey"
        "Content-Type" = "application/json"
    }

    $jsonBody = $submission | ConvertTo-Json -Depth 10
    $response = Invoke-RestMethod -Uri "$serverUrl/api/v1/compliance/submit" -Method POST -Headers $headers -Body $jsonBody

    Write-Host "✅ Submission accepted" -ForegroundColor Green
    Write-Host "   Submission ID: $($response.submission_id)" -ForegroundColor Gray
    Write-Host "   Status: $($response.status)" -ForegroundColor Gray
    Write-Host "   Message: $($response.message)" -ForegroundColor Gray
} catch {
    Write-Host "❌ Submission failed: $_" -ForegroundColor Red
    Write-Host "   $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
}

Write-Host ""

# Test 3: Get Submission Status
Write-Host "Test 3: Get Submission Status" -ForegroundColor Yellow
try {
    $headers = @{
        "Authorization" = "Bearer $apiKey"
    }

    $response = Invoke-RestMethod -Uri "$serverUrl/api/v1/compliance/status/test-submission-001" -Method GET -Headers $headers

    Write-Host "✅ Status retrieved" -ForegroundColor Green
    Write-Host "   Client ID: $($response.client_id)" -ForegroundColor Gray
    Write-Host "   Hostname: $($response.hostname)" -ForegroundColor Gray
    Write-Host "   Overall Status: $($response.overall_status)" -ForegroundColor Gray
    Write-Host "   Passed: $($response.passed_checks), Failed: $($response.failed_checks)" -ForegroundColor Gray
} catch {
    Write-Host "❌ Status retrieval failed: $_" -ForegroundColor Red
}

Write-Host ""

# Test 4: List Clients
Write-Host "Test 4: List Clients" -ForegroundColor Yellow
try {
    $headers = @{
        "Authorization" = "Bearer $apiKey"
    }

    $response = Invoke-RestMethod -Uri "$serverUrl/api/v1/clients" -Method GET -Headers $headers

    Write-Host "✅ Clients listed" -ForegroundColor Green
    Write-Host "   Total clients: $($response.Count)" -ForegroundColor Gray
    foreach ($client in $response) {
        Write-Host "   - $($client.hostname) ($($client.client_id))" -ForegroundColor Gray
    }
} catch {
    Write-Host "❌ Client list failed: $_" -ForegroundColor Red
}

Write-Host ""

# Test 5: Dashboard Summary
Write-Host "Test 5: Dashboard Summary" -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$serverUrl/api/v1/dashboard/summary" -Method GET

    Write-Host "✅ Dashboard summary retrieved" -ForegroundColor Green
    Write-Host "   Total Clients: $($response.total_clients)" -ForegroundColor Gray
    Write-Host "   Active Clients: $($response.active_clients)" -ForegroundColor Gray
    Write-Host "   Compliant Clients: $($response.compliant_clients)" -ForegroundColor Gray
    Write-Host "   Recent Submissions: $($response.recent_submissions.Count)" -ForegroundColor Gray
} catch {
    Write-Host "❌ Dashboard summary failed: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Tests Complete ===" -ForegroundColor Cyan
