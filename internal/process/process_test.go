package process

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
)

func TestRunInvokesRegisteredOwningDomain(t *testing.T) {
	called := false
	err := Run(
		context.Background(),
		Registration{
			Name:         "worker",
			OwningDomain: "synthetic-test-domain",
			Handler: func(context.Context) error {
				called = true
				return nil
			},
		},
		buildinfo.Current(),
		testLogger(),
	)
	if err != nil || !called {
		t.Fatalf("registered process did not run: called=%v error=%v", called, err)
	}
}

func TestRunRefusesSkeletonWithoutOwningDomain(t *testing.T) {
	err := Run(
		context.Background(),
		Registration{Name: "worker"},
		buildinfo.Current(),
		testLogger(),
	)
	if !errors.Is(err, ErrOwningDomainNotRegistered) {
		t.Fatalf("expected ownership error, got %v", err)
	}
	if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "token") {
		t.Fatalf("unsafe process diagnostic: %v", err)
	}
}

func TestRunRefusesOwningDomainWithoutHandler(t *testing.T) {
	err := Run(
		context.Background(),
		Registration{Name: "migrate", OwningDomain: "schema"},
		buildinfo.Current(),
		testLogger(),
	)
	if !errors.Is(err, ErrHandlerNotRegistered) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestRunTreatsLifecycleCancellationAsCleanStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(
		ctx,
		Registration{
			Name:         "worker",
			OwningDomain: "synthetic-test-domain",
			Handler: func(ctx context.Context) error {
				return ctx.Err()
			},
		},
		buildinfo.Current(),
		testLogger(),
	)
	if err != nil {
		t.Fatalf("lifecycle cancellation was not a clean stop: %v", err)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}
