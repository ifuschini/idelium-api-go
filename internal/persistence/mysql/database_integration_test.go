package mysql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/cliapi"
	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/health"
	"github.com/idelium/idelium-api-go/internal/integrations"
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

type integrationDoerFunc func(*http.Request) (*http.Response, error)

func (function integrationDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
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

func TestBrowserAuthRepositorySessionLookupIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	for _, statement := range []string{
		"DROP TABLE IF EXISTS go_browser_sessions",
		"DROP TABLE IF EXISTS users",
		`CREATE TABLE users (
			id BIGINT PRIMARY KEY,
			idCostumer BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			role BIGINT NOT NULL,
			password VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL
		)`,
		`CREATE TABLE go_browser_sessions (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idHash CHAR(64) NOT NULL UNIQUE,
			userId BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			activeTenantId BIGINT NULL,
			csrfTokenHash CHAR(64) NOT NULL,
			impersonationReason VARCHAR(255) NULL,
			impersonationExpiresAt TIMESTAMP NULL,
			expiresAt TIMESTAMP NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO users (id, idCostumer, name, email, role, password, status) VALUES
			(7, 42, 'Browser User', 'browser@example.test', 3, '$2y$10$legacyhash', 'active'),
			(8, 99, 'Other Tenant', 'other@example.test', 3, '$2y$10$legacyhash', 'active'),
			(9, 42, 'Disabled User', 'disabled@example.test', 3, '$2y$10$legacyhash', 'disabled')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare browser auth fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	if err := repository.Create(ctx, browserauth.Session{ID: "active-session", UserID: 7, TenantID: 42, CSRFToken: "active-csrf", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Create(active) returned an error: %v", err)
	}
	if err := repository.Create(ctx, browserauth.Session{ID: "expired-session", UserID: 7, TenantID: 42, CSRFToken: "expired-csrf", ExpiresAt: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("Create(expired) returned an error: %v", err)
	}
	if err := repository.Create(ctx, browserauth.Session{ID: "cross-tenant-session", UserID: 8, TenantID: 42, CSRFToken: "cross-csrf", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Create(cross tenant) returned an error: %v", err)
	}
	if err := repository.Create(ctx, browserauth.Session{ID: "disabled-session", UserID: 9, TenantID: 42, CSRFToken: "disabled-csrf", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Create(disabled) returned an error: %v", err)
	}

	user, err := repository.Get(ctx, "active-session", now)
	if err != nil {
		t.Fatalf("Get(active) returned an error: %v", err)
	}
	if user.ID != 7 || user.TenantID != 42 || user.Email != "browser@example.test" || user.Status != "active" {
		t.Fatalf("unexpected session user: %#v", user)
	}

	for _, sessionID := range []string{"expired-session", "cross-tenant-session", "disabled-session", "missing-session"} {
		_, err := repository.Get(ctx, sessionID, now)
		if !errors.Is(err, browserauth.ErrNotFound) {
			t.Fatalf("expected %s to be hidden, got %v", sessionID, err)
		}
	}
}

func TestBrowserAuthRepositoryTenantSwitchIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	for _, statement := range []string{
		"DROP TABLE IF EXISTS audit_events",
		"DROP TABLE IF EXISTS go_browser_sessions",
		"DROP TABLE IF EXISTS projects",
		"DROP TABLE IF EXISTS users",
		"DROP TABLE IF EXISTS costumers",
		`CREATE TABLE costumers (
			id BIGINT PRIMARY KEY,
			costumer VARCHAR(255) NOT NULL,
			description VARCHAR(255) NULL,
			licenseExpiration TIMESTAMP NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE users (
			id BIGINT PRIMARY KEY,
			idCostumer BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			role BIGINT NOT NULL,
			password VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL
		)`,
		`CREATE TABLE projects (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE go_browser_sessions (
			idHash CHAR(64) NOT NULL UNIQUE,
			userId BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			activeTenantId BIGINT NULL,
			csrfTokenHash CHAR(64) NOT NULL,
			impersonationReason VARCHAR(255) NULL,
			impersonationExpiresAt TIMESTAMP NULL,
			expiresAt TIMESTAMP NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE audit_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			actorUserId BIGINT NULL,
			actorTenantId BIGINT NULL,
			activeTenantId BIGINT NOT NULL,
			action VARCHAR(128) NOT NULL,
			targetType VARCHAR(128) NOT NULL,
			targetId VARCHAR(128) NULL,
			beforeValues JSON NULL,
			afterValues JSON NULL,
			result VARCHAR(32) NOT NULL,
			sourceIp VARCHAR(64) NULL,
			correlationId CHAR(36) NOT NULL,
			metadata JSON NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO costumers (id, costumer, description, licenseExpiration, created_at, updated_at) VALUES
			(11, 'ACTOR', 'Actor tenant', NULL, '2026-08-27 10:00:00', NULL),
			(42, 'TARGET', 'Target tenant', NULL, '2026-08-27 11:00:00', NULL)`,
		`INSERT INTO users (id, idCostumer, name, email, role, password, status) VALUES
			(7, 11, 'Super Admin', 'admin@example.test', 1, '$2y$10$legacyhash', 'active')`,
		`INSERT INTO projects (id, name, description, idCostumer, created_at, updated_at) VALUES
			(3, 'Target project', 'tenant scoped', 42, '2026-08-27 11:00:00', NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare tenant switch fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	if err := repository.Create(ctx, browserauth.Session{ID: "switch-session", UserID: 7, TenantID: 11, CSRFToken: "csrf", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}
	exists, err := repository.CustomerExists(ctx, 42)
	if err != nil || !exists {
		t.Fatalf("CustomerExists() = %v, %v", exists, err)
	}
	if err := repository.SwitchTenant(ctx, browserauth.TenantSwitch{SessionID: "switch-session", UserID: 7, ActorTenant: 11, ActiveTenant: 42, Reason: "support", ExpiresAt: now.Add(time.Hour), Now: now}); err != nil {
		t.Fatalf("SwitchTenant() returned an error: %v", err)
	}
	user, err := repository.Get(ctx, "switch-session", now)
	if err != nil {
		t.Fatalf("Get(switched) returned an error: %v", err)
	}
	if user.ActiveTenantID != 42 || user.ImpersonationReason == nil || *user.ImpersonationReason != "support" {
		t.Fatalf("unexpected switched session user: %#v", user)
	}
	projects, err := repository.ListProjects(ctx, 42)
	if err != nil || len(projects) != 1 || projects[0].IDCostumer != 42 {
		t.Fatalf("unexpected switched tenant projects: %#v, %v", projects, err)
	}
	if err := repository.RecordTenantSwitch(ctx, browserauth.AuditEvent{
		ActorUserID:    7,
		ActorTenantID:  11,
		ActiveTenantID: 42,
		TargetID:       42,
		SourceIP:       "203.0.113.10",
		CorrelationID:  "11111111-1111-4111-8111-111111111111",
		BeforeValues:   map[string]any{"activeTenantId": int64(11)},
		AfterValues:    map[string]any{"activeTenantId": int64(42), "sessionToken": "[REDACTED]"},
	}); err != nil {
		t.Fatalf("RecordTenantSwitch() returned an error: %v", err)
	}
	var afterValues string
	if err := database.QueryRowContext(ctx, "SELECT afterValues FROM audit_events WHERE action = 'tenant.switch'").Scan(&afterValues); err != nil {
		t.Fatalf("read audit event: %v", err)
	}
	if !strings.Contains(afterValues, "[REDACTED]") || strings.Contains(afterValues, "switch-session") {
		t.Fatalf("audit event exposed session data: %s", afterValues)
	}
	if err := repository.SwitchTenant(ctx, browserauth.TenantSwitch{SessionID: "switch-session", UserID: 999, ActorTenant: 11, ActiveTenant: 42, Reason: "support", ExpiresAt: now.Add(time.Hour), Now: now}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-user switch to be hidden, got %v", err)
	}
}

func TestBrowserAuthRepositoryAccountsIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS users",
		"DROP TABLE IF EXISTS roles",
		"DROP TABLE IF EXISTS costumers",
		`CREATE TABLE roles (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE costumers (
			id BIGINT PRIMARY KEY,
			costumer VARCHAR(255) NOT NULL,
			description VARCHAR(255) NULL,
			licenseExpiration TIMESTAMP NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			role BIGINT NOT NULL,
			password VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO roles (id, name, created_at, updated_at) VALUES
			(1, 'superadmin', NULL, NULL),
			(2, 'admin', NULL, NULL),
			(3, 'user', NULL, NULL)`,
		`INSERT INTO costumers (id, costumer, description, licenseExpiration, created_at, updated_at) VALUES
			(11, 'ACME', 'Own tenant', NULL, NULL, NULL),
			(42, 'OTHER', 'Foreign tenant', NULL, NULL, NULL)`,
		`INSERT INTO users (id, idCostumer, name, email, role, password, status, created_at, updated_at) VALUES
			(7, 11, 'Tenant Admin', 'admin@example.test', 2, '$2y$10$legacyhash', 'active', NULL, NULL),
			(8, 11, 'Tenant User', 'user@example.test', 3, '$2y$10$legacyhash', 'active', NULL, NULL),
			(9, 42, 'Foreign User', 'foreign@example.test', 3, '$2y$10$legacyhash', 'active', NULL, NULL),
			(10, 11, 'Super Admin', 'super@example.test', 1, '$2y$10$legacyhash', 'active', NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare account fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	request := httptest.NewRequest(http.MethodGet, "/admin/accounts?page=1&pageSize=25", nil)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}
	page, err := repository.ListAccounts(request, actor, browserauth.AccountQuery{ActorTenantID: 11, ActorRole: 2, Page: 1, PageSize: 25, Paged: true, Sort: "email", Direction: "asc"})
	if err != nil {
		t.Fatalf("ListAccounts() returned an error: %v", err)
	}
	if page.Meta.Total != 2 {
		t.Fatalf("expected role 2 to see only tenant non-superadmin accounts, got %#v", page)
	}
	for _, account := range page.Data {
		if account.IDCostumer != 11 || account.Role == 1 {
			t.Fatalf("account scope leaked: %#v", page.Data)
		}
	}

	if err := repository.CreateAccount(request, actor, browserauth.AccountCreate{Name: "Created User", Email: "created@example.test", Password: "StrongPassword123!", Role: 3, IDCostumer: 11}); err != nil {
		t.Fatalf("CreateAccount() returned an error: %v", err)
	}
	if err := repository.CreateAccount(request, actor, browserauth.AccountCreate{Name: "Foreign User", Email: "created-foreign@example.test", Password: "StrongPassword123!", Role: 3, IDCostumer: 42}); !errors.Is(err, browserauth.ErrForbidden) {
		t.Fatalf("expected cross-tenant create to be forbidden, got %v", err)
	}
	var createdID int64
	var passwordHash string
	if err := database.QueryRowContext(ctx, "SELECT id, password FROM users WHERE email = 'created@example.test'").Scan(&createdID, &passwordHash); err != nil {
		t.Fatalf("read created account: %v", err)
	}
	if passwordHash == "StrongPassword123!" || !strings.HasPrefix(passwordHash, "$2") {
		t.Fatalf("password was not bcrypt hashed: %s", passwordHash)
	}
	if err := repository.UpdateAccount(request, actor, browserauth.AccountUpdate{ID: createdID, Name: "Updated User", Password: "AnotherStrong123!"}); err != nil {
		t.Fatalf("UpdateAccount() returned an error: %v", err)
	}
	if err := repository.DeleteAccount(request, actor, 9); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-tenant delete to be hidden, got %v", err)
	}
	if err := repository.DeleteAccount(request, actor, createdID); err != nil {
		t.Fatalf("DeleteAccount() returned an error: %v", err)
	}
}

func TestBrowserAuthRepositoryCustomersIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS costumers",
		`CREATE TABLE costumers (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			costumer VARCHAR(255) NOT NULL,
			description VARCHAR(255) NULL,
			licenseExpiration TIMESTAMP NULL,
			apiKey VARCHAR(255) NULL,
			apiKeyCreatedAt TIMESTAMP NULL,
			logo TEXT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO costumers (id, costumer, description, licenseExpiration, apiKey, apiKeyCreatedAt, logo, created_at, updated_at) VALUES
			(11, 'ACME', 'Own tenant', NULL, 'secret-key', NULL, '[]', '2026-08-27 10:00:00', NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare customer fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	request := httptest.NewRequest(http.MethodGet, "/admin/costumers?page=1&pageSize=25", nil)
	page, err := repository.ListAdminCustomers(request, browserauth.CustomerQuery{Page: 1, PageSize: 25, Paged: true, Sort: "created_at", Direction: "asc"})
	if err != nil {
		t.Fatalf("ListAdminCustomers() returned an error: %v", err)
	}
	if page.Meta.Total != 1 || len(page.Data) != 1 || page.Data[0].Costumer != "ACME" {
		t.Fatalf("unexpected customer page: %#v", page)
	}

	if err := repository.CreateCustomer(request, browserauth.CustomerCreate{Costumer: "new co", Description: "fresh", Now: time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("CreateCustomer() returned an error: %v", err)
	}
	var createdID int64
	var createdName, apiKey string
	if err := database.QueryRowContext(ctx, "SELECT id, costumer, apiKey FROM costumers WHERE costumer = 'NEW CO'").Scan(&createdID, &createdName, &apiKey); err != nil {
		t.Fatalf("read created customer: %v", err)
	}
	if createdName != "NEW CO" || len(apiKey) < 32 {
		t.Fatalf("unexpected created customer: %d %s %s", createdID, createdName, apiKey)
	}
	if err := repository.UpdateCustomer(request, browserauth.CustomerUpdate{ID: createdID, Costumer: "updated co", Description: "changed"}); err != nil {
		t.Fatalf("UpdateCustomer() returned an error: %v", err)
	}
	if err := repository.DeleteCustomer(request, createdID); err != nil {
		t.Fatalf("DeleteCustomer() returned an error: %v", err)
	}
	if err := repository.DeleteCustomer(request, createdID); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected missing customer delete to be hidden, got %v", err)
	}
}

func TestBrowserAuthRepositoryTestCyclesAndStepOrderingIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS asset_versions",
		"DROP TABLE IF EXISTS steps",
		"DROP TABLE IF EXISTS tests",
		"DROP TABLE IF EXISTS test_cycles",
		"DROP TABLE IF EXISTS projects",
		`CREATE TABLE projects (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config TEXT NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE tests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE steps (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config TEXT NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			` + "`order`" + ` BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE asset_versions (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer BIGINT NOT NULL,
			idProject BIGINT NOT NULL,
			assetType VARCHAR(64) NOT NULL,
			assetId BIGINT NOT NULL,
			version INT NOT NULL,
			actorUserId BIGINT NULL,
			reason VARCHAR(255) NOT NULL,
			snapshot JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY asset_versions_unique_version (idCostumer, assetType, assetId, version)
		)`,
		`INSERT INTO projects (id, name, description, idCostumer, created_at, updated_at) VALUES
			(3, 'Own project', 'Own', 11, NULL, NULL),
			(4, 'Foreign project', 'Foreign', 42, NULL, NULL)`,
		`INSERT INTO test_cycles (id, name, description, config, idProject, idCostumer, created_at, updated_at) VALUES
			(5, 'Nightly', 'Browser', '{}', 3, 11, NULL, NULL),
			(6, 'Foreign', 'Hidden', '{}', 4, 42, NULL, NULL)`,
		`INSERT INTO tests (id, name, description, config, idProject, idCostumer, created_at, updated_at) VALUES
			(7, 'Checkout', 'Happy path', '{"steps":[9,10]}', 3, 11, NULL, NULL),
			(8, 'Foreign test', 'Hidden', '{"steps":[11]}', 4, 42, NULL, NULL)`,
		`INSERT INTO steps (id, name, description, config, idProject, idCostumer, ` + "`order`" + `, created_at, updated_at) VALUES
			(9, 'First', 'First step', '{}', 3, 11, 10, NULL, NULL),
			(10, 'Second', 'Second step', '{}', 3, 11, 20, NULL, NULL),
			(11, 'Foreign', 'Hidden step', '{}', 4, 42, 30, NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare authoring fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}
	request := httptest.NewRequest(http.MethodGet, "/admin/testcycles/3?page=1&pageSize=25", nil)
	page, err := repository.ListTestCycles(request, actor, browserauth.ResourceQuery{ProjectID: 3, Page: 1, PageSize: 25, Paged: true, Sort: "id", Direction: "asc"})
	if err != nil {
		t.Fatalf("ListTestCycles() returned an error: %v", err)
	}
	if page.Meta.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != 5 {
		t.Fatalf("unexpected test-cycle page: %#v", page)
	}
	if _, err := repository.ListTestCycles(request, actor, browserauth.ResourceQuery{ProjectID: 4, Page: 1, PageSize: 25, Paged: true, Sort: "id", Direction: "asc"}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign project to be hidden, got %v", err)
	}
	if err := repository.CreateTestCycle(request, actor, browserauth.TestCycleCreate{Name: "Smoke", Description: "Fast", Config: "{}", IDProject: 3}); err != nil {
		t.Fatalf("CreateTestCycle() returned an error: %v", err)
	}
	var versions int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_versions WHERE assetType = 'test_cycle' AND reason = 'asset.created'").Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("expected created asset version, count=%d err=%v", versions, err)
	}
	detail, err := repository.GetTestCycle(request, actor, 3, 5)
	if err != nil || detail.ID != 5 || detail.IDProject != 3 {
		t.Fatalf("unexpected test-cycle detail: %#v err=%v", detail, err)
	}
	if _, err := repository.GetTestCycle(request, actor, 3, 6); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-tenant cycle to be hidden, got %v", err)
	}
	if err := repository.UpdateTestCycle(request, actor, browserauth.TestCycleUpdate{ID: 5, IDProject: 3, Description: "Updated", Config: "{\"tests\":[]}"}); err != nil {
		t.Fatalf("UpdateTestCycle() returned an error: %v", err)
	}
	if err := repository.ReorderSteps(request, actor, browserauth.StepReorder{IDProject: 3, Offset: 25, Order: []browserauth.StepOrder{{ID: 10}, {ID: 999999}}}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected failed reorder to return not found, got %v", err)
	}
	var firstOrder int64
	if err := database.QueryRowContext(ctx, "SELECT `order` FROM steps WHERE id = 10").Scan(&firstOrder); err != nil || firstOrder != 20 {
		t.Fatalf("failed reorder was not rolled back, order=%d err=%v", firstOrder, err)
	}
	if err := repository.ReorderSteps(request, actor, browserauth.StepReorder{IDProject: 3, Offset: 25, Order: []browserauth.StepOrder{{ID: 10}, {ID: 9}}}); err != nil {
		t.Fatalf("ReorderSteps() returned an error: %v", err)
	}
	steps, err := repository.ListStepsForReorder(request, actor, browserauth.ResourceQuery{ProjectID: 3, Page: 1, PageSize: 25, Paged: true, Sort: "order", Direction: "asc"})
	if err != nil || len(steps.Data) != 2 || steps.Data[0].ID != 10 || steps.Data[0].Order != 25 {
		t.Fatalf("unexpected reordered steps: %#v err=%v", steps, err)
	}

	testsPage, err := repository.ListTests(request, actor, browserauth.ResourceQuery{ProjectID: 3, Page: 1, PageSize: 25, Paged: true, Sort: "id", Direction: "asc"})
	if err != nil {
		t.Fatalf("ListTests() returned an error: %v", err)
	}
	if testsPage.Meta.Total != 1 || len(testsPage.Data) != 1 || testsPage.Data[0].ID != 7 {
		t.Fatalf("unexpected test page: %#v", testsPage)
	}
	if _, err := repository.ListTests(request, actor, browserauth.ResourceQuery{ProjectID: 4, Page: 1, PageSize: 25, Paged: true, Sort: "id", Direction: "asc"}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign test project to be hidden, got %v", err)
	}
	if err := repository.CreateTest(request, actor, browserauth.TestCreate{Name: "Smoke", Description: "Fast", Config: "{\"steps\":[9]}", IDProject: 3}); err != nil {
		t.Fatalf("CreateTest() returned an error: %v", err)
	}
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_versions WHERE assetType = 'test' AND reason = 'asset.created'").Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("expected created test asset version, count=%d err=%v", versions, err)
	}
	testDetail, err := repository.GetTest(request, actor, 3, 7)
	if err != nil || testDetail.ID != 7 || testDetail.IDProject != 3 || testDetail.Config != "{\"steps\":[9,10]}" {
		t.Fatalf("unexpected test detail: %#v err=%v", testDetail, err)
	}
	if _, err := repository.GetTest(request, actor, 3, 8); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-tenant test to be hidden, got %v", err)
	}
	if err := repository.UpdateTest(request, actor, browserauth.TestUpdate{ID: 7, IDProject: 3, Config: "{\"steps\":[10,9]}"}); err != nil {
		t.Fatalf("UpdateTest() returned an error: %v", err)
	}
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_versions WHERE assetType = 'test' AND reason = 'asset.updated'").Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("expected updated test asset version, count=%d err=%v", versions, err)
	}

	importPayload := `[
		{"name":"Open Home","steps":[{"stepType":"browser_click"}]},
		{"name":"Run Postman","steps":[{"stepType":"postman_collection","collection":{"info":{"name":"Demo"},"item":[]}}]}
	]`
	if err := repository.ImportTest(request, actor, browserauth.TestImport{Name: "Imported", Description: "Imported flow", Import: importPayload, IDProject: 3}); err != nil {
		t.Fatalf("ImportTest() returned an error: %v", err)
	}
	var importedSteps int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM steps WHERE idProject = 3 AND idCostumer = 11 AND `order` = 9999999").Scan(&importedSteps); err != nil || importedSteps != 2 {
		t.Fatalf("expected imported steps, count=%d err=%v", importedSteps, err)
	}
	var importedConfig string
	if err := database.QueryRowContext(ctx, "SELECT config FROM tests WHERE name = 'Imported' AND idProject = 3 AND idCostumer = 11").Scan(&importedConfig); err != nil || !strings.Contains(importedConfig, "Open_Home") || !strings.Contains(importedConfig, "Run_Postman") {
		t.Fatalf("unexpected imported test config: %s err=%v", importedConfig, err)
	}
	if err := repository.ImportTest(request, actor, browserauth.TestImport{Name: "Foreign", Description: "Blocked", Import: importPayload, IDProject: 4}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign import project to be hidden, got %v", err)
	}
}

func TestBrowserAuthoringWorkflowEndToEndIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS asset_versions",
		"DROP TABLE IF EXISTS go_browser_sessions",
		"DROP TABLE IF EXISTS steps",
		"DROP TABLE IF EXISTS tests",
		"DROP TABLE IF EXISTS test_cycles",
		"DROP TABLE IF EXISTS projects",
		"DROP TABLE IF EXISTS users",
		`CREATE TABLE users (
			id BIGINT PRIMARY KEY,
			idCostumer BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			role BIGINT NOT NULL,
			password VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL
		)`,
		`CREATE TABLE go_browser_sessions (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idHash CHAR(64) NOT NULL UNIQUE,
			userId BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			activeTenantId BIGINT NULL,
			csrfTokenHash CHAR(64) NOT NULL,
			impersonationReason VARCHAR(255) NULL,
			impersonationExpiresAt TIMESTAMP NULL,
			expiresAt TIMESTAMP NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE projects (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE tests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config TEXT NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE steps (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			config JSON NOT NULL,
			idProject BIGINT NOT NULL,
			idCostumer BIGINT NOT NULL,
			` + "`order`" + ` BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE asset_versions (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer BIGINT NOT NULL,
			idProject BIGINT NOT NULL,
			assetType VARCHAR(64) NOT NULL,
			assetId BIGINT NOT NULL,
			version INT NOT NULL,
			actorUserId BIGINT NULL,
			reason VARCHAR(255) NOT NULL,
			snapshot JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY asset_versions_unique_version (idCostumer, assetType, assetId, version)
		)`,
		`INSERT INTO users (id, idCostumer, name, email, role, password, status) VALUES
			(7, 11, 'Author', 'author@example.test', 3, '$2y$10$legacyhash', 'active')`,
		`INSERT INTO projects (id, name, description, idCostumer, created_at, updated_at) VALUES
			(3, 'Own project', 'Own', 11, NULL, NULL),
			(4, 'Foreign project', 'Foreign', 42, NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare browser authoring workflow fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	if err := repository.Create(ctx, browserauth.Session{ID: "authoring-session", UserID: 7, TenantID: 11, CSRFToken: "authoring-csrf", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Create(authoring session) returned an error: %v", err)
	}
	handler := browserauth.NewHandler(repository, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	withSession := func(request *http.Request) *http.Request {
		request.AddCookie(&http.Cookie{Name: "idelium_session", Value: "authoring-session"})
		return request
	}

	importBody := `{"name":"Imported","description":"Imported flow","idProject":3,"import":"[{\"name\":\"Open Home\",\"steps\":[{\"stepType\":\"browser_click\"}]},{\"name\":\"Run Postman\",\"editorType\":\"postman\",\"steps\":[{\"stepType\":\"postman_collection\",\"collection\":{\"info\":{\"name\":\"Demo\"},\"item\":[]}}]}]"}`
	importResponse := httptest.NewRecorder()
	handler.ImportTest(importResponse, withSession(httptest.NewRequest(http.MethodPost, "/admin/importtest", strings.NewReader(importBody))))
	if importResponse.Code != http.StatusOK || strings.TrimSpace(importResponse.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("unexpected import response: %d %s", importResponse.Code, importResponse.Body.String())
	}
	var importedTestID int64
	if err := database.QueryRowContext(ctx, "SELECT id FROM tests WHERE name = 'Imported' AND idCostumer = 11").Scan(&importedTestID); err != nil {
		t.Fatalf("imported test was not stored: %v", err)
	}

	listTests := httptest.NewRequest(http.MethodGet, "/admin/tests/3?page=1&pageSize=25", nil)
	listTests.SetPathValue("idProject", "3")
	listTestsResponse := httptest.NewRecorder()
	handler.Tests(listTestsResponse, withSession(listTests))
	if listTestsResponse.Code != http.StatusOK || !strings.Contains(listTestsResponse.Body.String(), `"Imported"`) || strings.Contains(listTestsResponse.Body.String(), "idCostumer") {
		t.Fatalf("unexpected test list response: %d %s", listTestsResponse.Code, listTestsResponse.Body.String())
	}

	updateTest := httptest.NewRequest(http.MethodPut, "/admin/tests/3/1", strings.NewReader(`{"config":"[{\"id\":1,\"name\":\"Run_Postman\"}]"}`))
	updateTest.SetPathValue("idProject", "3")
	updateTest.SetPathValue("test", strconv.FormatInt(importedTestID, 10))
	updateTestResponse := httptest.NewRecorder()
	handler.UpdateTest(updateTestResponse, withSession(updateTest))
	if updateTestResponse.Code != http.StatusOK {
		t.Fatalf("unexpected test update response: %d %s", updateTestResponse.Code, updateTestResponse.Body.String())
	}

	createCycle := httptest.NewRequest(http.MethodPost, "/admin/testcycles", strings.NewReader(`{"name":"Nightly","description":"Browser","config":"{\"tests\":[1]}","idProject":3}`))
	createCycleResponse := httptest.NewRecorder()
	handler.CreateTestCycle(createCycleResponse, withSession(createCycle))
	if createCycleResponse.Code != http.StatusOK || !strings.Contains(createCycleResponse.Body.String(), `"Nightly"`) {
		t.Fatalf("unexpected cycle create response: %d %s", createCycleResponse.Code, createCycleResponse.Body.String())
	}

	var importedStepIDs []int64
	rows, err := database.QueryContext(ctx, "SELECT id FROM steps WHERE idProject = 3 AND idCostumer = 11 ORDER BY id ASC")
	if err != nil {
		t.Fatalf("list imported step ids: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan imported step id: %v", err)
		}
		importedStepIDs = append(importedStepIDs, id)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close imported step rows: %v", err)
	}
	if len(importedStepIDs) != 2 {
		t.Fatalf("expected two imported steps, got %#v", importedStepIDs)
	}
	reorder := httptest.NewRequest(http.MethodPost, "/admin/steps/3/updateorder", strings.NewReader(fmt.Sprintf(`{"offset":10,"order":[{"id":%d},{"id":%d}]}`, importedStepIDs[1], importedStepIDs[0])))
	reorder.SetPathValue("idProject", "3")
	reorderResponse := httptest.NewRecorder()
	handler.ReorderSteps(reorderResponse, withSession(reorder))
	if reorderResponse.Code != http.StatusOK || !strings.Contains(reorderResponse.Body.String(), `"order":10`) {
		t.Fatalf("unexpected step reorder response: %d %s", reorderResponse.Code, reorderResponse.Body.String())
	}

	foreignList := httptest.NewRequest(http.MethodGet, "/admin/tests/4", nil)
	foreignList.SetPathValue("idProject", "4")
	foreignResponse := httptest.NewRecorder()
	handler.Tests(foreignResponse, withSession(foreignList))
	if foreignResponse.Code != http.StatusNotFound {
		t.Fatalf("expected foreign project to be hidden, got %d %s", foreignResponse.Code, foreignResponse.Body.String())
	}
}

func TestBrowserPerformedResultExplorationIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS performed_steps",
		"DROP TABLE IF EXISTS performed_tests",
		"DROP TABLE IF EXISTS performed_test_cycles",
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY,
			testCycleId BIGINT NOT NULL,
			date DATETIME NULL,
			status INT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE performed_tests (
			id BIGINT PRIMARY KEY,
			testCycleDoneId BIGINT NOT NULL,
			testId BIGINT NOT NULL,
			status INT NOT NULL,
			postmanData LONGTEXT NULL,
			name VARCHAR(255) NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE performed_steps (
			id BIGINT PRIMARY KEY,
			testCycleDoneId BIGINT NOT NULL,
			testDoneId BIGINT NOT NULL,
			stepId BIGINT NOT NULL,
			status INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			screenshots JSON NOT NULL,
			type VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO performed_test_cycles (id, testCycleId, date, status, idCostumer, created_at, updated_at) VALUES
			(44, 5, '2026-08-31 09:00:00', 1, 11, NULL, NULL),
			(45, 5, '2026-08-31 10:00:00', 2, 11, NULL, NULL),
			(46, 5, '2026-08-31 11:00:00', 1, 42, NULL, NULL)`,
		`INSERT INTO performed_tests (id, testCycleDoneId, testId, status, postmanData, name, idCostumer, created_at, updated_at) VALUES
			(55, 44, 7, 1, '[{"request":{"headers":{"Authorization":"Bearer unsafe-token"},"url":"https://api.example.test"},"response":{"token":"unsafe-response-token"}}]', 'Checkout', 11, NULL, NULL),
			(56, 46, 7, 1, '[{"request":{"headers":{"Authorization":"Bearer foreign-token"}}}]', 'Foreign', 42, NULL, NULL)`,
		`INSERT INTO performed_steps (id, testCycleDoneId, testDoneId, stepId, status, name, screenshots, type, data, idCostumer, created_at, updated_at) VALUES
			(77, 44, 55, 9, 1, 'Open', '["screen.png"]', 'selenium', '{"headers":{"Authorization":"Bearer unsafe-token"},"body":"ok"}', 11, NULL, NULL),
			(78, 46, 56, 9, 1, 'Foreign', '["foreign.png"]', 'selenium', '{"token":"foreign-token"}', 42, NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare performed result fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}
	request := httptest.NewRequest(http.MethodGet, "/admin/testcyclesperfomed/5?page=1&perPage=1&status=1", nil)
	status := 1
	cycles, err := repository.ListPerformedCycles(request, actor, browserauth.ResultQuery{ParentID: 5, Page: 1, PerPage: 1, Paged: true, Status: &status, Sort: "date", Direction: "desc"})
	if err != nil {
		t.Fatalf("ListPerformedCycles() returned an error: %v", err)
	}
	if cycles.Meta.Pagination.Total != 1 || len(cycles.Data) != 1 || cycles.Data[0].ID != 44 || cycles.Data[0].TestCycleID != 5 {
		t.Fatalf("unexpected performed cycles page: %#v", cycles)
	}

	tests, err := repository.ListPerformedTests(request, actor, browserauth.ResultQuery{ParentID: 44, Page: 1, PerPage: 25, Paged: true, Sort: "id", Direction: "asc"})
	if err != nil {
		t.Fatalf("ListPerformedTests() returned an error: %v", err)
	}
	if tests.Meta.Pagination.Total != 1 || len(tests.Data) != 1 || tests.Data[0].ID != 55 || tests.Data[0].PostmanData == nil {
		t.Fatalf("unexpected performed tests page: %#v", tests)
	}
	if strings.Contains(*tests.Data[0].PostmanData, "unsafe-token") || !strings.Contains(*tests.Data[0].PostmanData, "[REDACTED]") {
		t.Fatalf("performed test leaked sensitive postman data: %s", *tests.Data[0].PostmanData)
	}

	steps, err := repository.ListPerformedSteps(request, actor, 55)
	if err != nil {
		t.Fatalf("ListPerformedSteps() returned an error: %v", err)
	}
	if len(steps) != 1 || steps[0].ID != 77 || steps[0].Type != "selenium" {
		t.Fatalf("unexpected performed steps: %#v", steps)
	}
	if strings.Contains(steps[0].Data, "unsafe-token") || !strings.Contains(steps[0].Data, "[REDACTED]") {
		t.Fatalf("performed step leaked sensitive data: %s", steps[0].Data)
	}
	if foreignSteps, err := repository.ListPerformedSteps(request, actor, 56); err != nil || len(foreignSteps) != 0 {
		t.Fatalf("expected foreign performed steps to be hidden, got %#v err=%v", foreignSteps, err)
	}
}

func TestBrowserResultExportsIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS result_exports",
		"DROP TABLE IF EXISTS performed_steps",
		"DROP TABLE IF EXISTS performed_tests",
		"DROP TABLE IF EXISTS performed_test_cycles",
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY,
			testCycleId BIGINT NOT NULL,
			date DATETIME NULL,
			status INT NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE performed_tests (
			id BIGINT PRIMARY KEY,
			testCycleDoneId BIGINT NOT NULL,
			testId BIGINT NOT NULL,
			status INT NOT NULL,
			postmanData LONGTEXT NULL,
			name VARCHAR(255) NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE performed_steps (
			id BIGINT PRIMARY KEY,
			testCycleDoneId BIGINT NOT NULL,
			testDoneId BIGINT NOT NULL,
			stepId BIGINT NOT NULL,
			status INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			screenshots JSON NOT NULL,
			type VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			idCostumer BIGINT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE result_exports (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer BIGINT NOT NULL,
			performedTestCycleId BIGINT NOT NULL,
			format VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			filename VARCHAR(255) NOT NULL,
			contentType VARCHAR(128) NOT NULL,
			payload LONGTEXT NULL,
			errorMessage TEXT NULL,
			expiresAt TIMESTAMP NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`INSERT INTO performed_test_cycles (id, testCycleId, date, status, idCostumer, created_at, updated_at) VALUES
			(44, 5, '2026-08-31 09:00:00', 1, 11, NULL, NULL),
			(45, 6, '2026-08-31 09:00:00', 1, 42, NULL, NULL)`,
		`INSERT INTO performed_tests (id, testCycleDoneId, testId, status, postmanData, name, idCostumer, created_at, updated_at) VALUES
			(55, 44, 7, 1, NULL, 'Checkout', 11, NULL, NULL)`,
		`INSERT INTO performed_steps (id, testCycleDoneId, testDoneId, stepId, status, name, screenshots, type, data, idCostumer, created_at, updated_at) VALUES
			(77, 44, 55, 9, 1, 'Open', '[]', 'selenium', '{}', 11, NULL, NULL)`,
		`INSERT INTO result_exports (id, idCostumer, performedTestCycleId, format, status, filename, contentType, payload, errorMessage, expiresAt, created_at, updated_at) VALUES
			(90, 11, 44, 'json', 'queued', 'idelium-run-44.json', 'application/json', NULL, NULL, '2026-09-01 09:00:00', NULL, NULL),
			(91, 11, 44, 'json', 'completed', 'idelium-run-44.json', 'application/json', '{\"ok\":true}', NULL, '2026-08-30 09:00:00', NULL, NULL),
			(92, 42, 45, 'json', 'completed', 'idelium-run-45.json', 'application/json', '{\"foreign\":true}', NULL, '2026-09-01 09:00:00', NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare result export fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}
	request := httptest.NewRequest(http.MethodPost, "/admin/result-exports", nil)
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	descriptor, err := repository.CreateResultExport(request, actor, browserauth.ResultExportCreate{PerformedTestCycleID: 44, Format: "json", Now: now})
	if err != nil {
		t.Fatalf("CreateResultExport() returned an error: %v", err)
	}
	if descriptor.Status != "completed" || !descriptor.Ready || descriptor.Filename != "idelium-run-44.json" || descriptor.ContentType != "application/json" {
		t.Fatalf("unexpected export descriptor: %#v", descriptor)
	}
	download, err := repository.DownloadResultExport(request, actor, descriptor.ID, now)
	if err != nil || !strings.Contains(download.Payload, "result-export.v1") || !strings.Contains(download.Payload, "Checkout") {
		t.Fatalf("unexpected export download: %#v err=%v", download, err)
	}
	shown, err := repository.GetResultExport(request, actor, descriptor.ID)
	if err != nil || shown.ID != descriptor.ID || !shown.Authorized {
		t.Fatalf("unexpected export show: %#v err=%v", shown, err)
	}
	if _, err := repository.CreateResultExport(request, actor, browserauth.ResultExportCreate{PerformedTestCycleID: 45, Format: "json", Now: now}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign run to be hidden, got %v", err)
	}
	if _, err := repository.DownloadResultExport(request, actor, 90, now); !errors.Is(err, browserauth.ErrConflict) {
		t.Fatalf("expected queued export conflict, got %v", err)
	}
	if _, err := repository.DownloadResultExport(request, actor, 91, now); !errors.Is(err, browserauth.ErrGone) {
		t.Fatalf("expected expired export gone, got %v", err)
	}
	if _, err := repository.GetResultExport(request, actor, 92); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign export to be hidden, got %v", err)
	}
}

func TestBrowserArtifactDescriptorsIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS audit_events",
		"DROP TABLE IF EXISTS artifact_descriptors",
		"DROP TABLE IF EXISTS performed_test_cycles",
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY,
			testCycleId BIGINT NULL,
			date DATETIME NULL,
			status INT NOT NULL DEFAULT 0,
			idCostumer BIGINT NOT NULL
		)`,
		`CREATE TABLE artifact_descriptors (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer BIGINT NOT NULL,
			idProject BIGINT NOT NULL,
			performedTestCycleId BIGINT NOT NULL,
			performedTestId BIGINT NULL,
			performedStepId BIGINT NULL,
			artifactType VARCHAR(64) NOT NULL,
			name VARCHAR(255) NOT NULL,
			contentType VARCHAR(128) NOT NULL,
			sizeBytes BIGINT UNSIGNED NOT NULL,
			checksumSha256 VARCHAR(64) NOT NULL,
			storageKey VARCHAR(512) NOT NULL,
			state VARCHAR(32) NOT NULL DEFAULT 'available',
			retentionUntil TIMESTAMP NULL,
			metadata JSON NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE audit_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			actorUserId BIGINT NULL,
			actorTenantId BIGINT NULL,
			activeTenantId BIGINT NOT NULL,
			idProject BIGINT NULL,
			action VARCHAR(128) NOT NULL,
			targetType VARCHAR(128) NOT NULL,
			targetId VARCHAR(128) NULL,
			beforeValues JSON NULL,
			afterValues JSON NULL,
			result VARCHAR(32) NOT NULL,
			sourceIp VARCHAR(64) NULL,
			correlationId CHAR(36) NOT NULL,
			metadata JSON NULL,
			created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO performed_test_cycles (id, testCycleId, date, status, idCostumer) VALUES
			(44, 5, '2026-08-31 09:00:00', 1, 11),
			(45, 5, '2026-08-31 09:00:00', 1, 42)`,
		`INSERT INTO artifact_descriptors (id, idCostumer, idProject, performedTestCycleId, performedTestId, performedStepId, artifactType, name, contentType, sizeBytes, checksumSha256, storageKey, state, retentionUntil, metadata, created_at, updated_at) VALUES
			(90, 11, 5, 44, NULL, NULL, 'screenshot', 'z-last.png', 'image/png', 10, 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'tenant/11/z-last.png', 'available', '2026-09-30 12:00:00', '{\"Authorization\":\"secret\",\"browser\":\"chrome\"}', NULL, NULL),
			(91, 11, 5, 44, NULL, NULL, 'json', 'a-first.json', 'application/json', 12, 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'tenant/11/a-first.json', 'available', '2026-09-30 12:00:00', NULL, NULL, NULL),
			(92, 42, 5, 45, NULL, NULL, 'json', 'foreign.json', 'application/json', 12, 'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', 'tenant/42/foreign.json', 'available', '2026-09-30 12:00:00', NULL, NULL, NULL)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare artifact descriptor fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}
	request := httptest.NewRequest(http.MethodGet, "/admin/projects/5/performed-test-cycles/44/artifacts", nil)
	descriptors, err := repository.ListArtifactDescriptors(request, actor, 5, 44)
	if err != nil {
		t.Fatalf("ListArtifactDescriptors() returned an error: %v", err)
	}
	if len(descriptors) != 2 || descriptors[0].ID != 91 || descriptors[1].ID != 90 {
		t.Fatalf("expected Laravel-compatible artifactType/name ordering, got %#v", descriptors)
	}
	if string(descriptors[1].Metadata) != `{"Authorization":"[REDACTED]","browser":"chrome"}` {
		t.Fatalf("expected sensitive metadata to be redacted, got %s", descriptors[1].Metadata)
	}
	shown, err := repository.GetArtifactDescriptor(request, actor, 5, 44, 90)
	if err != nil || shown.ID != 90 || shown.IDCostumer != 11 {
		t.Fatalf("unexpected shown artifact descriptor: %#v err=%v", shown, err)
	}
	if _, err := repository.GetArtifactDescriptor(request, actor, 5, 45, 92); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign artifact to be hidden, got %v", err)
	}
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	created, err := repository.RegisterArtifactDescriptor(request, actor, browserauth.ArtifactDescriptorCreate{
		IDProject:            5,
		PerformedTestCycleID: 44,
		ArtifactType:         "log",
		Name:                 "runner.log",
		ContentType:          "text/plain",
		SizeBytes:            24,
		ChecksumSHA256:       "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
		StorageKey:           "tenant/11/runner.log",
		Metadata:             []byte(`{"sessionToken":"unsafe","source":"runner"}`),
		Now:                  now,
	})
	if err != nil {
		t.Fatalf("RegisterArtifactDescriptor() returned an error: %v", err)
	}
	if created.IDCostumer != 11 || created.State != "available" || created.ChecksumSHA256 != strings.Repeat("d", 64) {
		t.Fatalf("unexpected created descriptor: %#v", created)
	}
	if string(created.Metadata) != `{"sessionToken":"[REDACTED]","source":"runner"}` {
		t.Fatalf("expected created metadata redaction, got %s", created.Metadata)
	}
	if _, err := repository.RegisterArtifactDescriptor(request, actor, browserauth.ArtifactDescriptorCreate{IDProject: 5, PerformedTestCycleID: 45, ArtifactType: "json", Name: "foreign.json", ContentType: "application/json", SizeBytes: 1, ChecksumSHA256: strings.Repeat("a", 64), StorageKey: "tenant/42/foreign.json", Now: now}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign run write to be hidden, got %v", err)
	}
	held, err := repository.SetArtifactLegalHold(request, actor, browserauth.ArtifactLifecycleUpdate{ProjectID: 5, PerformedTestCycleID: 44, ArtifactDescriptorID: 90, Enabled: true, Reason: stringPtr("Investigation hold"), Now: now})
	if err != nil || !strings.Contains(string(held.Metadata), "legalHold") {
		t.Fatalf("unexpected legal hold descriptor: %#v err=%v", held, err)
	}
	var validationFailure browserauth.ValidationFailure
	if _, err := repository.MarkArtifactDeleted(request, actor, browserauth.ArtifactLifecycleUpdate{ProjectID: 5, PerformedTestCycleID: 44, ArtifactDescriptorID: 90, Now: now}); !errors.As(err, &validationFailure) {
		t.Fatalf("expected legal hold delete validation, got %v", err)
	}
	released, err := repository.SetArtifactLegalHold(request, actor, browserauth.ArtifactLifecycleUpdate{ProjectID: 5, PerformedTestCycleID: 44, ArtifactDescriptorID: 90, Enabled: false, Now: now})
	if err != nil || !strings.Contains(string(released.Metadata), `"enabled":false`) {
		t.Fatalf("unexpected legal hold release: %#v err=%v", released, err)
	}
	archived, err := repository.ArchiveArtifact(request, actor, browserauth.ArtifactLifecycleUpdate{ProjectID: 5, PerformedTestCycleID: 44, ArtifactDescriptorID: 90, Reason: stringPtr("Retention grace period"), RestoreBy: stringPtr("2026-09-30T00:00:00Z"), Now: now})
	if err != nil || archived.State != "archived" || !strings.Contains(string(archived.Metadata), "Retention grace period") {
		t.Fatalf("unexpected archived descriptor: %#v err=%v", archived, err)
	}
	restored, err := repository.RestoreArtifact(request, actor, browserauth.ArtifactLifecycleUpdate{ProjectID: 5, PerformedTestCycleID: 44, ArtifactDescriptorID: 90, Now: now})
	if err != nil || restored.State != "available" || !strings.Contains(string(restored.Metadata), "restoredAt") {
		t.Fatalf("unexpected restored descriptor: %#v err=%v", restored, err)
	}
	var auditCount int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events WHERE activeTenantId = ? AND targetType = 'artifact_descriptor' AND targetId = ?", 11, "90").Scan(&auditCount); err != nil || auditCount != 4 {
		t.Fatalf("expected four lifecycle audit events, count=%d err=%v", auditCount, err)
	}
}

func TestBrowserGridBulkOperationsIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, statement := range []string{
		"DROP TABLE IF EXISTS grid_bulk_operation_jobs",
		"DROP TABLE IF EXISTS grid_query_snapshots",
		"DROP TABLE IF EXISTS audit_events",
		"DROP TABLE IF EXISTS projects",
		`CREATE TABLE projects (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT NULL,
			idCostumer BIGINT NOT NULL,
			archivedAt TIMESTAMP NULL,
			tags JSON NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE grid_query_snapshots (
			id CHAR(36) PRIMARY KEY,
			idCostumer BIGINT NOT NULL,
			actorUserId BIGINT NOT NULL,
			resourceType VARCHAR(64) NOT NULL,
			query JSON NOT NULL,
			entityIds JSON NOT NULL,
			total INT UNSIGNED NOT NULL,
			expiresAt TIMESTAMP NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE grid_bulk_operation_jobs (
			id CHAR(36) PRIMARY KEY,
			querySnapshotId CHAR(36) NOT NULL,
			idCostumer BIGINT NOT NULL,
			actorUserId BIGINT NOT NULL,
			resourceType VARCHAR(64) NOT NULL,
			action VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			payload JSON NULL,
			requestedCount INT UNSIGNED NOT NULL,
			processedCount INT UNSIGNED NOT NULL,
			failedCount INT UNSIGNED NOT NULL,
			result JSON NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		`CREATE TABLE audit_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			actorUserId BIGINT NULL,
			actorTenantId BIGINT NULL,
			activeTenantId BIGINT NOT NULL,
			idProject BIGINT NULL,
			action VARCHAR(128) NOT NULL,
			targetType VARCHAR(128) NOT NULL,
			targetId VARCHAR(128) NULL,
			beforeValues JSON NULL,
			afterValues JSON NULL,
			result VARCHAR(32) NOT NULL,
			sourceIp VARCHAR(64) NULL,
			correlationId VARCHAR(128) NOT NULL,
			metadata JSON NULL,
			created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO projects (id, name, description, idCostumer, archivedAt, tags, created_at, updated_at) VALUES
			(1, 'Checkout', '=Protected formula', 11, NULL, NULL, NOW(), NOW()),
			(2, 'Login', 'Login flow', 11, NULL, '["existing"]', NOW(), NOW()),
			(3, 'Foreign', 'Other tenant', 42, NULL, NULL, NOW(), NOW())`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare grid fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	request := httptest.NewRequest(http.MethodPost, "/admin/grid/query-snapshots", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("X-Correlation-ID", "grid-integration")
	snapshot, err := repository.CreateGridQuerySnapshot(request, actor, browserauth.GridQuerySnapshotCreate{ResourceType: "projects", Query: browserauth.GridQuery{Sort: "id", Direction: "asc", Filters: map[string]any{}, Raw: map[string]any{}}, Now: now})
	if err != nil || snapshot.Total != 2 {
		t.Fatalf("unexpected tenant-scoped snapshot: %#v err=%v", snapshot, err)
	}

	tagJob, err := repository.CreateGridBulkJob(request, actor, browserauth.GridBulkJobCreate{QuerySnapshotID: snapshot.ID, Action: "tag", Tags: []string{"critical", "existing"}, Payload: map[string]any{"tags": []any{"critical", "existing"}}, Now: now})
	if err != nil || tagJob.Status != "completed" || tagJob.ProcessedCount != 2 || tagJob.FailedCount != 0 {
		t.Fatalf("unexpected grid tag job: %#v err=%v", tagJob, err)
	}
	var tags string
	if err := database.QueryRowContext(ctx, "SELECT tags FROM projects WHERE id = 2 AND idCostumer = 11").Scan(&tags); err != nil || tags != `["existing","critical"]` {
		t.Fatalf("expected stable unique project tags, got %q err=%v", tags, err)
	}
	foreignActor := browserauth.User{ID: 8, TenantID: 42, ActiveTenantID: 42, Role: 2}
	if _, err := repository.CreateGridBulkJob(request, foreignActor, browserauth.GridBulkJobCreate{QuerySnapshotID: snapshot.ID, Action: "export", Now: now}); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-tenant snapshot to be hidden, got %v", err)
	}
	otherActor := browserauth.User{ID: 8, TenantID: 11, ActiveTenantID: 11, Role: 2}
	if _, err := repository.GetGridBulkJob(request, otherActor, tagJob.ID); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-actor job to be hidden, got %v", err)
	}

	exportSnapshot, err := repository.CreateGridQuerySnapshot(request, actor, browserauth.GridQuerySnapshotCreate{ResourceType: "projects", Query: browserauth.GridQuery{Search: "Checkout", Sort: "name", Direction: "asc", Filters: map[string]any{}, Raw: map[string]any{"q": "Checkout"}}, Now: now})
	if err != nil || exportSnapshot.Total != 1 {
		t.Fatalf("unexpected export snapshot: %#v err=%v", exportSnapshot, err)
	}
	exportJob, err := repository.CreateGridBulkJob(request, actor, browserauth.GridBulkJobCreate{QuerySnapshotID: exportSnapshot.ID, Action: "export", Now: now})
	if err != nil {
		t.Fatalf("CreateGridBulkJob(export) returned an error: %v", err)
	}
	exported, err := repository.ExportGridBulkJob(request, actor, exportJob.ID, now)
	if err != nil || !strings.Contains(exported.Payload, "'=Protected formula") || strings.Contains(exported.Payload, "Foreign") {
		t.Fatalf("unexpected safe tenant-scoped grid export: %#v err=%v", exported, err)
	}
	var auditCount int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events WHERE activeTenantId = 11 AND action IN ('grid.bulk.tag', 'grid.bulk.export')").Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("expected two grid audit events, count=%d err=%v", auditCount, err)
	}
}

func TestBrowserIntegrationEndpointsAndDeliveriesIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()
	t.Setenv("APP_KEY", "base64:"+base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, statement := range []string{
		"DROP TABLE IF EXISTS integration_deliveries",
		"DROP TABLE IF EXISTS integration_endpoints",
		"DROP TABLE IF EXISTS audit_events",
		"DROP TABLE IF EXISTS projects",
		`CREATE TABLE projects (id BIGINT PRIMARY KEY, name VARCHAR(255) NOT NULL, description TEXT NULL, idCostumer BIGINT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)`,
		`CREATE TABLE integration_endpoints (
			id BIGINT PRIMARY KEY AUTO_INCREMENT, idCostumer BIGINT NOT NULL, idProject BIGINT NOT NULL,
			name VARCHAR(128) NOT NULL, adapter VARCHAR(32) NOT NULL, url VARCHAR(2048) NOT NULL,
			secretEncrypted TEXT NOT NULL, events JSON NULL, status VARCHAR(32) NOT NULL,
			metadata JSON NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL,
			UNIQUE KEY integration_endpoint_name_unique (idCostumer, idProject, name)
		)`,
		`CREATE TABLE integration_deliveries (
			id BIGINT PRIMARY KEY AUTO_INCREMENT, idCostumer BIGINT NOT NULL, idProject BIGINT NOT NULL,
			integrationEndpointId BIGINT NOT NULL, event VARCHAR(128) NOT NULL, deliveryId VARCHAR(96) NOT NULL UNIQUE,
			idempotencyKey VARCHAR(160) NOT NULL, schemaVersion VARCHAR(64) NOT NULL, payloadDigest VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL, attempts SMALLINT UNSIGNED NOT NULL DEFAULT 0, responseStatus SMALLINT UNSIGNED NULL,
			lastError TEXT NULL, nextAttemptAt TIMESTAMP NULL, sentAt TIMESTAMP NULL, payload JSON NULL,
			created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL,
			UNIQUE KEY integration_delivery_idempotency_unique (idCostumer, idProject, integrationEndpointId, idempotencyKey)
		)`,
		`CREATE TABLE audit_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT, actorUserId BIGINT NULL, actorTenantId BIGINT NULL,
			activeTenantId BIGINT NOT NULL, idProject BIGINT NULL, action VARCHAR(128) NOT NULL,
			targetType VARCHAR(128) NOT NULL, targetId VARCHAR(128) NULL, beforeValues JSON NULL,
			afterValues JSON NULL, result VARCHAR(32) NOT NULL, sourceIp VARCHAR(64) NULL,
			correlationId VARCHAR(128) NOT NULL, metadata JSON NULL, created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO projects (id, name, description, idCostumer) VALUES (5, 'Primary', 'Primary', 11), (6, 'Foreign', 'Foreign', 42)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare integration endpoint fixture %q: %v", statement, err)
		}
	}

	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Email: "admin@example.test", Role: 2}
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	request := httptest.NewRequest(http.MethodPost, "/admin/projects/5/integrations", nil)
	request.Header.Set("X-Correlation-ID", "integration-endpoint-test")
	created, err := repository.CreateIntegrationEndpoint(request, actor, browserauth.IntegrationEndpointCreate{ProjectID: 5, Name: "Release events", Adapter: "webhook", URL: "https://93.184.216.34/hooks/idelium", Secret: "super-secret-value", Events: []string{"*"}, Now: now})
	if err != nil || !created.SecretConfigured || created.SchemaVersion != integrationSchemaVersion {
		t.Fatalf("unexpected created integration endpoint: %#v err=%v", created, err)
	}
	var encrypted string
	if err := database.QueryRowContext(ctx, "SELECT secretEncrypted FROM integration_endpoints WHERE id = ? AND idCostumer = ?", created.ID, 11).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	key, _ := integrations.ParseApplicationKey(os.Getenv("APP_KEY"))
	plaintext, err := integrations.DecryptLaravelString(key, encrypted)
	if err != nil || plaintext != "super-secret-value" || strings.Contains(encrypted, plaintext) {
		t.Fatalf("integration secret did not preserve encrypted Laravel compatibility: plaintext=%q err=%v", plaintext, err)
	}
	listed, err := repository.ListIntegrationEndpoints(request, actor, 5)
	if err != nil || len(listed) != 1 || listed[0].Name != "Release events" {
		t.Fatalf("unexpected integration endpoint list: %#v err=%v", listed, err)
	}
	if _, err := repository.ListIntegrationEndpoints(request, actor, 6); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected foreign project endpoints to be hidden, got %v", err)
	}

	disabled, err := repository.UpdateIntegrationEndpointStatus(request, actor, 5, created.ID, "disabled", now.Add(time.Minute))
	if err != nil || disabled.Status != "disabled" {
		t.Fatalf("unexpected disabled integration endpoint: %#v err=%v", disabled, err)
	}
	if _, err := repository.CreateIntegrationTestDelivery(request, actor, 5, created.ID, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected disabled endpoint delivery validation")
	}
	if _, err := repository.RotateIntegrationEndpointSecret(request, actor, 5, created.ID, "rotated-secret-value", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("RotateIntegrationEndpointSecret() returned an error: %v", err)
	}
	if _, err := repository.UpdateIntegrationEndpointStatus(request, actor, 5, created.ID, "active", now.Add(4*time.Minute)); err != nil {
		t.Fatalf("reactivate integration endpoint: %v", err)
	}
	delivery, err := repository.CreateIntegrationTestDelivery(request, actor, 5, created.ID, now.Add(5*time.Minute))
	if err != nil || delivery.Status != "pending" || delivery.Event != "integration.test" {
		t.Fatalf("unexpected integration test delivery: %#v err=%v", delivery, err)
	}
	dispatcher := integrations.Dispatcher{Store: repository, ApplicationKey: key, Now: func() time.Time { return now.Add(6 * time.Minute) }, Client: integrationDoerFunc(func(outbound *http.Request) (*http.Response, error) {
		if outbound.Header.Get("Idelium-Delivery-Id") != delivery.DeliveryID || outbound.Header.Get("Idelium-Signature") == "" {
			t.Fatalf("missing integration delivery signature headers: %#v", outbound.Header)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	if err := dispatcher.Dispatch(ctx, delivery.ID); err != nil {
		t.Fatalf("Dispatch() returned an error: %v", err)
	}
	dispatched, err := repository.integrationDelivery(ctx, actor, 5, delivery.ID)
	if err != nil || dispatched.Status != "sent" || dispatched.Attempts != 1 {
		t.Fatalf("unexpected dispatched delivery: %#v err=%v", dispatched, err)
	}
	if _, err := repository.ReplayIntegrationDelivery(request, actor, 6, delivery.ID, now.Add(6*time.Minute)); !errors.Is(err, browserauth.ErrNotFound) {
		t.Fatalf("expected cross-tenant delivery replay to be hidden, got %v", err)
	}
	if _, err := database.ExecContext(ctx, "UPDATE integration_deliveries SET status = 'dead_letter', attempts = 3 WHERE id = ?", delivery.ID); err != nil {
		t.Fatal(err)
	}
	deadLetters, err := repository.ListIntegrationDeliveries(request, actor, 5, "dead_letter")
	if err != nil || len(deadLetters) != 1 || deadLetters[0].DeliveryID != delivery.DeliveryID {
		t.Fatalf("unexpected dead-letter list: %#v err=%v", deadLetters, err)
	}
	replayed, err := repository.ReplayIntegrationDelivery(request, actor, 5, delivery.ID, now.Add(7*time.Minute))
	if err != nil || replayed.Status != "pending" || replayed.NextAttemptAt != nil {
		t.Fatalf("unexpected replayed integration delivery: %#v err=%v", replayed, err)
	}
	var auditCount int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events WHERE activeTenantId = 11 AND idProject = 5").Scan(&auditCount); err != nil || auditCount != 7 {
		t.Fatalf("expected seven integration audit events, count=%d err=%v", auditCount, err)
	}
}

func TestBrowserAuditEventReadsAreTenantScopedFilteredAndRedactedIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, statement := range []string{
		"DROP TABLE IF EXISTS audit_events",
		`CREATE TABLE audit_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT, actorUserId BIGINT NULL, actorTenantId BIGINT NULL,
			activeTenantId BIGINT NOT NULL, idProject BIGINT NULL, action VARCHAR(128) NOT NULL,
			targetType VARCHAR(128) NOT NULL, targetId VARCHAR(128) NULL, beforeValues JSON NULL,
			afterValues JSON NULL, result VARCHAR(32) NOT NULL, sourceIp VARCHAR(64) NULL,
			correlationId CHAR(36) NOT NULL, metadata JSON NULL, created_at TIMESTAMP NOT NULL
		)`,
		`INSERT INTO audit_events (actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata, created_at) VALUES
			(7, 11, 11, 5, 'secret.changed', 'environment', '1', NULL, '{"apiKey":"legacy-unsafe","nested":{"password":"legacy-unsafe"}}', 'success', '192.0.2.1', '018fb9d0-1f16-7abc-9f2f-4e1d8457f001', '{"sessionToken":"legacy-unsafe"}', '2026-08-31 12:00:00'),
			(NULL, 11, 11, NULL, 'project.read', 'project', '5', NULL, NULL, 'success', NULL, '018fb9d0-1f16-7abc-9f2f-4e1d8457f002', NULL, '2026-08-30 12:00:00'),
			(8, 42, 42, 6, 'secret.changed', 'environment', '2', NULL, '{"apiKey":"foreign-unsafe"}', 'success', '192.0.2.2', '018fb9d0-1f16-7abc-9f2f-4e1d8457f003', NULL, '2026-08-31 13:00:00')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare audit event fixture %q: %v", statement, err)
		}
	}
	repository := NewBrowserAuthRepository(database)
	actor := browserauth.User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}
	from := time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC)
	request := httptest.NewRequest(http.MethodGet, "/audit-events?action=secret.changed", nil)
	events, err := repository.ListAuditEvents(request, actor, browserauth.AuditEventFilter{Action: "secret.changed", From: &from, Limit: 100})
	if err != nil || len(events) != 1 || events[0].ActiveTenantID != 11 || events[0].TargetID == nil || *events[0].TargetID != "1" {
		t.Fatalf("unexpected tenant-scoped audit events: %#v err=%v", events, err)
	}
	if events[0].AfterValues["apiKey"] != "[REDACTED]" || events[0].Metadata["sessionToken"] != "[REDACTED]" {
		t.Fatalf("expected defense-in-depth audit redaction, got %#v", events[0])
	}
	allEvents, err := repository.ListAuditEvents(request, actor, browserauth.AuditEventFilter{Limit: 1})
	if err != nil || len(allEvents) != 1 || allEvents[0].Action != "secret.changed" {
		t.Fatalf("expected bounded newest-first audit list, got %#v err=%v", allEvents, err)
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
		"DROP TABLE IF EXISTS platforms",
		"CREATE TABLE types (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE statuses (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL)",
		"CREATE TABLE locations (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE brand_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, brand VARCHAR(255) NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE model_devices (id BIGINT PRIMARY KEY AUTO_INCREMENT, model VARCHAR(255) NOT NULL, idBrand INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE os (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, type INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE version_os (id BIGINT PRIMARY KEY AUTO_INCREMENT, version VARCHAR(255) NOT NULL, idOs INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE browsers (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255) NOT NULL, idOs INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		"CREATE TABLE version_browsers (id BIGINT PRIMARY KEY AUTO_INCREMENT, version VARCHAR(255) NOT NULL, idBrowser INT NOT NULL, created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
		`CREATE TABLE platforms (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			type INT NOT NULL,
			hostname VARCHAR(255) NOT NULL,
			location INT NOT NULL,
			os INT NOT NULL,
			osversion INT NOT NULL,
			brand INT NOT NULL,
			browser INT NOT NULL,
			brandDescription VARCHAR(255) NOT NULL,
			osDescription VARCHAR(255) NOT NULL,
			browserDescription VARCHAR(255) NOT NULL,
			status INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		)`,
		"INSERT INTO types (id, name) VALUES (2, 'mobile'), (1, 'desktop')",
		"INSERT INTO statuses (id, name) VALUES (2, 'busy'), (1, 'free')",
		"INSERT INTO locations (id, name, created_at, updated_at) VALUES (2, 'us-east', NULL, NULL), (1, 'eu-west', NULL, NULL)",
		"INSERT INTO brand_devices (id, brand, created_at, updated_at) VALUES (2, 'Samsung', NULL, NULL), (1, 'Apple', NULL, NULL)",
		"INSERT INTO model_devices (id, model, idBrand, created_at, updated_at) VALUES (3, 'Galaxy', 2, NULL, NULL), (2, 'iPad', 1, NULL, NULL), (1, 'iPhone', 1, NULL, NULL)",
		"INSERT INTO os (id, name, type, created_at, updated_at) VALUES (3, 'android', 2, NULL, NULL), (2, 'windows', 1, NULL, NULL), (1, 'linux', 1, NULL, NULL)",
		"INSERT INTO version_os (id, version, idOs, created_at, updated_at) VALUES (3, '13', 2, NULL, NULL), (2, '15', 1, NULL, NULL), (1, '14', 1, NULL, NULL)",
		"INSERT INTO browsers (id, name, idOs, created_at, updated_at) VALUES (3, 'safari', 2, NULL, NULL), (2, 'firefox', 1, NULL, NULL), (1, 'chrome', 1, NULL, NULL)",
		"INSERT INTO version_browsers (id, version, idBrowser, created_at, updated_at) VALUES (3, '17', 2, NULL, NULL), (2, '125', 1, NULL, NULL), (1, '124', 1, NULL, NULL)",
		`INSERT INTO platforms
			(id, type, hostname, location, os, osversion, brand, browser, brandDescription, osDescription, browserDescription, status, created_at, updated_at)
		 VALUES
			(1, 1, 'https://chrome-node.example:4444', 1, 1, 1, 1, 1, 'Dell', 'Linux', 'chrome', 1, NULL, '2026-08-25 10:00:00'),
			(2, 1, 'https://firefox-node.example:4444', 2, 1, 1, 1, 2, 'Dell', 'Linux', 'firefox', 2, NULL, '2026-08-25 10:00:00'),
			(3, 2, 'https://mobile-node.example:4723', 1, 3, 3, 2, 3, 'Samsung', 'Android', 'appium', 1, NULL, '2026-08-25 10:00:00')`,
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

	managedPlatforms, err := repository.ListManagedPlatforms(ctx, platforms.ManagedPlatformQuery{TypeID: 1})
	if err != nil {
		t.Fatalf("ListManagedPlatforms() returned an error: %v", err)
	}
	if len(managedPlatforms.Data) != 2 || managedPlatforms.Data[0].Hostname != "https://chrome-node.example:4444" {
		t.Fatalf("unexpected managed platforms: %#v", managedPlatforms.Data)
	}

	launchTargets, err := repository.ListLaunchTargets(ctx, 10)
	if err != nil {
		t.Fatalf("ListLaunchTargets() returned an error: %v", err)
	}
	if len(launchTargets) != 4 {
		t.Fatalf("expected pool plus three managed launch targets, got %#v", launchTargets)
	}
	if launchTargets[0].ID != "platform-pool" || launchTargets[0].Runtime != "selenium" {
		t.Fatalf("default platform-pool target missing or malformed: %#v", launchTargets[0])
	}
	if launchTargets[1].ID != "platform-1" || launchTargets[1].Health != "healthy" || launchTargets[1].Capacity.Available != 1 {
		t.Fatalf("healthy platform target missing expected capacity: %#v", launchTargets[1])
	}
	if launchTargets[2].ID != "platform-2" || launchTargets[2].Health != "disabled" || launchTargets[2].Capacity.Available != 0 {
		t.Fatalf("disabled platform target missing expected capacity: %#v", launchTargets[2])
	}
	if launchTargets[3].ID != "platform-3" || launchTargets[3].Runtime != "appium" {
		t.Fatalf("mobile platform target missing appium runtime: %#v", launchTargets[3])
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

func TestCLIPerformedCycleRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS performed_test_cycles",
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
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			testCycleId INT NOT NULL,
			date DATETIME NOT NULL,
			status INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL,
			idempotencyKey VARCHAR(128) NULL,
			UNIQUE KEY performed_test_cycles_tenant_idempotency_unique (idCostumer, idempotencyKey)
		)`,
		`INSERT INTO test_cycles
			(id, name, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(7, 'Own cycle', 'Own cycle', JSON_ARRAY(), 10, NULL, NULL, 42),
			(8, 'Foreign cycle', 'Foreign cycle', JSON_ARRAY(), 10, NULL, NULL, 99)`,
		`INSERT INTO performed_test_cycles
			(id, testCycleId, date, status, created_at, updated_at, idCostumer)
		 VALUES
			(44, 7, '2026-08-26 10:00:00', 0, NULL, NULL, 42),
			(45, 8, '2026-08-26 10:00:00', 0, NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI performed-cycle fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIPerformedCycleRepository(database)
	performedCycleID, err := repository.CreatePerformedCycle(ctx, 42, cliapi.CreatePerformedCycleRequest{TestCycleID: 7})
	if err != nil {
		t.Fatalf("CreatePerformedCycle() returned an error: %v", err)
	}
	var storedCustomerID int64
	var storedStatus int
	if err := database.QueryRowContext(
		ctx,
		"SELECT idCostumer, status FROM performed_test_cycles WHERE id = ?",
		performedCycleID,
	).Scan(&storedCustomerID, &storedStatus); err != nil {
		t.Fatalf("read created performed cycle: %v", err)
	}
	if storedCustomerID != 42 || storedStatus != 0 {
		t.Fatalf("created performed cycle was not tenant-scoped with default status: customer=%d status=%d", storedCustomerID, storedStatus)
	}

	firstID, err := repository.CreatePerformedCycle(ctx, 42, cliapi.CreatePerformedCycleRequest{TestCycleID: 7, IdempotencyKey: "cycle-retry-0001"})
	if err != nil {
		t.Fatalf("create idempotent performed cycle: %v", err)
	}
	replayedID, err := repository.CreatePerformedCycle(ctx, 42, cliapi.CreatePerformedCycleRequest{TestCycleID: 7, IdempotencyKey: "cycle-retry-0001"})
	if err != nil || replayedID != firstID {
		t.Fatalf("expected idempotent replay id %d, got %d with error %v", firstID, replayedID, err)
	}
	foreignID, err := repository.CreatePerformedCycle(ctx, 99, cliapi.CreatePerformedCycleRequest{TestCycleID: 8, IdempotencyKey: "cycle-retry-0001"})
	if err != nil || foreignID == firstID {
		t.Fatalf("expected tenant-isolated idempotency key, got id %d with error %v", foreignID, err)
	}

	_, err = repository.CreatePerformedCycle(ctx, 42, cliapi.CreatePerformedCycleRequest{TestCycleID: 8})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant source cycle to be hidden, got %v", err)
	}

	updatedID, err := repository.UpdatePerformedCycle(ctx, 42, cliapi.UpdatePerformedCycleRequest{TestCycleID: 44, Status: 2})
	if err != nil {
		t.Fatalf("UpdatePerformedCycle() returned an error: %v", err)
	}
	if updatedID != 44 {
		t.Fatalf("unexpected updated performed-cycle id: %d", updatedID)
	}
	if err := database.QueryRowContext(ctx, "SELECT status FROM performed_test_cycles WHERE id = ? AND idCostumer = ?", 44, 42).Scan(&storedStatus); err != nil {
		t.Fatalf("read updated performed cycle: %v", err)
	}
	if storedStatus != 2 {
		t.Fatalf("unexpected updated performed-cycle status: %d", storedStatus)
	}

	_, err = repository.UpdatePerformedCycle(ctx, 42, cliapi.UpdatePerformedCycleRequest{TestCycleID: 45, Status: 1})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed cycle to be hidden, got %v", err)
	}
}

func TestCLIPerformedCycleRepositoryPersistsExecutionContextWhenColumnExists(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS performed_test_cycles",
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
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			testCycleId INT NOT NULL,
			date DATETIME NOT NULL,
			status INT NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL,
			executionContext JSON NULL
		)`,
		`INSERT INTO test_cycles
			(id, name, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(7, 'Own cycle', 'Own cycle', JSON_ARRAY(), 10, NULL, NULL, 42)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI performed-cycle snapshot fixture %q: %v", statement, err)
		}
	}

	executionContext := `{"environment":"demo","browser":"firefox","operatingSystem":"darwin","apiToken":"[REDACTED]"}`
	repository := NewCLIPerformedCycleRepository(database)
	performedCycleID, err := repository.CreatePerformedCycle(
		ctx,
		42,
		cliapi.CreatePerformedCycleRequest{
			TestCycleID:              7,
			ExecutionContextProvided: true,
			ExecutionContext:         &executionContext,
		},
	)
	if err != nil {
		t.Fatalf("CreatePerformedCycle() returned an error: %v", err)
	}

	var storedContext string
	if err := database.QueryRowContext(
		ctx,
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(executionContext, '$.browser')) FROM performed_test_cycles WHERE id = ? AND idCostumer = ?",
		performedCycleID,
		42,
	).Scan(&storedContext); err != nil {
		t.Fatalf("read persisted execution context: %v", err)
	}
	if storedContext != "firefox" {
		t.Fatalf("unexpected persisted browser snapshot: %s", storedContext)
	}

	var storedToken string
	if err := database.QueryRowContext(
		ctx,
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(executionContext, '$.apiToken')) FROM performed_test_cycles WHERE id = ? AND idCostumer = ?",
		performedCycleID,
		42,
	).Scan(&storedToken); err != nil {
		t.Fatalf("read persisted redacted token: %v", err)
	}
	if storedToken != "[REDACTED]" {
		t.Fatalf("execution context stored an unredacted token: %s", storedToken)
	}
}

func TestCLIPerformedTestRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS performed_tests",
		"DROP TABLE IF EXISTS performed_test_cycles",
		"DROP TABLE IF EXISTS tests",
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer INT NOT NULL
		)`,
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
		`CREATE TABLE performed_tests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			testCycleDoneId INT NOT NULL,
			testId INT NOT NULL,
			status INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			postmanData LONGTEXT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		"INSERT INTO performed_test_cycles (id, idCostumer) VALUES (7, 42), (8, 99)",
		`INSERT INTO tests
			(id, name, description, config, idProject, created_at, updated_at, idCostumer)
		 VALUES
			(9, 'Own test', 'Own test', JSON_ARRAY(), 10, NULL, NULL, 42),
			(10, 'Foreign test', 'Foreign test', JSON_ARRAY(), 10, NULL, NULL, 99)`,
		`INSERT INTO performed_tests
			(id, testCycleDoneId, testId, status, name, postmanData, created_at, updated_at, idCostumer)
		 VALUES
			(55, 7, 9, 0, 'Existing own test', NULL, NULL, NULL, 42),
			(56, 8, 10, 0, 'Existing foreign test', NULL, NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI performed-test fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIPerformedTestRepository(database)
	performedTestID, err := repository.CreatePerformedTest(ctx, 42, cliapi.CreatePerformedTestRequest{
		TestCycleID: 7,
		TestID:      9,
		Name:        "Created own test",
	})
	if err != nil {
		t.Fatalf("CreatePerformedTest() returned an error: %v", err)
	}
	var storedCustomerID int64
	var storedStatus int
	if err := database.QueryRowContext(
		ctx,
		"SELECT idCostumer, status FROM performed_tests WHERE id = ?",
		performedTestID,
	).Scan(&storedCustomerID, &storedStatus); err != nil {
		t.Fatalf("read created performed test: %v", err)
	}
	if storedCustomerID != 42 || storedStatus != 0 {
		t.Fatalf("created performed test was not tenant-scoped with default status: customer=%d status=%d", storedCustomerID, storedStatus)
	}

	_, err = repository.CreatePerformedTest(ctx, 42, cliapi.CreatePerformedTestRequest{TestCycleID: 8, TestID: 9, Name: "Foreign cycle"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed cycle to be hidden, got %v", err)
	}
	_, err = repository.CreatePerformedTest(ctx, 42, cliapi.CreatePerformedTestRequest{TestCycleID: 7, TestID: 10, Name: "Foreign test"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant test to be hidden, got %v", err)
	}

	postmanData := `[{"request":{"header":{"Authorization":"[REDACTED]"},"url":"https://api.example.test"},"response":{"code":200}}]`
	updatedID, err := repository.UpdatePerformedTest(ctx, 42, cliapi.UpdatePerformedTestRequest{
		TestID:             55,
		Status:             1,
		PostmanDataPresent: true,
		PostmanData:        &postmanData,
	})
	if err != nil {
		t.Fatalf("UpdatePerformedTest() returned an error: %v", err)
	}
	if updatedID != 55 {
		t.Fatalf("unexpected updated performed-test id: %d", updatedID)
	}
	var storedPostmanData sql.NullString
	if err := database.QueryRowContext(
		ctx,
		"SELECT status, postmanData FROM performed_tests WHERE id = ? AND idCostumer = ?",
		55,
		42,
	).Scan(&storedStatus, &storedPostmanData); err != nil {
		t.Fatalf("read updated performed test: %v", err)
	}
	if storedStatus != 1 || !storedPostmanData.Valid || !strings.Contains(storedPostmanData.String, "[REDACTED]") {
		t.Fatalf("unexpected updated performed-test payload: status=%d postmanData=%q", storedStatus, storedPostmanData.String)
	}

	_, err = repository.UpdatePerformedTest(ctx, 42, cliapi.UpdatePerformedTestRequest{TestID: 56, Status: 1})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed test to be hidden, got %v", err)
	}
}

func TestCLIPerformedStepRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, statement := range []string{
		"DROP TABLE IF EXISTS performed_steps",
		"DROP TABLE IF EXISTS performed_tests",
		"DROP TABLE IF EXISTS performed_test_cycles",
		"DROP TABLE IF EXISTS steps",
		`CREATE TABLE performed_test_cycles (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			idCostumer INT NOT NULL
		)`,
		`CREATE TABLE performed_tests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			testCycleDoneId INT NOT NULL,
			testId INT NOT NULL,
			status INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			postmanData LONGTEXT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
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
		`CREATE TABLE performed_steps (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			testCycleDoneId INT NOT NULL,
			testDoneId INT NOT NULL,
			stepId INT NOT NULL,
			status INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			screenshots JSON NOT NULL,
			type VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL,
			idCostumer INT NOT NULL
		)`,
		"INSERT INTO performed_test_cycles (id, idCostumer) VALUES (44, 42), (45, 99)",
		`INSERT INTO performed_tests
			(id, testCycleDoneId, testId, status, name, postmanData, created_at, updated_at, idCostumer)
		 VALUES
			(55, 44, 9, 0, 'Own performed test', NULL, NULL, NULL, 42),
			(56, 45, 10, 0, 'Foreign performed test', NULL, NULL, NULL, 99)`,
		`INSERT INTO steps
			(id, name, description, config, idProject, ` + "`order`" + `, created_at, updated_at, idCostumer)
		 VALUES
			(12, 'Own step', 'Own step', JSON_ARRAY(), 10, 2, NULL, NULL, 42),
			(13, 'Foreign step', 'Foreign step', JSON_ARRAY(), 10, 3, NULL, NULL, 99)`,
		`INSERT INTO performed_steps
			(id, testCycleDoneId, testDoneId, stepId, status, name, screenshots, type, data, created_at, updated_at, idCostumer)
		 VALUES
			(77, 44, 55, 12, 0, 'Existing own step', JSON_ARRAY(), 'selenium', JSON_OBJECT('ok', true), NULL, NULL, 42),
			(78, 45, 56, 13, 0, 'Existing foreign step', JSON_ARRAY(), 'selenium', JSON_OBJECT('ok', true), NULL, NULL, 99)`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare CLI performed-step fixture %q: %v", statement, err)
		}
	}

	repository := NewCLIPerformedStepRepository(database)
	performedStepID, err := repository.CreatePerformedStep(ctx, 42, cliapi.CreatePerformedStepRequest{
		TestCycleID: 44,
		TestID:      55,
		StepID:      12,
		Name:        "Created own step",
		Status:      1,
		Screenshots: "[]",
		Data:        `{"result":"ok","token":"[REDACTED]"}`,
		Type:        "selenium",
	})
	if err != nil {
		t.Fatalf("CreatePerformedStep() returned an error: %v", err)
	}
	var storedCustomerID int64
	var storedStatus int
	var storedData string
	if err := database.QueryRowContext(
		ctx,
		"SELECT idCostumer, status, data FROM performed_steps WHERE id = ?",
		performedStepID,
	).Scan(&storedCustomerID, &storedStatus, &storedData); err != nil {
		t.Fatalf("read created performed step: %v", err)
	}
	if storedCustomerID != 42 || storedStatus != 1 || !strings.Contains(storedData, "[REDACTED]") {
		t.Fatalf("created performed step was not tenant-scoped with expected payload: customer=%d status=%d data=%q", storedCustomerID, storedStatus, storedData)
	}

	_, err = repository.CreatePerformedStep(ctx, 42, cliapi.CreatePerformedStepRequest{TestCycleID: 45, TestID: 55, StepID: 12, Name: "Foreign cycle", Status: 1, Screenshots: "[]", Data: `{}`, Type: "selenium"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed cycle to be hidden, got %v", err)
	}
	_, err = repository.CreatePerformedStep(ctx, 42, cliapi.CreatePerformedStepRequest{TestCycleID: 44, TestID: 56, StepID: 12, Name: "Foreign test", Status: 1, Screenshots: "[]", Data: `{}`, Type: "selenium"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed test to be hidden, got %v", err)
	}
	_, err = repository.CreatePerformedStep(ctx, 42, cliapi.CreatePerformedStepRequest{TestCycleID: 44, TestID: 55, StepID: 13, Name: "Foreign step", Status: 1, Screenshots: "[]", Data: `{}`, Type: "selenium"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant source step to be hidden, got %v", err)
	}

	updatedID, err := repository.UpdatePerformedStep(ctx, 42, cliapi.UpdatePerformedStepRequest{
		StepID:      77,
		Screenshots: `[{"name":"screen.png"}]`,
	})
	if err != nil {
		t.Fatalf("UpdatePerformedStep() returned an error: %v", err)
	}
	if updatedID != 77 {
		t.Fatalf("unexpected updated performed-step id: %d", updatedID)
	}
	var storedScreenshots string
	if err := database.QueryRowContext(
		ctx,
		"SELECT screenshots FROM performed_steps WHERE id = ? AND idCostumer = ?",
		77,
		42,
	).Scan(&storedScreenshots); err != nil {
		t.Fatalf("read updated performed step: %v", err)
	}
	if !strings.Contains(storedScreenshots, "screen.png") {
		t.Fatalf("unexpected updated performed-step screenshots: %q", storedScreenshots)
	}

	_, err = repository.UpdatePerformedStep(ctx, 42, cliapi.UpdatePerformedStepRequest{StepID: 78, Screenshots: "[]"})
	if !errors.Is(err, cliapi.ErrNotFound) {
		t.Fatalf("expected cross-tenant performed step to be hidden, got %v", err)
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

func stringPtr(value string) *string {
	return &value
}
