package tenant

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestAssertTenantIsolationAcceptsOwnerAndDeniesForeignTenant(t *testing.T) {
	owner := NewScope(t, "fixture-tenant-owner", "fixture-actor-owner")
	foreign := NewScope(t, "fixture-tenant-foreign", "fixture-actor-foreign")
	lookup := func(_ context.Context, scope Scope, _ string) (bool, error) {
		return scope.TenantID == owner.TenantID, nil
	}

	AssertTenantIsolation(t, context.Background(), owner, foreign, "fixture-resource", lookup)
}

func TestAssertOwnedRecordsAcceptsOnlyActiveTenantRecords(t *testing.T) {
	scope := NewScope(t, "fixture-tenant-alpha", "fixture-actor-alpha")

	AssertOwnedRecords(t, scope, []OwnedRecord{
		{ID: "fixture-resource-1", TenantID: "fixture-tenant-alpha"},
		{ID: "fixture-resource-2", TenantID: "fixture-tenant-alpha"},
	})
}

func TestAssertOwnerCanReadReportsLookupFailuresWithoutValues(t *testing.T) {
	recorder := &testingRecorder{}
	owner := Scope{TenantID: "fixture-tenant-owner", ActorID: "fixture-actor-owner"}
	lookup := func(context.Context, Scope, string) (bool, error) {
		return false, errors.New("database password leaked")
	}

	AssertOwnerCanRead(recorder, context.Background(), owner, "fixture-resource", lookup)

	if !recorder.failed {
		t.Fatal("expected the helper to fail on repository errors")
	}
	if recorder.message == "" || containsSensitiveValue(recorder.message) {
		t.Fatalf("helper diagnostic leaked sensitive details: %q", recorder.message)
	}
}

func TestAssertForeignTenantCannotReadRejectsSharedTenantSetup(t *testing.T) {
	recorder := &testingRecorder{}
	scope := Scope{TenantID: "fixture-tenant-shared", ActorID: "fixture-actor-one"}
	lookup := func(context.Context, Scope, string) (bool, error) { return false, nil }

	AssertForeignTenantCannotRead(recorder, context.Background(), scope, scope, "fixture-resource", lookup)

	if !recorder.failed {
		t.Fatal("expected the helper to reject ambiguous cross-tenant setup")
	}
}

type testingRecorder struct {
	failed  bool
	message string
}

func (recorder *testingRecorder) Helper() {}

func (recorder *testingRecorder) Fatal(args ...any) {
	recorder.failed = true
	recorder.message = fmt.Sprint(args...)
}

func (recorder *testingRecorder) Fatalf(format string, args ...any) {
	recorder.failed = true
	recorder.message = fmt.Sprintf(format, args...)
}

func containsSensitiveValue(value string) bool {
	return value == "database password leaked"
}
