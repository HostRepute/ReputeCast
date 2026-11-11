package tracker

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/useragent"
)

//go:embed static/rc.js
var staticFS embed.FS

var (
	scriptOnce sync.Once
	scriptJS   []byte
)

// Script is rc.js.
func Script() []byte {
	scriptOnce.Do(func() {
		b, err := staticFS.ReadFile("static/rc.js")
		if err != nil {
			panic(err)
		}
		scriptJS = b
	})
	return scriptJS
}

// Limits of the collect endpoint.
const (
	// MaxBody is the largest request body.
	MaxBody = 8 << 10
	// MaxEvents is the most events one request carries.
	MaxEvents = 20
	// perMinute is the calls one address may make to one site a minute
	// (the address is only ever counted in memory).
	perMinute = 240
	// maxPath and maxProps bound what an event keeps.
	maxPath  = 512
	maxProps = 2 << 10
	// skew is how far back an event's time may be.
	skew = 10 * time.Minute
	// siteFor is how long a site's settings are cached.
	siteFor = 30 * time.Second
)

// Routes mounts the script and the collect endpoint.
func (s *Service) Routes(r chi.Router) {
	r.Get("/track/v1/rc.js", serveScript)
	r.Head("/track/v1/rc.js", serveScript)
	r.Post("/p/{ws}/track/{site}/e", s.serveCollect)
	r.Options("/p/{ws}/track/{site}/e", s.servePreflight)
}

// serveScript writes rc.js with the headers every site may load it with.
func serveScript(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("Cache-Control", "public, max-age=600")
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Del("Content-Security-Policy")
	http.ServeContent(w, r, "rc.js", time.Time{}, bytes.NewReader(Script()))
}

// collectSite is a site as the collect endpoint reads it.
type collectSite struct {
	ID, Workspace uuid.UUID
	Slug, Origin  string
	Store         *uuid.UUID
	Wait          bool
	sealed        []byte
	missing       bool
}

