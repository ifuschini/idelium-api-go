package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/idelium/idelium-api-go/internal/identity"
)

type IdentityProviderRepository struct{ database *sql.DB }

func NewIdentityProviderRepository(db *sql.DB) *IdentityProviderRepository {
	return &IdentityProviderRepository{database: db}
}
func (r *IdentityProviderRepository) ListProviders(ctx context.Context, tenant int64) ([]identity.Provider, error) {
	rows, e := r.database.QueryContext(ctx, `SELECT id,type,name,issuer,audience,status FROM identity_providers WHERE idCostumer=? ORDER BY id`, tenant)
	if e != nil {
		return nil, safeDatabaseFailure("list identity providers", e)
	}
	defer rows.Close()
	out := []identity.Provider{}
	for rows.Next() {
		var p identity.Provider
		var issuer, audience sql.NullString
		if e := rows.Scan(&p.ID, &p.Type, &p.Name, &issuer, &audience, &p.Status); e != nil {
			return nil, safeDatabaseFailure("scan identity providers", e)
		}
		if issuer.Valid {
			p.Issuer = issuer.String
		}
		if audience.Valid {
			p.Audience = audience.String
		}
		out = append(out, p)
	}
	if e := rows.Err(); e != nil {
		return nil, safeDatabaseFailure("read identity providers", e)
	}
	return out, nil
}
func (r *IdentityProviderRepository) CreateProvider(ctx context.Context, tenant int64, p identity.Provider) (identity.Provider, error) {
	meta, _ := json.Marshal(map[string]string{"source": "go"})
	res, e := r.database.ExecContext(ctx, `INSERT INTO identity_providers (idCostumer,type,name,issuer,audience,status,metadata,created_at,updated_at) VALUES (?,?,?,?,?,'active',?,NOW(),NOW())`, tenant, p.Type, p.Name, p.Issuer, p.Audience, string(meta))
	if e != nil {
		return identity.Provider{}, safeDatabaseFailure("create identity provider", e)
	}
	p.ID, _ = res.LastInsertId()
	p.Status = "active"
	_, _ = r.database.ExecContext(ctx, `INSERT INTO audit_events(activeTenantId,action,targetType,targetId,result,created_at) VALUES (?, 'identity_provider.created','identity_provider',?,'success',NOW())`, tenant, p.ID)
	return p, nil
}
