package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestMigratePlanPrintsReviewedBaseline(t *testing.T) {
	if os.Getenv("IDELIUM_MIGRATE_HELPER") == "1" {
		os.Args = []string{"migrate", "--plan"}
		main()
		os.Exit(0)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=TestMigratePlanPrintsReviewedBaseline")
	command.Env = append(os.Environ(), "IDELIUM_MIGRATE_HELPER=1")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("migrate --plan failed: %v", err)
	}

	var plan struct {
		BaselineID                   string `json:"baseline_id"`
		MigrationCount               int    `json:"migration_count"`
		AggregateSHA256              string `json:"aggregate_sha256"`
		GoBaselineApplicationEnabled bool   `json:"go_baseline_application_enabled"`
		DualWritesAllowed            bool   `json:"dual_writes_allowed"`
	}
	if err := json.Unmarshal(output, &plan); err != nil {
		t.Fatalf("migrate --plan emitted invalid JSON %q: %v", output, err)
	}
	if plan.BaselineID != "go-baseline-2026-08-25" {
		t.Fatalf("unexpected baseline ID %q", plan.BaselineID)
	}
	if plan.MigrationCount != 66 {
		t.Fatalf("unexpected migration count %d", plan.MigrationCount)
	}
	if plan.AggregateSHA256 == "" {
		t.Fatal("plan did not include aggregate checksum")
	}
	if plan.GoBaselineApplicationEnabled || plan.DualWritesAllowed {
		t.Fatalf("plan is unsafe before cutover: %#v", plan)
	}
}
