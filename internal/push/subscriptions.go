package push

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/push/webpush"
	"github.com/hostrepute/reputecast/internal/store"
)

// Subscriptions (brief §3.8). Each has its site, browser and time zone
// (from the browser), and a consent record: the page, the exact prompt text
// and its version, the IP address and the time. It's linked to a contact
// when the reader is a signed-in member, arrived from a link that carried a
// token, or turned notifications on from the account page; otherwise it's
// anonymous. Subscriptions aren't audited: they're counted per site.

// Browsers, as the app groups them.
const (
	BrowserChrome        = "chrome"
	BrowserChromeAndroid = "chrome_android"
	BrowserEdge          = "edge"
	BrowserFirefox       = "firefox"
	BrowserSafari        = "safari"
	BrowserSafariIOS     = "safari_ios"
	BrowserOther         = "other"
)

// BrowserOf reads a browser and its platform from a User-Agent.
func BrowserOf(ua string) (browser, platform string) {
	switch {
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"):
		platform = "iOS"
	case strings.Contains(ua, "Android"):
		platform = "Android"
	case strings.Contains(ua, "Windows"):
		platform = "Windows"
	case strings.Contains(ua, "CrOS"):
		platform = "ChromeOS"
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		platform = "macOS"
	case strings.Contains(ua, "Linux"):
		platform = "Linux"
	}
	switch {
	case platform == "iOS":
		// Every browser on iPhone and iPad is WebKit, and only a site added
		// to the Home Screen can subscribe.
		browser = BrowserSafariIOS
	case strings.Contains(ua, "Edg/"), strings.Contains(ua, "EdgA/"):
		browser = BrowserEdge
	case strings.Contains(ua, "Firefox/"):
		browser = BrowserFirefox
	case strings.Contains(ua, "OPR/"), strings.Contains(ua, "SamsungBrowser/"), strings.Contains(ua, "YaBrowser/"):
		browser = BrowserOther
	case strings.Contains(ua, "Chrome/") && platform == "Android":
		browser = BrowserChromeAndroid
	case strings.Contains(ua, "Chrome/"), strings.Contains(ua, "Chromium/"):
		browser = BrowserChrome
	case strings.Contains(ua, "Safari/") && platform == "macOS":
		browser = BrowserSafari
	default:
		browser = BrowserOther
	}
	return browser, platform
}

// pushHosts are the push services ReputeCast sends to: a subscription's
// endpoint must be on one of them, so the instance never posts anywhere a
// page asks it to.
var pushHosts = []string{"fcm.googleapis.com", "android.googleapis.com", "updates.push.services.mozilla.com", "push.services.mozilla.com",
	".push.apple.com", ".notify.windows.com"}

// knownPushService reports whether an endpoint is on a push service.
func knownPushService(endpoint string) bool {
	o, err := webpush.Origin(endpoint)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(o, "https://")
	for _, h := range pushHosts {
		if host == strings.TrimPrefix(h, ".") || (strings.HasPrefix(h, ".") && strings.HasSuffix(host, h)) {
			return true
		}
	}
	return false
}

// Subscribe errors the public endpoint answers with.
var (
	ErrPushOff      = errors.New("push: push isn't on for this workspace")
	ErrSiteRefused  = errors.New("push: the site isn't a push site of this workspace")
	ErrBadEndpoint  = errors.New("push: the subscription isn't on a push service ReputeCast sends to")
	ErrSubscription = errors.New("push: the subscription's keys are malformed")
)

// SubscribeRequest is what the prompt sends once the browser allowed
// notifications.
type SubscribeRequest struct {
	Origin   string
	Endpoint string
	P256dh   string
	Auth     string
	TimeZone string
	Page     string
	// FormID and Version are the prompt that asked, when one did.
	FormID  uuid.UUID
	Version int
	// Token is a reader token from the page's address (a personal link
	// from an email or a text, the preference center's link).
	Token     string
	Member    uuid.UUID
	IP        *netip.Addr
	UserAgent string
	// LinkedBy says why a contact is known: member, link, account, preferences.
	LinkedBy string
}

// Subscribed is the answer to a subscribe.
type Subscribed struct {
	ID     uuid.UUID
	Linked bool
	New    bool
}

// readerKinds are the token kinds whose contact a subscription can be
// linked to: personal links readers carry to the site.
var readerKinds = []linktoken.Kind{linktoken.Preferences, linktoken.ViewOnline, linktoken.Unsubscribe, TokenReader, 160 /* earn's upgrade link */}

