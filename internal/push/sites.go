package push

import (
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sites (brief §3.8). Browsers accept a service worker only from the
// site's own origin, so each site serves one: ReputeCast serves it on the
// publication website; another site hosts rc-push-sw.js at its root (or
// adds one import line to the worker it already has), and "Check" fetches
// it to report whether it was found, its version and its scope.

// WorkerVersion is the version of the service worker this build serves.
// A site hosting an older rc-push-sw.js gets the "old version" warning.
const WorkerVersion = 1

// WorkerFile is the file a snippet site hosts at its root.
const WorkerFile = "rc-push-sw.js"

// workerMarker finds the version in a hosted worker's first lines.
var workerMarker = regexp.MustCompile(`ReputeCast push service worker · version (\d+)`)

// Site kinds.
const (
	KindPublication = "publication"
	KindSnippet     = "snippet"
)

// Check statuses. A publication site's worker is always "built_in".
const (
	WorkerBuiltIn    = "built_in"
	WorkerFound      = "found"
	WorkerOld        = "old_version"
	WorkerMissing    = "missing"
	WorkerUnreach    = "unreachable"
	WorkerNotAWorker = "not_a_worker"
	WorkerUnchecked  = "unchecked"
)

// PushWorker is a site's service worker as the last check found it.
type PushWorker struct {
	Status string `json:"status" enum:"built_in,found,old_version,missing,unreachable,not_a_worker,unchecked"`
	// URL is where the worker must answer (https://acme.io/rc-push-sw.js).
	URL string `json:"url"`
	// Mode is "file" (rc-push-sw.js) or "import" (one line in the site's
	// own worker).
	Mode    string `json:"mode" enum:"file,import"`
	Path    string `json:"path"`
	Version *int   `json:"version,omitempty"`
	// Current is the version this build serves.
	Current    int        `json:"current"`
	Scope      string     `json:"scope"`
	HTTPStatus *int       `json:"httpStatus,omitempty"`
	Detail     string     `json:"detail,omitempty"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
}

// PushSitePrompt is a site's prompt in the sites table.
type PushSitePrompt struct {
	FormID    string     `json:"formId"`
	Name      string     `json:"name"`
	Style     string     `json:"style" enum:"bar,dialog,bell"`
	Status    string     `json:"status" enum:"draft,live,paused"`
	LiveSince *time.Time `json:"liveSince,omitempty"`
}

// PushSite is a site readers turn notifications on from.
type PushSite struct {
	ID     string `json:"id"`
	Origin string `json:"origin"`
	Host   string `json:"host"`
	Kind   string `json:"kind" enum:"publication,snippet"`
	// Label is "Publication site", "Company site".
	Label         string           `json:"label"`
	Subscribers   int64            `json:"subscribers"`
	Linked        int64            `json:"linked"`
	Worker        PushWorker       `json:"worker"`
	Prompts       []PushSitePrompt `json:"prompts"`
	SnippetSeenAt *time.Time       `json:"snippetSeenAt,omitempty"`
	AddedAt       time.Time        `json:"addedAt"`
	AddedBy       string           `json:"addedBy"`
	IconURL       string           `json:"iconUrl,omitempty"`
	Demo          bool             `json:"demo"`
}

// ParseOrigin reads a site address ("help.acme.io", "https://help.acme.io/
// blog") as an origin: https, a host and maybe a port. http is accepted
// for localhost only, as browsers do for service workers.
func ParseOrigin(raw string) (origin, host string, ok bool) {
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
	h := strings.ToLower(u.Hostname())
	local := h == "localhost" || h == "127.0.0.1" || strings.HasSuffix(h, ".localhost")
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && local:
	default:
		return "", "", false
	}
	if !local && (!strings.Contains(h, ".") || net.ParseIP(h) != nil) {
		return "", "", false
	}
	host = h
	if p := u.Port(); p != "" {
		if (u.Scheme == "https" && p != "443") || (u.Scheme == "http" && p != "80") {
			host += ":" + p
		}
	}
	return u.Scheme + "://" + host, host, true
}

// insertSite adds a site in q's transaction, audited.
func (s *Service) insertSite(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, origin, kind, label string, now time.Time) (uuid.UUID, error) {
	_, host, _ := ParseOrigin(origin)
	if label == "" && kind == KindPublication {
		// Stored in English; read in each member's language (siteLabel).
		label = mLabelPublicationSite.Text(i18n.Default, nil)
	}
	status := WorkerUnchecked
	if kind == KindPublication {
		status = WorkerBuiltIn
	}
	id := uuid.Must(uuid.NewV7())
	var by *uuid.UUID
	if p != nil {
		by = uuidPtr(p.UserID)
	}
	if err := q.PushSiteInsert(ctx, store.PushSiteInsertParams{ID: id, WorkspaceID: ws.ID, Origin: origin, Host: host, Kind: kind, Label: label,
		WorkerPath: "/" + WorkerFile, WorkerMode: "file", CheckStatus: status, AddedBy: by, AddedAt: now}); err != nil {
		if strings.Contains(err.Error(), "push_sites_origin_idx") {
			return uuid.Nil, api.Conflict(mProblemSiteExists.Text(i18n.LangFrom(ctx), i18n.Args{"host": host}))
		}
		return uuid.Nil, err
	}
	return id, s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_site.added", TargetType: "push_site", TargetID: id.String(),
		TargetName: host, After: map[string]any{"origin": origin, "kind": kind, "label": label}})
}

// PushSiteInput adds a site.
type PushSiteInput struct {
	// Address is the site's address: "help.acme.io" or "https://help.acme.io".
	Address string `json:"address" minLength:"1" maxLength:"300"`
	Label   string `json:"label,omitempty" maxLength:"80"`
}

// AddSite adds a site. The publication website's address becomes a built-in
// site; any other gets the snippet and the file to host.
func (s *Service) AddSite(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushSiteInput) (PushSite, error) {
	l := i18n.LangFrom(ctx)
	origin, _, ok := ParseOrigin(in.Address)
	if !ok {
		return PushSite{}, api.Invalid("body.address", mProblemSiteAddress.Text(l, nil))
	}
	if _, err := s.liveKey(ctx, s.Q, ws); errors.Is(err, errNoKeys) {
		return PushSite{}, api.Conflict(mProblemSiteNoKeys.Text(l, nil))
	} else if err != nil {
		return PushSite{}, err
	}
	kind := KindSnippet
	if s.Website != nil && s.Website(ctx, ws) == origin {
		kind = KindPublication
	}
	var id uuid.UUID
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		id, err = s.insertSite(ctx, q, ws, p, origin, kind, strings.TrimSpace(in.Label), s.now())
		return err
	})
	if err != nil {
		return PushSite{}, err
	}
	return s.site(ctx, ws, id)
}

// PushSiteUpdate changes a site's label, its worker and its icon.
type PushSiteUpdate struct {
	Label      string `json:"label" maxLength:"80"`
	WorkerMode string `json:"workerMode" enum:"file,import"`
	// WorkerPath is where the site's worker answers: /rc-push-sw.js, or the
	// site's own worker (/sw.js) that imports ours.
	WorkerPath string `json:"workerPath" maxLength:"200"`
	// IconMediaID is a 192 × 192 image from the media library; empty uses
	// the workspace logo.
	IconMediaID string `json:"iconMediaId,omitempty" maxLength:"40"`
}

// UpdateSite saves a site's settings.
func (s *Service) UpdateSite(ctx context.Context, ws *store.Workspace, id uuid.UUID, in PushSiteUpdate) (PushSite, error) {
	st, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PushSite{}, api.MapError(err, "Push site")
	}
	wp := strings.TrimSpace(in.WorkerPath)
	if wp == "" {
		wp = "/" + WorkerFile
	}
	if !strings.HasPrefix(wp, "/") || strings.ContainsAny(wp, "?#\\ ") || strings.Contains(wp, "..") || !strings.HasSuffix(wp, ".js") {
		return PushSite{}, api.Invalid("body.workerPath", mProblemWorkerPath.Text(i18n.LangFrom(ctx), nil))
	}
	var icon *uuid.UUID
	if in.IconMediaID != "" {
		mid, err := uuid.Parse(in.IconMediaID)
		if err != nil {
			return PushSite{}, api.Invalid("body.iconMediaId", mProblemPickImage.Text(i18n.LangFrom(ctx), nil))
		}
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM media_assets WHERE id = $1 AND workspace_id = $2)`, mid, ws.ID).Scan(&ok); err != nil || !ok {
			return PushSite{}, api.Invalid("body.iconMediaId", mProblemPickImage.Text(i18n.LangFrom(ctx), nil))
		}
		icon = &mid
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushSiteUpdate(ctx, store.PushSiteUpdateParams{ID: id, WorkspaceID: ws.ID, Label: strings.TrimSpace(in.Label), WorkerPath: path.Clean(wp),
			WorkerMode: in.WorkerMode, IconMediaID: icon}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_site.updated", TargetType: "push_site", TargetID: id.String(),
			TargetName: st.Host, Before: map[string]any{"label": st.Label, "worker_mode": st.WorkerMode, "worker_path": st.WorkerPath},
			After: map[string]any{"label": strings.TrimSpace(in.Label), "worker_mode": in.WorkerMode, "worker_path": path.Clean(wp)}})
	})
	if err != nil {
		return PushSite{}, err
	}
	return s.site(ctx, ws, id)
}

