package push

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The public side of push:
//
//	GET  /push/v1/prompt.js, prompt.css   the soft prompt, for every site
//	GET  /push/v1/sw.js                   the worker, for a site's own worker to import
//	GET  /push/v1/rc-push-sw.js           the file a site hosts at its root (a download)
//	GET  /push/v1/test-sw.js              the app's worker for "Send to my browser"
//	GET  /rc-push-sw.js                   the publication website's built-in worker
//	GET  /p/{ws}/push/config?site=…       a site's live prompts and the public key
//	POST /p/{ws}/push/subscriptions       a browser allowed notifications
//	POST /p/{ws}/push/unsubscribe         a reader turned them off
//	POST /p/{ws}/push/events              the prompt's and the worker's reports
//	GET  /t/p/{id}.{token}[/{button}]     a click, then the notification's link
//
// Cross-origin calls are answered only for the workspace's listed sites.

//go:embed static
var staticFS embed.FS

// static reads an embedded file.
func static(name string) []byte {
	b, err := staticFS.ReadFile("static/" + name)
	if err != nil {
		panic(err)
	}
	return b
}

var (
	workerOnce sync.Once
	workerJS   []byte
)

// worker is the service worker's code.
func worker() []byte {
	workerOnce.Do(func() { workerJS = static("sw.js") })
	return workerJS
}

// HostedWorker is rc-push-sw.js: the worker as a site hosts it.
func HostedWorker() []byte {
	return append([]byte("self.RC_PUSH_OWN = true;\n"), worker()...)
}

// limiter counts requests per key in one-minute windows.
type limiter struct {
	mu     sync.Mutex
	window time.Time
	counts map[string]int
}

func (l *limiter) allow(key string, perMinute int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Truncate(time.Minute)
	if !now.Equal(l.window) {
		l.window, l.counts = now, map[string]int{}
	}
	l.counts[key]++
	return l.counts[key] <= perMinute
}

var publicLimits = &limiter{counts: map[string]int{}}

// clientIP is the request's address (RealIP already ran).
func clientIP(r *http.Request) *netip.Addr {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	a = a.Unmap()
	return &a
}

func ipKey(r *http.Request) string {
	if a := clientIP(r); a != nil {
		return a.String()
	}
	return "unknown"
}

// script writes a script with the headers every site may load it with.
func script(w http.ResponseWriter, r *http.Request, body []byte, cache string) {
	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("Cache-Control", cache)
	h.Set("Cross-Origin-Resource-Policy", "cross-origin")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Del("Content-Security-Policy")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}

// PublicRoutes mounts the scripts, the workers and the click link. The
// workspace endpoints are under /p/{ws}/push (PageRoutes).
func (s *Service) PublicRoutes(r chi.Router, spa http.Handler) {
	r.Get("/push/v1/prompt.js", func(w http.ResponseWriter, r *http.Request) {
		script(w, r, static("prompt.js"), "public, max-age=600")
	})
	r.Get("/push/v1/prompt.css", func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Type", "text/css; charset=utf-8")
		h.Set("Cache-Control", "public, max-age=600")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin")
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(static("prompt.css")))
	})
	r.Get("/push/v1/sw.js", func(w http.ResponseWriter, r *http.Request) {
		script(w, r, worker(), "no-cache")
	})
	r.Get("/push/v1/test-sw.js", func(w http.ResponseWriter, r *http.Request) {
		script(w, r, HostedWorker(), "no-cache")
	})
	r.Get("/push/v1/"+WorkerFile, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="`+WorkerFile+`"`)
		script(w, r, HostedWorker(), "no-cache")
	})
	// The publication website's worker, at its root.
	r.Get("/"+WorkerFile, func(w http.ResponseWriter, r *http.Request) {
		if !s.servesWorker(r.Context(), r.Host) {
			if spa != nil {
				spa.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Service-Worker-Allowed", "/")
		script(w, r, HostedWorker(), "no-cache")
	})
	r.Get("/t/p/{click}", s.serveClick)
	r.Head("/t/p/{click}", s.serveClick)
	r.Get("/t/p/{click}/{button}", s.serveClick)
}

// servesWorker reports whether host is a workspace's publication website
// with push on: its root serves the built-in worker.
func (s *Service) servesWorker(ctx context.Context, host string) bool {
	host = strings.ToLower(host)
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM push_sites s JOIN push_keys k ON k.workspace_id = s.workspace_id AND k.rotated_at IS NULL
		WHERE s.kind = 'publication' AND s.removed_at IS NULL AND s.host = $1`, host).Scan(&n)
	return err == nil && n > 0
}

// serveClick records a click and goes on to the link.
func (s *Service) serveClick(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	id, tok, ok := ParseClick(chi.URLParam(r, "click"))
	if !ok {
		gone(w)
		return
	}
	action := 0
	if b := chi.URLParam(r, "button"); b != "" {
		n, err := strconv.Atoi(b)
		if err != nil || n < 1 || n > 2 {
			gone(w)
			return
		}
		action = n
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	target, err := s.Click(ctx, id, tok, action, r.UserAgent())
	if err != nil || target == "" {
		gone(w)
		return
	}
	// target is one of the notification's own links (from its content or
	// its campaign), an http(s) URL: never a URL taken from the request.
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // G710: see target
}

func gone(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("This notification's link is no longer valid.\n"))
}

