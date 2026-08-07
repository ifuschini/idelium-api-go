package mysql

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

func TestDatabaseConnectionIntegration(t *testing.T) {
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

	database, err := Open(config.DatabaseConfig{
		Host:                  host,
		Port:                  port,
		Name:                  os.Getenv("IDELIUM_TEST_MYSQL_DATABASE"),
		User:                  os.Getenv("IDELIUM_TEST_MYSQL_USER"),
		Password:              os.Getenv("IDELIUM_TEST_MYSQL_PASSWORD"),
		TLSMode:               "false",
		ConnectTimeout:        5 * time.Second,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          5 * time.Second,
		MaxOpenConnections:    4,
		MaxIdleConnections:    2,
		ConnectionMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("Open() returned an error: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := Check(ctx, database); err != nil {
		t.Fatalf("Check() returned an error: %v", err)
	}
}

func TestPlatformCatalogRepositoryIntegration(t *testing.T) {
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

	database, err := Open(config.DatabaseConfig{
		Host:                  host,
		Port:                  port,
		Name:                  os.Getenv("IDELIUM_TEST_MYSQL_DATABASE"),
		User:                  os.Getenv("IDELIUM_TEST_MYSQL_USER"),
		Password:              os.Getenv("IDELIUM_TEST_MYSQL_PASSWORD"),
		TLSMode:               "false",
		ConnectTimeout:        5 * time.Second,
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          5 * time.Second,
		MaxOpenConnections:    4,
		MaxIdleConnections:    2,
		ConnectionMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("Open() returned an error: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS types",
		"DROP TABLE IF EXISTS statuses",
		"CREATE TABLE types (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE statuses (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"INSERT INTO types (id, name) VALUES (2, 'mobile'), (1, 'desktop')",
		"INSERT INTO statuses (id, name) VALUES (2, 'busy'), (1, 'free')",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare catalog fixture %q: %v", statement, err)
		}
	}

	repository := NewPlatformCatalogRepository(database)
	types, err := repository.ListTypes(ctx)
	if err != nil {
		t.Fatalf("ListTypes() returned an error: %v", err)
	}
	expectedTypes := []platforms.CatalogItem{{ID: 1, Name: "desktop"}, {ID: 2, Name: "mobile"}}
	if !reflect.DeepEqual(types, expectedTypes) {
		t.Fatalf("unexpected types: %#v", types)
	}

	statuses, err := repository.ListStatuses(ctx)
	if err != nil {
		t.Fatalf("ListStatuses() returned an error: %v", err)
	}
	expectedStatuses := []platforms.CatalogItem{{ID: 1, Name: "free"}, {ID: 2, Name: "busy"}}
	if !reflect.DeepEqual(statuses, expectedStatuses) {
		t.Fatalf("unexpected statuses: %#v", statuses)
	}
}
