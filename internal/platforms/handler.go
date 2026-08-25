package platforms

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes read-only platform catalog endpoints.
type Handler struct {
	repository CatalogRepository
	logger     *slog.Logger
}

// NewHandler creates a platform catalog handler.
func NewHandler(repository CatalogRepository, logger *slog.Logger) *Handler {
	return &Handler{repository: repository, logger: logger}
}

// Types returns the legacy platform type list contract.
func (handler *Handler) Types(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.repository.ListTypes(request.Context())
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform types failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, items)
}

// Statuses returns the legacy platform status list contract.
func (handler *Handler) Statuses(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.repository.ListStatuses(request.Context())
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform statuses failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, items)
}

// Locations returns the legacy platform location grid contract.
func (handler *Handler) Locations(writer http.ResponseWriter, request *http.Request) {
	query := parseLocationQuery(request)
	page, err := handler.repository.ListLocations(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform locations failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	if !query.IsPaged() {
		httpx.WriteJSON(writer, http.StatusOK, page.Data)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, page)
}

// Brands returns the legacy platform brand grid contract.
func (handler *Handler) Brands(writer http.ResponseWriter, request *http.Request) {
	query := parseBrandQuery(request)
	page, err := handler.repository.ListBrands(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform brands failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"PLATFORM_CATALOG_UNAVAILABLE",
			"The platform catalog could not be loaded.",
		)
		return
	}

	if !query.IsPaged() {
		httpx.WriteJSON(writer, http.StatusOK, page.Data)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, page)
}

func parseLocationQuery(request *http.Request) LocationQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "name" && sort != "created_at" && sort != "updated_at" {
		sort = "id"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := LocationQuery{
		Paged:     hasPage || hasPageSize,
		Search:    boundedString(values.Get("search"), 200),
		Sort:      sort,
		Direction: direction,
		FilterIDs: parseIDFilter(values.Get("filter[id]")),
	}
	if hasPage {
		query.Page = boundedInt(values.Get("page"), 1, 1, 1<<31-1)
	}
	if hasPageSize {
		query.PageSize = boundedInt(values.Get("pageSize"), 1, 1, 100)
	}
	return query
}

func parseBrandQuery(request *http.Request) BrandQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "brand" && sort != "created_at" && sort != "updated_at" {
		sort = "id"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := BrandQuery{
		Paged:     hasPage || hasPageSize,
		Search:    boundedString(values.Get("search"), 200),
		Sort:      sort,
		Direction: direction,
		FilterIDs: parseIDFilter(values.Get("filter[id]")),
	}
	if hasPage {
		query.Page = boundedInt(values.Get("page"), 1, 1, 1<<31-1)
	}
	if hasPageSize {
		query.PageSize = boundedInt(values.Get("pageSize"), 1, 1, 100)
	}
	return query
}

func boundedInt(value string, fallback int, minimum int, maximum int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum {
		return fallback
	}
	if parsed > maximum {
		return maximum
	}
	return parsed
}

func boundedString(value string, maximum int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
}

func parseIDFilter(value string) []int64 {
	if value == "" {
		return nil
	}
	ids := make([]int64, 0)
	for _, part := range strings.Split(value, ",") {
		parsed, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && parsed > 0 {
			ids = append(ids, parsed)
		}
	}
	return ids
}
