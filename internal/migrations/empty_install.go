package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

// SchemaInspector lists application-owned tables in a target database schema.
type SchemaInspector interface {
	UserTables(ctx context.Context) ([]string, error)
}

// EmptyInstallVerification records whether an empty-install migration rehearsal
// can proceed safely.
type EmptyInstallVerification struct {
	BaselineID      string   `json:"baseline_id"`
	MigrationCount  int      `json:"migration_count"`
	Status          string   `json:"status"`
	SchemaEmpty     bool     `json:"schema_empty"`
	ExistingTables  []string `json:"existing_tables,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
	VerifiedActions []string `json:"verified_actions"`
}

// MySQLSchemaInspector reads table names from a single configured schema.
type MySQLSchemaInspector struct {
	database *sql.DB
	schema   string
}

// NewMySQLSchemaInspector creates a schema inspector for empty-install checks.
func NewMySQLSchemaInspector(database *sql.DB, schema string) MySQLSchemaInspector {
	return MySQLSchemaInspector{database: database, schema: schema}
}

// UserTables returns sorted base table names without row or credential data.
func (inspector MySQLSchemaInspector) UserTables(ctx context.Context) ([]string, error) {
	if inspector.database == nil {
		return nil, errors.New("database handle is required")
	}
	if inspector.schema == "" {
		return nil, errors.New("database schema name is required")
	}
	rows, err := inspector.database.QueryContext(
		ctx,
		`SELECT table_name
		 FROM information_schema.tables
		 WHERE table_schema = ? AND table_type = 'BASE TABLE'
		 ORDER BY table_name ASC`,
		inspector.schema,
	)
	if err != nil {
		return nil, fmt.Errorf("inspect empty install schema: database unavailable")
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("inspect empty install schema: database unavailable")
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inspect empty install schema: database unavailable")
	}
	sort.Strings(tables)
	return tables, nil
}

// VerifyEmptyInstall checks whether a target schema can run Go-owned migrations
// from an empty database state.
func VerifyEmptyInstall(ctx context.Context, inspector SchemaInspector) (EmptyInstallVerification, error) {
	manifest, err := ReviewedBaseline()
	if err != nil {
		return EmptyInstallVerification{}, err
	}
	if inspector == nil {
		return EmptyInstallVerification{}, errors.New("schema inspector is required")
	}

	tables, err := inspector.UserTables(ctx)
	if err != nil {
		return EmptyInstallVerification{}, err
	}

	verification := EmptyInstallVerification{
		BaselineID:     manifest.BaselineID,
		MigrationCount: manifest.MigrationCount,
		SchemaEmpty:    len(tables) == 0,
		ExistingTables: tables,
		VerifiedActions: []string{
			"loaded reviewed Go baseline manifest",
			"inspected target schema table inventory",
			"confirmed no tenant data or row payloads were read",
		},
	}
	if len(tables) > 0 {
		verification.Status = "failed"
		verification.Blockers = []string{"target schema is not empty"}
		return verification, nil
	}
	if !manifest.HandoverPolicy.GoBaselineApplicationEnabled {
		verification.Status = "blocked"
		verification.Blockers = []string{"Go baseline application is disabled until bridge, upgrade, route cutover, and rollback rehearsal gates pass"}
		return verification, nil
	}

	verification.Status = "ready"
	return verification, nil
}
