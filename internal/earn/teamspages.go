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

// TeamRequest is a visit of a tier's team page.
type TeamRequest struct {
	Tier      string
	Period    string
	Seats     int
	Canceled  bool
	TestToken string
	Token     string
	Lang      i18n.Lang
}

// TeamView is a tier's team page.
type TeamView struct {
	Lang        i18n.Lang
	TestMode    bool
	Unavailable bool
	TierSlug    string
	TierName    string
	Heading     string
	Intro       string
	Benefits    []string
	Toggle      bool
	Period      string
	Monthly     string
	Yearly      string
	SeatsMin    int
	SeatsMax    int
	Seats       int
	// UnitCents is the price of a seat for the period; UnitText says it
	// ("$60 per seat a year") and TotalText the seats' total.
	UnitCents int64
	Currency  string
	UnitText  string
	TotalText string
	// TotalPattern is TotalText with {total} and {seats} for the page's
	// script to fill in as the seats change.
	TotalPattern string
	Team         string
	Email        string
	Errors       map[string]string
	Notice       *PageNotice
	TaxNote      string
	Words        map[string]string
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
	v := TeamView{Lang: l, TestMode: mode == "test", Unavailable: !ok, TierSlug: t.Slug, TierName: t.Name, Benefits: t.Benefits,
		Heading: s.t(l, "team.heading", i18n.Args{"tier": short}), Intro: s.t(l, "team.intro", i18n.Args{"tier": short}),
		Monthly: s.t(l, "page.monthly", nil), Yearly: s.t(l, "page.yearly", nil), Errors: map[string]string{},
		Words: s.teamWords(l)}
	if v.Benefits == nil {
		v.Benefits = []string{}
	}
	_, hasMonth := gp["month"]
	_, hasYear := gp["year"]
	v.Toggle = hasMonth && hasYear
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
		v.Seats = min(max(5, v.SeatsMin), v.SeatsMax)
	}
	v.UnitCents, v.Currency = p.AmountCents, p.Currency
	v.UnitText = s.t(l, "team.perSeat", i18n.Args{"price": publicPriceIn(l, p.AmountCents, p.Currency, v.Period)})
	v.TotalText = s.teamTotal(l, p, v.Period, v.Seats)
	v.TotalPattern = s.t(l, "team.total."+v.Period, i18n.Args{"total": "{total}", "seats": 2})
	v.TotalPattern = strings.Replace(v.TotalPattern, "2", "{seats}", 1)
	if r.Canceled {
		v.Notice = &PageNotice{Tone: "info", Title: s.t(l, "page.canceledTitle", nil), Text: s.t(l, "page.canceledText", nil)}
	}
	if ct, _, ok := s.readerOf(ctx, ws, r.Token); ok {
		v.Email = ct.Email
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err == nil {
		key := "page.taxExclusive"
		if st.PricesIncludeTax {
			key = "page.taxInclusive"
		}
		v.TaxNote = s.t(l, key, i18n.Args{"currency": currencyNameIn(l, firstNonEmpty(p.Currency, "usd"))})
	}
	return v, nil
}

// teamTotal is "$600 a year for 10 seats".
func (s *Service) teamTotal(l i18n.Lang, p store.EarnPrice, period string, seats int) string {
	return s.t(l, "team.total."+period, i18n.Args{"total": money(p.AmountCents*int64(seats), p.Currency, true), "seats": seats})
}

func (s *Service) teamWords(l i18n.Lang) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"seats", "fewer", "more", "teamName", "teamNameHint", "email", "emailHint", "continue", "unavailableTitle", "unavailableText",
		"billingPeriod", "back", "testRibbon", "testNote", "secure"} {
		out[k] = s.t(l, "team.word."+k, nil)
	}
	return out
}

// TeamFormSentence is a form error's sentence in a language.
func TeamFormSentence(l i18n.Lang, e *TeamFormError) string { return msgs.M(e.Key).Text(l, e.Args) }

