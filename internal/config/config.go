package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains validated runtime configuration.
type Config struct {
	Environment string
	HTTP        HTTPConfig
	Database    DatabaseConfig
}

// HTTPConfig contains bounded HTTP server settings.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// DatabaseConfig contains MySQL connectivity and pool settings.
type DatabaseConfig struct {
	Host                  string
	Port                  int
	Name                  string
	User                  string
	Password              string
	TLSMode               string
	ConnectTimeout        time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

// Load reads and validates configuration without exposing secret values.
func Load() (Config, error) {
	password, err := readSecret(
		firstNonEmpty("IDELIUM_DB_PASSWORD_FILE", "DB_PASSWORD_FILE"),
		firstNonEmpty("IDELIUM_DB_PASSWORD", "DB_PASSWORD"),
	)
	if err != nil {
		return Config{}, fmt.Errorf("database password configuration is invalid: %w", err)
	}

	config := Config{
		Environment: valueOrDefault("IDELIUM_ENVIRONMENT", valueOrDefault("APP_ENV", "production")),
		HTTP: HTTPConfig{
			Address:           valueOrDefault("IDELIUM_HTTP_ADDRESS", ":8080"),
			ReadHeaderTimeout: durationOrDefault("IDELIUM_HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       durationOrDefault("IDELIUM_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      durationOrDefault("IDELIUM_HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       durationOrDefault("IDELIUM_HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout:   durationOrDefault("IDELIUM_HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:                  firstNonEmptyOrDefault("127.0.0.1", "IDELIUM_DB_HOST", "DB_HOST"),
			Port:                  intOrDefault("IDELIUM_DB_PORT", intFromFallback("DB_PORT", 3306)),
			Name:                  firstNonEmptyOrDefault("ideliumdb", "IDELIUM_DB_NAME", "DB_DATABASE"),
			User:                  firstNonEmptyOrDefault("idelium", "IDELIUM_DB_USER", "DB_USERNAME"),
			Password:              password,
			TLSMode:               valueOrDefault("IDELIUM_DB_TLS_MODE", "false"),
			ConnectTimeout:        durationOrDefault("IDELIUM_DB_CONNECT_TIMEOUT", 5*time.Second),
			ReadTimeout:           durationOrDefault("IDELIUM_DB_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:          durationOrDefault("IDELIUM_DB_WRITE_TIMEOUT", 10*time.Second),
			MaxOpenConnections:    intOrDefault("IDELIUM_DB_MAX_OPEN_CONNECTIONS", 25),
			MaxIdleConnections:    intOrDefault("IDELIUM_DB_MAX_IDLE_CONNECTIONS", 10),
			ConnectionMaxLifetime: durationOrDefault("IDELIUM_DB_CONNECTION_MAX_LIFETIME", 5*time.Minute),
		},
	}

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (config Config) validate() error {
	if strings.TrimSpace(config.HTTP.Address) == "" {
		return errors.New("IDELIUM_HTTP_ADDRESS must not be empty")
	}
	if config.HTTP.ReadHeaderTimeout <= 0 || config.HTTP.ReadTimeout <= 0 ||
		config.HTTP.WriteTimeout <= 0 || config.HTTP.IdleTimeout <= 0 ||
		config.HTTP.ShutdownTimeout <= 0 {
		return errors.New("HTTP timeouts must be positive")
	}
	if strings.TrimSpace(config.Database.Host) == "" || strings.TrimSpace(config.Database.Name) == "" ||
		strings.TrimSpace(config.Database.User) == "" {
		return errors.New("database host, name, and user must not be empty")
	}
	if config.Database.Password == "" {
		return errors.New("database password must be provided through a secret file or environment variable")
	}
	if config.Database.Port < 1 || config.Database.Port > 65535 {
		return errors.New("database port must be between 1 and 65535")
	}
	if config.Database.MaxOpenConnections < 1 || config.Database.MaxIdleConnections < 0 ||
		config.Database.MaxIdleConnections > config.Database.MaxOpenConnections {
		return errors.New("database connection pool limits are invalid")
	}
	if config.Database.ConnectTimeout <= 0 || config.Database.ReadTimeout <= 0 ||
		config.Database.WriteTimeout <= 0 || config.Database.ConnectionMaxLifetime <= 0 {
		return errors.New("database timeouts must be positive")
	}
	if config.Database.TLSMode != "false" && config.Database.TLSMode != "true" &&
		config.Database.TLSMode != "preferred" {
		return errors.New("IDELIUM_DB_TLS_MODE must be false, true, or preferred")
	}

	return nil
}

func readSecret(filePath string, fallback string) (string, error) {
	if filePath == "" {
		return fallback, nil
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("secret file is not readable: %w", err)
	}

	value := strings.TrimRight(string(contents), "\r\n")
	if value == "" {
		return "", errors.New("secret file is empty")
	}

	return value, nil
}

func firstNonEmpty(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}

	return ""
}

func firstNonEmptyOrDefault(fallback string, names ...string) string {
	if value := firstNonEmpty(names...); value != "" {
		return value
	}

	return fallback
}

func valueOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func durationOrDefault(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}

	return duration
}

func intOrDefault(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return parsed
}

func intFromFallback(name string, fallback int) int {
	return intOrDefault(name, fallback)
}
