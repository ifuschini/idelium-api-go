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
var ErrConflict = errors.New("browser admin resource is not ready")
var ErrGone = errors.New("browser admin resource is gone")

type ValidationFailure struct {
	Errors map[string][]string
}

func (failure ValidationFailure) Error() string {
	return "browser admin validation failed"
}

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
	ListAdminCustomers(request *http.Request, query CustomerQuery) (CustomerPage, error)
	CreateCustomer(request *http.Request, customer CustomerCreate) error
	UpdateCustomer(request *http.Request, customer CustomerUpdate) error
	DeleteCustomer(request *http.Request, customerID int64) error
	ListTestCycles(request *http.Request, actor User, query ResourceQuery) (TestCyclePage, error)
	CreateTestCycle(request *http.Request, actor User, input TestCycleCreate) error
	GetTestCycle(request *http.Request, actor User, projectID int64, cycleID int64) (TestCycleDetail, error)
	UpdateTestCycle(request *http.Request, actor User, input TestCycleUpdate) error
	ReorderSteps(request *http.Request, actor User, input StepReorder) error
	ListStepsForReorder(request *http.Request, actor User, query ResourceQuery) (StepPage, error)
	ListTests(request *http.Request, actor User, query ResourceQuery) (TestPage, error)
	CreateTest(request *http.Request, actor User, input TestCreate) error
	GetTest(request *http.Request, actor User, projectID int64, testID int64) (TestDetail, error)
	UpdateTest(request *http.Request, actor User, input TestUpdate) error
	ImportTest(request *http.Request, actor User, input TestImport) error
	ListPerformedCycles(request *http.Request, actor User, query ResultQuery) (PerformedCyclePage, error)
	ListPerformedTests(request *http.Request, actor User, query ResultQuery) (PerformedTestPage, error)
	ListPerformedSteps(request *http.Request, actor User, performedTestID int64) ([]PerformedStep, error)
	CreateResultExport(request *http.Request, actor User, input ResultExportCreate) (ResultExportDescriptor, error)
	GetResultExport(request *http.Request, actor User, exportID int64) (ResultExportDescriptor, error)
	DownloadResultExport(request *http.Request, actor User, exportID int64, now time.Time) (ResultExportDownload, error)
	ListArtifactDescriptors(request *http.Request, actor User, projectID int64, performedTestCycleID int64) ([]ArtifactDescriptor, error)
	GetArtifactDescriptor(request *http.Request, actor User, projectID int64, performedTestCycleID int64, artifactDescriptorID int64) (ArtifactDescriptor, error)
	RegisterArtifactDescriptor(request *http.Request, actor User, input ArtifactDescriptorCreate) (ArtifactDescriptor, error)
	SetArtifactLegalHold(request *http.Request, actor User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error)
	MarkArtifactDeleted(request *http.Request, actor User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error)
	ArchiveArtifact(request *http.Request, actor User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error)
	RestoreArtifact(request *http.Request, actor User, input ArtifactLifecycleUpdate) (ArtifactDescriptor, error)
}

type CustomerQuery struct {
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
}

type CustomerPage struct {
	Data []Customer `json:"data"`
	Meta PageMeta   `json:"meta"`
}

type CustomerCreate struct {
	Costumer    string
	Description string
	Now         time.Time
}

type CustomerUpdate struct {
	ID          int64
	Costumer    string
	Description string
}

type ResourceQuery struct {
	ProjectID int64
	Page      int
	PageSize  int
	Paged     bool
	Search    string
	Sort      string
	Direction string
	FilterIDs []int64
}

type TestCycle struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type TestCyclePage struct {
	Data []TestCycle `json:"data"`
	Meta PageMeta    `json:"meta"`
}

type TestCycleDetail struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"`
	IDProject   int64  `json:"idProject"`
}

type TestCycleCreate struct {
	Name        string
	Description string
	Config      string
	IDProject   int64
}

type TestCycleUpdate struct {
	ID          int64
	IDProject   int64
	Description string
	Config      string
}

type Test struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type TestPage struct {
	Data []Test   `json:"data"`
	Meta PageMeta `json:"meta"`
}

type TestDetail struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"`
	IDProject   int64  `json:"idProject"`
}

