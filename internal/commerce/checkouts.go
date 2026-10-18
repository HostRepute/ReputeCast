package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// StoreTrigger is a store trigger's settings, as the automation's trigger
// keeps them (internal/automations' AutomationTrigger fields).
type StoreTrigger struct {
	Kind  string `json:"kind"`
	Store string `json:"store"`
	// AfterHours: an abandoned checkout waits 1 to 24 hours for an order.
	AfterHours int `json:"afterHours"`
	// SkipDays: the skip rule, a contact who got an abandoned-cart email in
	// the last N days doesn't enter (0: no rule).
	SkipDays *int `json:"skipDays"`
	// Order placed's filters.
	Products    []string `json:"products"`
	Collections []string `json:"collections"`
	MinTotal    *int64   `json:"minTotal"`
	FirstOnly   bool     `json:"firstOnly"`
}

// DefaultSkipDays is the abandoned-cart starter's skip rule.
const DefaultSkipDays = 7

// Hours is the delay, 1 to 24 (1 by default).
func (t StoreTrigger) Hours() int {
	switch {
	case t.AfterHours < 1:
		return 1
	case t.AfterHours > 24:
		return 24
	}
	return t.AfterHours
}

// Skip is the skip rule's days.
func (t StoreTrigger) Skip() int {
	if t.SkipDays == nil {
		return DefaultSkipDays
	}
	if *t.SkipDays < 0 {
		return 0
	}
	return *t.SkipDays
}

// Entry outcomes of an abandoned checkout in an automation.
const (
	EntryEntered     = "entered"
	EntryNoEmail     = "no_email"
	EntryNoConsent   = "no_consent"
	EntrySkipRule    = "skip_rule"
	EntryNotEligible = "not_eligible"
)

