package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Paid statuses (brief §2).
const (
	StatusIncomplete    = "incomplete"
	StatusTrialing      = "trialing"
	StatusActive        = "active"
	StatusPastDue       = "past_due"
	StatusCanceled      = "canceled"
	StatusComplimentary = "complimentary"
	StatusGift          = "gift"
)

func parseUUID(s string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(s)) }

// paidStatus maps the provider's subscription status onto paid status
// (brief 3.4): trialing; active, or trialing while a 100% discount makes
// it free; past due; canceled for canceled, unpaid and incomplete_expired;
// incomplete (a checkout not finished) is never shown.
func paidStatus(sub payments.Subscription, now time.Time) string {
	switch sub.Status {
	case "trialing":
		return StatusTrialing
	case "active":
		if d := sub.Discount; d != nil && d.PercentOff >= 100 && (d.End == nil || d.End.After(now)) {
			return StatusTrialing
		}
		return StatusActive
	case "past_due":
		return StatusPastDue
	case "incomplete":
		return StatusIncomplete
	}
	return StatusCanceled
}

// discounted is what a price costs after a discount.
func discounted(amount int64, d *payments.Discount) int64 {
	if d == nil {
		return amount
	}
	out := amount
	if d.PercentOff > 0 {
		out = int64(math.Round(float64(amount) * (100 - d.PercentOff) / 100))
	}
	if d.AmountOff > 0 {
		out = max(amount-d.AmountOff, 0)
	}
	return out
}

// discountJSON is a subscription's discount as paid_subscriptions keeps it.
func discountJSON(sub payments.Subscription) []byte {
	d := sub.Discount
	if d == nil {
		return nil
	}
	label := d.PromoCode
	if label == "" {
		label = d.Name
	}
	m := map[string]any{"label": label, "coupon": d.CouponID, "duration": d.Duration, "amountCents": discounted(sub.AmountCents, d)}
	if d.PercentOff > 0 {
		m["percent"] = d.PercentOff
	}
	if d.AmountOff > 0 {
		m["amountOff"] = d.AmountOff
	}
	if d.Months > 0 {
		m["months"] = d.Months
	}
	if d.End != nil {
		m["endsAt"] = d.End.UTC().Format(time.RFC3339)
	}
	b, _ := json.Marshal(m)
	return b
}

// outcome is what applying an event did, for the payments event log.
type outcome struct {
	status   string
	key      string
	detail   map[string]any
	contact  *uuid.UUID
	sub      *uuid.UUID
	amount   *int64
	currency string
	emails   []memberEmail
}

// memberEmail is a member email an event sends once its changes commit.
type memberEmail struct {
	email   string
	sub     uuid.UUID
	contact uuid.UUID
	key     string
	data    map[string]any
	// texts writes the values that are sentences, dates or prices in the
	// email's language: the version the send picks (data has them in
	// English).
	texts func(l i18n.Lang) map[string]any
}

// localized adds values written in the email's language.
func (m *memberEmail) localized(f func(l i18n.Lang) map[string]any) {
	prev := m.texts
	m.texts = func(l i18n.Lang) map[string]any {
		out := map[string]any{}
		if prev != nil {
			maps.Copy(out, prev(l))
		}
		maps.Copy(out, f(l))
		return out
	}
	if m.data == nil {
		m.data = map[string]any{}
	}
	maps.Copy(m.data, f(defaultLang))
}

func applied(key string, detail map[string]any) outcome {
	if detail == nil {
		detail = map[string]any{}
	}
	return outcome{status: "applied", key: key, detail: detail}
}

func ignored(key string) outcome {
	return outcome{status: "ignored", key: key, detail: map[string]any{}}
}