type TestCreate struct {
	Name        string
	Description string
	Config      string
	IDProject   int64
}

type TestUpdate struct {
	ID        int64
	IDProject int64
	Config    string
}

type TestImport struct {
	Name        string
	Description string
	Import      string
	IDProject   int64
}

type ResultQuery struct {
	ParentID  int64
	Page      int
	PerPage   int
	Paged     bool
	Status    *int
	Sort      string
	Direction string
}

type ResultPaginationMeta struct {
	Page      int    `json:"page"`
	PerPage   int    `json:"perPage"`
	Total     int64  `json:"total"`
	LastPage  int    `json:"lastPage"`
	Sort      string `json:"sort"`
	Direction string `json:"direction"`
}

type ResultMeta struct {
	Pagination ResultPaginationMeta `json:"pagination"`
}

type PerformedCycle struct {
	ID          int64      `json:"id"`
	TestCycleID int64      `json:"testCycleId"`
	Date        *time.Time `json:"date,omitempty"`
	Status      int64      `json:"status"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

type PerformedCyclePage struct {
	Data []PerformedCycle `json:"data"`
	Meta ResultMeta       `json:"meta"`
}

type PerformedTest struct {
	ID              int64      `json:"id"`
	TestCycleDoneID int64      `json:"testCycleDoneId"`
	TestID          int64      `json:"testId"`
	Status          int64      `json:"status"`
	PostmanData     *string    `json:"postmanData"`
	Name            string     `json:"name"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

type PerformedTestPage struct {
	Data []PerformedTest `json:"data"`
	Meta ResultMeta      `json:"meta"`
}

type PerformedStep struct {
	ID              int64      `json:"id"`
	TestCycleDoneID int64      `json:"testCycleDoneId"`
	TestDoneID      int64      `json:"testDoneId"`
	Name            string     `json:"name"`
	Status          int64      `json:"status"`
	Screenshots     string     `json:"screenshots"`
	Data            string     `json:"data"`
	Type            string     `json:"type"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

type ResultExportCreate struct {
	PerformedTestCycleID int64
	Format               string
	Now                  time.Time
}

type ResultExportDescriptor struct {
	ID           int64      `json:"id"`
	Format       string     `json:"format"`
	Status       string     `json:"status"`
	Filename     string     `json:"filename"`
	ContentType  string     `json:"contentType"`
	URL          string     `json:"url"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	Authorized   bool       `json:"authorized"`
	Ready        bool       `json:"ready"`
	ErrorMessage *string    `json:"errorMessage"`
}

type ResultExportDownload struct {
	Filename    string
	ContentType string
	Payload     string
}

type ArtifactDescriptor struct {
	ID                   int64           `json:"id"`
	IDCostumer           int64           `json:"idCostumer"`
	IDProject            int64           `json:"idProject"`
	PerformedTestCycleID int64           `json:"performedTestCycleId"`
	PerformedTestID      *int64          `json:"performedTestId"`
	PerformedStepID      *int64          `json:"performedStepId"`
	ArtifactType         string          `json:"artifactType"`
	Name                 string          `json:"name"`
	ContentType          string          `json:"contentType"`
	SizeBytes            uint64          `json:"sizeBytes"`
	ChecksumSHA256       string          `json:"checksumSha256"`
	StorageKey           string          `json:"storageKey"`
	State                string          `json:"state"`
	RetentionUntil       *time.Time      `json:"retentionUntil"`
	Metadata             json.RawMessage `json:"metadata"`
	CreatedAt            *time.Time      `json:"created_at,omitempty"`
	UpdatedAt            *time.Time      `json:"updated_at,omitempty"`
}

type ArtifactDescriptorCreate struct {
	IDProject            int64
	PerformedTestCycleID int64
	PerformedTestID      *int64
	PerformedStepID      *int64
	ArtifactType         string
	Name                 string
	ContentType          string
	SizeBytes            uint64
	ChecksumSHA256       string
	StorageKey           string
	State                string
	RetentionUntil       *time.Time
	Metadata             json.RawMessage
	Now                  time.Time
}

type ArtifactLifecycleUpdate struct {
	ProjectID            int64
	PerformedTestCycleID int64
	ArtifactDescriptorID int64
	Enabled              bool
	Reason               *string
	RestoreBy            *string
	Now                  time.Time
}

type StepReorder struct {
	IDProject int64
	Offset    int
	Order     []StepOrder
}

type StepOrder struct {
	ID int64 `json:"id"`
}

type StepListItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int64  `json:"order"`
}

