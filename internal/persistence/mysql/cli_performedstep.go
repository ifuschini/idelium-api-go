package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIPerformedStepRepository writes tenant-owned CLI result step records.
type CLIPerformedStepRepository struct {
	database *sql.DB
}

// NewCLIPerformedStepRepository creates a MySQL-backed CLI performed-step repository.
func NewCLIPerformedStepRepository(database *sql.DB) *CLIPerformedStepRepository {
	return &CLIPerformedStepRepository{database: database}
}

// CreatePerformedStep creates a performed step only when all referenced resources belong to the tenant.
func (repository *CLIPerformedStepRepository) CreatePerformedStep(ctx context.Context, customerID int64, command cliapi.CreatePerformedStepRequest) (int64, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin CLI performed-step create: %w", safeDatabaseFailure("begin CLI performed-step create", err))
	}
	defer rollbackUnlessCommitted(transaction)

	cycleExists, err := tenantRowExists(ctx, transaction, "performed_test_cycles", command.TestCycleID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check performed-cycle ownership: %w", safeDatabaseFailure("check performed-cycle ownership", err))
	}
	testExists, err := tenantPerformedTestExists(ctx, transaction, command.TestID, command.TestCycleID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check performed-test ownership: %w", safeDatabaseFailure("check performed-test ownership", err))
	}
	stepExists, err := tenantRowExists(ctx, transaction, "steps", command.StepID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check source step ownership: %w", safeDatabaseFailure("check source step ownership", err))
	}
	if !cycleExists || !testExists || !stepExists {
		return 0, cliapi.ErrNotFound
	}

	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO performed_steps
			(testCycleDoneId, testDoneId, stepId, status, name, screenshots, type, data, idCostumer, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		command.TestCycleID,
		command.TestID,
		command.StepID,
		command.Status,
		command.Name,
		command.Screenshots,
		command.Type,
		command.Data,
		customerID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert CLI performed step: %w", safeDatabaseFailure("insert CLI performed step", err))
	}
	performedStepID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read CLI performed-step identifier: %w", safeDatabaseFailure("read CLI performed-step identifier", err))
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit CLI performed-step create: %w", safeDatabaseFailure("commit CLI performed-step create", err))
	}
	return performedStepID, nil
}

// UpdatePerformedStep updates screenshots only when the performed step belongs to the tenant.
func (repository *CLIPerformedStepRepository) UpdatePerformedStep(ctx context.Context, customerID int64, command cliapi.UpdatePerformedStepRequest) (int64, error) {
	result, err := repository.database.ExecContext(
		ctx,
		`UPDATE performed_steps
		 SET screenshots = ?, updated_at = NOW()
		 WHERE id = ? AND idCostumer = ?`,
		command.Screenshots,
		command.StepID,
		customerID,
	)
	if err != nil {
		return 0, fmt.Errorf("update CLI performed step: %w", safeDatabaseFailure("update CLI performed step", err))
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read CLI performed-step update count: %w", safeDatabaseFailure("read CLI performed-step update count", err))
	}
	if rowsAffected == 0 {
		var existingID int64
		err := repository.database.QueryRowContext(
			ctx,
			"SELECT id FROM performed_steps WHERE id = ? AND idCostumer = ? LIMIT 1",
			command.StepID,
			customerID,
		).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, cliapi.ErrNotFound
		}
		if err != nil {
			return 0, fmt.Errorf("read CLI performed step after no-op update: %w", safeDatabaseFailure("read CLI performed step after no-op update", err))
		}
	}
	return command.StepID, nil
}

func tenantPerformedTestExists(ctx context.Context, transaction *sql.Tx, testID int64, testCycleID int64, customerID int64) (bool, error) {
	var exists bool
	err := transaction.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM performed_tests
			WHERE id = ? AND testCycleDoneId = ? AND idCostumer = ?
		)`,
		testID,
		testCycleID,
		customerID,
	).Scan(&exists)
	return exists, err
}
