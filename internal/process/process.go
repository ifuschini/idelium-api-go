// Package process provides a fail-closed lifecycle for non-HTTP binaries.
package process

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
)

var (
	// ErrOwningDomainNotRegistered prevents a skeleton from running prematurely.
	ErrOwningDomainNotRegistered = errors.New("owning domain is not registered")
	// ErrHandlerNotRegistered prevents a process without work from reporting success.
	ErrHandlerNotRegistered = errors.New("process handler is not registered")
)

// Registration binds a process to the domain and handler it exclusively owns.
type Registration struct {
	Name         string
	OwningDomain string
	Handler      func(context.Context) error
}

// Run validates ownership before invoking a cancellable process handler.
func Run(
	ctx context.Context,
	registration Registration,
	info buildinfo.Info,
	logger *slog.Logger,
) error {
	if registration.Name == "" {
		return errors.New("process name is required")
	}
	if registration.OwningDomain == "" {
		return fmt.Errorf("%s: %w", registration.Name, ErrOwningDomainNotRegistered)
	}
	if registration.Handler == nil {
		return fmt.Errorf("%s: %w", registration.Name, ErrHandlerNotRegistered)
	}

	logger.Info(
		"Process started",
		"process", registration.Name,
		"owning_domain", registration.OwningDomain,
		"version", info.Version,
		"commit", info.Commit,
	)
	if err := registration.Handler(ctx); err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			logger.Info("Process stopped", "process", registration.Name)
			return nil
		}
		return fmt.Errorf("%s handler failed: %w", registration.Name, err)
	}
	logger.Info("Process completed", "process", registration.Name)
	return nil
}
