# Development Guide

**Version:** 1.1.0
**Last Updated:** 2025-01-05

Complete guide for building, testing, and contributing to the Compliance Toolkit.

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Building the Project](#building-the-project)
3. [Project Structure](#project-structure)
4. [Development Workflow](#development-workflow)
5. [Testing](#testing)
6. [Contributing](#contributing)

---

## Getting Started

### Prerequisites

**Required:**
- Go 1.24.0 or later
- Windows OS (for registry access)
- Git

**Optional:**
- VS Code with Go extension
- GoLand IDE
- Windows Terminal

### Clone Repository

```bash
git clone https://github.com/yourorg/compliance-toolkit.git
cd compliance-toolkit
```

### Install Dependencies

```bash
go mod download
```

The project uses minimal dependencies:
- `golang.org/x/sys/windows/registry` - Windows registry access
- Go standard library (embed, html/template, log/slog, etc.)

---

## Building the Project

### Quick Build

```bash
# Build for current platform
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Build with Version Info

```bash
# Build with version and build info
go build -ldflags="-X main.Version=1.1.0 -X main.BuildDate=$(date -u +%Y-%m-%d)" -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Build for Release

```bash
# Build optimized release binary
go build -ldflags="-s -w" -o ComplianceToolkit.exe ./cmd/toolkit.go
```

**Flags explained:**
- `-s` - Omit symbol table
- `-w` - Omit DWARF symbol table
- Result: Smaller binary size

### Clean Build

```bash
# Clean build cache and rebuild
go clean -cache
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Verify Build

```bash
# Check binary info
ComplianceToolkit.exe -h

# List available reports
ComplianceToolkit.exe -list
```

---

## Project Structure

### Directory Layout

```
compliance-toolkit/
├── cmd/
│   └── toolkit.go              # Main entry point
├── pkg/
│   ├── registry.go             # Registry reader core
│   ├── config.go               # Configuration loader
│   ├── htmlreport.go           # HTML report generator
│   ├── menu.go                 # Interactive menu
│   ├── evidence.go             # Evidence logging
│   ├── templatedata.go         # Template data structures
│   └── templates/              # Embedded templates
│       ├── html/               # HTML templates
│       │   ├── base.html
│       │   └── components/
│       │       ├── header.html
│       │       ├── kpi-cards.html
│       │       ├── chart.html
│       │       └── data-table.html
│       └── css/                # CSS templates
│           ├── main.css
│           └── print.css
├── configs/
│   └── reports/                # Report configurations
│       ├── NIST_800_171_compliance.json
│       ├── fips_140_2_compliance.json
│       ├── system_info.json
│       ├── software_inventory.json
│       ├── network_config.json
│       ├── user_settings.json
│       └── performance_diagnostics.json
├── output/
│   ├── reports/                # Generated HTML reports
│   ├── evidence/               # Evidence JSON logs
│   └── logs/                   # Application logs
├── examples/                   # Example scripts
│   ├── scheduled_compliance_scan.ps1
│   └── scheduled_compliance_scan.bat
├── docs/                       # Documentation
│   ├── README.md
│   ├── user-guide/
│   ├── developer-guide/
│   ├── reference/
│   └── PROJECT_STATUS.md
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # Project README
```

### Core Components

| Component | File | Purpose |
|-----------|------|---------|
| **Registry Reader** | `pkg/registry.go` | Windows registry operations |
| **Config Loader** | `pkg/config.go` | JSON report configuration |
| **HTML Reporter** | `pkg/htmlreport.go` | HTML report generation |
| **Evidence Logger** | `pkg/evidence.go` | Compliance evidence logging |
| **Menu System** | `pkg/menu.go` | Interactive CLI menu |
| **Main Application** | `cmd/toolkit.go` | Entry point and orchestration |

---

## Development Workflow

### 1. Make Code Changes

Edit files in `pkg/` or `cmd/`:

```bash
# Edit registry reader
notepad pkg/registry.go

# Edit report template
notepad pkg/templates/html/components/header.html

# Edit CSS
notepad pkg/templates/css/main.css
```

### 2. Build and Test

```bash
# Quick build
go build -o ComplianceToolkit.exe ./cmd/toolkit.go

# Run with test report
ComplianceToolkit.exe -report=system_info.json

# Check output
start output\reports\
```

### 3. Verify Changes

**For code changes:**
```bash
# Run tests
go test ./pkg/...

# Run with verbose output
go test -v ./pkg/...

# Check coverage
go test -cover ./pkg/...
```

**For template changes:**
1. Build: `go build -o ComplianceToolkit.exe ./cmd/toolkit.go`
2. Generate report
3. Open in browser
4. Check changes

### 4. Format Code

```bash
# Format all Go files
go fmt ./...

# Or use gofmt directly
gofmt -w .
```

### 5. Lint Code

```bash
# Install golangci-lint (one time)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

---

## Testing

### Run Tests

**All tests:**
```bash
go test ./pkg/...
```

**Specific package:**
```bash
go test ./pkg -v
```

**With coverage:**
```bash
go test -cover ./pkg/...
```

**Generate coverage report:**
```bash
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
```

### Test Structure

Tests are located alongside source files:

```
pkg/
├── registry.go
├── registry_test.go           # Registry reader tests
├── config.go
├── config_test.go             # Config loader tests
├── htmlreport.go
└── htmlreport_test.go         # HTML report tests
```

### Writing Tests

**Example test:**

```go
package pkg_test

import (
    "context"
    "testing"

    "compliancetoolkit/pkg"
    "golang.org/x/sys/windows/registry"
)

func TestReadString(t *testing.T) {
    reader := pkg.NewRegistryReader()
    ctx := context.Background()

    // Read a known value
    value, err := reader.ReadString(
        ctx,
        registry.LOCAL_MACHINE,
        `SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
        "ProductName",
    )

    if err != nil {
        t.Fatalf("ReadString failed: %v", err)
    }

    if value == "" {
        t.Error("Expected non-empty ProductName")
    }
}
```

### Integration Tests

**Run integration tests:**

```bash
# Run all tests including integration
go test -tags=integration ./pkg/...
```

**Example integration test:**

```go
//go:build integration
// +build integration

package pkg_test

import (
    "context"
    "testing"

    "compliancetoolkit/pkg"
)

func TestFullReportGeneration(t *testing.T) {
    // Load config
    config, err := pkg.LoadReportConfig("../configs/reports/system_info.json")
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }

    // Execute report
    report := pkg.NewHTMLReport(config.Metadata.ReportTitle, "./test_output")
    // ... execute queries ...

    // Generate report
    err = report.Generate()
    if err != nil {
        t.Fatalf("Failed to generate report: %v", err)
    }
}
```

---

## Contributing

### Development Guidelines

1. **Code Style**
   - Follow Go standard formatting (`go fmt`)
   - Use meaningful variable names
   - Add comments for exported functions
   - Keep functions focused and small

2. **Error Handling**
   - Always check and handle errors
   - Provide context in error messages
   - Use custom error types when appropriate

3. **Logging**
   - Use structured logging (`log/slog`)
   - Include operation context
   - Log at appropriate levels (Debug, Info, Warn, Error)

4. **Testing**
   - Write tests for new features
   - Maintain >80% code coverage
   - Include both unit and integration tests

5. **Documentation**
   - Update docs for new features
   - Add code comments for complex logic
   - Include examples in documentation

### Adding a New Feature

**Example: Adding a new report type**

1. **Create report configuration:**

```json
// configs/reports/my_new_report.json
{
  "version": "1.0",
  "metadata": {
    "report_title": "My New Report",
    "report_version": "1.0.0",
    "author": "Your Name",
    "description": "Description of report",
    "category": "Category",
    "last_updated": "2025-01-05"
  },
  "queries": [
    {
      "name": "my_check",
      "description": "My Check Description",
      "root_key": "HKLM",
      "path": "SOFTWARE\\...",
      "value_name": "MyValue",
      "operation": "read"
    }
  ]
}
```

2. **Test the report:**

```bash
# Build
go build -o ComplianceToolkit.exe ./cmd/toolkit.go

# Run
ComplianceToolkit.exe -report=my_new_report.json

# Verify output
start output\reports\
```

3. **Update documentation:**

```markdown
// docs/reference/REPORTS.md
### My New Report

**File:** `my_new_report.json`
**Category:** Category
**Description:** Description of report
...
```

4. **Create pull request**

### Pull Request Process

1. **Fork the repository**

2. **Create a feature branch:**
   ```bash
   git checkout -b feature/my-new-feature
   ```

3. **Make your changes:**
   - Add code
   - Add tests
   - Update docs

4. **Test your changes:**
   ```bash
   go test ./pkg/...
   go build -o ComplianceToolkit.exe ./cmd/toolkit.go
   # Manual testing
   ```

5. **Commit with clear message:**
   ```bash
   git add .
   git commit -m "Add new feature: description"
   ```

6. **Push to your fork:**
   ```bash
   git push origin feature/my-new-feature
   ```

7. **Open pull request**
   - Describe changes
   - Link related issues
   - Request review

### Code Review Checklist

Before submitting:

- [ ] Code follows Go standards (`go fmt`)
- [ ] Tests pass (`go test ./pkg/...`)
- [ ] Coverage maintained (>80%)
- [ ] Documentation updated
- [ ] Examples added if needed
- [ ] No breaking changes (or documented)
- [ ] Commit messages are clear
- [ ] Build succeeds (`go build`)

---

## Build Automation

### Makefile (Optional)

Create `Makefile` for common tasks:

```makefile
.PHONY: build test clean fmt lint

build:
	go build -o ComplianceToolkit.exe ./cmd/toolkit.go

test:
	go test -v ./pkg/...

coverage:
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out

clean:
	go clean -cache
	rm -f ComplianceToolkit.exe coverage.out

fmt:
	go fmt ./...

lint:
	golangci-lint run

release:
	go build -ldflags="-s -w" -o ComplianceToolkit.exe ./cmd/toolkit.go

install:
	go install ./cmd/toolkit.go
```

**Usage:**
```bash
make build
make test
make coverage
```

### PowerShell Build Script

Create `build.ps1`:

```powershell
# build.ps1 - Build automation script

param(
    [switch]$Release,
    [switch]$Test,
    [switch]$Clean
)

if ($Clean) {
    Write-Host "Cleaning build cache..." -ForegroundColor Cyan
    go clean -cache
    Remove-Item -Path "ComplianceToolkit.exe" -ErrorAction SilentlyContinue
}

if ($Test) {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test -v ./pkg/...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed!"
        exit 1
    }
}

Write-Host "Building ComplianceToolkit..." -ForegroundColor Cyan

if ($Release) {
    # Release build with optimizations
    go build -ldflags="-s -w" -o ComplianceToolkit.exe ./cmd/toolkit.go
} else {
    # Development build
    go build -o ComplianceToolkit.exe ./cmd/toolkit.go
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host "Binary: ComplianceToolkit.exe" -ForegroundColor Cyan
} else {
    Write-Error "Build failed!"
    exit 1
}
```

**Usage:**
```powershell
# Development build
.\build.ps1

# With tests
.\build.ps1 -Test

# Release build
.\build.ps1 -Release

# Clean and build
.\build.ps1 -Clean
```

---

## Debugging

### VS Code Configuration

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Toolkit",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/toolkit.go",
            "args": []
        },
        {
            "name": "Launch with Report",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/toolkit.go",
            "args": ["-report=system_info.json", "-quiet"]
        }
    ]
}
```

### Debugging Tips

1. **Add debug logging:**
   ```go
   slog.Debug("Debug info", "key", value)
   ```

2. **Use delve debugger:**
   ```bash
   dlv debug ./cmd/toolkit.go
   ```

3. **Print variables:**
   ```go
   fmt.Printf("DEBUG: variable = %+v\n", variable)
   ```

4. **Check logs:**
   ```bash
   type output\logs\toolkit_*.log
   ```

---

## Performance Optimization

### Profiling

**CPU profiling:**
```go
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
```

**Memory profiling:**
```go
import "runtime/pprof"

f, _ := os.Create("mem.prof")
pprof.WriteHeapProfile(f)
f.Close()
```

**Analyze:**
```bash
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Benchmarking

```go
func BenchmarkReadString(b *testing.B) {
    reader := pkg.NewRegistryReader()
    ctx := context.Background()

    for i := 0; i < b.N; i++ {
        _, _ = reader.ReadString(
            ctx,
            registry.LOCAL_MACHINE,
            `SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
            "ProductName",
        )
    }
}
```

**Run:**
```bash
go test -bench=. ./pkg/...
```

---

## Troubleshooting

### Build Issues

**Issue:** `cannot find package`

**Solution:**
```bash
go mod tidy
go mod download
```

**Issue:** `templates not found`

**Solution:**
```bash
# Verify templates exist
ls pkg/templates/html/*.html
ls pkg/templates/css/*.css

# Rebuild
go clean -cache
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Runtime Issues

**Issue:** `Access denied` errors

**Solution:**
```bash
# Run as Administrator
Right-click ComplianceToolkit.exe → Run as administrator
```

**Issue:** Templates not updating

**Solution:**
```bash
# Clean build cache
go clean -cache
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

---

## Next Steps

- ✅ **Add Reports**: See [Adding Reports Guide](ADDING_REPORTS.md)
- ✅ **Customize Templates**: See [Template System](TEMPLATES.md)
- ✅ **Architecture**: See [Architecture Overview](ARCHITECTURE.md)
- ✅ **Project Status**: See [Project Status](../PROJECT_STATUS.md)

---

*Development Guide v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