// site reads (and caches) the site a call names; ok is false for a site
// that doesn't exist, was deleted, or whose workspace the request's host
// doesn't carry.
func (s *Service) site(r *http.Request) (collectSite, bool, error) {
	slug, raw := chi.URLParam(r, "ws"), chi.URLParam(r, "site")
	id, err := uuid.Parse(raw)
	if err != nil {
		return collectSite{}, false, nil
	}
	if s.Carries != nil && !s.Carries(r, slug) {
		return collectSite{}, false, nil
	}
	key := slug + "/" + id.String()
	now := time.Now()
	if c, ok := s.sites.get(key, now); ok {
		return c, !c.missing, nil
	}
	row, err := s.Q.TrackerCollectSite(r.Context(), store.TrackerCollectSiteParams{ID: id, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) {
		s.sites.put(key, collectSite{missing: true}, now.Add(siteFor))
		return collectSite{}, false, nil
	}
	if err != nil {
		return collectSite{}, false, err
	}
	c := collectSite{ID: row.ID, Workspace: row.WorkspaceID, Slug: row.Slug, Origin: row.Origin, Store: row.StoreID, Wait: row.WaitConsent, sealed: row.IdentifySecretSealed}
	s.sites.put(key, c, now.Add(siteFor))
	return c, true, nil
}

// requestOrigin is a request's Origin header, normalized: "" without one,
// "null" from a sandboxed frame (a Shopify custom pixel).
func requestOrigin(r *http.Request) string {
	o := strings.TrimSpace(r.Header.Get("Origin"))
	if o == "" || o == "null" {
		return o
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return clip(strings.ToLower(o), 255)
	}
	return originOf(u)
}

// originOf is a URL's origin: scheme and host, with the port when it isn't
// the scheme's.
func originOf(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if p := u.Port(); p != "" && !(scheme == "https" && p == "443") && !(scheme == "http" && p == "80") {
		host += ":" + p
	}
	return scheme + "://" + host
}

func clip(v string, n int) string {
	if len(v) <= n {
		return v
	}
	v = v[:n]
	for len(v) > 0 && !validTail(v) {
		v = v[:len(v)-1]
	}
	return v
}

// validTail reports whether v doesn't end inside a UTF-8 sequence.
func validTail(v string) bool {
	last := v[len(v)-1]
	if last < 0x80 {
		return true
	}
	// Walk back to the start of the last sequence.
	i := len(v) - 1
	for i > 0 && v[i]&0xC0 == 0x80 {
		i--
	}
	size := 1
	switch b := v[i]; {
	case b&0xE0 == 0xC0:
		size = 2
	case b&0xF0 == 0xE0:
		size = 3
	case b&0xF8 == 0xF0:
		size = 4
	}
	return len(v)-i == size
}

// cors answers whether a call from the request's origin may go on, and
// lets the site's own origin read the answer. Other origins are refused
// and counted (the site's page shows where its snippet was copied to).
// Calls without an Origin, or from a sandboxed frame ("null"), go on only
// for events on the site's own pages (checked per event).
func (s *Service) cors(w http.ResponseWriter, r *http.Request, c collectSite) bool {
	w.Header().Add("Vary", "Origin")
	o := requestOrigin(r)
	switch {
	case o == "" || o == "null":
		return true
	case o == c.Origin:
		w.Header().Set("Access-Control-Allow-Origin", o)
		return true
	}
	s.refused.add(c.ID, o, time.Now())
	return false
}

func (s *Service) servePreflight(w http.ResponseWriter, r *http.Request) {
	c, ok, err := s.site(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !s.cors(w, r, c) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	h := w.Header()
	h.Set("Access-Control-Allow-Methods", "POST")
	h.Set("Access-Control-Allow-Headers", "Content-Type")
	h.Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

// collectBody is what the script posts (text/plain JSON).
type collectBody struct {
	Vid string `json:"vid"`
	// C is 1 once the visitor consented (sites that wait).
	C        int            `json:"c"`
	RC       string         `json:"rc"`
	Link     string         `json:"link"`
	Identify *identifyBody  `json:"identify"`
	Events   []collectEvent `json:"events"`
}

type identifyBody struct {
	Email     string `json:"email"`
	Signature string `json:"signature"`
}

type collectEvent struct {
	T     string          `json:"t"`
	At    float64         `json:"at"`
	URL   string          `json:"url"`
	Ref   string          `json:"ref"`
	ID    json.RawMessage `json:"id"`
	Qty   float64         `json:"qty"`
	Name  string          `json:"name"`
	Props json.RawMessage `json:"props"`
}

var (
	vidRe  = regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)
	nameRe = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
	idRe   = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,64}$`)
)

// botWords are internal/website's countable words a browser's user agent
// never has, beyond what internal/useragent (the open and click rules)
// already recognizes.
var botWords = []string{"crawl", "spider", "slurp", "preview", "monitor", "uptime", "headless", "lighthouse", "scanner", "python-requests", "curl", "wget"}

// bot reports a call that isn't a person's browser: no user agent, a
// prefetch, or a machine's user agent.
func bot(r *http.Request) bool {
	if p := r.Header.Get("Sec-Purpose"); strings.Contains(p, "prefetch") || r.Header.Get("Purpose") == "prefetch" {
		return true
	}
	ua := strings.TrimSpace(r.UserAgent())
	if ua == "" || useragent.Bot(ua) != "" {
		return true
	}
	low := strings.ToLower(ua)
	for _, w := range botWords {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

func (s *Service) serveCollect(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	c, ok, err := s.site(r)
	if err != nil {
		s.log().ErrorContext(r.Context(), "tracker site", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !s.cors(w, r, c) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	key := c.ID.String()
	if a := server.ClientIP(r); a != nil {
		key += "/" + a.String()
	}
	if !s.limits.allow(key, perMinute) {
		h.Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(body) > MaxBody {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	// Bots, browsers sending Global Privacy Control, and waiting sites'
	// visitors who haven't consented: nothing is kept.
	if bot(r) || r.Header.Get("Sec-GPC") == "1" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var b collectBody
	if err := json.Unmarshal(body, &b); err != nil || !vidRe.MatchString(b.Vid) || len(b.Events) > MaxEvents {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if c.Wait && b.C != 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	now := s.now()
	got := &collected{site: c, hash: vidHash(c.Workspace, b.Vid), at: now, items: make([]item, 0, len(b.Events))}
	for _, ev := range b.Events {
		if it, ok := eventOf(ev, c.Origin, now); ok {
			got.items = append(got.items, it)
		}
	}
	got.claim = s.claimOf(r.Context(), c, &b, now)
	if len(got.items) > 0 || got.claim != nil {
		s.enqueue(got)
	}
	w.WriteHeader(http.StatusNoContent)
}

// eventOf checks one event: a known type, on the site's own pages, with
// its fields within bounds. The page keeps its path and utm_* parameters
// only, the referrer its domain only.
func eventOf(ev collectEvent, origin string, now time.Time) (item, bool) {
	it := item{at: eventTime(ev.At, now)}
	switch ev.T {
	case "page":
		it.kind = KindPage
	case "product":
		it.kind = KindProduct
	case "cart":
		it.kind = KindCart
	case "track":
		it.kind = KindCustom
	case "notify":
		it.notify = true
	default:
		return item{}, false
	}
	u, err := url.Parse(strings.TrimSpace(ev.URL))
	if err != nil || len(ev.URL) > 4096 || u.Host == "" || originOf(u) != origin {
		return item{}, false
	}
	it.path = pagePath(u)
	if it.kind == KindPage {
		it.ref = referrerDomain(ev.Ref, u.Hostname())
	}
	switch {
	case it.kind == KindProduct, it.kind == KindCart, it.notify:
		id, ok := productID(ev.ID)
		if !ok {
			return item{}, false
		}
		it.extID = id
		if it.kind == KindCart {
			it.qty = 1
			if q := ev.Qty; q >= 1 && q <= 1000 {
				it.qty = int32(q)
			}
		}
	case it.kind == KindCustom:
		if !nameRe.MatchString(ev.Name) {
			return item{}, false
		}
		it.name = ev.Name
		if p := bytes.TrimSpace(ev.Props); len(p) > 0 && !bytes.Equal(p, []byte("null")) {
			if len(p) > maxProps || p[0] != '{' || !json.Valid(p) {
				return item{}, false
			}
			it.props = p
		}
	}
	return it, true
}

// eventTime is an event's time (milliseconds since 1970), kept within the
// last ten minutes: the server's clock decides.
func eventTime(ms float64, now time.Time) time.Time {
	if ms <= 0 || math.IsNaN(ms) || math.IsInf(ms, 0) {
		return now
	}
	t := time.UnixMilli(int64(ms))
	switch {
	case t.After(now):
		return now
	case t.Before(now.Add(-skew)):
		return now.Add(-skew)
	}
	return t
}

// pagePath is a page's path with only its utm_* parameters.
func pagePath(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	var kept []string
	for _, kv := range strings.Split(u.RawQuery, "&") {
		k, _, _ := strings.Cut(kv, "=")
		if k, err := url.QueryUnescape(k); err == nil && strings.HasPrefix(strings.ToLower(k), "utm_") {
			kept = append(kept, kv)
		}
	}
	if len(kept) > 0 && len(p)+1+len(strings.Join(kept, "&")) <= maxPath {
		p += "?" + strings.Join(kept, "&")
	}
	return clip(p, maxPath)
}

// referrerDomain is where a page view came from: the referring page's
// domain, none for the site's own pages.
func referrerDomain(ref, own string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || len(ref) > 4096 {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == strings.ToLower(own) {
		return ""
	}
	host = strings.TrimPrefix(host, "www.")
	if len(host) > 255 {
		return ""
	}
	return host
}

// productID is a store's product id as a page names it: a string or a
// number; Shopify's global ids ("gid://shopify/Product/123") keep their
// number, as the store sync does.
func productID(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", false
	}
	var id string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &id); err != nil {
			return "", false
		}
	} else {
		n, err := strconv.ParseUint(string(raw), 10, 64)
		if err != nil {
			return "", false
		}
		id = strconv.FormatUint(n, 10)
	}
	id = strings.TrimSpace(id)
	if rest, ok := strings.CutPrefix(id, "gid://shopify/"); ok {
		if i := strings.LastIndexByte(rest, '/'); i >= 0 {
			id = rest[i+1:]
		}
	}
	return id, idRe.MatchString(id)
}

// limiter counts calls per key in one-minute windows (in memory).
type limiter struct {
	mu     sync.Mutex
	window time.Time
	counts map[string]int
}

func (l *limiter) allow(key string, perMinute int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Truncate(time.Minute)
	if !now.Equal(l.window) || l.counts == nil {
		l.window, l.counts = now, map[string]int{}
	}
	l.counts[key]++
	return l.counts[key] <= perMinute
}

// refusals counts calls refused per site and origin, in memory, until the
// writer stores them.
type refusals struct {
	mu sync.Mutex
	m  map[refusalKey]*refusalCount
}

type refusalKey struct {
	site   uuid.UUID
	origin string
}

type refusalCount struct {
	n           int64
	first, last time.Time
}

// refusalsMax bounds the origins counted between two flushes.
const refusalsMax = 10_000

func (f *refusals) add(site uuid.UUID, origin string, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.m == nil {
		f.m = map[refusalKey]*refusalCount{}
	}
	k := refusalKey{site: site, origin: clip(origin, 255)}
	c := f.m[k]
	if c == nil {
		if len(f.m) >= refusalsMax {
			return
		}
		c = &refusalCount{first: at}
		f.m[k] = c
	}
	c.n++
	c.last = at
}

func (f *refusals) take() map[refusalKey]*refusalCount {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.m
	f.m = nil
	return m
}

// flushRefusals stores the refused calls counted since the last flush.
func (s *Service) flushRefusals(ctx context.Context) {
	m := s.refused.take()
	if len(m) == 0 {
		return
	}
	p := store.TrackerRefusalsAddParams{}
	for k, c := range m {
		p.SiteIds = append(p.SiteIds, k.site)
		p.Origins = append(p.Origins, k.origin)
		p.Counts = append(p.Counts, c.n)
		p.FirstAt = append(p.FirstAt, c.first)
		p.LastAt = append(p.LastAt, c.last)
	}
	if err := s.Q.TrackerRefusalsAdd(ctx, p); err != nil {
		s.log().WarnContext(ctx, "tracker refusals not counted", "origins", len(m), "err", err)
	}
}
