package migrations

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestVerifyEmptyInstallReportsPolicyBlockerForEmptySchema(t *testing.T) {
	verification, err := VerifyEmptyInstall(context.Background(), fakeSchemaInspector{})
	if err != nil {
		t.Fatalf("VerifyEmptyInstall() returned an error: %v", err)
	}

	if verification.Status != "blocked" {
		t.Fatalf("expected blocked status while baseline application is disabled: %#v", verification)
	}
	if !verification.SchemaEmpty || len(verification.ExistingTables) != 0 {
		t.Fatalf("empty schema was not detected correctly: %#v", verification)
	}
	if len(verification.Blockers) != 1 {
		t.Fatalf("expected one policy blocker: %#v", verification)
	}
}

func TestVerifyEmptyInstallFailsForNonEmptySchema(t *testing.T) {
	verification, err := VerifyEmptyInstall(context.Background(), fakeSchemaInspector{tables: []string{"users", "projects"}})
	if err != nil {
		t.Fatalf("VerifyEmptyInstall() returned an error: %v", err)
	}

	if verification.Status != "failed" {
		t.Fatalf("expected failed status for non-empty schema: %#v", verification)
	}
	if verification.SchemaEmpty {
		t.Fatalf("non-empty schema was reported as empty: %#v", verification)
	}
	if len(verification.ExistingTables) != 2 {
		t.Fatalf("existing table inventory missing: %#v", verification)
	}
}

func TestVerifyEmptyInstallReturnsSafeDatabaseDiagnostics(t *testing.T) {
	_, err := VerifyEmptyInstall(context.Background(), fakeSchemaInspector{
		err: errors.New("password=secret dsn detail"),
	})

	if err == nil {
		t.Fatal("expected schema inspection failure")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "password") {
		t.Fatalf("empty-install diagnostic leaked sensitive details: %v", err)
	}
}

type fakeSchemaInspector struct {
	tables []string
	err    error
}

func (inspector fakeSchemaInspector) UserTables(context.Context) ([]string, error) {
	if inspector.err != nil {
		return nil, errors.New("inspect empty install schema: database unavailable")
	}
	return inspector.tables, nil
}
