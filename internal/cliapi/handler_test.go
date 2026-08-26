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

type fakePerformedCycleRepository struct {
	createCustomerID int64
	createCommand    CreatePerformedCycleRequest
	createID         int64
	createErr        error
	updateCustomerID int64
	updateCommand    UpdatePerformedCycleRequest
	updateID         int64
	updateErr        error
}

func (repository *fakePerformedCycleRepository) CreatePerformedCycle(ctx context.Context, customerID int64, command CreatePerformedCycleRequest) (int64, error) {
	repository.createCustomerID = customerID
	repository.createCommand = command
	if repository.createErr != nil {
		return 0, repository.createErr
	}
	if repository.createID != 0 {
		return repository.createID, nil
	}
	return 501, nil
}

func (repository *fakePerformedCycleRepository) UpdatePerformedCycle(ctx context.Context, customerID int64, command UpdatePerformedCycleRequest) (int64, error) {
	repository.updateCustomerID = customerID
	repository.updateCommand = command
	if repository.updateErr != nil {
		return 0, repository.updateErr
	}
	if repository.updateID != 0 {
		return repository.updateID, nil
	}
	return command.TestCycleID, nil
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

type fakePerformedTestRepository struct {
	createCustomerID int64
	createCommand    CreatePerformedTestRequest
	createID         int64
	createErr        error
	updateCustomerID int64
	updateCommand    UpdatePerformedTestRequest
	updateID         int64
	updateErr        error
}

func (repository *fakePerformedTestRepository) CreatePerformedTest(ctx context.Context, customerID int64, command CreatePerformedTestRequest) (int64, error) {
	repository.createCustomerID = customerID
	repository.createCommand = command
	if repository.createErr != nil {
		return 0, repository.createErr
	}
	if repository.createID != 0 {
		return repository.createID, nil
	}
	return 701, nil
}

func (repository *fakePerformedTestRepository) UpdatePerformedTest(ctx context.Context, customerID int64, command UpdatePerformedTestRequest) (int64, error) {
	repository.updateCustomerID = customerID
	repository.updateCommand = command
	if repository.updateErr != nil {
		return 0, repository.updateErr
	}
	if repository.updateID != 0 {
		return repository.updateID, nil
	}
	return command.TestID, nil
}

type fakePerformedStepRepository struct {
	createCustomerID int64
	createCommand    CreatePerformedStepRequest
	createID         int64
	createErr        error
	updateCustomerID int64
	updateCommand    UpdatePerformedStepRequest
	updateID         int64
	updateErr        error
}

func (repository *fakePerformedStepRepository) CreatePerformedStep(ctx context.Context, customerID int64, command CreatePerformedStepRequest) (int64, error) {
	repository.createCustomerID = customerID
	repository.createCommand = command
	if repository.createErr != nil {
		return 0, repository.createErr
	}
	if repository.createID != 0 {
		return repository.createID, nil
	}
	return 901, nil
}

func (repository *fakePerformedStepRepository) UpdatePerformedStep(ctx context.Context, customerID int64, command UpdatePerformedStepRequest) (int64, error) {
	repository.updateCustomerID = customerID
	repository.updateCommand = command
	if repository.updateErr != nil {
		return 0, repository.updateErr
	}
	if repository.updateID != 0 {
		return repository.updateID, nil
	}
	return command.StepID, nil
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

type fakeEnvironmentRepository struct {
	customerID    int64
	projectID     int64
	environmentID int64
	environments  []Environment
	environment   Environment
	err           error
}

func (repository *fakeEnvironmentRepository) ListEnvironments(ctx context.Context, customerID int64, projectID int64) ([]Environment, error) {
	repository.customerID = customerID
	repository.projectID = projectID
	if repository.err != nil {
		return nil, repository.err
	}
	return repository.environments, nil
}

func (repository *fakeEnvironmentRepository) GetEnvironment(ctx context.Context, customerID int64, environmentID int64) (Environment, error) {
	repository.customerID = customerID
	repository.environmentID = environmentID
	if repository.err != nil {
		return Environment{}, repository.err
	}
	return repository.environment, nil
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

func TestHandlerCreatesTenantScopedPerformedCycle(t *testing.T) {
	repository := &fakePerformedCycleRepository{createID: 44}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPost, "/ideliumcl/testcycle", `{"testCycleId":7}`, 42)

	handler.CreatePerformedCycle(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idCycle":44}` {
		t.Fatalf("unexpected create body: %s", response.Body.String())
	}
	if repository.createCustomerID != 42 || repository.createCommand.TestCycleID != 7 {
		t.Fatalf("repository was not called with tenant-scoped cycle command: %#v", repository)
	}
}

func TestHandlerCreatesPerformedCycleWithRedactedExecutionContext(t *testing.T) {
	repository := &fakePerformedCycleRepository{createID: 44}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPost,
		"/ideliumcl/testcycle",
		`{"testCycleId":7,"executionContext":{"environment":"demo","browser":"firefox","operatingSystem":"darwin","apiToken":"secret-value"}}`,
		42,
	)

	handler.CreatePerformedCycle(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !repository.createCommand.ExecutionContextProvided || repository.createCommand.ExecutionContext == nil {
		t.Fatalf("execution context was not captured: %#v", repository.createCommand)
	}
	contextJSON := *repository.createCommand.ExecutionContext
	for _, expected := range []string{`"environment":"demo"`, `"browser":"firefox"`, `"operatingSystem":"darwin"`, `"apiToken":"[REDACTED]"`} {
		if !strings.Contains(contextJSON, expected) {
			t.Fatalf("execution context %s did not contain %s", contextJSON, expected)
		}
	}
	if strings.Contains(contextJSON, "secret-value") {
		t.Fatalf("execution context leaked a sensitive value: %s", contextJSON)
	}
}

func TestHandlerRejectsInvalidExecutionContext(t *testing.T) {
	repository := &fakePerformedCycleRepository{createID: 44}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPost, "/ideliumcl/testcycle", `{"testCycleId":7,"executionContext":["not","an","object"]}`, 42)

	handler.CreatePerformedCycle(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
	if repository.createCustomerID != 0 {
		t.Fatalf("repository should not be called for invalid execution context: %#v", repository)
	}
}

func TestHandlerReturnsInvalidDetailsForMissingPerformedCycleReference(t *testing.T) {
	repository := &fakePerformedCycleRepository{createErr: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPost, "/ideliumcl/testcycle", `{"testCycleId":7}`, 42)

	handler.CreatePerformedCycle(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid details"}` {
		t.Fatalf("unexpected invalid details body: %s", response.Body.String())
	}
}

func TestHandlerUpdatesTenantScopedPerformedCycle(t *testing.T) {
	repository := &fakePerformedCycleRepository{updateID: 44}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPut, "/ideliumcl/testcycle", `{"testCycleId":44,"status":1}`, 42)

	handler.UpdatePerformedCycle(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idCycle":44}` {
		t.Fatalf("unexpected update body: %s", response.Body.String())
	}
	if repository.updateCustomerID != 42 ||
		repository.updateCommand.TestCycleID != 44 ||
		repository.updateCommand.Status != 1 {
		t.Fatalf("repository was not called with tenant-scoped cycle update command: %#v", repository)
	}
}

func TestHandlerValidatesPerformedCycleStatus(t *testing.T) {
	repository := &fakePerformedCycleRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedCycles = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPut, "/ideliumcl/testcycle", `{"testCycleId":44,"status":3}`, 42)

	handler.UpdatePerformedCycle(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
	if repository.updateCustomerID != 0 {
		t.Fatalf("repository should not be called for invalid status: %#v", repository)
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

func TestHandlerCreatesTenantScopedPerformedTest(t *testing.T) {
	repository := &fakePerformedTestRepository{createID: 55}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPost,
		"/ideliumcl/test",
		`{"testCycleId":7,"testId":9,"name":"browser test"}`,
		42,
	)

	handler.CreatePerformedTest(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idTest":55}` {
		t.Fatalf("unexpected create body: %s", response.Body.String())
	}
	if repository.createCustomerID != 42 ||
		repository.createCommand.TestCycleID != 7 ||
		repository.createCommand.TestID != 9 ||
		repository.createCommand.Name != "browser test" {
		t.Fatalf("repository was not called with tenant-scoped command: %#v", repository)
	}
}

func TestHandlerReturnsInvalidDetailsForMissingPerformedTestReferences(t *testing.T) {
	repository := &fakePerformedTestRepository{createErr: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPost,
		"/ideliumcl/test",
		`{"testCycleId":7,"testId":9,"name":"browser test"}`,
		42,
	)

	handler.CreatePerformedTest(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"message":"Invalid details"}` {
		t.Fatalf("unexpected invalid details body: %s", response.Body.String())
	}
}

func TestHandlerValidatesPerformedTestCreate(t *testing.T) {
	repository := &fakePerformedTestRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPost, "/ideliumcl/test", `{"testCycleId":7,"testId":9}`, 42)

	handler.CreatePerformedTest(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
	if repository.createCustomerID != 0 {
		t.Fatalf("repository should not be called for invalid requests: %#v", repository)
	}
}

func TestHandlerUpdatesTenantScopedPerformedTestWithRedactedPostmanData(t *testing.T) {
	repository := &fakePerformedTestRepository{updateID: 55}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPut,
		"/ideliumcl/test",
		`{"testId":55,"status":1,"postmanData":[{"request":{"header":{"Authorization":"Bearer unsafe-token"},"url":"https://api.example.test"},"response":{"code":200}}]}`,
		42,
	)

	handler.UpdatePerformedTest(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idTest":55}` {
		t.Fatalf("unexpected update body: %s", response.Body.String())
	}
	if repository.updateCustomerID != 42 ||
		repository.updateCommand.TestID != 55 ||
		repository.updateCommand.Status != 1 ||
		!repository.updateCommand.PostmanDataPresent ||
		repository.updateCommand.PostmanData == nil {
		t.Fatalf("repository was not called with tenant-scoped update command: %#v", repository)
	}
	if strings.Contains(*repository.updateCommand.PostmanData, "unsafe-token") ||
		!strings.Contains(*repository.updateCommand.PostmanData, "[REDACTED]") ||
		!strings.Contains(*repository.updateCommand.PostmanData, "https://api.example.test") {
		t.Fatalf("postmanData redaction did not preserve safe detail and hide secrets: %s", *repository.updateCommand.PostmanData)
	}
}

func TestHandlerAllowsNullPostmanDataOnPerformedTestUpdate(t *testing.T) {
	repository := &fakePerformedTestRepository{updateID: 55}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPut, "/ideliumcl/test", `{"testId":55,"status":0,"postmanData":null}`, 42)

	handler.UpdatePerformedTest(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !repository.updateCommand.PostmanDataPresent || repository.updateCommand.PostmanData != nil {
		t.Fatalf("null postmanData should be explicit and stored as nil: %#v", repository.updateCommand)
	}
}

func TestHandlerRejectsNonArrayPostmanData(t *testing.T) {
	repository := &fakePerformedTestRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedTests = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPut, "/ideliumcl/test", `{"testId":55,"status":1,"postmanData":{"token":"unsafe"}}`, 42)

	handler.UpdatePerformedTest(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
	if repository.updateCustomerID != 0 {
		t.Fatalf("repository should not be called for invalid postmanData: %#v", repository)
	}
}

func TestHandlerCreatesTenantScopedPerformedStepWithRedactedData(t *testing.T) {
	repository := &fakePerformedStepRepository{createID: 77}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedSteps = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPost,
		"/ideliumcl/step",
		`{"testCycleId":44,"testId":55,"stepId":12,"name":"open page","status":1,"screenshots":"[]","data":"{\"headers\":{\"Authorization\":\"Bearer unsafe-token\"},\"result\":\"ok\"}","type":"selenium"}`,
		42,
	)

	handler.CreatePerformedStep(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idStep":77}` {
		t.Fatalf("unexpected create body: %s", response.Body.String())
	}
	if repository.createCustomerID != 42 ||
		repository.createCommand.TestCycleID != 44 ||
		repository.createCommand.TestID != 55 ||
		repository.createCommand.StepID != 12 ||
		repository.createCommand.Type != "selenium" {
		t.Fatalf("repository was not called with tenant-scoped step command: %#v", repository)
	}
	if strings.Contains(repository.createCommand.Data, "unsafe-token") ||
		!strings.Contains(repository.createCommand.Data, "[REDACTED]") ||
		!strings.Contains(repository.createCommand.Data, "ok") {
		t.Fatalf("step data was not redacted safely: %s", repository.createCommand.Data)
	}
}

func TestHandlerValidatesPerformedStepTypeAndJSON(t *testing.T) {
	repository := &fakePerformedStepRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedSteps = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(
		http.MethodPost,
		"/ideliumcl/step",
		`{"testCycleId":44,"testId":55,"stepId":12,"name":"open page","status":1,"screenshots":"[]","data":"not-json","type":"shell"}`,
		42,
	)

	handler.CreatePerformedStep(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
	if repository.createCustomerID != 0 {
		t.Fatalf("repository should not be called for invalid performed-step payload: %#v", repository)
	}
}

func TestHandlerUpdatesTenantScopedPerformedStepScreenshots(t *testing.T) {
	repository := &fakePerformedStepRepository{updateID: 77}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.performedSteps = repository
	response := httptest.NewRecorder()
	request := requestWithTenantBody(http.MethodPut, "/ideliumcl/step", `{"stepId":77,"screenshots":"[\"screen.png\"]"}`, 42)

	handler.UpdatePerformedStep(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `{"idStep":77}` {
		t.Fatalf("unexpected update body: %s", response.Body.String())
	}
	if repository.updateCustomerID != 42 ||
		repository.updateCommand.StepID != 77 ||
		repository.updateCommand.Screenshots != `["screen.png"]` {
		t.Fatalf("repository was not called with tenant-scoped step update command: %#v", repository)
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

func TestHandlerReturnsInvalidIDForMalformedPluginListIdentifier(t *testing.T) {
	repository := &fakePluginRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Plugins(response, requestWithTenantParam("/ideliumcl/plugins/not-number", "idProject", "not-number", 42))

	assertInvalidID(t, response)
	if repository.projectID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
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

func TestHandlerReturnsInvalidIDForMalformedPluginIdentifier(t *testing.T) {
	repository := &fakePluginRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, repository, &bytes.Buffer{})
	response := httptest.NewRecorder()

	handler.Plugin(response, requestWithTenantParam("/ideliumcl/plugin/not-number", "idPlugin", "not-number", 42))

	assertInvalidID(t, response)
	if repository.pluginID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
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

func TestHandlerReturnsTenantScopedEnvironmentList(t *testing.T) {
	repository := &fakeEnvironmentRepository{
		environments: []Environment{{
			ID:          16,
			Code:        "demo",
			Description: "Demo environment",
			Config:      "{}",
			IDProject:   3,
			IDCostumer:  42,
		}},
	}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/environments/3", "idProject", "3", 42)

	handler.Environments(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":16`) ||
		!strings.Contains(response.Body.String(), `"code":"demo"`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("environment-list response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.projectID != 3 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsEmptyEnvironmentList(t *testing.T) {
	repository := &fakeEnvironmentRepository{environments: []Environment{}}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/environments/3", "idProject", "3", 42)

	handler.Environments(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.TrimSpace(response.Body.String()) != `[]` {
		t.Fatalf("expected empty list body, got %s", response.Body.String())
	}
}

func TestHandlerReturnsInvalidIDForMalformedEnvironmentListIdentifier(t *testing.T) {
	repository := &fakeEnvironmentRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()

	handler.Environments(response, requestWithTenantParam("/ideliumcl/environments/not-number", "idProject", "not-number", 42))

	assertInvalidID(t, response)
	if repository.projectID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsTenantScopedEnvironment(t *testing.T) {
	repository := &fakeEnvironmentRepository{
		environment: Environment{
			ID:          16,
			Code:        "demo",
			Description: "Demo environment",
			Config:      "{}",
			IDProject:   3,
			IDCostumer:  42,
		},
	}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()
	request := requestWithTenantParam("/ideliumcl/environment/16", "idEnvironment", "16", 42)

	handler.Environment(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"id":16`) ||
		!strings.Contains(response.Body.String(), `"code":"demo"`) ||
		!strings.Contains(response.Body.String(), `"idCostumer":42`) {
		t.Fatalf("environment response missing expected fields: %s", response.Body.String())
	}
	if repository.customerID != 42 || repository.environmentID != 16 {
		t.Fatalf("repository was not called with tenant-scoped identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForMalformedEnvironmentIdentifier(t *testing.T) {
	repository := &fakeEnvironmentRepository{}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()

	handler.Environment(response, requestWithTenantParam("/ideliumcl/environment/not-number", "idEnvironment", "not-number", 42))

	assertInvalidID(t, response)
	if repository.environmentID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingEnvironment(t *testing.T) {
	repository := &fakeEnvironmentRepository{err: ErrNotFound}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, &bytes.Buffer{})
	handler.environments = repository
	response := httptest.NewRecorder()

	handler.Environment(response, requestWithTenantParam("/ideliumcl/environment/17", "idEnvironment", "17", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsEnvironmentRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeEnvironmentRepository{err: errors.New("database failed near secret-value")}
	handler := testHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, &fakeStepRepository{}, &fakePluginRepository{}, logBuffer)
	handler.environments = repository
	response := httptest.NewRecorder()

	handler.Environments(response, requestWithTenantParam("/ideliumcl/environments/3", "idProject", "3", 42))

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
	return NewHandler(testCycles, &fakePerformedCycleRepository{}, tests, &fakePerformedTestRepository{}, &fakePerformedStepRepository{}, steps, plugins, &fakeEnvironmentRepository{}, slog.New(slog.NewTextHandler(logBuffer, nil)))
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

func requestWithTenantBody(method string, target string, body string, customerID int64) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	ctx := auth.ContextWithTenant(request.Context(), auth.TenantContext{CustomerID: customerID})
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
