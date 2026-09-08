package mysql

import (
	"context"
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
