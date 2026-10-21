package texting

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// What MCP agents can do with texts: read the providers' overview and a
// campaign's report, and draft a text campaign. They never send one:
// sending needs sms:send, and a person sends from Review & send.

// AgentOverview is the SMS providers page for an agent: providers,
// senders and their registrations, routes by country, the month's spend
// against the budget. No phone numbers of contacts.
func (s *Service) AgentOverview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (map[string]any, error) {
	o, err := s.Overview(ctx, ws, p)
	if err != nil {
		return nil, err
	}
	provs := []map[string]any{}
	for _, pr := range o.Providers {
		provs = append(provs, map[string]any{"name": pr.Name, "adapter": pr.Adapter, "status": pr.Status, "detail": pr.Detail, "live": pr.Live,
			"countries": pr.Countries, "held_countries": pr.Held, "senders": pr.SenderCount})
	}
	senders := []map[string]any{}
	for _, sn := range o.Senders {
		senders = append(senders, map[string]any{"sender": sn.Label, "kind": sn.Kind, "provider": sn.Provider, "countries": sn.Countries,
			"registration": sn.Registration.Status, "can_send": sn.CanSend})
	}
	routes := []map[string]any{}
	for _, r := range o.Routes {
		routes = append(routes, map[string]any{"countries": r.Countries, "provider": r.Provider, "sender": r.Sender, "fallback": r.Fallback,
			"registration": r.Registration, "skipped": r.Skipped})
	}
	t := o.Tiles
	return map[string]any{"providers": provs, "senders": senders, "routes": routes, "other_countries": o.Other,
		"month": map[string]any{"starts": t.Month, "texts_sent": t.Sent, "delivered_pct": t.DeliveredPct, "failed": t.Failed, "spend_usd": t.Spend,
			"budget_usd": t.Budget, "held_by_budget": t.BudgetHeld},
		"suppressed_numbers": t.Suppressed,
		"note":               "Texts go only to contacts with SMS consent on a list, inside the quiet hours. Agents can draft text campaigns; a person with sms:send sends them."}, nil
}

// agentCampaign finds a text campaign by id or name.
func (s *Service) agentCampaign(ctx context.Context, ws *store.Workspace, ref string) (uuid.UUID, error) {
	ref = strings.TrimSpace(ref)
	if id, err := uuid.Parse(ref); err == nil {
		return id, nil
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM campaigns WHERE workspace_id = $1 AND channel = 'sms' AND lower(name) = lower($2) ORDER BY created_at DESC LIMIT 1`,
		ws.ID, ref).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("there is no text campaign named %q", ref)
	}
	return id, nil
}

// AgentReport is a text campaign's report for an agent: totals, countries,
// failures with the providers' codes, links and replies by kind. Counts
// only: no numbers or reply texts.
func (s *Service) AgentReport(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (map[string]any, string, string, error) {
	id, err := s.agentCampaign(ctx, ws, ref)
	if err != nil {
		return nil, "", "", err
	}
	r, err := s.Report(ctx, ws, p, id)
	if err != nil {
		return nil, "", "", err
	}
	t := r.Tiles
	countries := []map[string]any{}
	for _, c := range r.Countries {
		countries = append(countries, map[string]any{"country": c.Country, "sender": c.Sender, "sent": c.Sent, "delivered": c.Delivered, "failed": c.Failed,
			"clicks": c.Clicks, "opt_outs": c.OptOuts, "skipped": c.Skipped, "skip_reason": c.SkipReason, "cost_usd": c.Cost})
	}
	failures := []map[string]any{}
	for _, f := range r.Failures {
		codes := []map[string]any{}
		for _, c := range f.Codes {
			codes = append(codes, map[string]any{"provider": c.Provider, "code": c.Code, "count": c.Count})
		}
		failures = append(failures, map[string]any{"reason": f.Reason, "count": f.Count, "codes": codes})
	}
	links := []map[string]any{}
	for _, l := range r.Links {
		links = append(links, map[string]any{"url": l.URL, "people": l.People, "machine": l.Machine})
	}
	return map[string]any{"id": r.CampaignID, "name": r.Name, "status": r.Status, "phase": r.Phase, "started_at": r.StartedAt, "completed_at": r.CompletedAt,
		"recipients": r.Recipients, "sent": t.Sent, "delivered": t.Delivered, "delivered_pct": t.DeliveredPct, "failed": t.Failed, "skipped": t.Skipped,
		"clicks": t.Clicks, "click_pct": t.ClickPct, "opt_outs": t.OptOuts, "opt_out_pct": t.OptOutPct, "cost_usd": t.Cost, "countries": countries,
		"failures": failures, "links": links, "replies": r.ReplyCounts,
		"note": "Texts have no opens. Clicks are people who clicked, machine clicks (link previews) apart."}, r.CampaignID, r.Name, nil
}

// AgentDraft drafts a text campaign for an agent: a name, the text between
// the rules' brand prefix and opt-out text, and lists by name or id.
func (s *Service) AgentDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, name, body string, lists []string) (map[string]any, error) {
	ids := []string{}
	if len(lists) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT id::text, name FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL`, ws.ID)
		if err != nil {
			return nil, err
		}
		byName := map[string]string{}
		byID := map[string]bool{}
		for rows.Next() {
			var id, n string
			if rows.Scan(&id, &n) == nil {
				byName[strings.ToLower(n)] = id
				byID[id] = true
			}
		}
		rows.Close()
		for _, l := range lists {
			l = strings.TrimSpace(l)
			switch {
			case byID[l]:
				ids = append(ids, l)
			case byName[strings.ToLower(l)] != "":
				ids = append(ids, byName[strings.ToLower(l)])
			default:
				return nil, fmt.Errorf("there is no list named %q", l)
			}
		}
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("give the draft a name")
	}
	c, err := s.Create(ctx, ws, p, SmsCampaignCreate{Name: name, Lists: ids, Body: body})
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": c.ID, "name": c.Name, "status": c.Status, "characters": c.Counter.Characters, "segments": c.Counter.Segments,
		"encoding": c.Counter.Encoding, "message": "Draft created. A person reviews and sends it from ReputeCast; texts need sms:send."}, nil
}
