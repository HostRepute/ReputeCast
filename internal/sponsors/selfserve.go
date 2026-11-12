package sponsors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg" // logos and images: their size
	_ "image/png"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// The sponsor page (internal/publicpages writes it): /p/{workspace}/sponsor
// is the media kit with the self-serve slots and their open issues,
// /sponsor/request the request form, /sponsor/b/{token} a request's status
// page, where an approved request is paid.

// openWeeks is how far ahead the sponsor page offers issues.
const openWeeks = 12

// Limits of a request's fields.
const (
	maxCompany  = 120
	maxName     = 120
	maxWebsite  = 300
	maxNotes    = 600
	maxHeadline = 120
	maxText     = 280
	maxButton   = 40
	maxLink     = 1000
	// MaxRequestImage is the largest logo or image a request sends.
	MaxRequestImage = 2 << 20
	// MaxRequestBody is the largest request form, its images included.
	MaxRequestBody = 3 << 20
)

// ErrPageOff is a sponsor page that's switched off (or can't take
// payments): it reads as closed.
var ErrPageOff = errors.New("sponsors: the sponsor page is off")

// ErrNoBooking is a status page's link that names no booking.
var ErrNoBooking = errors.New("sponsors: no such booking")

// SponsorPage is the sponsor page: the media kit and the self-serve slots
// with the issues still open.
type SponsorPage struct {
	// Test: the payment account is in test mode (no real payment).
	Test bool
	// Now is the page's moment (dates of this year leave out the year).
	Now time.Time
	Kit SponsorMediaKit
	// Contact is where sponsors write with questions.
	Contact string
	// Lists are the slots' lists' numbers.
	Lists []SponsorKitStats
	Slots []PageSlot
}

// PageSlot is a self-serve slot on the sponsor page.
type PageSlot struct {
	ID                                     uuid.UUID
	Name, ListName, Placement, Description string
	ListID                                 uuid.UUID
	PriceCents                             int64
	Currency                               string
	EveryDays, LeadDays, PayDays           int
	CreativeDays                           int
	// Issues are the issues still open in the next twelve weeks.
	Issues []PageIssue
}

// PageIssue is an open issue: its number and date (the workspace's
// calendar day at noon there, so it reads as that day anywhere).
type PageIssue struct {
	Number int
	On     time.Time
}

// Page is the sponsor page; ErrPageOff when it's off.
func (s *Service) Page(ctx context.Context, ws *store.Workspace) (SponsorPage, error) {
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return SponsorPage{}, err
	}
	connected, test, err := s.selling(ctx, ws)
	if err != nil {
		return SponsorPage{}, err
	}
	if !st.on || !connected {
		return SponsorPage{}, ErrPageOff
	}
	out := SponsorPage{Test: test, Now: s.now(), Kit: st.kit, Contact: s.contact(ctx, ws, st), Lists: []SponsorKitStats{}, Slots: []PageSlot{}}
	rows, err := s.Q.SponsorSelfServeSlots(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	open, err := s.openIssues(ctx, ws, rows)
	if err != nil {
		return out, err
	}
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		out.Slots = append(out.Slots, PageSlot{ID: r.ID, Name: r.Name, ListID: r.ListID, ListName: r.ListName, Placement: r.Placement, Description: r.Description,
			PriceCents: r.PriceCents, Currency: r.Currency, EveryDays: int(r.EveryDays), LeadDays: int(r.LeadDays), PayDays: int(r.PayDays),
			CreativeDays: int(r.CreativeDays), Issues: open[r.ID]})
		if !seen[r.ListID] {
			seen[r.ListID] = true
			k, err := s.kitStats(ctx, ws, r.ListID, r.ListName)
			if err != nil {
				return out, err
			}
			out.Lists = append(out.Lists, k)
		}
	}
	return out, nil
}

// contact is where sponsors write: the page's address, else Earn's
// support address.
func (s *Service) contact(ctx context.Context, ws *store.Workspace, st settings) string {
	if st.contact != "" {
		return st.contact
	}
	if s.Earn != nil && s.Earn() != nil {
		return s.Earn().PageSettingsOf(ctx, ws.ID).Support
	}
	return ""
}

