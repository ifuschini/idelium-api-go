package browserauth

import (
	"bytes"
	"context"
	"encoding/json"
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
	artifacts         []ArtifactDescriptor
	artifact          ArtifactDescriptor
	createdArtifact   ArtifactDescriptorCreate
	lifecycleUpdate   ArtifactLifecycleUpdate
	gridSnapshotInput GridQuerySnapshotCreate
	gridJobInput      GridBulkJobCreate
	integrationInput  IntegrationEndpointCreate
	auditFilter       AuditEventFilter
	assetImpact       AssetImpact
	assetVersions     []AssetVersion
	assetVersion      AssetVersion
	reviewEvent       AssetReviewEvent
	reviewErr         error
	parallelRuns      []ParallelRun
	parallelRun       ParallelRun
	parallelInput     ParallelRunCreate
	parallelMatrix    []map[string]string
	parallelClaim     ParallelRunClaim
	parallelClaimErr  error
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
func (s *sessionsStub) ListArtifactDescriptors(*http.Request, User, int64, int64) ([]ArtifactDescriptor, error) {
	return s.artifacts, s.accountErr
}
func (s *sessionsStub) GetArtifactDescriptor(*http.Request, User, int64, int64, int64) (ArtifactDescriptor, error) {
	return s.artifact, s.accountErr
}
func (s *sessionsStub) RegisterArtifactDescriptor(_ *http.Request, _ User, input ArtifactDescriptorCreate) (ArtifactDescriptor, error) {
	s.createdArtifact = input
	return s.artifact, s.accountErr
}
func (s *sessionsStub) SetArtifactLegalHold(_ *http.Request, _ User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error) {
	s.lifecycleUpdate = input
	return s.artifact, s.accountErr
}
func (s *sessionsStub) MarkArtifactDeleted(_ *http.Request, _ User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error) {
	s.lifecycleUpdate = input
	return s.artifact, s.accountErr
}
func (s *sessionsStub) ArchiveArtifact(_ *http.Request, _ User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error) {
	s.lifecycleUpdate = input
	return s.artifact, s.accountErr
}
func (s *sessionsStub) RestoreArtifact(_ *http.Request, _ User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error) {
	s.lifecycleUpdate = input
	return s.artifact, s.accountErr
}
func (s *sessionsStub) CreateGridQuerySnapshot(_ *http.Request, _ User, input GridQuerySnapshotCreate) (GridQuerySnapshot, error) {
	s.gridSnapshotInput = input
	return GridQuerySnapshot{ID: "123e4567-e89b-42d3-a456-426614174000", ResourceType: "projects", Total: 1, ExpiresAt: time.Now().Add(15 * time.Minute)}, s.accountErr
}
func (s *sessionsStub) CreateGridBulkJob(_ *http.Request, _ User, input GridBulkJobCreate) (GridBulkJob, error) {
	s.gridJobInput = input
	return GridBulkJob{ID: "123e4567-e89b-42d3-a456-426614174001", ResourceType: "projects", Action: "export", Status: "completed", RequestedCount: 1, ProcessedCount: 1, Result: map[string]any{"exportAvailable": true}}, s.accountErr
}
func (s *sessionsStub) GetGridBulkJob(*http.Request, User, string) (GridBulkJob, error) {
	return GridBulkJob{ID: "123e4567-e89b-42d3-a456-426614174001", ResourceType: "projects", Action: "export", Status: "completed", RequestedCount: 1, ProcessedCount: 1, Result: map[string]any{"exportAvailable": true}}, s.accountErr
}
func (s *sessionsStub) ExportGridBulkJob(*http.Request, User, string, time.Time) (GridBulkExport, error) {
	return GridBulkExport{Filename: "idelium-projects-export.csv", Payload: "id,name\n1,Project\n"}, s.accountErr
}
func (s *sessionsStub) ListIntegrationEndpoints(*http.Request, User, int64) ([]IntegrationEndpoint, error) {
	return []IntegrationEndpoint{{ID: 9, IDProject: 5, Name: "Release events", Adapter: "webhook", URL: "https://93.184.216.34/hooks", Events: []string{"*"}, Status: "active", SecretConfigured: true, SchemaVersion: "2026-07-28.v1"}}, s.accountErr
}
func (s *sessionsStub) CreateIntegrationEndpoint(_ *http.Request, _ User, input IntegrationEndpointCreate) (IntegrationEndpoint, error) {
	s.integrationInput = input
	return IntegrationEndpoint{ID: 9, IDProject: input.ProjectID, Name: input.Name, Adapter: input.Adapter, URL: input.URL, Events: input.Events, Status: "active", SecretConfigured: true, SchemaVersion: "2026-07-28.v1"}, s.accountErr
}
func (s *sessionsStub) CreateIntegrationTestDelivery(*http.Request, User, int64, int64, time.Time) (IntegrationDelivery, error) {
	return IntegrationDelivery{ID: 20, DeliveryID: "idwh_test", Event: "integration.test", Status: "pending"}, s.accountErr
}
func (s *sessionsStub) UpdateIntegrationEndpointStatus(*http.Request, User, int64, int64, string, time.Time) (IntegrationEndpoint, error) {
	return IntegrationEndpoint{ID: 9, IDProject: 5, Name: "Release events", Status: "disabled", SecretConfigured: true}, s.accountErr
}
func (s *sessionsStub) RotateIntegrationEndpointSecret(*http.Request, User, int64, int64, string, time.Time) (IntegrationEndpoint, error) {
	return IntegrationEndpoint{ID: 9, IDProject: 5, Name: "Release events", Status: "active", SecretConfigured: true}, s.accountErr
}
func (s *sessionsStub) ListIntegrationDeliveries(*http.Request, User, int64, string) ([]IntegrationDelivery, error) {
	return []IntegrationDelivery{{ID: 20, DeliveryID: "idwh_test", Event: "integration.test", Status: "dead_letter", Attempts: 3}}, s.accountErr
}
func (s *sessionsStub) ReplayIntegrationDelivery(*http.Request, User, int64, int64, time.Time) (IntegrationDelivery, error) {
	return IntegrationDelivery{ID: 20, DeliveryID: "idwh_test", Event: "integration.test", Status: "pending", Attempts: 3}, s.accountErr
}
func (s *sessionsStub) ListAuditEvents(_ *http.Request, _ User, filter AuditEventFilter) ([]AuditEventRecord, error) {
	s.auditFilter = filter
	return []AuditEventRecord{{ID: 30, ActiveTenantID: 11, Action: "secret.changed", TargetType: "environment", CorrelationID: "018fb9d0-1f16-7abc-9f2f-4e1d8457f001", AfterValues: map[string]any{"apiKey": "[REDACTED]"}, Result: "success", CreatedAt: time.Now()}}, s.accountErr
}
func (s *sessionsStub) AssetImpact(*http.Request, User, int64, string, int64) (AssetImpact, error) {
	return s.assetImpact, s.accountErr
}
func (s *sessionsStub) ListAssetVersions(*http.Request, User, int64, string, int64) ([]AssetVersion, error) {
	return s.assetVersions, s.accountErr
}
func (s *sessionsStub) GetAssetVersion(*http.Request, User, int64, int64) (AssetVersion, error) {
	return s.assetVersion, s.accountErr
}
func (s *sessionsStub) TransitionAssetVersionReview(*http.Request, User, int64, int64, string, *string, time.Time) (AssetReviewEvent, error) {
	return s.reviewEvent, s.reviewErr
}
func (s *sessionsStub) ListParallelRuns(*http.Request, int64, int64, map[string]string) ([]ParallelRun, error) {
	return s.parallelRuns, s.accountErr
}
func (s *sessionsStub) CreateParallelRun(_ *http.Request, input ParallelRunCreate) (ParallelRun, error) {
	s.parallelInput = input
	return s.parallelRun, s.accountErr
}
func (s *sessionsStub) CreateParallelRunMatrix(_ *http.Request, input ParallelRunCreate, combinations []map[string]string) ([]ParallelRun, error) {
	s.parallelInput = input
	s.parallelMatrix = combinations
	return s.parallelRuns, s.accountErr
}
func (s *sessionsStub) GetParallelRun(*http.Request, int64, int64, int64) (ParallelRun, error) {
	return s.parallelRun, s.accountErr
}
func (s *sessionsStub) ClaimParallelRun(_ *http.Request, input ParallelRunClaim) (ParallelRun, error) {
	s.parallelClaim = input
	return s.parallelRun, s.parallelClaimErr
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

func TestArtifactDescriptorsRequireReadCapabilityAndPreserveEnvelope(t *testing.T) {
	metadata := json.RawMessage(`{"browser":"chrome","token":"[REDACTED]"}`)
	sessions := &sessionsStub{
		user:      User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3},
		artifacts: []ArtifactDescriptor{{ID: 99, IDCostumer: 11, IDProject: 5, PerformedTestCycleID: 44, ArtifactType: "json", Name: "summary.json", ContentType: "application/json", SizeBytes: 12, ChecksumSHA256: strings.Repeat("a", 64), StorageKey: "tenant/11/summary.json", State: "available", Metadata: metadata}},
		artifact:  ArtifactDescriptor{ID: 99, IDCostumer: 11, IDProject: 5, PerformedTestCycleID: 44, ArtifactType: "json", Name: "summary.json", ContentType: "application/json", SizeBytes: 12, ChecksumSHA256: strings.Repeat("a", 64), StorageKey: "tenant/11/summary.json", State: "available", Metadata: metadata},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	indexRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/performed-test-cycles/44/artifacts", nil)
	indexRequest.SetPathValue("idProject", "5")
	indexRequest.SetPathValue("performedTestCycleId", "44")
	indexRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	indexResponse := httptest.NewRecorder()
	handler.ArtifactDescriptors(indexResponse, indexRequest)
	if indexResponse.Code != http.StatusOK || !strings.Contains(indexResponse.Body.String(), `"data":[`) || !strings.Contains(indexResponse.Body.String(), `"artifactType":"json"`) {
		t.Fatalf("unexpected artifact index response: %d %s", indexResponse.Code, indexResponse.Body.String())
	}

	showRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/performed-test-cycles/44/artifacts/99", nil)
	showRequest.SetPathValue("idProject", "5")
	showRequest.SetPathValue("performedTestCycleId", "44")
	showRequest.SetPathValue("artifactDescriptor", "99")
	showRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	showResponse := httptest.NewRecorder()
	handler.ShowArtifactDescriptor(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || !strings.Contains(showResponse.Body.String(), `"data":{`) || !strings.Contains(showResponse.Body.String(), `"name":"summary.json"`) {
		t.Fatalf("unexpected artifact show response: %d %s", showResponse.Code, showResponse.Body.String())
	}

	forbiddenSessions := &sessionsStub{user: User{ID: 8, TenantID: 11, ActiveTenantID: 11, Role: 99}}
	forbiddenHandler := NewHandler(usersStub{}, forbiddenSessions, testLogger())
	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/performed-test-cycles/44/artifacts", nil)
	forbiddenRequest.SetPathValue("idProject", "5")
	forbiddenRequest.SetPathValue("performedTestCycleId", "44")
	forbiddenRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	forbiddenResponse := httptest.NewRecorder()
	forbiddenHandler.ArtifactDescriptors(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected artifacts.read to be required, got %d %s", forbiddenResponse.Code, forbiddenResponse.Body.String())
	}
}

func TestArtifactLifecycleHandlersValidateAndPreserveLaravelEnvelope(t *testing.T) {
	sessions := &sessionsStub{
		user:     User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2},
		artifact: ArtifactDescriptor{ID: 99, IDCostumer: 11, IDProject: 5, PerformedTestCycleID: 44, ArtifactType: "json", Name: "summary.json", ContentType: "application/json", SizeBytes: 12, ChecksumSHA256: strings.Repeat("a", 64), StorageKey: "tenant/11/summary.json", State: "archived", Metadata: json.RawMessage(`{"legalHold":{"enabled":true}}`)},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	handler.now = func() time.Time { return time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC) }

	holdRequest := httptest.NewRequest(http.MethodPut, "/admin/projects/5/performed-test-cycles/44/artifacts/99/legal-hold", strings.NewReader(`{"enabled":true,"reason":"Investigation hold"}`))
	holdRequest.SetPathValue("idProject", "5")
	holdRequest.SetPathValue("performedTestCycleId", "44")
	holdRequest.SetPathValue("artifactDescriptor", "99")
	holdRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	holdResponse := httptest.NewRecorder()
	handler.SetArtifactLegalHold(holdResponse, holdRequest)
	if holdResponse.Code != http.StatusOK || !strings.Contains(holdResponse.Body.String(), `"data":{`) || sessions.lifecycleUpdate.ProjectID != 5 || sessions.lifecycleUpdate.ArtifactDescriptorID != 99 || sessions.lifecycleUpdate.Reason == nil {
		t.Fatalf("unexpected legal hold response/update: %d %s %#v", holdResponse.Code, holdResponse.Body.String(), sessions.lifecycleUpdate)
	}

	sessions.accountErr = ValidationFailure{Errors: map[string][]string{"artifact": {"Artifact is under legal hold and cannot be archived."}}}
	archiveRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/performed-test-cycles/44/artifacts/99/archive", strings.NewReader(`{}`))
	archiveRequest.SetPathValue("idProject", "5")
	archiveRequest.SetPathValue("performedTestCycleId", "44")
	archiveRequest.SetPathValue("artifactDescriptor", "99")
	archiveRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	archiveResponse := httptest.NewRecorder()
	handler.ArchiveArtifact(archiveResponse, archiveRequest)
	if archiveResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(archiveResponse.Body.String(), "legal hold") {
		t.Fatalf("expected Laravel-compatible validation response, got %d %s", archiveResponse.Code, archiveResponse.Body.String())
	}
}

func TestGridSnapshotAndBulkJobHandlersPreserveLaravelContracts(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	handler.now = func() time.Time { return time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC) }

	snapshotRequest := httptest.NewRequest(http.MethodPost, "/admin/grid/query-snapshots", strings.NewReader(`{"resourceType":"projects","query":{"q":"Checkout","sort":"name","direction":"desc","f":{"id":5}}}`))
	snapshotRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	snapshotResponse := httptest.NewRecorder()
	handler.CreateGridQuerySnapshot(snapshotResponse, snapshotRequest)
	if snapshotResponse.Code != http.StatusCreated || sessions.gridSnapshotInput.Query.Search != "Checkout" || sessions.gridSnapshotInput.Query.Sort != "name" || !strings.Contains(snapshotResponse.Body.String(), `"data":{"id":`) {
		t.Fatalf("unexpected grid snapshot response/input: %d %s %#v", snapshotResponse.Code, snapshotResponse.Body.String(), sessions.gridSnapshotInput)
	}

	jobRequest := httptest.NewRequest(http.MethodPost, "/admin/grid/bulk-jobs", strings.NewReader(`{"querySnapshotId":"123e4567-e89b-42d3-a456-426614174000","action":"tag","payload":{"tags":["critical","critical","release-1"]}}`))
	jobRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	jobResponse := httptest.NewRecorder()
	handler.CreateGridBulkJob(jobResponse, jobRequest)
	if jobResponse.Code != http.StatusAccepted || len(sessions.gridJobInput.Tags) != 2 || sessions.gridJobInput.Tags[1] != "release-1" || !strings.Contains(jobResponse.Body.String(), `"status":"completed"`) {
		t.Fatalf("unexpected grid job response/input: %d %s %#v", jobResponse.Code, jobResponse.Body.String(), sessions.gridJobInput)
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/admin/grid/query-snapshots", strings.NewReader(`{"resourceType":"accounts","query":{"sort":"password"}}`))
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.CreateGridQuerySnapshot(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "resourceType") || !strings.Contains(invalidResponse.Body.String(), "query.sort") {
		t.Fatalf("expected bounded grid validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func TestGridBulkJobHandlersHideForeignResourcesAndExportCSV(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	jobID := "123e4567-e89b-42d3-a456-426614174001"

	showRequest := httptest.NewRequest(http.MethodGet, "/admin/grid/bulk-jobs/"+jobID, nil)
	showRequest.SetPathValue("jobId", jobID)
	showRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	showResponse := httptest.NewRecorder()
	handler.ShowGridBulkJob(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || strings.Contains(showResponse.Body.String(), "querySnapshotId") {
		t.Fatalf("unexpected grid job show response: %d %s", showResponse.Code, showResponse.Body.String())
	}

	exportRequest := httptest.NewRequest(http.MethodGet, "/admin/grid/bulk-jobs/"+jobID+"/export", nil)
	exportRequest.SetPathValue("jobId", jobID)
	exportRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	exportResponse := httptest.NewRecorder()
	handler.ExportGridBulkJob(exportResponse, exportRequest)
	if exportResponse.Code != http.StatusOK || exportResponse.Header().Get("Content-Type") != "text/csv; charset=UTF-8" || !strings.Contains(exportResponse.Body.String(), "Project") {
		t.Fatalf("unexpected grid export response: %d %#v %s", exportResponse.Code, exportResponse.Header(), exportResponse.Body.String())
	}

	sessions.accountErr = ErrNotFound
	foreignResponse := httptest.NewRecorder()
	handler.ShowGridBulkJob(foreignResponse, showRequest)
	if foreignResponse.Code != http.StatusNotFound {
		t.Fatalf("expected hidden foreign grid job, got %d %s", foreignResponse.Code, foreignResponse.Body.String())
	}
}

func TestIntegrationEndpointHandlersValidateCapabilitiesAndRedactSecrets(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	createRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/integrations", strings.NewReader(`{"name":"Release events","adapter":"webhook","url":"https://93.184.216.34/hooks/idelium","secret":"super-secret-value","events":["test.completed","test.completed"]}`))
	createRequest.SetPathValue("idProject", "5")
	createRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	createResponse := httptest.NewRecorder()
	handler.CreateIntegrationEndpoint(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || sessions.integrationInput.ProjectID != 5 || len(sessions.integrationInput.Events) != 1 || strings.Contains(createResponse.Body.String(), "super-secret-value") || strings.Contains(createResponse.Body.String(), "secretEncrypted") {
		t.Fatalf("unexpected integration create response/input: %d %s %#v", createResponse.Code, createResponse.Body.String(), sessions.integrationInput)
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/integrations", strings.NewReader(`{"name":"Unsafe","adapter":"webhook","url":"http://127.0.0.1/metadata","secret":"super-secret-value"}`))
	invalidRequest.SetPathValue("idProject", "5")
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.CreateIntegrationEndpoint(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "url") {
		t.Fatalf("expected SSRF validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	userSessions := &sessionsStub{user: User{ID: 8, TenantID: 11, ActiveTenantID: 11, Role: 3}}
	userHandler := NewHandler(usersStub{}, userSessions, testLogger())
	forbiddenResponse := httptest.NewRecorder()
	userHandler.CreateIntegrationEndpoint(forbiddenResponse, createRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected integrations.manage capability enforcement, got %d", forbiddenResponse.Code)
	}
}

func TestIntegrationDeliveryHandlersPreserveStatusAndRetryEnvelopes(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	testRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/integrations/9/test", nil)
	testRequest.SetPathValue("idProject", "5")
	testRequest.SetPathValue("integrationEndpoint", "9")
	testRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	testResponse := httptest.NewRecorder()
	handler.TestIntegrationEndpoint(testResponse, testRequest)
	if testResponse.Code != http.StatusAccepted || !strings.Contains(testResponse.Body.String(), `"status":"pending"`) {
		t.Fatalf("unexpected test delivery response: %d %s", testResponse.Code, testResponse.Body.String())
	}

	deliveriesRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/integration-deliveries?status=dead_letter", nil)
	deliveriesRequest.SetPathValue("idProject", "5")
	deliveriesRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	deliveriesResponse := httptest.NewRecorder()
	handler.IntegrationDeliveries(deliveriesResponse, deliveriesRequest)
	if deliveriesResponse.Code != http.StatusOK || !strings.Contains(deliveriesResponse.Body.String(), `"deliveryId":"idwh_test"`) {
		t.Fatalf("unexpected delivery list response: %d %s", deliveriesResponse.Code, deliveriesResponse.Body.String())
	}

	replayRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/integration-deliveries/20/replay", nil)
	replayRequest.SetPathValue("idProject", "5")
	replayRequest.SetPathValue("integrationDelivery", "20")
	replayRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	replayResponse := httptest.NewRecorder()
	handler.ReplayIntegrationDelivery(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusAccepted || !strings.Contains(replayResponse.Body.String(), `"status":"pending"`) {
		t.Fatalf("unexpected delivery replay response: %d %s", replayResponse.Code, replayResponse.Body.String())
	}
}

func TestAuditEventHandlerValidatesFiltersAndRequiresCapability(t *testing.T) {
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2}}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/audit-events?action=secret.changed&correlationId=018fb9d0-1f16-7abc-9f2f-4e1d8457f001&from=2026-08-01&limit=50", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.AuditEvents(response, request)
	if response.Code != http.StatusOK || sessions.auditFilter.Action != "secret.changed" || sessions.auditFilter.Limit != 50 || sessions.auditFilter.From == nil || !strings.Contains(response.Body.String(), `"apiKey":"[REDACTED]"`) {
		t.Fatalf("unexpected audit list response/filter: %d %s %#v", response.Code, response.Body.String(), sessions.auditFilter)
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/audit-events?correlationId=not-a-uuid&limit=201", nil)
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.AuditEvents(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "correlationId") || !strings.Contains(invalidResponse.Body.String(), "limit") {
		t.Fatalf("expected audit filter validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	forbiddenSessions := &sessionsStub{user: User{ID: 8, TenantID: 11, ActiveTenantID: 11, Role: 99}}
	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	forbiddenRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	forbiddenResponse := httptest.NewRecorder()
	NewHandler(usersStub{}, forbiddenSessions, testLogger()).AuditEvents(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected audit_events.read capability enforcement, got %d", forbiddenResponse.Code)
	}
}

func TestAssetImpactAndVersionHandlersValidateCapabilityAndPayload(t *testing.T) {
	impact := AssetImpact{Tests: []AssetImpactItem{{ID: 7, Name: "Checkout"}}}
	impact.Asset.AssetType = "environment"
	impact.Asset.AssetID = 3
	impact.Summary.Tests = 1
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}, assetImpact: impact, assetVersions: []AssetVersion{{ID: 20, AssetType: "test", AssetID: 7, Version: 2}}}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	impactRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/asset-impact/environment/3", nil)
	impactRequest.SetPathValue("idProject", "5")
	impactRequest.SetPathValue("assetType", "environment")
	impactRequest.SetPathValue("assetId", "3")
	impactRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	impactResponse := httptest.NewRecorder()
	handler.AssetImpact(impactResponse, impactRequest)
	if impactResponse.Code != http.StatusOK || !strings.Contains(impactResponse.Body.String(), `"assetType":"environment"`) || !strings.Contains(impactResponse.Body.String(), `"name":"Checkout"`) {
		t.Fatalf("unexpected asset impact response: %d %s", impactResponse.Code, impactResponse.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/admin/projects/5/asset-versions/plugin/3", nil)
	invalidRequest.SetPathValue("idProject", "5")
	invalidRequest.SetPathValue("assetType", "plugin")
	invalidRequest.SetPathValue("assetId", "3")
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.AssetVersions(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "assetType") {
		t.Fatalf("expected asset type validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	forbiddenSessions := &sessionsStub{user: User{ID: 8, TenantID: 11, ActiveTenantID: 11, Role: 99}}
	forbiddenResponse := httptest.NewRecorder()
	NewHandler(usersStub{}, forbiddenSessions, testLogger()).AssetImpact(forbiddenResponse, impactRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected resources.read capability enforcement, got %d", forbiddenResponse.Code)
	}
}

func TestAssetReviewTransitionAndDiffContracts(t *testing.T) {
	sessions := &sessionsStub{
		user:        User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 2},
		reviewEvent: AssetReviewEvent{ID: 31, AssetVersionID: 20, FromStatus: "draft", ToStatus: "in_review"},
	}
	handler := NewHandler(usersStub{}, sessions, testLogger())
	reviewRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/asset-versions/20/review-events", strings.NewReader(`{"toStatus":"in_review","comment":"Ready"}`))
	reviewRequest.SetPathValue("idProject", "5")
	reviewRequest.SetPathValue("assetVersion", "20")
	reviewRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	reviewResponse := httptest.NewRecorder()
	handler.TransitionAssetVersionReview(reviewResponse, reviewRequest)
	if reviewResponse.Code != http.StatusCreated || !strings.Contains(reviewResponse.Body.String(), `"toStatus":"in_review"`) {
		t.Fatalf("unexpected review response: %d %s", reviewResponse.Code, reviewResponse.Body.String())
	}

	sessions.reviewErr = ReviewFailure{Message: "The requested review transition is not allowed.", FromStatus: "approved", ToStatus: "in_review"}
	invalidRequest := httptest.NewRequest(http.MethodPost, "/admin/projects/5/asset-versions/20/review-events", strings.NewReader(`{"toStatus":"in_review"}`))
	invalidRequest.SetPathValue("idProject", "5")
	invalidRequest.SetPathValue("assetVersion", "20")
	invalidRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.TransitionAssetVersionReview(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), `"fromStatus":"approved"`) {
		t.Fatalf("unexpected invalid transition response: %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	fromSnapshot := map[string]any{"removed": true, "changed": "old"}
	toSnapshot := map[string]any{"added": 2.0, "changed": "new"}
	diff := assetVersionDiff(AssetVersion{ID: 20, AssetType: "test", AssetID: 7, Version: 1, Snapshot: &fromSnapshot}, AssetVersion{ID: 21, AssetType: "test", AssetID: 7, Version: 2, Snapshot: &toSnapshot})
	changes := diff["changes"].(map[string]any)
	if changes["added"].(map[string]any)["added"] != 2.0 || changes["removed"].(map[string]any)["removed"] != true || changes["changed"].(map[string]any)["changed"].(map[string]any)["to"] != "new" {
		t.Fatalf("unexpected asset diff: %#v", diff)
	}
}

func TestParallelRunScheduleAndMatrixHandlersPreserveContracts(t *testing.T) {
	run := ParallelRun{ID: 40, IDProject: 5, TestCycleID: 7, IdempotencyKey: "release", Status: "queued", RequestedConcurrency: 2, Metadata: map[string]any{"run": map[string]any{"pipeline": "release"}}, ResultSummary: map[string]any{}}
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}, parallelRun: run, parallelRuns: []ParallelRun{run, run, run, run}}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	create := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs", strings.NewReader(`{"testCycleId":7,"idempotencyKey":"release","requestedConcurrency":2,"metadata":{"pipeline":"release","token":"remove-me"}}`))
	create.SetPathValue("idProject", "5")
	create.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	created := httptest.NewRecorder()
	handler.CreateParallelRun(created, create)
	if created.Code != http.StatusCreated || sessions.parallelInput.TenantID != 11 || sessions.parallelInput.ProjectID != 5 || sessions.parallelInput.Metadata["token"] != nil {
		t.Fatalf("unexpected parallel run create: %d %s %#v", created.Code, created.Body.String(), sessions.parallelInput)
	}

	matrix := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/matrix", strings.NewReader(`{"testCycleId":7,"idempotencyKey":"matrix-release","matrix":{"platforms":["linux"],"browsers":["chrome","firefox"],"environments":["demo","prod"]}}`))
	matrix.SetPathValue("idProject", "5")
	matrix.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	matrixResponse := httptest.NewRecorder()
	handler.CreateParallelRunMatrix(matrixResponse, matrix)
	if matrixResponse.Code != http.StatusCreated || len(sessions.parallelMatrix) != 4 || !strings.Contains(matrixResponse.Body.String(), `"requestedRuns":4`) {
		t.Fatalf("unexpected matrix create: %d %s %#v", matrixResponse.Code, matrixResponse.Body.String(), sessions.parallelMatrix)
	}

	tooLarge := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/matrix", strings.NewReader(`{"testCycleId":7,"idempotencyKey":"large","matrix":{"platforms":[1,2,3,4,5,6,7,8],"browsers":[1,2,3,4,5,6,7,8,9]}}`))
	tooLarge.SetPathValue("idProject", "5")
	tooLarge.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	tooLargeResponse := httptest.NewRecorder()
	handler.CreateParallelRunMatrix(tooLargeResponse, tooLarge)
	if tooLargeResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(tooLargeResponse.Body.String(), `"requestedRuns":72`) {
		t.Fatalf("expected matrix bound, got %d %s", tooLargeResponse.Code, tooLargeResponse.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs", strings.NewReader(`{"testCycleId":7,"idempotencyKey":"release","requestedConcurrency":33}`))
	invalid.SetPathValue("idProject", "5")
	invalid.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	handler.CreateParallelRun(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), `"requestedConcurrency"`) {
		t.Fatalf("expected concurrency validation error, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	sessions.accountErr = errors.New("database password must not be returned")
	failure := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs", strings.NewReader(`{"testCycleId":7,"idempotencyKey":"safe-error"}`))
	failure.SetPathValue("idProject", "5")
	failure.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	failureResponse := httptest.NewRecorder()
	handler.CreateParallelRun(failureResponse, failure)
	if failureResponse.Code != http.StatusInternalServerError || strings.Contains(failureResponse.Body.String(), "database password") {
		t.Fatalf("expected redacted internal error, got %d %s", failureResponse.Code, failureResponse.Body.String())
	}
}

func TestParallelRunMetadataRecursivelyRemovesSensitiveValues(t *testing.T) {
	metadata := normalizeRunMetadata(map[string]any{"build": 1042.0, "token": "remove", "nested": []any{map[string]any{"password": "remove", "safe": "keep"}}, "workloadIdentity": map[string]any{"provider": "github-actions", "authorization": "remove"}})
	encoded, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "remove") || !strings.Contains(string(encoded), `"build":"1042"`) || !strings.Contains(string(encoded), `"safe":"keep"`) {
		t.Fatalf("unexpected normalized metadata: %s", encoded)
	}
}

func TestParallelRunClaimRequiresTokenAndPreservesWorkerContract(t *testing.T) {
	run := ParallelRun{ID: 40, IDProject: 5, TestCycleID: 7, IdempotencyKey: "release", Status: "running", RequestedConcurrency: 1, ActiveWorkers: 1, TotalWorkers: 1, Metadata: map[string]any{}, ResultSummary: []any{}}
	sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}, parallelRun: run}
	handler := NewHandler(usersStub{}, sessions, testLogger())

	missingToken := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/40/claim", strings.NewReader(`{"workerId":"worker-a"}`))
	missingToken.SetPathValue("idProject", "5")
	missingToken.SetPathValue("parallelRun", "40")
	missingToken.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	missingTokenResponse := httptest.NewRecorder()
	handler.ClaimParallelRun(missingTokenResponse, missingToken)
	if missingTokenResponse.Code != http.StatusUnauthorized || !strings.Contains(missingTokenResponse.Body.String(), "short-lived run token") {
		t.Fatalf("expected required run token, got %d %s", missingTokenResponse.Code, missingTokenResponse.Body.String())
	}

	claim := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/40/claim", strings.NewReader(`{"workerId":"worker-a","capabilities":["selenium"]}`))
	claim.SetPathValue("idProject", "5")
	claim.SetPathValue("parallelRun", "40")
	claim.Header.Set("Idelium-Run-Token", "idrt_public.secret-value")
	claim.Header.Set("Idelium-Agent-Cert-Sha256", strings.Repeat("a", 64))
	claim.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	response := httptest.NewRecorder()
	handler.ClaimParallelRun(response, claim)
	if response.Code != http.StatusOK || sessions.parallelClaim.TenantID != 11 || sessions.parallelClaim.RunID != 40 || sessions.parallelClaim.WorkerID != "worker-a" || !sessions.parallelClaim.CapabilitiesSet {
		t.Fatalf("unexpected parallel run claim: %d %s %#v", response.Code, response.Body.String(), sessions.parallelClaim)
	}
}

func TestParallelRunClaimValidationAndSafeErrors(t *testing.T) {
	t.Setenv("IDELIUM_RUN_TOKEN_REQUIRED_FOR_CLAIM", "false")
	for _, test := range []struct {
		name       string
		err        error
		status     int
		contains   string
		notContain string
	}{
		{name: "concurrency", err: ErrParallelRunConcurrency, status: http.StatusConflict, contains: "Concurrency limit reached."},
		{name: "terminal", err: ErrParallelRunTerminal, status: http.StatusUnprocessableEntity, contains: "already terminal"},
		{name: "token", err: ErrRunTokenInvalid, status: http.StatusUnprocessableEntity, contains: `"runToken"`},
		{name: "internal", err: errors.New("database password must not be returned"), status: http.StatusInternalServerError, contains: "could not be completed", notContain: "database password"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sessions := &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11, Role: 3}, parallelClaimErr: test.err}
			handler := NewHandler(usersStub{}, sessions, testLogger())
			request := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/40/claim", strings.NewReader(`{"workerId":"worker-a"}`))
			request.SetPathValue("idProject", "5")
			request.SetPathValue("parallelRun", "40")
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
			response := httptest.NewRecorder()
			handler.ClaimParallelRun(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.contains) || (test.notContain != "" && strings.Contains(response.Body.String(), test.notContain)) {
				t.Fatalf("unexpected claim error response: %d %s", response.Code, response.Body.String())
			}
		})
	}

	invalid := httptest.NewRequest(http.MethodPost, "/admin/projects/5/parallel-runs/40/claim", strings.NewReader(`{"workerId":""}`))
	invalid.SetPathValue("idProject", "5")
	invalid.SetPathValue("parallelRun", "40")
	invalid.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "opaque-value"})
	invalidResponse := httptest.NewRecorder()
	NewHandler(usersStub{}, &sessionsStub{user: User{ID: 7, TenantID: 11, ActiveTenantID: 11}}, testLogger()).ClaimParallelRun(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), `"workerId"`) {
		t.Fatalf("expected worker validation, got %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var _ = time.Now
