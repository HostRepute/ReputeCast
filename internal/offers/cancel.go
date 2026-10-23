package offers

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The hosted cancel page (news.acme.io/cancel/{token}, a paid member's
// personal link): what happens, an optional reason, the retention offer
// when they're eligible, and two buttons of the same size. Continue to
// cancel opens the provider's portal on the subscription's cancel step;
// nothing changes until the reader confirms there.

// Reasons are the cancel page's reasons, as the overview sums them.
var Reasons = []string{"not_reading", "too_expensive", "not_for_me", "switching", "other"}

// CancelReason is one of the page's reasons.
type CancelReason struct {
	Value string
	Label string
}

// CancelOffer is the retention offer on the page.
type CancelOffer struct {
	Name   string
	Line   string
	Button string
}

// CancelView is the cancel page in one of its states: ask (the page),
// kept (took the offer), done (back from the provider), expired (the link)
// or none (nothing to cancel).
type CancelView struct {
	State   string
	Heading string
	Email   string
	Lead    string
	// PastDue: canceling ends access today; BillingURL updates the card.
	PastDue    bool
	BillingURL string
	Reasons    []CancelReason
	Offer      *CancelOffer
	// Keep is the keep button without an offer (none: no button).
	Continue string
	Note     string
	Text     string
	VisitID  string
	Words    map[string]string
}

// CancelResult is what deciding did: the page to show, or the provider's
// address to go to.
type CancelResult struct {
	View     *CancelView
	Redirect string
}

func words(ctx context.Context) map[string]string {
	return map[string]string{"for": page(ctx, "cancel.for", nil), "notYou": page(ctx, "cancel.notYou", nil), "why": page(ctx, "cancel.why", nil),
		"optional": page(ctx, "cancel.optional", nil), "changed": page(ctx, "cancel.changed", nil), "back": page(ctx, "cancel.back", nil),
		"updateCard": page(ctx, "cancel.updateCard", nil), "yourEmail": page(ctx, "cancel.yourEmail", nil), "emailMe": page(ctx, "cancel.emailMe", nil),
		"offerLabel": page(ctx, "cancel.offerLabel", nil), "terms": page(ctx, "cancel.terms", nil)}
}

func reasons(ctx context.Context) []CancelReason {
	out := make([]CancelReason, 0, len(Reasons))
	for _, r := range Reasons {
		out = append(out, CancelReason{Value: r, Label: page(ctx, "cancel.reason."+r, nil)})
	}
	return out
}

// freeName is what a paid member keeps getting ("the free Weekly Digest").
func freeName(t *earn.CancelTarget) string {
	if len(t.Lists) > 0 {
		return t.Lists[0]
	}
	return ""
}

// CancelPage is the page for a cancel link. done is the return from the
// provider's portal. Each visit is counted once per half hour.
func (s *Service) CancelPage(ctx context.Context, ws *store.Workspace, token string, done bool) (CancelView, error) {
	v := CancelView{Words: words(ctx)}
	t, err := s.Earn.CancelTargetOf(ctx, ws, token)
	switch {
	case errors.Is(err, earn.ErrLinkInvalid):
		return v, err
	case errors.Is(err, earn.ErrLinkExpired):
		v.State, v.Heading, v.Text = "expired", page(ctx, "cancel.expiredTitle", nil), page(ctx, "cancel.expiredText", nil)
		return v, nil
	case errors.Is(err, earn.ErrNoSubscription):
		v.State, v.Heading, v.Text = "none", page(ctx, "cancel.noneTitle", nil), page(ctx, "cancel.noneText", nil)
		return v, nil
	case err != nil:
		return v, err
	}
	loc := location(ws)
	end := t.Sub.CurrentPeriodEnd
	if t.Sub.CancelAt != nil {
		// Already canceled: the page says when it ends.
		end, done = t.Sub.CancelAt, true
	}
	if done && t.Sub.CancelAt == nil {
		// Back from the portal before its webhook: the end date as the
		// provider has it now.
		if sub, err := s.providerSub(ctx, ws, t.Sub); err == nil {
			switch {
			case sub.CancelAt != nil:
				end = sub.CancelAt
			case sub.CancelAtPeriodEnd:
				end = sub.CurrentPeriodEnd
			case sub.EndedAt != nil:
				end = sub.EndedAt
			default:
				// They didn't confirm: the page again.
				done = false
			}
		}
	}
	a := i18n.Args{"tier": t.Short, "free": freeName(t), "period": t.Sub.Period}
	if end != nil {
		a["date"] = dayOf(i18n.LangFrom(ctx), *end, loc, s.now())
	}
	if done {
		v.State, v.Heading = "done", page(ctx, "cancel.doneTitle", a)
		v.Text = page(ctx, "cancel.doneText", a)
		if freeName(t) == "" {
			v.Text = page(ctx, "cancel.doneTextNoList", a)
		}
		return v, nil
	}
	v.State, v.Email = "ask", t.Contact.Email
	v.Heading = page(ctx, "cancel.title", a)
	v.Reasons = reasons(ctx)
	v.Continue, v.Note = page(ctx, "cancel.continue", nil), page(ctx, "cancel.note", nil)
	switch {
	case t.Sub.Status == "past_due":
		v.PastDue, v.BillingURL = true, s.Earn.BillingURL(ctx, ws, t.Contact.ID)
		v.Lead = page(ctx, "cancel.leadPastDue", a)
	case end != nil && freeName(t) != "":
		v.Lead = page(ctx, "cancel.lead", a)
	case end != nil:
		v.Lead = page(ctx, "cancel.leadNoList", a)
	}
	mode := t.Sub.Mode
	ret, err := s.RetentionFor(ctx, ws, mode, t.Sub)
	if err != nil {
		s.log().WarnContext(ctx, "offers: the retention offer", "err", err)
	}
	var offerID *uuid.UUID
	if ret != nil {
		v.Offer = s.cancelOffer(ctx, ret, t.Short)
		id := ret.ID
		offerID = &id
	}
	visit, err := s.visit(ctx, ws, t, offerID)
	if err != nil {
		return v, err
	}
	v.VisitID = visit.String()
	return v, nil
}

// dayOf is "2 Nov", with the year when it isn't this year's.
func dayOf(lang i18n.Lang, t time.Time, loc *time.Location, now time.Time) string {
	l := t.In(loc)
	if l.Year() == now.In(loc).Year() {
		return earn.DayMonthIn(lang, l)
	}
	return earn.DateIn(lang, l)
}

func (s *Service) cancelOffer(ctx context.Context, r *Retention, short string) *CancelOffer {
	a := i18n.Args{"price": moneyOf(r.PriceCents, r.Currency), "full": moneyOf(r.FullCents, r.Currency), "months": r.Months, "period": r.Period, "tier": short}
	key := "months"
	switch {
	case r.Months < 0:
		key = "forever"
	case r.Months == 0:
		key = "once"
	}
	return &CancelOffer{Name: r.Name, Line: page(ctx, "cancel.offer."+key, a), Button: page(ctx, "cancel.keep."+key, a)}
}

// visit counts a view of the page, once per half hour.
func (s *Service) visit(ctx context.Context, ws *store.Workspace, t *earn.CancelTarget, offer *uuid.UUID) (uuid.UUID, error) {
	now := s.now()
	contact := t.Contact.ID
	if v, err := s.Q.CancelVisitRecent(ctx, store.CancelVisitRecentParams{WorkspaceID: ws.ID, ContactID: &contact, Since: now.Add(-30 * time.Minute)}); err == nil {
		return v.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	id := newID()
	sub := t.Sub.ID
	err := s.Q.CancelVisitInsert(ctx, store.CancelVisitInsertParams{ID: id, WorkspaceID: ws.ID, ContactID: &contact, SubscriptionID: &sub, Mode: t.Sub.Mode,
		VisitedAt: now, OfferID: offer})
	return id, err
}

// providerSub reads the subscription from the provider.
func (s *Service) providerSub(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription) (payments.Subscription, error) {
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, sub.Mode)
	if err != nil {
		return payments.Subscription{}, err
	}
	return client.GetSubscription(ctx, sub.StripeSubscriptionID)
}

// CancelDecide takes the page's answer: keep (with the offer) or continue
// to the provider's cancel step. The reason, when given, goes with it.
func (s *Service) CancelDecide(ctx context.Context, ws *store.Workspace, token, visitID, reason, decision, returnURL string) (CancelResult, error) {
	t, err := s.Earn.CancelTargetOf(ctx, ws, token)
	if err != nil {
		v, perr := s.CancelPage(ctx, ws, token, false)
		return CancelResult{View: &v}, perr
	}
	if !slices.Contains(Reasons, reason) {
		reason = ""
	}
	vid, err := uuid.Parse(visitID)
	contact := t.Contact.ID
	var visit store.EarnCancelVisit
	if err == nil {
		visit, err = s.Q.CancelVisitGet(ctx, store.CancelVisitGetParams{ID: vid, WorkspaceID: ws.ID, ContactID: &contact})
	}
	if err != nil {
		// No visit of theirs: count this one.
		id, verr := s.visit(ctx, ws, t, nil)
		if verr != nil {
			return CancelResult{}, verr
		}
		visit = store.EarnCancelVisit{ID: id}
	}
	now := s.now()
	if decision == "keep" {
		ret, err := s.RetentionFor(ctx, ws, t.Sub.Mode, t.Sub)
		if err != nil {
			return CancelResult{}, err
		}
		if ret == nil {
			// No offer (taken one already, or it ended): nothing to keep with.
			v, err := s.CancelPage(ctx, ws, token, false)
			return CancelResult{View: &v}, err
		}
		sub, err := s.applyRetention(ctx, ws, t, ret)
		if err != nil {
			return CancelResult{}, err
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.CancelVisitDecide(ctx, store.CancelVisitDecideParams{ID: visit.ID, Reason: reason, Decision: "kept", DecidedAt: &now}); err != nil {
				return err
			}
			subID := t.Sub.ID
			return q.OfferUseInsert(ctx, store.OfferUseInsertParams{ID: newID(), WorkspaceID: ws.ID, OfferID: ret.ID, ContactID: contact, SubscriptionID: &subID,
				Mode: t.Sub.Mode, UsedAt: now, Source: "cancel_page"})
		}); err != nil {
			return CancelResult{}, err
		}
		v := s.keptView(ctx, ws, t, ret, sub)
		return CancelResult{View: &v}, nil
	}
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CancelVisitDecide(ctx, store.CancelVisitDecideParams{ID: visit.ID, Reason: reason, Decision: "continued", DecidedAt: &now}); err != nil {
			return err
		}
		if reason == "" {
			return nil
		}
		return q.CancelSetReason(ctx, store.CancelSetReasonParams{ID: t.Sub.ID, Reason: reason, At: now})
	}); err != nil {
		return CancelResult{}, err
	}
	to, err := s.Earn.CancelPortal(ctx, ws, t.Sub, returnURL)
	if err != nil {
		return CancelResult{}, err
	}
	return CancelResult{Redirect: to}, nil
}

