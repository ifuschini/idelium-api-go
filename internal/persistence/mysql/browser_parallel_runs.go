package mysql

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
	"golang.org/x/crypto/bcrypt"
)

type parallelRunScanner interface {
	Scan(...any) error
}

type parallelRunQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *BrowserAuthRepository) ListParallelRuns(request *http.Request, tenantID, projectID int64, filters map[string]string) ([]browserauth.ParallelRun, error) {
	ctx := request.Context()
	if err := r.ensureProject(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	rows, err := r.database.QueryContext(ctx, parallelRunSelect+" WHERE idCostumer = ? AND idProject = ? ORDER BY updated_at DESC LIMIT 50", tenantID, projectID)
	if err != nil {
		return nil, safeDatabaseFailure("list parallel run schedules", err)
	}
	defer rows.Close()
	runs := []browserauth.ParallelRun{}
	for rows.Next() {
		run, err := scanParallelRun(rows)
		if err != nil {
			return nil, err
		}
		if parallelRunMatches(run.Metadata, filters) {
			runs = append(runs, run)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read parallel run schedules", err)
	}
	return runs, nil
}

func (r *BrowserAuthRepository) GetParallelRun(request *http.Request, tenantID, projectID, runID int64) (browserauth.ParallelRun, error) {
	if err := r.ensureProject(request.Context(), tenantID, projectID); err != nil {
		return browserauth.ParallelRun{}, err
	}
	return getParallelRun(request.Context(), r.database, tenantID, projectID, runID)
}

func (r *BrowserAuthRepository) CreateParallelRun(request *http.Request, input browserauth.ParallelRunCreate) (browserauth.ParallelRun, error) {
	runs, err := r.createParallelRuns(request.Context(), input, nil)
	if err != nil {
		return browserauth.ParallelRun{}, err
	}
	return runs[0], nil
}

func (r *BrowserAuthRepository) CreateParallelRunMatrix(request *http.Request, input browserauth.ParallelRunCreate, combinations []map[string]string) ([]browserauth.ParallelRun, error) {
	return r.createParallelRuns(request.Context(), input, combinations)
}

func (r *BrowserAuthRepository) ClaimParallelRun(request *http.Request, input browserauth.ParallelRunClaim) (browserauth.ParallelRun, error) {
	ctx := request.Context()
	if input.RunToken != "" {
		if err := r.consumeParallelRunToken(ctx, input); err != nil {
			if auditErr := r.recordParallelRunTokenAudit(request, input, "run_token.reject", "failure"); auditErr != nil {
				return browserauth.ParallelRun{}, auditErr
			}
			return browserauth.ParallelRun{}, err
		}
		if err := r.recordParallelRunTokenAudit(request, input, "run_token.consume", "success"); err != nil {
			return browserauth.ParallelRun{}, err
		}
	}
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("start parallel run claim transaction", err)
	}
	defer tx.Rollback()
	var status string
	var requestedConcurrency, activeWorkers int
	var workerJSON sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status, requestedConcurrency, activeWorkers, workerStates
		FROM parallel_run_schedules WHERE id = ? AND idCostumer = ? AND idProject = ? FOR UPDATE`,
		input.RunID, input.TenantID, input.ProjectID).Scan(&status, &requestedConcurrency, &activeWorkers, &workerJSON)
	if err != nil {
		return browserauth.ParallelRun{}, parallelRunNotFound("lock parallel run worker claim", err)
	}
	if parallelRunTerminal(status) {
		return browserauth.ParallelRun{}, browserauth.ErrParallelRunTerminal
	}
	if status == "cancelling" {
		return browserauth.ParallelRun{}, browserauth.ErrParallelRunCancelling
	}
	if err := validateParallelRunAgent(ctx, tx, input); err != nil {
		return browserauth.ParallelRun{}, err
	}
	workers := decodeParallelMap(workerJSON)
	existing, _ := workers[input.WorkerID].(map[string]any)
	if existing == nil && activeWorkers >= requestedConcurrency {
		return browserauth.ParallelRun{}, browserauth.ErrParallelRunConcurrency
	}
	capabilities := any([]any{})
	if input.CapabilitiesSet {
		capabilities = input.Capabilities
	} else if existing != nil && existing["capabilities"] != nil {
		capabilities = existing["capabilities"]
	}
	now := parallelRunISOString(input.Now)
	claimedAt := any(now)
	result := any(nil)
	if existing != nil {
		if existing["claimedAt"] != nil {
			claimedAt = existing["claimedAt"]
		}
		result = existing["result"]
	}
	workers[input.WorkerID] = map[string]any{
		"workerId": input.WorkerID, "status": "running", "capabilities": capabilities,
		"claimedAt": claimedAt, "lastHeartbeatAt": now,
		"leaseExpiresAt": parallelRunISOString(input.Now.Add(120 * time.Second)), "updatedAt": now, "result": result,
	}
	counters, summary := recalculateParallelRunWorkers(workers, input.Now)
	workersJSON, err := json.Marshal(workers)
	if err != nil {
		return browserauth.ParallelRun{}, errors.New("parallel run worker state is invalid")
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return browserauth.ParallelRun{}, errors.New("parallel run result summary is invalid")
	}
	_, err = tx.ExecContext(ctx, `UPDATE parallel_run_schedules SET status = 'running', workerStates = ?, resultSummary = ?,
		activeWorkers = ?, totalWorkers = ?, completedWorkers = ?, failedWorkers = ?, cancelledWorkers = ?,
		startedAt = COALESCE(startedAt, ?), updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?`,
		string(workersJSON), string(summaryJSON), counters.active, counters.total, counters.completed, counters.failed, counters.cancelled,
		input.Now, input.Now, input.RunID, input.TenantID, input.ProjectID)
	if err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("update parallel run worker claim", err)
	}
	run, err := getParallelRun(ctx, tx, input.TenantID, input.ProjectID, input.RunID)
	if err != nil {
		return browserauth.ParallelRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("commit parallel run worker claim", err)
	}
	return run, nil
}

func (r *BrowserAuthRepository) HeartbeatParallelRunWorker(request *http.Request, tenantID, projectID, runID int64, workerID string, leaseSeconds int, now time.Time) (browserauth.ParallelRunHeartbeat, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, safeDatabaseFailure("start parallel run heartbeat transaction", err)
	}
	defer tx.Rollback()
	var status string
	var aggregate sql.NullInt64
	var completedAt sql.NullTime
	var workerJSON sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status, aggregateStatus, completedAt, workerStates FROM parallel_run_schedules
		WHERE id = ? AND idCostumer = ? AND idProject = ? FOR UPDATE`, runID, tenantID, projectID).
		Scan(&status, &aggregate, &completedAt, &workerJSON)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, parallelRunNotFound("lock parallel run worker heartbeat", err)
	}
	workers := decodeParallelMap(workerJSON)
	recalculateParallelRunWorkers(workers, now)
	if parallelRunTerminal(status) {
		return browserauth.ParallelRunHeartbeat{}, browserauth.ErrParallelRunTerminal
	}
	worker, exists := workers[workerID].(map[string]any)
	if !exists {
		return browserauth.ParallelRunHeartbeat{}, browserauth.ErrParallelWorkerMissing
	}
	workerStatus, _ := worker["status"].(string)
	if workerStatus == "running" {
		worker["lastHeartbeatAt"] = parallelRunISOString(now)
		worker["leaseExpiresAt"] = parallelRunISOString(now.Add(time.Duration(leaseSeconds) * time.Second))
		worker["updatedAt"] = parallelRunISOString(now)
	}
	counters, summary := recalculateParallelRunWorkers(workers, now)
	nextStatus, nextAggregate, nextCompletedAt := parallelRunWorkerOutcome(status, aggregate, completedAt, counters, now)
	workersJSON, err := json.Marshal(workers)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, errors.New("parallel run worker state is invalid")
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, errors.New("parallel run result summary is invalid")
	}
	_, err = tx.ExecContext(ctx, `UPDATE parallel_run_schedules SET status = ?, aggregateStatus = ?, completedAt = ?,
		workerStates = ?, resultSummary = ?, activeWorkers = ?, totalWorkers = ?, completedWorkers = ?,
		failedWorkers = ?, cancelledWorkers = ?, updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?`,
		nextStatus, nextAggregate, nextCompletedAt, string(workersJSON), string(summaryJSON), counters.active, counters.total,
		counters.completed, counters.failed, counters.cancelled, now, runID, tenantID, projectID)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, safeDatabaseFailure("update parallel run worker heartbeat", err)
	}
	run, err := getParallelRun(ctx, tx, tenantID, projectID, runID)
	if err != nil {
		return browserauth.ParallelRunHeartbeat{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.ParallelRunHeartbeat{}, safeDatabaseFailure("commit parallel run worker heartbeat", err)
	}
	result := browserauth.ParallelRunHeartbeat{Run: run}
	if workerStatus != "running" {
		result.WorkerStatus = workerStatus
	}
	return result, nil
}

