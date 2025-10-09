# Quick Security Setup Guide

## 🚨 Critical First Steps (Do This NOW)

### 1. Enable Required Authentication (5 minutes)

**Docker Deployment:**
```bash
# Edit docker/.env or docker-compose.yml
AUTH_ENABLED=true
AUTH_REQUIRE_KEY=true  # ⚠️ THIS IS CRITICAL

# Restart server
cd docker && docker-compose restart
```

**Direct Deployment:**
```yaml
# Edit server.yaml
auth:
  enabled: true
  require_key: true  # ⚠️ THIS IS CRITICAL

  # ⚠️ IMPORTANT: Do NOT use static API keys in config
  # Static api_keys are DEPRECATED and will be removed
  # Use database-backed API keys instead (see Step 2 below)
  api_keys: []         # Leave empty
  api_key_hashes: []   # Leave empty
```

### 2. Generate First API Key (2 minutes)

**Via Dashboard:**
1. Visit `http://localhost:8080/login`
2. Login: `admin` / `admin`
3. Settings → API Keys → Generate
4. **Copy the key immediately** (only shown once!)

**Via Command Line:**
```bash
# Login first
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' > auth.json

# Generate key
ACCESS_TOKEN=$(cat auth.json | grep access_token | cut -d'"' -f4)

curl -X POST http://localhost:8080/api/v1/apikeys/generate \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-client-01"}' | grep api_key
```

### 3. Configure Client (3 minutes)

**Edit client.yaml:**
```yaml
server:
  url: "http://localhost:8080"  # Change to https:// in production
  api_key: "YOUR-GENERATED-KEY-HERE"
  timeout: 30s
  tls_verify: true
```

**Or use environment variable:**
```powershell
# Windows
[System.Environment]::SetEnvironmentVariable(
  "COMPLIANCE_API_KEY",
  "YOUR-KEY-HERE",
  [System.EnvironmentVariableTarget]::Machine
)
```

```bash
# Linux
export COMPLIANCE_API_KEY="YOUR-KEY-HERE"
```

### 4. Test Authentication (1 minute)

**Verify unauthenticated requests are blocked:**
```bash
# This should FAIL with 401 Unauthorized
curl -X POST http://localhost:8080/api/v1/compliance/submit \
  -H "Content-Type: application/json" \
  -d '{"test":"data"}'
# Expected: {"error":"Unauthorized","message":"Authentication required","code":401}
```

**Verify authenticated requests work:**
```bash
# This should SUCCEED
curl -X GET http://localhost:8080/api/v1/health \
  -H "Authorization: Bearer YOUR-API-KEY-HERE"
# Expected: {"status":"healthy","version":"1.0.0"}
```

---

## 📋 Security Checklist

Copy this checklist and verify each item:

### Server Security
- [ ] `AUTH_REQUIRE_KEY=true` is set
- [ ] TLS/HTTPS enabled in production
- [ ] JWT secret is random (not default)
- [ ] Default admin password changed from `admin`
- [ ] Firewall rules limit access to clients only
- [ ] Logs forwarded to central logging system
- [ ] Database backups automated

### Client Security
- [ ] Unique API key per client (not shared)
- [ ] API key stored in environment variable or secure credential store
- [ ] Config file permissions restricted (Administrators only)
- [ ] TLS verification enabled (`tls_verify: true`)
- [ ] Client runs as Windows Service with least privilege
- [ ] Retry and caching enabled for resilience

### Operational Security
- [ ] API key rotation schedule defined (recommend 90 days)
- [ ] Monitoring alerts configured for:
  - Failed authentication attempts
  - Submission anomalies
  - Offline clients
- [ ] Incident response procedure documented
- [ ] Key revocation process tested

---

## ⚡ Common Commands

### Server Operations

```bash
# Check authentication status
docker-compose logs compliance-server | grep "Authentication"

# List all API keys
curl -X GET http://localhost:8080/api/v1/apikeys \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Deactivate compromised key
curl -X POST http://localhost:8080/api/v1/apikeys/toggle \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"id":5,"active":false}'

# Check recent submissions
curl -X GET http://localhost:8080/api/v1/clients \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Client Operations

```bash
# Test client connection
./compliance-client.exe --config client.yaml --once

