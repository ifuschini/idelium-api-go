package cliapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/auth"
)

type fakeTestCycleRepository struct {
	customerID  int64
	testCycleID int64
	testCycle   TestCycle
	err         error
}

func (repository *fakeTestCycleRepository) GetTestCycle(ctx context.Context, customerID int64, testCycleID int64) (TestCycle, error) {
	repository.customerID = customerID
	repository.testCycleID = testCycleID
	if repository.err != nil {
		return TestCycle{}, repository.err
	}
	return repository.testCycle, nil
}

type fakeTestRepository struct {
	customerID int64
	testID     int64
	test       Test
	err        error
}

func (repository *fakeTestRepository) GetTest(ctx context.Context, customerID int64, testID int64) (Test, error) {
	repository.customerID = customerID
	repository.testID = testID
	if repository.err != nil {
		return Test{}, repository.err
	}
	return repository.test, nil
}

type fakeStepRepository struct {
	customerID int64
	stepID     int64
	step       Step
	err        error
}

func (repository *fakeStepRepository) GetStep(ctx context.Context, customerID int64, stepID int64) (Step, error) {
	repository.customerID = customerID
	repository.stepID = stepID
	if repository.err != nil {
		return Step{}, repository.err
	}
	return repository.step, nil
}

type fakePluginRepository struct {
	customerID int64
	projectID  int64
	pluginID   int64
	plugins    []Plugin
	plugin     Plugin
	err        error
}

func (repository *fakePluginRepository) ListPlugins(ctx context.Context, customerID int64, projectID int64) ([]Plugin, error) {
	repository.customerID = customerID
	repository.projectID = projectID
	if repository.err != nil {
		return nil, repository.err
	}
	return repository.plugins, nil
}

func (repository *fakePluginRepository) GetPlugin(ctx context.Context, customerID int64, pluginID int64) (Plugin, error) {
	repository.customerID = customerID
	repository.pluginID = pluginID
	if repository.err != nil {
		return Plugin{}, repository.err
	}
	return repository.plugin, nil
}

