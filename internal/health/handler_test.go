package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
)

type fakeChecker struct {
	err    error
	called bool
	check  func(context.Context) error
}

func (checker *fakeChecker) Check(ctx context.Context) error {
	checker.called = true
	if checker.check != nil {
		return checker.check(ctx)
	}
	return checker.err
}

func TestLiveReturnsBuildIdentity(t *testing.T) {
	checker := &fakeChecker{}
	handler := NewHandler(checker, buildinfo.Info{
		Service: "idelium-api-go",
		Version: "test",
		Commit:  "abc123",
	})
	response := httptest.NewRecorder()

	handler.Live(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if checker.called {
		t.Fatal("liveness queried a dependency")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("liveness response is cacheable")
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", response.Header().Get("Content-Type"))
	}
	for _, expected := range []string{`"status":"ok"`, `"version":"test"`, `"commit":"abc123"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response does not contain %s: %s", expected, response.Body.String())
		}
	}
}

func TestReadyReturnsServiceUnavailableWithoutInternalError(t *testing.T) {
	handler := NewHandler(&fakeChecker{err: errors.New("password was rejected")}, buildinfo.Current())
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
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("failed readiness response is cacheable")
	}
}

func TestReadyReturnsDatabaseStatus(t *testing.T) {
	handler := NewHandler(&fakeChecker{}, buildinfo.Current())
	response := httptest.NewRecorder()

	handler.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"database":"ok"`) {
		t.Fatalf("database readiness missing: %s", response.Body.String())
	}
}

func TestReadyBoundsDependencyCheck(t *testing.T) {
	checker := &fakeChecker{check: func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("readiness dependency check has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > readinessTimeout {
			t.Fatalf("unexpected readiness deadline: %s", remaining)
		}
		return nil
	}}
	handler := NewHandler(checker, buildinfo.Current())
	response := httptest.NewRecorder()

	handler.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}
