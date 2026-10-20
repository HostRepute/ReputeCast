package pages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// PageCreate starts a page from a template.
type PageCreate struct {
	Template string `json:"template" enum:"about,lead_magnet,launch,webinar,sales,blank"`
	Name     string `json:"name" maxLength:"120"`
	// Slug is the page's address on the site ("warm-up-planner"); empty
	// takes it from the name.
	Slug string `json:"slug,omitempty" maxLength:"80"`
}

// PagePerson is who saved or published.
type PagePerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	// AvatarColor is the member's avatar color ("" for the default).
	AvatarColor string `json:"avatarColor"`
}

// PageSettings are a page's settings (the builder's Page tab).
type PageSettings struct {
	SEOTitle       string   `json:"seoTitle"`
	SEODescription string   `json:"seoDescription"`
	SocialImage    string   `json:"socialImage"`
	Visibility     string   `json:"visibility" enum:"public,members,paid"`
	Tiers          []string `json:"tiers" doc:"Paid pages: the tiers that may read it (empty: every paid tier)"`
	HideHeader     bool     `json:"hideHeader" doc:"Focused landing pages: the site header shows the logo only"`
	ShowInNav      bool     `json:"showInNav"`
	NavLabel       string   `json:"navLabel"`
}

// PageDraft is the page as edited.
type PageDraft struct {
	Version int32           `json:"version"`
	HTML    string          `json:"html"`
	Project json.RawMessage `json:"project"`
	SavedAt time.Time       `json:"savedAt"`
	SavedBy *PagePerson     `json:"savedBy,omitempty"`
	// Autosaved: the last save was an autosave.
	Autosaved bool `json:"autosaved"`
}

