package commerce

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// PeriodDays is the store page's period.
const PeriodDays = 30

// summary is a store in lists, with its sync row.
func (s *Service) summary(ctx context.Context, st store.Store) CommerceStoreSummary {
	out := CommerceStoreSummary{ID: st.ID.String(), Platform: st.Platform, Name: st.Name, Address: st.Address, Domain: st.Domain, Currency: st.Currency,
		Status: st.Status, Step: st.Step, Sync: s.syncState(ctx, st), Demo: st.Demo}
	if n, err := s.Q.CommerceProductCount(ctx, st.ID); err == nil {
		out.Products = n
	}
	if st.ListID != nil {
		out.ListID = st.ListID.String()
		if n, err := s.Q.CommerceListName(ctx, *st.ListID); err == nil {
			out.ListName = n
		}
	}
	return out
}

// syncState is the store sync row: connected, importing, a webhook missing,
// rate limited, WooCommerce without its plugin, disconnected.
func (s *Service) syncState(ctx context.Context, st store.Store) CommerceSyncState {
	out := CommerceSyncState{LastEventAt: st.LastEventAt, Missing: []string{}, RevokedAt: st.RevokedAt, RevokedReason: st.RevokedReason,
		DisconnectedAt: st.DisconnectedAt, DisconnectedBy: s.userName(ctx, st.DisconnectedBy)}
	switch st.Status {
	case StatusConnecting:
		out.State = "connecting"
		return out
	case StatusDisconnected:
		out.State = "disconnected"
		return out
	case StatusImporting:
		out.State = "importing"
		out.Import = s.importView(ctx, st)
		return out
	}
	if rows, err := s.Q.CommerceWebhooks(ctx, st.ID); err == nil {
		for _, r := range rows {
			if r.Status != "missing" || r.Position >= lifecyclePosition {
				continue
			}
			out.Missing = append(out.Missing, r.Topic)
			if r.MissingSince != nil && (out.MissingSince == nil || r.MissingSince.Before(*out.MissingSince)) {
				out.MissingSince = r.MissingSince
			}
		}
	}
	switch {
	case len(out.Missing) > 0:
		out.State = "webhook_missing"
	case st.RateLimitedUntil != nil && st.RateLimitedUntil.After(s.now()):
		out.State, out.RateLimitedUntil = "rate_limited", st.RateLimitedUntil
	case st.Platform == WooCommerce && !st.PluginFound:
		out.State = "no_plugin"
	default:
		out.State = "connected"
	}
	return out
}

// importView is the first import's progress, with an estimate for the
// orders left.
func (s *Service) importView(ctx context.Context, st store.Store) *CommerceImport {
	p := progressOf(st.ImportProgress)
	out := &CommerceImport{Phase: p.Phase, Products: p.Products, Collections: p.Collections, Customers: p.Customers, Accepted: p.Accepted, Known: p.Known,
		NotAdded: p.NotAdded, CustomersSkipped: p.CustomersSkipped, Orders: p.Orders, Months: p.Months, Matching: p.Matching, FinishedAt: st.ImportFinishedAt,
		StartedBy: s.userName(ctx, st.ImportStartedBy)}
	if out.StartedBy == "" {
		out.StartedBy = p.By
	}
	if st.ImportStartedAt != nil {
		out.StartedAt = *st.ImportStartedAt
	}
	if p.OrdersStarted != nil && p.Orders.Done > 0 && p.Orders.Total > p.Orders.Done {
		elapsed := s.now().Sub(*p.OrdersStarted)
		left := time.Duration(float64(elapsed) / float64(p.Orders.Done) * float64(p.Orders.Total-p.Orders.Done))
		out.MinutesLeft = int(math.Ceil(left.Minutes()))
	}
	switch p.Phase {
	case "products", "customers", "orders", "matching", "done":
	default:
		out.Phase = "products"
	}
	return out
}

// Stores lists a workspace's stores.
func (s *Service) Stores(ctx context.Context, ws *store.Workspace, p *authz.Principal) (CommerceStoreList, error) {
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return CommerceStoreList{}, err
	}
	out := CommerceStoreList{Stores: []CommerceStoreSummary{}, CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID, authz.IntegrationsManage)}
	for _, st := range rows {
		out.Stores = append(out.Stores, s.summary(ctx, st))
	}
	return out, nil
}

