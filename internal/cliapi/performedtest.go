package cliapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/idelium/idelium-api-go/internal/auth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

const invalidDetailsMessage = "Invalid details"

// CreatePerformedTestRequest is the Laravel-compatible CLI performed-test creation command.
type CreatePerformedTestRequest struct {
	TestCycleID    int64
	TestID         int64
	Name           string
	IdempotencyKey string
}

// UpdatePerformedTestRequest is the Laravel-compatible CLI performed-test update command.
type UpdatePerformedTestRequest struct {
	TestID             int64
	Status             int
	PostmanDataPresent bool
	PostmanData        *string
}

// PerformedTestRepository writes tenant-scoped CLI result test records.
type PerformedTestRepository interface {
	CreatePerformedTest(ctx context.Context, customerID int64, command CreatePerformedTestRequest) (int64, error)
	UpdatePerformedTest(ctx context.Context, customerID int64, command UpdatePerformedTestRequest) (int64, error)
}

// CreatePerformedTest creates one tenant-owned performed test using the Laravel CLI contract.
func (handler *Handler) CreatePerformedTest(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedTests == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-test writer is not configured")
		return
	}

	command, ok := decodeCreatePerformedTest(writer, request)
	if !ok {
		return
	}

	performedTestID, err := handler.performedTests.CreatePerformedTest(request.Context(), tenant.CustomerID, command)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-test create failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-test record could not be created.")
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idTest": performedTestID})
}

// UpdatePerformedTest updates one tenant-owned performed test using the Laravel CLI contract.
func (handler *Handler) UpdatePerformedTest(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := handler.cliTenant(writer, request)
	if !ok {
		return
	}
	if handler.performedTests == nil {
		handler.writeCLIWriteUnavailable(writer, request, "CLI performed-test writer is not configured")
		return
	}

	command, ok := decodeUpdatePerformedTest(writer, request)
	if !ok {
		return
	}

	performedTestID, err := handler.performedTests.UpdatePerformedTest(request.Context(), tenant.CustomerID, command)
	if errors.Is(err, ErrNotFound) {
		writeInvalidDetails(writer)
		return
	}
	if err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"CLI performed-test update failed",
			"correlation_id", httpx.GetCorrelationID(request.Context()),
			"path", request.URL.Path,
		)
		handler.writeCLIWriteUnavailable(writer, request, "The CLI performed-test record could not be updated.")
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, map[string]int64{"idTest": performedTestID})
}

func (handler *Handler) cliTenant(writer http.ResponseWriter, request *http.Request) (auth.TenantContext, bool) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if ok {
		return tenant, true
	}
	handler.logger.ErrorContext(
		request.Context(),
		"CLI tenant context missing",
		"correlation_id", httpx.GetCorrelationID(request.Context()),
		"path", request.URL.Path,
	)
	httpx.WriteError(
		writer,
		request,
		http.StatusInternalServerError,
		"CLI_TENANT_CONTEXT_MISSING",
		"The CLI tenant context could not be resolved.",
	)
	return auth.TenantContext{}, false
}

func (handler *Handler) writeCLIWriteUnavailable(writer http.ResponseWriter, request *http.Request, message string) {
	handler.logger.LogAttrs(
		request.Context(),
		slog.LevelError,
		"CLI performed-test write unavailable",
		slog.String("correlation_id", httpx.GetCorrelationID(request.Context())),
		slog.String("path", request.URL.Path),
	)
	httpx.WriteError(writer, request, http.StatusInternalServerError, "CLI_RESULT_WRITE_UNAVAILABLE", message)
}

func decodeCreatePerformedTest(writer http.ResponseWriter, request *http.Request) (CreatePerformedTestRequest, bool) {
	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return CreatePerformedTestRequest{}, false
	}
	testCycleID, ok := requiredPositiveInt(fields, "testCycleId")
	if !ok {
		writeValidationError(writer, request, "testCycleId is required and must be a positive integer.")
		return CreatePerformedTestRequest{}, false
	}
	testID, ok := requiredPositiveInt(fields, "testId")
	if !ok {
		writeValidationError(writer, request, "testId is required and must be a positive integer.")
		return CreatePerformedTestRequest{}, false
	}
	name, ok := requiredString(fields, "name")
	if !ok {
		writeValidationError(writer, request, "name is required and must be a non-empty string.")
		return CreatePerformedTestRequest{}, false
	}
	key, ok := idempotencyKey(request)
	if !ok {
		writeValidationError(writer, request, "Idempotency-Key must be between 8 and 128 URL-safe characters.")
		return CreatePerformedTestRequest{}, false
	}
	return CreatePerformedTestRequest{TestCycleID: testCycleID, TestID: testID, Name: name, IdempotencyKey: key}, true
}

