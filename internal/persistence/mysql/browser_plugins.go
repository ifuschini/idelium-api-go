package mysql

import (
	"database/sql"
	"encoding/json"
	"github.com/idelium/idelium-api-go/internal/browserauth"
	"net/http"
)

func scanBrowserPlugin(s interface{ Scan(...any) error }) (browserauth.PluginDetail, error) {
	var p browserauth.PluginDetail
	var code sql.NullString
	if err := s.Scan(&p.ID, &p.Name, &p.Description, &code, &p.IDProject); err != nil {
		return p, err
	}
	p.Code = map[string]any{}
	if code.Valid {
		_ = json.Unmarshal([]byte(code.String), &p.Code)
	}
	return p, nil
}
func (r *BrowserAuthRepository) ListBrowserPlugins(req *http.Request, a browserauth.User, pid int64) ([]browserauth.PluginDetail, error) {
	rows, e := r.database.QueryContext(req.Context(), "SELECT id,name,description,code,idProject FROM plugins WHERE idProject=? AND idCostumer=? ORDER BY id", pid, a.ActiveTenant())
	if e != nil {
		return nil, safeDatabaseFailure("list browser plugins", e)
	}
	defer rows.Close()
	out := []browserauth.PluginDetail{}
	for rows.Next() {
		p, er := scanBrowserPlugin(rows)
		if er != nil {
			return nil, er
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *BrowserAuthRepository) GetBrowserPlugin(req *http.Request, a browserauth.User, pid, id int64) (browserauth.PluginDetail, error) {
	p, e := scanBrowserPlugin(r.database.QueryRowContext(req.Context(), "SELECT id,name,description,code,idProject FROM plugins WHERE id=? AND idProject=? AND idCostumer=?", id, pid, a.ActiveTenant()))
	if e == sql.ErrNoRows {
		return p, browserauth.ErrNotFound
	}
	if e != nil {
		return p, safeDatabaseFailure("show browser plugin", e)
	}
	return p, nil
}
