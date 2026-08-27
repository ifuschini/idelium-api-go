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

func TestMarkReviewedBaselineAppliedIntegration(t *testing.T) {
	database := openBridgeIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS migrations",
		"CREATE TABLE migrations (id BIGINT PRIMARY KEY AUTO_INCREMENT, migration VARCHAR(255) NOT NULL UNIQUE, batch INT NOT NULL)",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare migrations table fixture %q: %v", statement, err)
		}
	}

	result, err := migrations.MarkReviewedBaselineApplied(ctx, database, migrations.BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	})
	if err != nil {
		t.Fatalf("MarkReviewedBaselineApplied() returned an error: %v", err)
	}
	if result.Applied != 69 || result.Skipped != 0 {
		t.Fatalf("unexpected first bridge result: %#v", result)
	}

	result, err = migrations.MarkReviewedBaselineApplied(ctx, database, migrations.BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	})
	if err != nil {
		t.Fatalf("idempotent MarkReviewedBaselineApplied() returned an error: %v", err)
	}
	if result.Applied != 0 || result.Skipped != 69 {
		t.Fatalf("unexpected idempotent bridge result: %#v", result)
	}

	var count int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM migrations WHERE batch = 67").Scan(&count); err != nil {
		t.Fatalf("count bridge migration markers: %v", err)
	}
	if count != 69 {
		t.Fatalf("unexpected bridge marker count %d", count)
	}
}

func openBridgeIntegrationDatabase(t *testing.T) *sql.DB {
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
	database, err := mysqlpersistence.Open(config.DatabaseConfig{
		Host:                  host,
		Port:                  port,
		Name:                  os.Getenv("IDELIUM_TEST_MYSQL_DATABASE"),
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
	return database
}