type StepPage struct {
	Data []StepListItem `json:"data"`
	Meta PageMeta       `json:"meta"`
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

func (h *Handler) Customers(writer http.ResponseWriter, request *http.Request) {
	if _, ok := h.requireCapability(writer, request, "customers.manage"); !ok {
		return
	}
	query := parseCustomerQuery(request)
	page, err := h.sessions.ListAdminCustomers(request, query)
	if err != nil {
		h.internalError(writer, request, "list browser customers", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) CreateCustomer(writer http.ResponseWriter, request *http.Request) {
	if _, ok := h.requireCapability(writer, request, "customers.manage"); !ok {
		return
	}
	var input struct {
		Costumer    string `json:"costumer"`
		Description string `json:"description"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Costumer) == "" {
		validationError(writer, "costumer", "The costumer field is required.")
		return
	}
	if err := h.sessions.CreateCustomer(request, CustomerCreate{Costumer: strings.TrimSpace(input.Costumer), Description: strings.TrimSpace(input.Description), Now: h.now().UTC()}); err != nil {
		h.internalError(writer, request, "create browser customer", err)
		return
	}
	h.Customers(writer, request)
}

func (h *Handler) UpdateCustomer(writer http.ResponseWriter, request *http.Request) {
	if _, ok := h.requireCapability(writer, request, "customers.manage"); !ok {
		return
	}
	customerID, err := parsePathID(request.PathValue("idCostumer"))
	if err != nil {
		h.notFound(writer)
		return
	}
	var input struct {
		Costumer    string `json:"costumer"`
		Description string `json:"description"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Costumer) == "" || strings.TrimSpace(input.Description) == "" {
		validationErrors(writer, map[string][]string{"costumer": []string{"The costumer field is required."}, "description": []string{"The description field is required."}})
		return
	}
	err = h.sessions.UpdateCustomer(request, CustomerUpdate{ID: customerID, Costumer: strings.TrimSpace(input.Costumer), Description: strings.TrimSpace(input.Description)})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "update browser customer", err)
		return
	}
	h.Customers(writer, request)
}

func (h *Handler) DeleteCustomer(writer http.ResponseWriter, request *http.Request) {
	if _, ok := h.requireCapability(writer, request, "customers.manage"); !ok {
		return
	}
	customerID, err := parsePathID(request.PathValue("idCostumer"))
	if err != nil {
		h.notFound(writer)
		return
	}
	err = h.sessions.DeleteCustomer(request, customerID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "delete browser customer", err)
		return
	}
	h.Customers(writer, request)
}

func (h *Handler) TestCycles(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	query := parseResourceQuery(request, projectID, "id")
	page, err := h.sessions.ListTestCycles(request, user, query)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser test cycles", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) CreateTestCycle(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Config      string `json:"config"`
		IDProject   int64  `json:"idProject"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Name) == "" || input.IDProject <= 0 {
		validationErrors(writer, map[string][]string{"name": []string{"The name field is required."}, "idProject": []string{"The id project field is required."}})
		return
	}
	err := h.sessions.CreateTestCycle(request, user, TestCycleCreate{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), Config: input.Config, IDProject: input.IDProject})
	if errors.Is(err, ErrNotFound) {
		validationError(writer, "idProject", "The selected id project is invalid.")
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser test cycle", err)
		return
	}
	request.SetPathValue("idProject", strconv.FormatInt(input.IDProject, 10))
	h.TestCycles(writer, request)
}

func (h *Handler) ShowTestCycle(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, cycleID, ok := parseProjectResourcePath(writer, request, "testcycle")
	if !ok {
		return
	}
	cycle, err := h.sessions.GetTestCycle(request, user, projectID, cycleID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "load browser test cycle", err)
		return
	}
	writeJSON(writer, http.StatusOK, cycle)
}

func (h *Handler) UpdateTestCycle(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, cycleID, ok := parseProjectResourcePath(writer, request, "testcycle")
	if !ok {
		return
	}
	var input struct {
		Description string `json:"description"`
		Config      string `json:"config"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		validationError(writer, "payload", "The request payload is invalid.")
		return
	}
	err := h.sessions.UpdateTestCycle(request, user, TestCycleUpdate{ID: cycleID, IDProject: projectID, Description: input.Description, Config: input.Config})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "update browser test cycle", err)
		return
	}
	h.TestCycles(writer, request)
}

