// Package browserauth implements the Go-owned browser login, logout, and CSRF contract.
package browserauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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
	ID                     int64
	TenantID               int64
	ActiveTenantID         int64
	Name                   string
	Email                  string
	Role                   int64
	PasswordHash           string
	Status                 string
	ImpersonationReason    *string
	ImpersonationExpiresAt *time.Time
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
	Get(context.Context, string, time.Time) (User, error)
	ListProjects(context.Context, int64) ([]Project, error)
	ListCustomers(context.Context) ([]Customer, error)
	CustomerExists(context.Context, int64) (bool, error)
	SwitchTenant(context.Context, TenantSwitch) error
	RecordTenantSwitch(context.Context, AuditEvent) error
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

type Project struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	IDCostumer  int64      `json:"idCostumer"`
}

type Customer struct {
	ID                int64      `json:"id"`
	Costumer          string     `json:"costumer"`
	Description       *string    `json:"description,omitempty"`
	LicenseExpiration *time.Time `json:"licenseExpiration,omitempty"`
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type TenantSwitch struct {
	SessionID    string
	UserID       int64
	ActorTenant  int64
	ActiveTenant int64
	Reason       string
	ExpiresAt    time.Time
	Now          time.Time
}

type AuditEvent struct {
	ActorUserID    int64
	ActorTenantID  int64
	ActiveTenantID int64
	TargetID       int64
	SourceIP       string
	CorrelationID  string
	BeforeValues   map[string]any
	AfterValues    map[string]any
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

// CurrentUser returns the legacy minimal identity projection for an active Go session.
func (h *Handler) CurrentUser(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role})
}

// Header returns projects, optional customer choices, and the active tenant context.
func (h *Handler) Header(writer http.ResponseWriter, request *http.Request) {
	user, _, ok := h.authenticatedSession(writer, request)
	if !ok {
		return
	}
	projects, err := h.sessions.ListProjects(request.Context(), user.activeTenant())
	if err != nil {
		h.internalError(writer, request, "list header projects", err)
		return
	}
	response := map[string]any{
		"projects":      projects,
		"tenantContext": tenantContext(user),
	}
	if user.Role == 1 {
		customers, err := h.sessions.ListCustomers(request.Context())
		if err != nil {
			h.internalError(writer, request, "list header customers", err)
			return
		}
		response["costumers"] = customers
	}
	writeJSON(writer, http.StatusOK, response)
}

// Sidebar returns the legacy role-aware browser navigation entries.
func (h *Handler) Sidebar(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, sidebarForRole(user.Role))
}

