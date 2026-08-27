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
	created           Session
	createErr         error
	deleted           string
	deleteErr         error
	user              User
	getErr            error
	projects          []Project
	customers         []Customer
	customerExists    bool
	switched          TenantSwitch
	recorded          AuditEvent
	switchErr         error
	recordErr         error
	roles             []Role
	profile           Profile
	accounts          AccountPage
	createdAccount    AccountCreate
	updatedAccount    AccountUpdate
	deletedAccountID  int64
	accountErr        error
	customerPage      CustomerPage
	createdCustomer   CustomerCreate
	updatedCustomer   CustomerUpdate
	deletedCustomerID int64
}

func (s *sessionsStub) Create(_ context.Context, session Session) error {
	s.created = session
	return s.createErr
}
func (s *sessionsStub) Delete(_ context.Context, id string) error { s.deleted = id; return s.deleteErr }
func (s *sessionsStub) Get(_ context.Context, _ string, _ time.Time) (User, error) {
	return s.user, s.getErr
}
func (s *sessionsStub) ListProjects(_ context.Context, _ int64) ([]Project, error) {
	return s.projects, nil
}
func (s *sessionsStub) ListCustomers(context.Context) ([]Customer, error) { return s.customers, nil }
func (s *sessionsStub) CustomerExists(context.Context, int64) (bool, error) {
	return s.customerExists, nil
}
func (s *sessionsStub) SwitchTenant(_ context.Context, tenantSwitch TenantSwitch) error {
	s.switched = tenantSwitch
	return s.switchErr
}
func (s *sessionsStub) RecordTenantSwitch(_ context.Context, event AuditEvent) error {
	s.recorded = event
	return s.recordErr
}
func (s *sessionsStub) ListRoles(_ *http.Request, actor User) ([]Role, bool, error) {
	return s.roles, actor.Role > 2, s.accountErr
}
func (s *sessionsStub) Profile(*http.Request, User) (Profile, error) {
	return s.profile, s.accountErr
}
func (s *sessionsStub) UpdateProfilePassword(*http.Request, User, string) (Profile, error) {
	return s.profile, s.accountErr
}
func (s *sessionsStub) ListAccounts(*http.Request, User, AccountQuery) (AccountPage, error) {
	return s.accounts, s.accountErr
}
func (s *sessionsStub) CreateAccount(_ *http.Request, _ User, account AccountCreate) error {
	s.createdAccount = account
	return s.accountErr
}
func (s *sessionsStub) UpdateAccount(_ *http.Request, _ User, account AccountUpdate) error {
	s.updatedAccount = account
	return s.accountErr
}
func (s *sessionsStub) DeleteAccount(_ *http.Request, _ User, accountID int64) error {
	s.deletedAccountID = accountID
	return s.accountErr
}
func (s *sessionsStub) ListAdminCustomers(*http.Request, CustomerQuery) (CustomerPage, error) {
	return s.customerPage, s.accountErr
}
func (s *sessionsStub) CreateCustomer(_ *http.Request, customer CustomerCreate) error {
	s.createdCustomer = customer
	return s.accountErr
}
func (s *sessionsStub) UpdateCustomer(_ *http.Request, customer CustomerUpdate) error {
	s.updatedCustomer = customer
	return s.accountErr
}
func (s *sessionsStub) DeleteCustomer(_ *http.Request, customerID int64) error {
	s.deletedCustomerID = customerID
	return s.accountErr
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

func TestHeaderAndSidebarUseActiveTenantAndRole(t *testing.T) {
	sessions := &sessionsStub{
		user:      User{ID: 7, TenantID: 11, ActiveTenantID: 42, Name: "Browser user", Email: "browser@example.test", Role: 1},
		projects:  []Project{{ID: 3, Name: "Project", IDCostumer: 42}},
		customers: []Customer{{ID: 42, Costumer: "ACME"}},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/menu/header", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.Header(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"projects"`) || !strings.Contains(response.Body.String(), `"costumers"`) || !strings.Contains(response.Body.String(), `"activeTenantId":42`) {
		t.Fatalf("unexpected header response: %d %s", response.Code, response.Body.String())
	}

	sidebarRequest := httptest.NewRequest(http.MethodGet, "/menu/sidebar", nil)
	sidebarRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	sidebarResponse := httptest.NewRecorder()
	handler.Sidebar(sidebarResponse, sidebarRequest)
	if sidebarResponse.Code != http.StatusOK || !strings.Contains(sidebarResponse.Body.String(), `"costumers"`) || !strings.Contains(sidebarResponse.Body.String(), `"platforms"`) {
		t.Fatalf("unexpected sidebar response: %d %s", sidebarResponse.Code, sidebarResponse.Body.String())
	}
}

func TestChangeCustomerValidatesAndRecordsAudit(t *testing.T) {
	sessions := &sessionsStub{
		user:           User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 1},
		customerExists: true,
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	handler.now = func() time.Time { return time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC) }

	request := httptest.NewRequest(http.MethodPut, "/menu/header/42", strings.NewReader(`{"reason":"support","expiresAt":"2026-08-27T13:00:00Z"}`))
	request.SetPathValue("idCostumer", "42")
	request.RemoteAddr = "203.0.113.10:1234"
	request.Header.Set("X-Correlation-ID", "11111111-1111-4111-8111-111111111111")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.ChangeCustomer(response, request)
	if response.Code != http.StatusOK || sessions.switched.ActiveTenant != 42 || sessions.recorded.AfterValues["sessionToken"] != "[REDACTED]" {
		t.Fatalf("unexpected tenant switch: status=%d body=%s switch=%#v audit=%#v", response.Code, response.Body.String(), sessions.switched, sessions.recorded)
	}

	missing := &sessionsStub{user: User{ID: 7, TenantID: 11, Role: 1}}
	missingHandler := NewHandler(usersStub{}, missing, testLogger())
	missingRequest := httptest.NewRequest(http.MethodPut, "/menu/header/999", strings.NewReader(`{"reason":"support","expiresAt":"2026-08-27T13:00:00Z"}`))
	missingRequest.SetPathValue("idCostumer", "999")
	missingRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	missingResponse := httptest.NewRecorder()
	missingHandler.ChangeCustomer(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing target 404, got %d", missingResponse.Code)
	}

	forbiddenHandler := NewHandler(usersStub{}, &sessionsStub{user: User{ID: 8, TenantID: 11, Role: 2}}, testLogger())
	forbiddenRequest := httptest.NewRequest(http.MethodPut, "/menu/header/42", strings.NewReader(`{"reason":"support","expiresAt":"2026-08-27T13:00:00Z"}`))
	forbiddenRequest.SetPathValue("idCostumer", "42")
	forbiddenRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	forbiddenResponse := httptest.NewRecorder()
	forbiddenHandler.ChangeCustomer(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden tenant switch, got %d", forbiddenResponse.Code)
	}
}

