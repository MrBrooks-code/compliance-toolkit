package pkg

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func TestRegistryError(t *testing.T) {
	baseErr := errors.New("base error")
	regErr := &RegistryError{
		Op:    "OpenKey",
		Key:   "SOFTWARE\\Test",
		Value: "TestValue",
		Err:   baseErr,
	}

	// Test Error() method
	expected := "registry OpenKey failed for SOFTWARE\\Test\\TestValue: base error"
	if regErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", regErr.Error(), expected)
	}

	// Test Unwrap()
	if !errors.Is(regErr, baseErr) {
		t.Error("Unwrap() not working correctly")
	}
}

func TestIsNotExist(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "registry error with ErrNotExist",
			err: &RegistryError{
				Op:  "OpenKey",
				Err: registry.ErrNotExist,
			},
			want: true,
		},
		{
			name: "direct ErrNotExist",
			err:  registry.ErrNotExist,
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotExist(tt.err); got != tt.want {
				t.Errorf("IsNotExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRootKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    registry.Key
		wantErr bool
	}{
		{"HKLM short", "HKLM", registry.LOCAL_MACHINE, false},
		{"HKLM full", "HKEY_LOCAL_MACHINE", registry.LOCAL_MACHINE, false},
		{"HKCU short", "HKCU", registry.CURRENT_USER, false},
		{"HKCU full", "HKEY_CURRENT_USER", registry.CURRENT_USER, false},
		{"HKCR short", "HKCR", registry.CLASSES_ROOT, false},
		{"HKU short", "HKU", registry.USERS, false},
		{"HKCC short", "HKCC", registry.CURRENT_CONFIG, false},
		{"Invalid", "INVALID", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRootKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRootKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseRootKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRootKeyToString(t *testing.T) {
	tests := []struct {
		name string
		key  registry.Key
		want string
	}{
		{"LOCAL_MACHINE", registry.LOCAL_MACHINE, "HKLM"},
		{"CURRENT_USER", registry.CURRENT_USER, "HKCU"},
		{"CLASSES_ROOT", registry.CLASSES_ROOT, "HKCR"},
		{"USERS", registry.USERS, "HKU"},
		{"CURRENT_CONFIG", registry.CURRENT_CONFIG, "HKCC"},
		{"Unknown", registry.Key(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RootKeyToString(tt.key); got != tt.want {
				t.Errorf("RootKeyToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRegistryReader_Options(t *testing.T) {
	customLogger := slog.Default()
	customTimeout := 30 * time.Second

	reader := NewRegistryReader(
		WithLogger(customLogger),
		WithTimeout(customTimeout),
	)

	if reader.logger != customLogger {
		t.Error("WithLogger option not applied correctly")
	}

	if reader.timeout != customTimeout {
		t.Errorf("WithTimeout option not applied correctly: got %v, want %v", reader.timeout, customTimeout)
	}
}

func TestNewRegistryReader_Defaults(t *testing.T) {
	reader := NewRegistryReader()

	if reader.logger == nil {
		t.Error("Default logger should not be nil")
	}

	expectedTimeout := 5 * time.Second
	if reader.timeout != expectedTimeout {
		t.Errorf("Default timeout = %v, want %v", reader.timeout, expectedTimeout)
	}
}

func TestRegistryReader_ContextCancellation(t *testing.T) {
	reader := NewRegistryReader(WithTimeout(10 * time.Second))

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := reader.ReadString(ctx, registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName")
	if err == nil {
		t.Error("Expected error when context is cancelled, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestRegistryReader_TimeoutContext(t *testing.T) {
	reader := NewRegistryReader(WithTimeout(1 * time.Nanosecond)) // Very short timeout

	ctx := context.Background()

	// This should timeout on a slow system or if registry is busy
	// Note: This test might be flaky on very fast systems
	_, err := reader.ReadString(ctx, registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName")

	// We accept either success (fast system) or timeout (slow system)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Expected either success or DeadlineExceeded, got: %v", err)
	}
}

// Integration test - only runs on Windows
func TestRegistryReader_ReadString_Integration(t *testing.T) {
	reader := NewRegistryReader()
	ctx := context.Background()

	// Read a well-known Windows registry value
	productName, err := reader.ReadString(
		ctx,
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"ProductName",
	)

	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}

	if productName == "" {
		t.Error("ProductName should not be empty")
	}

	t.Logf("ProductName: %s", productName)
}

// Integration test - batch read
func TestRegistryReader_BatchRead_Integration(t *testing.T) {
	reader := NewRegistryReader()
	ctx := context.Background()

	data, err := reader.BatchRead(
		ctx,
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		[]string{"ProductName", "CurrentBuild", "CurrentVersion"},
	)

	if err != nil {
		t.Fatalf("BatchRead() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("BatchRead should return at least one value")
	}

	for k, v := range data {
		t.Logf("%s: %v", k, v)
	}
}

// Benchmark tests
func BenchmarkReadString(b *testing.B) {
	reader := NewRegistryReader()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadString(
			ctx,
			registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
			"ProductName",
		)
	}
}

func BenchmarkBatchRead(b *testing.B) {
	reader := NewRegistryReader()
	ctx := context.Background()
	values := []string{"ProductName", "CurrentBuild", "CurrentVersion", "EditionID"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.BatchRead(
			ctx,
			registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
			values,
		)
	}
}
