package platforms

import (
	"log/slog"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes read-only platform catalog endpoints.
type Handler struct {
	repository CatalogRepository
	logger     *slog.Logger
}

// NewHandler creates a platform catalog handler.
func NewHandler(repository CatalogRepository, logger *slog.Logger) *Handler {
	return &Handler{repository: repository, logger: logger}
}

// Types returns the legacy platform type list contract.
func (handler *Handler) Types(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.repository.ListTypes(request.Context())
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform types failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, items)
}

// Statuses returns the legacy platform status list contract.
func (handler *Handler) Statuses(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.repository.ListStatuses(request.Context())
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform statuses failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, items)
}
