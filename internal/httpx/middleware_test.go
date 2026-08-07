package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLoggerDoesNotRecordQueryOrHeaders(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := AccessLogger(logger)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health/live?token=secret-value", nil)
	request.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	logged := output.String()
	if strings.Contains(logged, "secret-value") || strings.Contains(logged, "secret-token") {
		t.Fatal("access log exposed a sensitive query or header value")
	}
	if !strings.Contains(logged, `"path":"/health/live"`) {
		t.Fatalf("access log did not include the safe path: %s", logged)
	}
}

func TestRecovererReturnsStableError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := Recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive internal detail")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "sensitive internal detail") {
		t.Fatal("panic detail was exposed to the client")
	}
	if !strings.Contains(response.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("stable error code missing from response: %s", response.Body.String())
	}
}
