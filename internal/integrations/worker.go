package integrations

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

var ErrNoPendingDelivery = errors.New("no pending integration delivery")

type PendingDeliveryStore interface {
	DeliveryStore
	NextDispatchID(context.Context, time.Time) (int64, error)
}

type Worker struct {
	Store          PendingDeliveryStore
	ApplicationKey []byte
	Logger         *slog.Logger
	PollInterval   time.Duration
	Dispatcher     Dispatcher
}

func (worker Worker) Run(ctx context.Context) error {
	if worker.Store == nil {
		return errors.New("integration delivery store is required")
	}
	if len(worker.ApplicationKey) != 32 {
		return ErrInvalidApplicationKey
	}
	if worker.Logger == nil {
		worker.Logger = slog.Default()
	}
	if worker.PollInterval <= 0 {
		worker.PollInterval = time.Second
	}
	worker.Dispatcher.Store = worker.Store
	worker.Dispatcher.ApplicationKey = worker.ApplicationKey
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		id, err := worker.Store.NextDispatchID(ctx, time.Now().UTC())
		if errors.Is(err, ErrNoPendingDelivery) {
			timer.Reset(worker.PollInterval)
			continue
		}
		if err != nil {
			return err
		}
		if err := worker.Dispatcher.Dispatch(ctx, id); err != nil && !errors.Is(err, ErrDeliveryNotFound) {
			worker.Logger.Error("Integration delivery attempt failed safely", "error", "delivery dispatch failed")
		}
		timer.Reset(0)
	}
}
