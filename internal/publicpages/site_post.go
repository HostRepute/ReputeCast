package publicpages

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// A post of the publication website (PublicPost, PublicSiteStates): the
// reader gets what the gate decides on the server. Text they may not read
// never reaches the page: the preview stops at the marker and the paywall
// card follows. Gated posts are never cached; a visitor's meter is a
// first-party cookie, set only on gated posts.

// tagLinkView is a tag of a post.
type tagLinkView struct{ Name, Href string }

// bylineView is who wrote a post.
type bylineView struct{ Name, Initials, Color string }

// noteView is a note over a post ("This is your free paid post for
// October.").
type noteView struct {
	Icon    string // star, person
	HTML    template.HTML
	Actions []linkAction
	Small   string
}

// linkAction is a button or link of a note.
type linkAction struct {
	Label, Href string
	Primary     bool
}

// paywallTierView is a tier row of the paywall card.
type paywallTierView struct {
	Name    string
	Price   template.HTML
	Href    string
	Button  string
	Primary bool
}

// offerView is a newsletter the sign-up form offers.
type offerView struct {
	ID, Text string
	Checked  bool
}

// signupFormView is the paywall's free sign-up.
type signupFormView struct {
	Heading     string
	Text        string
	Action      string
	ReturnTo    string
	EmailLabel  string
	Placeholder string
	Email       string
	Button      string
	Offers      []offerView
	Note        string
	Error       string
	// Strong: the form is the card's main part (members-only posts).
	Strong bool
}

// paywallView is the paywall card.
type paywallView struct {
	Heading    string
	Lead       string
	Tiers      []paywallTierView
	Free       *signupFormView
	SigninHref string
	Trust      string
	// SignedIn is "Signed in as … · free member" with the account link.
	SignedIn    string
	AccountHref string
	AccountText string
}

// moreView is "More from Engineering".
type moreView struct {
	Title, AllHref, AllText string
	Cards                   []postCardView
}

// bandView is the subscribe band under a post.
type bandView struct {
	Heading, Text, Note string
	Form                subFormView
}

// postBody is the post page.
type postBody struct {
	Tags     []tagLinkView
	Label    website.PostLabel
	Included string
	Title    string
	Dek      string
	Byline   bylineView
	Date     string
	ISO      string
	Minutes  string
	Notes    []noteView
	Cover    *coverView
	HTML     template.HTML
	Paywall  *paywallView
	More     *moreView
	Band     *bandView
}

// sitePost is /posts/{slug}.
func (h *Handler) sitePost(w http.ResponseWriter, r *http.Request) {
	h.servePost(w, r, nil)
}

// servePost answers a post's address; problem is a sign-up error to show
// on the paywall.
func (h *Handler) servePost(w http.ResponseWriter, r *http.Request, problem *signupProblem) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	addr, err := h.Website.ResolvePost(ctx, s.ws, chi.URLParam(r, "slug"))
	switch {
	case err != nil:
		h.siteFail(w, r, s, err)
		return
	case addr.RedirectTo != "":
		to := s.sitePath(addr.RedirectTo)
		if q := r.URL.RawQuery; q != "" {
			to += "?" + q
		}
		http.Redirect(w, r, to, http.StatusMovedPermanently) //nolint:gosec // G710: a path on this site, from the post's own address
		return
	case addr.Gone:
		h.siteGone(w, r, s)
		return
	case addr.Post == nil:
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	p := *addr.Post
	gated := p.Visibility != gate.Public
	meter := cookieValue(r, website.MeterCookie)
	if gated && v.Session == nil && website.BrowserKey(meter) == "" {
		// The visitor meter counts per browser: its cookie starts here.
		meter = website.Token()
		h.setCookie(w, r, s, website.MeterCookie, meter, 400*24*time.Hour)
	}
	reader, err := h.Website.Reader(ctx, s.ws, v.Session, meter)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	pv, err := h.Website.ReadPost(ctx, s.ws, p, reader, website.ReadOptions{Lang: v.labels.Lang()})
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	body, err := h.postBody(ctx, s, v, p, pv, reader, problem)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v.Body = body
	if gated {
		// What a gated post shows depends on the reader: never cached.
		v.private = true
	}
	h.postMeta(s, v, p, pv)
	if problem == nil && r.Method == http.MethodGet {
		h.Website.CountView(ctx, s.ws, r, website.PagePost, p.ID)
	}
	status := http.StatusOK
	if problem != nil {
		status = http.StatusUnprocessableEntity
	}
	h.renderSite(w, r, s, "post", status, v)
}

