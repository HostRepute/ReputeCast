package publicpages

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// Comments under a post (M61, PublicPostComments): pinned first, then
// oldest first, 50 threads a page with their replies; the form only for
// a signed-in member who may comment (a signed-in page is never cached,
// so no form token sits in a cached page). Posting checks, in order: the
// post came from the site itself, the form is readable, the member may
// comment and isn't blocked, the timing token is 3 seconds to a day old,
// the honeypot is empty (else a silent success), the limits, the text.

// commentRootPaths are the comments' own addresses at a custom domain's
// root (the forms post under /posts/*, the account under /account/*).
var commentRootPaths = []string{"/comments/*"}

// commentRoutes mounts the comments' forms and "Stop these emails".
func (h *Handler) commentRoutes(r chi.Router) {
	r.Post("/posts/{slug}/comments", h.commentPost)
	r.Post("/posts/{slug}/comments/{id}/edit", h.commentEdit)
	r.Post("/posts/{slug}/comments/{id}/delete", h.commentDelete)
	r.Get("/comments/mute/{token}", h.commentMutePage)
	r.Head("/comments/mute/{token}", h.headOnly)
	r.Post("/comments/mute/{token}", h.commentMutePost)
	r.Post("/account/comments", h.accountComments)
}

// commentsBlock is the comments section of a post.
type commentsBlock struct {
	Heading, Order string
	Prompt         *commentPrompt
	Form           *commentForm
	Items          []commentView
	More           *commentMore
}

// commentPrompt is the note in place of the form: sign in, upgrade,
// blocked.
type commentPrompt struct {
	Icon, Heading, Text string
	Muted               bool
	Actions             []linkAction
}

// commentMore is "Show more comments".
type commentMore struct{ Text, Href, Label string }

// commentView is one comment.
type commentView struct {
	Anchor, NameID            string
	Name, Initials, Color     string
	Logo                      string
	Team, Mine, Removed       bool
	Badges                    []commentBadge
	ISO, Date                 string
	Body                      template.HTML
	Note                      string
	RemovedLabel, RemovedText string
	ReplyHref, ReplyLabel     string
	EditHref, EditLabel       string
	DeleteAction, DeleteLabel string
	Actions                   commentWords
	Edit                      *commentForm
	Replies                   []commentView
	RepliesLabel              string
	ReplyForm                 *commentForm
}

// commentBadge is a label next to the name: Pinned, Team, Waiting for review.
type commentBadge struct {
	Text, Class string
	Pin         bool
}

// commentWords are the comment's action words.
type commentWords struct{ Reply, Edit, Delete string }

// commentForm is the comment form: a new thread, a reply, an edit.
type commentForm struct {
	Action, Token, Parent          string
	Error, ErrorField              string
	NeedsName                      bool
	Name, NameLabel, NameHint      string
	Label, Body, HPLabel           string
	Who                            template.HTML
	Note, Count, CountLabel        string
	CancelHref, CancelText, Submit string
}

// commentProblem is a comment to show again with what's wrong with it.
type commentProblem struct {
	status         int
	form           string // new, reply, edit
	id             uuid.UUID
	field, message string
	body, name     string
}

type commentProblemKey struct{}

func commentProblemOf(ctx context.Context) *commentProblem {
	p, _ := ctx.Value(commentProblemKey{}).(*commentProblem)
	return p
}

// commentKey is a post's comment forms' timing-token key.
func commentKey(post uuid.UUID) string { return "website_comment:" + post.String() }

// sameOrigin reports whether a form post came from the site's own page:
// the browser's Sec-Fetch-Site, else its Origin (a post with neither isn't
// a browser's form).
func sameOrigin(r *http.Request) bool {
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" {
		return sfs == "same-origin"
	}
	o := r.Header.Get("Origin")
	if o == "" || o == "null" {
		return false
	}
	u, err := url.Parse(o)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host)
}

// commentTime is "9 Oct, 10:42" (with the year when it isn't this one).
func commentTime(v *siteView, t, now time.Time) string {
	if t.Year() != now.Year() {
		return v.labels.Stamp(t)
	}
	if v.labels.Code() == "en" {
		return t.Format("2 Jan, 15:04")
	}
	return v.labels.DayMonth(t) + ", " + v.labels.Lang().Region.Time(t, "")
}

