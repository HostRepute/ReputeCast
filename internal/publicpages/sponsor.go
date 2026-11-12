package publicpages

// The sponsor pages (M65, G-40): the media kit with the self-serve slots
// and the issues still open, the request form, and a request's status page
// with Pay. Their words come from internal/sponsors, in the page's
// language; the branding is the publisher's.
//
//	/p/{workspace}/sponsor                 the media kit (indexable)
//	/p/{workspace}/sponsor/request         the request form; POST sends it (multipart, ≤ 3 MB)
//	/p/{workspace}/sponsor/b/{token}       a request's status page (noindex)
//	/p/{workspace}/sponsor/b/{token}/pay   POST: Stripe Checkout for an approved request
//
// A request is checked like the other public forms: it must come from the
// page itself, bots fill the hidden field, Turnstile answers when the
// workspace set it up, and an address sends at most five an hour.

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sponsors"
)

// sponsorRoutes adds the sponsor pages.
func (h *Handler) sponsorRoutes(r chi.Router) {
	r.Get("/sponsor", h.sponsorKit)
	r.Head("/sponsor", h.headOnly)
	r.Get("/sponsor/request", h.sponsorRequestPage)
	r.Head("/sponsor/request", h.headOnly)
	r.Post("/sponsor/request", h.sponsorRequestPost)
	r.Get("/sponsor/b/{token}", h.sponsorStatus)
	r.Head("/sponsor/b/{token}", h.headOnly)
	r.Post("/sponsor/b/{token}/pay", h.sponsorPay)
}

// sponsorKitBody is the media kit.
type sponsorKitBody struct {
	// State is kit, off (self-serve booking off) or full (nothing to book).
	State       string
	Title       string
	Intro       string
	ArchiveHref string
	ArchiveText string
	Stats       []sponsorStat
	StatsTitle  string
	StatsNote   string
	About       string
	Audience    string
	Words       map[string]string
	Slots       []sponsorSlotView
	RequestHref string
	Questions   template.HTML
	PriceNote   string
	OffTitle    string
	OffText     template.HTML
	Test        bool
	TermsURL    string
	TermsLabel  string
}

type sponsorStat struct{ Label, Value, Sub string }

// sponsorSlotView is a self-serve slot on the media kit.
type sponsorSlotView struct {
	Name, Price, Description, Terms string
	Issues                          []sponsorIssueView
}

type sponsorIssueView struct{ Date, Line, Href, Label string }

// sponsorRequestBody is the request form, or its done page.
type sponsorRequestBody struct {
	// State is form or done.
	State       string
	Action      string
	Title       string
	Intro       string
	Words       map[string]string
	ErrorsTitle string
	Gone        *earnNoticeView
	Limit       *earnNoticeView
	Slots       []sponsorOption
	Issues      []sponsorOption
	IssueNote   string
	BlockNote   string
	NotesLabel  string
	In          sponsorRequestIn
	Errors      map[string]string
	Logo        *sponsorFileView
	Image       *sponsorFileView
	TextLeft    string
	NotesLeft   string
	LeftOne     string
	LeftOther   string
	SiteKey     string
	TermsURL    string
	TermsLabel  string
	// The done page.
	DoneText  string
	DoneLine  string
	DonePrice string
	DoneWho   string
	DonePay   string
	Follow    string
	Back      string
}

type earnNoticeView struct{ Title, Text string }

type sponsorOption struct {
	Value, Label string
	Selected     bool
	// Issues are a slot's open issues, for the form's script.
	Issues string
}

// sponsorRequestIn is the form as sent.
type sponsorRequestIn struct {
	Company, Website, Name, Email, Headline, Text, Button, Link, Notes string
}

// sponsorFileView is an image the form keeps after errors.
type sponsorFileView struct {
	ID, Name string
	Bad      bool
}

