package health

import (
	"context"
	"net/http"
	"time"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

const readinessTimeout = 2 * time.Second

// Checker verifies a required service dependency.
type Checker interface {
	Check(context.Context) error
}

// Handler exposes liveness and readiness endpoints.
type Handler struct {
	checker   Checker
	buildInfo buildinfo.Info
}

// NewHandler creates a health handler.
func NewHandler(checker Checker, info buildinfo.Info) *Handler {
	return &Handler{checker: checker, buildInfo: info}
}

// Live reports process liveness without querying dependencies.
func (handler *Handler) Live(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(writer, http.StatusOK, struct {
		Status string `json:"status"`
		buildinfo.Info
	}{
		Status: "ok",
		Info:   handler.buildInfo,
	})
}

// Ready reports readiness only when the database is reachable.
func (handler *Handler) Ready(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(request.Context(), readinessTimeout)
	defer cancel()

	if err := handler.checker.Check(ctx); err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusServiceUnavailable,
			"DEPENDENCY_UNAVAILABLE",
			"Service dependencies are not ready.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, struct {
		Status       string            `json:"status"`
		Dependencies map[string]string `json:"dependencies"`
		buildinfo.Info
	}{
		Status:       "ok",
		Dependencies: map[string]string{"database": "ok"},
		Info:         handler.buildInfo,
	})
}
