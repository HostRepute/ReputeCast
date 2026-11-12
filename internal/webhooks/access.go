package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// What an event carries decides who may have it sent somewhere.
// webhooks:manage chooses where events go; the data in them is shown to
// those who hold its view permission, so subscribing to an event needs
// that permission too, and an event that carries contacts carries them
// from every list, so people and keys limited to some lists can't take it.
// Every delivery checks the same against the person the endpoint sends
// for, as they are at that moment: webhooks:manage alone must never be a
// way to stream contacts out, nor go on doing so after its maker left.

// EventNeeds is what receiving an event needs.
type EventNeeds struct {
	// Permissions are the view permissions of what the event carries.
	Permissions []authz.Permission
	// Contacts: the event carries contacts (an address, a phone number, a
	// profile) from every list.
	Contacts bool
}

var (
	contactData = EventNeeds{Permissions: []authz.Permission{authz.SubscribersView}, Contacts: true}
	paidMembers = EventNeeds{Permissions: []authz.Permission{authz.EarnView, authz.SubscribersView}, Contacts: true}
)

// groupNeeds are the needs of each group of events, by the part of the
// name before the dot.
var groupNeeds = map[string]EventNeeds{
	"contact": contactData,
	// The recipient's address and contact.
	"email": contactData,
	// The signup's answers.
	"form": contactData,
	// The phone number, or the contact who clicked.
	"sms":      contactData,
	"whatsapp": contactData,
	"push":     contactData,
	// The referrer's address and name.
	"referral": {Permissions: []authz.Permission{authz.GrowView, authz.SubscribersView}, Contacts: true},
	// The member's, buyer's or reader's address.
	"subscription": paidMembers,
	"access":       paidMembers,
	"gift":         paidMembers,
	"conversion":   paidMembers,
	"order":        {Permissions: []authz.Permission{authz.ProductsView, authz.SubscribersView}, Contacts: true},
	// The message log shows the same, recipients included, to
	// transactional:view; transactional mail isn't kept per list.
	"message":    {Permissions: []authz.Permission{authz.TransactionalView}},
	"campaign":   {Permissions: []authz.Permission{authz.CampaignsView}},
	"approval":   {Permissions: []authz.Permission{authz.CampaignsView}},
	"automation": {Permissions: []authz.Permission{authz.AutomationsView}},
	"provider":   {Permissions: []authz.Permission{authz.ProvidersView}},
	"post":       {Permissions: []authz.Permission{authz.WebsiteView}},
	// webhook.test carries a sentence and nothing else.
	"webhook": {},
}

// eventNeeds are the events whose data differs from their group's.
var eventNeeds = map[string]EventNeeds{
	// A template's name and status.
	"whatsapp.template_status": {Permissions: []authz.Permission{authz.ProvidersView}},
	// What people wrote back, with their address (M58): the replies inbox
	// shows it to replies:view.
	"email.replied":  {Permissions: []authz.Permission{authz.RepliesView}, Contacts: true},
	"email.received": {Permissions: []authz.Permission{authz.RepliesView}, Contacts: true},
}

// NeedsOf returns what receiving an event needs (an email.batch as its
// email.* events). A group nobody has described counts as carrying
// contacts, so a new event never goes further than contacts do until
// someone decides otherwise.
func NeedsOf(event string) EventNeeds {
	if n, ok := eventNeeds[event]; ok {
		return n
	}
	group, _, _ := strings.Cut(event, ".")
	if n, ok := groupNeeds[group]; ok {
		return n
	}
	return contactData
}

// Permissions returns the view permissions an event needs (the
// integrations' triggers share them).
func Permissions(event string) []authz.Permission {
	return slices.Clone(NeedsOf(event).Permissions)
}

// covered returns the catalogue's events a subscription takes: the event,
// a group's events for "group.*", every event for "*".
func covered(name string) []string {
	if name != "*" && !strings.HasSuffix(name, ".*") {
		return []string{name}
	}
	prefix := strings.TrimSuffix(name, "*")
	var out []string
	for _, e := range Events {
		if strings.HasPrefix(e.Name, prefix) {
			out = append(out, e.Name)
		}
	}
	return out
}

// missing returns what someone who holds perms, within scope, lacks to
// receive event: the first permission they don't hold, or contacts when
// the event carries every list's contacts and they reach some lists only.
func missing(event string, perms authz.Set, scope authz.Scope) (perm authz.Permission, contacts bool) {
	n := NeedsOf(event)
	for _, p := range n.Permissions {
		if !perms.Has(p) {
			return p, false
		}
	}
	return "", n.Contacts && len(scope.Lists) > 0
}

// CheckEvents refuses subscriptions (event names, "group.*" or "*") with
// an event the caller couldn't see in ReputeCast. The problem names the
// event, and who can help; the permission key is its permission field.
func CheckEvents(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal, names []string) *api.Problem {
	l := i18n.LangFrom(ctx)
	if p == nil {
		p = &authz.Principal{} // nobody: holds nothing
	}
	for _, name := range names {
		for _, ev := range covered(name) {
			perm, contacts := missing(ev, p.Permissions, p.Scope)
			if perm == "" && !contacts {
				continue
			}
			args := i18n.Args{"event": ev, "pattern": name, "ask": "no"}
			if contacts {
				msg := mEventScope
				if name != ev {
					msg = mPatternScope
				}
				return api.Err(http.StatusForbidden, api.CodePermissionDenied, msg.Text(l, args))
			}
			helpers := access.Helpers(ctx, q, wsID, perm)
			if len(helpers) > 0 {
				args["ask"], args["helpers"] = "yes", i18n.OrList(l.Locale, helpers)
			}
			args["permission"] = authz.LabelIn(l, perm)
			msg := mEventNeeds
			if name != ev {
				msg = mPatternNeeds
			}
			pr := api.Err(http.StatusForbidden, api.CodePermissionDenied, msg.Text(l, args))
			pr.Permission, pr.Helpers = string(perm), helpers
			return pr
		}
	}
	return nil
}

