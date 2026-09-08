package mysql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/idelium/idelium-api-go/internal/identity"
	"time"
)

func (r *BrowserAuthRepository) CreateSSOState(ctx context.Context, tenant, provider int64, state, challenge string, expires time.Time) error {
	_, e := r.database.ExecContext(ctx, `INSERT INTO sso_states(idCostumer,identityProviderId,state,codeChallenge,expiresAt,created_at) VALUES(?,?,?,?,?,NOW())`, tenant, provider, state, challenge, expires)
	if e != nil {
		return safeDatabaseFailure("persist SSO state", e)
	}
	return nil
}
func (r *BrowserAuthRepository) ConsumeSSOState(ctx context.Context, state string, now time.Time) (int64, int64, error) {
	var tenant, provider int64
	res, e := r.database.ExecContext(ctx, `UPDATE sso_states SET consumedAt=? WHERE state=? AND consumedAt IS NULL AND expiresAt>?`, now, state, now)
	if e != nil {
		return 0, 0, safeDatabaseFailure("consume SSO state", e)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, 0, errors.New("SSO state expired or already consumed")
	}
	e = r.database.QueryRowContext(ctx, `SELECT idCostumer,identityProviderId FROM sso_states WHERE state=?`, state).Scan(&tenant, &provider)
	if e != nil {
		return 0, 0, safeDatabaseFailure("load SSO state", e)
	}
	return tenant, provider, nil
}

// SSOState returns the tenant/provider binding for an unconsumed state without
// consuming it. Callers must consume the state after validating the token.
func (r *BrowserAuthRepository) SSOState(ctx context.Context, state string, now time.Time) (int64, int64, error) {
	var tenant, provider int64
	e := r.database.QueryRowContext(ctx, `SELECT idCostumer,identityProviderId FROM sso_states WHERE state=? AND consumedAt IS NULL AND expiresAt>?`, state, now).Scan(&tenant, &provider)
	if e == sql.ErrNoRows {
		return 0, 0, errors.New("SSO state expired or already consumed")
	}
	if e != nil {
		return 0, 0, safeDatabaseFailure("inspect SSO state", e)
	}
	return tenant, provider, nil
}

func (r *BrowserAuthRepository) Provider(ctx context.Context, tenant, name string) (identity.Provider, error) {
	var p identity.Provider
	var issuer, audience sql.NullString
	e := r.database.QueryRowContext(ctx, `SELECT id,type,name,issuer,audience,status FROM identity_providers WHERE idCostumer=? AND (name=? OR CAST(id AS CHAR)=?) LIMIT 1`, tenant, name, name).Scan(&p.ID, &p.Type, &p.Name, &issuer, &audience, &p.Status)
	if issuer.Valid {
		p.Issuer = issuer.String
	}
	if audience.Valid {
		p.Audience = audience.String
	}
	if e == sql.ErrNoRows {
		return p, identity.ErrProviderNotFound
	}
	if e != nil {
		return p, safeDatabaseFailure("load identity provider", e)
	}
	return p, nil
}
