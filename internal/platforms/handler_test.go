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

	"github.com/go-chi/chi/v5"
)

type catalogRepositoryStub struct {
	types     []CatalogItem
	statuses  []CatalogItem
	locations LocationPage
	brands    BrandPage
	models    ModelPage
	os        OperatingSystemPage
	osVersion OperatingSystemVersionPage
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

func (stub catalogRepositoryStub) ListModels(context.Context, ModelQuery) (ModelPage, error) {
	return stub.models, stub.err
}

func (stub catalogRepositoryStub) ListOperatingSystems(context.Context, OperatingSystemQuery) (OperatingSystemPage, error) {
	return stub.os, stub.err
}

func (stub catalogRepositoryStub) ListOperatingSystemVersions(context.Context, OperatingSystemVersionQuery) (OperatingSystemVersionPage, error) {
	return stub.osVersion, stub.err
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

func TestModelsReturnsLegacyArrayWhenUnpaged(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		models: ModelPage{
			Data: []ModelItem{{ID: 1, Model: "iPhone", IDBrand: 7}, {ID: 2, Model: "iPad", IDBrand: 7}},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/models/7", nil)

	handler.Models(response, withPathParam(request, "idBrand", "7"))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"model":"iPhone"`, `"idBrand":7`, `"model":"iPad"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, `"meta"`) {
		t.Fatalf("unpaged model response should preserve the legacy array shape: %s", body)
	}
}

func TestModelsRejectsInvalidBrandIdentifier(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/models/not-a-number", nil)

	handler.Models(response, withPathParam(request, "idBrand", "not-a-number"))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "INVALID_PLATFORM_BRAND") {
		t.Fatalf("stable validation code missing: %s", body)
	}
}

func TestOperatingSystemsReturnsLegacyArrayWhenUnpaged(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		os: OperatingSystemPage{
			Data: []OperatingSystemItem{{ID: 1, Name: "linux", Type: 1}, {ID: 2, Name: "windows", Type: 1}},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/os/1", nil)

	handler.OperatingSystems(response, withPathParam(request, "idType", "1"))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"name":"linux"`, `"type":1`, `"name":"windows"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, `"meta"`) {
		t.Fatalf("unpaged operating-system response should preserve the legacy array shape: %s", body)
	}
}

func TestOperatingSystemsReturnsPagedGridWhenRequested(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		os: OperatingSystemPage{
			Data: []OperatingSystemItem{{ID: 2, Name: "windows", Type: 1}},
			Meta: OperatingSystemPageMeta{
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
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/os/1?page=2&pageSize=1&sort=name&direction=desc", nil)

	handler.OperatingSystems(response, withPathParam(request, "idType", "1"))

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

func TestOperatingSystemsRejectsInvalidTypeIdentifier(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/os/not-a-number", nil)

	handler.OperatingSystems(response, withPathParam(request, "idType", "not-a-number"))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "INVALID_PLATFORM_TYPE") {
		t.Fatalf("stable validation code missing: %s", body)
	}
}

func TestOperatingSystemVersionsReturnsLegacyArrayWhenUnpaged(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		osVersion: OperatingSystemVersionPage{
			Data: []OperatingSystemVersionItem{{ID: 1, Version: "14", IDOs: 1}, {ID: 2, Version: "15", IDOs: 1}},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/osversion/1", nil)

	handler.OperatingSystemVersions(response, withPathParam(request, "idOs", "1"))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":1`, `"version":"14"`, `"idOs":1`, `"version":"15"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, `"meta"`) {
		t.Fatalf("unpaged OS-version response should preserve the legacy array shape: %s", body)
	}
}

func TestOperatingSystemVersionsReturnsPagedGridWhenRequested(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{
		osVersion: OperatingSystemVersionPage{
			Data: []OperatingSystemVersionItem{{ID: 2, Version: "15", IDOs: 1}},
			Meta: OperatingSystemVersionPageMeta{
				Page:        2,
				PageSize:    1,
				Total:       2,
				LastPage:    2,
				Sort:        "version",
				Direction:   "desc",
				Stale:       false,
				Partial:     false,
			},
		},
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/osversion/1?page=2&pageSize=1&sort=version&direction=desc", nil)

	handler.OperatingSystemVersions(response, withPathParam(request, "idOs", "1"))

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

func TestOperatingSystemVersionsRejectsInvalidOSIdentifier(t *testing.T) {
	handler := NewHandler(catalogRepositoryStub{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/osversion/not-a-number", nil)

	handler.OperatingSystemVersions(response, withPathParam(request, "idOs", "not-a-number"))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "INVALID_OPERATING_SYSTEM") {
		t.Fatalf("stable validation code missing: %s", body)
	}
}

func withPathParam(request *http.Request, name string, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}