// commentHTML is a comment's text escaped, its links linked (nofollow,
// ugc, noopener) and its line breaks kept.
func commentHTML(body string) template.HTML {
	var b strings.Builder
	for _, p := range website.CommentParts(body) {
		text := strings.ReplaceAll(template.HTMLEscapeString(p.Text), "\n", "<br>")
		if p.Href == "" {
			b.WriteString(text)
			continue
		}
		b.WriteString(`<a href="` + template.HTMLEscapeString(p.Href) + `" rel="nofollow ugc noopener">` + text + `</a>`)
	}
	return template.HTML(b.String()) //nolint:gosec // every part is escaped above
}

// postComments builds a post's comments section for its reader, with the
// published count for its structured data (nil: no comments here).
func (h *Handler) postComments(ctx context.Context, r *http.Request, s *site, v *siteView, p store.WebsitePost, reader gate.Reader) (*commentsBlock, int64, error) {
	q := r.URL.Query()
	cq := website.CommentsQuery{}
	cq.Page, _ = strconv.Atoi(q.Get("comments"))
	cq.Reply, _ = uuid.Parse(q.Get("reply"))
	cq.Edit, _ = uuid.Parse(q.Get("edit"))
	problem := commentProblemOf(ctx)
	if problem != nil {
		switch problem.form {
		case "reply":
			cq.Reply = problem.id
		case "edit":
			cq.Edit = problem.id
		}
	}
	cv, err := h.Website.PostComments(ctx, s.ws, p, v.Session, reader, cq)
	if err != nil || cv == nil {
		return nil, 0, err
	}
	self := s.path("posts", p.Slug)
	publisher := s.set.Branding.PublisherName
	site := v.Frame.Title
	now := h.now().In(location(s.ws))
	b := &commentsBlock{Heading: v.T("comments.heading", "count", cv.Count), Order: v.T("comments.order")}
	switch cv.Access {
	case website.CommentCanSignin:
		pr := &commentPrompt{Icon: "chat", Heading: v.T("comments.signinHeading"), Text: v.T("comments.signinMembers", "site", site)}
		if v.Frame.SigninOn {
			back := website.SafeReturn(s.returnPath(self) + "#comments")
			pr.Actions = append(pr.Actions, linkAction{Label: v.T("site.signIn"), Href: s.path("signin") + "?" + url.Values{"return": {back}}.Encode(), Primary: true})
		}
		switch {
		case cv.PaidOnly:
			pr.Text = v.T("comments.signinPaid", "site", site)
		case cv.SignupOn:
			pr.Text = v.T("comments.signinMembersFree", "site", site)
			pr.Actions = append(pr.Actions, linkAction{Label: v.T("comments.subscribe"), Href: s.links.Subscribe("")})
		}
		b.Prompt = pr
	case website.CommentCanUpgrade:
		b.Prompt = &commentPrompt{Icon: "star", Heading: v.T("comments.paidHeading"), Text: h.commentPrice(ctx, s, v),
			Actions: []linkAction{{Label: v.T("comments.upgrade"), Href: h.upgradeHref(ctx, s, reader), Primary: true}}}
	case website.CommentCanBlocked:
		b.Prompt = &commentPrompt{Icon: "ban", Muted: true, Heading: v.T("comments.blockedHeading", "site", site), Text: v.T("comments.blockedText")}
	}
	canWrite := cv.Access == website.CommentCanWrite
	token := ""
	if canWrite {
		token = h.Forms.FormToken(commentKey(p.ID), h.now())
	}
	newForm := func(label, submit string) *commentForm {
		f := &commentForm{Action: s.path("posts", p.Slug, "comments"), Token: token, Label: label, Submit: submit, HPLabel: v.T("comments.hp"),
			NeedsName: cv.NeedsName, Name: cv.NameDraft, NameLabel: v.T("comments.nameLabel"), NameHint: v.T("comments.nameHint"),
			Count:      v.T("comments.count", "count", v.labels.Count(0), "max", v.labels.Count(website.MaxCommentLength)),
			CountLabel: v.T("comments.count", "count", "{n}", "max", v.labels.Count(website.MaxCommentLength))}
		switch {
		case cv.FirstReviewed:
			f.Note = v.T("comments.firstNote", "publisher", publisher)
		case cv.Review == website.ReviewAll:
			f.Note = v.T("comments.allNote", "publisher", publisher)
		}
		return f
	}
	withProblem := func(f *commentForm, form string, id uuid.UUID) {
		if problem == nil || problem.form != form || problem.id != id {
			return
		}
		f.Error, f.ErrorField, f.Body = problem.message, problem.field, problem.body
		if problem.name != "" {
			f.Name = problem.name
		}
		f.Count = v.T("comments.count", "count", v.labels.Count(int64(len([]rune(problem.body)))), "max", v.labels.Count(website.MaxCommentLength))
	}
	if canWrite && cv.Reply == nil && cv.Edit == nil {
		f := newForm(v.T("comments.addLabel"), v.T("comments.post"))
		if !cv.NeedsName {
			f.Who = v.H("comments.as", "name", cv.DisplayName, "href", s.path("account")+"#comments")
		}
		withProblem(f, "new", uuid.Nil)
		b.Form = f
	}
	nameOf := func(c website.PublicComment) string {
		switch {
		case c.Team:
			return site
		case c.Name != "":
			return c.Name
		}
		return v.T("comments.reader")
	}
	view := func(c website.PublicComment, thread website.PublicComment) commentView {
		cm := commentView{Anchor: "comment-" + c.ID.String(), NameID: "cn-" + c.ID.String(),
			Actions: commentWords{Reply: v.T("comments.reply"), Edit: v.T("comments.edit"), Delete: v.T("comments.delete")}}
		if c.Status == "removed" {
			cm.Removed, cm.RemovedLabel, cm.RemovedText = true, v.T("comments.removedLabel"), v.T("comments.removed")
			return cm
		}
		name := nameOf(c)
		cm.Name, cm.Team, cm.Body = name, c.Team, commentHTML(c.Body)
		cm.Initials, cm.Color = initials(name), avatarColor(name)
		if c.Team {
			cm.Initials, cm.Logo = v.Brand.Initial, v.Brand.Logo
		}
		at := c.CreatedAt.In(location(s.ws))
		cm.ISO, cm.Date = at.Format(time.RFC3339), commentTime(v, at, now)
		if c.Pinned {
			cm.Badges = append(cm.Badges, commentBadge{Text: v.T("comments.pinned"), Class: "mem", Pin: true})
		}
		if c.Team {
			cm.Badges = append(cm.Badges, commentBadge{Text: v.T("comments.team"), Class: "ins"})
		}
		if c.Status == website.CommentPending {
			cm.Mine = true
			cm.Badges = append(cm.Badges, commentBadge{Text: v.T("comments.waiting"), Class: "mem"})
			cm.Note = v.T("comments.pendingNote", "publisher", publisher)
			if c.Editable {
				cm.Note = v.T("comments.pendingNoteEdit", "publisher", publisher)
			}
		}
		if canWrite && thread.Status == website.CommentPublished || canWrite && thread.Status == "removed" && c.Status == website.CommentPublished {
			if c.Status == website.CommentPublished {
				cm.ReplyHref, cm.ReplyLabel = "?reply="+thread.ID.String()+"#comment-form", v.T("comments.replyAria", "name", name)
			}
		}
		if c.Editable {
			cm.EditHref, cm.EditLabel = "?edit="+c.ID.String()+"#comment-form", v.T("comments.editAria")
		}
		if c.Own {
			cm.DeleteAction, cm.DeleteLabel = s.path("posts", p.Slug, "comments", c.ID.String(), "delete"), v.T("comments.deleteAria")
		}
		if cv.Edit != nil && cv.Edit.ID == c.ID {
			f := newForm(v.T("comments.editLabel"), v.T("comments.save"))
			f.Action, f.NeedsName, f.Note = s.path("posts", p.Slug, "comments", c.ID.String(), "edit"), false, v.T("comments.editNote")
			f.Body = c.Body
			f.Count = v.T("comments.count", "count", v.labels.Count(int64(len([]rune(c.Body)))), "max", v.labels.Count(website.MaxCommentLength))
			f.CancelHref, f.CancelText = self+"#"+cm.Anchor, v.T("comments.cancel")
			withProblem(f, "edit", c.ID)
			cm.Edit = f
		}
		return cm
	}
	for _, t := range cv.Threads {
		tv := view(t, t)
		tv.RepliesLabel = v.T("comments.repliesTo", "name", tv.Name)
		if tv.Removed {
			tv.RepliesLabel = v.T("comments.repliesToRemoved")
		}
		for _, rp := range t.Replies {
			tv.Replies = append(tv.Replies, view(rp, t))
		}
		if cv.Reply != nil && cv.Reply.ID == t.ID {
			f := newForm(v.T("comments.replyTo", "name", tv.Name), v.T("comments.postReply"))
			f.Parent = t.ID.String()
			if !cv.NeedsName {
				f.Who = v.H("comments.asShort", "name", cv.DisplayName)
			}
			f.CancelHref, f.CancelText = self+"#comments", v.T("comments.cancel")
			withProblem(f, "reply", t.ID)
			tv.ReplyForm = f
		}
		b.Items = append(b.Items, tv)
	}
	if cv.More {
		b.More = &commentMore{Text: v.T("comments.moreShown", "count", v.labels.Count(int64(cv.Shown))), Label: v.T("comments.more"),
			Href: "?comments=" + strconv.Itoa(cv.Page+1) + "#comments"}
	}
	return b, cv.Count, nil
}