// sponsorStatusBody is a request's status page.
type sponsorStatusBody struct {
	State      string
	Pill       string
	PillTone   string
	Heading    string
	Line       string
	Text       string
	Quote      string
	QuoteBy    string
	Facts      []sponsorFact
	PayFor     string
	Price      string
	Once       string
	PayAction  string
	PayButton  string
	PayNote    string
	Notice     *earnNoticeView
	Change     template.HTML
	BlockTitle string
	Block      *sponsorBlockView
	Another    string
	AnotherTxt string
	Charged    string
	Questions  template.HTML
	Private    string
	Test       bool
	TestText   string
	TermsURL   string
	TermsLabel string
}

type sponsorFact struct{ Label, Value string }

// sponsorBlockView is the sponsor block as the issue shows it.
type sponsorBlockView struct {
	Initials, Logo, Image, Label, Headline, Text, Button string
}

func (h *Handler) sponsorsOK(w http.ResponseWriter, r *http.Request) (*site, bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return nil, false
	}
	if h.Sponsors == nil {
		h.notFound(w, r, s)
		return nil, false
	}
	return s, true
}

// sponsorWords are a sponsor page's words in its language.
func sponsorWords(l i18n.Lang, keys ...string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		out[k[strings.LastIndex(k, ".")+1:]] = sponsors.TextIn(l, k, nil)
	}
	return out
}

func (h *Handler) sponsorKit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.sponsorsOK(w, r)
	if !ok {
		return
	}
	pl, r := h.earnLang(w, r, s)
	l := pl.Labels.Lang()
	ctx := r.Context()
	pub := s.ws.Name
	page, err := h.Sponsors.Page(ctx, s.ws)
	b := sponsorKitBody{State: "kit", Title: sponsors.TextIn(l, "kit.title", i18n.Args{"publisher": pub}), ArchiveHref: s.links.Archive(),
		ArchiveText: sponsors.TextIn(l, "kit.archive", nil), TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r)}
	status := http.StatusOK
	switch {
	case errors.Is(err, sponsors.ErrPageOff):
		contact := h.Sponsors.ContactOf(ctx, s.ws)
		b.State, b.OffTitle = "off", sponsors.TextIn(l, "kit.offTitle", i18n.Args{"publisher": pub})
		b.OffText = mailtoHTML(sponsors.TextIn(l, "kit.offText", i18n.Args{"email": contact}), contact)
		if contact == "" {
			b.OffText = template.HTML(template.HTMLEscapeString(sponsors.TextIn(l, "kit.offTextNoEmail", nil))) //nolint:gosec // escaped
		}
		status = http.StatusNotFound
	case err != nil:
		h.serverError(w, r, s, err)
		return
	default:
		h.fillKit(&b, page, s, l, pub)
	}
	vw := h.earnView(r, s, h.earnTitle(s, b.Title), b, pl)
	vw.Description = b.Intro
	vw.Canonical = earnLink(s.canon, "sponsor")
	if b.State == "kit" {
		vw.Robots = "index, follow"
	}
	h.render(w, r, s, "sponsorkit", status, vw, csp{}, false)
}

