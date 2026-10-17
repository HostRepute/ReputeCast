package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Statuses of a catalog entry.
const (
	StatusNotConnected = "not_connected"
	StatusWaiting      = "waiting"
	StatusConnected    = "connected"
	StatusAttention    = "attention"
)

// loaded is a connection with what its views and health need.
type loaded struct {
	conn  store.Integration
	by    string
	keys  []store.IntegrationKeysRow
	hooks []store.IntegrationWebhooksRow
	site  *store.WordpressSite
	rs    []IntegrationReason
}

func (l loaded) status() string {
	switch {
	case len(l.rs) > 0:
		return StatusAttention
	case l.site != nil && l.site.FirstReportAt == nil, l.site == nil && l.conn.FirstCallAt == nil:
		return StatusWaiting
	}
	return StatusConnected
}

func (s *Service) load(ctx context.Context, ws *store.Workspace, conn store.Integration, by string) (loaded, error) {
	l := loaded{conn: conn, by: by}
	var err error
	if l.keys, err = s.Q.IntegrationKeys(ctx, &conn.ID); err != nil {
		return l, err
	}
	if l.hooks, err = s.Q.IntegrationWebhooks(ctx, conn.ID); err != nil {
		return l, err
	}
	if conn.Kind == KindWordPress {
		st, err := s.Q.WordPressSite(ctx, conn.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return l, err
		}
		if err == nil {
			l.site = &st
		}
	}
	l.rs = s.reasons(ctx, ws.ID, conn, l.keys, l.hooks, l.site)
	return l, nil
}

