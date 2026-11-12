package earn

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The team pages (M65, G-38), rendered by internal/publicpages: the
// tier's team page with the seats and the price (/upgrade/{tier}/team),
// the buyer's manage page (/team/{token}, a week) and an invite's join
// page (/team/join/{token}, two weeks; accepting is a POST, so link
// scanners can't).

// TeamRequest is a visit of a tier's team page, or its form sent back
// with errors (Errors, in the page's language, by field).
type TeamRequest struct {
	Tier      string
	Period    string
	Seats     int
	Canceled  bool
	TestToken string
	Token     string
	Lang      i18n.Lang
	// The form as sent, when it comes back with errors.
	SeatsText string
	Team      string
	Email     string
	Errors    map[string]string
}

// TeamView is a tier's team page (PublicUpgradeTeam).
type TeamView struct {
	Lang        i18n.Lang
	TestMode    bool
	Unavailable bool
	TierSlug    string
	TierName    string
	Heading     string
	Intro       string
	Toggle      bool
	Period      string
	Monthly     string
	Yearly      string
	// Saving is the yearly price's saving ("2 months free").
	Saving   string
	SeatsMin int
	SeatsMax int
	// Seats is the stepper's number; SeatsText what was typed when it
	// came back out of range.
	Seats     int
	SeatsText string
	SeatsHint string
	// UnitText is "$60 per seat a year"; Times "× 3 seats", Total "$180"
	// and Per "a year"; Renews the tax and renewal line. TotalHint
	// replaces the total while the seats are out of range.
	UnitCents int64
	Currency  string
	UnitText  string
	Times     string
	Total     string
	Per       string
	Renews    string
	TotalHint string
	// Money is how the page's script writes the total as the seats change:
	// the currency's symbol, whether it has cents, and "× # seats" by
	// plural category.
	MoneySymbol string
	MoneyWhole  bool
	TimesForms  map[string]string
	Team        string
	Email       string
	Errors      map[string]string
	ErrorsTitle string
	Notice      *PageNotice
	Words       map[string]string
}

// TeamPage builds a tier's team page.
func (s *Service) TeamPage(ctx context.Context, ws *store.Workspace, r TeamRequest) (TeamView, error) {
	l := r.Lang
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return TeamView{}, err
	}
	conn, ok := active(conns)
	test := r.TestToken != "" && s.TestMode(ws, r.TestToken)
	if test {
		conn, ok = connected(conns, "test")
	}
	mode := ""
	if ok {
		mode = conn.Mode
	}
	t, gp, err := s.teamTier(ctx, ws, r.Tier, mode)
	if errors.Is(err, ErrUnavailable) && mode != "" {
		// Not sold in this mode yet: the page says so.
		if t, gp, err = s.teamTier(ctx, ws, r.Tier, ""); err == nil {
			ok = false
		}
	}
	if err != nil {
		return TeamView{}, err
	}
	names, _ := s.Q.EarnListNames(ctx, t.ListIds)
	short := shortTier(t.Name, names)
	v := TeamView{Lang: l, TestMode: mode == "test", Unavailable: !ok, TierSlug: t.Slug, TierName: t.Name,
		Heading: s.t(l, "team.heading", i18n.Args{"tier": t.Name}), Intro: s.t(l, "team.intro", i18n.Args{"tier": firstNonEmpty(short, t.Name)}),
		Monthly: s.t(l, "page.monthly", nil), Yearly: s.t(l, "page.yearly", nil), Errors: map[string]string{}, Team: r.Team, Email: r.Email,
		Words: s.teamWords(l)}
	month, hasMonth := gp["month"]
	year, hasYear := gp["year"]
	v.Toggle = hasMonth && hasYear
	if v.Toggle {
		if n := yearlySaving(month.price.AmountCents, year.price.AmountCents); n > 0 {
			v.Saving = s.t(l, "page.saving", i18n.Args{"count": n})
		}
	}
	v.Period = r.Period
	if _, has := gp[v.Period]; !has {
		v.Period = "year"
		if !hasYear {
			v.Period = "month"
		}
	}
	p := gp[v.Period].price
	v.SeatsMin, v.SeatsMax = int(deref32(p.SeatsMin)), int(deref32(p.SeatsMax))
	v.Seats = min(max(r.Seats, v.SeatsMin), v.SeatsMax)
	if r.Seats == 0 {
		v.Seats = v.SeatsMin
	}
	v.SeatsText = strconv.Itoa(v.Seats)
	if r.SeatsText != "" {
		v.SeatsText = r.SeatsText
	}
	v.SeatsHint = s.t(l, "team.seatsHint", i18n.Args{"min": v.SeatsMin, "max": v.SeatsMax})
	v.UnitCents, v.Currency = p.AmountCents, p.Currency
	v.UnitText = s.t(l, "team.perSeat", i18n.Args{"amount": money(p.AmountCents, p.Currency, true), "period": v.Period})
	v.Times = s.t(l, "team.times", i18n.Args{"seats": v.Seats})
	v.Total = money(p.AmountCents*int64(v.Seats), p.Currency, true)
	v.Per = s.t(l, "team.per", i18n.Args{"period": v.Period})
	cur := strings.ToLower(p.Currency)
	v.MoneySymbol, v.MoneyWhole = symbols[cur], zeroDecimal[cur]
	if v.MoneySymbol == "" {
		v.MoneySymbol = strings.ToUpper(cur) + " "
	}
	v.TimesForms = map[string]string{}
	for cat, n := range map[string]int{"one": 1, "other": 7} {
		v.TimesForms[cat] = strings.Replace(s.t(l, "team.times", i18n.Args{"seats": n}), strconv.Itoa(n), "#", 1)
	}
	tax := "exclusive"
	if st, err := s.settingsRow(ctx, s.Q, ws.ID); err == nil && st.PricesIncludeTax {
		tax = "inclusive"
	}
	renews := s.now().AddDate(1, 0, 0)
	key := "team.renewsYear"
	if v.Period == "month" {
		renews, key = s.now().AddDate(0, 1, 0), "team.renewsMonth"
	}
	v.Renews = s.t(l, key, i18n.Args{"tax": tax, "date": longDateIn(l, ws, renews)})
	for f, e := range r.Errors {
		v.Errors[f] = e
	}
	if _, bad := v.Errors["seats"]; bad {
		v.TotalHint = s.t(l, "team.totalHint", i18n.Args{"min": v.SeatsMin, "max": v.SeatsMax})
	}
	if n := len(v.Errors); n > 0 {
		v.ErrorsTitle = s.t(l, "team.errorsTitle", i18n.Args{"count": n})
	}
	if r.Canceled {
		v.Notice = &PageNotice{Tone: "info", Title: s.t(l, "page.canceledTitle", nil), Text: s.t(l, "page.canceledText", nil)}
	}
	if v.Email == "" {
		if ct, _, ok := s.readerOf(ctx, ws, r.Token); ok {
			v.Email = ct.Email
		}
	}
	return v, nil
}

