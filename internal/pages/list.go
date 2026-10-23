package pages

import (
	"context"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Result kinds: what a page brings in.
const (
	ResultSignups       = "signups"
	ResultRegistrations = "registrations"
	ResultSales         = "sales"
	ResultNone          = "none"
)

// PageResult is a page's result over the period.
type PageResult struct {
	Kind        string `json:"kind" enum:"signups,registrations,sales,none"`
	Count       int64  `json:"count"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	// Conversion is the result over the views, in percent (1.9).
	Conversion float64 `json:"conversion"`
}

// PageListItem is a row of Your pages.
type PageListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Path     string `json:"path"`
	URL      string `json:"url"`
	Template string `json:"template" enum:"about,lead_magnet,launch,webinar,sales,blank"`
	Status   string `json:"status" enum:"draft,scheduled,published,unpublished"`
	Changes  bool   `json:"changes"`
	// FirstPublishedAt is the status's date ("Published · 1 Oct").
	FirstPublishedAt *time.Time `json:"firstPublishedAt,omitempty"`
	PublishedAt      *time.Time `json:"publishedAt,omitempty"`
	PublishAt        *time.Time `json:"publishAt,omitempty"`
	UnpublishedAt    *time.Time `json:"unpublishedAt,omitempty"`
	// CountdownEnded is when a countdown of the live page ended ("Countdown
	// ended 6 Oct").
	CountdownEnded *time.Time  `json:"countdownEnded,omitempty"`
	Visibility     string      `json:"visibility" enum:"public,members,paid"`
	Views          int64       `json:"views"`
	Result         PageResult  `json:"result"`
	EditedBy       *PagePerson `json:"editedBy,omitempty"`
	EditedAt       time.Time   `json:"editedAt"`
}

// PagesStats are the four cards over the list.
type PagesStats struct {
	Views int64 `json:"views"`
	// ViewedPages counts the pages viewed; ShopViewed whether the shop was.
	ViewedPages       int      `json:"viewedPages"`
	ShopViewed        bool     `json:"shopViewed"`
	Signups           int64    `json:"signups"`
	SignupPages       []string `json:"signupPages"`
	Registrations     int64    `json:"registrations"`
	RegistrationPages []string `json:"registrationPages"`
	Sales             int64    `json:"sales"`
	SalesCents        int64    `json:"salesCents"`
	Currency          string   `json:"currency"`
	SalesPages        int      `json:"salesPages"`
	ShopSales         bool     `json:"shopSales"`
}

// PageTemplateCount is a template filter chip ("Sales page 1").
type PageTemplateCount struct {
	Template string `json:"template"`
	Count    int    `json:"count"`
}

// PagesShop is the shop's row of Built-in pages.
type PagesShop struct {
	On        bool   `json:"on"`
	Products  int64  `json:"products"`
	Path      string `json:"path"`
	URL       string `json:"url"`
	CanToggle bool   `json:"canToggle"`
}

// PagesTheme is the note under the lists: the theme pages use.
type PagesTheme struct {
	Publisher string `json:"publisher"`
	Logo      bool   `json:"logo"`
	Accent    string `json:"accent"`
	Ground    string `json:"ground"`
	Font      string `json:"font"`
}

// PagesOverview is Forms & public pages › Pages.
type PagesOverview struct {
	Period    string              `json:"period"`
	From      time.Time           `json:"from"`
	Until     time.Time           `json:"until"`
	Stats     PagesStats          `json:"stats"`
	Pages     []PageListItem      `json:"pages"`
	Templates []PageTemplateCount `json:"templates"`
	Shop      PagesShop           `json:"shop"`
	Host      string              `json:"host"`
	Theme     PagesTheme          `json:"theme"`
	WebsiteOn bool                `json:"websiteOn"`
	// Editors and Publishers are who to ask (a Viewer's note).
	Editors    []string `json:"editors"`
	Publishers []string `json:"publishers"`
	CanEdit    bool     `json:"canEdit"`
	CanPublish bool     `json:"canPublish"`
}

// siteLinks are the site's addresses.
type siteLinks struct {
	host string
	base string
}

func (l siteLinks) page(slug string) string { return l.base + "/" + slug }

// links are the site's addresses: the custom domain once it's verified,
// the instance's /p/<workspace> otherwise.
func (s *Service) links(ctx context.Context, ws *store.Workspace) siteLinks {
	base := "/p/" + ws.Slug
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			l := s.Forms.Links(ws, set)
			if l.Custom {
				base = strings.TrimRight(l.Origin(), "/")
			} else {
				base = strings.TrimRight(l.Path(), "/")
			}
		}
	}
	host := strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	return siteLinks{host: host, base: base}
}

// periodStart is the start of a period ending now.
func periodStart(period string, now time.Time) time.Time {
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7)
	case "90d":
		return now.AddDate(0, 0, -90)
	}
	return now.AddDate(0, 0, -30)
}

// resultKind is what a page brings in: sales for a page selling products,
// registrations for a webinar, signups otherwise.
func resultKind(template, src string) string {
	switch template {
	case TemplateSales, TemplateLaunch:
		return ResultSales
	case TemplateWebinar:
		return ResultRegistrations
	case TemplateLeadMagnet, TemplateAbout:
		return ResultSignups
	}
	u := Analyze(src)
	switch {
	case u.BuyButtons > 0:
		return ResultSales
	case len(u.Events) > 0:
		return ResultRegistrations
	case len(u.Forms) > 0:
		return ResultSignups
	}
	return ResultNone
}

func percent(n, of int64) float64 {
	if of <= 0 {
		return 0
	}
	return math.Round(float64(n)*1000/float64(of)) / 10
}

// conversions per page and kind.
type conv struct {
	signups, registrations, sales, cents int64
}

// numbers are pages' views and results between two times.
func (s *Service) numbers(ctx context.Context, ws *store.Workspace, ids []uuid.UUID, from, until time.Time) (map[uuid.UUID]PageNumbers, error) {
	views, convs, err := s.counts(ctx, ws, from, until)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]PageNumbers{}
	for _, id := range ids {
		pg, err := s.Q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			continue
		}
		out[id] = s.number(pg, views[id], convs[id])
	}
	return out, nil
}

func (s *Service) number(pg store.Page, views int64, c conv) PageNumbers {
	kind := resultKind(pg.Template, pg.Html)
	r := PageResult{Kind: kind, Currency: "usd"}
	switch kind {
	case ResultSales:
		r.Count, r.AmountCents = c.sales, c.cents
	case ResultRegistrations:
		r.Count = c.registrations
	case ResultSignups:
		r.Count = c.signups
	}
	if pg.FirstPublishedAt == nil && r.Count == 0 {
		r.Kind = ResultNone
	}
	r.Conversion = percent(r.Count, views)
	return PageNumbers{Views: views, Result: r}
}

// counts are views and conversions per page; the shop is uuid.Nil.
func (s *Service) counts(ctx context.Context, ws *store.Workspace, from, until time.Time) (map[uuid.UUID]int64, map[uuid.UUID]conv, error) {
	views := map[uuid.UUID]int64{}
	loc := time.UTC
	if l, err := time.LoadLocation(ws.Timezone); err == nil && ws.Timezone != "" {
		loc = l
	}
	vrows, err := s.Q.PageViewCounts(ctx, store.PageViewCountsParams{WorkspaceID: ws.ID, Since: dayIn(from, loc), Until: dayIn(until, loc)})
	if err != nil {
		return nil, nil, err
	}
	for _, v := range vrows {
		views[v.PageID] += v.Views
	}
	convs := map[uuid.UUID]conv{}
	crows, err := s.Q.PageConversionCounts(ctx, store.PageConversionCountsParams{WorkspaceID: ws.ID, Since: from, Until: until.Add(time.Second)})
	if err != nil {
		return nil, nil, err
	}
	for _, c := range crows {
		id := uuid.Nil
		if c.PageID != nil {
			id = *c.PageID
		}
		x := convs[id]
		switch c.Kind {
		case "signup":
			x.signups += c.N
		case "registration":
			x.registrations += c.N
		case "sale":
			x.sales += c.N
			x.cents += c.Amount
		}
		convs[id] = x
	}
	return views, convs, nil
}

// dayIn is the day t falls on in loc (views are counted by the
// workspace's day), as a date.
func dayIn(t time.Time, loc *time.Location) time.Time {
	u := t.In(loc)
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// Overview is Forms & public pages › Pages: the numbers, the pages and the shop.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, period string) (PagesOverview, error) {
	now := s.now()
	if period == "" {
		period = "30d"
	}
	from := periodStart(period, now)
	links := s.links(ctx, ws)
	out := PagesOverview{Period: period, From: from, Until: now, Pages: []PageListItem{}, Templates: []PageTemplateCount{}, Host: links.host,
		Stats:   PagesStats{SignupPages: []string{}, RegistrationPages: []string{}, Currency: "usd"},
		CanEdit: p.Can(authz.WebsiteEdit), CanPublish: p.Can(authz.WebsitePublish),
		Editors: s.helpers(ctx, ws, authz.WebsiteEdit, 2, true), Publishers: s.helpers(ctx, ws, authz.WebsitePublish, 3, false)}
	rows, err := s.Q.PagesList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	views, convs, err := s.counts(ctx, ws, from, now)
	if err != nil {
		return out, err
	}
	byTemplate := map[string]int{}
	for _, r := range rows {
		pg := r.Page
		byTemplate[pg.Template]++
		n := s.number(pg, views[pg.ID], convs[pg.ID])
		it := PageListItem{ID: pg.ID.String(), Name: pg.Name, Slug: pg.Slug, Path: "/" + pg.Slug, URL: links.page(pg.Slug), Template: pg.Template, Status: pg.Status,
			FirstPublishedAt: pg.FirstPublishedAt, PublishedAt: pg.PublishedAt, PublishAt: pg.PublishAt, UnpublishedAt: pg.UnpublishedAt,
			Visibility: pg.Visibility, Views: n.Views, Result: n.Result, EditedAt: pg.SavedAt}
		if pg.LiveVersion != nil {
			it.Changes = isLive(pg) && *pg.LiveVersion != pg.Version
		}
		if r.SavedByName != "" && pg.SavedBy != nil {
			it.EditedBy = &PagePerson{ID: pg.SavedBy.String(), Name: r.SavedByName, Initials: initialsOf(r.SavedByName), AvatarColor: r.SavedByColor}
		}
		if isLive(pg) {
			for _, c := range Analyze(s.liveHTML(ctx, pg)).Countdowns {
				if !c.At.After(now) {
					at := c.At
					it.CountdownEnded = &at
				}
			}
		}
		out.Pages = append(out.Pages, it)
		// The cards.
		if n.Views > 0 {
			out.Stats.ViewedPages++
		}
		out.Stats.Views += n.Views
		c := convs[pg.ID]
		if c.signups > 0 {
			out.Stats.Signups += c.signups
			out.Stats.SignupPages = append(out.Stats.SignupPages, pg.Name)
		}
		if c.registrations > 0 {
			out.Stats.Registrations += c.registrations
			out.Stats.RegistrationPages = append(out.Stats.RegistrationPages, pg.Name)
		}
		if c.sales > 0 {
			out.Stats.Sales += c.sales
			out.Stats.SalesCents += c.cents
			out.Stats.SalesPages++
		}
	}
	// The shop and the default product pages count as the nil page.
	if v := views[uuid.Nil]; v > 0 {
		out.Stats.Views += v
		out.Stats.ShopViewed = true
	}
	if c := convs[uuid.Nil]; c.sales > 0 {
		out.Stats.Sales += c.sales
		out.Stats.SalesCents += c.cents
		out.Stats.ShopSales = true
	}
	slices.Sort(out.Stats.SignupPages)
	slices.Sort(out.Stats.RegistrationPages)
	for _, t := range TemplateKeys {
		if byTemplate[t] > 0 {
			out.Templates = append(out.Templates, PageTemplateCount{Template: t, Count: byTemplate[t]})
		}
	}
	out.Shop, err = s.shop(ctx, ws, p, links)
	if err != nil {
		return out, err
	}
	out.Theme = s.theme(ctx, ws)
	if s.Website != nil {
		out.WebsiteOn, _ = s.Website.IsOn(ctx, ws)
	}
	return out, nil
}

// liveHTML is the HTML readers see ("" for a page never published).
func (s *Service) liveHTML(ctx context.Context, pg store.Page) string {
	if pg.LiveVersion == nil {
		return ""
	}
	v, err := s.Q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: pg.ID, Version: *pg.LiveVersion})
	if err != nil {
		return ""
	}
	return v.PageVersion.Html
}

// shop is the shop's switch.
func (s *Service) shop(ctx context.Context, ws *store.Workspace, p *authz.Principal, links siteLinks) (PagesShop, error) {
	if err := s.Q.ShopSettingsEnsure(ctx, ws.ID); err != nil {
		return PagesShop{}, err
	}
	set, err := s.Q.ShopSettingsGet(ctx, ws.ID)
	if err != nil {
		return PagesShop{}, err
	}
	var n int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE workspace_id = $1 AND deleted_at IS NULL AND status = 'published' AND (live->>'showInShop')::boolean`, ws.ID).Scan(&n); err != nil {
		return PagesShop{}, err
	}
	return PagesShop{On: set.ShopOn, Products: n, Path: "/products", URL: links.base + "/products", CanToggle: p.Can(authz.ProductsPublish)}, nil
}

// theme is the branding pages use.
func (s *Service) theme(ctx context.Context, ws *store.Workspace) PagesTheme {
	out := PagesTheme{Publisher: ws.Name}
	if s.Forms == nil {
		return out
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return out
	}
	b := set.Branding
	out.Publisher, out.Logo, out.Accent, out.Ground = b.PublisherName, b.Logo != nil, b.Palette.Accent, b.Palette.Background
	out.Font = fontName(lang(ctx), b.Font, b.CustomFontFamily)
	return out
}

// fontName names the theme's font in a sentence: fonts by their names,
// the system font and a serif in the reader's words.
func fontName(l i18n.Lang, key, custom string) string {
	switch key {
	case "plex":
		return "IBM Plex Sans"
	case "inter":
		return "Inter"
	case "system", "serif":
		return msgs.M("theme.font").Text(l, i18n.Args{"kind": key})
	case "custom":
		if custom != "" {
			return custom
		}
	}
	if key == "" {
		return "IBM Plex Sans"
	}
	return strings.ToUpper(key[:1]) + key[1:]
}
