package browserauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/idelium/idelium-api-go/internal/integrations"
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
	CreateGridQuerySnapshot(request *http.Request, actor User, input GridQuerySnapshotCreate) (GridQuerySnapshot, error)
	CreateGridBulkJob(request *http.Request, actor User, input GridBulkJobCreate) (GridBulkJob, error)
	GetGridBulkJob(request *http.Request, actor User, jobID string) (GridBulkJob, error)
	ExportGridBulkJob(request *http.Request, actor User, jobID string, now time.Time) (GridBulkExport, error)
	ListIntegrationEndpoints(request *http.Request, actor User, projectID int64) ([]IntegrationEndpoint, error)
	CreateIntegrationEndpoint(request *http.Request, actor User, input IntegrationEndpointCreate) (IntegrationEndpoint, error)
	CreateIntegrationTestDelivery(request *http.Request, actor User, projectID, endpointID int64, now time.Time) (IntegrationDelivery, error)
	UpdateIntegrationEndpointStatus(request *http.Request, actor User, projectID, endpointID int64, status string, now time.Time) (IntegrationEndpoint, error)
	RotateIntegrationEndpointSecret(request *http.Request, actor User, projectID, endpointID int64, secret string, now time.Time) (IntegrationEndpoint, error)
	ListIntegrationDeliveries(request *http.Request, actor User, projectID int64, status string) ([]IntegrationDelivery, error)
	ReplayIntegrationDelivery(request *http.Request, actor User, projectID, deliveryID int64, now time.Time) (IntegrationDelivery, error)
	ListAuditEvents(request *http.Request, actor User, filter AuditEventFilter) ([]AuditEventRecord, error)
	AssetImpact(request *http.Request, actor User, projectID int64, assetType string, assetID int64) (AssetImpact, error)
	ListAssetVersions(request *http.Request, actor User, projectID int64, assetType string, assetID int64) ([]AssetVersion, error)
	GetAssetVersion(request *http.Request, actor User, projectID, versionID int64) (AssetVersion, error)
	TransitionAssetVersionReview(request *http.Request, actor User, projectID, versionID int64, toStatus string, comment *string, now time.Time) (AssetReviewEvent, error)
	ListParallelRuns(request *http.Request, tenantID, projectID int64, filters map[string]string) ([]ParallelRun, error)
	CreateParallelRun(request *http.Request, input ParallelRunCreate) (ParallelRun, error)
	CreateParallelRunMatrix(request *http.Request, input ParallelRunCreate, combinations []map[string]string) ([]ParallelRun, error)
	GetParallelRun(request *http.Request, tenantID, projectID, runID int64) (ParallelRun, error)
	ClaimParallelRun(request *http.Request, input ParallelRunClaim) (ParallelRun, error)
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

type GridQuery struct {
	Search    string         `json:"-"`
	Sort      string         `json:"-"`
	Direction string         `json:"-"`
	Filters   map[string]any `json:"-"`
	Raw       map[string]any `json:"-"`
}

type GridQuerySnapshotCreate struct {
	ResourceType string
	Query        GridQuery
	Now          time.Time
}

type GridQuerySnapshot struct {
	ID           string    `json:"id"`
	ResourceType string    `json:"resourceType"`
	Total        int       `json:"total"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type GridBulkJobCreate struct {
	QuerySnapshotID string
	Action          string
	Tags            []string
	Payload         map[string]any
	Now             time.Time
}

type GridBulkJob struct {
	ID             string         `json:"id"`
	ResourceType   string         `json:"resourceType"`
	Action         string         `json:"action"`
	Status         string         `json:"status"`
	RequestedCount int            `json:"requestedCount"`
	ProcessedCount int            `json:"processedCount"`
	FailedCount    int            `json:"failedCount"`
	Result         map[string]any `json:"result"`
}

type GridBulkExport struct {
	Filename string
	Payload  string
}

type IntegrationEndpoint struct {
	ID               int64      `json:"id"`
	IDProject        int64      `json:"idProject"`
	Name             string     `json:"name"`
	Adapter          string     `json:"adapter"`
	URL              string     `json:"url"`
	Events           []string   `json:"events"`
	Status           string     `json:"status"`
	SecretConfigured bool       `json:"secretConfigured"`
	SchemaVersion    string     `json:"schemaVersion"`
	CreatedAt        *time.Time `json:"createdAt"`
}

type IntegrationEndpointCreate struct {
	ProjectID int64
	Name      string
	Adapter   string
	URL       string
	Secret    string
	Events    []string
	Now       time.Time
}

type IntegrationDelivery struct {
	ID             int64      `json:"id"`
	DeliveryID     string     `json:"deliveryId"`
	Event          string     `json:"event"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	ResponseStatus *int       `json:"responseStatus"`
	NextAttemptAt  *time.Time `json:"nextAttemptAt"`
	SentAt         *time.Time `json:"sentAt"`
}

