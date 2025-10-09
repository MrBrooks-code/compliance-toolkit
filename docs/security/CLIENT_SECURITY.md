# Client Submission Security Guide

## Overview

This document provides comprehensive guidance on securing client-to-server communication in the Compliance Toolkit. **By default, the server allows unauthenticated submissions** - this guide shows you how to properly secure your deployment.

## Current Security Issue

**⚠️ CRITICAL: Default Configuration is Insecure**

The default configuration in `server.yaml` has:
```yaml
auth:
  enabled: true
  require_key: false  # ⚠️ This allows unauthenticated access!
```

When `require_key: false`, the server's `authMiddleware` allows **any client** to submit compliance data without authentication. This means:
- ❌ Anyone can submit fake compliance reports
- ❌ No audit trail of who submitted what
- ❌ Malicious actors can pollute your compliance database
- ❌ No way to revoke compromised clients

## Threat Model

### Without Authentication (Current Default)
- **Unauthorized submissions**: Anyone on your network can submit data
- **Data poisoning**: Attackers can submit false compliance reports
- **Resource exhaustion**: Malicious clients can flood the server
- **No accountability**: Cannot trace submissions back to specific endpoints

### With Proper Authentication
- ✅ Only authorized clients can submit
- ✅ Each client has unique credentials that can be revoked
- ✅ Full audit trail of all submissions
- ✅ Ability to rotate compromised keys

---

## Securing Client Submissions

### Step 1: Enable Required Authentication

**Server Configuration** (`server.yaml` or Docker environment):

```yaml
auth:
  enabled: true
  require_key: true  # ✅ REQUIRED: Enforce authentication
  use_hashed_keys: false
  api_keys: []
  api_key_hashes: []

  # JWT configuration (for dashboard users)
  jwt:
    enabled: true
    secret_key: "your-secure-secret-key-here"  # Generate with: openssl rand -base64 32
    access_token_lifetime: 15
    refresh_token_lifetime: 7
    issuer: "ComplianceToolkit"
    audience: "ComplianceToolkit"
```

**Docker Environment** (`.env` file):
```bash
# CRITICAL: Enable authentication enforcement
AUTH_ENABLED=true
AUTH_REQUIRE_KEY=true  # ✅ THIS IS THE KEY SETTING

# Generate secure JWT secret: openssl rand -base64 32
JWT_ENABLED=true
JWT_SECRET_KEY=your-generated-secret-key-here

# Enable TLS for production
TLS_ENABLED=true
TLS_CERT_FILE=/app/certs/server.crt
TLS_KEY_FILE=/app/certs/server.key
```

### Step 2: Generate API Keys for Clients

The server provides two methods for API key management:

#### Option A: Database-Backed API Keys (Recommended)

**Advantages:**
- ✅ Centralized management via web dashboard
- ✅ Per-key metadata (name, created_by, last_used)
- ✅ Individual key activation/deactivation
- ✅ Audit trail of key usage
- ✅ Automatic expiration support

**Generate via Web Dashboard:**
1. Login at `https://your-server:8080/login`
2. Navigate to Settings → API Keys
3. Click "Generate API Key"
4. **Copy the key immediately** (only shown once!)
5. Give it a descriptive name (e.g., "Windows-Server-01")

**Generate via API:**
```bash
# Login and get JWT token
curl -X POST https://your-server:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}' \
  > auth.json

# Extract access token
ACCESS_TOKEN=$(jq -r '.access_token' auth.json)

# Generate API key
curl -X POST https://your-server:8080/api/v1/apikeys/generate \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Windows-Server-01",
    "expires_at": "2026-12-31T23:59:59Z"
  }' | jq -r '.api_key'
```

**Generate via CLI:**
```bash
# Start the server and generate a key
./compliance-server.exe

# In dashboard, create key and save securely
# Then distribute to client installations
```

#### Option B: Static API Keys in Configuration

**⚠️ DEPRECATED - DO NOT USE**

This feature will be removed in a future version due to security concerns.

```yaml
# ❌ DEPRECATED - DO NOT USE IN PRODUCTION OR DEVELOPMENT
auth:
  enabled: true
  require_key: true
  api_keys:
    - "dev-key-1234567890abcdef"  # INSECURE - will be removed
    - "test-client-key-xyz"        # INSECURE - will be removed
```

