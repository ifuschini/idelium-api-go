package mysql

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/auditlog"
	"github.com/idelium/idelium-api-go/internal/browserauth"
)

func (r *BrowserAuthRepository) ListAuditEvents(request *http.Request, actor browserauth.User, filter browserauth.AuditEventFilter) ([]browserauth.AuditEventRecord, error) {
	query := `SELECT id, actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId,
		beforeValues, afterValues, result, sourceIp, correlationId, metadata, created_at
		FROM audit_events WHERE activeTenantId = ?`
	args := []any{actor.ActiveTenant()}
	for _, item := range []struct {
		column string
		value  string
	}{{"action", filter.Action}, {"targetType", filter.TargetType}, {"targetId", filter.TargetID}, {"correlationId", filter.CorrelationID}} {
		if item.value != "" {
			query += " AND " + item.column + " = ?"
			args = append(args, item.value)
		}
	}
	if filter.From != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.To)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, filter.Limit)
	rows, err := r.database.QueryContext(request.Context(), query, args...)
	if err != nil {
		return nil, safeDatabaseFailure("list browser audit events", err)
	}
	defer rows.Close()
	events := []browserauth.AuditEventRecord{}
	for rows.Next() {
		var event browserauth.AuditEventRecord
		var actorUserID, actorTenantID, projectID sql.NullInt64
		var targetID, sourceIP sql.NullString
		var beforeJSON, afterJSON, metadataJSON sql.NullString
		if err := rows.Scan(&event.ID, &actorUserID, &actorTenantID, &event.ActiveTenantID, &projectID, &event.Action, &event.TargetType, &targetID, &beforeJSON, &afterJSON, &event.Result, &sourceIP, &event.CorrelationID, &metadataJSON, &event.CreatedAt); err != nil {
			return nil, safeDatabaseFailure("scan browser audit event", err)
		}
		event.ActorUserID = nullInt64Pointer(actorUserID)
		event.ActorTenantID = nullInt64Pointer(actorTenantID)
		event.IDProject = nullInt64Pointer(projectID)
		event.TargetID = nullStringPointer(targetID)
		event.SourceIP = nullStringPointer(sourceIP)
		if event.BeforeValues, err = decodeRedactedAuditMap(beforeJSON); err != nil {
			return nil, err
		}
		if event.AfterValues, err = decodeRedactedAuditMap(afterJSON); err != nil {
			return nil, err
		}
		if event.Metadata, err = decodeRedactedAuditMap(metadataJSON); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser audit events", err)
	}
	return events, nil
}

func decodeRedactedAuditMap(value sql.NullString) (map[string]any, error) {
	if !value.Valid {
		return nil, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(value.String), &decoded); err != nil {
		return nil, safeDatabaseFailure("decode browser audit event values", err)
	}
	redacted, _ := auditlog.Redact(decoded).(map[string]any)
	return redacted, nil
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