// active loads the workspace's connections.
func (s *Service) active(ctx context.Context, ws *store.Workspace) ([]loaded, error) {
	rows, err := s.Q.IntegrationsActive(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]loaded, 0, len(rows))
	for _, r := range rows {
		l, err := s.load(ctx, ws, integrationOf(r.ID, r.WorkspaceID, r.Kind, r.Label, r.Items, r.OffItems, r.Settings, r.ConnectedBy, r.ConnectedAt,
			r.FirstCallAt, r.LastCallAt, r.Attention, r.Notified, r.DisconnectedAt, r.DisconnectedBy), r.ConnectedByName)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

func integrationOf(id, ws uuid.UUID, kind, label string, items, off []string, settings json.RawMessage, by *uuid.UUID, at time.Time,
	first, last *time.Time, attention json.RawMessage, notified []string, disc *time.Time, discBy *uuid.UUID) store.Integration {
	return store.Integration{ID: id, WorkspaceID: ws, Kind: kind, Label: label, Items: items, OffItems: off, Settings: settings, ConnectedBy: by,
		ConnectedAt: at, FirstCallAt: first, LastCallAt: last, Attention: attention, Notified: notified, DisconnectedAt: disc, DisconnectedBy: discBy}
}

// GroupSource returns the cards another domain adds to a catalog group.
type GroupSource func(ctx context.Context, ws *store.Workspace) ([]IntegrationEntry, error)

// IntegrationHub is Settings › Integrations: the catalog.
type IntegrationHub struct {
	Groups []IntegrationGroup `json:"groups" doc:"Groups with something in them, in order"`
	// Alerts are the problems the banners name, worst first.
	Alerts []IntegrationAlert `json:"alerts"`
	// CanManage: the caller holds integrations:manage; Helpers hold it.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// IntegrationGroup is one group of the catalog.
type IntegrationGroup struct {
	Key     string             `json:"key" enum:"automation,websites,deliverability,developer,ai,verification,payments,commerce"`
	Entries []IntegrationEntry `json:"entries"`
}

// IntegrationEntry is a catalog card: its status, the reason it needs
// attention, and the facts of its fact line.
type IntegrationEntry struct {
	Key string `json:"key" enum:"zapier,make,n8n,wordpress,hostrepute,api,webhooks,mcp,zerobounce,neverbounce,kickbox,bouncer,emailable"`
	// Status: not_connected, waiting, connected, attention; none for the
	// developer cards.
	Status          string              `json:"status" enum:"not_connected,waiting,connected,attention,none"`
	UpdateAvailable bool                `json:"updateAvailable"`
	ConnectionID    string              `json:"connectionId,omitempty"`
	Reasons         []IntegrationReason `json:"reasons"`
	Facts           IntegrationFacts    `json:"facts"`
}

// IntegrationFacts feed a card's fact line ("3 Zaps · last call 2 min ago").
type IntegrationFacts struct {
	Workflows    int64      `json:"workflows"`
	LastCallAt   *time.Time `json:"lastCallAt,omitempty"`
	KeyCreatedAt *time.Time `json:"keyCreatedAt,omitempty"`
	Triggers     int        `json:"triggers"`
	Actions      int        `json:"actions"`
	Sites        int        `json:"sites"`
	SilentSites  int        `json:"silentSites"`
	Updates      int        `json:"updates"`
	Team         string     `json:"team,omitempty"`
	LastSyncAt   *time.Time `json:"lastSyncAt,omitempty"`
	Endpoints    int64      `json:"endpoints"`
	Failing      int64      `json:"failing"`
	MCPEnabled   bool       `json:"mcpEnabled"`
	Agents       int64      `json:"agents"`
	// Balance is a verification service's credits and Price what a check
	// costs on the workspace's plan, when connected.
	Balance *int64  `json:"balance,omitempty"`
	Price   float64 `json:"price,omitempty"`
}

// IntegrationAlert is a banner: an entry and its reason.
type IntegrationAlert struct {
	Entry        string            `json:"entry"`
	ConnectionID string            `json:"connectionId,omitempty"`
	Reason       IntegrationReason `json:"reason"`
	// Address is n8n's address, for "Open n8n".
	Address string `json:"address,omitempty"`
}

func countItems(kind string) (triggers, actions int) {
	for _, it := range Items(kind) {
		if it.Kind == ItemTrigger {
			triggers++
		} else {
			actions++
		}
	}
	return
}

func (s *Service) workflowCount(ctx context.Context, id uuid.UUID) int64 {
	wfs, err := s.Q.IntegrationWorkflows(ctx, store.IntegrationWorkflowsParams{IntegrationID: id, Since: s.now().Add(-WorkflowRetention)})
	if err != nil {
		return 0
	}
	return int64(len(wfs))
}

// Hub builds the catalog.
func (s *Service) Hub(ctx context.Context, ws *store.Workspace, p *authz.Principal) (IntegrationHub, error) {
	conns, err := s.active(ctx, ws)
	if err != nil {
		return IntegrationHub{}, err
	}
	out := IntegrationHub{Groups: []IntegrationGroup{}, Alerts: []IntegrationAlert{}, CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws, authz.IntegrationsManage)}
	latest := s.LatestPlugin(ctx)
	entries := map[string]IntegrationEntry{}
	for _, kind := range []string{KindZapier, KindMake, KindN8n} {
		t, a := countItems(kind)
		e := IntegrationEntry{Key: kind, Status: StatusNotConnected, Reasons: []IntegrationReason{}, Facts: IntegrationFacts{Triggers: t, Actions: a}}
		for _, l := range conns {
			if l.conn.Kind != kind {
				continue
			}
			e.Status = l.status()
			e.ConnectionID = l.conn.ID.String()
			e.Reasons = l.rs
			e.Facts.Workflows = s.workflowCount(ctx, l.conn.ID)
			e.Facts.LastCallAt = l.conn.LastCallAt
			if cur, _ := current(l.keys, s.now()); cur != nil {
				e.Facts.KeyCreatedAt = &cur.CreatedAt
			}
			for _, r := range l.rs {
				out.Alerts = append(out.Alerts, IntegrationAlert{Entry: kind, ConnectionID: e.ConnectionID, Reason: r, Address: settingString(l.conn.Settings, "address")})
			}
		}
		entries[kind] = e
	}
	wp := IntegrationEntry{Key: KindWordPress, Status: StatusNotConnected, Reasons: []IntegrationReason{}}
	waiting := 0
	for _, l := range conns {
		if l.conn.Kind != KindWordPress {
			continue
		}
		wp.Facts.Sites++
		if l.status() == StatusWaiting {
			waiting++
		}
		for _, r := range l.rs {
			if r.Code == ReasonSiteSilent {
				wp.Facts.SilentSites++
			}
			wp.Reasons = append(wp.Reasons, r)
			out.Alerts = append(out.Alerts, IntegrationAlert{Entry: KindWordPress, ConnectionID: l.conn.ID.String(), Reason: r})
		}
		if l.site != nil && l.site.PluginVersion != "" && newerVersion(latest, l.site.PluginVersion) {
			wp.Facts.Updates++
		}
	}
	switch {
	case wp.Facts.Sites == 0:
	case len(wp.Reasons) > 0:
		wp.Status = StatusAttention
	case waiting == wp.Facts.Sites:
		wp.Status = StatusWaiting
	default:
		wp.Status = StatusConnected
	}
	wp.UpdateAvailable = wp.Facts.Updates > 0
	entries[KindWordPress] = wp
	hr := IntegrationEntry{Key: "hostrepute", Status: StatusNotConnected, Reasons: []IntegrationReason{}}
	if c, err := s.Q.GetHostReputeConnection(ctx, ws.ID); err == nil {
		hr.Facts.Team, hr.Facts.LastSyncAt = c.TeamName, c.LastSyncAt
		hr.Status = StatusConnected
		if c.Status == "revoked" || c.Status == "expired" {
			hr.Status = StatusAttention
		}
	}
	entries["hostrepute"] = hr
	entries["api"] = IntegrationEntry{Key: "api", Status: "none", Reasons: []IntegrationReason{}}
	wh := IntegrationEntry{Key: "webhooks", Status: "none", Reasons: []IntegrationReason{}}
	if c, err := s.Q.IntegrationWebhookCounts(ctx, ws.ID); err == nil {
		wh.Facts.Endpoints, wh.Facts.Failing = c.Endpoints, c.Failing
	}
	entries["webhooks"] = wh
	mcp := IntegrationEntry{Key: "mcp", Status: "none", Reasons: []IntegrationReason{}}
	var mset struct {
		Enabled bool `json:"enabled"`
	}
	if json.Unmarshal(ws.McpSettings, &mset) == nil {
		mcp.Facts.MCPEnabled = mset.Enabled
	}
	if n, err := s.Q.HostReputeCountAgents(ctx, ws.ID); err == nil {
		mcp.Facts.Agents = n
	}
	entries["mcp"] = mcp
	for _, g := range Groups {
		grp := IntegrationGroup{Key: g.Key, Entries: []IntegrationEntry{}}
		for _, k := range g.Entries {
			grp.Entries = append(grp.Entries, entries[k])
		}
		if src := s.Sources[g.Key]; src != nil {
			if more, err := src(ctx, ws); err == nil {
				grp.Entries = append(grp.Entries, more...)
			}
		}
		if len(grp.Entries) == 0 {
			continue
		}
		out.Groups = append(out.Groups, grp)
	}
	// Worst first: rejected keys and owners who left, then failing
	// deliveries and calls, then silent sites and expiring keys.
	rank := map[string]int{ReasonKeyRevoked: 0, ReasonKeyExpired: 0, ReasonOwnerLeft: 0, ReasonDeliveriesFailing: 1, ReasonCallsFailing: 1, ReasonSiteSilent: 2, ReasonKeyExpiring: 3}
	slices.SortStableFunc(out.Alerts, func(a, b IntegrationAlert) int { return rank[a.Reason.Code] - rank[b.Reason.Code] })
	return out, nil
}

func settingString(raw json.RawMessage, key string) string {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	v, _ := m[key].(string)
	return v
}

// helpers names the members who hold perm, Admins first ("Ask Marcus Chen
// or Linh Nguyen", as the read-only lines say).
func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	out := []string{}
	rows, err := s.Q.IntegrationMembers(ctx, ws.ID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == 2 {
				break
			}
		}
	}
	return out
}

// IntegrationItemState is a trigger or action of an integration: whether
// it's on, available, and why not.
type IntegrationItemState struct {
	Key         string   `json:"key"`
	Kind        string   `json:"kind" enum:"trigger,action"`
	Event       string   `json:"event,omitempty"`
	Permissions []string `json:"permissions"`
	Sensitive   bool     `json:"sensitive"`
	New         bool     `json:"new"`
	On          bool     `json:"on"`
	Available   bool     `json:"available"`
	// OffReason: turned_off (when connecting or since), or owner_lacks (the
	// key's owner doesn't hold Missing).
	OffReason string `json:"offReason,omitempty" enum:"turned_off,owner_lacks,"`
	Missing   string `json:"missing,omitempty"`
	Owner     string `json:"owner,omitempty"`
}