func TestHandlerReturnsTenantScopedTestCycle(t *testing.T) {
	repository := &fakeTestCycleRepository{
		testCycle: TestCycle{
			ID:          7,
			Name:        "nightly",
			Description: "Nightly cycle",
			Config:      "[]",
			IDProject:   3,
			IDCostumer:  42,
		},
	}
	handler := testHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenant("/ideliumcl/testcycle/7", "7", 42)

	handler.TestCycle(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":7`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("test-cycle response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.testCycleID != 7 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForMalformedIdentifier(t *testing.T) {
	repository := &fakeTestCycleRepository{}
	handler := testHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.TestCycle(response, requestWithTenant("/ideliumcl/testcycle/not-number", "not-number", 42))

	assertInvalidID(t, response)
	if repository.testCycleID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingCycle(t *testing.T) {
	repository := &fakeTestCycleRepository{err: ErrNotFound}
	handler := testHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.TestCycle(response, requestWithTenant("/ideliumcl/testcycle/8", "8", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeTestCycleRepository{err: errors.New("database failed near secret-value")}
	handler := testHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, logBuffer)
	response := httptest.NewRecorder()

	handler.TestCycle(response, requestWithTenant("/ideliumcl/testcycle/8", "8", 42))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(logBuffer.String(), "secret-value") {
		t.Fatalf("repository error leaked into logs: %s", logBuffer.String())
	}
}

func TestHandlerReturnsTenantScopedTest(t *testing.T) {
	repository := &fakeTestRepository{
		test: Test{
			ID:          9,
			Name:        "browser test",
			Description: "Browser coverage",
			Config:      "[]",
			IDProject:   3,
			IDCostumer:  42,
		},
	}
	handler := testHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/test/9", "idTest", "9", 42)

	handler.Test(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":9`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("test response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.testID != 9 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForMalformedTestIdentifier(t *testing.T) {
	repository := &fakeTestRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Test(response, requestWithTenantParam("/ideliumcl/test/not-number", "idTest", "not-number", 42))

	assertInvalidID(t, response)
	if repository.testID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingTest(t *testing.T) {
	repository := &fakeTestRepository{err: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Test(response, requestWithTenantParam("/ideliumcl/test/10", "idTest", "10", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsTestRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeTestRepository{err: errors.New("database failed near secret-value")}
	handler := testHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, &fakePluginRepository{}, logBuffer)
	response := httptest.NewRecorder()

	handler.Test(response, requestWithTenantParam("/ideliumcl/test/10", "idTest", "10", 42))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(logBuffer.String(), "secret-value") {
		t.Fatalf("repository error leaked into logs: %s", logBuffer.String())
	}
}

func TestHandlerReturnsTenantScopedStep(t *testing.T) {
	repository := &fakeStepRepository{
		step: Step{
			ID:          12,
			Name:        "open page",
			Description: "Open the browser",
			Config:      "[]",
			IDProject:   3,
			Order:       2,
			IDCostumer:  42,
		},
	}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/step/12", "idStep", "12", 42)

	handler.Step(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":12`) ||
		!strings.Contains(response.Body.String(), `"order":2`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("step response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.stepID != 12 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForMalformedStepIdentifier(t *testing.T) {
	repository := &fakeStepRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/not-number", "idStep", "not-number", 42))

	assertInvalidID(t, response)
	if repository.stepID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingStep(t *testing.T) {
	repository := &fakeStepRepository{err: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, &fakePluginRepository{}, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/13", "idStep", "13", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsStepRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeStepRepository{err: errors.New("database failed near secret-value")}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, &fakePluginRepository{}, logBuffer)
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/13", "idStep", "13", 42))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(logBuffer.String(), "secret-value") {
		t.Fatalf("repository error leaked into logs: %s", logBuffer.String())
	}
}

func TestHandlerReturnsTenantScopedPluginList(t *testing.T) {
	repository := &fakePluginRepository{
		plugins: []Plugin{{
			ID:          14,
			Name:        "python wrapper",
			Code:        "{}",
			Description: "Plugin manifest",
			IDProject:   3,
			IDCostumer:  42,
		}},
	}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/plugins/3", "idProject", "3", 42)

	handler.Plugins(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":14`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("plugin-list response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.projectID != 3 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsEmptyPluginList(t *testing.T) {
	repository := &fakePluginRepository{plugins: []Plugin{}}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/plugins/3", "idProject", "3", 42)

	handler.Plugins(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `[]` {
		t.Fatalf("expected empty list body, got %s", response.Body.String())
	}
}

func TestHandlerReturnsTenantScopedPlugin(t *testing.T) {
	repository := &fakePluginRepository{
		plugin: Plugin{
			ID:          14,
			Name:        "python wrapper",
			Code:        "{}",
			Description: "Plugin manifest",
			IDProject:   3,
			IDCostumer:  42,
		},
	}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/plugin/14", "idPlugin", "14", 42)

	handler.Plugin(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":14`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("plugin response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.pluginID != 14 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingPlugin(t *testing.T) {
	repository := &fakePluginRepository{err: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Plugin(response, requestWithTenantParam("/ideliumcl/plugin/15", "idPlugin", "15", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsPluginRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakePluginRepository{err: errors.New("database failed near secret-value")}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, logBuffer)
	response := httptest.NewRecorder()

	handler.Plugins(response, requestWithTenantParam("/ideliumcl/plugins/3", "idProject", "3", 42))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(logBuffer.String(), "secret-value") {
		t.Fatalf("repository error leaked into logs: %s", logBuffer.String())
	}
}

func testHandler(
	testCycles TestCycleRepository,
	tests TestRepository,
	steps StepRepository,
	plugins PluginRepository,
	logBuffer *bytes.Buffer,
) *Handler {
	return NewHandler(testCycles, tests, steps, plugins, slog.New(slog.NewTextHandler(logBuffer, nil)))
}

func requestWithTenant(target string, pathID string, customerID int64) *http.Request {
	return requestWithTenantParam(target, "idTestCycle", pathID, customerID)
}

func requestWithTenantParam(target string, pathParam string, pathID string, customerID int64) *http.Request {
	routerContext := chi.NewRouteContext()
	routerContext.URLParams.Add(pathParam, pathID)
	request := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routerContext)
	ctx = auth.ContextWithTenant(ctx, auth.TenantContext{CustomerID: customerID})
	return request.WithContext(ctx)
}

func assertInvalidID(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid id"}` {
		t.Fatalf("unexpected invalid-id body: %s", response.Body.String())
	}
}
