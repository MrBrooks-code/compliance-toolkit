# Compliance Toolkit Server

REST API server for receiving and storing compliance submissions from clients.

## Quick Start

### 1. Generate Configuration

```bash
.\compliance-server.exe --generate-config
```

This creates `server.yaml` with default settings.

### 2. Generate Self-Signed Certificates (Testing Only)

```bash
mkdir certs
cd certs
openssl req -x509 -newkey rsa:2048 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"
```

**For production:** Use proper SSL certificates from a Certificate Authority.

### 3. Configure API Keys

Edit `server.yaml` and add your API keys:

```yaml
auth:
  enabled: true
  require_key: true
  api_keys:
    - "your-secure-api-key-here"
```

### 4. Start the Server

```bash
.\compliance-server.exe --config server.yaml
```

The server will start on `https://0.0.0.0:8443` by default.

### 5. Test the Server

Run the test script:

```powershell
.\test-server.ps1
```

Or manually test the health endpoint:

```powershell
curl -k https://localhost:8443/api/v1/health
```

## API Endpoints

### Public Endpoints

- `GET /` - Server information
- `GET /api/v1/health` - Health check (no auth required)

### Protected Endpoints (Require API Key)

- `POST /api/v1/compliance/submit` - Submit compliance report
- `POST /api/v1/clients/register` - Register a new client
- `GET /api/v1/compliance/status/{submission_id}` - Get submission status
- `GET /api/v1/clients` - List all registered clients
- `GET /api/v1/dashboard/summary` - Dashboard summary data

### Dashboard

- `GET /dashboard` - Web dashboard (coming in Phase 2.3)

## Authentication

All protected endpoints require an API key in the `Authorization` header:

```
Authorization: Bearer your-api-key-here
```

Example with curl:

```bash
curl -k -H "Authorization: Bearer test-api-key-12345" \
  https://localhost:8443/api/v1/clients
```

## Database

The server uses SQLite by default. The database file is created at:

```
data/compliance.db
```

### Schema

- **clients** - Registered clients with system information
- **submissions** - Compliance report submissions

## Configuration Reference

```yaml
server:
  host: "0.0.0.0"       # Bind address
  port: 8443            # HTTPS port
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

database:
  type: "sqlite"
  path: "data/compliance.db"

auth:
  enabled: true
  require_key: true
  api_keys:
    - "key1"
    - "key2"

dashboard:
  enabled: true
  path: "/dashboard"

logging:
  level: "info"
  format: "text"
  output_path: "stdout"
```

## Connecting Clients

Update your client configuration to point to this server:

```yaml
# client.yaml
server:
  url: "https://your-server-address:8443"
  api_key: "your-api-key-here"
  tls_verify: true  # Set to false for self-signed certs (testing only)
```

Then run the client:

```bash
.\compliance-client.exe --config client.yaml --once
```

## Production Deployment

### 1. Use Proper SSL Certificates

Replace self-signed certificates with certificates from a trusted CA.

### 2. Secure API Keys

- Use strong, random API keys (32+ characters)
- Rotate keys regularly
- Store keys securely (environment variables, secrets management)

### 3. Firewall Configuration

Open port 8443 (or your configured port) in your firewall:

```powershell
New-NetFirewallRule -DisplayName "Compliance Server" -Direction Inbound -Protocol TCP -LocalPort 8443 -Action Allow
```

### 4. Run as Windows Service

(Coming in future update)

### 5. Enable Logging to File

```yaml
logging:
  output_path: "C:\\ComplianceServer\\logs\\server.log"
```

### 6. Backup Database

Regularly backup `data/compliance.db`:

```powershell
# Backup script
$date = Get-Date -Format "yyyyMMdd_HHmmss"
Copy-Item "data\compliance.db" "backups\compliance_$date.db"
```

## Monitoring

### Health Check

Monitor server health:

```bash
curl -k https://localhost:8443/api/v1/health
```

Expected response:

```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### Logs

Monitor server logs for:

- Failed authentication attempts
- Database errors
- Submission failures

### Database Queries

Check submission counts:

```sql
sqlite3 data/compliance.db "SELECT COUNT(*) FROM submissions;"
```

Check client status:

```sql
sqlite3 data/compliance.db "SELECT client_id, hostname, last_seen, status FROM clients;"
```

## Troubleshooting

### Server Won't Start

**Error:** "Binary was compiled with 'CGO_ENABLED=0'"

**Solution:** Rebuild with CGO enabled:

```bash
set CGO_ENABLED=1
go build -o compliance-server.exe
```

### Certificate Errors

**Error:** "tls: failed to find any PEM data"

**Solution:** Ensure certificates exist:

```bash
ls certs/
# Should show server.crt and server.key
```

### Authentication Failures

**Error:** "Invalid API key"

**Solution:** Check API key in client config matches server config:

```yaml
# server.yaml
auth:
  api_keys:
    - "test-api-key-12345"

# client.yaml
server:
  api_key: "test-api-key-12345"
```

### Database Locked

**Error:** "database is locked"

**Solution:** Ensure only one server instance is running:

```powershell
Get-Process | Where-Object {$_.Name -eq "compliance-server"}
```

## Development

### Build

```bash
cd cmd/compliance-server
set CGO_ENABLED=1
go build -o compliance-server.exe
```

### Run Tests

```powershell
.\test-server.ps1
```

### View Database

```bash
sqlite3 data/compliance.db
.tables
.schema submissions
SELECT * FROM clients;
```

## Next Steps

- Phase 2.3: Web Dashboard UI
- Phase 2.4: Advanced analytics and reporting
- Phase 2.5: Multi-tenancy support
- Phase 2.6: PostgreSQL support

## Support

See main project documentation in `docs/` directory.
