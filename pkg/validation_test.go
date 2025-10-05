package pkg

import (
	"strings"
	"testing"
)

// TestValidateRootKey tests root key validation
func TestValidateRootKey(t *testing.T) {
	tests := []struct {
		name    string
		rootKey string
		wantErr bool
		errCode ValidationErrorCode
	}{
		// Valid cases
		{"valid HKLM", "HKLM", false, 0},
		{"valid HKEY_LOCAL_MACHINE", "HKEY_LOCAL_MACHINE", false, 0},
		{"valid HKCU", "HKCU", false, 0},
		{"valid HKEY_CURRENT_USER", "HKEY_CURRENT_USER", false, 0},
		{"valid HKCR", "HKCR", false, 0},
		{"valid HKU", "HKU", false, 0},
		{"valid HKCC", "HKCC", false, 0},

		// Invalid cases
		{"empty root key", "", true, ErrCodeEmptyField},
		{"invalid root key", "INVALID", true, ErrCodeInvalidRootKey},
		{"wrong case", "hklm", true, ErrCodeInvalidRootKey},
		{"partial name", "HK", true, ErrCodeInvalidRootKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRootKey(tt.rootKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRootKey() expected error, got nil")
					return
				}
				if verr, ok := err.(*ValidationError); ok {
					if verr.Code != tt.errCode {
						t.Errorf("ValidateRootKey() error code = %v, want %v", verr.Code, tt.errCode)
					}
				} else {
					t.Errorf("ValidateRootKey() error is not ValidationError: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRootKey() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateRegistryPath tests registry path validation
func TestValidateRegistryPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errCode ValidationErrorCode
	}{
		// Valid cases
		{"valid simple path", "SOFTWARE\\Microsoft\\Windows", false, 0},
		{"valid path with spaces", "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", false, 0},
		{"valid path with hyphen", "SOFTWARE\\Microsoft-Test", false, 0},
		{"valid path with underscore", "SOFTWARE\\Test_Key", false, 0},
		{"valid path with dots", "SOFTWARE\\Test.Key.Name", false, 0},
		{"valid path with parentheses", "SOFTWARE\\Test(1)", false, 0},

		// Invalid cases - empty/length
		{"empty path", "", true, ErrCodeEmptyField},
		{"too long path", strings.Repeat("A", MaxRegistryPathLength+1), true, ErrCodeTooLong},

		// Invalid cases - characters
		{"path with null byte", "SOFTWARE\x00Microsoft", true, ErrCodeInvalidCharacters},
		{"path with newline", "SOFTWARE\nMicrosoft", true, ErrCodeInvalidCharacters},
		{"path with tab", "SOFTWARE\tMicrosoft", true, ErrCodeInvalidCharacters},
		{"path with forward slash", "SOFTWARE/Microsoft", true, ErrCodeInvalidCharacters},
		{"path with special chars", "SOFTWARE\\Micr@soft", true, ErrCodeInvalidCharacters},

		// Invalid cases - format
		{"path starting with backslash", "\\SOFTWARE\\Microsoft", true, ErrCodeInvalidPath},
		{"path ending with backslash", "SOFTWARE\\Microsoft\\", true, ErrCodeInvalidPath},
		{"path too deep", strings.Repeat("A\\", MaxRegistryKeyDepth+1), true, ErrCodeTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistryPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRegistryPath() expected error, got nil")
					return
				}
				if verr, ok := err.(*ValidationError); ok {
					if verr.Code != tt.errCode {
						t.Errorf("ValidateRegistryPath() error code = %v, want %v", verr.Code, tt.errCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRegistryPath() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateValueName tests value name validation
func TestValidateValueName(t *testing.T) {
	tests := []struct {
		name      string
		valueName string
		wantErr   bool
		errCode   ValidationErrorCode
	}{
		// Valid cases
		{"empty value name (default)", "", false, 0},
		{"valid simple name", "TestValue", false, 0},
		{"valid name with spaces", "Test Value Name", false, 0},
		{"valid name with special chars", "Test-Value_123", false, 0},
		{"valid name with brackets", "Test[Value]", false, 0},

		// Invalid cases
		{"too long value name", strings.Repeat("A", MaxRegistryValueNameLength+1), true, ErrCodeTooLong},
		{"value name with null byte", "Test\x00Value", true, ErrCodeInvalidCharacters},
		{"value name with newline", "Test\nValue", true, ErrCodeInvalidCharacters},
		{"value name with tab", "Test\tValue", true, ErrCodeInvalidCharacters},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValueName(tt.valueName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateValueName() expected error, got nil")
					return
				}
				if verr, ok := err.(*ValidationError); ok {
					if verr.Code != tt.errCode {
						t.Errorf("ValidateValueName() error code = %v, want %v", verr.Code, tt.errCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateValueName() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateNoPathTraversal tests path traversal detection
func TestValidateNoPathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "SOFTWARE\\Microsoft\\Windows", false},
		{"path with dot", "SOFTWARE\\Microsoft.Test", false},
		{"path traversal with backslash", "SOFTWARE\\..\\System", true},
		{"path traversal with forward slash", "SOFTWARE/../System", true},
		{"multiple traversal", "SOFTWARE\\..\\..\\System", true},
		{"hidden traversal", "SOFTWARE\\Test..\\Bad", true}, // Regex matches ..\ anywhere
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoPathTraversal(tt.path)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateNoPathTraversal() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateNoPathTraversal() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateNoInjection tests injection detection
func TestValidateNoInjection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid input", "SOFTWARE\\Microsoft\\Windows", false},
		{"valid with spaces", "Test Value Name", false},
		{"null byte injection", "Test\x00Value", true},
		{"control character", "Test\x01Value", true},
		{"unicode control char", "Test\u0080Value", true},
		{"tab character", "Test\tValue", true},
		{"newline character", "Test\nValue", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoInjection(tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateNoInjection() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateNoInjection() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateAgainstDenyList tests deny list validation
func TestValidateAgainstDenyList(t *testing.T) {
	denyList := []string{
		"SECURITY\\Policy\\Secrets",
		"SAM\\SAM\\Domains\\Account\\Users",
		"SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon\\SpecialAccounts",
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"allowed path", "SOFTWARE\\Microsoft\\Windows", false},
		{"exact match deny", "SECURITY\\Policy\\Secrets", true},
		{"case insensitive deny", "security\\policy\\secrets", true},
		{"subkey of denied path", "SECURITY\\Policy\\Secrets\\SubKey", true},
		{"parent of denied path", "SECURITY\\Policy", false},
		{"similar but not denied", "SECURITY\\Policy\\Other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgainstDenyList(tt.path, denyList)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateAgainstDenyList() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateAgainstDenyList() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateAgainstAllowList tests allow list validation
func TestValidateAgainstAllowList(t *testing.T) {
	allowList := []string{
		"HKEY_LOCAL_MACHINE",
		"HKEY_CURRENT_USER",
	}

	tests := []struct {
		name    string
		rootKey string
		wantErr bool
	}{
		{"allowed root key", "HKEY_LOCAL_MACHINE", false},
		{"allowed HKCU", "HKEY_CURRENT_USER", false},
		{"case insensitive", "hkey_local_machine", false},
		{"not in allow list", "HKEY_USERS", true},
		{"not in allow list HKCR", "HKEY_CLASSES_ROOT", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgainstAllowList(tt.rootKey, allowList)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateAgainstAllowList() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateAgainstAllowList() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateFilePath tests file path validation
func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		allowedExtensions []string
		wantErr           bool
		errCode           ValidationErrorCode
	}{
		// Valid cases
		{"valid json path", "config.json", []string{".json"}, false, 0},
		{"valid path no extension check", "config.json", nil, false, 0},
		{"valid multiple extensions", "report.html", []string{".json", ".html", ".xml"}, false, 0},

		// Invalid cases
		{"empty path", "", nil, true, ErrCodeEmptyField},
		{"path traversal", "../config.json", nil, true, ErrCodePathTraversal},
		{"null byte", "config\x00.json", nil, true, ErrCodeInjectionAttempt},
		{"wrong extension", "config.txt", []string{".json"}, true, ErrCodeInvalidCharacters},
		{"no extension with check", "config", []string{".json"}, true, ErrCodeInvalidCharacters},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path, tt.allowedExtensions)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateFilePath() expected error, got nil")
					return
				}
				if verr, ok := err.(*ValidationError); ok {
					if tt.errCode != 0 && verr.Code != tt.errCode {
						t.Errorf("ValidateFilePath() error code = %v, want %v", verr.Code, tt.errCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFilePath() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestSanitizeRegistryPath tests path sanitization
func TestSanitizeRegistryPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no change needed", "SOFTWARE\\Microsoft\\Windows", "SOFTWARE\\Microsoft\\Windows"},
		{"remove null bytes", "SOFTWARE\x00Microsoft", "SOFTWAREMicrosoft"},
		{"remove control chars", "SOFTWARE\x01\x02Microsoft", "SOFTWAREMicrosoft"},
		{"trim whitespace", "  SOFTWARE\\Microsoft  ", "SOFTWARE\\Microsoft"},
		{"remove double backslash", "SOFTWARE\\\\Microsoft", "SOFTWARE\\Microsoft"},
		{"remove leading backslash", "\\SOFTWARE\\Microsoft", "SOFTWARE\\Microsoft"},
		{"remove trailing backslash", "SOFTWARE\\Microsoft\\", "SOFTWARE\\Microsoft"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeRegistryPath(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeRegistryPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRegistryQueryValidate tests the RegistryQuery.Validate method
func TestRegistryQueryValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   RegistryQuery
		wantErr bool
	}{
		{
			name: "valid query",
			query: RegistryQuery{
				Name:        "test_query",
				Description: "Test query",
				RootKey:     "HKLM",
				Path:        "SOFTWARE\\Microsoft\\Windows",
				ValueName:   "TestValue",
				Operation:   "read",
			},
			wantErr: false,
		},
		{
			name: "invalid root key",
			query: RegistryQuery{
				Name:      "test_query",
				RootKey:   "INVALID",
				Path:      "SOFTWARE\\Microsoft\\Windows",
				Operation: "read",
			},
			wantErr: true,
		},
		{
			name: "invalid path",
			query: RegistryQuery{
				Name:      "test_query",
				RootKey:   "HKLM",
				Path:      "SOFTWARE/../System",
				Operation: "read",
			},
			wantErr: true,
		},
		{
			name: "invalid operation",
			query: RegistryQuery{
				Name:      "test_query",
				RootKey:   "HKLM",
				Path:      "SOFTWARE\\Microsoft\\Windows",
				Operation: "write",
			},
			wantErr: true,
		},
		{
			name: "path with injection",
			query: RegistryQuery{
				Name:      "test_query",
				RootKey:   "HKLM",
				Path:      "SOFTWARE\x00Microsoft",
				Operation: "read",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("RegistryQuery.Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("RegistryQuery.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateConfig tests the ValidateConfig function
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *RegistryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RegistryConfig{
				Version: "1.0",
				Queries: []RegistryQuery{
					{
						Name:      "test_query",
						RootKey:   "HKLM",
						Path:      "SOFTWARE\\Microsoft\\Windows",
						Operation: "read",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty version",
			config: &RegistryConfig{
				Version: "",
				Queries: []RegistryQuery{},
			},
			wantErr: true,
		},
		{
			name: "invalid query",
			config: &RegistryConfig{
				Version: "1.0",
				Queries: []RegistryQuery{
					{
						Name:      "test_query",
						RootKey:   "INVALID",
						Path:      "SOFTWARE\\Microsoft\\Windows",
						Operation: "read",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateConfig() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateConfig() unexpected error: %v", err)
			}
		})
	}
}