// PageLive is what readers see.
type PageLive struct {
	Version     int32       `json:"version"`
	Slug        string      `json:"slug"`
	Path        string      `json:"path"`
	URL         string      `json:"url"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
	PublishedBy *PagePerson `json:"publishedBy,omitempty"`
}

// PageCounts are the status bar's numbers.
type PageCounts struct {
	Blocks        int `json:"blocks"`
	Images        int `json:"images"`
	ImagesWithAlt int `json:"imagesWithAlt"`
	Links         int `json:"links"`
}

// PagePublishState says what the person may do with the page.
type PagePublishState struct {
	CanEdit    bool `json:"canEdit"`
	CanPublish bool `json:"canPublish"`
	CanManage  bool `json:"canManage" doc:"Navigation, visibility and the paywall (website:manage)"`
	// Publishers are who can publish when the person can't.
	Publishers []string `json:"publishers"`
	// NeedsApproval: the workspace's rules want an approval first.
	NeedsApproval bool     `json:"needsApproval"`
	Rule          string   `json:"rule,omitempty"`
	Approvers     []string `json:"approvers"`
	Reason        string   `json:"reason,omitempty"`
	// Pending is an open approval request.
	Pending *PageApproval `json:"pending,omitempty"`
}

// PageApproval is a page's open approval request.
type PageApproval struct {
	ID          string    `json:"id"`
	RequestedBy string    `json:"requestedBy"`
	RequestedAt time.Time `json:"requestedAt"`
}

// PageDetail is a page in the builder.
type PageDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Template string `json:"template" enum:"about,lead_magnet,launch,webinar,sales,blank"`
	Status   string `json:"status" enum:"draft,scheduled,published,unpublished"`
	// Changes: published, with saved changes not published yet.
	Changes bool `json:"changes"`
	// Host and Path make the address: news.acme.io + /field-guide.
	Host     string       `json:"host"`
	Path     string       `json:"path"`
	URL      string       `json:"url"`
	Settings PageSettings `json:"settings"`
	Draft    PageDraft    `json:"draft"`
	Live     *PageLive    `json:"live,omitempty"`
	// Scheduled pages.
	PublishAt *time.Time `json:"publishAt,omitempty"`
	PublishTZ string     `json:"publishTz,omitempty"`
	// Unpublished pages.
	UnpublishedAt   *time.Time       `json:"unpublishedAt,omitempty"`
	UnpublishReason string           `json:"unpublishReason,omitempty"`
	RedirectTo      string           `json:"redirectTo,omitempty"`
	Checks          []PageCheck      `json:"checks"`
	Counts          PageCounts       `json:"counts"`
	Versions        int64            `json:"versions"`
	Publish         PagePublishState `json:"publish"`
	Numbers         PageNumbers      `json:"numbers"`
	CreatedAt       time.Time        `json:"createdAt"`
	// Nav says whether the header can take a link to the page.
	NavFull bool `json:"navFull"`
	// WebsiteOn: the website is on (Members and Paid need sign-in).
	WebsiteOn bool `json:"websiteOn"`
}

// PageNumbers are a page's 30-day numbers.
type PageNumbers struct {
	Views  int64      `json:"views"`
	Result PageResult `json:"result"`
}

// PageContentInput saves the draft's content (autosave or Save).
type PageContentInput struct {
	HTML    string          `json:"html" maxLength:"2097152"`
	Project json.RawMessage `json:"project,omitempty"`
	// BaseVersion is the version the editor started from: a newer one
	// saved meanwhile is a conflict unless Force.
	BaseVersion int32 `json:"baseVersion" minimum:"1"`
	// Manual: Save (or a block added, moved or deleted) rather than the
	// 30-second autosave.
	Manual bool `json:"manual,omitempty"`
	// Force keeps this version on top of a newer one ("Keep mine").
	Force bool `json:"force,omitempty"`
}

// PageSaved is a saved version.
type PageSaved struct {
	Version int32       `json:"version"`
	SavedAt time.Time   `json:"savedAt"`
	SavedBy *PagePerson `json:"savedBy,omitempty"`
	Changes bool        `json:"changes"`
	Checks  []PageCheck `json:"checks"`
	Counts  PageCounts  `json:"counts"`
}

// PageSettingsInput saves the Page tab.
type PageSettingsInput struct {
	Name           string   `json:"name" minLength:"1" maxLength:"120"`
	Slug           string   `json:"slug" minLength:"1" maxLength:"80"`
	SEOTitle       string   `json:"seoTitle" maxLength:"120"`
	SEODescription string   `json:"seoDescription" maxLength:"300"`
	SocialImage    string   `json:"socialImage" maxLength:"1000"`
	Visibility     string   `json:"visibility" enum:"public,members,paid"`
	Tiers          []string `json:"tiers,omitempty" maxItems:"20"`
	HideHeader     bool     `json:"hideHeader"`
	ShowInNav      bool     `json:"showInNav"`
	NavLabel       string   `json:"navLabel" maxLength:"40"`
}

// conflict is a newer version saved by someone else.
func conflict(ctx context.Context, cur store.Page, by string) *api.Problem {
	who := by
	if who == "" {
		who = word(ctx, "page.someone", nil)
	}
	return api.Err(http.StatusConflict, "version_conflict", word(ctx, "page.conflict", map[string]any{"name": who, "version": cur.Version,
		"time": cur.SavedAt.Format("15:04")})).WithMeta("version", cur.Version, "savedBy", by, "savedAt", cur.SavedAt)
}

// Create starts a page from a template, as a draft.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PageCreate) (PageDetail, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = word(ctx, "template."+in.Template, nil)
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = Slugify(name)
	}
	if !slugRe.MatchString(slug) {
		return PageDetail{}, api.Invalid("body.slug", word(ctx, "page.slugInvalid", nil))
	}
	if Reserved(slug) {
		return PageDetail{}, api.Invalid("body.slug", word(ctx, "page.slugReserved", map[string]any{"slug": slug}))
	}
	opts, err := s.templateOptions(ctx, ws, name)
	if err != nil {
		return PageDetail{}, err
	}
	src := UniqueIDs(Sanitize(TemplateHTML(in.Template, opts)))
	id := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		taken, err := q.PageSlugTaken(ctx, store.PageSlugTakenParams{WorkspaceID: ws.ID, ID: id, Slug: slug})
		if err != nil {
			return err
		}
		if taken {
			return api.Err(http.StatusConflict, "slug_taken", word(ctx, "page.slugTaken", map[string]any{"slug": slug})).WithMeta("suggestion", s.freeSlug(ctx, q, ws.ID, id, slug))
		}
		if err := q.PageInsert(ctx, store.PageInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Slug: slug, Template: in.Template, Visibility: VisibilityPublic,
			ShowInNav: false, NavLabel: "", Project: []byte(`{}`), Html: src, At: now, Actor: actor(p)}); err != nil {
			return err
		}
		pg, err := q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if err := s.snapshot(ctx, q, pg, "created", p); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.created", TargetType: "page", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "slug": slug, "template": in.Template, "status": StatusDraft}})
	})
	if err != nil {
		return PageDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// templateOptions fill a template in for the workspace.
func (s *Service) templateOptions(ctx context.Context, ws *store.Workspace, name string) (TemplateOptions, error) {
	o := TemplateOptions{Name: name, Publisher: ws.Name, TZ: ws.Timezone}
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			o.Publisher = set.Branding.PublisherName
		}
	}
	var form uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM forms WHERE workspace_id = $1 AND status = 'live' ORDER BY created_at LIMIT 1`, ws.ID).Scan(&form); err == nil {
		o.FormID = form.String()
	}
	loc := time.UTC
	if l, err := time.LoadLocation(o.TZ); err == nil {
		loc = l
	}
	o.Date = s.now().In(loc).AddDate(0, 0, 14).Format("2006-01-02")
	return o, nil
}