type AuditEventFilter struct {
	Action        string
	TargetType    string
	TargetID      string
	CorrelationID string
	From          *time.Time
	To            *time.Time
	Limit         int
}

type AuditEventRecord struct {
	ID             int64          `json:"id"`
	ActorUserID    *int64         `json:"actorUserId"`
	ActorTenantID  *int64         `json:"actorTenantId"`
	ActiveTenantID int64          `json:"activeTenantId"`
	IDProject      *int64         `json:"idProject"`
	Action         string         `json:"action"`
	TargetType     string         `json:"targetType"`
	TargetID       *string        `json:"targetId"`
	BeforeValues   map[string]any `json:"beforeValues"`
	AfterValues    map[string]any `json:"afterValues"`
	Result         string         `json:"result"`
	SourceIP       *string        `json:"sourceIp"`
	CorrelationID  string         `json:"correlationId"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

type AssetImpactItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AssetImpact struct {
	Asset struct {
		AssetType string `json:"assetType"`
		AssetID   int64  `json:"assetId"`
	} `json:"asset"`
	Summary struct {
		Tests      int `json:"tests"`
		TestCycles int `json:"testCycles"`
	} `json:"summary"`
	Tests      []AssetImpactItem `json:"tests"`
	TestCycles []AssetImpactItem `json:"testCycles"`
}

type AssetReview struct {
	Status           string     `json:"status"`
	LastEventID      *int64     `json:"lastEventId"`
	LastComment      *string    `json:"lastComment"`
	ReviewedByUserID *int64     `json:"reviewedByUserId"`
	ReviewedAt       *time.Time `json:"reviewedAt"`
	AuthorUserID     *int64     `json:"authorUserId"`
}

type AssetVersion struct {
	ID          int64           `json:"id"`
	IDProject   int64           `json:"idProject"`
	AssetType   string          `json:"assetType"`
	AssetID     int64           `json:"assetId"`
	Version     int             `json:"version"`
	ActorUserID *int64          `json:"actorUserId"`
	Reason      string          `json:"reason"`
	CreatedAt   *time.Time      `json:"createdAt"`
	Review      AssetReview     `json:"review"`
	Snapshot    *map[string]any `json:"snapshot,omitempty"`
}

type AssetReviewEvent struct {
	ID             int64      `json:"id"`
	AssetVersionID int64      `json:"assetVersionId"`
	FromStatus     string     `json:"fromStatus"`
	ToStatus       string     `json:"toStatus"`
	Comment        *string    `json:"comment"`
	ActorUserID    *int64     `json:"actorUserId"`
	CreatedAt      *time.Time `json:"createdAt"`
}

type ReviewFailure struct {
	Message    string
	FromStatus string
	ToStatus   string
}

func (failure ReviewFailure) Error() string { return failure.Message }

var gridUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var gridTagPattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)
var auditUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

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

func (h *Handler) CreateGridQuerySnapshot(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireAnyCapability(writer, request, "projects.read", "projects.manage")
	if !ok {
		return
	}
	var body struct {
		ResourceType string `json:"resourceType"`
		Query        struct {
			Q         string         `json:"q"`
			Search    string         `json:"search"`
			Sort      string         `json:"sort"`
			Direction string         `json:"direction"`
			Filters   map[string]any `json:"f"`
		} `json:"query"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	errorsByField := validateGridSnapshotInput(body.ResourceType, body.Query.Q, body.Query.Search, body.Query.Sort, body.Query.Direction, body.Query.Filters)
	if len(errorsByField) != 0 {
		validationErrors(writer, errorsByField)
		return
	}
	search := strings.TrimSpace(body.Query.Q)
	if search == "" {
		search = strings.TrimSpace(body.Query.Search)
	}
	sort := body.Query.Sort
	if sort == "" {
		sort = "id"
	}
	direction := body.Query.Direction
	if direction == "" {
		direction = "asc"
	}
	rawQuery := map[string]any{}
	encoded, _ := json.Marshal(body.Query)
	_ = json.Unmarshal(encoded, &rawQuery)
	snapshot, err := h.sessions.CreateGridQuerySnapshot(request, user, GridQuerySnapshotCreate{ResourceType: body.ResourceType, Query: GridQuery{Search: search, Sort: sort, Direction: direction, Filters: body.Query.Filters, Raw: rawQuery}, Now: h.now().UTC()})
	var validationFailure ValidationFailure
	if errors.As(err, &validationFailure) {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "GRID_SNAPSHOT_TOO_LARGE", "message": "The matching result exceeds the bulk operation limit.", "details": map[string]int{"limit": 1000}}})
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser grid query snapshot", err)
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]any{"data": snapshot})
}

