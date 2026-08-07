package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/idelium/idelium-api-go/internal/app"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	mysqlpersistence "github.com/idelium/idelium-api-go/internal/persistence/mysql"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	runtimeConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	database, err := mysqlpersistence.Open(runtimeConfig.Database)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer database.Close()

	startupContext, cancelStartup := context.WithTimeout(context.Background(), runtimeConfig.Database.ConnectTimeout)
	defer cancelStartup()
	if err := mysqlpersistence.Check(startupContext, database); err != nil {
		return fmt.Errorf("database startup check failed: %w", err)
	}

	server := &http.Server{
		Addr:              runtimeConfig.HTTP.Address,
		Handler:           app.NewRouter(logger, databaseChecker{database: database}, buildinfo.Current()),
		ReadHeaderTimeout: runtimeConfig.HTTP.ReadHeaderTimeout,
		ReadTimeout:       runtimeConfig.HTTP.ReadTimeout,
		WriteTimeout:      runtimeConfig.HTTP.WriteTimeout,
		IdleTimeout:       runtimeConfig.HTTP.IdleTimeout,
	}

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(
			"Idelium API Go started",
			"address", runtimeConfig.HTTP.Address,
			"environment", runtimeConfig.Environment,
			"version", buildinfo.Version,
			"commit", buildinfo.Commit,
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case serverErr := <-serverErrors:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", serverErr)
		}
		return nil
	case <-shutdownSignal.Done():
		logger.Info("Graceful shutdown started")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), runtimeConfig.HTTP.ShutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("Graceful shutdown completed")
	return nil
}

type databaseChecker struct {
	database *sql.DB
}

func (checker databaseChecker) Check(ctx context.Context) error {
	return mysqlpersistence.Check(ctx, checker.database)
}
