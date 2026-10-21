package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The prompt designer (brief §3.8, board PushPrompt): a form type. A soft
// prompt shows first; the browser's own prompt appears only after Allow,
// so "Not now" never uses up the browser's one chance to ask. iPhone and
// iPad support web push only for sites added to the Home Screen: the
// prompt shows how to add the site, or stays hidden there.

// Prompt styles.
const (
	StyleBar    = "bar"    // slide-in bar at the bottom of the page
	StyleDialog = "dialog" // a centred card
	StyleBell   = "bell"   // a bell button that opens the prompt on click
)

// PushPromptRule is a "When" rule: on or off and its number.
type PushPromptRule struct {
	On bool `json:"on"`
	N  int  `json:"n" minimum:"0" maximum:"10000"`
}

// PushPromptWhen is when the prompt shows: as soon as one checked rule is
// met. A bell button opens when the reader clicks it.
type PushPromptWhen struct {
	// Seconds on a page.
	Seconds PushPromptRule `json:"seconds"`
	// Scroll is the percent of a post scrolled.
	Scroll PushPromptRule `json:"scroll"`
	// Posts read on the site.
	Posts PushPromptRule `json:"posts"`
	// Button: when the reader clicks the publisher's own button
	// (an element with data-rc-push).
	Button bool `json:"button"`
}

// PushPromptConfig is a prompt: style, text, when, where, how often and
// what iPhone and iPad readers see.
type PushPromptConfig struct {
	Style string `json:"style" enum:"bar,dialog,bell"`
	// Corner is where a bell button sits.
	Corner  string         `json:"corner,omitempty" enum:"bottom_right,bottom_left,"`
	Title   string         `json:"title" maxLength:"60"`
	Message string         `json:"message" maxLength:"160"`
	Allow   string         `json:"allow" maxLength:"30"`
	NotNow  string         `json:"notNow" maxLength:"30"`
	When    PushPromptWhen `json:"when"`
	// Pages: every page except Paths ("all_except"), or only Paths.
	Pages string   `json:"pages" enum:"all_except,only"`
	Paths []string `json:"paths" maxItems:"20"`
	// AskAgainDays is how long "Not now" keeps the prompt away.
	AskAgainDays int `json:"askAgainDays" minimum:"1" maximum:"365"`
	// IOS: "steps" shows how to add the site to the Home Screen; "hidden"
	// keeps the prompt away on iPhone and iPad.
	IOS string `json:"ios" enum:"steps,hidden"`
}

// Limits the composer counts against.
const (
	PromptTitleMax   = 40
	PromptMessageMax = 120
)

// defaultConfig is a new prompt's: a slide-in bar after 30 seconds or
// half a post, asking again after 14 days.
func defaultConfig(site string) PushPromptConfig {
	return PushPromptConfig{Style: StyleBar, Title: "Get new posts as notifications",
		Message: "One notification when a new post is out. Never more than one a day.", Allow: "Allow", NotNow: "Not now",
		When:  PushPromptWhen{Seconds: PushPromptRule{On: true, N: 30}, Scroll: PushPromptRule{On: true, N: 50}, Posts: PushPromptRule{N: 3}},
		Pages: "all_except", Paths: []string{"/account", "/subscribe"}, AskAgainDays: 14, IOS: "steps"}
}

// parseConfig reads a stored prompt, filling what's missing.
func parseConfig(raw json.RawMessage) PushPromptConfig {
	c := defaultConfig("")
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	if c.Paths == nil {
		c.Paths = []string{}
	}
	if c.AskAgainDays <= 0 {
		c.AskAgainDays = 14
	}
	return c
}

var pathRe = regexp.MustCompile(`^/[A-Za-z0-9._~!$&'()*+,;=:@%/-]*$`)

