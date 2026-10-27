package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service stores routing and serves the routing screen.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
	// Mailbox classifies an address that isn't a contact from its domain's
	// MX records (internal/mailbox); nil uses the domain name only.
	Mailbox MailboxClassifier
	// WarmupToday is a warming provider's allowance today, for vetting it
	// as a failover target; WarmCap its cap on a day, for the campaign
	// simulation. Both come from internal/warmup, which imports routing.
	WarmupToday func(ctx context.Context, providerID uuid.UUID) *state.WarmupToday
	WarmCap     func(ctx context.Context, providerID uuid.UUID, at time.Time) (WarmCap, bool)
	// LookupMX finds a domain's first MX host for the route tester; nil
	// asks DNS.
	LookupMX func(ctx context.Context, domain string) string
	// SimulateLimit is how many recipients a campaign simulation reads
	// before it scales to the audience (0: 200,000).
	SimulateLimit int64
}

// MailboxClassifier is the part of mailbox.Classifier the route tester uses.
type MailboxClassifier interface {
	ForEmail(ctx context.Context, email string) mailbox.Class
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Load reads a workspace's routing set.
func Load(ctx context.Context, q *store.Queries, wsID uuid.UUID) (Set, error) {
	st, err := loadStored(ctx, q, wsID)
	return st.set, err
}

// stored is the routing as stored: the set, each stream's version and the
// policy rows.
type stored struct {
	set      Set
	versions map[providers.Stream]string
	policies []store.StreamPolicy
}

func loadStored(ctx context.Context, q *store.Queries, wsID uuid.UUID) (stored, error) {
	rows, err := q.RoutingListRules(ctx, wsID)
	if err != nil {
		return stored{}, err
	}
	set := Set{Policies: map[providers.Stream]Policy{}}
	for _, r := range rows {
		rule := Rule{ID: r.ID, Stream: providers.Stream(r.Stream), Priority: int(r.Priority), Name: r.Name, Fallback: r.FallbackProviderID, Enabled: r.Enabled}
		_ = json.Unmarshal(r.Conditions, &rule.Conditions)
		_ = json.Unmarshal(r.Splits, &rule.Splits)
		set.Rules = append(set.Rules, rule)
	}
	pols, err := q.RoutingListPolicies(ctx, wsID)
	if err != nil {
		return stored{}, err
	}
	for _, p := range pols {
		set.Policies[providers.Stream(p.Stream)] = Policy(p.Failover)
	}
	return stored{set: set, versions: streamVersions(rows, pols), policies: pols}, nil
}

// Route decides the provider for one recipient with the saved rules. The
// sending pipeline and the system mailer use it.
func (s *Service) Route(ctx context.Context, wsID uuid.UUID, r Recipient) (Decision, bool, error) {
	set, err := Load(ctx, s.Q, wsID)
	if err != nil {
		return Decision{}, false, err
	}
	d, ok := set.Resolve(r)
	return d, ok, nil
}

// RoutingSplit is one provider's share.
type RoutingSplit struct {
	ProviderID string `json:"providerId" format:"uuid"`
	Weight     int    `json:"weight" minimum:"0" maximum:"100"`
}

// RoutingCondition is a rule evaluated before the split: IF field matches
// one of values THEN providerId only, or a weighted split of its own
// (PR-12), sticky per recipient like the stream's split.
type RoutingCondition struct {
	Field  string   `json:"field" enum:"mailbox_provider,list,sender_domain,category" doc:"category: a transactional category (Transactional stream only)"`
	Values []string `json:"values" minItems:"1" maxItems:"50"`
	// ProviderID is the one provider the condition sends to; with Splits it
	// is the first of them.
	ProviderID string `json:"providerId,omitempty" format:"uuid"`
	// Splits, when set, split the matching recipients by weight (adding up
	// to 100) instead of sending them all to one provider.
	Splits []RoutingSplit `json:"splits,omitempty" maxItems:"5"`
	Note   string         `json:"note,omitempty" maxLength:"120"`
}

// RoutingStreamInput is one stream's routing in a save.
type RoutingStreamInput struct {
	Stream string `json:"stream" enum:"broadcast,automation,system,transactional"`
	// Splits are the default route; every stream needs one except
	// Transactional, where none removes its route.
	Splits     []RoutingSplit     `json:"splits" maxItems:"5"`
	Fallback   string             `json:"fallback,omitempty" doc:"Provider id, or empty for none"`
	Failover   string             `json:"failover" enum:"automatic,pause_and_ask,warm_only"`
	Conditions []RoutingCondition `json:"conditions" maxItems:"20"`
	// Priority (Transactional only): transactional messages go first within
	// the rate of a provider that also sends a campaign.
	Priority *bool `json:"priority,omitempty"`
	// BaseVersion is the stream's version the edit started from (the
	// stream's version in the routing read). When the stream changed since,
	// the save is refused with 409 code routing_changed and nothing is saved.
	BaseVersion string `json:"baseVersion,omitempty" maxLength:"32"`
}

// RoutingSaveInput replaces the routing of the streams it lists; the
// streams it leaves out stay exactly as they are.
type RoutingSaveInput struct {
	Streams []RoutingStreamInput `json:"streams" minItems:"1" maxItems:"4"`
	// AcknowledgeBulkTerms keeps a Broadcast route that a provider's terms
	// don't allow. The decision is recorded and Send Readiness blocks
	// broadcasts while it stands.
	AcknowledgeBulkTerms bool `json:"acknowledgeBulkTerms,omitempty"`
}

// RoutingConditionView is a condition with readable values.
type RoutingConditionView struct {
	RoutingCondition
	ValueLabels []string `json:"valueLabels"`
}

// RoutingVolume is how much mail a stream carries.
type RoutingVolume struct {
	Count int64  `json:"count"`
	Per   string `json:"per" enum:"day,week"`
	// Smallest and Largest are the audience range of recent broadcasts.
	Smallest int64 `json:"smallest,omitempty"`
	Largest  int64 `json:"largest,omitempty"`
}

// RoutingStreamView is one stream's saved routing.
type RoutingStreamView struct {
	Stream string `json:"stream" enum:"broadcast,automation,system,transactional"`
	// Version changes whenever the stream's routing changes; send it back as
	// baseVersion when saving the stream.
	Version string `json:"version"`
	// Splits are in the order they were saved: the order only changes how
	// the split reads, never who goes where.
	Splits     []RoutingSplit         `json:"splits"`
	Fallback   string                 `json:"fallback"`
	Failover   string                 `json:"failover" enum:"automatic,pause_and_ask,warm_only"`
	Conditions []RoutingConditionView `json:"conditions"`
	// BulkTermsAcknowledgedAt is set while someone chose to keep a Broadcast
	// route the provider's terms don't allow.
	BulkTermsAcknowledgedAt *time.Time     `json:"bulkTermsAcknowledgedAt,omitempty"`
	Volume                  *RoutingVolume `json:"volume,omitempty"`
	// Priority (Transactional only): first in line on shared providers.
	Priority *bool `json:"priority,omitempty"`
	// SenderDomains (Transactional only) are the domains marked for
	// transactional mail ("notify.acme.io").
	SenderDomains []string `json:"senderDomains,omitempty"`
	// FailoverVet is what's checked before the stream fails over: its
	// fallback, or the split's other provider (PR-14).
	FailoverVet *RoutingTargetVet `json:"failoverVet,omitempty"`
}

// RoutingVetCheck is one check of a failover target (state.Vet).
type RoutingVetCheck struct {
	Key    string `json:"key" enum:"available,bulk_terms,sandbox,domain,quota,warm"`
	Result string `json:"result" enum:"pass,warn,fail,unknown"`
	// Left is what the quota or today's warm-up cap has left.
	Left *int64 `json:"left,omitempty"`
	// Period is what a quota's Left counts: today ("day") or this month
	// ("month"); empty when the provider sets no quota.
	Period string `json:"period,omitempty" enum:"day,month"`
}

// RoutingTargetVet is a failover target and its checks.
type RoutingTargetVet struct {
	ProviderID string            `json:"providerId"`
	Provider   string            `json:"provider"`
	Checks     []RoutingVetCheck `json:"checks"`
	// WarmupDay of WarmupDays while the target warms up.
	WarmupDay  int `json:"warmupDay,omitempty"`
	WarmupDays int `json:"warmupDays,omitempty"`
}

// RoutingTerms is one row of "What each connected provider's terms allow".
type RoutingTerms struct {
	Variant string   `json:"variant"`
	Bulk    string   `json:"bulk" enum:"allowed,broadcast_streams_only,forbidden,your_policy"`
	Note    string   `json:"note"`
	Streams []string `json:"streams"`
	// Active marks the variant this connection uses; Available is false for
	// a variant that isn't set up yet (Postmark Broadcast stream).
	Active    bool `json:"active"`
	Available bool `json:"available"`
}

// RoutingProvider is a provider that can be picked in the routing selects.
type RoutingProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Short       string `json:"short"`
	Label       string `json:"label"`
	Adapter     string `json:"adapter"`
	Color       string `json:"color" enum:"ses,postal,postmark,other"`
	Status      string `json:"status"`
	Bulk        string `json:"bulk" enum:"allowed,broadcast_streams_only,forbidden,your_policy"`
	BulkAllowed bool   `json:"bulkAllowed"`
	// BroadcastStreamCreatable: ReputeCast can create the stream that makes
	// bulk mail allowed ("Postmark · Broadcast stream (create)").
	BroadcastStreamCreatable bool `json:"broadcastStreamCreatable"`
	// RemovesWebhook: removing the provider also removes the webhook
	// ReputeCast registered there.
	RemovesWebhook bool               `json:"removesWebhook"`
	WarmingUp      bool               `json:"warmingUp"`
	WarmupDay      int                `json:"warmupDay,omitempty"`
	WarmupDays     int                `json:"warmupDays,omitempty"`
	ShareSteps     []RoutingShareStep `json:"shareSteps"`
	Terms          []RoutingTerms     `json:"terms"`
}

