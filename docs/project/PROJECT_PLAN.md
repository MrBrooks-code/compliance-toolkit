# Compliance Toolkit - Client-Server Project Plan

## Executive Summary

Transform Compliance Toolkit from a standalone tool into a distributed client-server system for centralized compliance monitoring across multiple Windows machines.

**Timeline**: 6-8 weeks (20-30 hours total effort)
**Approach**: Incremental development with working deliverables at each phase
**Strategy**: Build client first (immediate value), then add server capabilities

---

## Phase 1: Client Agent Development (Week 1-2, 6-8 hours)

### Objective
Create a production-ready Windows agent that can run compliance scans and report results locally or to a server.

### Deliverables

#### 1.1 Core Client Executable (`cmd/compliance-client/`)
**Duration**: 2-3 hours

**Features**:
- [x] Shared API library (`pkg/api/types.go`, `pkg/api/client.go`)
- [ ] Main client executable
- [ ] Configuration file support (YAML)
- [ ] Command-line interface (flags)
- [ ] Local report generation (standalone mode)
- [ ] Server submission (when available)

**Files to Create**:
```
cmd/compliance-client/
├── main.go              - Entry point
├── config.go            - Configuration management
├── runner.go            - Report execution logic
└── submitter.go         - Server submission logic
```

**Acceptance Criteria**:
- ✅ Client runs all existing reports successfully
- ✅ Generates reports locally in standalone mode
- ✅ Configuration via YAML and environment variables
- ✅ Proper error handling and logging
- ✅ Returns appropriate exit codes

**Testing**:
```bash
# Standalone mode
compliance-client --config client.yaml --report NIST_800_171_compliance.json

# Server mode (will add later)
compliance-client --server https://compliance.company.com:8443 --api-key xxx
```

---

#### 1.2 Scheduling & Automation
**Duration**: 2 hours

**Features**:
- [ ] Built-in scheduler (cron-like syntax)
- [ ] Windows Task Scheduler integration
- [ ] Configurable execution schedule
- [ ] Multiple report support

**Configuration**:
```yaml
schedule:
  enabled: true
  cron: "0 2 * * *"  # Daily at 2 AM
  reports:
    - NIST_800_171_compliance.json
    - FIPS_140_2_compliance.json
```

**Files to Create**:
```
pkg/scheduler/
├── scheduler.go        - Cron scheduler
└── windows.go          - Windows Task integration
```

**Acceptance Criteria**:
- ✅ Client runs on schedule without manual intervention
- ✅ Configurable execution times
- ✅ Multiple reports can be scheduled
- ✅ Logs execution history

---

#### 1.3 Retry Logic & Local Caching
**Duration**: 2 hours

**Features**:
- [ ] Exponential backoff for server failures
- [ ] Local cache for submissions when server unavailable
- [ ] Automatic retry on server recovery
- [ ] Cache size limits and rotation

**Configuration**:
```yaml
retry:
  max_attempts: 3
  initial_backoff: 30s
  max_backoff: 5m
  backoff_multiplier: 2

cache:
  enabled: true
  max_size_mb: 100
  max_age: 7d
  path: "C:/ProgramData/ComplianceToolkit/cache"
```

**Files to Create**:
```
pkg/cache/
├── cache.go           - Local submission cache
└── retry.go           - Retry logic with backoff
```

**Acceptance Criteria**:
- ✅ Client survives server outages
- ✅ Cached submissions sent when server recovers
- ✅ Cache doesn't grow unbounded
- ✅ Configurable retry behavior

---

#### 1.4 System Information Collection
**Duration**: 1 hour

**Features**:
- [ ] OS version and build number
- [ ] System architecture
- [ ] Domain membership
- [ ] Network information (IP, MAC)
- [ ] Last boot time

**Files to Create**:
```
pkg/sysinfo/
└── windows.go         - Windows system info collector
```

**Acceptance Criteria**:
- ✅ Accurate system information collected
- ✅ Works on all Windows versions (10, 11, Server)
- ✅ Handles domain and workgroup systems

---

#### 1.5 Windows Service Installation
**Duration**: 1-2 hours