// check validates a prompt, field by field.
func (c *PushPromptConfig) check() error {
	c.Title, c.Message = strings.TrimSpace(c.Title), strings.TrimSpace(c.Message)
	c.Allow, c.NotNow = strings.TrimSpace(c.Allow), strings.TrimSpace(c.NotNow)
	switch {
	case !slices.Contains([]string{StyleBar, StyleDialog, StyleBell}, c.Style):
		return api.Invalid("body.style", "Pick a style: slide-in bar, dialog or bell button.")
	case c.Title == "":
		return api.Invalid("body.title", "Write the prompt's title.")
	case utf8.RuneCountInString(c.Title) > PromptTitleMax:
		return api.Invalid("body.title", fmt.Sprintf("Keep the title to %d characters.", PromptTitleMax))
	case utf8.RuneCountInString(c.Message) > PromptMessageMax:
		return api.Invalid("body.message", fmt.Sprintf("Keep the message to %d characters.", PromptMessageMax))
	case c.Allow == "" || c.NotNow == "":
		return api.Invalid("body.allow", "Name both buttons.")
	}
	if c.Style == StyleBell && c.Corner == "" {
		c.Corner = "bottom_right"
	}
	if c.Style != StyleBell {
		c.Corner = ""
		w := c.When
		if (!w.Seconds.On || w.Seconds.N < 0) && (!w.Scroll.On || w.Scroll.N <= 0) && (!w.Posts.On || w.Posts.N <= 0) && !w.Button {
			return api.Invalid("body.when", "Check at least one rule for when the prompt shows.")
		}
		if w.Scroll.On && (w.Scroll.N < 1 || w.Scroll.N > 100) {
			return api.Invalid("body.when.scroll", "Scrolling is a percent of the post, 1 to 100.")
		}
	}
	if c.Pages != "only" {
		c.Pages = "all_except"
	}
	clean := []string{}
	for _, p := range c.Paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if !pathRe.MatchString(p) || len(p) > 200 {
			return api.Invalid("body.paths", "Paths start with / and are part of the site's addresses, like /account.")
		}
		if !slices.Contains(clean, p) {
			clean = append(clean, p)
		}
	}
	c.Paths = clean
	if c.Pages == "only" && len(c.Paths) == 0 {
		return api.Invalid("body.paths", "Add the pages the prompt shows on.")
	}
	if c.AskAgainDays < 1 || c.AskAgainDays > 365 {
		return api.Invalid("body.askAgainDays", "Ask again after 1 to 365 days.")
	}
	if c.IOS != "hidden" {
		c.IOS = "steps"
	}
	return nil
}

// PushPromptStats are a prompt's numbers over a period.
type PushPromptStats struct {
	// Since is where the period starts: 30 days ago, or the day the site's
	// snippet was installed when that's later.
	Since        time.Time `json:"since"`
	SinceSnippet bool      `json:"sinceSnippet"`
	Shown        int64     `json:"shown"`
	Allowed      int64     `json:"allowed"`
	Granted      int64     `json:"granted"`
	NotNow       int64     `json:"notNow"`
	Closed       int64     `json:"closed"`
	BellClicked  int64     `json:"bellClicked"`
	Blocked      int64     `json:"blocked"`
	// AllowedPct, NotNowPct and ClosedPct are shares of the times it was
	// shown (bell clicks for a bell); GrantedPct is a share of Allow.
	AllowedPct float64 `json:"allowedPct"`
	GrantedPct float64 `json:"grantedPct"`
	NotNowPct  float64 `json:"notNowPct"`
	ClosedPct  float64 `json:"closedPct"`
	// iPhone and iPad.
	IOSSubscribers int64   `json:"iosSubscribers"`
	IOSSteps       int64   `json:"iosSteps"`
	IOSAdded       int64   `json:"iosAdded"`
	IOSAddedPct    float64 `json:"iosAddedPct"`
}