func idempotencyKey(request *http.Request) (string, bool) {
	key := request.Header.Get("Idempotency-Key")
	if key == "" {
		return "", true
	}
	if len(key) < 8 || len(key) > 128 {
		return "", false
	}
	for _, character := range key {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_') {
			return "", false
		}
	}
	return key, true
}

func decodeUpdatePerformedTest(writer http.ResponseWriter, request *http.Request) (UpdatePerformedTestRequest, bool) {
	fields, ok := decodeJSONObject(writer, request)
	if !ok {
		return UpdatePerformedTestRequest{}, false
	}
	testID, ok := requiredPositiveInt(fields, "testId")
	if !ok {
		writeValidationError(writer, request, "testId is required and must be a positive integer.")
		return UpdatePerformedTestRequest{}, false
	}
	status, ok := requiredInt(fields, "status")
	if !ok {
		writeValidationError(writer, request, "status is required and must be an integer.")
		return UpdatePerformedTestRequest{}, false
	}

	command := UpdatePerformedTestRequest{TestID: testID, Status: int(status)}
	if rawPostmanData, exists := fields["postmanData"]; exists {
		command.PostmanDataPresent = true
		if !bytes.Equal(bytes.TrimSpace(rawPostmanData), []byte("null")) {
			if !isJSONArray(rawPostmanData) {
				writeValidationError(writer, request, "postmanData must be a JSON array or null.")
				return UpdatePerformedTestRequest{}, false
			}
			redacted, err := redactPostmanJSON(rawPostmanData)
			if err != nil {
				writeValidationError(writer, request, "postmanData must be valid JSON.")
				return UpdatePerformedTestRequest{}, false
			}
			command.PostmanData = &redacted
		}
	}

	return command, true
}

func decodeJSONObject(writer http.ResponseWriter, request *http.Request) (map[string]json.RawMessage, bool) {
	defer request.Body.Close()
	body, err := io.ReadAll(io.LimitReader(request.Body, 8<<20))
	if err != nil {
		writeValidationError(writer, request, "The request body could not be read.")
		return nil, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		writeValidationError(writer, request, "The request body must be a JSON object.")
		return nil, false
	}
	return fields, true
}

func requiredPositiveInt(fields map[string]json.RawMessage, name string) (int64, bool) {
	value, ok := requiredInt(fields, name)
	return value, ok && value > 0
}

func requiredInt(fields map[string]json.RawMessage, name string) (int64, bool) {
	raw, ok := fields[name]
	if !ok {
		return 0, false
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}

func requiredString(fields map[string]json.RawMessage, name string) (string, bool) {
	raw, ok := fields[name]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func jsonStringField(fields map[string]json.RawMessage, name string) (string, bool) {
	raw, ok := fields[name]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, true
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		return string(raw), true
	}
	return "", false
}

func validJSONStringField(fields map[string]json.RawMessage, name string) (string, bool) {
	value, ok := jsonStringField(fields, name)
	if !ok {
		return "", false
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", false
	}
	return value, true
}

func isJSONArray(raw json.RawMessage) bool {
	var value []json.RawMessage
	return json.Unmarshal(raw, &value) == nil
}

func redactJSONString(value string) (string, error) {
	return redactPostmanJSON(json.RawMessage(value))
}

func redactPostmanJSON(raw json.RawMessage) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	redacted := redactJSONValue(value)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return "", fmt.Errorf("encode redacted postmanData: %w", err)
	}
	return string(encoded), nil
}

func redactJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveJSONKey(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactJSONValue(child)
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, child := range typed {
			redacted[index] = redactJSONValue(child)
		}
		return redacted
	default:
		return typed
	}
}

func isSensitiveJSONKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	sensitiveMarkers := []string{"authorization", "password", "secret", "token", "apikey", "api-key", "session", "csrf", "cookie"}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func writeValidationError(writer http.ResponseWriter, request *http.Request, message string) {
	httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "VALIDATION_FAILED", message)
}

func writeInvalidDetails(writer http.ResponseWriter) {
	httpx.WriteJSON(writer, http.StatusNotFound, map[string]string{"message": invalidDetailsMessage})
}