// fillKit writes the media kit's numbers, texts and slots.
func (h *Handler) fillKit(b *sponsorKitBody, page sponsors.SponsorPage, s *site, l i18n.Lang, pub string) {
	b.Intro, b.Test = sponsors.TextIn(l, "kit.intro", i18n.Args{"publisher": pub}), page.Test
	b.Words = sponsorWords(l, "kit.about", "kit.audience", "kit.book", "kit.perIssue", "kit.available", "kit.request", "kit.test")
	b.Words["availableNote"] = sponsors.TextIn(l, "kit.availableNote", i18n.Args{"publisher": pub})
	b.About, b.Audience = page.Kit.About, page.Kit.Audience
	if len(page.Lists) > 0 {
		st := page.Lists[0]
		a := i18n.Args{"list": st.ListName}
		b.StatsTitle, b.StatsNote = sponsors.TextIn(l, "kit.numbers", a), sponsors.TextIn(l, "kit.numbersNote", a)
		if page.Kit.Subscribers {
			b.Stats = append(b.Stats, sponsorStat{Label: sponsors.TextIn(l, "kit.subscribers", nil), Value: l.Region.Int(st.Subscribers),
				Sub: sponsors.TextIn(l, "kit.subscribersSub", a)})
		}
		if page.Kit.OpenRate && st.OpenRate != nil {
			b.Stats = append(b.Stats, sponsorStat{Label: sponsors.TextIn(l, "kit.openRate", nil), Value: l.Region.Decimal(*st.OpenRate, 1) + "%",
				Sub: sponsors.TextIn(l, "kit.openRateSub", nil)})
		}
		if page.Kit.ClickRate && st.ClickRate != nil {
			b.Stats = append(b.Stats, sponsorStat{Label: sponsors.TextIn(l, "kit.clickRate", nil), Value: l.Region.Decimal(*st.ClickRate, 1) + "%",
				Sub: sponsors.TextIn(l, "kit.clickRateSub", nil)})
		}
	}
	open, currency := 0, ""
	for _, sl := range page.Slots {
		price := sponsors.Money(sl.PriceCents, sl.Currency)
		currency = sl.Currency
		v := sponsorSlotView{Name: sl.Name, Price: price, Description: sl.Description,
			Terms: sponsors.TextIn(l, "kit.terms", i18n.Args{"list": sl.ListName, "every": sponsors.Every(l, sl.EveryDays, sl.Issues), "lead": sl.LeadDays, "pay": sl.PayDays})}
		for _, is := range sl.Issues {
			q := url.Values{}
			q.Set("slot", sl.ID.String())
			q.Set("issue", strconv.Itoa(is.Number))
			date := sponsors.IssueDate(l, is.On, page.Now)
			v.Issues = append(v.Issues, sponsorIssueView{Date: date, Line: sponsors.TextIn(l, "kit.issuePrice", i18n.Args{"issue": is.Number, "price": price}),
				Href: earnLink(s.links, "sponsor/request") + "?" + q.Encode(), Label: sponsors.TextIn(l, "kit.requestIssue", i18n.Args{"issue": is.Number, "date": date})})
		}
		open += len(v.Issues)
		b.Slots = append(b.Slots, v)
	}
	b.RequestHref = earnLink(s.links, "sponsor/request")
	b.Questions = mailtoHTML(sponsors.TextIn(l, "kit.questions", i18n.Args{"email": page.Contact}), page.Contact)
	if page.Contact == "" {
		b.Questions = ""
	}
	if currency != "" {
		b.PriceNote = sponsors.TextIn(l, "kit.priceNote", i18n.Args{"currency": sponsors.CurrencyName(l, currency)})
	}
	if open == 0 {
		b.State, b.OffTitle = "full", sponsors.TextIn(l, "kit.fullTitle", nil)
		b.OffText = mailtoHTML(sponsors.TextIn(l, "kit.fullText", i18n.Args{"email": page.Contact}), page.Contact)
		if page.Contact == "" {
			b.OffText = template.HTML(template.HTMLEscapeString(sponsors.TextIn(l, "kit.fullTextNoEmail", nil))) //nolint:gosec // escaped
		}
	}
}

func (h *Handler) sponsorRequestPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.sponsorsOK(w, r)
	if !ok {
		return
	}
	pl, r := h.earnLang(w, r, s)
	q := r.URL.Query()
	h.sponsorForm(w, r, s, pl, http.StatusOK, q.Get("slot"), q.Get("issue"), sponsorRequestIn{}, sponsors.RequestOutcome{})
}

// sponsorForm renders the request form: empty, or as sent with errors.
func (h *Handler) sponsorForm(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, status int, slot, issue string, in sponsorRequestIn, out sponsors.RequestOutcome) {
	h.sponsorFormWith(w, r, s, pl, status, slot, issue, in, out, nil)
}

