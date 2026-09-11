package platforms

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

// Handler exposes read-only platform catalog endpoints.
type Handler struct {
	repository CatalogRepository
	logger     *slog.Logger
	sessions   browserauth.SessionRepository
}

// NewHandler creates a platform catalog handler.
func NewHandler(repository CatalogRepository, logger *slog.Logger, sessions ...browserauth.SessionRepository) *Handler {
	var sessionRepository browserauth.SessionRepository
	if len(sessions) > 0 {
		sessionRepository = sessions[0]
	}
	return &Handler{repository: repository, logger: logger, sessions: sessionRepository}
}

func (handler *Handler) mutate(writer http.ResponseWriter, request *http.Request, kind string, id *int64) {
	if handler.sessions == nil {
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "PLATFORM_CATALOG_UNAVAILABLE", "The platform catalog is not available.")
		return
	}
	user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now())
	if !ok {
		httpx.WriteError(writer, request, http.StatusUnauthorized, "UNAUTHENTICATED", "An active browser session is required.")
		return
	}
	if user.Role != 1 {
		httpx.WriteError(writer, request, http.StatusForbidden, "FORBIDDEN", "Administrator access is required.")
		return
	}
	repository, ok := handler.repository.(CatalogMutationRepository)
	if !ok {
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "PLATFORM_CATALOG_UNAVAILABLE", "The platform catalog is not available.")
		return
	}
	values := map[string]any{}
	decoder := json.NewDecoder(io.LimitReader(request.Body, 64<<10))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "INVALID_PLATFORM_PAYLOAD", "The platform catalog payload is invalid.")
		return
	}
	var err error
	if id == nil && request.Method == http.MethodPut {
		if raw, present := values["id"]; present {
			number, valid := raw.(json.Number)
			if !valid {
				httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "INVALID_PLATFORM_ID", "The platform identifier must be a positive integer.")
				return
			}
			parsed, parseErr := number.Int64()
			if parseErr != nil || parsed <= 0 {
				httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "INVALID_PLATFORM_ID", "The platform identifier must be a positive integer.")
				return
			}
			id = &parsed
		}
		if id == nil {
			httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "INVALID_PLATFORM_ID", "The platform identifier is required for updates.")
			return
		}
	}
	if id == nil {
		err = repository.CreateCatalog(request.Context(), kind, values)
	} else {
		err = repository.UpdateCatalog(request.Context(), kind, *id, values)
	}
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "mutate platform catalog failed", "kind", kind, "error", err)
		httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "PLATFORM_CATALOG_MUTATION_FAILED", "The platform catalog mutation was rejected.")
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, map[string]any{"status": "ok"})
}

func (handler *Handler) DeleteManagedPlatform(writer http.ResponseWriter, request *http.Request) {
	if handler.sessions == nil {
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "PLATFORM_CATALOG_UNAVAILABLE", "The platform catalog is not available.")
		return
	}
	user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now())
	if !ok {
		httpx.WriteError(writer, request, http.StatusUnauthorized, "UNAUTHENTICATED", "An active browser session is required.")
		return
	}
	if user.Role != 1 {
		httpx.WriteError(writer, request, http.StatusForbidden, "FORBIDDEN", "Administrator access is required.")
		return
	}
	repository, ok := handler.repository.(CatalogMutationRepository)
	if !ok {
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "PLATFORM_CATALOG_UNAVAILABLE", "The platform catalog is not available.")
		return
	}
	id, err := parsePositivePathID(chi.URLParam(request, "id"))
	if err != nil {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_PLATFORM_ID", "The platform identifier must be a positive integer.")
		return
	}
	if err := repository.DeleteManagedPlatform(request.Context(), id); err != nil {
		handler.logger.ErrorContext(request.Context(), "delete managed platform failed", "error", err)
		httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "PLATFORM_CATALOG_MUTATION_FAILED", "The platform catalog mutation was rejected.")
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, map[string]any{"status": "ok"})
}