func (s *Service) teamWords(l i18n.Lang) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"seats", "fewer", "more", "teamName", "teamNameHint", "teamNamePlaceholder", "email", "emailHint", "emailPlaceholder", "continue",
		"unavailableTitle", "unavailableText", "billingPeriod", "back", "testRibbon", "testNote", "secure"} {
		out[k] = s.t(l, "team.word."+k, nil)
	}
	out["errorsText"] = s.t(l, "team.errorsText", nil)
	return out
}

// TeamManageView is a team buyer's manage page (PublicTeamManage).
type TeamManageView struct {
	// State is manage, expired (a fresh link was emailed), ended or bad.
	State   string
	Lang    i18n.Lang
	Heading string
	// Sub is "Yearly · $600 a year · renews 3 Sep 2027 · managed by …";
	// Line says why something can't be done (paused).
	Sub  string
	Line string
	Team string
	Tier string
	// Seats are paid for; People are in the team (the buyer and active
	// seats), Invites wait; Used = People + Invites; Over is how many to
	// remove when Stripe lowered the seats below them.
	Seats   int
	People  int
	Invites int
	Used    int
	Free    int
	Over    int
	// OverTitle and OverText explain Over.
	OverTitle string
	OverText  string
	Members   []TeamMemberView
	// PeopleTitle is "People · 9".
	PeopleTitle string
	// CanInvite: a free seat and payments running; InviteHint says how
	// many; FullTitle and FullText when every seat is taken.
	CanInvite  bool
	InviteHint string
	FullTitle  string
	FullText   string
	// SeatsMin and SeatsMax bound Change seats; CanChange when the
	// provider can change the quantity. SeatsNote explains the seats;
	// SeatsError is the change refused (too few for the people).
	SeatsMin   int
	SeatsMax   int
	SeatsValue int
	CanChange  bool
	SeatsNote  string
	SeatsError *PageNotice
	LinkNote   string
	Flash      *PageNotice
	Words      map[string]string
}

// TeamMemberView is one person on the manage page.
type TeamMemberView struct {
	ID     string
	Email  string
	Name   string
	Status string // buyer, invited, active
	Label  string
	// Line is "your seat" or "sent 13 Oct · works until 27 Oct".
	Line        string
	You         string
	CanResend   bool
	RemoveLabel string
	ResendLabel string
	// Confirm is Remove's question, shown under the row it asks about.
	Confirm *TeamConfirm
}

// TeamConfirm asks before removing someone.
type TeamConfirm struct {
	Title, Text, Remove, Keep string
}

// ErrTeamLink means a team link isn't valid (altered, another workspace's).
var ErrTeamLink = errors.New("earn: the team link isn't valid")

// teamOf verifies a manage link and loads its team and buyer.
func (s *Service) teamOf(ctx context.Context, ws *store.Workspace, token string) (store.EarnGroup, store.PaidSubscription, error) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return store.EarnGroup{}, store.PaidSubscription{}, ErrTeamLink
	}
	c, err := s.Links.Verify(token, TokenGroupManage)
	if err != nil || c.WorkspaceID != ws.ID || c.MessageID == uuid.Nil {
		if errors.Is(err, linktoken.ErrSignature) {
			return store.EarnGroup{}, store.PaidSubscription{}, ErrLinkExpired
		}
		return store.EarnGroup{}, store.PaidSubscription{}, ErrTeamLink
	}
	g, err := s.Q.EarnGroup(ctx, store.EarnGroupParams{ID: c.MessageID, WorkspaceID: ws.ID})
	if err != nil || g.OwnerContactID != c.ContactID {
		return store.EarnGroup{}, store.PaidSubscription{}, ErrTeamLink
	}
	if linktoken.Expired(c, GroupLinkValid, s.now()) {
		return g, store.PaidSubscription{}, ErrLinkExpired
	}
	buyer, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: g.SubscriptionID, WorkspaceID: ws.ID})
	return g, buyer, err
}

// TeamManage builds the manage page of a link. An expired link mails a
// fresh one to the buyer (at most one every ten minutes) and says so.
// confirm names the seat whose Remove asks first.
func (s *Service) TeamManage(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang, confirm string) (TeamManageView, error) {
	g, buyer, err := s.teamOf(ctx, ws, token)
	switch {
	case errors.Is(err, ErrLinkExpired) && g.ID != uuid.Nil:
		if err := s.SendTeamLink(ctx, ws, g, l); err != nil && !errors.Is(err, errNoSystemMail) {
			s.log().WarnContext(ctx, "team link", "err", err)
		}
		email := ""
		if c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: g.OwnerContactID, WorkspaceID: ws.ID}); err == nil {
			email = c.Email
		}
		return TeamManageView{State: "expired", Lang: l, Heading: s.t(l, "team.expiredTitle", nil), Line: s.t(l, "team.expiredText", i18n.Args{"email": email}),
			Words: s.manageWords(l)}, nil
	case errors.Is(err, ErrLinkExpired), errors.Is(err, ErrTeamLink):
		return TeamManageView{State: "bad", Lang: l, Heading: s.t(l, "team.badTitle", nil), Line: s.t(l, "team.badText", nil), Words: s.manageWords(l)}, nil
	case err != nil:
		return TeamManageView{}, err
	}
	return s.manageView(ctx, ws, g, buyer, l, confirm)
}