func TestBrowserAdminRolesProfileAndAccounts(t *testing.T) {
	sessions := &sessionsStub{
		user:     User{ID: 7, TenantID: 11, Role: 2},
		roles:    []Role{{ID: 2, Name: "admin"}, {ID: 3, Name: "user"}},
		profile:  Profile{Email: "admin@example.test", Name: "Admin", CompanyName: "ACME", RoleName: "admin"},
		accounts: AccountPage{Data: []Account{{ID: 8, Email: "user@example.test", Name: "User", Role: 3, IDCostumer: 11, Costumer: "ACME", RoleName: "user"}}},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	rolesRequest := httptest.NewRequest(http.MethodGet, "/admin/roles", nil)
	rolesRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	rolesResponse := httptest.NewRecorder()
	handler.Roles(rolesResponse, rolesRequest)
	if rolesResponse.Code != http.StatusOK || strings.Contains(rolesResponse.Body.String(), `"password"`) || !strings.Contains(rolesResponse.Body.String(), `"admin"`) {
		t.Fatalf("unexpected roles response: %d %s", rolesResponse.Code, rolesResponse.Body.String())
	}

	profileRequest := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	profileRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	profileResponse := httptest.NewRecorder()
	handler.Profile(profileResponse, profileRequest)
	if profileResponse.Code != http.StatusOK || !strings.Contains(profileResponse.Body.String(), `"companyName":"ACME"`) || strings.Contains(profileResponse.Body.String(), "password") {
		t.Fatalf("unexpected profile response: %d %s", profileResponse.Code, profileResponse.Body.String())
	}

	accountsRequest := httptest.NewRequest(http.MethodGet, "/admin/accounts?page=1&pageSize=25", nil)
	accountsRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	accountsResponse := httptest.NewRecorder()
	handler.Accounts(accountsResponse, accountsRequest)
	if accountsResponse.Code != http.StatusOK || !strings.Contains(accountsResponse.Body.String(), `"user@example.test"`) || strings.Contains(accountsResponse.Body.String(), "password") {
		t.Fatalf("unexpected accounts response: %d %s", accountsResponse.Code, accountsResponse.Body.String())
	}
}

func TestBrowserAdminRejectsWeakPasswordsAndForbiddenAccounts(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, Role: 2}}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	request := httptest.NewRequest(http.MethodPost, "/admin/accounts", strings.NewReader(`{"name":"User","email":"user@example.test","password":"password","role":3}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.CreateAccount(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "too common") {
		t.Fatalf("expected password policy rejection, got %d %s", response.Code, response.Body.String())
	}

	forbidden := &sessionsStub{user: User{ID: 7, TenantID: 11, Role: 3}}
	forbiddenHandler := NewHandler(usersStub{}, forbidden, testLogger())
	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/admin/accounts", nil)
	forbiddenRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	forbiddenResponse := httptest.NewRecorder()
	forbiddenHandler.Accounts(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden accounts list, got %d", forbiddenResponse.Code)
	}
}

func TestCustomerAdministrationRequiresSuperAdminAndMutatesCustomers(t *testing.T) {
	sessions := &sessionsStub{
		user:         User{ID: 1, TenantID: 11, Role: 1},
		customerPage: CustomerPage{Data: []Customer{{ID: 42, Costumer: "ACME"}}},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	request := httptest.NewRequest(http.MethodPost, "/admin/costumers", strings.NewReader(`{"costumer":"acme","description":"demo"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.CreateCustomer(response, request)
	if response.Code != http.StatusOK || sessions.createdCustomer.Costumer != "acme" || !strings.Contains(response.Body.String(), `"ACME"`) {
		t.Fatalf("unexpected customer create: %d %s %#v", response.Code, response.Body.String(), sessions.createdCustomer)
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/admin/costumers/42", strings.NewReader(`{"costumer":"new","description":"updated"}`))
	updateRequest.SetPathValue("idCostumer", "42")
	updateRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	updateResponse := httptest.NewRecorder()
	handler.UpdateCustomer(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK || sessions.updatedCustomer.ID != 42 {
		t.Fatalf("unexpected customer update: %d %#v", updateResponse.Code, sessions.updatedCustomer)
	}

	forbidden := &sessionsStub{user: User{ID: 2, TenantID: 11, Role: 2}}
	forbiddenHandler := NewHandler(usersStub{}, forbidden, testLogger())
	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/admin/costumers", nil)
	forbiddenRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	forbiddenResponse := httptest.NewRecorder()
	forbiddenHandler.Customers(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected role 2 customer administration to be forbidden, got %d", forbiddenResponse.Code)
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var _ = time.Now