// earliest is the first day a slot's issue can be booked from the sponsor
// page: lead_days ahead, and late enough for its creative to be due after
// today (else no payment could arrive before it).
func earliest(today time.Time, leadDays, creativeDays int32) time.Time {
	days := int(leadDays)
	if c := int(creativeDays) + 1; c > days {
		days = c
	}
	return today.AddDate(0, 0, days)
}

// openIssues are each slot's issues in the next twelve weeks, at least its
// lead days away, that no booking holds.
func (s *Service) openIssues(ctx context.Context, ws *store.Workspace, slots []store.SponsorSelfServeSlotsRow) (map[uuid.UUID][]PageIssue, error) {
	out := map[uuid.UUID][]PageIssue{}
	if len(slots) == 0 {
		return out, nil
	}
	today := s.today(ws)
	last := today.AddDate(0, 0, openWeeks*7)
	ids := make([]uuid.UUID, 0, len(slots))
	for _, sl := range slots {
		ids = append(ids, sl.ID)
	}
	held, err := s.Q.SponsorIssuesHeld(ctx, store.SponsorIssuesHeldParams{SlotIds: ids, FromOn: today, ToOn: last})
	if err != nil {
		return nil, err
	}
	taken := map[uuid.UUID]map[int]bool{}
	for _, h := range held {
		if taken[h.SlotID] == nil {
			taken[h.SlotID] = map[int]bool{}
		}
		taken[h.SlotID][int(h.IssueNumber)] = true
	}
	for _, sl := range slots {
		row := store.SponsorSlot{ID: sl.ID, FirstNumber: sl.FirstNumber, FirstOn: sl.FirstOn, EveryDays: sl.EveryDays}
		list := []PageIssue{}
		for n := max(NextNumber(row, earliest(today, sl.LeadDays, sl.CreativeDays)), 1); ; n++ {
			on := IssueOn(row, n)
			if on.After(last) {
				break
			}
			if !taken[sl.ID][n] {
				list = append(list, PageIssue{Number: n, On: noonOn(on, ws)})
			}
		}
		out[sl.ID] = list
	}
	return out, nil
}

// RequestInput is what a sponsor sends with the request form.
type RequestInput struct {
	Company, Name, Email, Website, Notes string
	Slot, Issue                          string
	// Headline, Text, Button and Link are the creative, as the Sponsor
	// block shows it; Logo and Image are optional files, KeepLogo and
	// KeepImage the ones a form sent before kept (KeptFile.ID).
	Headline, Text, Button, Link string
	Logo, Image                  *RequestFile
	KeepLogo, KeepImage          string
	// Lang is the page's language: the sponsor's emails are in it.
	Lang string
}

// RequestFile is a file sent with a request.
type RequestFile struct {
	Name string
	Data []byte
}

// KeptFile is an image a form with errors kept, so it needn't be chosen
// again.
type KeptFile struct {
	ID, Name, URL string
	Size          int
}

// RequestOutcome is a request sent (Token names its status page) or the
// form's errors by field, in the page's language. Gone: the issue asked
// for was booked meanwhile. Kept are the images the form keeps.
type RequestOutcome struct {
	Token  string
	Errors map[string]string
	Gone   int
	Kept   map[string]KeptFile
}

// requestDoc is what a sponsor wrote, kept with the booking.
type requestDoc struct {
	Company string `json:"company"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Website string `json:"website,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Lang    string `json:"lang"`
	// Slot and Issue are what the sponsor asked for (an approval may
	// change them).
	Slot  string `json:"slot"`
	Issue int    `json:"issue"`
}

func parseRequest(raw json.RawMessage) (requestDoc, bool) {
	var d requestDoc
	if len(raw) == 0 || json.Unmarshal(raw, &d) != nil || d.Email == "" {
		return d, false
	}
	return d, true
}

