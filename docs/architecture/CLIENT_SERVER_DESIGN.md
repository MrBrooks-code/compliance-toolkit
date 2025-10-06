# Compliance Toolkit - Client-Server Architecture

## Overview

This document describes the client-server architecture for Compliance Toolkit, enabling centralized compliance monitoring across multiple Windows systems.

## Goals

1. **Centralized Monitoring**: Collect compliance data from multiple Windows machines in one location
2. **Historical Tracking**: Store compliance history to track trends and changes over time
3. **Scalability**: Support hundreds of clients reporting to a single server
4. **Security**: Secure communication with TLS, authentication, and audit logging
5. **Ease of Deployment**: Simple client installation and configuration

## Architecture Components

### 1. Compliance Client (`cmd/compliance-client/`)

**Purpose**: Lightweight agent that runs on Windows machines to collect and report compliance data

**Features**:
- Reuses existing `pkg/` library for registry scanning
- Scheduled execution (e.g., daily at 2 AM)
- On-demand execution via API
- Local caching when server is unavailable
- Auto-retry with exponential backoff
- Minimal resource usage (<50MB RAM, <5% CPU)

**Configuration**:
```yaml
client:
  hostname: "ws-accounting-01"
  server_url: "https://compliance.company.com:8443"
  api_key: "${COMPLIANCE_API_KEY}"

  schedule:
    enabled: true
    cron: "0 2 * * *"  # Daily at 2 AM

  reports:
    - NIST_800_171_compliance.json
    - FIPS_140_2_compliance.json

  retry:
    max_attempts: 3
    backoff: "exponential"

  cache:
    enabled: true
    max_age: "24h"
```

**Build**:
```bash
go build -o compliance-client.exe ./cmd/compliance-client
```

### 2. Compliance Server (`cmd/compliance-server/`)

**Purpose**: Central server that receives, stores, and visualizes compliance data

**Features**:
- REST API for client submissions
- SQLite/PostgreSQL database for storage
- Web UI dashboard
- Multi-tenant support (optional)
- Alert rules and notifications
- Historical trend analysis
- Compliance drift detection

**API Endpoints**:
```
POST   /api/v1/compliance/submit      - Submit compliance report
GET    /api/v1/compliance/status/:id  - Get submission status
GET    /api/v1/clients                - List registered clients
GET    /api/v1/clients/:id/history    - Get client compliance history
POST   /api/v1/auth/token             - Get API token
GET    /api/v1/dashboard/summary      - Dashboard summary
```

**Configuration**:
```yaml
server:
  host: "0.0.0.0"
  port: 8443
  tls:
    enabled: true
    cert: "/etc/compliance/server.crt"
    key: "/etc/compliance/server.key"

database:
  type: "postgres"  # or "sqlite"
  host: "localhost"
  port: 5432
  name: "compliance"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"

auth:
  type: "api_key"  # or "oauth2", "mtls"
  api_keys_file: "/etc/compliance/api_keys.json"

alerts:
  enabled: true
  smtp:
    host: "smtp.company.com"
    port: 587
    from: "compliance@company.com"
```

**Build**:
```bash
go build -o compliance-server ./cmd/compliance-server
```

### 3. Shared API Library (`pkg/api/`)

**Purpose**: Common types and utilities shared between client and server

**Components**:
- `types.go` - DTOs for API communication
- `client.go` - Client SDK for submitting reports
- `auth.go` - Authentication helpers
- `protocol.go` - Protocol version handling

## Data Model

### ComplianceSubmission

