package pkg

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// RegistryConfig represents the JSON configuration structure
type RegistryConfig struct {
	Version  string          `json:"version"`
	Metadata ReportMetadata  `json:"metadata"`
	Queries  []RegistryQuery `json:"queries"`
}

// ReportMetadata contains report identification and versioning
type ReportMetadata struct {
	ReportTitle   string `json:"report_title"`
	ReportVersion string `json:"report_version"`
	Author        string `json:"author,omitempty"`
	Description   string `json:"description,omitempty"`
	Category      string `json:"category,omitempty"`
	LastUpdated   string `json:"last_updated,omitempty"`
	Compliance    string `json:"compliance,omitempty"` // e.g., "HIPAA", "PCI DSS", "SOC 2"
}

// RegistryQuery represents a single registry operation
type RegistryQuery struct {
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	RootKey       string      `json:"root_key"`
	Path          string      `json:"path"`
	ValueName     string      `json:"value_name,omitempty"`
	Operation     string      `json:"operation"`
	ReadAll       bool        `json:"read_all,omitempty"`
	WriteType     string      `json:"write_type,omitempty"`
	WriteValue    interface{} `json:"write_value,omitempty"`
	ExpectedValue string      `json:"expected_value,omitempty"` // For compliance reporting
}

// LoadRegistryConfig loads registry operations from a JSON file (renamed to avoid conflict)
func LoadRegistryConfig(path string) (*RegistryConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RegistryConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &config, nil
}

// ParseRootKey converts string root key to registry.Key
func ParseRootKey(rootKey string) (registry.Key, error) {
	switch rootKey {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, nil
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, nil
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, nil
	case "HKU", "HKEY_USERS":
		return registry.USERS, nil
	case "HKCC", "HKEY_CURRENT_CONFIG":
		return registry.CURRENT_CONFIG, nil
	default:
		return 0, fmt.Errorf("unknown root key: %s", rootKey)
	}
}

// RootKeyToString converts registry.Key to string representation
func RootKeyToString(key registry.Key) string {
	switch key {
	case registry.LOCAL_MACHINE:
		return "HKLM"
	case registry.CURRENT_USER:
		return "HKCU"
	case registry.CLASSES_ROOT:
		return "HKCR"
	case registry.USERS:
		return "HKU"
	case registry.CURRENT_CONFIG:
		return "HKCC"
	default:
		return "UNKNOWN"
	}
}
