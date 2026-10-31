package readiness

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/retryaudience"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// SegmentResolver lists a segment's contacts (the audience domain's
// segments.Service). Without one, segments in an audience aren't counted.
type SegmentResolver interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}

// AudienceDef is campaigns.audience as stored by the composer, plus the
// tiers Send Readiness can exclude.
type AudienceDef struct {
	Lists           []string `json:"lists,omitempty"`
	Segments        []string `json:"segments,omitempty"`
	ExcludeLists    []string `json:"exclude_lists,omitempty"`
	ExcludeSegments []string `json:"exclude_segments,omitempty"`
	ExcludeTiers    []string `json:"exclude_tiers,omitempty"`
	// IncludeSunset is the campaign overriding the sunset policy: contacts
	// whose Broadcasts the policy stopped receive this one anyway.
	IncludeSunset bool `json:"include_sunset,omitempty"`
	// IncludeUnengaged includes (true) or leaves out (false) the unengaged
	// contacts (the sunset tier) for this campaign. Unset, a send leaves
	// them out unless it overrides the sunset policy (SR-11, C2: "Include
	// them").
	IncludeUnengaged *bool `json:"include_unengaged,omitempty"`
	// Paid is who on the lists gets a paid-only campaign (Earn): paid
	// members only, at a tier and above, or free subscribers only.
	Paid paidaccess.Audience `json:"-"`
	// RetryOf is the issue a retry campaign sends again, only to its
	// addresses blocked for the sender's reputation (review AN-7); a fix
	// that edits the audience keeps it.
	RetryOf string `json:"retry_of,omitempty"`
}

// ParseAudience reads a stored audience definition.
func ParseAudience(raw json.RawMessage) AudienceDef {
	var a AudienceDef
	_ = json.Unmarshal(raw, &a)
	if a.IncludeUnengaged == nil {
		var camel struct {
			IncludeUnengaged *bool `json:"includeUnengaged"`
		}
		_ = json.Unmarshal(raw, &camel)
		a.IncludeUnengaged = camel.IncludeUnengaged
	}
	a.Paid = paidaccess.Parse(raw)
	return a
}

// retryOf is the issue a retry campaign's audience retries.
func (a AudienceDef) retryOf() (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(a.RetryOf))
	return id, err == nil && id != uuid.Nil
}

// Empty reports whether the definition selects nobody.
func (a AudienceDef) Empty() bool { return len(a.Lists) == 0 && len(a.Segments) == 0 }

// ExcludesSunset reports whether the unengaged contacts (the sunset tier)
// are left out of the send by default (LeavesOutUnengaged).
func (a AudienceDef) ExcludesSunset() bool { return a.LeavesOutUnengaged(true) }

// LeavesOutUnengaged reports whether the unengaged contacts (the sunset
// tier) are left out of the send: when the tier is excluded, when the
// campaign leaves them out, and otherwise as byDefault says unless the
// campaign overrides the sunset policy (SR-11). The sending engine leaves
// out the same contacts.
func (a AudienceDef) LeavesOutUnengaged(byDefault bool) bool {
	switch {
	case slices.Contains(a.ExcludeTiers, "sunset"):
		return true
	case a.IncludeUnengaged != nil:
		return !*a.IncludeUnengaged
	}
	return byDefault && !a.IncludeSunset
}

// Recipient is one final recipient, as routing needs it.
type Recipient struct {
	Email           string
	MailboxProvider string
	Lists           []string
	Tier            string
	// Language is the contact's language code as stored.
	Language string
}

// mailboxKey folds aliases that reach the same mailbox: plus-tags anywhere,
// dots in Gmail addresses.
func mailboxKey(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndexByte(e, '@')
	if at < 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
}

// audienceNames resolves list and segment names.
type audienceNames struct {
	lists    map[string]string
	segments map[string]string
	// public are the lists readers can sign up for, but the ones a store
	// fills (X20): an audience with all of them is "All subscribers".
	public []string
}

func (s *Service) names(ctx context.Context, q *store.Queries, wsID uuid.UUID) (audienceNames, error) {
	n := audienceNames{lists: map[string]string{}, segments: map[string]string{}}
	lists, err := q.ReadinessListNames(ctx, wsID)
	if err != nil {
		return n, err
	}
	for _, l := range lists {
		n.lists[l.ID.String()] = l.Name
		if l.Public && l.Origin != "store" {
			n.public = append(n.public, l.ID.String())
		}
	}
	segs, err := q.ReadinessSegmentNames(ctx, wsID)
	if err != nil {
		return n, err
	}
	for _, sg := range segs {
		n.segments[sg.ID.String()] = sg.Name
	}
	return n, nil
}