// payloadHidden says why a reader can't see a delivery's payload, or ""
// when they can.
func payloadHidden(l i18n.Lang, p *authz.Principal, event string) string {
	if p == nil {
		return ""
	}
	perm, contacts := missing(event, p.Permissions, p.Scope)
	switch {
	case perm != "":
		return mPayloadNeeds.Text(l, i18n.Args{"permission": authz.LabelIn(l, perm)})
	case contacts:
		return mPayloadScope.Text(l, nil)
	}
	return ""
}

// checkEvents is CheckEvents for the endpoint's form: the problem shows
// beside the events.
func (s *Service) checkEvents(ctx context.Context, ws *store.Workspace, p *authz.Principal, events []string) error {
	if pr := CheckEvents(ctx, s.Q, ws.ID, p, events); pr != nil {
		pr.Errors = []*huma.ErrorDetail{{Location: "body.events", Message: pr.Detail}}
		return pr
	}
	return nil
}

// sendsFor is whom an endpoint sends for from now on (webhooks.created_by):
// the person who sets its events, or resumes it, a key's creator for a key.
// Nil keeps whom it sent for, when a key has no creator.
func sendsFor(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// pauseReason is why ReputeCast paused an endpoint (webhooks.paused_reason),
// written in each reader's language when it's read.
type pauseReason struct {
	// Why is left (no longer an active member), lacks (a permission the
	// event needs) or scoped (some lists only, and the event carries every
	// list's contacts).
	Why        string `json:"why"`
	Name       string `json:"name"`
	Event      string `json:"event"`
	Permission string `json:"permission,omitempty"`
}

func (r pauseReason) in(l i18n.Lang) string {
	args := i18n.Args{"name": r.Name, "event": r.Event}
	switch r.Why {
	case "left":
		return mPausedLeft.Text(l, args)
	case "scoped":
		return mPausedScoped.Text(l, args)
	}
	args["permission"] = authz.LabelIn(l, authz.Permission(r.Permission))
	return mPausedLacks.Text(l, args)
}

// pausedIn reads a stored reason in a language; "" when a person paused
// the endpoint.
func pausedIn(l i18n.Lang, raw json.RawMessage) string {
	var r pauseReason
	if len(raw) == 0 || json.Unmarshal(raw, &r) != nil || r.Why == "" {
		return ""
	}
	return r.in(l)
}

// withheld says why a claimed delivery can't go: the person the endpoint
// sends for is no longer an active member, lost the permission its event
// needs, or reaches some lists only and the event carries every list's
// contacts. Endpoints that send for nobody (from before people were
// recorded) go on as they did; an error leaves the delivery for later.
func (s *Service) withheld(ctx context.Context, d store.HooksClaimDeliveryRow) (*pauseReason, error) {
	n := NeedsOf(d.Event)
	if d.CreatedBy == nil || len(n.Permissions) == 0 && !n.Contacts {
		return nil, nil
	}
	m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: d.WorkspaceID, UserID: *d.CreatedBy})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	why := &pauseReason{Event: d.Event}
	if err != nil || m.Status != "active" {
		why.Why = "left"
	} else {
		perm, contacts := missing(d.Event, authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions), access.ParseScope(m.Scope))
		switch {
		case perm != "":
			why.Why, why.Permission = "lacks", string(perm)
		case contacts:
			why.Why = "scoped"
		default:
			return nil, nil
		}
	}
	if u, err := s.Q.GetUser(ctx, *d.CreatedBy); err == nil {
		why.Name = strings.TrimSpace(u.Name)
		if why.Name == "" {
			why.Name = u.Email
		}
	}
	return why, nil
}

// withhold puts a delivery back to wait with the endpoint's other events
// and pauses the endpoint, saying why to the people who manage webhooks.
// Nothing more goes out until someone who can see what its events carry
// resumes it, on their own behalf.
func (s *Service) withhold(ctx context.Context, d store.HooksClaimDeliveryRow, why pauseReason) error {
	reason, err := json.Marshal(why)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HooksWithhold(ctx, d.ID); err != nil {
			return err
		}
		h, err := q.HooksAutoPause(ctx, store.HooksAutoPauseParams{ID: d.WebhookID, Reason: reason})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // another delivery paused it first
		}
		if err != nil {
			return err
		}
		meta := map[string]any{"event": d.Event, "sends_for": why.Name}
		if why.Permission != "" {
			meta["permission"] = why.Permission
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &h.WorkspaceID, Action: "webhook.paused", TargetType: "webhook", TargetID: h.ID.String(), TargetName: h.Url,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: why.in(i18n.Default),
			Before: map[string]any{"enabled": true}, After: map[string]any{"enabled": false}, Metadata: meta})
	})
}
