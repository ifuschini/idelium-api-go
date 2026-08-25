package migrations

import (
	"strings"
	"testing"
)

func TestReviewedBaselineManifestIsSafeAndConsistent(t *testing.T) {
	manifest, err := ReviewedBaseline()
	if err != nil {
		t.Fatalf("ReviewedBaseline() returned an error: %v", err)
	}

	if manifest.BaselineID != "go-baseline-2026-08-25" {
		t.Fatalf("unexpected baseline ID %q", manifest.BaselineID)
	}
	if manifest.MigrationCount != 66 {
		t.Fatalf("unexpected migration count %d", manifest.MigrationCount)
	}
	if !manifest.HandoverPolicy.LaravelRemainsSchemaOwner {
		t.Fatal("Laravel must remain the schema owner until cutover gates pass")
	}
	if manifest.HandoverPolicy.GoBaselineApplicationEnabled {
		t.Fatal("Go baseline application must be disabled in the reviewed baseline slice")
	}
	if manifest.HandoverPolicy.DualWritesAllowed {
		t.Fatal("dual writes must not be allowed")
	}

	serialized := manifest.AggregateSHA256 + manifest.Redaction
	for _, migration := range manifest.Migrations {
		serialized += migration.File + migration.SHA256
	}
	for _, unsafe := range []string{"password=", "authorization:", "cookie:", "bearer "} {
		if strings.Contains(strings.ToLower(serialized), unsafe) {
			t.Fatalf("baseline manifest contains unsafe data marker %q", unsafe)
		}
	}
}

func TestReviewedBaselinePlanDoesNotExposePerMigrationDetails(t *testing.T) {
	plan, err := ReviewedBaselinePlan()
	if err != nil {
		t.Fatalf("ReviewedBaselinePlan() returned an error: %v", err)
	}

	if plan.BaselineID == "" || plan.AggregateSHA256 == "" {
		t.Fatalf("plan lost baseline identity: %#v", plan)
	}
	if plan.MigrationCount != 66 {
		t.Fatalf("unexpected plan migration count %d", plan.MigrationCount)
	}
	if plan.GoBaselineApplicationEnabled {
		t.Fatal("plan must not report baseline application as enabled")
	}
	if plan.DualWritesAllowed {
		t.Fatal("plan must not report dual writes as allowed")
	}
}