func (r *BrowserAuthRepository) CancelParallelRun(request *http.Request, tenantID, projectID, runID int64, now time.Time) (browserauth.ParallelRun, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("start parallel run cancellation transaction", err)
	}
	defer tx.Rollback()
	var status string
	var workerJSON sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status, workerStates FROM parallel_run_schedules
		WHERE id = ? AND idCostumer = ? AND idProject = ? FOR UPDATE`, runID, tenantID, projectID).Scan(&status, &workerJSON)
	if err != nil {
		return browserauth.ParallelRun{}, parallelRunNotFound("lock parallel run cancellation", err)
	}
	if parallelRunTerminal(status) {
		return browserauth.ParallelRun{}, browserauth.ErrParallelRunTerminal
	}
	workers := decodeParallelMap(workerJSON)
	for _, value := range workers {
		worker, _ := value.(map[string]any)
		if worker["status"] == "running" {
			worker["status"] = "cancelled"
			worker["updatedAt"] = parallelRunISOString(now)
		}
	}
	counters, summary := recalculateParallelRunWorkers(workers, now)
	nextStatus, nextAggregate, nextCompletedAt := parallelRunWorkerOutcome("cancelled", sql.NullInt64{Int64: 3, Valid: true}, sql.NullTime{Time: now, Valid: true}, counters, now)
	workersJSON, err := json.Marshal(workers)
	if err != nil {
		return browserauth.ParallelRun{}, errors.New("parallel run worker state is invalid")
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return browserauth.ParallelRun{}, errors.New("parallel run result summary is invalid")
	}
	_, err = tx.ExecContext(ctx, `UPDATE parallel_run_schedules SET status = ?, aggregateStatus = ?, cancelledAt = ?, completedAt = ?,
		workerStates = ?, resultSummary = ?, activeWorkers = ?, totalWorkers = ?, completedWorkers = ?, failedWorkers = ?,
		cancelledWorkers = ?, updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?`,
		nextStatus, nextAggregate, now, nextCompletedAt, string(workersJSON), string(summaryJSON), counters.active, counters.total,
		counters.completed, counters.failed, counters.cancelled, now, runID, tenantID, projectID)
	if err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("update parallel run cancellation", err)
	}
	run, err := getParallelRun(ctx, tx, tenantID, projectID, runID)
	if err != nil {
		return browserauth.ParallelRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("commit parallel run cancellation", err)
	}
	return run, nil
}

func (r *BrowserAuthRepository) IssueParallelRunToken(request *http.Request, tenantID, projectID, runID int64, agentID string, now time.Time, ttl time.Duration) (browserauth.RunTokenIssued, error) {
	ctx := request.Context()
	var scheduleID int64
	if err := r.database.QueryRowContext(ctx, "SELECT id FROM parallel_run_schedules WHERE id = ? AND idCostumer = ? AND idProject = ?", runID, tenantID, projectID).Scan(&scheduleID); err != nil {
		return browserauth.RunTokenIssued{}, parallelRunNotFound("load parallel run for token issue", err)
	}
	tokenIDPart, err := randomParallelRunTokenPart(12)
	if err != nil {
		return browserauth.RunTokenIssued{}, errors.New("run token identifier generation failed")
	}
	secret, err := randomParallelRunTokenPart(32)
	if err != nil {
		return browserauth.RunTokenIssued{}, errors.New("run token secret generation failed")
	}
	tokenID := "idrt_" + tokenIDPart
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return browserauth.RunTokenIssued{}, errors.New("run token hashing failed")
	}
	expiresAt := now.Add(ttl)
	_, err = r.database.ExecContext(ctx, `INSERT INTO run_tokens
		(idCostumer, idProject, parallelRunScheduleId, agentId, tokenId, tokenHash, expiresAt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, tenantID, projectID, scheduleID, agentID, tokenID, string(hash), expiresAt, now, now)
	if err != nil {
		return browserauth.RunTokenIssued{}, safeDatabaseFailure("issue parallel run token", err)
	}
	if err := r.recordParallelRunTokenLifecycleAudit(request, tenantID, projectID, runID, "run_token.issue", nil,
		map[string]any{"agentId": agentID, "tokenId": "[REDACTED]", "token": "[REDACTED]", "expiresAt": parallelRunISOString(expiresAt)}); err != nil {
		return browserauth.RunTokenIssued{}, err
	}
	return browserauth.RunTokenIssued{Token: tokenID + "." + secret, ExpiresAt: parallelRunISOString(expiresAt), AgentID: agentID}, nil
}

