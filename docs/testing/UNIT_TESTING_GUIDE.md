# Unit Testing Guide - Compliance Toolkit

**Date:** October 6, 2025
**Version:** 1.1.0

## Overview

This guide provides step-by-step instructions for testing the Compliance Toolkit server/client architecture, including the web dashboard and client detail pages.

---

## Prerequisites

### Required Files
- ✅ `compliance-server.exe` - Server executable
- ✅ `compliance-client.exe` - Client executable
- ✅ `server.yaml` - Server configuration
- ✅ `client.yaml` - Client configuration
- ✅ `dashboard.html` - Dashboard page
- ✅ `settings.html` - Settings page
- ✅ `client-detail.html` - Client detail page

### File Locations
```
ComplianceToolkit/
├── compliance-client.exe          # Root directory
├── client.yaml                     # Root directory
├── cmd/
│   └── compliance-server/
│       ├── compliance-server.exe
│       ├── server.yaml
│       ├── dashboard.html
│       ├── settings.html
│       └── client-detail.html
└── configs/reports/
    └── NIST_800_171_compliance.json
```

---

## Part 1: Server Setup

### 1.1 Configure Server

**File:** `cmd/compliance-server/server.yaml`

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8443
  tls:
    enabled: false  # Set to false for testing (no SSL certs needed)

# Database
database:
  type: "sqlite"
  path: "data/compliance.db"

# Authentication
auth:
  enabled: true
  require_key: true
  api_keys:
    - "test-api-key-12345"
    - "demo-key-67890"

# Dashboard
dashboard:
  enabled: true
  path: "/dashboard"

# Logging
logging:
  level: "info"
  format: "text"
  output_path: "stdout"
```

**Key Settings:**
- `tls.enabled: false` - Disables HTTPS (no certificate required)
- `port: 8443` - Server will run on port 8443
- Two API keys configured for testing

---

### 1.2 Start Server

**Command:**
```bash
cd cmd/compliance-server
./compliance-server.exe --config server.yaml
```

**Expected Output:**
```
time=2025-10-06T21:44:18.185-05:00 level=INFO msg="Compliance Server starting" version=1.0.0 port=8443 tls_enabled=false
time=2025-10-06T21:44:18.186-05:00 level=INFO msg="Database initialized" path=data/compliance.db
time=2025-10-06T21:44:18.186-05:00 level=INFO msg="Server started successfully"
time=2025-10-06T21:44:18.187-05:00 level=INFO msg="Starting HTTP server" addr=0.0.0.0:8443
```

**Verify Server is Running:**
```bash
# From another terminal
curl http://localhost:8443/api/v1/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

---

### 1.3 Troubleshooting Server Issues

#### Issue: Missing HTML files
**Error:**
```
level=ERROR msg="Failed to read dashboard.html" error="open dashboard.html: The system cannot find the file specified."
```

**Solution:**
```bash
# Must run server from its own directory
cd cmd/compliance-server
./compliance-server.exe --config server.yaml
```

#### Issue: Missing SSL certificates
**Error:**
```
level=ERROR msg="Server error" error="open certs/server.crt: The system cannot find the path specified."
```

**Solution:**
Edit `server.yaml` and set `tls.enabled: false`

#### Issue: Port already in use
**Error:**
```
bind: address already in use
```

**Solution:**
```bash
# Kill existing process on port 8443
# Windows:
netstat -ano | findstr :8443
taskkill /PID <PID> /F

# Linux/Mac:
lsof -ti:8443 | xargs kill -9
```

---

## Part 2: Client Setup

### 2.1 Configure Client

**File:** `client.yaml` (in root directory)

```yaml
# Client identification
client:
  id: ""          # Auto-generated if empty
  hostname: ""    # Auto-detected if empty
  enabled: true

# Server configuration
server:
  url: "http://localhost:8443"      # ← Must match server port
  api_key: "demo-key-67890"         # ← Must match server config
  tls_verify: false                 # Set to false for HTTP
  timeout: 30s
  retry_on_startup: true

# Reports to run
reports:
  config_path: "configs/reports"
  output_path: "output/reports"
  save_local: true
  reports:
    - "NIST_800_171_compliance.json"

# Scheduling
schedule:
  enabled: false  # Disabled for manual testing

# Retry configuration
retry:
  max_attempts: 3
  initial_backoff: 30s
  max_backoff: 5m
  backoff_multiplier: 2.0
  retry_on_server_error: true

# Cache
cache:
  enabled: true
  path: "cache/submissions"
  max_size_mb: 100
  max_age: 168h
  auto_clean: true

# Logging
logging:
  level: "info"
  format: "text"
  output_path: "stdout"
```

**Critical Settings:**
- `server.url` - Must NOT be empty (otherwise runs in standalone mode)
- `server.api_key` - Must match one of the keys in `server.yaml`
- `tls_verify: false` - Required for HTTP connections

---

### 2.2 Run Client (Send Test Data)

