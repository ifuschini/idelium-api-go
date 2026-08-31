package browserauth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
)

const maxMatrixRuns = 64

type ParallelRun struct {
	ID                   int64          `json:"id"`
	RunURL               string         `json:"runUrl"`
	IDProject            int64          `json:"idProject"`
	TestCycleID          int64          `json:"testCycleId"`
	PerformedTestCycleID *int64         `json:"performedTestCycleId"`
	IdempotencyKey       string         `json:"idempotencyKey"`
	Status               string         `json:"status"`
	RequestedConcurrency int            `json:"requestedConcurrency"`
	ActiveWorkers        int            `json:"activeWorkers"`
	TotalWorkers         int            `json:"totalWorkers"`
	CompletedWorkers     int            `json:"completedWorkers"`
	FailedWorkers        int            `json:"failedWorkers"`
	CancelledWorkers     int            `json:"cancelledWorkers"`
	LostWorkers          int            `json:"lostWorkers"`
	AggregateStatus      *int           `json:"aggregateStatus"`
	Metadata             map[string]any `json:"metadata"`
	ResultSummary        map[string]any `json:"resultSummary"`
	ScheduledAt          *time.Time     `json:"scheduledAt"`
	StartedAt            *time.Time     `json:"startedAt"`
	CompletedAt          *time.Time     `json:"completedAt"`
	CancelledAt          *time.Time     `json:"cancelledAt"`
}

type ParallelRunCreate struct {
	TenantID             int64
	ProjectID            int64
	TestCycleID          int64
	IdempotencyKey       string
	RequestedConcurrency int
	Metadata             map[string]any
	Now                  time.Time
}

func (h *Handler) ParallelRuns(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.listParallelRuns(writer, request, user.ActiveTenant())
}

func (h *Handler) CLParallelRuns(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.listParallelRuns(writer, request, tenant.CustomerID)
}

func (h *Handler) CreateParallelRun(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.createParallelRun(writer, request, user.ActiveTenant(), false)
}

func (h *Handler) CLCreateParallelRun(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.createParallelRun(writer, request, tenant.CustomerID, false)
}

func (h *Handler) CreateParallelRunMatrix(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.createParallelRun(writer, request, user.ActiveTenant(), true)
}

func (h *Handler) CLCreateParallelRunMatrix(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.createParallelRun(writer, request, tenant.CustomerID, true)
}

func (h *Handler) ShowParallelRun(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.showParallelRun(writer, request, user.ActiveTenant())
}

func (h *Handler) CLShowParallelRun(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.showParallelRun(writer, request, tenant.CustomerID)
}

func (h *Handler) listParallelRuns(writer http.ResponseWriter, request *http.Request, tenantID int64) {
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	filters := map[string]string{}
	for _, field := range []string{"build", "commit", "branch", "repository", "initiator", "pipeline"} {
		if value := request.URL.Query().Get(field); value != "" {
			filters[field] = value
		}
	}
	runs, err := h.sessions.ListParallelRuns(request, tenantID, projectID, filters)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list parallel runs", err)
		return
	}
	writeJSON(writer, http.StatusOK, runs)
}

