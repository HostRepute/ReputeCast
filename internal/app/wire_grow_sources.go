package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
)

// Grow's sources in the Contacts list and on the profile (brief §3.7):
// "Referral · Sam Rivera" for a reader who joined through a referral link,
// "Partner · Ops Weekly" for one who asked on a partner's page.
func init() {
	audienceSourceNames = func(a *App) audience.SourceNamer { return growSourceNames(a) }
}

func growSourceNames(a *App) audience.SourceNamer {
	return func(ctx context.Context, ws uuid.UUID, contacts []uuid.UUID, sources []string) map[uuid.UUID]string {
		out := map[uuid.UUID]string{}
		var referred []uuid.UUID
		partnerOf := map[uuid.UUID][]uuid.UUID{}
		var links []uuid.UUID
		for i, src := range sources {
			if i >= len(contacts) {
				break
			}
			switch {
			case src == "referral" || strings.HasPrefix(src, "referral:"):
				referred = append(referred, contacts[i])
			case strings.HasPrefix(src, "partner:"):
				if id, err := uuid.Parse(strings.TrimPrefix(src, "partner:")); err == nil {
					if _, ok := partnerOf[id]; !ok {
						links = append(links, id)
					}
					partnerOf[id] = append(partnerOf[id], contacts[i])
				}
			}
		}
		if len(referred) > 0 {
			rows, err := a.Pool.Query(ctx, `SELECT DISTINCT ON (r.contact_id) r.contact_id, c.first_name, c.last_name, c.email
				FROM referrals r JOIN contacts c ON c.id = r.referrer_id AND c.erased_at IS NULL
				WHERE r.workspace_id = $1 AND r.contact_id = ANY ($2) ORDER BY r.contact_id, r.signed_up_at`, ws, referred)
			if err == nil {
				for rows.Next() {
					var id uuid.UUID
					var first, last, email string
					if rows.Scan(&id, &first, &last, &email) == nil {
						name := strings.TrimSpace(first + " " + last)
						if name == "" {
							name = email
						}
						out[id] = "Referral · " + name
					}
				}
				rows.Close()
			}
		}
		if len(links) > 0 {
			rows, err := a.Pool.Query(ctx, `SELECT id, remote_name FROM partner_links WHERE workspace_id = $1 AND id = ANY ($2)`, ws, links)
			if err == nil {
				for rows.Next() {
					var id uuid.UUID
					var name string
					if rows.Scan(&id, &name) == nil {
						for _, c := range partnerOf[id] {
							out[c] = "Partner · " + name
						}
					}
				}
				rows.Close()
			}
		}
		return out
	}
}
