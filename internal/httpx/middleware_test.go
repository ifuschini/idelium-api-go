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
	handler := CorrelationID(AccessLogger(logger)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})))

	request := httptest.NewRequest(http.MethodGet, "/health/live?token=secret-value", nil)
	request.Header.Set("Authorization", "Bearer secret-token")
	request.Header.Set(CorrelationIDHeader, "safe-request-123")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	logged := output.String()
	if strings.Contains(logged, "secret-value") || strings.Contains(logged, "secret-token") {
		t.Fatal("access log exposed a sensitive query or header value")
	}
	if !strings.Contains(logged, `"path":"/health/live"`) {
		t.Fatalf("access log did not include the safe path: %s", logged)
	}
	if !strings.Contains(logged, `"correlation_id":"safe-request-123"`) {
		t.Fatalf("access log did not include the validated correlation ID: %s", logged)
	}
}

func TestAccessLoggerRecordsImplicitSuccessStatus(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := CorrelationID(AccessLogger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/empty", nil))

	if !strings.Contains(output.String(), `"status":200`) {
		t.Fatalf("implicit success status was not recorded: %s", output.String())
	}
}

func TestRecovererReturnsStableError(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := CorrelationID(Recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive internal detail")
	})))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	request.Header.Set(CorrelationIDHeader, "panic-request-123")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "sensitive internal detail") {
		t.Fatal("panic detail was exposed to the client")
	}
	if !strings.Contains(response.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("stable error code missing from response: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "panic-request-123") {
		t.Fatalf("correlation ID missing from response: %s", response.Body.String())
	}
	if strings.Contains(output.String(), "sensitive internal detail") {
		t.Fatal("panic value was exposed in the structured log")
	}
	if !strings.Contains(output.String(), `"correlation_id":"panic-request-123"`) {
		t.Fatalf("panic log did not include correlation ID: %s", output.String())
	}
}

func TestCorrelationIDReplacesUnsafeCallerInput(t *testing.T) {
	handler := CorrelationID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
	}))
	request := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	request.Header.Set(CorrelationIDHeader, "unsafe\nvalue")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	correlationID := response.Header().Get(CorrelationIDHeader)
	if correlationID == "" || correlationID == "unsafe\nvalue" {
		t.Fatalf("unsafe correlation ID was not replaced: %q", correlationID)
	}
	if !strings.Contains(response.Body.String(), correlationID) {
		t.Fatalf("generated correlation ID missing from error response: %s", response.Body.String())
	}
}

func TestSecureHeadersAppliesAPIProtectionPolicy(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	expected := map[string]string{
		"Content-Security-Policy":           "default-src 'none'; frame-ancestors 'none'",
		"Permissions-Policy":                "camera=(), geolocation=(), microphone=()",
		"Referrer-Policy":                   "no-referrer",
		"Strict-Transport-Security":         "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                   "DENY",
		"X-Permitted-Cross-Domain-Policies": "none",
	}
	for header, value := range expected {
		if response.Header().Get(header) != value {
			t.Fatalf("expected %s header %q, got %q", header, value, response.Header().Get(header))
		}
	}
}