func (h *Handler) createParallelRun(writer http.ResponseWriter, request *http.Request, tenantID int64, matrix bool) {
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	var body struct {
		TestCycleID          int64            `json:"testCycleId"`
		IdempotencyKey       string           `json:"idempotencyKey"`
		RequestedConcurrency int              `json:"requestedConcurrency"`
		Metadata             map[string]any   `json:"metadata"`
		Matrix               map[string][]any `json:"matrix"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	keyLimit := 128
	if matrix {
		keyLimit = 96
	}
	if body.TestCycleID < 1 || strings.TrimSpace(body.IdempotencyKey) == "" || len(body.IdempotencyKey) > keyLimit {
		validationErrors(writer, map[string][]string{"testCycleId": {"The test cycle id field is required."}, "idempotencyKey": {"The idempotency key field is invalid."}})
		return
	}
	if body.RequestedConcurrency == 0 {
		body.RequestedConcurrency = 1
	}
	if body.RequestedConcurrency < 1 || body.RequestedConcurrency > 32 {
		validationError(writer, "requestedConcurrency", "The requested concurrency field must be between 1 and 32.")
		return
	}
	input := ParallelRunCreate{TenantID: tenantID, ProjectID: projectID, TestCycleID: body.TestCycleID, IdempotencyKey: strings.TrimSpace(body.IdempotencyKey), RequestedConcurrency: body.RequestedConcurrency, Metadata: normalizeRunMetadata(body.Metadata), Now: h.now().UTC()}
	if !matrix {
		run, err := h.sessions.CreateParallelRun(request, input)
		h.writeParallelRun(writer, request, run, err)
		return
	}
	if body.Matrix == nil {
		validationError(writer, "matrix", "The matrix field must be present.")
		return
	}
	combinations, invalid := matrixCombinations(body.Matrix)
	if invalid {
		validationError(writer, "matrix", "The matrix axes must contain no more than 16 values.")
		return
	}
	if len(combinations) == 0 {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "At least one matrix axis value is required."})
		return
	}
	if len(combinations) > maxMatrixRuns {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "Matrix launch exceeds the maximum number of generated runs.", "maximumRuns": maxMatrixRuns, "requestedRuns": len(combinations)})
		return
	}
	runs, err := h.sessions.CreateParallelRunMatrix(request, input, combinations)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "create parallel run matrix", err)
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]any{"data": runs, "summary": map[string]int{"requestedRuns": len(combinations), "scheduledRuns": len(runs)}})
}

func (h *Handler) showParallelRun(writer http.ResponseWriter, request *http.Request, tenantID int64) {
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	run, err := h.sessions.GetParallelRun(request, tenantID, projectID, runID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "show parallel run", err)
		return
	}
	writeJSON(writer, http.StatusOK, run)
}

func (h *Handler) writeParallelRun(writer http.ResponseWriter, request *http.Request, run ParallelRun, err error) {
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "create parallel run", err)
		return
	}
	writeJSON(writer, http.StatusCreated, run)
}

func matrixCombinations(matrix map[string][]any) ([]map[string]string, bool) {
	combinations := []map[string]string{{}}
	axisCount := 0
	for _, axis := range []struct{ input, output string }{{"platforms", "platform"}, {"browsers", "browser"}, {"devices", "device"}, {"environments", "environment"}} {
		values := matrix[axis.input]
		if len(values) > 16 {
			return nil, true
		}
		unique := []string{}
		seen := map[string]bool{}
		for _, value := range values {
			text, scalar := metadataScalar(value)
			text = strings.TrimSpace(text)
			if scalar && text != "" && !seen[text] {
				seen[text] = true
				unique = append(unique, text)
			}
		}
		if len(unique) == 0 {
			continue
		}
		axisCount++
		next := []map[string]string{}
		for _, combination := range combinations {
			for _, value := range unique {
				copyValue := map[string]string{}
				for key, item := range combination {
					copyValue[key] = item
				}
				copyValue[axis.output] = value
				next = append(next, copyValue)
			}
		}
		combinations = next
	}
	if axisCount == 0 {
		return nil, false
	}
	return combinations, false
}

func normalizeRunMetadata(metadata map[string]any) map[string]any {
	clean := removeSensitiveMetadata(metadata)
	run, _ := clean["run"].(map[string]any)
	if run == nil {
		run = map[string]any{}
	}
	for _, field := range []string{"build", "commit", "branch", "repository", "initiator", "pipeline"} {
		value := run[field]
		if value == nil {
			value = clean[field]
		}
		text, scalar := metadataScalar(value)
		if !scalar || text == "" {
			run[field] = nil
		} else {
			run[field] = text
		}
		delete(clean, field)
	}
	identity, _ := run["workloadIdentity"].(map[string]any)
	if identity == nil {
		identity, _ = clean["workloadIdentity"].(map[string]any)
	}
	normalizedIdentity := map[string]any{}
	for _, field := range []string{"provider", "issuer", "subject", "audience"} {
		text, scalar := metadataScalar(identity[field])
		if identity != nil && scalar && text != "" {
			normalizedIdentity[field] = text
		} else {
			normalizedIdentity[field] = nil
		}
	}
	run["workloadIdentity"] = normalizedIdentity
	clean["run"] = run
	delete(clean, "workloadIdentity")
	return clean
}

func removeSensitiveMetadata(values map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range values {
		lower := strings.ToLower(key)
		sensitive := false
		for _, marker := range []string{"password", "passwd", "secret", "token", "apikey", "api_key", "authorization", "cookie", "credential", "session"} {
			if strings.Contains(lower, marker) {
				sensitive = true
			}
		}
		if sensitive {
			continue
		}
		result[key] = removeSensitiveMetadataValue(value)
	}
	return result
}

func removeSensitiveMetadataValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return removeSensitiveMetadata(typed)
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = removeSensitiveMetadataValue(item)
		}
		return result
	default:
		return value
	}
}

func metadataScalar(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case bool:
		if typed {
			return "1", true
		}
		return "", true
	case nil:
		return "", false
	default:
		return "", false
	}
}