// freeSlug is base-2, base-3… the first one free.
func (s *Service) freeSlug(ctx context.Context, q *store.Queries, wsID, id uuid.UUID, base string) string {
	for i := 2; i < 200; i++ {
		suffix := fmt.Sprintf("-%d", i)
		b := base
		if len(b)+len(suffix) > 80 {
			b = strings.Trim(b[:80-len(suffix)], "-")
		}
		slug := b + suffix
		if taken, err := q.PageSlugTaken(ctx, store.PageSlugTakenParams{WorkspaceID: wsID, ID: id, Slug: slug}); err == nil && !taken && !Reserved(slug) {
			return slug
		}
	}
	return ""
}

// snapshot keeps the page as it is now as its version.
func (s *Service) snapshot(ctx context.Context, q *store.Queries, pg store.Page, label string, p *authz.Principal) error {
	project := pg.Project
	if len(project) == 0 {
		project = []byte(`{}`)
	}
	return q.PageVersionInsert(ctx, store.PageVersionInsertParams{PageID: pg.ID, Version: pg.Version, Name: pg.Name, Slug: pg.Slug, Settings: settingsOf(pg).json(),
		Project: project, Html: pg.Html, Label: label, SavedBy: actor(p), SavedAt: pg.SavedAt})
}

// get reads a page or answers 404.
func (s *Service) get(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.Page, error) {
	pg, err := q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return pg, api.NotFound("Page")
	}
	return pg, err
}

func (s *Service) lockPage(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.Page, error) {
	pg, err := q.PageLock(ctx, store.PageLockParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return pg, api.NotFound("Page")
	}
	return pg, err
}

// person names a user.
func (s *Service) person(ctx context.Context, id *uuid.UUID) *PagePerson {
	if id == nil {
		return nil
	}
	n, err := s.Q.PageNames(ctx, store.PageNamesParams{SavedBy: id})
	if err != nil || n.SavedByName == "" {
		return nil
	}
	return &PagePerson{ID: id.String(), Name: n.SavedByName, Initials: initialsOf(n.SavedByName), AvatarColor: n.SavedByColor}
}

// Detail reads a page for the builder.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PageDetail, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageDetail{}, err
	}
	links := s.links(ctx, ws)
	set := settingsOf(pg)
	out := PageDetail{ID: pg.ID.String(), Name: pg.Name, Slug: pg.Slug, Template: pg.Template, Status: pg.Status, Host: links.host,
		Path: "/" + pg.Slug, URL: links.page(pg.Slug), CreatedAt: pg.CreatedAt,
		Settings: PageSettings(set),
		Draft: PageDraft{Version: pg.Version, HTML: pg.Html, Project: pg.Project, SavedAt: pg.SavedAt, SavedBy: s.person(ctx, pg.SavedBy),
			Autosaved: pg.Autosaves > 0},
		PublishAt: pg.PublishAt, PublishTZ: pg.PublishTz, UnpublishedAt: pg.UnpublishedAt, UnpublishReason: pg.UnpublishReason, RedirectTo: pg.RedirectTo}
	if len(out.Draft.Project) == 0 {
		out.Draft.Project = json.RawMessage(`{}`)
	}
	if pg.LiveVersion != nil && pg.LiveSlug != nil {
		out.Live = &PageLive{Version: *pg.LiveVersion, Slug: *pg.LiveSlug, Path: "/" + *pg.LiveSlug, URL: links.page(*pg.LiveSlug), PublishedAt: pg.PublishedAt,
			PublishedBy: s.person(ctx, pg.PublishedBy)}
		out.Changes = pg.Status == StatusPublished && *pg.LiveVersion != pg.Version
	}
	out.Checks, out.Counts, err = s.checks(ctx, ws, pg)
	if err != nil {
		return out, err
	}
	if out.Versions, err = s.Q.PageVersionCount(ctx, pg.ID); err != nil {
		return out, err
	}
	out.Publish = s.publishState(ctx, ws, p, pg, out.Checks)
	nums, err := s.numbers(ctx, ws, []uuid.UUID{pg.ID}, s.now().AddDate(0, 0, -30), s.now())
	if err != nil {
		return out, err
	}
	if n, ok := nums[pg.ID]; ok {
		out.Numbers = n
	} else {
		out.Numbers = PageNumbers{Result: PageResult{Kind: resultKind(pg.Template, pg.Html)}}
	}
	if s.Website != nil {
		if st, err := s.Website.PageLink(ctx, s.Q, ws, pg.Slug); err == nil {
			out.NavFull = st.Full && !st.Linked
		}
		on, _ := s.Website.IsOn(ctx, ws)
		out.WebsiteOn = on
	}
	return out, nil
}