// postMeta sets a post's title, description, canonical address, social
// tags and structured data (a gated post says it isn't free).
func (h *Handler) postMeta(s *site, v *siteView, p store.WebsitePost, pv website.PostView) {
	title := pv.Post.Title
	if t := strings.TrimSpace(pv.SEO.Title); t != "" {
		title = t
	}
	v.Title = v.T("site.title", "page", title, "site", v.Frame.Title)
	v.Description = pv.Post.Excerpt
	if d := strings.TrimSpace(pv.SEO.Description); d != "" {
		v.Description = d
	}
	v.Canonical = s.canonPath("posts", p.Slug)
	if c := strings.TrimSpace(pv.SEO.CanonicalURL); c != "" {
		v.Canonical = c
	}
	if pv.SEO.HideFromSearch || !v.Frame.Index {
		v.Robots = "noindex, follow"
	}
	image := h.mediaURL(pv.SEO.SocialImage)
	if image == "" {
		image = h.mediaURL(pv.Post.CoverURL)
	}
	published := pv.Post.PublishedAt.UTC().Format(time.RFC3339)
	v.OG = &og{Type: "article", SiteName: v.Frame.Title, Title: title, Description: v.Description, URL: v.Canonical, Image: image, Published: published}
	v.Feeds = []siteFeed{{Title: v.Frame.Title, Href: s.canonPath("feed")}}
	ld := map[string]any{"@context": "https://schema.org", "@type": "NewsArticle", "headline": pv.Post.Title, "description": v.Description,
		"datePublished": published, "dateModified": pv.Post.UpdatedAt.UTC().Format(time.RFC3339), "mainEntityOfPage": v.Canonical,
		"publisher": map[string]any{"@type": "Organization", "name": s.set.Branding.PublisherName}}
	if pv.Post.Byline != "" {
		ld["author"] = map[string]any{"@type": "Person", "name": pv.Post.Byline}
	}
	if image != "" {
		ld["image"] = []string{image}
	}
	if pv.Post.Visibility.Level != gate.Public {
		// Flexible sampling: the paywalled part is marked, never sent.
		ld["isAccessibleForFree"] = false
		ld["hasPart"] = map[string]any{"@type": "WebPageElement", "isAccessibleForFree": false, "cssSelector": ".st-pw"}
	}
	v.JSONLD = append(v.JSONLD, ld)
}

