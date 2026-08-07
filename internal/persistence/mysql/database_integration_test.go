package mysql

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/health"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

func openIntegrationDatabase(t *testing.T) *sql.DB {
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
	return database
}

func TestReadinessHTTPContractIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	handler := health.NewHandler(integrationChecker{database: database}, buildinfo.Current())
	response := httptest.NewRecorder()
	handler.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected readiness status 200, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("readiness response is cacheable")
	}
}

type integrationChecker struct {
	database *sql.DB
}

func (checker integrationChecker) Check(ctx context.Context) error {
	return Check(ctx, checker.database)
}

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
