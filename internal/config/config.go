package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Auth     AuthConfig
	S3       S3Config
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database-related configuration.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxConnections  int
	MinConnections  int
	MaxConnLifetime int // seconds
}

// LoggerConfig holds logger-related configuration.
type LoggerConfig struct {
	Level  string
	Format string // "json" or "console"
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	APIKey string
}

// S3Config holds AWS S3 configuration for coupon files.
type S3Config struct {
	Enabled bool
	Bucket  string
	Region  string
	Prefix  string // Path prefix within bucket (e.g., "coupons/")
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			Database:        getEnv("DB_NAME", "minikart"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			MinConnections:  getEnvAsInt("DB_MIN_CONNECTIONS", 5),
			MaxConnLifetime: getEnvAsInt("DB_MAX_CONN_LIFETIME", 300),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Auth: AuthConfig{
			APIKey: getEnv("API_KEY", ""),
		},
		S3: S3Config{
			Enabled: getEnvAsBool("S3_ENABLED", false),
			Bucket:  getEnv("S3_BUCKET", ""),
			Region:  getEnv("S3_REGION", "us-east-1"),
			Prefix:  getEnv("S3_PREFIX", "coupons/"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Database.MaxConnections < 1 {
		return fmt.Errorf("database max connections must be at least 1")
	}

	if c.Database.MinConnections < 1 {
		return fmt.Errorf("database min connections must be at least 1")
	}

	if c.Database.MinConnections > c.Database.MaxConnections {
		return fmt.Errorf("database min connections cannot exceed max connections")
	}

	if c.Auth.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logger.Level)
	}

	if c.Logger.Format != "json" && c.Logger.Format != "console" {
		return fmt.Errorf("invalid log format: %s (must be json or console)", c.Logger.Format)
	}

	if c.S3.Enabled {
		if c.S3.Bucket == "" {
			return fmt.Errorf("S3 bucket is required when S3 is enabled")
		}
		if c.S3.Region == "" {
			return fmt.Errorf("S3 region is required when S3 is enabled")
		}
	}

	return nil
}

// ConnectionString returns the PostgreSQL connection string.
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// Address returns the server address.
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value.
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool retrieves an environment variable as a boolean or returns a default value.
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
