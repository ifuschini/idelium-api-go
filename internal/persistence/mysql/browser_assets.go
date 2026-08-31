package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/idelium/idelium-api-go/internal/auditlog"
	"github.com/idelium/idelium-api-go/internal/browserauth"
)

func (r *BrowserAuthRepository) AssetImpact(request *http.Request, actor browserauth.User, projectID int64, assetType string, assetID int64) (browserauth.AssetImpact, error) {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), projectID); err != nil {
		return browserauth.AssetImpact{}, err
	}
	tests, err := r.assetImpactItems(ctx, "tests", actor.ActiveTenant(), projectID, assetType, assetID, nil)
	if err != nil {
		return browserauth.AssetImpact{}, err
	}
	testIDs := make([]int64, len(tests))
	for index, test := range tests {
		testIDs[index] = test.ID
	}
	cycles, err := r.assetImpactItems(ctx, "test_cycles", actor.ActiveTenant(), projectID, assetType, assetID, testIDs)
	if err != nil {
		return browserauth.AssetImpact{}, err
	}
	impact := browserauth.AssetImpact{Tests: tests, TestCycles: cycles}
	impact.Asset.AssetType = assetType
	impact.Asset.AssetID = assetID
	impact.Summary.Tests = len(tests)
	impact.Summary.TestCycles = len(cycles)
	return impact, nil
}

func (r *BrowserAuthRepository) assetImpactItems(ctx context.Context, table string, tenantID, projectID int64, assetType string, assetID int64, dependentTestIDs []int64) ([]browserauth.AssetImpactItem, error) {
	rows, err := r.database.QueryContext(ctx, "SELECT id, name, description, config FROM "+table+" WHERE idCostumer = ? AND idProject = ? ORDER BY name", tenantID, projectID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser asset impact resources", err)
	}
	defer rows.Close()
	items := []browserauth.AssetImpactItem{}
	for rows.Next() {
		var item browserauth.AssetImpactItem
		var description, config sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &description, &config); err != nil {
			return nil, safeDatabaseFailure("scan browser asset impact resource", err)
		}
		item.Description = description.String
		include := false
		if table == "tests" && assetType == "test" {
			include = item.ID == assetID
		} else if table == "test_cycles" && assetType == "test_cycle" {
			include = item.ID == assetID
		} else {
			include = configReferences(config.String, referenceKeys(assetType), []int64{assetID})
			if !include && table == "test_cycles" && len(dependentTestIDs) != 0 {
				include = configReferences(config.String, referenceKeys("test"), dependentTestIDs)
			}
		}
		if include {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser asset impact resources", err)
	}
	return items, nil
}

func (r *BrowserAuthRepository) ListAssetVersions(request *http.Request, actor browserauth.User, projectID int64, assetType string, assetID int64) ([]browserauth.AssetVersion, error) {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), projectID); err != nil {
		return nil, err
	}
	rows, err := r.database.QueryContext(ctx, `SELECT id, idProject, assetType, assetId, version, actorUserId, reason, snapshot, created_at
		FROM asset_versions WHERE idCostumer = ? AND idProject = ? AND assetType = ? AND assetId = ? ORDER BY version DESC`, actor.ActiveTenant(), projectID, assetType, assetID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser asset versions", err)
	}
	defer rows.Close()
	versions := []browserauth.AssetVersion{}
	for rows.Next() {
		version, err := scanAssetVersion(rows, false)
		if err != nil {
			return nil, err
		}
		version.Review, err = r.assetReview(ctx, actor.ActiveTenant(), projectID, version)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser asset versions", err)
	}
	return versions, nil
}

func (r *BrowserAuthRepository) GetAssetVersion(request *http.Request, actor browserauth.User, projectID, versionID int64) (browserauth.AssetVersion, error) {
	ctx := request.Context()
	version, err := scanAssetVersion(r.database.QueryRowContext(ctx, `SELECT id, idProject, assetType, assetId, version, actorUserId, reason, snapshot, created_at
		FROM asset_versions WHERE id = ? AND idCostumer = ? AND idProject = ?`, versionID, actor.ActiveTenant(), projectID), true)
	if err != nil {
		return browserauth.AssetVersion{}, err
	}
	version.Review, err = r.assetReview(ctx, actor.ActiveTenant(), projectID, version)
	return version, err
}

