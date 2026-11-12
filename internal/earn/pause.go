package earn

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Pausing (M65, G-39): a Stripe subscription's payments pause until a
// date (Stripe's pause_collection, invoices voided), when Stripe resumes
// them on its own; "Resume now" resumes early. While paused the member has
// no paid access (the free membership stays) and the subscription counts
// nothing in MRR, with the ledger's paused and resumed movements instead of
// churn. A member pauses from the cancel page, for the lengths Earn ›
// Settings allows, once in 12 months; the team pauses from the member
// panel for 1, 3 or 6 months or until a date at most 6 months away.

// PauseMonths are the lengths a workspace may allow on the cancel page.
var PauseMonths = []int{1, 3, 6}

// MaxPause is the longest pause the team may set.
const MaxPause = 6

// PauseEvery is how often a member may pause from the cancel page.
const PauseEvery = 365 * 24 * time.Hour

// ErrCantPause means the subscription can't be paused now.
var ErrCantPause = errors.New("earn: the subscription can't be paused")

// billableOf is the subscription a contact pays through Stripe and that
// hasn't ended, paused or not (billing links, the cancel and resume
// pages); live first.
func (s *Service) billableOf(ctx context.Context, ws *store.Workspace, contact uuid.UUID) (*access, error) {
	rows, err := s.Q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	var out *access
	for _, r := range rows {
		p := r.PaidSubscription
		if p.Kind != "stripe" || p.EndedAt != nil || p.StripeSubscriptionID == "" || p.Status == StatusIncomplete {
			continue
		}
		if out == nil || (out.Mode != "live" && p.Mode == "live") {
			out = &access{PaidSubscription: p, TierName: r.TierName}
		}
	}
	return out, nil
}

// pauserFor opens the subscription's provider client that can pause.
func (s *Service) pauserFor(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription) (payments.Pauser, store.PaymentConnection, error) {
	client, conn, err := s.ClientFor(ctx, ws.ID, sub.Mode)
	if err != nil {
		return nil, conn, ErrUnavailable
	}
	p, ok := client.(payments.Pauser)
	if !ok {
		return nil, conn, ErrUnavailable
	}
	return p, conn, nil
}

// pausable reports whether a subscription can be paused at all: an
// active Stripe subscription (not trialing or past due) that isn't paused,
// ending or canceled.
func pausable(sub store.PaidSubscription) bool {
	return sub.Kind == "stripe" && sub.StripeSubscriptionID != "" && sub.EndedAt == nil && sub.PausedAt == nil && sub.CancelAt == nil &&
		sub.Status == StatusActive
}

// CancelPauseChoices are the pause lengths a member's cancel page offers:
// the workspace's allowed lengths, for a subscription that can pause and
// hasn't paused in the last 12 months (none otherwise).
func (s *Service) CancelPauseChoices(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription) []int {
	if !pausable(sub) || (sub.LastPausedAt != nil && s.now().Sub(*sub.LastPausedAt) < PauseEvery) {
		return nil
	}
	if _, _, err := s.pauserFor(ctx, ws, sub); err != nil {
		return nil
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return nil
	}
	out := []int{}
	for _, m := range PauseMonths {
		if slices.Contains(st.PauseMonths, int32(m)) {
			out = append(out, m)
		}
	}
	return out
}

// PauseFromCancel pauses a member's subscription from their cancel page
// for one of the allowed lengths.
func (s *Service) PauseFromCancel(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription, months int, visit uuid.UUID) (time.Time, error) {
	if !slices.Contains(s.CancelPauseChoices(ctx, ws, sub), months) {
		return time.Time{}, ErrCantPause
	}
	until := s.now().AddDate(0, months, 0)
	err := s.pause(ctx, ws, sub, until, nil, "cancel_page", func(q *store.Queries) error {
		if visit == uuid.Nil {
			return nil
		}
		return q.EarnCancelVisitPaused(ctx, store.EarnCancelVisitPausedParams{ID: visit, At: s.now()})
	})
	return until, err
}

// pause pauses a subscription at the provider, then here in one
// transaction (the timeline, the team's seats, the automation event, the
// MRR ledger's movement by its trigger), and sends the "paused" email.
func (s *Service) pause(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription, until time.Time, p *authz.Principal, source string, more func(q *store.Queries) error) error {
	return s.pauseWith(ctx, ws, sub, until, p, source, "", true, more)
}