func (s *Service) manageWords(l i18n.Lang) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"inviteTitle", "inviteLabel", "inviteButton", "noFree", "needFree", "seatsTitle", "seatsLabel", "seatsButton", "billingTitle",
		"billingText", "billingButton", "resend", "remove", "statusActive", "statusInvited"} {
		out[k] = s.t(l, "manage."+k, nil)
	}
	out["fewer"], out["more"] = s.t(l, "team.word.fewer", nil), s.t(l, "team.word.more", nil)
	return out
}

// manageView is the manage page of a team.
func (s *Service) manageView(ctx context.Context, ws *store.Workspace, g store.EarnGroup, buyer store.PaidSubscription, l i18n.Lang, confirm string) (TeamManageView, error) {
	tier := ""
	if buyer.TierID != nil {
		if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *buyer.TierID, WorkspaceID: ws.ID}); err == nil {
			tier = t.Name
		}
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID)
	v := TeamManageView{State: "manage", Lang: l, Team: g.Name, Tier: tier, Seats: int(g.Seats), Members: []TeamMemberView{}, Words: s.manageWords(l)}
	if buyer.EndedAt != nil || g.EndedAt != nil {
		v.State = "ended"
		v.Heading = s.t(l, "team.endedTitle", i18n.Args{"team": g.Name})
		v.Line = s.t(l, "team.endedText", nil)
		return v, nil
	}
	buyerContact, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID})
	if err != nil {
		return v, err
	}
	v.Members = append(v.Members, TeamMemberView{ID: buyer.ID.String(), Email: buyerContact.Email,
		Name: firstNonEmpty(strings.TrimSpace(buyerContact.FirstName+" "+buyerContact.LastName), buyerContact.Email), Status: "buyer",
		Label: s.t(l, "manage.statusActive", nil), Line: s.t(l, "manage.yourSeat", nil), You: s.t(l, "manage.you", nil)})
	rows, err := s.Q.EarnGroupSeats(ctx, g.ID)
	if err != nil {
		return v, err
	}
	v.People = 1
	lists, _ := s.Q.EarnListNames(ctx, s.tierLists(ctx, buyer.TierID, ws))
	free := ""
	if len(lists) > 0 {
		free = lists[0]
	}
	now := s.now()
	for _, r := range rows {
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		m := TeamMemberView{ID: r.ID.String(), Email: r.Email, Name: firstNonEmpty(name, r.Email), Status: r.Status}
		m.RemoveLabel = s.t(l, "manage.removeLabel", i18n.Args{"name": m.Name})
		if r.Status == "active" {
			v.People++
			m.Label = s.t(l, "manage.statusActive", nil)
		} else {
			v.Invites++
			m.Label, m.CanResend = s.t(l, "manage.statusInvited", nil), true
			m.ResendLabel = s.t(l, "manage.resendLabel", i18n.Args{"name": m.Name})
			if r.InviteExpiresAt != nil {
				a := i18n.Args{"sent": s.dateTextIn(l, ws, r.InviteSentAt), "until": s.dateTextIn(l, ws, *r.InviteExpiresAt)}
				m.Line = s.t(l, "manage.invitedLine", a)
				if !now.Before(*r.InviteExpiresAt) {
					m.Line = s.t(l, "manage.invitedExpired", a)
				}
			}
		}
		if confirm == m.ID {
			first := firstNonEmpty(strings.TrimSpace(r.FirstName), m.Name)
			c := &TeamConfirm{Title: s.t(l, "manage.removeTitle", i18n.Args{"name": m.Name}), Remove: s.t(l, "manage.removeButton", i18n.Args{"name": first}),
				Keep: s.t(l, "manage.keepButton", i18n.Args{"name": first})}
			switch {
			case r.Status == "invited":
				c.Text = s.t(l, "manage.removeInviteText", nil)
			case free != "":
				c.Text = s.t(l, "manage.removeText", i18n.Args{"name": first, "tier": firstNonEmpty(short, tier), "free": free, "seats": v.Seats})
			default:
				c.Text = s.t(l, "manage.removeTextNoList", i18n.Args{"name": first, "tier": firstNonEmpty(short, tier), "seats": v.Seats})
			}
			m.Confirm = c
		}
		v.Members = append(v.Members, m)
	}
	v.Used = v.People + v.Invites
	v.PeopleTitle = s.t(l, "manage.people", i18n.Args{"count": v.Used})
	v.Free = max(v.Seats-v.Used, 0)
	v.Over = max(v.Used-v.Seats, 0)
	v.SeatsValue = v.Seats
	v.Heading = s.t(l, "manage.heading", i18n.Args{"team": g.Name, "tier": firstNonEmpty(tier, short), "used": v.People, "seats": v.Seats})
	state, at := "renews", buyer.CurrentPeriodEnd
	switch {
	case buyer.PausedAt != nil:
		state, at = "paused", buyer.PauseResumesAt
	case buyer.CancelAt != nil:
		state, at = "ending", buyer.CancelAt
	}
	date := ""
	if at != nil {
		date = longDateIn(l, ws, *at)
	}
	if date == "" {
		state = "none"
	}
	v.Sub = s.t(l, "manage.sub", i18n.Args{"period": buyer.Period, "total": publicPriceIn(l, buyer.AmountCents*int64(max(buyer.Quantity, 1)), buyer.Currency, buyer.Period),
		"state": state, "date": date, "email": buyerContact.Email})
	v.LinkNote = s.t(l, "manage.linkNote", i18n.Args{"email": buyerContact.Email})
	paying := buyer.PausedAt == nil && paidAccess(buyer)
	v.CanInvite = v.Free > 0 && paying
	switch {
	case buyer.PausedAt != nil:
		v.Line = s.t(l, "manage.pausedLine", nil)
	case v.Free == 0 && v.Over == 0:
		v.FullTitle, v.FullText = s.t(l, "manage.fullTitle", i18n.Args{"seats": v.Seats}), s.t(l, "manage.fullText", nil)
	}
	if v.Free > 0 {
		v.InviteHint = s.t(l, "manage.inviteHint", i18n.Args{"free": v.Free})
	}
	seatsArgs := i18n.Args{"seats": v.Seats, "people": v.People, "invites": v.Invites, "used": v.Used, "period": buyer.Period}
	if v.Over > 0 {
		v.OverTitle = s.t(l, "manage.overTitle", i18n.Args{"seats": v.Seats, "count": v.Used})
		v.OverText = s.t(l, "manage.overText", i18n.Args{"seats": v.Seats, "over": v.Over, "date": s.dateTextIn(l, ws, g.UpdatedAt)})
		v.SeatsNote = s.t(l, "manage.seatsNoteOver", seatsArgs)
	} else {
		v.SeatsNote = s.t(l, "manage.seatsNote", seatsArgs)
	}
	if p, err := s.groupPriceOf(ctx, buyer); err == nil {
		v.SeatsMin, v.SeatsMax = max(int(deref32(p.SeatsMin)), 1), int(deref32(p.SeatsMax))
	} else {
		v.SeatsMin, v.SeatsMax = 1, max(int(g.Seats), 1000)
	}
	if client, _, err := s.ClientFor(ctx, ws.ID, buyer.Mode); err == nil {
		_, v.CanChange = client.(payments.Seats)
	}
	v.CanChange = v.CanChange && paying && buyer.StripeItemID != ""
	return v, nil
}

