package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

const (
	// IdeliumKeyHeader is the legacy CLI customer API-key header used by Laravel.
	IdeliumKeyHeader = "Idelium-Key"

	maxLegacyKeyLength = 4096
)

var (
	// ErrInvalidLegacyKey is returned when no active customer matches the key.
	ErrInvalidLegacyKey = errors.New("invalid legacy customer API key")
)

type customerContextKey struct{}
type tenantContextKey struct{}

// Customer is the authenticated legacy customer identity used by CLI routes.
type Customer struct {
	ID   int64
	Name string
}

// TenantContext carries the tenant ownership boundary attached to a CLI request.
type TenantContext struct {
	CustomerID int64
}

// LegacyKeyRepository verifies legacy customer API keys against persistence.
type LegacyKeyRepository interface {
	AuthenticateLegacyCustomerKey(ctx context.Context, key string, usedAt time.Time) (Customer, error)
}

// LegacyKeyAuthenticator protects CLI routes with the Laravel-compatible key contract.
type LegacyKeyAuthenticator struct {
	repository LegacyKeyRepository
	logger     *slog.Logger
	clock      func() time.Time
}

// NewLegacyKeyAuthenticator creates a redaction-safe legacy key authenticator.
func NewLegacyKeyAuthenticator(repository LegacyKeyRepository, logger *slog.Logger) *LegacyKeyAuthenticator {
	if logger == nil {
		logger = slog.Default()
	}
	return &LegacyKeyAuthenticator{
		repository: repository,
		logger:     logger,
		clock:      func() time.Time { return time.Now().UTC() },
	}
}

// Middleware enforces legacy customer API-key authentication.
func (authenticator *LegacyKeyAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		key, ok := readLegacyKey(request)
		if !ok {
			authenticator.reject(writer, request, "missing_or_malformed")
			return
		}

		customer, err := authenticator.repository.AuthenticateLegacyCustomerKey(
			request.Context(),
			key,
			authenticator.clock(),
		)
		if err != nil {
			authenticator.reject(writer, request, "invalid_or_expired")
			return
		}

		ctx := context.WithValue(request.Context(), customerContextKey{}, customer)
		ctx = context.WithValue(ctx, tenantContextKey{}, TenantContext{CustomerID: customer.ID})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (authenticator *LegacyKeyAuthenticator) reject(writer http.ResponseWriter, request *http.Request, reason string) {
	authenticator.logger.WarnContext(
		request.Context(),
		"legacy API-key authentication rejected",
		"correlation_id", httpx.GetCorrelationID(request.Context()),
		"reason", reason,
		"method", request.Method,
		"path", request.URL.Path,
	)
	httpx.WriteJSON(writer, http.StatusUnauthorized, map[string]string{"message": "Invalid key"})
}

func readLegacyKey(request *http.Request) (string, bool) {
	values := request.Header.Values(IdeliumKeyHeader)
	if len(values) != 1 {
		return "", false
	}
	key := strings.TrimSpace(values[0])
	if key == "" || len(key) > maxLegacyKeyLength || strings.ContainsAny(key, "\r\n") {
		return "", false
	}
	return key, true
}

// CustomerFromContext returns the authenticated customer attached to the request.
func CustomerFromContext(ctx context.Context) (Customer, bool) {
	customer, ok := ctx.Value(customerContextKey{}).(Customer)
	return customer, ok
}

// TenantFromContext returns the tenant boundary attached to the request.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	tenant, ok := ctx.Value(tenantContextKey{}).(TenantContext)
	return tenant, ok
}
