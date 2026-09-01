package browserauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/idelium/idelium-api-go/internal/auth"
)

var ErrAgentNotFound = errors.New("agent registration not found")

type AgentRegistration struct {
	ID             int64      `json:"id"`
	AgentID        string     `json:"agentId"`
	Status         string     `json:"status"`
	Version        *string    `json:"version"`
	Runtimes       any        `json:"runtimes"`
	Capabilities   any        `json:"capabilities"`
	IdentityProof  any        `json:"identityProof"`
	MaxConcurrency int        `json:"maxConcurrency"`
	Health         string     `json:"health"`
	LastSeenAt     *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}
type AgentRegistrationInput struct {
	AgentID        string  `json:"agentId"`
	Version        *string `json:"version"`
	Runtimes       any     `json:"runtimes"`
	Capabilities   any     `json:"capabilities"`
	IdentityProof  any     `json:"identityProof"`
	MaxConcurrency int     `json:"maxConcurrency"`
	Health         string  `json:"health"`
}
type AgentStatusUpdate struct {
	Status string `json:"status"`
}

type AgentRepository interface {
	ListAgents(*http.Request, int64) ([]AgentRegistration, error)
	RegisterAgent(*http.Request, int64, AgentRegistrationInput) (AgentRegistration, bool, error)
	UpdateAgentStatus(*http.Request, int64, int64, string) (AgentRegistration, error)
}

func (h *Handler) Agents(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "agents.read")
	if !ok {
		return
	}
	repo, ok := h.sessions.(AgentRepository)
	if !ok {
		h.internalError(writer, request, "list agents", errors.New("agent repository unavailable"))
		return
	}
	agents, err := repo.ListAgents(request, user.ActiveTenant())
	if err != nil {
		h.internalError(writer, request, "list agents", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": agents})
}
func (h *Handler) UpdateAgentStatus(writer http.ResponseWriter, request *http.Request) {
	user, ok := h.requireCapability(writer, request, "agents.manage")
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(request, "agentRegistration"), 10, 64)
	if err != nil {
		h.notFound(writer)
		return
	}
	var body AgentStatusUpdate
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil || !validAgentStatus(body.Status) {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "The given data was invalid."})
		return
	}
	repo, ok := h.sessions.(AgentRepository)
	if !ok {
		h.internalError(writer, request, "update agent status", errors.New("agent repository unavailable"))
		return
	}
	agent, err := repo.UpdateAgentStatus(request, user.ActiveTenant(), id, body.Status)
	if errors.Is(err, ErrAgentNotFound) {
		h.notFound(writer)
		return
	}
	if err != nil {
		h.internalError(writer, request, "update agent status", err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"data": agent})
}
func (h *Handler) CLRegisterAgent(writer http.ResponseWriter, request *http.Request) {
	tenant, ok := auth.TenantFromContext(request.Context())
	if !ok {
		h.unauthorized(writer)
		return
	}
	var body AgentRegistrationInput
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil || strings.TrimSpace(body.AgentID) == "" || len(body.AgentID) > 128 || body.MaxConcurrency < 0 || body.MaxConcurrency > 256 || !validAgentHealth(body.Health) {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]any{"message": "The given data was invalid."})
		return
	}
	repo, ok := h.sessions.(AgentRepository)
	if !ok {
		h.internalError(writer, request, "register agent", errors.New("agent repository unavailable"))
		return
	}
	agent, created, err := repo.RegisterAgent(request, tenant.CustomerID, body)
	if err != nil {
		h.internalError(writer, request, "register agent", err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(writer, status, map[string]any{"data": agent})
}
func validAgentStatus(s string) bool {
	return s == "approved" || s == "maintenance" || s == "draining" || s == "disabled"
}
func validAgentHealth(s string) bool {
	return s == "" || s == "unknown" || s == "healthy" || s == "unhealthy"
}
