package tenant

import (
	"context"
	"strings"
)

// TestingT is the minimal testing contract required by tenant assertions.
type TestingT interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// Scope identifies the synthetic tenant and actor used by an isolation test.
type Scope struct {
	TenantID string
	ActorID  string
}

// OwnedRecord is the minimal ownership projection expected from repository tests.
type OwnedRecord struct {
	ID       string
	TenantID string
}

// LookupFunc resolves one tenant-owned resource through the same authorization
// path that production code uses.
type LookupFunc func(context.Context, Scope, string) (bool, error)

// NewScope creates a tenant scope and rejects ambiguous test identities.
func NewScope(t TestingT, tenantID string, actorID string) Scope {
	t.Helper()
	if strings.TrimSpace(tenantID) == "" {
		t.Fatal("tenant test scope requires a synthetic tenant identifier")
	}
	if strings.TrimSpace(actorID) == "" {
		t.Fatal("tenant test scope requires a synthetic actor identifier")
	}
	return Scope{TenantID: tenantID, ActorID: actorID}
}

// AssertDistinctScopes fails when a negative test accidentally uses the same tenant.
func AssertDistinctScopes(t TestingT, owner Scope, foreign Scope) {
	t.Helper()
	if owner.TenantID == "" || foreign.TenantID == "" {
		t.Fatal("tenant isolation test scopes must be initialized")
	}
	if owner.TenantID == foreign.TenantID {
		t.Fatal("cross-tenant negative checks require different tenant scopes")
	}
}

// AssertOwnedRecords verifies that a list query only returns records for the active tenant.
func AssertOwnedRecords(t TestingT, scope Scope, records []OwnedRecord) {
	t.Helper()
	if strings.TrimSpace(scope.TenantID) == "" {
		t.Fatal("tenant isolation assertion requires an initialized scope")
	}
	for _, record := range records {
		if record.TenantID != scope.TenantID {
			t.Fatal("tenant-owned list query returned a record from another tenant")
		}
	}
}

// AssertOwnerCanRead verifies that the owner tenant can still access its resource.
func AssertOwnerCanRead(
	t TestingT,
	ctx context.Context,
	owner Scope,
	resourceID string,
	lookup LookupFunc,
) {
	t.Helper()
	if lookup == nil {
		t.Fatal("tenant lookup function is required")
	}
	found, err := lookup(ctx, owner, resourceID)
	if err != nil {
		t.Fatalf("owner lookup failed with %T", err)
	}
	if !found {
		t.Fatal("owner tenant could not read its own resource")
	}
}

// AssertForeignTenantCannotRead verifies that another tenant cannot resolve a resource.
func AssertForeignTenantCannotRead(
	t TestingT,
	ctx context.Context,
	owner Scope,
	foreign Scope,
	resourceID string,
	lookup LookupFunc,
) {
	t.Helper()
	AssertDistinctScopes(t, owner, foreign)
	if lookup == nil {
		t.Fatal("tenant lookup function is required")
	}
	found, err := lookup(ctx, foreign, resourceID)
	if err != nil {
		t.Fatalf("foreign tenant lookup failed with %T", err)
	}
	if found {
		t.Fatal("foreign tenant resolved a resource owned by another tenant")
	}
}

// AssertTenantIsolation verifies both the owner success path and the cross-tenant denial path.
func AssertTenantIsolation(
	t TestingT,
	ctx context.Context,
	owner Scope,
	foreign Scope,
	resourceID string,
	lookup LookupFunc,
) {
	t.Helper()
	AssertOwnerCanRead(t, ctx, owner, resourceID, lookup)
	AssertForeignTenantCannotRead(t, ctx, owner, foreign, resourceID, lookup)
}