```go
type ComplianceSubmission struct {
    SubmissionID  string                `json:"submission_id"`
    ClientID      string                `json:"client_id"`
    Hostname      string                `json:"hostname"`
    Timestamp     time.Time             `json:"timestamp"`
    ReportType    string                `json:"report_type"`
    ReportVersion string                `json:"report_version"`
    Compliance    ComplianceData        `json:"compliance"`
    Evidence      []EvidenceRecord      `json:"evidence"`
    SystemInfo    SystemInfo            `json:"system_info"`
}

type ComplianceData struct {
    OverallStatus  string              `json:"overall_status"`
    TotalChecks    int                 `json:"total_checks"`
    PassedChecks   int                 `json:"passed_checks"`
    FailedChecks   int                 `json:"failed_checks"`
    WarningChecks  int                 `json:"warning_checks"`
    Queries        []QueryResult       `json:"queries"`
}

type QueryResult struct {
    Name          string              `json:"name"`
    Description   string              `json:"description"`
    Status        string              `json:"status"`  // "pass", "fail", "warning", "error"
    Expected      string              `json:"expected"`
    Actual        string              `json:"actual"`
    Message       string              `json:"message,omitempty"`
}

type SystemInfo struct {
    OSVersion     string              `json:"os_version"`
    BuildNumber   string              `json:"build_number"`
    Architecture  string              `json:"architecture"`
    Domain        string              `json:"domain,omitempty"`
    IPAddress     string              `json:"ip_address,omitempty"`
}
```

### Database Schema

```sql
-- Clients table
CREATE TABLE clients (
    id UUID PRIMARY KEY,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    first_seen TIMESTAMP NOT NULL,
    last_seen TIMESTAMP NOT NULL,
    status VARCHAR(50),
    metadata JSONB
);

-- Submissions table
CREATE TABLE submissions (
    id UUID PRIMARY KEY,
    client_id UUID REFERENCES clients(id),
    submission_id VARCHAR(255) UNIQUE NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    report_type VARCHAR(255) NOT NULL,
    report_version VARCHAR(50),
    overall_status VARCHAR(50),
    total_checks INTEGER,
    passed_checks INTEGER,
    failed_checks INTEGER,
    warning_checks INTEGER,
    data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Query results table (for efficient searching)
CREATE TABLE query_results (
    id UUID PRIMARY KEY,
    submission_id UUID REFERENCES submissions(id),
    query_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    expected TEXT,
    actual TEXT,
    INDEX idx_query_status (query_name, status),
    INDEX idx_submission (submission_id)
);

-- Audit events
CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    client_id UUID REFERENCES clients(id),
    user_id VARCHAR(255),
    action VARCHAR(255) NOT NULL,
    details JSONB,
    INDEX idx_timestamp (timestamp),
    INDEX idx_event_type (event_type)
);
```

## Security

### Authentication

**API Key Authentication** (Phase 1):
- Each client has a unique API key
- Keys stored securely on server (hashed with bcrypt)
- Keys passed in `Authorization: Bearer <key>` header

**Mutual TLS** (Phase 2):
- Client certificates for authentication
- Server validates client certificate against CA
- Higher security for regulated environments

### Encryption

- All communication over TLS 1.3
- Client certificates stored in Windows Certificate Store
- API keys stored in Windows Credential Manager

### Authorization

- Role-based access control (RBAC)
- Clients can only submit their own data
- Admins can view all data
- Operators can view but not modify

## Deployment

### Client Deployment

**Windows Service**:
```powershell
# Install as Windows Service
sc.exe create ComplianceClient binPath= "C:\Program Files\ComplianceToolkit\compliance-client.exe" start= auto

# Configure service
sc.exe description ComplianceClient "Compliance Toolkit Client Agent"
sc.exe failure ComplianceClient reset= 86400 actions= restart/60000/restart/120000/restart/300000

# Start service
sc.exe start ComplianceClient
```

**Group Policy Deployment**:
- MSI installer for easy deployment
- Group Policy for configuration
- Automatic updates via WSUS/SCCM

### Server Deployment

**Docker**:
```yaml
version: '3.8'

services:
  compliance-server:
    image: compliance-toolkit/server:latest
    ports:
      - "8443:8443"
    environment:
      - DB_HOST=postgres
      - DB_NAME=compliance
    volumes:
      - ./config:/etc/compliance
      - ./certs:/etc/compliance/certs
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    environment:
      - POSTGRES_DB=compliance
      - POSTGRES_USER=compliance
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

**Kubernetes**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: compliance-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: compliance-server
  template:
    metadata:
      labels:
        app: compliance-server
    spec:
      containers:
      - name: server
        image: compliance-toolkit/server:latest
        ports:
        - containerPort: 8443
        envFrom:
        - secretRef:
            name: compliance-secrets
```

## Web Dashboard

### Features

