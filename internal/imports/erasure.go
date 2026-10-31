package imports

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the contact's rows in imports that haven't finished (a done
// import drops its rows): the address, its fields and tags (review 2026-11
// DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "imports",
		Tables: []erasure.Table{
			{Name: "import_rows", Action: "clear", Columns: []string{"email", "data", "tags", "engagement"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE import_rows SET email = '', data = '{}', tags = '{}', engagement = '{}'
				 WHERE lower(email) = ANY ($2) AND import_id IN (SELECT id FROM imports WHERE workspace_id = $4)`)
		},
	}
}
