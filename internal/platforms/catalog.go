package platforms

import (
	"context"
	"time"
)

// CatalogItem is a global platform catalog row exposed by legacy Laravel routes.
type CatalogItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// LocationItem is a legacy location catalog row.
type LocationItem struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// LocationQuery contains the bounded legacy grid query options.
type LocationQuery struct {
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query LocationQuery) IsPaged() bool {
	return query.Paged
}

// LocationPage is the paginated legacy grid response.
type LocationPage struct {
	Data []LocationItem    `json:"data"`
	Meta LocationPageMeta `json:"meta"`
}

// LocationPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type LocationPageMeta struct {
	Page            int    `json:"page"`
	PageSize        int    `json:"pageSize"`
	Total           int64  `json:"total"`
	LastPage        int    `json:"lastPage"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	Sort            string `json:"sort"`
	Direction       string `json:"direction"`
	Stale           bool   `json:"stale"`
	Partial         bool   `json:"partial"`
}

// CatalogRepository reads global platform reference data.
type CatalogRepository interface {
	ListTypes(ctx context.Context) ([]CatalogItem, error)
	ListStatuses(ctx context.Context) ([]CatalogItem, error)
	ListLocations(ctx context.Context, query LocationQuery) (LocationPage, error)
}