// tierLists are a tier's lists (none without a tier).
func (s *Service) tierLists(ctx context.Context, tier *uuid.UUID, ws *store.Workspace) []uuid.UUID {
	if tier == nil {
		return nil
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *tier, WorkspaceID: ws.ID})
	if err != nil {
		return nil
	}
	return t.ListIds
}

// groupPriceOf is the group price a buyer pays (for its seat range).
func (s *Service) groupPriceOf(ctx context.Context, buyer store.PaidSubscription) (store.EarnPrice, error) {
	if buyer.PriceID == nil {
		return store.EarnPrice{}, pgx.ErrNoRows
	}
	return s.Q.EarnPrice(ctx, *buyer.PriceID)
}

// TeamResult is what a manage-page action did: the page again, with a
// notice.
type TeamResult struct {
	View TeamManageView
	// Redirect is the provider's page to go to (billing).
	Redirect string
}

// teamAction verifies a manage link for a POST: an expired link mails a
// fresh one and shows the expired page.
func (s *Service) teamAction(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (store.EarnGroup, store.PaidSubscription, *TeamResult, error) {
	g, buyer, err := s.teamOf(ctx, ws, token)
	if err != nil {
		v, verr := s.TeamManage(ctx, ws, token, l, "")
		if verr != nil {
			return g, buyer, nil, verr
		}
		return g, buyer, &TeamResult{View: v}, nil
	}
	if buyer.EndedAt != nil || g.EndedAt != nil {
		v, err := s.manageView(ctx, ws, g, buyer, l, "")
		return g, buyer, &TeamResult{View: v}, err
	}
	return g, buyer, nil, nil
}

// flash is the manage page again with a notice.
func (s *Service) flash(ctx context.Context, ws *store.Workspace, g store.EarnGroup, l i18n.Lang, tone, key string, args i18n.Args) (TeamResult, error) {
	buyer, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: g.SubscriptionID, WorkspaceID: ws.ID})
	if err != nil {
		return TeamResult{}, err
	}
	fresh, err := s.Q.EarnGroup(ctx, store.EarnGroupParams{ID: g.ID, WorkspaceID: ws.ID})
	if err != nil {
		return TeamResult{}, err
	}
	v, err := s.manageView(ctx, ws, fresh, buyer, l, "")
	v.Flash = &PageNotice{Tone: tone, Text: s.t(l, key, args)}
	return TeamResult{View: v}, err
}

// TeamInvite invites people to a team: each address once, at most the
// free seats, while the team pays. Each gets the invite email.
func (s *Service) TeamInvite(ctx context.Context, ws *store.Workspace, token, emails string, l i18n.Lang) (TeamResult, error) {
	g, buyer, done, err := s.teamAction(ctx, ws, token, l)
	if err != nil || done != nil {
		return deref(done), err
	}
	list := splitEmails(emails)
	if len(list) == 0 || len(list) > MaxInvites {
		return s.flash(ctx, ws, g, l, "bad", "manage.inviteEmpty", nil)
	}
	for _, e := range list {
		if !emailaddr.Valid(e) {
			return s.flash(ctx, ws, g, l, "bad", "manage.inviteInvalid", i18n.Args{"email": e})
		}
	}
	if buyer.PausedAt != nil || !paidAccess(buyer) {
		return s.flash(ctx, ws, g, l, "bad", "manage.notPaying", nil)
	}
	now := s.now()
	type sent struct {
		email string
		token string
	}
	var invites []sent
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.EarnGroupForUpdate(ctx, store.EarnGroupForUpdateParams{ID: g.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		people, err := q.EarnGroupPeople(ctx, g.ID)
		if err != nil {
			return err
		}
		free := int(g.Seats) - 1 - int(people.People)
		buyerContact, err := q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		seats, err := q.EarnGroupSeats(ctx, g.ID)
		if err != nil {
			return err
		}
		fresh := []string{}
		for _, e := range list {
			if strings.EqualFold(e, buyerContact.Email) || slices.ContainsFunc(seats, func(x store.EarnGroupSeatsRow) bool { return strings.EqualFold(x.Email, e) }) {
				continue
			}
			fresh = append(fresh, e)
		}
		if len(fresh) > free {
			return &seatsFull{free: max(free, 0)}
		}
		for _, e := range fresh {
			id := newID()
			tok, hash := s.inviteToken(ws.ID, id)
			expires := now.Add(InviteValid)
			if err := q.EarnGroupSeatInsert(ctx, store.EarnGroupSeatInsertParams{ID: id, GroupID: g.ID, WorkspaceID: ws.ID, Email: e, InviteHash: hash, At: now,
				ExpiresAt: &expires}); err != nil {
				return err
			}
			invites = append(invites, sent{email: e, token: tok})
		}
		return s.timeline(ctx, q, ws.ID, buyer.ContactID, &buyer.ID, buyer.Mode, "team_invited", now, nil, "", nil, "", nil,
			map[string]any{"team": g.Name, "count": len(invites)})
	})
	var full *seatsFull
	if errors.As(err, &full) {
		return s.flash(ctx, ws, g, l, "bad", "manage.inviteFull", i18n.Args{"count": full.free})
	}
	if err != nil {
		return TeamResult{}, err
	}
	for _, in := range invites {
		s.sendInvite(ctx, ws, g, buyer, in.email, in.token, now)
	}
	if len(invites) == 0 {
		return s.flash(ctx, ws, g, l, "info", "manage.inviteNone", nil)
	}
	return s.flash(ctx, ws, g, l, "info", "manage.invitedFlash", i18n.Args{"count": len(invites)})
}