// Apply applies one stored event: the paid member's status, timeline and
// member emails, then the event log's line. An event applied before is
// skipped, so a retried job changes nothing twice.
func (s *Service) Apply(ctx context.Context, id uuid.UUID) error {
	pe, err := s.Q.PayEventGet(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if pe.Status != "pending" || !pe.SignatureOk {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, pe.WorkspaceID)
	if err != nil {
		return err
	}
	var conn store.PaymentConnection
	if pe.ConnectionID != nil {
		row, err := s.Q.PayConnection(ctx, store.PayConnectionParams{ID: *pe.ConnectionID, WorkspaceID: pe.WorkspaceID})
		if err != nil {
			return err
		}
		conn = row.PaymentConnection
	}
	v, ok := payments.ParserFor(firstNonEmpty(conn.Provider, "stripe"))
	if !ok {
		return nil
	}
	ev, err := v.ParseEvent(pe.Payload)
	if err != nil {
		return s.Q.PayEventResult(ctx, store.PayEventResultParams{ID: pe.ID, Status: "failed", Outcome: "unreadable", OutcomeDetail: []byte(`{}`), At: ptr(s.now()), Error: err.Error()})
	}
	var out outcome
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		out, err = s.applyEvent(ctx, tx, &ws, conn, pe, ev)
		if err != nil {
			return err
		}
		detail, _ := json.Marshal(out.detail)
		return store.New(tx).PayEventResult(ctx, store.PayEventResultParams{ID: pe.ID, Status: out.status, Outcome: out.key, OutcomeDetail: detail,
			ContactID: out.contact, SubscriptionID: out.sub, AmountCents: out.amount, Currency: out.currency, At: ptr(s.now()), Error: ""})
	})
	if err != nil {
		_ = s.Q.PayEventFailed(ctx, store.PayEventFailedParams{ID: pe.ID, Error: clip(err.Error(), 500)})
		return err
	}
	if len(out.emails) > 0 {
		statuses := map[string]string{}
		for _, m := range out.emails {
			statuses[m.email] = s.sendMemberEmail(ctx, &ws, pe.Mode, m, &pe.ID)
		}
		out.detail["emailStatus"] = statuses
		detail, _ := json.Marshal(out.detail)
		_, _ = s.Pool.Exec(ctx, `UPDATE payment_events SET outcome_detail = $2 WHERE id = $1`, pe.ID, detail)
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ApplyPending applies the events stored but not applied yet (tests, and
// the sweep after a job that never ran).
func (s *Service) ApplyPending(ctx context.Context, before time.Time) (int, error) {
	ids, err := s.Q.PayEventsPending(ctx, store.PayEventsPendingParams{Before: before, Lim: 500})
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := s.Apply(ctx, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func (s *Service) applyEvent(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	if conn.Status == "disconnected" {
		return ignored("disconnected"), nil
	}
	// One-time payments of other domains (digital products) go to them.
	if out, ok, err := s.consume(ctx, tx, ws, conn, pe); ok || err != nil {
		return out, err
	}
	switch ev.Kind {
	case payments.EventSubscriptionCreated, payments.EventSubscriptionUpdated, payments.EventSubscriptionDeleted:
		return s.applySubscription(ctx, tx, ws, conn, pe, ev)
	case payments.EventTrialEnding:
		return s.applyTrialEnding(ctx, tx, ws, conn, pe, ev)
	case payments.EventCheckoutCompleted:
		return s.applyCheckout(ctx, tx, ws, conn, pe, ev)
	case payments.EventInvoicePaid:
		return s.applyInvoicePaid(ctx, tx, ws, conn, pe, ev)
	case payments.EventPaymentFailed:
		return s.applyPaymentFailed(ctx, tx, ws, conn, pe, ev)
	case payments.EventInvoiceUpcoming:
		return s.applyUpcoming(ctx, tx, ws, conn, pe, ev)
	case payments.EventRefunded:
		return s.applyRefund(ctx, tx, ws, conn, pe, ev)
	case payments.EventDisputeOpened:
		return s.applyDispute(ctx, tx, ws, conn, pe, ev)
	case payments.EventCustomerUpdated:
		return s.applyCustomer(ctx, tx, ws, conn, pe, ev)
	case payments.EventCustomerDeleted:
		return applied("customer_deleted", nil), nil
	}
	return ignored("not_used"), nil
}

// timeline adds an entry to a paid member's timeline.
func (s *Service) timeline(ctx context.Context, q *store.Queries, wsID, contact uuid.UUID, sub *uuid.UUID, mode, kind string, at time.Time, amount *int64, currency string,
	pe *uuid.UUID, stripeEvent string, actor *uuid.UUID, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	b, _ := json.Marshal(data)
	if err := q.EarnEventInsert(ctx, store.EarnEventInsertParams{ID: newID(), WorkspaceID: wsID, ContactID: contact, SubscriptionID: sub, Mode: mode, Kind: kind,
		OccurredAt: at, AmountCents: amount, Currency: currency, PaymentEventID: pe, StripeEvent: stripeEvent, ActorID: actor, Data: b}); err != nil {
		return err
	}
	return s.emitHook(ctx, q, wsID, contact, sub, mode, kind, at, amount, currency, data)
}

// resolvePrice finds the tier and price a provider price stands for. A
// price no tier knows is recorded as unmapped (the sync error) and its
// subscriptions show no tier until it's mapped.
func (s *Service) resolvePrice(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode string, sub payments.Subscription, foundBy string) (tier, price *uuid.UUID, err error) {
	q := store.New(tx)
	if sub.PriceID == "" {
		return nil, nil, nil
	}
	p, err := q.EarnPriceByStripeID(ctx, store.EarnPriceByStripeIDParams{WorkspaceID: ws.ID, Mode: mode, StripeID: sub.PriceID})
	if err == nil {
		t, pid := p.TierID, p.ID
		return &t, &pid, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	r, err := q.EarnUnmappedPriceUpsert(ctx, store.EarnUnmappedPriceUpsertParams{ID: newID(), WorkspaceID: ws.ID, Mode: mode, StripePriceID: sub.PriceID,
		Period: periodOf(sub.Interval), AmountCents: sub.AmountCents, Currency: sub.Currency, FoundAt: s.now(), FoundBy: foundBy})
	if err != nil {
		return nil, nil, err
	}
	if r.Inserted && mode == "live" {
		args := i18n.Args{"amount": money(sub.AmountCents, sub.Currency, true), "period": periodKey(periodOf(sub.Interval)), "id": shortID(sub.PriceID)}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: KindUnmappedPrice, Severity: "warn",
			TitleText: msgs.M("notify.unmappedTitle").Keep(nil), BodyText: msgs.M("notify.unmappedBody").Keep(args),
			Link: "/w/" + ws.Slug + "/earn/settings?tab=payments", Data: map[string]any{"price": sub.PriceID}}
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.PaymentsManage, n); err != nil {
			return nil, nil, err
		}
		_ = q.EarnUnmappedPriceNotified(ctx, store.EarnUnmappedPriceNotifiedParams{ID: r.ID, At: ptr(s.now())})
	}
	return nil, nil, nil
}

func periodOf(interval string) string {
	switch interval {
	case "month", "year":
		return interval
	}
	return ""
}

// shortID is a provider id cut for sentences: price_1Q8…k2.
func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:9] + "…" + id[len(id)-2:]
}

// contactFor finds the contact of a provider customer: a paid
// subscription that already names the customer, else the customer's email
// (case-insensitive), else a new contact. New paid members join the tier's
// lists with consent recorded as "Paid subscription".
func (s *Service) contactFor(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode, customer, email, name string, lists []uuid.UUID, at time.Time) (uuid.UUID, bool, error) {
	q := store.New(tx)
	if customer != "" {
		rows, err := q.EarnSubscriptionsOfCustomer(ctx, store.EarnSubscriptionsOfCustomerParams{WorkspaceID: ws.ID, Mode: mode, StripeCustomerID: customer})
		if err != nil {
			return uuid.Nil, false, err
		}
		if len(rows) > 0 {
			return rows[0].ContactID, false, nil
		}
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return uuid.Nil, false, nil
	}
	first, last := splitName(name)
	if c, err := q.EarnContactByEmail(ctx, store.EarnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		if len(lists) > 0 && s.Signup != nil && mode == "live" {
			if _, _, err := s.Signup.Join(ctx, tx, ws, Join{Email: email, FirstName: first, LastName: last, Lists: lists, Source: SourcePaid, Note: "Paid subscription", At: at}); err != nil {
				return uuid.Nil, false, err
			}
		}
		return c.ID, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	if len(lists) > 0 && s.Signup != nil && mode == "live" {
		id, created, err := s.Signup.Join(ctx, tx, ws, Join{Email: email, FirstName: first, LastName: last, Lists: lists, Source: SourcePaid, Note: "Paid subscription", At: at})
		if err != nil {
			return uuid.Nil, false, err
		}
		if id != uuid.Nil {
			return id, created, nil
		}
	}
	id, err := q.EarnContactInsert(ctx, store.EarnContactInsertParams{ID: newID(), WorkspaceID: ws.ID, Email: email, FirstName: first, LastName: last,
		MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), Source: SourcePaid, RoleAddress: emailaddr.IsRole(email), At: at})
	return id, true, err
}

// SourcePaid is the consent source of a paid subscription.
const SourcePaid = "paid_subscription"

func splitName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	first, last, _ := strings.Cut(name, " ")
	return first, strings.TrimSpace(last)
}

