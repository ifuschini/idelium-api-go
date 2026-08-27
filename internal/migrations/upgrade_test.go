package migrations

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVerifyLaravelUpgradeReportsPolicyBlockerForCompleteBaseline(t *testing.T) {
	manifest, err := ReviewedBaseline()
	if err != nil {
		t.Fatalf("ReviewedBaseline() returned an error: %v", err)
	}
	applied := make([]string, 0, len(manifest.Migrations))
	for _, migration := range manifest.Migrations {
		applied = append(applied, migration.Name)
	}

	verification, err := VerifyLaravelUpgrade(context.Background(), fakeMigrationTableInspector{applied: applied}, "laravel-legacy")
	if err != nil {
		t.Fatalf("VerifyLaravelUpgrade() returned an error: %v", err)
	}

	if verification.Status != "blocked" {
		t.Fatalf("expected blocked status while baseline application is disabled: %#v", verification)
	}
	if len(verification.MissingMigrations) != 0 || len(verification.UnexpectedMarkers) != 0 {
		t.Fatalf("complete baseline reported drift: %#v", verification)
	}
	if verification.ReadsApplicationDB {
		t.Fatal("upgrade verification must not read application tables")
	}
}

func TestVerifyLaravelUpgradeFailsWhenBaselineMarkerIsMissing(t *testing.T) {
	verification, err := VerifyLaravelUpgrade(context.Background(), fakeMigrationTableInspector{}, "laravel-legacy")
	if err != nil {
		t.Fatalf("VerifyLaravelUpgrade() returned an error: %v", err)
	}

	if verification.Status != "failed" {
		t.Fatalf("expected failed status for missing migration markers: %#v", verification)
	}
	if len(verification.MissingMigrations) != 68 {
		t.Fatalf("unexpected missing migration count: %#v", verification)
	}
}

func TestVerifyLaravelUpgradeRequiresSourceRelease(t *testing.T) {
	_, err := VerifyLaravelUpgrade(context.Background(), fakeMigrationTableInspector{}, "")

	if err == nil {
		t.Fatal("VerifyLaravelUpgrade() accepted an empty source release")
	}
	if !strings.Contains(err.Error(), "source Laravel release is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestVerifyLaravelUpgradeReturnsSafeDatabaseDiagnostics(t *testing.T) {
	_, err := VerifyLaravelUpgrade(context.Background(), fakeMigrationTableInspector{
		err: errors.New("password=secret dsn detail"),
	}, "laravel-legacy")

	if err == nil {
		t.Fatal("expected migration marker inspection failure")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "password") {
		t.Fatalf("upgrade diagnostic leaked sensitive details: %v", err)
	}
}

type fakeMigrationTableInspector struct {
	applied []string
	err     error
}

func (inspector fakeMigrationTableInspector) AppliedMigrations(context.Context) ([]string, error) {
	if inspector.err != nil {
		return nil, errors.New("inspect Laravel migration markers: database unavailable")
	}
	return inspector.applied, nil
}