**Command:**
```bash
# From root directory (ComplianceToolkit/)
./compliance-client.exe --config client.yaml --once
```

**Expected Output:**
```
time=2025-10-06T21:50:00.123-05:00 level=INFO msg="Compliance Client starting" version=1.0.0
time=2025-10-06T21:50:00.124-05:00 level=INFO msg="Generating client ID" client_id=client-Ultrawide
time=2025-10-06T21:50:00.125-05:00 level=INFO msg="Running report" report=NIST_800_171_compliance.json
time=2025-10-06T21:50:00.500-05:00 level=INFO msg="Report completed" passed=35 failed=5 total=40
time=2025-10-06T21:50:00.501-05:00 level=INFO msg="Submitting to server" url=http://localhost:8443
time=2025-10-06T21:50:00.650-05:00 level=INFO msg="Submission successful" submission_id=sub-xyz789
```

**Server-Side Logs (should see):**
```
time=2025-10-06T21:50:00.502-05:00 level=INFO msg="Received compliance submission" submission_id=sub-xyz789 client_id=client-Ultrawide hostname=ULTRAWIDE
time=2025-10-06T21:50:00.650-05:00 level=INFO msg="HTTP request" method=POST path=/api/v1/compliance/submit status=200 duration=148
```

---

### 2.3 Run Multiple Test Submissions

To generate multiple data points for testing:

```bash
# Run 5 times to create history
./compliance-client.exe --config client.yaml --once
./compliance-client.exe --config client.yaml --once
./compliance-client.exe --config client.yaml --once
./compliance-client.exe --config client.yaml --once
./compliance-client.exe --config client.yaml --once
```

Or use a loop:

**PowerShell:**
```powershell
1..5 | ForEach-Object { .\compliance-client.exe --config client.yaml --once; Start-Sleep -Seconds 2 }
```

**Bash:**
```bash
for i in {1..5}; do ./compliance-client.exe --config client.yaml --once; sleep 2; done
```

---

### 2.4 Troubleshooting Client Issues

#### Issue: Client running in standalone mode
**Symptom:** No data appears in server dashboard

**Logs:**
```
level=INFO msg="Running in standalone mode" reason="no server URL configured"
```

**Solution:**
Check `client.yaml` - ensure `server.url` is NOT empty:
```yaml
server:
  url: "http://localhost:8443"  # Must be set!
```

#### Issue: Authentication failure
**Logs:**
```
level=ERROR msg="Failed to submit" error="401 Unauthorized"
```

**Server Logs:**
```
level=WARN msg="Invalid API key" remote_addr=127.0.0.1
```

**Solution:**
1. Check API key in `client.yaml` matches `server.yaml`:
```yaml
# client.yaml
server:
  api_key: "demo-key-67890"

# server.yaml
auth:
  api_keys:
    - "demo-key-67890"  # Must match!
```

2. Verify auth is enabled:
```yaml
# server.yaml
auth:
  enabled: true
  require_key: true
```

#### Issue: Connection refused
**Logs:**
```
level=ERROR msg="Failed to submit" error="connection refused"
```

**Solution:**
1. Verify server is running: `curl http://localhost:8443/api/v1/health`
2. Check port matches in both configs
3. Check firewall isn't blocking port 8443

#### Issue: Report config not found
**Logs:**
```
level=ERROR msg="Failed to load report config" error="configs/reports not found"
```

**Solution:**
```bash
# Must run client from root directory where configs/ exists
cd /d/golang-labs/ComplianceToolkit
./compliance-client.exe --config client.yaml --once
```

---

## Part 3: Dashboard Testing

### 3.1 Access Dashboard

**URL:** `http://localhost:8443/dashboard`

**Expected Elements:**

#### Header
- ✅ "Compliance Dashboard" title
- ✅ "Toggle Theme" button (switches light/dark mode)

#### Stats Cards (Top)
- ✅ **Total Clients** - Count of registered clients
- ✅ **Active Clients** - Clients seen in last 24 hours
- ✅ **Compliant Clients** - Clients with passing compliance
- ✅ **Compliance Rate** - Percentage with color coding:
  - Green: ≥80%
  - Yellow: 50-79%
  - Red: <50%

#### Recent Submissions Section
- ✅ Table with columns:
  - Hostname
  - Report Type
  - Status (badge: compliant/non-compliant/partial)
  - Passed (green text)
  - Failed (red text)
  - Time (relative + absolute)

#### Active Clients Section
- ✅ Table with columns:
  - Hostname (with client ID below)
  - OS / Build
  - Network (IP + MAC)
  - Status badge
  - Compliance score (color-coded %)
  - Last Seen
  - **Actions** - "View Details →" link ← **NEW**

---

### 3.2 Dashboard Functionality Tests

| Test | Action | Expected Result |
|------|--------|-----------------|
| **Theme Toggle** | Click "Toggle Theme" button | Page switches between light/dark mode |
| **Auto-refresh** | Wait 30 seconds | Dashboard data refreshes automatically |
| **Stats Update** | Send new submission | Stats cards update with new counts |
| **Submission List** | Check recent submissions | Shows last 10 submissions, newest first |
| **Client List** | Check active clients | Shows all registered clients with details |
| **Responsive Design** | Resize browser window | Layout adapts to different screen sizes |

