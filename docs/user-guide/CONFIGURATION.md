# Configuration Guide

Compliance Toolkit supports flexible configuration management with multiple sources and a clear precedence hierarchy.

## Configuration Precedence

Configuration values are loaded in this order (highest to lowest priority):

1. **CLI Flags** (highest priority)
2. **Environment Variables**
3. **YAML Configuration File**
4. **Default Values** (lowest priority)

This means CLI flags override environment variables, which override YAML config, which overrides defaults.

## Quick Start

### Generate Default Configuration

```bash
# Generate default config.yaml in config/ directory
ComplianceToolkit.exe --generate-config

# Generate config in a custom location
ComplianceToolkit.exe --generate-config --config=/path/to/custom/config.yaml
```

### Using Configuration File

```bash
# Use default config locations (./config.yaml, ./config/config.yaml, etc.)
ComplianceToolkit.exe

# Specify custom config file
ComplianceToolkit.exe --config=/path/to/config.yaml

# Run reports with config
ComplianceToolkit.exe --config=./myconfig.yaml --report=all
```

## Configuration File (YAML)

The configuration file uses YAML format and is organized into four main sections:

### Server/Runtime Configuration

Controls registry operation timeouts and concurrency:

```yaml
server:
  host: localhost                      # Reserved for future HTTP server
  port: 8080                           # Reserved for future HTTP server
  read_timeout: 5s                     # Registry operation timeout (e.g., "5s", "10s", "1m")
  max_concurrent_reads: 10             # Max concurrent registry reads
  graceful_shutdown_timeout: 30s       # Cleanup timeout on exit
```

**Key Settings:**
- `read_timeout`: Prevents hanging on locked registry keys (default: 5s)
- `max_concurrent_reads`: Limits parallel registry operations (default: 10)

### Logging Configuration

Controls logging behavior, format, and output:

```yaml
logging:
  level: info                          # Log level: debug, info, warn, error
  format: json                         # Format: json, text
  output_path: stdout                  # Output: "stdout", "stderr", or filepath
  enable_file_logging: true            # Also log to files in output/logs/
  max_log_file_size_mb: 100           # Rotation size (0 = no rotation)
  max_log_file_age_days: 30           # Retention period (0 = keep forever)
```

**Key Settings:**
- `level`: Controls verbosity (debug shows all registry operations)
- `format`: `json` for machine-readable logs, `text` for human-readable
- `enable_file_logging`: When true, logs to both console and files

### Reports Configuration

Controls report generation, output paths, and features:

```yaml
reports:
  config_path: configs/reports         # Directory containing report JSON configs
  output_path: output/reports          # Where HTML reports are saved
  evidence_path: output/evidence       # Where JSON evidence logs are saved
  template_path: ""                    # Custom templates (empty = use embedded)
  enable_evidence: true                # Enable JSON compliance evidence logs
  enable_dark_mode: true               # Enable dark mode in HTML reports
  parallel: false                      # Parallel report generation (experimental)
  max_parallel_reports: 0              # Max parallel (0 = use CPU count)
```

**Key Settings:**
- `config_path`: Location of report definition JSON files
- `enable_evidence`: Controls compliance audit trail generation
- `enable_dark_mode`: Toggles dark mode support in HTML reports

### Security Configuration

Controls security constraints and access policies:

```yaml
security:
  require_admin_privileges: false      # Enforce admin check on startup
  allowed_registry_roots:              # Permitted registry hives
    - HKEY_LOCAL_MACHINE
    - HKEY_CURRENT_USER
    - HKEY_CLASSES_ROOT
    - HKEY_USERS
    - HKEY_CURRENT_CONFIG
  deny_registry_paths:                 # Blocked paths (security-sensitive)
    - 'SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts'
    - 'SECURITY\Policy\Secrets'
    - 'SAM\SAM\Domains\Account\Users'
  read_only: true                      # Always true (compliance scanner is read-only)
  audit_mode: false                    # Log all registry access attempts
```

**Key Settings:**
- `allowed_registry_roots`: Whitelist of permitted registry hives
- `deny_registry_paths`: Blacklist of sensitive keys (blocks access)
- `read_only`: Always `true` (tool is read-only by design)
- `audit_mode`: Logs every registry read for security auditing

## Environment Variables

All configuration options can be set via environment variables with the prefix `COMPLIANCE_TOOLKIT_` and underscores separating nested keys.

### Format

```
COMPLIANCE_TOOLKIT_<SECTION>_<KEY>=value
```

Nested keys use underscores: `section.nested.key` → `COMPLIANCE_TOOLKIT_SECTION_NESTED_KEY`

### Examples

