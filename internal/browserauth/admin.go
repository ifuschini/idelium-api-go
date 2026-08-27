package browserauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrForbidden = errors.New("browser admin action is forbidden")

type Role struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type Profile struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	CompanyName string `json:"companyName"`
	RoleName    string `json:"roleName"`
}

type Account struct {
	ID         int64  `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       int64  `json:"role"`
	IDCostumer int64  `json:"idCostumer"`
	Costumer   string `json:"costumer"`
	RoleName   string `json:"roleName"`
}

type AccountQuery struct {
	ActorTenantID int64
	ActorRole     int64
	Page          int
	PageSize      int
	Paged         bool
	Search        string
	Sort          string
	Direction     string
}

type AccountPage struct {
	Data []Account `json:"data"`
	Meta PageMeta  `json:"meta"`
}

type PageMeta struct {
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

type AccountCreate struct {
	Name       string
	Email      string
	Password   string
	Role       int64
	IDCostumer int64
}

type AccountUpdate struct {
	ID       int64
	Name     string
	Password string
}

type AdminRepository interface {
	ListRoles(request *http.Request, actor User) ([]Role, bool, error)
	Profile(request *http.Request, actor User) (Profile, error)
	UpdateProfilePassword(request *http.Request, actor User, password string) (Profile, error)
	ListAccounts(request *http.Request, actor User, query AccountQuery) (AccountPage, error)
	CreateAccount(request *http.Request, actor User, account AccountCreate) error
	UpdateAccount(request *http.Request, actor User, account AccountUpdate) error
	DeleteAccount(request *http.Request, actor User, accountID int64) error
}

func (h *Handler) Roles(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	roles, legacyOK, err := h.sessions.ListRoles(request, user)
	if err != nil {
		h.internalError(writer, request, "list browser roles", err)
		return
	}
	if legacyOK {
		writeJSON(writer, http.StatusOK, "ok")
		return
	}
	writeJSON(writer, http.StatusOK, roles)
}

func (h *Handler) Profile(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	profile, err := h.sessions.Profile(request, user)
	if err != nil {
		h.internalError(writer, request, "load browser profile", err)
		return
	}
	writeJSON(writer, http.StatusOK, profile)
}

func (h *Handler) UpdateProfile(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || input.Password == "" {
		validationError(writer, "password", "The password field is required.")
		return
	}
	if violations := passwordViolations(input.Password); len(violations) > 0 {
		validationErrors(writer, map[string][]string{"password": violations})
		return
	}
	profile, err := h.sessions.UpdateProfilePassword(request, user, input.Password)
	if err != nil {
		h.internalError(writer, request, "update browser profile password", err)
		return
	}
	writeJSON(writer, http.StatusOK, profile)
}

func (h *Handler) Accounts(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "accounts.manage")
	if !ok {
		return
	}
	query := parseAccountQuery(request, user)
	page, err := h.sessions.ListAccounts(request, user, query)
	if err != nil {
		h.internalError(writer, request, "list browser accounts", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) CreateAccount(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "accounts.manage")
	if !ok {
		return
	}
	var input struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		Role       int64  `json:"role"`
		IDCostumer int64  `json:"idCostumer"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		validationError(writer, "payload", "The request payload is invalid.")
		return
	}
	errorsByField := accountValidation(input.Name, input.Email, input.Password, input.Role)
	if user.Role == 1 && input.IDCostumer <= 0 {
		errorsByField["idCostumer"] = []string{"The id costumer field is required."}
	}
	if len(errorsByField) > 0 {
		validationErrors(writer, errorsByField)
		return
	}
	tenantID := input.IDCostumer
	if user.Role != 1 {
		tenantID = user.activeTenant()
	}
	err := h.sessions.CreateAccount(request, user, AccountCreate{Name: strings.TrimSpace(input.Name), Email: strings.TrimSpace(input.Email), Password: input.Password, Role: input.Role, IDCostumer: tenantID})
	if errors.Is(err, ErrForbidden) {
		h.forbidden(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser account", err)
		return
	}
	h.Accounts(writer, request)
}