// RoutingShareStep is a planned increase of a warming provider's Broadcast share.
type RoutingShareStep struct {
	Day   int `json:"day"`
	Share int `json:"share"`
}

// RoutingNamed is a list or sender domain option.
type RoutingNamed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RoutingShared is the "Shared across providers" card.
type RoutingShared struct {
	Suppressions int64 `json:"suppressions"`
	// Pushed counts the suppressions providers get: hard bounces and
	// complaints that apply to all mail (review PR-1).
	Pushed   int64      `json:"pushed"`
	SyncedAt *time.Time `json:"syncedAt,omitempty"`
}

// RoutingConfig is the routing screen.
type RoutingConfig struct {
	Streams       []RoutingStreamView `json:"streams"`
	Providers     []RoutingProvider   `json:"providers"`
	Lists         []RoutingNamed      `json:"lists"`
	SenderDomains []string            `json:"senderDomains"`
	// Categories are the transactional categories (Transactional conditions);
	// Templates the transactional templates the route tester offers.
	Categories []RoutingNamed `json:"categories"`
	Templates  []RoutingNamed `json:"templates"`
	SavedAt    *time.Time     `json:"savedAt,omitempty"`
	SavedBy    string         `json:"savedBy"`
	Issues     []RoutingIssue `json:"issues"`
	Shared     RoutingShared  `json:"shared"`
}

