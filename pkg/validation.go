package pkg

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Validator interface for types that can validate themselves
type Validator interface {
	Validate() error
}

// ValidationError represents a validation failure with context
type ValidationError struct {
	Field   string
	Value   string
	Message string
	Code    ValidationErrorCode
}

// ValidationErrorCode categorizes validation errors
type ValidationErrorCode int

const (
	ErrCodeInvalidPath ValidationErrorCode = iota + 1000
	ErrCodeInvalidRootKey
	ErrCodeInvalidValueName
	ErrCodePathTraversal
	ErrCodeInjectionAttempt
	ErrCodeEmptyField
	ErrCodeTooLong
	ErrCodeInvalidCharacters
	ErrCodeDisallowedPath
)

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%d] in field '%s': %s (value: %q)",
		e.Code, e.Field, e.Message, e.Value)
}

// Registry path validation constants
const (
	// Maximum lengths to prevent buffer overflows and DoS
	MaxRegistryPathLength      = 255
	MaxRegistryValueNameLength = 16383 // Windows MAX_PATH limit
	MaxRegistryKeyDepth        = 512   // Reasonable nesting limit

	// Character restrictions
	invalidPathChars = "\x00\r\n\t"
)

// Regular expressions for validation
var (
	// Registry path must be alphanumeric, backslashes, spaces, hyphens, underscores, dots
	validRegistryPathRegex = regexp.MustCompile(`^[a-zA-Z0-9\\\s\-_.()]+$`)

	// Value names can include more characters but still need sanitization
	validValueNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-_.()\[\]{}@#$%&+=]+$`)

	// Detect potential injection attempts (null bytes, control chars, etc.)
	injectionPatternRegex = regexp.MustCompile(`[\x00-\x1F\x7F]`)

	// Path traversal patterns
	pathTraversalRegex = regexp.MustCompile(`\.\.[\\/]`)
)

// ValidRootKeys maps valid root key strings to registry.Key values
var ValidRootKeys = map[string]registry.Key{
	"HKLM":                 registry.LOCAL_MACHINE,
	"HKEY_LOCAL_MACHINE":   registry.LOCAL_MACHINE,
	"HKCU":                 registry.CURRENT_USER,
	"HKEY_CURRENT_USER":    registry.CURRENT_USER,
	"HKCR":                 registry.CLASSES_ROOT,
	"HKEY_CLASSES_ROOT":    registry.CLASSES_ROOT,
	"HKU":                  registry.USERS,
	"HKEY_USERS":           registry.USERS,
	"HKCC":                 registry.CURRENT_CONFIG,
	"HKEY_CURRENT_CONFIG":  registry.CURRENT_CONFIG,
}

// Validate implements the Validator interface for RegistryQuery
func (r *RegistryQuery) Validate() error {
	// Validate root key
	if err := ValidateRootKey(r.RootKey); err != nil {
		return err
	}

	// Validate registry path
	if err := ValidateRegistryPath(r.Path); err != nil {
		return err
	}

	// Validate value name (if provided)
	if r.ValueName != "" {
		if err := ValidateValueName(r.ValueName); err != nil {
			return err
		}
	}

	// Validate operation
	if err := ValidateOperation(r.Operation); err != nil {
		return err
	}

	// Additional security checks
	if err := ValidateNoPathTraversal(r.Path); err != nil {
		return err
	}

	if err := ValidateNoInjection(r.Path); err != nil {
		return err
	}

	if r.ValueName != "" {
		if err := ValidateNoInjection(r.ValueName); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRootKey validates a registry root key string
func ValidateRootKey(rootKey string) error {
	if rootKey == "" {
		return &ValidationError{
			Field:   "RootKey",
			Value:   rootKey,
			Message: "root key cannot be empty",
			Code:    ErrCodeEmptyField,
		}
	}

	if _, valid := ValidRootKeys[rootKey]; !valid {
		return &ValidationError{
			Field:   "RootKey",
			Value:   rootKey,
			Message: fmt.Sprintf("invalid root key, must be one of: %v", getValidRootKeyNames()),
			Code:    ErrCodeInvalidRootKey,
		}
	}

	return nil
}

// ValidateRegistryPath validates a registry path for security and correctness
func ValidateRegistryPath(path string) error {
	// Check for empty path
	if path == "" {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "registry path cannot be empty",
			Code:    ErrCodeEmptyField,
		}
	}

	// Check length
	if len(path) > MaxRegistryPathLength {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: fmt.Sprintf("registry path exceeds maximum length of %d characters", MaxRegistryPathLength),
			Code:    ErrCodeTooLong,
		}
	}

	// Check for invalid characters
	if strings.ContainsAny(path, invalidPathChars) {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "registry path contains invalid control characters",
			Code:    ErrCodeInvalidCharacters,
		}
	}

	// Validate against allowed character set
	if !validRegistryPathRegex.MatchString(path) {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "registry path contains disallowed characters (only alphanumeric, backslash, space, hyphen, underscore, dot, parentheses allowed)",
			Code:    ErrCodeInvalidCharacters,
		}
	}

	// Check nesting depth (prevent DoS via deep recursion)
	depth := strings.Count(path, "\\") + 1
	if depth > MaxRegistryKeyDepth {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: fmt.Sprintf("registry path depth (%d) exceeds maximum of %d", depth, MaxRegistryKeyDepth),
			Code:    ErrCodeTooLong,
		}
	}

	// Ensure path doesn't start or end with backslash
	if strings.HasPrefix(path, "\\") || strings.HasSuffix(path, "\\") {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "registry path must not start or end with backslash",
			Code:    ErrCodeInvalidPath,
		}
	}

	return nil
}

