package mocks

import (
	"context"

	"golang.org/x/sys/windows/registry"
)

// MockRegistryService is a mock implementation of RegistryService for testing
type MockRegistryService struct {
	ReadStringFunc  func(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	ReadIntegerFunc func(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error)
	ReadBinaryFunc  func(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error)
	ReadStringsFunc func(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error)
	ReadValueFunc   func(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error)
	BatchReadFunc   func(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error)
}

// ReadString mocks the ReadString method
func (m *MockRegistryService) ReadString(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
	if m.ReadStringFunc != nil {
		return m.ReadStringFunc(ctx, rootKey, path, valueName)
	}
	return "", nil
}

// ReadInteger mocks the ReadInteger method
func (m *MockRegistryService) ReadInteger(ctx context.Context, rootKey registry.Key, path, valueName string) (uint64, error) {
	if m.ReadIntegerFunc != nil {
		return m.ReadIntegerFunc(ctx, rootKey, path, valueName)
	}
	return 0, nil
}

// ReadBinary mocks the ReadBinary method
func (m *MockRegistryService) ReadBinary(ctx context.Context, rootKey registry.Key, path, valueName string) ([]byte, error) {
	if m.ReadBinaryFunc != nil {
		return m.ReadBinaryFunc(ctx, rootKey, path, valueName)
	}
	return nil, nil
}

// ReadStrings mocks the ReadStrings method
func (m *MockRegistryService) ReadStrings(ctx context.Context, rootKey registry.Key, path, valueName string) ([]string, error) {
	if m.ReadStringsFunc != nil {
		return m.ReadStringsFunc(ctx, rootKey, path, valueName)
	}
	return nil, nil
}

// ReadValue mocks the ReadValue method
func (m *MockRegistryService) ReadValue(ctx context.Context, rootKey registry.Key, path, valueName string) (string, error) {
	if m.ReadValueFunc != nil {
		return m.ReadValueFunc(ctx, rootKey, path, valueName)
	}
	return "", nil
}

// BatchRead mocks the BatchRead method
func (m *MockRegistryService) BatchRead(ctx context.Context, rootKey registry.Key, path string, values []string) (map[string]interface{}, error) {
	if m.BatchReadFunc != nil {
		return m.BatchReadFunc(ctx, rootKey, path, values)
	}
	return make(map[string]interface{}), nil
}
