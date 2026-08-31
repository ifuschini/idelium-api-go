package app

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/buildinfo"
	"github.com/idelium/idelium-api-go/internal/cliapi"
	"github.com/idelium/idelium-api-go/internal/platforms"
)

type readyChecker struct{}

func (readyChecker) Check(context.Context) error { return nil }

type fakeCatalogRepository struct{}

func (fakeCatalogRepository) ListTypes(context.Context) ([]platforms.CatalogItem, error) {
	return []platforms.CatalogItem{{ID: 1, Name: "desktop"}}, nil
}

func (fakeCatalogRepository) ListStatuses(context.Context) ([]platforms.CatalogItem, error) {
	return []platforms.CatalogItem{{ID: 1, Name: "free"}}, nil
}

func (fakeCatalogRepository) ListLocations(context.Context, platforms.LocationQuery) (platforms.LocationPage, error) {
	return platforms.LocationPage{
		Data: []platforms.LocationItem{{ID: 1, Name: "eu-west"}},
	}, nil
}

func (fakeCatalogRepository) ListBrands(context.Context, platforms.BrandQuery) (platforms.BrandPage, error) {
	return platforms.BrandPage{
		Data: []platforms.BrandItem{{ID: 1, Brand: "Apple"}},
	}, nil
}

func (fakeCatalogRepository) ListModels(context.Context, platforms.ModelQuery) (platforms.ModelPage, error) {
	return platforms.ModelPage{
		Data: []platforms.ModelItem{{ID: 1, Model: "iPhone", IDBrand: 1}},
	}, nil
}

func (fakeCatalogRepository) ListOperatingSystems(context.Context, platforms.OperatingSystemQuery) (platforms.OperatingSystemPage, error) {
	return platforms.OperatingSystemPage{
		Data: []platforms.OperatingSystemItem{{ID: 1, Name: "linux", Type: 1}},
	}, nil
}

func (fakeCatalogRepository) ListOperatingSystemVersions(context.Context, platforms.OperatingSystemVersionQuery) (platforms.OperatingSystemVersionPage, error) {
	return platforms.OperatingSystemVersionPage{
		Data: []platforms.OperatingSystemVersionItem{{ID: 1, Version: "14", IDOs: 1}},
	}, nil
}

func (fakeCatalogRepository) ListBrowsers(context.Context, platforms.BrowserQuery) (platforms.BrowserPage, error) {
	return platforms.BrowserPage{
		Data: []platforms.BrowserItem{{ID: 1, Name: "chrome", IDOs: 1}},
	}, nil
}

func (fakeCatalogRepository) ListBrowserVersions(context.Context, platforms.BrowserVersionQuery) (platforms.BrowserVersionPage, error) {
	return platforms.BrowserVersionPage{
		Data: []platforms.BrowserVersionItem{{ID: 1, Version: "124", IDBrowser: 1}},
	}, nil
}

func (fakeCatalogRepository) ListManagedPlatforms(context.Context, platforms.ManagedPlatformQuery) (platforms.ManagedPlatformPage, error) {
	return platforms.ManagedPlatformPage{
		Data: []platforms.ManagedPlatformItem{{ID: 6, Type: 1, Hostname: "https://node.example:4444", BrowserDescription: "chrome"}},
	}, nil
}

func (fakeCatalogRepository) ListLaunchTargets(context.Context, int64) ([]platforms.LaunchTargetItem, error) {
	return []platforms.LaunchTargetItem{{
		ID:           "platform-pool",
		Name:         "Platform pool",
		Type:         "platform-pool",
		Runtime:      "selenium",
		Capabilities: []string{"browserOverride", "parallel"},
		Capacity:     platforms.LaunchTargetCapacity{Available: 1, Max: 1, Queued: 0},
		Health:       "healthy",
	}}, nil
}

type fakeLegacyKeyRepository struct{}

type fakeBrowserAuthRepository struct{}

func (fakeBrowserAuthRepository) FindByEmail(context.Context, string) (browserauth.User, error) {
	return browserauth.User{}, browserauth.ErrNotFound
}
func (fakeBrowserAuthRepository) Create(context.Context, browserauth.Session) error { return nil }
func (fakeBrowserAuthRepository) Delete(context.Context, string) error {
	return browserauth.ErrNotFound
}