// subscriptionFor returns the row of a provider subscription, creating it
// from the provider's state when no event made it yet (checkout and
// invoice events may come first).
func (s *Service) subscriptionFor(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, subID string, at time.Time) (store.PaidSubscription, bool, error) {
	q := store.New(tx)
	row, err := q.EarnSubscriptionByStripeID(ctx, store.EarnSubscriptionByStripeIDParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeSubscriptionID: subID})
	if err == nil {
		return row, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return store.PaidSubscription{}, false, err
	}
	client, err := s.open(conn)
	if err != nil {
		return store.PaidSubscription{}, false, err
	}
	sub, err := client.GetSubscription(ctx, subID)
	if err != nil {
		return store.PaidSubscription{}, false, err
	}
	id, _, err := s.createSubscription(ctx, tx, ws, conn, client, sub, at, nil)
	if err != nil || id == uuid.Nil {
		return store.PaidSubscription{}, false, err
	}
	row, err = q.EarnSubscriptionForUpdate(ctx, id)
	return row, false, err
}

// createSubscription stores a provider subscription the first time it is
// seen, with its contact. attribution names the checkout it came from.
func (s *Service) createSubscription(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, client payments.Client,
	sub payments.Subscription, at time.Time, src *checkoutSource) (uuid.UUID, bool, error) {
	q := store.New(tx)
	tier, price, err := s.resolvePrice(ctx, tx, ws, conn.Mode, sub, "event")
	if err != nil {
		return uuid.Nil, false, err
	}
	var lists []uuid.UUID
	if tier != nil {
		if t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *tier, WorkspaceID: ws.ID}); err == nil {
			lists = t.ListIds
		}
	}
	email, name := "", ""
	if src != nil {
		email = src.email
	}
	if email == "" && sub.CustomerID != "" && client != nil {
		if cu, err := client.GetCustomer(ctx, sub.CustomerID); err == nil {
			email, name = cu.Email, cu.Name
		} else if !errors.Is(err, payments.ErrNotFound) {
			return uuid.Nil, false, err
		}
	}
	contact, _, err := s.contactFor(ctx, tx, ws, conn.Mode, sub.CustomerID, email, name, lists, at)
	if err != nil || contact == uuid.Nil {
		return uuid.Nil, false, err
	}
	status := paidStatus(sub, at)
	var ended *time.Time
	reason := ""
	if status == StatusCanceled {
		ended = firstTime(sub.EndedAt, &at)
		reason = "canceled"
	}
	id := newID()
	srcKind, srcLabel, srcCampaign, srcDetail := "direct", "", (*uuid.UUID)(nil), []byte(`{}`)
	if src != nil {
		srcKind, srcLabel, srcCampaign = src.kind, src.label, src.campaign
		srcDetail, _ = json.Marshal(src.detail)
	}
	billing := email
	err = q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: id, WorkspaceID: ws.ID, ContactID: contact, Mode: conn.Mode, Kind: "stripe",
		TierID: tier, PriceID: price, StripePriceID: sub.PriceID, Status: status, Period: periodOf(sub.Interval), AmountCents: sub.AmountCents, Currency: sub.Currency,
		Discount: discountJSON(sub), StartedAt: nonZero(sub.StartedAt, at), TrialEnd: sub.TrialEnd, CurrentPeriodEnd: sub.CurrentPeriodEnd, CancelAt: sub.CancelAt,
		CanceledAt: sub.CanceledAt, EndedAt: ended, EndReason: reason, StripeCustomerID: sub.CustomerID, StripeSubscriptionID: sub.ID, BillingEmail: billing,
		SourceKind: srcKind, SourceCampaignID: srcCampaign, SourceLabel: srcLabel, SourceDetail: srcDetail, StripeSyncedAt: &at, At: at})
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func nonZero(t, def time.Time) time.Time {
	if t.IsZero() {
		return def
	}
	return t
}