func (h *Handler) CreateGridBulkJob(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	var body struct {
		QuerySnapshotID string         `json:"querySnapshotId"`
		Action          string         `json:"action"`
		Payload         map[string]any `json:"payload"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	tags, validation := validateGridJobInput(body.QuerySnapshotID, body.Action, body.Payload)
	if len(validation) != 0 {
		validationErrors(writer, validation)
		return
	}
	required := "projects.manage"
	if body.Action == "export" {
		required = "projects.read"
	}
	if _, ok := h.requireAnyCapabilityForUser(writer, user, required, "projects.manage"); !ok {
		return
	}
	job, err := h.sessions.CreateGridBulkJob(request, user, GridBulkJobCreate{QuerySnapshotID: body.QuerySnapshotID, Action: body.Action, Tags: tags, Payload: body.Payload, Now: h.now().UTC()})
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "create browser grid bulk job", err)
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]any{"data": job})
}

func (h *Handler) ShowGridBulkJob(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return
	}
	jobID := request.PathValue("jobId")
	if !gridUUIDPattern.MatchString(jobID) {
		h.notFound(writer)
		return
	}
	job, err := h.sessions.GetGridBulkJob(request, user, jobID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "show browser grid bulk job", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": job})
}

func (h *Handler) ExportGridBulkJob(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireAnyCapability(writer, request, "projects.read", "projects.manage")
	if !ok {
		return
	}
	jobID := request.PathValue("jobId")
	if !gridUUIDPattern.MatchString(jobID) {
		h.notFound(writer)
		return
	}
	export, err := h.sessions.ExportGridBulkJob(request, user, jobID, h.now().UTC())
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if errors.Is(err, ErrConflict) {
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "The export is not available."})
		return
	}
	if err != nil {
		h.internalError(writer, request, "export browser grid bulk job", err)
		return
	}
	writer.Header().Set("Content-Type", "text/csv; charset=UTF-8")
	writer.Header().Set("Content-Disposition", `attachment; filename="`+export.Filename+`"`)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(export.Payload))
}

func validateGridSnapshotInput(resourceType, query, search, sort, direction string, filters map[string]any) map[string][]string {
	failures := map[string][]string{}
	if resourceType != "projects" {
		failures["resourceType"] = []string{"The selected resource type is invalid."}
	}
	if len(query) > 200 || len(search) > 200 {
		failures["query"] = []string{"The query must not exceed 200 characters."}
	}
	allowedSort := map[string]bool{"": true, "id": true, "name": true, "description": true, "created_at": true, "updated_at": true}
	if !allowedSort[sort] {
		failures["query.sort"] = []string{"The selected query sort is invalid."}
	}
	if direction != "" && direction != "asc" && direction != "desc" {
		failures["query.direction"] = []string{"The selected query direction is invalid."}
	}
	for key, value := range filters {
		if key != "id" && key != "name" {
			failures["query.f"] = []string{"The query filter is invalid."}
		}
		if key == "name" {
			name, valid := value.(string)
			if !valid || len(name) > 200 {
				failures["query.f.name"] = []string{"The query name filter is invalid."}
			}
		}
		if key == "id" {
			number, valid := value.(float64)
			if !valid || number < 1 || number != float64(int64(number)) {
				failures["query.f.id"] = []string{"The query id filter is invalid."}
			}
		}
	}
	return failures
}

func validateGridJobInput(snapshotID, action string, payload map[string]any) ([]string, map[string][]string) {
	failures := map[string][]string{}
	if !gridUUIDPattern.MatchString(snapshotID) {
		failures["querySnapshotId"] = []string{"The query snapshot id must be a valid UUID."}
	}
	if action != "archive" && action != "export" && action != "tag" {
		failures["action"] = []string{"The selected action is invalid."}
	}
	tags := []string{}
	if action == "tag" {
		rawTags, valid := payload["tags"].([]any)
		if !valid || len(rawTags) < 1 || len(rawTags) > 10 {
			failures["payload.tags"] = []string{"The payload tags field must contain between 1 and 10 items."}
		} else {
			seen := map[string]bool{}
			for _, rawTag := range rawTags {
				tag, valid := rawTag.(string)
				if !valid || len(tag) > 40 || !gridTagPattern.MatchString(tag) {
					failures["payload.tags"] = []string{"The payload tags format is invalid."}
					break
				}
				if !seen[tag] {
					seen[tag] = true
					tags = append(tags, tag)
				}
			}
		}
	}
	return tags, failures
}

func (h *Handler) IntegrationEndpoints(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "integrations.read")
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	endpoints, err := h.sessions.ListIntegrationEndpoints(request, user, projectID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser integration endpoints", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": endpoints})
}

func (h *Handler) CreateIntegrationEndpoint(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "integrations.manage")
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	var body struct {
		Name    string   `json:"name"`
		Adapter string   `json:"adapter"`
		URL     string   `json:"url"`
		Secret  string   `json:"secret"`
		Events  []string `json:"events"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	validation := validateIntegrationEndpointInput(body.Name, body.Adapter, body.URL, body.Secret, body.Events)
	if len(validation) != 0 {
		validationErrors(writer, validation)
		return
	}
	if len(body.Events) == 0 {
		body.Events = []string{"*"}
	}
	endpoint, err := h.sessions.CreateIntegrationEndpoint(request, user, IntegrationEndpointCreate{ProjectID: projectID, Name: body.Name, Adapter: body.Adapter, URL: body.URL, Secret: body.Secret, Events: uniqueStrings(body.Events), Now: h.now().UTC()})
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
		h.internalError(writer, request, "create browser integration endpoint", err)
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]any{"data": endpoint})
}

