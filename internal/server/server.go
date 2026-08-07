// Package server owns the bounded HTTP server lifecycle.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/idelium/idelium-api-go/internal/config"
)

// New creates an HTTP server with every externally configurable timeout bound.
func New(runtimeConfig config.HTTPConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              runtimeConfig.Address,
		Handler:           handler,
		ReadHeaderTimeout: runtimeConfig.ReadHeaderTimeout,
		ReadTimeout:       runtimeConfig.ReadTimeout,
		WriteTimeout:      runtimeConfig.WriteTimeout,
		IdleTimeout:       runtimeConfig.IdleTimeout,
	}
}

// Serve runs until the listener fails or the lifecycle context is cancelled.
func Serve(
	ctx context.Context,
	httpServer *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.Serve(listener)
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("Graceful shutdown started")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancelShutdown()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		_ = httpServer.Close()
		return err
	}

	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("Graceful shutdown completed")
	return nil
}
