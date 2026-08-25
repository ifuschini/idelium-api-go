package serviceaccounts

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

func TestServiceAccountRoutesFailClosedWithoutPayloadLeak(t *testing.T) {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID)
	handler := NewHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	router.Get("/admin/service-accounts", handler.Index)
	router.Post("/admin/service-accounts", handler.Store)
	router.Post("/admin/service-accounts/{serviceAccount}/revoke", handler.Revoke)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/service-accounts"},
		{http.MethodPost, "/admin/service-accounts"},
		{http.MethodPost, "/admin/service-accounts/17/revoke"},
	}

	for _, tt := range cases {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(
				tt.method,
				tt.path,
				strings.NewReader(`{"name":"robot","scopes":["admin"],"client_secret":"must-not-leak"}`),
			)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotImplemented {
				t.Fatalf("expected status 501, got %d", response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, "SERVICE_ACCOUNT_MIGRATION_DISABLED") {
				t.Fatalf("stable error code missing: %s", body)
			}
			if !strings.Contains(body, "correlationId") {
				t.Fatalf("correlation ID missing: %s", body)
			}
			for _, unsafe := range []string{"must-not-leak", "client_secret", "scopes", "robot"} {
				if strings.Contains(body, unsafe) {
					t.Fatalf("service-account gate leaked request payload marker %q: %s", unsafe, body)
				}
			}
		})
	}
}

func TestServiceAccountRevokeValidatesRequiredIdentifier(t *testing.T) {
	handler := NewHandler(nil)
	response := httptest.NewRecorder()

	handler.Revoke(response, httptest.NewRequest(http.MethodPost, "/admin/service-accounts//revoke", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "INVALID_SERVICE_ACCOUNT") {
		t.Fatalf("stable validation code missing: %s", response.Body.String())
	}
}
