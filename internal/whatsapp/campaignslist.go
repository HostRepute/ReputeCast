package whatsapp

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// FillChannelRows fills in the campaigns list's WhatsApp rows (implements
// campaigns.ChannelRows): the account's number as the route, delivered and
// read once messages went, the cost, and the review's verdict before.
func (s *Service) FillChannelRows(ctx context.Context, ws *store.Workspace, items []campaigns.CampaignListItem) {
	ids := []uuid.UUID{}
	for _, it := range items {
		if it.Channel == "whatsapp" {
			if id, err := uuid.Parse(it.ID); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	type stat struct {
		planned, sent, delivered, read int64
		cost                           float64
	}
	stats := map[uuid.UUID]*stat{}
	rows, err := s.Pool.Query(ctx, `SELECT m.campaign_id, count(*) FILTER (WHERE m.status NOT IN ('skipped','cancelled')),
			count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'delivered'), count(*) FILTER (WHERE w.read_at IS NOT NULL),
			coalesce(sum(m.cost), 0)
		FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
		WHERE m.campaign_id = ANY($1::uuid[]) AND m.kind = 'campaign' GROUP BY 1`, ids)
	if err != nil {
		s.log().WarnContext(ctx, "whatsapp campaign rows", "err", err)
		return
	}
	for rows.Next() {
		var cid uuid.UUID
		st := &stat{}
		if rows.Scan(&cid, &st.planned, &st.sent, &st.delivered, &st.read, &st.cost) == nil {
			stats[cid] = st
		}
	}
	rows.Close()
	number, name := "", ""
	if a, err := s.account(ctx, s.Q, ws); err == nil {
		number, name = a.PhoneNumber, a.DisplayName
	}
	optOuts := map[uuid.UUID]int64{}
	if r, err := s.Pool.Query(ctx, `SELECT campaign_id, count(*) FROM whatsapp_inbound WHERE campaign_id = ANY($1::uuid[]) AND kind = 'opt_out' GROUP BY 1`, ids); err == nil {
		for r.Next() {
			var cid uuid.UUID
			var n int64
			if r.Scan(&cid, &n) == nil {
				optOuts[cid] = n
			}
		}
		r.Close()
	}
	for i := range items {
		it := &items[i]
		if it.Channel != "whatsapp" {
			continue
		}
		id, err := uuid.Parse(it.ID)
		if err != nil {
			continue
		}
		route := func(planned, sent int64) {
			if number == "" {
				return
			}
			it.Routes = append(it.Routes, campaigns.RouteShare{ProviderID: "whatsapp", Provider: "Meta · " + name, Short: "WhatsApp", Adapter: "whatsapp",
				Percent: 100, Planned: planned, Sent: sent})
		}
		st, ok := stats[id]
		if !ok {
			c, err := s.campaign(ctx, ws, id)
			if err != nil {
				continue
			}
			if c.ScheduleMode == "local" {
				it.Local = true
			}
			if it.Status == "draft" || it.Status == "approval" || it.Status == "approved" || it.Status == "scheduled" {
				rv, err := s.buildReview(ctx, ws, nil, c)
				if err != nil {
					continue
				}
				n := int64(len(rv.aud.list))
				it.Recipients = &n
				cost := rv.cost.Amount
				it.Cost = &cost
				route(n, 0)
				if it.Status != "scheduled" {
					warns, blocks := rv.counts()
					verdict, issues := "ready", int64(warns)
					switch {
					case blocks > 0:
						verdict, issues = "blocked", int64(blocks)
					case warns > 0:
						verdict = "warnings"
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
		route(st.planned, st.sent)
		cost := float64(int64(st.cost*100+0.5)) / 100
		it.Cost = &cost
		oo := optOuts[id]
		it.OptOuts = &oo
		if st.sent > 0 {
			d := pct(st.delivered, st.sent)
			it.Delivered = &d
		}
	}
}
