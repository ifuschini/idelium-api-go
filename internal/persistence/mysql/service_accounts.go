package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/idelium/idelium-api-go/internal/serviceaccounts"
	"golang.org/x/crypto/bcrypt"
)

type ServiceAccountRepository struct{ database *sql.DB }

func NewServiceAccountRepository(database *sql.DB) *ServiceAccountRepository {
	return &ServiceAccountRepository{database: database}
}
func (r *ServiceAccountRepository) List(ctx context.Context, tenant int64) ([]serviceaccounts.Account, error) {
	rows, e := r.database.QueryContext(ctx, `SELECT id,name,credentialId,idProject,scopes,expiresAt,revokedAt,lastUsedAt FROM service_accounts WHERE idCostumer=? ORDER BY id`, tenant)
	if e != nil {
		return nil, safeDatabaseFailure("list service accounts", e)
	}
	defer rows.Close()
	out := []serviceaccounts.Account{}
	for rows.Next() {
		var a serviceaccounts.Account
		var raw sql.NullString
		if e := rows.Scan(&a.ID, &a.Name, &a.CredentialID, &a.ProjectID, &raw, &a.ExpiresAt, &a.RevokedAt, &a.LastUsedAt); e != nil {
			return nil, safeDatabaseFailure("scan service accounts", e)
		}
		if raw.Valid {
			_ = json.Unmarshal([]byte(raw.String), &a.Scopes)
		}
		out = append(out, a)
	}
	if e := rows.Err(); e != nil {
		return nil, safeDatabaseFailure("read service accounts", e)
	}
	return out, nil
}
func (r *ServiceAccountRepository) Create(ctx context.Context, tenant int64, a serviceaccounts.Account, secret string) (serviceaccounts.Account, error) {
	hash, e := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if e != nil {
		return serviceaccounts.Account{}, safeDatabaseFailure("hash service account secret", e)
	}
	scopes, _ := json.Marshal(a.Scopes)
	res, e := r.database.ExecContext(ctx, `INSERT INTO service_accounts (idCostumer,idProject,name,credentialId,secretHash,scopes,expiresAt,created_at,updated_at) VALUES (?,?,?,?,?,?,?,NOW(),NOW())`, tenant, a.ProjectID, a.Name, a.CredentialID, string(hash), string(scopes), a.ExpiresAt)
	if e != nil {
		return serviceaccounts.Account{}, safeDatabaseFailure("create service account", e)
	}
	a.ID, _ = res.LastInsertId()
	if _, e = r.database.ExecContext(ctx, `INSERT INTO audit_events (activeTenantId, action, targetType, targetId, result, created_at) VALUES (?, 'service_account.created', 'service_account', ?, 'success', NOW())`, tenant, a.ID); e != nil {
		return serviceaccounts.Account{}, safeDatabaseFailure("audit service account creation", e)
	}
	return a, nil
}
func (r *ServiceAccountRepository) Revoke(ctx context.Context, tenant, id int64, now time.Time) error {
	res, e := r.database.ExecContext(ctx, `UPDATE service_accounts SET revokedAt=?,updated_at=? WHERE id=? AND idCostumer=? AND revokedAt IS NULL`, now, now, id, tenant)
	if e != nil {
		return safeDatabaseFailure("revoke service account", e)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return serviceaccounts.ErrNotFound
	}
	if _, e = r.database.ExecContext(ctx, `INSERT INTO audit_events (activeTenantId, action, targetType, targetId, result, created_at) VALUES (?, 'service_account.revoked', 'service_account', ?, 'success', ?)`, tenant, id, now); e != nil {
		return safeDatabaseFailure("audit service account revocation", e)
	}
	return nil
}
