package publicpages

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// coverView is a post's cover in one place.
type coverView struct {
	Src, Alt, Class string
	Width, Height   int
}

// metaView is a post's meta line.
type metaView struct {
	DateFirst        bool
	Date, Short, ISO string
	Tag              string
	Label            website.PostLabel
}

// postCardView is a post in a list or a card.
type postCardView struct {
	Href    string
	Title   string
	Excerpt string
	Cover   coverView
	Meta    metaView
	// Title and excerpt with the search's matches marked (the archive).
	TitleHTML   template.HTML
	ExcerptHTML template.HTML
}

// subFormView is the subscribe form of the site.
type subFormView struct {
	Class, Action, Page, ID, EmailLabel, Placeholder, Button string
	Disabled                                                 bool
}

// subscribedView replaces the subscribe form for a signed-in member.
type subscribedView struct {
	Title       string
	Text        string
	ManageHref  string
	UpgradeHref string
	UpgradeText string
}

// tierCardView is a tier's card in the tiers band.
type tiersBandView struct {
	Headline, Intro string
	Toggle          bool
	MonthlyHref     string
	YearlyHref      string
	Monthly, Yearly string
	Yearly2         bool
	Free            *freeCardView
	Cards           []bandCardView
	Columns         string
}

type freeCardView struct {
	Name, Price, Line string
	Items             []string
	Href, Button      string
}

type bandCardView struct {
	Name, Highlight, Amount, Unit, Sub string
	Benefits                           []string
	Href, Button                       string
	Primary                            bool
}

// homeSectionView is a section of the home page.
type homeSectionView struct {
	Kind string
	// subscribe
	Form       *subFormView
	Note       string
	Subscribed *subscribedView
	// featured, latest, tag rows
	Cards    []postCardView
	Title    string
	AllHref  string
	AllText  string
	FeatCols string
	// about
	AboutTitle, AboutText, AboutHref, AboutLink string
	// tiers
	Tiers *tiersBandView
}

type homeBody struct {
	Sections []homeSectionView
	// HeroTitle is set when the subscribe band is hidden (the page's h1).
	HiddenTitle string
}

// home is "/" of the site: the home page while the website is on, else
// the subscribe page.
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !h.siteOn(r.Context(), s) {
		http.Redirect(w, r, s.links.Subscribe(""), http.StatusFound)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	period := r.URL.Query().Get("period")
	if period != "month" && period != "year" {
		period = ""
	}
	body, err := h.homeBody(r.Context(), s, v, period)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v.Body = body
	v.mark("home", v.Frame)
	v.Title = v.Frame.Title
	v.Description = v.Frame.Description
	if v.Description == "" {
		v.Description = v.Frame.Tagline
	}
	v.Canonical = s.canonPath()
	v.OG = &og{Type: "website", SiteName: v.Frame.Title, Title: v.Frame.Title, Description: v.Description, URL: v.Canonical, Image: v.Frame.SocialImage}
	v.Feeds = []siteFeed{{Title: v.Frame.Title, Href: s.canonPath("feed")}}
	v.JSONLD = append(v.JSONLD, map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": v.Frame.Title, "url": s.canonPath(),
		"description": v.Description, "publisher": map[string]any{"@type": "Organization", "name": s.set.Branding.PublisherName}})
	if period != "" {
		v.Robots = "noindex, follow"
	}
	h.Website.CountView(r.Context(), s.ws, r, website.PageHome, website.HomeID)
	h.renderSite(w, r, s, "home", http.StatusOK, v)
}