**Features**:
- [ ] Install as Windows Service
- [ ] Service lifecycle management (start, stop, restart)
- [ ] Service recovery on failure
- [ ] Event log integration

**Files to Create**:
```
cmd/compliance-client/
├── service.go         - Windows service wrapper
└── install.go         - Service installation logic
```

**Commands**:
```bash
# Install service
compliance-client install --config "C:\Program Files\ComplianceToolkit\client.yaml"

# Uninstall service
compliance-client uninstall

# Run as service
compliance-client service
```

**Acceptance Criteria**:
- ✅ Installs as Windows Service
- ✅ Starts automatically on boot
- ✅ Recovers from failures
- ✅ Logs to Windows Event Log

---

### Phase 1 Milestones

**Week 1 Checkpoint**:
- [ ] Basic client executable working
- [ ] Local report generation functional
- [ ] Configuration system in place

**Week 2 Checkpoint**:
- [ ] Scheduling implemented
- [ ] Service installation working
- [ ] Retry and caching complete

**Phase 1 Deliverable**:
A production-ready Windows agent that:
- Runs compliance scans on schedule
- Generates local reports
- Operates as a Windows Service
- Ready to submit to server (when available)

---

## Phase 2: Server Development - Core (Week 3-4, 8-10 hours)

### Objective
Build a central server that receives, stores, and serves compliance data.

### Deliverables

#### 2.1 REST API Server (`cmd/compliance-server/`)
**Duration**: 3-4 hours

**Features**:
- [ ] HTTP/HTTPS server (Gin framework)
- [ ] API endpoints for submission
- [ ] Health check endpoint
- [ ] API key authentication
- [ ] Request validation
- [ ] Structured logging

**Files to Create**:
```
cmd/compliance-server/
├── main.go                    - Entry point
├── server.go                  - HTTP server setup
├── routes.go                  - Route definitions
└── handlers/
    ├── submit.go              - Submission handler
    ├── status.go              - Status query handler
    ├── clients.go             - Client management
    └── health.go              - Health check
```

**API Endpoints**:
```
POST   /api/v1/compliance/submit      - Submit compliance report
GET    /api/v1/compliance/status/:id  - Get submission status
GET    /api/v1/clients                - List registered clients
GET    /api/v1/clients/:id            - Get client details
GET    /api/v1/clients/:id/history    - Get client history
POST   /api/v1/clients/register       - Register new client
GET    /api/v1/health                 - Health check
GET    /api/v1/metrics                - Prometheus metrics
```

**Acceptance Criteria**:
- ✅ Server starts and listens on configured port
- ✅ API endpoints respond correctly
- ✅ Request validation prevents bad data
- ✅ Proper HTTP status codes
- ✅ Comprehensive error responses

---

#### 2.2 Database Integration
**Duration**: 3 hours

**Features**:
- [ ] SQLite for simple deployments
- [ ] PostgreSQL support for production
- [ ] Database migrations
- [ ] Query builders and models
- [ ] Connection pooling

**Files to Create**:
```
pkg/storage/
├── storage.go               - Storage interface
├── sqlite.go                - SQLite implementation
├── postgres.go              - PostgreSQL implementation
├── models.go                - Data models
└── migrations/
    ├── 001_initial.sql      - Initial schema
    ├── 002_indices.sql      - Performance indices
    └── 003_audit.sql        - Audit tables
```

**Database Schema**:
```sql
-- See CLIENT_SERVER_DESIGN.md for complete schema
CREATE TABLE clients (...);
CREATE TABLE submissions (...);
CREATE TABLE query_results (...);
CREATE TABLE audit_events (...);
```

**Acceptance Criteria**:
- ✅ Database schema created via migrations
- ✅ CRUD operations for all entities
- ✅ Efficient queries with proper indices
- ✅ Transaction support
- ✅ Connection pooling configured

---

#### 2.3 Authentication & Authorization
**Duration**: 2 hours

**Features**:
- [ ] API key authentication
- [ ] Key generation and management
- [ ] Rate limiting per key
- [ ] Audit logging for auth events

**Files to Create**:
```
pkg/auth/
├── auth.go                  - Authentication middleware
├── apikey.go                - API key management
└── ratelimit.go             - Rate limiting
```