func (fakeBrowserAuthRepository) Get(context.Context, string, time.Time) (browserauth.User, error) {
	return browserauth.User{}, browserauth.ErrNotFound
}
func (fakeBrowserAuthRepository) ListProjects(context.Context, int64) ([]browserauth.Project, error) {
	return nil, nil
}
func (fakeBrowserAuthRepository) ListCustomers(context.Context) ([]browserauth.Customer, error) {
	return nil, nil
}
func (fakeBrowserAuthRepository) CustomerExists(context.Context, int64) (bool, error) {
	return false, nil
}
func (fakeBrowserAuthRepository) SwitchTenant(context.Context, browserauth.TenantSwitch) error {
	return nil
}
func (fakeBrowserAuthRepository) RecordTenantSwitch(context.Context, browserauth.AuditEvent) error {
	return nil
}
func (fakeBrowserAuthRepository) ListRoles(*http.Request, browserauth.User) ([]browserauth.Role, bool, error) {
	return nil, false, nil
}
func (fakeBrowserAuthRepository) Profile(*http.Request, browserauth.User) (browserauth.Profile, error) {
	return browserauth.Profile{}, nil
}
func (fakeBrowserAuthRepository) UpdateProfilePassword(*http.Request, browserauth.User, string) (browserauth.Profile, error) {
	return browserauth.Profile{}, nil
}
func (fakeBrowserAuthRepository) ListAccounts(*http.Request, browserauth.User, browserauth.AccountQuery) (browserauth.AccountPage, error) {
	return browserauth.AccountPage{}, nil
}
func (fakeBrowserAuthRepository) CreateAccount(*http.Request, browserauth.User, browserauth.AccountCreate) error {
	return nil
}
func (fakeBrowserAuthRepository) UpdateAccount(*http.Request, browserauth.User, browserauth.AccountUpdate) error {
	return nil
}
func (fakeBrowserAuthRepository) DeleteAccount(*http.Request, browserauth.User, int64) error {
	return nil
}
func (fakeBrowserAuthRepository) ListAdminCustomers(*http.Request, browserauth.CustomerQuery) (browserauth.CustomerPage, error) {
	return browserauth.CustomerPage{}, nil
}
func (fakeBrowserAuthRepository) CreateCustomer(*http.Request, browserauth.CustomerCreate) error {
	return nil
}
func (fakeBrowserAuthRepository) UpdateCustomer(*http.Request, browserauth.CustomerUpdate) error {
	return nil
}
func (fakeBrowserAuthRepository) DeleteCustomer(*http.Request, int64) error {
	return nil
}
func (fakeBrowserAuthRepository) ListTestCycles(*http.Request, browserauth.User, browserauth.ResourceQuery) (browserauth.TestCyclePage, error) {
	return browserauth.TestCyclePage{}, nil
}
func (fakeBrowserAuthRepository) CreateTestCycle(*http.Request, browserauth.User, browserauth.TestCycleCreate) error {
	return nil
}
func (fakeBrowserAuthRepository) GetTestCycle(*http.Request, browserauth.User, int64, int64) (browserauth.TestCycleDetail, error) {
	return browserauth.TestCycleDetail{}, nil
}
func (fakeBrowserAuthRepository) UpdateTestCycle(*http.Request, browserauth.User, browserauth.TestCycleUpdate) error {
	return nil
}
func (fakeBrowserAuthRepository) ReorderSteps(*http.Request, browserauth.User, browserauth.StepReorder) error {
	return nil
}
func (fakeBrowserAuthRepository) ListStepsForReorder(*http.Request, browserauth.User, browserauth.ResourceQuery) (browserauth.StepPage, error) {
	return browserauth.StepPage{}, nil
}
func (fakeBrowserAuthRepository) ListTests(*http.Request, browserauth.User, browserauth.ResourceQuery) (browserauth.TestPage, error) {
	return browserauth.TestPage{}, nil
}
func (fakeBrowserAuthRepository) CreateTest(*http.Request, browserauth.User, browserauth.TestCreate) error {
	return nil
}
func (fakeBrowserAuthRepository) GetTest(*http.Request, browserauth.User, int64, int64) (browserauth.TestDetail, error) {
	return browserauth.TestDetail{}, nil
}
func (fakeBrowserAuthRepository) UpdateTest(*http.Request, browserauth.User, browserauth.TestUpdate) error {
	return nil
}
func (fakeBrowserAuthRepository) ImportTest(*http.Request, browserauth.User, browserauth.TestImport) error {
	return nil
}
func (fakeBrowserAuthRepository) ListPerformedCycles(*http.Request, browserauth.User, browserauth.ResultQuery) (browserauth.PerformedCyclePage, error) {
	return browserauth.PerformedCyclePage{}, nil
}
func (fakeBrowserAuthRepository) ListPerformedTests(*http.Request, browserauth.User, browserauth.ResultQuery) (browserauth.PerformedTestPage, error) {
	return browserauth.PerformedTestPage{}, nil
}
func (fakeBrowserAuthRepository) ListPerformedSteps(*http.Request, browserauth.User, int64) ([]browserauth.PerformedStep, error) {
	return nil, nil
}
func (fakeBrowserAuthRepository) CreateResultExport(*http.Request, browserauth.User, browserauth.ResultExportCreate) (browserauth.ResultExportDescriptor, error) {
	return browserauth.ResultExportDescriptor{}, nil
}
func (fakeBrowserAuthRepository) GetResultExport(*http.Request, browserauth.User, int64) (browserauth.ResultExportDescriptor, error) {
	return browserauth.ResultExportDescriptor{}, nil
}
func (fakeBrowserAuthRepository) DownloadResultExport(*http.Request, browserauth.User, int64, time.Time) (browserauth.ResultExportDownload, error) {
	return browserauth.ResultExportDownload{}, nil
}
func (fakeBrowserAuthRepository) ListArtifactDescriptors(*http.Request, browserauth.User, int64, int64) ([]browserauth.ArtifactDescriptor, error) {
	return nil, nil
}
func (fakeBrowserAuthRepository) GetArtifactDescriptor(*http.Request, browserauth.User, int64, int64, int64) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}
func (fakeBrowserAuthRepository) RegisterArtifactDescriptor(*http.Request, browserauth.User, browserauth.ArtifactDescriptorCreate) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}
func (fakeBrowserAuthRepository) SetArtifactLegalHold(*http.Request, browserauth.User, browserauth.ArtifactLifecycleUpdate) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}
func (fakeBrowserAuthRepository) MarkArtifactDeleted(*http.Request, browserauth.User, browserauth.ArtifactLifecycleUpdate) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}
func (fakeBrowserAuthRepository) ArchiveArtifact(*http.Request, browserauth.User, browserauth.ArtifactLifecycleUpdate) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}
func (fakeBrowserAuthRepository) RestoreArtifact(*http.Request, browserauth.User, browserauth.ArtifactLifecycleUpdate) (browserauth.ArtifactDescriptor, error) {
	return browserauth.ArtifactDescriptor{}, nil
}