// SaveContent saves the draft's blocks: a new version, the live page
// unchanged.
func (s *Service) SaveContent(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PageContentInput) (PageSaved, error) {
	if len(in.Project) > MaxProject {
		return PageSaved{}, api.Invalid("body.project", word(ctx, "page.tooBig", nil))
	}
	src := UniqueIDs(Sanitize(in.HTML))
	project := []byte(in.Project)
	if len(project) == 0 || !json.Valid(project) {
		project = []byte(`{}`)
	}
	var out PageSaved
	var saved store.Page
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if cur.Version != in.BaseVersion && !in.Force {
			by := ""
			if pp := s.person(ctx, cur.SavedBy); pp != nil {
				by = pp.Name
			}
			if cur.SavedBy == nil || p == nil || *cur.SavedBy != p.UserID {
				return conflict(ctx, cur, by)
			}
			// The same person in another tab: theirs is older, this one wins.
		}
		if cur.Html == src && string(cur.Project) == string(project) {
			saved = cur
			return nil
		}
		now := s.now()
		if err := q.PageSaveContent(ctx, store.PageSaveContentParams{ID: cur.ID, Project: project, Html: src, At: now, Actor: actor(p), Manual: in.Manual}); err != nil {
			return err
		}
		saved, err = q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		label := "autosave"
		if in.Manual {
			label = "saved"
		}
		if in.Force && cur.Version != in.BaseVersion {
			label = "kept"
		}
		return s.snapshot(ctx, q, saved, label, p)
	})
	if err != nil {
		return out, err
	}
	s.changed(ctx, ws, p, saved)
	out = PageSaved{Version: saved.Version, SavedAt: saved.SavedAt, SavedBy: s.person(ctx, saved.SavedBy),
		Changes: saved.Status == StatusPublished && saved.LiveVersion != nil && *saved.LiveVersion != saved.Version}
	out.Checks, out.Counts, err = s.checks(ctx, ws, saved)
	return out, err
}

