package identity

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

func TestMFARoutesFailClosedUntilNativeStepUpIsEnabled(t *testing.T) {
	router := chi.NewRouter()
	router.Use(httpx.CorrelationID)
	handler := NewHandler(nil)
	router.Post("/admin/profile/mfa/enroll", handler.MFAEnroll)
	router.Post("/admin/profile/mfa/confirm", handler.MFAConfirm)
	router.Post("/admin/profile/mfa/step-up", handler.MFAStepUp)

	for _, path := range []string{
		"/admin/profile/mfa/enroll",
		"/admin/profile/mfa/confirm",
		"/admin/profile/mfa/step-up",
	} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost,
				path,
				strings.NewReader(`{"otp":"123456","recovery_code":"do-not-leak"}`),
			)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotImplemented {
				t.Fatalf("expected status 501, got %d", response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, "IDENTITY_MIGRATION_DISABLED") {
				t.Fatalf("stable MFA gate code missing: %s", body)
			}
			for _, unsafe := range []string{"123456", "do-not-leak", "recovery_code", "otp"} {
				if strings.Contains(body, unsafe) {
					t.Fatalf("MFA gate leaked sensitive request marker %q: %s", unsafe, body)
				}
			}
		})
	}
}