// TeamManageView is a team buyer's manage page.
type TeamManageView struct {
	// State is manage, expired (a fresh link was emailed), ended or bad.
	State   string
	Lang    i18n.Lang
	Heading string
	Line    string
	Team    string
	Tier    string
	Seats   int
	Used    int
	Free    int
	// Over is how many people to remove when Stripe lowered the seats
	// below the people in the team.
	Over    int
	Members []TeamMemberView
	// CanInvite: free seats and paying (not paused or ended).
	CanInvite bool
	// SeatsMin and SeatsMax bound Change seats; CanChange when the
	// provider can change the quantity.
	SeatsMin  int
	SeatsMax  int
	CanChange bool
	Flash     *PageNotice
	Words     map[string]string
}

// TeamMemberView is one person on the manage page.
type TeamMemberView struct {
	ID     string
	Email  string
	Name   string
	Status string // buyer, invited, active
	Label  string
	Since  string
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
func (s *Service) TeamManage(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (TeamManageView, error) {
	g, buyer, err := s.teamOf(ctx, ws, token)
	switch {
	case errors.Is(err, ErrLinkExpired) && g.ID != uuid.Nil:
		if err := s.SendTeamLink(ctx, ws, g, l); err != nil && !errors.Is(err, errNoSystemMail) {
			s.log().WarnContext(ctx, "team link", "err", err)
		}
		return TeamManageView{State: "expired", Lang: l, Heading: s.t(l, "team.expiredTitle", nil), Line: s.t(l, "team.expiredText", nil), Words: s.manageWords(l)}, nil
	case errors.Is(err, ErrLinkExpired), errors.Is(err, ErrTeamLink):
		return TeamManageView{State: "bad", Lang: l, Heading: s.t(l, "team.badTitle", nil), Line: s.t(l, "team.badText", nil), Words: s.manageWords(l)}, nil
	case err != nil:
		return TeamManageView{}, err
	}
	return s.manageView(ctx, ws, g, buyer, l)
}

func (s *Service) manageWords(l i18n.Lang) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"invite", "inviteHint", "inviteButton", "people", "resend", "remove", "removeConfirm", "removeKeep", "changeSeats", "seatsLabel",
		"saveSeats", "billing", "billingHint", "statusBuyer", "statusInvited", "statusActive", "you", "full", "paused", "ended"} {
		out[k] = s.t(l, "manage.word."+k, nil)
	}
	return out
}

