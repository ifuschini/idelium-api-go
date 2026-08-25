package identity

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

func TestAdvancedIdentityRoutesFailClosedWithoutPayloadLeak(t *testing.T) {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID)
	handler := NewHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	router.Get("/admin/identity/providers", handler.Providers)
	router.Post("/admin/identity/providers", handler.Providers)
	router.Put("/admin/identity/accounts/{user}/break-glass", handler.BreakGlass)
	router.Post("/admin/identity/accounts/{user}/break-glass/test", handler.BreakGlassTest)
	router.Post("/admin/identity/providers/{identityProvider}/scim/users", handler.SCIMUsers)
	router.Post("/admin/profile/mfa/enroll", handler.MFAEnroll)
	router.Post("/admin/profile/mfa/confirm", handler.MFAConfirm)
	router.Post("/admin/profile/mfa/step-up", handler.MFAStepUp)
	router.Post("/oidc/token-exchange", handler.OIDCTokenExchange)
	router.Post("/sso/{identityProvider}/start", handler.SSOStart)
	router.Post("/sso/{identityProvider}/oidc/callback", handler.OIDCCallback)
	router.Post("/sso/{identityProvider}/saml/callback", handler.SAMLCallback)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/identity/providers"},
		{http.MethodPost, "/admin/identity/providers"},
		{http.MethodPut, "/admin/identity/accounts/7/break-glass"},
		{http.MethodPost, "/admin/identity/accounts/7/break-glass/test"},
		{http.MethodPost, "/admin/identity/providers/3/scim/users"},
		{http.MethodPost, "/admin/profile/mfa/enroll"},
		{http.MethodPost, "/admin/profile/mfa/confirm"},
		{http.MethodPost, "/admin/profile/mfa/step-up"},
		{http.MethodPost, "/oidc/token-exchange"},
		{http.MethodPost, "/sso/3/start"},
		{http.MethodPost, "/sso/3/oidc/callback"},
		{http.MethodPost, "/sso/3/saml/callback"},
	}

	for _, tt := range cases {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(
				tt.method,
				tt.path,
				strings.NewReader(`{"client_secret":"must-not-leak","assertion":"token"}`),
			)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotImplemented {
				t.Fatalf("expected status 501, got %d", response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, "IDENTITY_MIGRATION_DISABLED") {
				t.Fatalf("stable error code missing: %s", body)
			}
			if !strings.Contains(body, "correlationId") {
				t.Fatalf("correlation ID missing: %s", body)
			}
			for _, unsafe := range []string{"must-not-leak", "client_secret", "assertion", "token"} {
				if strings.Contains(body, unsafe) {
					t.Fatalf("identity gate leaked request payload marker %q: %s", unsafe, body)
				}
			}
		})
	}
}

func TestIdentityPathRoutesValidateRequiredIdentifiers(t *testing.T) {
	handler := NewHandler(nil)
	response := httptest.NewRecorder()

	handler.SCIMUsers(response, httptest.NewRequest(http.MethodPost, "/admin/identity/providers//scim/users", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "INVALID_IDENTITY_PROVIDER") {
		t.Fatalf("stable validation code missing: %s", response.Body.String())
	}
}