// ValidateValueName validates a registry value name
func ValidateValueName(valueName string) error {
	// Empty value name is valid (refers to default value)
	if valueName == "" {
		return nil
	}

	// Check length
	if len(valueName) > MaxRegistryValueNameLength {
		return &ValidationError{
			Field:   "ValueName",
			Value:   valueName,
			Message: fmt.Sprintf("value name exceeds maximum length of %d characters", MaxRegistryValueNameLength),
			Code:    ErrCodeTooLong,
		}
	}

	// Check for invalid characters
	if strings.ContainsAny(valueName, invalidPathChars) {
		return &ValidationError{
			Field:   "ValueName",
			Value:   valueName,
			Message: "value name contains invalid control characters",
			Code:    ErrCodeInvalidCharacters,
		}
	}

	// Validate against allowed character set (more permissive than paths)
	if !validValueNameRegex.MatchString(valueName) {
		return &ValidationError{
			Field:   "ValueName",
			Value:   valueName,
			Message: "value name contains disallowed characters",
			Code:    ErrCodeInvalidCharacters,
		}
	}

	return nil
}

// ValidateOperation validates a registry operation type
func ValidateOperation(operation string) error {
	if operation == "" {
		return &ValidationError{
			Field:   "Operation",
			Value:   operation,
			Message: "operation cannot be empty",
			Code:    ErrCodeEmptyField,
		}
	}

	validOps := map[string]bool{
		"read": true,
		// Future: "write", "delete", etc. (currently read-only by design)
	}

	if !validOps[strings.ToLower(operation)] {
		return &ValidationError{
			Field:   "Operation",
			Value:   operation,
			Message: "invalid operation, must be 'read' (tool is read-only)",
			Code:    ErrCodeInvalidCharacters,
		}
	}

	return nil
}

// ValidateNoPathTraversal checks for path traversal attempts
func ValidateNoPathTraversal(path string) error {
	// Check for ../ or ..\ patterns
	if pathTraversalRegex.MatchString(path) {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "path traversal attempt detected (../ or ..\\)",
			Code:    ErrCodePathTraversal,
		}
	}

	// Additional check: normalize path and compare
	normalized := filepath.Clean(path)
	if normalized != path && strings.Contains(normalized, "..") {
		return &ValidationError{
			Field:   "Path",
			Value:   path,
			Message: "path normalization reveals traversal attempt",
			Code:    ErrCodePathTraversal,
		}
	}

	return nil
}

// ValidateNoInjection checks for potential injection attacks
func ValidateNoInjection(input string) error {
	// Check for null bytes and control characters
	if injectionPatternRegex.MatchString(input) {
		return &ValidationError{
			Field:   "Input",
			Value:   input,
			Message: "potential injection attack detected (null bytes or control characters)",
			Code:    ErrCodeInjectionAttempt,
		}
	}

	// Check for unusual Unicode characters that might be used in attacks
	for _, r := range input {
		// Block non-printable Unicode except common whitespace
		if r > 127 && r < 160 {
			return &ValidationError{
				Field:   "Input",
				Value:   input,
				Message: "suspicious Unicode control characters detected",
				Code:    ErrCodeInjectionAttempt,
			}
		}
	}

	return nil
}