**Configuration**:
```yaml
auth:
  type: "api_key"
  api_keys_file: "/etc/compliance/api_keys.json"

rate_limit:
  enabled: true
  requests_per_minute: 60
  burst: 10
```

**Acceptance Criteria**:
- ✅ Requests without valid API key rejected
- ✅ API keys can be generated and revoked
- ✅ Rate limiting prevents abuse
- ✅ All auth events logged

---

#### 2.4 Testing & Documentation
**Duration**: 1 hour

**Features**:
- [ ] API integration tests
- [ ] Database tests
- [ ] Load tests (100+ concurrent clients)
- [ ] API documentation (OpenAPI/Swagger)

**Files to Create**:
```
cmd/compliance-server/
└── server_test.go           - Integration tests

docs/api/
└── openapi.yaml             - API specification
```

**Acceptance Criteria**:
- ✅ 90%+ test coverage
- ✅ All endpoints tested
- ✅ Load tests pass
- ✅ API documentation complete

---

### Phase 2 Milestones

**Week 3 Checkpoint**:
- [ ] Basic server running
- [ ] Database integration complete
- [ ] Client can submit to server

**Week 4 Checkpoint**:
- [ ] Authentication working
- [ ] Tests passing
- [ ] Documentation complete

**Phase 2 Deliverable**:
A functional server that:
- Receives compliance submissions
- Stores data in database
- Provides API access to historical data
- Authenticates clients
- Ready for dashboard integration

---

## Phase 3: Dashboard & Reporting (Week 5-6, 6-8 hours)

### Objective
Create a web-based dashboard for viewing and analyzing compliance data.

### Deliverables

#### 3.1 Web UI Framework
**Duration**: 2 hours

**Features**:
- [ ] HTML templates (Go templates or Templ)
- [ ] HTMX for dynamic updates
- [ ] CSS framework (TailwindCSS)
- [ ] Responsive design

**Files to Create**:
```
cmd/compliance-server/
├── web/
│   ├── templates/
│   │   ├── base.html          - Base layout
│   │   ├── dashboard.html     - Main dashboard
│   │   ├── client.html        - Client details
│   │   ├── submission.html    - Submission view
│   │   └── components/        - Reusable components
│   └── static/
│       ├── css/
│       └── js/
└── handlers/
    └── web/
        ├── dashboard.go       - Dashboard handler
        ├── client.go          - Client views
        └── submission.go      - Submission views
```

**Routes**:
```
GET    /                       - Redirect to dashboard
GET    /dashboard              - Main dashboard
GET    /clients                - Client list
GET    /clients/:id            - Client details
GET    /submissions/:id        - Submission details
GET    /search                 - Search interface
```

**Acceptance Criteria**:
- ✅ Dashboard loads in <2 seconds
- ✅ Responsive on desktop and mobile
- ✅ Clean, professional UI
- ✅ Accessible (WCAG 2.1 AA)

---

#### 3.2 Dashboard Features
**Duration**: 3 hours

**Features**:
- [ ] Overview statistics (total clients, compliance rate)
- [ ] Recent submissions list
- [ ] Compliance trends chart
- [ ] Alert notifications
- [ ] Client status overview

**Components**:
- Summary cards (total clients, compliant, non-compliant)
- Time-series chart (compliance over time)
- Recent submissions table
- Failed checks summary
- Client status grid

**Libraries**:
- Charts: Chart.js or Apache ECharts
- Tables: DataTables or custom HTMX
- Icons: Heroicons or Lucide

**Acceptance Criteria**:
- ✅ Real-time updates via HTMX
- ✅ Charts render correctly
- ✅ Data refreshes automatically
- ✅ Drill-down navigation works

---

#### 3.3 Search & Filtering
**Duration**: 1-2 hours

**Features**:
- [ ] Full-text search across submissions
- [ ] Filter by date range
- [ ] Filter by status (compliant/non-compliant)
- [ ] Filter by client
- [ ] Filter by check name
- [ ] Saved searches

**Files to Create**:
```
pkg/search/
├── search.go               - Search engine
└── filters.go              - Filter logic

cmd/compliance-server/handlers/web/
└── search.go               - Search handler
```