// PushConsentRecord is one subscription's consent, as the designer shows
// the latest one made through the prompt.
type PushConsentRecord struct {
	Browser   string    `json:"browser"`
	Site      string    `json:"site"`
	Contact   string    `json:"contact,omitempty"`
	ContactID string    `json:"contactId,omitempty"`
	LinkedBy  string    `json:"linkedBy,omitempty"`
	At        time.Time `json:"at"`
	TimeZone  string    `json:"timeZone"`
	IP        string    `json:"ip,omitempty"`
	Page      string    `json:"page"`
	Version   int       `json:"version"`
	Text      string    `json:"text"`
	// PublishedAt and PublishedBy are when that version went live.
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	PublishedBy string     `json:"publishedBy,omitempty"`
}

// PushPromptSiteOption is a site the prompt can be on.
type PushPromptSiteOption struct {
	// ID is the push site's id, empty for the publication website when it
	// isn't a push site yet.
	ID       string `json:"id,omitempty"`
	Host     string `json:"host"`
	Origin   string `json:"origin"`
	Kind     string `json:"kind" enum:"publication,snippet"`
	PushSite bool   `json:"pushSite"`
	// Website is the publication website's own home page (the preview
	// shows it).
	Website bool `json:"website"`
}

// PushPrompt is the designer's view of a prompt.
type PushPrompt struct {
	FormID string `json:"formId"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,live,paused"`
	// Draft is what's being edited; Live what readers see (nil before the
	// first publication).
	Draft       PushPromptConfig  `json:"draft"`
	Live        *PushPromptConfig `json:"live,omitempty"`
	LiveVersion int               `json:"liveVersion"`
	LiveSince   *time.Time        `json:"liveSince,omitempty"`
	// Changed is true while the draft differs from what's live.
	Changed   bool                   `json:"changed"`
	Site      *PushPromptSiteOption  `json:"site,omitempty"`
	Sites     []PushPromptSiteOption `json:"sites"`
	PushOn    bool                   `json:"pushOn"`
	Stats     PushPromptStats        `json:"stats"`
	Consent   *PushConsentRecord     `json:"consent,omitempty"`
	IconURL   string                 `json:"iconUrl,omitempty"`
	UpdatedAt time.Time              `json:"updatedAt"`
	UpdatedBy string                 `json:"updatedBy"`
	CanEdit   bool                   `json:"canEdit"`
}

// PushPromptCreate makes a prompt.
type PushPromptCreate struct {
	Name string `json:"name" minLength:"1" maxLength:"120"`
	// SiteID is the push site it's for; empty picks the first.
	SiteID string `json:"siteId,omitempty" maxLength:"40"`
}

// formSlug makes a unique form slug from a name.
func (s *Service) formSlug(ctx context.Context, q store.DBTX, ws *store.Workspace, name string) (string, error) {
	base := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "-"), "-")
	if base == "" {
		base = "push-prompt"
	}
	base = clip(base, 50)
	slug := base
	for i := 2; i < 200; i++ {
		var taken bool
		if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM forms WHERE workspace_id = $1 AND slug = $2)`, ws.ID, slug).Scan(&taken); err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return base + "-" + uuid.NewString()[:8], nil
}

// CreatePrompt makes a push prompt: a form of type "push", a draft until
// it's published.
func (s *Service) CreatePrompt(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushPromptCreate) (PushPrompt, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return PushPrompt{}, api.Invalid("body.name", "Name the prompt.")
	}
	var site *uuid.UUID
	if in.SiteID != "" {
		id, err := uuid.Parse(in.SiteID)
		if err != nil {
			return PushPrompt{}, api.Invalid("body.siteId", "Pick one of the workspace's push sites.")
		}
		if _, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: id, WorkspaceID: ws.ID}); err != nil {
			return PushPrompt{}, api.Invalid("body.siteId", "Pick one of the workspace's push sites.")
		}
		site = &id
	} else if sites, err := s.Q.PushSites(ctx, ws.ID); err == nil && len(sites) > 0 {
		site = &sites[0].ID
	}
	cfg := defaultConfig("")
	draft, _ := json.Marshal(cfg)
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		slug, err := s.formSlug(ctx, tx, ws, name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, kind, slug, status, content, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, 'push', $4, 'draft', '{}', $5, $6, $6)`, id, ws.ID, name, slug, uuidPtr(p.UserID), now); err != nil {
			return err
		}
		if err := q.PushPromptInsert(ctx, store.PushPromptInsertParams{FormID: id, WorkspaceID: ws.ID, SiteID: site, Draft: draft,
			UpdatedBy: uuidPtr(p.UserID), UpdatedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.create", TargetType: "form", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "kind": "push", "slug": slug}})
	})
	if err != nil {
		return PushPrompt{}, err
	}
	return s.Prompt(ctx, ws, p, id)
}

