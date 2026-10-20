package gifts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The gift page (news.acme.io/gift): pick a gift, say who it's for and
// when it should arrive, add a message, and pay once through the
// provider's checkout. Nothing renews.

// ErrUnavailable means gifts can't be bought now: the gift page is off, it
// has no options, or no provider takes payments.
var ErrUnavailable = errors.New("gifts: not available")

// PageView is the gift page.
type PageView struct {
	// State: form, or off ("Gifts aren't available right now").
	State   string
	Heading string
	Intro   string
	Options []PageOption
	In      BuyInput
	// Errors are per field (option, recipientEmail, date, message,
	// buyerName, buyerEmail); Alert is the page's.
	Errors map[string]string
	Alert  string
	// Canceled: back from a checkout that wasn't finished.
	Canceled bool
	// Test: the provider's test mode takes the payment.
	Test bool
	// Zone is the time zone a date is read in (the browser's, filled in
	// by the page's script; the workspace's until then).
	Zone     string
	ZoneNote string
	MinDate  string
	// MessageLeft counts down the message's characters ("300 characters
	// left"); the page's script keeps it current with Words leftOne and
	// leftOther.
	MessageLeft string
	Footer      string
	Text        string
	Support     string
	Words       map[string]string
}

// PageOption is a gift on the page.
type PageOption struct {
	ID    string
	Title string
	Price string
	Sub   string
	// Tier names the tier when the page gives more than one.
	Tier string
}

// BuyInput is the gift page's form.
type BuyInput struct {
	Option         string
	To             string // email, person
	RecipientName  string
	RecipientEmail string
	When           string // now, date
	Date, Time     string // 2026-12-24, 08:00
	Timezone       string
	Message        string
	BuyerName      string
	BuyerEmail     string
}

func pageWords() map[string]string {
	keys := []string{"choose", "who", "toEmail", "toPerson", "theirName", "theirEmail", "personNote", "when", "now", "onDate", "date", "time", "message",
		"optional", "messagePlaceholder", "from", "yourName", "yourEmail", "fromNote", "continue", "namePlaceholder", "emailPlaceholder", "yourEmailPlaceholder",
		"messageLeft", "terms", "canceledTitle", "canceledText", "testTitle", "testText", "copy", "copied"}
	out := map[string]string{}
	for _, k := range keys {
		out[k] = page("page."+k, nil)
	}
	return out
}

// giftable is what the page sells: its options, the mode and the client.
type giftable struct {
	options []store.GiftOptionsRow
	mode    string
	client  payments.Client
	months  int
}

