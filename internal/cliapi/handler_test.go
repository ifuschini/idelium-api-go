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
	handler := NewHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
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
	handler := NewHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.TestCycle(response, requestWithTenant("/ideliumcl/testcycle/not-number", "not-number", 42))

	assertInvalidID(t, response)
	if repository.testCycleID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingCycle(t *testing.T) {
	repository := &fakeTestCycleRepository{err: ErrNotFound}
	handler := NewHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.TestCycle(response, requestWithTenant("/ideliumcl/testcycle/8", "8", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeTestCycleRepository{err: errors.New("database failed near secret-value")}
	handler := NewHandler(repository, &fakeTestRepository{}, &fakeStepRepository{}, slog.New(slog.NewTextHandler(logBuffer, nil)))
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
	handler := NewHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
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
	handler := NewHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Test(response, requestWithTenantParam("/ideliumcl/test/not-number", "idTest", "not-number", 42))

	assertInvalidID(t, response)
	if repository.testID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingTest(t *testing.T) {
	repository := &fakeTestRepository{err: ErrNotFound}
	handler := NewHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Test(response, requestWithTenantParam("/ideliumcl/test/10", "idTest", "10", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsTestRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeTestRepository{err: errors.New("database failed near secret-value")}
	handler := NewHandler(&fakeTestCycleRepository{}, repository, &fakeStepRepository{}, slog.New(slog.NewTextHandler(logBuffer, nil)))
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
	handler := NewHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
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
	handler := NewHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/not-number", "idStep", "not-number", 42))

	assertInvalidID(t, response)
	if repository.stepID != 0 {
		t.Fatalf("repository should not be called for malformed identifiers: %#v", repository)
	}
}

func TestHandlerReturnsInvalidIDForCrossTenantOrMissingStep(t *testing.T) {
	repository := &fakeStepRepository{err: ErrNotFound}
	handler := NewHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/13", "idStep", "13", 42))

	assertInvalidID(t, response)
}

func TestHandlerRedactsStepRepositoryFailures(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	repository := &fakeStepRepository{err: errors.New("database failed near secret-value")}
	handler := NewHandler(&fakeTestCycleRepository{}, &fakeTestRepository{}, repository, slog.New(slog.NewTextHandler(logBuffer, nil)))
	response := httptest.NewRecorder()

	handler.Step(response, requestWithTenantParam("/ideliumcl/step/13", "idStep", "13", 42))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(logBuffer.String(), "secret-value") {
		t.Fatalf("repository error leaked into logs: %s", logBuffer.String())
	}
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