// homeBody builds the home page's sections.
func (h *Handler) homeBody(ctx context.Context, s *site, v *siteView, period string) (homeBody, error) {
	l := v.labels.Lang()
	home, err := h.Website.Home(ctx, s.ws, l, period)
	if err != nil {
		return homeBody{}, err
	}
	b := homeBody{Sections: []homeSectionView{}}
	hero := false
	for _, sec := range home.Sections {
		sv := homeSectionView{Kind: sec.Kind}
		switch sec.Kind {
		case website.SectionSubscribe:
			hero = true
			sv.Note = sec.Note
			if v.Member != nil {
				sv.Subscribed = h.subscribedBand(ctx, s, v, home.Tiers)
			}
			if sv.Subscribed == nil {
				sv.Form = &subFormView{Class: "st-sub", Action: s.links.Subscribe(""), Page: s.canonPath(), ID: "hero-email",
					EmailLabel: v.T("site.emailLabel"), Placeholder: v.T("site.emailPlaceholder"), Button: v.T("site.subscribe"), Disabled: v.Preview}
			}
		case website.SectionFeatured:
			for _, p := range sec.Posts {
				sv.Cards = append(sv.Cards, h.card(s, v, p, home.Tiers, "feature"))
			}
			sv.FeatCols = ""
			if len(sv.Cards) == 3 {
				sv.FeatCols = "n3"
			}
		case website.SectionLatest:
			place := "row"
			if home.Layout == website.LayoutGrid {
				place = "grid"
			}
			for _, p := range sec.Posts {
				sv.Cards = append(sv.Cards, h.card(s, v, p, home.Tiers, place))
			}
			sv.Title = v.T("site.latest")
			sv.AllHref = s.links.Archive()
			sv.AllText = v.T("site.latestAll", "count", home.Total)
			if home.Layout == website.LayoutGrid {
				sv.Kind = "grid"
			}
		case website.SectionTag:
			for _, p := range sec.Posts {
				sv.Cards = append(sv.Cards, h.card(s, v, p, home.Tiers, "card"))
			}
			sv.Title = sec.Tag.Name
			sv.AllHref = s.path("tag", sec.Tag.Slug)
			sv.AllText = v.T("site.tagAll", "count", sec.TagCount, "tag", sec.Tag.Name)
		case website.SectionAbout:
			sv.AboutTitle = sec.AboutTitle
			if sv.AboutTitle == "" {
				sv.AboutTitle = v.T("site.aboutTitle")
			}
			sv.AboutText = sec.AboutText
			if link := strings.TrimSpace(sec.AboutLink); link != "" {
				sv.AboutLink = v.T("site.aboutMore")
				if strings.HasPrefix(link, "/") {
					sv.AboutHref = s.sitePath(link)
				} else if strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://") {
					sv.AboutHref = link
				}
			}
		case website.SectionTiers:
			sv.Tiers = h.tiersBand(s, v, sec.Upgrade, period)
			if sv.Tiers == nil {
				continue
			}
		}
		b.Sections = append(b.Sections, sv)
	}
	if !hero {
		b.HiddenTitle = v.Frame.Title
	}
	return b, nil
}

// subscribedBand is the hero for a signed-in member: what they get.
func (h *Handler) subscribedBand(ctx context.Context, s *site, v *siteView, tiers []gate.Tier) *subscribedView {
	sess := v.Session
	if sess == nil {
		return nil
	}
	pc, err := h.Audience.LoadPublicContact(ctx, h.Q, s.ws, sess.ContactID)
	if err != nil {
		return nil
	}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return nil
	}
	copyOf := h.copyOf(ctx, s)
	var on []audience.PublicList
	for _, l := range lists {
		if pc.Lists[l.ID] == "subscribed" {
			on = append(on, l)
		}
	}
	name := func(l audience.PublicList) string {
		if copyOf != nil {
			return copyOf.ListName(v.Lang, l)
		}
		return l.Name
	}
	b := &subscribedView{ManageHref: s.path("account")}
	if len(on) == 0 {
		b.Title = v.T("site.subscribed.none")
		b.Text = v.T("site.subscribed.noneText")
	} else {
		b.Title = v.T("site.subscribed.title", "list", name(on[0]))
		next, _ := h.nextIssues(ctx, s)
		if t, ok := next[on[0].ID]; ok {
			b.Text = v.T("site.subscribed.next", "email", sess.Email, "day", weekday(v, t.In(location(s.ws))))
		}
		if len(on) > 1 {
			rest := make([]string, 0, len(on)-1)
			for _, l := range on[1:] {
				rest = append(rest, name(l))
			}
			also := v.T("site.subscribed.also", "lists", i18n.AndList(v.labels.Lang().Locale, rest), "count", len(rest))
			b.Text = strings.TrimSpace(b.Text + " " + also)
		}
	}
	if len(tiers) > 0 && !h.memberPaid(ctx, s, sess) {
		b.UpgradeText = v.T("site.upgradeTo", "tier", tiers[0].Label)
		b.UpgradeHref = s.path("upgrade")
	}
	return b
}

func (h *Handler) memberPaid(ctx context.Context, s *site, sess *website.Session) bool {
	rd, err := h.Website.Reader(ctx, s.ws, sess, "")
	return err == nil && rd.Member.Paid()
}

