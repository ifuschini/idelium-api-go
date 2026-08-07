package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
)

type fakeChecker struct {
	err error
}

func (checker fakeChecker) Check(context.Context) error {
	return checker.err
}

func TestLiveReturnsBuildIdentity(t *testing.T) {
	handler := NewHandler(fakeChecker{}, buildinfo.Info{
		Service: "idelium-api-go",
		Version: "test",
		Commit:  "abc123",
	})
	response := httptest.NewRecorder()

	handler.Live(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	for _, expected := range []string{`"status":"ok"`, `"version":"test"`, `"commit":"abc123"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response does not contain %s: %s", expected, response.Body.String())
		}
	}
}

func TestReadyReturnsServiceUnavailableWithoutInternalError(t *testing.T) {
	handler := NewHandler(fakeChecker{err: errors.New("password was rejected")}, buildinfo.Current())
	response := httptest.NewRecorder()

	handler.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "password was rejected") {
		t.Fatal("dependency detail was exposed to the client")
	}
	if !strings.Contains(response.Body.String(), "DEPENDENCY_UNAVAILABLE") {
		t.Fatalf("stable error code missing: %s", response.Body.String())
	}
}

func TestReadyReturnsDatabaseStatus(t *testing.T) {
	handler := NewHandler(fakeChecker{}, buildinfo.Current())
	response := httptest.NewRecorder()

	handler.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"database":"ok"`) {
		t.Fatalf("database readiness missing: %s", response.Body.String())
	}
}