func (s *Service) itemStates(ctx context.Context, ws *store.Workspace, l loaded) []IntegrationItemState {
	var ownerPerms authz.Set
	owner := ""
	if cur, _ := current(l.keys, s.now()); cur != nil && cur.CreatedBy != nil {
		owner = cur.OwnerName
		if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: *cur.CreatedBy}); err == nil && m.Status == "active" {
			ownerPerms = authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)
		}
	}
	out := []IntegrationItemState{}
	for _, it := range Items(l.conn.Kind) {
		st := IntegrationItemState{Key: it.Key, Kind: it.Kind, Event: it.Event, Permissions: permStrings(it.Permissions), Sensitive: it.Sensitive(), New: it.New,
			On: slices.Contains(l.conn.Items, it.Key)}
		switch {
		case !st.On:
			st.OffReason = "turned_off"
		default:
			st.Available = true
			for _, need := range it.Permissions {
				if ownerPerms != nil && !ownerPerms.Has(need) {
					st.Available, st.OffReason, st.Missing, st.Owner = false, "owner_lacks", string(need), owner
					break
				}
			}
		}
		out = append(out, st)
	}
	return out
}

func permStrings(ps []authz.Permission) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = string(p)
	}
	return out
}

// IntegrationApp is an automation platform's page (Zapier, Make, n8n).
type IntegrationApp struct {
	Kind         string              `json:"kind" enum:"zapier,make,n8n"`
	Status       string              `json:"status" enum:"not_connected,waiting,connected,attention"`
	Reasons      []IntegrationReason `json:"reasons"`
	ConnectionID string              `json:"connectionId,omitempty"`
	Label        string              `json:"label,omitempty"`
	ConnectedBy  *IntegrationPerson  `json:"connectedBy,omitempty"`
	ConnectedAt  *time.Time          `json:"connectedAt,omitempty"`
	// Address is n8n's address ("https://n8n.acme.io").
	Address   string                   `json:"address,omitempty"`
	Numbers   IntegrationNumbers       `json:"numbers"`
	Workflows []IntegrationWorkflowRow `json:"workflows"`
	Webhooks  []IntegrationManagedHook `json:"webhooks"`
	Items     []IntegrationItemState   `json:"items"`
	Key       *IntegrationKeyView      `json:"key,omitempty"`
	// OldKey is the rotated key while it still works.
	OldKey      *IntegrationKeyView `json:"oldKey,omitempty"`
	InstanceURL string              `json:"instanceUrl"`
	// Receives lists the triggers in use (with a managed webhook).
	Receives []string `json:"receives"`
	// Rejected are the calls refused since the key stopped working.
	Rejected []IntegrationRejectedCall `json:"rejected"`
	// FirstCall is the key's first call (the waiting timeline).
	FirstCall    *IntegrationFirstCall   `json:"firstCall,omitempty"`
	Reachability IntegrationReachability `json:"reachability"`
	CanManage    bool                    `json:"canManage"`
	Helpers      []string                `json:"helpers"`
	Triggers     int                     `json:"triggers"`
	Actions      int                     `json:"actions"`
}

// IntegrationNumbers are the four numbers of a page.
type IntegrationNumbers struct {
	Calls7d          int64      `json:"calls7d"`
	Errors7d         int64      `json:"errors7d"`
	Fixed7d          int64      `json:"fixed7d" doc:"Errors a later call from the same workflow got past"`
	ConsentKept7d    int64      `json:"consentKept7d" doc:"Calls refused to keep consent (409)"`
	LastCallAt       *time.Time `json:"lastCallAt,omitempty"`
	LastCallWorkflow string     `json:"lastCallWorkflow,omitempty"`
	ManagedWebhooks  int        `json:"managedWebhooks"`
}

// IntegrationWorkflowRow is a workflow seen in the last 30 days.
type IntegrationWorkflowRow struct {
	Key     string        `json:"key"`
	Name    string        `json:"name"`
	Unnamed bool          `json:"unnamed"`
	Uses    []WorkflowUse `json:"uses"`
	Calls7d int64         `json:"calls7d"`
	// Errors7d splits into failures (FixedAt when later runs went through)
	// and calls refused to keep consent.
	Errors7d      int64      `json:"errors7d"`
	ConsentKept7d int64      `json:"consentKept7d"`
	FixedAt       *time.Time `json:"fixedAt,omitempty"`
	LastRunAt     time.Time  `json:"lastRunAt"`
	// Deliveries7d counts what its trigger's managed webhook delivered.
	Deliveries7d int64 `json:"deliveries7d"`
	Trigger      bool  `json:"trigger"`
}

// IntegrationManagedHook is a webhook an integration manages.
type IntegrationManagedHook struct {
	ID           string     `json:"id"`
	Event        string     `json:"event"`
	Endpoint     string     `json:"endpoint" doc:"Masked: https://n8n.acme.io/webhook/7c1e…/reputecast"`
	CreatedAt    time.Time  `json:"createdAt"`
	Workflow     string     `json:"workflow,omitempty"`
	Delivered24h int64      `json:"delivered24h"`
	Total24h     int64      `json:"total24h"`
	Retrying     int64      `json:"retrying"`
	Queued       int64      `json:"queued"`
	Dead         int64      `json:"dead"`
	Status       string     `json:"status" enum:"new,healthy,failing,dead_letters"`
	FailingSince *time.Time `json:"failingSince,omitempty"`
	LastStatus   *int32     `json:"lastStatus,omitempty"`
	StatusText   string     `json:"statusText,omitempty"`
	NextAt       *time.Time `json:"nextAt,omitempty"`
	// LastAttempt is the oldest delivery's latest attempt while failing.
	LastAttempt *IntegrationAttempt `json:"lastAttempt,omitempty"`
	// RecoveredAt is when a failing endpoint answered 2xx again (dead letters wait).
	RecoveredAt *time.Time `json:"recoveredAt,omitempty"`
}

// IntegrationAttempt is one delivery attempt of a failing managed webhook.
type IntegrationAttempt struct {
	Event       string     `json:"event"`
	Attempt     int32      `json:"attempt"`
	MaxAttempts int32      `json:"maxAttempts"`
	Status      *int32     `json:"status,omitempty"`
	StatusText  string     `json:"statusText,omitempty"`
	At          *time.Time `json:"at,omitempty"`
	Queued      int64      `json:"queued"`
}

// IntegrationRejectedCall is a call refused because the key stopped working.
type IntegrationRejectedCall struct {
	At       time.Time `json:"at"`
	Workflow string    `json:"workflow"`
	Status   int32     `json:"status"`
	Code     string    `json:"code"`
}

// IntegrationFirstCall is a key's first call.
type IntegrationFirstCall struct {
	At        time.Time `json:"at"`
	Operation string    `json:"operation"`
	Status    int32     `json:"status"`
	IP        string    `json:"ip,omitempty"`
	Client    string    `json:"client,omitempty"`
}