func firstTime(ts ...*time.Time) *time.Time {
	for _, t := range ts {
		if t != nil && !t.IsZero() {
			return t
		}
	}
	return nil
}

// checkoutSource is what a checkout is credited to.
type checkoutSource struct {
	kind     string
	label    string
	campaign *uuid.UUID
	email    string
	detail   map[string]any
}

// tierName names a tier for timelines and the event log.
func tierName(ctx context.Context, q *store.Queries, wsID uuid.UUID, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *id, WorkspaceID: wsID})
	if err != nil {
		return ""
	}
	return t.Name
}

func (s *Service) applySubscription(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	sub := *ev.Subscription
	at := ev.Created
	row, err := q.EarnSubscriptionByStripeID(ctx, store.EarnSubscriptionByStripeIDParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeSubscriptionID: sub.ID})
	created := false
	if errors.Is(err, pgx.ErrNoRows) {
		client, cerr := s.open(conn)
		if cerr != nil {
			client = nil
		}
		id, ok, err := s.createSubscription(ctx, tx, ws, conn, client, sub, at, nil)
		if err != nil {
			return outcome{}, err
		}
		if !ok {
			return ignored("no_customer_email"), nil
		}
		row, err = q.EarnSubscriptionForUpdate(ctx, id)
		if err != nil {
			return outcome{}, err
		}
		created = true
	} else if err != nil {
		return outcome{}, err
	}
	// An event older than the state already applied changes nothing: it
	// arrived out of order.
	if !created && row.StripeSyncedAt != nil && at.Before(*row.StripeSyncedAt) {
		o := ignored("stale")
		o.contact, o.sub = &row.ContactID, &row.ID
		return o, nil
	}
	tname := ""
	tier, price, err := s.resolvePrice(ctx, tx, ws, conn.Mode, sub, "event")
	if err != nil {
		return outcome{}, err
	}
	tname = tierName(ctx, q, ws.ID, tier)
	status := paidStatus(sub, s.now())
	prevStatus := row.Status
	ended := row.EndedAt
	reason := row.EndReason
	if status == StatusCanceled && ended == nil {
		ended = firstTime(sub.EndedAt, &at)
		switch {
		case prevStatus == StatusTrialing && row.PaidCents == 0:
			reason = "trial_ended"
		case sub.Status == "unpaid" || prevStatus == StatusPastDue:
			reason = "payment_failed"
		default:
			reason = "canceled"
		}
	}
	if status != StatusCanceled {
		ended, reason = nil, ""
	}
	if err := q.EarnSubscriptionSetState(ctx, store.EarnSubscriptionSetStateParams{ID: row.ID, TierID: tier, PriceID: price, StripePriceID: sub.PriceID, Status: status,
		Period: periodOf(sub.Interval), AmountCents: sub.AmountCents, Currency: sub.Currency, Discount: discountJSON(sub), TrialEnd: sub.TrialEnd,
		CurrentPeriodEnd: sub.CurrentPeriodEnd, CancelAt: sub.CancelAt, CanceledAt: sub.CanceledAt, EndedAt: ended, EndReason: reason,
		StripeCustomerID: sub.CustomerID, SyncedAt: &at, At: s.now()}); err != nil {
		return outcome{}, err
	}
	if d := sub.Discount; d != nil && d.PromoCode != "" {
		if r, ok := s.Offers.(OfferCodeRedeemer); ok {
			if err := r.CodeRedeemed(ctx, tx, ws, d.PromoCode, at); err != nil {
				return outcome{}, err
			}
		}
	}
	if status != StatusPastDue && row.AccessPaused {
		_ = q.EarnSubscriptionSetAccessPaused(ctx, store.EarnSubscriptionSetAccessPausedParams{ID: row.ID, AccessPaused: false, At: s.now()})
	}
	pid := pe.ID
	data := map[string]any{"tier": tname, "period": periodOf(sub.Interval), "amount": sub.AmountCents, "currency": sub.Currency}
	if d := sub.Discount; d != nil {
		data["offer"] = firstNonEmpty(d.PromoCode, d.Name)
		data["discounted"] = discounted(sub.AmountCents, d)
	}
	o := applied("updated", data)
	o.contact, o.sub = &row.ContactID, &row.ID
	tl := func(kind string, extra map[string]any) error {
		d := map[string]any{}
		for k, v := range data {
			d[k] = v
		}
		for k, v := range extra {
			d[k] = v
		}
		return s.timeline(ctx, q, ws.ID, row.ContactID, &row.ID, conn.Mode, kind, at, nil, "", &pid, ev.Type, nil, d)
	}
	cancelScheduled := sub.CancelAtPeriodEnd || (sub.CancelAt != nil && status != StatusCanceled)
	wasScheduled := row.CancelAt != nil && row.EndedAt == nil && !created
	switch {
	case ev.Kind == payments.EventSubscriptionDeleted || (status == StatusCanceled && prevStatus != StatusCanceled):
		o.key = "access_ended"
		if row.CanceledBy != nil {
			if n, err := q.EarnUserName(ctx, *row.CanceledBy); err == nil {
				o.detail["by"] = n
			}
		}
		o.detail["reason"] = reason
		if err := tl("access_ended", map[string]any{"reason": reason}); err != nil {
			return outcome{}, err
		}
		if !hasEmailed(row, "canceled") && reason == "canceled" {
			o.emails = append(o.emails, s.memberEmailFor("canceled", row, ws, tname, at))
		}
	case created && (status == StatusTrialing || status == StatusActive):
		kind := "subscribed"
		if status == StatusTrialing {
			kind = "trial_started"
			if sub.TrialEnd != nil {
				o.detail["trialEnd"] = sub.TrialEnd.UTC().Format(time.RFC3339)
			}
		}
		o.key = kind
		if err := tl(kind, map[string]any{"trialEnd": o.detail["trialEnd"]}); err != nil {
			return outcome{}, err
		}
		if !hasEmailed(row, "welcome") {
			o.emails = append(o.emails, s.memberEmailFor("welcome", row, ws, tname, at))
		}
	case cancelScheduled && !wasScheduled:
		o.key = "cancel_scheduled"
		if sub.CancelAt != nil {
			o.detail["date"] = sub.CancelAt.UTC().Format(time.RFC3339)
		}
		if row.CancelReason != "" {
			o.detail["reason"] = row.CancelReason
		}
		if err := tl("cancel_scheduled", map[string]any{"date": o.detail["date"], "reason": row.CancelReason}); err != nil {
			return outcome{}, err
		}
		if !hasEmailed(row, "canceled") {
			o.emails = append(o.emails, s.memberEmailFor("canceled", row, ws, tname, at))
		}
	case !cancelScheduled && wasScheduled:
		o.key = "cancel_undone"
		if err := tl("cancel_undone", nil); err != nil {
			return outcome{}, err
		}
	case row.PriceID != nil && price != nil && *row.PriceID != *price:
		o.key = "plan_changed"
		o.detail["from"] = tierName(ctx, q, ws.ID, row.TierID)
		if err := tl("plan_changed", map[string]any{"fromTier": o.detail["from"], "fromAmount": row.AmountCents, "fromPeriod": row.Period}); err != nil {
			return outcome{}, err
		}
	case sub.Discount != nil && len(row.Discount) == 0:
		o.key = "offer_applied"
		if err := tl("offer_applied", nil); err != nil {
			return outcome{}, err
		}
	case prevStatus == StatusIncomplete && (status == StatusTrialing || status == StatusActive):
		o.key = "subscribed"
		if status == StatusTrialing {
			o.key = "trial_started"
		}
		if err := tl(o.key, nil); err != nil {
			return outcome{}, err
		}
		if !hasEmailed(row, "welcome") {
			o.emails = append(o.emails, s.memberEmailFor("welcome", row, ws, tname, at))
		}
	case status != prevStatus:
		o.key = "status_" + status
	}
	return o, nil
}