func (fakeLegacyKeyRepository) AuthenticateLegacyCustomerKey(ctx context.Context, key string, usedAt time.Time) (auth.Customer, error) {
	if key != "valid-key" {
		return auth.Customer{}, auth.ErrInvalidLegacyKey
	}
	return auth.Customer{ID: 42, Name: "demo"}, nil
}

type fakeTestCycleRepository struct{}

func (fakeTestCycleRepository) GetTestCycle(ctx context.Context, customerID int64, testCycleID int64) (cliapi.TestCycle, error) {
	if customerID != 42 || testCycleID != 7 {
		return cliapi.TestCycle{}, cliapi.ErrNotFound
	}
	return cliapi.TestCycle{
		ID:          7,
		Name:        "nightly",
		Description: "Nightly cycle",
		Config:      "[]",
		IDProject:   3,
		IDCostumer:  42,
	}, nil
}

type fakePerformedCycleRepository struct{}

func (fakePerformedCycleRepository) CreatePerformedCycle(ctx context.Context, customerID int64, command cliapi.CreatePerformedCycleRequest) (int64, error) {
	if customerID != 42 || command.TestCycleID != 7 {
		return 0, cliapi.ErrNotFound
	}
	return 44, nil
}

func (fakePerformedCycleRepository) UpdatePerformedCycle(ctx context.Context, customerID int64, command cliapi.UpdatePerformedCycleRequest) (int64, error) {
	if customerID != 42 || command.TestCycleID != 44 {
		return 0, cliapi.ErrNotFound
	}
	return command.TestCycleID, nil
}

