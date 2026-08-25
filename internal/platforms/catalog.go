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

// BrandItem is a legacy device brand catalog row.
type BrandItem struct {
	ID        int64      `json:"id"`
	Brand     string     `json:"brand"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// ModelItem is a legacy device model catalog row.
type ModelItem struct {
	ID        int64      `json:"id"`
	Model     string     `json:"model"`
	IDBrand   int64      `json:"idBrand"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// OperatingSystemItem is a legacy operating-system catalog row.
type OperatingSystemItem struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Type      int64      `json:"type"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// OperatingSystemVersionItem is a legacy operating-system version catalog row.
type OperatingSystemVersionItem struct {
	ID        int64      `json:"id"`
	Version   string     `json:"version"`
	IDOs      int64      `json:"idOs"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// BrowserItem is a legacy browser catalog row.
type BrowserItem struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	IDOs      int64      `json:"idOs"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// BrowserVersionItem is a legacy browser version catalog row.
type BrowserVersionItem struct {
	ID        int64      `json:"id"`
	Version   string     `json:"version"`
	IDBrowser int64      `json:"idBrowser"`
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

// BrandQuery contains the bounded legacy brand grid query options.
type BrandQuery struct {
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query BrandQuery) IsPaged() bool {
	return query.Paged
}

// ModelQuery contains the bounded legacy model grid query options.
type ModelQuery struct {
	IDBrand   int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query ModelQuery) IsPaged() bool {
	return query.Paged
}

// OperatingSystemQuery contains the bounded legacy operating-system grid query options.
type OperatingSystemQuery struct {
	TypeID    int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query OperatingSystemQuery) IsPaged() bool {
	return query.Paged
}

// OperatingSystemVersionQuery contains the bounded legacy OS-version grid query options.
type OperatingSystemVersionQuery struct {
	IDOs      int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query OperatingSystemVersionQuery) IsPaged() bool {
	return query.Paged
}

// BrowserQuery contains the bounded legacy browser grid query options.
type BrowserQuery struct {
	IDOs      int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query BrowserQuery) IsPaged() bool {
	return query.Paged
}

// BrowserVersionQuery contains the bounded legacy browser-version grid query options.
type BrowserVersionQuery struct {
	IDBrowser int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

// IsPaged reports whether the legacy endpoint should return data and meta.
func (query BrowserVersionQuery) IsPaged() bool {
	return query.Paged
}

// LocationPage is the paginated legacy grid response.
type LocationPage struct {
	Data []LocationItem    `json:"data"`
	Meta LocationPageMeta `json:"meta"`
}

// BrandPage is the paginated legacy brand grid response.
type BrandPage struct {
	Data []BrandItem    `json:"data"`
	Meta BrandPageMeta `json:"meta"`
}

// ModelPage is the paginated legacy model grid response.
type ModelPage struct {
	Data []ModelItem    `json:"data"`
	Meta ModelPageMeta `json:"meta"`
}

// OperatingSystemPage is the paginated legacy operating-system grid response.
type OperatingSystemPage struct {
	Data []OperatingSystemItem    `json:"data"`
	Meta OperatingSystemPageMeta `json:"meta"`
}

// OperatingSystemVersionPage is the paginated legacy OS-version grid response.
type OperatingSystemVersionPage struct {
	Data []OperatingSystemVersionItem    `json:"data"`
	Meta OperatingSystemVersionPageMeta `json:"meta"`
}

// BrowserPage is the paginated legacy browser grid response.
type BrowserPage struct {
	Data []BrowserItem    `json:"data"`
	Meta BrowserPageMeta `json:"meta"`
}

// BrowserVersionPage is the paginated legacy browser-version grid response.
type BrowserVersionPage struct {
	Data []BrowserVersionItem    `json:"data"`
	Meta BrowserVersionPageMeta `json:"meta"`
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

// BrandPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type BrandPageMeta struct {
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

// ModelPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type ModelPageMeta struct {
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

// OperatingSystemPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type OperatingSystemPageMeta struct {
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

// OperatingSystemVersionPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type OperatingSystemVersionPageMeta struct {
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

// BrowserPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type BrowserPageMeta struct {
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

// BrowserVersionPageMeta contains Laravel EnterpriseGridResponse-compatible metadata.
type BrowserVersionPageMeta struct {
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
	ListBrands(ctx context.Context, query BrandQuery) (BrandPage, error)
	ListModels(ctx context.Context, query ModelQuery) (ModelPage, error)
	ListOperatingSystems(ctx context.Context, query OperatingSystemQuery) (OperatingSystemPage, error)
	ListOperatingSystemVersions(ctx context.Context, query OperatingSystemVersionQuery) (OperatingSystemVersionPage, error)
	ListBrowsers(ctx context.Context, query BrowserQuery) (BrowserPage, error)
	ListBrowserVersions(ctx context.Context, query BrowserVersionQuery) (BrowserVersionPage, error)
}