// DetectAbandoned finds the checkouts each live abandoned-cart automation
// should hear about: with no order after its delay, created after it went
// live and after the store's import. Only contacts who accepted marketing
// in the store enter (the trigger's locked filter); the skip rule keeps
// out anyone who got an abandoned-cart email in the last 7 days.
func (s *Service) DetectAbandoned(ctx context.Context) (int, error) {
	flows, err := s.Q.CommerceAbandonedFlows(ctx)
	if err != nil {
		return 0, err
	}
	entered := 0
	now := s.now()
	for _, f := range flows {
		var t StoreTrigger
		if json.Unmarshal(f.Trigger, &t) != nil {
			continue
		}
		storeID, err := uuid.Parse(t.Store)
		if err != nil {
			continue
		}
		st, err := s.Q.CommerceStoreByID(ctx, storeID)
		if err != nil || st.WorkspaceID != f.WorkspaceID || st.Status != StatusConnected || !st.SyncCheckouts || st.ImportFinishedAt == nil {
			continue
		}
		since := *st.ImportFinishedAt
		if f.LiveVersion != nil {
			if at, err := s.Q.CommerceFlowLiveSince(ctx, store.CommerceFlowLiveSinceParams{AutomationID: f.ID, Version: *f.LiveVersion}); err == nil && at != nil && at.After(since) {
				since = *at
			}
		}
		before := now.Add(-time.Duration(t.Hours()) * time.Hour)
		cands, err := s.Q.CommerceCheckoutsAbandoned(ctx, store.CommerceCheckoutsAbandonedParams{StoreID: st.ID, Before: before, Since: since, AutomationID: f.ID})
		if err != nil {
			return entered, err
		}
		if len(cands) == 0 {
			continue
		}
		name := ""
		if row, err := s.Q.CommerceAutomationName(ctx, f.ID); err == nil {
			name = row.Name
		}
		n := 0
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, ch := range cands {
				outcome, err := s.entryOutcome(ctx, q, st, t, ch, now)
				if err != nil {
					return err
				}
				if err := q.CommerceCheckoutEntry(ctx, store.CommerceCheckoutEntryParams{AutomationID: f.ID, CheckoutID: ch.ID, Outcome: outcome, At: now}); err != nil {
					return err
				}
				if outcome != EntryEntered {
					continue
				}
				payload := map[string]any{"automation": f.ID.String(), "store": st.ID.String(), "checkout": ch.ID.String(), "total": ch.TotalCents,
					"currency": ch.Currency, "ctx": map[string]string{"store": st.ID.String(), "checkout": ch.ID.String()}}
				if err := q.CommerceEmitEvent(ctx, store.CommerceEmitEventParams{WorkspaceID: st.WorkspaceID, Kind: TriggerAbandonedCheckout, ContactID: *ch.ContactID,
					Ref: "store:" + st.ID.String(), Payload: mustJSON(payload), OccurredAt: now}); err != nil {
					return err
				}
				n++
				who := ""
				if c, err := q.CommerceCustomerByContact(ctx, ch.ContactID); err == nil {
					for _, sc := range c {
						if sc.StoreID == st.ID {
							who = strings.TrimSpace(sc.FirstName + " " + sc.LastName)
						}
					}
				}
				if err := s.activity(ctx, q, st, activity{Kind: "checkout_abandoned", Tone: "info", Customer: who, Amount: ch.TotalCents, Currency: ch.Currency,
					Accepted: true, Contact: true, ContactID: ch.ContactID, CheckoutID: &ch.ID, Flow: name, Hours: t.Hours()}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return entered, err
		}
		entered += n
	}
	if entered > 0 && s.Kick != nil {
		s.Kick(ctx)
	}
	return entered, nil
}

// entryOutcome decides whether an abandoned checkout's contact enters.
func (s *Service) entryOutcome(ctx context.Context, q *store.Queries, st store.Store, t StoreTrigger, ch store.StoreCheckout, now time.Time) (string, error) {
	if !ch.HasEmail {
		return EntryNoEmail, nil
	}
	if ch.ContactID == nil {
		return EntryNoConsent, nil
	}
	accepted := ch.AcceptedMarketing
	if !accepted {
		rows, err := q.CommerceCustomerByContact(ctx, ch.ContactID)
		if err != nil {
			return "", err
		}
		for _, r := range rows {
			if r.StoreID == st.ID && r.Marketing == stores.ConsentSubscribed {
				accepted = true
			}
		}
	}
	if !accepted {
		return EntryNoConsent, nil
	}
	c, err := q.CommerceContactByID(ctx, *ch.ContactID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (c.Status != "active" || c.ErasedAt != nil)) {
		return EntryNotEligible, nil
	}
	if err != nil {
		return "", err
	}
	if days := t.Skip(); days > 0 {
		sent, err := q.CommerceAbandonedCartEmailedSince(ctx, store.CommerceAbandonedCartEmailedSinceParams{ContactID: ch.ContactID,
			Since: now.AddDate(0, 0, -days), WorkspaceID: st.WorkspaceID})
		if err != nil {
			return "", err
		}
		if sent {
			return EntrySkipRule, nil
		}
	}
	return EntryEntered, nil
}

// TriggerStats are an abandoned-checkout trigger's numbers since its
// automation went live: abandoned checkouts, with an email, entered, not
// emailed because they didn't accept marketing, skipped by the skip rule,
// and the orders credited to the automation.
func (s *Service) TriggerStats(ctx context.Context, ws *store.Workspace, automationID uuid.UUID) (CommerceTriggerStats, error) {
	out := CommerceTriggerStats{}
	flows, err := s.Q.CommerceAbandonedFlows(ctx)
	if err != nil {
		return out, err
	}
	for _, f := range flows {
		if f.ID != automationID || f.WorkspaceID != ws.ID {
			continue
		}
		var t StoreTrigger
		_ = json.Unmarshal(f.Trigger, &t)
		storeID, err := uuid.Parse(t.Store)
		if err != nil {
			return out, nil //nolint:nilerr // a trigger without its store has no numbers
		}
		var since time.Time
		if f.LiveVersion != nil {
			if at, err := s.Q.CommerceFlowLiveSince(ctx, store.CommerceFlowLiveSinceParams{AutomationID: f.ID, Version: *f.LiveVersion}); err == nil && at != nil {
				since = *at
			}
		}
		if !since.IsZero() {
			out.Since = &since
		}
		stats, err := s.Q.CommerceCheckoutStats(ctx, store.CommerceCheckoutStatsParams{StoreID: storeID, Since: since,
			Before: s.now().Add(-time.Duration(t.Hours()) * time.Hour)})
		if err == nil {
			out.Abandoned, out.WithEmail = stats.Abandoned, stats.WithEmail
		}
	}
	rows, err := s.Q.CommerceCheckoutEntries(ctx, automationID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		switch r.Outcome {
		case EntryEntered:
			out.Entered = r.N
		case EntryNoConsent:
			out.NoConsent = r.N
		case EntrySkipRule:
			out.Skipped = r.N
		}
	}
	since := time.Time{}
	if out.Since != nil {
		since = *out.Since
	}
	if g, err := s.Q.CommerceAutomationGoalRevenue(ctx, store.CommerceAutomationGoalRevenueParams{AutomationID: &automationID, Since: since}); err == nil {
		out.Orders, out.Revenue = g.Orders, g.Revenue
	}
	return out, nil
}

// writeBack tells Shopify about unsubscribes here ("Tell Shopify when
// someone unsubscribes here"): contacts who left the store's list, or every
// list, since the last pass. Unsubscribes the store itself sent aren't
// echoed back.
func (s *Service) writeBack(ctx context.Context) error {
	sts, err := s.Q.CommerceStoresActive(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, st := range sts {
		if !st.WriteBack || st.Platform != Shopify || st.ListID == nil || st.Status != StatusConnected {
			continue
		}
		since := now.Add(-time.Hour)
		switch {
		case st.WritebackThrough != nil:
			since = *st.WritebackThrough
		case st.ConnectedAt != nil:
			since = *st.ConnectedAt
		}
		until := now.Add(-5 * time.Second)
		rows, err := s.Q.CommerceUnsubscribesSince(ctx, store.CommerceUnsubscribesSinceParams{StoreID: st.ID, Since: since, Until: until, ListID: st.ListID})
		if err != nil {
			return err
		}
		ok := true
		if len(rows) > 0 {
			c, err := s.client(st)
			if err != nil {
				continue
			}
			for _, r := range rows {
				if strings.HasPrefix(r.ExternalID, "guest:") {
					continue
				}
				if err := c.SetConsent(ctx, r.ExternalID, stores.ConsentUnsubscribed, r.OccurredAt); err != nil {
					if errors.Is(err, stores.ErrUnauthorized) {
						_ = s.revoke(ctx, st, "revoked", now)
					}
					ok = false
					break
				}
				if err := s.Q.CommerceCustomerToldStore(ctx, store.CommerceCustomerToldStoreParams{ID: r.ID, At: now}); err != nil {
					return err
				}
				name := ""
				if r.ContactID != nil {
					if ct, err := s.Q.CommerceContactByID(ctx, *r.ContactID); err == nil {
						name = strings.TrimSpace(ct.FirstName + " " + ct.LastName)
					}
				}
				if err := s.activity(ctx, s.Q, st, activity{Kind: "unsubscribe_passed", Tone: "neutral", Customer: name, ContactID: r.ContactID}); err != nil {
					return err
				}
			}
		}
		if ok {
			if err := s.Q.CommerceStoreWritebackThrough(ctx, store.CommerceStoreWritebackThroughParams{ID: st.ID, At: until}); err != nil {
				return err
			}
		}
	}
	return nil
}