// RemoveSite removes a site. Its subscriptions end: browsers there can no
// longer be reached.
func (s *Service) RemoveSite(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	st, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Push site")
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE push_subscriptions SET removed_at = $2, removed_reason = 'site_removed', endpoint = '', p256dh = NULL, auth = NULL
			WHERE site_id = $1 AND removed_at IS NULL`, id, now)
		if err != nil {
			return err
		}
		if err := q.PushSiteRemove(ctx, store.PushSiteRemoveParams{ID: id, WorkspaceID: ws.ID, At: &now, RemovedBy: uuidPtr(p.UserID)}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_prompts SET site_id = NULL WHERE site_id = $1`, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_site.removed", TargetType: "push_site", TargetID: id.String(),
			TargetName: st.Host, Before: map[string]any{"origin": st.Origin, "kind": st.Kind}, Metadata: map[string]any{"subscriptions_ended": tag.RowsAffected()}})
	})
}

// workerURL is where a site's worker answers.
func workerURL(st store.PushSite) string { return st.Origin + st.WorkerPath }

// workerView is the site's worker as the app shows it.
func (s *Service) workerView(ctx context.Context, ws *store.Workspace, st store.PushSite) PushWorker {
	w := PushWorker{Status: st.CheckStatus, URL: workerURL(st), Mode: st.WorkerMode, Path: st.WorkerPath, Current: WorkerVersion, Scope: st.CheckScope,
		Detail: st.CheckDetail, CheckedAt: st.CheckedAt}
	if st.CheckVersion != nil {
		v := int(*st.CheckVersion)
		w.Version = &v
	}
	if st.CheckHttpStatus != nil {
		c := int(*st.CheckHttpStatus)
		w.HTTPStatus = &c
	}
	if st.Kind == KindPublication {
		w.URL = st.Origin + "/" + WorkerFile
		w.Status, w.Scope, w.Version = WorkerBuiltIn, "/", ptr(WorkerVersion)
		// The website must still serve it: off, or on another address,
		// it doesn't.
		if s.Website != nil && s.Website(ctx, ws) != st.Origin {
			w.Status, w.Version, w.Detail = WorkerMissing, nil, "website_off"
		}
	}
	return w
}