func (r *BrowserAuthRepository) TransitionAssetVersionReview(request *http.Request, actor browserauth.User, projectID, versionID int64, toStatus string, comment *string, now time.Time) (browserauth.AssetReviewEvent, error) {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("start browser asset review transition", err)
	}
	defer tx.Rollback()
	version, err := scanAssetVersion(tx.QueryRowContext(ctx, `SELECT id, idProject, assetType, assetId, version, actorUserId, reason, snapshot, created_at
		FROM asset_versions WHERE id = ? AND idCostumer = ? AND idProject = ? FOR UPDATE`, versionID, actor.ActiveTenant(), projectID), false)
	if err != nil {
		return browserauth.AssetReviewEvent{}, err
	}
	fromStatus := "draft"
	var latest string
	err = tx.QueryRowContext(ctx, `SELECT toStatus FROM asset_version_review_events
		WHERE idCostumer = ? AND idProject = ? AND assetVersionId = ? ORDER BY id DESC LIMIT 1 FOR UPDATE`, actor.ActiveTenant(), projectID, versionID).Scan(&latest)
	if err == nil {
		fromStatus = latest
	} else if !errors.Is(err, sql.ErrNoRows) {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("load browser asset review status", err)
	}
	allowed := map[string][]string{"draft": {"in_review"}, "in_review": {"approved", "deprecated"}, "approved": {"deprecated"}, "deprecated": {}}
	valid := false
	for _, candidate := range allowed[fromStatus] {
		if candidate == toStatus {
			valid = true
		}
	}
	if !valid {
		return browserauth.AssetReviewEvent{}, browserauth.ReviewFailure{Message: "The requested review transition is not allowed.", FromStatus: fromStatus, ToStatus: toStatus}
	}
	if toStatus == "approved" && version.ActorUserID != nil && *version.ActorUserID == actor.ID {
		return browserauth.AssetReviewEvent{}, browserauth.ReviewFailure{Message: "Asset authors cannot approve their own versions."}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO asset_version_review_events
		(idCostumer, idProject, assetVersionId, fromStatus, toStatus, comment, actorUserId, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, actor.ActiveTenant(), projectID, versionID, fromStatus, toStatus, nullableString(comment), actor.ID, now)
	if err != nil {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("create browser asset review event", err)
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("read browser asset review event id", err)
	}
	beforeJSON, _ := json.Marshal(auditlog.Redact(map[string]any{"status": fromStatus}))
	afterJSON, _ := json.Marshal(auditlog.Redact(map[string]any{"status": toStatus}))
	metadataJSON, _ := json.Marshal(auditlog.Redact(map[string]any{"assetType": version.AssetType, "assetId": version.AssetID, "version": version.Version, "reviewEventId": eventID}))
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, idProject, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (?, ?, ?, ?, 'asset_version.review_transitioned', 'asset_version', ?, ?, ?, 'success', ?, ?, ?)`, actor.ID, actor.TenantID, actor.ActiveTenant(), projectID, fmt.Sprint(versionID), string(beforeJSON), string(afterJSON), sourceIP(request), correlationID(request), string(metadataJSON))
	if err != nil {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("record browser asset review audit", err)
	}
	if err := tx.Commit(); err != nil {
		return browserauth.AssetReviewEvent{}, safeDatabaseFailure("commit browser asset review transition", err)
	}
	return browserauth.AssetReviewEvent{ID: eventID, AssetVersionID: versionID, FromStatus: fromStatus, ToStatus: toStatus, Comment: comment, ActorUserID: &actor.ID, CreatedAt: &now}, nil
}

func scanAssetVersion(scanner integrationScanner, includeSnapshot bool) (browserauth.AssetVersion, error) {
	var version browserauth.AssetVersion
	var actorUserID sql.NullInt64
	var snapshotJSON string
	var createdAt sql.NullTime
	if err := scanner.Scan(&version.ID, &version.IDProject, &version.AssetType, &version.AssetID, &version.Version, &actorUserID, &version.Reason, &snapshotJSON, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return browserauth.AssetVersion{}, browserauth.ErrNotFound
		}
		return browserauth.AssetVersion{}, safeDatabaseFailure("scan browser asset version", err)
	}
	version.ActorUserID = nullInt64Pointer(actorUserID)
	if createdAt.Valid {
		version.CreatedAt = &createdAt.Time
	}
	if includeSnapshot {
		snapshot := map[string]any{}
		if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
			return browserauth.AssetVersion{}, safeDatabaseFailure("decode browser asset version snapshot", err)
		}
		redacted, _ := auditlog.Redact(snapshot).(map[string]any)
		version.Snapshot = &redacted
	}
	return version, nil
}

func (r *BrowserAuthRepository) assetReview(ctx context.Context, tenantID, projectID int64, version browserauth.AssetVersion) (browserauth.AssetReview, error) {
	review := browserauth.AssetReview{Status: "draft", AuthorUserID: version.ActorUserID}
	var eventID, actorUserID sql.NullInt64
	var status string
	var comment sql.NullString
	var createdAt sql.NullTime
	err := r.database.QueryRowContext(ctx, `SELECT id, toStatus, comment, actorUserId, created_at FROM asset_version_review_events
		WHERE idCostumer = ? AND idProject = ? AND assetVersionId = ? ORDER BY id DESC LIMIT 1`, tenantID, projectID, version.ID).Scan(&eventID, &status, &comment, &actorUserID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return review, nil
	}
	if err != nil {
		return browserauth.AssetReview{}, safeDatabaseFailure("load browser asset review", err)
	}
	review.Status = status
	review.LastEventID = nullInt64Pointer(eventID)
	review.LastComment = nullStringPointer(comment)
	review.ReviewedByUserID = nullInt64Pointer(actorUserID)
	if createdAt.Valid {
		review.ReviewedAt = &createdAt.Time
	}
	return review, nil
}

func referenceKeys(assetType string) []string {
	switch assetType {
	case "environment":
		return []string{"environment", "environments"}
	case "plugin":
		return []string{"plugin", "plugins"}
	case "step":
		return []string{"step", "steps"}
	case "test":
		return []string{"test", "tests"}
	case "test_cycle":
		return []string{"testCycle", "testCycles", "test_cycle", "test_cycles"}
	default:
		return nil
	}
}

func configReferences(config string, keys []string, assetIDs []int64) bool {
	var decoded any
	if err := json.Unmarshal([]byte(config), &decoded); err != nil {
		return false
	}
	keySet := map[string]bool{}
	for _, key := range keys {
		keySet[key] = true
	}
	idSet := map[int64]bool{}
	for _, id := range assetIDs {
		idSet[id] = true
	}
	return nodeReferences(decoded, keySet, idSet, "")
}

func nodeReferences(node any, keys map[string]bool, ids map[int64]bool, parent string) bool {
	switch value := node.(type) {
	case float64:
		return parent != "" && keys[parent] && value == float64(int64(value)) && ids[int64(value)]
	case map[string]any:
		for key, child := range value {
			if keys[key] && arrayReferences(child, ids) {
				return true
			}
			if nodeReferences(child, keys, ids, key) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if nodeReferences(child, keys, ids, parent) {
				return true
			}
		}
	}
	return false
}

func arrayReferences(value any, ids map[int64]bool) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		switch typed := item.(type) {
		case float64:
			if typed == float64(int64(typed)) && ids[int64(typed)] {
				return true
			}
		case map[string]any:
			for _, key := range []string{"id", "assetId"} {
				if number, ok := typed[key].(float64); ok && number == float64(int64(number)) && ids[int64(number)] {
					return true
				}
			}
		}
	}
	return false
}