// postBody builds the post page.
func (h *Handler) postBody(ctx context.Context, s *site, v *siteView, p store.WebsitePost, pv website.PostView, reader gate.Reader, problem *signupProblem) (postBody, error) {
	loc := location(s.ws)
	at := pv.Post.PublishedAt.In(loc)
	b := postBody{Label: pv.Post.Label, Title: pv.Post.Title, Dek: pv.Post.Excerpt, Date: v.labels.FullDate(at), ISO: pv.Post.PublishedAt.UTC().Format(time.RFC3339),
		Minutes: v.T("site.readMin", "count", pv.Minutes), HTML: template.HTML(h.mediaHTML(pv.Body))} //nolint:gosec // the renderer's sanitized HTML of the publisher's own post
	for _, t := range pv.Post.Tags {
		// A tag that only repeats the paid label ("Insider", next to
		// "☆ Insider") is left to the label.
		if b.Label.Text != "" && strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(b.Label.Text)) {
			continue
		}
		b.Tags = append(b.Tags, tagLinkView{Name: t.Name, Href: s.path("tag", t.Slug)})
	}
	name := pv.Post.Byline
	if name == "" {
		name = s.set.Branding.PublisherName
	}
	b.Byline = bylineView{Name: name, Initials: initials(name), Color: avatarColor(name)}
	if pv.Post.CoverURL != "" {
		b.Cover = &coverView{Src: h.mediaURL(pv.Post.CoverURL), Alt: pv.Post.CoverAlt, Width: 1000, Height: 563}
	}
	d := pv.Decision
	month := monthName(v, pv.Month)
	switch {
	case d.Reason == gate.ReasonTier && d.Tier != nil:
		b.Included = v.T("gate.memberIncluded", "tier", d.Tier.Label)
	case (d.Reason == gate.ReasonMetered || d.Reason == gate.ReasonReread) && d.Rule == gate.RuleMember:
		b.Notes = append(b.Notes, noteView{Icon: "star", HTML: template.HTML("<b>" + template.HTMLEscapeString(v.T("gate.freePost", "month", month)) + "</b> " + //nolint:gosec // escaped words
			template.HTMLEscapeString(v.T("gate.freePostUpgrade"))), Actions: []linkAction{{Label: v.T("site.upgrade"), Href: h.upgradeHref(ctx, s, reader), Primary: true}},
			Small: v.T("gate.freePostRule", "count", d.Limit)})
	case (d.Reason == gate.ReasonMetered || d.Reason == gate.ReasonReread) && d.Rule == gate.RuleVisitor:
		n := noteView{Icon: "person", HTML: v.H("gate.visitorMeter", "used", d.Used, "limit", d.Limit)}
		if pv.SignupOn {
			n.Actions = append(n.Actions, linkAction{Label: v.T("site.signupFree"), Href: s.links.Subscribe(""), Primary: true})
		}
		n.Actions = append(n.Actions, linkAction{Label: v.T("site.signIn"), Href: h.signinHref(s, s.path("posts", p.Slug))})
		b.Notes = append(b.Notes, n)
	}
	if !d.Readable() {
		pw, err := h.paywall(ctx, s, v, p, pv, reader, problem)
		if err != nil {
			return b, err
		}
		b.Paywall = pw
	}
	if pv.MoreTag != nil && len(pv.More) > 0 {
		m := &moreView{Title: v.T("site.moreFrom", "tag", pv.MoreTag.Name), AllHref: s.path("tag", pv.MoreTag.Slug), AllText: v.T("site.allPosts", "count", pv.MoreTotal)}
		for _, mp := range pv.More {
			m.Cards = append(m.Cards, h.card(s, v, mp, nil, "card"))
		}
		b.More = m
	}
	if v.Member == nil && !v.Preview {
		sp := s.set.Pages.Subscribe
		text := pv.BandText
		if text == "" {
			text = sp.Description
		}
		b.Band = &bandView{Heading: v.T("site.nextIssue"), Text: text, Note: sp.Footnote,
			Form: subFormView{Class: "", Action: s.links.Subscribe(""), Page: s.canonPath("posts", p.Slug), ID: "band-email", EmailLabel: v.T("site.emailLabel"),
				Placeholder: v.T("site.emailPlaceholder"), Button: v.T("site.subscribe")}}
	}
	return b, nil
}

// monthName names the meter's month ("October").
func monthName(v *siteView, month time.Time) string {
	if v.labels.Code() == "en" {
		return month.Format("January")
	}
	return v.labels.Lang().Region.Date(month)
}

// nextName names the first day of the next month ("1 November").
func nextName(v *siteView, t time.Time) string {
	if v.labels.Code() == "en" {
		return t.Format("2 January")
	}
	return v.labels.NextDate(t)
}

// upgradeHref is the upgrade page: a member's personal link (the page
// knows them), else the page itself.
func (h *Handler) upgradeHref(ctx context.Context, s *site, reader gate.Reader) string {
	if h.Earn != nil && reader.Member != nil && reader.Member.ContactID != uuid.Nil {
		return h.Earn.UpgradeURL(ctx, s.ws, reader.Member.ContactID, uuid.Nil)
	}
	return s.path("upgrade")
}

// signinHref is the sign-in page, back to path afterwards.
func (h *Handler) signinHref(s *site, back string) string {
	return s.path("signin") + "?" + url.Values{"return": {s.returnPath(back)}}.Encode()
}

