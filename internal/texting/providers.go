package texting

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsCatalogItem is an SMS adapter in the connect wizard's catalog.
type SmsCatalogItem struct {
	Adapter  string      `json:"adapter"`
	Name     string      `json:"name"`
	Summary  string      `json:"summary"`
	Badges   []sms.Badge `json:"badges"`
	Fields   []sms.Field `json:"fields"`
	Webhooks string      `json:"webhooks" enum:"registered,manual"`
	Version  string      `json:"version"`
	// Connected counts this workspace's providers of this adapter.
	Connected int `json:"connected"`
}

// Catalog lists the SMS adapters this build has.
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) ([]SmsCatalogItem, error) {
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []SmsCatalogItem{}
	for _, d := range sms.Catalog() {
		// The manifest's texts in the reader's language (internal/sms, text.go).
		d.Manifest = d.In(i18n.LangFrom(ctx))
		it := SmsCatalogItem{Adapter: d.Adapter, Name: d.Name, Summary: d.Summary, Badges: d.Badges, Fields: d.Fields, Webhooks: d.Webhooks, Version: d.Version}
		if it.Badges == nil {
			it.Badges = []sms.Badge{}
		}
		if it.Fields == nil {
			it.Fields = []sms.Field{}
		}
		for _, p := range provs {
			if p.Adapter == d.Adapter {
				it.Connected++
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// SmsOverviewTiles are the tiles above the provider cards.
type SmsOverviewTiles struct {
	Month time.Time `json:"month"`
	Sent  int64     `json:"sent"`
	// ByKind counts this month's texts: campaign, automation, confirmation,
	// test, reply.
	ByKind       map[string]int64 `json:"byKind"`
	Delivered    int64            `json:"delivered"`
	Failed       int64            `json:"failed"`
	DeliveredPct float64          `json:"deliveredPct"`
	Spend        float64          `json:"spend"`
	Budget       float64          `json:"budget"`
	AlertPct     int              `json:"alertPct"`
	// SpendBy is each provider's share of the month's spend.
	SpendBy    []SmsProviderSpend `json:"spendBy"`
	Suppressed int64              `json:"suppressed"`
	// SuppressedBy counts the list by reason.
	SuppressedBy map[string]int64 `json:"suppressedBy"`
	// BudgetHeld counts the texts waiting for the budget to be raised.
	BudgetHeld int64 `json:"budgetHeld"`
}

// SmsProviderSpend is a provider's spend this month.
type SmsProviderSpend struct {
	Name  string  `json:"name"`
	Spend float64 `json:"spend"`
}

// SmsProviderCard is a connected provider.
type SmsProviderCard struct {
	ID      string `json:"id"`
	Adapter string `json:"adapter"`
	Name    string `json:"name"`
	Status  string `json:"status" enum:"healthy,degraded,down,credentials_rejected"`
	// Detail qualifies Degraded: receipts_silent.
	Detail       string     `json:"detail,omitempty"`
	Since        *time.Time `json:"since,omitempty"`
	AccountLabel string     `json:"accountLabel"`
	Live         bool       `json:"live"`
	// TestMode is true when the adapter has test credentials: the card then
	// says whether these are live.
	TestMode    bool   `json:"testMode"`
	AccountHint string `json:"accountHint"`
	// SecretLabel names the secret fields kept ("API key", "signature
	// secret"); they're never shown.
	SecretLabels []string `json:"secretLabels"`
	// Countries are the routes it serves; Held the countries whose route
	// it holds (a pending registration).
	Countries []string `json:"countries"`
	Held      []string `json:"held"`
	// Senders sums up its senders.
	SenderKinds []string `json:"senderKinds"`
	SenderCount int      `json:"senderCount"`
	// SenderIDs and SenderIDCountries: its alphanumeric sender IDs.
	SenderIDs         []string    `json:"senderIds"`
	SenderIDCountries int         `json:"senderIdCountries"`
	Webhooks          string      `json:"webhooks" enum:"registered,manual"`
	WebhookURL        string      `json:"webhookUrl"`
	ConnectedAt       time.Time   `json:"connectedAt"`
	ConnectedBy       string      `json:"connectedBy"`
	LastReceiptAt     *time.Time  `json:"lastReceiptAt,omitempty"`
	LastCheckedAt     *time.Time  `json:"lastCheckedAt,omitempty"`
	FailedPct         float64     `json:"failedPct"`
	FilteredPct       float64     `json:"filteredPct"`
	PauseFailed       float64     `json:"pauseFailed"`
	PauseFiltered     float64     `json:"pauseFiltered"`
	Spend             float64     `json:"spend"`
	PriceMin          float64     `json:"priceMin"`
	PriceMax          float64     `json:"priceMax"`
	PriceCountries    []string    `json:"priceCountries"`
	Badges            []sms.Badge `json:"badges"`
	Demo              bool        `json:"demo"`
	// Answer is the provider's answer to the last failed call; Held the
	// texts waiting for it; SentSince and ReceiptsSince the texts and
	// receipts since receipts went silent.
	Answer        string   `json:"answer,omitempty"`
	HeldTexts     int64    `json:"heldTexts"`
	SentSince     int64    `json:"sentSince"`
	ReceiptsSince int64    `json:"receiptsSince"`
	Told          []string `json:"told"`
	Console       string   `json:"console"`
	OptOutSync    string   `json:"optOutSync" enum:"two_way,read_only,off"`
}

// SmsRegistrationView is a sender's registration in one country (or for
// all of its countries).
type SmsRegistrationView struct {
	Kind    string     `json:"kind" enum:"toll_free,brand,campaign,sender_id,"`
	Country string     `json:"country"`
	Name    string     `json:"name,omitempty"`
	Status  string     `json:"status" enum:"not_registered,pending,verified,approved,rejected,allowed,not_supported"`
	Reason  string     `json:"reason,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
	// Source is where the status comes from: api (read from the provider),
	// manual (set by hand) or rule (sender IDs in the US and Canada).
	Source string     `json:"source" enum:"api,manual,rule,"`
	SetBy  string     `json:"setBy,omitempty"`
	ReadAt *time.Time `json:"readAt,omitempty"`
}

// SmsSenderRow is a row of the senders table: a sender, or a sender ID's
// countries that share one registration status.
type SmsSenderRow struct {
	ID         string   `json:"id"`
	Value      string   `json:"value"`
	Label      string   `json:"label"`
	Kind       string   `json:"kind" enum:"toll_free,10dlc,local,mobile,sender_id"`
	ProviderID string   `json:"providerId"`
	Provider   string   `json:"provider"`
	Adapter    string   `json:"adapter"`
	Countries  []string `json:"countries"`
	// Registration is the row's status; Brand and Campaign a 10DLC
	// number's two parts.
	Registration SmsRegistrationView  `json:"registration"`
	Brand        *SmsRegistrationView `json:"brand,omitempty"`
	Campaign     *SmsRegistrationView `json:"campaign,omitempty"`
	Throughput   float64              `json:"throughput"`
	// Routes are the routes that use it, by their countries; Held is true
	// when they hold its texts (registration pending).
	Routes [][]string `json:"routes"`
	Held   bool       `json:"held"`
	// Manual is true when the status is set by hand here.
	Manual bool `json:"manual"`
	// CanSend is false when its registration stops it.
	CanSend bool `json:"canSend"`
}

// SmsRouteSummary is a route as the providers page sums it up.
type SmsRouteSummary struct {
	ID        string   `json:"id"`
	Countries []string `json:"countries"`
	Provider  string   `json:"provider"`
	Sender    string   `json:"sender"`
	// Fallback is the fallback sender ("" is "None: hold and notify").
	Fallback         string `json:"fallback"`
	FallbackProvider string `json:"fallbackProvider"`
	// Registration is the sender's status there; Skipped is true when it
	// holds every text (pending, rejected).
	Registration string `json:"registration"`
	Skipped      bool   `json:"skipped"`
}

// SmsOverview is Sending providers › SMS › Providers.
type SmsOverview struct {
	Tiles        SmsOverviewTiles  `json:"tiles"`
	Providers    []SmsProviderCard `json:"providers"`
	NotConnected []SmsCatalogItem  `json:"notConnected"`
	Senders      []SmsSenderRow    `json:"senders"`
	Routes       []SmsRouteSummary `json:"routes"`
	// Other is "every other country": dont_send or route.
	Other       string     `json:"other" enum:"dont_send,route"`
	OtherSender string     `json:"otherSender,omitempty"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	CanManage   bool       `json:"canManage"`
	Helpers     []string   `json:"helpers"`
	Timezone    string     `json:"timezone"`
}

// Overview builds the providers page.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SmsOverview, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsOverview{}, err
	}
	loc := location(ws.Timezone)
	now := s.now()
	month := monthStart(now, loc)
	out := SmsOverview{Providers: []SmsProviderCard{}, NotConnected: []SmsCatalogItem{}, Senders: []SmsSenderRow{}, Routes: []SmsRouteSummary{},
		Other: g.rules.OtherCountries, CanManage: p.Can(authz.ProvidersManage), Helpers: []string{}, Timezone: ws.Timezone}
	if !out.CanManage {
		out.Helpers = access.Helpers(ctx, s.Q, ws.ID, authz.ProvidersManage)
	}
	// Tiles.
	t := SmsOverviewTiles{Month: month, ByKind: map[string]int64{}, SpendBy: []SmsProviderSpend{}, SuppressedBy: map[string]int64{}, Budget: g.rules.Budget,
		AlertPct: g.rules.BudgetAlertPct}
	rows, err := s.Pool.Query(ctx, `SELECT kind, count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status = 'delivered'),
			count(*) FILTER (WHERE status IN ('undelivered','failed') AND sent_at IS NOT NULL)
		FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms' AND kind <> 'reply' AND coalesce(sent_at, created_at) >= $2 GROUP BY kind`, ws.ID, month)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var kind string
		var sent, delivered, failed int64
		if err := rows.Scan(&kind, &sent, &delivered, &failed); err != nil {
			rows.Close()
			return out, err
		}
		if sent > 0 {
			t.ByKind[kind] = sent
		}
		t.Sent += sent
		t.Delivered += delivered
		t.Failed += failed
	}
	rows.Close()
	t.DeliveredPct = pct(t.Delivered, t.Sent)
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms' AND status = 'held' AND reason = 'budget'`,
		ws.ID).Scan(&t.BudgetHeld)
	if t.Spend, err = s.spentThisMonth(ctx, ws); err != nil {
		return out, err
	}
	t.Spend = round2(t.Spend)
	spendBy := map[uuid.UUID]float64{}
	rows, err = s.Pool.Query(ctx, `SELECT provider_id, coalesce(sum(cost), 0)::float8 FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms'
		AND status IN ('sending','sent','delivered','undelivered','failed') AND coalesce(sent_at, created_at) >= $2 AND provider_id IS NOT NULL GROUP BY provider_id`, ws.ID, month)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id uuid.UUID
		var v float64
		if err := rows.Scan(&id, &v); err != nil {
			rows.Close()
			return out, err
		}
		spendBy[id] = v
	}
	rows.Close()
	counts, err := s.Q.ChanSuppressionCounts(ctx, store.ChanSuppressionCountsParams{WorkspaceID: ws.ID, Channel: "sms"})
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		t.SuppressedBy[c.Reason] = c.N
		t.Suppressed += c.N
	}
	// Provider cards.
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	syncs, _ := s.Q.SmsOptOutSync(ctx, ws.ID)
	weekAgo := now.AddDate(0, 0, -7)
	for _, pr := range provs {
		d, _ := sms.Lookup(pr.Adapter)
		d.Manifest = d.In(i18n.LangFrom(ctx))
		h := healthOf(providerOf(pr))
		c := SmsProviderCard{ID: pr.ID.String(), Adapter: pr.Adapter, Name: pr.Name, Status: pr.Status, Detail: pr.StatusDetail, Since: pr.StatusSince,
			AccountLabel: pr.AccountLabel, Live: settingsOf(providerOf(pr))["mode"] != "test", AccountHint: pr.AccountHint, SecretLabels: []string{},
			Countries: []string{}, Held: []string{}, SenderKinds: []string{}, SenderIDs: []string{}, Webhooks: pr.Webhooks, WebhookURL: s.hookURL(providerOf(pr)),
			ConnectedAt: pr.ConnectedAt, ConnectedBy: pr.ConnectedByName, LastReceiptAt: pr.LastReceiptAt, LastCheckedAt: pr.LastCheckedAt,
			PauseFailed: g.rules.PauseFailed, PauseFiltered: g.rules.PauseFiltered, Spend: round2(spendBy[pr.ID]), PriceCountries: []string{}, Badges: d.Badges,
			Demo: pr.Demo, Answer: answerIn(i18n.LangFrom(ctx), h.Answer), Told: []string{}, OptOutSync: map[sms.Support]string{sms.Yes: "two_way", sms.Partial: "read_only", sms.No: "off"}[d.Has(sms.CapOptOutSync)]}
		if c.Badges == nil {
			c.Badges = []sms.Badge{}
		}
		if d.Console != nil {
			c.Console = d.Console("", "")
		}
		for _, f := range d.Fields {
			if f.Secret {
				c.SecretLabels = append(c.SecretLabels, f.Label)
			}
			if f.Key == "mode" {
				c.TestMode = true
			}
		}
		if out.CheckedAt == nil || (pr.LastCheckedAt != nil && pr.LastCheckedAt.After(*out.CheckedAt)) {
			out.CheckedAt = pr.LastCheckedAt
		}
		var sent, failed, filtered int64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL), count(*) FILTER (WHERE status IN ('undelivered','failed') AND reason <> 'filtered'),
			count(*) FILTER (WHERE reason = 'filtered') FROM channel_messages WHERE provider_id = $1 AND created_at >= $2`, pr.ID, weekAgo).Scan(&sent, &failed, &filtered)
		c.FailedPct, c.FilteredPct = pct(failed, sent), pct(filtered, sent)
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE provider_id = $1 AND status = 'held' AND reason = 'provider_down'`, pr.ID).Scan(&c.HeldTexts)
		if h.SilentSince != nil {
			_ = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at > $2), count(*) FILTER (WHERE delivered_at > $2 OR failed_at > $2)
				FROM channel_messages WHERE provider_id = $1`, pr.ID, *h.SilentSince).Scan(&c.SentSince, &c.ReceiptsSince)
		}
		if pr.Status != "healthy" {
			c.Told = access.Helpers(ctx, s.Q, ws.ID, authz.ProvidersManage)
		}
		for _, sn := range g.senders {
			if sn.ProviderID != pr.ID {
				continue
			}
			c.SenderCount++
			if !slices.Contains(c.SenderKinds, sn.Kind) {
				c.SenderKinds = append(c.SenderKinds, sn.Kind)
			}
			if sn.Kind == "sender_id" {
				c.SenderIDs = append(c.SenderIDs, sn.Value)
				c.SenderIDCountries += len(sn.Countries)
			}
		}
		sort.Strings(c.SenderKinds)
		for _, r := range g.routes {
			sn, ok := g.senders[r.SenderID]
			if !ok || sn.ProviderID != pr.ID {
				continue
			}
			for _, country := range r.Countries {
				if g.reg(sn, country).CanSend() {
					c.Countries = append(c.Countries, country)
				} else {
					c.Held = append(c.Held, country)
				}
			}
		}
		first := true
		for country, price := range g.prices[pr.ID] {
			if !slices.Contains(c.Countries, country) && !slices.Contains(c.Held, country) {
				continue
			}
			if first || price.Price < c.PriceMin {
				c.PriceMin = price.Price
			}
			if first || price.Price > c.PriceMax {
				c.PriceMax = price.Price
			}
			first = false
			c.PriceCountries = append(c.PriceCountries, country)
		}
		sort.Strings(c.PriceCountries)
		for _, sy := range syncs {
			if sy.ProviderID == pr.ID && sy.Mode != "" {
				c.OptOutSync = sy.Mode
			}
		}
		if !out.CanManage {
			// Viewers see providers, senders and costs, never the account's
			// identifiers or the webhook address (it carries its token).
			c.AccountHint, c.WebhookURL = "", ""
		}
		out.Providers = append(out.Providers, c)
		t.SpendBy = append(t.SpendBy, SmsProviderSpend{Name: pr.Name, Spend: c.Spend})
	}
	out.Tiles = t
	cat, err := s.Catalog(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, it := range cat {
		if it.Connected == 0 {
			out.NotConnected = append(out.NotConnected, it)
		}
	}
	out.Senders = s.senderRows(g)
	out.Routes = s.routeSummaries(g)
	if g.rules.OtherCountries == "route" && g.rules.OtherRouteID != nil {
		for _, r := range g.routes {
			if r.ID == *g.rules.OtherRouteID {
				if sn, ok := g.senders[r.SenderID]; ok {
					out.OtherSender = sn.ProviderName + " · " + senderLabel(sn)
				}
			}
		}
	}
	return out, nil
}

func regView(r Reg) SmsRegistrationView {
	return SmsRegistrationView{Kind: r.Kind, Status: r.Status, Name: r.Name, Reason: r.Reason, Since: r.Since, Source: r.Source}
}

func regRowView(r *store.SmsRegistrationsRow) *SmsRegistrationView {
	if r == nil {
		return nil
	}
	return &SmsRegistrationView{Kind: r.Kind, Country: r.Country, Name: r.Name, Status: r.Status, Reason: r.Reason, Since: r.StatusOn, Source: r.Source,
		SetBy: r.SetByName, ReadAt: r.ReadAt}
}

// senderRows is the senders table: a sender ID's countries are grouped by
// their registration status.
func (s *Service) senderRows(g *graph) []SmsSenderRow {
	out := []SmsSenderRow{}
	for _, id := range g.order {
		sn := g.senders[id]
		d, _ := sms.Lookup(sn.ProviderAdapter)
		manual := d.Has(sms.CapRegistration) != sms.Yes
		routesOf := func(countries []string) ([][]string, bool) {
			rs := [][]string{}
			held := false
			for _, r := range g.routes {
				if r.SenderID != sn.ID {
					continue
				}
				cs := []string{}
				for _, c := range r.Countries {
					if len(countries) == 0 || slices.Contains(countries, c) {
						cs = append(cs, c)
						if !g.reg(sn, c).CanSend() {
							held = true
						}
					}
				}
				if len(cs) > 0 {
					rs = append(rs, cs)
				}
			}
			return rs, held
		}
		base := SmsSenderRow{ID: sn.ID.String(), Value: sn.Value, Label: senderLabel(sn), Kind: sn.Kind, ProviderID: sn.ProviderID.String(), Provider: sn.ProviderName,
			Adapter: sn.ProviderAdapter, Throughput: sn.Throughput, Manual: manual}
		if sn.Kind != "sender_id" {
			row := base
			row.Countries = append([]string{}, sn.Countries...)
			country := ""
			if len(sn.Countries) > 0 {
				country = sn.Countries[0]
			}
			if sn.Kind == "10dlc" {
				country = "US"
			}
			r := g.reg(sn, country)
			row.Registration = regView(r)
			row.Brand, row.Campaign = regRowView(r.Brand), regRowView(r.Campaign)
			row.CanSend = r.CanSend()
			row.Routes, row.Held = routesOf(nil)
			if sn.Kind == "local" || sn.Kind == "mobile" {
				row.Manual = false
			}
			out = append(out, row)
			continue
		}
		// Sender IDs: one row per registration status.
		type grp struct {
			reg       Reg
			countries []string
		}
		groups := []*grp{}
		for _, c := range sn.Countries {
			r := g.reg(sn, c)
			var found *grp
			for _, gr := range groups {
				if gr.reg.Status == r.Status && (r.Status == "allowed" || gr.reg.Since == nil && r.Since == nil || gr.reg.Since != nil && r.Since != nil && gr.reg.Since.Equal(*r.Since)) {
					found = gr
					break
				}
			}
			if found == nil {
				found = &grp{reg: r}
				groups = append(groups, found)
			}
			found.countries = append(found.countries, c)
		}
		for _, gr := range groups {
			row := base
			row.Countries = gr.countries
			row.Registration = regView(gr.reg)
			if len(gr.countries) == 1 {
				row.Registration.Country = gr.countries[0]
			}
			row.CanSend = gr.reg.CanSend()
			row.Routes, row.Held = routesOf(gr.countries)
			out = append(out, row)
		}
	}
	return out
}

// routeSummaries sums the routes up: a route whose sender is pending in
// some of its countries shows those apart ("Ireland · skipped until ACME is
// approved").
func (s *Service) routeSummaries(g *graph) []SmsRouteSummary {
	out := []SmsRouteSummary{}
	for _, r := range g.routes {
		sn, ok := g.senders[r.SenderID]
		if !ok {
			continue
		}
		base := SmsRouteSummary{ID: r.ID.String(), Provider: sn.ProviderName, Sender: senderLabel(sn)}
		if r.FallbackSenderID != nil {
			if fb, ok := g.senders[*r.FallbackSenderID]; ok {
				base.Fallback, base.FallbackProvider = senderLabel(fb), fb.ProviderName
			}
		}
		ok1, held := []string{}, map[string][]string{}
		heldOrder := []string{}
		for _, c := range r.Countries {
			reg := g.reg(sn, c)
			if reg.CanSend() {
				ok1 = append(ok1, c)
				continue
			}
			if _, seen := held[reg.Status]; !seen {
				heldOrder = append(heldOrder, reg.Status)
			}
			held[reg.Status] = append(held[reg.Status], c)
		}
		if len(ok1) > 0 {
			x := base
			x.Countries = ok1
			x.Registration = g.reg(sn, ok1[0]).Status
			out = append(out, x)
		}
		for _, st := range heldOrder {
			x := base
			x.Countries = held[st]
			x.Registration, x.Skipped = st, true
			out = append(out, x)
		}
	}
	return out
}

// SmsConnectInput connects an SMS provider: the wizard's second step.
type SmsConnectInput struct {
	Adapter string `json:"adapter" minLength:"1"`
	Name    string `json:"name,omitempty" maxLength:"60"`
	// Fields are the connection form's values, by key; secret ones are
	// sealed and never returned.
	Fields map[string]string `json:"fields"`
	// SetupID is the wizard's setup, when it started one.
	SetupID string `json:"setupId,omitempty"`
}

// SmsWebhookSetup is how receipts and inbound texts reach ReputeCast.
type SmsWebhookSetup struct {
	Mode string `json:"mode" enum:"registered,manual"`
	URL  string `json:"url"`
	// Secret is shown once, in manual mode, to paste in the provider's
	// console; ReputeCast keeps it sealed.
	Secret string `json:"secret,omitempty"`
	// Error is why registering through the API failed.
	Error string `json:"error,omitempty"`
}

// SmsFoundSender is a number found in the account.
type SmsFoundSender struct {
	Kind         string              `json:"kind" enum:"toll_free,10dlc,local,mobile,sender_id"`
	Value        string              `json:"value"`
	Label        string              `json:"label"`
	Countries    []string            `json:"countries"`
	Throughput   float64             `json:"throughput"`
	ProviderRef  string              `json:"providerRef"`
	Registration SmsRegistrationView `json:"registration"`
	Imported     bool                `json:"imported"`
}

// SmsConnectResult is the wizard after the connection step.
type SmsConnectResult struct {
	ProviderID string           `json:"providerId"`
	SetupID    string           `json:"setupId"`
	Account    sms.Account      `json:"account"`
	Webhooks   SmsWebhookSetup  `json:"webhooks"`
	Senders    []SmsFoundSender `json:"senders"`
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Connect checks the credentials, registers the webhooks where the API
// allows it, and lists the senders in the account. The credentials are
// sealed and never returned. Audited.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsConnectInput) (SmsConnectResult, error) {
	d, ok := sms.Lookup(in.Adapter)
	if !ok {
		return SmsConnectResult{}, api.Invalid("body.adapter", say(ctx, mPickCatalog, nil))
	}
	settings, creds := map[string]string{}, map[string]string{}
	for _, f := range d.Fields {
		v := strings.TrimSpace(in.Fields[f.Key])
		if v == "" {
			v = f.Default
		}
		if f.Required && v == "" {
			return SmsConnectResult{}, api.Invalid("body.fields."+f.Key, say(ctx, mFieldRequired, i18n.Args{"field": sms.TextIn(f.Label, f.LabelText, i18n.LangFrom(ctx))}))
		}
		if v == "" {
			continue
		}
		if f.Secret {
			creds[f.Key] = v
		} else {
			settings[f.Key] = v
		}
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	token := randomToken(16)
	cfg := sms.Config{ProviderID: id, WorkspaceID: ws.ID, Settings: settings, Credentials: creds, HTTP: s.HTTP, Now: s.now}
	prov, err := d.New(cfg)
	if err != nil {
		return SmsConnectResult{}, api.Invalid("body.fields", err.Error())
	}
	acct, err := prov.Check(ctx)
	if err != nil {
		if errors.Is(err, sms.ErrCredentialsRejected) {
			return SmsConnectResult{}, api.Err(http.StatusUnprocessableEntity, "credentials_rejected", say(ctx, mCredentialsRejected, i18n.Args{"provider": d.Name}))
		}
		return SmsConnectResult{}, api.Err(http.StatusBadGateway, "provider_unreachable",
			say(ctx, mNoAnswerRetry, i18n.Args{"provider": d.Name, "answer": answerIn(i18n.LangFrom(ctx), answerOf(err))}))
	}
	stub := store.SmsProvider{ID: id, Adapter: d.Adapter, WebhookToken: token}
	hooks := SmsWebhookSetup{Mode: "registered", URL: s.hookURL(stub)}
	var found []sms.Sender
	if sl, ok := prov.(sms.SenderLister); ok {
		if found, err = sl.Senders(ctx); err != nil {
			found = nil
		}
	}
	secret := ""
	if reg, ok := prov.(sms.WebhookRegistrar); ok {
		setup, err := reg.RegisterWebhooks(ctx, sms.Webhooks{StatusURL: hooks.URL, InboundURL: hooks.URL}, found)
		if err != nil {
			hooks.Mode, hooks.Error = "manual", answerIn(i18n.LangFrom(ctx), answerOf(err))
		} else {
			secret = setup.Secret
			for k, v := range setup.Settings {
				settings[k] = v
			}
		}
	} else {
		hooks.Mode = "manual"
	}
	if hooks.Mode == "manual" && secret == "" {
		secret = "whsec_" + randomToken(24)
		hooks.Secret = secret
	}
	credsJSON, _ := json.Marshal(creds)
	sealed, err := s.Box.Seal(credsJSON, credentialsAD(id))
	if err != nil {
		return SmsConnectResult{}, err
	}
	var secretSealed []byte
	if secret != "" {
		if secretSealed, err = s.Box.Seal([]byte(secret), secretAD(id)); err != nil {
			return SmsConnectResult{}, err
		}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = d.Name
	}
	label := acct.Name
	setupID := uuid.Must(uuid.NewV7())
	if in.SetupID != "" {
		if sid, err := uuid.Parse(in.SetupID); err == nil {
			setupID = sid
		}
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsProviderInsert(ctx, store.SmsProviderInsertParams{ID: id, WorkspaceID: ws.ID, Adapter: d.Adapter, Name: name, AccountLabel: label,
			AccountHint: acct.Hint, Config: jsonOf(settings), CredentialsEnc: sealed, WebhookToken: token, WebhookSecretEnc: secretSealed, Webhooks: hooks.Mode,
			ConnectedAt: &now, ConnectedBy: uuidPtr(p.UserID)}); err != nil {
			return err
		}
		if _, err := q.SmsSetup(ctx, store.SmsSetupParams{ID: setupID, WorkspaceID: ws.ID}); notFound(err) {
			if err := q.SmsSetupInsert(ctx, store.SmsSetupInsertParams{ID: setupID, WorkspaceID: ws.ID, Adapter: d.Adapter, CreatedBy: uuidPtr(p.UserID), At: now}); err != nil {
				return err
			}
		}
		if err := q.SmsSetupUpdate(ctx, store.SmsSetupUpdateParams{Step: "senders", Status: "open", State: jsonOf(map[string]any{"providerId": id.String()}),
			ProviderID: &id, At: now, ID: setupID}); err != nil {
			return err
		}
		if err := s.ensureRules(ctx, q, ws); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.connected", TargetType: "sms_provider", TargetID: id.String(), TargetName: name,
			After: map[string]any{"adapter": d.Adapter, "account": label, "hint": acct.Hint, "webhooks": hooks.Mode, "live": acct.Live}})
	})
	if err != nil {
		return SmsConnectResult{}, err
	}
	out := SmsConnectResult{ProviderID: id.String(), SetupID: setupID.String(), Account: acct, Webhooks: hooks, Senders: []SmsFoundSender{}}
	if p, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID}); err == nil {
		out.Senders = s.foundSenders(ctx, p, found)
	}
	return out, nil
}

// foundSenders lists the account's numbers with their registration as the
// provider reports it.
func (s *Service) foundSenders(ctx context.Context, p store.SmsProvider, found []sms.Sender) []SmsFoundSender {
	out := []SmsFoundSender{}
	regs := map[string]sms.Registration{}
	if ad, err := s.open(p); err == nil {
		if rr, ok := ad.(sms.RegistrationReader); ok {
			if list, err := rr.Registrations(ctx, found); err == nil {
				for _, r := range list {
					cur, ok := regs[r.Sender]
					// A 10DLC number shows its campaign over its brand.
					if !ok || r.Kind == "campaign" || cur.Status == "approved" && r.Status != "approved" {
						regs[r.Sender] = r
					}
				}
			}
		}
	}
	existing, _ := s.Q.SmsSenders(ctx, p.WorkspaceID)
	for _, f := range found {
		fs := SmsFoundSender{Kind: f.Kind, Value: f.Value, Label: f.Value, Countries: f.Countries, Throughput: f.Throughput, ProviderRef: f.ProviderRef,
			Registration: SmsRegistrationView{Status: "not_registered"}}
		if strings.HasPrefix(f.Value, "+") {
			fs.Label = displayPhone(f.Value)
		}
		if fs.Countries == nil {
			fs.Countries = []string{}
		}
		if r, ok := regs[f.Value]; ok {
			fs.Registration = SmsRegistrationView{Kind: r.Kind, Country: r.Country, Name: r.Name, Status: r.Status, Reason: r.Reason, Since: r.Since, Source: "api"}
		} else if f.Kind == "local" || f.Kind == "mobile" {
			fs.Registration.Status = "allowed"
		}
		for _, e := range existing {
			if e.ProviderID == p.ID && e.Value == f.Value {
				fs.Imported = true
			}
		}
		out = append(out, fs)
	}
	return out
}

// SmsFoundSenderList is the numbers in a provider's account.
type SmsFoundSenderList struct {
	Items []SmsFoundSender `json:"items"`
}

// AccountNumbers lists the numbers in a provider's account with their
// registration, marking the ones already imported (Import from providers).
func (s *Service) AccountNumbers(ctx context.Context, ws *store.Workspace, providerID uuid.UUID) (SmsFoundSenderList, error) {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: providerID, WorkspaceID: ws.ID})
	if err != nil {
		return SmsFoundSenderList{}, api.MapError(err, "SMS provider")
	}
	ad, err := s.open(prov)
	if err != nil {
		return SmsFoundSenderList{}, err
	}
	sl, ok := ad.(sms.SenderLister)
	if !ok {
		return SmsFoundSenderList{}, api.Conflict(say(ctx, mNoNumberList, i18n.Args{"provider": prov.Name}))
	}
	found, err := sl.Senders(ctx)
	if err != nil {
		return SmsFoundSenderList{}, api.Err(http.StatusBadGateway, "provider_unreachable", noAnswer(ctx, prov.Name, err))
	}
	return SmsFoundSenderList{Items: s.foundSenders(ctx, prov, found)}, nil
}

// SmsImportInput imports numbers found in the account.
type SmsImportInput struct {
	Senders []SmsImportSender `json:"senders" minItems:"1" maxItems:"50"`
	SetupID string            `json:"setupId,omitempty"`
}

// SmsImportSender is one number to import, with its countries.
type SmsImportSender struct {
	Value     string   `json:"value" minLength:"1"`
	Countries []string `json:"countries,omitempty" maxItems:"60"`
}

// ImportSenders adds numbers from the provider's account and reads their
// registrations. Audited.
func (s *Service) ImportSenders(ctx context.Context, ws *store.Workspace, p *authz.Principal, providerID uuid.UUID, in SmsImportInput) ([]SmsSenderRow, error) {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: providerID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, api.MapError(err, "SMS provider")
	}
	ad, err := s.open(prov)
	if err != nil {
		return nil, err
	}
	sl, ok := ad.(sms.SenderLister)
	if !ok {
		return nil, api.Conflict(say(ctx, mNoNumberList, i18n.Args{"provider": prov.Name}))
	}
	found, err := sl.Senders(ctx)
	if err != nil {
		return nil, api.Err(http.StatusBadGateway, "provider_unreachable", noAnswer(ctx, prov.Name, err))
	}
	now := s.now()
	added := []string{}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		for _, want := range in.Senders {
			var f *sms.Sender
			for i := range found {
				if found[i].Value == want.Value {
					f = &found[i]
				}
			}
			if f == nil {
				return api.Invalid("body.senders", say(ctx, mNotInAccount, i18n.Args{"value": want.Value, "provider": prov.Name}))
			}
			countries := f.Countries
			if len(want.Countries) > 0 {
				countries = []string{}
				for _, c := range want.Countries {
					if !validCountry(c) {
						return api.Invalid("body.senders.countries", say(ctx, mCountryCodes, nil))
					}
					countries = append(countries, c)
				}
			}
			var n int64
			_ = tx.QueryRow(ctx, `SELECT count(*) FROM sms_senders WHERE provider_id = $1 AND value = $2 AND removed_at IS NULL`, prov.ID, f.Value).Scan(&n)
			if n > 0 {
				continue
			}
			tp := f.Throughput
			if tp <= 0 {
				tp = 1
			}
			if err := q.SmsSenderInsert(ctx, store.SmsSenderInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ProviderID: prov.ID, Kind: f.Kind, Value: f.Value,
				Countries: countries, Throughput: tp, ProviderRef: f.ProviderRef, AddedBy: uuidPtr(p.UserID), AddedAt: now}); err != nil {
				return err
			}
			added = append(added, f.Value)
		}
		if len(added) == 0 {
			return nil
		}
		if in.SetupID != "" {
			if sid, err := uuid.Parse(in.SetupID); err == nil {
				_ = q.SmsSetupUpdate(ctx, store.SmsSetupUpdateParams{Step: "test", Status: "open", State: jsonOf(map[string]any{"providerId": prov.ID.String()}),
					ProviderID: &prov.ID, At: now, ID: sid})
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_sender.imported", TargetType: "sms_provider", TargetID: prov.ID.String(),
			TargetName: prov.Name, After: map[string]any{"senders": added}})
	})
	if err != nil {
		return nil, err
	}
	if err := s.ReadRegistrations(ctx, prov); err != nil {
		s.log().WarnContext(ctx, "sms registrations after import", "provider", prov.ID, "err", err)
	}
	if err := s.ReadPrices(ctx, prov); err != nil {
		s.log().WarnContext(ctx, "sms prices after import", "provider", prov.ID, "err", err)
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	return s.senderRows(g), nil
}

// SmsSenderIDInput adds an alphanumeric sender ID.
type SmsSenderIDInput struct {
	ProviderID string   `json:"providerId" format:"uuid"`
	Value      string   `json:"value" minLength:"1" maxLength:"11"`
	Countries  []string `json:"countries" minItems:"1" maxItems:"60"`
}

// senderIDRules says how a country treats alphanumeric sender IDs when
// nobody recorded otherwise: allowed without registration, or registered
// first. The US and Canada never allow them.
var senderIDAllowed = []string{"GB", "DE", "NL", "AT", "CH", "DK", "SE", "NO", "FI", "PL", "PT", "IT", "ES", "CZ", "HU", "GR", "RO", "EE", "LV", "LT", "LU", "IS"}

// AddSenderID adds a sender ID with its countries; each country starts as
// allowed or not registered, and its status is set here by hand when the
// provider doesn't report it. Audited.
func (s *Service) AddSenderID(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsSenderIDInput) ([]SmsSenderRow, error) {
	pid, err := uuid.Parse(in.ProviderID)
	if err != nil {
		return nil, api.Invalid("body.providerId", say(ctx, mPickProvider, nil))
	}
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: pid, WorkspaceID: ws.ID})
	if err != nil {
		return nil, api.MapError(err, "SMS provider")
	}
	value := strings.ToUpper(strings.TrimSpace(in.Value))
	if !validSenderID(value) {
		return nil, api.Invalid("body.value", say(ctx, mSenderIDFormat, nil))
	}
	countries := []string{}
	for _, c := range in.Countries {
		if !validCountry(c) {
			return nil, api.Invalid("body.countries", say(ctx, mCountryCodes, nil))
		}
		if !slices.Contains(countries, c) {
			countries = append(countries, c)
		}
	}
	now := s.now()
	id := uuid.Must(uuid.NewV7())
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var n int64
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM sms_senders WHERE provider_id = $1 AND value = $2 AND removed_at IS NULL`, prov.ID, value).Scan(&n)
		if n > 0 {
			return api.Conflict(say(ctx, mSenderExists, i18n.Args{"value": value, "provider": prov.Name}))
		}
		if err := q.SmsSenderInsert(ctx, store.SmsSenderInsertParams{ID: id, WorkspaceID: ws.ID, ProviderID: prov.ID, Kind: "sender_id", Value: value,
			Countries: countries, Throughput: 30, AddedBy: uuidPtr(p.UserID), AddedAt: now}); err != nil {
			return err
		}
		for _, c := range countries {
			status := "not_registered"
			switch {
			case slices.Contains(nanp, c):
				status = "not_supported"
			case slices.Contains(senderIDAllowed, c):
				status = "allowed"
			}
			if err := q.SmsRegistrationUpsert(ctx, store.SmsRegistrationUpsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, SenderID: id, Kind: "sender_id",
				Country: c, Status: status, Source: "rule", Details: []byte("{}"), UpdatedAt: now}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_sender.added", TargetType: "sms_sender", TargetID: id.String(), TargetName: value,
			After: map[string]any{"kind": "sender_id", "provider": prov.Name, "countries": countries}})
	})
	if err != nil {
		return nil, err
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	return s.senderRows(g), nil
}