// sponsorFormWith is sponsorForm with a notice over it (too many requests).
func (h *Handler) sponsorFormWith(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, status int, slot, issue string, in sponsorRequestIn,
	out sponsors.RequestOutcome, limit *earnNoticeView) {
	ctx := r.Context()
	l := pl.Labels.Lang()
	pub := s.ws.Name
	page, err := h.Sponsors.Page(ctx, s.ws)
	if errors.Is(err, sponsors.ErrPageOff) {
		h.sponsorKit(w, r)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	b := sponsorRequestBody{State: "form", Action: earnLink(s.links, "sponsor/request"), Title: sponsors.TextIn(l, "req.title", nil),
		Intro: sponsors.TextIn(l, "req.intro", i18n.Args{"publisher": pub}), In: in, Errors: out.Errors, TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r), Limit: limit,
		Words: sponsorWords(l, "req.issueSection", "req.slot", "req.issue", "req.chooseIssue", "req.aboutYou", "req.company", "req.website", "req.optional",
			"req.name", "req.email", "req.emailNote", "req.block", "req.logo", "req.noFile", "req.logoNote", "req.chooseFile", "req.replace", "req.image",
			"req.imageNote", "req.headline", "req.headlineNote", "req.text", "req.button", "req.link", "req.send", "req.sendNote", "req.errorsText",
			"req.websitePlaceholder", "req.emailPlaceholder", "req.textPlaceholder", "req.buttonPlaceholder", "req.notesPlaceholder")}
	if b.Errors == nil {
		b.Errors = map[string]string{}
	}
	b.NotesLabel = sponsors.TextIn(l, "req.notes", i18n.Args{"publisher": pub})
	if n := len(b.Errors); n > 0 && out.Gone == 0 {
		b.ErrorsTitle = sponsors.TextIn(l, "req.errorsTitle", i18n.Args{"count": n})
	}
	if out.Gone > 0 {
		b.Gone = &earnNoticeView{Title: sponsors.TextIn(l, "req.goneTitle", i18n.Args{"issue": out.Gone}), Text: sponsors.TextIn(l, "req.goneText", nil)}
	}
	var chosen *sponsors.PageSlot
	for i, sl := range page.Slots {
		if sl.ID.String() == slot || (slot == "" && i == 0) {
			chosen = &page.Slots[i]
		}
	}
	if chosen == nil && len(page.Slots) > 0 {
		chosen = &page.Slots[0]
	}
	for _, sl := range page.Slots {
		opts := []map[string]string{}
		for _, is := range sl.Issues {
			opts = append(opts, map[string]string{"value": strconv.Itoa(is.Number), "label": sponsors.TextIn(l, "req.issueOption",
				i18n.Args{"issue": is.Number, "date": sponsors.IssueDate(l, is.On, page.Now)})})
		}
		raw, _ := json.Marshal(opts)
		b.Slots = append(b.Slots, sponsorOption{Value: sl.ID.String(), Label: sponsors.TextIn(l, "req.slotOption",
			i18n.Args{"slot": sl.Name, "price": sponsors.Money(sl.PriceCents, sl.Currency)}), Selected: chosen != nil && sl.ID == chosen.ID, Issues: string(raw)})
	}
	if chosen != nil {
		for _, is := range chosen.Issues {
			v := strconv.Itoa(is.Number)
			b.Issues = append(b.Issues, sponsorOption{Value: v, Label: sponsors.TextIn(l, "req.issueOption",
				i18n.Args{"issue": is.Number, "date": sponsors.IssueDate(l, is.On, page.Now)}), Selected: v == issue && out.Gone == 0})
		}
		b.IssueNote = sponsors.TextIn(l, "req.issueNote", i18n.Args{"lead": chosen.LeadDays})
		if p := strings.TrimSpace(chosen.Placement); p != "" {
			b.BlockNote = sponsors.TextIn(l, "req.blockNote", i18n.Args{"publisher": pub, "placement": sponsors.PlacementInSentence(l, p)})
		} else {
			b.BlockNote = sponsors.TextIn(l, "req.blockNoteNoPlace", i18n.Args{"publisher": pub})
		}
	}
	for field, f := range out.Kept {
		v := &sponsorFileView{ID: f.ID, Name: sponsors.TextIn(l, "req.fileSize", i18n.Args{"name": f.Name, "size": fileSize(l, f.Size)})}
		if field == "logo" {
			b.Logo = v
		} else {
			b.Image = v
		}
	}
	left := func(max int, text string) string {
		return sponsors.TextIn(l, "req.left", i18n.Args{"count": max - len([]rune(text))})
	}
	b.TextLeft, b.NotesLeft = left(280, in.Text), left(600, in.Notes)
	b.LeftOne = strings.Replace(sponsors.TextIn(l, "req.left", i18n.Args{"count": 1}), "1", "#", 1)
	b.LeftOther = strings.Replace(sponsors.TextIn(l, "req.left", i18n.Args{"count": 7}), "7", "#", 1)
	b.SiteKey = s.set.TurnstileSiteKey()
	vw := h.earnView(r, s, h.earnTitle(s, b.Title), b, pl)
	vw.Description = b.Intro
	vw.Canonical = earnLink(s.canon, "sponsor/request")
	vw.Robots = "noindex, follow"
	vw.Scripts = []string{s.links.Static("sponsor.js?v=" + h.version)}
	policy := csp{}
	if b.SiteKey != "" {
		vw.Scripts = append(vw.Scripts, forms.TurnstileScript)
		policy.scripts, policy.frames = []string{forms.TurnstileOrigin}, []string{forms.TurnstileOrigin}
	}
	h.render(w, r, s, "sponsorrequest", status, vw, policy, false)
}