**Why this feature is being removed:**
- ❌ **Not auditable** - No usage tracking or last_used timestamps
- ❌ **Not revocable** - Cannot disable individual keys without server restart
- ❌ **Easily leaked** - Often committed to version control accidentally
- ❌ **Cannot expire** - No automatic expiration support
- ❌ **No metadata** - Cannot track which key belongs to which client
- ❌ **Shared risk** - If one key leaks, must rotate ALL keys and restart server
- ❌ **No rotation** - Difficult to implement proper key rotation procedures

**Migration Path:**
1. Generate database-backed API keys for all clients
2. Update client configurations with new keys
3. Remove all entries from `api_keys` array in server.yaml
4. Set `api_keys: []` (empty array)

**Future versions will:**
- Remove `api_keys` configuration option entirely
- Remove `api_key_hashes` configuration option
- Require all API keys to be database-backed

### Step 3: Configure Clients with API Keys

**Client Configuration** (`client.yaml`):

```yaml
# Server connection
server:
  url: "https://compliance-server.company.com:8080"
  api_key: "YOUR-SECURE-API-KEY-HERE"  # ✅ Required for authentication
  timeout: 30s
  tls_verify: true  # ✅ Verify server certificate

# Reports to run
reports:
  reports:
    - "NIST_800_171_compliance.json"
    - "Windows_Security_Audit.json"

# Scheduling
schedule:
  enabled: true
  cron: "0 2 * * *"  # Run at 2 AM daily

# Retry configuration
retry:
  max_attempts: 3
  initial_backoff: 5s
  max_backoff: 60s
  backoff_multiplier: 2.0
  retry_on_server_error: true

# Caching (for offline resilience)
cache:
  enabled: true
  path: "cache/submissions"
  max_size_mb: 100
  max_age: 168h  # 7 days
  auto_clean: true
```

### Step 4: Secure API Key Storage on Clients

**Option A: Environment Variables (Recommended)**
```yaml
# client.yaml - reference environment variable
server:
  url: "${COMPLIANCE_SERVER_URL}"
  api_key: "${COMPLIANCE_API_KEY}"  # ✅ Read from environment
  tls_verify: true
```

```powershell
# Set in Windows (system-level)
[System.Environment]::SetEnvironmentVariable(
  "COMPLIANCE_API_KEY",
  "YOUR-API-KEY-HERE",
  [System.EnvironmentVariableTarget]::Machine
)

[System.Environment]::SetEnvironmentVariable(
  "COMPLIANCE_SERVER_URL",
  "https://compliance-server.company.com:8080",
  [System.EnvironmentVariableTarget]::Machine
)
```

**Option B: File Permissions (Windows)**
```powershell
# Restrict client.yaml to SYSTEM and Administrators only
icacls "C:\Program Files\ComplianceClient\client.yaml" /inheritance:r
icacls "C:\Program Files\ComplianceClient\client.yaml" /grant:r "SYSTEM:(F)"
icacls "C:\Program Files\ComplianceClient\client.yaml" /grant:r "Administrators:(F)"
```

**Option C: Windows Credential Manager**
```powershell
# Store API key in Windows Credential Manager
cmdkey /generic:ComplianceToolkitAPIKey /user:client /pass:YOUR-API-KEY-HERE

# Client reads from credential manager (requires code modification)
```

**Option D: Azure Key Vault / AWS Secrets Manager** (Enterprise)
- Store API keys in cloud secret management service
- Client retrieves key at runtime using managed identity
- Centralized rotation and auditing

---

## TLS/HTTPS Configuration

### Why TLS is Critical

Without TLS encryption:
- ❌ API keys transmitted in plaintext
- ❌ Compliance data visible to network eavesdroppers
- ❌ Man-in-the-middle attacks possible
- ❌ Compliance violations (NIST 800-171 requires encryption in transit)

### Server TLS Setup

**Step 1: Generate TLS Certificates**

```bash
# Self-signed certificate (for testing only)
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout docker/certs/server.key \
  -out docker/certs/server.crt \
  -days 365 \
  -subj "/CN=compliance-server.company.com"

# Production: Use certificates from trusted CA
# - Let's Encrypt (free, automated)
# - Internal enterprise CA
# - Commercial CA (DigiCert, Sectigo)
```

