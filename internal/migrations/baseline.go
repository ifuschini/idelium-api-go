// Package migrations exposes reviewed schema migration metadata.
package migrations

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

//go:embed baseline_manifest.json
var baselineFS embed.FS

// BaselineManifest describes the reviewed Laravel schema baseline.
type BaselineManifest struct {
	SchemaVersion   int                 `json:"schema_version"`
	BaselineID      string              `json:"baseline_id"`
	GeneratedOn     string              `json:"generated_on"`
	SourceRuntime   string              `json:"source_runtime"`
	TargetRuntime   string              `json:"target_runtime"`
	SourceDirectory string              `json:"source_directory"`
	MigrationCount  int                 `json:"migration_count"`
	AggregateSHA256 string              `json:"aggregate_sha256"`
	ReviewStatus    string              `json:"review_status"`
	HandoverPolicy  BaselinePolicy      `json:"handover_policy"`
	Redaction       string              `json:"redaction"`
	Migrations      []BaselineMigration `json:"migrations"`
}

// BaselinePolicy records the safe handover state for the reviewed baseline.
type BaselinePolicy struct {
	LaravelRemainsSchemaOwner    bool   `json:"laravel_remains_schema_owner"`
	GoBaselineApplicationEnabled bool   `json:"go_baseline_application_enabled"`
	DualWritesAllowed            bool   `json:"dual_writes_allowed"`
	Rollback                     string `json:"rollback"`
}

// BaselineMigration records one Laravel migration covered by the baseline.
type BaselineMigration struct {
	Name   string `json:"name"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// BaselinePlan is the safe public plan emitted by cmd/migrate before cutover.
type BaselinePlan struct {
	BaselineID                   string `json:"baseline_id"`
	MigrationCount               int    `json:"migration_count"`
	AggregateSHA256              string `json:"aggregate_sha256"`
	ReviewStatus                 string `json:"review_status"`
	LaravelRemainsSchemaOwner    bool   `json:"laravel_remains_schema_owner"`
	GoBaselineApplicationEnabled bool   `json:"go_baseline_application_enabled"`
	DualWritesAllowed            bool   `json:"dual_writes_allowed"`
	Rollback                     string `json:"rollback"`
}

// ReviewedBaseline returns the embedded reviewed schema baseline manifest.
func ReviewedBaseline() (BaselineManifest, error) {
	contents, err := baselineFS.ReadFile("baseline_manifest.json")
	if err != nil {
		return BaselineManifest{}, fmt.Errorf("read Go baseline manifest: %w", err)
	}

	var manifest BaselineManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return BaselineManifest{}, fmt.Errorf("decode Go baseline manifest: %w", err)
	}
	if err := validateBaseline(manifest); err != nil {
		return BaselineManifest{}, err
	}
	return manifest, nil
}

// ReviewedBaselinePlan returns the safe migration plan without source file hashes.
func ReviewedBaselinePlan() (BaselinePlan, error) {
	manifest, err := ReviewedBaseline()
	if err != nil {
		return BaselinePlan{}, err
	}
	return BaselinePlan{
		BaselineID:                   manifest.BaselineID,
		MigrationCount:               manifest.MigrationCount,
		AggregateSHA256:              manifest.AggregateSHA256,
		ReviewStatus:                 manifest.ReviewStatus,
		LaravelRemainsSchemaOwner:    manifest.HandoverPolicy.LaravelRemainsSchemaOwner,
		GoBaselineApplicationEnabled: manifest.HandoverPolicy.GoBaselineApplicationEnabled,
		DualWritesAllowed:            manifest.HandoverPolicy.DualWritesAllowed,
		Rollback:                     manifest.HandoverPolicy.Rollback,
	}, nil
}

func validateBaseline(manifest BaselineManifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported Go baseline manifest schema version %d", manifest.SchemaVersion)
	}
	if manifest.BaselineID == "" || manifest.AggregateSHA256 == "" {
		return errors.New("Go baseline manifest is missing required identity fields")
	}
	if manifest.MigrationCount != len(manifest.Migrations) || manifest.MigrationCount == 0 {
		return errors.New("Go baseline manifest migration count is inconsistent")
	}
	if manifest.HandoverPolicy.GoBaselineApplicationEnabled {
		return errors.New("Go baseline application must remain disabled until cutover gates pass")
	}
	if manifest.HandoverPolicy.DualWritesAllowed {
		return errors.New("Go baseline manifest must not allow dual writes")
	}

	aggregate := sha256.New()
	for _, migration := range manifest.Migrations {
		if migration.File == "" || migration.SHA256 == "" || migration.Bytes <= 0 {
			return errors.New("Go baseline manifest contains an incomplete migration record")
		}
		aggregate.Write([]byte(migration.File))
		aggregate.Write([]byte{0})
		aggregate.Write([]byte(migration.SHA256))
		aggregate.Write([]byte{0})
	}
	if hex.EncodeToString(aggregate.Sum(nil)) != manifest.AggregateSHA256 {
		return errors.New("Go baseline aggregate checksum does not match migration records")
	}

	return nil
}