// fileSize is "18 KB" or "3.4 MB".
func fileSize(l i18n.Lang, n int) string {
	if n < 1<<20 {
		return l.Region.Int(int64((n+1023)/1024)) + " KB"
	}
	return l.Region.Decimal(float64(n)/(1<<20), 1) + " MB"
}

func (h *Handler) sponsorRequestPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.sponsorsOK(w, r)
	if !ok {
		return
	}
	if !h.samePage(w, r) {
		return
	}
	if !h.parseForm(w, r, sponsors.MaxRequestBody) {
		http.Error(w, h.formUnreadable(r, s), http.StatusRequestEntityTooLarge)
		return
	}
	pl, r := h.earnLang(w, r, s)
	l := pl.Labels.Lang()
	ctx := r.Context()
	f := r.PostForm
	in := sponsorRequestIn{Company: f.Get("company"), Website: f.Get("website_url"), Name: f.Get("name"), Email: f.Get("email"), Headline: f.Get("headline"),
		Text: f.Get("text"), Button: f.Get("button"), Link: f.Get("link"), Notes: f.Get("notes")}
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	// Bots fill the invisible field; they get the form again.
	if f.Get("website") != "" {
		h.sponsorForm(w, r, s, pl, http.StatusOK, f.Get("slot"), f.Get("issue"), in, sponsors.RequestOutcome{})
		return
	}
	if verify := h.Forms.TurnstileVerifier(ctx, s.set, ip); verify != nil && !verify(f.Get("cf-turnstile-response")) {
		out := sponsors.RequestOutcome{Errors: map[string]string{"turnstile": h.formUnreadable(r, s)}}
		h.sponsorForm(w, r, s, pl, http.StatusUnprocessableEntity, f.Get("slot"), f.Get("issue"), in, out)
		return
	}
	req := sponsors.RequestInput{Company: in.Company, Name: in.Name, Email: in.Email, Website: in.Website, Notes: in.Notes, Slot: f.Get("slot"), Issue: f.Get("issue"),
		Headline: in.Headline, Text: in.Text, Button: in.Button, Link: in.Link, KeepLogo: f.Get("keep_logo"), KeepImage: f.Get("keep_image"), Lang: string(l.Locale)}
	req.Logo, req.Image = formFile(r, "logo"), formFile(r, "image")
	// Five sent requests an hour per address: checked before the request
	// is stored, counted once it is.
	if !h.Forms.HourlyLeft("sponsor-request-ip:"+ip, 5) {
		h.sponsorFormWith(w, r, s, pl, http.StatusTooManyRequests, f.Get("slot"), f.Get("issue"), in, sponsors.RequestOutcome{}, h.sponsorLimit(l))
		return
	}
	out, err := h.Sponsors.Request(ctx, s.ws, req)
	switch {
	case errors.Is(err, sponsors.ErrPageOff):
		h.sponsorKit(w, r)
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	case len(out.Errors) > 0:
		h.sponsorForm(w, r, s, pl, http.StatusUnprocessableEntity, f.Get("slot"), f.Get("issue"), in, out)
		return
	}
	h.Forms.HourlyTake("sponsor-request-ip:"+ip, 5)
	h.sponsorDone(w, r, s, pl, out.Token)
}