type fakeTestRepository struct{}

func (fakeTestRepository) GetTest(ctx context.Context, customerID int64, testID int64) (cliapi.Test, error) {
	if customerID != 42 || testID != 9 {
		return cliapi.Test{}, cliapi.ErrNotFound
	}
	return cliapi.Test{
		ID:          9,
		Name:        "browser test",
		Description: "Browser coverage",
		Config:      "[]",
		IDProject:   3,
		IDCostumer:  42,
	}, nil
}

type fakePerformedTestRepository struct{}

func (fakePerformedTestRepository) CreatePerformedTest(ctx context.Context, customerID int64, command cliapi.CreatePerformedTestRequest) (int64, error) {
	if customerID != 42 || command.TestCycleID != 7 || command.TestID != 9 {
		return 0, cliapi.ErrNotFound
	}
	return 55, nil
}

func (fakePerformedTestRepository) UpdatePerformedTest(ctx context.Context, customerID int64, command cliapi.UpdatePerformedTestRequest) (int64, error) {
	if customerID != 42 || command.TestID != 55 {
		return 0, cliapi.ErrNotFound
	}
	return command.TestID, nil
}

type fakePerformedStepRepository struct{}

func (fakePerformedStepRepository) CreatePerformedStep(ctx context.Context, customerID int64, command cliapi.CreatePerformedStepRequest) (int64, error) {
	if customerID != 42 || command.TestCycleID != 44 || command.TestID != 55 || command.StepID != 12 {
		return 0, cliapi.ErrNotFound
	}
	return 77, nil
}

func (fakePerformedStepRepository) UpdatePerformedStep(ctx context.Context, customerID int64, command cliapi.UpdatePerformedStepRequest) (int64, error) {
	if customerID != 42 || command.StepID != 77 {
		return 0, cliapi.ErrNotFound
	}
	return command.StepID, nil
}

type fakeStepRepository struct{}

func (fakeStepRepository) GetStep(ctx context.Context, customerID int64, stepID int64) (cliapi.Step, error) {
	if customerID != 42 || stepID != 12 {
		return cliapi.Step{}, cliapi.ErrNotFound
	}
	return cliapi.Step{
		ID:          12,
		Name:        "open page",
		Description: "Open the browser",
		Config:      "[]",
		IDProject:   3,
		Order:       2,
		IDCostumer:  42,
	}, nil
}

type fakePluginRepository struct{}

func (fakePluginRepository) ListPlugins(ctx context.Context, customerID int64, projectID int64) ([]cliapi.Plugin, error) {
	if customerID != 42 || projectID != 3 {
		return []cliapi.Plugin{}, nil
	}
	return []cliapi.Plugin{{
		ID:          14,
		Name:        "python wrapper",
		Code:        "{}",
		Description: "Plugin manifest",
		IDProject:   3,
		IDCostumer:  42,
	}}, nil
}

func (fakePluginRepository) GetPlugin(ctx context.Context, customerID int64, pluginID int64) (cliapi.Plugin, error) {
	if customerID != 42 || pluginID != 14 {
		return cliapi.Plugin{}, cliapi.ErrNotFound
	}
	return cliapi.Plugin{
		ID:          14,
		Name:        "python wrapper",
		Code:        "{}",
		Description: "Plugin manifest",
		IDProject:   3,
		IDCostumer:  42,
	}, nil
}

type fakeEnvironmentRepository struct{}