func (r *BrowserAuthRepository) RevokeParallelRunToken(request *http.Request, tenantID, projectID, runID int64, tokenID string, now time.Time) (browserauth.RunTokenRevoked, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.RunTokenRevoked{}, safeDatabaseFailure("start run token revocation transaction", err)
	}
	defer tx.Rollback()
	var id int64
	var agentID string
	var revokedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id, agentId, revokedAt FROM run_tokens
		WHERE tokenId = ? AND idCostumer = ? AND idProject = ? AND parallelRunScheduleId = ? FOR UPDATE`, tokenID, tenantID, projectID, runID).
		Scan(&id, &agentID, &revokedAt)
	if err != nil {
		return browserauth.RunTokenRevoked{}, parallelRunNotFound("lock parallel run token revocation", err)
	}
	before := map[string]any{"tokenId": "[REDACTED]", "revokedAt": nil}
	if revokedAt.Valid {
		before["revokedAt"] = parallelRunISOString(revokedAt.Time)
	} else {
		if _, err := tx.ExecContext(ctx, "UPDATE run_tokens SET revokedAt = ?, updated_at = ? WHERE id = ? AND revokedAt IS NULL", now, now, id); err != nil {
			return browserauth.RunTokenRevoked{}, safeDatabaseFailure("revoke parallel run token", err)
		}
		revokedAt = sql.NullTime{Time: now, Valid: true}
	}
	if err := tx.Commit(); err != nil {
		return browserauth.RunTokenRevoked{}, safeDatabaseFailure("commit run token revocation", err)
	}
	after := map[string]any{"agentId": agentID, "tokenId": "[REDACTED]", "revokedAt": parallelRunISOString(revokedAt.Time)}
	if err := r.recordParallelRunTokenLifecycleAudit(request, tenantID, projectID, runID, "run_token.revoke", before, after); err != nil {
		return browserauth.RunTokenRevoked{}, err
	}
	return browserauth.RunTokenRevoked{TokenID: tokenID, RevokedAt: parallelRunISOString(revokedAt.Time)}, nil
}

func (r *BrowserAuthRepository) recordParallelRunTokenLifecycleAudit(request *http.Request, tenantID, projectID, runID int64, action string, before, after map[string]any) error {
	var beforeJSON any
	if before != nil {
		encoded, err := json.Marshal(before)
		if err != nil {
			return errors.New("run token audit values are invalid")
		}
		beforeJSON = string(encoded)
	}
	var afterJSON any
	if after != nil {
		encoded, err := json.Marshal(after)
		if err != nil {
			return errors.New("run token audit values are invalid")
		}
		afterJSON = string(encoded)
	}
	_, err := r.database.ExecContext(request.Context(), `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (NULL, ?, ?, ?, ?, 'parallel_run_schedule', ?, ?, ?, 'success', ?, ?, NULL)`,
		tenantID, tenantID, projectID, action, runID, beforeJSON, afterJSON, sourceIP(request), correlationID(request))
	if err != nil {
		return safeDatabaseFailure("record run token lifecycle audit", err)
	}
	return nil
}

func randomParallelRunTokenPart(bytesCount int) (string, error) {
	value := make([]byte, bytesCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func (r *BrowserAuthRepository) recordParallelRunTokenAudit(request *http.Request, input browserauth.ParallelRunClaim, action, result string) error {
	values, err := json.Marshal(map[string]any{"agentId": input.WorkerID, "tokenId": "[REDACTED]", "token": "[REDACTED]"})
	if err != nil {
		return errors.New("run token audit values are invalid")
	}
	_, err = r.database.ExecContext(request.Context(), `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (?, ?, ?, ?, ?, 'parallel_run_schedule', ?, NULL, ?, ?, ?, ?, NULL)`,
		input.ActorUserID, input.ActorTenantID, input.TenantID, input.ProjectID, action, input.RunID, string(values), result, sourceIP(request), correlationID(request))
	if err != nil {
		return safeDatabaseFailure("record run token claim audit", err)
	}
	return nil
}

func (r *BrowserAuthRepository) consumeParallelRunToken(ctx context.Context, input browserauth.ParallelRunClaim) error {
	tokenID, secret, valid := strings.Cut(input.RunToken, ".")
	if !valid || !strings.HasPrefix(tokenID, "idrt_") || secret == "" {
		return browserauth.ErrRunTokenInvalid
	}
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start run token consumption", err)
	}
	defer tx.Rollback()
	var id int64
	var hash string
	var expiresAt time.Time
	var usedAt, revokedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id, tokenHash, expiresAt, usedAt, revokedAt FROM run_tokens
		WHERE tokenId = ? AND idCostumer = ? AND idProject = ? AND parallelRunScheduleId = ? AND agentId = ? FOR UPDATE`,
		tokenID, input.TenantID, input.ProjectID, input.RunID, input.WorkerID).Scan(&id, &hash, &expiresAt, &usedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ErrRunTokenInvalid
	}
	if err != nil {
		return safeDatabaseFailure("lock run token for claim", err)
	}
	if usedAt.Valid || revokedAt.Valid || !expiresAt.After(input.Now) || bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) != nil {
		return browserauth.ErrRunTokenInvalid
	}
	result, err := tx.ExecContext(ctx, "UPDATE run_tokens SET usedAt = ?, updated_at = ? WHERE id = ? AND usedAt IS NULL AND revokedAt IS NULL", input.Now, input.Now, id)
	if err != nil {
		return safeDatabaseFailure("consume run token for claim", err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return browserauth.ErrRunTokenInvalid
	}
	if err := tx.Commit(); err != nil {
		return safeDatabaseFailure("commit run token consumption", err)
	}
	return nil
}