**Acceptance Criteria**:
- ✅ Search returns results in <1 second
- ✅ Filters combine correctly (AND/OR)
- ✅ Pagination for large result sets
- ✅ Export results to CSV

---

#### 3.4 Reporting & Exports
**Duration**: 1-2 hours

**Features**:
- [ ] Generate PDF reports
- [ ] Export to CSV/Excel
- [ ] Scheduled email reports
- [ ] Custom report templates

**Files to Create**:
```
pkg/export/
├── pdf.go                  - PDF generation
├── csv.go                  - CSV export
└── excel.go                - Excel export

pkg/reports/
└── scheduler.go            - Report scheduling
```

**Acceptance Criteria**:
- ✅ PDF reports match HTML design
- ✅ CSV exports include all data
- ✅ Scheduled reports sent on time
- ✅ Custom templates work

---

### Phase 3 Milestones

**Week 5 Checkpoint**:
- [ ] Basic dashboard functional
- [ ] Client and submission views working

**Week 6 Checkpoint**:
- [ ] Search and filtering complete
- [ ] Export features working
- [ ] All UI features tested

**Phase 3 Deliverable**:
A complete web dashboard with:
- Overview of compliance status
- Drill-down to individual clients
- Search and filter capabilities
- Export and reporting features

---

## Phase 4: Advanced Features (Week 7-8, 4-6 hours)

### Objective
Add enterprise-grade features for production deployment.

### Deliverables

#### 4.1 Alerting System
**Duration**: 2 hours

**Features**:
- [ ] Configurable alert rules
- [ ] Email notifications
- [ ] Slack/Teams webhooks
- [ ] Alert history and acknowledgment
- [ ] Escalation policies

**Alert Types**:
- Compliance drift (status changed from compliant to non-compliant)
- New failed checks
- Client offline (no submission in X days)
- Critical checks failed
- Multiple failures across clients

**Files to Create**:
```
pkg/alerts/
├── engine.go               - Alert evaluation engine
├── rules.go                - Alert rule definitions
├── notifiers/
│   ├── email.go            - Email notifications
│   ├── slack.go            - Slack integration
│   └── webhook.go          - Generic webhooks
└── escalation.go           - Escalation logic
```

**Configuration**:
```yaml
alerts:
  enabled: true

  rules:
    - name: "Critical Compliance Failure"
      condition: "failed_checks > 5"
      severity: "critical"
      notifiers: ["email", "slack"]

  email:
    smtp_host: "smtp.company.com"
    smtp_port: 587
    from: "compliance@company.com"
    to: ["security-team@company.com"]

  slack:
    webhook_url: "https://hooks.slack.com/..."
    channel: "#compliance-alerts"
```

**Acceptance Criteria**:
- ✅ Alerts triggered correctly
- ✅ Notifications delivered reliably
- ✅ Alert history viewable in UI
- ✅ Alerts can be acknowledged

---

#### 4.2 Metrics & Monitoring
**Duration**: 1-2 hours

**Features**:
- [ ] Prometheus metrics endpoint
- [ ] Application metrics (requests, latency, errors)
- [ ] Business metrics (compliance rate, submissions)
- [ ] Health checks (liveness, readiness)
- [ ] Grafana dashboards

**Metrics**:
```
# Application metrics
http_requests_total
http_request_duration_seconds
http_requests_in_flight

# Business metrics
compliance_submissions_total
compliance_clients_total
compliance_score_gauge
compliance_failed_checks_total

# Database metrics
db_connections_total
db_query_duration_seconds
```

**Files to Create**:
```
pkg/metrics/
├── prometheus.go           - Prometheus metrics
└── health.go               - Health checks

monitoring/
└── grafana/
    └── dashboards/
        └── compliance.json - Grafana dashboard
```

**Acceptance Criteria**:
- ✅ Metrics endpoint works
- ✅ Grafana dashboard visualizes data
- ✅ Health checks accurate
- ✅ Metrics help diagnose issues

---

#### 4.3 Deployment & Packaging
**Duration**: 1-2 hours

**Features**:
- [ ] Docker images for server
- [ ] Docker Compose for local testing
- [ ] Kubernetes manifests
- [ ] Helm chart
- [ ] MSI installer for Windows client
- [ ] Systemd service for Linux server