// promptRow loads a prompt.
func (s *Service) promptRow(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.PushPromptRow, error) {
	r, err := s.Q.PushPrompt(ctx, store.PushPromptParams{FormID: id, WorkspaceID: ws.ID})
	if err != nil {
		return r, api.MapError(err, "Push prompt")
	}
	return r, nil
}

// liveConfig is the published prompt, nil before the first publication.
func (s *Service) liveConfig(ctx context.Context, formID uuid.UUID, version int32) (*PushPromptConfig, *store.PushPromptVersionRow) {
	if version <= 0 {
		return nil, nil
	}
	v, err := s.Q.PushPromptVersion(ctx, store.PushPromptVersionParams{FormID: formID, Version: version})
	if err != nil {
		return nil, nil
	}
	c := parseConfig(v.Config)
	return &c, &v
}

// Prompt is the designer's view.
func (s *Service) Prompt(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushPrompt, error) {
	r, err := s.promptRow(ctx, ws, id)
	if err != nil {
		return PushPrompt{}, err
	}
	out := PushPrompt{FormID: id.String(), Name: r.Name, Status: r.Status, Draft: parseConfig(r.Draft), LiveVersion: int(r.LiveVersion), LiveSince: r.LiveSince,
		UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName, CanEdit: p.Can(authz.FormsEdit), Sites: []PushPromptSiteOption{}}
	if live, _ := s.liveConfig(ctx, id, r.LiveVersion); live != nil {
		out.Live = live
		a, _ := json.Marshal(live)
		b, _ := json.Marshal(out.Draft)
		out.Changed = string(a) != string(b)
	} else {
		out.Changed = true
	}
	_, keyErr := s.liveKey(ctx, s.Q, ws)
	out.PushOn = keyErr == nil
	sites, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	website := ""
	if s.Website != nil {
		website = s.Website(ctx, ws)
	}
	for _, st := range sites {
		o := PushPromptSiteOption{ID: st.ID.String(), Host: st.Host, Origin: st.Origin, Kind: st.Kind, PushSite: true, Website: st.Origin == website}
		out.Sites = append(out.Sites, o)
		if r.SiteID != nil && *r.SiteID == st.ID {
			oo := o
			out.Site = &oo
			out.IconURL = s.iconURL(ctx, ws, store.PushSite{ID: st.ID, IconMediaID: st.IconMediaID})
		}
	}
	if website != "" && !slices.ContainsFunc(out.Sites, func(o PushPromptSiteOption) bool { return o.Origin == website }) {
		_, host, _ := ParseOrigin(website)
		o := PushPromptSiteOption{Host: host, Origin: website, Kind: KindPublication, Website: true}
		out.Sites = append([]PushPromptSiteOption{o}, out.Sites...)
		if r.SiteID == nil {
			oo := o
			out.Site = &oo
		}
	}
	if out.IconURL == "" && s.Logo != nil {
		out.IconURL = s.Logo(ctx, ws)
	}
	out.Stats, err = s.promptStats(ctx, ws, r)
	if err != nil {
		return out, err
	}
	out.Consent = s.consentRecord(ctx, ws, id)
	return out, nil
}

