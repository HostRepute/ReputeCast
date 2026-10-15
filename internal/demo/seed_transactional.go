package demo

import (
	"context"

	"github.com/hostrepute/reputecast/internal/authz"
)

// Transactional email (design expansion phase 1, brief §4): Acme Cloud,
// Acme's SaaS product, sends its product mail through the Acme Newsletter
// workspace.
func init() { seedTransactional = seedTransactionalImpl }

func seedTransactionalImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	// Content Writer edits transactional drafts; publishing needs
	// transactional:publish, so Daniel asks Sofia, Marcus or Linh.
	if cw, ok := s.Roles[ws.Slug+"/content-writer"]; ok {
		perms := authz.ParseSet(cw.Permissions)
		for _, p := range []authz.Permission{authz.TransactionalView, authz.TransactionalEdit} {
			perms[p] = struct{}{}
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE roles SET permissions = $2 WHERE id = $1`, cw.ID, perms.Strings()); err != nil {
			return err
		}
	}
	return nil
}
