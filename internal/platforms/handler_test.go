package platforms

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type catalogRepositoryStub struct {
	types    []CatalogItem
	statuses []CatalogItem
	err      error
}

func (stub catalogRepositoryStub) ListTypes(context.Context) ([]CatalogItem, error) {
	return stub.types, stub.err
}

func (stub catalogRepositoryStub) ListStatuses(context.Context) ([]CatalogItem, error) {
	return stub.statuses, stub.err
}

func TestTypesReturnsLegacyArrayShape(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		types: []CatalogItem{{ID: 1, Name: "desktop"}, {ID: 2, Name: "mobile"}},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Types(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/types", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"name":"desktop"`, `"id":2`, `"name":"mobile"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
}

func TestStatusesReturnsStableErrorEnvelope(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		err: errors.New("database password rejected"),
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Statuses(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/status", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "PLATFORM_CATALOG_UNAVAILABLE") {
		t.Fatalf("stable error code missing: %s", body)
	}
	if strings.Contains(body, "password rejected") {
		t.Fatalf("internal error leaked to client: %s", body)
	}
}
