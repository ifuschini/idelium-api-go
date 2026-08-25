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
	types     []CatalogItem
	statuses  []CatalogItem
	locations LocationPage
	brands    BrandPage
	err       error
}

func (stub catalogRepositoryStub) ListTypes(context.Context) ([]CatalogItem, error) {
	return stub.types, stub.err
}

func (stub catalogRepositoryStub) ListStatuses(context.Context) ([]CatalogItem, error) {
	return stub.statuses, stub.err
}

func (stub catalogRepositoryStub) ListLocations(context.Context, LocationQuery) (LocationPage, error) {
	return stub.locations, stub.err
}

func (stub catalogRepositoryStub) ListBrands(context.Context, BrandQuery) (BrandPage, error) {
	return stub.brands, stub.err
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

func TestLocationsReturnsLegacyArrayWhenUnpaged(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		locations: LocationPage{
			Data: []LocationItem{{ID: 1, Name: "eu-west"}, {ID: 2, Name: "us-east"}},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Locations(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/locations", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"name":"eu-west"`, `"id":2`, `"name":"us-east"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, `"meta"`) {
		t.Fatalf("unpaged location response should preserve the legacy array shape: %s", body)
	}
}

func TestLocationsReturnsPagedGridWhenRequested(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		locations: LocationPage{
			Data: []LocationItem{{ID: 2, Name: "us-east"}},
			Meta: LocationPageMeta{
				Page:        2,
				PageSize:    1,
				Total:       2,
				LastPage:    2,
				Sort:        "name",
				Direction:   "desc",
				Stale:       false,
				Partial:     false,
			},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Locations(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/locations?page=2&pageSize=1&sort=name&direction=desc", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"data"`, `"meta"`, `"page":2`, `"pageSize":1`, `"direction":"desc"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
}

func TestBrandsReturnsLegacyArrayWhenUnpaged(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		brands: BrandPage{
			Data: []BrandItem{{ID: 1, Brand: "Apple"}, {ID: 2, Brand: "Samsung"}},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Brands(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/brands", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"brand":"Apple"`, `"id":2`, `"brand":"Samsung"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, `"meta"`) {
		t.Fatalf("unpaged brand response should preserve the legacy array shape: %s", body)
	}
}

func TestBrandsReturnsPagedGridWhenRequested(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		brands: BrandPage{
			Data: []BrandItem{{ID: 2, Brand: "Samsung"}},
			Meta: BrandPageMeta{
				Page:        2,
				PageSize:    1,
				Total:       2,
				LastPage:    2,
				Sort:        "brand",
				Direction:   "desc",
				Stale:       false,
				Partial:     false,
			},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Brands(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/brands?page=2&pageSize=1&sort=brand&direction=desc", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"data"`, `"meta"`, `"page":2`, `"pageSize":1`, `"direction":"desc"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
}