// pauseWith is pause with a note for the timeline, and the email or not.
func (s *Service) pauseWith(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription, until time.Time, p *authz.Principal, source, note string, email bool,
	more func(q *store.Queries) error) error {
	pauser, conn, err := s.pauserFor(ctx, ws, sub)
	if err != nil {
		return err
	}
	now := s.now()
	if err := pauser.Pause(ctx, sub.StripeSubscriptionID, until); err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		return keyProblem(ctx, err)
	}
	var emails []memberEmail
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before := sub
		if err := q.EarnSubscriptionSetPause(ctx, store.EarnSubscriptionSetPauseParams{ID: sub.ID, PausedAt: &now, PauseResumesAt: &until, At: now}); err != nil {
			return err
		}
		data := map[string]any{"until": until.UTC().Format(time.RFC3339), "source": source}
		if note != "" {
			data["note"] = note
		}
		if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, "paused", now, nil, "", nil, "", userID(p), data); err != nil {
			return err
		}
		m, err := s.afterWrite(ctx, q, ws, sub.Mode, &before, sub.ID, now, nil)
		if err != nil {
			return err
		}
		emails = m
		if more != nil {
			if err := more(q); err != nil {
				return err
			}
		}
		if p != nil {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.subscription_paused", TargetType: "contact", TargetID: sub.ContactID.String(),
				After: map[string]any{"until": until.UTC().Format("2006-01-02"), "subscription": sub.StripeSubscriptionID}})
		}
		return nil
	})
	if err != nil {
		return err
	}
	row, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sub.ID, WorkspaceID: ws.ID})
	if err == nil && email {
		emails = append(emails, s.pausedEmail(ctx, ws, row, now))
	}
	for _, m := range emails {
		s.sendMemberEmail(ctx, ws, sub.Mode, m, nil)
	}
	return nil
}

// pausedEmail is the "paused" member email of a subscription.
func (s *Service) pausedEmail(ctx context.Context, ws *store.Workspace, row store.PaidSubscription, at time.Time) memberEmail {
	short := s.tierShortName(ctx, s.Q, ws.ID, row.TierID)
	m := s.memberEmailFor(EmailPaused, row, ws, short, at)
	stamp := at
	if row.PausedAt != nil {
		stamp = *row.PausedAt
	}
	m.key = "earn:" + row.ID.String() + ":paused:" + stamp.UTC().Format(time.RFC3339)
	return m
}

// resume resumes a paused subscription at the provider, then here.
func (s *Service) resume(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription, p *authz.Principal, source string) error {
	if sub.PausedAt == nil || sub.EndedAt != nil {
		return ErrCantPause
	}
	pauser, conn, err := s.pauserFor(ctx, ws, sub)
	if err != nil {
		return err
	}
	now := s.now()
	if err := pauser.Resume(ctx, sub.StripeSubscriptionID); err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		return keyProblem(ctx, err)
	}
	var emails []memberEmail
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before := sub
		if err := q.EarnSubscriptionSetPause(ctx, store.EarnSubscriptionSetPauseParams{ID: sub.ID, At: now}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, "resumed", now, nil, "", nil, "", userID(p), map[string]any{"source": source}); err != nil {
			return err
		}
		m, err := s.afterWrite(ctx, q, ws, sub.Mode, &before, sub.ID, now, nil)
		if err != nil {
			return err
		}
		emails = m
		if p != nil {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.subscription_resumed", TargetType: "contact", TargetID: sub.ContactID.String(),
				Before: map[string]any{"until": dateOrEmpty(sub.PauseResumesAt)}, After: map[string]any{"subscription": sub.StripeSubscriptionID}})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, m := range emails {
		s.sendMemberEmail(ctx, ws, sub.Mode, m, nil)
	}
	return nil
}

// EarnPauseInput pauses a paid member's payments: for 1, 3 or 6 months,
// or until a date at most 6 months away.
type EarnPauseInput struct {
	Months int    `json:"months,omitempty" enum:"0,1,3,6"`
	Until  string `json:"until,omitempty" format:"date" doc:"YYYY-MM-DD, in the workspace's time zone; at most 6 months away"`
	// Note goes on the member's timeline ("Asked by email: travelling").
	Note string `json:"note,omitempty" maxLength:"300"`
	// SkipEmail doesn't send the "Membership paused" email.
	SkipEmail bool `json:"skipEmail,omitempty"`
}