// TokenReader is a link that carries the reader to a site, so a
// subscription made there is linked to them (design expansion, X22:
// 220–229).
const TokenReader linktoken.Kind = 220

// readerOf resolves a reader token to a contact of the workspace.
func (s *Service) readerOf(ws *store.Workspace, token string) uuid.UUID {
	if s.Links == nil || token == "" || len(token) > 300 {
		return uuid.Nil
	}
	for _, k := range readerKinds {
		if c, err := s.Links.Verify(token, k); err == nil && c.WorkspaceID == ws.ID && c.ContactID != uuid.Nil {
			return c.ContactID
		}
	}
	return uuid.Nil
}

// Subscribe saves a browser's subscription from a listed site, with its
// consent record, linked to a contact when ReputeCast knows who it is.
func (s *Service) Subscribe(ctx context.Context, ws *store.Workspace, in SubscribeRequest) (Subscribed, error) {
	key, err := s.liveKey(ctx, s.Q, ws)
	if errors.Is(err, errNoKeys) {
		return Subscribed{}, ErrPushOff
	}
	if err != nil {
		return Subscribed{}, err
	}
	site, err := s.siteOf(ctx, ws, in.Origin)
	if err != nil {
		return Subscribed{}, ErrSiteRefused
	}
	sub, err := webpush.ParseSubscription(in.Endpoint, in.P256dh, in.Auth)
	if err != nil {
		if errors.Is(err, webpush.ErrEndpoint) {
			return Subscribed{}, ErrBadEndpoint
		}
		return Subscribed{}, ErrSubscription
	}
	if !knownPushService(sub.Endpoint) {
		return Subscribed{}, ErrBadEndpoint
	}
	browser, platform := BrowserOf(in.UserAgent)
	zone := strings.TrimSpace(in.TimeZone)
	if _, err := time.LoadLocation(zone); err != nil || len(zone) > 64 {
		zone = ""
	}
	contact, linkedBy := in.Member, in.LinkedBy
	if contact != uuid.Nil && linkedBy == "" {
		linkedBy = "member"
	}
	if contact == uuid.Nil {
		if c := s.readerOf(ws, in.Token); c != uuid.Nil {
			contact, linkedBy = c, "link"
		}
	}
	if contact != uuid.Nil {
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL)`,
			contact, ws.ID).Scan(&ok); err != nil || !ok {
			contact, linkedBy = uuid.Nil, ""
		}
	}
	now := s.now()
	var prompt struct {
		form    *uuid.UUID
		version *int32
		text    string
	}
	if in.FormID != uuid.Nil {
		if r, err := s.Q.PushPrompt(ctx, store.PushPromptParams{FormID: in.FormID, WorkspaceID: ws.ID}); err == nil {
			v := r.LiveVersion
			if in.Version > 0 && int32(in.Version) <= r.LiveVersion {
				v = int32(in.Version)
			}
			if live, _ := s.liveConfig(ctx, in.FormID, v); live != nil {
				prompt.form, prompt.version = &in.FormID, &v
				prompt.text = live.Title
				if live.Message != "" {
					prompt.text += " · " + live.Message
				}
			}
		}
	}
	page := clip(strings.TrimSpace(in.Page), 500)
	out := Subscribed{}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		cur, err := q.PushSubscriptionByEndpoint(ctx, store.PushSubscriptionByEndpointParams{WorkspaceID: ws.ID, Endpoint: sub.Endpoint})
		switch {
		case err == nil:
			out.ID = cur.ID
			if err := q.PushSubscriptionRefresh(ctx, store.PushSubscriptionRefreshParams{ID: cur.ID, P256dh: sub.P256dh, Auth: sub.Auth, TimeZone: zone, At: &now}); err != nil {
				return err
			}
			if contact != uuid.Nil && cur.ContactID == nil {
				if err := q.PushSubscriptionLink(ctx, store.PushSubscriptionLinkParams{ID: cur.ID, ContactID: &contact, At: &now, LinkedBy: linkedBy}); err != nil {
					return err
				}
			}
			out.Linked = cur.ContactID != nil || contact != uuid.Nil
			return nil
		case !notFound(err):
			return err
		}
		out.ID, out.New = uuid.Must(uuid.NewV7()), true
		var linkedAt *time.Time
		if contact != uuid.Nil {
			linkedAt = &now
			out.Linked = true
		}
		return q.PushSubscriptionInsert(ctx, store.PushSubscriptionInsertParams{ID: out.ID, WorkspaceID: ws.ID, SiteID: site.ID, KeyID: key.ID,
			Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth, Browser: browser, Platform: platform, TimeZone: zone, ContactID: uuidPtr(contact),
			LinkedAt: linkedAt, LinkedBy: linkedBy, PromptFormID: prompt.form, PromptVersion: prompt.version, PromptText: prompt.text, Page: page,
			Ip: in.IP, UserAgent: clip(in.UserAgent, 300), CreatedAt: now, Demo: site.Demo})
	})
	if err != nil {
		return out, err
	}
	if out.New && prompt.form != nil {
		if err := s.RecordPromptEvent(ctx, ws, *prompt.form, "granted"); err != nil {
			s.log().WarnContext(ctx, "push prompt granted", "err", err)
		}
	}
	return out, nil
}

// Unsubscribe ends a browser's subscription: the reader turned
// notifications off on the account page or the preference center
// ("turned_off"), or the browser said it ended ("unsubscribed").
func (s *Service) Unsubscribe(ctx context.Context, ws *store.Workspace, endpoint, reason string) (bool, error) {
	if reason != "turned_off" {
		reason = "unsubscribed"
	}
	cur, err := s.Q.PushSubscriptionByEndpoint(ctx, store.PushSubscriptionByEndpointParams{WorkspaceID: ws.ID, Endpoint: strings.TrimSpace(endpoint)})
	if notFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, s.Q.PushSubscriptionRemove(ctx, store.PushSubscriptionRemoveParams{ID: cur.ID, At: ptr(s.now()), Reason: reason})
}

// PushContactSubscription is one of a contact's subscriptions.
type PushContactSubscription struct {
	ID       string     `json:"id"`
	Browser  string     `json:"browser" enum:"chrome,chrome_android,edge,firefox,safari,safari_ios,other"`
	Platform string     `json:"platform"`
	Site     string     `json:"site"`
	Since    time.Time  `json:"since"`
	LinkedBy string     `json:"linkedBy" enum:"member,link,account,preferences,"`
	LastSeen *time.Time `json:"lastSeen,omitempty"`
	TimeZone string     `json:"timeZone"`
}

// PushContactCard is the contact profile's push subscriptions card.
type PushContactCard struct {
	PushOn        bool                      `json:"pushOn"`
	Subscriptions []PushContactSubscription `json:"subscriptions"`
	CanEdit       bool                      `json:"canEdit"`
}

// ContactCard lists a contact's linked subscriptions.
func (s *Service) ContactCard(ctx context.Context, ws *store.Workspace, p *authz.Principal, contactID uuid.UUID) (PushContactCard, error) {
	out := PushContactCard{Subscriptions: []PushContactSubscription{}, CanEdit: p.Can(authz.SubscribersEdit)}
	var ok bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL)`, contactID, ws.ID).Scan(&ok); err != nil {
		return out, err
	}
	if !ok {
		return out, api.NotFound("Contact")
	}
	_, err := s.liveKey(ctx, s.Q, ws)
	out.PushOn = err == nil
	rows, err := s.Q.PushSubscriptionsOfContact(ctx, store.PushSubscriptionsOfContactParams{WorkspaceID: ws.ID, ContactID: &contactID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Subscriptions = append(out.Subscriptions, PushContactSubscription{ID: r.ID.String(), Browser: r.Browser, Platform: r.Platform, Site: r.SiteHost,
			Since: r.CreatedAt, LinkedBy: r.LinkedBy, LastSeen: r.LastSeenAt, TimeZone: r.TimeZone})
	}
	return out, nil
}

// RemoveContactSubscription ends one of a contact's subscriptions, by a
// person (audited).
func (s *Service) RemoveContactSubscription(ctx context.Context, ws *store.Workspace, contactID, id uuid.UUID) error {
	sub, err := s.Q.PushSubscription(ctx, store.PushSubscriptionParams{ID: id, WorkspaceID: ws.ID})
	if err != nil || sub.ContactID == nil || *sub.ContactID != contactID || sub.RemovedAt != nil {
		return api.NotFound("Push subscription")
	}
	site, _ := s.Q.PushSiteAny(ctx, sub.SiteID)
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushSubscriptionRemove(ctx, store.PushSubscriptionRemoveParams{ID: id, At: ptr(s.now()), Reason: "removed"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_subscription.removed", TargetType: "contact", TargetID: contactID.String(),
			TargetName: site.Host, Before: map[string]any{"browser": sub.Browser, "site": site.Host, "since": sub.CreatedAt}})
	})
}