---

### 3.3 Troubleshooting Dashboard Issues

#### Issue: Dashboard not loading
**Browser Error:** 404 Not Found

**Solution:**
1. Verify server is running
2. Check dashboard is enabled in `server.yaml`:
```yaml
dashboard:
  enabled: true
  path: "/dashboard"
```

#### Issue: No data showing
**Symptom:** Dashboard loads but shows "No submissions yet" / "No clients registered yet"

**Solution:**
1. Run client to send test data: `./compliance-client.exe --config client.yaml --once`
2. Wait 30 seconds for auto-refresh or hard refresh browser (Ctrl+Shift+R)

#### Issue: "Failed to load dashboard data"
**Browser Console Error:** `Failed to fetch`

**Solution:**
1. Check server logs for errors
2. Verify server is responding: `curl http://localhost:8443/api/v1/dashboard/summary`
3. Check browser console (F12) for CORS or network errors

---

## Part 4: Client Detail Page Testing

### 4.1 Access Client Detail Page

**Method 1: From Dashboard**
1. Navigate to `http://localhost:8443/dashboard`
2. Find client in "Active Clients" table
3. Click "View Details →" link

**Method 2: Direct URL**
```
http://localhost:8443/client-detail?client_id=<CLIENT_ID>
```

**To find client ID:**
- Look in dashboard clients table (shown below hostname)
- Check server logs for `client_id` in submission messages
- Example: `client-Ultrawide`, `client-WIN-SERVER-01`

---

### 4.2 Client Detail Page Elements

#### Header Section
- ✅ "← Back to Dashboard" button
- ✅ "Client Details" title
- ✅ Breadcrumb: "Dashboard / <hostname>"
- ✅ "Toggle Theme" button

#### Client Profile Card
- ✅ Hostname (large title)
- ✅ Status badge (active/inactive)
- ✅ Client ID
- ✅ First Seen timestamp
- ✅ Last Seen timestamp
- ✅ Compliance Score percentage

#### Quick Stats (4 cards)
- ✅ **Compliance Score** - Percentage (large number)
- ✅ **Total Submissions** - Count of all submissions
- ✅ **Last Submission** - Timestamp
- ✅ **Status** - Badge (active/inactive)

#### System Information Section
- ✅ OS Version
- ✅ Build Number
- ✅ Architecture (x64, ARM64, etc.)
- ✅ Domain
- ✅ IP Address
- ✅ MAC Address

#### Compliance Trend Chart
- ✅ Line chart showing compliance over time
- ✅ X-axis: Dates (oldest to newest)
- ✅ Y-axis: Compliance percentage (0-100%)
- ✅ Interactive tooltips on hover
- ✅ Responsive to theme changes

#### Submission History Table
- ✅ Columns:
  - Timestamp
  - Report Type
  - Status badge
  - Passed checks
  - Failed checks
  - Score percentage
  - Actions ("View" button)
- ✅ Sorted by newest first
- ✅ Color-coded status badges

#### Export Functionality
- ✅ "Export History" button
- ✅ Downloads JSON file with complete client data
- ✅ Filename: `client_{id}_history_{timestamp}.json`

---

### 4.3 Client Detail Page Functionality Tests

| Test | Action | Expected Result |
|------|--------|-----------------|
| **Load Client Profile** | Navigate to client detail page | All client metadata displays correctly |
| **Load Submission History** | Check submissions table | All past submissions listed, newest first |
| **Compliance Trend Chart** | View chart | Line graph shows historical compliance scores |
| **Chart Hover** | Hover over data points | Tooltip shows exact score for that date |
| **Export History** | Click "Export History" button | JSON file downloads with client data |
| **View Submission** | Click "View" on submission | Alert shows (feature not yet implemented) |
| **Back to Dashboard** | Click "← Back to Dashboard" | Returns to dashboard |
| **Theme Toggle** | Switch theme | Chart and page colors update accordingly |
| **Authentication** | Check browser dev tools | No hardcoded API keys visible (uses cookies) |

---

### 4.4 Troubleshooting Client Detail Page Issues

#### Issue: "Failed to load client data"
**Error Message:** "Failed to load client data: Failed to load client data"

**Solution:**
1. Verify client exists:
```bash
curl -H "Authorization: Bearer demo-key-67890" http://localhost:8443/api/v1/clients
```

2. Check server logs for authentication errors:
```
level=WARN msg="Invalid API key"
```

3. Clear browser cookies and refresh page (forces new session)

#### Issue: "Failed to load submissions"
**Error Message:** "Failed to load submissions: Failed to load submissions"

**Causes:**
1. **Authentication issue** - Cookie expired or invalid
2. **No submissions exist** - Client hasn't sent any reports yet
3. **Database error** - Check server logs

