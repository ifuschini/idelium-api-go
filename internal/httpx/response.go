package httpx

import (
	"encoding/json"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// ErrorResponse is the stable public error envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError contains safe client-facing diagnostic fields.
type APIError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// WriteJSON writes an explicit JSON response.
func WriteJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

// WriteError writes the stable error envelope without internal error details.
func WriteError(writer http.ResponseWriter, request *http.Request, status int, code string, message string) {
	WriteJSON(writer, status, ErrorResponse{
		Error: APIError{
			Code:          code,
			Message:       message,
			CorrelationID: chimiddleware.GetReqID(request.Context()),
		},
	})
}
