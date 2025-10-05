package pkg

import (
	"golang.org/x/sys/windows/registry"
)

// ConfigServiceImpl implements ConfigService interface
type ConfigServiceImpl struct{}

// NewConfigService creates a new config service
func NewConfigService() ConfigService {
	return &ConfigServiceImpl{}
}

// LoadConfig loads a configuration from file
func (cs *ConfigServiceImpl) LoadConfig(path string) (*RegistryConfig, error) {
	return LoadConfig(path) // Delegate to existing function
}

// ParseRootKey parses a root key string to registry.Key
func (cs *ConfigServiceImpl) ParseRootKey(rootKeyStr string) (registry.Key, error) {
	return ParseRootKey(rootKeyStr) // Delegate to existing function
}
