package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
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
	if plan.MigrationCount != 69 {
		t.Fatalf("unexpected migration count %d", plan.MigrationCount)
	}
	if plan.AggregateSHA256 == "" {
		t.Fatal("plan did not include aggregate checksum")
	}
	if plan.GoBaselineApplicationEnabled || plan.DualWritesAllowed {
		t.Fatalf("plan is unsafe before cutover: %#v", plan)
	}
}

func TestMigrateBridgePlanPrintsSafeDryRun(t *testing.T) {
	if os.Getenv("IDELIUM_MIGRATE_BRIDGE_HELPER") == "1" {
		os.Args = []string{
			"migrate",
			"--mark-laravel-baseline-applied",
			"--confirm-baseline-id",
			"go-baseline-2026-08-25",
			"--batch",
			"67",
		}
		main()
		os.Exit(0)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=TestMigrateBridgePlanPrintsSafeDryRun")
	command.Env = append(os.Environ(), "IDELIUM_MIGRATE_BRIDGE_HELPER=1")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("migrate bridge dry-run failed: %v", err)
	}

	var plan struct {
		BaselineID          string   `json:"baseline_id"`
		MigrationCount      int      `json:"migration_count"`
		Batch               int      `json:"batch"`
		Table               string   `json:"table"`
		Migrations          []string `json:"migrations"`
		RequiresExplicitRun bool     `json:"requires_explicit_run"`
	}
	if err := json.Unmarshal(output, &plan); err != nil {
		t.Fatalf("migrate bridge dry-run emitted invalid JSON %q: %v", output, err)
	}
	if plan.BaselineID != "go-baseline-2026-08-25" || plan.MigrationCount != 69 || plan.Batch != 67 {
		t.Fatalf("unexpected bridge plan: %#v", plan)
	}
	if plan.Table != "migrations" || !plan.RequiresExplicitRun {
		t.Fatalf("bridge plan lost safety metadata: %#v", plan)
	}
}

func TestMigrateBridgePlanRejectsWrongConfirmation(t *testing.T) {
	if os.Getenv("IDELIUM_MIGRATE_BRIDGE_VALIDATION_HELPER") == "1" {
		os.Args = []string{
			"migrate",
			"--mark-laravel-baseline-applied",
			"--confirm-baseline-id",
			"wrong-baseline",
			"--batch",
			"67",
		}
		main()
		os.Exit(0)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=TestMigrateBridgePlanRejectsWrongConfirmation")
	command.Env = append(os.Environ(), "IDELIUM_MIGRATE_BRIDGE_VALIDATION_HELPER=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("migrate bridge accepted wrong baseline confirmation")
	}
	if strings.Contains(string(output), "password") || strings.Contains(string(output), "secret") {
		t.Fatalf("bridge validation diagnostic leaked sensitive text: %s", string(output))
	}
	if !strings.Contains(string(output), "confirmed baseline ID does not match") {
		t.Fatalf("unexpected bridge validation output: %s", string(output))
	}
}