**Step 2: Enable TLS in Server**

```yaml
# server.yaml
server:
  host: "0.0.0.0"
  port: 8443  # Standard HTTPS port
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
```

**Docker TLS Configuration:**
```yaml
# docker-compose.yml
services:
  compliance-server:
    environment:
      - TLS_ENABLED=true
      - TLS_CERT_FILE=/app/certs/server.crt
      - TLS_KEY_FILE=/app/certs/server.key
    ports:
      - "8443:8443"  # HTTPS port
    volumes:
      - ./certs:/app/certs:ro  # Read-only certificate mount
```

**Step 3: Client TLS Configuration**

```yaml
# client.yaml
server:
  url: "https://compliance-server.company.com:8443"
  tls_verify: true  # ✅ Verify server certificate

  # For self-signed certs (testing only):
  # tls_verify: false  # ⚠️ INSECURE - Never use in production
```

**Installing Custom CA Certificates on Clients:**

```powershell
# Windows: Install root CA certificate to Trusted Root store
Import-Certificate -FilePath "ca.crt" -CertStoreLocation Cert:\LocalMachine\Root
```

```bash
# Linux: Add to system trust store
sudo cp ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

---

## API Key Rotation Strategy

### When to Rotate Keys

- **Immediately**: If a key is compromised or leaked
- **Regularly**: Every 90 days (compliance requirement)
- **On employee departure**: If person with access leaves
- **After incidents**: Security breach or audit finding

### Rotation Procedure

**1. Generate New Key:**
```bash
# Via dashboard or API
curl -X POST https://server:8080/api/v1/apikeys/generate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Windows-Server-01-NEW"}'
```

**2. Update Client Configuration:**
```powershell
# Update environment variable
[System.Environment]::SetEnvironmentVariable(
  "COMPLIANCE_API_KEY",
  "NEW-API-KEY-HERE",
  [System.EnvironmentVariableTarget]::Machine
)

# Restart compliance client service
Restart-Service -Name "ComplianceClient"
```

**3. Verify New Key Works:**
```bash
# Check server logs for successful submission
docker-compose logs -f compliance-server | grep "Submission accepted"
```

**4. Deactivate Old Key:**
```bash
# Via dashboard: Settings → API Keys → Disable old key
# Or via API:
curl -X POST https://server:8080/api/v1/apikeys/toggle \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"id":123,"active":false}'
```

**5. Monitor for 24 hours:**
- Ensure no errors from clients using old key
- After confirmation, delete old key permanently

### Automated Rotation (Advanced)

```yaml
# client.yaml - support for key rotation
server:
  api_key: "${COMPLIANCE_API_KEY}"
  api_key_rotation:
    enabled: true
    fetch_url: "https://keyvault.company.com/api/keys/compliance-client"
    refresh_interval: 1h
```

---

## Network Security

### Firewall Rules

**Server-Side Rules:**
```bash
# Allow HTTPS from client subnet only
iptables -A INPUT -p tcp --dport 8443 -s 10.0.0.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 8443 -j DROP

# Windows Firewall
New-NetFirewallRule -DisplayName "Compliance Server HTTPS" `
  -Direction Inbound -LocalPort 8443 -Protocol TCP `
  -Action Allow -RemoteAddress 10.0.0.0/24
```

**Client-Side Rules:**
```bash
# Allow outbound HTTPS to server only
New-NetFirewallRule -DisplayName "Compliance Client Outbound" `
  -Direction Outbound -RemoteAddress compliance-server.company.com `
  -RemotePort 8443 -Protocol TCP -Action Allow
```

### Network Segmentation

**Recommended Architecture:**
```
┌─────────────────────────────────────────────┐
│  Compliance Clients (Endpoints)             │
│  - 10.0.0.0/24                              │
│  - Windows workstations & servers           │
└──────────────┬──────────────────────────────┘
               │ HTTPS (8443)
               │ TLS 1.3
               │ API Key Auth
               ▼
