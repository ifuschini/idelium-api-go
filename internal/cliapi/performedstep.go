package cliapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

var allowedPerformedStepTypes = map[string]struct{}{
	"selenium":         {},
	"seleniumOrAppium": {},
	"postman":          {},
	"dsl":              {},
}

// CreatePerformedStepRequest is the Laravel-compatible CLI performed-step creation command.
type CreatePerformedStepRequest struct {
	TestCycleID int64
	TestID      int64
	StepID      int64
	Name        string
	Status      int
	Screenshots string
	Data        string
	Type        string
}

// UpdatePerformedStepRequest is the Laravel-compatible CLI performed-step screenshot update command.
type UpdatePerformedStepRequest struct {
	StepID      int64
	Screenshots string
}

// PerformedStepRepository writes tenant-scoped CLI result step records.
type PerformedStepRepository interface {
	CreatePerformedStep(ctx context.Context, customerID int64, command CreatePerformedStepRequest) (int64, error)
	UpdatePerformedStep(ctx context.Context, customerID int64, command UpdatePerformedStepRequest) (int64, error)
}

// CreatePerformedStep creates one tenant-owned performed step using the Laravel CLI contract.
func (handler *Handler) CreatePerformedStep(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedSteps == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-step writer is not configured.")
		return
	}

	command, ok := decodeCreatePerformedStep(writer, request)
	if !ok {
		return
	}
	performedStepID, err := handler.performedSteps.CreatePerformedStep(request.Context(), tenant.CustomerID, command)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-step create failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-step record could not be created.")
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idStep": performedStepID})
}

// UpdatePerformedStep updates screenshots for one tenant-owned performed step using the Laravel CLI contract.
func (handler *Handler) UpdatePerformedStep(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedSteps == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-step writer is not configured.")
		return
	}

	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return
	}
	stepID, ok := requiredPositiveInt(fields, "stepId")
	if !ok {
		writeValidationError(writer, request, "stepId is required and must be a positive integer.")
		return
	}
	screenshots, ok := jsonStringField(fields, "screenshots")
	if !ok {
		writeValidationError(writer, request, "screenshots is required.")
		return
	}

	performedStepID, err := handler.performedSteps.UpdatePerformedStep(
		request.Context(),
		tenant.CustomerID,
		UpdatePerformedStepRequest{StepID: stepID, Screenshots: screenshots},
	)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-step update failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-step record could not be updated.")
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idStep": performedStepID})
}

func decodeCreatePerformedStep(writer http.ResponseWriter, request *http.Request) (CreatePerformedStepRequest, bool) {
	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return CreatePerformedStepRequest{}, false
	}
	testCycleID, ok := requiredPositiveInt(fields, "testCycleId")
	if !ok {
		writeValidationError(writer, request, "testCycleId is required and must be a positive integer.")
		return CreatePerformedStepRequest{}, false
	}
	testID, ok := requiredPositiveInt(fields, "testId")
	if !ok {
		writeValidationError(writer, request, "testId is required and must be a positive integer.")
		return CreatePerformedStepRequest{}, false
	}
	stepID, ok := requiredPositiveInt(fields, "stepId")
	if !ok {
		writeValidationError(writer, request, "stepId is required and must be a positive integer.")
		return CreatePerformedStepRequest{}, false
	}
	name, ok := requiredString(fields, "name")
	if !ok {
		writeValidationError(writer, request, "name is required and must be a non-empty string.")
		return CreatePerformedStepRequest{}, false
	}
	status, ok := requiredInt(fields, "status")
	if !ok {
		writeValidationError(writer, request, "status is required and must be an integer.")
		return CreatePerformedStepRequest{}, false
	}
	screenshots, ok := validJSONStringField(fields, "screenshots")
	if !ok {
		writeValidationError(writer, request, "screenshots is required and must be a valid JSON string.")
		return CreatePerformedStepRequest{}, false
	}
	data, ok := validJSONStringField(fields, "data")
	if !ok {
		writeValidationError(writer, request, "data is required and must be a valid JSON string.")
		return CreatePerformedStepRequest{}, false
	}
	redactedData, err := redactJSONString(data)
	if err != nil {
		writeValidationError(writer, request, "data is required and must be valid JSON.")
		return CreatePerformedStepRequest{}, false
	}
	stepType, ok := requiredString(fields, "type")
	if !ok {
		writeValidationError(writer, request, "type is required.")
		return CreatePerformedStepRequest{}, false
	}
	if _, allowed := allowedPerformedStepTypes[stepType]; !allowed {
		writeValidationError(writer, request, "type must be selenium, seleniumOrAppium, postman, or dsl.")
		return CreatePerformedStepRequest{}, false
	}
	return CreatePerformedStepRequest{
		TestCycleID: testCycleID,
		TestID:      testID,
		StepID:      stepID,
		Name:        name,
		Status:      int(status),
		Screenshots: screenshots,
		Data:        redactedData,
		Type:        stepType,
	}, true
}