func (s *Service) managedHooks(ctx context.Context, l loaded) []IntegrationManagedHook {
	out := []IntegrationManagedHook{}
	for _, h := range l.hooks {
		v := IntegrationManagedHook{ID: h.ID.String(), Event: h.Event, Endpoint: maskEndpoint(h.Url), CreatedAt: h.CreatedAt, Workflow: h.WorkflowName,
			Delivered24h: h.Delivered24h, Total24h: h.Total24h, Retrying: h.Retrying, Queued: h.Queued, Dead: h.Dead, FailingSince: h.FailingSince,
			LastStatus: h.LastStatus, NextAt: asTime(h.NextAttemptAt), Status: "healthy"}
		if h.LastStatus != nil {
			v.StatusText = http.StatusText(int(*h.LastStatus))
		}
		switch {
		case h.FailingSince != nil:
			v.Status = "failing"
			if a, err := s.Q.IntegrationLastAttempt(ctx, h.ID); err == nil {
				v.LastAttempt = &IntegrationAttempt{Event: a.Event, Attempt: a.Attempt, MaxAttempts: a.MaxAttempts, Status: a.StatusCode, At: a.LastAttemptAt, Queued: h.Queued}
				if a.StatusCode != nil {
					v.LastAttempt.StatusText = http.StatusText(int(*a.StatusCode))
				}
			}
		case h.Dead > 0:
			v.Status = "dead_letters"
			v.RecoveredAt = h.LastDeliveryAt
		case h.LastDeliveryAt == nil:
			v.Status = "new"
		}
		out = append(out, v)
	}
	return out
}

