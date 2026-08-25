package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLITestCycleRepository reads tenant-owned CLI test-cycle configuration.
type CLITestCycleRepository struct {
	database *sql.DB
}

// NewCLITestCycleRepository creates a MySQL-backed CLI test-cycle repository.
func NewCLITestCycleRepository(database *sql.DB) *CLITestCycleRepository {
	return &CLITestCycleRepository{database: database}
}

// GetTestCycle returns one test cycle scoped to the authenticated customer.
func (repository *CLITestCycleRepository) GetTestCycle(ctx context.Context, customerID int64, testCycleID int64) (cliapi.TestCycle, error) {
	var cycle cliapi.TestCycle
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT id, name, description, config, idProject, created_at, updated_at, idCostumer
		 FROM test_cycles
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1`,
		testCycleID,
		customerID,
	).Scan(
		&cycle.ID,
		&cycle.Name,
		&cycle.Description,
		&cycle.Config,
		&cycle.IDProject,
		&createdAt,
		&updatedAt,
		&cycle.IDCostumer,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cliapi.TestCycle{}, cliapi.ErrNotFound
	}
	if err != nil {
		return cliapi.TestCycle{}, fmt.Errorf("read CLI test cycle: %w", safeDatabaseFailure("read CLI test cycle", err))
	}
	if createdAt.Valid {
		cycle.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		cycle.UpdatedAt = &updatedAt.Time
	}

	return cycle, nil
}