// returnPath is a page's path on the site, kept for after signing in:
// "/posts/x" on a custom domain, "/p/acme/posts/x" on the instance.
func (s *site) returnPath(href string) string {
	if u, err := url.Parse(href); err == nil {
		p := u.EscapedPath()
		if u.RawQuery != "" {
			p += "?" + u.RawQuery
		}
		return p
	}
	return "/"
}

// paywall builds the paywall card of a post the reader can't read whole.
func (h *Handler) paywall(ctx context.Context, s *site, v *siteView, p store.WebsitePost, pv website.PostView, reader gate.Reader, problem *signupProblem) (*paywallView, error) {
	d := pv.Decision
	self := s.path("posts", p.Slug)
	pw := &paywallView{}
	if v.Member == nil && v.Frame.SigninOn {
		pw.SigninHref = h.signinHref(s, self)
	}
	month := monthName(v, pv.Month)
	free := func(strong bool, heading, text string) {
		if !pv.SignupOn || v.Member != nil {
			return
		}
		f := &signupFormView{Heading: heading, Text: text, Action: self, ReturnTo: s.returnPath(self), EmailLabel: v.T("site.emailLabel"),
			Placeholder: v.T("site.emailPlaceholder"), Button: pv.Paywall.Signup, Note: v.T("gate.confirmNote"), Strong: strong}
		if f.Button == "" || f.Button == website.DefaultPaywall().Signup {
			f.Button = v.T("site.signupFree")
		}
		for _, o := range pv.Offers {
			ov := offerView{ID: o.ID.String(), Text: o.Consent, Checked: o.Checked}
			if problem != nil {
				ov.Checked = problem.checked[o.ID.String()]
			}
			f.Offers = append(f.Offers, ov)
		}
		if problem != nil {
			f.Email, f.Error = problem.email, problem.message
		}
		pw.Free = f
	}
	switch d.Reason {
	case gate.ReasonSigninOff:
		pw.Heading, pw.Lead = v.T("gate.signinOffHeading"), v.T("gate.signinOffText")
		pw.SigninHref = ""
		return pw, nil
	case gate.ReasonSignin:
		pw.Heading = v.T("gate.signupHeading")
		free(true, "", v.T("gate.signupText"))
		if pw.Free == nil {
			pw.Heading, pw.Lead = v.T("gate.signinOffHeading"), v.T("gate.signinOffText")
		}
		return pw, nil
	case gate.ReasonSignupMeter:
		pw.Heading = v.T("gate.signupHeading")
		free(true, "", v.T("gate.signupMeterUsed", "count", d.Limit, "month", month))
		if pw.Free == nil {
			pw.Heading, pw.Lead = v.T("gate.signinOffHeading"), v.T("gate.signinOffText")
		}
		return pw, nil
	}
	// A paid post: the tiers that read it.
	pw.Heading = v.T("gate.headingPaid")
	if len(pv.Readers) > 0 {
		pw.Heading = v.T("gate.heading", "tier", pv.Readers[0].Label)
	}
	if hd := strings.TrimSpace(pv.Paywall.Heading); hd != "" && hd != website.DefaultPaywall().Heading {
		pw.Heading = hd
	}
	pw.Lead = strings.TrimSpace(pv.Paywall.Text)
	if d.Reason == gate.ReasonUpgradeMeter {
		pw.Lead = v.T("gate.meterUsed", "count", d.Limit, "month", month, "next", nextName(v, pv.Next))
	}
	tiers, pay, err := h.Website.PaywallTiers(ctx, s.ws, pv.Readers)
	if err != nil {
		return nil, err
	}
	button := pv.Paywall.Upgrade
	if button == "" {
		button = v.T("site.upgrade")
	}
	up := h.upgradeHref(ctx, s, reader)
	for i, t := range tiers {
		tv := paywallTierView{Name: t.Name, Href: up, Button: button, Primary: i == 0}
		tv.Price = tierPrice(v, t)
		pw.Tiers = append(pw.Tiers, tv)
	}
	if pay.Provider != "" && len(tiers) > 0 {
		pw.Trust = v.T("gate.paymentsVia", "publisher", s.set.Branding.PublisherName, "provider", pay.Provider)
	}
	if d.FreeRow {
		text := pv.Paywall.Free
		if text == "" || text == website.DefaultPaywall().Free {
			text = v.T("gate.freeRow", "count", pv.Rules.Members)
		}
		free(false, text, "")
	}
	if v.Member != nil {
		pw.SignedIn = v.T("gate.signedInAs", "email", v.Member.Email)
		pw.AccountHref, pw.AccountText = s.path("account"), v.T("gate.yourAccount")
	}
	return pw, nil
}

