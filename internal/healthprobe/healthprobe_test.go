package healthprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckAcceptsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := Check(context.Background(), server.URL, time.Second); err != nil {
		t.Fatalf("expected successful healthcheck: %v", err)
	}
}

func TestCheckRejectsFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if err := Check(context.Background(), server.URL, time.Second); err == nil {
		t.Fatal("expected healthcheck failure")
	}
}
