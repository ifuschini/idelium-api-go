package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIEnvironmentRepository reads tenant-owned CLI environment configuration.
type CLIEnvironmentRepository struct {
	database *sql.DB
}

// NewCLIEnvironmentRepository creates a MySQL-backed CLI environment repository.
func NewCLIEnvironmentRepository(database *sql.DB) *CLIEnvironmentRepository {
	return &CLIEnvironmentRepository{database: database}
}

// ListEnvironments returns environments scoped to the authenticated customer and project.
func (repository *CLIEnvironmentRepository) ListEnvironments(ctx context.Context, customerID int64, projectID int64) ([]cliapi.Environment, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, code, description, config, idProject, created_at, updated_at, idCostumer
		 FROM environments
		 WHERE idProject = ? AND idCostumer = ?
		 ORDER BY id ASC`,
		projectID,
		customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list CLI environments: %w", safeDatabaseFailure("list CLI environments", err))
	}
	defer rows.Close()

	environments := []cliapi.Environment{}
	for rows.Next() {
		environment, err := scanCLIEnvironment(rows)
		if err != nil {
			return nil, err
		}
		environments = append(environments, environment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list CLI environments rows: %w", safeDatabaseFailure("list CLI environments rows", err))
	}
	return environments, nil
}

// GetEnvironment returns one environment scoped to the authenticated customer.
func (repository *CLIEnvironmentRepository) GetEnvironment(ctx context.Context, customerID int64, environmentID int64) (cliapi.Environment, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`SELECT id, code, description, config, idProject, created_at, updated_at, idCostumer
		 FROM environments
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1`,
		environmentID,
		customerID,
	)
	environment, err := scanCLIEnvironment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return cliapi.Environment{}, cliapi.ErrNotFound
	}
	if err != nil {
		return cliapi.Environment{}, fmt.Errorf("read CLI environment: %w", safeDatabaseFailure("read CLI environment", err))
	}
	return environment, nil
}

type cliEnvironmentScanner interface {
	Scan(dest ...any) error
}

func scanCLIEnvironment(scanner cliEnvironmentScanner) (cliapi.Environment, error) {
	var environment cliapi.Environment
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := scanner.Scan(
		&environment.ID,
		&environment.Code,
		&environment.Description,
		&environment.Config,
		&environment.IDProject,
		&createdAt,
		&updatedAt,
		&environment.IDCostumer,
	)
	if err != nil {
		return cliapi.Environment{}, err
	}
	if createdAt.Valid {
		environment.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		environment.UpdatedAt = &updatedAt.Time
	}
	return environment, nil
}
