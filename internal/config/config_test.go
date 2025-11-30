package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Success with minimal required config",
			envVars: map[string]string{
				"API_KEY": "test-api-key",
			},
			expectError: false,
		},
		{
			name: "Success with all config specified",
			envVars: map[string]string{
				"SERVER_HOST":          "localhost",
				"SERVER_PORT":          "9090",
				"DB_HOST":              "db.example.com",
				"DB_PORT":              "5433",
				"DB_USER":              "testuser",
				"DB_PASSWORD":          "testpass",
				"DB_NAME":              "testdb",
				"DB_MAX_CONNECTIONS":   "50",
				"DB_MIN_CONNECTIONS":   "10",
				"DB_MAX_CONN_LIFETIME": "600",
				"LOG_LEVEL":            "debug",
				"LOG_FORMAT":           "console",
				"API_KEY":              "test-key-123",
			},
			expectError: false,
		},
		{
			name: "Error - missing API key",
			envVars: map[string]string{
				"API_KEY": "",
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name: "Error - invalid server port",
			envVars: map[string]string{
				"SERVER_PORT": "99999",
				"API_KEY":     "test-key",
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "Error - invalid log level",
			envVars: map[string]string{
				"LOG_LEVEL": "invalid",
				"API_KEY":   "test-key",
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
		{
			name: "Error - invalid log format",
			envVars: map[string]string{
				"LOG_FORMAT": "xml",
				"API_KEY":    "test-key",
			},
			expectError: true,
			errorMsg:    "invalid log format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
			}

			// Clean up
			os.Clearenv()
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Password:        "password",
					Database:        "testdb",
					MaxConnections:  25,
					MinConnections:  5,
					MaxConnLifetime: 300,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: false,
		},
		{
			name: "Invalid - server port too high",
			config: &Config{
				Server: ServerConfig{
					Port: 99999,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           5432,
					User:           "postgres",
					Database:       "testdb",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "Invalid - database port zero",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           0,
					User:           "postgres",
					Database:       "testdb",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "invalid database port",
		},
		{
			name: "Invalid - empty database host",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "",
					Port:           5432,
					User:           "postgres",
					Database:       "testdb",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "database host is required",
		},
		{
			name: "Invalid - empty database user",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           5432,
					User:           "",
					Database:       "testdb",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "database user is required",
		},
		{
			name: "Invalid - empty database name",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           5432,
					User:           "postgres",
					Database:       "",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "database name is required",
		},
		{
			name: "Invalid - min connections exceeds max",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           5432,
					User:           "postgres",
					Database:       "testdb",
					MaxConnections: 5,
					MinConnections: 10,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "test-key",
				},
			},
			expectError: true,
			errorMsg:    "min connections cannot exceed max connections",
		},
		{
			name: "Invalid - empty API key",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:           "localhost",
					Port:           5432,
					User:           "postgres",
					Database:       "testdb",
					MaxConnections: 25,
					MinConnections: 5,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: AuthConfig{
					APIKey: "",
				},
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	assert.Equal(t, expected, cfg.ConnectionString())
}

func TestServerConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected string
	}{
		{
			name: "Standard configuration",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			expected: "localhost:8080",
		},
		{
			name: "All interfaces",
			config: ServerConfig{
				Host: "0.0.0.0",
				Port: 9090,
			},
			expected: "0.0.0.0:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.Address())
		})
	}
}

func TestGetEnv(t *testing.T) {
	os.Clearenv()

	// Test with environment variable set
	os.Setenv("TEST_VAR", "test_value")
	assert.Equal(t, "test_value", getEnv("TEST_VAR", "default"))

	// Test with environment variable not set
	assert.Equal(t, "default", getEnv("NON_EXISTENT_VAR", "default"))

	os.Clearenv()
}

func TestGetEnvAsInt(t *testing.T) {
	os.Clearenv()

	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	assert.Equal(t, 42, getEnvAsInt("TEST_INT", 10))

	// Test with invalid integer (should return default)
	os.Setenv("TEST_INVALID", "not_a_number")
	assert.Equal(t, 10, getEnvAsInt("TEST_INVALID", 10))

	// Test with non-existent variable
	assert.Equal(t, 10, getEnvAsInt("NON_EXISTENT_INT", 10))

	os.Clearenv()
}