// requestFromRaw is a booking's request for the booking panel.
func requestFromRaw(raw json.RawMessage, at time.Time) *SponsorRequestFrom {
	d, ok := parseRequest(raw)
	if !ok {
		return nil
	}
	return &SponsorRequestFrom{Company: d.Company, Name: d.Name, Email: d.Email, Website: d.Website, Notes: d.Notes, At: at}
}

// requestLang is the language of a request's page, for its emails.
func requestLang(d requestDoc) i18n.Lang { return pagelabels.Lang(d.Lang) }

// clean trims a field, joins its lines and drops control characters.
func clean(v string, multiline bool) string {
	v = strings.ReplaceAll(v, "\r\n", "\n")
	v = strings.Map(func(r rune) rune {
		if r == '\n' && multiline {
			return r
		}
		if unicode.IsControl(r) {
			if r == '\n' || r == '\t' {
				return ' '
			}
			return -1
		}
		return r
	}, v)
	return strings.TrimSpace(v)
}

// webAddress checks a website: http or https with a host. Without a
// scheme it is https.
func webAddress(v string) (string, bool) {
	if v == "" {
		return "", true
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	return linkAddress(v)
}

// linkAddress checks a creative's link: written with https:// (or
// http://) and a host.
func linkAddress(v string) (string, bool) {
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || strings.ContainsAny(u.Host, " <>\"") ||
		!strings.HasPrefix(strings.ToLower(v), u.Scheme+"://") {
		return "", false
	}
	return u.String(), true
}

// minLogoWidth is the narrowest logo a request takes.
const minLogoWidth = 120

// imageFile checks a request's image: PNG or JPEG, up to 2 MB, and a
// logo at least 120 px wide.
func imageFile(ctx context.Context, f *RequestFile, logo bool) string {
	if len(f.Data) > MaxRequestImage {
		return text(ctx, "request.error.imageSize", i18n.Args{"size": lang(ctx).Region.Decimal(math.Ceil(float64(len(f.Data))/(1<<20)*10)/10, 1)})
	}
	t := http.DetectContentType(f.Data)
	if t != "image/png" && t != "image/jpeg" {
		return text(ctx, "request.error.imageType", nil)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(f.Data))
	if err != nil {
		return text(ctx, "request.error.imageDamaged", nil)
	}
	if logo && cfg.Width < minLogoWidth {
		return text(ctx, "request.error.logoWidth", i18n.Args{"width": cfg.Width, "min": minLogoWidth})
	}
	return ""
}

// requestImages keeps a request's images: each new one checked and
// stored outside the media library, or the one a form kept before. They
// are kept even when other fields need fixing, so the form keeps them.
func (s *Service) requestImages(ctx context.Context, ws *store.Workspace, in RequestInput, errs map[string]string) (map[string]KeptFile, error) {
	kept := map[string]KeptFile{}
	for _, f := range []struct {
		field string
		file  *RequestFile
		keep  string
	}{{"logo", in.Logo, in.KeepLogo}, {"image", in.Image, in.KeepImage}} {
		if f.file == nil || len(f.file.Data) == 0 {
			if id, err := uuid.Parse(strings.TrimSpace(f.keep)); err == nil {
				if a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: id}); err == nil {
					kept[f.field] = KeptFile{ID: a.ID.String(), Name: a.FileName, URL: media.URL(a.ID, a.FileName), Size: int(a.SizeBytes)}
				}
			}
			continue
		}
		if e := imageFile(ctx, f.file, f.field == "logo"); e != "" {
			errs[f.field] = e
			continue
		}
		if s.Hidden == nil {
			errs[f.field] = text(ctx, "request.error.imageUnavailable", nil)
			continue
		}
		a, err := s.Hidden.Hidden(ctx, s.Q, ws, f.file.Name, f.file.Data)
		var ie media.ImageError
		if errors.As(err, &ie) {
			errs[f.field] = ie.Error()
			continue
		}
		if err != nil {
			return kept, err
		}
		kept[f.field] = KeptFile{ID: a.ID, Name: a.FileName, URL: a.URL, Size: a.SizeBytes}
	}
	return kept, nil
}