// Get returns the routing screen for a workspace.
func (s *Service) Get(ctx context.Context, ws *store.Workspace) (RoutingConfig, error) {
	st, err := loadStored(ctx, s.Q, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	set := st.set
	facts, provs, err := s.facts(ctx, ws)
	if err != nil {
		return RoutingConfig{}, err
	}
	lists, err := s.Q.RoutingLists(ctx, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	listNames := map[string]string{}
	l := i18n.LangFrom(ctx)
	out := RoutingConfig{Providers: provs, Lists: []RoutingNamed{}, SenderDomains: []string{}, Categories: []RoutingNamed{}, Templates: []RoutingNamed{}, Issues: Validate(l, set, facts)}
	for _, l := range lists {
		out.Lists = append(out.Lists, RoutingNamed{ID: l.ID.String(), Name: l.Name})
		listNames[l.ID.String()] = l.Name
	}
	domains, err := s.Q.RoutingSenderDomains(ctx, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	for _, d := range domains {
		out.SenderDomains = append(out.SenderDomains, d.Domain)
	}
	// Domains a condition names can be tested too, set up here or not.
	for _, r := range set.Rules {
		for _, d := range r.Conditions.SenderDomains {
			if !slices.Contains(out.SenderDomains, d) {
				out.SenderDomains = append(out.SenderDomains, d)
			}
		}
	}
	categoryNames := map[string]string{}
	if cats, err := s.Q.TxnRouteCategories(ctx, ws.ID); err == nil {
		for _, c := range cats {
			out.Categories = append(out.Categories, RoutingNamed{ID: c.ID.String(), Name: c.Name})
			categoryNames[c.ID.String()] = c.Name
		}
	} else {
		return RoutingConfig{}, err
	}
	if tpls, err := s.Q.TxnRouteTemplates(ctx, ws.ID); err == nil {
		for _, t := range tpls {
			out.Templates = append(out.Templates, RoutingNamed{ID: t.Key, Name: t.Name})
		}
	} else {
		return RoutingConfig{}, err
	}
	txDomains, err := s.Q.TxnRouteDomains(ctx, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	acks := map[string]*time.Time{}
	priority := true
	for _, p := range st.policies {
		acks[p.Stream] = p.BulkTermsAckAt
		if p.Stream == string(StreamTransactional) {
			priority = p.Priority
		}
	}
	volumes := s.volumes(ctx, ws)
	// The default sender's verification per provider (failover checks).
	var sender map[string]string
	if raw, err := s.Q.RoutingDefaultSenderVerification(ctx, ws.ID); err == nil {
		sender = map[string]string{}
		_ = json.Unmarshal(raw, &sender)
	}
	for _, stream := range Streams {
		v := RoutingStreamView{Stream: string(stream), Version: st.versions[stream], Splits: []RoutingSplit{}, Failover: string(set.Policy(stream)), Conditions: []RoutingConditionView{},
			BulkTermsAcknowledgedAt: acks[string(stream)], Volume: volumes[stream]}
		if stream == StreamTransactional {
			pr := priority
			v.Priority, v.SenderDomains = &pr, txDomains
		}
		for _, r := range set.StreamRules(stream) {
			if r.Conditions.Empty() {
				for _, sp := range r.Splits {
					v.Splits = append(v.Splits, RoutingSplit{ProviderID: sp.ProviderID.String(), Weight: sp.Weight})
				}
				if r.Fallback != nil {
					v.Fallback = r.Fallback.String()
				}
				continue
			}
			if len(r.Splits) == 0 {
				continue
			}
			c := conditionView(l, r, listNames, categoryNames)
			v.Conditions = append(v.Conditions, c)
		}
		v.FailoverVet = s.failoverVet(ctx, ws, stream, v, provs, sender)
		out.Streams = append(out.Streams, v)
	}
	if saved, err := s.Q.RoutingSaved(ctx, ws.ID); err == nil {
		t := saved.UpdatedAt
		out.SavedAt, out.SavedBy = &t, saved.UpdatedByName
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return RoutingConfig{}, err
	}
	out.Shared.Suppressions, _ = s.Q.SuppCount(ctx, ws.ID)
	if summary, err := s.Q.SuppPushSummary(ctx, ws.ID); err == nil {
		for _, r := range summary {
			if r.Pushed {
				out.Shared.Pushed += r.N
			}
		}
	}
	out.Shared.SyncedAt = s.syncedAt(ctx, provs)
	return out, nil
}

// quotaPeriod is what a provider's quota kind counts.
func quotaPeriod(kind string) string {
	switch kind {
	case "rolling_24h", "daily", "warmup_daily":
		return "day"
	case "monthly_plan", "monthly":
		return "month"
	}
	return ""
}

func conditionView(l i18n.Lang, r Rule, listNames, categoryNames map[string]string) RoutingConditionView {
	c := RoutingConditionView{RoutingCondition: RoutingCondition{ProviderID: r.Splits[0].ProviderID.String(), Note: r.Name}}
	if len(r.Splits) > 1 {
		for _, sp := range r.Splits {
			c.Splits = append(c.Splits, RoutingSplit{ProviderID: sp.ProviderID.String(), Weight: sp.Weight})
		}
	}
	switch {
	case len(r.Conditions.Categories) > 0:
		c.Field, c.Values = "category", r.Conditions.Categories
		for _, v := range c.Values {
			name := categoryNames[v]
			if name == "" {
				name = text(l, "label.deletedCategory", nil)
			}
			c.ValueLabels = append(c.ValueLabels, name)
		}
	case len(r.Conditions.MailboxProviders) > 0:
		c.Field, c.Values = "mailbox_provider", r.Conditions.MailboxProviders
		for _, v := range c.Values {
			c.ValueLabels = append(c.ValueLabels, MailboxNameIn(l, v))
		}
	case len(r.Conditions.Lists) > 0:
		c.Field, c.Values = "list", r.Conditions.Lists
		for _, v := range c.Values {
			name := listNames[v]
			if name == "" {
				name = text(l, "label.deletedList", nil)
			}
			c.ValueLabels = append(c.ValueLabels, name)
		}
	default:
		c.Field, c.Values = "sender_domain", r.Conditions.SenderDomains
		c.ValueLabels = append(c.ValueLabels, c.Values...)
	}
	return c
}

// facts loads what validation and the selects need about each provider.
func (s *Service) facts(ctx context.Context, ws *store.Workspace) (map[uuid.UUID]Facts, []RoutingProvider, error) {
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	plans, err := s.Q.WarmupListPlans(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	loc := state.Location(ws.Timezone)
	facts := map[uuid.UUID]Facts{}
	out := []RoutingProvider{}
	for _, p := range provs {
		caps := state.Caps(p)
		f := Facts{ID: p.ID, Name: p.Name, Short: state.Short(p.Name, p.Adapter), Adapter: p.Adapter, Status: p.Status, Bulk: caps.Bulk, BulkAllowed: caps.BulkAllowed(),
			Available: state.Available(p)}
		rp := RoutingProvider{ID: p.ID.String(), Name: p.Name, Short: f.Short, Label: p.Label, Adapter: p.Adapter, Color: state.Color(p.Adapter), Status: p.Status,
			Bulk: string(caps.Bulk), BulkAllowed: f.BulkAllowed, ShareSteps: []RoutingShareStep{}, Terms: []RoutingTerms{}}
		if d, ok := providers.Lookup(p.Adapter); ok {
			rp.RemovesWebhook = d.Capabilities.RemovesWebhook
			// The adapter's terms in the reader's language (providers, text.go).
			for _, t := range d.In(i18n.LangFrom(ctx)).Terms {
				rt := RoutingTerms{Variant: t.Variant, Bulk: string(t.Bulk), Note: t.Note, Streams: []string{}, Available: true}
				for _, st := range t.Streams {
					rt.Streams = append(rt.Streams, string(st))
				}
				if caps.Bulk == providers.BulkBroadcastStreams {
					// Postmark: the transactional variant is the one in use;
					// the broadcast one exists only once created.
					rt.Active = t.Bulk != providers.BulkAllowed
					if t.Bulk == providers.BulkAllowed {
						rt.Available = caps.BroadcastStream
						rt.Active = caps.BroadcastStream
					}
				} else {
					rt.Active = true
				}
				rp.Terms = append(rp.Terms, rt)
			}
			if caps.Bulk == providers.BulkBroadcastStreams && !caps.BroadcastStream {
				f.BroadcastStreamCreatable = true
			}
		}
		rp.BroadcastStreamCreatable = f.BroadcastStreamCreatable
		for _, w := range plans {
			if w.ProviderID != p.ID || w.Status == "done" {
				continue
			}
			f.WarmingUp, f.WarmupDays = true, int(w.Days)
			f.WarmupDay = min(max(state.WarmupDay(w.StartDate, s.now(), loc), 1), int(w.Days))
			var sched []struct {
				Day   int `json:"day"`
				Share int `json:"share"`
			}
			_ = json.Unmarshal(w.Schedule, &sched)
			last := 0
			for _, d := range sched {
				if d.Share > 0 && d.Share != last {
					if last != 0 && d.Day > f.WarmupDay {
						rp.ShareSteps = append(rp.ShareSteps, RoutingShareStep{Day: d.Day, Share: d.Share})
					}
					last = d.Share
				}
			}
			break
		}
		rp.WarmingUp, rp.WarmupDay, rp.WarmupDays = f.WarmingUp, f.WarmupDay, f.WarmupDays
		facts[p.ID] = f
		out = append(out, rp)
	}
	return facts, out, nil
}

// volumes estimates how much each stream carries: Broadcasts per week over
// the last four weeks, Automations and System per day over the last seven
// whole days (in the workspace time zone, so the estimate holds all day).
func (s *Service) volumes(ctx context.Context, ws *store.Workspace) map[providers.Stream]*RoutingVolume {
	out := map[providers.Stream]*RoutingVolume{}
	wsID := ws.ID
	since := s.now().AddDate(0, 0, -28)
	if b, err := s.Q.ProvBroadcastWeekly(ctx, store.ProvBroadcastWeeklyParams{WorkspaceID: wsID, Since: &since}); err == nil && b > 0 {
		v := &RoutingVolume{Count: about(b / 4), Per: "week"}
		quarter := s.now().AddDate(0, 0, -90)
		if sz, err := s.Q.RoutingBroadcastSizes(ctx, store.RoutingBroadcastSizesParams{WorkspaceID: wsID, Since: &quarter}); err == nil {
			v.Smallest, v.Largest = sz.Smallest, sz.Largest
		}
		out[providers.StreamBroadcast] = v
	}
	now := s.now().In(state.Location(ws.Timezone))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	week := today.AddDate(0, 0, -7)
	rows, err := s.Q.ProvStreamVolume(ctx, store.ProvStreamVolumeParams{WorkspaceID: wsID, Since: &week, Until: &today})
	if err == nil {
		for _, r := range rows {
			st := providers.Stream(r.Stream)
			if st == providers.StreamBroadcast || r.N == 0 {
				continue
			}
			out[st] = &RoutingVolume{Count: about(r.N / 7), Per: "day"}
		}
	}
	return out
}

// about keeps the first two significant digits: 47,980 is "about 47,000",
// 1,912 "about 1,900".
func about(n int64) int64 {
	if n < 100 {
		return n
	}
	unit := int64(math.Pow10(len(strconv.FormatInt(n, 10)) - 2))
	return n / unit * unit
}

func (s *Service) syncedAt(ctx context.Context, provs []RoutingProvider) *time.Time {
	ids := make([]uuid.UUID, 0, len(provs))
	for _, p := range provs {
		ids = append(ids, uuid.MustParse(p.ID))
	}
	rows, err := s.Q.SuppSyncList(ctx, ids)
	if err != nil {
		return nil
	}
	var oldest *time.Time
	for _, r := range rows {
		if r.Mode == "off" || r.LastPushAt == nil {
			continue
		}
		if oldest == nil || r.LastPushAt.Before(*oldest) {
			t := *r.LastPushAt
			oldest = &t
		}
	}
	return oldest
}

// toSet converts a save input to rules, with field errors for bad ids (in
// the request's language).
func toSet(ctx context.Context, in RoutingSaveInput, base Set) (Set, error) {
	out := Set{Policies: map[providers.Stream]Policy{}}
	for k, v := range base.Policies {
		out.Policies[k] = v
	}
	replaced := map[providers.Stream]bool{}
	for i, st := range in.Streams {
		stream := providers.Stream(st.Stream)
		if replaced[stream] {
			return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].stream", i), say(ctx, "problem.streamTwice", nil))
		}
		replaced[stream] = true
		out.Policies[stream] = Policy(st.Failover)
		if len(st.Splits) == 0 {
			if stream == StreamTransactional {
				continue // no Transactional route
			}
			return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].splits", i), say(ctx, "problem.pickPrimary", i18n.Args{"stream": streamArg(stream)}))
		}
		var fallback *uuid.UUID
		if st.Fallback != "" {
			id, err := uuid.Parse(st.Fallback)
			if err != nil {
				return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].fallback", i), say(ctx, "problem.pickFallback", nil))
			}
			fallback = &id
		}
		def := Rule{ID: uuid.Must(uuid.NewV7()), Stream: stream, Priority: 100, Fallback: fallback, Enabled: true}
		for j, sp := range st.Splits {
			id, err := uuid.Parse(sp.ProviderID)
			if err != nil {
				return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].splits[%d].providerId", i, j), say(ctx, "problem.pickProvider", nil))
			}
			if sp.Weight == 0 && len(st.Splits) > 1 {
				continue
			}
			def.Splits = append(def.Splits, Split{ProviderID: id, Weight: sp.Weight})
		}
		out.Rules = append(out.Rules, def)
		for j, c := range st.Conditions {
			splits, err := conditionSplits(ctx, c, i, j)
			if err != nil {
				return Set{}, err
			}
			values := []string{}
			for _, v := range c.Values {
				if v = strings.TrimSpace(v); v != "" && !slices.Contains(values, v) {
					values = append(values, v)
				}
			}
			if len(values) == 0 {
				return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].values", i, j), say(ctx, "problem.pickValue", nil))
			}
			var cond Conditions
			switch c.Field {
			case "mailbox_provider":
				for _, v := range values {
					if !slices.Contains(MailboxProviders, v) {
						return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].values", i, j), say(ctx, "problem.mailboxValues", nil))
					}
				}
				cond.MailboxProviders = values
			case "list":
				for _, v := range values {
					if _, err := uuid.Parse(v); err != nil {
						return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].values", i, j), say(ctx, "problem.pickLists", nil))
					}
				}
				cond.Lists = values
			case "category":
				if stream != StreamTransactional {
					return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].field", i, j), say(ctx, "problem.categoryStream", nil))
				}
				for _, v := range values {
					if _, err := uuid.Parse(v); err != nil {
						return Set{}, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].values", i, j), say(ctx, "problem.pickCategories", nil))
					}
				}
				cond.Categories = values
			default:
				for k := range values {
					values[k] = strings.ToLower(values[k])
				}
				cond.SenderDomains = values
			}
			out.Rules = append(out.Rules, Rule{ID: uuid.Must(uuid.NewV7()), Stream: stream, Priority: 10 * (j + 1), Name: strings.TrimSpace(c.Note),
				Conditions: cond, Splits: splits, Fallback: fallback, Enabled: true})
		}
	}
	for _, r := range base.Rules {
		if !replaced[r.Stream] {
			out.Rules = append(out.Rules, r)
		}
	}
	return out, nil
}

