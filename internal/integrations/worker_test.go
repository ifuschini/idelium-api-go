package integrations

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

type workerStoreStub struct {
	storeStub
	ids   []int64
	calls int
}

func (store *workerStoreStub) NextDispatchID(context.Context, time.Time) (int64, error) {
	store.calls++
	if len(store.ids) == 0 {
		return 0, ErrNoPendingDelivery
	}
	id := store.ids[0]
	store.ids = store.ids[1:]
	return id, nil
}

func TestWorkerDispatchesVersionedDeliveryAndStopsCleanly(t *testing.T) {
	key, _ := ParseApplicationKey("base64:" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	store := &workerStoreStub{storeStub: storeStub{endpoint: Endpoint{ID: 9, TenantID: 11, ProjectID: 5, Adapter: "webhook", URL: "https://93.184.216.34/hook", Status: "active"}, delivery: Delivery{ID: 20, TenantID: 11, ProjectID: 5, EndpointID: 9, DeliveryID: "idwh_test", Event: "integration.test", SchemaVersion: "2026-07-28.v1", Status: "pending"}}, ids: []int64{20}}
	encrypted, _ := EncryptLaravelString(key, "secret-value")
	store.endpoint.SecretEncrypted = encrypted
	ctx, cancel := context.WithCancel(context.Background())
	doer := doerFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Idelium-Schema-Version") != "2026-07-28.v1" {
			t.Fatalf("versioned delivery header missing: %#v", request.Header)
		}
		cancel()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	err := (Worker{Store: store, ApplicationKey: key, PollInterval: time.Millisecond, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Dispatcher: Dispatcher{Client: doer}}).Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("worker returned an error: %v", err)
	}
	if store.outcome.Status != "sent" || store.calls == 0 {
		t.Fatalf("worker did not dispatch delivery: %#v", store)
	}
}

func TestWorkerFailsClosedWithoutStoreOrApplicationKey(t *testing.T) {
	if err := (Worker{}).Run(context.Background()); err == nil {
		t.Fatal("worker accepted missing store")
	}
	if err := (Worker{Store: &workerStoreStub{}}).Run(context.Background()); !errors.Is(err, ErrInvalidApplicationKey) {
		t.Fatalf("expected application-key validation, got %v", err)
	}
}
