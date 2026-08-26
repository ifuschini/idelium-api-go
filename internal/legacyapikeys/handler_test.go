package legacyapikeys_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idelium/idelium-api-go/internal/legacyapikeys"
)

func TestShowFailsClosed(t *testing.T) {
	handler := legacyapikeys.NewHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/apikey", nil)

	handler.Show(response, request)

	assertMigrationDisabled(t, response)
}

func TestReplaceFailsClosedWithoutLeakingPayload(t *testing.T) {
	handler := legacyapikeys.NewHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/apikey",
		strings.NewReader(`{"apiKey":"must-not-leak","expiration":"never"}`),
	)

	handler.Replace(response, request)

	body := assertMigrationDisabled(t, response)
	if strings.Contains(body, "must-not-leak") || strings.Contains(body, "apiKey") {
		t.Fatalf("legacy API-key route leaked credential payload: %s", body)
	}
}

func assertMigrationDisabled(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "LEGACY_API_KEY_MIGRATION_DISABLED") {
		t.Fatalf("stable legacy API-key migration code missing: %s", body)
	}
	return body
}