// conditionSplits is where a condition sends its recipients: one provider,
// or a split by weight (a 0% share is left out; Validate checks the rest).
func conditionSplits(ctx context.Context, c RoutingCondition, i, j int) ([]Split, error) {
	if len(c.Splits) == 0 {
		id, err := uuid.Parse(c.ProviderID)
		if err != nil {
			return nil, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].providerId", i, j), say(ctx, "problem.pickConditionProvider", nil))
		}
		return []Split{{ProviderID: id, Weight: 100}}, nil
	}
	out := []Split{}
	for k, sp := range c.Splits {
		id, err := uuid.Parse(sp.ProviderID)
		if err != nil {
			return nil, api.Invalid(fmt.Sprintf("body.streams[%d].conditions[%d].splits[%d].providerId", i, j, k), say(ctx, "problem.pickConditionProvider", nil))
		}
		if sp.Weight == 0 && len(c.Splits) > 1 {
			continue
		}
		out = append(out, Split{ProviderID: id, Weight: sp.Weight})
	}
	if len(out) == 1 {
		out[0].Weight = 100
	}
	return out, nil
}

// CodeRoutingChanged is the problem code of a save whose base version is
// stale: someone (or the warm-up planner) changed a stream since the edit
// began. Its meta "streams" lists the streams that changed.
const CodeRoutingChanged = "routing_changed"

