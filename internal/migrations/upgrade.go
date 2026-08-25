package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

// MigrationTableInspector lists migration markers from a Laravel-compatible
// migrations table.
type MigrationTableInspector interface {
	AppliedMigrations(ctx context.Context) ([]string, error)
}

// LaravelUpgradeVerification records compatibility evidence for upgrading from
// the last Laravel-owned schema release into the Go migration handover path.
type LaravelUpgradeVerification struct {
	BaselineID         string   `json:"baseline_id"`
	SourceRelease      string   `json:"source_release"`
	MigrationCount     int      `json:"migration_count"`
	Status             string   `json:"status"`
	MissingMigrations  []string `json:"missing_migrations,omitempty"`
	UnexpectedMarkers  []string `json:"unexpected_markers,omitempty"`
	Blockers           []string `json:"blockers,omitempty"`
	VerifiedActions    []string `json:"verified_actions"`
	ReadsApplicationDB bool     `json:"reads_application_database"`
}

// MySQLMigrationTableInspector reads migration names without application rows.
type MySQLMigrationTableInspector struct {
	database *sql.DB
}

// NewMySQLMigrationTableInspector creates a Laravel migration marker inspector.
func NewMySQLMigrationTableInspector(database *sql.DB) MySQLMigrationTableInspector {
	return MySQLMigrationTableInspector{database: database}
}

// AppliedMigrations returns sorted migration marker names.
func (inspector MySQLMigrationTableInspector) AppliedMigrations(ctx context.Context) ([]string, error) {
	if inspector.database == nil {
		return nil, errors.New("database handle is required")
	}
	rows, err := inspector.database.QueryContext(ctx, "SELECT migration FROM migrations ORDER BY migration ASC")
	if err != nil {
		return nil, fmt.Errorf("inspect Laravel migration markers: database unavailable")
	}
	defer rows.Close()

	migrations := []string{}
	for rows.Next() {
		var migration string
		if err := rows.Scan(&migration); err != nil {
			return nil, fmt.Errorf("inspect Laravel migration markers: database unavailable")
		}
		migrations = append(migrations, migration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inspect Laravel migration markers: database unavailable")
	}
	sort.Strings(migrations)
	return migrations, nil
}

// VerifyLaravelUpgrade checks that the target database contains every reviewed
// Laravel migration marker required by the Go baseline.
func VerifyLaravelUpgrade(
	ctx context.Context,
	inspector MigrationTableInspector,
	sourceRelease string,
) (LaravelUpgradeVerification, error) {
	if sourceRelease == "" {
		return LaravelUpgradeVerification{}, errors.New("source Laravel release is required")
	}
	if inspector == nil {
		return LaravelUpgradeVerification{}, errors.New("migration table inspector is required")
	}
	manifest, err := ReviewedBaseline()
	if err != nil {
		return LaravelUpgradeVerification{}, err
	}
	applied, err := inspector.AppliedMigrations(ctx)
	if err != nil {
		return LaravelUpgradeVerification{}, err
	}

	required := map[string]struct{}{}
	for _, migration := range manifest.Migrations {
		required[migration.Name] = struct{}{}
	}
	appliedSet := map[string]struct{}{}
	for _, migration := range applied {
		appliedSet[migration] = struct{}{}
	}

	missing := []string{}
	for name := range required {
		if _, ok := appliedSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	unexpected := []string{}
	for name := range appliedSet {
		if _, ok := required[name]; !ok {
			unexpected = append(unexpected, name)
		}
	}
	sort.Strings(unexpected)

	verification := LaravelUpgradeVerification{
		BaselineID:         manifest.BaselineID,
		SourceRelease:      sourceRelease,
		MigrationCount:     manifest.MigrationCount,
		MissingMigrations:  missing,
		UnexpectedMarkers:  unexpected,
		ReadsApplicationDB: false,
		VerifiedActions: []string{
			"loaded reviewed Go baseline manifest",
			"read Laravel migration marker names only",
			"compared applied markers against reviewed baseline",
		},
	}
	if len(missing) > 0 {
		verification.Status = "failed"
		verification.Blockers = []string{"source Laravel schema is missing reviewed baseline migration markers"}
		return verification, nil
	}
	if len(unexpected) > 0 {
		verification.Status = "review-required"
		verification.Blockers = []string{"migration table contains markers outside the reviewed Go baseline"}
		return verification, nil
	}
	if !manifest.HandoverPolicy.GoBaselineApplicationEnabled {
		verification.Status = "blocked"
		verification.Blockers = []string{"Go baseline application is disabled until route cutover and rollback rehearsal gates pass"}
		return verification, nil
	}
	verification.Status = "ready"
	return verification, nil
}