func validSenderID(v string) bool {
	if v == "" || len(v) > 11 {
		return false
	}
	letter := false
	for _, r := range v {
		switch {
		case r >= 'A' && r <= 'Z':
			letter = true
		case r >= '0' && r <= '9', r == ' ', r == '-', r == '.':
		default:
			return false
		}
	}
	return letter
}

// SmsSenderUpdate changes a sender's countries.
type SmsSenderUpdate struct {
	Countries []string `json:"countries" minItems:"1" maxItems:"60"`
}

// UpdateSender narrows or widens a sender's countries. Audited.
func (s *Service) UpdateSender(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsSenderUpdate) ([]SmsSenderRow, error) {
	sn, err := s.Q.SmsSender(ctx, store.SmsSenderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return nil, api.MapError(err, "Sender")
	}
	for _, c := range in.Countries {
		if !validCountry(c) {
			return nil, api.Invalid("body.countries", say(ctx, mCountryCodes, nil))
		}
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsSenderUpdate(ctx, store.SmsSenderUpdateParams{Countries: in.Countries, Throughput: sn.Throughput, ID: sn.ID}); err != nil {
			return err
		}
		if sn.Kind == "sender_id" {
			for _, c := range in.Countries {
				if slices.Contains(sn.Countries, c) {
					continue
				}
				status := "not_registered"
				if slices.Contains(nanp, c) {
					status = "not_supported"
				} else if slices.Contains(senderIDAllowed, c) {
					status = "allowed"
				}
				if err := q.SmsRegistrationUpsert(ctx, store.SmsRegistrationUpsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, SenderID: sn.ID, Kind: "sender_id",
					Country: c, Status: status, Source: "rule", Details: []byte("{}"), UpdatedAt: now}); err != nil {
					return err
				}
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_sender.updated", TargetType: "sms_sender", TargetID: sn.ID.String(),
			TargetName: senderLabel(storeSenderRow(sn)), Before: map[string]any{"countries": sn.Countries}, After: map[string]any{"countries": in.Countries}})
	})
	if err != nil {
		return nil, err
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	return s.senderRows(g), nil
}