// SaveSettings saves the Page tab: a new version, the live page unchanged
// until it's published. Visibility and navigation need website:manage.
func (s *Service) SaveSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PageSettingsInput) (PageDetail, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.SEOTitle, in.SEODescription, in.NavLabel = strings.TrimSpace(in.SEOTitle), strings.TrimSpace(in.SEODescription), strings.TrimSpace(in.NavLabel)
	in.SocialImage = strings.TrimSpace(in.SocialImage)
	switch {
	case in.Name == "":
		return PageDetail{}, api.Invalid("body.name", word(ctx, "page.nameRequired", nil))
	case !slugRe.MatchString(in.Slug):
		return PageDetail{}, api.Invalid("body.slug", word(ctx, "page.slugInvalid", nil))
	case Reserved(in.Slug):
		return PageDetail{}, api.Invalid("body.slug", word(ctx, "page.slugReserved", map[string]any{"slug": in.Slug}))
	}
	if in.SocialImage != "" {
		if _, ok := SafeURL(in.SocialImage, false); !ok {
			return PageDetail{}, api.Invalid("body.socialImage", word(ctx, "page.imageURL", nil))
		}
	}
	if in.Visibility == "" {
		in.Visibility = VisibilityPublic
	}
	tiers := []uuid.UUID{}
	for _, t := range in.Tiers {
		tid, err := uuid.Parse(t)
		if err != nil {
			return PageDetail{}, api.Invalid("body.tiers", word(ctx, "page.tierInvalid", nil))
		}
		if !slices.Contains(tiers, tid) {
			tiers = append(tiers, tid)
		}
	}
	if in.Visibility != VisibilityPaid {
		tiers = []uuid.UUID{}
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		curTiers := cur.Tiers
		if curTiers == nil {
			curTiers = []uuid.UUID{}
		}
		access := cur.Visibility != in.Visibility || !slices.Equal(curTiers, tiers) || cur.ShowInNav != in.ShowInNav || (in.ShowInNav && cur.NavLabel != in.NavLabel)
		if access && !p.Can(authz.WebsiteManage) {
			return api.Forbidden(authz.WebsiteManage, s.helpers(ctx, ws, authz.WebsiteManage, 2, true))
		}
		if in.Visibility != VisibilityPublic && s.Website != nil {
			if on, _ := s.Website.IsOn(ctx, ws); !on {
				return api.Invalid("body.visibility", word(ctx, "page.needsWebsite", nil))
			}
		}
		if in.Slug != cur.Slug {
			taken, err := q.PageSlugTaken(ctx, store.PageSlugTakenParams{WorkspaceID: ws.ID, ID: cur.ID, Slug: in.Slug})
			if err != nil {
				return err
			}
			if taken {
				return api.Err(http.StatusConflict, "slug_taken", word(ctx, "page.slugTaken", map[string]any{"slug": in.Slug})).
					WithMeta("suggestion", s.freeSlug(ctx, q, ws.ID, cur.ID, in.Slug))
			}
		}
		now := s.now()
		if err := q.PageSaveSettings(ctx, store.PageSaveSettingsParams{ID: cur.ID, Name: in.Name, Slug: in.Slug, SeoTitle: in.SEOTitle, SeoDescription: in.SEODescription,
			SocialImage: in.SocialImage, Visibility: in.Visibility, Tiers: tiers, HideHeader: in.HideHeader, ShowInNav: in.ShowInNav, NavLabel: in.NavLabel,
			At: now, Actor: actor(p)}); err != nil {
			return err
		}
		next, err := q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if err := s.snapshot(ctx, q, next, "settings", p); err != nil {
			return err
		}
		before, after := map[string]any{}, map[string]any{}
		diff := func(k string, a, b any) {
			if fmt.Sprint(a) != fmt.Sprint(b) {
				before[k], after[k] = a, b
			}
		}
		diff("name", cur.Name, next.Name)
		diff("slug", cur.Slug, next.Slug)
		diff("seo_title", cur.SeoTitle, next.SeoTitle)
		diff("seo_description", cur.SeoDescription, next.SeoDescription)
		diff("social_image", cur.SocialImage, next.SocialImage)
		diff("visibility", cur.Visibility, next.Visibility)
		diff("tiers", curTiers, tiers)
		diff("hide_header", cur.HideHeader, next.HideHeader)
		diff("show_in_nav", cur.ShowInNav, next.ShowInNav)
		diff("nav_label", cur.NavLabel, next.NavLabel)
		if len(after) == 0 {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.updated", TargetType: "page", TargetID: cur.ID.String(), TargetName: next.Name,
			Before: before, After: after})
	})
	if err != nil {
		return PageDetail{}, err
	}
	s.changedNow(ctx, ws, p, id)
	return s.Detail(ctx, ws, p, id)
}

// changedNow tells the approvals the page changed, as it is now.
func (s *Service) changedNow(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) {
	if pg, err := s.get(ctx, s.Q, ws, id); err == nil {
		s.changed(ctx, ws, p, pg)
	}
}

// isLive reports whether a page is on the site: published, or published
// with changes scheduled.
func isLive(pg store.Page) bool {
	return pg.LiveVersion != nil && pg.Status != StatusUnpublished
}

// Delete deletes a page. A page that was published answers 410 at its
// address; its header link goes.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if isLive(cur) && !p.Can(authz.WebsitePublish) {
			return api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish, 3, false))
		}
		now := s.now()
		if err := q.PageSoftDelete(ctx, store.PageSoftDeleteParams{ID: cur.ID, At: now, Actor: actor(p)}); err != nil {
			return err
		}
		if cur.LiveSlug != nil && s.Website != nil {
			if _, err := s.Website.SetPageLink(ctx, q, ws, actor(p), "", *cur.LiveSlug, "", false); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.deleted", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status, "slug": cur.Slug}})
	})
}

