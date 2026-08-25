package mysql

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/cliapi"
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
		"DROP TABLE IF EXISTS browsers",
		"DROP TABLE IF EXISTS version_browsers",
		"CREATE TABLE types (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE statuses (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE locations (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE brand_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, brand VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE model_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, model VARCHAR(255) NOT NULL, idBrand INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE os (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, type INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE version_os (id BIGINT PRIMARY KEY AUTO_INCREMENT, version VARCHAR(255) NOT NULL, idOs INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE browsers (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, idOs INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE version_browsers (id BIGINT PRIMARY KEY AUTO_INCREMENT, version VARCHAR(255) NOT NULL, idBrowser INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"INSERT INTO types (id, name) VALUES (2, 'mobile'), (1, 'desktop')",
		"INSERT INTO statuses (id, name) VALUES (2, 'busy'), (1, 'free')",
		"INSERT INTO locations (id, name, created_at, updated_at) VALUES (2, 'us-east', NULL, NULL), (1, 'eu-west', NULL, NULL)",
		"INSERT INTO brand_devices (id, brand, created_at, updated_at) VALUES (2, 'Samsung', NULL, NULL), (1, 'Apple', NULL, NULL)",
		"INSERT INTO model_devices (id, model, idBrand, created_at, updated_at) VALUES (3, 'Galaxy', 2, NULL, NULL), (2, 'iPad', 1, NULL, NULL), (1, 'iPhone', 1, NULL, NULL)",
		"INSERT INTO os (id, name, type, created_at, updated_at) VALUES (3, 'android', 2, NULL, NULL), (2, 'windows', 1, NULL, NULL), (1, 'linux', 1, NULL, NULL)",
		"INSERT INTO version_os (id, version, idOs, created_at, updated_at) VALUES (3, '13', 2, NULL, NULL), (2, '15', 1, NULL, NULL), (1, '14', 1, NULL, NULL)",
		"INSERT INTO browsers (id, name, idOs, created_at, updated_at) VALUES (3, 'safari', 2, NULL, NULL), (2, 'firefox', 1, NULL, NULL), (1, 'chrome', 1, NULL, NULL)",
		"INSERT INTO version_browsers (id, version, idBrowser, created_at, updated_at) VALUES (3, '17', 2, NULL, NULL), (2, '125', 1, NULL, NULL), (1, '124', 1, NULL, NULL)",
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

	browsers, err := repository.ListBrowsers(ctx, platforms.BrowserQuery{IDOs: 1})
	if err != nil {
		t.Fatalf("ListBrowsers() returned an error: %v", err)
	}
	expectedBrowsers := []platforms.BrowserItem{{ID: 1, Name: "chrome", IDOs: 1}, {ID: 2, Name: "firefox", IDOs: 1}}
	if !reflect.DeepEqual(browsers.Data, expectedBrowsers) {
		t.Fatalf("unexpected browsers: %#v", browsers.Data)
	}

	browserVersions, err := repository.ListBrowserVersions(ctx, platforms.BrowserVersionQuery{IDBrowser: 1})
	if err != nil {
		t.Fatalf("ListBrowserVersions() returned an error: %v", err)
	}
	expectedBrowserVersions := []platforms.BrowserVersionItem{{ID: 1, Version: "124", IDBrowser: 1}, {ID: 2, Version: "125", IDBrowser: 1}}
	if !reflect.DeepEqual(browserVersions.Data, expectedBrowserVersions) {
		t.Fatalf("unexpected browser versions: %#v", browserVersions.Data)
	}
}

func TestLegacyKeyRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS costumers",
		`CREATE TABLE costumers (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			costumer VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			logo JSON NULL,
			licenseExpiration DATETIME NULL,
			apiKey VARCHAR(255) NOT NULL UNIQUE,
			apiKeyLastUsedAt TIMESTAMP NULL,
			apiKeyExpiresAt TIMESTAMP NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO costumers
			(id, costumer, description, logo, licenseExpiration, apiKey, apiKeyExpiresAt)
		 VALUES
			(1, 'First customer', 'First customer', JSON_OBJECT(), NULL, 'first-api-key', NULL),
			(2, 'Expired customer', 'Expired customer', JSON_OBJECT(), NULL, 'expired-api-key', '2026-08-25 09:00:00')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare legacy key fixture %q: %v", statement, err)
		}
	}

	repository := NewLegacyKeyRepository(database)
	usedAt := time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC)
	customer, err := repository.AuthenticateLegacyCustomerKey(ctx, "first-api-key", usedAt)
	if err != nil {
		t.Fatalf("AuthenticateLegacyCustomerKey() returned an error: %v", err)
	}
	if customer.ID != 1 || customer.Name != "First customer" {
		t.Fatalf("unexpected authenticated customer: %#v", customer)
	}

	var lastUsed sql.NullTime
	if err := database.QueryRowContext(ctx, "SELECT apiKeyLastUsedAt FROM costumers WHERE id = 1").Scan(&lastUsed); err != nil {
		t.Fatalf("read last-used timestamp: %v", err)
	}
	if !lastUsed.Valid || !lastUsed.Time.Equal(usedAt) {
		t.Fatalf("last-used timestamp was not recorded: %#v", lastUsed)
	}

	_, err = repository.AuthenticateLegacyCustomerKey(ctx, "expired-api-key", usedAt)
	if !errors.Is(err, auth.ErrInvalidLegacyKey) {
		t.Fatalf("expected expired key to be rejected safely, got %v", err)
	}

	_, err = repository.AuthenticateLegacyCustomerKey(ctx, "missing-api-key", usedAt)
	if !errors.Is(err, auth.ErrInvalidLegacyKey) {
		t.Fatalf("expected missing key to be rejected safely, got %v", err)
	}
}

func TestCLITestCycleRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS test_cycles",
		`CREATE TABLE test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		`INSERT INTO test_cycles
			(id, name, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(1, 'First cycle', 'Own cycle', JSON_ARRAY(), 10, NULL, NULL, 42),
			(2, 'Second cycle', 'Foreign cycle', JSON_ARRAY(), 20, NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI test-cycle fixture %q: %v", statement, err)
		}
	}

	repository := NewCLITestCycleRepository(database)
	cycle, err := repository.GetTestCycle(ctx, 42, 1)
	if err != nil {
		t.Fatalf("GetTestCycle() returned an error: %v", err)
	}
	if cycle.ID != 1 || cycle.IDCostumer != 42 || cycle.IDProject != 10 || cycle.Config != "[]" {
		t.Fatalf("unexpected test-cycle payload: %#v", cycle)
	}

	_, err = repository.GetTestCycle(ctx, 42, 2)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant test cycle to be hidden, got %v", err)
	}

	_, err = repository.GetTestCycle(ctx, 42, 999)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected missing test cycle to be hidden, got %v", err)
	}
}

func TestCLITestRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS tests",
		`CREATE TABLE tests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		`INSERT INTO tests
			(id, name, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(1, 'First test', 'Own test', JSON_ARRAY(), 10, NULL, NULL, 42),
			(2, 'Second test', 'Foreign test', JSON_ARRAY(), 20, NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI test fixture %q: %v", statement, err)
		}
	}

	repository := NewCLITestRepository(database)
	test, err := repository.GetTest(ctx, 42, 1)
	if err != nil {
		t.Fatalf("GetTest() returned an error: %v", err)
	}
	if test.ID != 1 || test.IDCostumer != 42 || test.IDProject != 10 || test.Config != "[]" {
		t.Fatalf("unexpected test payload: %#v", test)
	}

	_, err = repository.GetTest(ctx, 42, 2)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant test to be hidden, got %v", err)
	}

	_, err = repository.GetTest(ctx, 42, 999)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected missing test to be hidden, got %v", err)
	}
}

func TestCLIStepRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS steps",
		`CREATE TABLE steps (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject INT NOT NULL,
			` + "`order`" + ` INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		`INSERT INTO steps
			(id, name, description, config, idProject, ` + "`order`" + `, created_at, updated_at, idCostumer)
		 VALUES
			(1, 'First step', 'Own step', JSON_ARRAY(), 10, 2, NULL, NULL, 42),
			(2, 'Second step', 'Foreign step', JSON_ARRAY(), 20, 3, NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI step fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIStepRepository(database)
	step, err := repository.GetStep(ctx, 42, 1)
	if err != nil {
		t.Fatalf("GetStep() returned an error: %v", err)
	}
	if step.ID != 1 || step.IDCostumer != 42 || step.IDProject != 10 || step.Order != 2 || step.Config != "[]" {
		t.Fatalf("unexpected step payload: %#v", step)
	}

	_, err = repository.GetStep(ctx, 42, 2)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant step to be hidden, got %v", err)
	}

	_, err = repository.GetStep(ctx, 42, 999)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected missing step to be hidden, got %v", err)
	}
}

func TestCLIPluginRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS plugins",
		`CREATE TABLE plugins (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			code JSON NOT NULL,
			description VARCHAR(255) NOT NULL,
			idProject INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		`INSERT INTO plugins
			(id, name, code, description, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(1, 'First plugin', JSON_OBJECT(), 'Own plugin', 10, NULL, NULL, 42),
			(2, 'Second plugin', JSON_OBJECT(), 'Foreign plugin', 10, NULL, NULL, 99),
			(3, 'Third plugin', JSON_OBJECT(), 'Other project', 20, NULL, NULL, 42)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI plugin fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIPluginRepository(database)
	plugins, err := repository.ListPlugins(ctx, 42, 10)
	if err != nil {
		t.Fatalf("ListPlugins() returned an error: %v", err)
	}
	if len(plugins) != 1 || plugins[0].ID != 1 || plugins[0].IDCostumer != 42 || plugins[0].IDProject != 10 || plugins[0].Code != "{}" {
		t.Fatalf("unexpected plugin-list payload: %#v", plugins)
	}

	emptyPlugins, err := repository.ListPlugins(ctx, 42, 999)
	if err != nil {
		t.Fatalf("ListPlugins() for empty project returned an error: %v", err)
	}
	if len(emptyPlugins) != 0 {
		t.Fatalf("expected an empty plugin list, got %#v", emptyPlugins)
	}

	plugin, err := repository.GetPlugin(ctx, 42, 1)
	if err != nil {
		t.Fatalf("GetPlugin() returned an error: %v", err)
	}
	if plugin.ID != 1 || plugin.IDCostumer != 42 || plugin.IDProject != 10 || plugin.Code != "{}" {
		t.Fatalf("unexpected plugin payload: %#v", plugin)
	}

	_, err = repository.GetPlugin(ctx, 42, 2)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant plugin to be hidden, got %v", err)
	}

	_, err = repository.GetPlugin(ctx, 42, 999)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected missing plugin to be hidden, got %v", err)
	}
}

func TestCLIEnvironmentRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS environments",
		`CREATE TABLE environments (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			code VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		`INSERT INTO environments
			(id, code, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(1, 'demo', 'Own environment', JSON_OBJECT(), 10, NULL, NULL, 42),
			(2, 'foreign', 'Foreign environment', JSON_OBJECT(), 10, NULL, NULL, 99),
			(3, 'other', 'Other project', JSON_OBJECT(), 20, NULL, NULL, 42)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI environment fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIEnvironmentRepository(database)
	environments, err := repository.ListEnvironments(ctx, 42, 10)
	if err != nil {
		t.Fatalf("ListEnvironments() returned an error: %v", err)
	}
	if len(environments) != 1 || environments[0].ID != 1 || environments[0].IDCostumer != 42 || environments[0].IDProject != 10 || environments[0].Config != "{}" {
		t.Fatalf("unexpected environment-list payload: %#v", environments)
	}

	emptyEnvironments, err := repository.ListEnvironments(ctx, 42, 999)
	if err != nil {
		t.Fatalf("ListEnvironments() for empty project returned an error: %v", err)
	}
	if len(emptyEnvironments) != 0 {
		t.Fatalf("expected an empty environment list, got %#v", emptyEnvironments)
	}

	environment, err := repository.GetEnvironment(ctx, 42, 1)
	if err != nil {
		t.Fatalf("GetEnvironment() returned an error: %v", err)
	}
	if environment.ID != 1 || environment.IDCostumer != 42 || environment.IDProject != 10 || environment.Config != "{}" {
		t.Fatalf("unexpected environment payload: %#v", environment)
	}

	_, err = repository.GetEnvironment(ctx, 42, 2)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant environment to be hidden, got %v", err)
	}

	_, err = repository.GetEnvironment(ctx, 42, 999)
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected missing environment to be hidden, got %v", err)
	}
}
