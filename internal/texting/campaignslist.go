package texting

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// FillChannelRows fills in the campaigns list's SMS rows (implements
// campaigns.ChannelRows): the providers and how many texts each takes
// ("Twilio 659 · Vonage 195"), the results once texts went out (delivered,
// clicks, opt-outs, cost), and the review's verdict before the send.
func (s *Service) FillChannelRows(ctx context.Context, ws *store.Workspace, items []campaigns.CampaignListItem) {
	ids := []uuid.UUID{}
	for _, it := range items {
		if it.Channel == "sms" {
			if id, err := uuid.Parse(it.ID); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	type stat struct {
		recipients, sent, delivered, clicks, optOuts int64
		cost                                         float64
		routes                                       []campaigns.RouteShare
	}
	stats := map[string]*stat{}
	rows, err := s.Pool.Query(ctx, `SELECT m.campaign_id, coalesce(p.id::text, ''), coalesce(p.name, ''), coalesce(p.adapter, ''),
			count(*) FILTER (WHERE m.status NOT IN ('skipped','cancelled')), count(*) FILTER (WHERE m.sent_at IS NOT NULL),
			count(*) FILTER (WHERE m.status = 'delivered'),
			coalesce(sum(m.cost) FILTER (WHERE m.status IN ('sending','sent','delivered','undelivered','failed')), 0)::float8
		FROM channel_messages m LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.campaign_id = ANY($1::uuid[]) AND m.kind = 'campaign'
		GROUP BY 1, 2, 3, 4 ORDER BY 1, min(p.connected_at)`, ids)
	if err != nil {
		s.log().WarnContext(ctx, "sms campaign rows", "err", err)
		return
	}
	for rows.Next() {
		var cid uuid.UUID
		var pid, name, adapter string
		var planned, sent, delivered int64
		var cost float64
		if rows.Scan(&cid, &pid, &name, &adapter, &planned, &sent, &delivered, &cost) != nil {
			continue
		}
		st, ok := stats[cid.String()]
		if !ok {
			st = &stat{}
			stats[cid.String()] = st
		}
		st.recipients += planned
		st.sent += sent
		st.delivered += delivered
		st.cost += cost
		if pid != "" && planned > 0 {
			st.routes = append(st.routes, campaigns.RouteShare{ProviderID: pid, Provider: name, Short: name, Adapter: adapter, Planned: planned, Sent: sent})
		}
	}
	rows.Close()
	rows, err = s.Pool.Query(ctx, `SELECT c.id, (SELECT count(DISTINCT k.contact_id) FROM sms_link_clicks k WHERE k.campaign_id = c.id AND NOT k.machine),
			(SELECT count(*) FROM sms_inbound i WHERE i.campaign_id = c.id AND i.kind = 'opt_out')
		FROM campaigns c WHERE c.id = ANY($1::uuid[])`, ids)
	if err == nil {
		for rows.Next() {
			var cid uuid.UUID
			var clicks, optOuts int64
			if rows.Scan(&cid, &clicks, &optOuts) == nil {
				if st, ok := stats[cid.String()]; ok {
					st.clicks, st.optOuts = clicks, optOuts
				}
			}
		}
		rows.Close()
	}
	for i := range items {
		it := &items[i]
		if it.Channel != "sms" {
			continue
		}
		id, _ := uuid.Parse(it.ID)
		c, err := s.Q.SmsCampaign(ctx, store.SmsCampaignParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			continue
		}
		it.Local = c.ScheduleMode == "local"
		if it.Local && c.LocalMinute != nil {
			it.LocalTime = fmt.Sprintf("%02d:%02d", *c.LocalMinute/60, *c.LocalMinute%60)
		}
		st, scheduled := stats[it.ID]
		if scheduled && st.recipients > 0 {
			n := st.recipients
			it.Recipients = &n
			for j := range st.routes {
				st.routes[j].Percent = pct(st.routes[j].Planned, st.recipients)
			}
			sort.SliceStable(st.routes, func(i, j int) bool { return st.routes[i].Planned > st.routes[j].Planned })
			it.Routes = st.routes
			if st.sent > 0 {
				it.Delivered = ptr(pct(st.delivered, st.sent))
				it.Clicks = ptr(pct(st.clicks, st.delivered))
				it.OptOuts = &st.optOuts
				it.Cost = ptr(round2(st.cost))
				it.HumanOpens = nil
			}
		}
		// Before the send: the review's verdict, and its plan until the
		// texts are queued.
		switch c.Status {
		case "draft", "approval", "approved", "scheduled":
		default:
			continue
		}
		rv, err := s.buildReview(ctx, ws, c)
		if err != nil {
			continue
		}
		_, warns, blocks := rv.counts()
		issues := int64(warns)
		if rv.verdict == VerdictBlocked {
			issues = int64(blocks)
		}
		it.Readiness = &campaigns.ReadinessSummary{Verdict: rv.verdict, Issues: issues}
		it.Blocked = rv.verdict == VerdictBlocked && c.Status == "draft"
		if scheduled && st.recipients > 0 {
			continue
		}
		n := rv.sent
		it.Recipients = &n
		byProv := map[uuid.UUID]*campaigns.RouteShare{}
		order := []uuid.UUID{}
		for _, pl := range rv.plan {
			if pl.decision.Sender == nil {
				continue
			}
			sn := pl.decision.Sender
			rs, ok := byProv[sn.ProviderID]
			if !ok {
				rs = &campaigns.RouteShare{ProviderID: sn.ProviderID.String(), Provider: sn.ProviderName, Short: sn.ProviderName, Adapter: sn.ProviderAdapter}
				byProv[sn.ProviderID] = rs
				order = append(order, sn.ProviderID)
			}
			rs.Planned++
		}
		it.Routes = []campaigns.RouteShare{}
		// The provider with the most texts first, as on Review & send.
		sort.SliceStable(order, func(i, j int) bool { return byProv[order[i]].Planned > byProv[order[j]].Planned })
		for _, pid := range order {
			rs := byProv[pid]
			rs.Percent = pct(rs.Planned, rv.sent)
			it.Routes = append(it.Routes, *rs)
		}
	}
}
