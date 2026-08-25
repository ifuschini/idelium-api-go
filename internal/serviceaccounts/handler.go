// Package serviceaccounts contains fail-closed gates for service-account and
// scoped-credential migration.
package serviceaccounts

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes service-account routes only after Go-native credential
// lifecycle migration passes compatibility and tenant-isolation gates.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a service-account migration gate.
func NewHandler(logger *slog.Logger) Handler {
	return Handler{logger: logger}
}

// Index blocks service-account listing until Go owns browser-session
// authorization and tenant-scoped credential retrieval.
func (handler Handler) Index(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "service-account-index")
}

// Store blocks service-account credential creation until Go owns credential
// hashing, scoped grants, expiration, and one-time secret disclosure.
func (handler Handler) Store(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "service-account-store")
}

// Revoke blocks service-account credential revocation until Go owns credential
// lifecycle writes and audit events.
func (handler Handler) Revoke(writer http.ResponseWriter, request *http.Request) {
	if chi.URLParam(request, "serviceAccount") == "" {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_SERVICE_ACCOUNT",
			"The service account identifier is required.",
		)
		return
	}
	handler.writeMigrationDisabled(writer, request, "service-account-revoke")
}

func (handler Handler) writeMigrationDisabled(
	writer http.ResponseWriter,
	request *http.Request,
	surface string,
) {
	if handler.logger != nil {
		handler.logger.Info(
			"Service account route rejected before Go-native cutover",
			"surface", surface,
			"correlation_id", httpx.GetCorrelationID(request.Context()),
		)
	}
	httpx.WriteError(
		writer,
		request,
		http.StatusNotImplemented,
		"SERVICE_ACCOUNT_MIGRATION_DISABLED",
		"Service account credential migration is not enabled for the Go runtime.",
	)
}