// App builds an automation platform's page.
func (s *Service) App(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string) (IntegrationApp, error) {
	if !IsPlatform(kind) {
		return IntegrationApp{}, api.NotFound("Integration")
	}
	t, a := countItems(kind)
	out := IntegrationApp{Kind: kind, Status: StatusNotConnected, Reasons: []IntegrationReason{}, Workflows: []IntegrationWorkflowRow{}, Webhooks: []IntegrationManagedHook{},
		Items: []IntegrationItemState{}, Receives: []string{}, Rejected: []IntegrationRejectedCall{}, InstanceURL: s.base(), CanManage: p.Can(authz.IntegrationsManage),
		Helpers: s.helpers(ctx, ws, authz.IntegrationsManage), Triggers: t, Actions: a}
	row, err := s.Q.IntegrationActiveByKind(ctx, store.IntegrationActiveByKindParams{WorkspaceID: ws.ID, Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		for _, it := range Items(kind) {
			out.Items = append(out.Items, IntegrationItemState{Key: it.Key, Kind: it.Kind, Event: it.Event, Permissions: permStrings(it.Permissions),
				Sensitive: it.Sensitive(), New: it.New, On: !it.Sensitive(), Available: true})
		}
		out.Reachability = s.Reachability(ctx)
		return out, nil
	}
	if err != nil {
		return out, err
	}
	l, err := s.load(ctx, ws, integrationOf(row.ID, row.WorkspaceID, row.Kind, row.Label, row.Items, row.OffItems, row.Settings, row.ConnectedBy, row.ConnectedAt,
		row.FirstCallAt, row.LastCallAt, row.Attention, row.Notified, row.DisconnectedAt, row.DisconnectedBy), row.ConnectedByName)
	if err != nil {
		return out, err
	}
	out.Status, out.Reasons, out.ConnectionID, out.Label = l.status(), l.rs, row.ID.String(), row.Label
	out.ConnectedAt = &row.ConnectedAt
	if row.ConnectedBy != nil {
		out.ConnectedBy = &IntegrationPerson{ID: row.ConnectedBy.String(), Name: firstNonEmpty(row.ConnectedByName, "Former member")}
	}
	out.Address = settingString(row.Settings, "address")
	out.Items = s.itemStates(ctx, ws, l)
	out.Webhooks = s.managedHooks(ctx, l)
	lists, _ := access.ListNames(ctx, s.Q, ws.ID)
	cur, old := current(l.keys, s.now())
	if cur != nil {
		v := keyView(*cur, lists, s.now())
		out.Key = &v
		if fc, err := s.Q.IntegrationFirstCall(ctx, store.IntegrationFirstCallParams{IntegrationID: row.ID, ApiKeyID: &cur.ID}); err == nil {
			out.FirstCall = &IntegrationFirstCall{At: fc.OccurredAt, Operation: fc.Operation, Status: fc.Status, Client: fc.Client}
			if fc.Ip != nil {
				out.FirstCall.IP = fc.Ip.String()
			}
		}
	}
	if old != nil {
		v := keyView(*old, lists, s.now())
		out.OldKey = &v
	}
	for _, h := range l.hooks {
		if !slices.Contains(out.Receives, h.Event) {
			out.Receives = append(out.Receives, h.Event)
		}
	}
	out.Numbers, out.Workflows, err = s.numbers(ctx, l)
	if err != nil {
		return out, err
	}
	for _, r := range l.rs {
		if r.Code == ReasonKeyRevoked || r.Code == ReasonKeyExpired || r.Code == ReasonOwnerLeft {
			if r.Since != nil {
				rows, err := s.Q.IntegrationCallsRejected(ctx, store.IntegrationCallsRejectedParams{IntegrationID: row.ID, Since: *r.Since, Codes: rejectedCodes})
				if err == nil {
					for _, c := range rows {
						out.Rejected = append(out.Rejected, IntegrationRejectedCall{At: c.OccurredAt, Workflow: c.WorkflowName, Status: c.Status, Code: c.Code})
					}
				}
			}
		}
	}
	out.Reachability = s.Reachability(ctx)
	return out, nil
}

// numbers computes the page's numbers and its workflows.
func (s *Service) numbers(ctx context.Context, l loaded) (IntegrationNumbers, []IntegrationWorkflowRow, error) {
	since := s.now().Add(-CallRetention)
	n := IntegrationNumbers{LastCallAt: l.conn.LastCallAt, ManagedWebhooks: len(l.hooks)}
	counts, err := s.Q.IntegrationCallCounts(ctx, store.IntegrationCallCountsParams{IntegrationID: l.conn.ID, Since: since})
	if err != nil {
		return n, nil, err
	}
	n.Calls7d, n.Errors7d, n.ConsentKept7d = counts.Calls, counts.Errors, counts.ConsentKept
	per, err := s.Q.IntegrationCallsByWorkflow(ctx, store.IntegrationCallsByWorkflowParams{IntegrationID: l.conn.ID, Since: since})
	if err != nil {
		return n, nil, err
	}
	byKey := map[string]store.IntegrationCallsByWorkflowRow{}
	for _, r := range per {
		byKey[r.WorkflowKey] = r
	}
	wfs, err := s.Q.IntegrationWorkflows(ctx, store.IntegrationWorkflowsParams{IntegrationID: l.conn.ID, Since: s.now().Add(-WorkflowRetention)})
	if err != nil {
		return n, nil, err
	}
	deliveries := map[string]int64{}
	delivered := map[string]time.Time{}
	for _, h := range l.hooks {
		deliveries[h.WorkflowKey] += h.Deliveries7d
		if h.LastDeliveryAt != nil && h.LastDeliveryAt.After(delivered[h.WorkflowKey]) {
			delivered[h.WorkflowKey] = *h.LastDeliveryAt
		}
	}
	out := []IntegrationWorkflowRow{}
	var flows *eventFlows
	for _, w := range wfs {
		row := IntegrationWorkflowRow{Key: w.Key, Name: w.Name, Unnamed: w.Name == "", Uses: []WorkflowUse{}, LastRunAt: w.LastSeenAt}
		_ = json.Unmarshal(w.Uses, &row.Uses)
		for i, u := range row.Uses {
			if it, ok := ItemByKey(l.conn.Kind, u.Item); ok && it.Kind == ItemTrigger {
				row.Trigger = true
			}
			if u.Event != "" {
				if flows == nil {
					flows = s.eventFlows(ctx, l.conn.WorkspaceID)
				}
				row.Uses[i].Starts, row.Uses[i].GoalOf = flows.starts[u.Event], flows.goals[u.Event]
			}
		}
		if c, ok := byKey[w.Key]; ok {
			row.Calls7d, row.Errors7d, row.ConsentKept7d = c.Calls, c.Errors, c.ConsentKept
		}
		row.Deliveries7d = deliveries[w.Key]
		out = append(out, row)
	}
	// Resolved errors: a later call from the same workflow and action went
	// through (or the error carries a note saying who fixed it and when).
	calls, err := s.Q.IntegrationCalls(ctx, store.IntegrationCallsParams{IntegrationID: l.conn.ID, Since: since, ErrorsOnly: true, Lim: 500})
	if err != nil {
		return n, nil, err
	}
	for _, c := range calls {
		if c.Status == http.StatusConflict {
			continue
		}
		res := s.resolution(ctx, l.conn.ID, c.WorkflowKey, c.Item, c.OccurredAt, c.Note)
		if res == nil {
			continue
		}
		n.Fixed7d++
		for i := range out {
			if out[i].Key == c.WorkflowKey && (out[i].FixedAt == nil || res.FixedAt != nil && res.FixedAt.After(*out[i].FixedAt)) {
				out[i].FixedAt = res.FixedAt
			}
		}
	}
	if len(out) > 0 {
		// The workflow of the last call.
		latest := out[0]
		for _, w := range out {
			if w.LastRunAt.After(latest.LastRunAt) {
				latest = w
			}
		}
		n.LastCallWorkflow = latest.Name
	}
	// A trigger's workflow last ran when its webhook last delivered.
	for i := range out {
		if d, ok := delivered[out[i].Key]; ok && d.After(out[i].LastRunAt) {
			out[i].LastRunAt = d
		}
	}
	return n, out, nil
}

// eventFlows are the live automations an API event starts, and those that
// have it as a goal, by event.
type eventFlows struct {
	starts, goals map[string][]string
}

func (s *Service) eventFlows(ctx context.Context, wsID uuid.UUID) *eventFlows {
	out := &eventFlows{starts: map[string][]string{}, goals: map[string][]string{}}
	rows, err := s.Q.IntegrationEventFlows(ctx, wsID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		var t struct {
			Kind  string `json:"kind"`
			Event string `json:"event"`
		}
		if json.Unmarshal(r.Trigger, &t) == nil && t.Kind == "api_event" && t.Event != "" {
			out.starts[t.Event] = append(out.starts[t.Event], r.Name)
		}
		var g struct {
			Nodes []struct {
				Type string `json:"type"`
				Data struct {
					Condition *struct {
						Kind  string `json:"kind"`
						Event string `json:"event"`
					} `json:"condition"`
				} `json:"data"`
			} `json:"nodes"`
		}
		if json.Unmarshal(r.Graph, &g) != nil {
			continue
		}
		for _, n := range g.Nodes {
			if c := n.Data.Condition; n.Type == "goal" && c != nil && c.Kind == "event" && c.Event != "" && !slices.Contains(out.goals[c.Event], r.Name) {
				out.goals[c.Event] = append(out.goals[c.Event], r.Name)
			}
		}
	}
	return out
}

// IntegrationResolution says an error is behind us: who fixed it and when,
// or the later runs that went through.
type IntegrationResolution struct {
	FixedAt   *time.Time  `json:"fixedAt,omitempty"`
	FixedBy   string      `json:"fixedBy,omitempty"`
	LaterRuns []time.Time `json:"laterRuns"`
}

// resolution works out whether an error was resolved.
func (s *Service) resolution(ctx context.Context, id uuid.UUID, workflow, item string, at time.Time, note string) *IntegrationResolution {
	later, err := s.Q.IntegrationCallsLater(ctx, store.IntegrationCallsLaterParams{IntegrationID: id, WorkflowKey: workflow, Item: item, After: at})
	if err != nil || len(later) == 0 {
		return nil
	}
	r := &IntegrationResolution{LaterRuns: later}
	var n struct {
		FixedAt *time.Time `json:"fixedAt"`
		FixedBy string     `json:"fixedBy"`
	}
	if note != "" && json.Unmarshal([]byte(note), &n) == nil {
		r.FixedAt, r.FixedBy = n.FixedAt, n.FixedBy
	}
	if r.FixedAt == nil {
		r.FixedAt = &later[0]
	}
	return r
}

// IntegrationCallFilter narrows the call log.
type IntegrationCallFilter struct {
	ErrorsOnly bool
	Window     string // 24h or 7d
	Offset     int
	Limit      int
}

// IntegrationCallList is a page of the call log.
type IntegrationCallList struct {
	Items  []IntegrationCallView `json:"items"`
	Total  int64                 `json:"total" doc:"Calls in the window"`
	Errors int64                 `json:"errors" doc:"Errors in the window"`
	Offset int                   `json:"offset"`
	Limit  int                   `json:"limit"`
}

// IntegrationCallView is one call of the log.
type IntegrationCallView struct {
	ID         string     `json:"id"`
	At         time.Time  `json:"at"`
	Workflow   string     `json:"workflow"`
	Unnamed    bool       `json:"unnamed"`
	Item       string     `json:"item,omitempty"`
	Operation  string     `json:"operation"`
	Method     string     `json:"method"`
	Path       string     `json:"path"`
	Status     int32      `json:"status"`
	Code       string     `json:"code,omitempty"`
	Detail     string     `json:"detail,omitempty"`
	DurationMs int32      `json:"durationMs"`
	Target     CallTarget `json:"target"`
	// Who is whether the target's address is shown: email, contact (the
	// member can't see contacts), erased, or none.
	Who        string                 `json:"who" enum:"email,contact,erased,none"`
	Missing    []string               `json:"missing,omitempty"`
	Resolution *IntegrationResolution `json:"resolution,omitempty"`
	IP         string                 `json:"ip,omitempty"`
	Client     string                 `json:"client,omitempty"`
}

func (s *Service) callView(ctx context.Context, connID uuid.UUID, canSee bool, id uuid.UUID, at time.Time, wfKey, wfName, op, item, method, path string,
	status int32, code, detail string, dur int32, target json.RawMessage, contactEmail string, erased bool, note string, ip string, client string, response json.RawMessage) IntegrationCallView {
	v := IntegrationCallView{ID: id.String(), At: at, Workflow: wfName, Unnamed: wfName == "", Item: item, Operation: op, Method: method, Path: path,
		Status: status, Code: code, Detail: detail, DurationMs: dur, IP: ip, Client: client, Who: "none"}
	_ = json.Unmarshal(target, &v.Target)
	if contactEmail != "" && !erased && v.Target.Email == "" {
		v.Target.Email = contactEmail
	}
	switch {
	case erased:
		v.Who, v.Target.Email = "erased", ""
		v.Detail = emailRe.ReplaceAllString(v.Detail, "Erased contact")
	case v.Target.Email != "" && !canSee:
		v.Who, v.Target.Email = "contact", ""
		v.Detail = emailRe.ReplaceAllString(v.Detail, "a contact")
	case v.Target.Email != "":
		v.Who = "email"
	}
	if status >= 400 {
		var p problemBody
		if json.Unmarshal(response, &p) == nil {
			v.Missing = p.Missing
		}
		if status != http.StatusConflict && status != http.StatusUnauthorized && status != http.StatusForbidden {
			v.Resolution = s.resolution(ctx, connID, wfKey, item, at, note)
		}
	}
	return v
}

// Calls returns a page of a connection's call log.
func (s *Service) Calls(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, f IntegrationCallFilter) (IntegrationCallList, error) {
	conn, err := s.Q.IntegrationGet(ctx, store.IntegrationGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return IntegrationCallList{}, api.MapError(err, "Integration")
	}
	since := s.now().Add(-CallRetention)
	if f.Window == "24h" {
		since = s.now().Add(-24 * time.Hour)
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	rows, err := s.Q.IntegrationCalls(ctx, store.IntegrationCallsParams{IntegrationID: conn.ID, Since: since, ErrorsOnly: f.ErrorsOnly, Lim: int32(f.Limit), Off: int32(f.Offset)})
	if err != nil {
		return IntegrationCallList{}, err
	}
	counts, err := s.Q.IntegrationCallCounts(ctx, store.IntegrationCallCountsParams{IntegrationID: conn.ID, Since: since})
	if err != nil {
		return IntegrationCallList{}, err
	}
	out := IntegrationCallList{Items: []IntegrationCallView{}, Total: counts.Calls, Errors: counts.Errors, Offset: f.Offset, Limit: f.Limit}
	canSee := p.Can(authz.SubscribersView)
	for _, r := range rows {
		ip := ""
		if r.Ip != nil {
			ip = r.Ip.String()
		}
		resp := json.RawMessage(nil)
		if r.Status >= 400 {
			if full, err := s.Q.IntegrationCallGet(ctx, store.IntegrationCallGetParams{ID: r.ID, IntegrationID: conn.ID}); err == nil {
				resp = full.Response
			}
		}
		out.Items = append(out.Items, s.callView(ctx, conn.ID, canSee, r.ID, r.OccurredAt, r.WorkflowKey, r.WorkflowName, r.Operation, r.Item, r.Method, r.Path,
			r.Status, r.Code, r.Detail, r.DurationMs, r.Target, r.ContactEmail, r.ContactErased, r.Note, ip, r.Client, resp))
	}
	return out, nil
}

// IntegrationCallDetail is one call with its request and response, secrets
// masked and addresses shown by permission.
type IntegrationCallDetail struct {
	Call     IntegrationCallView `json:"call"`
	Headers  map[string]string   `json:"headers"`
	Request  json.RawMessage     `json:"request,omitempty"`
	Response json.RawMessage     `json:"response,omitempty"`
}

// Call returns one call of the log.
func (s *Service) Call(ctx context.Context, ws *store.Workspace, p *authz.Principal, id, callID uuid.UUID) (IntegrationCallDetail, error) {
	conn, err := s.Q.IntegrationGet(ctx, store.IntegrationGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return IntegrationCallDetail{}, api.MapError(err, "Integration")
	}
	r, err := s.Q.IntegrationCallGet(ctx, store.IntegrationCallGetParams{ID: callID, IntegrationID: conn.ID})
	if err != nil {
		return IntegrationCallDetail{}, api.MapError(err, "Call")
	}
	canSee := p.Can(authz.SubscribersView)
	ip := ""
	if r.Ip != nil {
		ip = r.Ip.String()
	}
	out := IntegrationCallDetail{Call: s.callView(ctx, conn.ID, canSee, r.ID, r.OccurredAt, r.WorkflowKey, r.WorkflowName, r.Operation, r.Item, r.Method, r.Path,
		r.Status, r.Code, r.Detail, r.DurationMs, r.Target, r.ContactEmail, r.ContactErased, r.Note, ip, r.Client, r.Response), Headers: map[string]string{}}
	var req CallRequest
	if json.Unmarshal(r.Request, &req) == nil {
		out.Headers = req.Headers
		if out.Headers == nil {
			out.Headers = map[string]string{}
		}
		out.Request = req.Body
	}
	out.Response = r.Response
	switch {
	case r.ContactErased:
		out.Request, out.Response = hideAddresses(out.Request, "Erased contact"), hideAddresses(out.Response, "Erased contact")
	case !canSee:
		out.Request, out.Response = hideAddresses(out.Request, "a contact"), hideAddresses(out.Response, "a contact")
	}
	return out, nil
}

// IntegrationStatus is what the connect dialog's last step polls.
type IntegrationStatus struct {
	Status       string                `json:"status" enum:"waiting,connected,attention"`
	KeyCreatedAt *time.Time            `json:"keyCreatedAt,omitempty"`
	FirstCall    *IntegrationFirstCall `json:"firstCall,omitempty"`
	// Reported is a WordPress site's first report.
	Reported *time.Time `json:"reported,omitempty"`
}

// Status says whether a new key has been called yet.
func (s *Service) Status(ctx context.Context, ws *store.Workspace, id uuid.UUID) (IntegrationStatus, error) {
	conn, err := s.Q.IntegrationGet(ctx, store.IntegrationGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return IntegrationStatus{}, api.MapError(err, "Integration")
	}
	l, err := s.load(ctx, ws, integrationOf(conn.ID, conn.WorkspaceID, conn.Kind, conn.Label, conn.Items, conn.OffItems, conn.Settings, conn.ConnectedBy, conn.ConnectedAt,
		conn.FirstCallAt, conn.LastCallAt, conn.Attention, conn.Notified, conn.DisconnectedAt, conn.DisconnectedBy), conn.ConnectedByName)
	if err != nil {
		return IntegrationStatus{}, err
	}
	out := IntegrationStatus{Status: l.status()}
	if cur, _ := current(l.keys, s.now()); cur != nil {
		out.KeyCreatedAt = &cur.CreatedAt
		if fc, err := s.Q.IntegrationFirstCall(ctx, store.IntegrationFirstCallParams{IntegrationID: conn.ID, ApiKeyID: &cur.ID}); err == nil {
			out.FirstCall = &IntegrationFirstCall{At: fc.OccurredAt, Operation: fc.Operation, Status: fc.Status, Client: fc.Client}
			if fc.Ip != nil {
				out.FirstCall.IP = fc.Ip.String()
			}
		}
	}
	if l.site != nil {
		out.Reported = l.site.FirstReportAt
	}
	return out, nil
}

// IntegrationConnectInfo is the connect dialog's first step.
type IntegrationConnectInfo struct {
	Kind  string                  `json:"kind"`
	Items []IntegrationItemLock   `json:"items"`
	Lists []access.ScopedList     `json:"lists"`
	Name  string                  `json:"name" doc:"The suggested key name, \"Make · Acme ops\""`
	Reach IntegrationReachability `json:"reachability"`
	// ConnectionID is set when reconnecting a connection whose key stopped.
	ConnectionID string `json:"connectionId,omitempty"`
	InstanceURL  string `json:"instanceUrl"`
	// Order and Includes let the dialog show the key's permissions as the
	// switches change: in this order, leaving out a permission another one
	// includes ("subscribers:edit includes subscribers:view").
	Order    []string            `json:"order"`
	Includes map[string][]string `json:"includes"`
}

// permissionIncludes says which of the catalog's permissions each one includes.
func permissionIncludes() ([]string, map[string][]string) {
	order := permStrings(permissionOrder)
	inc := map[string][]string{}
	for _, p := range permissionOrder {
		set := authz.NewSet(p)
		for _, q := range permissionOrder {
			if q != p && set.Has(q) {
				inc[string(p)] = append(inc[string(p)], string(q))
			}
		}
	}
	return order, inc
}

// IntegrationItemLock is a trigger or action in the connect dialog: on by
// default unless it needs a ⚑ permission, locked when the caller doesn't
// hold what it needs.
type IntegrationItemLock struct {
	Key         string   `json:"key"`
	Kind        string   `json:"kind" enum:"trigger,action"`
	Event       string   `json:"event,omitempty"`
	Permissions []string `json:"permissions"`
	Sensitive   bool     `json:"sensitive"`
	New         bool     `json:"new"`
	On          bool     `json:"on"`
	Locked      bool     `json:"locked"`
	// Missing is the permission the caller lacks; Helpers hold it.
	Missing string   `json:"missing,omitempty"`
	Helpers []string `json:"helpers,omitempty"`
}

// ConnectInfo prepares the connect dialog for the caller.
func (s *Service) ConnectInfo(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string) (IntegrationConnectInfo, error) {
	if !Valid(kind) {
		return IntegrationConnectInfo{}, api.NotFound("Integration")
	}
	out := IntegrationConnectInfo{Kind: kind, Items: []IntegrationItemLock{}, Lists: []access.ScopedList{}, Reach: s.Reachability(ctx), InstanceURL: s.base()}
	out.Order, out.Includes = permissionIncludes()
	on := DefaultItems(kind)
	if IsPlatform(kind) {
		if row, err := s.Q.IntegrationActiveByKind(ctx, store.IntegrationActiveByKindParams{WorkspaceID: ws.ID, Kind: kind}); err == nil {
			out.ConnectionID = row.ID.String()
			on = row.Items
			out.Name = Name(kind) + " · " + row.Label
		}
	}
	if out.Name == "" {
		out.Name = Name(kind) + " · " + s.DefaultLabel(ctx, ws)
	}
	for _, it := range Items(kind) {
		lock := IntegrationItemLock{Key: it.Key, Kind: it.Kind, Event: it.Event, Permissions: permStrings(it.Permissions), Sensitive: it.Sensitive(), New: it.New,
			On: slices.Contains(on, it.Key)}
		for _, need := range Needs(it) {
			if !p.Can(need) {
				lock.Locked, lock.On, lock.Missing = true, false, string(need)
				lock.Helpers = access.Helpers(ctx, s.Q, ws.ID, need)
				break
			}
		}
		out.Items = append(out.Items, lock)
	}
	tl, err := s.Q.TeamWorkspaceLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range tl {
		if p.Scope.AllowsList(l.ID) {
			out.Lists = append(out.Lists, access.ScopedList{ID: l.ID.String(), Name: l.Name})
		}
	}
	return out, nil
}

// IntegrationOwned is an integration whose key a member owns.
type IntegrationOwned struct {
	ConnectionID string `json:"connectionId"`
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	KeyName      string `json:"keyName"`
	Workflows    int64  `json:"workflows" doc:"Workflows seen in the last 30 days"`
	// Silent is when a WordPress site last reported in, if it went quiet.
	SilentSince *time.Time `json:"silentSince,omitempty"`
}

// IntegrationOwnedList is what the deactivate and remove dialogs list.
type IntegrationOwnedList struct {
	Items []IntegrationOwned `json:"items"`
	// CanManage: the caller can take them over or disconnect them.
	CanManage bool `json:"canManage"`
}

// OwnedBy lists the integrations whose keys a member owns.
func (s *Service) OwnedBy(ctx context.Context, ws *store.Workspace, p *authz.Principal, user uuid.UUID) (IntegrationOwnedList, error) {
	rows, err := s.Q.IntegrationsOwnedBy(ctx, store.IntegrationsOwnedByParams{WorkspaceID: ws.ID, UserID: &user})
	if err != nil {
		return IntegrationOwnedList{}, err
	}
	out := IntegrationOwnedList{Items: []IntegrationOwned{}, CanManage: p.Can(authz.IntegrationsManage)}
	for _, r := range rows {
		o := IntegrationOwned{ConnectionID: r.ID.String(), Kind: r.Kind, Label: r.Label, KeyName: Name(r.Kind) + " · " + r.Label, Workflows: s.workflowCount(ctx, r.ID)}
		if r.Kind == KindWordPress {
			if st, err := s.Q.WordPressSite(ctx, r.ID); err == nil && st.LastReportAt != nil && s.now().Sub(*st.LastReportAt) >= SiteSilentAfter {
				o.SilentSince = st.LastReportAt
			}
		}
		out.Items = append(out.Items, o)
	}
	return out, nil
}

// IntegrationManagedKey is a key in Keys & MCP's "Managed by integrations".
type IntegrationManagedKey struct {
	IntegrationKeyView
	ConnectionID string `json:"connectionId"`
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	// Workflows are the workflows that stop if the key is revoked.
	Workflows []IntegrationWorkflowRef `json:"workflows"`
	Calls7d   int64                    `json:"calls7d"`
}

// IntegrationManagedKeyList is Keys & MCP's second table.
type IntegrationManagedKeyList struct {
	Items []IntegrationManagedKey `json:"items"`
}

// ManagedKeys lists the live keys of the workspace's integrations.
func (s *Service) ManagedKeys(ctx context.Context, ws *store.Workspace) (IntegrationManagedKeyList, error) {
	conns, err := s.active(ctx, ws)
	if err != nil {
		return IntegrationManagedKeyList{}, err
	}
	lists, _ := access.ListNames(ctx, s.Q, ws.ID)
	out := IntegrationManagedKeyList{Items: []IntegrationManagedKey{}}
	for _, l := range conns {
		cur, _ := current(l.keys, s.now())
		if cur == nil || cur.RevokedAt != nil {
			continue
		}
		k := IntegrationManagedKey{IntegrationKeyView: keyView(*cur, lists, s.now()), ConnectionID: l.conn.ID.String(), Kind: l.conn.Kind, Label: l.conn.Label,
			Workflows: []IntegrationWorkflowRef{}}
		wfs, err := s.Q.IntegrationWorkflows(ctx, store.IntegrationWorkflowsParams{IntegrationID: l.conn.ID, Since: s.now().Add(-WorkflowRetention)})
		if err == nil {
			for _, w := range wfs {
				k.Workflows = append(k.Workflows, IntegrationWorkflowRef{Name: w.Name, Unnamed: w.Name == "", LastRunAt: w.LastSeenAt})
			}
		}
		if c, err := s.Q.IntegrationCallCounts(ctx, store.IntegrationCallCountsParams{IntegrationID: l.conn.ID, Since: s.now().Add(-CallRetention)}); err == nil {
			k.Calls7d = c.Calls
		}
		out.Items = append(out.Items, k)
	}
	return out, nil
}

// IntegrationMe is what "Test connection" returns to the app: the key's
// workspace and what it can do.
type IntegrationMe struct {
	Workspace     string   `json:"workspace" doc:"The workspace slug, for the other calls' paths"`
	WorkspaceName string   `json:"workspaceName"`
	Integration   string   `json:"integration" enum:"zapier,make,n8n,wordpress,"`
	Key           string   `json:"key" doc:"The key's name"`
	Triggers      []string `json:"triggers" doc:"Events of the triggers turned on"`
	Actions       []string `json:"actions" doc:"Actions turned on"`
	Permissions   []string `json:"permissions"`
}

// Me describes the calling key.
func (s *Service) Me(ctx context.Context, ws *store.Workspace, p *authz.Principal) (IntegrationMe, error) {
	out := IntegrationMe{Workspace: ws.Slug, WorkspaceName: ws.Name, Key: p.Name, Triggers: []string{}, Actions: []string{}, Permissions: p.Permissions.Strings()}
	noteTargetJSON(ctx, CallTarget{Kind: "test"})
	ik := api.IntegrationFrom(ctx)
	if ik == nil {
		return out, nil
	}
	row, err := s.Q.IntegrationKeyLimit(ctx, ik.ID)
	if err != nil {
		return out, err
	}
	out.Integration = row.Kind
	for _, it := range Items(row.Kind) {
		if !slices.Contains(row.Items, it.Key) {
			continue
		}
		if it.Kind == ItemTrigger {
			out.Triggers = append(out.Triggers, it.Event)
		} else {
			out.Actions = append(out.Actions, it.Key)
		}
	}
	return out, nil
}

// IntegrationDeadLetter is a delivery waiting in the dead-letter queue.
type IntegrationDeadLetter struct {
	ID          string     `json:"id"`
	Event       string     `json:"event"`
	Email       string     `json:"email,omitempty"`
	FirstTryAt  time.Time  `json:"firstTryAt"`
	LastAnswer  *int32     `json:"lastAnswer,omitempty"`
	StatusText  string     `json:"statusText,omitempty"`
	Error       string     `json:"error,omitempty"`
	KeptUntil   time.Time  `json:"keptUntil"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
}

// IntegrationDeadLetterList is a managed webhook's dead-letter queue.
type IntegrationDeadLetterList struct {
	Items []IntegrationDeadLetter `json:"items"`
}

// DeadLetters lists a managed webhook's dead-lettered deliveries.
func (s *Service) DeadLetters(ctx context.Context, ws *store.Workspace, p *authz.Principal, id, hookID uuid.UUID) (IntegrationDeadLetterList, error) {
	conn, err := s.Q.IntegrationGet(ctx, store.IntegrationGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return IntegrationDeadLetterList{}, api.MapError(err, "Integration")
	}
	if _, err := s.Q.IntegrationWebhookOf(ctx, store.IntegrationWebhookOfParams{WebhookID: hookID, IntegrationID: conn.ID}); err != nil {
		return IntegrationDeadLetterList{}, api.MapError(err, "Webhook")
	}
	rows, err := s.Q.IntegrationDeadLetters(ctx, hookID)
	if err != nil {
		return IntegrationDeadLetterList{}, err
	}
	canSee := p.Can(authz.SubscribersView)
	out := IntegrationDeadLetterList{Items: []IntegrationDeadLetter{}}
	for _, r := range rows {
		d := IntegrationDeadLetter{ID: r.ID.String(), Event: r.Event, FirstTryAt: r.CreatedAt, LastAnswer: r.StatusCode, Error: r.Error, LastAttempt: r.LastAttemptAt}
		last := r.CreatedAt
		if r.LastAttemptAt != nil {
			last = *r.LastAttemptAt
		}
		d.KeptUntil = last.Add(7 * 24 * time.Hour)
		if r.StatusCode != nil {
			d.StatusText = http.StatusText(int(*r.StatusCode))
		}
		var env struct {
			Data struct {
				Email string `json:"email"`
			} `json:"data"`
		}
		if canSee && json.Unmarshal(r.Payload, &env) == nil {
			d.Email = strings.ToLower(env.Data.Email)
		}
		out.Items = append(out.Items, d)
	}
	return out, nil
}