// PageVersionItem is a row of Version history.
type PageVersionItem struct {
	Version     int32       `json:"version"`
	Name        string      `json:"name"`
	Label       string      `json:"label" doc:"created, autosave, saved, settings, kept, restored, published"`
	SavedAt     time.Time   `json:"savedAt"`
	SavedBy     *PagePerson `json:"savedBy,omitempty"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
	PublishedBy string      `json:"publishedBy,omitempty"`
	Live        bool        `json:"live"`
	Current     bool        `json:"current"`
}

// PageVersionList is Version history.
type PageVersionList struct {
	Items []PageVersionItem `json:"items"`
}

// Versions lists a page's versions, newest first.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PageVersionList, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageVersionList{}, err
	}
	rows, err := s.Q.PageVersions(ctx, pg.ID)
	if err != nil {
		return PageVersionList{}, err
	}
	out := PageVersionList{Items: []PageVersionItem{}}
	for _, r := range rows {
		it := PageVersionItem{Version: r.Version, Name: r.Name, Label: r.Label, SavedAt: r.SavedAt, PublishedAt: r.PublishedAt, PublishedBy: r.PublishedByName,
			Live: pg.LiveVersion != nil && *pg.LiveVersion == r.Version, Current: r.Version == pg.Version}
		if r.SavedByName != "" {
			it.SavedBy = &PagePerson{Name: r.SavedByName, Initials: initialsOf(r.SavedByName), AvatarColor: r.SavedByColor}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// PageVersionDetail is one version's content.
type PageVersionDetail struct {
	PageVersionItem
	Slug     string          `json:"slug"`
	HTML     string          `json:"html"`
	Project  json.RawMessage `json:"project"`
	Settings PageSettings    `json:"settings"`
}

// Version reads one version.
func (s *Service) Version(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int32) (PageVersionDetail, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageVersionDetail{}, err
	}
	v, err := s.Q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: pg.ID, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		return PageVersionDetail{}, api.NotFound("Version")
	}
	if err != nil {
		return PageVersionDetail{}, err
	}
	set := parseSettings(v.PageVersion.Settings)
	out := PageVersionDetail{PageVersionItem: PageVersionItem{Version: v.PageVersion.Version, Name: v.PageVersion.Name, Label: v.PageVersion.Label,
		SavedAt: v.PageVersion.SavedAt, PublishedAt: v.PageVersion.PublishedAt, PublishedBy: v.PublishedByName,
		Live: pg.LiveVersion != nil && *pg.LiveVersion == version, Current: version == pg.Version},
		Slug: v.PageVersion.Slug, HTML: v.PageVersion.Html, Project: v.PageVersion.Project,
		Settings: PageSettings(set)}
	if v.SavedByName != "" {
		out.SavedBy = &PagePerson{Name: v.SavedByName, Initials: initialsOf(v.SavedByName), AvatarColor: v.SavedByColor}
	}
	return out, nil
}

// Restore makes an earlier version the draft again, as a new version.
func (s *Service) Restore(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, version int32) (PageDetail, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		v, err := q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: cur.ID, Version: version})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Version")
		}
		if err != nil {
			return err
		}
		now := s.now()
		if err := q.PageSaveContent(ctx, store.PageSaveContentParams{ID: cur.ID, Project: v.PageVersion.Project, Html: v.PageVersion.Html, At: now, Actor: actor(p), Manual: true}); err != nil {
			return err
		}
		next, err := q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if err := s.snapshot(ctx, q, next, "restored", p); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.version_restored", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
			Before: map[string]any{"version": cur.Version}, After: map[string]any{"version": next.Version, "from": version}})
	})
	if err != nil {
		return PageDetail{}, err
	}
	s.changedNow(ctx, ws, p, id)
	return s.Detail(ctx, ws, p, id)
}

// helpers name who holds perm, as "Sofia Rossi, Marcus Chen or Linh
// Nguyen can publish it" reads: other roles first, then Admins, then
// Owners; Admins and Owners only when adminsOnly; at most max.
func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission, max int, adminsOnly bool) []string {
	out := []string{}
	rows, err := s.Q.ListMembersWithRoles(ctx, ws.ID)
	if err != nil {
		return out
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleAdmin:
			return 1
		case authz.RoleOwner:
			return 2
		}
		return 0
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	for _, r := range rows {
		if adminsOnly && rank(r.RoleKey) == 0 {
			continue
		}
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == max {
				break
			}
		}
	}
	return out
}
