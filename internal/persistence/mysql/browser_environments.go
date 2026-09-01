package mysql

import (
	"database/sql"
	"encoding/json"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"net/http"
	"strings"
	"time"
)

func envConfig(v any) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func (r *BrowserAuthRepository) CreateEnvironment(req *http.Request, a browserauth.User, in browserauth.EnvironmentInput) error {
	if err := r.ensureProject(req.Context(), a.ActiveTenant(), in.IDProject); err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err := r.database.ExecContext(req.Context(), "INSERT INTO environments (code,description,config,idProject,idCostumer,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", in.Code, in.Description, envConfig(in.Config), in.IDProject, a.ActiveTenant(), now, now)
	if err != nil {
		return safeDatabaseFailure("create browser environment", err)
	}
	return nil
}
func (r *BrowserAuthRepository) UpdateEnvironment(req *http.Request, a browserauth.User, in browserauth.EnvironmentInput) error {
	res, err := r.database.ExecContext(req.Context(), "UPDATE environments SET code=?,description=?,config=?,updated_at=? WHERE id=? AND idProject=? AND idCostumer=?", in.Code, in.Description, envConfig(in.Config), time.Now().UTC(), in.ID, in.IDProject, a.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("update browser environment", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}
func (r *BrowserAuthRepository) DeleteEnvironment(req *http.Request, a browserauth.User, pid, id int64) error {
	res, err := r.database.ExecContext(req.Context(), "DELETE FROM environments WHERE id=? AND idProject=? AND idCostumer=?", id, pid, a.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("delete browser environment", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func redactEnvironment(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "secret") || strings.Contains(lk, "password") || strings.Contains(lk, "token") || strings.Contains(lk, "cookie") {
				out[k] = "[REDACTED]"
			} else {
				out[k] = redactEnvironment(val)
			}
		}
		return out
	case []any:
		for i := range x {
			x[i] = redactEnvironment(x[i])
		}
		return x
	default:
		return v
	}
}
func scanBrowserEnvironment(s interface{ Scan(...any) error }) (browserauth.EnvironmentDetail, error) {
	var e browserauth.EnvironmentDetail
	var cfg sql.NullString
	if err := s.Scan(&e.ID, &e.Code, &e.Description, &cfg, &e.IDProject); err != nil {
		return e, err
	}
	e.Config = map[string]any{}
	if cfg.Valid {
		var v any
		if json.Unmarshal([]byte(cfg.String), &v) == nil {
			e.Config = redactEnvironment(v)
		}
	}
	return e, nil
}
func (r *BrowserAuthRepository) ListEnvironments(req *http.Request, a browserauth.User, pid int64) ([]browserauth.EnvironmentDetail, error) {
	rows, err := r.database.QueryContext(req.Context(), "SELECT id,code,description,config,idProject FROM environments WHERE idProject=? AND idCostumer=? ORDER BY id", pid, a.ActiveTenant())
	if err != nil {
		return nil, safeDatabaseFailure("list browser environments", err)
	}
	defer rows.Close()
	out := []browserauth.EnvironmentDetail{}
	for rows.Next() {
		e, er := scanBrowserEnvironment(rows)
		if er != nil {
			return nil, er
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r *BrowserAuthRepository) GetEnvironment(req *http.Request, a browserauth.User, pid, id int64) (browserauth.EnvironmentDetail, error) {
	e, err := scanBrowserEnvironment(r.database.QueryRowContext(req.Context(), "SELECT id,code,description,config,idProject FROM environments WHERE id=? AND idProject=? AND idCostumer=?", id, pid, a.ActiveTenant()))
	if err == sql.ErrNoRows {
		return e, browserauth.ErrNotFound
	}
	if err != nil {
		return e, safeDatabaseFailure("show browser environment", err)
	}
	return e, nil
}
