package tracker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sites (Settings › Integrations › Website tracking): a site is an origin
// the tracker runs on, maybe linked to a connected store whose product ids
// its events name, whether it waits for the visitor's consent, and the
// secret its server signs identify calls with (sealed, shown once).

// TrackerStoreOption is a connected store a site can be linked to.
type TrackerStoreOption struct {
	ID       string `json:"id" format:"uuid"`
	Name     string `json:"name"`
	Platform string `json:"platform" enum:"shopify,woocommerce"`
}

// TrackerRefusal is an origin whose events a site refused (another site
// using its snippet), in the last 24 hours.
type TrackerRefusal struct {
	Origin string    `json:"origin"`
	Count  int64     `json:"count"`
	LastAt time.Time `json:"lastAt"`
}

// TrackerSnippet is what a site adds: the script tag, the Shopify custom
// pixel's code, and the WordPress plugin's setting.
type TrackerSnippet struct {
	Script       string `json:"script"`
	ShopifyPixel string `json:"shopifyPixel"`
	WordPress    string `json:"wordpress"`
}

// TrackerSite is a tracked site.
type TrackerSite struct {
	ID          string              `json:"id" format:"uuid"`
	Name        string              `json:"name"`
	Origin      string              `json:"origin" example:"https://shop.acme.io"`
	Store       *TrackerStoreOption `json:"store,omitempty"`
	WaitConsent bool                `json:"waitConsent"`
	// HasIdentifySecret: the site has a secret to sign identify calls with
	// (never returned but by create and rotate).
	HasIdentifySecret bool  `json:"hasIdentifySecret"`
	EventsToday       int64 `json:"eventsToday" doc:"Events since midnight in the workspace's time zone"`
	// CreatedBy is the name of the member who added the site ("" once
	// they left).
	CreatedBy   string           `json:"createdBy"`
	LastEventAt *time.Time       `json:"lastEventAt,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
	Refused     []TrackerRefusal `json:"refused" doc:"Origins refused in the last 24 hours, most refused first"`
	Snippet     TrackerSnippet   `json:"snippet"`
}

// TrackerSiteList is the Website tracking page.
type TrackerSiteList struct {
	Items []TrackerSite `json:"items"`
	// Stores are the connected stores a site can be linked to.
	Stores    []TrackerStoreOption `json:"stores"`
	ScriptURL string               `json:"scriptUrl"`
	// CanManage: the caller holds integrations:manage; Helpers are up to
	// two members who do (a Viewer's "Ask Marcus Chen or Linh Nguyen.").
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// TrackerSiteInput adds a site.
type TrackerSiteInput struct {
	Name string `json:"name" maxLength:"200"`
	// Origin is the site's address: "shop.acme.io" or "https://shop.acme.io".
	Origin      string `json:"origin" maxLength:"300"`
	StoreID     string `json:"storeId,omitempty" maxLength:"40"`
	WaitConsent bool   `json:"waitConsent" doc:"Nothing is stored until the site's consent banner calls reputecast('consent', true)"`
}

// TrackerSiteUpdate changes a site; its origin stays.
type TrackerSiteUpdate struct {
	Name        string `json:"name" maxLength:"200"`
	StoreID     string `json:"storeId,omitempty" maxLength:"40"`
	WaitConsent bool   `json:"waitConsent"`
}

// TrackerSiteCreated is a new site, with its identify secret, shown once.
type TrackerSiteCreated struct {
	Site           TrackerSite `json:"site"`
	IdentifySecret string      `json:"identifySecret" doc:"Shown once: the site's server signs identify calls with it"`
}

// TrackerSecret is a site's new identify secret, shown once.
type TrackerSecret struct {
	IdentifySecret string `json:"identifySecret"`
}

// TrackerRecentEvent is one of a site's last events (never an address).
type TrackerRecentEvent struct {
	Kind         string    `json:"kind" enum:"page,product,cart,custom"`
	Path         string    `json:"path"`
	ProductTitle string    `json:"productTitle"`
	Name         string    `json:"name"`
	Known        bool      `json:"known" doc:"A known contact's, else a visitor's"`
	At           time.Time `json:"at"`
}

// TrackerRecentEvents is the setup's live check.
type TrackerRecentEvents struct {
	Items []TrackerRecentEvent `json:"items"`
}

// ParseOrigin reads a site's address ("shop.acme.io",
// "https://shop.acme.io/products") as an origin: https, a host and maybe a
// port. http is accepted only for localhost, and only when local is true
// (development).
func ParseOrigin(raw string, local bool) (origin, host string, ok bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", "", false
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || u.User != nil {
		return "", "", false
	}
	h := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	isLocal := h == "localhost" || h == "127.0.0.1" || strings.HasSuffix(h, ".localhost")
	scheme := strings.ToLower(u.Scheme)
	switch {
	case scheme == "https":
	case scheme == "http" && isLocal && local:
	default:
		return "", "", false
	}
	if !isLocal && (!strings.Contains(h, ".") || net.ParseIP(h) != nil) {
		return "", "", false
	}
	host = h
	if p := u.Port(); p != "" {
		if (scheme == "https" && p != "443") || (scheme == "http" && p != "80") {
			host += ":" + p
		}
	}
	origin = scheme + "://" + host
	if len(origin) < 8 || len(origin) > 255 {
		return "", "", false
	}
	return origin, host, true
}

// ownHost reports whether host shows ReputeCast's own pages: the
// instance's, or one a workspace holds (a custom domain, the publication
// website, a link domain). Those pages are never tracked.
func (s *Service) ownHost(ctx context.Context, origin string) (bool, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return false, err
	}
	h := u.Hostname()
	if b, err := url.Parse(s.URLs.Base()); err == nil && strings.EqualFold(strings.TrimSuffix(b.Hostname(), "."), h) {
		return true, nil
	}
	holders, err := s.Hosts.Holders(ctx, h)
	return len(holders) > 0, err
}

// connectedStore checks a site's store: one of the workspace's connected
// stores, or none.
func (s *Service) connectedStore(ctx context.Context, ws *store.Workspace, raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	l := i18n.LangFrom(ctx)
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, api.Invalid("body.storeId", mProblemStore.Text(l, nil))
	}
	rows, err := s.Q.CommerceStoresConnected(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, st := range rows {
		if st.ID == id {
			return &id, nil
		}
	}
	return nil, api.Invalid("body.storeId", mProblemStore.Text(l, nil))
}

func siteName(ctx context.Context, raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if n := len([]rune(name)); n < 1 || n > 80 {
		return "", api.Invalid("body.name", mProblemName.Text(i18n.LangFrom(ctx), nil))
	}
	return name, nil
}

// newSecret is an identify secret: rcid_ and 32 random bytes.
func newSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "rcid_" + base64.RawURLEncoding.EncodeToString(b)
}

func secretAD(id uuid.UUID) string { return "tracker_site:" + id.String() + ":identify_secret" }

func siteGone(ctx context.Context) error {
	return api.Err(http.StatusNotFound, api.CodeNotFound, mProblemSiteGone.Text(i18n.LangFrom(ctx), nil))
}

func storeOf(id *uuid.UUID, name, platform string) *TrackerStoreOption {
	if id == nil {
		return nil
	}
	return &TrackerStoreOption{ID: id.String(), Name: name, Platform: platform}
}

// startOfDay is midnight today in the workspace's time zone.
func startOfDay(ws *store.Workspace, now time.Time) time.Time {
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// Sites lists the workspace's sites.
func (s *Service) Sites(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TrackerSiteList, error) {
	rows, err := s.Q.TrackerSites(ctx, ws.ID)
	if err != nil {
		return TrackerSiteList{}, err
	}
	out := TrackerSiteList{Items: make([]TrackerSite, 0, len(rows)), Stores: []TrackerStoreOption{}, ScriptURL: s.URLs.TrackerScript(),
		CanManage: p != nil && p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID)}
	stores, err := s.Q.CommerceStoresConnected(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, st := range stores {
		out.Stores = append(out.Stores, TrackerStoreOption{ID: st.ID.String(), Name: st.Name, Platform: st.Platform})
	}
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	now := s.now()
	counts, refused, err := s.siteNumbers(ctx, ws, ids, now)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, s.siteOf(ctx, ws, store.TrackerSiteRow(r), counts[r.ID], refused[r.ID]))
	}
	return out, nil
}

// helpers are up to two members who may manage the sites.
func (s *Service) helpers(ctx context.Context, ws uuid.UUID) []string {
	out := []string{}
	rows, err := s.Q.CommerceMembers(ctx, ws)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if name := strings.TrimSpace(r.Name); name != "" && authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.IntegrationsManage) {
			out = append(out, name)
			if len(out) == 2 {
				break
			}
		}
	}
	return out
}

// siteNumbers are the sites' events today and their refused origins.
func (s *Service) siteNumbers(ctx context.Context, ws *store.Workspace, ids []uuid.UUID, now time.Time) (map[uuid.UUID]int64, map[uuid.UUID][]TrackerRefusal, error) {
	counts := map[uuid.UUID]int64{}
	ev, err := s.Q.TrackerSiteEventsSince(ctx, store.TrackerSiteEventsSinceParams{SiteIds: ids, Since: startOfDay(ws, now)})
	if err != nil {
		return nil, nil, err
	}
	for _, e := range ev {
		counts[e.SiteID] = e.Events
	}
	refused := map[uuid.UUID][]TrackerRefusal{}
	rf, err := s.Q.TrackerSiteRefusedSince(ctx, store.TrackerSiteRefusedSinceParams{SiteIds: ids, Since: now.Add(-24 * time.Hour)})
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rf {
		refused[r.SiteID] = append(refused[r.SiteID], TrackerRefusal{Origin: r.Origin, Count: r.Count, LastAt: r.LastAt})
	}
	return counts, refused, nil
}

func (s *Service) siteOf(ctx context.Context, ws *store.Workspace, r store.TrackerSiteRow, events int64, refused []TrackerRefusal) TrackerSite {
	if refused == nil {
		refused = []TrackerRefusal{}
	}
	return TrackerSite{ID: r.ID.String(), Name: r.Name, Origin: r.Origin, Store: storeOf(r.StoreID, r.StoreName, r.StorePlatform), WaitConsent: r.WaitConsent,
		HasIdentifySecret: len(r.IdentifySecretSealed) > 0, EventsToday: events, LastEventAt: r.LastEventAt, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedByName, Refused: refused,
		Snippet: s.snippet(ctx, ws, r.ID, r.Name, r.WaitConsent)}
}

// Site reads one site.
func (s *Service) Site(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TrackerSite, error) {
	r, err := s.Q.TrackerSite(ctx, store.TrackerSiteParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return TrackerSite{}, siteGone(ctx)
	}
	if err != nil {
		return TrackerSite{}, err
	}
	counts, refused, err := s.siteNumbers(ctx, ws, []uuid.UUID{id}, s.now())
	if err != nil {
		return TrackerSite{}, err
	}
	return s.siteOf(ctx, ws, r, counts[id], refused[id]), nil
}

// siteAudit is what audit entries keep of a site (never its secret).
func siteAudit(name, origin string, storeID *uuid.UUID, wait bool) map[string]any {
	st := ""
	if storeID != nil {
		st = storeID.String()
	}
	return map[string]any{"name": name, "origin": origin, "store_id": st, "wait_consent": wait}
}

// CreateSite adds a site with a new identify secret, returned once.
func (s *Service) CreateSite(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TrackerSiteInput) (TrackerSiteCreated, error) {
	l := i18n.LangFrom(ctx)
	name, err := siteName(ctx, in.Name)
	if err != nil {
		return TrackerSiteCreated{}, err
	}
	origin, host, ok := ParseOrigin(in.Origin, s.development())
	if !ok {
		msg := mProblemOrigin
		if s.development() {
			msg = mProblemOriginLocal
		}
		return TrackerSiteCreated{}, api.Invalid("body.origin", msg.Text(l, nil))
	}
	if own, err := s.ownHost(ctx, origin); err != nil {
		return TrackerSiteCreated{}, err
	} else if own {
		return TrackerSiteCreated{}, api.Invalid("body.origin", mProblemOriginOwn.Text(l, i18n.Args{"host": host}))
	}
	storeID, err := s.connectedStore(ctx, ws, in.StoreID)
	if err != nil {
		return TrackerSiteCreated{}, err
	}
	id := uuid.Must(uuid.NewV7())
	secret := newSecret()
	sealed, err := s.Box.Seal([]byte(secret), secretAD(id))
	if err != nil {
		return TrackerSiteCreated{}, err
	}
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TrackerSiteInsert(ctx, store.TrackerSiteInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Origin: origin, StoreID: storeID,
			WaitConsent: in.WaitConsent, IdentifySecretSealed: sealed, CreatedBy: by, At: now}); err != nil {
			if strings.Contains(err.Error(), "tracker_sites_origin") {
				return api.Conflict(mProblemOriginTaken.Text(l, i18n.Args{"origin": origin}))
			}
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tracker_site.created", TargetType: "tracker_site", TargetID: id.String(),
			TargetName: name, After: siteAudit(name, origin, storeID, in.WaitConsent)})
	})
	if err != nil {
		return TrackerSiteCreated{}, err
	}
	s.forget(ws.ID)
	site, err := s.Site(ctx, ws, id)
	return TrackerSiteCreated{Site: site, IdentifySecret: secret}, err
}

// UpdateSite changes a site's name, store and consent mode.
func (s *Service) UpdateSite(ctx context.Context, ws *store.Workspace, id uuid.UUID, in TrackerSiteUpdate) (TrackerSite, error) {
	name, err := siteName(ctx, in.Name)
	if err != nil {
		return TrackerSite{}, err
	}
	storeID, err := s.connectedStore(ctx, ws, in.StoreID)
	if err != nil {
		return TrackerSite{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.TrackerSite(ctx, store.TrackerSiteParams{ID: id, WorkspaceID: ws.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return siteGone(ctx)
		}
		if err != nil {
			return err
		}
		if err := q.TrackerSiteUpdate(ctx, store.TrackerSiteUpdateParams{ID: id, WorkspaceID: ws.ID, Name: name, StoreID: storeID, WaitConsent: in.WaitConsent,
			At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tracker_site.updated", TargetType: "tracker_site", TargetID: id.String(),
			TargetName: name, Before: siteAudit(before.Name, before.Origin, before.StoreID, before.WaitConsent),
			After: siteAudit(name, before.Origin, storeID, in.WaitConsent)})
	})
	if err != nil {
		return TrackerSite{}, err
	}
	s.forget(ws.ID)
	return s.Site(ctx, ws, id)
}

// DeleteSite stops a site's tracking at once; its events go with the
// retention.
func (s *Service) DeleteSite(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.TrackerSite(ctx, store.TrackerSiteParams{ID: id, WorkspaceID: ws.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return siteGone(ctx)
		}
		if err != nil {
			return err
		}
		now := s.now()
		if _, err := q.TrackerSiteDelete(ctx, store.TrackerSiteDeleteParams{ID: id, WorkspaceID: ws.ID, At: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tracker_site.deleted", TargetType: "tracker_site", TargetID: id.String(),
			TargetName: before.Name, Before: siteAudit(before.Name, before.Origin, before.StoreID, before.WaitConsent)})
	})
	if err == nil {
		s.forget(ws.ID)
	}
	return err
}

// RotateSecret gives a site a new identify secret, returned once; the old
// one stops working at once.
func (s *Service) RotateSecret(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TrackerSecret, error) {
	secret := newSecret()
	sealed, err := s.Box.Seal([]byte(secret), secretAD(id))
	if err != nil {
		return TrackerSecret{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.TrackerSite(ctx, store.TrackerSiteParams{ID: id, WorkspaceID: ws.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return siteGone(ctx)
		}
		if err != nil {
			return err
		}
		if _, err := q.TrackerSiteSetSecret(ctx, store.TrackerSiteSetSecretParams{ID: id, WorkspaceID: ws.ID, IdentifySecretSealed: sealed, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tracker_site.secret_rotated", TargetType: "tracker_site", TargetID: id.String(),
			TargetName: before.Name})
	})
	if err != nil {
		return TrackerSecret{}, err
	}
	s.forget(ws.ID)
	return TrackerSecret{IdentifySecret: secret}, nil
}

// RecentEvents are a site's last 20 events of the last day.
func (s *Service) RecentEvents(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TrackerRecentEvents, error) {
	if _, err := s.Q.TrackerSite(ctx, store.TrackerSiteParams{ID: id, WorkspaceID: ws.ID}); errors.Is(err, pgx.ErrNoRows) {
		return TrackerRecentEvents{}, siteGone(ctx)
	} else if err != nil {
		return TrackerRecentEvents{}, err
	}
	rows, err := s.Q.TrackerRecentEvents(ctx, store.TrackerRecentEventsParams{SiteID: id, WorkspaceID: ws.ID, Since: s.now().Add(-24 * time.Hour)})
	if err != nil {
		return TrackerRecentEvents{}, err
	}
	out := TrackerRecentEvents{Items: make([]TrackerRecentEvent, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, TrackerRecentEvent{Kind: kindName(r.Kind), Path: r.Path, ProductTitle: r.ProductTitle, Name: r.Name, Known: r.Known, At: r.OccurredAt})
	}
	return out, nil
}

func kindName(k int16) string {
	switch k {
	case KindProduct:
		return "product"
	case KindCart:
		return "cart"
	case KindCustom:
		return "custom"
	}
	return "page"
}

// forget drops what this process cached of a workspace's sites: a change
// applies to the next event (other processes see it within a minute).
func (s *Service) forget(ws uuid.UUID) {
	prefix := ws.String()
	s.origins.drop(func(k string) bool { return k == prefix })
	s.sites.drop(func(string) bool { return true })
}
