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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := process.Run(
		ctx,
		process.Registration{Name: "worker"},
		buildinfo.Current(),
		logger,
	); err != nil {
		logger.Error("Worker is not runnable", "error", err)
		os.Exit(1)
	}
}