// Request takes a sponsor's request: the library's sponsor with the same
// address (or a new one), a booking that doesn't hold the issue, its first
// activity line, the email with the status page's link and a notification
// to the people who manage sponsors. ErrPageOff when the page is off; the
// outcome's Errors when a field needs fixing.
func (s *Service) Request(ctx context.Context, ws *store.Workspace, in RequestInput) (RequestOutcome, error) {
	page, err := s.Page(ctx, ws)
	if err != nil {
		return RequestOutcome{}, err
	}
	errs := map[string]string{}
	req := requestDoc{Company: clean(in.Company, false), Name: clean(in.Name, false), Email: emailaddr.Normalize(in.Email), Notes: clean(in.Notes, true), Lang: in.Lang}
	cr := SponsorCreative{Headline: clean(in.Headline, false), Text: clean(in.Text, true), Button: clean(in.Button, false)}
	switch n := utf8.RuneCountInString(req.Company); {
	case n == 0:
		errs["company"] = text(ctx, "request.error.company", nil)
	case n > maxCompany:
		errs["company"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxCompany})
	}
	switch n := utf8.RuneCountInString(req.Name); {
	case n == 0:
		errs["name"] = text(ctx, "request.error.name", nil)
	case n > maxName:
		errs["name"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxName})
	}
	switch {
	case req.Email == "":
		errs["email"] = text(ctx, "request.error.email", nil)
	case len(req.Email) > 254 || !emailaddr.Valid(req.Email):
		errs["email"] = text(ctx, "request.error.emailInvalid", nil)
	}
	if w, ok := webAddress(clean(in.Website, false)); !ok || len(w) > maxWebsite {
		errs["website"] = text(ctx, "request.error.website", nil)
	} else {
		req.Website = w
	}
	if utf8.RuneCountInString(req.Notes) > maxNotes {
		errs["notes"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxNotes})
	}
	var slot *PageSlot
	for i := range page.Slots {
		if page.Slots[i].ID.String() == strings.TrimSpace(in.Slot) {
			slot = &page.Slots[i]
		}
	}
	var issue *PageIssue
	if slot == nil {
		errs["slot"] = text(ctx, "request.error.slot", nil)
	} else {
		n, _ := strconv.Atoi(strings.TrimSpace(in.Issue))
		for i := range slot.Issues {
			if slot.Issues[i].Number == n {
				issue = &slot.Issues[i]
			}
		}
		if issue == nil {
			errs["issue"] = text(ctx, "request.error.issue", nil)
		}
	}
	switch n := utf8.RuneCountInString(cr.Headline); {
	case n == 0:
		errs["headline"] = text(ctx, "request.error.headline", nil)
	case n > maxHeadline:
		errs["headline"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxHeadline})
	}
	switch n := utf8.RuneCountInString(cr.Text); {
	case n == 0:
		errs["text"] = text(ctx, "request.error.text", nil)
	case n > maxText:
		errs["text"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxText})
	}
	if utf8.RuneCountInString(cr.Button) > maxButton {
		errs["button"] = text(ctx, "request.error.tooLong", i18n.Args{"max": maxButton})
	}
	link, ok := linkAddress(clean(in.Link, false))
	switch {
	case clean(in.Link, false) == "":
		errs["link"] = text(ctx, "request.error.link", nil)
	case !ok || len(link) > maxLink:
		errs["link"] = text(ctx, "request.error.linkInvalid", nil)
	default:
		cr.Link = link
	}
	kept, err := s.requestImages(ctx, ws, in, errs)
	if err != nil {
		return RequestOutcome{}, err
	}
	cr.Logo, cr.Image = kept["logo"].URL, kept["image"].URL
	gone := 0
	if slot != nil && issue == nil {
		// An issue of the slot that another booking took meanwhile.
		if n, err := strconv.Atoi(strings.TrimSpace(in.Issue)); err == nil && n > 0 {
			if _, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: slot.ID, IssueNumber: int32(n)}); err == nil {
				gone = n
				errs["issue"] = text(ctx, "request.error.issueGone", nil)
			}
		}
	}
	if len(errs) > 0 {
		return RequestOutcome{Errors: errs, Gone: gone, Kept: kept}, nil
	}
	req.Slot, req.Issue = slot.ID.String(), issue.Number
	issueOn := civil(issue.On)
	cr.UTM = defaultUTM(ws, slot.ListName, issue.Number)
	creativeDue := issueOn.AddDate(0, 0, -slot.CreativeDays)
	token, hash := authn.NewToken()
	id := newID()
	now := s.now()
	sealed, err := s.sealToken(id, token)
	if err != nil {
		return RequestOutcome{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		sponsorID, sponsorName, err := s.requestSponsor(ctx, q, ws, req, cr.Logo, now)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(req)
		crRaw, _ := json.Marshal(cr)
		if err := q.SponsorRequestInsert(ctx, store.SponsorRequestInsertParams{ID: id, WorkspaceID: ws.ID, SlotID: slot.ID, IssueNumber: int32(issue.Number),
			IssueOn: issueOn, SponsorID: &sponsorID, PriceCents: slot.PriceCents, Currency: slot.Currency, Creative: crRaw, CreativeDue: &creativeDue,
			Request: raw, StatusTokenHash: hash, At: now}); err != nil {
			return err
		}
		if err := s.event(ctx, q, id, nil, "requested", map[string]any{"company": req.Company, "name": req.Name, "sponsor": sponsorName}); err != nil {
			return err
		}
		args := i18n.Args{"company": req.Company, "issue": issue.Number, "slot": slot.Name, "list": slot.ListName}
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.SponsorsManage, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSponsorRequest, Severity: "info",
			TitleText: msgs.M("notify.requestTitle").Keep(args), BodyText: msgs.M("notify.requestBody").Keep(args), Link: requestLink(ws, id),
			Data: map[string]any{"booking": id.String()}}); err != nil {
			return err
		}
		return s.queueMail(ctx, tx, MailArgs{WorkspaceID: ws.ID, BookingID: id, Email: MailReceived, Token: sealed})
	})
	if err != nil {
		return RequestOutcome{}, err
	}
	return RequestOutcome{Token: token, Kept: kept}, nil
}