func storeSenderRow(sn store.SmsSenderRow) store.SmsSendersRow {
	return store.SmsSendersRow(sn)
}

// RemoveSender removes a sender; routes that use it must change first.
// Audited.
func (s *Service) RemoveSender(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	sn, err := s.Q.SmsSender(ctx, store.SmsSenderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Sender")
	}
	var n int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_routes WHERE sender_id = $1 OR fallback_sender_id = $1`, id).Scan(&n)
	if n > 0 {
		return api.Conflict(say(ctx, mSenderInRoute, nil))
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsSenderRemove(ctx, store.SmsSenderRemoveParams{At: &now, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_sender.removed", TargetType: "sms_sender", TargetID: id.String(),
			TargetName: senderLabel(storeSenderRow(sn)), Before: map[string]any{"provider": sn.ProviderName, "countries": sn.Countries}})
	})
}

// SmsRegistrationInput sets a registration status by hand, with its date.
type SmsRegistrationInput struct {
	Kind    string    `json:"kind" enum:"toll_free,brand,campaign,sender_id"`
	Country string    `json:"country,omitempty" pattern:"^([A-Z]{2})?$"`
	Name    string    `json:"name,omitempty" maxLength:"120"`
	Status  string    `json:"status" enum:"not_registered,pending,verified,approved,rejected,allowed,not_supported"`
	Reason  string    `json:"reason,omitempty" maxLength:"500"`
	On      time.Time `json:"on" doc:"The day the provider gave this status."`
}

// SetRegistration records a status the provider doesn't report through
// its API (Vonage's sender IDs). Texts held for the sender go once it can
// send. Audited.
func (s *Service) SetRegistration(ctx context.Context, ws *store.Workspace, p *authz.Principal, senderID uuid.UUID, in SmsRegistrationInput) (SmsRegistrationPanel, error) {
	sn, err := s.Q.SmsSender(ctx, store.SmsSenderParams{ID: senderID, WorkspaceID: ws.ID})
	if err != nil {
		return SmsRegistrationPanel{}, api.MapError(err, "Sender")
	}
	if in.Kind == "sender_id" && slices.Contains(nanp, in.Country) {
		return SmsRegistrationPanel{}, api.Invalid("body.country", say(ctx, mSenderIDNanp, nil))
	}
	if in.Status == "rejected" && strings.TrimSpace(in.Reason) == "" {
		return SmsRegistrationPanel{}, api.Invalid("body.reason", say(ctx, mRejectReason, nil))
	}
	on := time.Date(in.On.Year(), in.On.Month(), in.On.Day(), 0, 0, 0, 0, time.UTC)
	if on.After(s.now()) {
		return SmsRegistrationPanel{}, api.Invalid("body.on", say(ctx, mFutureDate, nil))
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsRegistrationPanel{}, err
	}
	before := g.reg(storeSenderRow(sn), firstNonEmpty(in.Country, "US"))
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsRegistrationUpsert(ctx, store.SmsRegistrationUpsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, SenderID: sn.ID, Kind: in.Kind,
			Country: in.Country, Name: strings.TrimSpace(in.Name), Status: in.Status, Reason: strings.TrimSpace(in.Reason), StatusOn: &on, Source: "manual",
			Details: []byte("{}"), SetBy: uuidPtr(p.UserID), UpdatedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_registration.set", TargetType: "sms_sender", TargetID: sn.ID.String(),
			TargetName: senderLabel(storeSenderRow(sn)), Reason: in.Reason, Before: map[string]any{"status": before.Status},
			After: map[string]any{"kind": in.Kind, "country": in.Country, "status": in.Status, "on": on.Format("2006-01-02")}})
	})
	if err != nil {
		return SmsRegistrationPanel{}, err
	}
	if sms.CanSend(in.Status) {
		s.releasePending(ctx, ws)
	}
	return s.Registration(ctx, ws, p, sn.ID)
}

// SmsRegistrationPanel is a sender's registration panel: every status, what
// the provider's form asks for, and where to register.
type SmsRegistrationPanel struct {
	Sender   SmsSenderRow          `json:"sender"`
	Items    []SmsRegistrationView `json:"items"`
	Helper   SmsRegistrationHelper `json:"helper"`
	Console  string                `json:"console"`
	ReadFrom bool                  `json:"readFrom"`
	ReadAt   *time.Time            `json:"readAt,omitempty"`
	// Details are what was submitted, where the provider reports them.
	Details map[string]string `json:"details"`
	// Skipped counts the recipients the pending registration leaves out
	// of the campaigns being written; Held the confirmations waiting.
	Held int64 `json:"held"`
	// Checks are the pages carriers look at, checked by ReputeCast.
	Checks    []SmsPageCheck `json:"checks"`
	CanManage bool           `json:"canManage"`
}

// SmsRegistrationHelper is what the provider's form asks for.
type SmsRegistrationHelper struct {
	OptIn       string   `json:"optIn"`
	Samples     []string `json:"samples"`
	HelpReply   string   `json:"helpReply"`
	OptOutReply string   `json:"optOutReply"`
	TermsURL    string   `json:"termsUrl"`
}

// SmsPageCheck is a page carriers check, and how it answered.
type SmsPageCheck struct {
	URL    string     `json:"url"`
	Status int        `json:"status"`
	At     *time.Time `json:"at,omitempty"`
}

// Registration builds a sender's registration panel.
func (s *Service) Registration(ctx context.Context, ws *store.Workspace, p *authz.Principal, senderID uuid.UUID) (SmsRegistrationPanel, error) {
	sn, err := s.Q.SmsSender(ctx, store.SmsSenderParams{ID: senderID, WorkspaceID: ws.ID})
	if err != nil {
		return SmsRegistrationPanel{}, api.MapError(err, "Sender")
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsRegistrationPanel{}, err
	}
	out := SmsRegistrationPanel{Items: []SmsRegistrationView{}, Details: map[string]string{}, Checks: []SmsPageCheck{}, CanManage: p.Can(authz.ProvidersManage)}
	for _, row := range s.senderRows(g) {
		if row.ID == sn.ID.String() {
			out.Sender = row
			break
		}
	}
	d, _ := sms.Lookup(sn.ProviderAdapter)
	out.ReadFrom = d.Has(sms.CapRegistration) == sms.Yes
	for _, r := range g.regs[sn.ID] {
		v := regRowView(&r)
		out.Items = append(out.Items, *v)
		if out.ReadAt == nil || (r.ReadAt != nil && r.ReadAt.After(*out.ReadAt)) {
			out.ReadAt = r.ReadAt
		}
		for k, val := range detailsOf(r) {
			out.Details[k] = val
		}
	}
	if sn.Kind == "sender_id" {
		// Every country, with the rule for the US and Canada.
		for _, c := range append(append([]string{}, sn.Countries...), nanp...) {
			found := false
			for _, it := range out.Items {
				if it.Country == c {
					found = true
				}
			}
			if !found {
				v := regView(g.reg(storeSenderRow(sn), c))
				v.Country = c
				out.Items = append(out.Items, v)
			}
		}
	}
	if d.Console != nil {
		kind := "registration"
		if sn.Kind == "toll_free" {
			kind = "tollfree"
		}
		out.Console = d.Console(kind, sn.ProviderRef)
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE sender_id = $1 AND status = 'held'`, sn.ID).Scan(&out.Held)
	out.Helper = s.helper(ctx, ws, g)
	return out, nil
}

