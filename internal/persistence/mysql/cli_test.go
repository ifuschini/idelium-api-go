package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLITestRepository reads tenant-owned CLI test configuration.
type CLITestRepository struct {
	database *sql.DB
}

// NewCLITestRepository creates a MySQL-backed CLI test repository.
func NewCLITestRepository(database *sql.DB) *CLITestRepository {
	return &CLITestRepository{database: database}
}

// GetTest returns one test scoped to the authenticated customer.
func (repository *CLITestRepository) GetTest(ctx context.Context, customerID int64, testID int64) (cliapi.Test, error) {
	var test cliapi.Test
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT id, name, description, config, idProject, created_at, updated_at, idCostumer
		 FROM tests
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1`,
		testID,
		customerID,
	).Scan(
		&test.ID,
		&test.Name,
		&test.Description,
		&test.Config,
		&test.IDProject,
		&createdAt,
		&updatedAt,
		&test.IDCostumer,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cliapi.Test{}, cliapi.ErrNotFound
	}
	if err != nil {
		return cliapi.Test{}, fmt.Errorf("read CLI test: %w", safeDatabaseFailure("read CLI test", err))
	}
	if createdAt.Valid {
		test.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		test.UpdatedAt = &updatedAt.Time
	}

	return test, nil
}
