# Input Validation & Security

This document describes the comprehensive input validation and sanitization system implemented in Compliance Toolkit to prevent security vulnerabilities and system crashes.

## Overview

The validation system protects against:
- **Path traversal attacks** (../ or ..\\ patterns)
- **Injection attacks** (null bytes, control characters)
- **Buffer overflows** (excessive length inputs)
- **Invalid registry operations** (malformed paths, unsupported operations)
- **Security policy violations** (access to blocked registry keys)

## Architecture

### Validation Framework

Located in `pkg/validation.go`, the framework provides:

```go
// Validator interface for self-validating types
type Validator interface {
    Validate() error
}

// Structured validation errors with error codes
type ValidationError struct {
    Field   string
    Value   string
    Message string
    Code    ValidationErrorCode
}
```

### Error Codes

```go
const (
    ErrCodeInvalidPath          // Malformed registry path
    ErrCodeInvalidRootKey       // Unknown root key
    ErrCodeInvalidValueName     // Malformed value name
    ErrCodePathTraversal        // Path traversal attempt detected
    ErrCodeInjectionAttempt     // Injection attack detected
    ErrCodeEmptyField           // Required field is empty
    ErrCodeTooLong              // Exceeds maximum length
    ErrCodeInvalidCharacters    // Contains disallowed characters
    ErrCodeDisallowedPath       // Blocked by security policy
)
```

## Validation Rules

### Registry Root Keys

**Valid Values:**
- `HKLM` or `HKEY_LOCAL_MACHINE`
- `HKCU` or `HKEY_CURRENT_USER`
- `HKCR` or `HKEY_CLASSES_ROOT`
- `HKU` or `HKEY_USERS`
- `HKCC` or `HKEY_CURRENT_CONFIG`

**Checks:**
- Cannot be empty
- Must match one of the valid values (case-sensitive)
- Must be in security allow list (if configured)

**Usage:**
```go
if err := ValidateRootKey("HKLM"); err != nil {
    // Handle validation error
}
```

### Registry Paths

**Constraints:**
- Maximum length: 255 characters
- Maximum nesting depth: 512 levels
- Allowed characters: `a-zA-Z0-9\\ -_.()` (alphanumeric, backslash, space, hyphen, underscore, dot, parentheses)
- Cannot start or end with backslash
- No consecutive backslashes
- No null bytes or control characters

**Security Checks:**
- Path traversal detection (../ or ..\\ patterns)
- Injection prevention (null bytes, control chars)
- Deny list enforcement (blocked registry paths)

**Usage:**
```go
if err := ValidateRegistryPath(path); err != nil {
    // Invalid path
}

// Additional security checks
if err := ValidateNoPathTraversal(path); err != nil {
    // Path traversal attempt
}

if err := ValidateNoInjection(path); err != nil {
    // Injection attempt
}
```

### Registry Value Names

**Constraints:**
- Maximum length: 16,383 characters (Windows MAX_PATH limit)
- Empty value name is valid (refers to default value)
- Allowed characters: `a-zA-Z0-9 -_.()\[\]{}@#$%&+=`
- No null bytes or control characters

**Usage:**
```go
if err := ValidateValueName(valueName); err != nil {
    // Invalid value name
}
```

### Operations

**Valid Operations:**
- `read` - Read registry values (currently supported)
- Future: `write`, `delete` (not implemented, read-only by design)

**Usage:**
```go
if err := ValidateOperation("read"); err != nil {
    // Invalid operation
}
```

## Security Policies

### Deny List

Blocks access to security-sensitive registry paths:

```go
denyList := []string{
    "SECURITY\\Policy\\Secrets",
    "SAM\\SAM\\Domains\\Account\\Users",
    "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon\\SpecialAccounts",
}

if err := ValidateAgainstDenyList(path, denyList); err != nil {
    // Access blocked by security policy
}
```

**Features:**
- Case-insensitive matching
- Exact path matching
- Prefix matching (blocks subkeys of denied paths)

**Example:**
```go
// Blocked: exact match
path = "SECURITY\\Policy\\Secrets"

// Blocked: subkey of denied path
path = "SECURITY\\Policy\\Secrets\\SubKey"

// Allowed: parent of denied path
path = "SECURITY\\Policy"
```

### Allow List

Restricts which registry root keys can be accessed:

```go
allowList := []string{
    "HKEY_LOCAL_MACHINE",
    "HKEY_CURRENT_USER",
}

if err := ValidateAgainstAllowList("HKLM", allowList); err != nil {
    // Root key not in allow list
}
```

**Features:**
- Case-insensitive matching
- Empty allow list means all are allowed

## Sanitization

### Path Sanitization

Cleans potentially dangerous input while preserving valid data:

```go
sanitized := SanitizeRegistryPath(userInput)
```

**Transformations:**
- Removes null bytes and control characters
- Trims leading/trailing whitespace
- Removes consecutive backslashes
- Removes leading/trailing backslashes

**Example:**
```go
input := "  SOFTWARE\\\\Microsoft\x00Test\\  "
output := SanitizeRegistryPath(input)
// Result: "SOFTWARE\\MicrosoftTest"
```

### Value Name Sanitization

```go
sanitized := SanitizeValueName(userInput)
```

**Transformations:**
- Removes null bytes and control characters
- Trims whitespace

## Integration Points

### Configuration Loading

All `RegistryQuery` objects are validated when loaded:

```go
config, err := LoadRegistryConfig(configPath)
if err != nil {
    return err
}

// Automatic validation of all queries
if err := ValidateConfig(config); err != nil {
    return fmt.Errorf("config validation failed: %w", err)
}
```

### Runtime Validation

Security policies are enforced at runtime:

```go
// Before executing any registry read
for _, query := range config.Queries {
    // Validate against deny list
    if err := ValidateAgainstDenyList(query.Path, denyPaths); err != nil {
        log.Error("Query blocked by security policy", err)
        continue
    }

    // Validate against allow list
    if err := ValidateAgainstAllowList(query.RootKey, allowedRoots); err != nil {
        log.Error("Root key not allowed", err)
        continue
    }

    // Execute query...
}
```

### CLI Flag Validation

Command-line inputs are validated before use:

```go
// Validate directory paths
if err := ValidateFilePath(outputDir, nil); err != nil {
    fmt.Fprintf(os.Stderr, "Invalid output directory: %v\n", err)
    os.Exit(1)
}

// Validate timeout range
if timeout < time.Second || timeout > 5*time.Minute {
    fmt.Fprintf(os.Stderr, "Timeout must be between 1s and 5m\n")
    os.Exit(1)
}

// Validate log level
validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
if !validLevels[strings.ToLower(logLevel)] {
    fmt.Fprintf(os.Stderr, "Invalid log level\n")
    os.Exit(1)
}
```

## Testing

Comprehensive test suite in `pkg/validation_test.go`:

```bash
# Run all validation tests
go test ./pkg -run TestValidate -v

# Run specific test
go test ./pkg -run TestValidateRegistryPath -v

# Run with coverage
go test ./pkg -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

- **Root key validation**: 11 test cases
- **Path validation**: 16 test cases
- **Value name validation**: 9 test cases
- **Path traversal detection**: 6 test cases
- **Injection detection**: 7 test cases
- **Deny list enforcement**: 6 test cases
- **Allow list enforcement**: 5 test cases
- **File path validation**: 8 test cases
- **Sanitization**: 7 test cases
- **Query validation**: 5 test cases
- **Config validation**: 4 test cases

**Total: 84 test cases**

## Common Patterns

### Validating User Input

```go
func ProcessUserInput(userPath string) error {
    // Step 1: Sanitize
    cleanPath := SanitizeRegistryPath(userPath)

    // Step 2: Validate
    if err := ValidateRegistryPath(cleanPath); err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }

    // Step 3: Security checks
    if err := ValidateNoPathTraversal(cleanPath); err != nil {
        return fmt.Errorf("security violation: %w", err)
    }

    if err := ValidateNoInjection(cleanPath); err != nil {
        return fmt.Errorf("injection attempt: %w", err)
    }

    // Step 4: Policy enforcement
    if err := ValidateAgainstDenyList(cleanPath, denyList); err != nil {
        return fmt.Errorf("access denied: %w", err)
    }

    // Safe to use
    return ProcessPath(cleanPath)
}
```

### Implementing Validator Interface

```go
type CustomQuery struct {
    RootKey   string
    Path      string
    ValueName string
}

func (q *CustomQuery) Validate() error {
    if err := ValidateRootKey(q.RootKey); err != nil {
        return err
    }

    if err := ValidateRegistryPath(q.Path); err != nil {
        return err
    }

    if q.ValueName != "" {
        if err := ValidateValueName(q.ValueName); err != nil {
            return err
        }
    }

    return nil
}
```

### Handling Validation Errors

```go
err := query.Validate()
if err != nil {
    if verr, ok := err.(*ValidationError); ok {
        switch verr.Code {
        case ErrCodePathTraversal:
            log.Error("Security alert: path traversal attempt", "path", verr.Value)
            // Take defensive action
        case ErrCodeInjectionAttempt:
            log.Error("Security alert: injection attempt", "input", verr.Value)
            // Take defensive action
        default:
            log.Warn("Validation failed", "field", verr.Field, "error", verr.Message)
        }
    }
    return err
}
```

## Security Best Practices

1. **Always validate before use**: Never use user input without validation
2. **Sanitize, then validate**: Clean input first, then check constraints
3. **Use deny lists for sensitive data**: Block access to security-critical registry keys
4. **Use allow lists for operations**: Whitelist allowed registry root keys
5. **Log security violations**: Track attempted attacks for forensics
6. **Fail securely**: Reject on validation failure, don't attempt to fix
7. **Test edge cases**: Fuzz testing, boundary conditions, Unicode attacks

## Performance Considerations

- **Regex compilation**: Patterns are compiled once as package-level variables
- **Map lookups**: O(1) for root key and operation validation
- **String operations**: Minimal allocations, use strings package efficiently
- **Validation cost**: ~100-500 nanoseconds per validation check
- **Negligible overhead**: <0.01% of total registry read time

## Future Enhancements

- [ ] Rate limiting for validation failures (detect brute force attacks)
- [ ] Validation metrics (track validation failures by type)
- [ ] Custom validation rules (plugin system for organization-specific policies)
- [ ] Validation caching (cache validated inputs for repeated checks)
- [ ] SIEM integration (send security violations to security tools)

## See Also

- [Architecture Overview](ARCHITECTURE.md)
- [Security Configuration](../user-guide/CONFIGURATION.md#security-configuration)
- [Error Handling](ERROR_HANDLING.md)
- [Testing Guide](TESTING.md)