func validateParallelRunAgent(ctx context.Context, tx *sql.Tx, input browserauth.ParallelRunClaim) error {
	var status, health string
	var identityJSON sql.NullString
	err := tx.QueryRowContext(ctx, "SELECT status, health, identityProof FROM agent_registrations WHERE idCostumer = ? AND agentId = ?", input.TenantID, input.WorkerID).Scan(&status, &health, &identityJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return safeDatabaseFailure("validate parallel run agent", err)
	}
	identity := decodeParallelMap(identityJSON)
	if expected, ok := identity["certificateSha256"].(string); ok && expected != "" {
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(expected)), []byte(strings.ToLower(input.CertificateHash))) != 1 {
			return browserauth.ErrAgentProofInvalid
		}
	}
	if status != "approved" || health == "unhealthy" {
		return &browserauth.AgentUnavailableError{Status: status, Health: health}
	}
	return nil
}

type parallelWorkerCounters struct{ active, total, completed, failed, cancelled, lost int }

func recalculateParallelRunWorkers(workers map[string]any, now time.Time) (parallelWorkerCounters, []map[string]any) {
	ids := make([]string, 0, len(workers))
	for id := range workers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	counters := parallelWorkerCounters{total: len(ids)}
	summary := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		worker, _ := workers[id].(map[string]any)
		status, _ := worker["status"].(string)
		if status == "" {
			status = "running"
		}
		if status == "running" {
			if expiry, ok := worker["leaseExpiresAt"].(string); ok {
				if parsed, err := time.Parse(time.RFC3339Nano, expiry); err == nil && parsed.Before(now) {
					status = "lost"
					worker["status"] = status
					worker["lostAt"] = parallelRunISOString(now)
					worker["updatedAt"] = parallelRunISOString(now)
				}
			}
		}
		switch status {
		case "running":
			counters.active++
		case "completed":
			counters.completed++
		case "failed":
			counters.failed++
		case "cancelled":
			counters.cancelled++
		case "lost":
			counters.lost++
		}
		summary = append(summary, map[string]any{"workerId": id, "status": status, "result": worker["result"]})
	}
	return counters, summary
}

