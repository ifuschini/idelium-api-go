package server

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/config"
)

func TestNewAppliesEveryBoundedTimeout(t *testing.T) {
	runtimeConfig := config.HTTPConfig{
		Address:           "127.0.0.1:0",
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       4 * time.Second,
		ShutdownTimeout:   5 * time.Second,
	}
	server := New(runtimeConfig, http.NotFoundHandler())

	if server.Addr != runtimeConfig.Address ||
		server.ReadHeaderTimeout != runtimeConfig.ReadHeaderTimeout ||
		server.ReadTimeout != runtimeConfig.ReadTimeout ||
		server.WriteTimeout != runtimeConfig.WriteTimeout ||
		server.IdleTimeout != runtimeConfig.IdleTimeout {
		t.Fatalf("server did not apply the bounded HTTP configuration: %#v", server)
	}
}

func TestServeStopsAfterLifecycleCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	httpServer := New(config.HTTPConfig{Address: listener.Addr().String()}, http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		},
	))

	go func() {
		done <- Serve(ctx, httpServer, listener, time.Second, logger)
	}()
	deadline := time.Now().Add(time.Second)
	for {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become reachable: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() returned an error during graceful shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not complete after lifecycle cancellation")
	}
}
