package browserauth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
)

const maxMatrixRuns = 64

var (
	ErrParallelRunTerminal    = errors.New("parallel run is terminal")
	ErrParallelRunCancelling  = errors.New("parallel run is cancelling")
	ErrParallelRunConcurrency = errors.New("parallel run concurrency limit reached")
	ErrRunTokenInvalid        = errors.New("run token is invalid")
	ErrAgentProofInvalid      = errors.New("agent identity proof is invalid")
	ErrAgentUnavailable       = errors.New("agent is unavailable")
	ErrParallelWorkerMissing  = errors.New("parallel run worker has not claimed")
)

type AgentUnavailableError struct {
	Status string
	Health string
}

func (e *AgentUnavailableError) Error() string { return ErrAgentUnavailable.Error() }
func (e *AgentUnavailableError) Is(target error) bool {
	return target == ErrAgentUnavailable
}

type ParallelRun struct {
	ID                   int64            `json:"id"`
	RunURL               string           `json:"runUrl"`
	IDProject            int64            `json:"idProject"`
	TestCycleID          int64            `json:"testCycleId"`
	PerformedTestCycleID *int64           `json:"performedTestCycleId"`
	IdempotencyKey       string           `json:"idempotencyKey"`
	Status               string           `json:"status"`
	RequestedConcurrency int              `json:"requestedConcurrency"`
	ActiveWorkers        int              `json:"activeWorkers"`
	TotalWorkers         int              `json:"totalWorkers"`
	CompletedWorkers     int              `json:"completedWorkers"`
	FailedWorkers        int              `json:"failedWorkers"`
	CancelledWorkers     int              `json:"cancelledWorkers"`
	LostWorkers          int              `json:"lostWorkers"`
	AggregateStatus      *int             `json:"aggregateStatus"`
	Metadata             map[string]any   `json:"metadata"`
	ResultSummary        any              `json:"resultSummary"`
	Workers              []map[string]any `json:"workers,omitempty"`
	ScheduledAt          *time.Time       `json:"scheduledAt"`
	StartedAt            *time.Time       `json:"startedAt"`
	CompletedAt          *time.Time       `json:"completedAt"`
	CancelledAt          *time.Time       `json:"cancelledAt"`
}

// Results returns the Laravel-compatible execution summary for a run.
func (h *Handler) Results(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.results(writer, request, user.ActiveTenant())
}

// CLResults returns the execution summary for an API-key authenticated run.
func (h *Handler) CLResults(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.results(writer, request, tenant.CustomerID)
}

// UpdateParallelRunWorker updates a worker from the browser session surface.
func (h *Handler) UpdateParallelRunWorker(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.updateWorker(writer, request, user.ActiveTenant())
}

// CLUpdateParallelRunWorker updates a worker from the API-key surface.
func (h *Handler) CLUpdateParallelRunWorker(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.updateWorker(writer, request, tenant.CustomerID)
}

func (h *Handler) updateWorker(writer http.ResponseWriter, request *http.Request, tenantID int64) {
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	workerID := strings.TrimSpace(request.PathValue("workerId"))
	if workerID == "" || len(workerID) > 128 {
		h.notFound(writer)
		return
	}
	var body struct {
		Status string `json:"status"`
		Result any    `json:"result"`
	}
	if err := decodeJSON(writer, request, &body); err != nil || !validWorkerStatus(body.Status) {
		validationError(writer, "status", "The worker status is invalid.")
		return
	}
	repository, ok := h.sessions.(RunnerRepository)
	if !ok {
		h.internalError(writer, request, "update parallel run worker", errors.New("runner repository unavailable"))
		return
	}
	run, err := repository.UpdateRunnerWorker(request, RunnerWorkerUpdate{TenantID: tenantID, ProjectID: projectID, RunID: runID, WorkerID: workerID, Status: body.Status, Result: body.Result, Now: h.now().UTC()})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrParallelWorkerMissing):
		h.notFound(writer)
	case errors.Is(err, ErrParallelRunTerminal):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Parallel run is already terminal."})
	case errors.Is(err, ErrParallelRunCancelling):
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "Parallel run is cancelling."})
	case err != nil:
		h.internalError(writer, request, "update parallel run worker", err)
	default:
		writeJSON(writer, http.StatusOK, run)
	}
}

