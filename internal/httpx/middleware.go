package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// CorrelationIDHeader is the public request and response correlation header.
	CorrelationIDHeader    = "X-Correlation-ID"
	maxCorrelationIDLength = 128
)

type correlationIDContextKey struct{}

var fallbackCorrelationSequence atomic.Uint64

// CorrelationID accepts a bounded safe identifier or generates one for the
// request. Invalid caller input is replaced rather than reflected or logged.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		correlationID := strings.TrimSpace(request.Header.Get(CorrelationIDHeader))
		if !validCorrelationID(correlationID) {
			correlationID = newCorrelationID()
		}

		writer.Header().Set(CorrelationIDHeader, correlationID)
		ctx := context.WithValue(request.Context(), correlationIDContextKey{}, correlationID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// GetCorrelationID returns the validated identifier attached to a request.
func GetCorrelationID(ctx context.Context) string {
	correlationID, _ := ctx.Value(correlationIDContextKey{}).(string)
	return correlationID
}

func validCorrelationID(value string) bool {
	if value == "" || len(value) > maxCorrelationIDLength {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

func newCorrelationID() string {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err == nil {
		return hex.EncodeToString(identifier)
	}
	return "request-" + time.Now().UTC().Format("20060102T150405.000000000") +
		"-" + strconv.FormatUint(fallbackCorrelationSequence.Add(1), 36)
}

// SecureHeaders adds response headers that are safe for API endpoints.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		writer.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		writer.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
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
						"correlation_id", GetCorrelationID(request.Context()),
						"method", request.Method,
						"path", request.URL.Path,
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
			response := &statusRecorder{ResponseWriter: writer}

			next.ServeHTTP(response, request)
			if response.status == 0 {
				response.status = http.StatusOK
			}

			logger.InfoContext(
				request.Context(),
				"HTTP request completed",
				"correlation_id", GetCorrelationID(request.Context()),
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

// Unwrap preserves support for optional response-controller interfaces.
func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(payload []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}
	written, err := recorder.ResponseWriter.Write(payload)
	recorder.bytes += written
	return written, err
}
