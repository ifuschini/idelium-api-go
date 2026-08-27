// Package browserauth implements the Go-owned browser login, logout, and CSRF contract.
package browserauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "idelium_session"
	csrfCookieName    = "XSRF-TOKEN"
	csrfHeaderName    = "X-XSRF-TOKEN"
)

var ErrNotFound = errors.New("browser authentication record not found")

// User is the minimum authenticated identity projection. PasswordHash is never returned.
type User struct {
	ID           int64
	TenantID     int64
	Name         string
	Email        string
	Role         int64
	PasswordHash string
	Status       string
}

// Session is an opaque Go-owned browser session. Its raw identifier is never persisted.
type Session struct {
	ID        string
	UserID    int64
	TenantID  int64
	CSRFToken string
	ExpiresAt time.Time
}

type UserRepository interface {
	FindByEmail(context.Context, string) (User, error)
}
type SessionRepository interface {
	Create(context.Context, Session) error
	Delete(context.Context, string) error
}
type Repository interface {
	UserRepository
	SessionRepository
}

// Handler preserves the frontend-visible Laravel browser-auth response contract.
type Handler struct {
	users    UserRepository
	sessions SessionRepository
	logger   *slog.Logger
	now      func() time.Time
}

func NewHandler(users UserRepository, sessions SessionRepository, logger *slog.Logger) *Handler {
	return &Handler{users: users, sessions: sessions, logger: logger, now: time.Now}
}

func (h *Handler) CSRF(writer http.ResponseWriter, request *http.Request) {
	token, err := randomToken()
	if err != nil {
		h.internalError(writer, request, "issue csrf cookie", err)
		return
	}
	http.SetCookie(writer, &http.Cookie{Name: csrfCookieName, Value: token, Path: "/", MaxAge: 7200, Secure: true, HttpOnly: false, SameSite: http.SameSiteLaxMode})
	writer.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Login(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.Email) == "" || input.Password == "" {
		h.invalidLogin(writer)
		return
	}
	user, err := h.users.FindByEmail(request.Context(), strings.TrimSpace(input.Email))
	if err != nil || user.Status != "active" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		if err != nil && !errors.Is(err, ErrNotFound) {
			h.logFailure(request, "lookup browser user", err)
		}
		h.invalidLogin(writer)
		return
	}
	sessionID, err := randomToken()
	if err != nil {
		h.internalError(writer, request, "create browser session", err)
		return
	}
	csrfToken, err := randomToken()
	if err != nil {
		h.internalError(writer, request, "create csrf token", err)
		return
	}
	session := Session{ID: sessionID, UserID: user.ID, TenantID: user.TenantID, CSRFToken: csrfToken, ExpiresAt: h.now().UTC().Add(120 * time.Minute)}
	if err := h.sessions.Create(request.Context(), session); err != nil {
		h.internalError(writer, request, "store browser session", err)
		return
	}
	http.SetCookie(writer, &http.Cookie{Name: sessionCookieName, Value: sessionID, Path: "/", MaxAge: 7200, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(writer, &http.Cookie{Name: csrfCookieName, Value: csrfToken, Path: "/", MaxAge: 7200, Secure: true, HttpOnly: false, SameSite: http.SameSiteLaxMode})
	writeJSON(writer, http.StatusOK, map[string]any{"authenticated": true, "user": map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role}})
}

func (h *Handler) Logout(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.unauthorized(writer)
		return
	}
	if err := h.sessions.Delete(request.Context(), cookie.Value); err != nil && !errors.Is(err, ErrNotFound) {
		h.internalError(writer, request, "delete browser session", err)
		return
	}
	expireCookie(writer, sessionCookieName, true)
	expireCookie(writer, csrfCookieName, false)
	writer.WriteHeader(http.StatusNoContent)
}

func (h *Handler) invalidLogin(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid login details"})
}
func (h *Handler) unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthenticated."})
}
func expireCookie(w http.ResponseWriter, name string, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: httpOnly, SameSite: http.SameSiteLaxMode})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, action string, err error) {
	h.logFailure(r, action, err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "The request could not be completed."})
}
func (h *Handler) logFailure(r *http.Request, action string, err error) {
	if h.logger != nil {
		h.logger.ErrorContext(r.Context(), "browser authentication failed", "action", action, "error", err)
	}
}
