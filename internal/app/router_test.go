package app

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

type readyChecker struct{}

func (readyChecker) Check(context.Context) error { return nil }

type fakeCatalogRepository struct{}

func (fakeCatalogRepository) ListTypes(context.Context) ([]platforms.CatalogItem, error) {
	return []platforms.CatalogItem{{ID: 1, Name: "desktop"}}, nil
}

func (fakeCatalogRepository) ListStatuses(context.Context) ([]platforms.CatalogItem, error) {
	return []platforms.CatalogItem{{ID: 1, Name: "free"}}, nil
}

func (fakeCatalogRepository) ListLocations(context.Context, platforms.LocationQuery) (platforms.LocationPage, error) {
	return platforms.LocationPage{
		Data: []platforms.LocationItem{{ID: 1, Name: "eu-west"}},
	}, nil
}

func TestRouterReturnsStableNotFoundResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := NewRouter(logger, readyChecker{}, buildinfo.Current(), fakeCatalogRepository{})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "ROUTE_NOT_FOUND") {
		t.Fatalf("stable error code missing: %s", response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("secure response headers were not applied")
	}
}

func TestRouterRejectsUnsupportedMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := NewRouter(logger, readyChecker{}, buildinfo.Current(), fakeCatalogRepository{})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/health/live", nil))

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "METHOD_NOT_ALLOWED") {
		t.Fatalf("stable error code missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := NewRouter(logger, readyChecker{}, buildinfo.Current(), fakeCatalogRepository{})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/types", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"desktop"`) {
		t.Fatalf("platform type response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformLocations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := NewRouter(logger, readyChecker{}, buildinfo.Current(), fakeCatalogRepository{})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/locations", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"eu-west"`) {
		t.Fatalf("platform location response missing: %s", response.Body.String())
	}
}
