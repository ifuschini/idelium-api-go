package integrations

import (
	"context"
	"errors"
)

type QueueDrainInspector interface {
	QueuedJobs(context.Context, string) (int64, error)
	FailedJobs(context.Context) (int64, error)
	PendingDomainJobs(context.Context) (map[string]int64, error)
}

type QueueDrainVerification struct {
	Status             string           `json:"status"`
	QueueDriver        string           `json:"queue_driver"`
	WorkersStopped     bool             `json:"workers_stopped"`
	QueuedJobs         int64            `json:"queued_jobs"`
	FailedJobs         int64            `json:"failed_jobs"`
	PendingDomainJobs  map[string]int64 `json:"pending_domain_jobs"`
	TotalBlockingJobs  int64            `json:"total_blocking_jobs"`
	Blockers           []string         `json:"blockers,omitempty"`
	VerifiedConditions []string         `json:"verified_conditions"`
}

func VerifyQueueDrain(ctx context.Context, inspector QueueDrainInspector, queueDriver string, workersStopped bool) (QueueDrainVerification, error) {
	if inspector == nil {
		return QueueDrainVerification{}, errors.New("queue drain inspector is required")
	}
	if queueDriver != "sync" && queueDriver != "database" {
		return QueueDrainVerification{}, errors.New("Laravel queue driver must be sync or database")
	}
	queued, err := inspector.QueuedJobs(ctx, queueDriver)
	if err != nil {
		return QueueDrainVerification{}, err
	}
	failed, err := inspector.FailedJobs(ctx)
	if err != nil {
		return QueueDrainVerification{}, err
	}
	domain, err := inspector.PendingDomainJobs(ctx)
	if err != nil {
		return QueueDrainVerification{}, err
	}
	verification := QueueDrainVerification{
		QueueDriver:       queueDriver,
		WorkersStopped:    workersStopped,
		QueuedJobs:        queued,
		FailedJobs:        failed,
		PendingDomainJobs: domain,
		VerifiedConditions: []string{
			"Laravel queue workers are explicitly confirmed stopped",
			"queue and failed-job tables were inspected using aggregate counts only",
			"pending integration deliveries and result exports were inspected globally",
			"no tenant identifiers, serialized payloads, exceptions, or credentials were read",
		},
	}
	verification.TotalBlockingJobs = queued + failed
	for _, count := range domain {
		verification.TotalBlockingJobs += count
	}
	if !workersStopped {
		verification.Blockers = append(verification.Blockers, "Laravel queue workers are not confirmed stopped")
	}
	if verification.TotalBlockingJobs != 0 {
		verification.Blockers = append(verification.Blockers, "Laravel queue or domain work remains pending")
	}
	if len(verification.Blockers) == 0 {
		verification.Status = "ready"
	} else {
		verification.Status = "blocked"
	}
	return verification, nil
}
