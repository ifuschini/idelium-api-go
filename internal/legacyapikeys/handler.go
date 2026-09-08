// Package legacyapikeys implements browser-managed legacy API-key lifecycle.
package legacyapikeys

import (
	"context"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/httpx"
	"log/slog"
	"net/http"
	"time"
)

type Lifecycle interface {
	Show(context.Context, int64, time.Time) (map[string]any, error)
	Replace(context.Context, int64, time.Time) (string, map[string]any, error)
}
type Handler struct {
	logger    *slog.Logger
	sessions  browserauth.SessionRepository
	lifecycle Lifecycle
	now       func() time.Time
}

func NewHandler(logger *slog.Logger, deps ...any) Handler {
	h := Handler{logger: logger, now: func() time.Time { return time.Now().UTC() }}
	if len(deps) > 0 {
		h.sessions, _ = deps[0].(browserauth.SessionRepository)
	}
	if len(deps) > 1 {
		h.lifecycle, _ = deps[1].(Lifecycle)
	}
	return h
}
func (h Handler) user(w http.ResponseWriter, r *http.Request) (browserauth.User, bool) {
	if h.lifecycle == nil {
		httpx.WriteError(w, r, http.StatusServiceUnavailable, "LEGACY_API_KEY_UNAVAILABLE", "Legacy API-key storage is unavailable.")
		return browserauth.User{}, false
	}
	u, ok := browserauth.AuthenticateRequest(r.Context(), r, h.sessions, h.now())
	if !ok {
		httpx.WriteError(w, r, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return browserauth.User{}, false
	}
	return u, true
}
func (h Handler) Show(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	v, e := h.lifecycle.Show(r.Context(), u.ActiveTenant(), h.now())
	if e != nil {
		httpx.WriteError(w, r, 500, "LEGACY_API_KEY_UNAVAILABLE", "API-key metadata could not be loaded.")
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func (h Handler) Replace(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	secret, v, e := h.lifecycle.Replace(r.Context(), u.ActiveTenant(), h.now())
	if e != nil {
		httpx.WriteError(w, r, 500, "LEGACY_API_KEY_UNAVAILABLE", "API key could not be rotated.")
		return
	}
	v["apiKey"] = secret
	httpx.WriteJSON(w, 200, v)
}