func (h *Handler) ReorderSteps(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	var input struct {
		Offset int         `json:"offset"`
		Order  []StepOrder `json:"order"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || len(input.Order) == 0 || input.Offset < 0 {
		validationError(writer, "order", "The order field is required.")
		return
	}
	err = h.sessions.ReorderSteps(request, user, StepReorder{IDProject: projectID, Offset: input.Offset, Order: input.Order})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "reorder browser steps", err)
		return
	}
	query := parseResourceQuery(request, projectID, "order")
	page, err := h.sessions.ListStepsForReorder(request, user, query)
	if err != nil {
		h.internalError(writer, request, "list reordered browser steps", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) Tests(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	query := parseResourceQuery(request, projectID, "id")
	page, err := h.sessions.ListTests(request, user, query)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser tests", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) CreateTest(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Config      string `json:"config"`
		IDProject   int64  `json:"idProject"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.Config) == "" || input.IDProject <= 0 {
		validationErrors(writer, map[string][]string{"name": {"The name field is required."}, "description": {"The description field is required."}, "config": {"The config field is required."}, "idProject": {"The id project field is required."}})
		return
	}
	err := h.sessions.CreateTest(request, user, TestCreate{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), Config: input.Config, IDProject: input.IDProject})
	if errors.Is(err, ErrNotFound) {
		validationError(writer, "idProject", "The selected id project is invalid.")
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser test", err)
		return
	}
	request.SetPathValue("idProject", strconv.FormatInt(input.IDProject, 10))
	h.Tests(writer, request)
}

func (h *Handler) ShowTest(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, testID, ok := parseProjectResourcePath(writer, request, "test")
	if !ok {
		return
	}
	test, err := h.sessions.GetTest(request, user, projectID, testID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "load browser test", err)
		return
	}
	writeJSON(writer, http.StatusOK, test)
}

