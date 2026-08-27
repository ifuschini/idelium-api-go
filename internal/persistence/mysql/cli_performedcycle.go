package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIPerformedCycleRepository writes tenant-owned CLI result cycle records.
type CLIPerformedCycleRepository struct {
	database *sql.DB
}

// NewCLIPerformedCycleRepository creates a MySQL-backed CLI performed-cycle repository.
func NewCLIPerformedCycleRepository(database *sql.DB) *CLIPerformedCycleRepository {
	return &CLIPerformedCycleRepository{database: database}
}

// CreatePerformedCycle creates a performed cycle only when the referenced cycle belongs to the tenant.
func (repository *CLIPerformedCycleRepository) CreatePerformedCycle(ctx context.Context, customerID int64, command cliapi.CreatePerformedCycleRequest) (int64, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin CLI performed-cycle create: %w", safeDatabaseFailure("begin CLI performed-cycle create", err))
	}
	defer rollbackUnlessCommitted(transaction)

	cycleExists, err := tenantRowExists(ctx, transaction, "test_cycles", command.TestCycleID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check test-cycle ownership: %w", safeDatabaseFailure("check test-cycle ownership", err))
	}
	if !cycleExists {
		return 0, cliapi.ErrNotFound
	}

	result, err := repository.insertPerformedCycle(ctx, transaction, customerID, command)
	if err != nil {
		return 0, fmt.Errorf("insert CLI performed cycle: %w", safeDatabaseFailure("insert CLI performed cycle", err))
	}
	performedCycleID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read CLI performed-cycle identifier: %w", safeDatabaseFailure("read CLI performed-cycle identifier", err))
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit CLI performed-cycle create: %w", safeDatabaseFailure("commit CLI performed-cycle create", err))
	}
	return performedCycleID, nil
}

func (repository *CLIPerformedCycleRepository) insertPerformedCycle(ctx context.Context, transaction *sql.Tx, customerID int64, command cliapi.CreatePerformedCycleRequest) (sql.Result, error) {
	if command.IdempotencyKey != "" {
		return transaction.ExecContext(ctx, `INSERT INTO performed_test_cycles (testCycleId, date, status, idCostumer, idempotencyKey, created_at, updated_at) VALUES (?, NOW(), 0, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, command.TestCycleID, customerID, command.IdempotencyKey)
	}
	if command.ExecutionContext != nil {
		column, found, err := optionalPerformedCycleExecutionContextColumn(ctx, transaction)
		if err != nil {
			return nil, err
		}
		if found {
			columnSQL, ok := performedCycleExecutionContextColumnSQL(column)
			if !ok {
				return nil, fmt.Errorf("unsupported performed-cycle execution context column %q", column)
			}
			return transaction.ExecContext(
				ctx,
				fmt.Sprintf(`INSERT INTO performed_test_cycles
					(testCycleId, date, status, idCostumer, created_at, updated_at, %s)
				 VALUES (?, NOW(), 0, ?, NOW(), NOW(), ?)`, columnSQL),
				command.TestCycleID,
				customerID,
				*command.ExecutionContext,
			)
		}
	}

	return transaction.ExecContext(
		ctx,
		`INSERT INTO performed_test_cycles
			(testCycleId, date, status, idCostumer, created_at, updated_at)
		 VALUES (?, NOW(), 0, ?, NOW(), NOW())`,
		command.TestCycleID,
		customerID,
	)
}

func optionalPerformedCycleExecutionContextColumn(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (string, bool, error) {
	for _, column := range []string{"executionContext", "execution_context", "context"} {
		exists, err := mysqlColumnExists(ctx, queryer, "performed_test_cycles", column)
		if err != nil {
			return "", false, err
		}
		if exists {
			return column, true, nil
		}
	}
	return "", false, nil
}

func performedCycleExecutionContextColumnSQL(column string) (string, bool) {
	switch column {
	case "executionContext":
		return "`executionContext`", true
	case "execution_context":
		return "`execution_context`", true
	case "context":
		return "`context`", true
	default:
		return "", false
	}
}

// UpdatePerformedCycle updates a performed cycle only when the record belongs to the tenant.
func (repository *CLIPerformedCycleRepository) UpdatePerformedCycle(ctx context.Context, customerID int64, command cliapi.UpdatePerformedCycleRequest) (int64, error) {
	result, err := repository.database.ExecContext(
		ctx,
		`UPDATE performed_test_cycles
		 SET status = ?, updated_at = NOW()
		 WHERE id = ? AND idCostumer = ?`,
		command.Status,
		command.TestCycleID,
		customerID,
	)
	if err != nil {
		return 0, fmt.Errorf("update CLI performed cycle: %w", safeDatabaseFailure("update CLI performed cycle", err))
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read CLI performed-cycle update count: %w", safeDatabaseFailure("read CLI performed-cycle update count", err))
	}
	if rowsAffected == 0 {
		var existingID int64
		err := repository.database.QueryRowContext(
			ctx,
			"SELECT id FROM performed_test_cycles WHERE id = ? AND idCostumer = ? LIMIT 1",
			command.TestCycleID,
			customerID,
		).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, cliapi.ErrNotFound
		}
		if err != nil {
			return 0, fmt.Errorf("read CLI performed cycle after no-op update: %w", safeDatabaseFailure("read CLI performed cycle after no-op update", err))
		}
	}
	return command.TestCycleID, nil
}