// PageRoutes mounts a workspace's push endpoints (under /p/{ws}/push).
func (s *Service) PageRoutes(r chi.Router, workspace func(*http.Request) (*store.Workspace, bool)) {
	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveConfig(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
	for _, p := range []string{"/config", "/subscriptions", "/unsubscribe", "/events"} {
		r.Options(p, func(w http.ResponseWriter, r *http.Request) {
			ws, ok := workspace(r)
			if !ok {
				http.NotFound(w, r)
				return
			}
			if !s.cors(w, r, ws) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
		})
	}
	r.Post("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveSubscribe(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
	r.Post("/unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveUnsubscribe(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
	r.Post("/events", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveEvents(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
	// The account page: the signed-in member's browsers.
	r.Get("/account", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveAccount(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
	r.Post("/account/off", func(w http.ResponseWriter, r *http.Request) {
		if ws, ok := workspace(r); ok {
			s.serveAccountOff(w, r, ws)
			return
		}
		http.NotFound(w, r)
	})
}

// origin is a request's Origin, normalized; "" when it has none.
func requestOrigin(r *http.Request) string {
	o := strings.TrimSpace(r.Header.Get("Origin"))
	if o == "" || o == "null" {
		return ""
	}
	origin, _, ok := ParseOrigin(o)
	if !ok {
		return ""
	}
	return origin
}

// cors allows a listed site to read the answer. It reports whether the
// request may go on: same-origin requests (no Origin, or the instance's
// own) always may.
func (s *Service) cors(w http.ResponseWriter, r *http.Request, ws *store.Workspace) bool {
	o := requestOrigin(r)
	w.Header().Add("Vary", "Origin")
	if o == "" || o == s.origin() || strings.EqualFold(o, "https://"+r.Host) || strings.EqualFold(o, "http://"+r.Host) {
		return true
	}
	if _, err := s.siteOf(r.Context(), ws, o); err != nil {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", o)
	return true
}

// siteOrigin is the site a request comes from: its Origin header, else
// the page it names.
func siteOrigin(r *http.Request, page string) string {
	if o := requestOrigin(r); o != "" {
		return o
	}
	if origin, _, ok := ParseOrigin(page); ok {
		return origin
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// PushPublicPrompt is a live prompt as the prompt script reads it.
type PushPublicPrompt struct {
	ID      string           `json:"id"`
	Version int              `json:"version"`
	Config  PushPromptConfig `json:"config"`
}

// PushPublicConfig is what a site's prompt script reads.
type PushPublicConfig struct {
	Key     string             `json:"key"`
	Site    string             `json:"site"`
	Name    string             `json:"name"`
	Icon    string             `json:"icon,omitempty"`
	Accent  string             `json:"accent,omitempty"`
	Worker  string             `json:"worker"`
	Scope   string             `json:"scope"`
	Prompts []PushPublicPrompt `json:"prompts"`
	// Base is where the script calls (the site itself for the publication
	// website, else the instance).
	Base      string `json:"base"`
	Workspace string `json:"workspace"`
	// Text is ReputeCast's own words in the prompt (the Home Screen steps),
	// in the reader's language.
	Text map[string]string `json:"text"`
}

func (s *Service) serveConfig(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	if !publicLimits.allow("config:"+ipKey(r), 120) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	if !s.cors(w, r, ws) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "refused"})
		return
	}
	ctx := r.Context()
	key, err := s.liveKey(ctx, s.Q, ws)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "off"})
		return
	}
	siteParam := r.URL.Query().Get("site")
	origin, _, ok := ParseOrigin(siteParam)
	if !ok {
		origin = requestOrigin(r)
	}
	site, err := s.siteOf(ctx, ws, origin)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "refused"})
		return
	}
	if site.Kind == KindSnippet {
		if err := s.Q.PushSiteSnippetSeen(ctx, store.PushSiteSnippetSeenParams{ID: site.ID, At: s.now()}); err != nil {
			s.log().WarnContext(ctx, "push snippet seen", "err", err)
		}
	}
	out := PushPublicConfig{Key: key.PublicKey, Site: site.Host, Name: ws.Name, Icon: s.iconURL(ctx, ws, site), Worker: site.WorkerPath,
		Scope: workerScope(site.WorkerPath, ""), Prompts: []PushPublicPrompt{}, Workspace: ws.Slug, Base: s.origin()}
	if site.Kind == KindPublication {
		out.Worker, out.Scope, out.Base = "/"+WorkerFile, "/", site.Origin
	}
	if site.WorkerMode == "import" {
		out.Scope = "/"
	}
	if s.Brand != nil {
		out.Accent, out.Name = s.Brand(ctx, ws)
	}
	loc, _ := i18n.Negotiate(r.Header.Get("Accept-Language"))
	l := i18n.For(loc)
	names := i18n.Args{"site": out.Name}
	out.Text = map[string]string{"iosTitle": msgs.M("ios.title").Text(l, names), "iosShare": msgs.M("ios.share").Text(l, nil),
		"iosAdd": msgs.M("ios.add").Text(l, nil), "iosOpen": msgs.M("ios.open").Text(l, names)}
	prompts, err := s.Q.PushPrompts(ctx, ws.ID)
	if err == nil {
		for _, p := range prompts {
			if p.SiteID == nil || *p.SiteID != site.ID || p.Status != "live" || p.LiveVersion == 0 {
				continue
			}
			if live, _ := s.liveConfig(ctx, p.FormID, p.LiveVersion); live != nil {
				out.Prompts = append(out.Prompts, PushPublicPrompt{ID: p.FormID.String(), Version: int(p.LiveVersion), Config: *live})
			}
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, out)
}

// subscribeBody is what the prompt posts.
type subscribeBody struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	TimeZone string `json:"timeZone"`
	Page     string `json:"page"`
	Prompt   string `json:"prompt"`
	Version  int    `json:"version"`
	Token    string `json:"token"`
	// From is "account" or "preferences" when the reader turned
	// notifications on there.
	From string `json:"from"`
}

func readBody(r *http.Request, v any, limit int64) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(b)) > limit {
		return errors.New("too large")
	}
	return json.Unmarshal(b, v)
}

