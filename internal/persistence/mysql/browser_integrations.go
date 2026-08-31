package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/auditlog"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/integrations"
)

const integrationSchemaVersion = "2026-07-28.v1"

func (r *BrowserAuthRepository) ListIntegrationEndpoints(request *http.Request, actor browserauth.User, projectID int64) ([]browserauth.IntegrationEndpoint, error) {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), projectID); err != nil {
		return nil, err
	}
	rows, err := r.database.QueryContext(ctx, `SELECT id, idProject, name, adapter, url, events, status, secretEncrypted, metadata, created_at
		FROM integration_endpoints WHERE idCostumer = ? AND idProject = ? ORDER BY name`, actor.ActiveTenant(), projectID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser integration endpoints", err)
	}
	defer rows.Close()
	endpoints := []browserauth.IntegrationEndpoint{}
	for rows.Next() {
		endpoint, err := scanIntegrationEndpoint(rows)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser integration endpoints", err)
	}
	return endpoints, nil
}

func (r *BrowserAuthRepository) CreateIntegrationEndpoint(request *http.Request, actor browserauth.User, input browserauth.IntegrationEndpointCreate) (browserauth.IntegrationEndpoint, error) {
	key, err := integrationApplicationKey()
	if err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	encrypted, err := integrations.EncryptLaravelString(key, input.Secret)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, fmt.Errorf("encrypt integration endpoint secret: %w", err)
	}
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("start browser integration endpoint create", err)
	}
	defer tx.Rollback()
	if err := ensureProjectTx(ctx, tx, actor.ActiveTenant(), input.ProjectID); err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	eventsJSON, _ := json.Marshal(input.Events)
	metadataJSON, _ := json.Marshal(map[string]any{"schemaVersion": integrationSchemaVersion, "secretRotatedAt": input.Now.Format(time.RFC3339Nano)})
	result, err := tx.ExecContext(ctx, `INSERT INTO integration_endpoints
		(idCostumer, idProject, name, adapter, url, secretEncrypted, events, status, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?)`, actor.ActiveTenant(), input.ProjectID, input.Name, input.Adapter, input.URL, encrypted, string(eventsJSON), string(metadataJSON), input.Now, input.Now)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("create browser integration endpoint", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("read created browser integration endpoint id", err)
	}
	after := map[string]any{"name": input.Name, "adapter": input.Adapter, "url": input.URL, "events": input.Events, "secret": "[REDACTED]"}
	if err := recordIntegrationAudit(ctx, tx, request, actor, input.ProjectID, "integration_endpoint.create", "integration_endpoint", id, nil, after); err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("commit browser integration endpoint create", err)
	}
	return r.integrationEndpoint(ctx, actor, input.ProjectID, id)
}