func hasEmailed(row store.PaidSubscription, email string) bool {
	for _, e := range row.Emailed {
		if e == email {
			return true
		}
	}
	return false
}

func (s *Service) applyTrialEnding(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	row, _, err := s.subscriptionFor(ctx, tx, ws, conn, ev.Subscription.ID, ev.Created)
	if err != nil || row.ID == uuid.Nil {
		return ignored("unknown_subscription"), err
	}
	tname := tierName(ctx, q, ws.ID, row.TierID)
	o := applied("sent_trial_ending", map[string]any{"tier": tname})
	if t := firstTime(ev.Subscription.TrialEnd, row.TrialEnd); t != nil {
		o.detail["date"] = t.UTC().Format(time.RFC3339)
	}
	o.contact, o.sub = &row.ContactID, &row.ID
	m := s.memberEmailFor("trial_ending", row, ws, tname, ev.Created)
	m.key = "earn:" + pe.ProviderEventID + ":trial_ending"
	o.emails = append(o.emails, m)
	return o, nil
}

func (s *Service) applyCheckout(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	cs := ev.Checkout
	chk, err := q.EarnCheckoutBySession(ctx, cs.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return outcome{}, err
	}
	found := err == nil
	if !found && cs.ClientRef != "" {
		if id, perr := parseUUID(cs.ClientRef); perr == nil {
			if c, err := q.EarnCheckout(ctx, id); err == nil && c.WorkspaceID == ws.ID {
				chk, found = c, true
			}
		}
	}
	if cs.SubscriptionID == "" {
		// A one-time payment: a gift (X17).
		if s.OneTime != nil {
			if handled, detail, err := s.OneTime.CheckoutCompleted(ctx, tx, ws, conn.Mode, *cs, ev.Created); err != nil {
				return outcome{}, err
			} else if handled {
				o := applied("gift_paid", detail)
				amount := cs.AmountTotal
				o.amount, o.currency = &amount, cs.Currency
				return o, nil
			}
		}
		return ignored("not_a_subscription"), nil
	}
	src := &checkoutSource{kind: "page", email: cs.CustomerEmail, detail: map[string]any{"session": cs.ID}}
	if found {
		src = sourceOfCheckout(ctx, q, chk)
		if src.email == "" {
			src.email = cs.CustomerEmail
		}
	}
	row, err := q.EarnSubscriptionByStripeID(ctx, store.EarnSubscriptionByStripeIDParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeSubscriptionID: cs.SubscriptionID})
	if errors.Is(err, pgx.ErrNoRows) {
		client, err := s.open(conn)
		if err != nil {
			return outcome{}, err
		}
		sub, err := client.GetSubscription(ctx, cs.SubscriptionID)
		if err != nil {
			return outcome{}, err
		}
		id, ok, err := s.createSubscription(ctx, tx, ws, conn, client, sub, ev.Created, src)
		if err != nil {
			return outcome{}, err
		}
		if !ok {
			return ignored("no_customer_email"), nil
		}
		if row, err = q.EarnSubscriptionForUpdate(ctx, id); err != nil {
			return outcome{}, err
		}
	} else if err != nil {
		return outcome{}, err
	} else {
		// The subscription came first: credit it now.
		detail, _ := json.Marshal(src.detail)
		if err := q.EarnSubscriptionSetSource(ctx, store.EarnSubscriptionSetSourceParams{ID: row.ID, SourceKind: src.kind, SourceCampaignID: src.campaign,
			SourceLabel: src.label, SourceDetail: detail, At: s.now()}); err != nil {
			return outcome{}, err
		}
		// …and it joins the tier's lists like any paid member.
		if row.TierID != nil && conn.Mode == "live" {
			if t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *row.TierID, WorkspaceID: ws.ID}); err == nil && len(t.ListIds) > 0 {
				if _, _, err := s.contactFor(ctx, tx, ws, conn.Mode, "", firstNonEmpty(src.email, cs.CustomerEmail), "", t.ListIds, ev.Created); err != nil {
					return outcome{}, err
				}
			}
		}
	}
	if found {
		sid, cid := row.ID, row.ContactID
		if err := q.EarnCheckoutComplete(ctx, store.EarnCheckoutCompleteParams{ID: chk.ID, At: ev.Created, SubscriptionID: &sid, ContactID: &cid, Email: cs.CustomerEmail}); err != nil {
			return outcome{}, err
		}
	}
	tname := tierName(ctx, q, ws.ID, row.TierID)
	pid := pe.ID
	data := map[string]any{"tier": tname, "period": row.Period}
	if len(row.Discount) > 0 {
		var d map[string]any
		_ = json.Unmarshal(row.Discount, &d)
		data["offer"] = d["label"]
	}
	if err := s.timeline(ctx, q, ws.ID, row.ContactID, &row.ID, conn.Mode, "checkout_completed", ev.Created, nil, "", &pid, ev.Type, nil, data); err != nil {
		return outcome{}, err
	}
	o := applied("checkout_completed", data)
	o.contact, o.sub = &row.ContactID, &row.ID
	return o, nil
}