// weekday names the day of a send: "Tuesday" in English, the date in
// other languages.
func weekday(v *siteView, t time.Time) string {
	if v.labels.Code() == "en" {
		return t.Format("Monday")
	}
	return v.labels.NextDate(t)
}

// siteDate is a post's date without its weekday: "9 October 2026".
func siteDate(v *siteView, t time.Time) string {
	if v.labels.Code() == "en" {
		return t.Format("2 January 2006")
	}
	return v.labels.FullDate(t)
}

// card is a post in one place: feature (large card), row (the list), card
// (a tag row), grid (the grid layout).
func (h *Handler) card(s *site, v *siteView, p website.PublicPost, tiers []gate.Tier, place string) postCardView {
	loc := location(s.ws)
	at := p.PublishedAt.In(loc)
	label := p.Label
	if tiers != nil || label.Kind == "" {
		label = h.Website.Label(p.Visibility, tiers, v.labels.Lang())
	}
	c := postCardView{Href: s.path("posts", p.Slug), Title: p.Title, Excerpt: p.Excerpt,
		Meta: metaView{Date: v.labels.FullDate(at), Short: v.labels.DayMonth(at), ISO: p.PublishedAt.UTC().Format(time.RFC3339), Label: label}}
	if len(p.Tags) > 0 {
		c.Meta.Tag = p.Tags[0].Name
	}
	c.Cover = coverView{Src: h.mediaURL(p.CoverURL), Alt: p.CoverAlt}
	switch place {
	case "feature":
		c.Cover.Class = "r14"
		c.Meta.Date = siteDate(v, at)
	case "row":
		c.Meta.DateFirst = true
		c.Cover.Class = "r10"
	case "card", "grid":
		c.Meta.Date = v.labels.DayMonth(at)
		c.Cover.Class = "r12"
	}
	return c
}

// tiersBand is the home's paid tiers band from the upgrade page.
func (h *Handler) tiersBand(s *site, v *siteView, up *earn.UpgradeView, period string) *tiersBandView {
	if up == nil || len(up.Cards) == 0 {
		return nil
	}
	t := &tiersBandView{Headline: up.Headline, Intro: up.Intro, Toggle: up.Toggle, Monthly: up.Monthly, Yearly: up.Yearly, Yearly2: up.Period == "year"}
	base := s.path()
	t.MonthlyHref = base + "?" + url.Values{"period": {"month"}}.Encode() + "#tiers-h"
	t.YearlyHref = base + "?" + url.Values{"period": {"year"}}.Encode() + "#tiers-h"
	if up.Free != nil {
		f := &freeCardView{Name: up.Free.Name, Price: up.Free.Price, Line: up.Free.Line, Href: s.links.Subscribe(""), Button: up.Free.Button}
		for _, line := range strings.Split(up.Free.Text, "\n") {
			if line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")); line != "" {
				f.Items = append(f.Items, line)
			}
		}
		t.Free = f
	}
	for _, c := range up.Cards {
		bc := bandCardView{Name: c.Name, Highlight: c.Highlight, Amount: c.Amount, Unit: c.Unit, Sub: strings.TrimSpace(strings.Trim(c.Sub+" · "+c.Trial, " ·")),
			Benefits: c.Benefits, Button: c.Button, Primary: c.Primary}
		q := url.Values{"period": {c.Period}}
		bc.Href = s.path("upgrade") + "?" + q.Encode()
		t.Cards = append(t.Cards, bc)
	}
	n := len(t.Cards)
	if t.Free != nil {
		n++
	}
	if n == 2 {
		t.Columns = "n2"
	}
	return t
}

// mediaURL is an image of the image library at the instance's address:
// the site's own domain doesn't serve /media/.
func (h *Handler) mediaURL(src string) string {
	if strings.HasPrefix(src, "/media/") {
		return strings.TrimRight(h.URLs.Base(), "/") + src
	}
	return src
}

// mediaHTML points a body's library images at the instance.
func (h *Handler) mediaHTML(body string) string {
	base := strings.TrimRight(h.URLs.Base(), "/")
	body = strings.ReplaceAll(body, `src="/media/`, `src="`+base+`/media/`)
	return strings.ReplaceAll(body, `href="/media/`, `href="`+base+`/media/`)
}
