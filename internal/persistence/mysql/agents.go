package mysql

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

const agentSelect = `SELECT id, agentId, status, version, runtimes, capabilities, identityProof, maxConcurrency, health, lastSeenAt, created_at, updated_at FROM agent_registrations`

func scanAgent(s interface{ Scan(...any) error }) (browserauth.AgentRegistration, error) {
	var a browserauth.AgentRegistration
	var v, rt, cap, proof sql.NullString
	var last, created, updated sql.NullTime
	err := s.Scan(&a.ID, &a.AgentID, &a.Status, &v, &rt, &cap, &proof, &a.MaxConcurrency, &a.Health, &last, &created, &updated)
	if err != nil {
		return a, err
	}
	if v.Valid {
		a.Version = &v.String
	}
	a.Runtimes = decodeJSONValue(rt)
	a.Capabilities = decodeJSONValue(cap)
	a.IdentityProof = decodeJSONValue(proof)
	if last.Valid {
		a.LastSeenAt = &last.Time
	}
	if created.Valid {
		a.CreatedAt = &created.Time
	}
	if updated.Valid {
		a.UpdatedAt = &updated.Time
	}
	return a, nil
}
func decodeJSONValue(v sql.NullString) any {
	if !v.Valid || v.String == "" {
		return []any{}
	}
	var x any
	if json.Unmarshal([]byte(v.String), &x) != nil {
		return []any{}
	}
	return x
}

func (r *BrowserAuthRepository) ListAgents(req *http.Request, tenantID int64) ([]browserauth.AgentRegistration, error) {
	rows, err := r.database.QueryContext(req.Context(), agentSelect+` WHERE idCostumer = ? ORDER BY agentId`, tenantID)
	if err != nil {
		return nil, safeDatabaseFailure("list agents", err)
	}
	defer rows.Close()
	out := []browserauth.AgentRegistration{}
	for rows.Next() {
		a, e := scanAgent(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *BrowserAuthRepository) RegisterAgent(req *http.Request, tenantID int64, in browserauth.AgentRegistrationInput) (browserauth.AgentRegistration, bool, error) {
	ctx := req.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return browserauth.AgentRegistration{}, false, safeDatabaseFailure("start agent registration", err)
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM agent_registrations WHERE idCostumer=? AND agentId=? FOR UPDATE", tenantID, in.AgentID).Scan(&id)
	created := err == sql.ErrNoRows
	if err != nil && !created {
		return browserauth.AgentRegistration{}, false, safeDatabaseFailure("find agent registration", err)
	}
	rt, _ := json.Marshal(in.Runtimes)
	if in.Runtimes == nil {
		rt = []byte("[]")
	}
	cap, _ := json.Marshal(in.Capabilities)
	if in.Capabilities == nil {
		cap = []byte("[]")
	}
	proof, _ := json.Marshal(in.IdentityProof)
	if in.IdentityProof == nil {
		proof = []byte("[]")
	}
	health := in.Health
	if health == "" {
		health = "unknown"
	}
	max := in.MaxConcurrency
	if max == 0 {
		max = 1
	}
	if created {
		_, err = tx.ExecContext(ctx, `INSERT INTO agent_registrations (idCostumer,agentId,status,version,runtimes,capabilities,identityProof,maxConcurrency,health,lastSeenAt,created_at,updated_at) VALUES (?,?, 'pending',?,?,?,?,?, ?, NOW(),NOW(),NOW())`, tenantID, in.AgentID, in.Version, string(rt), string(cap), string(proof), max, health)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE agent_registrations SET version=COALESCE(?,version),runtimes=?,capabilities=?,identityProof=?,maxConcurrency=?,health=?,lastSeenAt=NOW(),updated_at=NOW() WHERE id=? AND idCostumer=?`, in.Version, string(rt), string(cap), string(proof), max, health, id, tenantID)
	}
	if err != nil {
		return browserauth.AgentRegistration{}, false, safeDatabaseFailure("save agent registration", err)
	}
	if created {
		err = tx.QueryRowContext(ctx, "SELECT id FROM agent_registrations WHERE idCostumer=? AND agentId=?", tenantID, in.AgentID).Scan(&id)
	}
	if err != nil {
		return browserauth.AgentRegistration{}, false, safeDatabaseFailure("read agent registration", err)
	}
	a, err := scanAgent(tx.QueryRowContext(ctx, agentSelect+" WHERE id=? AND idCostumer=?", id, tenantID))
	if err != nil {
		return a, false, err
	}
	if err = tx.Commit(); err != nil {
		return a, false, safeDatabaseFailure("commit agent registration", err)
	}
	return a, created, nil
}
func (r *BrowserAuthRepository) UpdateAgentStatus(req *http.Request, tenantID, id int64, status string) (browserauth.AgentRegistration, error) {
	res, err := r.database.ExecContext(req.Context(), "UPDATE agent_registrations SET status=?,updated_at=NOW() WHERE id=? AND idCostumer=?", status, id, tenantID)
	if err != nil {
		return browserauth.AgentRegistration{}, safeDatabaseFailure("update agent status", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.AgentRegistration{}, browserauth.ErrAgentNotFound
	}
	return scanAgent(r.database.QueryRowContext(req.Context(), agentSelect+" WHERE id=? AND idCostumer=?", id, tenantID))
}
