package customhost

import (
	"context"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Hosts says whose host a request came to, for what answers on every host
// that points at the instance: link tracking (/t/o, /t/c, /t/h), short
// links (/t/s), push clicks (/t/p) and a workspace's pages under
// /p/{workspace}. On a host a workspace holds (its verified custom domain,
// short-link domain or link tracking domain, or its publication website)
// only that workspace's links and pages answer: another workspace's would
// look as if they came from the host's owner, and could send people
// anywhere under its name. Any other host, the instance's own or one that
// nobody holds now, answers for every workspace, so links in mail already
// sent keep working: a workspace's links only ever name its own hosts and
// the instance's, and a host it stops using holds nobody else's.
//
// A nil Hosts answers for every workspace on every host.
type Hosts struct {
	Q *store.Queries
	// Instance returns the instance's public URL, whose host is nobody's.
	Instance func() string
	Now      func() time.Time

	mu    sync.Mutex
	known map[string]holders
}

// Holder is a workspace that holds a host.
type Holder struct {
	ID   uuid.UUID
	Slug string
}

type holders struct {
	list    []Holder
	expires time.Time
}

const (
	// holdersFor is how long an answer is kept: a host a workspace
	// verifies or lets go answers as such within a minute, as the custom
	// domain's root pages do.
	holdersFor = time.Minute
	// maxKnown bounds the hosts kept; past it the answers start over.
	maxKnown = 10_000
)

// RequestHost is the host of a request's Host header: lower case, without
// its port or a final dot.
func RequestHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, ".")
}

// Holders returns the workspaces that hold host (a Host header): none for
// the instance's own host and for a host nobody holds.
func (h *Hosts) Holders(ctx context.Context, host string) ([]Holder, error) {
	if h == nil {
		return nil, nil
	}
	host = RequestHost(host)
	if host == "" || host == h.instanceHost() {
		return nil, nil
	}
	now := h.now()
	h.mu.Lock()
	if e, ok := h.known[host]; ok && now.Before(e.expires) {
		h.mu.Unlock()
		return e.list, nil
	}
	h.mu.Unlock()
	rows, err := h.Q.CustomHostHolders(ctx, host)
	if err != nil {
		return nil, err
	}
	list := make([]Holder, len(rows))
	for i, r := range rows {
		list[i] = Holder{ID: r.WorkspaceID, Slug: r.Slug}
	}
	h.mu.Lock()
	if h.known == nil || len(h.known) >= maxKnown {
		h.known = map[string]holders{}
	}
	h.known[host] = holders{list: list, expires: now.Add(holdersFor)}
	h.mu.Unlock()
	return list, nil
}

// Carries reports whether host (a Host header) may carry workspace's links
// and pages: it is one of workspace's hosts, or nobody's.
func (h *Hosts) Carries(ctx context.Context, host string, workspace uuid.UUID) (bool, error) {
	return h.carries(ctx, host, func(x Holder) bool { return x.ID == workspace })
}

// CarriesSlug is Carries for the workspace a path names by its slug.
func (h *Hosts) CarriesSlug(ctx context.Context, host, slug string) (bool, error) {
	return h.carries(ctx, host, func(x Holder) bool { return strings.EqualFold(x.Slug, slug) })
}

func (h *Hosts) carries(ctx context.Context, host string, match func(Holder) bool) (bool, error) {
	list, err := h.Holders(ctx, host)
	if err != nil {
		return false, err
	}
	return len(list) == 0 || slices.ContainsFunc(list, match), nil
}

func (h *Hosts) instanceHost() string {
	if h.Instance == nil {
		return ""
	}
	u, err := url.Parse(h.Instance())
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
}

func (h *Hosts) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}
