// Package serviceaccounts implements tenant-scoped service-account lifecycle.
package serviceaccounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

type Account struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	CredentialID string     `json:"credentialId"`
	ProjectID    *int64     `json:"idProject,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
}
type Repository interface {
	List(context.Context, int64) ([]Account, error)
	Create(context.Context, int64, Account, string) (Account, error)
	Revoke(context.Context, int64, int64, time.Time) error
}
type Handler struct {
	logger     *slog.Logger
	sessions   browserauth.SessionRepository
	repository Repository
	now        func() time.Time
}

func NewHandler(logger *slog.Logger, deps ...any) Handler {
	var sessions browserauth.SessionRepository
	var repository Repository
	if len(deps) > 0 {
		sessions, _ = deps[0].(browserauth.SessionRepository)
	}
	if len(deps) > 1 {
		repository, _ = deps[1].(Repository)
	}
	return Handler{logger: logger, sessions: sessions, repository: repository, now: func() time.Time { return time.Now().UTC() }}
}
func (h Handler) user(w http.ResponseWriter, r *http.Request) (browserauth.User, bool) {
	if h.repository == nil {
		httpx.WriteError(w, r, http.StatusNotImplemented, "SERVICE_ACCOUNT_MIGRATION_DISABLED", "Service-account credential migration is not enabled for the Go runtime.")
		return browserauth.User{}, false
	}
	if h.sessions == nil || h.repository == nil {
		httpx.WriteError(w, r, 503, "SERVICE_ACCOUNT_UNAVAILABLE", "Service-account storage is unavailable.")
		return browserauth.User{}, false
	}
	u, ok := browserauth.AuthenticateRequest(r.Context(), r, h.sessions, h.now())
	if !ok {
		httpx.WriteError(w, r, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return browserauth.User{}, false
	}
	return u, true
}
func (h Handler) Index(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	v, e := h.repository.List(r.Context(), u.ActiveTenant())
	if e != nil {
		httpx.WriteError(w, r, 500, "SERVICE_ACCOUNT_UNAVAILABLE", "Service accounts could not be loaded.")
		return
	}
	httpx.WriteJSON(w, 200, v)
}
func (h Handler) Store(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	var in struct {
		Name      string     `json:"name"`
		ProjectID *int64     `json:"idProject"`
		Scopes    []string   `json:"scopes"`
		ExpiresAt *time.Time `json:"expiresAt"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		httpx.WriteError(w, r, 400, "INVALID_SERVICE_ACCOUNT", "The service-account name is required.")
		return
	}
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		httpx.WriteError(w, r, 500, "SERVICE_ACCOUNT_UNAVAILABLE", "Service account could not be created.")
		return
	}
	secret := base64.RawURLEncoding.EncodeToString(b)
	a := Account{Name: strings.TrimSpace(in.Name), ProjectID: in.ProjectID, Scopes: in.Scopes, ExpiresAt: in.ExpiresAt, CredentialID: base64.RawURLEncoding.EncodeToString(b[:12])}
	created, e := h.repository.Create(r.Context(), u.ActiveTenant(), a, secret)
	if e != nil {
		httpx.WriteError(w, r, 500, "SERVICE_ACCOUNT_UNAVAILABLE", "Service account could not be created.")
		return
	}
	httpx.WriteJSON(w, 201, map[string]any{"serviceAccount": created, "secret": secret})
}
func (h Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(chi.URLParam(r, "serviceAccount"), 10, 64)
	if e != nil || id <= 0 {
		httpx.WriteError(w, r, 400, "INVALID_SERVICE_ACCOUNT", "The service-account identifier is required.")
		return
	}
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	if e = h.repository.Revoke(r.Context(), u.ActiveTenant(), id, h.now()); e != nil {
		if errors.Is(e, ErrNotFound) {
			httpx.WriteError(w, r, 404, "SERVICE_ACCOUNT_NOT_FOUND", "Service account not found.")
			return
		}
		httpx.WriteError(w, r, 500, "SERVICE_ACCOUNT_UNAVAILABLE", "Service account could not be revoked.")
		return
	}
	httpx.WriteJSON(w, 200, map[string]string{"status": "revoked"})
}

var ErrNotFound = errors.New("service account not found")