1. **Overview Dashboard**
   - Total clients registered
   - Compliance status summary (pie chart)
   - Recent submissions timeline
   - Alert notifications

2. **Client Details**
   - Individual client compliance history
   - Trend graphs over time
   - Failed checks detail view
   - System information

3. **Reports**
   - Compliance by report type
   - Compliance by time period
   - Failed checks across all clients
   - Export to PDF/CSV

4. **Search & Filters**
   - Search by hostname, status, check name
   - Filter by date range, report type
   - Saved searches

5. **Alerts**
   - Configure alert rules
   - Email/Slack notifications
   - Compliance drift detection

## API Examples

### Client Submitting Report

```go
package main

import (
    "compliancetoolkit/pkg/api"
)

func main() {
    client := api.NewClient("https://compliance.company.com:8443", "api-key-123")

    submission := &api.ComplianceSubmission{
        ClientID:   "ws-01",
        Hostname:   "WS-ACCOUNTING-01",
        ReportType: "NIST_800_171_compliance",
        // ... compliance data
    }

    resp, err := client.Submit(submission)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Submitted: %s", resp.SubmissionID)
}
```

### Server Receiving Report

```go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func handleSubmit(c *gin.Context) {
    var submission api.ComplianceSubmission
    if err := c.BindJSON(&submission); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate
    if err := submission.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Store in database
    id, err := db.StoreSubmission(&submission)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "storage failed"})
        return
    }

    // Check alerts
    alerts.CheckCompliance(&submission)

    c.JSON(http.StatusOK, gin.H{
        "submission_id": id,
        "status": "accepted",
    })
}
```

## Roadmap

### Phase 1 - MVP (Current Sprint)
- [x] Basic client that submits compliance data
- [x] Server REST API with SQLite
- [x] API key authentication
- [ ] Simple web UI (read-only)

### Phase 2 - Production Ready
- [ ] PostgreSQL support
- [ ] TLS mutual authentication
- [ ] Email alerting
- [ ] Client retry logic with caching
- [ ] Dashboard with filtering

### Phase 3 - Enterprise Features
- [ ] Multi-tenancy
- [ ] RBAC with Active Directory integration
- [ ] Custom report definitions via UI
- [ ] Scheduled reports via email
- [ ] API rate limiting
- [ ] Prometheus metrics export

### Phase 4 - Advanced Analytics
- [ ] Machine learning for anomaly detection
- [ ] Compliance trend prediction
- [ ] Automated remediation suggestions
- [ ] Integration with SIEM systems

## Performance Targets

- **Client**: <5% CPU, <50MB RAM
- **Server**: Support 1000+ clients
- **Database**: Store 1 year of daily submissions efficiently (~365k records)
- **API Response**: <200ms p95 latency
- **Dashboard Load**: <2s page load time

## Alternatives Considered

### Why Not Use Existing Solutions?

1. **SCCM/Intune**: Limited to Microsoft-specific compliance, not extensible
2. **OpenSCAP**: Linux-focused, Windows support limited
3. **Chef InSpec**: Requires Ruby, complex infrastructure
4. **Commercial Solutions**: Expensive, not registry-focused

### Why This Approach?

- Lightweight and Windows-native
- Reuses existing registry scanning library
- Simple deployment (single binary)
- Full control over compliance definitions
- Cost-effective (open source)

## Questions & Decisions

1. **Database Choice**: SQLite for simplicity vs PostgreSQL for scale?
   - **Decision**: Start with SQLite, add PostgreSQL support in Phase 2

2. **Authentication**: API keys vs mTLS?
   - **Decision**: API keys for Phase 1, mTLS option in Phase 2

3. **Protocol**: REST vs gRPC?
   - **Decision**: REST/JSON for simplicity and debugging

4. **Web Framework**: Which Go framework for server?
   - **Decision**: Gin for REST API, Templ for HTML templates

5. **Frontend**: SPA vs Server-side rendering?
   - **Decision**: Server-side with HTMX for simplicity

## Next Steps

1. Create `pkg/api` with shared types
2. Implement basic client in `cmd/compliance-client`
3. Implement basic server in `cmd/compliance-server`
4. Create database schema and migrations
5. Build simple web UI
6. Write deployment documentation
7. Create Docker images
