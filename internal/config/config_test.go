package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPrefersSecretFileOverEnvironmentFallback(t *testing.T) {
	clearConfigurationEnvironment(t)

	secretPath := filepath.Join(t.TempDir(), "db-password")
	if err := os.WriteFile(secretPath, []byte("file-password\n"), 0o600); err != nil {
		t.Fatalf("write test secret: %v", err)
	}
	t.Setenv("IDELIUM_DB_PASSWORD_FILE", secretPath)
	t.Setenv("IDELIUM_DB_PASSWORD", "environment-password")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if loaded.Database.Password != "file-password" {
		t.Fatal("Load() did not give the secret file precedence")
	}
}

func TestLoadRedactsUnreadableSecretPath(t *testing.T) {
	clearConfigurationEnvironment(t)

	sensitivePath := filepath.Join(t.TempDir(), "customer-secret-password")
	t.Setenv("IDELIUM_DB_PASSWORD_FILE", sensitivePath)
	t.Setenv("IDELIUM_DB_PASSWORD", "fallback-must-not-be-printed")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an unreadable secret file")
	}
	diagnostic := err.Error()
	if strings.Contains(diagnostic, sensitivePath) ||
		strings.Contains(diagnostic, "fallback-must-not-be-printed") {
		t.Fatalf("configuration diagnostic exposed sensitive input: %s", diagnostic)
	}
}

func TestLoadReadsDatabasePasswordFromSecretFile(t *testing.T) {
	clearConfigurationEnvironment(t)

	secretPath := filepath.Join(t.TempDir(), "db-password")
	if err := os.WriteFile(secretPath, []byte("safe-test-password\n"), 0o600); err != nil {
		t.Fatalf("write test secret: %v", err)
	}
	t.Setenv("IDELIUM_DB_PASSWORD_FILE", secretPath)

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if loaded.Database.Password != "safe-test-password" {
		t.Fatal("Load() did not read the expected secret value")
	}
	if loaded.Database.Host != "127.0.0.1" || loaded.Database.Port != 3306 {
		t.Fatalf("unexpected database defaults: %s:%d", loaded.Database.Host, loaded.Database.Port)
	}
}

func TestLoadRejectsMissingDatabasePassword(t *testing.T) {
	clearConfigurationEnvironment(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted a missing database password")
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	clearConfigurationEnvironment(t)
	t.Setenv("IDELIUM_DB_PASSWORD", "safe-test-password")
	t.Setenv("IDELIUM_HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an invalid timeout")
	}
}

func TestLoadPrefersIdeliumDatabaseVariables(t *testing.T) {
	clearConfigurationEnvironment(t)
	t.Setenv("DB_HOST", "legacy-db")
	t.Setenv("IDELIUM_DB_HOST", "go-db")
	t.Setenv("IDELIUM_DB_PASSWORD", "safe-test-password")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if loaded.Database.Host != "go-db" {
		t.Fatalf("expected Idelium-specific host, got %q", loaded.Database.Host)
	}
}

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"APP_ENV",
		"DB_HOST",
		"DB_PORT",
		"DB_DATABASE",
		"DB_USERNAME",
		"DB_PASSWORD",
		"DB_PASSWORD_FILE",
		"IDELIUM_ENVIRONMENT",
		"IDELIUM_HTTP_ADDRESS",
		"IDELIUM_HTTP_READ_HEADER_TIMEOUT",
		"IDELIUM_HTTP_READ_TIMEOUT",
		"IDELIUM_HTTP_WRITE_TIMEOUT",
		"IDELIUM_HTTP_IDLE_TIMEOUT",
		"IDELIUM_HTTP_SHUTDOWN_TIMEOUT",
		"IDELIUM_DB_HOST",
		"IDELIUM_DB_PORT",
		"IDELIUM_DB_NAME",
		"IDELIUM_DB_USER",
		"IDELIUM_DB_PASSWORD",
		"IDELIUM_DB_PASSWORD_FILE",
		"IDELIUM_DB_TLS_MODE",
		"IDELIUM_DB_CONNECT_TIMEOUT",
		"IDELIUM_DB_READ_TIMEOUT",
		"IDELIUM_DB_WRITE_TIMEOUT",
		"IDELIUM_DB_MAX_OPEN_CONNECTIONS",
		"IDELIUM_DB_MAX_IDLE_CONNECTIONS",
		"IDELIUM_DB_CONNECTION_MAX_LIFETIME",
	} {
		t.Setenv(name, "")
	}
}