```bash
# Windows (PowerShell)
$env:COMPLIANCE_TOOLKIT_LOGGING_LEVEL="debug"
$env:COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE="false"
$env:COMPLIANCE_TOOLKIT_SERVER_READ_TIMEOUT="10s"

# Windows (CMD)
set COMPLIANCE_TOOLKIT_LOGGING_LEVEL=debug
set COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE=false
set COMPLIANCE_TOOLKIT_SERVER_READ_TIMEOUT=10s

# Linux/macOS (Bash)
export COMPLIANCE_TOOLKIT_LOGGING_LEVEL=debug
export COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE=false
export COMPLIANCE_TOOLKIT_SERVER_READ_TIMEOUT=10s
```

### Common Use Cases

**Debug Logging:**
```bash
set COMPLIANCE_TOOLKIT_LOGGING_LEVEL=debug
ComplianceToolkit.exe --report=all
```

**Custom Output Paths:**
```bash
set COMPLIANCE_TOOLKIT_REPORTS_OUTPUT_PATH=C:\ComplianceReports
set COMPLIANCE_TOOLKIT_REPORTS_EVIDENCE_PATH=C:\Evidence
ComplianceToolkit.exe --report=all
```

**Disable Evidence Logging:**
```bash
set COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE=false
ComplianceToolkit.exe --report=NIST_800_171_compliance.json
```

## Command-Line Flags

CLI flags provide the highest priority overrides for quick one-off changes.

### Available Flags

```bash
# Configuration file
--config, -c <path>           Path to YAML config file
--generate-config             Generate default config.yaml and exit

# Report execution
--report, -r <name>           Report to run (filename or "all")
--list, -l                    List available reports and exit
--quiet, -q                   Suppress non-essential output

# Override flags (take precedence over config)
--output <path>               Output directory for HTML reports
--logs <path>                 Logs directory
--evidence <path>             Evidence logs directory
--timeout <duration>          Registry operation timeout (e.g., "10s", "1m")
--log-level <level>           Log level: debug, info, warn, error
```

### Examples

```bash
# Use custom config file
ComplianceToolkit.exe --config=./production.yaml --report=all

# Override timeout for specific run
ComplianceToolkit.exe --timeout=30s --report=NIST_800_171_compliance.json

# Debug logging for troubleshooting
ComplianceToolkit.exe --log-level=debug --report=all

# Custom output paths
ComplianceToolkit.exe --output=C:\Reports --evidence=C:\Evidence --report=all

# Quiet mode for scheduled tasks
ComplianceToolkit.exe --quiet --report=all
```

## Configuration Scenarios

### Development Environment

```yaml
# config/dev.yaml
server:
  read_timeout: 10s              # Longer timeout for debugging
logging:
  level: debug                   # Verbose logging
  format: text                   # Human-readable logs
  enable_file_logging: true
reports:
  enable_evidence: true
  enable_dark_mode: true
security:
  audit_mode: true               # Log all registry access
```

Usage:
```bash
ComplianceToolkit.exe --config=config/dev.yaml
```

### Production/Scheduled Tasks

```yaml
# config/production.yaml
server:
  read_timeout: 5s
logging:
  level: warn                    # Only warnings and errors
  format: json                   # Machine-readable
  enable_file_logging: true
reports:
  output_path: C:\ComplianceReports
  evidence_path: C:\ComplianceEvidence
  enable_evidence: true
  parallel: false
security:
  require_admin_privileges: true # Enforce admin
  audit_mode: true
```

Usage:
```bash
ComplianceToolkit.exe --config=C:\Config\production.yaml --report=all --quiet
```

### CI/CD Pipeline

Use environment variables for dynamic configuration:

```bash
# GitHub Actions / Jenkins / Azure DevOps
export COMPLIANCE_TOOLKIT_LOGGING_LEVEL=info
export COMPLIANCE_TOOLKIT_REPORTS_OUTPUT_PATH=$CI_ARTIFACTS_DIR/reports
export COMPLIANCE_TOOLKIT_REPORTS_EVIDENCE_PATH=$CI_ARTIFACTS_DIR/evidence
export COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE=true

./ComplianceToolkit.exe --report=all --quiet
```

### Docker Container

Use environment variables in Dockerfile or docker-compose:

```dockerfile
# Dockerfile
FROM mcr.microsoft.com/windows/servercore:ltsc2022
COPY ComplianceToolkit.exe /app/
ENV COMPLIANCE_TOOLKIT_LOGGING_LEVEL=info \
    COMPLIANCE_TOOLKIT_REPORTS_OUTPUT_PATH=/app/output/reports \
    COMPLIANCE_TOOLKIT_REPORTS_EVIDENCE_PATH=/app/output/evidence
CMD ["C:/app/ComplianceToolkit.exe", "--report=all", "--quiet"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  compliance:
    image: compliancetoolkit:latest
    environment:
      - COMPLIANCE_TOOLKIT_LOGGING_LEVEL=info
      - COMPLIANCE_TOOLKIT_REPORTS_ENABLE_EVIDENCE=true
    volumes:
      - ./output:/app/output
```

## Validation

The configuration system includes built-in validation:

