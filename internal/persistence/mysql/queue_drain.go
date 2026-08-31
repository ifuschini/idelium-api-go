package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/idelium/idelium-api-go/internal/integrations"
)

func (r *BrowserAuthRepository) QueuedJobs(ctx context.Context, queueDriver string) (int64, error) {
	if queueDriver == "sync" {
		return 0, nil
	}
	return r.aggregateCount(ctx, "jobs", "SELECT COUNT(*) FROM jobs")
}

func (r *BrowserAuthRepository) FailedJobs(ctx context.Context) (int64, error) {
	return r.aggregateCount(ctx, "failed_jobs", "SELECT COUNT(*) FROM failed_jobs")
}

func (r *BrowserAuthRepository) PendingDomainJobs(ctx context.Context) (map[string]int64, error) {
	counts := map[string]int64{}
	queries := []struct {
		name  string
		query string
	}{
		{"integration_deliveries", "SELECT COUNT(*) FROM integration_deliveries WHERE status IN ('pending', 'failed')"},
		{"result_exports", "SELECT COUNT(*) FROM result_exports WHERE status = 'queued'"},
	}
	for _, item := range queries {
		count, err := r.aggregateCount(ctx, item.name, item.query)
		if err != nil {
			return nil, err
		}
		counts[item.name] = count
	}
	return counts, nil
}

func (r *BrowserAuthRepository) aggregateCount(ctx context.Context, table, query string) (int64, error) {
	var exists int
	if err := r.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&exists); err != nil {
		return 0, safeDatabaseFailure("inspect Laravel queue drain schema", err)
	}
	if exists != 1 {
		return 0, errors.New("Laravel queue drain schema is incomplete")
	}
	var count int64
	if err := r.database.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, safeDatabaseFailure("inspect Laravel queue drain aggregate", err)
	}
	return count, nil
}

func (r *BrowserAuthRepository) NextDispatchID(ctx context.Context, now time.Time) (int64, error) {
	var id int64
	err := r.database.QueryRowContext(ctx, `SELECT id FROM integration_deliveries
		WHERE status = 'pending' OR (status = 'failed' AND (nextAttemptAt IS NULL OR nextAttemptAt <= ?))
		ORDER BY COALESCE(nextAttemptAt, created_at), id LIMIT 1`, now).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, integrations.ErrNoPendingDelivery
	}
	if err != nil {
		return 0, safeDatabaseFailure("select pending integration delivery", err)
	}
	return id, nil
}