// Page is a store's page in the integrations hub.
func (s *Service) Page(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CommerceStorePage, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceStorePage{}, err
	}
	now := s.now()
	since := now.AddDate(0, 0, -PeriodDays)
	out := CommerceStorePage{Store: s.summary(ctx, st), AppName: st.AppName, APIVersion: st.ApiVersion, ConnectedBy: s.userName(ctx, st.ConnectedBy),
		ConnectedAt: st.ConnectedAt, PublicDomain: st.PublicDomain, UsedBy: []CommerceUse{}, Webhooks: []CommerceWebhook{}, Activity: []CommerceActivityItem{},
		Flows: []CommerceFlow{}, CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID, authz.IntegrationsManage),
		Lists: s.listOptions(ctx, ws.ID)}
	out.Capabilities = s.capabilities(st)
	totals, err := s.Q.CommerceStoreTotals(ctx, store.CommerceStoreTotalsParams{StoreID: st.ID, Since: since, Until: now})
	if err != nil {
		return out, err
	}
	out.Numbers = CommerceNumbers{Days: PeriodDays, Orders: totals.Orders, Revenue: totals.Revenue, Currency: st.Currency, Credited: totals.Credited,
		CreditedRevenue: totals.CreditedRevenue, Share: share(totals.CreditedRevenue, totals.Revenue)}
	for _, o := range out.Lists {
		if st.ListID != nil && o.ID == st.ListID.String() {
			out.Numbers.ListMembers = o.Members
			out.Consent.ListMembers = o.Members
			out.Consent.ListName = o.Name
			out.Consent.ListCreated = o.Store
		}
	}
	if st.ListID != nil {
		out.Consent.ListID = st.ListID.String()
	}
	out.Consent.WriteBack = st.WriteBack
	out.Consent.WriteGranted = slices.Contains(st.Scopes, shopify.OptionalScope)
	out.Consent.WriteBackSupported = st.Platform == Shopify

	// What uses the store.
	flows, err := s.Q.CommerceStoreFlows(ctx, store.CommerceStoreFlowsParams{WorkspaceID: ws.ID, Store: st.ID.String()})
	if err != nil {
		return out, err
	}
	var recoveredSince *time.Time
	for _, f := range flows {
		var t StoreTrigger
		_ = json.Unmarshal(f.Trigger, &t)
		out.Flows = append(out.Flows, CommerceFlow{ID: f.ID.String(), Name: f.Name, Status: f.Status, Kind: t.Kind, PauseReason: f.PauseReason})
		out.UsedBy = append(out.UsedBy, CommerceUse{Kind: "automation", ID: f.ID.String(), Name: f.Name, Status: f.Status})
		if t.Kind != TriggerAbandonedCheckout {
			continue
		}
		live := time.Time{}
		if row, err := s.Q.CommerceAutomationName(ctx, f.ID); err == nil && row.LiveVersion != nil {
			if at, err := s.Q.CommerceFlowLiveSince(ctx, store.CommerceFlowLiveSinceParams{AutomationID: f.ID, Version: *row.LiveVersion}); err == nil && at != nil {
				live = *at
			}
		}
		if g, err := s.Q.CommerceAutomationGoalRevenue(ctx, store.CommerceAutomationGoalRevenueParams{AutomationID: &f.ID, Since: live}); err == nil {
			out.Recovered.Orders += g.Orders
			out.Recovered.Revenue += g.Revenue
		}
		if !live.IsZero() && (recoveredSince == nil || live.Before(*recoveredSince)) {
			l := live
			recoveredSince = &l
		}
	}
	out.Recovered.Since = recoveredSince
	if camps, err := s.Q.CommerceCampaignsUsing(ctx, store.CommerceCampaignsUsingParams{WorkspaceID: ws.ID, StoreID: st.ID.String()}); err == nil {
		for _, c := range camps {
			out.UsedBy = append(out.UsedBy, CommerceUse{Kind: "campaign", ID: c.ID.String(), Name: c.Name, Status: c.Status})
		}
	}
	if totals.Credited > 0 {
		out.UsedBy = append(out.UsedBy, CommerceUse{Kind: "reports"})
	}

	// Sync.
	prog := progressOf(st.ImportProgress)
	out.Sync = CommerceSyncFacts{Products: out.Store.Products, Collections: []string{}, Known: prog.Known, New: prog.Accepted - prog.Known,
		NotAdded: prog.NotAdded, OrdersImported: prog.Orders.Done, Months: int(st.ImportMonths), ImportFinished: st.ImportFinishedAt,
		ReconciledAt: st.ReconciledAt, Missed: int(st.ReconcileMissed), CustomersFromOrders: st.Platform == WooCommerce}
	if out.Sync.New < 0 {
		out.Sync.New = 0
	}
	if st.ImportStartedAt != nil && st.ImportFinishedAt != nil {
		out.Sync.ImportMinutes = int(math.Round(st.ImportFinishedAt.Sub(*st.ImportStartedAt).Minutes()))
	}
	if cols, err := s.Q.CommerceCollections(ctx, st.ID); err == nil {
		for _, c := range cols {
			out.Sync.Collections = append(out.Sync.Collections, c.Title)
		}
	}
	if stats, err := s.Q.CommerceCustomerStats(ctx, st.ID); err == nil {
		out.Sync.Accepted = stats.Joined
		if stats.Subscribed > out.Sync.Accepted {
			out.Sync.Accepted = stats.Subscribed
		}
	}
	if hooks, err := s.Q.CommerceWebhooks(ctx, st.ID); err == nil {
		for _, h := range hooks {
			if h.Position >= lifecyclePosition {
				continue
			}
			out.Sync.WebhooksTotal++
			if h.Status == "registered" {
				out.Sync.Webhooks++
			}
			out.Webhooks = append(out.Webhooks, CommerceWebhook{Topic: h.Topic, Status: h.Status, LastEventAt: h.LastEventAt, MissingSince: h.MissingSince, Error: h.Error})
		}
	}

	// Attribution.
	out.Attribution = s.attributionView(ctx, ws, st, totals, since, now)

	// Activity.
	if act, err := s.Activity(ctx, ws, st.ID, "", nil, 7); err == nil {
		out.Activity = act.Items
	}
	if st.Platform == WooCommerce {
		out.Plugin = s.pluginState(ctx, st)
	}
	return out, nil
}