// labels names the audience in f: the base label ("Weekly Digest list",
// "All subscribers"), the group label, what kind of audience it is and the
// names of the lists and segments.
func (n audienceNames) labels(a AudienceDef, f *AudienceFacts) {
	names := []string{}
	for _, id := range a.Lists {
		if v, ok := n.lists[id]; ok {
			names = append(names, v)
		}
	}
	for _, id := range a.Segments {
		if v, ok := n.segments[id]; ok {
			names = append(names, v)
		}
	}
	f.Lists = names
	switch {
	case len(a.Segments) == 0 && len(a.ExcludeLists) == 0 && len(n.public) > 1 && !slices.ContainsFunc(n.public, func(id string) bool { return !slices.Contains(a.Lists, id) }):
		f.BaseKind = BaseAll
	case len(names) == 1 && len(a.Lists) == 1:
		f.BaseKind, f.BaseName = BaseList, names[0]
	case len(names) == 1:
		f.BaseKind, f.BaseName = BaseSegment, names[0]
	case len(names) == 0:
		f.BaseKind = BaseNone
	default:
		joined := strings.Join(names, " + ")
		f.BaseKind, f.BaseLabel, f.GroupLabel = BaseMany, joined, joined
		return
	}
	b := &builder{f: Facts{Audience: *f}}
	f.BaseLabel, f.GroupLabel = b.baseLabel().en(), b.groupLabel().en()
}

// computeAudience counts the final recipients of a definition: suppressed,
// role, duplicate and (when excluded) sunset-tier contacts are removed. With
// withRecipients it returns them too, each with their lists, for routing;
// counts alone (Audience) don't read the lists.
func (s *Service) computeAudience(ctx context.Context, q *store.Queries, wsID uuid.UUID, def AudienceDef, unengagedDays int, now time.Time, withRecipients bool) (AudienceFacts, []Recipient, error) {
	leaveOut := def.LeavesOutUnengaged(sendDefaults(ctx, q, wsID).ExcludeUnengaged)
	out := AudienceFacts{Mailboxes: map[string]int64{}, Languages: map[string]int64{}, UnengagedDays: unengagedDays, UnengagedExcluded: leaveOut, Lists: []string{},
		SunsetOverride: def.IncludeSunset && !leaveOut}
	names, err := s.names(ctx, q, wsID)
	if err != nil {
		return out, nil, err
	}
	names.labels(def, &out)
	if def.Empty() {
		out.Empty = true
		return out, nil, nil
	}
	ids := func(in []string) []uuid.UUID {
		res := []uuid.UUID{}
		for _, v := range in {
			if u, err := uuid.Parse(v); err == nil {
				res = append(res, u)
			}
		}
		return res
	}
	contacts, exclude := []uuid.UUID{}, []uuid.UUID{}
	if len(def.Segments)+len(def.ExcludeSegments) > 0 {
		if s.Segments == nil {
			out.SegmentsSkipped = true
		} else {
			for _, sg := range ids(def.Segments) {
				c, err := s.Segments.ContactIDs(ctx, wsID, sg)
				if err != nil {
					return out, nil, err
				}
				contacts = append(contacts, c...)
			}
			for _, sg := range ids(def.ExcludeSegments) {
				c, err := s.Segments.ContactIDs(ctx, wsID, sg)
				if err != nil {
					return out, nil, err
				}
				exclude = append(exclude, c...)
			}
		}
	}
	rows, err := q.ReadinessRecipients(ctx, store.ReadinessRecipientsParams{
		WithLists: withRecipients, BouncedSince: now.AddDate(0, 0, -30), WorkspaceID: wsID, Lists: ids(def.Lists), Contacts: contacts,
		ExcludeLists: ids(def.ExcludeLists), ExcludeContacts: exclude,
	})
	if err != nil {
		return out, nil, err
	}
	// A paid-only campaign goes to the paid members on its lists, or to
	// the free subscribers.
	keep, err := paidaccess.Load(ctx, q, wsID, def.Paid)
	if err != nil {
		return out, nil, err
	}
	// A retry goes only to its issue's addresses that were blocked for the
	// sender's reputation and haven't had a retry yet (review AN-7).
	retry := retryaudience.Keep{}
	if issue, ok := def.retryOf(); ok {
		if retry, err = retryaudience.LoadIssue(ctx, q, wsID, issue, uuid.Nil); err != nil {
			return out, nil, err
		}
		name, err := q.RetryIssueName(ctx, store.RetryIssueNameParams{WorkspaceID: wsID, ID: issue})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return out, nil, err
		}
		out.Retry = &RetryAudienceFacts{Issue: name, Blocked: int64(retry.Len())}
	}
	if def.Paid.Set() {
		out.Paid = &PaidAudienceFacts{Who: def.Paid.Who}
		if def.Paid.Tier != "" {
			if id, err := uuid.Parse(def.Paid.Tier); err == nil {
				if tiers, err := paidaccess.Tiers(ctx, q, wsID); err == nil {
					for _, t := range tiers {
						if t.ID == id {
							out.Paid.Tier, out.Paid.TierTop = t.Short, t.Top
						}
					}
				}
			}
		}
	}
	seen := make(map[string]bool, len(rows))
	var recipients []Recipient
	if withRecipients {
		recipients = make([]Recipient, 0, len(rows))
	}
	sunset := out.UnengagedExcluded
	counted := countingFrom(ctx)
	for i, r := range rows {
		if i%5000 == 0 {
			counted(int64(i), int64(len(rows)))
		}
		if !keep.Keeps(r.ID) || !retry.Keeps(r.ID) {
			continue
		}
		out.Base++
		switch {
		case r.Suppressed:
			out.Suppressed++
			out.SuppressedEmails = append(out.SuppressedEmails, r.Email)
			continue
		case r.RoleAddress || emailaddr.IsRole(r.Email):
			out.Role++
			continue
		}
		key := mailboxKey(r.Email)
		if seen[key] {
			out.Duplicates++
			continue
		}
		seen[key] = true
		if r.BroadcastsStopped {
			// The sunset policy stopped Broadcasts to them, unless the
			// campaign overrides it.
			if !out.SunsetOverride {
				out.Stopped++
				continue
			}
			out.StoppedIncluded++
		}
		if sunset && r.Tier == "sunset" {
			out.SunsetExcluded++
			continue
		}
		out.Final++
		if r.Tier == "sunset" {
			out.Unengaged++
		}
		if m, ok := keep.Member(r.ID); ok {
			out.Paid.count(m)
		}
		if r.Bounced {
			out.RecentlyBounced++
		}
		if r.NoFirstName {
			out.NoFirstName++
		}
		mbp := r.MailboxProvider
		if mbp == "" || mbp == routing.Other {
			mbp = routing.MailboxProvider(r.Email)
		}
		out.Mailboxes[mbp]++
		out.Languages[strings.ToLower(strings.TrimSpace(r.Language))]++
		if !withRecipients {
			continue
		}
		lists := make([]string, len(r.ListIds))
		for i, l := range r.ListIds {
			lists[i] = l.String()
		}
		recipients = append(recipients, Recipient{Email: r.Email, MailboxProvider: mbp, Lists: lists, Tier: r.Tier, Language: r.Language})
	}
	counted(int64(len(rows)), int64(len(rows)))
	return out, recipients, nil
}

