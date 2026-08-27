package cliapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

const maxExecutionContextBytes = 64 << 10

// CreatePerformedCycleRequest is the Laravel-compatible CLI performed-cycle creation command.
type CreatePerformedCycleRequest struct {
	TestCycleID              int64
	IdempotencyKey           string
	ExecutionContext         *string
	ExecutionContextProvided bool
}

// UpdatePerformedCycleRequest is the Laravel-compatible CLI performed-cycle update command.
type UpdatePerformedCycleRequest struct {
	TestCycleID int64
	Status      int
}

// PerformedCycleRepository writes tenant-scoped CLI result cycle records.
type PerformedCycleRepository interface {
	CreatePerformedCycle(ctx context.Context, customerID int64, command CreatePerformedCycleRequest) (int64, error)
	UpdatePerformedCycle(ctx context.Context, customerID int64, command UpdatePerformedCycleRequest) (int64, error)
}

// CreatePerformedCycle creates one tenant-owned performed cycle using the Laravel CLI contract.
func (handler *Handler) CreatePerformedCycle(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedCycles == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-cycle writer is not configured.")
		return
	}

	command, ok := decodeCreatePerformedCycle(writer, request)
	if !ok {
		return
	}

	performedCycleID, err := handler.performedCycles.CreatePerformedCycle(
		request.Context(),
		tenant.CustomerID,
		command,
	)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-cycle create failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-cycle record could not be created.")
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idCycle": performedCycleID})
}

func decodeCreatePerformedCycle(writer http.ResponseWriter, request *http.Request) (CreatePerformedCycleRequest, bool) {
	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return CreatePerformedCycleRequest{}, false
	}
	testCycleID, ok := requiredPositiveInt(fields, "testCycleId")
	if !ok {
		writeValidationError(writer, request, "testCycleId is required and must be a positive integer.")
		return CreatePerformedCycleRequest{}, false
	}

	key, ok := idempotencyKey(request)
	if !ok {
		writeValidationError(writer, request, "Idempotency-Key must be between 8 and 128 URL-safe characters.")
		return CreatePerformedCycleRequest{}, false
	}
	command := CreatePerformedCycleRequest{TestCycleID: testCycleID, IdempotencyKey: key}
	rawExecutionContext, exists := fields["executionContext"]
	if !exists || bytes.Equal(bytes.TrimSpace(rawExecutionContext), []byte("null")) {
		return command, true
	}
	command.ExecutionContextProvided = true
	if len(rawExecutionContext) > maxExecutionContextBytes {
		writeValidationError(writer, request, "executionContext must be a JSON object smaller than 64 KiB.")
		return CreatePerformedCycleRequest{}, false
	}
	var executionContext map[string]any
	if err := json.Unmarshal(rawExecutionContext, &executionContext); err != nil || executionContext == nil {
		writeValidationError(writer, request, "executionContext must be a JSON object.")
		return CreatePerformedCycleRequest{}, false
	}
	redacted, err := redactPostmanJSON(rawExecutionContext)
	if err != nil {
		writeValidationError(writer, request, "executionContext must be valid JSON.")
		return CreatePerformedCycleRequest{}, false
	}
	command.ExecutionContext = &redacted
	return command, true
}

// UpdatePerformedCycle updates one tenant-owned performed cycle using the Laravel CLI contract.
func (handler *Handler) UpdatePerformedCycle(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedCycles == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-cycle writer is not configured.")
		return
	}

	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return
	}
	testCycleID, ok := requiredPositiveInt(fields, "testCycleId")
	if !ok {
		writeValidationError(writer, request, "testCycleId is required and must be a positive integer.")
		return
	}
	status, ok := requiredInt(fields, "status")
	if !ok || (status != 1 && status != 2) {
		writeValidationError(writer, request, "status is required and must be 1 or 2.")
		return
	}

	performedCycleID, err := handler.performedCycles.UpdatePerformedCycle(
		request.Context(),
		tenant.CustomerID,
		UpdatePerformedCycleRequest{TestCycleID: testCycleID, Status: int(status)},
	)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-cycle update failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-cycle record could not be updated.")
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idCycle": performedCycleID})
}
