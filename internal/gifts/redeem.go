package gifts

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Redeeming (news.acme.io/gift/redeem/{code}), with no card: the recipient
// confirms the address the gift is for and joins the tier's lists, with
// the page's text as consent evidence. Access starts at once and ends
// after the gift's period. A link works once, until it expires; until
// it's used, anyone who has it can redeem it.

// ErrNoGift means no gift has the code.
var ErrNoGift = errors.New("gifts: no such gift")

// RedeemView is the redeem page in one of its states: redeem (the page),
// redeemed (done), have (they already have paid access), used, expired,
// refunded.
type RedeemView struct {
	State   string
	Heading string
	Text    string
	// The giver's message and its caption ("Priya Natarajan · 11 Oct").
	Message string
	From    string
	// What the tier includes.
	Included string
	Benefits []string
	Email    string
	Error    string
	Button   string
	Fine     string
	PassOn   string
	// Link is the gift link to pass on (the have state), its words.
	Link    string
	LinkURL string
	Copy    string
	Copied  string
	// Support is the publisher's address for the used, expired and
	// refunded states.
	Support string
	Words   map[string]string
}

// RedeemInput is the redeem page's form.
type RedeemInput struct {
	Email     string
	IP        *netip.Addr
	UserAgent string
}

func jsonUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

// RedeemPage builds the redeem page of a code.
func (s *Service) RedeemPage(ctx context.Context, ws *store.Workspace, code string) (RedeemView, error) {
	code = NormalizeCode(code)
	if code == "" {
		return RedeemView{}, ErrNoGift
	}
	row, err := s.Q.GiftByCode(ctx, store.GiftByCodeParams{WorkspaceID: ws.ID, Code: code})
	if errors.Is(err, pgx.ErrNoRows) {
		return RedeemView{}, ErrNoGift
	}
	if err != nil {
		return RedeemView{}, err
	}
	return s.redeemView(ctx, ws, row, "", ""), nil
}

// redeemView is the page for a gift's state.
func (s *Service) redeemView(ctx context.Context, ws *store.Workspace, row store.GiftByCodeRow, email, problem string) RedeemView {
	g := row.EarnGift
	ps := s.Earn.PageSettingsOf(ctx, ws.ID)
	short := s.tierShort(ctx, row.TierName, row.ListIds)
	v := RedeemView{Support: ps.Support, Words: map[string]string{"email": page("redeem.emailLabel", nil), "hint": page("redeem.emailHint", i18n.Args{"tier": short})}}
	now := s.now()
	contact := func(key string, args i18n.Args) string {
		if ps.Support == "" {
			return page(key+"NoSupport", args)
		}
		args["email"] = ps.Support
		return page(key, args)
	}
	switch {
	case g.Status == StatusRedeemed:
		v.State, v.Heading, v.Text = "used", page("redeem.usedTitle", nil), contact("redeem.usedText", i18n.Args{})
		return v
	case g.Status == StatusRefunded:
		v.State, v.Heading, v.Text = "refunded", page("redeem.refundedTitle", nil), contact("redeem.refundedText", i18n.Args{})
		return v
	case g.Status == StatusExpired || (g.LinkExpiresAt != nil && !g.LinkExpiresAt.After(now)):
		months := 12
		if st, err := s.Q.EarnSettings(ctx, ws.ID); err == nil {
			months = int(st.GiftLinkMonths)
		}
		v.State, v.Heading, v.Text = "expired", page("redeem.expiredTitle", nil), contact("redeem.expiredText", i18n.Args{"months": months})
		return v
	case g.Status != StatusPaid && g.Status != StatusDelivered:
		// A checkout that wasn't paid has no gift to redeem.
		v.State, v.Heading, v.Text = "refunded", page("redeem.refundedTitle", nil), contact("redeem.refundedText", i18n.Args{})
		return v
	}
	giver := firstNonEmpty(g.BuyerName, page("redeem.someone", nil))
	v.State = "redeem"
	v.Heading = page("redeem.title", i18n.Args{"giver": giver, "gift": period(int(g.Months)), "tier": row.TierName})
	if g.Message != "" {
		v.Message = g.Message
		at := g.CreatedAt
		if g.PaidAt != nil {
			at = *g.PaidAt
		}
		v.From = page("redeem.from", i18n.Args{"giver": giver, "date": dayText(at, ws, now)})
	}
	v.Included, v.Benefits = page("redeem.included", nil), row.Benefits
	v.Email = firstNonEmpty(email, g.RecipientEmail)
	v.Error = problem
	ends := now.AddDate(0, int(g.Months), 0)
	v.Button = page("redeem.button", i18n.Args{"tier": short, "months": g.Months})
	free := ""
	if names, _ := s.Q.EarnListNames(ctx, row.ListIds); len(names) > 0 {
		free = names[0]
	}
	key := "redeem.fine"
	if free == "" {
		key = "redeem.fineNoList"
	}
	v.Fine = page(key, i18n.Args{"tier": short, "date": ends.In(location(ws)).Format("2 Jan 2006"), "list": free})
	v.PassOn = page("redeem.passOn", nil)
	return v
}