// sellable reads whether gifts can be bought now and what.
func (s *Service) sellable(ctx context.Context, ws *store.Workspace) (*giftable, error) {
	if err := s.Q.EarnSettingsEnsure(ctx, ws.ID); err != nil {
		return nil, err
	}
	st, err := s.Q.EarnSettings(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	if !st.GiftPageOn {
		return nil, ErrUnavailable
	}
	opts, err := s.Q.GiftOptions(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	live := opts[:0]
	for _, o := range opts {
		if o.TierName != "" {
			live = append(live, o)
		}
	}
	if len(live) == 0 {
		return nil, ErrUnavailable
	}
	mode := s.Earn.ActiveMode(ctx, ws.ID)
	if mode == "" {
		return nil, ErrUnavailable
	}
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		return nil, ErrUnavailable
	}
	if _, ok := client.(payments.OneTimePayments); !ok {
		return nil, ErrUnavailable
	}
	return &giftable{options: live, mode: mode, client: client, months: int(st.GiftLinkMonths)}, nil
}

// Page builds the gift page; in carries what was typed (a correction).
func (s *Service) Page(ctx context.Context, ws *store.Workspace, in BuyInput, canceled bool) (PageView, error) {
	ps := s.Earn.PageSettingsOf(ctx, ws.ID)
	v := PageView{Words: pageWords(), Errors: map[string]string{}, Canceled: canceled, Support: ps.Support}
	gt, err := s.sellable(ctx, ws)
	if errors.Is(err, ErrUnavailable) {
		v.State, v.Heading = "off", page("off.title", nil)
		v.Text = page("off.text", nil)
		if ps.Support != "" {
			v.Text = page("off.textSupport", i18n.Args{"email": ps.Support})
		}
		return v, nil
	}
	if err != nil {
		return v, err
	}
	v.State = "form"
	v.Test = gt.mode == "test"
	st, err := s.Q.EarnSettings(ctx, ws.ID)
	if err != nil {
		return v, err
	}
	var custom struct {
		Headline string `json:"headline"`
		Intro    string `json:"intro"`
	}
	_ = jsonUnmarshal(st.GiftPage, &custom)
	tiers := map[uuid.UUID]bool{}
	for _, o := range gt.options {
		tiers[o.TierID] = true
	}
	first := gt.options[0]
	v.Heading, v.Intro = custom.Headline, custom.Intro
	if v.Heading == "" {
		if len(tiers) == 1 {
			v.Heading = page("page.heading", i18n.Args{"tier": first.TierName})
		} else {
			v.Heading = page("page.headingMany", nil)
		}
	}
	if v.Intro == "" {
		if len(tiers) == 1 && strings.TrimSpace(first.TierDescription) != "" {
			v.Intro = page("page.introTier", i18n.Args{"description": strings.TrimSpace(first.TierDescription)})
		} else {
			v.Intro = page("page.intro", nil)
		}
	}
	best := gt.options[0]
	for _, o := range gt.options {
		po := PageOption{ID: o.ID.String(), Title: periodTitle(int(o.Months)), Price: earn.Money(o.AmountCents, o.Currency), Sub: page("page.paidOnce", nil)}
		if len(tiers) > 1 {
			po.Tier = s.tierShort(ctx, o.TierName, o.ListIds)
		}
		v.Options = append(v.Options, po)
		if o.Months > best.Months {
			best = o
		}
	}
	v.In = in
	if v.In.Option == "" {
		v.In.Option = best.ID.String()
	}
	if v.In.To == "" {
		v.In.To = "email"
	}
	if v.In.When == "" {
		v.In.When = "now"
	}
	zone := in.Timezone
	if _, err := time.LoadLocation(zone); zone == "" || err != nil {
		zone = location(ws).String()
	}
	v.Zone = zone
	v.ZoneNote = page("page.whenNote", i18n.Args{"zone": zone})
	v.MinDate = s.now().In(zoneOr(zone, ws)).Format("2006-01-02")
	v.MessageLeft = page("page.messageLeft", i18n.Args{"n": MessageMax - utf8.RuneCountInString(v.In.Message)})
	v.Words["leftOne"] = strings.Replace(page("page.messageLeft", i18n.Args{"n": 1}), "1", "{n}", 1)
	v.Words["leftOther"] = strings.Replace(page("page.messageLeft", i18n.Args{"n": 7}), "7", "{n}", 1)
	v.Footer = page("page.footer", i18n.Args{"provider": "Stripe", "months": gt.months})
	return v, nil
}

// Buy checks the gift page's form and opens the provider's checkout for
// one payment. It returns the checkout's address, or the page again with
// what to correct.
func (s *Service) Buy(ctx context.Context, ws *store.Workspace, in BuyInput) (string, *PageView, error) {
	in = trimInput(in)
	v, err := s.Page(ctx, ws, in, false)
	if err != nil {
		return "", nil, err
	}
	if v.State != "form" {
		return "", &v, nil
	}
	gt, err := s.sellable(ctx, ws)
	if errors.Is(err, ErrUnavailable) {
		// Turned off (or disconnected) while the form was open.
		v.State, v.Heading, v.Text = "off", page("off.title", nil), page("off.text", nil)
		return "", &v, nil
	}
	if err != nil {
		return "", nil, err
	}
	var opt *store.GiftOptionsRow
	for i := range gt.options {
		if gt.options[i].ID.String() == in.Option {
			opt = &gt.options[i]
		}
	}
	if opt == nil {
		v.Errors["option"] = page("error.option", nil)
	}
	person := in.To == "person"
	if !person {
		switch {
		case in.RecipientEmail == "":
			v.Errors["recipientEmail"] = page("error.recipientEmail", nil)
		case !emailaddr.Valid(in.RecipientEmail):
			v.Errors["recipientEmail"] = page("error.emailInvalid", nil)
		}
	}
	now := s.now()
	var deliverAt *time.Time
	tz := v.Zone
	if in.When == "date" {
		at, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Time, zoneOr(tz, ws))
		switch {
		case err != nil:
			v.Errors["date"] = page("error.date", nil)
		case !at.After(now.Add(5 * time.Minute)):
			v.Errors["date"] = page("error.datePast", nil)
		case at.After(now.Add(MaxDeliverOut)):
			v.Errors["date"] = page("error.dateFar", nil)
		default:
			deliverAt = &at
		}
	}
	if utf8.RuneCountInString(in.Message) > MessageMax {
		v.Errors["message"] = page("error.message", i18n.Args{"max": MessageMax})
	}
	if in.BuyerName == "" {
		v.Errors["buyerName"] = page("error.buyerName", nil)
	}
	switch {
	case in.BuyerEmail == "":
		v.Errors["buyerEmail"] = page("error.buyerEmail", nil)
	case !emailaddr.Valid(in.BuyerEmail):
		v.Errors["buyerEmail"] = page("error.emailInvalid", nil)
	}
	if len(v.Errors) > 0 {
		v.Alert = page("error.alert", i18n.Args{"count": len(v.Errors)})
		return "", &v, nil
	}
	ps := s.Earn.PageSettingsOf(ctx, ws.ID)
	id := newID()
	code, err := s.freeCode(ctx)
	if err != nil {
		return "", nil, err
	}
	recipientName, recipientEmail := in.RecipientName, emailaddr.Normalize(in.RecipientEmail)
	if person {
		recipientName, recipientEmail = "", ""
	}
	tier := opt.TierID
	optID := opt.ID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return store.New(tx).GiftInsert(ctx, store.GiftInsertParams{ID: id, WorkspaceID: ws.ID, Mode: gt.mode, OptionID: &optID, TierID: &tier, Months: opt.Months,
			AmountCents: opt.AmountCents, Currency: opt.Currency, BuyerName: in.BuyerName, BuyerEmail: emailaddr.Normalize(in.BuyerEmail), InPerson: person,
			RecipientName: recipientName, RecipientEmail: recipientEmail, Message: in.Message, DeliverAt: deliverAt, Timezone: tz, Code: code, At: now})
	})
	if err != nil {
		return "", nil, err
	}
	pay := gt.client.(payments.OneTimePayments)
	cs, err := pay.CreatePaymentCheckout(ctx, payments.PaymentCheckoutRequest{AmountCents: opt.AmountCents, Currency: opt.Currency,
		Name: page("checkout.name", i18n.Args{"tier": opt.TierName, "months": opt.Months}), Description: page("checkout.description", nil),
		CustomerEmail: emailaddr.Normalize(in.BuyerEmail), Tax: ps.Tax, SuccessURL: s.Earn.PageURL(ctx, ws, "gift/thanks?session={CHECKOUT_SESSION_ID}"),
		CancelURL: s.Earn.PageURL(ctx, ws, "gift?canceled=1"), ClientRef: id.String(), Metadata: map[string]string{"reputecast_gift": id.String()}})
	if err != nil {
		s.log().WarnContext(ctx, "gift checkout", "err", err)
		v.Alert = page("error.unavailable", nil)
		return "", &v, nil
	}
	if err := s.Q.GiftSetSession(ctx, store.GiftSetSessionParams{Session: cs.ID, At: now, ID: id}); err != nil {
		return "", nil, err
	}
	return cs.URL, nil, nil
}

