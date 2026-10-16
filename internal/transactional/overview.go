package transactional

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Templates tab's header and side cards, and the Overview's
// Transactional card: the period's numbers, the route, any active alert
// or failover, the incidents of 30 days, and the keys that can send.

// TransactionalOverviewStats are the workspace's numbers over a period.
type TransactionalOverviewStats struct {
	TransactionalStats
	Suppressed int64 `json:"suppressed"`
	OptedOut   int64 `json:"optedOut"`
	// Keys is how many API keys called in the period, test-mode keys
	// included; SMTP whether the relay sent.
	Keys int64 `json:"keys"`
	SMTP bool  `json:"smtp"`
	// ComplainedOn names the templates with complaints.
	ComplainedOn []string `json:"complainedOn"`
}

// TransactionalRouteSummary is the Transactional route at a glance.
type TransactionalRouteSummary struct {
	Configured bool              `json:"configured"`
	Primary    *TransactionalRef `json:"primary,omitempty"`
	Fallback   *TransactionalRef `json:"fallback,omitempty"`
	Policy     string            `json:"policy,omitempty" enum:"automatic,pause_and_ask,warm_only,"`
	// Priority: "First in line on shared providers".
	Priority bool `json:"priority"`
	// Conditions counts the route's rules besides the default one.
	Conditions int `json:"conditions"`
	// System is the System route's provider, offered when there is no
	// Transactional route yet ("Use the System route for now").
	System *TransactionalRef `json:"system,omitempty"`
}

// TransactionalFailoverState is an outage of the primary: mail goes
// through the fallback.
type TransactionalFailoverState struct {
	From TransactionalRef `json:"from"`
	To   TransactionalRef `json:"to"`
	// Since is when the primary went down; Code its last answer ("503").
	Since    time.Time `json:"since"`
	Code     string    `json:"code,omitempty"`
	Messages int64     `json:"messages"`
	Lost     int64     `json:"lost"`
	MedianMs *int64    `json:"medianMs,omitempty"`
	// Recovering: calls succeed again; the primary takes over after ten
	// healthy minutes.
	Recovering bool `json:"recovering"`
}

