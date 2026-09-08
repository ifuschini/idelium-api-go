package mysql

import (
	"context"
	"database/sql"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"golang.org/x/crypto/bcrypt"
)

func (r *BrowserAuthRepository) CreateSCIMUser(ctx context.Context, tenant int64, email, name string, active bool) (browserauth.User, error) {
	status := "active"
	if !active {
		status = "disabled"
	}
	hash, e := bcrypt.GenerateFromPassword([]byte("scim-disabled-login"), bcrypt.DefaultCost)
	if e != nil {
		return browserauth.User{}, e
	}
	res, e := r.database.ExecContext(ctx, `INSERT INTO users(name,email,password,role,idCostumer,status,created_at,updated_at) VALUES (?,? ,?,3,?,?,NOW(),NOW())`, name, email, string(hash), tenant, status)
	if e != nil {
		return browserauth.User{}, safeDatabaseFailure("create SCIM user", e)
	}
	id, _ := res.LastInsertId()
	_, _ = r.database.ExecContext(ctx, `INSERT INTO audit_events(activeTenantId,action,targetType,targetId,result,created_at) VALUES (?, 'scim.user.created','user',?,'success',NOW())`, tenant, id)
	return browserauth.User{ID: id, TenantID: tenant, ActiveTenantID: tenant, Name: name, Email: email, Role: 3, Status: status}, nil
}

func (r *BrowserAuthRepository) UpdateSCIMUser(ctx context.Context, tenant, id int64, email, name string, active bool) (browserauth.User, error) {
	status := "active"
	if !active {
		status = "disabled"
	}
	res, e := r.database.ExecContext(ctx, `UPDATE users SET name=?,email=?,status=?,updated_at=NOW() WHERE id=? AND idCostumer=?`, name, email, status, id, tenant)
	if e != nil {
		return browserauth.User{}, safeDatabaseFailure("update SCIM user", e)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return browserauth.User{}, sql.ErrNoRows
	}
	_, _ = r.database.ExecContext(ctx, `INSERT INTO audit_events(activeTenantId,action,targetType,targetId,result,created_at) VALUES (?, 'scim.user.updated','user',?,'success',NOW())`, tenant, id)
	return browserauth.User{ID: id, TenantID: tenant, ActiveTenantID: tenant, Name: name, Email: email, Role: 3, Status: status}, nil
}

func (r *BrowserAuthRepository) DeleteSCIMUser(ctx context.Context, tenant, id int64) error {
	res, e := r.database.ExecContext(ctx, `DELETE FROM users WHERE id=? AND idCostumer=?`, id, tenant)
	if e != nil {
		return safeDatabaseFailure("delete SCIM user", e)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	_, _ = r.database.ExecContext(ctx, `INSERT INTO audit_events(activeTenantId,action,targetType,targetId,result,created_at) VALUES (?, 'scim.user.deleted','user',?,'success',NOW())`, tenant, id)
	return nil
}
