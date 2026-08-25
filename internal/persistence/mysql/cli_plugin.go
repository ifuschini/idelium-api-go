package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/cliapi"
)

// CLIPluginRepository reads tenant-owned CLI plugin configuration.
type CLIPluginRepository struct {
	database *sql.DB
}

// NewCLIPluginRepository creates a MySQL-backed CLI plugin repository.
func NewCLIPluginRepository(database *sql.DB) *CLIPluginRepository {
	return &CLIPluginRepository{database: database}
}

// ListPlugins returns plugins scoped to the authenticated customer and project.
func (repository *CLIPluginRepository) ListPlugins(ctx context.Context, customerID int64, projectID int64) ([]cliapi.Plugin, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, name, code, description, idProject, created_at, updated_at, idCostumer
		 FROM plugins
		 WHERE idProject = ? AND idCostumer = ?
		 ORDER BY id ASC`,
		projectID,
		customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list CLI plugins: %w", safeDatabaseFailure("list CLI plugins", err))
	}
	defer rows.Close()

	plugins := []cliapi.Plugin{}
	for rows.Next() {
		plugin, err := scanCLIPlugin(rows)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list CLI plugins rows: %w", safeDatabaseFailure("list CLI plugins rows", err))
	}
	return plugins, nil
}

// GetPlugin returns one plugin scoped to the authenticated customer.
func (repository *CLIPluginRepository) GetPlugin(ctx context.Context, customerID int64, pluginID int64) (cliapi.Plugin, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`SELECT id, name, code, description, idProject, created_at, updated_at, idCostumer
		 FROM plugins
		 WHERE id = ? AND idCostumer = ?
		 LIMIT 1`,
		pluginID,
		customerID,
	)
	plugin, err := scanCLIPlugin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return cliapi.Plugin{}, cliapi.ErrNotFound
	}
	if err != nil {
		return cliapi.Plugin{}, fmt.Errorf("read CLI plugin: %w", safeDatabaseFailure("read CLI plugin", err))
	}
	return plugin, nil
}

type cliPluginScanner interface {
	Scan(dest ...any) error
}

func scanCLIPlugin(scanner cliPluginScanner) (cliapi.Plugin, error) {
	var plugin cliapi.Plugin
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := scanner.Scan(
		&plugin.ID,
		&plugin.Name,
		&plugin.Code,
		&plugin.Description,
		&plugin.IDProject,
		&createdAt,
		&updatedAt,
		&plugin.IDCostumer,
	)
	if err != nil {
		return cliapi.Plugin{}, err
	}
	if createdAt.Valid {
		plugin.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		plugin.UpdatedAt = &updatedAt.Time
	}
	return plugin, nil
}
