package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/idelium/idelium-api-go/internal/auth"
)

// LegacyKeyRepository authenticates legacy Laravel customer API keys.
type LegacyKeyRepository struct {
	database *sql.DB
}

// NewLegacyKeyRepository creates a MySQL-backed legacy key repository.
func NewLegacyKeyRepository(database *sql.DB) *LegacyKeyRepository {
	return &LegacyKeyRepository{database: database}
}

// AuthenticateLegacyCustomerKey returns the matching customer and records last use.
func (repository *LegacyKeyRepository) AuthenticateLegacyCustomerKey(ctx context.Context, key string, usedAt time.Time) (auth.Customer, error) {
	var customer auth.Customer
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT id, costumer
		 FROM costumers
		 WHERE apiKey = ?
		   AND (apiKeyExpiresAt IS NULL OR apiKeyExpiresAt > ?)
		 LIMIT 1`,
		key,
		usedAt,
	).Scan(&customer.ID, &customer.Name)
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
