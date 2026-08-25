package cliapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes Idelium CLI configuration read endpoints.
type Handler struct {
	testCycles TestCycleRepository
	tests      TestRepository
	logger     *slog.Logger
}

// NewHandler creates a CLI API handler.
func NewHandler(testCycles TestCycleRepository, tests TestRepository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{testCycles: testCycles, tests: tests, logger: logger}
}

// TestCycle returns one tenant-owned test cycle using the Laravel CLI contract.
func (handler *Handler) TestCycle(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI tenant context missing",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"CLI_TENANT_CONTEXT_MISSING",
			"The CLI tenant context could not be resolved.",
		)
		return
	}

	testCycleID, ok := parseLegacyID(chi.URLParam(request, "idTestCycle"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	testCycle, err := handler.testCycles.GetTestCycle(request.Context(), tenant.CustomerID, testCycleID)
	if errors.Is(err, ErrNotFound) {
		writeInvalidID(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI test-cycle read failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"CLI_CONFIGURATION_UNAVAILABLE",
			"The CLI configuration could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, testCycle)
}

// Test returns one tenant-owned test using the Laravel CLI contract.
func (handler *Handler) Test(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI tenant context missing",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"CLI_TENANT_CONTEXT_MISSING",
			"The CLI tenant context could not be resolved.",
		)
		return
	}

	testID, ok := parseLegacyID(chi.URLParam(request, "idTest"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	test, err := handler.tests.GetTest(request.Context(), tenant.CustomerID, testID)
	if errors.Is(err, ErrNotFound) {
		writeInvalidID(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI test read failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"CLI_CONFIGURATION_UNAVAILABLE",
			"The CLI configuration could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, test)
}

func parseLegacyID(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

func writeInvalidID(writer http.ResponseWriter) {
	httpx.WriteJSON(writer, http.StatusNotFound, map[string]string{"message": "Invalid id"})
}
