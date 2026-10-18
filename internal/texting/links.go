package texting

import (
	"context"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Short links: every link in a text is shortened on the workspace's own
// short-link domain (go.acme.io/k3P9xq) with a code per recipient, and
// redirects with UTM parameters for 12 months. Messaging apps fetch links
// to draw a preview: those are machine clicks, never counted as people.

var codeRe = regexp.MustCompile(`^[A-Za-z0-9]{4,12}$`)

// previewAgents are the user agents of link previews.
var previewAgents = []string{"facebookexternalhit", "facebot", "twitterbot", "whatsapp", "slackbot", "telegrambot", "discordbot", "linkedinbot", "skypeuripreview",
	"applebot", "googlebot", "bingbot", "bingpreview", "iframely", "embedly", "pinterest", "redditbot", "vkshare", "snapchat", "viber", "signal", "line-poker",
	"kakaotalk-scrap", "yandex", "duckduckbot", "preview", "crawler", "spider", "bot/", "headless", "python-requests", "curl/", "go-http-client", "okhttp"}

// machineClick reports whether a request is a link preview or another
// automated fetch.
func machineClick(r *http.Request) bool {
	if r.Method == http.MethodHead {
		return true
	}
	ua := strings.ToLower(r.UserAgent())
	if ua == "" {
		return true
	}
	for _, a := range previewAgents {
		if strings.Contains(ua, a) {
			return true
		}
	}
	// iMessage fetches previews as Facebook's crawler; iOS link previews
	// also send this purpose header.
	return strings.EqualFold(r.Header.Get("Purpose"), "preview") || strings.EqualFold(r.Header.Get("X-Purpose"), "preview")
}

// LinkRoutes mounts the short links: the root of the verified short-link
// domain, and /t/s/{code} on the instance's own host. Requests for other
// hosts at the root fall through to spa.
func (s *Service) LinkRoutes(r chi.Router, spa http.Handler) {
	r.Get("/t/s/{code}", s.serveLink)
	r.Head("/t/s/{code}", s.serveLink)
	root := func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if _, err := s.Q.SmsShortDomainWorkspace(r.Context(), host); err != nil {
			if spa != nil {
				spa.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		s.serveLink(w, r)
	}
	r.Get("/{code:[A-Za-z0-9]{6}}", root)
	r.Head("/{code:[A-Za-z0-9]{6}}", root)
}

func (s *Service) serveLink(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if !codeRe.MatchString(code) {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	l, err := s.Q.SmsLink(ctx, code)
	if err != nil || !l.ExpiresAt.After(s.now()) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("This link has expired. Short links in texts work for 12 months.\n"))
		return
	}
	s.recordClick(ctx, l, r)
	http.Redirect(w, r, l.Target, http.StatusFound)
}

// recordClick logs a click: people apart from link previews.
func (s *Service) recordClick(ctx context.Context, l store.SmsLink, r *http.Request) {
	machine := machineClick(r)
	now := s.now()
	ua := r.UserAgent()
	if len(ua) > 300 {
		ua = ua[:300]
	}
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.SmsLinkClickInsert(ctx, store.SmsLinkClickInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: l.WorkspaceID, Code: l.Code, CampaignID: l.CampaignID,
			MessageID: l.MessageID, ContactID: l.ContactID, Machine: machine, UserAgent: ua, OccurredAt: now}); err != nil {
			return err
		}
		if l.MessageID == nil {
			return nil
		}
		if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: l.WorkspaceID, MessageID: *l.MessageID,
			Kind: "clicked", Detail: jsonOf(map[string]any{"code": l.Code, "url": l.Original}), Machine: machine, OccurredAt: now}); err != nil {
			return err
		}
		ws, err := q.GetWorkspace(ctx, l.WorkspaceID)
		if err != nil {
			return err
		}
		m, err := q.ChanMessage(ctx, *l.MessageID)
		if err != nil {
			return nil
		}
		_, err = webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "sms.clicked", s.hookData(m, map[string]any{"url": l.Original, "machine": machine, "clicked_at": now}))
		return err
	})
	if err != nil {
		s.log().WarnContext(ctx, "sms click", "code", l.Code, "err", err)
	}
}