// commentPrice is what it takes to comment on a paid-only site: "Weekly
// Digest Insider is $8 a month or $80 a year, with 14 days free."
func (h *Handler) commentPrice(ctx context.Context, s *site, v *siteView) string {
	_, tiers, err := h.Website.Rules(ctx, s.ws)
	if err != nil || len(tiers) == 0 {
		return v.T("comments.paidText")
	}
	offers, _, err := h.Website.PaywallTiers(ctx, s.ws, tiers)
	if err != nil || len(offers) == 0 {
		return v.T("comments.paidText")
	}
	t := offers[0]
	price := ""
	switch {
	case t.Monthly != "" && t.Yearly != "":
		price = v.T("comments.priceBoth", "monthly", t.Monthly, "yearly", t.Yearly)
	case t.Monthly != "":
		price = v.T("comments.priceMonth", "monthly", t.Monthly)
	case t.Yearly != "":
		price = v.T("comments.priceYear", "yearly", t.Yearly)
	default:
		return v.T("comments.paidText")
	}
	if t.TrialDays > 0 {
		return v.T("comments.paidTierTrial", "tier", t.Name, "price", price, "days", t.TrialDays)
	}
	return v.T("comments.paidTier", "tier", t.Name, "price", price)
}

// commentPostOf is the published post a comment form posts under.
func (h *Handler) commentPostOf(w http.ResponseWriter, r *http.Request, s *site) (store.WebsitePost, bool) {
	addr, err := h.Website.ResolvePost(r.Context(), s.ws, chi.URLParam(r, "slug"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return store.WebsitePost{}, false
	}
	if addr.Post == nil || addr.RedirectTo != "" || addr.Gone || addr.Post.Status != website.PostPublished {
		h.siteNotFound(w, r, s)
		return store.WebsitePost{}, false
	}
	return *addr.Post, true
}

// commentStart runs the checks every comment form shares: the site's own
// page, a readable form, the post, a signed-in member who may comment.
// ok is false when it answered.
func (h *Handler) commentStart(w http.ResponseWriter, r *http.Request, s *site) (store.WebsitePost, *website.Session, gate.Reader, bool) {
	var p store.WebsitePost
	if h.Website == nil {
		h.siteNotFound(w, r, s)
		return p, nil, gate.Reader{}, false
	}
	pl := h.visitorLang(w, r, s, nil)
	if !sameOrigin(r) {
		http.Error(w, website.WordsIn(pl.Labels.Lang()).Text("comments.crossSite", nil), http.StatusForbidden)
		return p, nil, gate.Reader{}, false
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, h.formUnreadable(r, s), http.StatusBadRequest)
		return p, nil, gate.Reader{}, false
	}
	p, ok := h.commentPostOf(w, r, s)
	if !ok {
		return p, nil, gate.Reader{}, false
	}
	sess := h.session(r, s)
	back := website.SafeReturn(s.returnPath(s.path("posts", p.Slug)) + "#comments")
	if sess == nil {
		http.Redirect(w, r, s.path("signin")+"?"+url.Values{"return": {back}}.Encode(), http.StatusSeeOther) //nolint:gosec // G710: the site's own sign-in page
		return p, nil, gate.Reader{}, false
	}
	reader, err := h.Website.Reader(r.Context(), s.ws, sess, "")
	if err != nil {
		h.siteFail(w, r, s, err)
		return p, nil, gate.Reader{}, false
	}
	return p, sess, reader, true
}