func (h *Handler) TestIntegrationEndpoint(writer http.ResponseWriter, request *http.Request) {
	user, projectID, endpointID, ok := h.integrationEndpointPath(writer, request, "integrations.manage")
	if !ok {
		return
	}
	delivery, err := h.sessions.CreateIntegrationTestDelivery(request, user, projectID, endpointID, h.now().UTC())
	h.writeIntegrationDelivery(writer, request, "create browser integration test delivery", delivery, err, http.StatusAccepted)
}

func (h *Handler) UpdateIntegrationEndpointStatus(writer http.ResponseWriter, request *http.Request) {
	user, projectID, endpointID, ok := h.integrationEndpointPath(writer, request, "integrations.manage")
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	if body.Status != "active" && body.Status != "disabled" {
		validationError(writer, "status", "The selected status is invalid.")
		return
	}
	endpoint, err := h.sessions.UpdateIntegrationEndpointStatus(request, user, projectID, endpointID, body.Status, h.now().UTC())
	h.writeIntegrationEndpoint(writer, request, "update browser integration endpoint status", endpoint, err, http.StatusOK)
}

func (h *Handler) RotateIntegrationEndpointSecret(writer http.ResponseWriter, request *http.Request) {
	user, projectID, endpointID, ok := h.integrationEndpointPath(writer, request, "integrations.manage")
	if !ok {
		return
	}
	var body struct {
		Secret string `json:"secret"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	if len(body.Secret) < 16 || len(body.Secret) > 2000 {
		validationError(writer, "secret", "The secret field must be between 16 and 2000 characters.")
		return
	}
	endpoint, err := h.sessions.RotateIntegrationEndpointSecret(request, user, projectID, endpointID, body.Secret, h.now().UTC())
	h.writeIntegrationEndpoint(writer, request, "rotate browser integration endpoint secret", endpoint, err, http.StatusOK)
}

func (h *Handler) IntegrationDeliveries(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "integrations.read")
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	status := request.URL.Query().Get("status")
	if status != "" && status != "pending" && status != "sent" && status != "failed" && status != "dead_letter" {
		validationError(writer, "status", "The selected status is invalid.")
		return
	}
	deliveries, err := h.sessions.ListIntegrationDeliveries(request, user, projectID, status)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser integration deliveries", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": deliveries})
}

func (h *Handler) ReplayIntegrationDelivery(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "integrations.manage")
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	deliveryID, err := parsePathID(request.PathValue("integrationDelivery"))
	if err != nil {
		h.notFound(writer)
		return
	}
	delivery, err := h.sessions.ReplayIntegrationDelivery(request, user, projectID, deliveryID, h.now().UTC())
	h.writeIntegrationDelivery(writer, request, "replay browser integration delivery", delivery, err, http.StatusAccepted)
}

func (h *Handler) AuditEvents(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "audit_events.read")
	if !ok {
		return
	}
	values := request.URL.Query()
	filter := AuditEventFilter{Action: values.Get("action"), TargetType: values.Get("targetType"), TargetID: values.Get("targetId"), CorrelationID: values.Get("correlationId"), Limit: 100}
	validation := map[string][]string{}
	for field, value := range map[string]string{"action": filter.Action, "targetType": filter.TargetType, "targetId": filter.TargetID} {
		if len(value) > 128 {
			validation[field] = []string{"The " + field + " field must not exceed 128 characters."}
		}
	}
	if filter.CorrelationID != "" && !auditUUIDPattern.MatchString(filter.CorrelationID) {
		validation["correlationId"] = []string{"The correlation id field must be a valid UUID."}
	}
	if value := values.Get("from"); value != "" {
		parsed, valid := parseAuditDate(value)
		if !valid {
			validation["from"] = []string{"The from field must be a valid date."}
		} else {
			filter.From = &parsed
		}
	}
	if value := values.Get("to"); value != "" {
		parsed, valid := parseAuditDate(value)
		if !valid {
			validation["to"] = []string{"The to field must be a valid date."}
		} else {
			filter.To = &parsed
		}
	}
	if value := values.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			validation["limit"] = []string{"The limit field must be between 1 and 200."}
		} else {
			filter.Limit = parsed
		}
	}
	if len(validation) != 0 {
		validationErrors(writer, validation)
		return
	}
	events, err := h.sessions.ListAuditEvents(request, user, filter)
	if err != nil {
		h.internalError(writer, request, "list browser audit events", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": events})
}

func parseAuditDate(value string) (time.Time, bool) {
	for _, format := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func (h *Handler) AssetImpact(writer http.ResponseWriter, request *http.Request) {
	user, projectID, assetType, assetID, ok := h.assetPath(writer, request, true)
	if !ok {
		return
	}
	impact, err := h.sessions.AssetImpact(request, user, projectID, assetType, assetID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "load browser asset impact", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": impact})
}

func (h *Handler) AssetVersions(writer http.ResponseWriter, request *http.Request) {
	user, projectID, assetType, assetID, ok := h.assetPath(writer, request, false)
	if !ok {
		return
	}
	versions, err := h.sessions.ListAssetVersions(request, user, projectID, assetType, assetID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "list browser asset versions", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": versions})
}

func (h *Handler) ShowAssetVersion(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "resources.read")
	if !ok {
		return
	}
	projectID, versionID, ok := parseAssetVersionPath(writer, request, "assetVersion")
	if !ok {
		return
	}
	version, err := h.sessions.GetAssetVersion(request, user, projectID, versionID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "show browser asset version", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": version})
}

func (h *Handler) DiffAssetVersions(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "resources.read")
	if !ok {
		return
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return
	}
	fromID, err := parsePathID(request.PathValue("fromVersion"))
	if err != nil {
		h.notFound(writer)
		return
	}
	toID, err := parsePathID(request.PathValue("toVersion"))
	if err != nil {
		h.notFound(writer)
		return
	}
	from, err := h.sessions.GetAssetVersion(request, user, projectID, fromID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "load browser asset diff source", err)
		return
	}
	to, err := h.sessions.GetAssetVersion(request, user, projectID, toID)
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "load browser asset diff target", err)
		return
	}
	if from.AssetType != to.AssetType || from.AssetID != to.AssetID {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Asset versions must belong to the same asset before they can be compared."})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": assetVersionDiff(from, to)})
}

func (h *Handler) TransitionAssetVersionReview(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "resources.manage")
	if !ok {
		return
	}
	projectID, versionID, ok := parseAssetVersionPath(writer, request, "assetVersion")
	if !ok {
		return
	}
	var body struct {
		ToStatus string  `json:"toStatus"`
		Comment  *string `json:"comment"`
	}
	if err := decodeJSON(writer, request, &body); err != nil {
		return
	}
	if body.ToStatus != "in_review" && body.ToStatus != "approved" && body.ToStatus != "deprecated" {
		validationError(writer, "toStatus", "The selected to status is invalid.")
		return
	}
	if body.Comment != nil && len(*body.Comment) > 2000 {
		validationError(writer, "comment", "The comment field must not exceed 2000 characters.")
		return
	}
	event, err := h.sessions.TransitionAssetVersionReview(request, user, projectID, versionID, body.ToStatus, body.Comment, h.now().UTC())
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	var reviewFailure ReviewFailure
	if errors.As(err, &reviewFailure) {
		payload := map[string]any{"message": reviewFailure.Message}
		if reviewFailure.FromStatus != "" {
			payload["fromStatus"] = reviewFailure.FromStatus
			payload["toStatus"] = reviewFailure.ToStatus
		}
		writeJSON(writer, http.StatusUnprocessableEntity, payload)
		return
	}
	if err != nil {
		h.internalError(writer, request, "transition browser asset review", err)
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]any{"data": event})
}

func (h *Handler) assetPath(writer http.ResponseWriter, request *http.Request, allowPlugin bool) (User, int64, string, int64, bool) {
	user, ok := h.requireCapability(writer, request, "resources.read")
	if !ok {
		return User{}, 0, "", 0, false
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return User{}, 0, "", 0, false
	}
	assetID, err := parsePathID(request.PathValue("assetId"))
	if err != nil {
		h.notFound(writer)
		return User{}, 0, "", 0, false
	}
	assetType := request.PathValue("assetType")
	allowed := assetType == "environment" || assetType == "step" || assetType == "test" || assetType == "test_cycle" || (allowPlugin && assetType == "plugin")
	if !allowed {
		validationError(writer, "assetType", "The selected asset type is invalid.")
		return User{}, 0, "", 0, false
	}
	return user, projectID, assetType, assetID, true
}

func parseAssetVersionPath(writer http.ResponseWriter, request *http.Request, versionName string) (int64, int64, bool) {
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return 0, 0, false
	}
	versionID, err := parsePathID(request.PathValue(versionName))
	if err != nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return 0, 0, false
	}
	return projectID, versionID, true
}

func assetVersionDiff(from, to AssetVersion) map[string]any {
	fromSnapshot := map[string]any{}
	toSnapshot := map[string]any{}
	if from.Snapshot != nil {
		fromSnapshot = *from.Snapshot
	}
	if to.Snapshot != nil {
		toSnapshot = *to.Snapshot
	}
	keys := make([]string, 0, len(fromSnapshot)+len(toSnapshot))
	seen := map[string]bool{}
	for key := range fromSnapshot {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range toSnapshot {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	added := map[string]any{}
	removed := map[string]any{}
	changed := map[string]any{}
	for _, key := range keys {
		fromValue, fromExists := fromSnapshot[key]
		toValue, toExists := toSnapshot[key]
		if !fromExists {
			added[key] = toValue
		} else if !toExists {
			removed[key] = fromValue
		} else if !reflect.DeepEqual(fromValue, toValue) {
			changed[key] = map[string]any{"from": fromValue, "to": toValue}
		}
	}
	return map[string]any{"from": map[string]any{"assetType": from.AssetType, "assetId": from.AssetID, "version": from.Version, "versionId": from.ID}, "to": map[string]any{"assetType": to.AssetType, "assetId": to.AssetID, "version": to.Version, "versionId": to.ID}, "changes": map[string]any{"added": added, "removed": removed, "changed": changed}}
}

func (h *Handler) integrationEndpointPath(writer http.ResponseWriter, request *http.Request, capability string) (User, int64, int64, bool) {
	user, ok := h.requireCapability(writer, request, capability)
	if !ok {
		return User{}, 0, 0, false
	}
	projectID, err := parsePathID(request.PathValue("idProject"))
	if err != nil {
		h.notFound(writer)
		return User{}, 0, 0, false
	}
	endpointID, err := parsePathID(request.PathValue("integrationEndpoint"))
	if err != nil {
		h.notFound(writer)
		return User{}, 0, 0, false
	}
	return user, projectID, endpointID, true
}

func (h *Handler) writeIntegrationEndpoint(writer http.ResponseWriter, request *http.Request, action string, endpoint IntegrationEndpoint, err error, status int) {
	if errors.Is(err, ErrNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, action, err)
		return
	}
	writeJSON(writer, status, map[string]any{"data": endpoint})
}

func (h *Handler) writeIntegrationDelivery(writer http.ResponseWriter, request *http.Request, action string, delivery IntegrationDelivery, err error, status int) {
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
	writeJSON(writer, status, map[string]any{"data": delivery})
}

func validateIntegrationEndpointInput(name, adapter, destination, secret string, events []string) map[string][]string {
	failures := map[string][]string{}
	if strings.TrimSpace(name) == "" || len(name) > 128 {
		failures["name"] = []string{"The name field is required and must not exceed 128 characters."}
	}
	if adapter != "webhook" && adapter != "jira" && adapter != "slack" && adapter != "teams" {
		failures["adapter"] = []string{"The selected adapter is invalid."}
	}
	if len(destination) > 2048 || !safeIntegrationDestination(destination) {
		failures["url"] = []string{"The integration destination must be a safe public HTTP or HTTPS URL."}
	}
	if len(secret) < 16 || len(secret) > 2000 {
		failures["secret"] = []string{"The secret field must be between 16 and 2000 characters."}
	}
	for _, event := range events {
		if strings.TrimSpace(event) == "" || len(event) > 128 {
			failures["events"] = []string{"Each event must be a non-empty string of at most 128 characters."}
			break
		}
	}
	return failures
}

func safeIntegrationDestination(value string) bool {
	return integrations.SafeDestination(value)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
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

func (h *Handler) requireAnyCapability(writer http.ResponseWriter, request *http.Request, capabilities ...string) (User, bool) {
	user, ok := h.authenticatedUser(writer, request)
	if !ok {
		return User{}, false
	}
	return h.requireAnyCapabilityForUser(writer, user, capabilities...)
}

func (h *Handler) requireAnyCapabilityForUser(writer http.ResponseWriter, user User, capabilities ...string) (User, bool) {
	for _, allowed := range capabilitiesForRole(user.Role) {
		for _, required := range capabilities {
			if allowed == required {
				return user, true
			}
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
