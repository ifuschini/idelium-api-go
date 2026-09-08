// Package identity contains fail-closed gates for late-wave identity migration.
package identity

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes advanced identity routes only after migration gates enable a
// Go-native implementation. Until then it fails closed with safe diagnostics.
type Handler struct {
	logger    *slog.Logger
	sessions  browserauth.SessionRepository
	providers ProviderRepository
}

type Provider struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer,omitempty"`
	Audience string `json:"audience,omitempty"`
	Status   string `json:"status"`
}
type ProviderRepository interface {
	ListProviders(context.Context, int64) ([]Provider, error)
	CreateProvider(context.Context, int64, Provider) (Provider, error)
}

// NewHandler creates an advanced identity migration gate.
func NewHandler(logger *slog.Logger, deps ...any) Handler {
	h := Handler{logger: logger}
	if len(deps) > 0 {
		h.sessions, _ = deps[0].(browserauth.SessionRepository)
	}
	if len(deps) > 1 {
		h.providers, _ = deps[1].(ProviderRepository)
	}
	return h
}

// Providers blocks identity provider reads and writes until Go-native identity
// cutover has passed compatibility, tenant-isolation, and rollback gates.
func (handler Handler) Providers(writer http.ResponseWriter, request *http.Request) {
	if handler.sessions != nil && handler.providers != nil {
		user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
		if !ok {
			httpx.WriteError(writer, request, http.StatusUnauthorized, "UNAUTHENTICATED", "An active browser session is required.")
			return
		}
		if request.Method == http.MethodGet {
			v, e := handler.providers.ListProviders(request.Context(), user.ActiveTenant())
			if e != nil {
				httpx.WriteError(writer, request, 500, "IDENTITY_UNAVAILABLE", "Identity providers could not be loaded.")
				return
			}
			httpx.WriteJSON(writer, 200, v)
			return
		}
		var in Provider
		if json.NewDecoder(http.MaxBytesReader(writer, request.Body, 64<<10)).Decode(&in) != nil || strings.TrimSpace(in.Type) == "" || strings.TrimSpace(in.Name) == "" {
			httpx.WriteError(writer, request, 400, "INVALID_IDENTITY_PROVIDER", "Provider type and name are required.")
			return
		}
		in.Type = strings.ToLower(strings.TrimSpace(in.Type))
		in.Name = strings.TrimSpace(in.Name)
		created, e := handler.providers.CreateProvider(request.Context(), user.ActiveTenant(), in)
		if e != nil {
			httpx.WriteError(writer, request, 500, "IDENTITY_UNAVAILABLE", "Identity provider could not be created.")
			return
		}
		httpx.WriteJSON(writer, 201, created)
		return
	}
	handler.writeMigrationDisabled(writer, request, "identity-providers")
}

// BreakGlass blocks break-glass account updates until the Go-native
// implementation owns browser authentication and administration.
func (handler Handler) BreakGlass(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "user") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_USER", "The user identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "break-glass")
}

// BreakGlassTest blocks break-glass verification writes until cutover.
func (handler Handler) BreakGlassTest(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "user") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_USER", "The user identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "break-glass-test")
}

// SCIMUsers blocks SCIM lifecycle writes until Go owns the identity provider.
func (handler Handler) SCIMUsers(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "scim-users")
}

// MFAEnroll blocks MFA enrollment until Go-native browser authentication is enabled.
func (handler Handler) MFAEnroll(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "mfa-enroll")
}

// MFAConfirm blocks MFA confirmation until Go-native browser authentication is enabled.
func (handler Handler) MFAConfirm(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "mfa-confirm")
}

// MFAStepUp blocks MFA step-up until Go-native browser authentication is enabled.
func (handler Handler) MFAStepUp(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "mfa-step-up")
}

// OIDCTokenExchange blocks workload identity exchange until Go-native trust
// validation is enabled.
func (handler Handler) OIDCTokenExchange(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "oidc-token-exchange")
}

// SSOStart blocks SSO bootstrap until Go-native SSO is enabled.
func (handler Handler) SSOStart(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "sso-start")
}

// OIDCCallback blocks OIDC callbacks until Go-native SSO is enabled.
func (handler Handler) OIDCCallback(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "oidc-callback")
}

// SAMLCallback blocks SAML callbacks until Go-native SSO is enabled.
func (handler Handler) SAMLCallback(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "saml-callback")
}

func (handler Handler) writeMigrationDisabled(
	writer http.ResponseWriter,
	request *http.Request,
	surface string,
) {
	if handler.logger != nil {
		handler.logger.Info(
			"Advanced identity route rejected before Go-native cutover",
			"surface", surface,
			"correlation_id", httpx.GetCorrelationID(request.Context()),
		)
	}
	httpx.WriteError(
		writer,
		request,
		http.StatusConflict,
		"IDENTITY_LARAVEL_OWNER",
		"Advanced identity operations remain owned by the Laravel runtime.",
	)
}

func hasPathParam(request *http.Request, name string) bool {
	return chi.URLParam(request, name) != ""
}
