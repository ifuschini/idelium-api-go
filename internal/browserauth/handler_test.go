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
	testCyclePage     TestCyclePage
	testCycleDetail   TestCycleDetail
	createdTestCycle  TestCycleCreate
	updatedTestCycle  TestCycleUpdate
	reorderedSteps    StepReorder
	stepPage          StepPage
	testPage          TestPage
	testDetail        TestDetail
	createdTest       TestCreate
	updatedTest       TestUpdate
	importedTest      TestImport
	performedCycles   PerformedCyclePage
	performedTests    PerformedTestPage
	performedSteps    []PerformedStep
	createdExport     ResultExportCreate
	exportDescriptor  ResultExportDescriptor
	exportDownload    ResultExportDownload
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
func (s *sessionsStub) ListTestCycles(*http.Request, User, ResourceQuery) (TestCyclePage, error) {
	return s.testCyclePage, s.accountErr
}
func (s *sessionsStub) CreateTestCycle(_ *http.Request, _ User, input TestCycleCreate) error {
	s.createdTestCycle = input
	return s.accountErr
}
func (s *sessionsStub) GetTestCycle(*http.Request, User, int64, int64) (TestCycleDetail, error) {
	return s.testCycleDetail, s.accountErr
}
func (s *sessionsStub) UpdateTestCycle(_ *http.Request, _ User, input TestCycleUpdate) error {
	s.updatedTestCycle = input
	return s.accountErr
}
func (s *sessionsStub) ReorderSteps(_ *http.Request, _ User, input StepReorder) error {
	s.reorderedSteps = input
	return s.accountErr
}
func (s *sessionsStub) ListStepsForReorder(*http.Request, User, ResourceQuery) (StepPage, error) {
	return s.stepPage, s.accountErr
}
func (s *sessionsStub) ListTests(*http.Request, User, ResourceQuery) (TestPage, error) {
	return s.testPage, s.accountErr
}
func (s *sessionsStub) CreateTest(_ *http.Request, _ User, input TestCreate) error {
	s.createdTest = input
	return s.accountErr
}
func (s *sessionsStub) GetTest(*http.Request, User, int64, int64) (TestDetail, error) {
	return s.testDetail, s.accountErr
}
func (s *sessionsStub) UpdateTest(_ *http.Request, _ User, input TestUpdate) error {
	s.updatedTest = input
	return s.accountErr
}
func (s *sessionsStub) ImportTest(_ *http.Request, _ User, input TestImport) error {
	s.importedTest = input
	return s.accountErr
}
func (s *sessionsStub) ListPerformedCycles(*http.Request, User, ResultQuery) (PerformedCyclePage, error) {
	return s.performedCycles, s.accountErr
}
func (s *sessionsStub) ListPerformedTests(*http.Request, User, ResultQuery) (PerformedTestPage, error) {
	return s.performedTests, s.accountErr
}
func (s *sessionsStub) ListPerformedSteps(*http.Request, User, int64) ([]PerformedStep, error) {
	return s.performedSteps, s.accountErr
}
func (s *sessionsStub) CreateResultExport(_ *http.Request, _ User, input ResultExportCreate) (ResultExportDescriptor, error) {
	s.createdExport = input
	return s.exportDescriptor, s.accountErr
}
func (s *sessionsStub) GetResultExport(*http.Request, User, int64) (ResultExportDescriptor, error) {
	return s.exportDescriptor, s.accountErr
}
func (s *sessionsStub) DownloadResultExport(*http.Request, User, int64, time.Time) (ResultExportDownload, error) {
	return s.exportDownload, s.accountErr
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
	missingHandler.now = func() time.Time { return time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC) }
	missingRequest := httptest.NewRequest(http.MethodPut, "/menu/header/999", strings.NewReader(`{"reason":"support","expiresAt":"2026-08-27T13:00:00Z"}`))
	missingRequest.SetPathValue("idCostumer", "999")
	missingRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	missingResponse := httptest.NewRecorder()
	missingHandler.ChangeCustomer(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing target 404, got %d", missingResponse.Code)
	}

	forbiddenHandler := NewHandler(usersStub{}, &sessionsStub{user: User{ID: 8, TenantID: 11, Role: 2}}, testLogger())
	forbiddenHandler.now = func() time.Time { return time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC) }
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

