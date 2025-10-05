package pkg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

// RegistryError provides detailed error information for registry operations
type RegistryError struct {
	Op    string // Operation that failed (OpenKey, GetStringValue, etc)
	Key   string // Registry key path
	Value string // Value name
	Err   error  // Underlying error
}

func (e *RegistryError) Error() string {
	return fmt.Sprintf("registry %s failed for %s\\%s: %v", e.Op, e.Key, e.Value, e.Err)
}

func (e *RegistryError) Unwrap() error {
	return e.Err
}

// IsNotExist returns true if the error is because the key or value doesn't exist
func IsNotExist(err error) bool {
	var regErr *RegistryError
	if errors.As(err, &regErr) {
		return errors.Is(regErr.Err, registry.ErrNotExist)
	}
	return errors.Is(err, registry.ErrNotExist)
}

// RegistryReader encapsulates registry operations with context support and observability
type RegistryReader struct {
	logger  *slog.Logger
	timeout time.Duration
}

// RegistryReaderOption configures a RegistryReader
type RegistryReaderOption func(*RegistryReader)

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) RegistryReaderOption {
	return func(r *RegistryReader) {
		r.logger = logger
	}
}

// WithTimeout sets the default timeout for registry operations
func WithTimeout(timeout time.Duration) RegistryReaderOption {
	return func(r *RegistryReader) {
		r.timeout = timeout
	}
}

// NewRegistryReader creates a new RegistryReader instance with options
func NewRegistryReader(opts ...RegistryReaderOption) *RegistryReader {
	r := &RegistryReader{
		logger:  slog.Default(),
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ReadString reads a string value from the registry with context support
func (r *RegistryReader) ReadString(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
	return r.ReadStringWithTimeout(ctx, rootKey, path, valueName, r.timeout)
}

// ReadStringWithTimeout reads a string value with explicit timeout
func (r *RegistryReader) ReadStringWithTimeout(ctx context.Context, rootKey registry.Key, path, valueName string, timeout time.Duration) (string, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("registry read completed",
			slog.String("operation", "ReadString"),
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	// Create timeout context if parent doesn't have deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	type result struct {
		value string
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{"", &RegistryError{
				Op:    "OpenKey",
				Key:   path,
				Value: valueName,
				Err:   err,
			}}
			return
		}
		defer key.Close()

		value, _, err := key.GetStringValue(valueName)
		if err != nil {
			resultCh <- result{"", &RegistryError{
				Op:    "GetStringValue",
				Key:   path,
				Value: valueName,
				Err:   err,
			}}
			return
		}

		resultCh <- result{value, nil}
	}()

	select {
	case <-ctx.Done():
		r.logger.Warn("registry read cancelled",
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Any("error", ctx.Err()),
		)
		return "", fmt.Errorf("registry read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// ReadValue reads any registry value and returns it as a string (auto-detects type)
func (r *RegistryReader) ReadValue(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("registry read completed",
			slog.String("operation", "ReadValue"),
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	timeout := r.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Create timeout context if parent doesn't have deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	type result struct {
		value string
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{"", &RegistryError{
				Op:    "OpenKey",
				Key:   path,
				Value: valueName,
				Err:   err,
			}}
			return
		}
		defer key.Close()

		// Try string first (REG_SZ - most common)
		if value, _, err := key.GetStringValue(valueName); err == nil {
			resultCh <- result{value, nil}
			return
		}

		// Try multi-string (REG_MULTI_SZ)
		if values, _, err := key.GetStringsValue(valueName); err == nil {
			resultCh <- result{strings.Join(values, ", "), nil}
			return
		}

		// Try integer (DWORD/QWORD)
		if value, _, err := key.GetIntegerValue(valueName); err == nil {
			resultCh <- result{fmt.Sprintf("%d", value), nil}
			return
		}

		// Try binary (REG_BINARY)
		if value, _, err := key.GetBinaryValue(valueName); err == nil {
			resultCh <- result{fmt.Sprintf("%x", value), nil}
			return
		}

		// If all fail, return error
		resultCh <- result{"", &RegistryError{
			Op:    "GetValue",
			Key:   path,
			Value: valueName,
			Err:   fmt.Errorf("unable to read value (tried string, multi-string, integer, and binary types)"),
		}}
	}()

	select {
	case <-ctx.Done():
		r.logger.Warn("registry read cancelled",
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Any("error", ctx.Err()),
		)
		return "", fmt.Errorf("registry read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// ReadInteger reads a DWORD/QWORD value from the registry with context support
func (r *RegistryReader) ReadInteger(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("registry read completed",
			slog.String("operation", "ReadInteger"),
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	type result struct {
		value uint64
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{0, &RegistryError{Op: "OpenKey", Key: path, Value: valueName, Err: err}}
			return
		}
		defer key.Close()

		value, _, err := key.GetIntegerValue(valueName)
		if err != nil {
			resultCh <- result{0, &RegistryError{Op: "GetIntegerValue", Key: path, Value: valueName, Err: err}}
			return
		}

		resultCh <- result{value, nil}
	}()

	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("registry read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// ReadBinary reads a binary value from the registry with context support
func (r *RegistryReader) ReadBinary(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("registry read completed",
			slog.String("operation", "ReadBinary"),
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	type result struct {
		value []byte
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{nil, &RegistryError{Op: "OpenKey", Key: path, Value: valueName, Err: err}}
			return
		}
		defer key.Close()

		value, _, err := key.GetBinaryValue(valueName)
		if err != nil {
			resultCh <- result{nil, &RegistryError{Op: "GetBinaryValue", Key: path, Value: valueName, Err: err}}
			return
		}

		resultCh <- result{value, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("registry read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// ReadStrings reads a multi-string value from the registry with context support
func (r *RegistryReader) ReadStrings(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("registry read completed",
			slog.String("operation", "ReadStrings"),
			slog.String("path", path),
			slog.String("value", valueName),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	type result struct {
		value []string
		err   error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{nil, &RegistryError{Op: "OpenKey", Key: path, Value: valueName, Err: err}}
			return
		}
		defer key.Close()

		value, _, err := key.GetStringsValue(valueName)
		if err != nil {
			resultCh <- result{nil, &RegistryError{Op: "GetStringsValue", Key: path, Value: valueName, Err: err}}
			return
		}

		resultCh <- result{value, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("registry read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// BatchRead reads multiple values from the same registry key efficiently
func (r *RegistryReader) BatchRead(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error) {
	start := time.Now()
	defer func() {
		r.logger.Debug("batch registry read completed",
			slog.String("path", path),
			slog.Int("values", len(values)),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	type result struct {
		data map[string]interface{}
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		key, err := registry.OpenKey(rootKey, path, registry.QUERY_VALUE)
		if err != nil {
			resultCh <- result{nil, &RegistryError{Op: "OpenKey", Key: path, Err: err}}
			return
		}
		defer key.Close()

		data := make(map[string]interface{})
		for _, valueName := range values {
			// Try string first
			if val, _, err := key.GetStringValue(valueName); err == nil {
				data[valueName] = val
				continue
			}
			// Try integer
			if val, _, err := key.GetIntegerValue(valueName); err == nil {
				data[valueName] = val
				continue
			}
			// Try binary
			if val, _, err := key.GetBinaryValue(valueName); err == nil {
				data[valueName] = val
				continue
			}
			// Try multi-string
			if val, _, err := key.GetStringsValue(valueName); err == nil {
				data[valueName] = val
			}
		}

		resultCh <- result{data, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("batch read cancelled: %w", ctx.Err())
	case res := <-resultCh:
		return res.data, res.err
	}
}
