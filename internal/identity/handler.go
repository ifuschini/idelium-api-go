// Package identity contains fail-closed gates for late-wave identity migration.
package identity

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes advanced identity routes only after migration gates enable a
// Go-native implementation. Until then it fails closed with safe diagnostics.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates an advanced identity migration gate.
func NewHandler(logger *slog.Logger) Handler {
	return Handler{logger: logger}
}

// Providers blocks identity provider reads and writes until Go-native identity
// cutover has passed compatibility, tenant-isolation, and rollback gates.
func (handler Handler) Providers(writer http.ResponseWriter, request *http.Request) {
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
		http.StatusNotImplemented,
		"IDENTITY_MIGRATION_DISABLED",
		"Advanced identity migration is not enabled for the Go runtime.",
	)
}

func hasPathParam(request *http.Request, name string) bool {
	return chi.URLParam(request, name) != ""
}
