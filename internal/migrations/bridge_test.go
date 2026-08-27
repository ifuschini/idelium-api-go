package migrations

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestReviewedBaselineBridgePlanRequiresExplicitConfirmation(t *testing.T) {
	_, err := ReviewedBaselineBridgePlan(BridgeOptions{Batch: 1})

	if err == nil {
		t.Fatal("bridge plan accepted missing baseline confirmation")
	}
	if !strings.Contains(err.Error(), "confirmed baseline ID is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestReviewedBaselineBridgePlanListsSafeMigrationMarkers(t *testing.T) {
	plan, err := ReviewedBaselineBridgePlan(BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	})
	if err != nil {
		t.Fatalf("ReviewedBaselineBridgePlan() returned an error: %v", err)
	}

	if plan.BaselineID != "go-baseline-2026-08-25" {
		t.Fatalf("unexpected baseline ID %q", plan.BaselineID)
	}
	if plan.MigrationCount != 68 || len(plan.Migrations) != 68 {
		t.Fatalf("unexpected migration count in plan: %#v", plan)
	}
	if plan.Batch != 67 || plan.Table != "migrations" {
		t.Fatalf("unexpected target table or batch: %#v", plan)
	}
	if !plan.RequiresExplicitRun {
		t.Fatal("bridge plan must require an explicit execute flag")
	}
	serialized := strings.ToLower(strings.Join(plan.Migrations, "\n"))
	for _, unsafe := range []string{"password=", "authorization:", "cookie:", "bearer "} {
		if strings.Contains(serialized, unsafe) {
			t.Fatalf("bridge plan exposed unsafe marker %q", unsafe)
		}
	}
}

func TestMarkReviewedBaselineAppliedIsIdempotent(t *testing.T) {
	execer := &fakeBridgeExecer{rows: []int64{1, 0, 1}}

	result, err := MarkReviewedBaselineApplied(context.Background(), execer, BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	})
	if err != nil {
		t.Fatalf("MarkReviewedBaselineApplied() returned an error: %v", err)
	}

	if result.Applied != 2 || result.Skipped != 66 {
		t.Fatalf("unexpected applied/skipped counters: %#v", result)
	}
	if len(execer.queries) != 68 {
		t.Fatalf("expected one statement per migration, got %d", len(execer.queries))
	}
	for _, query := range execer.queries {
		if query != bridgeStatementTemplate {
			t.Fatalf("unexpected bridge statement: %s", query)
		}
	}
}

func TestMarkReviewedBaselineAppliedReturnsSafeDatabaseDiagnostics(t *testing.T) {
	execer := &fakeBridgeExecer{err: errors.New("password=secret dsn detail")}

	_, err := MarkReviewedBaselineApplied(context.Background(), execer, BridgeOptions{
		ConfirmBaselineID: "go-baseline-2026-08-25",
		Batch:             67,
	})

	if err == nil {
		t.Fatal("expected database failure")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "password") {
		t.Fatalf("database diagnostic leaked sensitive details: %v", err)
	}
}

type fakeBridgeExecer struct {
	rows    []int64
	err     error
	queries []string
}

func (execer *fakeBridgeExecer) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	if execer.err != nil {
		return nil, execer.err
	}
	execer.queries = append(execer.queries, query)
	index := len(execer.queries) - 1
	if index < len(execer.rows) {
		return fakeBridgeResult(execer.rows[index]), nil
	}
	return fakeBridgeResult(0), nil
}

type fakeBridgeResult int64

func (result fakeBridgeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (result fakeBridgeResult) RowsAffected() (int64, error) {
	return int64(result), nil
}
