package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// SecureHeaders adds response headers that are safe for API endpoints.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(writer, request)
	})
}

// Recoverer converts panics into a safe error and records a server-side stack.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(
						request.Context(),
						"request panic recovered",
						"error_type", "panic",
						"stack", string(debug.Stack()),
					)
					WriteError(
						writer,
						request,
						http.StatusInternalServerError,
						"INTERNAL_ERROR",
						"The request could not be completed.",
					)
				}
			}()

			next.ServeHTTP(writer, request)
		})
	}
}

// AccessLogger records safe request metadata and deliberately excludes query
// strings, headers, cookies, and bodies.
func AccessLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			response := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}

			next.ServeHTTP(response, request)

			logger.InfoContext(
				request.Context(),
				"HTTP request completed",
				"method", request.Method,
				"path", request.URL.Path,
				"status", response.status,
				"bytes", response.bytes,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(payload []byte) (int, error) {
	written, err := recorder.ResponseWriter.Write(payload)
	recorder.bytes += written
	return written, err
}
