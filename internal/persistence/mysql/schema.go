package mysql

import (
	"context"
	"database/sql"
)

func mysqlColumnExists(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, table string, column string) (bool, error) {
	var exists bool
	err := queryer.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			  FROM information_schema.columns
			 WHERE table_schema = DATABASE()
			   AND table_name = ?
			   AND column_name = ?
		)`,
		table,
		column,
	).Scan(&exists)
	return exists, err
}