// PauseMember pauses a paid member's payments from the member panel.
func (s *Service) PauseMember(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnPauseInput) (EarnMember, error) {
	sub, err := s.memberSub(ctx, ws, id)
	if err != nil {
		return EarnMember{}, err
	}
	if !pausable(sub) {
		return EarnMember{}, problem(ctx, http.StatusConflict, "not_pausable", "pause.notPausable", nil)
	}
	now := s.now()
	var until time.Time
	switch {
	case in.Until != "":
		d, err := time.ParseInLocation("2006-01-02", in.Until, location(ws))
		if err != nil {
			return EarnMember{}, invalid(ctx, "body.until", "pause.until", nil)
		}
		until = d.Add(9 * time.Hour)
		if !until.After(now.Add(24*time.Hour)) || until.After(now.AddDate(0, MaxPause, 1)) {
			return EarnMember{}, invalid(ctx, "body.until", "pause.until", nil)
		}
	case slices.Contains(PauseMonths, in.Months):
		until = now.AddDate(0, in.Months, 0)
	default:
		return EarnMember{}, invalid(ctx, "body.months", "pause.length", nil)
	}
	if err := s.pauseWith(ctx, ws, sub, until, p, "team", strings.TrimSpace(in.Note), !in.SkipEmail, nil); err != nil {
		if errors.Is(err, ErrUnavailable) {
			return EarnMember{}, problem(ctx, http.StatusConflict, CodeNotConnected, "cancel.notConnected", nil)
		}
		return EarnMember{}, err
	}
	return s.Member(ctx, ws, p, sub.ID.String())
}

// ResumeMember resumes a paused member's payments now.
func (s *Service) ResumeMember(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnMember, error) {
	sub, err := s.memberSub(ctx, ws, id)
	if err != nil {
		return EarnMember{}, err
	}
	if err := s.resume(ctx, ws, sub, p, "team"); err != nil {
		switch {
		case errors.Is(err, ErrCantPause):
			return EarnMember{}, problem(ctx, http.StatusConflict, "not_paused", "pause.notPaused", nil)
		case errors.Is(err, ErrUnavailable):
			return EarnMember{}, problem(ctx, http.StatusConflict, CodeNotConnected, "cancel.notConnected", nil)
		}
		return EarnMember{}, err
	}
	return s.Member(ctx, ws, p, sub.ID.String())
}

// memberSub loads a paid member's subscription by its id.
func (s *Service) memberSub(ctx context.Context, ws *store.Workspace, id string) (store.PaidSubscription, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return store.PaidSubscription{}, notFound(ctx, "paidMember")
	}
	sub, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return store.PaidSubscription{}, mapErr(ctx, err, "paidMember")
	}
	return sub, nil
}

// ResumeURL is a paused member's "Resume now" link (the paused email's):
// a billing link, working for LinkValidInEmail.
func (s *Service) ResumeURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID) string {
	return s.pageURL(ctx, ws, "resume/"+s.memberToken(ws.ID, TokenBilling, contact, LinkValidInEmail))
}

// ResumeView is the page of a "Resume now" link (PublicEarnStates).
type ResumeView struct {
	// State is ask (the page), resumed or none (nothing paused).
	State   string
	Heading string
	Text    string
	Email   string
	Button  string
	// Later says when it resumes on its own.
	Later string
}

// resumeTarget verifies a resume link and finds the paused subscription.
func (s *Service) resumeTarget(ctx context.Context, ws *store.Workspace, token string) (*access, store.EarnContactBriefRow, error) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return nil, store.EarnContactBriefRow{}, ErrBadLink
	}
	c, err := s.Links.Verify(token, TokenBilling)
	switch {
	case errors.Is(err, linktoken.ErrSignature):
		return nil, store.EarnContactBriefRow{}, ErrLinkExpired
	case err != nil || c.WorkspaceID != ws.ID || c.ContactID == uuid.Nil:
		return nil, store.EarnContactBriefRow{}, ErrBadLink
	case s.linkExpired(c):
		return nil, store.EarnContactBriefRow{}, ErrLinkExpired
	}
	ct, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: c.ContactID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, ct, ErrLinkExpired
	}
	acc, err := s.billableOf(ctx, ws, c.ContactID)
	return acc, ct, err
}

