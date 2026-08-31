package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/config"
	"github.com/idelium/idelium-api-go/internal/integrations"
	"github.com/idelium/idelium-api-go/internal/migrations"
	mysqlpersistence "github.com/idelium/idelium-api-go/internal/persistence/mysql"
	"github.com/idelium/idelium-api-go/internal/process"
)

func main() {
	showVersion := flag.Bool("version", false, "print safe build identity and exit")
	showPlan := flag.Bool("plan", false, "print the reviewed migration baseline plan and exit")
	markLaravelBaselineApplied := flag.Bool("mark-laravel-baseline-applied", false, "print or execute the Laravel baseline bridge marker plan")
	verifyEmptyInstall := flag.Bool("verify-empty-install", false, "verify whether the configured database is ready for an empty Go migration install")
	verifyLaravelUpgrade := flag.Bool("verify-laravel-upgrade", false, "verify whether the configured database can upgrade from the last Laravel-owned release")
	verifyLaravelQueueDrain := flag.Bool("verify-laravel-queue-drain", false, "verify aggregate Laravel queue drain conditions before moving ownership")
	laravelQueueDriver := flag.String("laravel-queue-driver", "", "Laravel queue driver being drained: sync or database")
	confirmLaravelWorkersStopped := flag.Bool("confirm-laravel-workers-stopped", false, "confirm that all Laravel queue workers are stopped")
	fromLaravelRelease := flag.String("from-laravel-release", "", "source Laravel release identifier for upgrade verification")
	confirmBaselineID := flag.String("confirm-baseline-id", "", "reviewed baseline ID required before marking Laravel migrations as applied")
	batch := flag.Int("batch", 0, "Laravel migrations table batch number for bridge markers")
	execute := flag.Bool("execute", false, "execute the bridge marker plan against the configured database")
	flag.Parse()
	if *showVersion {
		if err := json.NewEncoder(os.Stdout).Encode(buildinfo.Current()); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not encode build identity")
			os.Exit(1)
		}
		return
	}
	if *showPlan {
		plan, err := migrations.ReviewedBaselinePlan()
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not load the reviewed baseline plan")
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(plan); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not encode the reviewed baseline plan")
			os.Exit(1)
		}
		return
	}
	if *markLaravelBaselineApplied {
		options := migrations.BridgeOptions{
			ConfirmBaselineID: *confirmBaselineID,
			Batch:             *batch,
		}
		if !*execute {
			plan, err := migrations.ReviewedBaselineBridgePlan(options)
			if err != nil {
				fmt.Fprintln(os.Stderr, safeBridgeError(err))
				os.Exit(1)
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(plan); err != nil {
				fmt.Fprintln(os.Stderr, "migrate could not encode the Laravel baseline bridge plan")
				os.Exit(1)
			}
			return
		}

		runtimeConfig, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		database, err := mysqlpersistence.Open(runtimeConfig.Database)
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not initialize the database bridge")
			os.Exit(1)
		}
		defer database.Close()

		ctx := context.Background()
		transaction, err := database.BeginTx(ctx, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not start the database bridge transaction")
			os.Exit(1)
		}
		result, err := migrations.MarkReviewedBaselineApplied(ctx, transaction, options)
		if err != nil {
			_ = transaction.Rollback()
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		if err := transaction.Commit(); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not commit the database bridge transaction")
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not encode the Laravel baseline bridge result")
			os.Exit(1)
		}
		return
	}
	if *verifyEmptyInstall {
		runtimeConfig, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		database, err := mysqlpersistence.Open(runtimeConfig.Database)
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not initialize the database bridge")
			os.Exit(1)
		}
		defer database.Close()

		verification, err := migrations.VerifyEmptyInstall(
			context.Background(),
			migrations.NewMySQLSchemaInspector(database, runtimeConfig.Database.Name),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(verification); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not encode the empty-install verification result")
			os.Exit(1)
		}
		if verification.Status != "ready" {
			os.Exit(2)
		}
		return
	}
	if *verifyLaravelUpgrade {
		runtimeConfig, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		database, err := mysqlpersistence.Open(runtimeConfig.Database)
		if err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not initialize the database bridge")
			os.Exit(1)
		}
		defer database.Close()

		verification, err := migrations.VerifyLaravelUpgrade(
			context.Background(),
			migrations.NewMySQLMigrationTableInspector(database),
			*fromLaravelRelease,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, safeBridgeError(err))
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(verification); err != nil {
			fmt.Fprintln(os.Stderr, "migrate could not encode the Laravel upgrade verification result")
			os.Exit(1)
		}
		if verification.Status != "ready" {
			os.Exit(2)
		}
		return
	}
	if *verifyLaravelQueueDrain {
		runtimeConfig, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "queue drain verification configuration is invalid")
			os.Exit(1)
		}
		database, err := mysqlpersistence.Open(runtimeConfig.Database)
		if err != nil {
			fmt.Fprintln(os.Stderr, "queue drain verification could not initialize the database")
			os.Exit(1)
		}
		defer database.Close()
		verification, err := integrations.VerifyQueueDrain(
			context.Background(),
			mysqlpersistence.NewBrowserAuthRepository(database),
			*laravelQueueDriver,
			*confirmLaravelWorkersStopped,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, "queue drain verification failed safely:", err)
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(verification); err != nil {
			fmt.Fprintln(os.Stderr, "queue drain verification could not encode its aggregate result")
			os.Exit(1)
		}
		if verification.Status != "ready" {
			os.Exit(2)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := process.Run(
		ctx,
		process.Registration{Name: "migrate"},
		buildinfo.Current(),
		logger,
	); err != nil {
		logger.Error("Migration process is not runnable", "error", err)
		os.Exit(1)
	}
}

func safeBridgeError(err error) string {
	if err == nil {
		return "migrate bridge failed"
	}
	return "migrate bridge failed: " + err.Error()
}