func (h *Handler) sponsorLimit(l i18n.Lang) *earnNoticeView {
	return &earnNoticeView{Title: sponsors.TextIn(l, "req.limitTitle", nil), Text: sponsors.TextIn(l, "req.limitText", nil)}
}

// formFile reads an uploaded image of the form (nil when none was chosen).
func formFile(r *http.Request, field string) *sponsors.RequestFile {
	if r.MultipartForm == nil {
		return nil
	}
	fh := r.MultipartForm.File[field]
	if len(fh) == 0 || fh[0].Size == 0 {
		return nil
	}
	data, err := readFile(fh[0])
	if err != nil {
		return nil
	}
	return &sponsors.RequestFile{Name: fh[0].Filename, Data: data}
}

func readFile(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, sponsors.MaxRequestBody))
}

// sponsorDone is the page after a request: what was sent and the link to
// follow it.
func (h *Handler) sponsorDone(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, token string) {
	ctx := r.Context()
	l := pl.Labels.Lang()
	st, err := h.Sponsors.Status(ctx, s.ws, token)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	pub := s.ws.Name
	date := sponsors.IssueDate(l, st.IssueOn, st.Now)
	b := sponsorRequestBody{State: "done", Title: sponsors.TextIn(l, "req.doneTitle", nil), TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r),
		DoneText:  sponsors.TextIn(l, "req.doneText", i18n.Args{"email": st.Email, "publisher": pub}),
		DoneLine:  sponsors.TextIn(l, "req.doneLine", i18n.Args{"slot": st.Slot, "issue": st.Issue, "date": date}),
		DonePrice: sponsors.Money(st.Price, st.Currency),
		DoneWho:   sponsors.TextIn(l, "req.doneWho", i18n.Args{"company": st.Company, "name": st.Name, "at": sponsors.Stamp(l, st.SentAt, s.ws)}),
		DonePay:   sponsors.TextIn(l, "req.donePay", i18n.Args{"days": st.PayDays}),
		Follow:    earnLink(s.links, "sponsor/b/"+token), Back: earnLink(s.links, "sponsor"),
		Words: sponsorWords(l, "req.follow", "req.back")}
	vw := h.earnView(r, s, h.earnTitle(s, b.Title), b, pl)
	h.render(w, r, s, "sponsorrequest", http.StatusOK, vw, csp{}, true)
}