// TransactionalSendingKey is a key that can send.
type TransactionalSendingKey struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Masked        string     `json:"masked"`
	Preset        string     `json:"preset" enum:"send_only,smtp,"`
	Mode          string     `json:"mode" enum:"live,test"`
	Templates     []string   `json:"templates"`
	Categories    []string   `json:"categories" doc:"Category names the key is limited to"`
	TestAddresses []string   `json:"testAddresses"`
	RatePerSecond int        `json:"ratePerSecond"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     string     `json:"createdBy"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Expired       bool       `json:"expired"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedIP    string     `json:"lastUsedIp,omitempty"`
}

// TransactionalFirstRun is what the first-run state and the guide check.
type TransactionalFirstRun struct {
	Templates      int64 `json:"templates"`
	Published      int64 `json:"published"`
	Messages       int64 `json:"messages"`
	Keys           int   `json:"keys"`
	Senders        int   `json:"senders" doc:"Senders on Transactional domains"`
	VerifiedSender bool  `json:"verifiedSender"`
}

// TransactionalHour is the last hour, for the alert banner.
type TransactionalHour struct {
	Sent       int64   `json:"sent"`
	Bounced    int64   `json:"bounced"`
	BounceRate float64 `json:"bounceRate"`
	MedianMs   *int64  `json:"medianMs,omitempty"`
}

// TransactionalAccess is what the person can do from the overview.
type TransactionalAccess struct {
	Edit       bool `json:"edit"`
	Publish    bool `json:"publish"`
	Send       bool `json:"send"`
	Content    bool `json:"content"`
	ManageKeys bool `json:"manageKeys"`
	EditRoute  bool `json:"editRoute"`
}

// TransactionalOverview is the Templates tab's header and side cards.
type TransactionalOverview struct {
	Period      string                      `json:"period" enum:"24h,7d,30d"`
	IncludeTest bool                        `json:"includeTest"`
	Stats       TransactionalOverviewStats  `json:"stats"`
	Route       TransactionalRouteSummary   `json:"route"`
	Failover    *TransactionalFailoverState `json:"failover,omitempty"`
	// Alerts are the open delivery alerts (failovers are in Failover).
	Alerts []TransactionalAlert `json:"alerts"`
	// Incidents are the alerts and failovers of the last 30 days.
	Incidents []TransactionalAlert      `json:"incidents"`
	Hour      TransactionalHour         `json:"hour"`
	Keys      []TransactionalSendingKey `json:"keys"`
	FirstRun  TransactionalFirstRun     `json:"firstRun"`
	// Endpoint is the send API's URL on this instance.
	Endpoint string              `json:"endpoint"`
	Can      TransactionalAccess `json:"can"`
}

// Overview returns the Templates tab's header and side cards.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, period string, includeTest bool) (TransactionalOverview, error) {
	if period != Period24h && period != Period30d {
		period = Period7d
	}
	span, _ := periodSpan(period)
	now := s.now()
	out := TransactionalOverview{Period: period, IncludeTest: includeTest, Alerts: []TransactionalAlert{}, Incidents: []TransactionalAlert{}, Keys: []TransactionalSendingKey{},
		Endpoint: s.URLs.Base() + "/api/v1/workspaces/" + ws.Slug + "/transactional/messages"}
	if p != nil {
		out.Can = TransactionalAccess{Edit: p.Can(authz.TransactionalEdit), Publish: p.Can(authz.TransactionalPublish), Send: p.Can(authz.TransactionalSend),
			Content: p.Can(authz.TransactionalContent), ManageKeys: p.Can(authz.APIKeysManage), EditRoute: p.Can(authz.ProvidersManage)}
	}
	st, err := s.Q.TxnOverviewStats(ctx, store.TxnOverviewStatsParams{WorkspaceID: ws.ID, Since: now.Add(-span), IncludeTest: includeTest})
	if err != nil {
		return out, err
	}
	out.Stats = TransactionalOverviewStats{TransactionalStats: statsOf(st.Sent, st.Delivered, st.Bounced, st.Complained, st.Deferred, st.Suppressed+st.OptedOut, st.Failed, st.Opened, st.Clicked, st.MedianMs),
		Suppressed: st.Suppressed, OptedOut: st.OptedOut, Keys: st.Keys, SMTP: st.Smtp, ComplainedOn: []string{}}
	if st.ComplainedTemplates != "" {
		out.Stats.ComplainedOn = strings.Split(st.ComplainedTemplates, ",")
	}

	// The route, and an outage of its primary.
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, pr := range provs {
		byID[pr.ID] = pr
	}
	ref := func(id uuid.UUID) *TransactionalRef {
		if pr, ok := byID[id]; ok {
			return &TransactionalRef{ID: pr.ID.String(), Name: pr.Name}
		}
		return nil
	}
	if def, ok := set.Default(routing.StreamTransactional); ok && len(def.Splits) > 0 {
		out.Route.Configured = true
		top := def.Splits[0]
		for _, sp := range def.Splits[1:] {
			if sp.Weight > top.Weight {
				top = sp
			}
		}
		out.Route.Primary = ref(top.ProviderID)
		if def.Fallback != nil {
			out.Route.Fallback = ref(*def.Fallback)
		}
		out.Route.Policy = string(set.Policy(routing.StreamTransactional))
		out.Route.Conditions = len(set.StreamRules(routing.StreamTransactional)) - 1
		if primary, ok := byID[top.ProviderID]; ok && !state.Available(primary) && def.Fallback != nil {
			if fb, ok := byID[*def.Fallback]; ok && state.Available(fb) {
				h := state.HealthOf(primary)
				fo := &TransactionalFailoverState{From: TransactionalRef{ID: primary.ID.String(), Name: primary.Name}, To: TransactionalRef{ID: fb.ID.String(), Name: fb.Name}, Since: now}
				if h.Down != nil {
					fo.Since, fo.Recovering = h.Down.Since, h.Down.RecoveringSince != nil
				}
				if h.LastError != nil {
					fo.Code = h.LastError.Code
					if fo.Code == "" && h.LastError.Status > 0 {
						fo.Code = itoa(h.LastError.Status)
					}
				}
				out.Failover = fo
			}
		}
	} else if def, ok := set.Default(providers.StreamSystem); ok && len(def.Splits) > 0 {
		out.Route.System = ref(def.Splits[0].ProviderID)
	}
	if out.Route.Priority, err = s.Q.TxnRoutePriority(ctx, ws.ID); err != nil {
		return out, err
	}

	// Alerts, open and of the last 30 days.
	rows, err := s.Q.TxnAlerts(ctx, store.TxnAlertsParams{WorkspaceID: ws.ID, Since: now.Add(-30 * 24 * time.Hour)})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		a := alertOf(r)
		out.Incidents = append(out.Incidents, a)
		if r.EndedAt != nil {
			continue
		}
		if r.Kind == AlertFailover {
			if out.Failover != nil {
				out.Failover.Messages, out.Failover.Lost = a.Messages, a.Lost
				if a.MedianMs > 0 {
					m := a.MedianMs
					out.Failover.MedianMs = &m
				}
			}
			continue
		}
		out.Alerts = append(out.Alerts, a)
	}
	h, err := s.Q.TxnHourStats(ctx, store.TxnHourStatsParams{WorkspaceID: ws.ID, Since: now.Add(-time.Hour)})
	if err != nil {
		return out, err
	}
	out.Hour = TransactionalHour{Sent: h.Sent, Bounced: h.Bounced, BounceRate: ratePct(h.Bounced, h.Sent)}
	if h.MedianMs >= 0 {
		m := h.MedianMs
		out.Hour.MedianMs = &m
	}

	// Keys that can send, and the first run.
	keys, err := s.sendingKeys(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Keys = keys
	fr, err := s.Q.TxnFirstRun(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.FirstRun = TransactionalFirstRun{Templates: fr.Templates, Published: fr.Published, Messages: fr.Messages, Keys: len(keys)}
	senders, err := s.transactionalSenders(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.FirstRun.Senders = len(senders)
	for _, sn := range senders {
		out.FirstRun.VerifiedSender = out.FirstRun.VerifiedSender || sn.Verified
	}
	return out, nil
}

// sendingKeys lists the keys that hold transactional:send.
func (s *Service) sendingKeys(ctx context.Context, wsID uuid.UUID) ([]TransactionalSendingKey, error) {
	rows, err := s.Q.TxnSendingKeys(ctx, wsID)
	if err != nil {
		return nil, err
	}
	cats, err := s.Q.TxnRouteCategories(ctx, wsID)
	if err != nil {
		return nil, err
	}
	names := map[uuid.UUID]string{}
	for _, c := range cats {
		names[c.ID] = c.Name
	}
	now := s.now()
	out := make([]TransactionalSendingKey, 0, len(rows))
	for _, r := range rows {
		k := TransactionalSendingKey{ID: r.ID.String(), Name: r.Name, Masked: r.Prefix, Preset: r.Preset, Mode: r.Mode, Templates: nonNilStrings(r.Templates),
			Categories: []string{}, TestAddresses: nonNilStrings(r.TestAddresses), RatePerSecond: int(r.RatePerSecond), CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName,
			ExpiresAt: r.ExpiresAt, LastUsedAt: r.LastUsedAt}
		for _, c := range r.Categories {
			if n := names[c]; n != "" {
				k.Categories = append(k.Categories, n)
			}
		}
		if r.ExpiresAt != nil && !r.ExpiresAt.After(now) {
			k.Expired = true
		}
		if r.LastUsedIp != nil {
			k.LastUsedIP = r.LastUsedIp.String()
		}
		out = append(out, k)
	}
	return out, nil
}