// ValidateAgainstDenyList checks if a path is in the security deny list
func ValidateAgainstDenyList(path string, denyList []string) error {
	normalizedPath := strings.ToLower(strings.TrimSpace(path))

	for _, deniedPath := range denyList {
		normalizedDenied := strings.ToLower(strings.TrimSpace(deniedPath))

		// Exact match
		if normalizedPath == normalizedDenied {
			return &ValidationError{
				Field:   "Path",
				Value:   path,
				Message: "access to this registry path is blocked by security policy",
				Code:    ErrCodeDisallowedPath,
			}
		}

		// Prefix match (block subkeys of denied paths)
		if strings.HasPrefix(normalizedPath, normalizedDenied+"\\") {
			return &ValidationError{
				Field:   "Path",
				Value:   path,
				Message: "access to this registry path is blocked by security policy (parent path denied)",
				Code:    ErrCodeDisallowedPath,
			}
		}
	}

	return nil
}

// ValidateAgainstAllowList checks if a root key is in the allow list
func ValidateAgainstAllowList(rootKey string, allowList []string) error {
	if len(allowList) == 0 {
		// Empty allow list means all are allowed
		return nil
	}

	normalizedRootKey := strings.ToUpper(strings.TrimSpace(rootKey))

	for _, allowed := range allowList {
		normalizedAllowed := strings.ToUpper(strings.TrimSpace(allowed))
		if normalizedRootKey == normalizedAllowed {
			return nil
		}
	}

	return &ValidationError{
		Field:   "RootKey",
		Value:   rootKey,
		Message: fmt.Sprintf("root key not in allow list: %v", allowList),
		Code:    ErrCodeInvalidRootKey,
	}
}

// ValidateFilePath validates a file path for safety
func ValidateFilePath(path string, allowedExtensions []string) error {
	if path == "" {
		return &ValidationError{
			Field:   "FilePath",
			Value:   path,
			Message: "file path cannot be empty",
			Code:    ErrCodeEmptyField,
		}
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return &ValidationError{
			Field:   "FilePath",
			Value:   path,
			Message: "path traversal not allowed in file paths",
			Code:    ErrCodePathTraversal,
		}
	}

	// Check for null bytes
	if strings.ContainsRune(path, '\x00') {
		return &ValidationError{
			Field:   "FilePath",
			Value:   path,
			Message: "null bytes not allowed in file paths",
			Code:    ErrCodeInjectionAttempt,
		}
	}

	// Validate extension if allowList provided
	if len(allowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(path))
		allowed := false
		for _, allowedExt := range allowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return &ValidationError{
				Field:   "FilePath",
				Value:   path,
				Message: fmt.Sprintf("file extension not allowed, must be one of: %v", allowedExtensions),
				Code:    ErrCodeInvalidCharacters,
			}
		}
	}

	return nil
}

// SanitizeRegistryPath sanitizes a registry path by removing dangerous characters
func SanitizeRegistryPath(path string) string {
	// Remove null bytes and control characters
	path = strings.Map(func(r rune) rune {
		if r == '\x00' || r < 32 || r == 127 {
			return -1 // Remove character
		}
		return r
	}, path)

	// Trim whitespace
	path = strings.TrimSpace(path)

	// Remove multiple consecutive backslashes
	for strings.Contains(path, "\\\\") {
		path = strings.ReplaceAll(path, "\\\\", "\\")
	}

	// Remove leading/trailing backslashes
	path = strings.Trim(path, "\\")

	return path
}

// SanitizeValueName sanitizes a registry value name
func SanitizeValueName(valueName string) string {
	// Remove null bytes and control characters
	valueName = strings.Map(func(r rune) rune {
		if r == '\x00' || r < 32 || r == 127 {
			return -1
		}
		return r
	}, valueName)

	// Trim whitespace
	return strings.TrimSpace(valueName)
}

// getValidRootKeyNames returns a list of valid root key names for error messages
func getValidRootKeyNames() []string {
	keys := make([]string, 0, len(ValidRootKeys))
	seen := make(map[string]bool)

	for k := range ValidRootKeys {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}

	return keys
}

// ValidateConfig validates the entire RegistryConfig structure
func ValidateConfig(config *RegistryConfig) error {
	if config == nil {
		return &ValidationError{
			Field:   "Config",
			Value:   "<nil>",
			Message: "config cannot be nil",
			Code:    ErrCodeEmptyField,
		}
	}

	// Validate version
	if config.Version == "" {
		return &ValidationError{
			Field:   "Version",
			Value:   config.Version,
			Message: "config version cannot be empty",
			Code:    ErrCodeEmptyField,
		}
	}

	// Validate each query
	for i, query := range config.Queries {
		if err := query.Validate(); err != nil {
			return fmt.Errorf("query[%d] (%s) validation failed: %w", i, query.Name, err)
		}
	}

	return nil
}
