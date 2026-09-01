package mysql

import (
	"testing"
	"time"
)

func TestRecalculateParallelRunWorkersExpiryBoundaryAndClockSkew(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	workers := map[string]any{
		"exact":  map[string]any{"status": "running", "leaseExpiresAt": now.Format(time.RFC3339Nano)},
		"past":   map[string]any{"status": "running", "leaseExpiresAt": now.Add(-time.Nanosecond).Format(time.RFC3339Nano)},
		"future": map[string]any{"status": "running", "leaseExpiresAt": now.Add(time.Nanosecond).Format(time.RFC3339Nano)},
	}
	counters, summary := recalculateParallelRunWorkers(workers, now)
	if counters.active != 2 || counters.lost != 1 {
		t.Fatalf("unexpected counters: %+v", counters)
	}
	if workers["exact"].(map[string]any)["status"] != "running" {
		t.Fatal("lease expiring exactly at now must remain active")
	}
	if workers["past"].(map[string]any)["status"] != "lost" {
		t.Fatal("expired lease must converge to lost")
	}
	if len(summary) != 3 || summary[0]["workerId"] != "exact" {
		t.Fatalf("summary must be deterministic: %#v", summary)
	}
}