- **Log levels**: Must be `debug`, `info`, `warn`, or `error`
- **Log formats**: Must be `json` or `text`
- **Timeouts**: Must be positive durations (e.g., `5s`, `1m`)
- **Paths**: Must exist or be creatable
- **Read-only**: `security.read_only` must always be `true`
- **Registry roots**: Must not be empty

Invalid configurations will cause startup to fail with a descriptive error message.

## Troubleshooting

### Config File Not Found

**Error:** `No config file found, using defaults`

**Solution:** The config file is optional. This is a debug message, not an error. To use a config file:
1. Generate default config: `ComplianceToolkit.exe --generate-config`
2. Place `config.yaml` in one of these locations:
   - `./config.yaml` (current directory)
   - `./config/config.yaml`
   - `../config/config.yaml` (when running from bin/)
   - `$HOME/.compliancetoolkit/config.yaml`

### Invalid Configuration

**Error:** `invalid configuration: <details>`

**Solution:** Review the error message for specific validation failures. Common issues:
- Invalid log level (must be debug/info/warn/error)
- Invalid timeout format (use `5s`, `1m`, etc.)
- Non-existent paths (ensure directories exist or are creatable)

### Environment Variable Not Working

**Problem:** Setting `COMPLIANCE_TOOLKIT_X` has no effect

**Solution:**
1. Verify correct prefix: `COMPLIANCE_TOOLKIT_` (with underscore)
2. Use underscores for nesting: `COMPLIANCE_TOOLKIT_LOGGING_LEVEL` (not `COMPLIANCE_TOOLKIT_LOGGING.LEVEL`)
3. Check variable is set in current session:
   ```bash
   # PowerShell
   $env:COMPLIANCE_TOOLKIT_LOGGING_LEVEL

   # CMD
   echo %COMPLIANCE_TOOLKIT_LOGGING_LEVEL%
   ```

### CLI Flag Not Working

**Problem:** Flag value ignored

**Solution:**
1. Ensure flag comes before positional arguments:
   ```bash
   # Correct
   ComplianceToolkit.exe --timeout=10s --report=all

   # Incorrect (flag after report name)
   ComplianceToolkit.exe --report=all --timeout=10s
   ```
2. Use `=` for values: `--timeout=10s` (not `--timeout 10s`)

## Best Practices

1. **Use YAML for base configuration**: Store common settings in `config.yaml`
2. **Use ENV vars for environment-specific overrides**: Different settings per environment (dev/staging/prod)
3. **Use CLI flags for one-off runs**: Quick testing without modifying config
4. **Version control your config files**: Commit `config.yaml` to Git (exclude sensitive values)
5. **Use separate configs for different environments**: `config/dev.yaml`, `config/prod.yaml`
6. **Enable audit mode in production**: Track all registry access for security compliance
7. **Rotate logs in production**: Set `max_log_file_size_mb` and `max_log_file_age_days`
8. **Use JSON logs for production**: Easier to parse and analyze with log aggregation tools

## Reference

### Complete Default Configuration

See [`config/config.yaml`](../../config/config.yaml) for the complete default configuration with detailed comments.

### Configuration Schema

| Section | Key | Type | Default | Description |
|---------|-----|------|---------|-------------|
| **server** | | | | |
| | host | string | localhost | Future HTTP server host |
| | port | int | 8080 | Future HTTP server port |
| | read_timeout | duration | 5s | Registry operation timeout |
| | max_concurrent_reads | int | 10 | Max concurrent reads |
| | graceful_shutdown_timeout | duration | 30s | Cleanup timeout |
| **logging** | | | | |
| | level | string | info | debug, info, warn, error |
| | format | string | json | json, text |
| | output_path | string | stdout | stdout, stderr, filepath |
| | enable_file_logging | bool | true | Log to files |
| | max_log_file_size_mb | int | 100 | Rotation size (0=disabled) |
| | max_log_file_age_days | int | 30 | Retention (0=forever) |
| **reports** | | | | |
| | config_path | string | configs/reports | Report configs directory |
| | output_path | string | output/reports | HTML reports directory |
| | evidence_path | string | output/evidence | Evidence logs directory |
| | template_path | string | "" | Custom templates (empty=embedded) |
| | enable_evidence | bool | true | Enable evidence logging |
| | enable_dark_mode | bool | true | Dark mode in reports |
| | parallel | bool | false | Parallel execution |
| | max_parallel_reports | int | 0 | Max parallel (0=CPU count) |
| **security** | | | | |
| | require_admin_privileges | bool | false | Enforce admin check |
| | allowed_registry_roots | []string | HKLM, HKCU, etc. | Permitted hives |
| | deny_registry_paths | []string | See config | Blocked paths |
| | read_only | bool | true | Read-only mode (always true) |
| | audit_mode | bool | false | Log all access |

## See Also

- [CLI Usage Guide](CLI_USAGE.md) - Command-line interface documentation
- [Automation Guide](AUTOMATION.md) - Scheduled tasks and CI/CD integration
- [Developer Guide: Architecture](../developer-guide/ARCHITECTURE.md) - Configuration system internals
