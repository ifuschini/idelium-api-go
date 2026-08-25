package migrations_test

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/migrations"
	mysqlpersistence "github.com/idelium/idelium-api-go/internal/persistence/mysql"
)

func TestVerifyEmptyInstallIntegration(t *testing.T) {
	database, schema := openEmptyInstallIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS empty_install_probe"); err != nil {
		t.Fatalf("clean empty-install probe table: %v", err)
	}

	verification, err := migrations.VerifyEmptyInstall(
		ctx,
		migrations.NewMySQLSchemaInspector(database, schema),
	)
	if err != nil {
		t.Fatalf("VerifyEmptyInstall() returned an error: %v", err)
	}
	if verification.Status != "blocked" || !verification.SchemaEmpty {
		t.Fatalf("unexpected empty-schema verification: %#v", verification)
	}

	if _, err := database.ExecContext(ctx, "CREATE TABLE empty_install_probe (id BIGINT PRIMARY KEY)"); err != nil {
		t.Fatalf("create empty-install probe table: %v", err)
	}
	defer database.ExecContext(context.Background(), "DROP TABLE IF EXISTS empty_install_probe")

	verification, err = migrations.VerifyEmptyInstall(
		ctx,
		migrations.NewMySQLSchemaInspector(database, schema),
	)
	if err != nil {
		t.Fatalf("VerifyEmptyInstall() returned an error for non-empty schema: %v", err)
	}
	if verification.Status != "failed" || verification.SchemaEmpty {
		t.Fatalf("unexpected non-empty-schema verification: %#v", verification)
	}
}

func openEmptyInstallIntegrationDatabase(t *testing.T) (*sql.DB, string) {
	t.Helper()
	host := os.Getenv("IDELIUM_TEST_MYSQL_HOST")
	if host == "" {
		t.Skip("IDELIUM_TEST_MYSQL_HOST is not configured")
	}
	port := 3306
	if value := os.Getenv("IDELIUM_TEST_MYSQL_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("parse test database port: %v", err)
		}
		port = parsed
	}
	schema := os.Getenv("IDELIUM_TEST_MYSQL_DATABASE")
	database, err := mysqlpersistence.Open(config.DatabaseConfig{
		Host:                  host,
		Port:                  port,
		Name:                  schema,
		User:                  os.Getenv("IDELIUM_TEST_MYSQL_USER"),
		Password:              os.Getenv("IDELIUM_TEST_MYSQL_PASSWORD"),
		TLSMode:               "false",
		ConnectTimeout:        5 * time.Second,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		MaxOpenConnections:    2,
		MaxIdleConnections:    1,
		ConnectionMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("create integration database pool: %v", err)
	}
	return database, schema
}