type seatsFull struct{ free int }

func (e *seatsFull) Error() string { return "earn: the team's seats are full" }

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// splitEmails reads the addresses of an invite: one per line or separated
// by commas, semicolons or spaces, lower-cased, each once.
func splitEmails(s string) []string {
	out := []string{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t' }) {
		e := strings.ToLower(strings.TrimSpace(f))
		if e != "" && !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

// sendInvite sends a seat's invite email.
func (s *Service) sendInvite(ctx context.Context, ws *store.Workspace, g store.EarnGroup, buyer store.PaidSubscription, email, token string, now time.Time) {
	short := s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID)
	inviter := g.Name
	if c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID}); err == nil {
		inviter = firstNonEmpty(strings.TrimSpace(c.FirstName+" "+c.LastName), g.Name)
	}
	expires := now.Add(InviteValid)
	data := map[string]any{"tier": short, "team": g.Name, "inviter": inviter, "join_url": s.pageURL(ctx, ws, "team/join/"+token),
		"expires_at": s.dateText(ws, expires)}
	texts := func(l i18n.Lang) map[string]any { return map[string]any{"expires_at": s.dateTextIn(l, ws, expires)} }
	h := sha256.Sum256([]byte(token))
	res := s.SendMemberEmailIn(ctx, ws, EmailGroupInvite, email, "", data, fmt.Sprintf("earn:invite:%x", h[:8]), texts)
	if res.Status != "sent" {
		s.log().WarnContext(ctx, "team invite not sent", "reason", res.Reason)
	}
}

// TeamResend sends an invite again with a new link (the older one stops
// working), at most once a minute per seat.
func (s *Service) TeamResend(ctx context.Context, ws *store.Workspace, token, seatID string, l i18n.Lang) (TeamResult, error) {
	g, buyer, done, err := s.teamAction(ctx, ws, token, l)
	if err != nil || done != nil {
		return deref(done), err
	}
	id, err := uuid.Parse(seatID)
	if err != nil {
		return s.flash(ctx, ws, g, l, "bad", "team.seatGone", nil)
	}
	seat, err := s.Q.EarnGroupSeat(ctx, store.EarnGroupSeatParams{ID: id, GroupID: g.ID})
	if err != nil || seat.Status != "invited" {
		return s.flash(ctx, ws, g, l, "bad", "team.seatGone", nil)
	}
	now := s.now()
	if now.Sub(seat.InviteSentAt) < time.Minute {
		return s.flash(ctx, ws, g, l, "info", "manage.resentFlash", i18n.Args{"email": seat.Email})
	}
	tok, hash := s.inviteToken(ws.ID, seat.ID)
	expires := now.Add(InviteValid)
	if err := s.Q.EarnGroupSeatReinvite(ctx, store.EarnGroupSeatReinviteParams{ID: seat.ID, InviteHash: hash, At: now, ExpiresAt: &expires}); err != nil {
		return TeamResult{}, err
	}
	s.sendInvite(ctx, ws, g, buyer, seat.Email, tok, now)
	return s.flash(ctx, ws, g, l, "info", "manage.resentFlash", i18n.Args{"email": seat.Email})
}

// TeamRemove removes a person from the team: an invite stops working, an
// active seat's access ends now.
func (s *Service) TeamRemove(ctx context.Context, ws *store.Workspace, token, seatID string, l i18n.Lang) (TeamResult, error) {
	g, buyer, done, err := s.teamAction(ctx, ws, token, l)
	if err != nil || done != nil {
		return deref(done), err
	}
	id, err := uuid.Parse(seatID)
	if err != nil {
		return s.flash(ctx, ws, g, l, "bad", "team.seatGone", nil)
	}
	var emails []memberEmail
	var removed store.EarnGroupSeat
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, m, err := s.removeSeat(ctx, q, ws, g, id, s.now())
		removed, emails = row, m
		if err != nil {
			return err
		}
		return s.timeline(ctx, q, ws.ID, buyer.ContactID, &buyer.ID, buyer.Mode, "team_removed", s.now(), nil, "", nil, "", nil,
			map[string]any{"team": g.Name, "email": row.Email})
	})
	if err != nil {
		var p *api.Problem
		if errors.As(err, &p) {
			return s.flash(ctx, ws, g, l, "bad", "team.seatGone", nil)
		}
		return TeamResult{}, err
	}
	for _, m := range emails {
		s.sendMemberEmail(ctx, ws, buyer.Mode, m, nil)
	}
	return s.flash(ctx, ws, g, l, "info", "manage.removedFlash", i18n.Args{"email": removed.Email})
}