// ChangeCustomer updates the active tenant carried by the Go browser session.
func (h *Handler) ChangeCustomer(writer http.ResponseWriter, request *http.Request) {
	user, sessionID, ok := h.authenticatedSession(writer, request)
	if !ok {
		return
	}
	if user.Role != 1 {
		h.forbidden(writer)
		return
	}
	targetTenantID, err := strconv.ParseInt(request.PathValue("idCostumer"), 10, 64)
	if err != nil || targetTenantID <= 0 {
		writeJSON(writer, http.StatusNotFound, map[string]any{"message": "Tenant was not found.", "error": map[string]string{"code": "TENANT_NOT_FOUND"}})
		return
	}
	var input struct {
		Reason    string `json:"reason"`
		ExpiresAt string `json:"expiresAt"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.Reason) == "" {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "The given data was invalid.", "errors": map[string][]string{"reason": {"The reason field is required."}}})
		return
	}
	expiresAt, err := parseFutureTime(input.ExpiresAt, h.now().UTC())
	if err != nil {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "The given data was invalid.", "errors": map[string][]string{"expiresAt": {"The expires at must be a date after now."}}})
		return
	}
	exists, err := h.sessions.CustomerExists(request.Context(), targetTenantID)
	if err != nil {
		h.internalError(writer, request, "lookup target customer", err)
		return
	}
	if !exists {
		writeJSON(writer, http.StatusNotFound, map[string]any{"message": "Tenant was not found.", "error": map[string]string{"code": "TENANT_NOT_FOUND"}})
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if err := h.sessions.SwitchTenant(request.Context(), TenantSwitch{SessionID: sessionID, UserID: user.ID, ActorTenant: user.TenantID, ActiveTenant: targetTenantID, Reason: reason, ExpiresAt: expiresAt, Now: h.now().UTC()}); err != nil {
		h.internalError(writer, request, "switch browser tenant", err)
		return
	}
	audit := AuditEvent{
		ActorUserID:    user.ID,
		ActorTenantID:  user.TenantID,
		ActiveTenantID: targetTenantID,
		TargetID:       targetTenantID,
		SourceIP:       sourceIP(request),
		CorrelationID:  correlationID(request),
		BeforeValues:   map[string]any{"activeTenantId": user.activeTenant()},
		AfterValues:    map[string]any{"activeTenantId": targetTenantID, "sessionToken": "[REDACTED]", "reason": reason, "expiresAt": input.ExpiresAt},
	}
	if err := h.sessions.RecordTenantSwitch(request.Context(), audit); err != nil {
		h.internalError(writer, request, "record tenant switch", err)
		return
	}
	updated := user
	updated.ActiveTenantID = targetTenantID
	updated.ImpersonationReason = &reason
	updated.ImpersonationExpiresAt = &expiresAt
	writeJSON(writer, http.StatusOK, map[string]any{"tenantContext": tenantContext(updated)})
}

// Capabilities returns the versioned capability set associated with the session user.
func (h *Handler) Capabilities(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"version": "2026-07-28", "capabilities": capabilitiesForRole(user.Role)})
}

func (h *Handler) authenticatedUser(writer http.ResponseWriter, request *http.Request) (User, bool) {
	user, _, ok := h.authenticatedSession(writer, request)
	return user, ok
}

func (h *Handler) authenticatedSession(writer http.ResponseWriter, request *http.Request) (User, string, bool) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		h.unauthorized(writer)
		return User{}, "", false
	}
	user, err := h.sessions.Get(request.Context(), cookie.Value, h.now().UTC())
	if errors.Is(err, ErrNotFound) {
		expireCookie(writer, sessionCookieName, true)
		expireCookie(writer, csrfCookieName, false)
		h.unauthorized(writer)
		return User{}, "", false
	}
	if err != nil {
		h.internalError(writer, request, "load browser session", err)
		return User{}, "", false
	}
	return user, cookie.Value, true
}

func capabilitiesForRole(role int64) []string {
	switch role {
	case 1:
		return []string{"tenant.switch", "accounts.manage", "customers.manage", "api_keys.manage", "agents.manage", "agents.read", "audit_events.read", "artifacts.manage", "artifacts.read", "integrations.manage", "integrations.read", "identity.manage", "identity.read", "projects.manage", "resources.manage", "resources.read", "runs.launch", "profile.manage"}
	case 2:
		return []string{"accounts.manage", "api_keys.manage", "agents.manage", "agents.read", "audit_events.read", "artifacts.manage", "artifacts.read", "integrations.manage", "integrations.read", "identity.manage", "identity.read", "projects.manage", "resources.manage", "resources.read", "runs.launch", "profile.manage"}
	case 3:
		return []string{"artifacts.read", "agents.read", "integrations.read", "identity.read", "projects.read", "resources.read", "runs.launch", "profile.manage"}
	default:
		return []string{}
	}
}

func (h *Handler) invalidLogin(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid login details"})
}
func (h *Handler) unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthenticated."})
}
func (h *Handler) forbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"message": "This action is unauthorized."})
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

func (u User) activeTenant() int64 {
	if u.ActiveTenantID > 0 {
		return u.ActiveTenantID
	}
	return u.TenantID
}

func tenantContext(user User) map[string]any {
	context := map[string]any{
		"actorUserId":            user.ID,
		"actorTenantId":          user.TenantID,
		"activeTenantId":         user.activeTenant(),
		"impersonating":          user.activeTenant() != user.TenantID,
		"impersonationReason":    nil,
		"impersonationExpiresAt": nil,
	}
	if user.ImpersonationReason != nil {
		context["impersonationReason"] = *user.ImpersonationReason
	}
	if user.ImpersonationExpiresAt != nil {
		context["impersonationExpiresAt"] = user.ImpersonationExpiresAt.Format(time.RFC3339)
	}
	return context
}

type sidebarItem struct {
	Icon            string `json:"icon"`
	Name            string `json:"name"`
	Link            string `json:"link"`
	Class           string `json:"class"`
	IsActiveEmptyDB bool   `json:"isActiveEmptyDb"`
}

func sidebarForRole(role int64) []sidebarItem {
	items := []sidebarItem{
		{Icon: "vials", Name: "testsperformed", Link: "testsperformed"},
		{Icon: "rocket", Name: "testlauncher", Link: "testlauncher"},
		{Icon: "sync", Name: "testcycles", Link: "testcycles"},
		{Icon: "vial", Name: "tests", Link: "tests"},
		{Icon: "shoe-prints", Name: "steps", Link: "steps", Class: "fa-rotate-270"},
		{Icon: "plug", Name: "plugins", Link: "plugins"},
		{Icon: "leaf", Name: "environments", Link: "environments"},
		{Icon: "project-diagram", Name: "projects", Link: "projects", IsActiveEmptyDB: true},
	}
	if role < 3 {
		items = append(items,
			sidebarItem{Icon: "users", Name: "account", Link: "account", IsActiveEmptyDB: true},
			sidebarItem{Icon: "key", Name: "apikey", Link: "apikey", IsActiveEmptyDB: true},
		)
	}
	if role == 1 {
		items = append(items,
			sidebarItem{Icon: "building", Name: "costumers", Link: "costumers", IsActiveEmptyDB: true},
			sidebarItem{Icon: "laptop", Name: "platforms", Link: "platforms", IsActiveEmptyDB: true},
		)
	}
	return items
}

func parseFutureTime(value string, now time.Time) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05", value)
	}
	if err != nil || !parsed.After(now) {
		return time.Time{}, errors.New("expiresAt must be after now")
	}
	return parsed.UTC(), nil
}

func sourceIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}

func correlationID(request *http.Request) string {
	if value := request.Header.Get("X-Correlation-ID"); value != "" {
		return value
	}
	token, err := randomToken()
	if err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	return token[:8] + "-" + token[8:12] + "-4" + token[13:16] + "-8" + token[17:20] + "-" + token[20:32]
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