func trimInput(in BuyInput) BuyInput {
	t := strings.TrimSpace
	in.Option, in.To, in.When = t(in.Option), t(in.To), t(in.When)
	in.RecipientName, in.RecipientEmail = clip(t(in.RecipientName), 120), clip(t(in.RecipientEmail), 254)
	in.Date, in.Time, in.Timezone = clip(t(in.Date), 10), clip(t(in.Time), 5), clip(t(in.Timezone), 64)
	in.Message = clipRunes(strings.TrimSpace(strings.ReplaceAll(in.Message, "\r\n", "\n")), MessageMax+50)
	in.BuyerName, in.BuyerEmail = clip(t(in.BuyerName), 120), clip(t(in.BuyerEmail), 254)
	return in
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// freeCode draws a code no gift has.
func (s *Service) freeCode(ctx context.Context) (string, error) {
	for range 5 {
		c := newCode()
		taken, err := s.Q.GiftCodeTaken(ctx, c)
		if err != nil {
			return "", err
		}
		if !taken {
			return c, nil
		}
	}
	return "", fmt.Errorf("gifts: no free code")
}

// ThanksView is the page after paying (gift/thanks?session=…).
type ThanksView struct {
	// State: scheduled, sent (on its way now), person (the link goes to
	// the buyer), pending (the payment isn't confirmed yet).
	State   string
	Heading string
	Text    string
	// The gift: "Weekly Digest Insider · 1 year", "$80.00, paid once",
	// and to whom.
	Gift string
	Paid string
	To   string
	Note string
}

// Thanks is the page a buyer comes back to from checkout. A payment
// whose webhook isn't in yet is read from the provider.
func (s *Service) Thanks(ctx context.Context, ws *store.Workspace, session string) (ThanksView, error) {
	if session == "" || len(session) > 200 {
		return ThanksView{}, ErrUnavailable
	}
	row, err := s.Q.GiftBySession(ctx, store.GiftBySessionParams{WorkspaceID: ws.ID, Session: session})
	if errors.Is(err, pgx.ErrNoRows) {
		return ThanksView{}, ErrUnavailable
	}
	if err != nil {
		return ThanksView{}, err
	}
	g := row.EarnGift
	if g.Status == StatusCheckout || g.Status == StatusAbandoned {
		if client, _, err := s.Earn.ClientFor(ctx, ws.ID, g.Mode); err == nil {
			if cs, err := client.GetCheckout(ctx, session); err == nil && cs.Status == "complete" {
				_ = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
					q := store.New(tx)
					locked, err := q.GiftForUpdate(ctx, g.ID)
					if err != nil {
						return err
					}
					return s.markPaid(ctx, q, ws, locked, cs, s.now())
				})
				if r, err := s.Q.GiftBySession(ctx, store.GiftBySessionParams{WorkspaceID: ws.ID, Session: session}); err == nil {
					g = r.EarnGift
				}
			}
		}
	}
	now := s.now()
	v := ThanksView{Gift: page("thanks.gift", i18n.Args{"tier": row.TierName, "months": g.Months}),
		Paid: page("thanks.paid", i18n.Args{"amount": earn.MoneyCents(g.AmountCents, g.Currency)})}
	who := firstNonEmpty(firstName(g.RecipientName), g.RecipientEmail)
	switch {
	case g.Status == StatusCheckout || g.Status == StatusAbandoned:
		v.State, v.Heading, v.Text = "pending", page("thanks.pendingTitle", nil), page("thanks.pendingText", nil)
	case g.InPerson:
		v.State, v.Heading, v.Text = "person", page("thanks.personTitle", nil), page("thanks.personText", nil)
		v.To = page("thanks.toPerson", nil)
	case g.DeliverAt != nil && g.DeliverAt.After(now) && g.Status == StatusPaid:
		date, clock := deliveryText(*g.DeliverAt, g.Timezone, ws, now)
		v.State, v.Heading = "scheduled", page("thanks.scheduledTitle", i18n.Args{"name": who, "date": date, "time": clock})
		v.Text = page("thanks.receipt", nil)
		v.Note = page("thanks.moveNote", i18n.Args{"publisher": ws.Name})
	default:
		v.State, v.Heading, v.Text = "sent", page("thanks.sentTitle", i18n.Args{"name": who}), page("thanks.receipt", nil)
	}
	if !g.InPerson {
		to := g.RecipientEmail
		if g.RecipientName != "" {
			to = page("thanks.toNamed", i18n.Args{"name": g.RecipientName, "email": g.RecipientEmail})
		}
		key := "thanks.to"
		if g.Message != "" {
			key = "thanks.toMessage"
		}
		v.To = page(key, i18n.Args{"to": to})
	}
	return v, nil
}