# View client logs
tail -f /var/log/compliance-client.log

# Check cached submissions (offline queue)
ls cache/submissions/
```

### Database Queries

```bash
# Connect to database
sqlite3 docker/data/compliance.db

# List active API keys
SELECT name, key_prefix, created_at, last_used, is_active
FROM api_keys
WHERE is_active = 1;

# Find stale keys (not used in 30 days)
SELECT name, key_prefix,
  julianday('now') - julianday(last_used) as days_unused
FROM api_keys
WHERE days_unused > 30;

# List all clients and last contact
SELECT client_id, hostname, last_seen
FROM clients
ORDER BY last_seen DESC;
```

---

## 🔥 Emergency Procedures

### Compromised API Key

```bash
# 1. Immediately deactivate the key
curl -X POST http://localhost:8080/api/v1/apikeys/toggle \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"id":COMPROMISED_KEY_ID,"active":false}'

# 2. Generate replacement key
curl -X POST http://localhost:8080/api/v1/apikeys/generate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"replacement-for-server-01"}'

# 3. Update affected client
# 4. Delete old key after 24 hour grace period
curl -X POST http://localhost:8080/api/v1/apikeys/delete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"id":COMPROMISED_KEY_ID}'
```

### Lost Admin Password

```bash
# Reset via database (server must be stopped)
cd docker
docker-compose down

# Connect to database
sqlite3 data/compliance.db

# Hash new password (use bcrypt hash generator)
# Then update directly:
UPDATE users
SET password_hash = '$2a$10$NEW_BCRYPT_HASH_HERE'
WHERE username = 'admin';

# Restart server
docker-compose up -d
```

### Accidental AUTH_REQUIRE_KEY=false

```bash
# If you accidentally deployed with auth disabled:

# 1. IMMEDIATELY update configuration
# Edit docker-compose.yml or .env:
AUTH_REQUIRE_KEY=true

# 2. Restart server
docker-compose restart

# 3. Review logs for unauthorized submissions
docker-compose logs compliance-server | grep "Submission accepted" | \
  grep -v "YOUR-KNOWN-API-KEYS"

# 4. Delete suspicious submissions if found
sqlite3 data/compliance.db
DELETE FROM submissions WHERE client_id IN ('suspicious-id-1', 'suspicious-id-2');
```

---

## 📊 Monitoring Queries

### Daily Health Check
```sql
-- API keys created in last 7 days
SELECT name, created_by, created_at
FROM api_keys
WHERE created_at > datetime('now', '-7 days');

-- Submissions in last 24 hours
SELECT COUNT(*) as submission_count,
       COUNT(DISTINCT client_id) as unique_clients
FROM submissions
WHERE timestamp > datetime('now', '-1 day');

-- Failed submissions (indicates auth or network issues)
SELECT client_id, COUNT(*) as failure_count
FROM (
  SELECT s.client_id
  FROM submissions s
  WHERE s.timestamp > datetime('now', '-1 day')
    AND s.compliance_score < 0.5  -- Anomalously low
)
GROUP BY client_id;
```

### Security Audit
```sql
-- Keys never used (potential security risk)
SELECT name, created_at,
  julianday('now') - julianday(created_at) as days_old
FROM api_keys
WHERE last_used IS NULL
  AND is_active = 1;

-- Multiple clients with same IP (key sharing detection)
SELECT remote_ip, COUNT(DISTINCT client_id) as client_count
FROM (
  SELECT client_id, remote_ip
  FROM submissions
  WHERE timestamp > datetime('now', '-7 days')
)
GROUP BY remote_ip
HAVING client_count > 1;
```

---

## 📞 Getting Help

- **Full Security Guide**: See `docs/security/CLIENT_SECURITY.md`
- **Architecture Details**: See `docs/developer-guide/ARCHITECTURE.md`
- **API Reference**: See `docs/reference/API.md`

**Security Issues**: Report immediately to security team