// requestLink is a request's place in the app.
func requestLink(ws *store.Workspace, id uuid.UUID) string {
	return "/w/" + ws.Slug + "/earn/sponsors?tab=requests&request=" + id.String()
}

// sealToken keeps a status page's token encrypted for the email job that
// sends it; the booking keeps only its hash.
func (s *Service) sealToken(id uuid.UUID, token string) ([]byte, error) {
	if s.Box == nil {
		return nil, nil
	}
	return s.Box.SealString(token, "sponsor_booking:"+id.String()+":status_token")
}

// requestSponsor is the library's sponsor a request is from: the one with
// its address (whose blanks it fills), or a new one under a name nobody
// else in the library has.
func (s *Service) requestSponsor(ctx context.Context, q *store.Queries, ws *store.Workspace, req requestDoc, logo string, now time.Time) (uuid.UUID, string, error) {
	website := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(req.Website, "https://"), "http://"), "/")
	sp, err := q.SponsorByContactEmail(ctx, store.SponsorByContactEmailParams{WorkspaceID: ws.ID, Email: req.Email})
	if err == nil {
		if err := q.SponsorFillBlanks(ctx, store.SponsorFillBlanksParams{ID: sp.ID, WorkspaceID: ws.ID, Website: website, ContactName: req.Name, LogoUrl: logo, At: now}); err != nil {
			return uuid.Nil, "", err
		}
		return sp.ID, sp.Name, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", err
	}
	name := req.Company
	for i := 0; ; i++ {
		taken, err := q.SponsorNameTaken(ctx, store.SponsorNameTakenParams{WorkspaceID: ws.ID, Name: name, ID: uuid.Nil})
		if err != nil {
			return uuid.Nil, "", err
		}
		if !taken {
			break
		}
		switch {
		case i == 0 && website != "":
			name = req.Company + " (" + website + ")"
		case i == 0:
			name = req.Company + " (" + emailaddr.Domain(req.Email) + ")"
		default:
			name = req.Company + " (" + strconv.Itoa(i+1) + ")"
		}
	}
	id := newID()
	if err := q.SponsorInsert(ctx, store.SponsorInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Website: website, LogoUrl: logo, ContactName: req.Name,
		ContactEmail: req.Email, TermsDays: 14, At: now}); err != nil {
		return uuid.Nil, "", err
	}
	return id, name, nil
}

