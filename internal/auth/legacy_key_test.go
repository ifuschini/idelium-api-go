package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeLegacyKeyRepository struct {
	expectedKey string
	customer    Customer
	err         error
	usedAt      time.Time
}

func (repository *fakeLegacyKeyRepository) AuthenticateLegacyCustomerKey(ctx context.Context, key string, usedAt time.Time) (Customer, error) {
	repository.usedAt = usedAt
	if repository.err != nil {
		return Customer{}, repository.err
	}
	if key != repository.expectedKey {
		return Customer{}, ErrInvalidLegacyKey
	}
	return repository.customer, nil
}

func TestLegacyKeyAuthenticatorAcceptsValidCustomerKey(t *testing.T) {
	repository := &fakeLegacyKeyRepository{
		expectedKey: "legacy-api-key",
		customer:    Customer{ID: 42, Name: "demo"},
	}
	authenticator := NewLegacyKeyAuthenticator(
		repository,
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)
	fixedTime := time.Date(2026, 8, 25, 10, 30, 0, 0, time.UTC)
	authenticator.clock = func() time.Time { return fixedTime }
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/testcycle/1", nil)
	request.Header.Set(IdeliumKeyHeader, "legacy-api-key")

	authenticator.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		customer, ok := CustomerFromContext(request.Context())
		if !ok || customer.ID != 42 {
			t.Fatalf("authenticated customer missing from context: %#v", customer)
		}
		tenant, ok := TenantFromContext(request.Context())
		if !ok || tenant.CustomerID != 42 {
			t.Fatalf("tenant context missing from request: %#v", tenant)
		}
		writer.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", response.Code, response.Body.String())
	}
	if !repository.usedAt.Equal(fixedTime) {
		t.Fatalf("expected used timestamp to be passed through, got %s", repository.usedAt)
	}
}

func TestLegacyKeyAuthenticatorRejectsMissingKeyWithLaravelCompatibleBody(t *testing.T) {
	response, logs := exerciseRejectedLegacyKey(t, "", nil)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid key"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	if strings.Contains(logs, "legacy-api-key") || strings.Contains(logs, "Idelium-Key") {
		t.Fatalf("authentication diagnostics exposed key material: %s", logs)
	}
}

func TestLegacyKeyAuthenticatorRejectsInvalidKeyWithRedactedDiagnostics(t *testing.T) {
	response, logs := exerciseRejectedLegacyKey(t, "legacy-api-key", ErrInvalidLegacyKey)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid key"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	if strings.Contains(logs, "legacy-api-key") || strings.Contains(logs, "Idelium-Key") {
		t.Fatalf("authentication diagnostics exposed key material: %s", logs)
	}
	if !strings.Contains(logs, "invalid_or_expired") {
		t.Fatalf("safe rejection reason missing from logs: %s", logs)
	}
}

func TestLegacyKeyAuthenticatorRejectsRepositoryFailureWithoutLeakingCause(t *testing.T) {
	response, logs := exerciseRejectedLegacyKey(
		t,
		"legacy-api-key",
		errors.New("database failure near secret legacy-api-key"),
	)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if strings.Contains(logs, "secret") || strings.Contains(logs, "legacy-api-key") {
		t.Fatalf("repository error leaked into diagnostics: %s", logs)
	}
}

func exerciseRejectedLegacyKey(t *testing.T, key string, repositoryErr error) (*httptest.ResponseRecorder, string) {
	t.Helper()

	logBuffer := &bytes.Buffer{}
	repository := &fakeLegacyKeyRepository{
		expectedKey: "different-key",
		err:         repositoryErr,
	}
	authenticator := NewLegacyKeyAuthenticator(
		repository,
		slog.New(slog.NewTextHandler(logBuffer, nil)),
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/testcycle/1", nil)
	if key != "" {
		request.Header.Set(IdeliumKeyHeader, key)
	}

	authenticator.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("protected handler was called for an unauthenticated request")
	})).ServeHTTP(response, request)

	return response, logBuffer.String()
}
