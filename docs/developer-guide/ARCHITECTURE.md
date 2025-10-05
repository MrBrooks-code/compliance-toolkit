# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Production-grade Windows Registry reading library with context support, structured logging, and comprehensive error handling. Part of a compliance toolkit for defensive security and system information gathering.

**Platform Requirement**: Windows-only (depends on `golang.org/x/sys/windows/registry`)

## Project Structure

```
cmd/
  main.go                    - Example usage with all features
pkg/
  registryreader.go          - Core registry reader with context & observability
  registryreader_test.go     - Unit & integration tests
  config.go                  - JSON config loader for registry operations
  config_test.go             - Config tests
configs/
  registry_operations.json   - Declarative registry operation definitions
```

## Common Commands

```bash
# Build
go build -o registryreader.exe ./cmd

# Run
go run ./cmd/main.go

# Test (Windows only)
go test ./pkg/... -v

# Test with coverage
go test ./pkg/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Benchmarks
go test ./pkg/... -bench=. -benchmem

# Run specific test
go test ./pkg -run TestRegistryReader_ReadString_Integration -v
```

## Architecture

### Context-Aware Registry Operations

All read operations accept `context.Context` for cancellation and timeout control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := reader.ReadString(ctx, registry.LOCAL_MACHINE, path, name)
```

Operations spawn goroutines that respect context cancellation via select channels.

### Functional Options Pattern

Constructor uses variadic options for clean configuration:

```go
reader := pkg.NewRegistryReader(
    pkg.WithLogger(customLogger),
    pkg.WithTimeout(10*time.Second),
)
```

### Structured Error Handling

Custom `RegistryError` type provides operation context:

```go
type RegistryError struct {
    Op    string  // "OpenKey", "GetStringValue", etc.
    Key   string  // Registry path
    Value string  // Value name
    Err   error   // Underlying error
}
```

Use `pkg.IsNotExist(err)` to check for missing keys/values.

### Structured Logging

Uses `log/slog` for JSON structured logs with operation timing:

```json
{
  "level": "debug",
  "msg": "registry read completed",
  "operation": "ReadString",
  "path": "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
  "value": "ProductName",
  "duration": "15ms"
}
```

### Batch Operations

`BatchRead()` efficiently reads multiple values from one key (single open/close):

```go
data, err := reader.BatchRead(ctx, rootKey, path, []string{"Value1", "Value2"})
// Returns map[string]interface{} with auto-detected types
```

### Config-Driven Execution

`LoadConfig()` parses JSON operation definitions. Main demonstrates executing config-based queries.

Root key parsing supports both short (`HKLM`) and full (`HKEY_LOCAL_MACHINE`) forms.

## Read Methods

All methods are context-aware and include structured logging:

- `ReadString(ctx, rootKey, path, valueName)` - REG_SZ
- `ReadInteger(ctx, rootKey, path, valueName)` - DWORD/QWORD → uint64
- `ReadBinary(ctx, rootKey, path, valueName)` - REG_BINARY → []byte
- `ReadStrings(ctx, rootKey, path, valueName)` - REG_MULTI_SZ → []string
- `BatchRead(ctx, rootKey, path, []valueName)` - Multiple values → map[string]interface{}

`ReadStringWithTimeout()` provides explicit per-operation timeout override.

## Testing Strategy

- **Unit tests**: Error handling, config parsing, type conversions
- **Integration tests**: Real registry reads (Windows-only, marked with `_Integration` suffix)
- **Benchmarks**: Performance testing for single vs batch operations
- **Table-driven tests**: Root key parsing, error conditions

Run integration tests on Windows only. CI should skip them on non-Windows platforms.

## Performance Notes

- Batch operations > 3x faster than individual reads for same key
- Each read spawns goroutine for context cancellation (minimal overhead ~1-2µs)
- Registry operations typically complete in 1-20ms depending on key depth

## Security & Constraints

- **Read-only**: All operations use `registry.QUERY_VALUE` (no write/modify)
- **Defensive**: Proper resource cleanup with deferred Close()
- **Timeout protection**: Default 5s timeout prevents hanging on registry locks
- **Error transparency**: Wrapped errors preserve original registry errors