**Files to Create**:
```
docker/
├── Dockerfile.client       - Client image
├── Dockerfile.server       - Server image
└── docker-compose.yml      - Local development

k8s/
├── deployment.yaml         - K8s deployment
├── service.yaml            - K8s service
├── ingress.yaml            - K8s ingress
└── configmap.yaml          - Configuration

helm/
└── compliance-toolkit/
    ├── Chart.yaml
    ├── values.yaml
    └── templates/

installers/
├── windows/
│   └── client.wxs          - WiX installer
└── linux/
    └── compliance.service  - Systemd unit
```

**Acceptance Criteria**:
- ✅ Docker images build successfully
- ✅ Docker Compose works locally
- ✅ Kubernetes deploys correctly
- ✅ MSI installer installs client
- ✅ All packaging tested

---

#### 4.4 Documentation
**Duration**: 1 hour

**Features**:
- [ ] Installation guides
- [ ] Configuration reference
- [ ] API documentation
- [ ] Troubleshooting guide
- [ ] Architecture diagrams

**Files to Create**:
```
docs/
├── installation/
│   ├── client.md           - Client installation
│   ├── server.md           - Server installation
│   └── docker.md           - Docker deployment
├── configuration/
│   ├── client-config.md    - Client config reference
│   └── server-config.md    - Server config reference
├── api/
│   └── reference.md        - API documentation
├── operations/
│   ├── monitoring.md       - Monitoring guide
│   ├── backup.md           - Backup procedures
│   └── troubleshooting.md  - Common issues
└── architecture/
    └── diagrams/           - Architecture diagrams
```

**Acceptance Criteria**:
- ✅ All deployment methods documented
- ✅ Configuration examples provided
- ✅ API fully documented
- ✅ Troubleshooting covers common issues

---

### Phase 4 Milestones

**Week 7 Checkpoint**:
- [ ] Alerting system functional
- [ ] Metrics and monitoring working

**Week 8 Checkpoint**:
- [ ] Deployment artifacts ready
- [ ] Documentation complete
- [ ] System production-ready

**Phase 4 Deliverable**:
Production-ready system with:
- Automated alerting
- Comprehensive monitoring
- Multiple deployment options
- Complete documentation

---

## Success Criteria

### Phase 1 Success (Client)
- [x] Client builds and runs on Windows
- [ ] Reports execute correctly
- [ ] Service installation works
- [ ] Scheduling functional
- [ ] Retry and caching working

### Phase 2 Success (Server Core)
- [ ] Server receives submissions
- [ ] Data stored in database
- [ ] API endpoints functional
- [ ] Authentication working
- [ ] Tests passing (90%+ coverage)

### Phase 3 Success (Dashboard)
- [ ] Dashboard loads and displays data
- [ ] Search and filtering works
- [ ] Export features functional
- [ ] UI responsive and accessible

### Phase 4 Success (Production)
- [ ] Alerting operational
- [ ] Metrics and monitoring configured
- [ ] Deployment artifacts tested
- [ ] Documentation complete

### Overall Success Criteria
- [ ] 100+ clients can report to single server
- [ ] API p95 latency <200ms
- [ ] Dashboard loads in <2s
- [ ] Zero data loss during network failures
- [ ] 99.9% uptime for server
- [ ] Complete audit trail for compliance

---

## Risk Management

### Technical Risks

**Risk**: Client compatibility issues across Windows versions
- **Mitigation**: Test on Windows 10, 11, Server 2019, Server 2022
- **Contingency**: Maintain compatibility matrix, add version checks

**Risk**: Database performance with 1000+ clients
- **Mitigation**: Proper indexing, query optimization, connection pooling
- **Contingency**: Add caching layer (Redis), database sharding

**Risk**: Network failures causing data loss
- **Mitigation**: Retry logic, local caching, idempotent APIs
- **Contingency**: Manual data recovery procedures

**Risk**: Security vulnerabilities in API
- **Mitigation**: Input validation, rate limiting, audit logging
- **Contingency**: Security incident response plan

### Project Risks

