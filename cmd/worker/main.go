package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/integrations"
	mysqlpersistence "github.com/idelium/idelium-api-go/internal/persistence/mysql"
	"github.com/idelium/idelium-api-go/internal/process"
)

func main() {
	showVersion := flag.Bool("version", false, "print safe build identity and exit")
	flag.Parse()
	if *showVersion {
		if err := json.NewEncoder(os.Stdout).Encode(buildinfo.Current()); err != nil {
			fmt.Fprintln(os.Stderr, "worker could not encode build identity")
			os.Exit(1)
		}
		return
	}
	if strings.ToLower(strings.TrimSpace(os.Getenv("IDELIUM_INTEGRATION_WORKER_ENABLED"))) != "true" {
		fmt.Fprintln(os.Stderr, "worker is disabled until the Laravel queue drain is verified")
		os.Exit(1)
	}
	runtimeConfig, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker configuration is invalid")
		os.Exit(1)
	}
	database, err := mysqlpersistence.Open(runtimeConfig.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "worker could not initialize its database")
		os.Exit(1)
	}
	defer database.Close()
	applicationKey, err := integrations.ApplicationKeyFromEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	lease, err := mysqlpersistence.AcquireIntegrationWorkerLease(ctx, database)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = lease.Release(context.Background()) }()
	if err := process.Run(
		ctx,
		process.Registration{
			Name:         "worker",
			OwningDomain: "integration-deliveries",
			Handler: (integrations.Worker{
				Store:          mysqlpersistence.NewBrowserAuthRepository(database),
				ApplicationKey: applicationKey,
				Logger:         logger,
				PollInterval:   time.Second,
			}).Run,
		},
		buildinfo.Current(),
		logger,
	); err != nil {
		logger.Error("Worker is not runnable", "error", err)
		os.Exit(1)
	}
}