// commentBack is a post's address at a comment: its page, its anchor.
func commentBack(s *site, slug string, page int, anchor string) string {
	to := s.path("posts", slug)
	if page > 1 {
		to += "?comments=" + strconv.Itoa(page)
	}
	return to + "#" + anchor
}

// commentAgain shows the post again with the comment and its problem.
func (h *Handler) commentAgain(w http.ResponseWriter, r *http.Request, cp *commentProblem) {
	h.servePost(w, r.WithContext(context.WithValue(r.Context(), commentProblemKey{}, cp)), nil)
}

// commentAllowed checks what a member may do under the post, then the
// timing token and the honeypot. ok is false when it answered.
func (h *Handler) commentAllowed(w http.ResponseWriter, r *http.Request, s *site, p store.WebsitePost, sess *website.Session, reader gate.Reader, cp *commentProblem) bool {
	ctx := r.Context()
	access, err := h.Website.CommentAccess(ctx, s.ws, p, sess, reader)
	if err != nil {
		h.siteFail(w, r, s, err)
		return false
	}
	if access == "" {
		h.siteNotFound(w, r, s)
		return false
	}
	if access != website.CommentCanWrite {
		http.Redirect(w, r, commentBack(s, p.Slug, 1, "comments"), http.StatusSeeOther) //nolint:gosec // G710: the post on this site
		return false
	}
	words := website.WordsIn(h.visitorLang(w, r, s, nil).Labels.Lang())
	age, ok := h.Forms.TokenAge(commentKey(p.ID), r.PostForm.Get("_t"), h.now())
	if !ok || age < 3*time.Second {
		cp.status, cp.field, cp.message = http.StatusUnprocessableEntity, "body", words.Text("comments.expired", nil)
		if ok {
			cp.message = words.Text("comments.tooFast", nil)
		}
		h.commentAgain(w, r, cp)
		return false
	}
	if r.PostForm.Get("website") != "" {
		// A filled honeypot: the same answer as a comment that went, and nothing kept.
		http.Redirect(w, r, commentBack(s, p.Slug, 1, "comments"), http.StatusSeeOther) //nolint:gosec // G710: the post on this site
		return false
	}
	return true
}