**Risk**: Scope creep extending timeline
- **Mitigation**: Strict phase boundaries, MVP-first approach
- **Contingency**: Defer features to future phases

**Risk**: Testing insufficient for production
- **Mitigation**: Comprehensive test suite, load testing
- **Contingency**: Extended beta testing period

---

## Resource Requirements

### Development Environment
- Windows 10/11 for client development
- Linux for server development (or WSL2)
- Go 1.24+
- SQLite for local testing
- PostgreSQL for production testing
- Docker Desktop
- Visual Studio Code or GoLand

### Infrastructure (Production)
- **Client**: Any Windows machine (10+, Server 2019+)
  - 50MB disk, 50MB RAM, <5% CPU

- **Server**: Linux VM or container
  - 2 CPU, 4GB RAM, 20GB disk (small deployment)
  - 4+ CPU, 8GB+ RAM, 100GB+ disk (large deployment)

- **Database**: PostgreSQL 14+
  - 2 CPU, 4GB RAM, storage depends on retention

### External Dependencies
- SMTP server (for email alerts)
- Slack/Teams (for notifications, optional)
- Prometheus + Grafana (for monitoring, optional)

---

## Timeline Summary

```
Week 1-2:  Phase 1 - Client Development
Week 3-4:  Phase 2 - Server Core
Week 5-6:  Phase 3 - Dashboard
Week 7-8:  Phase 4 - Production Features

Total: 6-8 weeks, 20-30 hours
```

### Critical Path
1. Client executable → Service → Scheduling (Phase 1)
2. Server API → Database → Auth (Phase 2)
3. Dashboard UI → Search → Reports (Phase 3)
4. Alerts → Monitoring → Deployment (Phase 4)

### Optional/Parallel Work
- Documentation (ongoing)
- Testing (ongoing)
- Performance optimization (Phase 3-4)
- Additional features (post-Phase 4)

---

## Post-Launch Roadmap

### Version 2.0 Features
- Multi-tenancy support
- Custom report builder (UI)
- Machine learning for anomaly detection
- Integration with SIEM systems
- Mobile app for dashboard

### Version 3.0 Features
- Cross-platform support (Linux, macOS)
- Configuration management (push configs from server)
- Remote remediation triggers
- Compliance templates marketplace
- SaaS offering

---

## Appendix

### Tech Stack

**Client**:
- Language: Go 1.24+
- Windows API: golang.org/x/sys/windows
- Service: kardianos/service
- Config: spf13/viper
- Scheduling: robfig/cron

**Server**:
- Framework: gin-gonic/gin
- Database: gorm.io/gorm
- SQLite: gorm.io/driver/sqlite
- PostgreSQL: gorm.io/driver/postgres
- Auth: API keys (custom)
- Metrics: prometheus/client_golang

**Dashboard**:
- Templates: Go templates or a-h/templ
- Frontend: HTMX + TailwindCSS
- Charts: Chart.js or ECharts
- Icons: Heroicons

**DevOps**:
- Docker: Multi-stage builds
- Kubernetes: Standard manifests
- CI/CD: GitHub Actions
- Testing: testify, gomock

### Key Dependencies

```go
// Client
github.com/spf13/viper
github.com/kardianos/service
github.com/robfig/cron/v3
golang.org/x/sys/windows

// Server
github.com/gin-gonic/gin
gorm.io/gorm
gorm.io/driver/sqlite
gorm.io/driver/postgres
github.com/prometheus/client_golang

// Shared
github.com/google/uuid
github.com/stretchr/testify
```

### References
- Architecture: `docs/architecture/CLIENT_SERVER_DESIGN.md`
- API Spec: `docs/api/openapi.yaml` (to be created)
- User Guide: `docs/user-guide/` (existing)
- Developer Guide: `docs/developer-guide/` (existing)

---

## Sign-off

This project plan represents the roadmap for transforming Compliance Toolkit into a distributed compliance monitoring system. Each phase delivers working functionality and can be deployed independently.

**Recommended Approach**: Start with Phase 1 (client) to deliver immediate value, then incrementally add server capabilities as needed.

**Questions or Changes**: Update this document as requirements evolve. Track progress in project management tool or GitHub issues.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-05
**Status**: Ready for implementation
