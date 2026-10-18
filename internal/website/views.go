package website

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
)

// Page kinds whose views are counted.
const (
	PagePost    = "post"
	PageHome    = "home"
	PageArchive = "archive"
	PageTag     = "tag"
	PagePage    = "page"
)

// HomeID stands for the home page and the archive in the view counts.
var HomeID = uuid.Nil

// Views are counted on the server, without cookies: one view per visitor,
// page and day. A visitor is the hash of their address and browser with a
// salt that changes every day; the hashes and the salt are deleted once
// the day is over, so nothing links a visitor across days and nothing
// that identifies them is kept.

// salts caches today's salt per process.
type saltCache struct {
	mu   sync.Mutex
	day  time.Time
	salt []byte
}

var salts saltCache

func (s *Service) salt(ctx context.Context, d time.Time) ([]byte, error) {
	salts.mu.Lock()
	defer salts.mu.Unlock()
	if salts.day.Equal(d) && salts.salt != nil {
		return salts.salt, nil
	}
	fresh := make([]byte, 32)
	_, _ = rand.Read(fresh)
	// The first process to need the day's salt makes it; the others read it.
	salt, err := s.Q.WebsiteSaltInsert(ctx, store.WebsiteSaltInsertParams{Day: d, Salt: fresh})
	if err != nil {
		return nil, err
	}
	salts.day, salts.salt = d, salt
	return salt, nil
}

// botWords mark user agents that aren't readers (crawlers, link previews,
// uptime checks): they get the page but aren't counted.
var botWords = []string{"bot", "crawl", "spider", "slurp", "preview", "monitor", "uptime", "fetch", "curl", "wget", "python-requests", "headless", "lighthouse", "scanner"}

// countable reports whether a request is a reader's view.
func countable(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if p := r.Header.Get("Sec-Purpose"); strings.Contains(p, "prefetch") || r.Header.Get("Purpose") == "prefetch" {
		return false
	}
	ua := strings.ToLower(r.UserAgent())
	if ua == "" {
		return false
	}
	for _, w := range botWords {
		if strings.Contains(ua, w) {
			return false
		}
	}
	return true
}

// CountView counts a page view. It never fails a page: errors are logged.
func (s *Service) CountView(ctx context.Context, ws *store.Workspace, r *http.Request, kind string, page uuid.UUID) {
	if r == nil || !countable(r) {
		return
	}
	if err := s.countView(ctx, ws, r, kind, page); err != nil {
		s.log().WarnContext(ctx, "count website view", "err", err)
	}
}

func (s *Service) countView(ctx context.Context, ws *store.Workspace, r *http.Request, kind string, page uuid.UUID) error {
	now := s.now()
	d := day(now, location(ws))
	salt, err := s.salt(ctx, d)
	if err != nil {
		return err
	}
	ip := ""
	if a := server.ClientIP(r); a != nil {
		ip = a.String()
	}
	visitor := func(parts ...string) []byte {
		h := sha256.New()
		h.Write(salt)
		h.Write([]byte(ws.ID.String()))
		h.Write([]byte{0})
		h.Write([]byte(ip))
		h.Write([]byte{0})
		h.Write([]byte(r.UserAgent()))
		for _, p := range parts {
			h.Write([]byte{0})
			h.Write([]byte(p))
		}
		return h.Sum(nil)
	}
	newView, err := s.Q.WebsiteSeen(ctx, store.WebsiteSeenParams{Day: d, Hash: visitor("view", kind, page.String())})
	if err != nil || !newView {
		return err
	}
	if err := s.Q.WebsiteViewAdd(ctx, store.WebsiteViewAddParams{WorkspaceID: ws.ID, Day: d, PageKind: kind, PageID: page, Views: 1}); err != nil {
		return err
	}
	newVisitor, err := s.Q.WebsiteSeen(ctx, store.WebsiteSeenParams{Day: d, Hash: visitor("visitor")})
	if err != nil {
		return err
	}
	if newVisitor {
		if err := s.Q.WebsiteVisitorAdd(ctx, store.WebsiteVisitorAddParams{WorkspaceID: ws.ID, Day: d, Visitors: 1}); err != nil {
			return err
		}
	}
	if kind != PagePost {
		return nil
	}
	ref := referrerDomain(r)
	if err := s.Q.WebsiteViewSourceAdd(ctx, store.WebsiteViewSourceAddParams{WorkspaceID: ws.ID, PostID: page, Day: d, Dim: "referrer", Key: ref, Views: 1}); err != nil {
		return err
	}
	if src := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("utm_source"))); src != "" && len(src) <= 100 {
		if err := s.Q.WebsiteViewSourceAdd(ctx, store.WebsiteViewSourceAddParams{WorkspaceID: ws.ID, PostID: page, Day: d, Dim: "utm", Key: src, Views: 1}); err != nil {
			return err
		}
	}
	return nil
}

// referrerDomain is where a view came from: the referring page's domain
// ("" for email clients and direct visits, which send none).
func referrerDomain(r *http.Request) string {
	ref := r.Referer()
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if len(host) > 100 {
		return ""
	}
	return host
}

// PurgeViews deletes the hashes and salts of the days before today: run
// by the nightly job.
func (s *Service) PurgeViews(ctx context.Context) error {
	cut := s.now().UTC().AddDate(0, 0, -1)
	d := time.Date(cut.Year(), cut.Month(), cut.Day(), 0, 0, 0, 0, time.UTC)
	if err := s.Q.WebsiteViewsPurge(ctx, d); err != nil {
		return err
	}
	return s.Q.WebsiteSaltsPurge(ctx, d)
}