**Solution:**
```bash
# 1. Send a test submission for this client
./compliance-client.exe --config client.yaml --once

# 2. Refresh the client detail page
# (Hard refresh: Ctrl+Shift+R)

# 3. Check server logs for errors
```

#### Issue: Chart not rendering
**Symptom:** Chart section shows but graph is blank

**Causes:**
1. **Chart.js not loaded** - Check browser console for CDN errors
2. **No submission data** - Need at least 1 submission
3. **JavaScript error** - Check browser console (F12)

**Solution:**
```bash
# Send multiple submissions to create trend data
for i in {1..5}; do ./compliance-client.exe --config client.yaml --once; sleep 2; done
```

#### Issue: "No client ID specified"
**Error Message:** "No client ID specified"

**Cause:** Missing `?client_id=` parameter in URL

**Solution:**
Navigate from dashboard (click "View Details →") or use correct URL format:
```
http://localhost:8443/client-detail?client_id=client-Ultrawide
```

---

## Part 5: Settings Page Testing

### 5.1 Access Settings Page

**URL:** `http://localhost:8443/settings`

**Expected Sections:**

#### Server Information
- ✅ Server Version
- ✅ Server Address (protocol://host:port)
- ✅ TLS Status badge (Enabled/Disabled)
- ✅ Database Type (SQLite)
- ✅ Database Path
- ✅ Authentication Status (shows key count)

#### API Key Management
- ✅ List of existing API keys (masked)
  - Format: `test****2345`
- ✅ "Add New API Key" button
- ✅ "Generate Random" button (32-char key generator)
- ✅ "Copy" button for each key
- ✅ "Delete" button for each key

---

### 5.2 Settings Page Functionality Tests

| Test | Action | Expected Result |
|------|--------|-----------------|
| **Load Server Info** | Navigate to settings page | All server config displays correctly |
| **View API Keys** | Check API keys section | Keys shown masked (first 4 + last 4 chars) |
| **Generate Random Key** | Click "Generate Random" | 32-character key appears in input field |
| **Add API Key** | Enter key and save | Key added to list, page refreshes |
| **Copy API Key** | Click "Copy" button | Key copied to clipboard, success notification |
| **Delete API Key** | Click "Delete" and confirm | Key removed from list, page refreshes |
| **Theme Toggle** | Switch theme | Page colors update |

---

### 5.3 Settings Page Limitations

**Important Notes:**
1. ⚠️ **Runtime Only** - Changes are NOT saved to `server.yaml`
2. ⚠️ **Temporary** - Server restart loses all changes
3. ⚠️ **No Audit Trail** - Changes only logged to server logs

**To make changes permanent:**
1. Manually update `server.yaml`
2. Restart server

**Future Enhancement:** YAML file persistence (see `docs/project/FUTURE_ENHANCEMENTS.md`)

---

## Part 6: API Testing

### 6.1 API Endpoint Tests

Use `curl` or PowerShell to test API endpoints directly.

#### Health Check
```bash
curl http://localhost:8443/api/v1/health
```

**Expected:**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

---

#### List Clients
```bash
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients
```

**Expected:**
```json
[
  {
    "id": "1",
    "client_id": "client-Ultrawide",
    "hostname": "ULTRAWIDE",
    "first_seen": "2025-10-06T21:50:00Z",
    "last_seen": "2025-10-06T21:55:00Z",
    "status": "active",
    "compliance_score": 87.5,
    "system_info": {
      "os_version": "Windows 11 Pro",
      "build_number": "22621.2715",
      "architecture": "x64",
      "domain": "WORKGROUP",
      "ip_address": "192.168.1.100",
      "mac_address": "00:1A:2B:3C:4D:5E"
    }
  }
]
```

---

#### Get Client Details
```bash
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients/client-Ultrawide
```

**Expected:** Same structure as list, but single client

---

#### Get Client Submissions
```bash
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients/client-Ultrawide/submissions
```

**Expected:**
```json
[
  {
    "submission_id": "sub-xyz789",
    "client_id": "client-Ultrawide",
    "hostname": "ULTRAWIDE",
    "timestamp": "2025-10-06T21:55:00Z",
    "report_type": "NIST_800_171_compliance",
    "overall_status": "compliant",
    "total_checks": 40,
    "passed_checks": 35,
    "failed_checks": 5
  }
]
```

---

#### Dashboard Summary
```bash
curl http://localhost:8443/api/v1/dashboard/summary
```

**Expected:**
```json
{
  "total_clients": 1,
  "active_clients": 1,
  "compliant_clients": 1,
  "recent_submissions": [...],
  "compliance_by_type": {
    "NIST_800_171_compliance": {
      "total_submissions": 5,
      "average_score": 87.5,
      "pass_rate": 80.0,
      "fail_rate": 20.0
    }
  }
}
```

---

### 6.2 PowerShell API Testing

```powershell
# Set variables
$ServerUrl = "http://localhost:8443"
$ApiKey = "demo-key-67890"
$Headers = @{
    "Authorization" = "Bearer $ApiKey"
    "Content-Type" = "application/json"
}

# Health check
Invoke-RestMethod -Uri "$ServerUrl/api/v1/health"

# List clients
Invoke-RestMethod -Uri "$ServerUrl/api/v1/clients" -Headers $Headers

# Get specific client
$ClientID = "client-Ultrawide"
Invoke-RestMethod -Uri "$ServerUrl/api/v1/clients/$ClientID" -Headers $Headers

# Get client submissions
Invoke-RestMethod -Uri "$ServerUrl/api/v1/clients/$ClientID/submissions" -Headers $Headers

# Dashboard summary
Invoke-RestMethod -Uri "$ServerUrl/api/v1/dashboard/summary"
```

---

### 6.3 API Error Testing

#### Test Invalid API Key
```bash
curl -H "Authorization: Bearer invalid-key" \
     http://localhost:8443/api/v1/clients
```

**Expected:**
```json
{
  "error": "Unauthorized",
  "message": "Invalid API key",
  "code": 401
}
```

---

#### Test Missing API Key
```bash
curl http://localhost:8443/api/v1/clients
```

**Expected:**
```json
{
  "error": "Unauthorized",
  "message": "API key required",
  "code": 401
}
```

---

#### Test Non-existent Client
```bash
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients/invalid-client-id
```

**Expected:**
```json
{
  "error": "Not Found",
  "message": "Client not found",
  "code": 404
}
```

---

## Part 7: Database Testing

### 7.1 Inspect Database

**Location:** `cmd/compliance-server/data/compliance.db`

**Tools:**
- SQLite CLI: `sqlite3 data/compliance.db`
- DB Browser for SQLite (GUI)
- VS Code SQLite extension

---

### 7.2 Common Queries

```sql
-- Count clients
SELECT COUNT(*) FROM clients;

-- List all clients
SELECT client_id, hostname, status, last_seen FROM clients;

-- Count submissions
SELECT COUNT(*) FROM submissions;

-- List recent submissions
SELECT submission_id, client_id, hostname, timestamp, overall_status
FROM submissions
ORDER BY timestamp DESC
LIMIT 10;

-- Get client compliance score
SELECT
    c.client_id,
    c.hostname,
    COUNT(s.id) as total_submissions,
    SUM(CASE WHEN s.overall_status = 'compliant' THEN 1 ELSE 0 END) * 100.0 / COUNT(s.id) as compliance_rate
FROM clients c
LEFT JOIN submissions s ON c.client_id = s.client_id
GROUP BY c.client_id;

-- Get submissions by report type
SELECT report_type, COUNT(*) as count
FROM submissions
GROUP BY report_type;
```

---

### 7.3 Reset Database

**To start fresh (deletes all data):**

```bash
# Stop server first (Ctrl+C)

# Delete database file
rm cmd/compliance-server/data/compliance.db

# Or Windows PowerShell:
Remove-Item cmd\compliance-server\data\compliance.db

# Restart server (recreates database)
cd cmd/compliance-server
./compliance-server.exe --config server.yaml
```

---

## Part 8: Integration Testing Workflow

### 8.1 Complete Test Cycle

**Scenario:** Test complete client-to-dashboard flow

```bash
# 1. Reset database (optional)
rm cmd/compliance-server/data/compliance.db

# 2. Start server
cd cmd/compliance-server
./compliance-server.exe --config server.yaml &

# 3. Wait for server to start
sleep 2

# 4. Verify server health
curl http://localhost:8443/api/v1/health

# 5. Send 5 test submissions
cd ../..
for i in {1..5}; do
    ./compliance-client.exe --config client.yaml --once
    sleep 2
done

# 6. Open dashboard in browser
# Navigate to: http://localhost:8443/dashboard

# 7. Verify data appears:
#    - Total Clients: 1
#    - Recent Submissions: 5
#    - Compliance trend visible

# 8. Test client detail page:
#    - Click "View Details →" on client
#    - Verify profile loads
#    - Verify submissions table shows 5 entries
#    - Verify chart shows trend line

# 9. Test export functionality:
#    - Click "Export History"
#    - Verify JSON file downloads

# 10. Test settings page:
#     - Navigate to /settings
#     - Verify server info loads
#     - Add a test API key
#     - Delete the test API key
```

---

### 8.2 Multi-Client Testing

**Scenario:** Test with multiple clients

```yaml
# Create client2.yaml (copy of client.yaml)
# Change hostname or let it auto-detect different hostname

# Run from different machine OR
# Modify client_id generation in code to use different prefix
```

```bash
# Terminal 1: Client 1
./compliance-client.exe --config client.yaml --once

# Terminal 2: Client 2 (from different machine)
./compliance-client.exe --config client2.yaml --once

# Dashboard should now show:
# - Total Clients: 2
# - Two entries in Active Clients table
```

---

### 8.3 Load Testing

**Scenario:** Stress test with many submissions

```bash
# Send 100 submissions rapidly
for i in {1..100}; do
    ./compliance-client.exe --config client.yaml --once &
done

# Wait for all to complete
wait

# Check server logs for errors
# Check dashboard still responsive
# Check database size: du -h cmd/compliance-server/data/compliance.db
```

---

## Part 9: Security Testing

### 9.1 Authentication Tests

#### Test 1: Cookie-based auth (Dashboard)
```bash
# 1. Open browser dev tools (F12)
# 2. Navigate to http://localhost:8443/dashboard
# 3. Check Application > Cookies
# 4. Verify "api_token" cookie exists
# 5. Verify cookie properties:
#    - HttpOnly: true
#    - SameSite: Strict
#    - Path: /
```

#### Test 2: Header-based auth (API)
```bash
# Should work with Bearer token
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients

# Should fail without token
curl http://localhost:8443/api/v1/clients
```

#### Test 3: Mixed auth
```bash
# Dashboard pages should work with cookie only (no Authorization header needed)
# API endpoints should work with either cookie OR header
```

---

### 9.2 Authorization Tests

#### Test 1: Disable auth
```yaml
# Edit server.yaml
auth:
  enabled: false  # Disable auth

# Restart server
# All endpoints should work without API key
curl http://localhost:8443/api/v1/clients  # Should succeed
```

#### Test 2: Require key
```yaml
# Edit server.yaml
auth:
  enabled: true
  require_key: true

# Restart server
# All endpoints should require API key
curl http://localhost:8443/api/v1/clients  # Should fail (401)
```

---

### 9.3 XSS/CSRF Protection Tests

#### Test 1: XSS protection
```javascript
// In browser console on client detail page
// Try to access cookie
document.cookie
// Should NOT see "api_token" (HttpOnly prevents access)
```

#### Test 2: CSRF protection
```bash
# Try to make request from different origin
# SameSite=Strict should prevent cross-site requests
```

---

## Part 10: Browser Compatibility Testing

### 10.1 Supported Browsers

| Browser | Version | Status |
|---------|---------|--------|
| Chrome | 120+ | ✅ Fully Supported |
| Edge | 120+ | ✅ Fully Supported |
| Firefox | 120+ | ✅ Fully Supported |
| Safari | 17+ | ✅ Fully Supported |

---

### 10.2 Browser Feature Tests

#### Test in Each Browser:

1. **Dashboard Page**
   - Loads without errors
   - Stats cards display correctly
   - Tables are formatted properly
   - Auto-refresh works (30 sec)
   - Theme toggle works

2. **Client Detail Page**
   - Profile loads
   - Chart.js renders correctly
   - Export downloads file
   - All interactive elements work

3. **Settings Page**
   - Server info displays
   - API key management works
   - Copy to clipboard works

4. **Responsive Design**
   - Resize window to mobile size (375px)
   - Verify layout adapts
   - Verify no horizontal scrolling
   - Verify touch-friendly buttons

---

## Part 11: Performance Testing

### 11.1 Response Time Tests

**Expected response times:**

| Endpoint | Expected | Acceptable |
|----------|----------|------------|
| /api/v1/health | <10ms | <50ms |
| /api/v1/clients | <50ms | <200ms |
| /api/v1/clients/{id} | <20ms | <100ms |
| /api/v1/clients/{id}/submissions | <50ms | <200ms |
| /api/v1/dashboard/summary | <100ms | <500ms |

**Measure with curl:**
```bash
curl -o /dev/null -s -w "Time: %{time_total}s\n" \
     http://localhost:8443/api/v1/health
```

---

### 11.2 Database Performance

```sql
-- Check database size
SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size();

-- Analyze query performance
EXPLAIN QUERY PLAN
SELECT * FROM submissions WHERE client_id = 'client-Ultrawide';

-- Should show "SEARCH submissions USING INDEX idx_submissions_client_id"
```

---

### 11.3 Memory Usage

**Monitor server memory:**

```bash
# Linux/Mac
ps aux | grep compliance-server

# Windows PowerShell
Get-Process compliance-server | Select-Object PM, VM, CPU
```

**Expected:**
- Initial: ~20-30 MB
- With 100 clients: ~50-80 MB
- With 1000 submissions: ~100-150 MB

---

## Part 12: Error Handling Tests

### 12.1 Server Errors

#### Test 1: Invalid config
```yaml
# Edit server.yaml with invalid port
server:
  port: "not-a-number"  # Invalid

# Try to start server
# Should show error and exit gracefully
```

#### Test 2: Database corruption
```bash
# Corrupt database file
echo "corrupt" > cmd/compliance-server/data/compliance.db

# Try to start server
# Should show error about database
```

#### Test 3: Disk full
```bash
# Simulate disk full (testing environment only!)
# Server should handle gracefully and log error
```

---

### 12.2 Client Errors

#### Test 1: Server unreachable
```yaml
# Edit client.yaml
server:
  url: "http://localhost:9999"  # Wrong port

# Run client
# Should retry and eventually cache submission
```

#### Test 2: Invalid API key
```yaml
# Edit client.yaml
server:
  api_key: "invalid-key"

# Run client
# Should fail with 401 Unauthorized
```

#### Test 3: Report config not found
```yaml
# Edit client.yaml
reports:
  reports:
    - "non-existent-report.json"

# Run client
# Should show error about missing config
```

---

### 12.3 Dashboard Errors

#### Test 1: API endpoint down
```bash
# Stop server
# Open dashboard
# Should show "Failed to load dashboard data"
```

#### Test 2: Invalid client ID
```
# Navigate to invalid client detail
http://localhost:8443/client-detail?client_id=invalid-id

# Should show error: "Failed to load client data"
```

#### Test 3: Network timeout
```javascript
// Simulate slow network in browser dev tools
// Network tab > Throttling > Slow 3G
// Dashboard should show loading states
```

---

## Part 13: Logging and Debugging

### 13.1 Enable Debug Logging

**Server:**
```yaml
# Edit server.yaml
logging:
  level: "debug"  # Change from "info"
```

**Client:**
```yaml
# Edit client.yaml
logging:
  level: "debug"
```

**Restart both and observe detailed logs**

---

### 13.2 Log Analysis

**Server logs to watch for:**

✅ **Good:**
```
level=INFO msg="HTTP request" method=POST path=/api/v1/compliance/submit status=200
level=INFO msg="Received compliance submission" submission_id=sub-xyz
```

❌ **Bad:**
```
level=ERROR msg="Failed to save submission" error="database locked"
level=WARN msg="Invalid API key" remote_addr=127.0.0.1
level=ERROR msg="Database connection lost"
```

---

### 13.3 Browser Console Debugging

**Press F12 to open dev tools:**

1. **Console tab** - Check for JavaScript errors
2. **Network tab** - Check API calls and responses
3. **Application tab** - Check cookies and local storage

**Common issues:**
- `Failed to fetch` - Server unreachable or CORS issue
- `401 Unauthorized` - Auth cookie missing or invalid
- `Chart is not defined` - Chart.js CDN failed to load

---

## Part 14: Cleanup and Reset

### 14.1 Clean Shutdown

```bash
# Stop server gracefully
# Press Ctrl+C in server terminal

# Expected output:
level=INFO msg="Shutting down server..."
level=INFO msg="Server shutdown complete"
```

---

### 14.2 Reset Everything

```bash
# 1. Stop server (Ctrl+C)

# 2. Delete database
rm cmd/compliance-server/data/compliance.db

# 3. Clear client cache
rm -rf cache/submissions/*

# 4. Clear output files
rm -rf output/reports/*
rm -rf output/evidence/*

# 5. Restart server
cd cmd/compliance-server
./compliance-server.exe --config server.yaml
```

---

### 14.3 Backup Data

**Before resetting, backup important data:**

```bash
# Backup database
cp cmd/compliance-server/data/compliance.db \
   cmd/compliance-server/data/compliance.db.backup

# Backup configurations
cp cmd/compliance-server/server.yaml server.yaml.backup
cp client.yaml client.yaml.backup

# Export data via API
curl -H "Authorization: Bearer demo-key-67890" \
     http://localhost:8443/api/v1/clients > clients_backup.json
```

---

## Part 15: Common Testing Scenarios

### Scenario 1: Fresh Installation Test

**Goal:** Verify clean installation works

```bash
# 1. Extract toolkit to new directory
# 2. Generate configs
cd cmd/compliance-server
./compliance-server.exe --generate-config
cd ../..
./compliance-client.exe --generate-config

# 3. Edit configs (set server URL, API keys)
# 4. Start server
# 5. Run client
# 6. Access dashboard
# 7. Verify all features work
```

---

### Scenario 2: Upgrade Test

**Goal:** Verify upgrade preserves data

```bash
# 1. Backup existing database
cp cmd/compliance-server/data/compliance.db compliance.db.backup

# 2. Stop old server
# 3. Replace executables with new versions
# 4. Start new server (same config)
# 5. Verify existing data still accessible
# 6. Run client with new version
# 7. Verify backwards compatibility
```

---

### Scenario 3: High Availability Test

**Goal:** Test server restart without data loss

```bash
# 1. Server running with active clients
# 2. Stop server (Ctrl+C)
# 3. Database should be intact
# 4. Restart server
# 5. Verify all data recovered
# 6. Clients should reconnect and continue
```

---

## Part 16: Test Checklist

### Pre-Test Checklist

- [ ] Server executable built (`go build`)
- [ ] Client executable built
- [ ] `server.yaml` configured correctly
- [ ] `client.yaml` configured correctly
- [ ] HTML files in correct location
- [ ] Report configs available (`configs/reports/`)
- [ ] No port conflicts (8443 available)

---

### Server Test Checklist

- [ ] Server starts without errors
- [ ] Health check responds
- [ ] Dashboard loads
- [ ] Settings page loads
- [ ] Client detail page loads (after data submitted)
- [ ] API endpoints respond correctly
- [ ] Authentication works (cookie + header)
- [ ] Database initializes correctly
- [ ] Logs show no errors

---

### Client Test Checklist

- [ ] Client runs without errors
- [ ] Report execution completes
- [ ] Submission sent to server
- [ ] Server confirms receipt
- [ ] Data appears in dashboard
- [ ] Retry logic works on failure
- [ ] Cache works when offline
- [ ] Logs show no errors

---

### Dashboard Test Checklist

- [ ] Page loads without errors
- [ ] Stats cards show correct data
- [ ] Recent submissions table populates
- [ ] Active clients table populates
- [ ] "View Details →" links work
- [ ] Auto-refresh works (30 sec)
- [ ] Theme toggle works
- [ ] Responsive design adapts to screen size

---

### Client Detail Page Test Checklist

- [ ] Page loads without errors
- [ ] Client profile displays
- [ ] Stats cards populate
- [ ] System information displays
- [ ] Submission history table populates
- [ ] Compliance trend chart renders
- [ ] Export history downloads JSON
- [ ] "Back to Dashboard" works
- [ ] Theme toggle works
- [ ] No hardcoded API keys visible

---

### Settings Page Test Checklist

- [ ] Page loads without errors
- [ ] Server info displays correctly
- [ ] API keys list (masked)
- [ ] Add API key works
- [ ] Delete API key works
- [ ] Copy key to clipboard works
- [ ] Theme toggle works

---

## Part 17: Troubleshooting Quick Reference

| Symptom | Possible Cause | Solution |
|---------|---------------|----------|
| Server won't start | Port in use | Change port or kill process |
| Dashboard 404 | Wrong URL | Use `/dashboard` not `/` |
| No data in dashboard | Client not sending | Check client config `server.url` |
| 401 Unauthorized | API key mismatch | Check keys match in configs |
| Client detail fails | Cookie expired | Clear cookies and refresh |
| Chart not rendering | No submissions | Send multiple submissions |
| Database locked | Multiple instances | Stop all servers, restart one |
| HTML files not found | Wrong directory | Run from `cmd/compliance-server/` |

---

## Part 18: Success Criteria

### Minimum Viable Test (5 minutes)

✅ **Pass if:**
1. Server starts without errors
2. Client sends one submission successfully
3. Dashboard shows the submission
4. Client detail page loads with data

---

### Complete Test Suite (30 minutes)

✅ **Pass if:**
1. All server endpoints respond correctly
2. Multiple clients can submit concurrently
3. Dashboard displays all data accurately
4. Client detail page shows history and chart
5. Settings page manages API keys
6. Export functionality works
7. Authentication is secure (no exposed tokens)
8. Theme toggle works on all pages
9. Browser compatibility verified
10. No errors in logs

---

## Part 19: Documentation References

- **Architecture:** `docs/developer-guide/ARCHITECTURE.md`
- **Client Detail Page:** `docs/project/CLIENT_DETAIL_PAGE.md`
- **Settings Page:** `docs/project/SETTINGS_PAGE_ENHANCEMENTS.md`
- **Future Features:** `docs/project/FUTURE_ENHANCEMENTS.md`
- **API Reference:** (To be created)
- **User Guide:** `docs/user-guide/QUICKSTART.md`

---

## Part 20: Continuous Testing

### Automated Testing Setup

**PowerShell script for repeated testing:**

```powershell
# test-cycle.ps1
param(
    [int]$Iterations = 5,
    [int]$DelaySeconds = 2
)

Write-Host "Running $Iterations test submissions..."

for ($i = 1; $i -le $Iterations; $i++) {
    Write-Host "Submission $i of $Iterations"
    .\compliance-client.exe --config client.yaml --once
    Start-Sleep -Seconds $DelaySeconds
}

Write-Host "Test cycle complete!"
Write-Host "Check dashboard at: http://localhost:8443/dashboard"
```

**Run:**
```powershell
.\test-cycle.ps1 -Iterations 10 -DelaySeconds 3
```

---

## Summary

This testing guide covers:

✅ **Server Setup** - Configuration and startup
✅ **Client Setup** - Configuration and test data generation
✅ **Dashboard Testing** - UI and functionality
✅ **Client Detail Page** - New feature testing
✅ **Settings Page** - Configuration management
✅ **API Testing** - Direct endpoint testing
✅ **Database Testing** - Data verification
✅ **Integration Testing** - End-to-end workflows
✅ **Security Testing** - Authentication and authorization
✅ **Performance Testing** - Response times and load
✅ **Error Handling** - Graceful failure scenarios
✅ **Browser Testing** - Cross-browser compatibility

**For questions or issues, refer to:**
- GitHub Issues: (if applicable)
- Documentation: `docs/` directory
- Server logs: Check console output

---

**Version:** 1.1.0
**Last Updated:** October 6, 2025
**Status:** Complete and Ready for Use
