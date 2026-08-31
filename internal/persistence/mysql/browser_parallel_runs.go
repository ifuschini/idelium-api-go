package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
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
	run.ResultSummary = decodeParallelMap(resultJSON)
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
