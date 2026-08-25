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
	testCycles   TestCycleRepository
	tests        TestRepository
	steps        StepRepository
	plugins      PluginRepository
	environments EnvironmentRepository
	logger       *slog.Logger
}

// NewHandler creates a CLI API handler.
func NewHandler(testCycles TestCycleRepository, tests TestRepository, steps StepRepository, plugins PluginRepository, environments EnvironmentRepository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{testCycles: testCycles, tests: tests, steps: steps, plugins: plugins, environments: environments, logger: logger}
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

// Step returns one tenant-owned step using the Laravel CLI contract.
func (handler *Handler) Step(writer http.ResponseWriter, request *http.Request) {
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

	stepID, ok := parseLegacyID(chi.URLParam(request, "idStep"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	step, err := handler.steps.GetStep(request.Context(), tenant.CustomerID, stepID)
	if errors.Is(err, ErrNotFound) {
		writeInvalidID(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI step read failed",
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

	httpx.WriteJSON(writer, http.StatusOK, step)
}

// Plugins returns tenant-owned plugins for a project using the Laravel CLI contract.
func (handler *Handler) Plugins(writer http.ResponseWriter, request *http.Request) {
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

	projectID, ok := parseLegacyID(chi.URLParam(request, "idProject"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	plugins, err := handler.plugins.ListPlugins(request.Context(), tenant.CustomerID, projectID)
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI plugin-list read failed",
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

	httpx.WriteJSON(writer, http.StatusOK, plugins)
}

// Plugin returns one tenant-owned plugin using the Laravel CLI contract.
func (handler *Handler) Plugin(writer http.ResponseWriter, request *http.Request) {
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

	pluginID, ok := parseLegacyID(chi.URLParam(request, "idPlugin"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	plugin, err := handler.plugins.GetPlugin(request.Context(), tenant.CustomerID, pluginID)
	if errors.Is(err, ErrNotFound) {
		writeInvalidID(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI plugin read failed",
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

	httpx.WriteJSON(writer, http.StatusOK, plugin)
}

// Environments returns tenant-owned environments for a project using the Laravel CLI contract.
func (handler *Handler) Environments(writer http.ResponseWriter, request *http.Request) {
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

	projectID, ok := parseLegacyID(chi.URLParam(request, "idProject"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	environments, err := handler.environments.ListEnvironments(request.Context(), tenant.CustomerID, projectID)
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI environment-list read failed",
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

	httpx.WriteJSON(writer, http.StatusOK, environments)
}

// Environment returns one tenant-owned environment using the Laravel CLI contract.
func (handler *Handler) Environment(writer http.ResponseWriter, request *http.Request) {
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

	environmentID, ok := parseLegacyID(chi.URLParam(request, "idEnvironment"))
	if !ok {
		writeInvalidID(writer)
		return
	}

	environment, err := handler.environments.GetEnvironment(request.Context(), tenant.CustomerID, environmentID)
	if errors.Is(err, ErrNotFound) {
		writeInvalidID(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI environment read failed",
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

	httpx.WriteJSON(writer, http.StatusOK, environment)
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