// helper prepares what a 10DLC campaign or toll-free verification form
// asks for, from the workspace's forms, texts and rules.
func (s *Service) helper(ctx context.Context, ws *store.Workspace, g *graph) SmsRegistrationHelper {
	h := SmsRegistrationHelper{Samples: []string{}, HelpReply: g.rules.HelpReply, OptOutReply: g.rules.OptOutReply, TermsURL: s.TermsURL(ctx, ws)}
	// The opt-in: the latest SMS consent with its page and text.
	var page, text string
	_ = s.Pool.QueryRow(ctx, `SELECT page, consent_text FROM channel_consent_events WHERE workspace_id = $1 AND channel = 'sms' AND action IN ('subscribed','pending')
		AND consent_text <> '' ORDER BY occurred_at DESC LIMIT 1`, ws.ID).Scan(&page, &text)
	if page != "" && text != "" {
		h.OptIn = "People sign up at " + page + " and tick an unchecked SMS consent box with this text: “" + text + "”"
	}
	rows, err := s.Pool.Query(ctx, `SELECT s.body FROM sms_campaigns s JOIN campaigns c ON c.id = s.campaign_id
		WHERE c.workspace_id = $1 AND c.deleted_at IS NULL AND s.body <> '' ORDER BY c.created_at DESC LIMIT 2`, ws.ID)
	if err == nil {
		for rows.Next() {
			var b string
			if rows.Scan(&b) == nil {
				sample := Text{Prefix: g.rules.BrandPrefix, Body: withShortLinks(b, s.shortDomain(g.rules), func(i int, _ Link) string { return "k3P9xq" }), OptOut: g.rules.OptOutText}
				h.Samples = append(h.Samples, render(sample.Full(), Recipient{FirstName: "Sam"}))
			}
		}
		rows.Close()
	}
	return h
}