// promptStats are the prompt's numbers over the last 30 days, or since the
// site's snippet was installed when that's later.
func (s *Service) promptStats(ctx context.Context, ws *store.Workspace, r store.PushPromptRow) (PushPromptStats, error) {
	loc := wsLocation(ws)
	now := s.now().In(loc)
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -29)
	out := PushPromptStats{Since: since}
	if r.SiteID != nil {
		if st, err := s.Q.PushSiteAny(ctx, *r.SiteID); err == nil && st.SnippetSeenAt != nil && st.SnippetSeenAt.After(since) {
			d := st.SnippetSeenAt.In(loc)
			out.Since, out.SinceSnippet = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc), true
		}
	}
	t, err := s.Q.PushPromptTotals(ctx, store.PushPromptTotalsParams{FormID: r.FormID, Since: out.Since})
	if err != nil {
		return out, err
	}
	out.Shown, out.Allowed, out.Granted, out.NotNow, out.Closed = t.Shown, t.Allowed, t.Granted, t.NotNow, t.Closed
	out.BellClicked, out.Blocked, out.IOSSteps, out.IOSAdded = t.BellClicked, t.Blocked, t.IosSteps, t.IosAdded
	base := out.Shown
	if cfg := parseConfig(r.Draft); cfg.Style == StyleBell && out.BellClicked > 0 {
		base = out.BellClicked
	}
	out.AllowedPct, out.NotNowPct, out.ClosedPct = pct(out.Allowed, base), pct(out.NotNow, base), pct(out.Closed, base)
	out.GrantedPct = pct(out.Granted, out.Allowed)
	out.IOSAddedPct = pct(out.IOSAdded, out.IOSSteps)
	if r.SiteID != nil {
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM push_subscriptions WHERE site_id = $1 AND removed_at IS NULL AND browser = 'safari_ios'`, *r.SiteID).
			Scan(&out.IOSSubscribers)
	}
	return out, nil
}

// consentRecord is the latest subscription made through the prompt, a
// linked one when there is one.
func (s *Service) consentRecord(ctx context.Context, ws *store.Workspace, formID uuid.UUID) *PushConsentRecord {
	var (
		rec      PushConsentRecord
		contact  *uuid.UUID
		first    string
		last     string
		email    string
		ip       *string
		version  *int32
		siteHost string
	)
	err := s.Pool.QueryRow(ctx, `SELECT s.browser, st.host, s.contact_id, coalesce(c.first_name, ''), coalesce(c.last_name, ''), coalesce(c.email, ''),
			s.linked_by, s.created_at, s.time_zone, host(s.ip), s.page, s.prompt_version, s.prompt_text
		FROM push_subscriptions s JOIN push_sites st ON st.id = s.site_id LEFT JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.prompt_form_id = $2 AND s.removed_at IS NULL
		ORDER BY (s.contact_id IS NOT NULL) DESC, s.created_at DESC LIMIT 1`, ws.ID, formID).
		Scan(&rec.Browser, &siteHost, &contact, &first, &last, &email, &rec.LinkedBy, &rec.At, &rec.TimeZone, &ip, &rec.Page, &version, &rec.Text)
	if err != nil {
		return nil
	}
	rec.Site = siteHost
	if contact != nil {
		rec.ContactID = contact.String()
		rec.Contact = strings.TrimSpace(first + " " + last)
		if rec.Contact == "" {
			rec.Contact = email
		}
	}
	if ip != nil {
		rec.IP = *ip
	}
	if version != nil {
		rec.Version = int(*version)
		if v, err := s.Q.PushPromptVersion(ctx, store.PushPromptVersionParams{FormID: formID, Version: *version}); err == nil {
			rec.PublishedAt, rec.PublishedBy = &v.PublishedAt, v.PublishedByName
		}
	}
	return &rec
}

// PushPromptSave is the draft from the designer.
type PushPromptSave struct {
	Config PushPromptConfig `json:"config"`
	// SiteID is the push site the prompt is for.
	SiteID string `json:"siteId,omitempty" maxLength:"40"`
}

// SaveDraft keeps the designer's draft. Readers see it once it's published.
func (s *Service) SaveDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PushPromptSave) (PushPrompt, error) {
	if _, err := s.promptRow(ctx, ws, id); err != nil {
		return PushPrompt{}, err
	}
	cfg := in.Config
	if err := cfg.check(); err != nil {
		return PushPrompt{}, err
	}
	var site *uuid.UUID
	if in.SiteID != "" {
		sid, err := uuid.Parse(in.SiteID)
		if err != nil {
			return PushPrompt{}, api.Invalid("body.siteId", "Pick one of the workspace's push sites.")
		}
		if _, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: sid, WorkspaceID: ws.ID}); err != nil {
			return PushPrompt{}, api.Invalid("body.siteId", "Pick one of the workspace's push sites.")
		}
		site = &sid
	}
	draft, _ := json.Marshal(cfg)
	if err := s.Q.PushPromptSaveDraft(ctx, store.PushPromptSaveDraftParams{FormID: id, WorkspaceID: ws.ID, Draft: draft, SiteID: site,
		UpdatedBy: uuidPtr(p.UserID), UpdatedAt: s.now()}); err != nil {
		return PushPrompt{}, err
	}
	return s.Prompt(ctx, ws, p, id)
}

// Publish makes the draft what readers see: a new version, live.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushPrompt, error) {
	r, err := s.promptRow(ctx, ws, id)
	if err != nil {
		return PushPrompt{}, err
	}
	if _, err := s.liveKey(ctx, s.Q, ws); errors.Is(err, errNoKeys) {
		return PushPrompt{}, api.Err(409, "push_off", "Turn push on in Sending providers › Push first: the prompt subscribes readers with the workspace's keys.")
	} else if err != nil {
		return PushPrompt{}, err
	}
	if r.SiteID == nil {
		return PushPrompt{}, api.Err(409, "not_a_push_site", "The prompt's site isn't a push site yet. Add it in Push setup; readers can't subscribe until it is.")
	}
	st, err := s.Q.PushSite(ctx, store.PushSiteParams{ID: *r.SiteID, WorkspaceID: ws.ID})
	if err != nil {
		return PushPrompt{}, api.Err(409, "not_a_push_site", "The prompt's site isn't a push site yet. Add it in Push setup; readers can't subscribe until it is.")
	}
	cfg := parseConfig(r.Draft)
	if err := cfg.check(); err != nil {
		return PushPrompt{}, err
	}
	live, _ := s.liveConfig(ctx, id, r.LiveVersion)
	version := r.LiveVersion
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		changed := live == nil
		if live != nil {
			a, _ := json.Marshal(live)
			b, _ := json.Marshal(cfg)
			changed = string(a) != string(b)
		}
		if changed {
			version++
			raw, _ := json.Marshal(cfg)
			if err := q.PushPromptVersionInsert(ctx, store.PushPromptVersionInsertParams{FormID: id, Version: version, Config: raw,
				PublishedBy: uuidPtr(p.UserID), PublishedAt: now}); err != nil {
				return err
			}
			if err := q.PushPromptPublish(ctx, store.PushPromptPublishParams{FormID: id, LiveVersion: version, At: now, UpdatedBy: uuidPtr(p.UserID)}); err != nil {
				return err
			}
		} else if r.Status == "live" {
			return api.Conflict("Nothing new to publish: readers see these settings.")
		}
		if _, err := tx.Exec(ctx, `UPDATE forms SET status = 'live', updated_at = $2 WHERE id = $1`, id, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_prompt.published", TargetType: "form", TargetID: id.String(),
			TargetName: r.Name, Before: map[string]any{"status": r.Status, "version": r.LiveVersion},
			After: map[string]any{"status": "live", "version": version, "site": st.Host, "style": cfg.Style, "title": cfg.Title, "message": cfg.Message}})
	})
	if err != nil {
		return PushPrompt{}, err
	}
	return s.Prompt(ctx, ws, p, id)
}

// DeletePrompt deletes a prompt (logged): readers stop seeing it. The
// subscriptions it made stay, with their consent records (the exact text
// and version are kept with each).
func (s *Service) DeletePrompt(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	r, err := s.promptRow(ctx, ws, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM forms WHERE id = $1 AND workspace_id = $2 AND kind = 'push'`, id, ws.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_prompt.deleted", TargetType: "form", TargetID: id.String(), TargetName: r.Name,
			Before: map[string]any{"status": r.Status, "live_version": r.LiveVersion}})
	})
}

