package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIStepRepository reads tenant-owned CLI step configuration.
type CLIStepRepository struct {
	database *sql.DB
}

// NewCLIStepRepository creates a MySQL-backed CLI step repository.
func NewCLIStepRepository(database *sql.DB) *CLIStepRepository {
	return &CLIStepRepository{database: database}
}

// GetStep returns one step scoped to the authenticated customer.
func (repository *CLIStepRepository) GetStep(ctx context.Context, customerID int64, stepID int64) (cliapi.Step, error) {
	var step cliapi.Step
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT id, name, description, config, idProject, `+"`order`"+`, created_at, updated_at, idCostumer
		 FROM steps
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1`,
		stepID,
		customerID,
	).Scan(
		&step.ID,
		&step.Name,
		&step.Description,
		&step.Config,
		&step.IDProject,
		&step.Order,
		&createdAt,
		&updatedAt,
		&step.IDCostumer,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cliapi.Step{}, cliapi.ErrNotFound
	}
	if err != nil {
		return cliapi.Step{}, fmt.Errorf("read CLI step: %w", safeDatabaseFailure("read CLI step", err))
	}
	if createdAt.Valid {
		step.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		step.UpdatedAt = &updatedAt.Time
	}

	return step, nil
}
