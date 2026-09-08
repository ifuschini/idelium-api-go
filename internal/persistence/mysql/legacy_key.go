package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
)

// LegacyKeyRepository authenticates legacy Laravel customer API keys.
type LegacyKeyRepository struct {
	database *sql.DB
}

type LegacyKeyLifecycle interface {
	Show(context.Context, int64, time.Time) (map[string]any, error)
	Replace(context.Context, int64, time.Time) (string, map[string]any, error)
}

func (repository *LegacyKeyRepository) Show(ctx context.Context, tenant int64, now time.Time) (map[string]any, error) {
	var expires, last sql.NullTime
	if err := repository.database.QueryRowContext(ctx, `SELECT apiKeyExpiresAt,apiKeyLastUsedAt FROM costumers WHERE id=?`, tenant).Scan(&expires, &last); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrInvalidLegacyKey
		}
		return nil, safeDatabaseFailure("show legacy API key", err)
	}
	result := map[string]any{"active": !expires.Valid || expires.Time.After(now)}
	if expires.Valid {
		result["expiresAt"] = expires.Time
	}
	if last.Valid {
		result["lastUsedAt"] = last.Time
	}
	return result, nil
}

func (repository *LegacyKeyRepository) Replace(ctx context.Context, tenant int64, now time.Time) (string, map[string]any, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw := "idk_" + base64.RawURLEncoding.EncodeToString(b)
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}
	expires := now.Add(365 * 24 * time.Hour)
	res, err := repository.database.ExecContext(ctx, `UPDATE costumers SET apiKey=?,apiKeyExpiresAt=?,apiKeyLastUsedAt=NULL WHERE id=?`, string(hash), expires, tenant)
	if err != nil {
		return "", nil, safeDatabaseFailure("replace legacy API key", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", nil, auth.ErrInvalidLegacyKey
	}
	_, _ = repository.database.ExecContext(ctx, `INSERT INTO audit_events (activeTenantId,action,targetType,targetId,result,afterValues,created_at) VALUES (?, 'legacy_api_key.replaced','costumer',?,'success',?,?)`, tenant, tenant, json.RawMessage(`{"credential":"rotated"}`), now)
	return raw, map[string]any{"active": true, "expiresAt": expires}, nil
}

// NewLegacyKeyRepository creates a MySQL-backed legacy key repository.
func NewLegacyKeyRepository(database *sql.DB) *LegacyKeyRepository {
	return &LegacyKeyRepository{database: database}
}

// AuthenticateLegacyCustomerKey returns the matching customer and records last use.
func (repository *LegacyKeyRepository) AuthenticateLegacyCustomerKey(ctx context.Context, key string, usedAt time.Time) (auth.Customer, error) {
	var customer auth.Customer
	rows, err := repository.database.QueryContext(ctx, `SELECT id,costumer,apiKey,apiKeyExpiresAt FROM costumers WHERE apiKeyExpiresAt IS NULL OR apiKeyExpiresAt > ?`, usedAt)
	if err != nil {
		return auth.Customer{}, safeDatabaseFailure("authenticate legacy API key", err)
	}
	defer rows.Close()
	var stored string
	var expires sql.NullTime
	for rows.Next() {
		if err := rows.Scan(&customer.ID, &customer.Name, &stored, &expires); err != nil {
			return auth.Customer{}, safeDatabaseFailure("authenticate legacy API key", err)
		}
		if strings.HasPrefix(stored, "$2") {
			if bcrypt.CompareHashAndPassword([]byte(stored), []byte(key)) == nil {
				break
			}
		} else if stored == key {
			break
		}
		customer = auth.Customer{}
	}
	if customer.ID == 0 {
		err = sql.ErrNoRows
	}
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Customer{}, auth.ErrInvalidLegacyKey
	}
	if err != nil {
		return auth.Customer{}, safeDatabaseFailure("authenticate legacy API key", err)
	}

	if _, err := repository.database.ExecContext(
		ctx,
		`UPDATE costumers SET apiKeyLastUsedAt = ? WHERE id = ?`,
		usedAt,
		customer.ID,
	); err != nil {
		return auth.Customer{}, fmt.Errorf("record legacy API key use: %w", safeDatabaseFailure("record legacy API key use", err))
	}

	return customer, nil
}