┌──────────────────────────────────────────────┐
│  DMZ / Management Network                    │
│  - 10.1.0.0/24                               │
│  ┌──────────────────────────────────┐        │
│  │  Compliance Server (Docker)      │        │
│  │  - Port 8443 (HTTPS)             │        │
│  │  - No direct internet access     │        │
│  └──────────────────────────────────┘        │
└──────────────┬───────────────────────────────┘
               │ HTTPS (443)
               │ Admin access only
               ▼
┌──────────────────────────────────────────────┐
│  Admin Workstations                          │
│  - 10.2.0.0/24                               │
│  - Dashboard access via JWT                  │
└──────────────────────────────────────────────┘
```

### VPN/Zero Trust Access

**Option A: Site-to-Site VPN**
- Clients connect to server over encrypted VPN tunnel
- Server only accessible via VPN
- Additional layer of network authentication

**Option B: Zero Trust Network Access (ZTNA)**
- Per-request verification of identity
- Device posture checks before allowing connections
- Integration with tools like Cloudflare Access, Zscaler

---

## Monitoring and Auditing

### Server-Side Audit Logging

The server automatically logs:
- ✅ Authentication attempts (success/failure)
- ✅ API key usage with timestamps
- ✅ Client submissions with source IP
- ✅ Configuration changes

**Enable structured logging:**
```yaml
# server.yaml
logging:
  level: "info"
  format: "json"  # ✅ Structured logs for SIEM ingestion
  output_path: "/var/log/compliance-server.log"