// CheckPages asks the pages carriers check (the terms page and the
// sign-up page) and records how they answered.
func (s *Service) CheckPages(ctx context.Context, ws *store.Workspace, p *authz.Principal, senderID uuid.UUID) (SmsRegistrationPanel, error) {
	panel, err := s.Registration(ctx, ws, p, senderID)
	if err != nil {
		return panel, err
	}
	urls := []string{panel.Helper.TermsURL}
	var page string
	_ = s.Pool.QueryRow(ctx, `SELECT page FROM channel_consent_events WHERE workspace_id = $1 AND channel = 'sms' AND page LIKE '%.%' ORDER BY occurred_at DESC LIMIT 1`,
		ws.ID).Scan(&page)
	if page != "" {
		if !strings.HasPrefix(page, "http") {
			page = "https://" + page
		}
		urls = append(urls, page)
	}
	now := s.now()
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	for _, u := range urls {
		if u == "" {
			continue
		}
		c := SmsPageCheck{URL: u, At: &now}
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		req, err := http.NewRequestWithContext(cctx, http.MethodGet, u, nil)
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				c.Status = resp.StatusCode
				_ = resp.Body.Close()
			}
		}
		cancel()
		panel.Checks = append(panel.Checks, c)
	}
	return panel, nil
}

// SmsCredentialsInput replaces a provider's credentials.
type SmsCredentialsInput struct {
	Fields map[string]string `json:"fields"`
}