// sourceOfCheckout is what a checkout is credited to: the email clicked
// within the attribution window, else how the reader reached the page.
func sourceOfCheckout(ctx context.Context, q *store.Queries, chk store.EarnCheckout) *checkoutSource {
	var a struct {
		Kind      string `json:"kind"`
		Campaign  string `json:"campaign_id"`
		Message   string `json:"message_id"`
		ClickedAt string `json:"clicked_at"`
		Label     string `json:"label"`
	}
	_ = json.Unmarshal(chk.Attribution, &a)
	src := &checkoutSource{kind: firstNonEmpty(a.Kind, "page"), email: chk.Email, label: a.Label,
		detail: map[string]any{"checkout": chk.ID.String(), "clicked_at": a.ClickedAt, "message_id": a.Message}}
	if a.Campaign != "" {
		if id, err := uuid.Parse(a.Campaign); err == nil {
			src.campaign = &id
			if src.label == "" {
				if name, err := q.EarnCampaignName(ctx, id); err == nil {
					src.label = name
				}
			}
		}
	}
	if chk.Offer != "" {
		src.detail["offer"] = chk.Offer
	}
	return src
}

func (s *Service) applyInvoicePaid(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	inv := ev.Invoice
	if inv.SubscriptionID == "" {
		return ignored("not_a_subscription"), nil
	}
	row, _, err := s.subscriptionFor(ctx, tx, ws, conn, inv.SubscriptionID, ev.Created)
	if err != nil || row.ID == uuid.Nil {
		return ignored("unknown_subscription"), err
	}
	tname := tierName(ctx, q, ws.ID, row.TierID)
	o := applied("paid", map[string]any{"tier": tname, "period": row.Period})
	o.contact, o.sub, o.currency = &row.ContactID, &row.ID, inv.Currency
	if inv.AmountPaid <= 0 {
		// A trial's $0 invoice: nothing was paid.
		o.status, o.key = "ignored", "nothing_paid"
		return o, nil
	}
	amount := inv.AmountPaid
	o.amount = &amount
	if err := q.EarnSubscriptionPaid(ctx, store.EarnSubscriptionPaidParams{ID: row.ID, Amount: amount, At: s.now()}); err != nil {
		return outcome{}, err
	}
	if row.AccessPaused {
		_ = q.EarnSubscriptionSetAccessPaused(ctx, store.EarnSubscriptionSetAccessPausedParams{ID: row.ID, AccessPaused: false, At: s.now()})
	}
	kind := "paid"
	if inv.BillingReason == "subscription_cycle" {
		kind, o.key = "renewed", "renewed"
	}
	if inv.PeriodEnd != nil {
		o.detail["until"] = inv.PeriodEnd.UTC().Format(time.RFC3339)
	}
	if len(row.Discount) > 0 {
		var d map[string]any
		_ = json.Unmarshal(row.Discount, &d)
		o.detail["offer"] = d["label"]
	}
	pid := pe.ID
	if err := s.timeline(ctx, q, ws.ID, row.ContactID, &row.ID, conn.Mode, kind, ev.Created, &amount, inv.Currency, &pid, ev.Type, nil,
		map[string]any{"tier": tname, "period": row.Period, "offer": o.detail["offer"], "until": o.detail["until"], "reason": inv.BillingReason}); err != nil {
		return outcome{}, err
	}
	return o, nil
}

