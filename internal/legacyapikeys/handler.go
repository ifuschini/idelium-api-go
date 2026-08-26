// Package legacyapikeys contains fail-closed gates for browser-managed legacy
// API-key lifecycle routes.
package legacyapikeys

import (
	"log/slog"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes legacy API-key lifecycle routes only after Go-native browser
// authentication and tenant ownership gates have been completed.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a legacy API-key lifecycle migration gate.
func NewHandler(logger *slog.Logger) Handler {
	return Handler{logger: logger}
}

// Show blocks legacy API-key reads until Go owns browser-session
// authentication and tenant-scoped customer administration.
func (handler Handler) Show(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "legacy-api-key-show")
}

// Replace blocks legacy API-key replacement until Go owns key generation,
// expiration policy, audit logging, and tenant-scoped writes.
func (handler Handler) Replace(writer http.ResponseWriter, request *http.Request) {
	handler.writeMigrationDisabled(writer, request, "legacy-api-key-replace")
}

func (handler Handler) writeMigrationDisabled(
	writer http.ResponseWriter,
	request *http.Request,
	surface string,
) {
	if handler.logger != nil {
		handler.logger.Info(
			"Legacy API-key lifecycle route rejected before Go-native cutover",
			"surface", surface,
			"correlation_id", httpx.GetCorrelationID(request.Context()),
		)
	}
	httpx.WriteError(
		writer,
		request,
		http.StatusNotImplemented,
		"LEGACY_API_KEY_MIGRATION_DISABLED",
		"Legacy API-key lifecycle migration is not enabled for the Go runtime.",
	)
}