// TeamSetSeats changes a team's seats at the provider (prorated): not
// below the people in the team, within the price's range.
func (s *Service) TeamSetSeats(ctx context.Context, ws *store.Workspace, token, seats string, l i18n.Lang) (TeamResult, error) {
	g, buyer, done, err := s.teamAction(ctx, ws, token, l)
	if err != nil || done != nil {
		return deref(done), err
	}
	n64, err := strconv.ParseInt(strings.TrimSpace(seats), 10, 32)
	n := int(n64)
	v, verr := s.manageView(ctx, ws, g, buyer, l, "")
	if verr != nil {
		return TeamResult{}, verr
	}
	switch {
	case err != nil || n < v.SeatsMin || n > v.SeatsMax:
		return s.flash(ctx, ws, g, l, "bad", "manage.seatsRange", i18n.Args{"min": v.SeatsMin, "max": v.SeatsMax})
	case n < v.Used:
		// The board's answer sits in Change seats, with the number typed.
		v.SeatsValue = n
		v.SeatsError = &PageNotice{Tone: "bad", Title: s.t(l, "manage.removeFirst", i18n.Args{"count": v.Used - n}),
			Text: s.t(l, "manage.removeFirstText", i18n.Args{"team": g.Name, "people": v.People, "invites": v.Invites, "seats": n, "count": v.Used - n})}
		return TeamResult{View: v}, nil
	case !v.CanChange:
		return s.flash(ctx, ws, g, l, "bad", "manage.cantChange", nil)
	case n == int(g.Seats):
		return s.flash(ctx, ws, g, l, "info", "manage.seatsSaved", i18n.Args{"count": n})
	}
	client, conn, err := s.ClientFor(ctx, ws.ID, buyer.Mode)
	if err != nil {
		return s.flash(ctx, ws, g, l, "bad", "manage.cantChange", nil)
	}
	seatsAPI, ok := client.(payments.Seats)
	if !ok {
		return s.flash(ctx, ws, g, l, "bad", "manage.cantChange", nil)
	}
	now := s.now()
	if err := seatsAPI.SetQuantity(ctx, buyer.StripeSubscriptionID, buyer.StripeItemID, n); err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		s.log().WarnContext(ctx, "team seats", "err", err)
		return s.flash(ctx, ws, g, l, "bad", "manage.seatsFailed", nil)
	}
	// The new quantity applies here at once; Stripe's event finds it in place.
	var emails []memberEmail
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before := buyer
		if err := q.EarnSubscriptionSetQuantity(ctx, store.EarnSubscriptionSetQuantityParams{ID: buyer.ID, Quantity: int32(n), At: now}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, buyer.ContactID, &buyer.ID, buyer.Mode, "seats_changed", now, nil, "", nil, "", nil,
			map[string]any{"from": g.Seats, "to": n, "team": g.Name}); err != nil {
			return err
		}
		m, err := s.afterWrite(ctx, q, ws, buyer.Mode, &before, buyer.ID, now, nil)
		emails = m
		return err
	})
	if err != nil {
		return TeamResult{}, err
	}
	for _, m := range emails {
		s.sendMemberEmail(ctx, ws, buyer.Mode, m, nil)
	}
	return s.flash(ctx, ws, g, l, "info", "manage.seatsSaved", i18n.Args{"count": n})
}

// TeamBilling opens the buyer's Stripe portal (invoices, card, billing
// details).
func (s *Service) TeamBilling(ctx context.Context, ws *store.Workspace, token string, returnURL string, l i18n.Lang) (TeamResult, error) {
	g, buyer, done, err := s.teamAction(ctx, ws, token, l)
	if err != nil || done != nil {
		return deref(done), err
	}
	client, _, err := s.ClientFor(ctx, ws.ID, buyer.Mode)
	if err != nil || buyer.StripeCustomerID == "" {
		return s.flash(ctx, ws, g, l, "bad", "manage.billingDown", nil)
	}
	u, err := client.PortalSession(ctx, buyer.StripeCustomerID, payments.PortalFlow{ReturnURL: returnURL})
	if err != nil {
		return s.flash(ctx, ws, g, l, "bad", "manage.billingDown", nil)
	}
	return TeamResult{Redirect: u}, nil
}

// SendTeamLink emails a team's buyer a fresh manage link (System mail),
// at most one every ten minutes.
func (s *Service) SendTeamLink(ctx context.Context, ws *store.Workspace, g store.EarnGroup, l i18n.Lang) error {
	key := "team-link:" + g.ID.String()
	if !s.linkThrottle.allow(key, s.now(), 10*time.Minute) {
		return nil
	}
	buyer, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: g.SubscriptionID, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	tier := s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID)
	return s.notice(ctx, ws, g.OwnerContactID, l, noticeText{subject: "team.link.subject", lead: "team.link.lead", button: "team.link.button",
		args: i18n.Args{"team": g.Name, "tier": firstNonEmpty(tier, "none")}, url: s.manageURL(ctx, ws, g), kind: "team_link"})
}

// noticeText is a team notice's words (catalog keys) and its link.
type noticeText struct {
	subject, lead, button string
	args                  i18n.Args
	url, kind             string
}

// notice sends a team notice in System mail, in a language: the
// publisher's branding, one button.
func (s *Service) notice(ctx context.Context, ws *store.Workspace, contact uuid.UUID, l i18n.Lang, n noticeText) error {
	var m mailer.Sender
	if s.Mail != nil {
		m = s.Mail()
	}
	if m == nil || !m.SystemAvailable(ctx, &ws.ID) {
		return errNoSystemMail
	}
	c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return err
	}
	b := s.brand(ctx, ws)
	esc := html.EscapeString
	greeting := s.t(l, "portal.email.greeting", i18n.Args{"name": firstNonEmpty(strings.TrimSpace(c.FirstName), "none")})
	subject, lead, button := s.t(l, n.subject, n.args), s.t(l, n.lead, n.args), s.t(l, n.button, n.args)
	text := greeting + "\n\n" + lead + "\n\n" + button + ": " + n.url + "\n\n" + b.Publisher + "\n"
	body := `<p style="margin:0 0 12px">` + esc(greeting) + `</p>` +
		`<p style="margin:0 0 20px">` + esc(lead) + `</p>` +
		`<p style="margin:0 0 16px"><a href="` + esc(n.url) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(b.Accent) +
		`;color:` + esc(b.OnAccent) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(button) + `</a></p>`
	_, err = m.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: c.Email, ToName: strings.TrimSpace(c.FirstName + " " + c.LastName),
		Subject: subject, Text: text, HTML: body, Kind: n.kind, Lang: l})
	return err
}