// CheckResult is what fetching a site's worker found.
type CheckResult struct {
	Status     string
	Version    *int
	Scope      string
	HTTPStatus *int
	Detail     string
}

// workerScope is the widest scope a worker at p can take: its directory,
// or the Service-Worker-Allowed header.
func workerScope(p, allowed string) string {
	if allowed != "" {
		if u, err := url.Parse(allowed); err == nil && strings.HasPrefix(u.Path, "/") {
			return u.Path
		}
	}
	d := path.Dir(p)
	if !strings.HasSuffix(d, "/") {
		d += "/"
	}
	return d
}

// isJavaScript reports whether a Content-Type is one browsers accept for
// a service worker.
func isJavaScript(ct string) bool {
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	switch mt {
	case "text/javascript", "application/javascript", "application/x-javascript", "text/ecmascript", "application/ecmascript":
		return true
	}
	return false
}

// importLine matches an import of the instance's worker.
var importLine = regexp.MustCompile(`importScripts\(\s*["']([^"']+/push/v1/sw\.js)["']\s*\)`)

// FetchWorker fetches a site's worker and says what it is. Browsers don't
// follow redirects for service workers, so neither does the check.
func FetchWorker(ctx context.Context, client *http.Client, u, mode string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return CheckResult{Status: WorkerUnreach, Detail: "invalid address"}
	}
	req.Header.Set("Service-Worker", "script")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "ReputeCast push check (+https://reputecast.com)")
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		return CheckResult{Status: WorkerUnreach, Detail: clip(netguard.Describe(ctx, err), 200)}
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	res := CheckResult{HTTPStatus: &code}
	switch {
	case code == http.StatusNotFound || code == http.StatusGone:
		res.Status = WorkerMissing
		return res
	case code >= 300 && code < 400:
		res.Status, res.Detail = WorkerNotAWorker, "redirect"
		return res
	case code >= 500 || code == http.StatusTooManyRequests:
		res.Status = WorkerUnreach
		return res
	case code != http.StatusOK:
		res.Status = WorkerMissing
		return res
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		res.Status, res.Detail = WorkerUnreach, "the answer was cut off"
		return res
	}
	if !isJavaScript(resp.Header.Get("Content-Type")) {
		res.Status, res.Detail = WorkerNotAWorker, "content_type:"+clip(resp.Header.Get("Content-Type"), 80)
		return res
	}
	res.Scope = workerScope(req.URL.Path, resp.Header.Get("Service-Worker-Allowed"))
	text := string(body)
	if m := workerMarker.FindStringSubmatch(text); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 32)
		v := int(n)
		res.Version = &v
		res.Status = WorkerFound
		if v < WorkerVersion {
			res.Status = WorkerOld
		}
		return res
	}
	if importLine.MatchString(text) {
		// It imports the instance's worker, always the current version.
		v := WorkerVersion
		res.Version, res.Status = &v, WorkerFound
		return res
	}
	res.Status, res.Detail = WorkerNotAWorker, "no_marker"
	if mode == "import" {
		res.Detail = "no_import"
	}
	return res
}