type countingKey struct{}

// withCounting has the audience count of a run report how many contacts it
// went through (the progress of a run).
func withCounting(ctx context.Context, fn func(done, total int64)) context.Context {
	return context.WithValue(ctx, countingKey{}, fn)
}

func countingFrom(ctx context.Context) func(done, total int64) {
	if fn, ok := ctx.Value(countingKey{}).(func(done, total int64)); ok && fn != nil {
		return fn
	}
	return func(int64, int64) {}
}

// RouteOverride is campaigns.route_override: a split that replaces the
// stream's rules for this campaign ("Send via SES only").
type RouteOverride struct {
	Splits []routing.Split `json:"splits"`
}

// ParseRouteOverride reads campaigns.route_override (nil when unset).
func ParseRouteOverride(raw json.RawMessage) *RouteOverride {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var o RouteOverride
	if json.Unmarshal(raw, &o) != nil || len(o.Splits) == 0 {
		return nil
	}
	return &o
}

// planRoute decides the provider of every recipient with the stream's rules
// (or the campaign's override) and counts them per provider, and per
// provider and mailbox provider in cells (what the estimate plays out).
// order lists the providers in the order of the default split, then the
// others.
func planRoute(set routing.Set, override *RouteOverride, recipients []Recipient, senderDomain string) (counts map[uuid.UUID]int64, cells map[uuid.UUID]map[string]int64, order []uuid.UUID, ok bool) {
	counts = map[uuid.UUID]int64{}
	cells = map[uuid.UUID]map[string]int64{}
	if override != nil {
		set = routing.Set{Rules: []routing.Rule{{Stream: providers.StreamBroadcast, Splits: override.Splits, Enabled: true, Priority: 1}}, Policies: set.Policies}
	}
	def, hasDefault := set.Default(providers.StreamBroadcast)
	if !hasDefault {
		return counts, cells, order, false
	}
	for _, sp := range def.Splits {
		if !slices.Contains(order, sp.ProviderID) {
			order = append(order, sp.ProviderID)
		}
	}
	for _, r := range recipients {
		d, found := set.Resolve(routing.Recipient{Stream: providers.StreamBroadcast, Email: r.Email, Lists: r.Lists, SenderDomain: senderDomain, MailboxProvider: r.MailboxProvider})
		if !found {
			continue
		}
		counts[d.ProviderID]++
		if cells[d.ProviderID] == nil {
			cells[d.ProviderID] = map[string]int64{}
		}
		cells[d.ProviderID][r.MailboxProvider]++
		if !slices.Contains(order, d.ProviderID) {
			order = append(order, d.ProviderID)
		}
	}
	return counts, cells, order, true
}
