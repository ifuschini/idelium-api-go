package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	err := r.database.QueryRowContext(ctx, `SELECT u.id, u.idCostumer, COALESCE(sessions.activeTenantId, sessions.idCostumer), u.name, u.email, u.role, u.password, u.status, sessions.impersonationReason, sessions.impersonationExpiresAt
		FROM go_browser_sessions AS sessions
		JOIN users AS u ON u.id = sessions.userId AND u.idCostumer = sessions.idCostumer
		WHERE sessions.idHash = ? AND sessions.expiresAt > ? AND u.status = 'active'
		LIMIT 1`, tokenHash(sessionID), now).Scan(&user.ID, &user.TenantID, &user.ActiveTenantID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.Status, &user.ImpersonationReason, &user.ImpersonationExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.User{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.User{}, safeDatabaseFailure("load browser session", err)
	}
	return user, nil
}

func (r *BrowserAuthRepository) ListProjects(ctx context.Context, tenantID int64) ([]browserauth.Project, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at, idCostumer FROM projects WHERE idCostumer = ? ORDER BY created_at ASC`, tenantID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser header projects", err)
	}
	defer rows.Close()

	projects := []browserauth.Project{}
	for rows.Next() {
		var project browserauth.Project
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt, &project.IDCostumer); err != nil {
			return nil, safeDatabaseFailure("scan browser header projects", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser header projects", err)
	}
	return projects, nil
}

func (r *BrowserAuthRepository) ListCustomers(ctx context.Context) ([]browserauth.Customer, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, costumer, description, licenseExpiration, created_at, updated_at FROM costumers ORDER BY created_at ASC`)
	if err != nil {
		return nil, safeDatabaseFailure("list browser header customers", err)
	}
	defer rows.Close()

	customers := []browserauth.Customer{}
	for rows.Next() {
		var customer browserauth.Customer
		if err := rows.Scan(&customer.ID, &customer.Costumer, &customer.Description, &customer.LicenseExpiration, &customer.CreatedAt, &customer.UpdatedAt); err != nil {
			return nil, safeDatabaseFailure("scan browser header customers", err)
		}
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser header customers", err)
	}
	return customers, nil
}

func (r *BrowserAuthRepository) CustomerExists(ctx context.Context, customerID int64) (bool, error) {
	var exists int
	err := r.database.QueryRowContext(ctx, `SELECT 1 FROM costumers WHERE id = ? LIMIT 1`, customerID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, safeDatabaseFailure("lookup browser target customer", err)
	}
	return true, nil
}

func (r *BrowserAuthRepository) SwitchTenant(ctx context.Context, tenantSwitch browserauth.TenantSwitch) error {
	result, err := r.database.ExecContext(ctx, `UPDATE go_browser_sessions
		SET activeTenantId = ?, impersonationReason = ?, impersonationExpiresAt = ?, updated_at = ?
		WHERE idHash = ? AND userId = ? AND idCostumer = ? AND expiresAt > ?`, tenantSwitch.ActiveTenant, tenantSwitch.Reason, tenantSwitch.ExpiresAt, tenantSwitch.Now, tokenHash(tenantSwitch.SessionID), tenantSwitch.UserID, tenantSwitch.ActorTenant, tenantSwitch.Now)
	if err != nil {
		return safeDatabaseFailure("switch browser tenant", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return safeDatabaseFailure("count browser tenant switch", err)
	}
	if affected == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) RecordTenantSwitch(ctx context.Context, event browserauth.AuditEvent) error {
	beforeValues, err := json.Marshal(event.BeforeValues)
	if err != nil {
		return fmt.Errorf("encode tenant switch audit before values: %w", err)
	}
	afterValues, err := json.Marshal(event.AfterValues)
	if err != nil {
		return fmt.Errorf("encode tenant switch audit after values: %w", err)
	}
	_, err = r.database.ExecContext(ctx, `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (?, ?, ?, 'tenant.switch', 'costumer', ?, ?, ?, 'success', ?, ?, NULL)`, event.ActorUserID, event.ActorTenantID, event.ActiveTenantID, fmt.Sprint(event.TargetID), string(beforeValues), string(afterValues), event.SourceIP, event.CorrelationID)
	if err != nil {
		return safeDatabaseFailure("record tenant switch audit", err)
	}
	return nil
}

func tokenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
