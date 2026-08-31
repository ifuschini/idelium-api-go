package integrations

import (
	"context"
	"errors"
	"testing"
)

type queueDrainInspectorStub struct {
	queued int64
	failed int64
	domain map[string]int64
	err    error
}

func (stub queueDrainInspectorStub) QueuedJobs(context.Context, string) (int64, error) {
	return stub.queued, stub.err
}
func (stub queueDrainInspectorStub) FailedJobs(context.Context) (int64, error) {
	return stub.failed, stub.err
}
func (stub queueDrainInspectorStub) PendingDomainJobs(context.Context) (map[string]int64, error) {
	return stub.domain, stub.err
}

func TestVerifyQueueDrainRequiresStoppedWorkersAndNoBacklog(t *testing.T) {
	ready, err := VerifyQueueDrain(context.Background(), queueDrainInspectorStub{domain: map[string]int64{"integration_deliveries": 0, "result_exports": 0}}, "sync", true)
	if err != nil || ready.Status != "ready" || ready.TotalBlockingJobs != 0 {
		t.Fatalf("unexpected ready verification: %#v err=%v", ready, err)
	}
	blocked, err := VerifyQueueDrain(context.Background(), queueDrainInspectorStub{queued: 2, failed: 1, domain: map[string]int64{"integration_deliveries": 3}}, "database", false)
	if err != nil || blocked.Status != "blocked" || blocked.TotalBlockingJobs != 6 || len(blocked.Blockers) != 2 {
		t.Fatalf("unexpected blocked verification: %#v err=%v", blocked, err)
	}
}

func TestVerifyQueueDrainRejectsUnsafeConfigurationAndDiagnostics(t *testing.T) {
	if _, err := VerifyQueueDrain(context.Background(), queueDrainInspectorStub{}, "redis", true); err == nil {
		t.Fatal("expected unsupported queue driver rejection")
	}
	sentinel := errors.New("database unavailable")
	_, err := VerifyQueueDrain(context.Background(), queueDrainInspectorStub{err: sentinel}, "sync", true)
	if !errors.Is(err, sentinel) || err.Error() != "database unavailable" {
		t.Fatalf("unexpected safe inspector failure: %v", err)
	}
}
