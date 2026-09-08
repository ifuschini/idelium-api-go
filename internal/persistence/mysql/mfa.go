package mysql

import (
	"context"
	"database/sql"
	"time"
)

func (r *BrowserAuthRepository) SetMFASecret(ctx context.Context, user int64, encrypted string) error {
	_, e := r.database.ExecContext(ctx, `UPDATE users SET mfaSecretEncrypted=?,mfaConfirmedAt=NULL WHERE id=? AND status='active'`, encrypted, user)
	if e != nil {
		return safeDatabaseFailure("store MFA secret", e)
	}
	return nil
}
func (r *BrowserAuthRepository) GetMFASecret(ctx context.Context, user int64) (string, error) {
	var value sql.NullString
	e := r.database.QueryRowContext(ctx, `SELECT mfaSecretEncrypted FROM users WHERE id=? AND status='active'`, user).Scan(&value)
	if e != nil {
		return "", safeDatabaseFailure("load MFA secret", e)
	}
	if !value.Valid || value.String == "" {
		return "", sql.ErrNoRows
	}
	return value.String, nil
}
func (r *BrowserAuthRepository) MarkMFAConfirmed(ctx context.Context, user int64, at time.Time) error {
	_, e := r.database.ExecContext(ctx, `UPDATE users SET mfaConfirmedAt=? WHERE id=? AND status='active'`, at, user)
	if e != nil {
		return safeDatabaseFailure("confirm MFA", e)
	}
	return nil
}