func (h *Handler) UpdateTest(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	projectID, testID, ok := parseProjectResourcePath(writer, request, "test")
	if !ok {
		return
	}
	var input struct {
		Config string `json:"config"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Config) == "" {
		validationError(writer, "config", "The config field is required.")
		return
	}
	err := h.sessions.UpdateTest(request, user, TestUpdate{ID: testID, IDProject: projectID, Config: input.Config})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "update browser test", err)
		return
	}
	h.Tests(writer, request)
}

func (h *Handler) ImportTest(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Import      string `json:"import"`
		IDProject   int64  `json:"idProject"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.Import) == "" || input.IDProject <= 0 {
		validationErrors(writer, map[string][]string{"name": {"The name field is required."}, "description": {"The description field is required."}, "import": {"The import field is required."}, "idProject": {"The id project field is required."}})
		return
	}
	if err := validateImportedSteps(input.Import); err != nil {
		validationError(writer, "import", err.Error())
		return
	}
	err := h.sessions.ImportTest(request, user, TestImport{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), Import: input.Import, IDProject: input.IDProject})
	if errors.Is(err, ErrNotFound) {
		validationError(writer, "idProject", "The selected id project is invalid.")
		return
	}
	if err != nil {
		h.internalError(writer, request, "import browser test", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) PerformedCycles(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	parentID, err := parsePathID(request.PathValue("idTestCyclePerformed"))
	if err != nil {
		h.notFound(writer)
		return
	}
	query := parseResultQuery(request, parentID, "date", "desc")
	page, err := h.sessions.ListPerformedCycles(request, user, query)
	if err != nil {
		h.internalError(writer, request, "list browser performed cycles", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) PerformedTests(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	parentID, err := parsePathID(request.PathValue("idTestPerformed"))
	if err != nil {
		h.notFound(writer)
		return
	}
	query := parseResultQuery(request, parentID, "id", "asc")
	page, err := h.sessions.ListPerformedTests(request, user, query)
	if err != nil {
		h.internalError(writer, request, "list browser performed tests", err)
		return
	}
	if !query.Paged {
		writeJSON(writer, http.StatusOK, page.Data)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (h *Handler) PerformedSteps(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	performedTestID, err := parsePathID(request.PathValue("idTestPerformed"))
	if err != nil {
		h.notFound(writer)
		return
	}
	steps, err := h.sessions.ListPerformedSteps(request, user, performedTestID)
	if err != nil {
		h.internalError(writer, request, "list browser performed steps", err)
		return
	}
	writeJSON(writer, http.StatusOK, steps)
}

func (h *Handler) CreateResultExport(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var input struct {
		PerformedTestCycleID int64  `json:"performedTestCycleId"`
		Format               string `json:"format"`
	}
	if err := decodeJSON(writer, request, &input); err != nil || input.PerformedTestCycleID <= 0 || (input.Format != "json" && input.Format != "markdown") {
		validationErrors(writer, map[string][]string{"performedTestCycleId": {"The performed test cycle id field is required."}, "format": {"The selected format is invalid."}})
		return
	}
	descriptor, err := h.sessions.CreateResultExport(request, user, ResultExportCreate{PerformedTestCycleID: input.PerformedTestCycleID, Format: input.Format, Now: h.now()})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser result export", err)
		return
	}
	writeJSON(writer, http.StatusAccepted, descriptor)
}

func (h *Handler) ShowResultExport(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	exportID, err := parsePathID(request.PathValue("resultExport"))
	if err != nil {
		h.notFound(writer)
		return
	}
	descriptor, err := h.sessions.GetResultExport(request, user, exportID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "show browser result export", err)
		return
	}
	writeJSON(writer, http.StatusOK, descriptor)
}

func (h *Handler) DownloadResultExport(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	exportID, err := parsePathID(request.PathValue("resultExport"))
	if err != nil {
		h.notFound(writer)
		return
	}
	download, err := h.sessions.DownloadResultExport(request, user, exportID, h.now())
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "Conflict."})
		return
	}
	if errors.Is(err, ErrGone) {
		writeJSON(writer, http.StatusGone, map[string]string{"message": "Gone."})
		return
	}
	if err != nil {
		h.internalError(writer, request, "download browser result export", err)
		return
	}
	writer.Header().Set("Content-Type", download.ContentType)
	writer.Header().Set("Content-Disposition", `attachment; filename="`+download.Filename+`"`)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(download.Payload))
}

func (h *Handler) ArtifactDescriptors(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "artifacts.read")
	if !ok {
		return
	}
	projectID, performedTestCycleID, ok := parseProjectPerformedCyclePath(writer, request)
	if !ok {
		return
	}
	descriptors, err := h.sessions.ListArtifactDescriptors(request, user, projectID, performedTestCycleID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser artifact descriptors", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": descriptors})
}

func (h *Handler) ShowArtifactDescriptor(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "artifacts.read")
	if !ok {
		return
	}
	projectID, performedTestCycleID, ok := parseProjectPerformedCyclePath(writer, request)
	if !ok {
		return
	}
	artifactDescriptorID, err := parsePathID(request.PathValue("artifactDescriptor"))
	if err != nil {
		h.notFound(writer)
		return
	}
	descriptor, err := h.sessions.GetArtifactDescriptor(request, user, projectID, performedTestCycleID, artifactDescriptorID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "show browser artifact descriptor", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": descriptor})
}

func (h *Handler) SetArtifactLegalHold(writer http.ResponseWriter, request *http.Request) {
	user, input, ok := h.artifactLifecycleInput(writer, request)
	if !ok {
		return
	}
	var body struct {
		Enabled *bool   `json:"enabled"`
		Reason  *string `json:"reason"`
	}
	if err := decodeJSON(writer, request, &body); err != nil || body.Enabled == nil {
		validationError(writer, "enabled", "The enabled field is required.")
		return
	}
	input.Enabled = *body.Enabled
	if body.Reason != nil {
		reason := strings.TrimSpace(*body.Reason)
		input.Reason = &reason
	}
	descriptor, err := h.sessions.SetArtifactLegalHold(request, user, input)
	h.writeArtifactLifecycleResult(writer, request, "set browser artifact legal hold", descriptor, err)
}

