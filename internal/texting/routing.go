package texting

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// NANP is where alphanumeric sender IDs aren't supported.
var nanp = []string{"US", "CA"}

// graph is everything routing reads, loaded once per request or job.
type graph struct {
	providers map[uuid.UUID]store.SmsProvider
	senders   map[uuid.UUID]store.SmsSendersRow
	order     []uuid.UUID // senders in display order
	regs      map[uuid.UUID][]store.SmsRegistrationsRow
	routes    []store.SmsRoute
	prices    map[uuid.UUID]map[string]store.SmsPrice
	rules     Rules
}

func (s *Service) graph(ctx context.Context, q *store.Queries, ws *store.Workspace) (*graph, error) {
	g := &graph{providers: map[uuid.UUID]store.SmsProvider{}, senders: map[uuid.UUID]store.SmsSendersRow{}, regs: map[uuid.UUID][]store.SmsRegistrationsRow{},
		prices: map[uuid.UUID]map[string]store.SmsPrice{}}
	provs, err := q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range provs {
		g.providers[p.ID] = providerOf(p)
	}
	senders, err := q.SmsSenders(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, sn := range senders {
		g.senders[sn.ID] = sn
		g.order = append(g.order, sn.ID)
	}
	regs, err := q.SmsRegistrations(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, r := range regs {
		g.regs[r.SenderID] = append(g.regs[r.SenderID], r)
	}
	if g.routes, err = q.SmsRoutes(ctx, ws.ID); err != nil {
		return nil, err
	}
	prices, err := q.SmsPrices(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range prices {
		if g.prices[p.ProviderID] == nil {
			g.prices[p.ProviderID] = map[string]store.SmsPrice{}
		}
		g.prices[p.ProviderID][p.Country] = p
	}
	g.rules, err = s.rules(ctx, q, ws)
	return g, err
}

// providerOf turns the list row into the stored provider.
func providerOf(p store.SmsProvidersRow) store.SmsProvider {
	return store.SmsProvider{ID: p.ID, WorkspaceID: p.WorkspaceID, Adapter: p.Adapter, Name: p.Name, AccountLabel: p.AccountLabel, AccountHint: p.AccountHint,
		Config: p.Config, CredentialsEnc: p.CredentialsEnc, WebhookToken: p.WebhookToken, WebhookSecretEnc: p.WebhookSecretEnc, Webhooks: p.Webhooks,
		Status: p.Status, StatusDetail: p.StatusDetail, StatusSince: p.StatusSince, Health: p.Health, LastReceiptAt: p.LastReceiptAt,
		LastInboundAt: p.LastInboundAt, LastCheckedAt: p.LastCheckedAt, RegistrationsReadAt: p.RegistrationsReadAt, Demo: p.Demo,
		ConnectedBy: p.ConnectedBy, ConnectedAt: p.ConnectedAt, RemovedAt: p.RemovedAt}
}

// Reg is a sender's effective registration in one country.
type Reg struct {
	Status string
	// Kind is what the status comes from: toll_free, brand, campaign,
	// sender_id, or "" when nothing was recorded.
	Kind   string
	Name   string
	Reason string
	Since  *time.Time
	Source string
	// Brand and Campaign are a 10DLC number's two registrations.
	Brand, Campaign *store.SmsRegistrationsRow
}

// CanSend reports whether the registration lets the sender text.
func (r Reg) CanSend() bool { return sms.CanSend(r.Status) }

// reg works out a sender's registration for a country:
//   - toll-free numbers need toll-free verification, for the US and Canada;
//   - 10DLC numbers need an approved brand and campaign, for the US;
//   - sender IDs are registered country by country, and never work in the
//     US and Canada;
//   - other numbers send where they're from without registration.
func (g *graph) reg(sn store.SmsSendersRow, country string) Reg {
	regs := g.regs[sn.ID]
	find := func(kind, c string) *store.SmsRegistrationsRow {
		for i := range regs {
			if regs[i].Kind == kind && regs[i].Country == c {
				return &regs[i]
			}
		}
		return nil
	}
	of := func(r *store.SmsRegistrationsRow) Reg {
		if r == nil {
			return Reg{Status: "not_registered"}
		}
		return Reg{Status: r.Status, Kind: r.Kind, Name: r.Name, Reason: r.Reason, Since: r.StatusOn, Source: r.Source}
	}
	switch sn.Kind {
	case "sender_id":
		if slices.Contains(nanp, country) {
			out := Reg{Status: "not_supported", Kind: "sender_id", Source: "rule"}
			if r := find("sender_id", country); r != nil {
				out.Since = r.StatusOn
			}
			return out
		}
		r := find("sender_id", country)
		if r == nil && !slices.Contains(sn.Countries, country) {
			return Reg{Status: "not_registered", Kind: "sender_id"}
		}
		return of(r)
	case "toll_free":
		if !slices.Contains(sn.Countries, country) {
			return Reg{Status: "not_supported", Kind: "toll_free", Source: "rule"}
		}
		r := find("toll_free", "")
		if r == nil {
			r = find("toll_free", country)
		}
		return of(r)
	case "10dlc":
		if country != "US" {
			return Reg{Status: "not_supported", Kind: "campaign", Source: "rule"}
		}
		b, c := find("brand", ""), find("campaign", "")
		out := Reg{Brand: b, Campaign: c, Kind: "campaign"}
		switch {
		case b == nil || c == nil:
			out.Status = "not_registered"
		case b.Status == "rejected" || c.Status == "rejected":
			out.Status = "rejected"
			if c.Status == "rejected" {
				out.Reason, out.Since, out.Name = c.Reason, c.StatusOn, c.Name
			} else {
				out.Reason, out.Since, out.Name = b.Reason, b.StatusOn, b.Name
			}
		case b.Status == "approved" && c.Status == "approved":
			out.Status, out.Since, out.Name = "approved", c.StatusOn, c.Name
		case b.Status == "pending" || c.Status == "pending":
			out.Status, out.Since, out.Name = "pending", c.StatusOn, c.Name
		default:
			out.Status = "not_registered"
		}
		if c != nil {
			out.Source = c.Source
		}
		return out
	}
	if r := find(sn.Kind, country); r != nil {
		return of(r)
	}
	if slices.Contains(sn.Countries, country) {
		return Reg{Status: "allowed", Source: "rule"}
	}
	return Reg{Status: "not_supported", Source: "rule"}
}

// Decision is where a text to a country goes.
type Decision struct {
	Country string
	// Route is the route that covers the country; nil for "every other
	// country".
	Route *store.SmsRoute
	// Sender is who sends, nil when nobody can: Hold says why.
	Sender *store.SmsSendersRow
	// Fallback takes over when the sender's provider is down; nil is
	// "None: hold and notify".
	Fallback *store.SmsSendersRow
	Reg      Reg
	// Price is the price of one segment; 0 when unknown.
	Price   float64
	PriceOn *time.Time
	// Hold is why nothing can send: no_route (every other country: don't
	// send), or the sender's registration (pending, rejected,
	// not_registered, not_supported).
	Hold string
}

// routeFor is the route covering a country.
func (g *graph) routeFor(country string) *store.SmsRoute {
	for i := range g.routes {
		if slices.Contains(g.routes[i].Countries, country) {
			return &g.routes[i]
		}
	}
	if g.rules.OtherCountries == "route" && g.rules.OtherRouteID != nil {
		for i := range g.routes {
			if g.routes[i].ID == *g.rules.OtherRouteID {
				return &g.routes[i]
			}
		}
	}
	return nil
}

// decide routes a country.
func (g *graph) decide(country string) Decision {
	d := Decision{Country: country}
	r := g.routeFor(country)
	if r == nil {
		d.Hold = "no_route"
		return d
	}
	d.Route = r
	sn, ok := g.senders[r.SenderID]
	if !ok {
		d.Hold = "no_route"
		return d
	}
	d.Reg = g.reg(sn, country)
	if r.FallbackSenderID != nil {
		if fb, ok := g.senders[*r.FallbackSenderID]; ok && g.reg(fb, country).CanSend() {
			d.Fallback = &fb
		}
	}
	if p, ok := g.prices[sn.ProviderID][country]; ok {
		d.Price, d.PriceOn = p.Price, &p.AsOf
	}
	if !d.Reg.CanSend() {
		d.Hold = d.Reg.Status
		return d
	}
	d.Sender = &sn
	return d
}

// down reports whether a provider can't take texts now (an outage or
// rejected credentials): its senders fail over or hold.
func (g *graph) down(providerID uuid.UUID) bool {
	p, ok := g.providers[providerID]
	return ok && (p.Status == "down" || p.Status == "credentials_rejected")
}

// senderLabel is how people see a sender: "+1 888 555 0142" or "ACME".
func senderLabel(sn store.SmsSendersRow) string {
	if strings.HasPrefix(sn.Value, "+") {
		return displayPhone(sn.Value)
	}
	return sn.Value
}

// detailsOf reads a registration's submitted details.
func detailsOf(r store.SmsRegistrationsRow) map[string]string {
	out := map[string]string{}
	_ = json.Unmarshal(r.Details, &out)
	return out
}