func parallelRunWorkerOutcome(currentStatus string, aggregate sql.NullInt64, completedAt sql.NullTime, counters parallelWorkerCounters, now time.Time) (string, any, any) {
	aggregateValue := any(nil)
	if aggregate.Valid {
		aggregateValue = aggregate.Int64
	}
	completedValue := any(nil)
	if completedAt.Valid {
		completedValue = completedAt.Time
	}
	if counters.total == 0 || counters.active > 0 {
		return currentStatus, aggregateValue, completedValue
	}
	if !completedAt.Valid {
		completedValue = now
	}
	if counters.failed > 0 {
		return "failed", 2, completedValue
	}
	if counters.lost > 0 {
		return "lost", 4, completedValue
	}
	if counters.cancelled > 0 && counters.completed == 0 {
		return "cancelled", 3, completedValue
	}
	if counters.cancelled > 0 {
		return "failed", 3, completedValue
	}
	return "completed", 1, completedValue
}

func parallelRunTerminal(status string) bool {
	return status == "cancelled" || status == "completed" || status == "failed" || status == "lost"
}

func parallelRunISOString(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func (r *BrowserAuthRepository) createParallelRuns(ctx context.Context, input browserauth.ParallelRunCreate, combinations []map[string]string) ([]browserauth.ParallelRun, error) {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, safeDatabaseFailure("start parallel run schedule transaction", err)
	}
	defer tx.Rollback()
	if err := ensureParallelRunProjectAndCycle(ctx, tx, input); err != nil {
		return nil, err
	}
	snapshot, err := parallelRunExecutionSnapshot(ctx, tx, input.TenantID, input.TestCycleID)
	if err != nil {
		return nil, err
	}
	if combinations == nil {
		combinations = []map[string]string{nil}
	}
	runs := make([]browserauth.ParallelRun, 0, len(combinations))
	for index, combination := range combinations {
		metadata := cloneJSONMap(input.Metadata)
		metadata["executionSnapshot"] = snapshot
		idempotencyKey := input.IdempotencyKey
		if combination != nil {
			metadata["matrix"] = map[string]any{"index": index, "total": len(combinations), "combination": combination}
			idempotencyKey = parallelMatrixIdempotencyKey(input.IdempotencyKey, combination)
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return nil, errors.New("parallel run metadata is invalid")
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO parallel_run_schedules
			(idCostumer, idProject, testCycleId, idempotencyKey, status, requestedConcurrency, activeWorkers, totalWorkers, completedWorkers, failedWorkers, cancelledWorkers, workerStates, resultSummary, metadata, scheduledAt, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'queued', ?, 0, 0, 0, 0, 0, JSON_OBJECT(), JSON_OBJECT(), ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, input.TenantID, input.ProjectID, input.TestCycleID, idempotencyKey, input.RequestedConcurrency, string(metadataJSON), input.Now, input.Now, input.Now)
		if err != nil {
			return nil, safeDatabaseFailure("create parallel run schedule", err)
		}
		runID, err := result.LastInsertId()
		if err != nil {
			return nil, safeDatabaseFailure("read parallel run schedule id", err)
		}
		run, err := getParallelRun(ctx, tx, input.TenantID, input.ProjectID, runID)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := tx.Commit(); err != nil {
		return nil, safeDatabaseFailure("commit parallel run schedules", err)
	}
	return runs, nil
}

func ensureParallelRunProjectAndCycle(ctx context.Context, tx *sql.Tx, input browserauth.ParallelRunCreate) error {
	var projectID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM projects WHERE id = ? AND idCostumer = ? FOR UPDATE", input.ProjectID, input.TenantID).Scan(&projectID); err != nil {
		return parallelRunNotFound("lock parallel run project", err)
	}
	var cycleID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM test_cycles WHERE id = ? AND idProject = ? AND idCostumer = ? FOR UPDATE", input.TestCycleID, input.ProjectID, input.TenantID).Scan(&cycleID); err != nil {
		return parallelRunNotFound("lock parallel run test cycle", err)
	}
	return nil
}

