package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/idelium/idelium-api-go/internal/integrations"
)

// LoadForDispatch loads a delivery only when its endpoint carries the same
// tenant and project ownership tuple.
func (r *BrowserAuthRepository) LoadForDispatch(ctx context.Context, deliveryID int64) (integrations.Endpoint, integrations.Delivery, error) {
	var endpoint integrations.Endpoint
	var delivery integrations.Delivery
	var payloadJSON sql.NullString
	err := r.database.QueryRowContext(ctx, `SELECT
		e.id, e.idCostumer, e.idProject, e.adapter, e.url, e.status, e.secretEncrypted,
		d.id, d.idCostumer, d.idProject, d.integrationEndpointId, d.deliveryId, d.event,
		d.schemaVersion, d.status, d.attempts, d.payload
		FROM integration_deliveries d
		JOIN integration_endpoints e ON e.id = d.integrationEndpointId
			AND e.idCostumer = d.idCostumer AND e.idProject = d.idProject
		WHERE d.id = ?`, deliveryID).Scan(
		&endpoint.ID, &endpoint.TenantID, &endpoint.ProjectID, &endpoint.Adapter, &endpoint.URL, &endpoint.Status, &endpoint.SecretEncrypted,
		&delivery.ID, &delivery.TenantID, &delivery.ProjectID, &delivery.EndpointID, &delivery.DeliveryID, &delivery.Event,
		&delivery.SchemaVersion, &delivery.Status, &delivery.Attempts, &payloadJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return integrations.Endpoint{}, integrations.Delivery{}, integrations.ErrDeliveryNotFound
	}
	if err != nil {
		return integrations.Endpoint{}, integrations.Delivery{}, safeDatabaseFailure("load integration delivery for dispatch", err)
	}
	delivery.Payload = map[string]any{}
	if payloadJSON.Valid {
		if err := json.Unmarshal([]byte(payloadJSON.String), &delivery.Payload); err != nil {
			return integrations.Endpoint{}, integrations.Delivery{}, safeDatabaseFailure("decode integration delivery payload", err)
		}
	}
	return endpoint, delivery, nil
}

// SaveDispatchOutcome applies an optimistic attempt guard so two consumers
// cannot both finalize the same attempt.
func (r *BrowserAuthRepository) SaveDispatchOutcome(ctx context.Context, delivery integrations.Delivery, outcome integrations.DispatchOutcome) error {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start integration delivery outcome", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE integration_deliveries SET
		status = ?, attempts = ?, responseStatus = ?, lastError = ?, nextAttemptAt = ?, sentAt = ?, updated_at = ?
		WHERE id = ? AND idCostumer = ? AND idProject = ? AND integrationEndpointId = ? AND attempts = ? AND status <> 'sent'`,
		outcome.Status, outcome.Attempts, nullableInt(outcome.ResponseStatus), nullableString(outcome.LastError), nullableTime(outcome.NextAttemptAt), nullableTime(outcome.SentAt), time.Now().UTC(),
		delivery.ID, delivery.TenantID, delivery.ProjectID, delivery.EndpointID, delivery.Attempts)
	if err != nil {
		return safeDatabaseFailure("save integration delivery outcome", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return safeDatabaseFailure("count integration delivery outcome", err)
	}
	if count != 1 {
		return integrations.ErrDeliveryNotFound
	}
	metadata, _ := json.Marshal(map[string]any{"responseStatus": nullableInt(outcome.ResponseStatus), "attempts": outcome.Attempts, "nextAttemptAt": nullableTime(outcome.NextAttemptAt), "reason": nullableString(outcome.LastError), "job": "go.integration-delivery"})
	after, _ := json.Marshal(map[string]any{"status": outcome.Status, "deliveryId": delivery.DeliveryID})
	resultName := "failure"
	if outcome.Status == "sent" {
		resultName = "success"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (NULL, ?, ?, ?, 'integration_delivery.dispatch', 'integration_delivery', ?, NULL, ?, ?, NULL, ?, ?)`, delivery.TenantID, delivery.TenantID, delivery.ProjectID, fmt.Sprint(delivery.ID), string(after), resultName, randomUUID(), string(metadata))
	if err != nil {
		return safeDatabaseFailure("record integration delivery dispatch audit", err)
	}
	if err := tx.Commit(); err != nil {
		return safeDatabaseFailure("commit integration delivery outcome", err)
	}
	return nil
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
