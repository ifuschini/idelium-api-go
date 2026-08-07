package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/health"
	"github.com/idelium/idelium-api-go/internal/httpx"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

// NewRouter builds the API router and common middleware chain.
func NewRouter(
	logger *slog.Logger,
	checker health.Checker,
	info buildinfo.Info,
	catalogRepository platforms.CatalogRepository,
) http.Handler {
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(httpx.SecureHeaders)
	router.Use(httpx.AccessLogger(logger))
	router.Use(httpx.Recoverer(logger))

	healthHandler := health.NewHandler(checker, info)
	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)

	platformHandler := platforms.NewHandler(catalogRepository, logger)
	router.Get("/admin/platforms/types", platformHandler.Types)
	router.Get("/admin/platforms/status", platformHandler.Statuses)

	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested route does not exist.")
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		httpx.WriteError(writer, request, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The HTTP method is not supported for this route.")
	})

	return router
}
