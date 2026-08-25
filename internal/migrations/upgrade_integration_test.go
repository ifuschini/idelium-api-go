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

func TestVerifyLaravelUpgradeIntegration(t *testing.T) {
	database := openUpgradeIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS migrations",
		"CREATE TABLE migrations (id BIGINT PRIMARY KEY AUTO_INCREMENT, migration VARCHAR(255) NOT NULL UNIQUE, batch INT NOT NULL)",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare upgrade migration fixture %q: %v", statement, err)
		}
	}
	if _, err := migrations.MarkReviewedBaselineApplied(ctx, database, migrations.BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	}); err != nil {
		t.Fatalf("seed reviewed baseline markers: %v", err)
	}

	verification, err := migrations.VerifyLaravelUpgrade(
		ctx,
		migrations.NewMySQLMigrationTableInspector(database),
		"laravel-legacy",
	)
	if err != nil {
		t.Fatalf("VerifyLaravelUpgrade() returned an error: %v", err)
	}
	if verification.Status != "blocked" || len(verification.MissingMigrations) != 0 {
		t.Fatalf("unexpected complete-upgrade verification: %#v", verification)
	}

	if _, err := database.ExecContext(ctx, "INSERT INTO migrations (migration, batch) VALUES ('future_laravel_marker', 68)"); err != nil {
		t.Fatalf("seed unexpected marker: %v", err)
	}
	verification, err = migrations.VerifyLaravelUpgrade(
		ctx,
		migrations.NewMySQLMigrationTableInspector(database),
		"laravel-legacy",
	)
	if err != nil {
		t.Fatalf("VerifyLaravelUpgrade() returned an error with drift: %v", err)
	}
	if verification.Status != "review-required" || len(verification.UnexpectedMarkers) != 1 {
		t.Fatalf("unexpected drift verification: %#v", verification)
	}
}

func openUpgradeIntegrationDatabase(t *testing.T) *sql.DB {
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