func (s *Service) applyPaymentFailed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	inv := ev.Invoice
	if inv.SubscriptionID == "" {
		return ignored("not_a_subscription"), nil
	}
	row, _, err := s.subscriptionFor(ctx, tx, ws, conn, inv.SubscriptionID, ev.Created)
	if err != nil || row.ID == uuid.Nil {
		return ignored("unknown_subscription"), err
	}
	reason := ""
	if inv.ChargeID != "" {
		if client, err := s.open(conn); err == nil {
			if ch, err := client.GetCharge(ctx, inv.ChargeID); err == nil {
				reason = ch.FailureMessage
			}
		}
	}
	amount := inv.AmountDue
	if err := q.EarnSubscriptionFailed(ctx, store.EarnSubscriptionFailedParams{ID: row.ID, FailedAt: &ev.Created, FailureReason: reason, NextRetryAt: inv.NextAttempt, At: s.now()}); err != nil {
		return outcome{}, err
	}
	st, err := s.settingsRow(ctx, q, ws.ID)
	if err != nil {
		return outcome{}, err
	}
	if row.Status != StatusCanceled {
		if err := q.EarnSubscriptionSetState(ctx, store.EarnSubscriptionSetStateParams{ID: row.ID, TierID: row.TierID, PriceID: row.PriceID, StripePriceID: row.StripePriceID,
			Status: StatusPastDue, Period: row.Period, AmountCents: row.AmountCents, Currency: row.Currency, Discount: row.Discount, TrialEnd: row.TrialEnd,
			CurrentPeriodEnd: row.CurrentPeriodEnd, CancelAt: row.CancelAt, CanceledAt: row.CanceledAt, EndedAt: row.EndedAt, EndReason: row.EndReason,
			StripeCustomerID: row.StripeCustomerID, SyncedAt: maxTime(row.StripeSyncedAt, ev.Created), At: s.now()}); err != nil {
			return outcome{}, err
		}
		if st.OnPaymentFailure == "end" {
			_ = q.EarnSubscriptionSetAccessPaused(ctx, store.EarnSubscriptionSetAccessPausedParams{ID: row.ID, AccessPaused: true, At: s.now()})
		}
	}
	tname := tierName(ctx, q, ws.ID, row.TierID)
	o := applied("past_due", map[string]any{"tier": tname, "reason": reason, "rule": st.OnPaymentFailure})
	if inv.NextAttempt != nil {
		o.detail["retry"] = inv.NextAttempt.UTC().Format(time.RFC3339)
	}
	o.contact, o.sub, o.amount, o.currency = &row.ContactID, &row.ID, &amount, inv.Currency
	pid := pe.ID
	if err := s.timeline(ctx, q, ws.ID, row.ContactID, &row.ID, conn.Mode, "payment_failed", ev.Created, &amount, inv.Currency, &pid, ev.Type, nil,
		map[string]any{"reason": reason, "retry": o.detail["retry"], "tier": tname}); err != nil {
		return outcome{}, err
	}
	m := s.memberEmailFor("payment_failed", row, ws, tname, ev.Created)
	m.key = "earn:" + inv.ID + ":payment_failed"
	if inv.ID == "" {
		m.key = "earn:" + pe.ProviderEventID + ":payment_failed"
	}
	m.data["amount"] = money(amount, inv.Currency, false)
	next := inv.NextAttempt
	m.localized(func(l i18n.Lang) map[string]any {
		out := map[string]any{"reason": failureSentence(l, reason)}
		if next != nil {
			out["next_try"] = s.dateTextIn(l, ws, *next)
		}
		return out
	})
	o.emails = append(o.emails, m)
	return o, nil
}

func maxTime(a *time.Time, b time.Time) *time.Time {
	if a != nil && a.After(b) {
		return a
	}
	return &b
}

func (s *Service) applyUpcoming(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	inv := ev.Invoice
	if inv.SubscriptionID == "" {
		return ignored("not_a_subscription"), nil
	}
	row, _, err := s.subscriptionFor(ctx, tx, ws, conn, inv.SubscriptionID, ev.Created)
	if err != nil || row.ID == uuid.Nil {
		return ignored("unknown_subscription"), err
	}
	amount := inv.AmountDue
	o := applied("upcoming_monthly", map[string]any{})
	o.contact, o.sub, o.amount, o.currency = &row.ContactID, &row.ID, &amount, inv.Currency
	// Renewal reminders are for yearly prices only (7 days before).
	if row.Period != "year" || row.CancelAt != nil {
		o.status = "ignored"
		return o, nil
	}
	tname := tierName(ctx, q, ws.ID, row.TierID)
	o.key = "sent_renewal_reminder"
	o.detail["tier"] = tname
	renews := firstTime(row.CurrentPeriodEnd, inv.PeriodEnd)
	if renews != nil {
		o.detail["date"] = renews.UTC().Format(time.RFC3339)
	}
	m := s.memberEmailFor("renewal_reminder", row, ws, tname, ev.Created)
	m.key = "earn:" + row.ID.String() + ":renewal:" + fmt.Sprint(renews)
	m.data["amount"] = money(amount, inv.Currency, false)
	o.emails = append(o.emails, m)
	return o, nil
}