const parallelRunSelect = `SELECT id, idProject, testCycleId, performedTestCycleId, idempotencyKey, status,
	requestedConcurrency, activeWorkers, totalWorkers, completedWorkers, failedWorkers, cancelledWorkers,
	aggregateStatus, workerStates, resultSummary, metadata, scheduledAt, startedAt, completedAt, cancelledAt FROM parallel_run_schedules`

func getParallelRun(ctx context.Context, queryer parallelRunQueryer, tenantID, projectID, runID int64) (browserauth.ParallelRun, error) {
	return scanParallelRun(queryer.QueryRowContext(ctx, parallelRunSelect+" WHERE id = ? AND idCostumer = ? AND idProject = ?", runID, tenantID, projectID))
}

func scanParallelRun(scanner parallelRunScanner) (browserauth.ParallelRun, error) {
	var run browserauth.ParallelRun
	var performed sql.NullInt64
	var aggregate sql.NullInt64
	var workerJSON, resultJSON, metadataJSON sql.NullString
	var scheduled, started, completed, cancelled sql.NullTime
	err := scanner.Scan(&run.ID, &run.IDProject, &run.TestCycleID, &performed, &run.IdempotencyKey, &run.Status,
		&run.RequestedConcurrency, &run.ActiveWorkers, &run.TotalWorkers, &run.CompletedWorkers, &run.FailedWorkers, &run.CancelledWorkers,
		&aggregate, &workerJSON, &resultJSON, &metadataJSON, &scheduled, &started, &completed, &cancelled)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ParallelRun{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.ParallelRun{}, safeDatabaseFailure("scan parallel run schedule", err)
	}
	run.PerformedTestCycleID = nullInt64Pointer(performed)
	if aggregate.Valid {
		value := int(aggregate.Int64)
		run.AggregateStatus = &value
	}
	run.Metadata = decodeParallelMap(metadataJSON)
	run.ResultSummary = decodeParallelJSON(resultJSON, []any{})
	workers := decodeParallelMap(workerJSON)
	for _, value := range workers {
		if worker, ok := value.(map[string]any); ok && worker["status"] == "lost" {
			run.LostWorkers++
		}
	}
	run.ScheduledAt = nullTimePointer(scheduled)
	run.StartedAt = nullTimePointer(started)
	run.CompletedAt = nullTimePointer(completed)
	run.CancelledAt = nullTimePointer(cancelled)
	run.RunURL = fmt.Sprintf("/api/admin/projects/%d/parallel-runs/%d", run.IDProject, run.ID)
	return run, nil
}

