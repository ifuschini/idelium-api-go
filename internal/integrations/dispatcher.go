package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var ErrDeliveryNotFound = errors.New("integration delivery not found")

type Endpoint struct {
	ID              int64
	TenantID        int64
	ProjectID       int64
	Adapter         string
	URL             string
	Status          string
	SecretEncrypted string
}

type Delivery struct {
	ID            int64
	TenantID      int64
	ProjectID     int64
	EndpointID    int64
	DeliveryID    string
	Event         string
	SchemaVersion string
	Status        string
	Attempts      int
	Payload       map[string]any
}

type DispatchOutcome struct {
	Status         string
	Attempts       int
	ResponseStatus *int
	LastError      *string
	NextAttemptAt  *time.Time
	SentAt         *time.Time
}

type DeliveryStore interface {
	LoadForDispatch(context.Context, int64) (Endpoint, Delivery, error)
	SaveDispatchOutcome(context.Context, Delivery, DispatchOutcome) error
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Dispatcher struct {
	Store          DeliveryStore
	Client         HTTPDoer
	ApplicationKey []byte
	MaxAttempts    int
	Backoff        []time.Duration
	Now            func() time.Time
}

// Dispatch sends one durable delivery. Queue polling is intentionally wired
// only after the Laravel queue-drain gate is complete.
func (dispatcher Dispatcher) Dispatch(ctx context.Context, deliveryID int64) error {
	endpoint, delivery, err := dispatcher.Store.LoadForDispatch(ctx, deliveryID)
	if err != nil {
		return err
	}
	if delivery.Status == "sent" {
		return nil
	}
	if endpoint.TenantID != delivery.TenantID || endpoint.ProjectID != delivery.ProjectID || endpoint.ID != delivery.EndpointID {
		return ErrDeliveryNotFound
	}
	dispatcher = dispatcher.normalized()
	if endpoint.Status != "active" {
		return dispatcher.fail(ctx, delivery, nil, "The integration endpoint is not active.")
	}
	if !SafeDestination(endpoint.URL) {
		return dispatcher.fail(ctx, delivery, nil, "The integration destination is not safe.")
	}
	body, err := json.Marshal(adapterPayload(endpoint.Adapter, delivery, dispatcher.Now()))
	if err != nil {
		return dispatcher.fail(ctx, delivery, nil, "The integration payload could not be encoded.")
	}
	secret, err := DecryptLaravelString(dispatcher.ApplicationKey, endpoint.SecretEncrypted)
	if err != nil {
		return dispatcher.fail(ctx, delivery, nil, "The integration endpoint secret could not be decrypted.")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return dispatcher.fail(ctx, delivery, nil, "The integration destination is invalid.")
	}
	timestamp := strconv.FormatInt(dispatcher.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "." + string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Idelium-Webhook/1.0")
	request.Header.Set("Idelium-Delivery-Id", delivery.DeliveryID)
	request.Header.Set("Idelium-Event", delivery.Event)
	request.Header.Set("Idelium-Tenant-Id", fmt.Sprint(delivery.TenantID))
	request.Header.Set("Idelium-Project-Id", fmt.Sprint(delivery.ProjectID))
	request.Header.Set("Idelium-Signature", "t="+timestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("Idelium-Signature-Tolerance", "300")
	request.Header.Set("Idelium-Schema-Version", delivery.SchemaVersion)
	response, err := dispatcher.Client.Do(request)
	if err != nil {
		return dispatcher.fail(ctx, delivery, nil, "The integration delivery request failed.")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	status := response.StatusCode
	if status >= 200 && status < 300 {
		now := dispatcher.Now()
		return dispatcher.Store.SaveDispatchOutcome(ctx, delivery, DispatchOutcome{Status: "sent", Attempts: delivery.Attempts + 1, ResponseStatus: &status, SentAt: &now})
	}
	return dispatcher.fail(ctx, delivery, &status, "Webhook delivery returned a non-success status.")
}

func (dispatcher Dispatcher) fail(ctx context.Context, delivery Delivery, responseStatus *int, message string) error {
	dispatcher = dispatcher.normalized()
	attempts := delivery.Attempts + 1
	status := "failed"
	var nextAttemptAt *time.Time
	if attempts >= dispatcher.MaxAttempts {
		status = "dead_letter"
	} else {
		index := attempts - 1
		if index >= len(dispatcher.Backoff) {
			index = len(dispatcher.Backoff) - 1
		}
		next := dispatcher.Now().Add(dispatcher.Backoff[index])
		nextAttemptAt = &next
	}
	return dispatcher.Store.SaveDispatchOutcome(ctx, delivery, DispatchOutcome{Status: status, Attempts: attempts, ResponseStatus: responseStatus, LastError: &message, NextAttemptAt: nextAttemptAt})
}

func (dispatcher Dispatcher) normalized() Dispatcher {
	if dispatcher.Client == nil {
		dispatcher.Client = &http.Client{Timeout: 5 * time.Second}
	}
	if dispatcher.MaxAttempts < 1 {
		dispatcher.MaxAttempts = 3
	}
	if len(dispatcher.Backoff) == 0 {
		dispatcher.Backoff = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
	}
	if dispatcher.Now == nil {
		dispatcher.Now = time.Now
	}
	return dispatcher
}

func adapterPayload(adapter string, delivery Delivery, occurredAt time.Time) map[string]any {
	base := map[string]any{"schemaVersion": delivery.SchemaVersion, "event": delivery.Event, "deliveryId": delivery.DeliveryID, "tenantId": delivery.TenantID, "projectId": delivery.ProjectID, "occurredAt": occurredAt.Format(time.RFC3339Nano), "data": delivery.Payload}
	switch adapter {
	case "slack":
		return map[string]any{"text": "[Idelium] " + delivery.Event, "idelium": base}
	case "teams":
		return map[string]any{"type": "message", "text": "[Idelium] " + delivery.Event, "idelium": base}
	case "jira":
		description, _ := json.Marshal(base)
		return map[string]any{"summary": "[Idelium] " + delivery.Event, "description": string(description), "labels": []string{"idelium"}, "idelium": base}
	default:
		return base
	}
}