// SponsorStatus is a request's status page.
type SponsorStatus struct {
	// State: requested, awaiting_payment (approved: pay by PaymentDueAt),
	// booked (sent too), declined (Reason), expired, canceled.
	State    string
	Company  string
	Sponsor  string
	Slot     string
	List     string
	Issue    int
	IssueOn  time.Time
	Price    int64
	Currency string
	// PaymentDueAt is when an approved request's payment must arrive.
	PaymentDueAt *time.Time
	PaidOn       *time.Time
	PaidCents    *int64
	Reason       string
	SentAt       time.Time
	// CanPay: approved, before its deadline.
	CanPay   bool
	Creative SponsorCreative
	Contact  string
	Test     bool
	// Lang is the request's page language.
	Lang string
	// The request's sender, the slot's terms and placement, how it was
	// paid, when its creative is due and when it was decided.
	Email       string
	Name        string
	PayDays     int
	Placement   string
	PaidSource  string
	CreativeDue *time.Time
	DecidedAt   *time.Time
	// Now is the page's moment (dates of this year leave out the year).
	Now time.Time
	id  uuid.UUID
}

// Status is a request's status page by its link's token.
func (s *Service) Status(ctx context.Context, ws *store.Workspace, token string) (SponsorStatus, error) {
	token = strings.TrimSpace(token)
	if len(token) < 32 || len(token) > 64 {
		return SponsorStatus{}, ErrNoBooking
	}
	b, err := s.Q.SponsorBookingByToken(ctx, store.SponsorBookingByTokenParams{WorkspaceID: ws.ID, TokenHash: authn.HashToken(token)})
	if errors.Is(err, pgx.ErrNoRows) {
		return SponsorStatus{}, ErrNoBooking
	}
	if err != nil {
		return SponsorStatus{}, err
	}
	d, _ := parseRequest(b.Request)
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return SponsorStatus{}, err
	}
	out := SponsorStatus{State: b.Status, Company: firstNonEmpty(d.Company, b.SponsorName), Sponsor: b.SponsorName, Slot: b.SlotName, List: b.ListName,
		Issue: int(b.IssueNumber), IssueOn: noonOn(b.IssueOn, ws), Price: b.PriceCents, Currency: b.Currency, PaymentDueAt: b.PaymentDueAt, PaidCents: b.PaidCents,
		PaidOn: noonPtr(b.PaidOn, ws), Reason: b.DeclinedReason, SentAt: b.CreatedAt, Creative: parseCreative(b.Creative), Contact: s.contact(ctx, ws, st),
		Lang: d.Lang, Email: d.Email, Name: d.Name, PayDays: int(b.SlotPayDays), Placement: b.SlotPlacement, PaidSource: b.PaidSource,
		CreativeDue: noonPtr(b.CreativeDue, ws), Now: s.now(), id: b.ID}
	if b.Status == "declined" || b.Status == "expired" || b.Status == "canceled" {
		at := b.UpdatedAt
		out.DecidedAt = &at
	}
	if out.State == "sent" {
		out.State = "booked"
	}
	_, out.Test, _ = s.selling(ctx, ws)
	out.CanPay = b.Status == "awaiting_payment" && b.PaymentDueAt != nil && s.now().Before(*b.PaymentDueAt)
	return out, nil
}