func (h *Handler) UpdateAccount(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "accounts.manage")
	if !ok {
		return
	}
	accountID, err := parsePathID(request.PathValue("idUser"))
	if err != nil {
		h.notFound(writer)
		return
	}
	var input struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		validationError(writer, "payload", "The request payload is invalid.")
		return
	}
	errorsByField := map[string][]string{}
	if strings.TrimSpace(input.Name) == "" {
		errorsByField["name"] = []string{"The name field is required."}
	}
	if input.Password == "" {
		errorsByField["password"] = []string{"The password field is required."}
	} else if violations := passwordViolations(input.Password); len(violations) > 0 {
		errorsByField["password"] = violations
	}
	if len(errorsByField) > 0 {
		validationErrors(writer, errorsByField)
		return
	}
	err = h.sessions.UpdateAccount(request, user, AccountUpdate{ID: accountID, Name: strings.TrimSpace(input.Name), Password: input.Password})
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "update browser account", err)
		return
	}
	h.Accounts(writer, request)
}

func (h *Handler) DeleteAccount(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "accounts.manage")
	if !ok {
		return
	}
	accountID, err := parsePathID(request.PathValue("idUser"))
	if err != nil {
		h.notFound(writer)
		return
	}
	err = h.sessions.DeleteAccount(request, user, accountID)
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "delete browser account", err)
		return
	}
	h.Accounts(writer, request)
}

func (h *Handler) requireCapability(writer http.ResponseWriter, request *http.Request, capability string) (User, bool) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return User{}, false
	}
	for _, allowed := range capabilitiesForRole(user.Role) {
		if allowed == capability {
			return user, true
		}
	}
	h.forbidden(writer)
	return User{}, false
}

func parseAccountQuery(request *http.Request, user User) AccountQuery {
	page, pageSet := positiveQueryInt(request, "page", 1)
	pageSize, pageSizeSet := positiveQueryInt(request, "pageSize", 25)
	if pageSize > 100 {
		pageSize = 100
	}
	sort := request.URL.Query().Get("sort")
	if sort == "" {
		sort = "email"
	}
	direction := strings.ToLower(request.URL.Query().Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}
	return AccountQuery{
		ActorTenantID: user.activeTenant(),
		ActorRole:     user.Role,
		Page:          page,
		PageSize:      pageSize,
		Paged:         pageSet || pageSizeSet,
		Search:        strings.TrimSpace(request.URL.Query().Get("search")),
		Sort:          sort,
		Direction:     direction,
	}
}

func accountValidation(name, email, password string, role int64) map[string][]string {
	errorsByField := map[string][]string{}
	if strings.TrimSpace(name) == "" {
		errorsByField["name"] = []string{"The name field is required."}
	}
	if !strings.Contains(email, "@") {
		errorsByField["email"] = []string{"The email must be a valid email address."}
	}
	if role <= 0 {
		errorsByField["role"] = []string{"The role field is required."}
	}
	if password == "" {
		errorsByField["password"] = []string{"The password field is required."}
	} else if violations := passwordViolations(password); len(violations) > 0 {
		errorsByField["password"] = violations
	}
	return errorsByField
}

func passwordViolations(password string) []string {
	violations := []string{}
	if len([]rune(password)) < 12 {
		violations = append(violations, "The password must be at least 12 characters.")
	}
	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		violations = append(violations, "The password must contain both uppercase and lowercase letters.")
	}
	if !strings.ContainsAny(password, "0123456789") {
		violations = append(violations, "The password must contain at least one number.")
	}
	if !containsSymbol(password) {
		violations = append(violations, "The password must contain at least one symbol.")
	}
	switch strings.ToLower(strings.TrimSpace(password)) {
	case "admin", "password", "password1", "password123", "qwerty", "qwerty123", "letmein", "welcome", "welcome1", "changeme", "idelium", "idelium123":
		violations = append(violations, "The password is too common.")
	}
	return violations
}

func containsSymbol(value string) bool {
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return true
		}
	}
	return false
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// HashPasswordForRepository hashes browser account passwords with bcrypt.
func HashPasswordForRepository(password string) (string, error) {
	return hashPassword(password)
}

func positiveQueryInt(request *http.Request, key string, fallback int) (int, bool) {
	value := request.URL.Query().Get(key)
	if value == "" {
		return fallback, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback, true
	}
	return parsed, true
}

func parsePathID(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid path id")
	}
	return parsed, nil
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func validationError(writer http.ResponseWriter, field, message string) {
	validationErrors(writer, map[string][]string{field: []string{message}})
}

func validationErrors(writer http.ResponseWriter, errorsByField map[string][]string) {
	writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "The given data was invalid.", "errors": errorsByField})
}
