package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestLoadRegistryConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
  "version": "1.0",
  "queries": [
    {
      "name": "test_query",
      "description": "Test query",
      "root_key": "HKLM",
      "path": "SOFTWARE\\Test",
      "value_name": "TestValue",
      "operation": "read"
    }
  ]
}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading the config
	config, err := LoadRegistryConfig(configPath)
	if err != nil {
		t.Fatalf("LoadRegistryConfig() error = %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Version = %q, want %q", config.Version, "1.0")
	}

	if len(config.Queries) != 1 {
		t.Fatalf("len(Queries) = %d, want 1", len(config.Queries))
	}

	query := config.Queries[0]
	if query.Name != "test_query" {
		t.Errorf("Query.Name = %q, want %q", query.Name, "test_query")
	}

	if query.RootKey != "HKLM" {
		t.Errorf("Query.RootKey = %q, want %q", query.RootKey, "HKLM")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(configPath, []byte("invalid json{{{"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadRegistryConfig(configPath)
	if err == nil {
		t.Error("LoadRegistryConfig() should return error for invalid JSON")
	}
}

func TestLoadConfig_FileNotExist(t *testing.T) {
	_, err := LoadRegistryConfig("nonexistent_file.json")
	if err == nil {
		t.Error("LoadRegistryConfig() should return error when file doesn't exist")
	}
}

func TestLoadConfig_ActualConfigFile(t *testing.T) {
	// Test with the actual config file in the project
	config, err := LoadRegistryConfig("../configs/registry_operations.json")
	if err != nil {
		t.Skipf("Skipping: actual config file not found: %v", err)
	}

	if config.Version == "" {
		t.Error("Version should not be empty")
	}

	if len(config.Queries) == 0 {
		t.Error("Queries should not be empty")
	}

	// Verify each query has required fields
	for i, query := range config.Queries {
		if query.Name == "" {
			t.Errorf("Query[%d].Name should not be empty", i)
		}
		if query.Operation == "" {
			t.Errorf("Query[%d].Operation should not be empty", i)
		}
		if query.RootKey == "" {
			t.Errorf("Query[%d].RootKey should not be empty", i)
		}
	}
}

func TestRegistryQuery_WriteTypes(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "write_test.json")

	configContent := `{
  "version": "1.0",
  "queries": [
    {
      "name": "write_string",
      "root_key": "HKCU",
      "path": "Software\\Test",
      "value_name": "StringValue",
      "operation": "write",
      "write_type": "string",
      "write_value": "test string"
    },
    {
      "name": "write_dword",
      "root_key": "HKCU",
      "path": "Software\\Test",
      "value_name": "DwordValue",
      "operation": "write",
      "write_type": "dword",
      "write_value": 42
    },
    {
      "name": "write_multi_string",
      "root_key": "HKCU",
      "path": "Software\\Test",
      "value_name": "MultiString",
      "operation": "write",
      "write_type": "multi_string",
      "write_value": ["string1", "string2"]
    }
  ]
}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	config, err := LoadRegistryConfig(configPath)
	if err != nil {
		t.Fatalf("LoadRegistryConfig() error = %v", err)
	}

	// Verify write configurations
	if len(config.Queries) != 3 {
		t.Fatalf("Expected 3 queries, got %d", len(config.Queries))
	}

	// Test string write config
	if config.Queries[0].WriteType != "string" {
		t.Errorf("Query[0].WriteType = %q, want %q", config.Queries[0].WriteType, "string")
	}

	// Test dword write config
	if config.Queries[1].WriteType != "dword" {
		t.Errorf("Query[1].WriteType = %q, want %q", config.Queries[1].WriteType, "dword")
	}

	// Test multi_string write config
	if config.Queries[2].WriteType != "multi_string" {
		t.Errorf("Query[2].WriteType = %q, want %q", config.Queries[2].WriteType, "multi_string")
	}
}

func TestParseRootKey_AllKeys(t *testing.T) {
	tests := []struct {
		short string
		full  string
		want  registry.Key
	}{
		{"HKLM", "HKEY_LOCAL_MACHINE", registry.LOCAL_MACHINE},
		{"HKCU", "HKEY_CURRENT_USER", registry.CURRENT_USER},
		{"HKCR", "HKEY_CLASSES_ROOT", registry.CLASSES_ROOT},
		{"HKU", "HKEY_USERS", registry.USERS},
		{"HKCC", "HKEY_CURRENT_CONFIG", registry.CURRENT_CONFIG},
	}

	for _, tt := range tests {
		t.Run(tt.short, func(t *testing.T) {
			// Test short form
			got, err := ParseRootKey(tt.short)
			if err != nil {
				t.Errorf("ParseRootKey(%q) error = %v", tt.short, err)
			}
			if got != tt.want {
				t.Errorf("ParseRootKey(%q) = %v, want %v", tt.short, got, tt.want)
			}

			// Test full form
			got, err = ParseRootKey(tt.full)
			if err != nil {
				t.Errorf("ParseRootKey(%q) error = %v", tt.full, err)
			}
			if got != tt.want {
				t.Errorf("ParseRootKey(%q) = %v, want %v", tt.full, got, tt.want)
			}
		})
	}
}
