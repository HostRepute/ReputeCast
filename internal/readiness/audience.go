package readiness

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
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
}

// ParseAudience reads a stored audience definition.
func ParseAudience(raw json.RawMessage) AudienceDef {
	var a AudienceDef
	_ = json.Unmarshal(raw, &a)
	return a
}

// Empty reports whether the definition selects nobody.
func (a AudienceDef) Empty() bool { return len(a.Lists) == 0 && len(a.Segments) == 0 }

// ExcludesSunset reports whether the sunset tier is excluded.
func (a AudienceDef) ExcludesSunset() bool { return slices.Contains(a.ExcludeTiers, "sunset") }

// Recipient is one final recipient, as routing needs it.
type Recipient struct {
	Email           string
	MailboxProvider string
	Lists           []string
	Tier            string
}

// roleLocals are shared inboxes. Role addresses never get Broadcasts: there
// is no personal consent behind them and their complaint rates are high.
// (Once the audience domain marks contacts.role_address, that flag decides.)
var roleLocals = map[string]bool{
	"abuse": true, "admin": true, "administrator": true, "billing": true, "contact": true, "enquiries": true, "hello": true,
	"help": true, "helpdesk": true, "hostmaster": true, "info": true, "inquiries": true, "marketing": true, "noreply": true,
	"no-reply": true, "office": true, "postmaster": true, "press": true, "sales": true, "security": true, "support": true,
	"team": true, "webmaster": true, "careers": true, "jobs": true, "hr": true, "orders": true, "privacy": true,
}

// IsRoleAddress reports whether an address is a shared inbox (info@, sales@…).
func IsRoleAddress(email string) bool {
	local := strings.ToLower(email)
	if i := strings.LastIndexByte(local, '@'); i >= 0 {
		local = local[:i]
	}
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	return roleLocals[local]
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
	allLists int
}

func (s *Service) names(ctx context.Context, q *store.Queries, wsID uuid.UUID) (audienceNames, error) {
	n := audienceNames{lists: map[string]string{}, segments: map[string]string{}}
	lists, err := q.ReadinessListNames(ctx, wsID)
	if err != nil {
		return n, err
	}
	for _, l := range lists {
		n.lists[l.ID.String()] = l.Name
	}
	n.allLists = len(lists)
	segs, err := q.ReadinessSegmentNames(ctx, wsID)
	if err != nil {
		return n, err
	}
	for _, sg := range segs {
		n.segments[sg.ID.String()] = sg.Name
	}
	return n, nil
}

// labels returns the base label ("Weekly Digest list", "All subscribers"),
// the group label and the names of the lists and segments.
func (n audienceNames) labels(a AudienceDef) (base, group string, names []string) {
	names = []string{}
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
	switch {
	case len(a.Segments) == 0 && len(a.ExcludeLists) == 0 && n.allLists > 1 && len(names) == n.allLists:
		return "All subscribers", "All subscribers", names
	case len(names) == 1 && len(a.Lists) == 1:
		return names[0] + " list", names[0], names
	case len(names) == 1:
		return names[0] + " segment", names[0], names
	case len(names) == 0:
		return "Audience", "Audience", names
	}
	joined := strings.Join(names, " + ")
	return joined, joined, names
}

// computeAudience counts the final recipients of a definition: suppressed,
// role, duplicate and (when excluded) sunset-tier contacts are removed.
func (s *Service) computeAudience(ctx context.Context, q *store.Queries, wsID uuid.UUID, def AudienceDef, unengagedDays int, now time.Time) (AudienceFacts, []Recipient, error) {
	out := AudienceFacts{Mailboxes: map[string]int64{}, UnengagedDays: unengagedDays, UnengagedExcluded: def.ExcludesSunset(), Lists: []string{}}
	names, err := s.names(ctx, q, wsID)
	if err != nil {
		return out, nil, err
	}
	out.BaseLabel, out.GroupLabel, out.Lists = names.labels(def)
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
		BouncedSince: now.AddDate(0, 0, -30), WorkspaceID: wsID, Lists: ids(def.Lists), Contacts: contacts,
		ExcludeLists: ids(def.ExcludeLists), ExcludeContacts: exclude,
	})
	if err != nil {
		return out, nil, err
	}
	seen := make(map[string]bool, len(rows))
	recipients := make([]Recipient, 0, len(rows))
	sunset := out.UnengagedExcluded
	for _, r := range rows {
		out.Base++
		switch {
		case r.Suppressed:
			out.Suppressed++
			continue
		case IsRoleAddress(r.Email):
			out.Role++
			continue
		}
		key := mailboxKey(r.Email)
		if seen[key] {
			out.Duplicates++
			continue
		}
		seen[key] = true
		if sunset && r.Tier == "sunset" {
			out.SunsetExcluded++
			continue
		}
		out.Final++
		if r.Tier == "sunset" {
			out.Unengaged++
		}
		if r.Bounced {
			out.RecentlyBounced++
		}
		mbp := r.MailboxProvider
		if mbp == "" || mbp == routing.Other {
			mbp = routing.MailboxProvider(r.Email)
		}
		out.Mailboxes[mbp]++
		lists := make([]string, len(r.ListIds))
		for i, l := range r.ListIds {
			lists[i] = l.String()
		}
		recipients = append(recipients, Recipient{Email: r.Email, MailboxProvider: mbp, Lists: lists, Tier: r.Tier})
	}
	return out, recipients, nil
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
// (or the campaign's override) and counts them per provider. order lists the
// providers in the order of the default split, then the others.
func planRoute(set routing.Set, override *RouteOverride, recipients []Recipient, senderDomain string) (counts map[uuid.UUID]int64, order []uuid.UUID, ok bool) {
	counts = map[uuid.UUID]int64{}
	if override != nil {
		set = routing.Set{Rules: []routing.Rule{{Stream: providers.StreamBroadcast, Splits: override.Splits, Enabled: true, Priority: 1}}, Policies: set.Policies}
	}
	def, hasDefault := set.Default(providers.StreamBroadcast)
	if !hasDefault {
		return counts, order, false
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
		if !slices.Contains(order, d.ProviderID) {
			order = append(order, d.ProviderID)
		}
	}
	return counts, order, true
}