// commentDone answers a comment that went: the flash and the comment.
func (h *Handler) commentDone(w http.ResponseWriter, r *http.Request, s *site, p store.WebsitePost, out website.CommentOutcome, cp *commentProblem, posted, held string) {
	switch out.Status {
	case website.CommentPosted, website.CommentHeld:
		kind := posted
		if out.Status == website.CommentHeld {
			kind = held
		}
		h.flash(w, r, s, kind)
		http.Redirect(w, r, commentBack(s, p.Slug, out.Page, "comment-"+out.ID.String()), http.StatusSeeOther) //nolint:gosec // G710: the post on this site
	case website.CommentInvalid, website.CommentLimited:
		cp.status, cp.field, cp.message = http.StatusUnprocessableEntity, out.Field, out.Message
		if out.Status == website.CommentLimited {
			cp.status = http.StatusTooManyRequests
		}
		h.commentAgain(w, r, cp)
	default:
		http.Redirect(w, r, commentBack(s, p.Slug, 1, "comments"), http.StatusSeeOther) //nolint:gosec // G710: the post on this site
	}
}

// commentPost is POST /posts/{slug}/comments: a new thread or a reply.
func (h *Handler) commentPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	p, sess, reader, ok := h.commentStart(w, r, s)
	if !ok {
		return
	}
	parent, _ := uuid.Parse(r.PostForm.Get("parent"))
	cp := &commentProblem{form: "new", body: r.PostForm.Get("body"), name: r.PostForm.Get("name")}
	if parent != uuid.Nil {
		cp.form, cp.id = "reply", parent
	}
	if !h.commentAllowed(w, r, s, p, sess, reader, cp) {
		return
	}
	out, err := h.Website.WriteComment(r.Context(), s.ws, p, sess, reader, website.CommentInput{Body: r.PostForm.Get("body"), Parent: parent,
		Name: r.PostForm.Get("name"), IP: clientIP(r), Lang: h.visitorLang(w, r, s, nil).Labels.Lang()})
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	if out.Status == website.CommentInvalid && out.Field == "parent" {
		cp.form, cp.id = "new", uuid.Nil
	}
	h.commentDone(w, r, s, p, out, cp, "comment_posted", "comment_held")
}

