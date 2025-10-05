package mocks

import (
	"compliancetoolkit/pkg"

	"golang.org/x/sys/windows/registry"
)

// MockConfigService is a mock implementation of ConfigService for testing
type MockConfigService struct {
	LoadConfigFunc   func(path string) (*pkg.RegistryConfig, error)
	ParseRootKeyFunc func(rootKeyStr string) (registry.Key, error)
}

// LoadConfig mocks the LoadConfig method
func (m *MockConfigService) LoadConfig(path string) (*pkg.RegistryConfig, error) {
	if m.LoadConfigFunc != nil {
		return m.LoadConfigFunc(path)
	}
	return &pkg.RegistryConfig{}, nil
}

// ParseRootKey mocks the ParseRootKey method
func (m *MockConfigService) ParseRootKey(rootKeyStr string) (registry.Key, error) {
	if m.ParseRootKeyFunc != nil {
		return m.ParseRootKeyFunc(rootKeyStr)
	}
	return registry.LOCAL_MACHINE, nil
}