func (h *Handler) results(writer http.ResponseWriter, request *http.Request, tenantID int64) {
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
		h.internalError(writer, request, "show parallel run results", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"id": run.ID, "idProject": run.IDProject, "testCycleId": run.TestCycleID, "status": run.Status, "aggregateStatus": run.AggregateStatus, "resultSummary": run.ResultSummary, "workers": run.Workers})
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

type ParallelRunClaim struct {
	TenantID        int64
	ActorUserID     *int64
	ActorTenantID   int64
	ProjectID       int64
	RunID           int64
	WorkerID        string
	Capabilities    []any
	CapabilitiesSet bool
	RunToken        string
	CertificateHash string
	Now             time.Time
}
type RunnerWorkerUpdate struct {
	TenantID, ProjectID, RunID int64
	WorkerID, Status           string
	Result                     any
	Now                        time.Time
}
type RunnerRepository interface {
	UpdateRunnerWorker(*http.Request, RunnerWorkerUpdate) (ParallelRun, error)
}

type ParallelRunHeartbeat struct {
	Run          ParallelRun
	WorkerStatus string
}

type RunTokenIssued struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	AgentID   string `json:"agentId"`
}

type RunTokenRevoked struct {
	TokenID   string `json:"tokenId"`
	RevokedAt string `json:"revokedAt"`
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

func (h *Handler) ClaimParallelRun(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.claimParallelRun(writer, request, user.ActiveTenant(), &user)
}

func (h *Handler) CLClaimParallelRun(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.claimParallelRun(writer, request, tenant.CustomerID, nil)
}

func (h *Handler) HeartbeatParallelRunWorker(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.heartbeatParallelRunWorker(writer, request, user.ActiveTenant())
}

func (h *Handler) CLHeartbeatParallelRunWorker(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.heartbeatParallelRunWorker(writer, request, tenant.CustomerID)
}

func (h *Handler) CancelParallelRun(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	h.cancelParallelRun(writer, request, user.ActiveTenant())
}

func (h *Handler) CLCancelParallelRun(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	h.cancelParallelRun(writer, request, tenant.CustomerID)
}

func (h *Handler) CLIssueParallelRunToken(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	var body struct {
		AgentID string `json:"agentId"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		validationError(writer, "agentId", "The agent id field is invalid.")
		return
	}
	body.AgentID = strings.TrimSpace(body.AgentID)
	if body.AgentID == "" || len(body.AgentID) > 128 {
		validationError(writer, "agentId", "The agent id field is invalid.")
		return
	}
	issued, err := h.sessions.IssueParallelRunToken(request, tenant.CustomerID, projectID, runID, body.AgentID, h.now().UTC(), h.runTokenTTL)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "issue parallel run token", err)
		return
	}
	writeJSON(writer, http.StatusCreated, issued)
}

func (h *Handler) CLRevokeParallelRunToken(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	tokenID := request.PathValue("tokenId")
	if tokenID == "" || len(tokenID) > 64 {
		h.notFound(writer)
		return
	}
	revoked, err := h.sessions.RevokeParallelRunToken(request, tenant.CustomerID, projectID, runID, tokenID, h.now().UTC())
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "revoke parallel run token", err)
		return
	}
	writeJSON(writer, http.StatusOK, revoked)
}

// RunnerClaim accepts the runner-only token contract, whose identifiers are in the JSON body.
func (h *Handler) RunnerClaim(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	var body struct {
		ProjectID    int64  `json:"idProject"`
		RunID        int64  `json:"parallelRun"`
		WorkerID     string `json:"workerId"`
		Capabilities []any  `json:"capabilities"`
	}
	if err := decodeJSON(writer, request, &body); err != nil || body.ProjectID < 1 || body.RunID < 1 || strings.TrimSpace(body.WorkerID) == "" || len(body.WorkerID) > 128 {
		validationError(writer, "workerId", "The worker id field is invalid.")
		return
	}
	in := ParallelRunClaim{TenantID: tenant.CustomerID, ProjectID: body.ProjectID, RunID: body.RunID, WorkerID: strings.TrimSpace(body.WorkerID), Capabilities: body.Capabilities, CapabilitiesSet: body.Capabilities != nil, RunToken: request.Header.Get("Idelium-Run-Token"), CertificateHash: request.Header.Get("Idelium-Agent-Cert-Sha256"), Now: h.now().UTC()}
	if h.runTokenRequiredClaim && in.RunToken == "" {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "A short-lived run token is required to claim a worker slot."})
		return
	}
	run, err := h.sessions.ClaimParallelRun(request, in)
	if errors.Is(err, ErrRunTokenInvalid) {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "The run token is invalid, expired, used, or not bound to this agent."})
		return
	}
	if err != nil {
		h.internalError(writer, request, "runner claim", err)
		return
	}
	writeJSON(writer, http.StatusOK, run)
}

// RunnerHeartbeat accepts the runner-only token contract, whose identifiers are in the JSON body.
func (h *Handler) RunnerHeartbeat(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	var body struct {
		ProjectID    int64  `json:"idProject"`
		RunID        int64  `json:"parallelRun"`
		WorkerID     string `json:"workerId"`
		LeaseSeconds int    `json:"leaseSeconds"`
	}
	if err := decodeJSON(writer, request, &body); err != nil || body.ProjectID < 1 || body.RunID < 1 || strings.TrimSpace(body.WorkerID) == "" {
		validationError(writer, "workerId", "The worker id field is invalid.")
		return
	}
	if body.LeaseSeconds == 0 {
		body.LeaseSeconds = 120
	}
	if body.LeaseSeconds < 15 || body.LeaseSeconds > 3600 {
		validationError(writer, "leaseSeconds", "The lease seconds field must be between 15 and 3600.")
		return
	}
	result, err := h.sessions.HeartbeatParallelRunWorker(request, tenant.CustomerID, body.ProjectID, body.RunID, strings.TrimSpace(body.WorkerID), body.LeaseSeconds, h.now().UTC())
	if err != nil {
		h.internalError(writer, request, "runner heartbeat", err)
		return
	}
	if result.WorkerStatus != "" {
		p := parallelRunPayload(result.Run)
		p["message"] = "Worker lease is no longer active."
		p["workerStatus"] = result.WorkerStatus
		writeJSON(writer, http.StatusConflict, p)
		return
	}
	writeJSON(writer, http.StatusOK, result.Run)
}

func (h *Handler) RunnerUpdateWorker(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	var body struct {
		ProjectID int64  `json:"idProject"`
		RunID     int64  `json:"parallelRun"`
		WorkerID  string `json:"workerId"`
		Status    string `json:"status"`
		Result    any    `json:"result"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.ProjectID < 1 || body.RunID < 1 || strings.TrimSpace(body.WorkerID) == "" || !validWorkerStatus(body.Status) {
		validationError(writer, "status", "The worker status is invalid.")
		return
	}
	repo, ok := h.sessions.(RunnerRepository)
	if !ok {
		h.internalError(writer, request, "runner worker update", errors.New("runner repository unavailable"))
		return
	}
	run, err := repo.UpdateRunnerWorker(request, RunnerWorkerUpdate{TenantID: tenant.CustomerID, ProjectID: body.ProjectID, RunID: body.RunID, WorkerID: strings.TrimSpace(body.WorkerID), Status: body.Status, Result: body.Result, Now: h.now().UTC()})
	if err != nil {
		h.internalError(writer, request, "runner worker update", err)
		return
	}
	writeJSON(writer, http.StatusOK, run)
}
func validWorkerStatus(s string) bool {
	return s == "running" || s == "completed" || s == "failed" || s == "cancelled" || s == "lost"
}

func (h *Handler) cancelParallelRun(writer http.ResponseWriter, request *http.Request, tenantID int64) {
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	run, err := h.sessions.CancelParallelRun(request, tenantID, projectID, runID, h.now().UTC())
	switch {
	case errors.Is(err, ErrNotFound):
		h.notFound(writer)
	case errors.Is(err, ErrParallelRunTerminal):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Parallel run is already terminal."})
	case err != nil:
		h.internalError(writer, request, "cancel parallel run", err)
	default:
		writeJSON(writer, http.StatusOK, run)
	}
}

func (h *Handler) heartbeatParallelRunWorker(writer http.ResponseWriter, request *http.Request, tenantID int64) {
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	workerID := request.PathValue("workerId")
	if workerID == "" || len(workerID) > 128 {
		h.notFound(writer)
		return
	}
	var body struct {
		LeaseSeconds *int `json:"leaseSeconds"`
	}
	if err := decodeJSON(writer, request, &body); err != nil && !errors.Is(err, io.EOF) {
		validationError(writer, "leaseSeconds", "The lease seconds field must be an integer.")
		return
	}
	leaseSeconds := 120
	if body.LeaseSeconds != nil {
		leaseSeconds = *body.LeaseSeconds
	}
	if leaseSeconds < 15 || leaseSeconds > 3600 {
		validationError(writer, "leaseSeconds", "The lease seconds field must be between 15 and 3600.")
		return
	}
	heartbeat, err := h.sessions.HeartbeatParallelRunWorker(request, tenantID, projectID, runID, workerID, leaseSeconds, h.now().UTC())
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrParallelWorkerMissing):
		if errors.Is(err, ErrParallelWorkerMissing) {
			writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Worker has not claimed this run."})
		} else {
			h.notFound(writer)
		}
	case errors.Is(err, ErrParallelRunTerminal):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Parallel run is already terminal."})
	case err != nil:
		h.internalError(writer, request, "heartbeat parallel run worker", err)
	case heartbeat.WorkerStatus != "":
		payload := parallelRunPayload(heartbeat.Run)
		payload["message"] = "Worker lease is no longer active."
		payload["workerStatus"] = heartbeat.WorkerStatus
		writeJSON(writer, http.StatusConflict, payload)
	default:
		writeJSON(writer, http.StatusOK, heartbeat.Run)
	}
}