// dayText is "11 Oct", with the year when it isn't this year's.
func dayText(t time.Time, ws *store.Workspace, now time.Time) string {
	loc := location(ws)
	if t.In(loc).Year() == now.In(loc).Year() {
		return t.In(loc).Format("2 Jan")
	}
	return t.In(loc).Format("2 Jan 2006")
}

// errHave rolls a redemption back: the address already has paid access.
var errHave = errors.New("gifts: already has paid access")

// Redeem redeems a gift for the address given.
func (s *Service) Redeem(ctx context.Context, ws *store.Workspace, code string, in RedeemInput) (RedeemView, error) {
	code = NormalizeCode(code)
	row, err := s.Q.GiftByCode(ctx, store.GiftByCodeParams{WorkspaceID: ws.ID, Code: code})
	if errors.Is(err, pgx.ErrNoRows) || code == "" {
		return RedeemView{}, ErrNoGift
	}
	if err != nil {
		return RedeemView{}, err
	}
	email := emailaddr.Normalize(in.Email)
	if !emailaddr.Valid(email) {
		key := "error.emailInvalid"
		if email == "" {
			key = "redeem.emailMissing"
		}
		return s.redeemView(ctx, ws, row, strings.TrimSpace(in.Email), page(key, nil)), nil
	}
	v := s.redeemView(ctx, ws, row, email, "")
	if v.State != "redeem" {
		return v, nil
	}
	g := row.EarnGift
	now := s.now()
	ends := now.AddDate(0, int(g.Months), 0)
	giver := firstNonEmpty(g.BuyerName, g.BuyerEmail)
	var contact uuid.UUID
	pending := false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		locked, err := q.GiftForUpdate(ctx, g.ID)
		if err != nil {
			return err
		}
		if locked.Status != StatusPaid && locked.Status != StatusDelivered {
			return errUsed
		}
		if c, err := q.EarnContactByEmail(ctx, store.EarnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
			if acc, _, err := s.Earn.PaidAccessOf(ctx, ws, c.ID, g.Mode); err != nil {
				return err
			} else if acc != nil {
				return errHave
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		first, last := "", ""
		if strings.EqualFold(email, g.RecipientEmail) {
			first, last = splitName(g.RecipientName)
		}
		if s.Signup != nil && g.Mode == "live" && len(row.ListIds) > 0 {
			contact, pending, err = s.Signup.Join(ctx, tx, ws, Join{Email: email, FirstName: first, LastName: last, Lists: row.ListIds,
				ConsentText: v.Fine, Note: page("sourceLabel", i18n.Args{"giver": giver}), IP: in.IP, UserAgent: in.UserAgent, At: now})
			if err != nil {
				return err
			}
		}
		if contact == uuid.Nil {
			contact, err = q.EarnContactInsert(ctx, store.EarnContactInsertParams{ID: newID(), WorkspaceID: ws.ID, Email: email, FirstName: first, LastName: last,
				MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), Source: "gift", RoleAddress: emailaddr.IsRole(email), At: now})
			if err != nil {
				return err
			}
		}
		sub := newID()
		detail, _ := json.Marshal(map[string]any{"from": giver, "months": g.Months, "gift": g.ID.String()})
		if err := q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: sub, WorkspaceID: ws.ID, ContactID: contact, Mode: g.Mode, Kind: "gift",
			TierID: g.TierID, Status: "gift", Currency: g.Currency, StartedAt: now, CurrentPeriodEnd: &ends, SourceKind: "gift",
			SourceLabel: page("sourceLabel", i18n.Args{"giver": giver}), SourceDetail: detail, At: now}); err != nil {
			return err
		}
		if err := q.GiftRedeemed(ctx, store.GiftRedeemedParams{At: now, Email: email, ContactID: &contact, SubscriptionID: &sub, ID: g.ID}); err != nil {
			return err
		}
		if err := event(ctx, q, g.ID, "redeemed", now, nil, nil, map[string]any{"email": email}); err != nil {
			return err
		}
		if err := s.Earn.Timeline(ctx, q, ws.ID, contact, &sub, g.Mode, "gift_redeemed", now, nil,
			map[string]any{"from": giver, "months": g.Months, "gift": g.ID.String()}); err != nil {
			return err
		}
		return s.emit(ctx, q, ws, "gift.redeemed", GiftEvent{GiftID: g.ID.String(), Mode: g.Mode, Tier: row.TierName, Months: int(g.Months),
			AmountCents: g.AmountCents, Currency: g.Currency, BuyerEmail: g.BuyerEmail, InPerson: g.InPerson, ContactID: contact.String(), Email: email,
			AccessEndsAt: &ends, OccurredAt: now.UTC()})
	})
	switch {
	case errors.Is(err, errHave):
		return s.haveView(ctx, ws, row), nil
	case errors.Is(err, errUsed):
		fresh, ferr := s.Q.GiftByCode(ctx, store.GiftByCodeParams{WorkspaceID: ws.ID, Code: code})
		if ferr != nil {
			return RedeemView{}, ferr
		}
		return s.redeemView(ctx, ws, fresh, email, ""), nil
	case err != nil:
		return RedeemView{}, err
	}
	if pending && s.Confirm != nil {
		if err := s.Confirm(ctx, ws, contact); err != nil {
			s.log().WarnContext(ctx, "gift: confirmation email", "err", err)
		}
	}
	return s.redeemedView(ctx, ws, row, ends), nil
}

