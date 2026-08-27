package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

type BrowserAuthRepository struct{ database *sql.DB }

func NewBrowserAuthRepository(database *sql.DB) *BrowserAuthRepository {
	return &BrowserAuthRepository{database: database}
}

func (r *BrowserAuthRepository) FindByEmail(ctx context.Context, email string) (browserauth.User, error) {
	var user browserauth.User
	err := r.database.QueryRowContext(ctx, `SELECT id, idCostumer, name, email, role, password, status FROM users WHERE email = ? LIMIT 1`, email).Scan(&user.ID, &user.TenantID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.User{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.User{}, safeDatabaseFailure("find browser user", err)
	}
	return user, nil
}

func (r *BrowserAuthRepository) Create(ctx context.Context, session browserauth.Session) error {
	_, err := r.database.ExecContext(ctx, `INSERT INTO go_browser_sessions (idHash, userId, idCostumer, csrfTokenHash, expiresAt) VALUES (?, ?, ?, ?, ?)`, tokenHash(session.ID), session.UserID, session.TenantID, tokenHash(session.CSRFToken), session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create browser session: %w", safeDatabaseFailure("create browser session", err))
	}
	return nil
}

func (r *BrowserAuthRepository) Delete(ctx context.Context, sessionID string) error {
	result, err := r.database.ExecContext(ctx, `DELETE FROM go_browser_sessions WHERE idHash = ?`, tokenHash(sessionID))
	if err != nil {
		return fmt.Errorf("delete browser session: %w", safeDatabaseFailure("delete browser session", err))
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted browser session: %w", safeDatabaseFailure("count deleted browser session", err))
	}
	if affected == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) Get(ctx context.Context, sessionID string, now time.Time) (browserauth.User, error) {
	var user browserauth.User
	err := r.database.QueryRowContext(ctx, `SELECT u.id, u.idCostumer, u.name, u.email, u.role, u.password, u.status
		FROM go_browser_sessions AS sessions
		JOIN users AS u ON u.id = sessions.userId AND u.idCostumer = sessions.idCostumer
		WHERE sessions.idHash = ? AND sessions.expiresAt > ? AND u.status = 'active'
		LIMIT 1`, tokenHash(sessionID), now).Scan(&user.ID, &user.TenantID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.User{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.User{}, safeDatabaseFailure("load browser session", err)
	}
	return user, nil
}

func tokenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