func (h *Handler) MarkArtifactDeleted(writer http.ResponseWriter, request *http.Request) {
	user, input, ok := h.artifactLifecycleInput(writer, request)
	if !ok {
		return
	}
	descriptor, err := h.sessions.MarkArtifactDeleted(request, user, input)
	h.writeArtifactLifecycleResult(writer, request, "mark browser artifact deleted", descriptor, err)
}

func (h *Handler) ArchiveArtifact(writer http.ResponseWriter, request *http.Request) {
	user, input, ok := h.artifactLifecycleInput(writer, request)
	if !ok {
		return
	}
	var body struct {
		Reason    *string `json:"reason"`
		RestoreBy *string `json:"restoreBy"`
	}
	if request.Body != nil && request.ContentLength != 0 {
		if err := decodeJSON(writer, request, &body); err != nil {
			validationError(writer, "payload", "The request payload is invalid.")
			return
		}
	}
	if body.Reason != nil {
		reason := strings.TrimSpace(*body.Reason)
		input.Reason = &reason
	}
	if body.RestoreBy != nil {
		restoreBy := strings.TrimSpace(*body.RestoreBy)
		input.RestoreBy = &restoreBy
	}
	descriptor, err := h.sessions.ArchiveArtifact(request, user, input)
	h.writeArtifactLifecycleResult(writer, request, "archive browser artifact", descriptor, err)
}

func (h *Handler) RestoreArtifact(writer http.ResponseWriter, request *http.Request) {
	user, input, ok := h.artifactLifecycleInput(writer, request)
	if !ok {
		return
	}
	descriptor, err := h.sessions.RestoreArtifact(request, user, input)
	h.writeArtifactLifecycleResult(writer, request, "restore browser artifact", descriptor, err)
}

func (h *Handler) artifactLifecycleInput(writer http.ResponseWriter, request *http.Request) (User, ArtifactLifecycleUpdate, bool) {
	user, ok := h.requireCapability(writer, request, "artifacts.manage")
	if !ok {
		return User{}, ArtifactLifecycleUpdate{}, false
	}
	projectID, performedTestCycleID, ok := parseProjectPerformedCyclePath(writer, request)
	if !ok {
		return User{}, ArtifactLifecycleUpdate{}, false
	}
	artifactDescriptorID, err := parsePathID(request.PathValue("artifactDescriptor"))
	if err != nil {
		h.notFound(writer)
		return User{}, ArtifactLifecycleUpdate{}, false
	}
	return user, ArtifactLifecycleUpdate{ProjectID: projectID, PerformedTestCycleID: performedTestCycleID, ArtifactDescriptorID: artifactDescriptorID, Now: h.now().UTC()}, true
}