var errUsed = errors.New("gifts: the gift was redeemed meanwhile")

// haveView tells someone with paid access already that they can pass the
// gift on.
func (s *Service) haveView(ctx context.Context, ws *store.Workspace, row store.GiftByCodeRow) RedeemView {
	short := s.tierShort(ctx, row.TierName, row.ListIds)
	return RedeemView{State: "have", Heading: page("redeem.haveTitle", i18n.Args{"tier": short}), Text: page("redeem.haveText", nil),
		Link: s.RedeemHost(ctx, ws, row.EarnGift.Code), LinkURL: s.RedeemURL(ctx, ws, row.EarnGift.Code), Copy: page("redeem.copy", nil), Copied: page("redeem.copied", nil), Words: map[string]string{"link": page("redeem.linkLabel", nil)}}
}

// redeemedView is the page after redeeming: until when, and what comes
// next.
func (s *Service) redeemedView(ctx context.Context, ws *store.Workspace, row store.GiftByCodeRow, ends time.Time) RedeemView {
	short := s.tierShort(ctx, row.TierName, row.ListIds)
	v := RedeemView{State: "redeemed", Heading: page("redeem.doneTitle", i18n.Args{"tier": short, "article": article(short), "date": ends.In(location(ws)).Format("2 Jan 2006")}),
		Button: page("redeem.preferences", nil), Words: map[string]string{}}
	remind := page("redeem.remind", nil)
	if line := s.Earn.WelcomeLine(ctx, ws, row.EarnGift.TierID, firstName(row.EarnGift.RecipientName)); line != "" {
		v.Text = page("redeem.doneText", i18n.Args{"welcome": line, "remind": remind})
	} else {
		v.Text = remind
	}
	return v
}

// article is "an" before a word English spells with a vowel first.
func article(word string) string {
	if word != "" && strings.ContainsRune("aeioAEIO", rune(word[0])) {
		return "an"
	}
	return "a"
}
