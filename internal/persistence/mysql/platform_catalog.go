package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/idelium/idelium-api-go/internal/platforms"
)

// PlatformCatalogRepository reads legacy global platform catalog tables.
type PlatformCatalogRepository struct {
	database *sql.DB
}

// NewPlatformCatalogRepository creates a MySQL-backed catalog repository.
func NewPlatformCatalogRepository(database *sql.DB) *PlatformCatalogRepository {
	return &PlatformCatalogRepository{database: database}
}

// ListTypes returns platform target types ordered like the Laravel endpoint.
func (repository *PlatformCatalogRepository) ListTypes(ctx context.Context) ([]platforms.CatalogItem, error) {
	return repository.listCatalogItems(ctx, "SELECT id, name FROM types ORDER BY id ASC")
}

// ListStatuses returns platform statuses ordered like the Laravel endpoint.
func (repository *PlatformCatalogRepository) ListStatuses(ctx context.Context) ([]platforms.CatalogItem, error) {
	return repository.listCatalogItems(ctx, "SELECT id, name FROM statuses ORDER BY id ASC")
}

func (repository *PlatformCatalogRepository) listCatalogItems(ctx context.Context, query string) ([]platforms.CatalogItem, error) {
	rows, err := repository.database.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query platform catalog: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.CatalogItem, 0)
	for rows.Next() {
		var item platforms.CatalogItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan platform catalog row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read platform catalog rows: %w", err)
	}

	return items, nil
}
