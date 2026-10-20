package push

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// FillChannelRows fills in the campaigns list's push rows (implements
// campaigns.ChannelRows): the sites and how many each reached, shown and
// click rates once notifications went, and the review's verdict before.
func (s *Service) FillChannelRows(ctx context.Context, ws *store.Workspace, items []campaigns.CampaignListItem) {
	ids := []uuid.UUID{}
	for _, it := range items {
		if it.Channel == "push" {
			if id, err := uuid.Parse(it.ID); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	type stat struct {
		planned, sent, shown, clicks int64
		routes                       []campaigns.RouteShare
	}
	stats := map[uuid.UUID]*stat{}
	rows, err := s.Pool.Query(ctx, `SELECT m.campaign_id, coalesce(st.id::text, ''), coalesce(st.host, ''),
			count(*) FILTER (WHERE m.status NOT IN ('skipped','cancelled')), count(*) FILTER (WHERE m.sent_at IS NOT NULL),
			count(*) FILTER (WHERE p.shown_at IS NOT NULL), count(*) FILTER (WHERE p.clicked_at IS NOT NULL)
		FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		LEFT JOIN push_sites st ON st.id = p.site_id
		WHERE m.campaign_id = ANY($1::uuid[]) AND m.kind = 'campaign'
		GROUP BY 1, 2, 3 ORDER BY 1, 5 DESC`, ids)
	if err != nil {
		s.log().WarnContext(ctx, "push campaign rows", "err", err)
		return
	}
	for rows.Next() {
		var cid uuid.UUID
		var sid, host string
		var planned, sent, shown, clicks int64
		if rows.Scan(&cid, &sid, &host, &planned, &sent, &shown, &clicks) != nil {
			continue
		}
		st, ok := stats[cid]
		if !ok {
			st = &stat{}
			stats[cid] = st
		}
		st.planned += planned
		st.sent += sent
		st.shown += shown
		st.clicks += clicks
		if host != "" && planned > 0 {
			st.routes = append(st.routes, campaigns.RouteShare{ProviderID: sid, Provider: host, Short: host, Adapter: "push", Planned: planned, Sent: sent})
		}
	}
	rows.Close()
	for i := range items {
		it := &items[i]
		if it.Channel != "push" {
			continue
		}
		id, err := uuid.Parse(it.ID)
		if err != nil {
			continue
		}
		st, ok := stats[id]
		if !ok {
			// Not sent yet: the sites it's for, and the review's verdict.
			c, err := s.Q.PushCampaign(ctx, store.PushCampaignParams{ID: id, WorkspaceID: ws.ID})
			if err != nil {
				continue
			}
			aud, err := s.audience(ctx, ws, c.SiteIds, c.Lists, c.Segments)
			if err != nil {
				continue
			}
			n := int64(len(aud.list))
			it.Recipients = &n
			bySite := map[uuid.UUID]int64{}
			for _, r := range aud.list {
				bySite[r.SiteID]++
			}
			if sites, err := s.Q.PushSites(ctx, ws.ID); err == nil {
				for _, site := range sites {
					if k := bySite[site.ID]; k > 0 {
						pct := float64(k) * 100 / float64(max(n, 1))
						it.Routes = append(it.Routes, campaigns.RouteShare{ProviderID: site.ID.String(), Provider: site.Host, Short: site.Host, Adapter: "push",
							Percent: float64(int64(pct*10+0.5)) / 10, Planned: k})
					}
				}
			}
			if it.Status == "draft" || it.Status == "approval" || it.Status == "approved" {
				if rv, err := s.buildReview(ctx, ws, nil, c, false); err == nil {
					warns, blocks := rv.counts()
					verdict := "ready"
					switch {
					case blocks > 0:
						verdict = "blocked"
					case warns > 0:
						verdict = "warnings"
					}
					issues := int64(warns)
					if blocks > 0 {
						issues = int64(blocks)
					}
					it.Readiness = &campaigns.ReadinessSummary{Verdict: verdict, Issues: issues}
					it.Blocked = blocks > 0
				}
			}
			continue
		}
		n := st.sent
		if n == 0 {
			n = st.planned
		}
		it.Recipients = &n
		it.Routes = st.routes
		if st.sent > 0 {
			shown, clicks := pct(st.shown, st.sent), pct(st.clicks, st.sent)
			it.Delivered, it.Clicks = &shown, &clicks
		}
	}
}