// capabilities are the store's badges: WooCommerce without the plugin
// has only orders and products.
func (s *Service) capabilities(st store.Store) []string {
	out := []string{}
	for _, c := range descriptor(st.Platform).Capabilities {
		if st.Platform == WooCommerce && !st.PluginFound && c != "orders" && c != "products" {
			continue
		}
		out = append(out, string(c))
	}
	return out
}

func share(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round(float64(part)/float64(whole)*1000) / 10
}

// topRows is how many emails the attribution table names before "N other
// campaigns".
const topRows = 5

func (s *Service) attributionView(ctx context.Context, ws *store.Workspace, st store.Store, totals store.CommerceStoreTotalsRow, since, until time.Time) CommerceAttribution {
	out := CommerceAttribution{Days: int(st.AttributionDays), SetBy: s.userName(ctx, st.AttributionSetBy), SetAt: st.AttributionSetAt, TagLinks: st.TagLinks,
		TagHost: st.PublicDomain, Param: clicktoken.Param, Orders: totals.Orders, Revenue: totals.Revenue, Credited: totals.Credited,
		CreditedRevenue: totals.CreditedRevenue, ByToken: totals.ByToken, ByEmail: totals.ByEmail, Rows: []CommerceRevenueRow{}}
	if out.SetBy == "" {
		out.SetBy = s.userName(ctx, st.ConnectedBy)
	}
	if out.TagHost == "" {
		out.TagHost = st.Domain
	}
	campaign := "your-campaign"
	if name, err := s.Q.CommerceLatestCampaign(ctx, ws.ID); err == nil {
		campaign = clicktoken.Slug(name)
	}
	out.TagExample = "?utm_source=" + ws.Slug + "&utm_medium=email&utm_campaign=" + campaign + "&" + clicktoken.Param + "=…"
	rows, err := s.Q.CommerceRevenueByEmail(ctx, store.CommerceRevenueByEmailParams{StoreID: st.ID, Since: since, Until: until})
	if err != nil {
		return out
	}
	for i, r := range rows {
		row := CommerceRevenueRow{Kind: "campaign", Name: r.Name, Count: 1, Orders: r.Orders, Revenue: r.Revenue}
		switch {
		case r.Automation && r.AutomationID != nil:
			row.Kind, row.ID = "automation", r.AutomationID.String()
		case r.CampaignID != nil:
			row.ID = r.CampaignID.String()
		}
		if i < topRows || (i == topRows && len(rows) == topRows+1) {
			out.Rows = append(out.Rows, row)
			continue
		}
		if out.Others == nil {
			out.Others = &CommerceRevenueRow{Kind: "others"}
		}
		out.Others.Count++
		out.Others.Orders += r.Orders
		out.Others.Revenue += r.Revenue
	}
	return out
}