// Save replaces the routing of the streams in the input and leaves the
// others exactly as stored. A stream sent with a baseVersion that is no
// longer its version is refused with 409 (CodeRoutingChanged): nothing is
// saved, so a tab left open never overwrites newer routing.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, in RoutingSaveInput) (RoutingConfig, error) {
	before, err := s.Get(ctx, ws)
	if err != nil {
		return RoutingConfig{}, err
	}
	facts, _, err := s.facts(ctx, ws)
	if err != nil {
		return RoutingConfig{}, err
	}
	saving := map[providers.Stream]bool{}
	for _, st := range in.Streams {
		saving[providers.Stream(st.Stream)] = true
	}
	// Lists and categories named in conditions must belong to the workspace.
	lists, err := s.Q.RoutingLists(ctx, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	known := map[string]bool{}
	for _, l := range lists {
		known[l.ID.String()] = true
	}
	cats, err := s.Q.TxnRouteCategories(ctx, ws.ID)
	if err != nil {
		return RoutingConfig{}, err
	}
	knownCat := map[string]bool{}
	for _, c := range cats {
		knownCat[c.ID.String()] = true
	}
	p := api.PrincipalFrom(ctx)
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RoutingLock(ctx, ws.ID); err != nil {
			return err
		}
		base, err := loadStored(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		var stale []string
		for _, st := range in.Streams {
			if st.BaseVersion != "" && st.BaseVersion != base.versions[providers.Stream(st.Stream)] {
				stale = append(stale, st.Stream)
			}
		}
		if len(stale) > 0 {
			return api.Err(http.StatusConflict, CodeRoutingChanged, say(ctx, "problem.changed", nil)).WithMeta("streams", stale)
		}
		set, err := toSet(ctx, in, base.set)
		if err != nil {
			return err
		}
		for _, r := range set.Rules {
			if !saving[r.Stream] {
				continue
			}
			for _, l := range r.Conditions.Lists {
				if !known[l] {
					return api.Invalid("body.streams", say(ctx, "problem.unknownList", nil))
				}
			}
			for _, c := range r.Conditions.Categories {
				if !knownCat[c] {
					return api.Invalid("body.streams", say(ctx, "problem.unknownCategory", nil))
				}
			}
		}
		// Only the streams being saved can stop the save: a problem in a
		// stream left as it is stays on screen as a warning of its own.
		issues := []RoutingIssue{}
		for _, i := range Validate(i18n.LangFrom(ctx), set, facts) {
			if saving[providers.Stream(i.Stream)] {
				issues = append(issues, i)
			}
		}
		for _, i := range issues {
			if i.Blocking {
				return api.Err(http.StatusUnprocessableEntity, api.CodeValidation, i.Message).WithMeta("issues", issues)
			}
		}
		bulk := []RoutingIssue{}
		for _, i := range issues {
			if i.Code == IssueBulkTerms {
				bulk = append(bulk, i)
			}
		}
		if len(bulk) > 0 && !in.AcknowledgeBulkTerms {
			return api.Err(http.StatusConflict, "bulk_terms", say(ctx, "problem.bulkTerms", i18n.Args{"issue": bulk[0].Message})).WithMeta("issues", bulk)
		}
		names := make([]string, 0, len(saving))
		for _, st := range Streams {
			if saving[st] {
				names = append(names, string(st))
			}
		}
		if err := q.RoutingDeleteStreamRules(ctx, store.RoutingDeleteStreamRulesParams{WorkspaceID: ws.ID, Streams: names}); err != nil {
			return err
		}
		for _, r := range set.Rules {
			if !saving[r.Stream] {
				continue
			}
			conds, _ := json.Marshal(r.Conditions)
			splits, _ := json.Marshal(r.Splits)
			if err := q.RoutingInsertRule(ctx, store.RoutingInsertRuleParams{
				ID: r.ID, WorkspaceID: ws.ID, Stream: string(r.Stream), Priority: int32(r.Priority), Name: r.Name,
				Conditions: conds, Splits: splits, FallbackProviderID: r.Fallback, UpdatedBy: by, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
		for _, stream := range Streams {
			if !saving[stream] {
				continue
			}
			params := store.RoutingUpsertPolicyParams{WorkspaceID: ws.ID, Stream: string(stream), Failover: string(set.Policy(stream))}
			if stream == providers.StreamBroadcast && len(bulk) > 0 {
				params.BulkTermsAckAt, params.BulkTermsAckBy = &now, by
			}
			if err := q.RoutingUpsertPolicy(ctx, params); err != nil {
				return err
			}
		}
		for _, st := range in.Streams {
			if providers.Stream(st.Stream) == StreamTransactional && st.Priority != nil {
				if err := q.TxnRouteSetPriority(ctx, store.TxnRouteSetPriorityParams{WorkspaceID: ws.ID, Priority: *st.Priority}); err != nil {
					return err
				}
			}
		}
		wsID := ws.ID
		meta := map[string]any{"streams": names}
		if len(bulk) > 0 {
			meta["acknowledged_bulk_terms"] = bulk
		}
		prev := []RoutingStreamView{}
		for _, v := range before.Streams {
			if saving[providers.Stream(v.Stream)] {
				prev = append(prev, v)
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &wsID, Action: "routing.update", TargetType: "routing", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: prev, After: in.Streams, Metadata: meta,
		})
	})
	if err != nil {
		return RoutingConfig{}, err
	}
	return s.Get(ctx, ws)
}

// failoverVet checks the stream's failover target the way a failover
// would (state.Vet): its fallback, or for a split the other provider.
func (s *Service) failoverVet(ctx context.Context, ws *store.Workspace, stream providers.Stream, v RoutingStreamView, provs []RoutingProvider,
	sender map[string]string) *RoutingTargetVet {
	target := v.Fallback
	if target == "" && stream == providers.StreamBroadcast && len(v.Splits) > 1 {
		target = v.Splits[1].ProviderID
	}
	id, err := uuid.Parse(target)
	if err != nil {
		return nil
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return nil
	}
	load := state.Load{Broadcast: stream == providers.StreamBroadcast, Sender: sender}
	if s.WarmupToday != nil {
		load.Warmup = s.WarmupToday(ctx, p.ID)
	}
	vet := state.Vet(p, load)
	out := &RoutingTargetVet{ProviderID: p.ID.String(), Provider: p.Name, Checks: []RoutingVetCheck{}}
	for _, c := range vet.Checks {
		// Availability and bulk terms show only when they fail: the
		// stream card says them elsewhere.
		if (c.Key == state.VetAvailable || c.Key == state.VetBulkTerms) && c.Result == state.VetPass {
			continue
		}
		rc := RoutingVetCheck{Key: c.Key, Result: c.Result, Left: c.Left}
		if c.Key == state.VetQuota {
			rc.Period = quotaPeriod(c.Detail)
		}
		out.Checks = append(out.Checks, rc)
	}
	for _, rp := range provs {
		if rp.ID == out.ProviderID && rp.WarmingUp {
			out.WarmupDay, out.WarmupDays = rp.WarmupDay, rp.WarmupDays
		}
	}
	return out
}
