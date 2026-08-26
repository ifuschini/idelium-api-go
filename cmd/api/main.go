package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/idelium/idelium-api-go/internal/app"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	mysqlpersistence "github.com/idelium/idelium-api-go/internal/persistence/mysql"
	"github.com/idelium/idelium-api-go/internal/server"
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

	httpServer := server.New(
		runtimeConfig.HTTP,
		app.NewRouter(
			logger,
			databaseChecker{database: database},
			buildinfo.Current(),
			mysqlpersistence.NewPlatformCatalogRepository(database),
			mysqlpersistence.NewLegacyKeyRepository(database),
			mysqlpersistence.NewCLITestCycleRepository(database),
			mysqlpersistence.NewCLIPerformedCycleRepository(database),
			mysqlpersistence.NewCLITestRepository(database),
			mysqlpersistence.NewCLIPerformedTestRepository(database),
			mysqlpersistence.NewCLIPerformedStepRepository(database),
			mysqlpersistence.NewCLIStepRepository(database),
			mysqlpersistence.NewCLIPluginRepository(database),
			mysqlpersistence.NewCLIEnvironmentRepository(database),
		),
	)
	listener, err := net.Listen("tcp", runtimeConfig.HTTP.Address)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer listener.Close()

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	logger.Info(
		"Idelium API Go started",
		"address", runtimeConfig.HTTP.Address,
		"environment", runtimeConfig.Environment,
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
	)
	if err := server.Serve(
		shutdownSignal,
		httpServer,
		listener,
		runtimeConfig.HTTP.ShutdownTimeout,
		logger,
	); err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

type databaseChecker struct {
	database *sql.DB
}

func (checker databaseChecker) Check(ctx context.Context) error {
	return mysqlpersistence.Check(ctx, checker.database)
}