func (h *Handler) sponsorStatus(w http.ResponseWriter, r *http.Request) {
	s, ok := h.sponsorsOK(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	ctx := r.Context()
	back := r.URL.Query().Get("paid") == "1"
	if back {
		// Back from Stripe: a payment whose event hasn't arrived is
		// applied now.
		if _, err := h.Sponsors.Returned(ctx, s.ws, token); err != nil && !errors.Is(err, sponsors.ErrNoBooking) {
			h.log().WarnContext(ctx, "sponsor payment on return", "err", err)
		}
	}
	h.renderSponsorStatus(w, r, s, token, back, nil)
}

func (h *Handler) renderSponsorStatus(w http.ResponseWriter, r *http.Request, s *site, token string, back bool, notice *earnNoticeView) {
	ctx := r.Context()
	st, err := h.Sponsors.Status(ctx, s.ws, token)
	if errors.Is(err, sponsors.ErrNoBooking) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	pl := h.langOf(w, r, s, st.Lang)
	r = r.WithContext(i18n.WithLang(ctx, pl.Labels.Lang()))
	l := pl.Labels.Lang()
	pub := s.ws.Name
	date := sponsors.IssueDate(l, st.IssueOn, st.Now)
	price := sponsors.Money(st.Price, st.Currency)
	a := i18n.Args{"publisher": pub, "company": st.Company, "slot": st.Slot, "issue": st.Issue, "date": date, "list": st.List}
	b := sponsorStatusBody{State: st.State, Pill: sponsors.TextIn(l, "status.pill."+st.State, nil), PillTone: st.State, Line: sponsors.TextIn(l, "status.line", a),
		BlockTitle: sponsors.TextIn(l, "status.yourBlock", nil), Private: sponsors.TextIn(l, "status.private", nil), Test: st.Test,
		TestText: sponsors.TextIn(l, "kit.test", nil), TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r),
		Another: earnLink(s.links, "sponsor/request"), AnotherTxt: sponsors.TextIn(l, "status.requestAnother", nil),
		Charged: sponsors.TextIn(l, "status.nothingCharged", nil), Notice: notice}
	if st.Contact != "" {
		b.Questions = mailtoHTML(sponsors.TextIn(l, "status.questions", i18n.Args{"email": st.Contact}), st.Contact)
	}
	b.Block = &sponsorBlockView{Initials: sponsors.Initials(st.Company), Logo: st.Creative.Logo, Image: st.Creative.Image,
		Label: sponsors.TextIn(l, "status.sponsoredBy", i18n.Args{"company": st.Company}), Headline: st.Creative.Headline, Text: st.Creative.Text,
		Button: st.Creative.Button}
	due := ""
	if st.PaymentDueAt != nil {
		due = sponsors.IssueDate(l, *st.PaymentDueAt, st.Now)
	}
	switch st.State {
	case "requested":
		b.Heading = sponsors.TextIn(l, "status.requestedTitle", a)
		b.Text = sponsors.TextIn(l, "status.requestedText", nil)
		b.Facts = []sponsorFact{{sponsors.TextIn(l, "status.price", nil), price}, {sponsors.TextIn(l, "status.sent", nil), sponsors.Stamp(l, st.SentAt, s.ws)},
			{sponsors.TextIn(l, "status.updatesTo", nil), st.Email}}
	case "awaiting_payment":
		a["date"] = due
		b.Heading = sponsors.TextIn(l, "status.approvedTitle", a)
		a["date"] = date
		b.Line = sponsors.TextIn(l, "status.line", a)
		a["date"] = due
		if st.CanPay {
			b.Text = sponsors.TextIn(l, "status.approvedText", a)
			b.PayFor, b.Price, b.Once = sponsors.TextIn(l, "status.payFor", a), price, sponsors.TextIn(l, "status.once", nil)
			b.PayAction = earnLink(s.links, "sponsor/b/"+token+"/pay")
			b.PayButton = sponsors.TextIn(l, "status.pay", i18n.Args{"amount": price})
			b.PayNote = sponsors.TextIn(l, "status.payNote", a)
			if back && b.Notice == nil {
				b.Notice = &earnNoticeView{Text: sponsors.TextIn(l, "status.paidBack", nil)}
			}
		} else {
			b.State = "late"
			b.Notice = &earnNoticeView{Title: sponsors.TextIn(l, "status.lateTitle", nil), Text: sponsors.TextIn(l, "status.lateText", a)}
		}
	case "booked":
		b.Heading = sponsors.TextIn(l, "status.bookedTitle", a)
		b.Line = sponsors.TextIn(l, "status.lineBooked", a)
		place := sponsors.PlacementInSentence(l, st.Placement)
		if place == "" {
			place = "none"
		}
		if st.PaidOn != nil && st.PaidSource == "stripe" {
			paid := price
			if st.PaidCents != nil {
				paid = sponsors.Money(*st.PaidCents, st.Currency)
			}
			b.Text = sponsors.TextIn(l, "status.bookedText", i18n.Args{"amount": paid, "paid": sponsors.DayMonth(l, *st.PaidOn), "list": st.List,
				"issue": st.Issue, "placement": place})
		} else {
			b.Text = sponsors.TextIn(l, "status.bookedTextManual", i18n.Args{"list": st.List, "issue": st.Issue, "placement": place})
		}
		if st.Contact != "" && st.CreativeDue != nil {
			b.Change = mailtoHTML(sponsors.TextIn(l, "status.change", i18n.Args{"email": st.Contact, "date": sponsors.IssueDate(l, *st.CreativeDue, st.Now)}), st.Contact)
		}
		b.BlockTitle = sponsors.TextIn(l, "status.yourBlockIn", a)
	case "declined":
		b.Heading = sponsors.TextIn(l, "status.declinedTitle", a)
		b.Quote = st.Reason
		if st.DecidedAt != nil {
			b.QuoteBy = sponsors.TextIn(l, "status.declinedBy", i18n.Args{"publisher": pub, "date": sponsors.DayMonth(l, *st.DecidedAt)})
		}
		b.Block = nil
	case "expired":
		a["date"] = due
		b.Heading = sponsors.TextIn(l, "status.expiredTitle", nil)
		b.Text = sponsors.TextIn(l, "status.expiredText", a)
		b.Block = nil
	default:
		b.State = "canceled"
		b.Heading = sponsors.TextIn(l, "status.canceledTitle", nil)
		b.Text = sponsors.TextIn(l, "status.canceledText", a)
		b.Block = nil
	}
	if b.Heading == "" {
		b.Heading = sponsors.TextIn(l, "status.title", nil)
	}
	vw := h.earnView(r, s, h.earnTitle(s, sponsors.TextIn(l, "status.title", nil)), b, pl)
	h.render(w, r, s, "sponsorstatus", http.StatusOK, vw, csp{formAction: []string{"https://checkout.stripe.com"}}, true)
}