func TestTestCycleAdministrationAndStepReorder(t *testing.T) {
	sessions := &sessionsStub{
		user:            User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3},
		testCyclePage:   TestCyclePage{Data: []TestCycle{{ID: 5, Name: "Nightly", Description: "Browser"}}},
		testCycleDetail: TestCycleDetail{ID: 5, Name: "Nightly", Description: "Browser", Config: "{}", IDProject: 3},
		stepPage:        StepPage{Data: []StepListItem{{ID: 9, Name: "Login", Description: "Open", Order: 25}}},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/testcycles/3?page=1&pageSize=25", nil)
	listRequest.SetPathValue("idProject", "3")
	listRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	listResponse := httptest.NewRecorder()
	handler.TestCycles(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"Nightly"`) || strings.Contains(listResponse.Body.String(), "idCostumer") {
		t.Fatalf("unexpected test cycle list: %d %s", listResponse.Code, listResponse.Body.String())
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/admin/testcycles", strings.NewReader(`{"name":"Smoke","description":"Fast","config":"{}","idProject":3}`))
	createRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	createResponse := httptest.NewRecorder()
	handler.CreateTestCycle(createResponse, createRequest)
	if createResponse.Code != http.StatusOK || sessions.createdTestCycle.Name != "Smoke" || sessions.createdTestCycle.IDProject != 3 {
		t.Fatalf("unexpected test cycle create: %d %#v", createResponse.Code, sessions.createdTestCycle)
	}

	showRequest := httptest.NewRequest(http.MethodGet, "/admin/testcycles/3/5", nil)
	showRequest.SetPathValue("idProject", "3")
	showRequest.SetPathValue("testcycle", "5")
	showRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	showResponse := httptest.NewRecorder()
	handler.ShowTestCycle(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || !strings.Contains(showResponse.Body.String(), `"config":"{}"`) {
		t.Fatalf("unexpected test cycle detail: %d %s", showResponse.Code, showResponse.Body.String())
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/admin/testcycles/3/5", strings.NewReader(`{"description":"Updated","config":"{\"tests\":[]}"}`))
	updateRequest.SetPathValue("idProject", "3")
	updateRequest.SetPathValue("testcycle", "5")
	updateRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	updateResponse := httptest.NewRecorder()
	handler.UpdateTestCycle(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK || sessions.updatedTestCycle.ID != 5 || sessions.updatedTestCycle.IDProject != 3 {
		t.Fatalf("unexpected test cycle update: %d %#v", updateResponse.Code, sessions.updatedTestCycle)
	}

	reorderRequest := httptest.NewRequest(http.MethodPost, "/admin/steps/3/updateorder", strings.NewReader(`{"offset":25,"order":[{"id":9}]}`))
	reorderRequest.SetPathValue("idProject", "3")
	reorderRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	reorderResponse := httptest.NewRecorder()
	handler.ReorderSteps(reorderResponse, reorderRequest)
	if reorderResponse.Code != http.StatusOK || sessions.reorderedSteps.Offset != 25 || sessions.reorderedSteps.Order[0].ID != 9 {
		t.Fatalf("unexpected step reorder: %d %#v", reorderResponse.Code, sessions.reorderedSteps)
	}
}

func TestTestAdministrationPreservesStepMembershipConfig(t *testing.T) {
	sessions := &sessionsStub{
		user:       User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3},
		testPage:   TestPage{Data: []Test{{ID: 12, Name: "Checkout", Description: "Happy path"}}},
		testDetail: TestDetail{ID: 12, Name: "Checkout", Description: "Happy path", Config: `{"steps":[9,10]}`, IDProject: 3},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/tests/3?page=1&pageSize=25", nil)
	listRequest.SetPathValue("idProject", "3")
	listRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	listResponse := httptest.NewRecorder()
	handler.Tests(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"Checkout"`) || strings.Contains(listResponse.Body.String(), "idCostumer") {
		t.Fatalf("unexpected test list: %d %s", listResponse.Code, listResponse.Body.String())
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/admin/tests", strings.NewReader(`{"name":"Smoke","description":"Fast","config":"{\"steps\":[9]}","idProject":3}`))
	createRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	createResponse := httptest.NewRecorder()
	handler.CreateTest(createResponse, createRequest)
	if createResponse.Code != http.StatusOK || sessions.createdTest.Name != "Smoke" || sessions.createdTest.IDProject != 3 || !strings.Contains(sessions.createdTest.Config, "steps") {
		t.Fatalf("unexpected test create: %d %#v", createResponse.Code, sessions.createdTest)
	}

	invalidCreate := httptest.NewRequest(http.MethodPost, "/admin/tests", strings.NewReader(`{"name":"","description":"","config":"","idProject":0}`))
	invalidCreate.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.CreateTest(invalidResponse, invalidCreate)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "config") {
		t.Fatalf("expected validation failure, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	showRequest := httptest.NewRequest(http.MethodGet, "/admin/tests/3/12", nil)
	showRequest.SetPathValue("idProject", "3")
	showRequest.SetPathValue("test", "12")
	showRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	showResponse := httptest.NewRecorder()
	handler.ShowTest(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || !strings.Contains(showResponse.Body.String(), `"config":"{\"steps\":[9,10]}"`) {
		t.Fatalf("unexpected test detail: %d %s", showResponse.Code, showResponse.Body.String())
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/admin/tests/3/12", strings.NewReader(`{"config":"{\"steps\":[10,9]}"}`))
	updateRequest.SetPathValue("idProject", "3")
	updateRequest.SetPathValue("test", "12")
	updateRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	updateResponse := httptest.NewRecorder()
	handler.UpdateTest(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK || sessions.updatedTest.ID != 12 || sessions.updatedTest.IDProject != 3 || !strings.Contains(sessions.updatedTest.Config, "10") {
		t.Fatalf("unexpected test update: %d %#v", updateResponse.Code, sessions.updatedTest)
	}
}

func TestPostmanImportCreatesTestFromValidatedImportedSteps(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	payload := `{"name":"Imported","description":"Postman flow","idProject":3,"import":"[{\"name\":\"Open Home\",\"editorType\":\"postman\",\"steps\":[{\"stepType\":\"postman_collection\",\"collection\":{\"info\":{\"name\":\"Demo\"},\"item\":[]}}]}]"}`
	request := httptest.NewRequest(http.MethodPost, "/admin/importtest", strings.NewReader(payload))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.ImportTest(response, request)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"status":"ok"}` || sessions.importedTest.Name != "Imported" || sessions.importedTest.IDProject != 3 {
		t.Fatalf("unexpected import response: %d %s %#v", response.Code, response.Body.String(), sessions.importedTest)
	}

	missingCollection := httptest.NewRequest(http.MethodPost, "/admin/importtest", strings.NewReader(`{"name":"Imported","description":"Postman flow","idProject":3,"import":"[{\"name\":\"Open Home\",\"editorType\":\"postman\",\"steps\":[{\"stepType\":\"postman_collection\"}]}]"}`))
	missingCollection.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	missingCollectionResponse := httptest.NewRecorder()
	handler.ImportTest(missingCollectionResponse, missingCollection)
	if missingCollectionResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(missingCollectionResponse.Body.String(), "postman_collection") {
		t.Fatalf("expected postman validation failure, got %d %s", missingCollectionResponse.Code, missingCollectionResponse.Body.String())
	}

	emptyImport := httptest.NewRequest(http.MethodPost, "/admin/importtest", strings.NewReader(`{"name":"Imported","description":"Postman flow","idProject":3,"import":"[]"}`))
	emptyImport.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	emptyImportResponse := httptest.NewRecorder()
	handler.ImportTest(emptyImportResponse, emptyImport)
	if emptyImportResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(emptyImportResponse.Body.String(), "non-empty JSON array") {
		t.Fatalf("expected empty import validation failure, got %d %s", emptyImportResponse.Code, emptyImportResponse.Body.String())
	}
}

func TestPerformedResultExplorationUsesBrowserSessionAndPagination(t *testing.T) {
	postmanData := `[{"request":{"headers":{"Authorization":"[REDACTED]"}}}]`
	sessions := &sessionsStub{
		user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3},
		performedCycles: PerformedCyclePage{Data: []PerformedCycle{{ID: 44, TestCycleID: 5, Status: 1}}, Meta: ResultMeta{Pagination: ResultPaginationMeta{
			Page: 1, PerPage: 25, Total: 1, LastPage: 1, Sort: "date", Direction: "desc",
		}}},
		performedTests: PerformedTestPage{Data: []PerformedTest{{ID: 55, TestCycleDoneID: 44, TestID: 7, Status: 1, Name: "Checkout", PostmanData: &postmanData}}},
		performedSteps: []PerformedStep{{ID: 77, TestCycleDoneID: 44, TestDoneID: 55, Name: "Open", Status: 1, Screenshots: `[]`, Data: `{"token":"[REDACTED]"}`, Type: "selenium"}},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	cyclesRequest := httptest.NewRequest(http.MethodGet, "/admin/testcyclesperfomed/5?page=1&perPage=25&status=1", nil)
	cyclesRequest.SetPathValue("idTestCyclePerformed", "5")
	cyclesRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	cyclesResponse := httptest.NewRecorder()
	handler.PerformedCycles(cyclesResponse, cyclesRequest)
	if cyclesResponse.Code != http.StatusOK || !strings.Contains(cyclesResponse.Body.String(), `"pagination"`) || !strings.Contains(cyclesResponse.Body.String(), `"testCycleId":5`) {
		t.Fatalf("unexpected performed cycle response: %d %s", cyclesResponse.Code, cyclesResponse.Body.String())
	}

	testsRequest := httptest.NewRequest(http.MethodGet, "/admin/testsperfomed/44", nil)
	testsRequest.SetPathValue("idTestPerformed", "44")
	testsRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	testsResponse := httptest.NewRecorder()
	handler.PerformedTests(testsResponse, testsRequest)
	if testsResponse.Code != http.StatusOK || !strings.Contains(testsResponse.Body.String(), `"postmanData"`) || strings.Contains(testsResponse.Body.String(), "unsafe-token") {
		t.Fatalf("unexpected performed test response: %d %s", testsResponse.Code, testsResponse.Body.String())
	}

	stepsRequest := httptest.NewRequest(http.MethodGet, "/admin/stepsperfomed/55", nil)
	stepsRequest.SetPathValue("idTestPerformed", "55")
	stepsRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	stepsResponse := httptest.NewRecorder()
	handler.PerformedSteps(stepsResponse, stepsRequest)
	if stepsResponse.Code != http.StatusOK || !strings.Contains(stepsResponse.Body.String(), `"type":"selenium"`) || strings.Contains(stepsResponse.Body.String(), "unsafe-token") {
		t.Fatalf("unexpected performed step response: %d %s", stepsResponse.Code, stepsResponse.Body.String())
	}
}

func TestResultExportsCreateShowAndDownload(t *testing.T) {
	expiresAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	sessions := &sessionsStub{
		user:             User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3},
		exportDescriptor: ResultExportDescriptor{ID: 88, Format: "json", Status: "completed", Filename: "idelium-run-44.json", ContentType: "application/json", URL: "/api/admin/result-exports/88/download", ExpiresAt: &expiresAt, Authorized: true, Ready: true},
		exportDownload:   ResultExportDownload{Filename: "idelium-run-44.json", ContentType: "application/json", Payload: `{"schemaVersion":"result-export.v1"}`},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	createRequest := httptest.NewRequest(http.MethodPost, "/admin/result-exports", strings.NewReader(`{"performedTestCycleId":44,"format":"json"}`))
	createRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	createResponse := httptest.NewRecorder()
	handler.CreateResultExport(createResponse, createRequest)
	if createResponse.Code != http.StatusAccepted || sessions.createdExport.PerformedTestCycleID != 44 || !strings.Contains(createResponse.Body.String(), `"ready":true`) {
		t.Fatalf("unexpected export create: %d %s %#v", createResponse.Code, createResponse.Body.String(), sessions.createdExport)
	}

	showRequest := httptest.NewRequest(http.MethodGet, "/admin/result-exports/88", nil)
	showRequest.SetPathValue("resultExport", "88")
	showRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	showResponse := httptest.NewRecorder()
	handler.ShowResultExport(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || !strings.Contains(showResponse.Body.String(), `"filename":"idelium-run-44.json"`) {
		t.Fatalf("unexpected export show: %d %s", showResponse.Code, showResponse.Body.String())
	}

	downloadRequest := httptest.NewRequest(http.MethodGet, "/admin/result-exports/88/download", nil)
	downloadRequest.SetPathValue("resultExport", "88")
	downloadRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	downloadResponse := httptest.NewRecorder()
	handler.DownloadResultExport(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK || downloadResponse.Header().Get("Content-Disposition") != `attachment; filename="idelium-run-44.json"` || !strings.Contains(downloadResponse.Body.String(), "result-export.v1") {
		t.Fatalf("unexpected export download: %d headers=%#v body=%s", downloadResponse.Code, downloadResponse.Header(), downloadResponse.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/admin/result-exports", strings.NewReader(`{"performedTestCycleId":44,"format":"pdf"}`))
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.CreateResultExport(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "format") {
		t.Fatalf("expected invalid format validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var _ = time.Now
