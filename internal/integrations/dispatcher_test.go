package integrations

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type storeStub struct {
	endpoint Endpoint
	delivery Delivery
	outcome  DispatchOutcome
	err      error
}

func (store *storeStub) LoadForDispatch(context.Context, int64) (Endpoint, Delivery, error) {
	return store.endpoint, store.delivery, store.err
}
func (store *storeStub) SaveDispatchOutcome(_ context.Context, _ Delivery, outcome DispatchOutcome) error {
	store.outcome = outcome
	return store.err
}

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) { return function(request) }

func TestDispatcherSignsCanonicalPayloadAndMarksSent(t *testing.T) {
	key, _ := ParseApplicationKey("base64:" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	encrypted, _ := EncryptLaravelString(key, "super-secret-value")
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	store := &storeStub{endpoint: Endpoint{ID: 9, TenantID: 11, ProjectID: 5, Adapter: "slack", URL: "https://93.184.216.34/hooks", Status: "active", SecretEncrypted: encrypted}, delivery: Delivery{ID: 20, TenantID: 11, ProjectID: 5, EndpointID: 9, DeliveryID: "idwh_test", Event: "test.completed", SchemaVersion: "2026-07-28.v1", Status: "pending", Payload: map[string]any{"status": "passed"}}}
	doer := doerFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"text":"[Idelium] test.completed"`) || request.Header.Get("Idelium-Delivery-Id") != "idwh_test" || !strings.HasPrefix(request.Header.Get("Idelium-Signature"), "t=1788177600,v1=") {
			t.Fatalf("unexpected signed integration request: headers=%#v body=%s", request.Header, body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	if err := (Dispatcher{Store: store, Client: doer, ApplicationKey: key, Now: func() time.Time { return now }}).Dispatch(context.Background(), 20); err != nil {
		t.Fatal(err)
	}
	if store.outcome.Status != "sent" || store.outcome.Attempts != 1 || store.outcome.SentAt == nil {
		t.Fatalf("unexpected sent outcome: %#v", store.outcome)
	}
}

func TestDispatcherUsesBoundedRetryAndDeadLetterWithoutLeakingFailures(t *testing.T) {
	key, _ := ParseApplicationKey("base64:" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	encrypted, _ := EncryptLaravelString(key, "super-secret-value")
	store := &storeStub{endpoint: Endpoint{ID: 9, TenantID: 11, ProjectID: 5, Adapter: "webhook", URL: "https://93.184.216.34/hooks", Status: "active", SecretEncrypted: encrypted}, delivery: Delivery{ID: 20, TenantID: 11, ProjectID: 5, EndpointID: 9, DeliveryID: "idwh_test", Event: "test.failed", SchemaVersion: "2026-07-28.v1", Status: "failed", Attempts: 2}}
	doer := doerFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("upstream included sensitive-value")
	})
	if err := (Dispatcher{Store: store, Client: doer, ApplicationKey: key, MaxAttempts: 3}).Dispatch(context.Background(), 20); err != nil {
		t.Fatal(err)
	}
	if store.outcome.Status != "dead_letter" || store.outcome.Attempts != 3 || store.outcome.NextAttemptAt != nil || store.outcome.LastError == nil || strings.Contains(*store.outcome.LastError, "sensitive-value") {
		t.Fatalf("unexpected safe dead-letter outcome: %#v", store.outcome)
	}
}

func TestDispatcherRejectsCrossTenantEndpointLink(t *testing.T) {
	store := &storeStub{endpoint: Endpoint{ID: 9, TenantID: 42, ProjectID: 5}, delivery: Delivery{ID: 20, TenantID: 11, ProjectID: 5, EndpointID: 9}}
	err := (Dispatcher{Store: store}).Dispatch(context.Background(), 20)
	if !errors.Is(err, ErrDeliveryNotFound) {
		t.Fatalf("expected tenant mismatch to be hidden, got %v", err)
	}
}
