// Package healthprobe provides the container healthcheck used by the Go API.
package healthprobe

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Check performs a bounded request and succeeds only for a 2xx response.
func Check(ctx context.Context, endpoint string, timeout time.Duration) error {
	if endpoint == "" {
		return fmt.Errorf("healthcheck endpoint must not be empty")
	}
	if timeout <= 0 {
		return fmt.Errorf("healthcheck timeout must be positive")
	}
	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create healthcheck request: %w", err)
	}
	response, err := (&http.Client{Timeout: timeout}).Do(request)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("healthcheck returned status %d", response.StatusCode)
	}
	return nil
}
