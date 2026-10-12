package demo

import (
	"context"
	"time"
)

func init() { seedAudience = seedAudienceImpl }

// seedAudienceImpl creates the three lists of the sample data. Contacts,
// custom fields, segments and suppressions are added by the Audience domain.
func seedAudienceImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	lists := []struct {
		name, desc       string
		requiresApproval bool
		created          time.Duration
	}{
		{"Weekly Digest", "Every Tuesday: product news, guides and the changelog.", false, 400 * 24 * time.Hour},
		{"Product Updates", "Launches and deprecations, about twice a month.", false, 380 * 24 * time.Hour},
		{"Beta Program", "Early access to features before they ship.", false, 120 * 24 * time.Hour},
	}
	for _, l := range lists {
		id := ID()
		if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, requires_approval, created_at) VALUES ($1,$2,$3,$4,true,$5,$6)`,
			id, ws.ID, l.name, l.desc, l.requiresApproval, s.Ago(l.created)); err != nil {
			return err
		}
		s.Lists[l.name] = id
	}
	eng := s.Workspaces["acme-engineering-blog"]
	id := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, created_at) VALUES ($1,$2,'Engineering Blog','New posts from the Acme engineering team.',$3)`, id, eng.ID, s.Ago(300*24*time.Hour)); err != nil {
		return err
	}
	s.Lists["Engineering Blog"] = id
	return nil
}