// ReplaceCredentials checks and seals new credentials; held texts go once
// they work. Audited.
func (s *Service) ReplaceCredentials(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsCredentialsInput) error {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "SMS provider")
	}
	d, ok := sms.Lookup(prov.Adapter)
	if !ok {
		return api.Conflict(say(ctx, mNoAdapter, i18n.Args{"adapter": prov.Adapter}))
	}
	settings := settingsOf(prov)
	creds := map[string]string{}
	for _, f := range d.Fields {
		v := strings.TrimSpace(in.Fields[f.Key])
		if v == "" {
			if f.Secret && f.Required {
				return api.Invalid("body.fields."+f.Key, say(ctx, mFieldRequired, i18n.Args{"field": sms.TextIn(f.Label, f.LabelText, i18n.LangFrom(ctx))}))
			}
			continue
		}
		if f.Secret {
			creds[f.Key] = v
		} else {
			settings[f.Key] = v
		}
	}
	cfg := sms.Config{ProviderID: prov.ID, WorkspaceID: ws.ID, Settings: settings, Credentials: creds, HTTP: s.HTTP, Now: s.now}
	if prov.Demo && s.Demo != nil {
		cfg.HTTP, cfg.BaseURL = s.Demo(prov.Adapter)
	}
	ad, err := d.New(cfg)
	if err != nil {
		return api.Invalid("body.fields", err.Error())
	}
	acct, err := ad.Check(ctx)
	if err != nil {
		if errors.Is(err, sms.ErrCredentialsRejected) {
			return api.Err(http.StatusUnprocessableEntity, "credentials_rejected", say(ctx, mCredentialsAgain, i18n.Args{"provider": d.Name}))
		}
		return api.Err(http.StatusBadGateway, "provider_unreachable", noAnswer(ctx, d.Name, err))
	}
	b, _ := json.Marshal(creds)
	sealed, err := s.Box.Seal(b, credentialsAD(prov.ID))
	if err != nil {
		return err
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsProviderSetCredentials(ctx, store.SmsProviderSetCredentialsParams{CredentialsEnc: sealed, Config: jsonOf(settings), AccountLabel: firstNonEmpty(acct.Name, prov.AccountLabel),
			AccountHint: acct.Hint, At: &now, ID: prov.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.credentials_replaced", TargetType: "sms_provider", TargetID: prov.ID.String(),
			TargetName: prov.Name, Before: map[string]any{"status": prov.Status, "hint": prov.AccountHint}, After: map[string]any{"status": "healthy", "hint": acct.Hint}})
	})
	if err != nil {
		return err
	}
	s.releaseHeld(ctx, ws.ID, &prov.ID, "provider_down")
	return nil
}

