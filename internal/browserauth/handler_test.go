package browserauth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type usersStub struct {
	user User
	err  error
}

func (s usersStub) FindByEmail(context.Context, string) (User, error) { return s.user, s.err }

type sessionsStub struct {
	created   Session
	createErr error
	deleted   string
	deleteErr error
	user      User
	getErr    error
}

func (s *sessionsStub) Create(_ context.Context, session Session) error {
	s.created = session
	return s.createErr
}
func (s *sessionsStub) Delete(_ context.Context, id string) error { s.deleted = id; return s.deleteErr }
func (s *sessionsStub) Get(_ context.Context, _ string, _ time.Time) (User, error) {
	return s.user, s.getErr
}

func TestLoginCreatesOpaqueSecureSessionForActiveTenantUser(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("SensitivePassword123!"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &sessionsStub{}
	handler := NewHandler(usersStub{user: User{ID: 7, TenantID: 11, Name: "Browser user", Email: "browser@example.test", Role: 3, PasswordHash: string(hash), Status: "active"}}, sessions, testLogger())
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"browser@example.test","password":"SensitivePassword123!"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"authenticated":true`, `"id":7`, `"name":"Browser user"`, `"email":"browser@example.test"`, `"role":3`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}
	if strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "session") {
		t.Fatalf("response leaked a secret: %s", response.Body.String())
	}
	if sessions.created.UserID != 7 || sessions.created.TenantID != 11 || sessions.created.ID == "" || sessions.created.CSRFToken == "" {
		t.Fatalf("incorrect persisted session: %#v", sessions.created)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected session and csrf cookies, got %#v", cookies)
	}
	if cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe session cookie: %#v", cookies[0])
	}
	if cookies[1].Name != csrfCookieName || cookies[1].HttpOnly || !cookies[1].Secure {
		t.Fatalf("incorrect csrf cookie: %#v", cookies[1])
	}
}

func TestLoginRejectsBadOrDisabledCredentialsWithLaravelCompatibleResponse(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []User{{PasswordHash: string(hash), Status: "active"}, {PasswordHash: string(hash), Status: "disabled"}} {
		handler := NewHandler(usersStub{user: user}, &sessionsStub{}, testLogger())
		response := httptest.NewRecorder()
		handler.Login(response, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.test","password":"incorrect"}`)))
		if response.Code != http.StatusUnauthorized || response.Body.String() != "{\"message\":\"Invalid login details\"}\n" {
			t.Fatalf("unexpected rejected login: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestLoginDoesNotExposeRepositoryFailures(t *testing.T) {
	logs := &bytes.Buffer{}
	handler := NewHandler(usersStub{err: errors.New("database password rejected")}, &sessionsStub{}, slog.New(slog.NewTextHandler(logs, nil)))
	response := httptest.NewRecorder()
	handler.Login(response, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.test","password":"value"}`)))
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "password rejected") {
		t.Fatalf("repository detail leaked: %s", response.Body.String())
	}
}

func TestLogoutRequiresSessionAndDeletesOpaqueIdentifier(t *testing.T) {
	sessions := &sessionsStub{}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	unauthenticated := httptest.NewRecorder()
	handler.Logout(unauthenticated, httptest.NewRequest(http.MethodPost, "/logout", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthenticated.Code)
	}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.Logout(response, request)
	if response.Code != http.StatusNoContent || sessions.deleted != "opaque-value" {
		t.Fatalf("logout failed: %d %#v", response.Code, sessions)
	}
	if len(response.Result().Cookies()) != 2 {
		t.Fatalf("logout did not expire both cookies")
	}
}

func TestCSRFIssuesFrontendCompatibleCookie(t *testing.T) {
	handler := NewHandler(usersStub{}, &sessionsStub{}, testLogger())
	response := httptest.NewRecorder()
	handler.CSRF(response, httptest.NewRequest(http.MethodGet, "/sanctum/csrf-cookie", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != csrfCookieName || cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("unexpected csrf cookie: %#v", cookies)
	}
}

func TestCurrentUserAndCapabilitiesRequireAnActiveSession(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, Name: "Browser user", Email: "browser@example.test", Role: 3}}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/user", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.CurrentUser(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"email":"browser@example.test"`) || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("unexpected current user response: %d %s", response.Code, response.Body.String())
	}
	capabilitiesRequest := httptest.NewRequest(http.MethodGet, "/me/capabilities", nil)
	capabilitiesRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	capabilitiesResponse := httptest.NewRecorder()
	handler.Capabilities(capabilitiesResponse, capabilitiesRequest)
	if capabilitiesResponse.Code != http.StatusOK || !strings.Contains(capabilitiesResponse.Body.String(), `"version":"2026-07-28"`) || !strings.Contains(capabilitiesResponse.Body.String(), `"projects.read"`) {
		t.Fatalf("unexpected capabilities response: %d %s", capabilitiesResponse.Code, capabilitiesResponse.Body.String())
	}
	unauthorized := httptest.NewRecorder()
	handler.CurrentUser(unauthorized, httptest.NewRequest(http.MethodGet, "/user", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var _ = time.Now
