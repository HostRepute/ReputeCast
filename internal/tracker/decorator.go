package tracker

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Links to a tracked site get the click token rc= at the click redirect
// (after the stores' own tagging, which already adds it on a store's
// links): the tracker on the landing page sends it once, and the visitor
// is the contact the email went to.

var _ tracking.LinkDecorator = (*Service)(nil)

// originsFor is how long a workspace's site hosts are cached: the
// decorator runs on every click.
const originsFor = time.Minute

// Decorate implements tracking.LinkDecorator.
func (s *Service) Decorate(ctx context.Context, c tracking.ClickedLink) string {
	if c.ContactID == nil || s.Links == nil {
		return c.Target
	}
	u, err := url.Parse(c.Target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c.Target
	}
	if u.Query().Has(clicktoken.Param) || !s.tracksHost(ctx, c.WorkspaceID, hostOf(u)) {
		return c.Target
	}
	at := c.At
	if at.IsZero() {
		at = s.now()
	}
	return withParam(c.Target, clicktoken.Param, clicktoken.Sign(s.Links, clicktoken.Click{WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: *c.ContactID, At: at}))
}

// hostOf is a URL's host as an origin names it: lower case, the port only
// when it isn't the scheme's.
func hostOf(u *url.URL) string {
	o := originOf(u)
	_, h, _ := strings.Cut(o, "://")
	return h
}

// tracksHost reports whether host is one of the workspace's live sites'.
func (s *Service) tracksHost(ctx context.Context, ws uuid.UUID, host string) bool {
	now := time.Now()
	hosts, ok := s.origins.get(ws.String(), now)
	if !ok {
		origins, err := s.Q.TrackerSiteOrigins(ctx, ws)
		if err != nil {
			return false
		}
		hosts = make([]string, 0, len(origins))
		for _, o := range origins {
			if _, h, ok := strings.Cut(o, "://"); ok {
				hosts = append(hosts, h)
			}
		}
		s.origins.put(ws.String(), hosts, now.Add(originsFor))
	}
	for _, h := range hosts {
		if h == host {
			return true
		}
	}
	return false
}

// withParam adds k=v to a link's query, before its fragment, leaving the
// rest as written.
func withParam(link, k, v string) string {
	base, fragment, hasFragment := strings.Cut(link, "#")
	sep := "?"
	if _, q, ok := strings.Cut(base, "?"); ok {
		sep = "&"
		if q == "" || strings.HasSuffix(q, "&") {
			sep = ""
		}
	}
	out := base + sep + url.QueryEscape(k) + "=" + url.QueryEscape(v)
	if hasFragment {
		out += "#" + fragment
	}
	return out
}