// RegisterWebhooks registers the receipt and inbound webhooks again.
// Audited.
func (s *Service) RegisterWebhooks(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsWebhookSetup, error) {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SmsWebhookSetup{}, api.MapError(err, "SMS provider")
	}
	ad, err := s.open(prov)
	if err != nil {
		return SmsWebhookSetup{}, err
	}
	out := SmsWebhookSetup{Mode: "registered", URL: s.hookURL(prov)}
	reg, ok := ad.(sms.WebhookRegistrar)
	if !ok {
		return out, api.Conflict(say(ctx, mNoWebhookAPI, i18n.Args{"provider": prov.Name}))
	}
	senders := []sms.Sender{}
	all, _ := s.Q.SmsSenders(ctx, ws.ID)
	for _, sn := range all {
		if sn.ProviderID == prov.ID {
			senders = append(senders, sms.Sender{Kind: sn.Kind, Value: sn.Value, Countries: sn.Countries, ProviderRef: sn.ProviderRef})
		}
	}
	setup, err := reg.RegisterWebhooks(ctx, sms.Webhooks{StatusURL: out.URL, InboundURL: out.URL}, senders)
	if err != nil {
		return out, api.Err(http.StatusBadGateway, "provider_unreachable",
			say(ctx, mWebhooksFailed, i18n.Args{"provider": prov.Name, "answer": answerIn(i18n.LangFrom(ctx), answerOf(err))}))
	}
	settings := settingsOf(prov)
	for k, v := range setup.Settings {
		settings[k] = v
	}
	var sealed []byte
	if setup.Secret != "" {
		if sealed, err = s.Box.Seal([]byte(setup.Secret), secretAD(prov.ID)); err != nil {
			return out, err
		}
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsProviderSetWebhooks(ctx, store.SmsProviderSetWebhooksParams{WebhookSecretEnc: sealed, Config: jsonOf(settings), Webhooks: "registered", ID: prov.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.webhooks_registered", TargetType: "sms_provider", TargetID: prov.ID.String(),
			TargetName: prov.Name, After: map[string]any{"url": out.URL}})
	})
	return out, err
}