func decodeParallelMap(value sql.NullString) map[string]any {
	result := map[string]any{}
	if value.Valid {
		_ = json.Unmarshal([]byte(value.String), &result)
	}
	return result
}

func decodeParallelJSON(value sql.NullString, fallback any) any {
	if !value.Valid {
		return fallback
	}
	var decoded any
	if err := json.Unmarshal([]byte(value.String), &decoded); err != nil {
		return fallback
	}
	return decoded
}

func parallelRunMatches(metadata map[string]any, filters map[string]string) bool {
	run, _ := metadata["run"].(map[string]any)
	for field, expected := range filters {
		if fmt.Sprint(run[field]) != expected {
			return false
		}
	}
	return true
}

func parallelRunNotFound(action string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ErrNotFound
	}
	return safeDatabaseFailure(action, err)
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func cloneJSONMap(value map[string]any) map[string]any {
	encoded, _ := json.Marshal(value)
	result := map[string]any{}
	_ = json.Unmarshal(encoded, &result)
	return result
}

func parallelMatrixIdempotencyKey(base string, combination map[string]string) string {
	encoded, _ := json.Marshal(combination)
	digest := sha256.Sum256(encoded)
	return base + "-" + hex.EncodeToString(digest[:8])
}

func parallelRunExecutionSnapshot(ctx context.Context, queryer parallelRunQueryer, tenantID, testCycleID int64) (map[string]any, error) {
	var configJSON string
	if err := queryer.QueryRowContext(ctx, "SELECT config FROM test_cycles WHERE id = ? AND idCostumer = ?", testCycleID, tenantID).Scan(&configJSON); err != nil {
		return nil, parallelRunNotFound("load parallel run execution snapshot", err)
	}
	config := map[string]any{}
	_ = json.Unmarshal([]byte(configJSON), &config)
	snapshot := map[string]any{"schemaVersion": "2026-07-28"}
	cycle, err := latestParallelAssetVersion(ctx, queryer, tenantID, "test_cycle", testCycleID)
	if err != nil {
		return nil, err
	}
	snapshot["testCycle"] = cycle
	for _, reference := range []struct {
		key       string
		assetType string
	}{{"tests", "test"}, {"steps", "step"}, {"environments", "environment"}} {
		items := []any{}
		for _, id := range parallelReferenceIDs(config[reference.key]) {
			version, err := latestParallelAssetVersion(ctx, queryer, tenantID, reference.assetType, id)
			if err != nil {
				return nil, err
			}
			if version == nil {
				version = map[string]any{"assetType": reference.assetType, "assetId": id, "version": nil}
			}
			items = append(items, version)
		}
		snapshot[reference.key] = items
	}
	return snapshot, nil
}

func latestParallelAssetVersion(ctx context.Context, queryer parallelRunQueryer, tenantID int64, assetType string, assetID int64) (map[string]any, error) {
	var id int64
	var version int
	err := queryer.QueryRowContext(ctx, `SELECT id, version FROM asset_versions
		WHERE idCostumer = ? AND assetType = ? AND assetId = ? ORDER BY version DESC LIMIT 1`, tenantID, assetType, assetID).Scan(&id, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, safeDatabaseFailure("load parallel run asset version", err)
	}
	return map[string]any{"assetType": assetType, "assetId": assetID, "version": version, "versionId": id}, nil
}

func parallelReferenceIDs(value any) []int64 {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := []int64{}
	seen := map[int64]bool{}
	for _, item := range items {
		var id int64
		switch typed := item.(type) {
		case float64:
			if typed == float64(int64(typed)) {
				id = int64(typed)
			}
		case map[string]any:
			for _, key := range []string{"id", "assetId"} {
				if number, ok := typed[key].(float64); ok && number == float64(int64(number)) {
					id = int64(number)
					break
				}
			}
		}
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}