func (s *Service) serveSubscribe(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	if !publicLimits.allow("subscribe:"+ipKey(r), 20) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"status": "slow_down"})
		return
	}
	if !s.cors(w, r, ws) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "refused"})
		return
	}
	var b subscribeBody
	if err := readBody(r, &b, 8<<10); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	ctx := r.Context()
	form, _ := uuid.Parse(b.Prompt)
	req := SubscribeRequest{Origin: siteOrigin(r, b.Page), Endpoint: b.Endpoint, P256dh: b.Keys.P256dh, Auth: b.Keys.Auth, TimeZone: b.TimeZone, Page: b.Page,
		FormID: form, Version: b.Version, Token: b.Token, IP: clientIP(r), UserAgent: r.UserAgent()}
	if s.Member != nil {
		req.Member = s.Member(ctx, ws, r)
		if req.Member != uuid.Nil && (b.From == "account" || b.From == "preferences") {
			req.LinkedBy = b.From
		}
	}
	res, err := s.Subscribe(ctx, ws, req)
	switch {
	case errors.Is(err, ErrPushOff):
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "off"})
	case errors.Is(err, ErrSiteRefused):
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "refused"})
	case errors.Is(err, ErrBadEndpoint), errors.Is(err, ErrSubscription):
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
	case err != nil:
		s.log().ErrorContext(ctx, "push subscribe", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "subscribed", "id": res.ID.String(), "linked": res.Linked})
	}
}

func (s *Service) serveUnsubscribe(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	if !publicLimits.allow("unsubscribe:"+ipKey(r), 30) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"status": "slow_down"})
		return
	}
	if !s.cors(w, r, ws) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "refused"})
		return
	}
	var b struct {
		Endpoint string `json:"endpoint"`
		Reason   string `json:"reason"`
	}
	if err := readBody(r, &b, 4<<10); err != nil || b.Endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	ok, err := s.Unsubscribe(r.Context(), ws, b.Endpoint, b.Reason)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}
	status := "unsubscribed"
	if !ok {
		status = "unknown"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

// eventBody is a report: a prompt's event, or a notification's.
type eventBody struct {
	Kind  string `json:"kind"`
	Event string `json:"event"`
	// Prompt events.
	Prompt string `json:"prompt"`
	// Notification events.
	ID    string `json:"id"`
	Token string `json:"t"`
}

func (s *Service) serveEvents(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	if !publicLimits.allow("events:"+ipKey(r), 240) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	// Reports come from the worker with mode "no-cors" and from the prompt:
	// the answer is never read, so any origin may post one; the token or a
	// listed site's prompt makes it count.
	s.cors(w, r, ws)
	var b eventBody
	if err := readBody(r, &b, 2<<10); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	switch b.Kind {
	case "notification":
		id, err := uuid.Parse(b.ID)
		if err != nil || (b.Event != "shown" && b.Event != "closed") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.Report(ctx, ws, id, b.Token, b.Event); err != nil && !errors.Is(err, ErrUnknownNotification) && !errors.Is(err, pgx.ErrNoRows) {
			s.log().WarnContext(ctx, "push report", "err", err)
		}
	case "prompt":
		form, err := uuid.Parse(b.Prompt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p, err := s.Q.PushPrompt(ctx, store.PushPromptParams{FormID: form, WorkspaceID: ws.ID})
		if err != nil || p.SiteID == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Only the prompt's own site counts.
		if o := requestOrigin(r); o != "" {
			if st, err := s.Q.PushSiteAny(ctx, *p.SiteID); err != nil || !strings.EqualFold(st.Origin, o) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		if err := s.RecordPromptEvent(ctx, ws, form, b.Event); err != nil {
			s.log().WarnContext(ctx, "push prompt event", "err", err)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