// CheckNow probes a provider at once ("Retry now" during an outage).
func (s *Service) CheckNow(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SmsOverview, error) {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SmsOverview{}, api.MapError(err, "SMS provider")
	}
	_ = s.CheckProvider(ctx, prov)
	return s.Overview(ctx, ws, api.PrincipalFrom(ctx))
}

// RemoveProvider disconnects a provider: its routes must go first, and
// queued texts through it are cancelled. Audited.
func (s *Service) RemoveProvider(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "SMS provider")
	}
	var n int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_routes r JOIN sms_senders s ON s.id = r.sender_id OR s.id = r.fallback_sender_id
		WHERE s.provider_id = $1`, id).Scan(&n)
	if n > 0 {
		return api.Conflict(say(ctx, mProviderInRoute, i18n.Args{"provider": prov.Name}))
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'provider_removed', updated_at = $2
			WHERE provider_id = $1 AND status IN ('queued','held')`, id, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE sms_senders SET removed_at = $2 WHERE provider_id = $1 AND removed_at IS NULL`, id, now); err != nil {
			return err
		}
		if err := q.SmsProviderRemove(ctx, store.SmsProviderRemoveParams{At: &now, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.removed", TargetType: "sms_provider", TargetID: id.String(), TargetName: prov.Name,
			Before: map[string]any{"adapter": prov.Adapter, "account": prov.AccountLabel}})
	})
}

// SmsProviderTestInput sends the wizard's test text to a member's phone.
type SmsProviderTestInput struct {
	// TestNumberID is a test number; Phone a member's number typed here.
	TestNumberID string `json:"testNumberId,omitempty"`
	Phone        string `json:"phone,omitempty" maxLength:"40"`
	SetupID      string `json:"setupId,omitempty"`
}

// SmsMessageView is one text with its timeline: the wizard's test, a
// contact's text.
type SmsMessageView struct {
	ID                string            `json:"id"`
	Kind              string            `json:"kind"`
	Status            string            `json:"status" enum:"queued,held,skipped,sending,sent,delivered,undelivered,failed,cancelled"`
	Reason            string            `json:"reason,omitempty"`
	Recipient         string            `json:"recipient"`
	Sender            string            `json:"sender"`
	Provider          string            `json:"provider"`
	Body              string            `json:"body"`
	Characters        int               `json:"characters"`
	Segments          int               `json:"segments"`
	Encoding          string            `json:"encoding"`
	ProviderMessageID string            `json:"providerMessageId,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	SentAt            *time.Time        `json:"sentAt,omitempty"`
	DeliveredAt       *time.Time        `json:"deliveredAt,omitempty"`
	Seconds           *float64          `json:"seconds,omitempty"`
	Events            []SmsMessageEvent `json:"events"`
}

// SmsMessageEvent is one step of a text's timeline.
type SmsMessageEvent struct {
	Kind string    `json:"kind"`
	Code string    `json:"code,omitempty"`
	At   time.Time `json:"at"`
}

// ProviderTest texts "This is a test from ReputeCast" from the provider's
// first sender that can text the number's country.
func (s *Service) ProviderTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsProviderTestInput) (SmsMessageView, error) {
	prov, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SmsMessageView{}, api.MapError(err, "SMS provider")
	}
	ph := in.Phone
	if in.TestNumberID != "" {
		nums, err := s.Q.SmsTestNumbers(ctx, ws.ID)
		if err != nil {
			return SmsMessageView{}, err
		}
		for _, n := range nums {
			if n.ID.String() == in.TestNumberID {
				ph = n.Phone
			}
		}
	}
	n, err := parsePhone(ph)
	if err != nil {
		return SmsMessageView{}, api.Invalid("body.phone", say(ctx, mPhoneFormat, nil))
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsMessageView{}, err
	}
	var sender *store.SmsSendersRow
	for _, sid := range g.order {
		sn := g.senders[sid]
		if sn.ProviderID == prov.ID && g.reg(sn, n.Country).CanSend() {
			sender = &sn
			break
		}
	}
	if sender == nil {
		return SmsMessageView{}, api.Err(http.StatusConflict, "no_sender",
			say(ctx, mNoProviderSender, i18n.Args{"provider": prov.Name, "country": countryNameIn(i18n.LangFrom(ctx), n.Country)}))
	}
	text := Text{Prefix: g.rules.BrandPrefix, Body: "This is a test from ReputeCast.", OptOut: g.rules.OptOutText}
	m, err := s.sendOne(ctx, ws, g, oneText{kind: "test", recipient: Recipient{Phone: n.E164}, text: text, sender: sender, by: p.UserID})
	if err != nil {
		return SmsMessageView{}, err
	}
	if in.SetupID != "" {
		if sid, err := uuid.Parse(in.SetupID); err == nil {
			_ = s.Q.SmsSetupUpdate(ctx, store.SmsSetupUpdateParams{Step: "countries", Status: "open", State: jsonOf(map[string]any{"providerId": prov.ID.String(),
				"testMessageId": m.ID.String()}), ProviderID: &prov.ID, At: s.now(), ID: sid})
		}
	}
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.test_sent", TargetType: "sms_provider", TargetID: prov.ID.String(), TargetName: prov.Name,
		After: map[string]any{"to": maskPhone(n.E164), "sender": senderLabel(*sender)}})
	return s.Message(ctx, ws, m.ID)
}

// Message is one text with its timeline.
func (s *Service) Message(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SmsMessageView, error) {
	m, err := s.Q.ChanMessage(ctx, id)
	if err != nil || m.WorkspaceID != ws.ID {
		return SmsMessageView{}, api.NotFound("Text")
	}
	out := SmsMessageView{ID: m.ID.String(), Kind: m.Kind, Status: m.Status, Reason: m.Reason, Recipient: maskPhone(m.Recipient), Sender: senderShown(m.Sender),
		ProviderMessageID: m.ProviderMessageID, CreatedAt: m.CreatedAt, SentAt: m.SentAt, DeliveredAt: m.DeliveredAt, Segments: int(m.Units), Encoding: m.Encoding,
		Events: []SmsMessageEvent{}}
	if m.Body != nil {
		out.Body = *m.Body
		c := sms.CountText(out.Body)
		out.Characters = c.Characters
	}
	if m.ProviderID != nil {
		if p, err := s.Q.SmsProviderAny(ctx, *m.ProviderID); err == nil {
			out.Provider = p.Name
		}
	}
	if m.DeliveredAt != nil && m.SentAt != nil {
		sec := m.DeliveredAt.Sub(*m.SentAt).Seconds()
		out.Seconds = &sec
	}
	evs, err := s.Q.ChanMessageEvents(ctx, m.ID)
	if err != nil {
		return out, err
	}
	for _, e := range evs {
		out.Events = append(out.Events, SmsMessageEvent{Kind: e.Kind, Code: e.ProviderCode, At: e.OccurredAt})
	}
	return out, nil
}

// FinishSetup closes the wizard. Audited.
func (s *Service) FinishSetup(ctx context.Context, ws *store.Workspace, p *authz.Principal, setupID uuid.UUID) error {
	st, err := s.Q.SmsSetup(ctx, store.SmsSetupParams{ID: setupID, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Setup")
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsSetupUpdate(ctx, store.SmsSetupUpdateParams{Step: "done", Status: "finished", State: st.State, ProviderID: st.ProviderID, At: now, ID: st.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.setup_finished", TargetType: "sms_setup", TargetID: st.ID.String(),
			TargetName: st.Adapter, After: map[string]any{"step": "done"}})
	})
}

func smsProviderParams(id, workspaceID uuid.UUID) store.SmsProviderParams {
	return store.SmsProviderParams{ID: id, WorkspaceID: workspaceID}
}