// sendNotice sends a system notice queued as a member email (the seats a
// team is short of), in the buyer's language.
func (s *Service) sendNotice(ctx context.Context, ws *store.Workspace, m memberEmail) string {
	l := s.contactLang(ctx, ws, m.contact)
	var err error
	switch m.email {
	case "team_seats_over":
		url, _ := m.data["manage"].(string)
		err = s.notice(ctx, ws, m.contact, l, noticeText{subject: "team.over.subject", lead: "team.over.lead", button: "team.over.button",
			args: i18n.Args{"team": m.data["team"], "tier": m.data["tier"], "seats": m.data["seats"], "count": m.data["over"]}, url: url, kind: "team_seats_over"})
	}
	if err != nil {
		s.log().WarnContext(ctx, "team notice", "notice", m.email, "err", err)
		return "not_sent"
	}
	return "sent"
}

// TeamJoinView is an invite's join page (PublicTeamJoin).
type TeamJoinView struct {
	// State is ask, accepted, full, expired, ended or bad.
	State   string
	Lang    i18n.Lang
	Heading string
	Text    string
	Email   string
	Consent string
	Button  string
	// Ignore says until when the invite works; Note follows an accepted
	// seat's benefits.
	Ignore string
	Note   string
	// Included titles the tier's benefits.
	Included string
	Benefits []string
}

// joinSeat verifies an invite link: the seat's newest one. An accepted
// seat's link keeps working; an invited one's until it expires.
func (s *Service) joinSeat(ctx context.Context, ws *store.Workspace, token string) (store.EarnGroupSeat, store.EarnGroup, error) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return store.EarnGroupSeat{}, store.EarnGroup{}, ErrTeamLink
	}
	c, err := s.Links.Verify(token, TokenGroupJoin)
	if err != nil || c.WorkspaceID != ws.ID {
		if errors.Is(err, linktoken.ErrSignature) {
			return store.EarnGroupSeat{}, store.EarnGroup{}, ErrLinkExpired
		}
		return store.EarnGroupSeat{}, store.EarnGroup{}, ErrTeamLink
	}
	h := sha256.Sum256([]byte(token))
	seat, err := s.Q.EarnGroupSeatByInvite(ctx, h[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return store.EarnGroupSeat{}, store.EarnGroup{}, ErrLinkExpired
	}
	if err != nil {
		return seat, store.EarnGroup{}, err
	}
	if seat.ID != c.MessageID || seat.WorkspaceID != ws.ID {
		return store.EarnGroupSeat{}, store.EarnGroup{}, ErrTeamLink
	}
	g, err := s.Q.EarnGroup(ctx, store.EarnGroupParams{ID: seat.GroupID, WorkspaceID: ws.ID})
	if err != nil {
		return seat, g, err
	}
	if seat.Status == "invited" && seat.InviteExpiresAt != nil && !s.now().Before(*seat.InviteExpiresAt) {
		return seat, g, ErrLinkExpired
	}
	return seat, g, nil
}

// TeamJoin builds an invite's join page.
func (s *Service) TeamJoin(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (TeamJoinView, error) {
	seat, g, err := s.joinSeat(ctx, ws, token)
	v := TeamJoinView{Lang: l, Benefits: []string{}, Included: s.t(l, "join.included", nil)}
	switch {
	case errors.Is(err, ErrLinkExpired):
		v.State, v.Heading, v.Text = "expired", s.t(l, "join.expiredTitle", nil), s.t(l, "join.expiredText", nil)
		return v, nil
	case errors.Is(err, ErrTeamLink):
		v.State, v.Heading, v.Text = "bad", s.t(l, "team.badTitle", nil), s.t(l, "join.badText", nil)
		return v, nil
	case err != nil:
		return v, err
	}
	buyer, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: g.SubscriptionID, WorkspaceID: ws.ID})
	if err != nil {
		return v, err
	}
	tier, short, lists := "", s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID), []string{}
	if buyer.TierID != nil {
		if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *buyer.TierID, WorkspaceID: ws.ID}); err == nil {
			tier = t.Name
			if t.Benefits != nil {
				v.Benefits = t.Benefits
			}
			if names, err := s.Q.EarnListNames(ctx, t.ListIds); err == nil && names != nil {
				lists = names
			}
		}
	}
	a := i18n.Args{"team": g.Name, "tier": firstNonEmpty(tier, short), "email": seat.Email, "lists": i18n.AndList(l.Locale, lists)}
	switch {
	case buyer.EndedAt != nil || g.EndedAt != nil || !paidAccess(buyer):
		v.State, v.Heading, v.Text = "ended", s.t(l, "join.endedTitle", a), s.t(l, "join.endedText", a)
		v.Benefits = []string{}
		return v, nil
	case seat.Status == "active":
		first := ""
		if seat.ContactID != nil {
			if c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: *seat.ContactID, WorkspaceID: ws.ID}); err == nil {
				first = strings.TrimSpace(c.FirstName)
			}
		}
		v.State = "accepted"
		v.Heading = s.t(l, "join.acceptedTitle", i18n.Args{"tier": firstNonEmpty(short, tier), "name": firstNonEmpty(first, "none")})
		v.Text = s.t(l, "join.acceptedText", i18n.Args{"team": g.Name, "next": s.nextIssue(ctx, ws, buyer.TierID, l)})
		v.Note = s.t(l, "join.acceptedNote", a)
		return v, nil
	}
	expires := ""
	if seat.InviteExpiresAt != nil {
		expires = s.dateTextIn(l, ws, *seat.InviteExpiresAt)
	}
	if people, err := s.Q.EarnGroupPeople(ctx, g.ID); err == nil && int(people.Active)+1 >= int(g.Seats) {
		buyerName := g.Name
		if c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID}); err == nil {
			buyerName = firstNonEmpty(strings.TrimSpace(c.FirstName+" "+c.LastName), c.Email)
		}
		v.State = "full"
		v.Heading = s.t(l, "join.fullTitle", a)
		v.Text = s.t(l, "join.fullText", i18n.Args{"team": g.Name, "seats": g.Seats, "tier": firstNonEmpty(tier, short), "buyer": buyerName, "date": expires})
		v.Benefits = []string{}
		return v, nil
	}
	inviter := g.Name
	if c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID}); err == nil {
		inviter = firstNonEmpty(strings.TrimSpace(c.FirstName+" "+c.LastName), g.Name)
	}
	v.State, v.Email = "ask", seat.Email
	v.Heading = s.t(l, "join.heading", a)
	v.Text = s.t(l, "join.text", i18n.Args{"inviter": inviter, "email": seat.Email, "team": g.Name})
	v.Consent = s.t(l, "join.consent", a)
	v.Button = s.t(l, "join.accept", nil)
	if expires != "" {
		v.Ignore = s.t(l, "join.ignore", i18n.Args{"date": expires})
	}
	return v, nil
}