func (h *Handler) sponsorPay(w http.ResponseWriter, r *http.Request) {
	s, ok := h.sponsorsOK(w, r)
	if !ok {
		return
	}
	if !h.parseForm(w, r, 4<<10) {
		h.notFound(w, r, s)
		return
	}
	if !h.samePage(w, r) {
		return
	}
	token := chi.URLParam(r, "token")
	if !h.allowLink(r, "sponsor-pay", token, 5) {
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}
	ctx := r.Context()
	back := h.Earn.PageURL(ctx, s.ws, "sponsor/b/"+token)
	to, err := h.Sponsors.Pay(ctx, s.ws, token, back)
	switch {
	case errors.Is(err, sponsors.ErrNoBooking):
		h.notFound(w, r, s)
		return
	case errors.Is(err, sponsors.ErrPayClosed):
		h.renderSponsorStatus(w, r, s, token, false, nil)
		return
	case errors.Is(err, sponsors.ErrPaymentsUnavailable):
		st, _ := h.Sponsors.Status(ctx, s.ws, token)
		l := h.langOf(w, r, s, st.Lang).Labels.Lang()
		h.renderSponsorStatus(w, r, s, token, false, &earnNoticeView{Text: sponsors.TextIn(l, "status.payUnavailable", nil)})
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.toProvider(w, r, s, to)
}