func (h *Handler) writeArtifactLifecycleResult(writer http.ResponseWriter, request *http.Request, action string, descriptor ArtifactDescriptor, err error) {
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	var validationFailure ValidationFailure
	if errors.As(err, &validationFailure) {
		validationErrors(writer, validationFailure.Errors)
		return
	}
	if err != nil {
		h.internalError(writer, request, action, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": descriptor})
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

func parseCustomerQuery(request *http.Request) CustomerQuery {
	page, pageSet := positiveQueryInt(request, "page", 1)
	pageSize, pageSizeSet := positiveQueryInt(request, "pageSize", 25)
	if pageSize > 100 {
		pageSize = 100
	}
	sort := request.URL.Query().Get("sort")
	if sort == "" {
		sort = "created_at"
	}
	direction := strings.ToLower(request.URL.Query().Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}
	return CustomerQuery{Page: page, PageSize: pageSize, Paged: pageSet || pageSizeSet, Search: strings.TrimSpace(request.URL.Query().Get("search")), Sort: sort, Direction: direction}
}

func parseResultQuery(request *http.Request, parentID int64, defaultSort string, defaultDirection string) ResultQuery {
	page, pageSet := positiveQueryInt(request, "page", 1)
	perPage, perPageSet := positiveQueryInt(request, "perPage", 25)
	if perPage > 100 {
		perPage = 100
	}
	sort := request.URL.Query().Get("sort")
	if sort == "" {
		sort = defaultSort
	}
	direction := strings.ToLower(request.URL.Query().Get("direction"))
	if direction != "asc" && direction != "desc" {
		direction = defaultDirection
	}
	query := ResultQuery{ParentID: parentID, Page: page, PerPage: perPage, Paged: pageSet || perPageSet, Sort: sort, Direction: direction}
	if rawStatus := strings.TrimSpace(request.URL.Query().Get("status")); rawStatus != "" {
		if status, err := strconv.Atoi(rawStatus); err == nil {
			query.Status = &status
		}
	}
	return query
}

func validateImportedSteps(rawImport string) error {
	var steps []map[string]any
	if err := json.Unmarshal([]byte(rawImport), &steps); err != nil || len(steps) == 0 {
		return errors.New("The import field must contain a non-empty JSON array of Idelium steps.")
	}
	for _, step := range steps {
		name, ok := step["name"].(string)
		if !ok || strings.TrimSpace(name) == "" {
			return errors.New("Every imported step must have a non-empty name.")
		}
		actions, ok := step["steps"].([]any)
		if !ok || len(actions) == 0 {
			return errors.New("Every imported Idelium step must include at least one executable action.")
		}
		for _, action := range actions {
			actionObject, ok := action.(map[string]any)
			if !ok {
				return errors.New("Every imported action must include a stepType.")
			}
			stepType, ok := actionObject["stepType"].(string)
			if !ok || strings.TrimSpace(stepType) == "" {
				return errors.New("Every imported action must include a stepType.")
			}
		}
		if isPostmanPayload(step) && !containsExecutablePostmanCollection(step) {
			return errors.New("Every imported Postman step must include a postman_collection action with a collection payload.")
		}
	}
	return nil
}

func containsExecutablePostmanCollection(step map[string]any) bool {
	actions, _ := step["steps"].([]any)
	for _, action := range actions {
		actionObject, ok := action.(map[string]any)
		if !ok || !isPostmanPayload(actionObject) {
			continue
		}
		if isPostmanCollectionPayload(actionObject["collection"]) {
			return true
		}
	}
	return false
}

func isPostmanPayload(payload map[string]any) bool {
	for _, field := range []string{"editorType", "runtime", "stepType", "type", "actionType"} {
		if value, ok := payload[field].(string); ok && strings.Contains(strings.ToLower(value), "postman") {
			return true
		}
	}
	return false
}

func isPostmanCollectionPayload(payload any) bool {
	object, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	if nested, ok := object["collection"].(map[string]any); ok {
		return isPostmanCollectionPayload(nested)
	}
	_, hasInfo := object["info"].(map[string]any)
	_, hasItems := object["item"].([]any)
	return hasInfo && hasItems
}

func parseResourceQuery(request *http.Request, projectID int64, defaultSort string) ResourceQuery {
	page, pageSet := positiveQueryInt(request, "page", 1)
	pageSize, pageSizeSet := positiveQueryInt(request, "pageSize", 25)
	if pageSize > 100 {
		pageSize = 100
	}
	sort := request.URL.Query().Get("sort")
	if sort == "" {
		sort = defaultSort
	}
	direction := strings.ToLower(request.URL.Query().Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}
	query := ResourceQuery{ProjectID: projectID, Page: page, PageSize: pageSize, Paged: pageSet || pageSizeSet, Search: strings.TrimSpace(request.URL.Query().Get("search")), Sort: sort, Direction: direction}
	if ids := request.URL.Query().Get("filter[id]"); ids != "" {
		for _, raw := range strings.Split(ids, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err == nil && id > 0 {
				query.FilterIDs = append(query.FilterIDs, id)
			}
		}
	}
	return query
}

func parseProjectResourcePath(writer http.ResponseWriter, request *http.Request, resourceName string) (int64, int64, bool) {
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not found."})
		return 0, 0, false
	}
	resourceID, err := parsePathID(request.PathValue(resourceName))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not found."})
		return 0, 0, false
	}
	return projectID, resourceID, true
}

func parseProjectPerformedCyclePath(writer http.ResponseWriter, request *http.Request) (int64, int64, bool) {
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not found."})
		return 0, 0, false
	}
	performedTestCycleID, err := parsePathID(request.PathValue("performedTestCycleId"))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not found."})
		return 0, 0, false
	}
	return projectID, performedTestCycleID, true
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