func (handler *Handler) CreateBrand(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "brand", nil)
}
func (handler *Handler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "brand", nil)
}
func (handler *Handler) CreateBrowser(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "browser", nil)
}
func (handler *Handler) UpdateBrowser(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "browser", nil)
}
func (handler *Handler) CreateBrowserVersion(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "browser-version", nil)
}
func (handler *Handler) UpdateBrowserVersion(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "browser-version", nil)
}
func (handler *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "location", nil)
}
func (handler *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "location", nil)
}
func (handler *Handler) CreateManagedPlatform(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "managed-platform", nil)
}
func (handler *Handler) UpdateManagedPlatform(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "managed-platform", nil)
}
func (handler *Handler) CreateModel(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "model", nil)
}
func (handler *Handler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "model", nil)
}
func (handler *Handler) CreateOperatingSystem(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "os", nil)
}
func (handler *Handler) UpdateOperatingSystem(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "os", nil)
}
func (handler *Handler) CreateOperatingSystemVersion(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "os-version", nil)
}
func (handler *Handler) UpdateOperatingSystemVersion(w http.ResponseWriter, r *http.Request) {
	handler.mutate(w, r, "os-version", nil)
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

// Models returns the legacy platform model grid contract scoped to a brand.
func (handler *Handler) Models(writer http.ResponseWriter, request *http.Request) {
	idBrand, err := parsePositivePathID(chi.URLParam(request, "idBrand"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_PLATFORM_BRAND",
			"The platform brand identifier must be a positive integer.",
		)
		return
	}

	query := parseModelQuery(request)
	query.IDBrand = idBrand
	page, err := handler.repository.ListModels(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform models failed", "error", err)
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

// OperatingSystems returns the legacy operating-system grid contract scoped to a platform type.
func (handler *Handler) OperatingSystems(writer http.ResponseWriter, request *http.Request) {
	typeID, err := parsePositivePathID(chi.URLParam(request, "idType"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_PLATFORM_TYPE",
			"The platform type identifier must be a positive integer.",
		)
		return
	}

	query := parseOperatingSystemQuery(request)
	query.TypeID = typeID
	page, err := handler.repository.ListOperatingSystems(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform operating systems failed", "error", err)
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

// OperatingSystemVersions returns the legacy OS-version grid contract scoped to an operating system.
func (handler *Handler) OperatingSystemVersions(writer http.ResponseWriter, request *http.Request) {
	idOs, err := parsePositivePathID(chi.URLParam(request, "idOs"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_OPERATING_SYSTEM",
			"The operating-system identifier must be a positive integer.",
		)
		return
	}

	query := parseOperatingSystemVersionQuery(request)
	query.IDOs = idOs
	page, err := handler.repository.ListOperatingSystemVersions(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform operating-system versions failed", "error", err)
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

// Browsers returns the legacy browser grid contract scoped to an operating system.
func (handler *Handler) Browsers(writer http.ResponseWriter, request *http.Request) {
	idOs, err := parsePositivePathID(chi.URLParam(request, "idOs"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_OPERATING_SYSTEM",
			"The operating-system identifier must be a positive integer.",
		)
		return
	}

	query := parseBrowserQuery(request)
	query.IDOs = idOs
	page, err := handler.repository.ListBrowsers(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform browsers failed", "error", err)
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

// BrowserVersions returns the legacy browser-version grid contract scoped to a browser.
func (handler *Handler) BrowserVersions(writer http.ResponseWriter, request *http.Request) {
	idBrowser, err := parsePositivePathID(chi.URLParam(request, "idBrowser"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_BROWSER",
			"The browser identifier must be a positive integer.",
		)
		return
	}

	query := parseBrowserVersionQuery(request)
	query.IDBrowser = idBrowser
	page, err := handler.repository.ListBrowserVersions(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list platform browser versions failed", "error", err)
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

// ManagedPlatforms returns the legacy managed-platform grid contract scoped to a platform type.
func (handler *Handler) ManagedPlatforms(writer http.ResponseWriter, request *http.Request) {
	typeID, err := parsePositivePathID(chi.URLParam(request, "type"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_PLATFORM_TYPE",
			"The platform type identifier must be a positive integer.",
		)
		return
	}

	query := parseManagedPlatformQuery(request)
	query.TypeID = typeID
	page, err := handler.repository.ListManagedPlatforms(request.Context(), query)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list managed platforms failed", "error", err)
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

// LaunchTargets returns safe launcher target candidates for the Web launcher setup.
func (handler *Handler) LaunchTargets(writer http.ResponseWriter, request *http.Request) {
	projectID, err := parsePositivePathID(chi.URLParam(request, "idProject"))
	if err != nil {
		httpx.WriteError(
			writer,
			request,
			http.StatusBadRequest,
			"INVALID_PROJECT",
			"The project identifier must be a positive integer.",
		)
		return
	}

	targets, err := handler.repository.ListLaunchTargets(request.Context(), projectID)
	if err != nil {
		handler.logger.ErrorContext(request.Context(), "list launcher targets failed", "error", err)
		httpx.WriteError(
			writer,
			request,
			http.StatusInternalServerError,
			"LAUNCH_TARGETS_UNAVAILABLE",
			"The launcher targets could not be loaded.",
		)
		return
	}

	httpx.WriteJSON(writer, http.StatusOK, targets)
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

func parseOperatingSystemQuery(request *http.Request) OperatingSystemQuery {
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
	query := OperatingSystemQuery{
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

func parseOperatingSystemVersionQuery(request *http.Request) OperatingSystemVersionQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "id" && sort != "version" && sort != "created_at" && sort != "updated_at" {
		sort = "version"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := OperatingSystemVersionQuery{
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

func parseBrowserQuery(request *http.Request) BrowserQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "id" && sort != "name" && sort != "created_at" && sort != "updated_at" {
		sort = "name"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := BrowserQuery{
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

func parseBrowserVersionQuery(request *http.Request) BrowserVersionQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "id" && sort != "version" && sort != "created_at" && sort != "updated_at" {
		sort = "version"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := BrowserVersionQuery{
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

func parseManagedPlatformQuery(request *http.Request) ManagedPlatformQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	switch sort {
	case "hostname", "brandDescription", "osDescription", "browserDescription", "status", "created_at", "updated_at":
	default:
		sort = "osDescription"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := ManagedPlatformQuery{
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

func parseModelQuery(request *http.Request) ModelQuery {
	values := request.URL.Query()
	sort := values.Get("sort")
	if sort != "model" && sort != "created_at" && sort != "updated_at" {
		sort = "model"
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	_, hasPage := values["page"]
	_, hasPageSize := values["pageSize"]
	query := ModelQuery{
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

func parsePositivePathID(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, strconv.ErrSyntax
	}
	return parsed, nil
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