// manageView is the manage page of a team.
func (s *Service) manageView(ctx context.Context, ws *store.Workspace, g store.EarnGroup, buyer store.PaidSubscription, l i18n.Lang) (TeamManageView, error) {
	short := s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID)
	v := TeamManageView{State: "manage", Lang: l, Team: g.Name, Tier: short, Seats: int(g.Seats), Members: []TeamMemberView{}, Words: s.manageWords(l)}
	if buyer.EndedAt != nil || g.EndedAt != nil {
		v.State = "ended"
		v.Heading = s.t(l, "team.endedTitle", i18n.Args{"team": g.Name})
		v.Line = s.t(l, "team.endedText", i18n.Args{"tier": firstNonEmpty(short, "none")})
		return v, nil
	}
	buyerContact, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: buyer.ContactID, WorkspaceID: ws.ID})
	if err != nil {
		return v, err
	}
	v.Members = append(v.Members, TeamMemberView{ID: buyer.ID.String(), Email: buyerContact.Email, Name: strings.TrimSpace(buyerContact.FirstName + " " + buyerContact.LastName),
		Status: "buyer", Label: v.Words["statusBuyer"]})
	rows, err := s.Q.EarnGroupSeats(ctx, g.ID)
	if err != nil {
		return v, err
	}
	for _, r := range rows {
		m := TeamMemberView{ID: r.ID.String(), Email: r.Email, Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Status: r.Status}
		if r.Status == "active" {
			m.Label = v.Words["statusActive"]
			if r.AcceptedAt != nil {
				m.Since = s.t(l, "manage.joined", i18n.Args{"date": s.dateTextIn(l, ws, *r.AcceptedAt)})
			}
		} else {
			m.Label = v.Words["statusInvited"]
			m.Since = s.t(l, "manage.invited", i18n.Args{"date": s.dateTextIn(l, ws, r.InviteSentAt)})
		}
		v.Members = append(v.Members, m)
	}
	v.Used = len(v.Members)
	v.Free = max(v.Seats-v.Used, 0)
	v.Over = max(v.Used-v.Seats, 0)
	v.Heading = s.t(l, "manage.heading", i18n.Args{"team": g.Name, "tier": firstNonEmpty(short, "none"), "used": v.Used, "seats": v.Seats})
	paying := buyer.PausedAt == nil && paidAccess(buyer)
	v.CanInvite = v.Free > 0 && paying
	switch {
	case v.Over > 0:
		v.Line = s.t(l, "manage.over", i18n.Args{"count": v.Over, "seats": v.Seats})
	case buyer.PausedAt != nil:
		v.Line = s.t(l, "manage.pausedLine", nil)
	case v.Free == 0:
		v.Line = s.t(l, "manage.fullLine", nil)
	default:
		v.Line = s.t(l, "manage.freeLine", i18n.Args{"count": v.Free})
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
		v, verr := s.TeamManage(ctx, ws, token, l)
		if verr != nil {
			return g, buyer, nil, verr
		}
		return g, buyer, &TeamResult{View: v}, nil
	}
	if buyer.EndedAt != nil || g.EndedAt != nil {
		v, err := s.manageView(ctx, ws, g, buyer, l)
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
	v, err := s.manageView(ctx, ws, fresh, buyer, l)
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
	n, err := strconv.Atoi(strings.TrimSpace(seats))
	v, verr := s.manageView(ctx, ws, g, buyer, l)
	if verr != nil {
		return TeamResult{}, verr
	}
	switch {
	case err != nil || n < v.SeatsMin || n > v.SeatsMax:
		return s.flash(ctx, ws, g, l, "bad", "manage.seatsRange", i18n.Args{"min": v.SeatsMin, "max": v.SeatsMax})
	case n < v.Used:
		return s.flash(ctx, ws, g, l, "bad", "manage.removeFirst", i18n.Args{"count": v.Used - n})
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

// TeamJoinView is an invite's join page.
type TeamJoinView struct {
	// State is ask, accepted, expired, full, ended or bad.
	State   string
	Lang    i18n.Lang
	Heading string
	Text    string
	Email   string
	Consent string
	Button  string
	// Benefits are the tier's; Read links to the archive once accepted.
	Benefits []string
	Read     string
	Words    map[string]string
}

// joinSeat verifies an invite link: the seat's newest one, still within
// its time.
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
	if seat.InviteExpiresAt != nil && !s.now().Before(*seat.InviteExpiresAt) {
		return seat, g, ErrLinkExpired
	}
	return seat, g, nil
}

func (s *Service) joinWords(l i18n.Lang) map[string]string {
	return map[string]string{"for": s.t(l, "join.for", nil), "read": s.t(l, "join.read", nil)}
}

// TeamJoin builds an invite's join page.
func (s *Service) TeamJoin(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (TeamJoinView, error) {
	seat, g, err := s.joinSeat(ctx, ws, token)
	v := TeamJoinView{Lang: l, Words: s.joinWords(l), Benefits: []string{}}
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
	short := s.tierShortName(ctx, s.Q, ws.ID, buyer.TierID)
	a := i18n.Args{"team": g.Name, "tier": firstNonEmpty(short, "none"), "publisher": s.publisher(ctx, ws)}
	if buyer.TierID != nil {
		if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *buyer.TierID, WorkspaceID: ws.ID}); err == nil && t.Benefits != nil {
			v.Benefits = t.Benefits
		}
	}
	switch {
	case buyer.EndedAt != nil || g.EndedAt != nil || !paidAccess(buyer):
		v.State, v.Heading, v.Text = "ended", s.t(l, "join.endedTitle", a), s.t(l, "join.endedText", a)
		return v, nil
	case seat.Status == "active":
		v.State, v.Heading, v.Text, v.Read = "accepted", s.t(l, "join.acceptedTitle", a), s.t(l, "join.acceptedText", a), s.pageURL(ctx, ws, "archive")
		return v, nil
	}
	v.State, v.Email = "ask", seat.Email
	v.Heading = s.t(l, "join.heading", a)
	v.Text = s.t(l, "join.text", a)
	v.Consent = s.t(l, "join.consent", a)
	v.Button = s.t(l, "join.accept", nil)
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
		short := s.tierShortName(ctx, q, ws.ID, buyer.TierID)
		consent := s.t(l, "join.consent", i18n.Args{"team": g.Name, "tier": firstNonEmpty(short, "none"), "publisher": s.publisher(ctx, ws)})
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