// Discard throws the draft away: it becomes what's live again.
func (s *Service) Discard(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushPrompt, error) {
	r, err := s.promptRow(ctx, ws, id)
	if err != nil {
		return PushPrompt{}, err
	}
	live, _ := s.liveConfig(ctx, id, r.LiveVersion)
	cfg := defaultConfig("")
	if live != nil {
		cfg = *live
	}
	draft, _ := json.Marshal(cfg)
	if err := s.Q.PushPromptSaveDraft(ctx, store.PushPromptSaveDraftParams{FormID: id, WorkspaceID: ws.ID, Draft: draft, SiteID: r.SiteID,
		UpdatedBy: uuidPtr(p.UserID), UpdatedAt: s.now()}); err != nil {
		return PushPrompt{}, err
	}
	return s.Prompt(ctx, ws, p, id)
}

// PushPromptStatus pauses or resumes a live prompt.
type PushPromptStatus struct {
	Status string `json:"status" enum:"live,paused"`
}

// SetStatus pauses a prompt (readers stop seeing it; subscriptions stay) or
// resumes it.
func (s *Service) SetStatus(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PushPromptStatus) (PushPrompt, error) {
	r, err := s.promptRow(ctx, ws, id)
	if err != nil {
		return PushPrompt{}, err
	}
	if r.LiveVersion == 0 {
		return PushPrompt{}, api.Conflict("Publish the prompt first: a draft has nothing to pause.")
	}
	if r.Status == in.Status {
		return s.Prompt(ctx, ws, p, id)
	}
	action := "push_prompt.paused"
	if in.Status == "live" {
		action = "push_prompt.resumed"
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE forms SET status = $2, updated_at = $3 WHERE id = $1`, id, in.Status, s.now()); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "form", TargetID: id.String(), TargetName: r.Name,
			Before: map[string]any{"status": r.Status}, After: map[string]any{"status": in.Status}})
	})
	if err != nil {
		return PushPrompt{}, err
	}
	return s.Prompt(ctx, ws, p, id)
}

// promptEventFields map the events a prompt reports to its counters.
var promptEventFields = map[string]bool{"shown": true, "allow": true, "granted": true, "not_now": true, "closed": true, "bell": true, "blocked": true,
	"ios_steps": true, "ios_added": true}

// RecordPromptEvent adds one event to the prompt's day.
func (s *Service) RecordPromptEvent(ctx context.Context, ws *store.Workspace, formID uuid.UUID, event string) error {
	if !promptEventFields[event] {
		return nil
	}
	p := store.PushPromptDayAddParams{FormID: formID, WorkspaceID: ws.ID, Day: dayOf(s.now(), wsLocation(ws))}
	switch event {
	case "shown":
		p.Shown = 1
	case "allow":
		p.Allowed = 1
	case "granted":
		p.Granted = 1
	case "not_now":
		p.NotNow = 1
	case "closed":
		p.Closed = 1
	case "bell":
		p.BellClicked = 1
	case "blocked":
		p.Blocked = 1
	case "ios_steps":
		p.IosSteps = 1
	case "ios_added":
		p.IosAdded = 1
	}
	return s.Q.PushPromptDayAdd(ctx, p)
}

// dayOf is t's calendar day in loc, as a date.
func dayOf(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