```

**Forward logs to SIEM:**
```bash
# Splunk Universal Forwarder
[monitor:///var/log/compliance-server.log]
sourcetype = json
index = security

# Elasticsearch/Filebeat
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/compliance-server.log
  json.keys_under_root: true
```

### Key Metrics to Monitor

**Authentication Metrics:**
- Failed authentication attempts per client
- API keys not used in >30 days (stale keys)
- Multiple clients using same API key (key sharing)
- Submissions from unexpected IP addresses

**Submission Metrics:**
- Submission rate anomalies (sudden spikes)
- Failed submissions by client
- Compliance score sudden drops (potential tampering)

**Alert Examples:**
```yaml
# Prometheus AlertManager rule
- alert: UnauthorizedAccessAttempts
  expr: rate(auth_failures[5m]) > 10
  annotations:
    summary: "High rate of authentication failures"

- alert: StaleAPIKey
  expr: (time() - api_key_last_used_timestamp) > 2592000  # 30 days
  annotations:
    summary: "API key not used in 30 days"
```

### Database Audit Queries

```sql
-- List all API keys with usage statistics
SELECT
  name,
  key_prefix,
  created_by,
  created_at,
  last_used,
  is_active,
  (julianday('now') - julianday(last_used)) as days_since_use
FROM api_keys
ORDER BY last_used DESC;

-- Find clients that haven't submitted in 7 days
SELECT
  client_id,
  hostname,
  last_seen,
  (julianday('now') - julianday(last_seen)) as days_offline
FROM clients
WHERE days_offline > 7
ORDER BY last_seen DESC;

-- Compliance score trends
SELECT
  client_id,
  timestamp,
  compliance_score,
  passed_checks,
  failed_checks
FROM submissions
WHERE timestamp > datetime('now', '-30 days')
ORDER BY client_id, timestamp;
```

---

## Security Best Practices Checklist

### Server Deployment
- [ ] Set `AUTH_REQUIRE_KEY=true` in production
- [ ] Enable TLS with valid certificates
- [ ] Generate secure JWT secret (32+ bytes)
- [ ] Use database-backed API keys (not static config)
- [ ] Enable structured JSON logging
- [ ] Configure firewall rules (allow clients only)
- [ ] Set up log forwarding to SIEM
- [ ] Regular database backups
- [ ] Keep server software updated

### Client Deployment
- [ ] Store API keys in environment variables or credential manager
- [ ] Restrict config file permissions (SYSTEM/Administrators only)
- [ ] Enable TLS verification (`tls_verify: true`)
- [ ] Configure submission caching for offline resilience
- [ ] Run as Windows Service with least privilege
- [ ] Enable retry logic with exponential backoff
- [ ] Monitor client logs for submission failures

### Operational Security
- [ ] Rotate API keys every 90 days
- [ ] Deactivate keys for decommissioned clients
- [ ] Review API key usage monthly
- [ ] Monitor for unusual submission patterns
- [ ] Set up alerts for authentication failures
- [ ] Document key distribution procedures
- [ ] Maintain inventory of all clients and their keys
- [ ] Test key rotation procedures quarterly

### Incident Response
- [ ] Documented procedure for compromised keys
- [ ] Ability to quickly revoke individual keys
- [ ] Audit trail of all administrative actions
- [ ] Backup authentication method (emergency admin access)
- [ ] Contact list for security incidents

---

## Example Secure Deployment

### Production Configuration

**server.yaml:**
```yaml
server:
  host: "0.0.0.0"
  port: 8443
  tls:
    enabled: true
    cert_file: "/app/certs/server.crt"
    key_file: "/app/certs/server.key"

database:
  type: "sqlite"
  path: "/app/data/compliance.db"

auth:
  enabled: true
  require_key: true  # ✅ CRITICAL

  # ⚠️ DEPRECATED - These options will be removed in future versions
  # Use database-backed API keys via /api/v1/apikeys instead
  use_hashed_keys: false
  api_keys: []           # ⚠️ Deprecated - leave empty
  api_key_hashes: []     # ⚠️ Deprecated - leave empty

  jwt:
    enabled: true
    secret_key: "WDETeiYcIou7EfZMxP5vK3qB9jN2hL8R"  # openssl rand -base64 32
    access_token_lifetime: 15
    refresh_token_lifetime: 7
    issuer: "CompanyName-ComplianceTK"
    audience: "CompanyName-ComplianceTK"

dashboard:
  enabled: true
  path: "/dashboard"
  login_message: "Authorized Access Only - All Activity Monitored"

logging:
  level: "info"
  format: "json"
  output_path: "/app/logs/server.log"
```

**client.yaml:**
```yaml
server:
  url: "https://compliance-prod.company.com:8443"
  api_key: "${COMPLIANCE_API_KEY}"
  timeout: 30s
  tls_verify: true

reports:
  reports:
    - "NIST_800_171_compliance.json"

schedule:
  enabled: true
  cron: "0 2 * * *"

retry:
  max_attempts: 3
  initial_backoff: 5s
  max_backoff: 60s

cache:
  enabled: true
  path: "C:\\ProgramData\\ComplianceClient\\cache"
  max_size_mb: 100
  max_age: 168h
  auto_clean: true

logging:
  level: "info"
  format: "json"
  output_path: "C:\\ProgramData\\ComplianceClient\\logs\\client.log"
```

---

## FAQ

**Q: Can I use the same API key for multiple clients?**
A: Technically yes, but **strongly discouraged**. Use unique keys per client for:
- Individual revocation if one client is compromised
- Audit trail showing which client submitted what
- Ability to track compliance per endpoint

**Q: What happens if a client's API key is revoked while it's running?**
A: The next submission will fail with HTTP 401 Unauthorized. The client will:
1. Retry with exponential backoff
2. Cache the submission locally
3. Log error for admin notification
4. Continue retrying on schedule

**Q: Do JWT tokens work for client submissions?**
A: JWT is designed for dashboard users (humans). For automated clients, use API keys because:
- API keys don't expire (unless explicitly configured)
- No need for token refresh logic
- Simpler client implementation

**Q: Can I disable authentication for internal networks?**
A: **No**. Never rely on network security alone. Insider threats, compromised endpoints, and lateral movement attacks make internal networks just as vulnerable. Always enforce authentication.

**Q: How do I handle certificate errors with self-signed certs?**
A: Options:
1. **Best**: Install your CA certificate in system trust store
2. **Acceptable**: Use cert pinning in client config
3. **NEVER**: Set `tls_verify: false` in production

---

## Additional Resources

- **NIST 800-171 Rev 2**: Access Control requirements (3.1.x)
- **NIST 800-53**: IA-5 (Authenticator Management)
- **CIS Controls**: Access Control Management
- **OWASP API Security Top 10**: API2:2023 Broken Authentication

## Support

For security-related questions or to report vulnerabilities:
- Email: security@company.com
- Internal: #security-team Slack channel
