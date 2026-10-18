package commerce

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Links to a connected store (its own domain, or the domain shoppers see)
// get the UTM parameters and the click token at the click redirect, unless
// the store's "Tag links" setting is off. The token is how an order is
// credited to the email that brought the click.

// domainSet is a workspace's stores whose links are tagged, cached for a
// minute: the decorator runs on every click.
type domainSet struct {
	at    time.Time
	hosts map[string]bool
	slug  string
}

// cachedName is a campaign's or an automation's utm_campaign value.
type cachedName struct {
	at   time.Time
	slug string
}

const linkCacheTTL = time.Minute

var _ tracking.LinkDecorator = (*Service)(nil)

// Decorate implements tracking.LinkDecorator.
func (s *Service) Decorate(ctx context.Context, c tracking.ClickedLink) string {
	if c.ContactID == nil || s.Links == nil {
		return c.Target
	}
	u, err := url.Parse(c.Target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c.Target
	}
	d := s.storeHosts(ctx, c.WorkspaceID)
	if !d.hosts[bareHost(u.Hostname())] {
		return c.Target
	}
	at := c.At
	if at.IsZero() {
		at = s.now()
	}
	tok := clicktoken.Sign(s.Links, clicktoken.Click{WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: *c.ContactID, At: at})
	return clicktoken.Tag(c.Target, clicktoken.UTM{Source: d.slug, Medium: "email", Campaign: s.utmCampaign(ctx, c)}, tok)
}

// TagsLink reports whether a link is to one of the workspace's tagged
// stores (the composer's hint).
func (s *Service) TagsLink(ctx context.Context, ws uuid.UUID, link string) bool {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return false
	}
	return s.storeHosts(ctx, ws).hosts[bareHost(u.Hostname())]
}

func bareHost(h string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(h, ".")), "www.")
}

func (s *Service) storeHosts(ctx context.Context, ws uuid.UUID) domainSet {
	now := s.now()
	if v, ok := s.domains.Load(ws); ok {
		if d := v.(domainSet); now.Sub(d.at) < linkCacheTTL {
			return d
		}
	}
	d := domainSet{at: now, hosts: map[string]bool{}}
	rows, err := s.Q.CommerceStoresConnected(ctx, ws)
	if err != nil {
		return d
	}
	for _, st := range rows {
		if !st.TagLinks {
			continue
		}
		for _, h := range []string{st.Domain, st.PublicDomain} {
			if h = bareHost(h); h != "" {
				d.hosts[h] = true
			}
		}
	}
	if len(d.hosts) > 0 {
		if w, err := s.Q.GetWorkspace(ctx, ws); err == nil {
			d.slug = w.Slug
		}
	}
	s.domains.Store(ws, d)
	return d
}

// forget drops a workspace's cached store domains (a store's settings or
// status changed).
func (s *Service) forget(ws uuid.UUID) { s.domains.Delete(ws) }

// utmCampaign is the campaign's slug, or the automation's.
func (s *Service) utmCampaign(ctx context.Context, c tracking.ClickedLink) string {
	var id uuid.UUID
	switch {
	case c.CampaignID != nil:
		id = *c.CampaignID
	case c.AutomationID != nil:
		id = *c.AutomationID
	default:
		return ""
	}
	now := s.now()
	if v, ok := s.names.Load(id); ok {
		if n := v.(cachedName); now.Sub(n.at) < 10*linkCacheTTL {
			return n.slug
		}
	}
	name := ""
	if c.CampaignID != nil {
		name, _ = s.Q.CommerceCampaignName(ctx, id)
	} else if a, err := s.Q.CommerceAutomationName(ctx, id); err == nil {
		name = a.Name
	}
	slug := clicktoken.Slug(name)
	s.names.Store(id, cachedName{at: now, slug: slug})
	return slug
}

// MessageVars is tracking's MessageVars: an automation message's
// checkout_url.
func (s *Service) MessageVars(ctx context.Context, msgCtx map[string]string) map[string]string {
	return s.Vars(ctx, msgCtx)
}
