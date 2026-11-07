package texting

import (
	"context"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
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
// domain, and /t/s/{code} on every host. Requests for other hosts at the
// root fall through to spa.
func (s *Service) LinkRoutes(r chi.Router, spa http.Handler) {
	r.Get("/t/s/{code}", s.serveLink)
	r.Head("/t/s/{code}", s.serveLink)
	root := func(w http.ResponseWriter, r *http.Request) {
		holders, err := s.Q.SmsShortDomainHolders(r.Context(), requestHost(r))
		if err != nil || len(holders) == 0 {
			if spa != nil {
				spa.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		s.redirect(w, r, holders)
	}
	r.Get("/{code:[A-Za-z0-9]{6}}", root)
	r.Head("/{code:[A-Za-z0-9]{6}}", root)
}

// serveLink answers /t/s/{code}. That path answers on every host, so on a
// host a workspace holds (its short-link domain, and its pages, website or
// tracking domain too) only that workspace's codes answer, as at the root
// of a short-link domain.
func (s *Service) serveLink(w http.ResponseWriter, r *http.Request) {
	holders, err := s.linkHolders(r)
	if err != nil {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	s.redirect(w, r, holders)
}

// linkHolders are the workspaces whose codes /t/s/ answers for on the
// request's host: none (every workspace's) on the instance's own host and
// on a host nobody holds.
func (s *Service) linkHolders(r *http.Request) ([]uuid.UUID, error) {
	if s.Hosts == nil {
		return s.Q.SmsShortDomainHolders(r.Context(), requestHost(r))
	}
	list, err := s.Hosts.Holders(r.Context(), r.Host)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(list))
	for i, h := range list {
		ids[i] = h.ID
	}
	return ids, nil
}

// redirect sends the visitor of a short link on. On a host a workspace
// holds, holders are the workspaces whose host it is, and only their codes
// answer: a workspace's domain carries its own links, never another
// workspace's (which could otherwise send people anywhere under its name).
// Elsewhere (no holders) every code answers.
func (s *Service) redirect(w http.ResponseWriter, r *http.Request, holders []uuid.UUID) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if !codeRe.MatchString(code) {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	l, err := s.Q.SmsLink(ctx, code)
	if err != nil || !l.ExpiresAt.After(s.now()) || (len(holders) > 0 && !slices.Contains(holders, l.WorkspaceID)) {
		expired(w, r)
		return
	}
	s.recordClick(ctx, l, r)
	http.Redirect(w, r, l.Target, http.StatusFound)
}

// requestHost is the host a request came to, without its port.
func requestHost(r *http.Request) string {
	host := strings.ToLower(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host
}

// expired answers a short link that no longer works, in the language the
// visitor's browser asks for.
func expired(w http.ResponseWriter, r *http.Request) {
	lang, _ := i18n.Negotiate(r.Header.Get("Accept-Language"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(mLinkExpired.Text(i18n.For(lang), i18n.Args{"months": 12}) + "\n")) //nolint:gosec // G705: a catalog sentence, picked by the browser's language, served as text/plain
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
		// The text may be gone (past retention): the click still counts.
		if m, err := q.ChanMessage(ctx, *l.MessageID); err == nil {
			_, err = webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "sms.clicked", s.hookData(m, map[string]any{"url": l.Original, "machine": machine, "clicked_at": now}))
			return err
		}
		return nil
	})
	if err != nil {
		s.log().WarnContext(ctx, "sms click", "code", l.Code, "err", err)
	}
}