// JoinInput is the evidence of an invite's acceptance.
type JoinInput struct {
	IP        string
	UserAgent string
}

// AcceptTeamInvite accepts an invite: the address joins the tier's lists
// with this acceptance as its consent (single opt-in), and gets a seat's
// access row that follows the buyer's.
func (s *Service) AcceptTeamInvite(ctx context.Context, ws *store.Workspace, token string, in JoinInput, l i18n.Lang) (TeamJoinView, error) {
	seat, g, err := s.joinSeat(ctx, ws, token)
	if err != nil {
		return s.TeamJoin(ctx, ws, token, l)
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.EarnGroupForUpdate(ctx, store.EarnGroupForUpdateParams{ID: g.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		seat, err := q.EarnGroupSeatForUpdate(ctx, store.EarnGroupSeatForUpdateParams{ID: seat.ID, GroupID: g.ID})
		if err != nil || seat.Status != "invited" {
			return errJoinDone
		}
		buyer, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: g.SubscriptionID, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if buyer.EndedAt != nil || g.EndedAt != nil || !paidAccess(buyer) {
			return errJoinDone
		}
		var lists []uuid.UUID
		if buyer.TierID != nil {
			if t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *buyer.TierID, WorkspaceID: ws.ID}); err == nil {
				lists = t.ListIds
			}
		}
		people, err := q.EarnGroupPeople(ctx, g.ID)
		if err != nil {
			return err
		}
		if int(people.Active)+1 >= int(g.Seats) {
			// Stripe lowered the seats: the page says the team is full.
			return errJoinDone
		}
		short := s.tierShortName(ctx, q, ws.ID, buyer.TierID)
		page, err := s.TeamJoin(ctx, ws, token, l)
		if err != nil {
			return err
		}
		consent := page.Consent
		contact := uuid.Nil
		if s.Signup != nil && buyer.Mode == "live" && len(lists) > 0 {
			contact, _, err = s.Signup.Join(ctx, tx, ws, Join{Email: seat.Email, Lists: lists, Source: SourceGroupPrefix + g.ID.String(),
				Note: "Team seat in " + g.Name, ConsentText: consent, IP: in.IP, UserAgent: in.UserAgent, Language: string(l.Locale), At: now})
			if err != nil {
				return err
			}
		}
		if contact == uuid.Nil {
			if c, err := q.EarnContactByEmail(ctx, store.EarnContactByEmailParams{WorkspaceID: ws.ID, Email: seat.Email}); err == nil {
				contact = c.ID
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			} else if contact, err = q.EarnContactInsert(ctx, store.EarnContactInsertParams{ID: newID(), WorkspaceID: ws.ID, Email: seat.Email,
				MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(seat.Email)), Source: SourceGroupPrefix + g.ID.String(),
				RoleAddress: emailaddr.IsRole(seat.Email), At: now}); err != nil {
				return err
			}
		}
		sub := newID()
		detail := []byte(`{"group":"` + g.ID.String() + `"}`)
		if err := q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: sub, WorkspaceID: ws.ID, ContactID: contact, Mode: buyer.Mode, Kind: KindGroup,
			TierID: buyer.TierID, Status: StatusGroup, Currency: buyer.Currency, StartedAt: now, SourceKind: "group", SourceLabel: g.Name, SourceDetail: detail,
			At: now, GroupID: &g.ID, PausedAt: buyer.PausedAt, PauseResumesAt: buyer.PauseResumesAt}); err != nil {
			return err
		}
		if err := q.EarnGroupSeatAccept(ctx, store.EarnGroupSeatAcceptParams{ID: seat.ID, ContactID: &contact, SubscriptionID: &sub, At: now}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, contact, &sub, buyer.Mode, "team_joined", now, nil, "", nil, "", nil, map[string]any{"team": g.Name, "tier": short}); err != nil {
			return err
		}
		return s.timeline(ctx, q, ws.ID, buyer.ContactID, &buyer.ID, buyer.Mode, "team_accepted", now, nil, "", nil, "", nil,
			map[string]any{"team": g.Name, "email": seat.Email})
	})
	if err != nil && !errors.Is(err, errJoinDone) {
		return TeamJoinView{}, err
	}
	return s.TeamJoin(ctx, ws, token, l)
}

var errJoinDone = errors.New("earn: the invite was used or the team ended")

// SourceGroupPrefix is the consent source of a team seat ("group:<team id>").
const SourceGroupPrefix = "group:"

// contactLang is a contact's language for System mail, the workspace's
// when they have none.
func (s *Service) contactLang(ctx context.Context, ws *store.Workspace, contact uuid.UUID) i18n.Lang {
	var code string
	_ = s.Pool.QueryRow(ctx, `SELECT coalesce(language, '') FROM contacts WHERE id = $1 AND workspace_id = $2`, contact, ws.ID).Scan(&code)
	if code != "" {
		return pagelabels.Lang(code)
	}
	return workspaceLang(ctx, s.Q, ws.ID)
}

// TeamPageStatus is a team page's HTTP status for a result's state.
func TeamPageStatus(state string) int {
	switch state {
	case "expired", "ended":
		return http.StatusGone
	case "bad":
		return http.StatusNotFound
	}
	return http.StatusOK
}
