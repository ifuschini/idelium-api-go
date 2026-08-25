package mysql

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/health"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := Open(integrationDatabaseConfig(t))
	if err != nil {
		t.Fatalf("Open() returned an error: %v", err)
	}
	return database
}

func integrationDatabaseConfig(t *testing.T) config.DatabaseConfig {
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
	return config.DatabaseConfig{
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
	}
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
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := Check(ctx, database); err != nil {
		t.Fatalf("Check() returned an error: %v", err)
	}

	var databaseName string
	if err := database.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&databaseName); err != nil {
		t.Fatalf("read current database: %v", err)
	}
	if databaseName != "idelium_test" {
		t.Fatalf("integration test connected to unexpected database %q", databaseName)
	}
}

func TestDatabaseAuthenticationFailureIsRedactedIntegration(t *testing.T) {
	databaseConfig := integrationDatabaseConfig(t)
	databaseConfig.Password = "integration-secret-that-must-not-leak"

	database, err := Open(databaseConfig)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = Check(ctx, database)
	if err == nil {
		t.Fatal("Check() succeeded with invalid credentials")
	}
	if strings.Contains(err.Error(), databaseConfig.Password) || strings.Contains(err.Error(), databaseConfig.User) {
		t.Fatalf("database authentication failure exposed credentials: %v", err)
	}
}

func TestPlatformCatalogRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS types",
		"DROP TABLE IF EXISTS statuses",
		"DROP TABLE IF EXISTS locations",
		"DROP TABLE IF EXISTS brand_devices",
		"DROP TABLE IF EXISTS model_devices",
		"DROP TABLE IF EXISTS os",
		"DROP TABLE IF EXISTS version_os",
		"CREATE TABLE types (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE statuses (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE locations (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE brand_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, brand VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE model_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, model VARCHAR(255) NOT NULL, idBrand INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE os (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, type INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE version_os (id BIGINT PRIMARY KEY AUTO_INCREMENT, version VARCHAR(255) NOT NULL, idOs INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"INSERT INTO types (id, name) VALUES (2, 'mobile'), (1, 'desktop')",
		"INSERT INTO statuses (id, name) VALUES (2, 'busy'), (1, 'free')",
		"INSERT INTO locations (id, name, created_at, updated_at) VALUES (2, 'us-east', NULL, NULL), (1, 'eu-west', NULL, NULL)",
		"INSERT INTO brand_devices (id, brand, created_at, updated_at) VALUES (2, 'Samsung', NULL, NULL), (1, 'Apple', NULL, NULL)",
		"INSERT INTO model_devices (id, model, idBrand, created_at, updated_at) VALUES (3, 'Galaxy', 2, NULL, NULL), (2, 'iPad', 1, NULL, NULL), (1, 'iPhone', 1, NULL, NULL)",
		"INSERT INTO os (id, name, type, created_at, updated_at) VALUES (3, 'android', 2, NULL, NULL), (2, 'windows', 1, NULL, NULL), (1, 'linux', 1, NULL, NULL)",
		"INSERT INTO version_os (id, version, idOs, created_at, updated_at) VALUES (3, '13', 2, NULL, NULL), (2, '15', 1, NULL, NULL), (1, '14', 1, NULL, NULL)",
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

	locations, err := repository.ListLocations(ctx, platforms.LocationQuery{})
	if err != nil {
		t.Fatalf("ListLocations() returned an error: %v", err)
	}
	expectedLocations := []platforms.LocationItem{{ID: 1, Name: "eu-west"}, {ID: 2, Name: "us-east"}}
	if !reflect.DeepEqual(locations.Data, expectedLocations) {
		t.Fatalf("unexpected locations: %#v", locations.Data)
	}

	brands, err := repository.ListBrands(ctx, platforms.BrandQuery{})
	if err != nil {
		t.Fatalf("ListBrands() returned an error: %v", err)
	}
	expectedBrands := []platforms.BrandItem{{ID: 1, Brand: "Apple"}, {ID: 2, Brand: "Samsung"}}
	if !reflect.DeepEqual(brands.Data, expectedBrands) {
		t.Fatalf("unexpected brands: %#v", brands.Data)
	}

	models, err := repository.ListModels(ctx, platforms.ModelQuery{IDBrand: 1})
	if err != nil {
		t.Fatalf("ListModels() returned an error: %v", err)
	}
	expectedModels := []platforms.ModelItem{{ID: 2, Model: "iPad", IDBrand: 1}, {ID: 1, Model: "iPhone", IDBrand: 1}}
	if !reflect.DeepEqual(models.Data, expectedModels) {
		t.Fatalf("unexpected models: %#v", models.Data)
	}

	operatingSystems, err := repository.ListOperatingSystems(ctx, platforms.OperatingSystemQuery{TypeID: 1})
	if err != nil {
		t.Fatalf("ListOperatingSystems() returned an error: %v", err)
	}
	expectedOperatingSystems := []platforms.OperatingSystemItem{{ID: 1, Name: "linux", Type: 1}, {ID: 2, Name: "windows", Type: 1}}
	if !reflect.DeepEqual(operatingSystems.Data, expectedOperatingSystems) {
		t.Fatalf("unexpected operating systems: %#v", operatingSystems.Data)
	}

	operatingSystemVersions, err := repository.ListOperatingSystemVersions(ctx, platforms.OperatingSystemVersionQuery{IDOs: 1})
	if err != nil {
		t.Fatalf("ListOperatingSystemVersions() returned an error: %v", err)
	}
	expectedOperatingSystemVersions := []platforms.OperatingSystemVersionItem{{ID: 1, Version: "14", IDOs: 1}, {ID: 2, Version: "15", IDOs: 1}}
	if !reflect.DeepEqual(operatingSystemVersions.Data, expectedOperatingSystemVersions) {
		t.Fatalf("unexpected operating-system versions: %#v", operatingSystemVersions.Data)
	}
}
