package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIPerformedTestRepository writes tenant-owned CLI result test records.
type CLIPerformedTestRepository struct {
	database *sql.DB
}

// NewCLIPerformedTestRepository creates a MySQL-backed CLI performed-test repository.
func NewCLIPerformedTestRepository(database *sql.DB) *CLIPerformedTestRepository {
	return &CLIPerformedTestRepository{database: database}
}

// CreatePerformedTest creates a performed test only when the referenced cycle and test belong to the tenant.
func (repository *CLIPerformedTestRepository) CreatePerformedTest(ctx context.Context, customerID int64, command cliapi.CreatePerformedTestRequest) (int64, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin CLI performed-test create: %w", safeDatabaseFailure("begin CLI performed-test create", err))
	}
	defer rollbackUnlessCommitted(transaction)

	cycleExists, err := tenantRowExists(ctx, transaction, "performed_test_cycles", command.TestCycleID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check performed-test cycle ownership: %w", safeDatabaseFailure("check performed-test cycle ownership", err))
	}
	testExists, err := tenantRowExists(ctx, transaction, "tests", command.TestID, customerID)
	if err != nil {
		return 0, fmt.Errorf("check test ownership: %w", safeDatabaseFailure("check test ownership", err))
	}
	if !cycleExists || !testExists {
		return 0, cliapi.ErrNotFound
	}

	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO performed_tests
			(testCycleDoneId, testId, name, status, idCostumer, created_at, updated_at)
		 VALUES (?, ?, ?, 0, ?, NOW(), NOW())`,
		command.TestCycleID,
		command.TestID,
		command.Name,
		customerID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert CLI performed test: %w", safeDatabaseFailure("insert CLI performed test", err))
	}
	performedTestID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read CLI performed-test identifier: %w", safeDatabaseFailure("read CLI performed-test identifier", err))
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit CLI performed-test create: %w", safeDatabaseFailure("commit CLI performed-test create", err))
	}
	return performedTestID, nil
}

// UpdatePerformedTest updates a performed test only when the record belongs to the tenant.
func (repository *CLIPerformedTestRepository) UpdatePerformedTest(ctx context.Context, customerID int64, command cliapi.UpdatePerformedTestRequest) (int64, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin CLI performed-test update: %w", safeDatabaseFailure("begin CLI performed-test update", err))
	}
	defer rollbackUnlessCommitted(transaction)

	var existingID int64
	err = transaction.QueryRowContext(
		ctx,
		`SELECT id
		 FROM performed_tests
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1
		 FOR UPDATE`,
		command.TestID,
		customerID,
	).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, cliapi.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock CLI performed test: %w", safeDatabaseFailure("lock CLI performed test", err))
	}

	if command.PostmanDataPresent {
		_, err = transaction.ExecContext(
			ctx,
			`UPDATE performed_tests
			 SET status = ?, postmanData = ?, updated_at = NOW()
			 WHERE id = ? AND idCostumer = ?`,
			command.Status,
			command.PostmanData,
			command.TestID,
			customerID,
		)
	} else {
		_, err = transaction.ExecContext(
			ctx,
			`UPDATE performed_tests
			 SET status = ?, updated_at = NOW()
			 WHERE id = ? AND idCostumer = ?`,
			command.Status,
			command.TestID,
			customerID,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("update CLI performed test: %w", safeDatabaseFailure("update CLI performed test", err))
	}

	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit CLI performed-test update: %w", safeDatabaseFailure("commit CLI performed-test update", err))
	}
	return existingID, nil
}

func tenantRowExists(ctx context.Context, transaction *sql.Tx, tableName string, id int64, customerID int64) (bool, error) {
	query := ""
	switch tableName {
	case "performed_test_cycles":
		query = "SELECT EXISTS (SELECT 1 FROM performed_test_cycles WHERE id = ? AND idCostumer = ?)"
	case "test_cycles":
		query = "SELECT EXISTS (SELECT 1 FROM test_cycles WHERE id = ? AND idCostumer = ?)"
	case "tests":
		query = "SELECT EXISTS (SELECT 1 FROM tests WHERE id = ? AND idCostumer = ?)"
	case "steps":
		query = "SELECT EXISTS (SELECT 1 FROM steps WHERE id = ? AND idCostumer = ?)"
	default:
		return false, fmt.Errorf("unsupported tenant lookup table")
	}
	var exists bool
	err := transaction.QueryRowContext(
		ctx,
		query,
		id,
		customerID,
	).Scan(&exists)
	return exists, err
}

func rollbackUnlessCommitted(transaction *sql.Tx) {
	_ = transaction.Rollback()
}
