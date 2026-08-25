package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// BridgeOptions controls how the Laravel migrations table is marked.
type BridgeOptions struct {
	ConfirmBaselineID string
	Batch             int
}

// BridgePlan is the safe public plan for marking a Laravel schema baseline as
// already applied by the Go runtime.
type BridgePlan struct {
	BaselineID          string   `json:"baseline_id"`
	MigrationCount      int      `json:"migration_count"`
	Batch               int      `json:"batch"`
	Table               string   `json:"table"`
	StatementTemplate   string   `json:"statement_template"`
	Migrations          []string `json:"migrations"`
	RequiresExplicitRun bool     `json:"requires_explicit_run"`
}

// BridgeResult summarizes an executed Laravel baseline mark operation without
// exposing database details.
type BridgeResult struct {
	BaselineID     string `json:"baseline_id"`
	MigrationCount int    `json:"migration_count"`
	Batch          int    `json:"batch"`
	Applied        int    `json:"applied"`
	Skipped        int    `json:"skipped"`
}

// SQLExecer is implemented by sql.DB and sql.Tx.
type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const bridgeStatementTemplate = "INSERT INTO migrations (migration, batch) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM migrations WHERE migration = ?)"

// ReviewedBaselineBridgePlan returns a safe dry-run plan for inserting Laravel
// migration-table markers for the reviewed baseline.
func ReviewedBaselineBridgePlan(options BridgeOptions) (BridgePlan, error) {
	manifest, err := ReviewedBaseline()
	if err != nil {
		return BridgePlan{}, err
	}
	if err := validateBridgeOptions(manifest, options); err != nil {
		return BridgePlan{}, err
	}

	migrationNames := make([]string, 0, len(manifest.Migrations))
	for _, migration := range manifest.Migrations {
		migrationNames = append(migrationNames, migration.Name)
	}

	return BridgePlan{
		BaselineID:          manifest.BaselineID,
		MigrationCount:      len(migrationNames),
		Batch:               options.Batch,
		Table:               "migrations",
		StatementTemplate:   bridgeStatementTemplate,
		Migrations:          migrationNames,
		RequiresExplicitRun: true,
	}, nil
}

// MarkReviewedBaselineApplied records the reviewed Laravel baseline in the
// Laravel-compatible migrations table. It is idempotent and inserts only missing
// migration names.
func MarkReviewedBaselineApplied(
	ctx context.Context,
	execer SQLExecer,
	options BridgeOptions,
) (BridgeResult, error) {
	if execer == nil {
		return BridgeResult{}, errors.New("migration marker executor is required")
	}
	plan, err := ReviewedBaselineBridgePlan(options)
	if err != nil {
		return BridgeResult{}, err
	}

	result := BridgeResult{
		BaselineID:     plan.BaselineID,
		MigrationCount: plan.MigrationCount,
		Batch:          plan.Batch,
	}
	for _, migration := range plan.Migrations {
		execResult, err := execer.ExecContext(
			ctx,
			bridgeStatementTemplate,
			migration,
			plan.Batch,
			migration,
		)
		if err != nil {
			return BridgeResult{}, fmt.Errorf("mark Laravel baseline as applied: database unavailable")
		}
		rowsAffected, err := execResult.RowsAffected()
		if err != nil {
			return BridgeResult{}, fmt.Errorf("mark Laravel baseline as applied: rows affected unavailable")
		}
		if rowsAffected > 0 {
			result.Applied++
		} else {
			result.Skipped++
		}
	}
	return result, nil
}

func validateBridgeOptions(manifest BaselineManifest, options BridgeOptions) error {
	if options.ConfirmBaselineID == "" {
		return errors.New("confirmed baseline ID is required")
	}
	if options.ConfirmBaselineID != manifest.BaselineID {
		return errors.New("confirmed baseline ID does not match reviewed baseline")
	}
	if options.Batch <= 0 {
		return errors.New("migration batch must be greater than zero")
	}
	return nil
}