// ResumePage is the page of a resume link, in a language.
func (s *Service) ResumePage(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (ResumeView, error) {
	acc, ct, err := s.resumeTarget(ctx, ws, token)
	if err != nil {
		return ResumeView{}, err
	}
	if acc == nil || acc.PausedAt == nil {
		return ResumeView{State: "none", Heading: s.t(l, "resume.noneTitle", nil), Text: s.t(l, "resume.noneText", nil)}, nil
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	a := i18n.Args{"tier": firstNonEmpty(short, "none"), "price": publicPriceIn(l, acc.AmountCents*int64(max(acc.Quantity, 1)), acc.Currency, acc.Period),
		"next": s.nextIssue(ctx, ws, acc.TierID, l)}
	v := ResumeView{State: "ask", Email: ct.Email, Button: s.t(l, "resume.button", nil), Heading: s.t(l, "resume.titleNoDate", a)}
	if acc.PauseResumesAt != nil {
		a["date"] = longDateIn(l, ws, *acc.PauseResumesAt)
		v.Heading = s.t(l, "resume.title", a)
		v.Later = s.t(l, "resume.later", a)
	}
	// Stripe charges again at the end of the period that was paid before
	// the pause: the next invoice after today.
	if end := acc.CurrentPeriodEnd; end != nil && end.After(s.now()) {
		a["charge"] = s.dateTextIn(l, ws, *end)
		v.Text = s.t(l, "resume.text", a)
	} else {
		v.Text = s.t(l, "resume.textNoCharge", a)
	}
	return v, nil
}

// ResumeFromLink resumes a member's paused subscription from their link.
func (s *Service) ResumeFromLink(ctx context.Context, ws *store.Workspace, token string, l i18n.Lang) (ResumeView, error) {
	acc, _, err := s.resumeTarget(ctx, ws, token)
	if err != nil {
		return ResumeView{}, err
	}
	if acc == nil || acc.PausedAt == nil {
		return s.ResumePage(ctx, ws, token, l)
	}
	if err := s.resume(ctx, ws, acc.PaidSubscription, nil, "member"); err != nil {
		return ResumeView{}, err
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	return ResumeView{State: "resumed", Heading: s.t(l, "resume.doneTitle", i18n.Args{"tier": firstNonEmpty(short, "none")}),
		Text: s.t(l, "resume.doneText", nil)}, nil
}

// remindResuming sends "resuming soon" a week before Stripe resumes a
// paused subscription, once per resume date.
func (s *Service) remindResuming(ctx context.Context) error {
	now := s.now()
	rows, err := s.Q.EarnResumingSoon(ctx, store.EarnResumingSoonParams{Now: now, Until: now.Add(7 * 24 * time.Hour)})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := s.Q.EarnResumeNotified(ctx, store.EarnResumeNotifiedParams{ID: row.ID, ResumesAt: row.PauseResumesAt}); errors.Is(err, pgx.ErrNoRows) {
			continue
		} else if err != nil {
			return err
		}
		ws, err := s.Q.GetWorkspace(ctx, row.WorkspaceID)
		if err != nil {
			return err
		}
		short := s.tierShortName(ctx, s.Q, ws.ID, row.TierID)
		m := s.memberEmailFor(EmailResumingSoon, row, &ws, short, now)
		m.key = "earn:" + row.ID.String() + ":resuming:" + row.PauseResumesAt.UTC().Format(time.RFC3339)
		s.sendMemberEmail(ctx, &ws, row.Mode, m, nil)
	}
	return nil
}

// EarnPausingInput sets the pause lengths members may choose on the cancel
// page.
type EarnPausingInput struct {
	Months []int `json:"months" maxItems:"3" doc:"Any of 1, 3 and 6; none turns pausing off."`
}

// EarnPausing is Earn › Settings › Pausing.
type EarnPausing struct {
	Months []int `json:"months"`
}

// SavePausing saves the pause lengths members may choose.
func (s *Service) SavePausing(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnPausingInput) (EarnPausing, error) {
	months := []int32{}
	for _, m := range PauseMonths {
		if slices.Contains(in.Months, m) {
			months = append(months, int32(m))
		}
	}
	for _, m := range in.Months {
		if !slices.Contains(PauseMonths, m) {
			return EarnPausing{}, invalid(ctx, "body.months", "pause.length", nil)
		}
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnPausing{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.EarnSettingsSetPauseMonths(ctx, store.EarnSettingsSetPauseMonthsParams{WorkspaceID: ws.ID, PauseMonths: months, At: now, UpdatedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.pausing_updated", TargetType: "earn_settings", TargetID: ws.ID.String(),
			TargetName: "Pausing", Before: map[string]any{"months": st.PauseMonths}, After: map[string]any{"months": months}})
	})
	if err != nil {
		return EarnPausing{}, err
	}
	out := EarnPausing{Months: []int{}}
	for _, m := range months {
		out.Months = append(out.Months, int(m))
	}
	return out, nil
}