// applyRetention puts the offer's coupon on the subscription at once.
func (s *Service) applyRetention(ctx context.Context, ws *store.Workspace, t *earn.CancelTarget, r *Retention) (payments.Subscription, error) {
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, t.Sub.Mode)
	if err != nil {
		return payments.Subscription{}, earn.ErrUnavailable
	}
	off, ok := client.(payments.Offers)
	if !ok || r.Coupon == "" {
		return payments.Subscription{}, earn.ErrUnavailable
	}
	return off.ApplyCoupon(ctx, t.Sub.StripeSubscriptionID, r.Coupon)
}

// keptView is the page after taking the offer: the new price, until when,
// and the next payment.
func (s *Service) keptView(ctx context.Context, ws *store.Workspace, t *earn.CancelTarget, r *Retention, sub payments.Subscription) CancelView {
	loc := location(ws)
	now := s.now()
	a := i18n.Args{"price": moneyOf(r.PriceCents, r.Currency), "full": moneyOf(r.FullCents, r.Currency), "tier": t.Short, "period": r.Period,
		"next": earn.MoneyCents(r.PriceCents, r.Currency)}
	until := now.AddDate(0, max(r.Months, 0), 0)
	if sub.Discount != nil && sub.Discount.End != nil {
		until = *sub.Discount.End
	}
	a["until"] = dayOf(i18n.LangFrom(ctx), until, loc, now)
	// A coupon doesn't move the period: the next payment is when it ends,
	// as the last webhook told us, else as the provider has it now.
	nextAt := t.Sub.CurrentPeriodEnd
	if nextAt == nil {
		nextAt = sub.CurrentPeriodEnd
	}
	if nextAt != nil {
		a["date"] = dayOf(i18n.LangFrom(ctx), *nextAt, loc, now)
	}
	key := "months"
	switch {
	case r.Months < 0:
		key = "forever"
	case r.Months == 0:
		key = "once"
	}
	v := CancelView{State: "kept", Words: words(ctx), Heading: page(ctx, "cancel.kept."+key, a)}
	if nextAt != nil {
		v.Text = page(ctx, "cancel.keptText", a)
	} else {
		v.Text = page(ctx, "cancel.keptTextNoDate", a)
	}
	return v
}
