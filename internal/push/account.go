package push

import (
	"context"
	"html"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The account page's Notifications section (brief §6, PushPrompt › "a
// browser that blocked notifications"): this browser's switch, how to allow
// notifications again when the browser blocked them, and the member's other
// browsers with "Turn off". The section is drawn hidden; the prompt script,
// which the site loads only when it's a push site, fills it in and shows it.

func init() {
	website.RegisterAccountSection(accountSection)
}

// services are the push services of the running apps: the account
// section, registered once for the process, asks them whether a
// workspace's website is a push site. Production runs one app; tests start
// several, and a stopped one's closed pool just fails its query.
var (
	servicesMu sync.RWMutex
	services   []*Service
)

// Track makes a service answer for the account section.
func Track(s *Service) {
	servicesMu.Lock()
	services = append(services, s)
	servicesMu.Unlock()
}

// websitePushOn reports whether the workspace's publication website is a
// push site.
func websitePushOn(ctx context.Context, ws *store.Workspace) bool {
	servicesMu.RLock()
	list := slices.Clone(services)
	servicesMu.RUnlock()
	for i := len(list) - 1; i >= 0; i-- {
		var on bool
		err := list[i].Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM push_sites s JOIN push_keys k ON k.workspace_id = s.workspace_id AND k.rotated_at IS NULL
			WHERE s.workspace_id = $1 AND s.kind = 'publication' AND s.removed_at IS NULL)`, ws.ID).Scan(&on)
		if err == nil {
			return on
		}
	}
	return false
}

func accountSection(ctx context.Context, ws *store.Workspace, _ uuid.UUID, l i18n.Lang) (website.AccountSection, bool, error) {
	if !websitePushOn(ctx, ws) {
		return website.AccountSection{}, false, nil
	}
	t := func(key string) string { return msgs.M(key).Text(l, nil) }
	e := html.EscapeString
	var b strings.Builder
	b.WriteString(`<h2 id="pn-h">` + e(t("account.title")) + `</h2><div class="st-pn" data-rc-push-account`)
	for _, k := range []string{"on", "off", "blocked", "unsupported"} {
		b.WriteString(` data-` + k + `="` + e(t("account."+k)) + `"`)
	}
	b.WriteString(` data-turnoff="` + e(t("account.turnOff")) + `" data-since="` + e(msgs.M("account.since").Text(l, i18n.Args{"date": "{date}"})) + `"`)
	// The member's other browsers, named in the page's language.
	for _, k := range []string{BrowserChrome, BrowserChromeAndroid, BrowserEdge, BrowserFirefox, BrowserSafari, BrowserSafariIOS, BrowserOther} {
		b.WriteString(` data-browser-` + k + `="` + e(browserLabel(l, k)) + `"`)
	}
	b.WriteString(`><div class="st-line"><div class="tx"><b id="pn-this">` + e(t("account.browser")) + `</b><span class="st-k" data-rc-push-state></span></div>`)
	b.WriteString(`<input type="checkbox" class="st-sw" role="switch" aria-labelledby="pn-this" data-rc-push-toggle disabled></div>`)
	b.WriteString(`<p class="st-note st-pn-help" data-rc-push-blocked hidden>` + msgs.M("account.allowAgain").HTML(l, nil, i18n.Tags{"b": func(in string) string { return "<b>" + in + "</b>" }}) + `</p>`)
	b.WriteString(`<ul class="st-pn-list" data-rc-push-list></ul></div>`)
	return website.AccountSection{ID: "notifications", Title: t("account.title"), HTML: b.String()}, true, nil
}

// PushAccountSubscription is one of the member's browsers.
type PushAccountSubscription struct {
	ID string `json:"id"`
	// Browser names it in English; Kind is the browser's key, which the
	// account page names in its own language (data-browser-<kind>).
	Browser string    `json:"browser"`
	Kind    string    `json:"kind"`
	Since   time.Time `json:"since"`
	// Endpoint is how the page tells this browser apart: its own
	// subscription's push resource.
	Endpoint string `json:"endpoint"`
}

// memberFrom is the member signed in with the request, on the site the
// request is for.
func (s *Service) memberFrom(ctx context.Context, ws *store.Workspace, r *http.Request) uuid.UUID {
	if s.Member == nil {
		return uuid.Nil
	}
	// A cookie-carrying change must come from the site itself.
	if r.Method != http.MethodGet {
		if o := requestOrigin(r); o == "" || !strings.EqualFold(strings.TrimPrefix(strings.TrimPrefix(o, "https://"), "http://"), r.Host) {
			return uuid.Nil
		}
	}
	return s.Member(ctx, ws, r)
}

// serveAccount lists the member's browsers.
func (s *Service) serveAccount(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	ctx := r.Context()
	contact := s.memberFrom(ctx, ws, r)
	if contact == uuid.Nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "signed_out"})
		return
	}
	rows, err := s.Q.PushSubscriptionsOfContact(ctx, store.PushSubscriptionsOfContactParams{WorkspaceID: ws.ID, ContactID: &contact})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}
	out := []PushAccountSubscription{}
	for _, row := range rows {
		out = append(out, PushAccountSubscription{ID: row.ID.String(), Browser: browserLabel(i18n.Default, row.Browser), Kind: row.Browser, Since: row.CreatedAt,
			Endpoint: row.Endpoint})
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": out, "timeZone": tzName(ws)})
}

// serveAccountOff turns notifications off in one of the member's browsers.
func (s *Service) serveAccountOff(w http.ResponseWriter, r *http.Request, ws *store.Workspace) {
	ctx := r.Context()
	contact := s.memberFrom(ctx, ws, r)
	if contact == uuid.Nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "signed_out"})
		return
	}
	var b struct {
		ID string `json:"id"`
	}
	if err := readBody(r, &b, 1<<10); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	id, err := uuid.Parse(b.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	sub, err := s.Q.PushSubscription(ctx, store.PushSubscriptionParams{ID: id, WorkspaceID: ws.ID})
	if err != nil || sub.ContactID == nil || *sub.ContactID != contact || sub.RemovedAt != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "unknown"})
		return
	}
	if err := s.Q.PushSubscriptionRemove(ctx, store.PushSubscriptionRemoveParams{ID: id, At: ptr(s.now()), Reason: "turned_off"}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "turned_off"})
}

// browserLabel names a browser as the account page writes it, in l:
// "Chrome", "Safari on iPhone and iPad".
func browserLabel(l i18n.Lang, b string) string {
	switch b {
	case BrowserChrome:
		return "Chrome"
	case BrowserEdge:
		return "Edge"
	case BrowserFirefox:
		return "Firefox"
	case BrowserChromeAndroid, BrowserSafari, BrowserSafariIOS:
		return msgs.M("browser."+b).Text(l, nil)
	}
	return msgs.M("browser.other").Text(l, nil)
}

// SiteScripts are the scripts a site page of the publication website loads
// for push: the prompt, when the site is a push site.
func (s *Service) SiteScripts(ctx context.Context, ws *store.Workspace, host string) []string {
	if !s.servesWorker(ctx, host) {
		return nil
	}
	var wsSlug = ws.Slug
	return []string{"/push/v1/prompt.js?w=" + wsSlug + "&s=" + strings.ToLower(host)}
}

// TurnOff is the preference center's "Turn off notifications in this
// browser" when host is a push site.
func (s *Service) TurnOff(ctx context.Context, ws *store.Workspace, host string, l i18n.Lang) (label, done, none string, ok bool) {
	if !s.servesWorker(ctx, host) {
		return "", "", "", false
	}
	return msgs.M("preferences.turnOff").Text(l, nil), msgs.M("preferences.done").Text(l, nil), msgs.M("preferences.none").Text(l, nil), true
}