// tierPrice is "<b>$8</b> a month or <b>$80</b> a year · 14 days free".
func tierPrice(v *siteView, t website.PaywallTier) template.HTML {
	esc := template.HTMLEscapeString
	var parts []string
	if t.Monthly != "" {
		parts = append(parts, "<b>"+esc(t.Monthly)+"</b> "+esc(v.T("gate.perMonth")))
	}
	if t.Yearly != "" {
		parts = append(parts, "<b>"+esc(t.Yearly)+"</b> "+esc(v.T("gate.perYear")))
	}
	out := strings.Join(parts, " "+esc(v.T("gate.or"))+" ")
	var extra []string
	if t.TrialDays > 0 {
		extra = append(extra, esc(v.T("site.trialDays", "count", t.TrialDays)))
	}
	if t.Includes != "" {
		extra = append(extra, esc(v.T("gate.includes", "tier", t.Includes)))
	}
	if len(extra) > 0 {
		out += " · " + strings.Join(extra, " · ")
	}
	return template.HTML(out) //nolint:gosec // every value is escaped above
}

// signupProblem is a paywall sign-up to show again with its error.
type signupProblem struct {
	email, message string
	checked        map[string]bool
}

// sitePostSignup is the paywall's free sign-up (POST /posts/{slug}): one
// email confirms the newsletters ticked and signs the reader in.
func (h *Handler) sitePostSignup(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.parseForm(w, r, 16<<10) {
		h.notFound(w, r, s)
		return
	}
	slug := chi.URLParam(r, "slug")
	h.join(w, r, s, s.path("posts", slug), func(problem *signupProblem) { h.servePost(w, r, problem) })
}

// join signs a reader up from a page of the site and answers "check your
// email" (the same for every address). retry shows the page again with a
// problem.
func (h *Handler) join(w http.ResponseWriter, r *http.Request, s *site, page string, retry func(*signupProblem)) {
	ctx := r.Context()
	var lists []uuid.UUID
	checked := map[string]bool{}
	for _, raw := range r.PostForm["list"] {
		if id, err := uuid.Parse(raw); err == nil {
			lists = append(lists, id)
			checked[raw] = true
		}
	}
	browser := h.browserCookie(w, r, s)
	ip := clientIP(r)
	remote := ""
	if ip != nil {
		remote = ip.String()
	}
	pl := h.visitorLang(w, r, s, nil)
	out, err := h.Website.Join(ctx, s.ws, website.JoinInput{Email: r.PostForm.Get("email"), Lists: lists, ReturnTo: s.returnPath(page), Browser: browser, IP: ip,
		UserAgent: r.UserAgent(), Turnstile: r.PostForm.Get("cf-turnstile-response"), Verify: h.Forms.TurnstileVerifier(ctx, s.set, remote),
		Page: strings.TrimPrefix(strings.TrimPrefix(s.canonPath(), "https://"), "http://") + strings.TrimPrefix(s.returnPath(page), "/"), Language: pl.Code})
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	switch out.Status {
	case website.SigninInvalid:
		retry(&signupProblem{email: r.PostForm.Get("email"), message: out.Message, checked: checked})
	case website.SigninOff:
		h.siteNotFound(w, r, s)
	case website.SigninLimited:
		h.signinLimited(w, r, s, out, page)
	case website.SigninChallenge:
		h.signinChallenge(w, r, s, page, r.PostForm.Get("email"), lists)
	default:
		h.joinSent(w, r, s, out.Email, page)
	}
}