func (s *Service) applyRefund(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	ch := ev.Charge
	if ch == nil {
		return ignored("unknown_customer"), nil
	}
	// A one-time payment's refund: a gift (X17).
	if s.OneTime != nil && ch.InvoiceID == "" && ch.PaymentIntentID != "" {
		if handled, detail, err := s.OneTime.Refunded(ctx, tx, ws, conn.Mode, *ch, ev.Created); err != nil {
			return outcome{}, err
		} else if handled {
			o := applied("gift_refunded", detail)
			neg := -ch.RefundAmount
			if neg == 0 {
				neg = -ch.AmountRefunded
			}
			o.amount, o.currency = &neg, ch.Currency
			return o, nil
		}
	}
	if ch.CustomerID == "" {
		// A one-time payment without a customer (a guest checkout) belongs
		// to no subscription.
		return ignored("unknown_customer"), nil
	}
	rows, err := q.EarnSubscriptionsOfCustomer(ctx, store.EarnSubscriptionsOfCustomerParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeCustomerID: ch.CustomerID})
	if err != nil {
		return outcome{}, err
	}
	if len(rows) == 0 {
		return ignored("unknown_customer"), nil
	}
	row := rows[0]
	for _, r := range rows[1:] {
		if r.StartedAt.After(row.StartedAt) {
			row = r
		}
	}
	amount := ch.RefundAmount
	if amount <= 0 {
		amount = ch.AmountRefunded
	}
	if err := q.EarnSubscriptionRefunded(ctx, store.EarnSubscriptionRefundedParams{ID: row.ID, Amount: amount, At: s.now()}); err != nil {
		return outcome{}, err
	}
	neg := -amount
	pid := pe.ID
	if err := s.timeline(ctx, q, ws.ID, row.ContactID, &row.ID, conn.Mode, "refunded", ev.Created, &neg, ch.Currency, &pid, ev.Type, nil, map[string]any{}); err != nil {
		return outcome{}, err
	}
	ltv := row.PaidCents - row.RefundedCents - amount
	o := applied("refund_recorded", map[string]any{"ltv": ltv})
	o.contact, o.sub, o.amount, o.currency = &row.ContactID, &row.ID, &neg, ch.Currency
	return o, nil
}

func (s *Service) applyDispute(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	o := applied("dispute_opened", map[string]any{"charge": ev.Charge.ID})
	amount := ev.Charge.Amount
	o.amount, o.currency = &amount, ev.Charge.Currency
	if ev.Charge.ID == "" {
		return o, nil
	}
	client, err := s.open(conn)
	if err != nil {
		return o, err
	}
	ch, err := client.GetCharge(ctx, ev.Charge.ID)
	if err != nil {
		// The dispute is recorded; the member stays as they are.
		s.log().WarnContext(ctx, "dispute: reading the charge", "charge", ev.Charge.ID, "err", err)
		return o, nil
	}
	if ch.CustomerID == "" {
		return o, nil
	}
	rows, err := q.EarnSubscriptionsOfCustomer(ctx, store.EarnSubscriptionsOfCustomerParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeCustomerID: ch.CustomerID})
	if err != nil || len(rows) == 0 {
		return o, err
	}
	pid := pe.ID
	if err := s.timeline(ctx, q, ws.ID, rows[0].ContactID, &rows[0].ID, conn.Mode, "dispute", ev.Created, &amount, ev.Charge.Currency, &pid, ev.Type, nil, nil); err != nil {
		return outcome{}, err
	}
	o.contact, o.sub = &rows[0].ContactID, &rows[0].ID
	return o, nil
}

// applyCustomer follows a billing email changed in the portal: the billing
// email shown changes, never the contact's own email.
func (s *Service) applyCustomer(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent, ev payments.Event) (outcome, error) {
	q := store.New(tx)
	cu := ev.Customer
	if _, changed := ev.Previous["email"]; !changed {
		return ignored("no_billing_change"), nil
	}
	rows, err := q.EarnSubscriptionsOfCustomer(ctx, store.EarnSubscriptionsOfCustomerParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeCustomerID: cu.ID})
	if err != nil {
		return outcome{}, err
	}
	if len(rows) == 0 {
		return ignored("unknown_customer"), nil
	}
	if err := q.EarnSubscriptionSetBillingEmail(ctx, store.EarnSubscriptionSetBillingEmailParams{WorkspaceID: ws.ID, Mode: conn.Mode, StripeCustomerID: cu.ID,
		BillingEmail: cu.Email, At: s.now()}); err != nil {
		return outcome{}, err
	}
	pid := pe.ID
	if err := s.timeline(ctx, q, ws.ID, rows[0].ContactID, &rows[0].ID, conn.Mode, "billing_email_changed", ev.Created, nil, "", &pid, ev.Type, nil,
		map[string]any{"email": cu.Email}); err != nil {
		return outcome{}, err
	}
	o := applied("billing_email_changed", map[string]any{"email": cu.Email})
	o.contact, o.sub = &rows[0].ContactID, &rows[0].ID
	return o, nil
}