// CheckSite fetches a snippet site's worker now and keeps what it found.
// The demo's sites answer from the demo data: they're never fetched.
func (s *Service) CheckSite(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PushSite, error) {
	st, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PushSite{}, api.MapError(err, "Push site")
	}
	if err := s.check(ctx, ws, st); err != nil {
		return PushSite{}, err
	}
	return s.site(ctx, ws, id)
}

// check runs one site's check, stores it and notifies when the worker
// went missing.
func (s *Service) check(ctx context.Context, ws *store.Workspace, st store.PushSite) error {
	if st.Kind == KindPublication {
		return nil
	}
	now := s.now()
	var res CheckResult
	if st.Demo {
		res = CheckResult{Status: st.CheckStatus, Version: nil, Scope: st.CheckScope, HTTPStatus: nil, Detail: st.CheckDetail}
		if st.CheckVersion != nil {
			v := int(*st.CheckVersion)
			res.Version = &v
		}
		if st.CheckHttpStatus != nil {
			c := int(*st.CheckHttpStatus)
			res.HTTPStatus = &c
		}
		if res.Status == WorkerUnchecked {
			v, c := WorkerVersion, 200
			res = CheckResult{Status: WorkerFound, Version: &v, Scope: "/", HTTPStatus: &c}
		}
	} else {
		client := s.HTTP
		if client == nil {
			client = &http.Client{Timeout: 10 * time.Second}
		}
		res = FetchWorker(ctx, client, workerURL(st), st.WorkerMode)
	}
	var version, code *int32
	if res.Version != nil {
		version = ptr(int32(*res.Version))
	}
	if res.HTTPStatus != nil {
		code = ptr(int32(*res.HTTPStatus))
	}
	wasBad := st.CheckStatus == WorkerMissing || st.CheckStatus == WorkerUnreach || st.CheckStatus == WorkerNotAWorker
	isBad := res.Status == WorkerMissing || res.Status == WorkerUnreach || res.Status == WorkerNotAWorker
	var released int64
	defer func() {
		if released > 0 {
			s.kick(ctx)
		}
	}()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.PushSiteSetCheck(ctx, store.PushSiteSetCheckParams{ID: st.ID, CheckStatus: res.Status, CheckVersion: version, CheckScope: res.Scope,
			CheckHttpStatus: code, CheckDetail: res.Detail, CheckedAt: &now}); err != nil {
			return err
		}
		if wasBad && !isBad {
			// The file is back: what was held goes.
			var err error
			released, err = releaseSite(ctx, tx, st.ID, now)
			return err
		}
		if !isBad || wasBad {
			return nil
		}
		if err := holdSite(ctx, tx, st.ID, now); err != nil {
			return err
		}
		var subs int64
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM push_subscriptions WHERE site_id = $1 AND removed_at IS NULL`, st.ID).Scan(&subs)
		// Kept as messages, so each member reads it in their own language.
		args := i18n.Args{"host": st.Host, "url": workerURL(st), "answer": "none", "status": "", "count": subs, "file": WorkerFile}
		if res.HTTPStatus != nil {
			args["answer"], args["status"] = "status", strconv.Itoa(*res.HTTPStatus)
		}
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.ProvidersView, notify.Notification{Kind: notify.KindPushWorkerMissing, Severity: "bad",
			TitleText: mNotifyWorkerTitle.Keep(args), SummaryText: mNotifyWorkerSummary.Keep(args), BodyText: mNotifyWorkerBody.Keep(args),
			Link: "/w/" + ws.Slug + "/providers/push", Data: map[string]any{"site_id": st.ID.String(), "status": res.Status}})
	})
}

// CheckAll is the daily check of every snippet site.
func (s *Service) CheckAll(ctx context.Context) error {
	sites, err := s.Q.PushSitesAll(ctx)
	if err != nil {
		return err
	}
	for _, st := range sites {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
		if err != nil {
			continue
		}
		if err := s.check(ctx, &ws, st); err != nil && !errors.Is(err, context.Canceled) {
			s.log().WarnContext(ctx, "push site check", "site", st.ID, "err", err)
		}
	}
	return nil
}

// site reads one site with its counts and prompts.
func (s *Service) site(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PushSite, error) {
	all, err := s.sites(ctx, ws)
	if err != nil {
		return PushSite{}, err
	}
	for _, st := range all {
		if st.ID == id.String() {
			return st, nil
		}
	}
	return PushSite{}, api.NotFound("Push site")
}

// sites reads every site with its subscribers, worker and prompts.
func (s *Service) sites(ctx context.Context, ws *store.Workspace) ([]PushSite, error) {
	rows, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	counts, err := s.Q.PushSubscriptionsBySite(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	bySite := map[uuid.UUID]store.PushSubscriptionsBySiteRow{}
	for _, c := range counts {
		bySite[c.SiteID] = c
	}
	prompts, err := s.Q.PushPrompts(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	l := i18n.LangFrom(ctx)
	out := make([]PushSite, 0, len(rows))
	for _, r := range rows {
		st := store.PushSite{ID: r.ID, WorkspaceID: r.WorkspaceID, Origin: r.Origin, Host: r.Host, Kind: r.Kind, Label: r.Label, WorkerPath: r.WorkerPath,
			WorkerMode: r.WorkerMode, CheckStatus: r.CheckStatus, CheckVersion: r.CheckVersion, CheckScope: r.CheckScope, CheckHttpStatus: r.CheckHttpStatus,
			CheckDetail: r.CheckDetail, CheckedAt: r.CheckedAt, MissingSince: r.MissingSince, IconMediaID: r.IconMediaID, SnippetSeenAt: r.SnippetSeenAt,
			Demo: r.Demo, AddedBy: r.AddedBy, AddedAt: r.AddedAt}
		v := PushSite{ID: r.ID.String(), Origin: r.Origin, Host: r.Host, Kind: r.Kind, Label: siteLabel(l, r.Kind, r.Label), Subscribers: bySite[r.ID].N, Linked: bySite[r.ID].Linked,
			Worker: s.workerView(ctx, ws, st), Prompts: []PushSitePrompt{}, SnippetSeenAt: r.SnippetSeenAt, AddedAt: r.AddedAt, AddedBy: r.AddedByName, Demo: r.Demo}
		v.IconURL = s.iconURL(ctx, ws, st)
		for _, p := range prompts {
			if p.SiteID == nil || *p.SiteID != r.ID {
				continue
			}
			cfg := parseConfig(p.Draft)
			if p.LiveVersion > 0 {
				if pv, err := s.Q.PushPromptVersion(ctx, store.PushPromptVersionParams{FormID: p.FormID, Version: p.LiveVersion}); err == nil {
					cfg = parseConfig(pv.Config)
				}
			}
			v.Prompts = append(v.Prompts, PushSitePrompt{FormID: p.FormID.String(), Name: p.Name, Style: cfg.Style, Status: p.Status, LiveSince: p.LiveSince})
		}
		out = append(out, v)
	}
	return out, nil
}

// siteLabel is a site's label in a language: the publication website's
// default label, which ReputeCast wrote, is translated; a label someone
// typed stays as it is.
func siteLabel(l i18n.Lang, kind, label string) string {
	if kind == KindPublication && label == mLabelPublicationSite.Text(i18n.Default, nil) {
		return mLabelPublicationSite.Text(l, nil)
	}
	return label
}

// iconURL is a site's notification icon: its own, else the workspace logo.
func (s *Service) iconURL(ctx context.Context, ws *store.Workspace, st store.PushSite) string {
	if st.IconMediaID != nil {
		var name string
		if err := s.Pool.QueryRow(ctx, `SELECT file_name FROM media_assets WHERE id = $1`, *st.IconMediaID).Scan(&name); err == nil {
			return s.origin() + media.URL(*st.IconMediaID, name)
		}
	}
	if s.Logo != nil {
		return s.Logo(ctx, ws)
	}
	return ""
}

// siteOf finds the listed site an origin belongs to.
func (s *Service) siteOf(ctx context.Context, ws *store.Workspace, origin string) (store.PushSite, error) {
	return s.Q.PushSiteByOrigin(ctx, store.PushSiteByOriginParams{WorkspaceID: ws.ID, Origin: strings.ToLower(origin)})
}