// commentEdit is POST /posts/{slug}/comments/{id}/edit: a member's own
// comment, within 15 minutes.
func (h *Handler) commentEdit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	p, sess, reader, ok := h.commentStart(w, r, s)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.siteNotFound(w, r, s)
		return
	}
	cp := &commentProblem{form: "edit", id: id, body: r.PostForm.Get("body")}
	if !h.commentAllowed(w, r, s, p, sess, reader, cp) {
		return
	}
	out, err := h.Website.EditComment(r.Context(), s.ws, p, sess, reader, id, r.PostForm.Get("body"), h.visitorLang(w, r, s, nil).Labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	h.commentDone(w, r, s, p, out, cp, "comment_edited", "comment_held")
}

// commentDelete is POST /posts/{slug}/comments/{id}/delete: a member's own
// comment, any time.
func (h *Handler) commentDelete(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	p, sess, _, ok := h.commentStart(w, r, s)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.siteNotFound(w, r, s)
		return
	}
	out, err := h.Website.DeleteOwnComment(r.Context(), s.ws, p, sess, id, h.visitorLang(w, r, s, nil).Labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	if out.Status == website.CommentPosted {
		h.flash(w, r, s, "comment_deleted")
	}
	http.Redirect(w, r, commentBack(s, p.Slug, out.Page, "comments"), http.StatusSeeOther) //nolint:gosec // G710: the post on this site
}

// accountComments saves the account page's Comments section.
func (h *Handler) accountComments(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.parseForm(w, r, 16<<10) {
		h.siteNotFound(w, r, s)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, website.WordsIn(h.visitorLang(w, r, s, nil).Labels.Lang()).Text("comments.crossSite", nil), http.StatusForbidden)
		return
	}
	sess := h.session(r, s)
	if sess == nil {
		http.Redirect(w, r, h.signinHref(s, s.path("account")), http.StatusSeeOther) //nolint:gosec // G710: the site's own sign-in page
		return
	}
	if err := h.Website.SaveCommentSettings(r.Context(), s.ws, sess, r.PostForm.Get("display_name"), r.PostForm.Get("emails") == "on"); err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	h.flash(w, r, s, "comment_settings")
	http.Redirect(w, r, s.path("account")+"#comments", http.StatusSeeOther) //nolint:gosec // G710: the site's own account page
}

// commentToast is the flash after a comment form.
func commentToast(v *siteView, kind string) string {
	switch kind {
	case "comment_posted":
		return v.T("comments.flashPosted")
	case "comment_held":
		return v.T("comments.flashHeld")
	case "comment_edited":
		return v.T("comments.flashEdited")
	case "comment_deleted":
		return v.T("comments.flashDeleted")
	case "comment_settings":
		return v.T("comments.flashSettings")
	}
	return ""
}

// muteBody is the "Stop these emails" page.
type muteBody struct {
	Heading, Text, Action, Button, Note string
	Done                                bool
	After                               template.HTML
}

// commentMutePage is GET /comments/mute/{token}: one button; opening the
// page changes nothing, so a link checker can't turn the emails off.
func (h *Handler) commentMutePage(w http.ResponseWriter, r *http.Request) { h.commentMute(w, r, false) }

// commentMutePost is the page's button: reply emails off.
func (h *Handler) commentMutePost(w http.ResponseWriter, r *http.Request) { h.commentMute(w, r, true) }

func (h *Handler) commentMute(w http.ResponseWriter, r *http.Request, mute bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	c, ok := h.claims(s, token, website.CommentMute)
	if !ok || h.Website == nil || !h.siteOn(r.Context(), s) {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	if mute {
		if _, err := h.Website.MuteCommentEmails(ctx, s.ws, c.ContactID); err != nil {
			h.siteFail(w, r, s, err)
			return
		}
	}
	email, on, err := h.Website.CommentEmailsOf(ctx, s.ws, c.ContactID)
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	b := muteBody{Heading: v.T("comments.muteTitle"), Text: v.T("comments.muteText", "email", email, "site", v.Frame.Title),
		Action: s.path("comments", "mute", token), Button: v.T("comments.muteButton"), Note: v.T("comments.muteNote")}
	if !on {
		b = muteBody{Done: true, Heading: v.T("comments.mutedTitle"), Text: v.T("comments.mutedText"),
			After: v.H("comments.mutedAccount", "href", s.path("account")+"#comments")}
	}
	v.Body = b
	v.Title = v.T("site.title", "page", b.Heading, "site", v.Frame.Title)
	v.Robots = "noindex, nofollow"
	v.private, v.secret = true, true
	h.renderSite(w, r, s, "mute", http.StatusOK, v)
}
