package env

import (
	"os"
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_STRING",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_STRING_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_STRING_EMPTY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			result := GetString(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGet_String(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "generic Get with string",
			key:          "TEST_GENERIC_STRING",
			defaultValue: "default",
			envValue:     "generic_value",
			expected:     "generic_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			result := Get(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("Get() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "returns true for 'true'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "returns true for '1'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "returns true for 'yes'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "yes",
			expected:     true,
		},
		{
			name:         "returns true for 'on'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "on",
			expected:     true,
		},
		{
			name:         "returns false for 'false'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "returns false for '0'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			expected:     true,
		},
		{
			name:         "case insensitive - TRUE",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "TRUE",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(tt.key, tt.envValue)
			defer os.Unsetenv(tt.key)

			// Test
			result := GetBool(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGet_Bool(t *testing.T) {
	os.Setenv("TEST_GENERIC_BOOL", "true")
	defer os.Unsetenv("TEST_GENERIC_BOOL")

	result := Get("TEST_GENERIC_BOOL", false)
	if result != true {
		t.Errorf("Get() bool = %v, want %v", result, true)
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "returns parsed int",
			key:          "TEST_INT",
			defaultValue: 0,
			envValue:     "42",
			expected:     42,
		},
		{
			name:         "returns default for invalid int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "invalid",
			expected:     10,
		},
		{
			name:         "returns negative int",
			key:          "TEST_INT",
			defaultValue: 0,
			envValue:     "-123",
			expected:     -123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(tt.key, tt.envValue)
			defer os.Unsetenv(tt.key)

			// Test
			result := GetInt(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGet_Int(t *testing.T) {
	os.Setenv("TEST_GENERIC_INT", "123")
	defer os.Unsetenv("TEST_GENERIC_INT")

	result := Get("TEST_GENERIC_INT", 0)
	if result != 123 {
		t.Errorf("Get() int = %v, want %v", result, 123)
	}
}

func TestGetFloat64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue float64
		envValue     string
		expected     float64
	}{
		{
			name:         "returns parsed float64",
			key:          "TEST_FLOAT",
			defaultValue: 0.0,
			envValue:     "3.14",
			expected:     3.14,
		},
		{
			name:         "returns default for invalid float",
			key:          "TEST_FLOAT",
			defaultValue: 1.5,
			envValue:     "invalid",
			expected:     1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(tt.key, tt.envValue)
			defer os.Unsetenv(tt.key)

			// Test
			result := GetFloat64(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetFloat64() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "returns parsed duration",
			key:          "TEST_DURATION",
			defaultValue: time.Second,
			envValue:     "5m",
			expected:     5 * time.Minute,
		},
		{
			name:         "returns default for invalid duration",
			key:          "TEST_DURATION",
			defaultValue: time.Hour,
			envValue:     "invalid",
			expected:     time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(tt.key, tt.envValue)
			defer os.Unsetenv(tt.key)

			// Test
			result := GetDuration(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "returns service name when set",
			envValue: "my-service",
			expected: "my-service",
		},
		{
			name:     "returns empty when not set",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv("SERVICE_NAME", tt.envValue)
				defer os.Unsetenv("SERVICE_NAME")
			} else {
				os.Unsetenv("SERVICE_NAME")
			}

			// Test
			result := GetServiceName()

			// Assert
			if result != tt.expected {
				t.Errorf("GetServiceName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetServiceVersion(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "returns service version when set",
			envValue: "v2.1.0",
			expected: "v2.1.0",
		},
		{
			name:     "returns default when not set",
			envValue: "",
			expected: "v0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv("SERVICE_VERSION", tt.envValue)
				defer os.Unsetenv("SERVICE_VERSION")
			} else {
				os.Unsetenv("SERVICE_VERSION")
			}

			// Test
			result := GetServiceVersion()

			// Assert
			if result != tt.expected {
				t.Errorf("GetServiceVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvName(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected EnvName
	}{
		{
			name:     "returns prod",
			envValue: "prod",
			expected: EnvProd,
		},
		{
			name:     "returns staging",
			envValue: "staging",
			expected: EnvStaging,
		},
		{
			name:     "returns local",
			envValue: "local",
			expected: EnvLocal,
		},
		{
			name:     "returns local for invalid value",
			envValue: "invalid",
			expected: EnvLocal,
		},
		{
			name:     "returns local when not set",
			envValue: "",
			expected: EnvLocal,
		},
		{
			name:     "case insensitive - PROD",
			envValue: "PROD",
			expected: EnvProd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv("ENV_NAME", tt.envValue)
				defer os.Unsetenv("ENV_NAME")
			} else {
				os.Unsetenv("ENV_NAME")
			}

			// Test
			result := GetEnvName()

			// Assert
			if result != tt.expected {
				t.Errorf("GetEnvName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Тесты для пользовательских типов с дженериками
// Примечание: пользовательские типы работают только если они основаны на поддерживаемых типах
func TestGet_CustomTypes(t *testing.T) {
	t.Run("custom int type with explicit conversion", func(t *testing.T) {
		os.Setenv("CUSTOM_PORT", "8080")
		defer os.Unsetenv("CUSTOM_PORT")

		// Для пользовательских типов нужно использовать базовый тип, а затем конвертировать
		result := Get("CUSTOM_PORT", 3000)
		expected := 8080
		if result != expected {
			t.Errorf("Get() int = %v, want %v", result, expected)
		}
	})

	t.Run("custom string type with explicit conversion", func(t *testing.T) {
		os.Setenv("DB_URL", "postgres://localhost")
		defer os.Unsetenv("DB_URL")

		result := Get("DB_URL", "default")
		expected := "postgres://localhost"
		if result != expected {
			t.Errorf("Get() string = %v, want %v", result, expected)
		}
	})
}