func (fakeEnvironmentRepository) ListEnvironments(ctx context.Context, customerID int64, projectID int64) ([]cliapi.Environment, error) {
	if customerID != 42 || projectID != 3 {
		return []cliapi.Environment{}, nil
	}
	return []cliapi.Environment{{
		ID:          16,
		Code:        "demo",
		Description: "Demo environment",
		Config:      "{}",
		IDProject:   3,
		IDCostumer:  42,
	}}, nil
}

func (fakeEnvironmentRepository) GetEnvironment(ctx context.Context, customerID int64, environmentID int64) (cliapi.Environment, error) {
	if customerID != 42 || environmentID != 16 {
		return cliapi.Environment{}, cliapi.ErrNotFound
	}
	return cliapi.Environment{
		ID:          16,
		Code:        "demo",
		Description: "Demo environment",
		Config:      "{}",
		IDProject:   3,
		IDCostumer:  42,
	}, nil
}

func testRouter(logger *slog.Logger) http.Handler {
	return NewRouter(
		logger,
		readyChecker{},
		buildinfo.Current(),
		fakeCatalogRepository{},
		fakeLegacyKeyRepository{},
		fakeTestCycleRepository{},
		fakePerformedCycleRepository{},
		fakeTestRepository{},
		fakePerformedTestRepository{},
		fakePerformedStepRepository{},
		fakeStepRepository{},
		fakePluginRepository{},
		fakeEnvironmentRepository{},
		fakeBrowserAuthRepository{},
	)
}

func TestRouterReturnsStableNotFoundResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "ROUTE_NOT_FOUND") {
		t.Fatalf("stable error code missing: %s", response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("secure response headers were not applied")
	}
}

func TestRouterRejectsUnsupportedMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/health/live", nil))

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "METHOD_NOT_ALLOWED") {
		t.Fatalf("stable error code missing: %s", response.Body.String())
	}
}

func TestRouterFailsClosedForAdvancedIdentityRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/sso/7/oidc/callback", strings.NewReader("id_token=must-not-leak")),
	)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "IDENTITY_MIGRATION_DISABLED") {
		t.Fatalf("stable identity migration code missing: %s", body)
	}
	if strings.Contains(body, "must-not-leak") || strings.Contains(body, "id_token") {
		t.Fatalf("identity route leaked callback payload: %s", body)
	}
}

func TestRouterFailsClosedForServiceAccountRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/admin/service-accounts", strings.NewReader(`{"token":"must-not-leak"}`)),
	)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "SERVICE_ACCOUNT_MIGRATION_DISABLED") {
		t.Fatalf("stable service-account migration code missing: %s", body)
	}
	if strings.Contains(body, "must-not-leak") || strings.Contains(body, "token") {
		t.Fatalf("service-account route leaked credential payload: %s", body)
	}
}

func TestRouterFailsClosedForLegacyAPIKeyRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPut, "/admin/apikey", strings.NewReader(`{"apiKey":"must-not-leak"}`)),
	)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "LEGACY_API_KEY_MIGRATION_DISABLED") {
		t.Fatalf("stable legacy API-key migration code missing: %s", body)
	}
	if strings.Contains(body, "must-not-leak") || strings.Contains(body, "apiKey") {
		t.Fatalf("legacy API-key route leaked credential payload: %s", body)
	}
}

func TestRouterFailsClosedForLegacyAPIKeyHeadRoute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/admin/apikey", nil))

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.Code)
	}
}

func TestRouterReturnsPlatformTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/types", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"desktop"`) {
		t.Fatalf("platform type response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformLocations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/locations", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"eu-west"`) {
		t.Fatalf("platform location response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformBrands(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/brands", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"brand":"Apple"`) {
		t.Fatalf("platform brand response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformModels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/models/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"model":"iPhone"`) {
		t.Fatalf("platform model response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformOperatingSystems(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/os/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"linux"`) {
		t.Fatalf("platform operating-system response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformOperatingSystemVersions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/osversion/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"version":"14"`) {
		t.Fatalf("platform operating-system version response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformBrowsers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/browsers/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name":"chrome"`) {
		t.Fatalf("platform browser response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsPlatformBrowserVersions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/platforms/browserversions/1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"version":"124"`) {
		t.Fatalf("platform browser-version response missing: %s", response.Body.String())
	}
}

func TestRouterReturnsCLITestCycleWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/testcycle/7", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":7`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI test-cycle response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterRejectsCLITestCycleWithoutLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ideliumcl/testcycle/7", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid key"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterHidesForeignCLITestCycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/testcycle/8", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterCreatesCLIPerformedCycleWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ideliumcl/testcycle", strings.NewReader(`{"testCycleId":7}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idCycle":44}` {
		t.Fatalf("unexpected CLI performed-cycle create body: %s", response.Body.String())
	}
}

func TestRouterUpdatesCLIPerformedCycleWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/ideliumcl/testcycle", strings.NewReader(`{"testCycleId":44,"status":2}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idCycle":44}` {
		t.Fatalf("unexpected CLI performed-cycle update body: %s", response.Body.String())
	}
}

func TestRouterReturnsCLITestWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/test/9", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":9`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI test response missing expected fields: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"code"`) {
		t.Fatalf("CLI test read transformed the legacy payload unexpectedly: %s", response.Body.String())
	}
}

func TestRouterHidesForeignCLITest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/test/10", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterCreatesCLIPerformedTestWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ideliumcl/test", strings.NewReader(`{"testCycleId":7,"testId":9,"name":"browser test"}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idTest":55}` {
		t.Fatalf("unexpected CLI performed-test create body: %s", response.Body.String())
	}
}

func TestRouterRejectsCLIPerformedTestCreateWithoutLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ideliumcl/test", strings.NewReader(`{"testCycleId":7,"testId":9,"name":"browser test"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid key"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterUpdatesCLIPerformedTestWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/ideliumcl/test", strings.NewReader(`{"testId":55,"status":1,"postmanData":[]}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idTest":55}` {
		t.Fatalf("unexpected CLI performed-test update body: %s", response.Body.String())
	}
}

func TestRouterCreatesCLIPerformedStepWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ideliumcl/step", strings.NewReader(`{"testCycleId":44,"testId":55,"stepId":12,"name":"open page","status":1,"screenshots":"[]","data":"{\"result\":\"ok\"}","type":"selenium"}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idStep":77}` {
		t.Fatalf("unexpected CLI performed-step create body: %s", response.Body.String())
	}
}

func TestRouterUpdatesCLIPerformedStepWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/ideliumcl/step", strings.NewReader(`{"stepId":77,"screenshots":"[]"}`))
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idStep":77}` {
		t.Fatalf("unexpected CLI performed-step update body: %s", response.Body.String())
	}
}

func TestRouterReturnsCLIStepWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/step/12", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":12`) ||
		!strings.Contains(response.Body.String(), `"order":2`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI step response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterHidesForeignCLIStep(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/step/13", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterReturnsCLIPluginsWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/plugins/3", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":14`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI plugin-list response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterReturnsCLIPluginWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/plugin/14", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":14`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI plugin response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterHidesForeignCLIPlugin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/plugin/15", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestRouterReturnsCLIEnvironmentsWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/environments/3", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":16`) ||
		!strings.Contains(response.Body.String(), `"code":"demo"`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI environment-list response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterReturnsCLIEnvironmentWithLegacyKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/environment/16", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":16`) ||
		!strings.Contains(response.Body.String(), `"code":"demo"`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("CLI environment response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterReturnsManagedPlatforms(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/platforms/manageplatforms/1", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"hostname":"https://node.example:4444"`) {
		t.Fatalf("managed-platform response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterReturnsLaunchTargets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/launch/targets/3", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":"platform-pool"`) ||
		!strings.Contains(response.Body.String(), `"runtime":"selenium"`) {
		t.Fatalf("launch-target response missing expected fields: %s", response.Body.String())
	}
}

func TestRouterHidesForeignCLIEnvironment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	router := testRouter(logger)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ideliumcl/environment/17", nil)
	request.Header.Set(auth.IdeliumKeyHeader, "valid-key")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}