func (r *BrowserAuthRepository) CreateIntegrationTestDelivery(request *http.Request, actor browserauth.User, projectID, endpointID int64, now time.Time) (browserauth.IntegrationDelivery, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("start browser integration test delivery", err)
	}
	defer tx.Rollback()
	endpoint, err := integrationEndpointTx(ctx, tx, actor.ActiveTenant(), projectID, endpointID, true)
	if err != nil {
		return browserauth.IntegrationDelivery{}, err
	}
	if endpoint.Status != "active" {
		return browserauth.IntegrationDelivery{}, browserauth.ValidationFailure{Errors: map[string][]string{"endpoint": {"The integration endpoint is disabled."}}}
	}
	accepted := false
	for _, event := range endpoint.Events {
		if event == "*" || event == "integration.test" {
			accepted = true
			break
		}
	}
	if !accepted {
		return browserauth.IntegrationDelivery{}, browserauth.ValidationFailure{Errors: map[string][]string{"event": {"The integration endpoint is not subscribed to this event."}}}
	}
	payload := map[string]any{"message": "Idelium integration test delivery.", "requestedBy": actor.Email}
	payloadJSON, _ := json.Marshal(payload)
	digest := sha256Hex(payloadJSON)
	deliveryIdentifier := "idwh_" + randomUUID()
	idempotencyKey := "integration.test:" + now.Format("20060102150405") + ":" + fmt.Sprint(actor.ID)
	result, err := tx.ExecContext(ctx, `INSERT INTO integration_deliveries
		(idCostumer, idProject, integrationEndpointId, event, deliveryId, idempotencyKey, schemaVersion, payloadDigest, status, attempts, payload, created_at, updated_at)
		VALUES (?, ?, ?, 'integration.test', ?, ?, ?, ?, 'pending', 0, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, actor.ActiveTenant(), projectID, endpointID, deliveryIdentifier, idempotencyKey, integrationSchemaVersion, digest, string(payloadJSON), now, now)
	if err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("create browser integration test delivery", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("read browser integration test delivery id", err)
	}
	createdDelivery, err := integrationDeliveryTx(ctx, tx, actor.ActiveTenant(), projectID, id, false)
	if err != nil {
		return browserauth.IntegrationDelivery{}, err
	}
	after := map[string]any{"deliveryId": createdDelivery.DeliveryID, "event": createdDelivery.Event, "status": createdDelivery.Status}
	if err := recordIntegrationAudit(ctx, tx, request, actor, projectID, "integration_delivery.test", "integration_delivery", id, nil, after); err != nil {
		return browserauth.IntegrationDelivery{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("commit browser integration test delivery", err)
	}
	return r.integrationDelivery(ctx, actor, projectID, id)
}

func (r *BrowserAuthRepository) UpdateIntegrationEndpointStatus(request *http.Request, actor browserauth.User, projectID, endpointID int64, status string, now time.Time) (browserauth.IntegrationEndpoint, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("start browser integration status update", err)
	}
	defer tx.Rollback()
	endpoint, err := integrationEndpointTx(ctx, tx, actor.ActiveTenant(), projectID, endpointID, true)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE integration_endpoints SET status = ?, updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?", status, now, endpointID, actor.ActiveTenant(), projectID); err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("update browser integration endpoint status", err)
	}
	if err := recordIntegrationAudit(ctx, tx, request, actor, projectID, "integration_endpoint.status_update", "integration_endpoint", endpointID, map[string]any{"status": endpoint.Status}, map[string]any{"status": status}); err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("commit browser integration status update", err)
	}
	return r.integrationEndpoint(ctx, actor, projectID, endpointID)
}

func (r *BrowserAuthRepository) RotateIntegrationEndpointSecret(request *http.Request, actor browserauth.User, projectID, endpointID int64, secret string, now time.Time) (browserauth.IntegrationEndpoint, error) {
	key, err := integrationApplicationKey()
	if err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	encrypted, err := integrations.EncryptLaravelString(key, secret)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, fmt.Errorf("encrypt rotated integration endpoint secret: %w", err)
	}
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("start browser integration secret rotation", err)
	}
	defer tx.Rollback()
	endpoint, err := integrationEndpointTx(ctx, tx, actor.ActiveTenant(), projectID, endpointID, true)
	if err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	metadata := endpoint.metadata
	metadata["secretRotatedAt"] = now.Format(time.RFC3339Nano)
	metadataJSON, _ := json.Marshal(metadata)
	if _, err := tx.ExecContext(ctx, "UPDATE integration_endpoints SET secretEncrypted = ?, metadata = ?, updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?", encrypted, string(metadataJSON), now, endpointID, actor.ActiveTenant(), projectID); err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("rotate browser integration endpoint secret", err)
	}
	if err := recordIntegrationAudit(ctx, tx, request, actor, projectID, "integration_endpoint.rotate_secret", "integration_endpoint", endpointID, nil, map[string]any{"secret": "[REDACTED]", "secretRotatedAt": metadata["secretRotatedAt"]}); err != nil {
		return browserauth.IntegrationEndpoint{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.IntegrationEndpoint{}, safeDatabaseFailure("commit browser integration secret rotation", err)
	}
	return r.integrationEndpoint(ctx, actor, projectID, endpointID)
}

func (r *BrowserAuthRepository) ListIntegrationDeliveries(request *http.Request, actor browserauth.User, projectID int64, status string) ([]browserauth.IntegrationDelivery, error) {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), projectID); err != nil {
		return nil, err
	}
	query := `SELECT id, deliveryId, event, status, attempts, responseStatus, nextAttemptAt, sentAt
		FROM integration_deliveries WHERE idCostumer = ? AND idProject = ?`
	args := []any{actor.ActiveTenant(), projectID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := r.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, safeDatabaseFailure("list browser integration deliveries", err)
	}
	defer rows.Close()
	deliveries := []browserauth.IntegrationDelivery{}
	for rows.Next() {
		delivery, err := scanIntegrationDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser integration deliveries", err)
	}
	return deliveries, nil
}

func (r *BrowserAuthRepository) ReplayIntegrationDelivery(request *http.Request, actor browserauth.User, projectID, deliveryID int64, now time.Time) (browserauth.IntegrationDelivery, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("start browser integration delivery replay", err)
	}
	defer tx.Rollback()
	if _, err := integrationDeliveryTx(ctx, tx, actor.ActiveTenant(), projectID, deliveryID, true); err != nil {
		return browserauth.IntegrationDelivery{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE integration_deliveries SET status = 'pending', nextAttemptAt = NULL, updated_at = ? WHERE id = ? AND idCostumer = ? AND idProject = ?", now, deliveryID, actor.ActiveTenant(), projectID); err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("replay browser integration delivery", err)
	}
	if err := recordIntegrationAudit(ctx, tx, request, actor, projectID, "integration_delivery.replay", "integration_delivery", deliveryID, nil, map[string]any{"status": "pending"}); err != nil {
		return browserauth.IntegrationDelivery{}, err
	}
	if err := tx.Commit(); err != nil {
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("commit browser integration delivery replay", err)
	}
	return r.integrationDelivery(ctx, actor, projectID, deliveryID)
}

type integrationEndpointRecord struct {
	browserauth.IntegrationEndpoint
	metadata map[string]any
}

type integrationScanner interface {
	Scan(...any) error
}

func scanIntegrationEndpoint(scanner integrationScanner) (browserauth.IntegrationEndpoint, error) {
	record, err := scanIntegrationEndpointRecord(scanner)
	return record.IntegrationEndpoint, err
}

func scanIntegrationEndpointRecord(scanner integrationScanner) (integrationEndpointRecord, error) {
	var endpoint browserauth.IntegrationEndpoint
	var eventsJSON, metadataJSON sql.NullString
	var encrypted string
	var createdAt sql.NullTime
	if err := scanner.Scan(&endpoint.ID, &endpoint.IDProject, &endpoint.Name, &endpoint.Adapter, &endpoint.URL, &eventsJSON, &endpoint.Status, &encrypted, &metadataJSON, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return integrationEndpointRecord{}, browserauth.ErrNotFound
		}
		return integrationEndpointRecord{}, safeDatabaseFailure("scan browser integration endpoint", err)
	}
	if eventsJSON.Valid {
		if err := json.Unmarshal([]byte(eventsJSON.String), &endpoint.Events); err != nil {
			return integrationEndpointRecord{}, safeDatabaseFailure("decode browser integration endpoint events", err)
		}
	}
	if len(endpoint.Events) == 0 {
		endpoint.Events = []string{"*"}
	}
	metadata := map[string]any{}
	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return integrationEndpointRecord{}, safeDatabaseFailure("decode browser integration endpoint metadata", err)
		}
	}
	endpoint.SecretConfigured = encrypted != ""
	endpoint.SchemaVersion, _ = metadata["schemaVersion"].(string)
	if endpoint.SchemaVersion == "" {
		endpoint.SchemaVersion = integrationSchemaVersion
	}
	if createdAt.Valid {
		endpoint.CreatedAt = &createdAt.Time
	}
	return integrationEndpointRecord{IntegrationEndpoint: endpoint, metadata: metadata}, nil
}

func (r *BrowserAuthRepository) integrationEndpoint(ctx context.Context, actor browserauth.User, projectID, endpointID int64) (browserauth.IntegrationEndpoint, error) {
	record, err := scanIntegrationEndpoint(r.database.QueryRowContext(ctx, `SELECT id, idProject, name, adapter, url, events, status, secretEncrypted, metadata, created_at
		FROM integration_endpoints WHERE id = ? AND idCostumer = ? AND idProject = ?`, endpointID, actor.ActiveTenant(), projectID))
	return record, err
}

func integrationEndpointTx(ctx context.Context, tx *sql.Tx, tenantID, projectID, endpointID int64, lock bool) (integrationEndpointRecord, error) {
	query := `SELECT id, idProject, name, adapter, url, events, status, secretEncrypted, metadata, created_at
		FROM integration_endpoints WHERE id = ? AND idCostumer = ? AND idProject = ?`
	if lock {
		query += " FOR UPDATE"
	}
	return scanIntegrationEndpointRecord(tx.QueryRowContext(ctx, query, endpointID, tenantID, projectID))
}

func scanIntegrationDelivery(scanner integrationScanner) (browserauth.IntegrationDelivery, error) {
	var delivery browserauth.IntegrationDelivery
	var responseStatus sql.NullInt64
	var nextAttemptAt, sentAt sql.NullTime
	if err := scanner.Scan(&delivery.ID, &delivery.DeliveryID, &delivery.Event, &delivery.Status, &delivery.Attempts, &responseStatus, &nextAttemptAt, &sentAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return browserauth.IntegrationDelivery{}, browserauth.ErrNotFound
		}
		return browserauth.IntegrationDelivery{}, safeDatabaseFailure("scan browser integration delivery", err)
	}
	if responseStatus.Valid {
		value := int(responseStatus.Int64)
		delivery.ResponseStatus = &value
	}
	if nextAttemptAt.Valid {
		delivery.NextAttemptAt = &nextAttemptAt.Time
	}
	if sentAt.Valid {
		delivery.SentAt = &sentAt.Time
	}
	return delivery, nil
}

func (r *BrowserAuthRepository) integrationDelivery(ctx context.Context, actor browserauth.User, projectID, deliveryID int64) (browserauth.IntegrationDelivery, error) {
	return scanIntegrationDelivery(r.database.QueryRowContext(ctx, `SELECT id, deliveryId, event, status, attempts, responseStatus, nextAttemptAt, sentAt
		FROM integration_deliveries WHERE id = ? AND idCostumer = ? AND idProject = ?`, deliveryID, actor.ActiveTenant(), projectID))
}

func integrationDeliveryTx(ctx context.Context, tx *sql.Tx, tenantID, projectID, deliveryID int64, lock bool) (browserauth.IntegrationDelivery, error) {
	query := `SELECT id, deliveryId, event, status, attempts, responseStatus, nextAttemptAt, sentAt
		FROM integration_deliveries WHERE id = ? AND idCostumer = ? AND idProject = ?`
	if lock {
		query += " FOR UPDATE"
	}
	return scanIntegrationDelivery(tx.QueryRowContext(ctx, query, deliveryID, tenantID, projectID))
}

func recordIntegrationAudit(ctx context.Context, tx *sql.Tx, request *http.Request, actor browserauth.User, projectID int64, action, targetType string, targetID int64, before, after map[string]any) error {
	beforeJSON, _ := json.Marshal(auditlog.Redact(before))
	afterJSON, _ := json.Marshal(auditlog.Redact(after))
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'success', ?, ?, NULL)`, actor.ID, actor.TenantID, actor.ActiveTenant(), projectID, action, targetType, fmt.Sprint(targetID), nullableJSON(beforeJSON, before != nil), nullableJSON(afterJSON, after != nil), sourceIP(request), correlationID(request))
	if err != nil {
		return safeDatabaseFailure("record browser integration audit", err)
	}
	return nil
}

func integrationApplicationKey() ([]byte, error) {
	value := os.Getenv("APP_KEY")
	filePath := os.Getenv("IDELIUM_APP_KEY_FILE")
	if filePath == "" {
		filePath = os.Getenv("APP_KEY_FILE")
	}
	if filePath != "" {
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errors.New("integration application key file is not readable")
		}
		value = strings.TrimRight(string(contents), "\r\n")
	}
	key, err := integrations.ParseApplicationKey(value)
	if err != nil {
		return nil, errors.New("integration application key is not configured")
	}
	return key, nil
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return fmt.Sprintf("%x", digest[:])
}