func parallelRunPayload(run ParallelRun) map[string]any {
	encoded, _ := json.Marshal(run)
	payload := map[string]any{}
	_ = json.Unmarshal(encoded, &payload)
	return payload
}

func (h *Handler) claimParallelRun(writer http.ResponseWriter, request *http.Request, tenantID int64, actor *User) {
	projectID, runID, ok := parseAssetVersionPath(writer, request, "parallelRun")
	if !ok {
		return
	}
	var body struct {
		WorkerID     string          `json:"workerId"`
		Capabilities json.RawMessage `json:"capabilities"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	body.WorkerID = strings.TrimSpace(body.WorkerID)
	if body.WorkerID == "" || len(body.WorkerID) > 128 {
		validationError(writer, "workerId", "The worker id field is invalid.")
		return
	}
	runToken := request.Header.Get("Idelium-Run-Token")
	if h.runTokenRequiredClaim && runToken == "" {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "A short-lived run token is required to claim a worker slot."})
		return
	}
	input := ParallelRunClaim{
		TenantID: tenantID, ProjectID: projectID, RunID: runID, WorkerID: body.WorkerID,
		RunToken: runToken, CertificateHash: request.Header.Get("Idelium-Agent-Cert-Sha256"), Now: h.now().UTC(),
	}
	input.ActorTenantID = tenantID
	if actor != nil {
		input.ActorUserID = &actor.ID
		input.ActorTenantID = actor.TenantID
	}
	if len(body.Capabilities) > 0 {
		if json.Unmarshal(body.Capabilities, &input.Capabilities) != nil || input.Capabilities == nil {
			validationError(writer, "capabilities", "The capabilities field must be an array.")
			return
		}
		input.CapabilitiesSet = true
	}
	run, err := h.sessions.ClaimParallelRun(request, input)
	switch {
	case errors.Is(err, ErrNotFound):
		h.notFound(writer)
	case errors.Is(err, ErrRunTokenInvalid):
		validationError(writer, "runToken", "The run token is invalid, expired, used, or not bound to this agent.")
	case errors.Is(err, ErrAgentProofInvalid):
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "Agent identity proof is invalid for this run ownership request."})
	case errors.Is(err, ErrAgentUnavailable):
		payload := map[string]string{"message": "Agent is not approved and healthy for new run ownership."}
		var unavailable *AgentUnavailableError
		if errors.As(err, &unavailable) {
			payload["agentStatus"] = unavailable.Status
			payload["agentHealth"] = unavailable.Health
		}
		writeJSON(writer, http.StatusConflict, payload)
	case errors.Is(err, ErrParallelRunConcurrency):
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "Concurrency limit reached."})
	case errors.Is(err, ErrParallelRunCancelling):
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "Parallel run is cancelling."})
	case errors.Is(err, ErrParallelRunTerminal):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Parallel run is already terminal."})
	case err != nil:
		h.internalError(writer, request, "claim parallel run worker", err)
	default:
		writeJSON(writer, http.StatusOK, run)
	}
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
