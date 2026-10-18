package earn

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Reasons for complimentary access.
var grantReasons = []string{"team", "contributor", "press", "partner", "giveaway", "other"}

// EarnComplimentaryInput gives a contact, or a new address, a tier.
type EarnComplimentaryInput struct {
	ContactID string `json:"contactId,omitempty" doc:"An existing contact"`
	Email     string `json:"email,omitempty" maxLength:"320" doc:"A new address: it joins the tier's lists through the usual double opt-in"`
	TierID    string `json:"tierId"`
	// Until is the day access ends; empty for no end date.
	Until       string `json:"until,omitempty" maxLength:"10" doc:"YYYY-MM-DD; empty for no end date"`
	Reason      string `json:"reason" enum:"team,contributor,press,partner,giveaway,other"`
	Note        string `json:"note,omitempty" maxLength:"500"`
	SendWelcome bool   `json:"sendWelcome"`
}

func (s *Service) parseUntil(ctx context.Context, ws *store.Workspace, v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	d, err := time.ParseInLocation("2006-01-02", v, location(ws))
	if err != nil {
		return nil, invalid(ctx, "body.until", "grant.until", nil)
	}
	end := d.Add(24*time.Hour - time.Second)
	if !end.After(s.now()) {
		return nil, invalid(ctx, "body.until", "grant.untilPast", nil)
	}
	return &end, nil
}

// GiveComplimentary gives complimentary access: it lives in ReputeCast
// only (nothing is created in the provider, nothing is charged, it never
// counts as revenue). A new address joins the tier's lists through the
// usual double opt-in.
func (s *Service) GiveComplimentary(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnComplimentaryInput) (EarnMember, error) {
	tid, err := uuid.Parse(in.TierID)
	if err != nil {
		return EarnMember{}, invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnMember{}, invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	if !slices.Contains(grantReasons, in.Reason) {
		return EarnMember{}, invalid(ctx, "body.reason", "grant.reason", nil)
	}
	until, err := s.parseUntil(ctx, ws, in.Until)
	if err != nil {
		return EarnMember{}, err
	}
	now := s.now()
	id := newID()
	var contact uuid.UUID
	newAddress := false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		switch {
		case in.ContactID != "":
			cid, err := uuid.Parse(in.ContactID)
			if err != nil {
				return invalid(ctx, "body.contactId", "grant.contact", nil)
			}
			c, err := q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: cid, WorkspaceID: ws.ID})
			if err != nil {
				return invalid(ctx, "body.contactId", "grant.contact", nil)
			}
			contact = c.ID
		default:
			email, problemText := emailaddr.Check(in.Email)
			if problemText != "" {
				return invalid(ctx, "body.email", "grant.email", nil)
			}
			if c, err := q.EarnContactByEmail(ctx, store.EarnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
				contact = c.ID
				break
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			newAddress = true
			if len(t.ListIds) > 0 && s.Signup != nil {
				cid, _, err := s.Signup.Join(ctx, tx, ws, Join{Email: email, Lists: t.ListIds, Double: true, Source: SourcePaid, Note: "Complimentary access", At: now})
				if err != nil {
					return err
				}
				contact = cid
			}
			if contact == uuid.Nil {
				cid, err := q.EarnContactInsert(ctx, store.EarnContactInsertParams{ID: newID(), WorkspaceID: ws.ID, Email: email, MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)),
					Source: SourcePaid, RoleAddress: emailaddr.IsRole(email), At: now})
				if err != nil {
					return err
				}
				contact = cid
			}
		}
		if acc, err := s.accessTx(ctx, q, ws, contact); err != nil {
			return err
		} else if acc != nil && acc.Kind == "complimentary" {
			return problem(ctx, http.StatusConflict, "already_complimentary", "grant.already", nil)
		}
		detail, _ := json.Marshal(map[string]any{"reason": in.Reason})
		grantedBy := userID(p)
		if err := q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: id, WorkspaceID: ws.ID, ContactID: contact, Mode: "live", Kind: "complimentary",
			TierID: &tid, Status: StatusComplimentary, StartedAt: now, SourceKind: "grant", SourceLabel: actorName(p), SourceDetail: detail,
			GrantReason: in.Reason, GrantNote: strings.TrimSpace(in.Note), GrantedBy: grantedBy, GrantUntil: until, At: now}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, contact, &id, "live", "granted", now, nil, "", nil, "", grantedBy,
			map[string]any{"tier": t.Name, "reason": in.Reason, "note": strings.TrimSpace(in.Note), "until": dateOrEmpty(until)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.complimentary_granted", TargetType: "contact", TargetID: contact.String(),
			Reason: in.Reason, After: map[string]any{"tier": t.Name, "until": dateOrEmpty(until), "reason": in.Reason, "note": strings.TrimSpace(in.Note),
				"new_address": newAddress, "welcome_email": in.SendWelcome}})
	})
	if err != nil {
		return EarnMember{}, err
	}
	if newAddress && s.Confirmations != nil && len(t.ListIds) > 0 {
		if err := s.Confirmations.Send(ctx, ws, contact); err != nil {
			s.log().WarnContext(ctx, "complimentary access: confirmation email", "err", err)
		}
	}
	if in.SendWelcome {
		row, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: id, WorkspaceID: ws.ID})
		if err == nil {
			short := s.tierShortName(ctx, s.Q, ws.ID, &tid)
			m := s.memberEmailFor(EmailWelcome, row, ws, short, now)
			s.sendMemberEmail(ctx, ws, "live", m, nil)
		}
	}
	return s.Member(ctx, ws, p, id.String())
}

func dateOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func (s *Service) accessTx(ctx context.Context, q *store.Queries, ws *store.Workspace, contact uuid.UUID) (*access, error) {
	rows, err := q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		x := r.PaidSubscription
		if x.EndedAt == nil && x.Mode == "live" && hasAccess(x.Status, x.AccessPaused) {
			return &access{PaidSubscription: x, TierName: r.TierName}, nil
		}
	}
	return nil, nil
}

// EarnGrantInput changes complimentary access: its tier or end date.
type EarnGrantInput struct {
	TierID string `json:"tierId"`
	Until  string `json:"until,omitempty" maxLength:"10" doc:"YYYY-MM-DD; empty for no end date"`
	Reason string `json:"reason" enum:"team,contributor,press,partner,giveaway,other"`
	Note   string `json:"note,omitempty" maxLength:"500"`
}

func (s *Service) grant(ctx context.Context, ws *store.Workspace, id string) (store.PaidSubscription, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return store.PaidSubscription{}, api.NotFound("Complimentary access")
	}
	sub, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil || sub.Kind != "complimentary" || sub.EndedAt != nil {
		return store.PaidSubscription{}, api.NotFound("Complimentary access")
	}
	return sub, nil
}

// ChangeComplimentary changes complimentary access.
func (s *Service) ChangeComplimentary(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnGrantInput) (EarnMember, error) {
	sub, err := s.grant(ctx, ws, id)
	if err != nil {
		return EarnMember{}, err
	}
	tid, err := uuid.Parse(in.TierID)
	if err != nil {
		return EarnMember{}, invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	if _, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID}); err != nil {
		return EarnMember{}, invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	if !slices.Contains(grantReasons, in.Reason) {
		return EarnMember{}, invalid(ctx, "body.reason", "grant.reason", nil)
	}
	until, err := s.parseUntil(ctx, ws, in.Until)
	if err != nil {
		return EarnMember{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSubscriptionSetGrant(ctx, store.EarnSubscriptionSetGrantParams{ID: sub.ID, TierID: &tid, GrantUntil: until, GrantReason: in.Reason,
			GrantNote: strings.TrimSpace(in.Note), At: now}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, "grant_changed", now, nil, "", nil, "", userID(p),
			map[string]any{"until": dateOrEmpty(until), "reason": in.Reason}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.complimentary_changed", TargetType: "contact", TargetID: sub.ContactID.String(),
			Reason: in.Reason, Before: map[string]any{"until": dateOrEmpty(sub.GrantUntil), "reason": sub.GrantReason, "tier": idString(sub.TierID)},
			After: map[string]any{"until": dateOrEmpty(until), "reason": in.Reason, "tier": tid.String()}})
	})
	if err != nil {
		return EarnMember{}, err
	}
	return s.Member(ctx, ws, p, sub.ID.String())
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// RemoveComplimentary ends complimentary access now.
func (s *Service) RemoveComplimentary(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) error {
	sub, err := s.grant(ctx, ws, id)
	if err != nil {
		return err
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSubscriptionEnd(ctx, store.EarnSubscriptionEndParams{ID: sub.ID, At: now, EndReason: "grant_removed"}); err != nil {
			return err
		}
		if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, "grant_removed", now, nil, "", nil, "", userID(p), nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.complimentary_removed", TargetType: "contact", TargetID: sub.ContactID.String(),
			Before: map[string]any{"tier": idString(sub.TierID), "until": dateOrEmpty(sub.GrantUntil), "reason": sub.GrantReason}})
	})
}

// EarnCancelInput cancels a subscription for a reader who asked by email.
type EarnCancelInput struct {
	When  string `json:"when" enum:"period_end,now"`
	Email bool   `json:"email" doc:"Send the Membership canceled email"`
	Note  string `json:"note,omitempty" maxLength:"500"`
}

// CancelSubscription cancels a paid subscription in the provider, at the
// end of the period or now. Refunds are made in the provider.
func (s *Service) CancelSubscription(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnCancelInput) (EarnMember, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return EarnMember{}, api.NotFound("Paid member")
	}
	sub, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnMember{}, api.MapError(err, "Paid member")
	}
	if sub.Kind != "stripe" || sub.EndedAt != nil || sub.StripeSubscriptionID == "" {
		return EarnMember{}, problem(ctx, http.StatusConflict, "not_cancelable", "cancel.notCancelable", nil)
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnMember{}, err
	}
	conn, ok := connected(conns, sub.Mode)
	if !ok {
		return EarnMember{}, problem(ctx, http.StatusConflict, CodeNotConnected, "cancel.notConnected", nil)
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return EarnMember{}, err
	}
	now := s.now()
	note := strings.TrimSpace(in.Note)
	// The reason goes on the row first, so the events that follow name it.
	if err := s.Q.EarnSubscriptionCancelRequested(ctx, store.EarnSubscriptionCancelRequestedParams{ID: sub.ID, At: now, CanceledBy: userID(p), Reason: note}); err != nil {
		return EarnMember{}, err
	}
	updated, err := client.CancelSubscription(ctx, sub.StripeSubscriptionID, in.When != "now")
	if err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		return EarnMember{}, keyProblem(ctx, err)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		status := paidStatus(updated, now)
		ended, reason := sub.EndedAt, sub.EndReason
		if status == StatusCanceled && ended == nil {
			ended, reason = firstTime(updated.EndedAt, &now), "canceled"
		}
		if err := q.EarnSubscriptionSetState(ctx, store.EarnSubscriptionSetStateParams{ID: sub.ID, TierID: sub.TierID, PriceID: sub.PriceID, StripePriceID: sub.StripePriceID,
			Status: status, Period: sub.Period, AmountCents: sub.AmountCents, Currency: sub.Currency, Discount: sub.Discount, TrialEnd: sub.TrialEnd,
			CurrentPeriodEnd: firstTime(updated.CurrentPeriodEnd, sub.CurrentPeriodEnd), CancelAt: updated.CancelAt, CanceledAt: firstTime(updated.CanceledAt, &now),
			EndedAt: ended, EndReason: reason, StripeCustomerID: sub.StripeCustomerID, SyncedAt: maxTime(sub.StripeSyncedAt, now), At: now}); err != nil {
			return err
		}
		kind := "canceled_by_team"
		data := map[string]any{"when": in.When, "note": note}
		if in.When != "now" && updated.CancelAt != nil {
			data["date"] = updated.CancelAt.UTC().Format(time.RFC3339)
		}
		if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, kind, now, nil, "", nil, "", userID(p), data); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.subscription_canceled", TargetType: "contact", TargetID: sub.ContactID.String(),
			Reason: note, After: map[string]any{"when": in.When, "subscription": sub.StripeSubscriptionID, "email": in.Email}})
	})
	if err != nil {
		return EarnMember{}, err
	}
	if in.Email {
		row, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sub.ID, WorkspaceID: ws.ID})
		if err == nil && !hasEmailed(row, EmailCanceled) {
			short := s.tierShortName(ctx, s.Q, ws.ID, row.TierID)
			s.sendMemberEmail(ctx, ws, row.Mode, s.memberEmailFor(EmailCanceled, row, ws, short, now), nil)
		}
	} else {
		_ = s.Q.EarnSubscriptionEmailed(ctx, store.EarnSubscriptionEmailedParams{ID: sub.ID, Email: EmailCanceled})
	}
	return s.Member(ctx, ws, p, sub.ID.String())
}

// EarnResendInput resends a member email.
type EarnResendInput struct {
	Email string `json:"email" enum:"welcome,trial_ending,payment_failed,renewal_reminder,canceled"`
}

// EarnResent says how resending went.
type EarnResent struct {
	Status string `json:"status" enum:"sent,not_sent"`
	Reason string `json:"reason,omitempty"`
}

// Resend sends a member email again ("Resend Payment failed").
func (s *Service) Resend(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnResendInput) (EarnResent, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return EarnResent{}, api.NotFound("Paid member")
	}
	sub, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnResent{}, api.MapError(err, "Paid member")
	}
	now := s.now()
	short := s.tierShortName(ctx, s.Q, ws.ID, sub.TierID)
	m := s.memberEmailFor(in.Email, sub, ws, short, now)
	m.key = "earn:resend:" + newID().String()
	if in.Email == EmailPaymentFailed {
		m.data["amount"] = money(sub.AmountCents, sub.Currency, false)
		m.data["reason"] = sub.FailureReason
		if sub.NextRetryAt != nil {
			m.data["next_try"] = s.dateText(ws, *sub.NextRetryAt)
		}
	}
	status := s.sendMemberEmail(ctx, ws, sub.Mode, m, nil)
	out := EarnResent{Status: status}
	if status != "sent" {
		out.Reason = "not_published"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.email_resent", TargetType: "contact", TargetID: sub.ContactID.String(),
			Metadata: map[string]any{"email": in.Email, "template": StarterKey(in.Email), "status": status}})
	})
	return out, err
}

// EarnAccessRules are the access rules (Earn › Settings › Access).
type EarnAccessRules struct {
	OnPaymentFailure string   `json:"onPaymentFailure" enum:"keep,end"`
	CanManage        bool     `json:"canManage"`
	Helpers          []string `json:"helpers"`
}

// AccessRules reads the access rules.
func (s *Service) AccessRules(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnAccessRules, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnAccessRules{}, err
	}
	return EarnAccessRules{OnPaymentFailure: st.OnPaymentFailure, CanManage: p.Can(authz.PaymentsManage), Helpers: s.helpers(ctx, ws.ID, authz.PaymentsManage, p.UserID)}, nil
}

// EarnAccessInput changes the access rules.
type EarnAccessInput struct {
	OnPaymentFailure string `json:"onPaymentFailure" enum:"keep,end"`
}

// SaveAccessRules changes the access rules; they apply from the next event.
func (s *Service) SaveAccessRules(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnAccessInput) (EarnAccessRules, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnAccessRules{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSettingsSetAccess(ctx, store.EarnSettingsSetAccessParams{WorkspaceID: ws.ID, OnPaymentFailure: in.OnPaymentFailure, At: now, UpdatedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.access_rules_updated", TargetType: "earn_settings", TargetID: ws.ID.String(),
			TargetName: "Access rules", Before: map[string]any{"on_payment_failure": st.OnPaymentFailure}, After: map[string]any{"on_payment_failure": in.OnPaymentFailure}})
	})
	if err != nil {
		return EarnAccessRules{}, err
	}
	return s.AccessRules(ctx, ws, p)
}

// EarnPagesSettings are the Pages tab: what the upgrade, cancel and gift
// pages show in their footer, and the pages themselves.
type EarnPagesSettings struct {
	TermsURL     string   `json:"termsUrl"`
	SupportEmail string   `json:"supportEmail"`
	UpgradeURL   string   `json:"upgradeUrl"`
	UpgradeHost  string   `json:"upgradeHost"`
	UpgradeLive  bool     `json:"upgradeLive"`
	Visits       int64    `json:"visits"`
	CancelHost   string   `json:"cancelHost"`
	GiftHost     string   `json:"giftHost"`
	CanEdit      bool     `json:"canEdit"`
	Helpers      []string `json:"helpers"`
}

// Pages reads the Pages tab.
func (s *Service) Pages(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnPagesSettings, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnPagesSettings{}, err
	}
	out := EarnPagesSettings{TermsURL: st.TermsUrl, SupportEmail: st.SupportEmail, UpgradeURL: s.pageURL(ctx, ws, "upgrade"), UpgradeHost: s.pageHost(ctx, ws, "upgrade"),
		CancelHost: s.pageHost(ctx, ws, "cancel"), GiftHost: s.pageHost(ctx, ws, "gift"), CanEdit: p.Can(authz.EarnEdit), Helpers: s.helpers(ctx, ws.ID, authz.EarnEdit, p.UserID)}
	out.Visits, _ = s.Q.EarnPageVisits(ctx, store.EarnPageVisitsParams{WorkspaceID: ws.ID, Page: "upgrade", Since: s.now().AddDate(0, 0, -30)})
	if plans, err := s.Plans(ctx, ws, p); err == nil {
		out.UpgradeLive = plans.UpgradePage.Live
	}
	return out, nil
}

// EarnPagesInput changes the pages' footer.
type EarnPagesInput struct {
	TermsURL     string `json:"termsUrl" maxLength:"500"`
	SupportEmail string `json:"supportEmail" maxLength:"320"`
}

// SavePages changes the pages' footer.
func (s *Service) SavePages(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnPagesInput) (EarnPagesSettings, error) {
	terms := strings.TrimSpace(in.TermsURL)
	if terms != "" && !strings.HasPrefix(terms, "https://") && !strings.HasPrefix(terms, "http://") {
		return EarnPagesSettings{}, invalid(ctx, "body.termsUrl", "pages.terms", nil)
	}
	support := strings.TrimSpace(in.SupportEmail)
	if support != "" {
		if e, prob := emailaddr.Check(support); prob != "" {
			return EarnPagesSettings{}, invalid(ctx, "body.supportEmail", "pages.support", nil)
		} else {
			support = e
		}
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnPagesSettings{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSettingsSetPages(ctx, store.EarnSettingsSetPagesParams{WorkspaceID: ws.ID, TermsUrl: terms, SupportEmail: support, At: now, UpdatedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.pages_updated", TargetType: "earn_settings", TargetID: ws.ID.String(), TargetName: "Pages",
			Before: map[string]any{"terms_url": st.TermsUrl, "support_email": st.SupportEmail}, After: map[string]any{"terms_url": terms, "support_email": support}})
	})
	if err != nil {
		return EarnPagesSettings{}, err
	}
	return s.Pages(ctx, ws, p)
}

// EarnMemberEmail is one member email on Earn › Settings › Member emails.
type EarnMemberEmail struct {
	Email      string `json:"email" enum:"welcome,trial_ending,payment_failed,renewal_reminder,canceled,gift_purchased,gift_received,gift_ending"`
	TemplateID string `json:"templateId,omitempty"`
	Template   string `json:"template"`
	Name       string `json:"name"`
	// Status: live (LiveVersion is live), draft (never published), missing.
	Status       string `json:"status" enum:"live,draft,missing"`
	LiveVersion  int    `json:"liveVersion"`
	DraftVersion int    `json:"draftVersion"`
}

// EarnMemberEmails is the Member emails tab.
type EarnMemberEmails struct {
	Emails []EarnMemberEmail `json:"emails"`
	// The sender, reply-to, category and route they go out with.
	FromName      string `json:"fromName"`
	FromEmail     string `json:"fromEmail"`
	ReplyTo       string `json:"replyTo"`
	CategoryName  string `json:"categoryName"`
	CategoryKind  string `json:"categoryKind"`
	RoutePrimary  string `json:"routePrimary"`
	RouteFallback string `json:"routeFallback"`
	// FirstGiftEnding is when the first Gift ending reminder is due (the
	// unpublished warning).
	FirstGiftEnding *time.Time `json:"firstGiftEnding,omitempty"`
	// TierShort names the first tier ("Insider"), for the email names.
	TierShort  string   `json:"tierShort"`
	CanPublish bool     `json:"canPublish"`
	Helpers    []string `json:"helpers"`
	EmailsURL  string   `json:"emailsUrl"`
}

// MemberEmails is the Member emails tab.
func (s *Service) MemberEmails(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnMemberEmails, error) {
	out := EarnMemberEmails{Emails: []EarnMemberEmail{}, CanPublish: p.Can(authz.TransactionalPublish), Helpers: s.helpers(ctx, ws.ID, authz.TransactionalPublish, p.UserID),
		EmailsURL: "https://dashboard.stripe.com/settings/emails"}
	rows, err := s.Q.EarnMemberEmails(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byEmail := map[string]*uuid.UUID{}
	ids := []uuid.UUID{}
	for _, r := range rows {
		byEmail[r.Email] = r.TemplateID
		if r.TemplateID != nil {
			ids = append(ids, *r.TemplateID)
		}
	}
	states := map[uuid.UUID]TemplateState{}
	if s.Mailer != nil && len(ids) > 0 {
		if states, err = s.Mailer.Templates(ctx, ws, ids); err != nil {
			return out, err
		}
	}
	for _, e := range MemberEmailOrder {
		st, _ := starterOf(e)
		me := EarnMemberEmail{Email: e, Template: st.key, Name: st.name, Status: "missing"}
		if id := byEmail[e]; id != nil {
			if ts, ok := states[*id]; ok && !ts.Deleted {
				me.TemplateID, me.Template, me.Name = id.String(), ts.Key, ts.Name
				me.LiveVersion, me.DraftVersion = ts.LiveVersion, ts.DraftVersion
				me.Status = "draft"
				if ts.LiveVersion > 0 {
					me.Status = "live"
				}
				if out.FromEmail == "" {
					out.FromName, out.FromEmail, out.ReplyTo, out.CategoryName, out.CategoryKind = ts.FromName, ts.FromEmail, ts.ReplyTo, ts.CategoryName, ts.CategoryKind
				}
			}
		}
		out.Emails = append(out.Emails, me)
	}
	if s.Mailer != nil {
		if r, err := s.Mailer.Route(ctx, ws); err == nil {
			out.RoutePrimary, out.RouteFallback = r.Primary, r.Fallback
		}
	}
	var at *time.Time
	_ = s.Pool.QueryRow(ctx, `SELECT min(current_period_end) - interval '7 days' FROM paid_subscriptions WHERE workspace_id = $1 AND kind = 'gift'
		AND ended_at IS NULL AND current_period_end > now()`, ws.ID).Scan(&at)
	out.FirstGiftEnding = at
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err == nil && len(tiers) > 0 {
		names, _ := s.Q.EarnListNames(ctx, tiers[0].ListIds)
		out.TierShort = shortTier(tiers[0].Name, names)
	}
	return out, nil
}

// EarnMapPriceInput maps a price no tier knows.
type EarnMapPriceInput struct {
	TierID string `json:"tierId"`
	// Legacy keeps it as a legacy price of the tier (its paid members keep
	// paying it); otherwise it becomes the tier's current price of its period.
	Legacy bool `json:"legacy"`
}

// EarnUnmappedPrice is a price found in the provider that no tier knows.
type EarnUnmappedPrice struct {
	ID            string    `json:"id"`
	Mode          string    `json:"mode" enum:"live,test"`
	PriceID       string    `json:"priceId"`
	Period        string    `json:"period" enum:"month,year,"`
	AmountCents   int64     `json:"amountCents"`
	Currency      string    `json:"currency"`
	FoundAt       time.Time `json:"foundAt"`
	Subscriptions int64     `json:"subscriptions"`
	// Members are the paid members on it (up to 5).
	Members []EarnMemberRow `json:"members"`
}

func (s *Service) unmapped(ctx context.Context, ws *store.Workspace) ([]EarnUnmappedPrice, error) {
	rows, err := s.Q.EarnUnmappedPrices(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []EarnUnmappedPrice{}
	names := s.listNamer(ctx, ws.ID)
	for _, r := range rows {
		u := EarnUnmappedPrice{ID: r.ID.String(), Mode: r.Mode, PriceID: r.StripePriceID, Period: r.Period, AmountCents: r.AmountCents, Currency: r.Currency,
			FoundAt: r.FoundAt, Subscriptions: r.Subscriptions, Members: []EarnMemberRow{}}
		mrows, err := s.Pool.Query(ctx, "SELECT "+memberColumns+" "+memberFrom+" WHERE s.workspace_id = $1 AND s.mode = $2 AND s.stripe_price_id = $3 AND s.ended_at IS NULL ORDER BY s.started_at LIMIT 5",
			ws.ID, r.Mode, r.StripePriceID)
		if err == nil {
			for mrows.Next() {
				if m, err := scanMember(mrows, names); err == nil {
					u.Members = append(u.Members, m)
				}
			}
			mrows.Close()
		}
		out = append(out, u)
	}
	return out, nil
}

// MapPrice maps a price no tier knows to a tier: as a legacy price, or as
// the tier's current price of its period. Nothing changes in the provider;
// MRR and paid status update at once.
func (s *Service) MapPrice(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnMapPriceInput) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return api.NotFound("Price")
	}
	u, err := s.Q.EarnUnmappedPrice(ctx, store.EarnUnmappedPriceParams{ID: uid, WorkspaceID: ws.ID})
	if err != nil || u.MappedAt != nil {
		return api.NotFound("Price")
	}
	tid, err := uuid.Parse(in.TierID)
	if err != nil {
		return invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil {
		return invalid(ctx, "body.tierId", "grant.tier", nil)
	}
	period := u.Period
	if period == "" {
		period = "month"
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		status := "legacy"
		if !in.Legacy {
			status = "current"
			prices, err := q.EarnTierPrices(ctx, tid)
			if err != nil {
				return err
			}
			for _, pr := range prices {
				if pr.Period == period && pr.Status == "current" {
					if err := q.EarnPriceRetire(ctx, store.EarnPriceRetireParams{ID: pr.ID, At: &now}); err != nil {
						return err
					}
				}
			}
		}
		pid := newID()
		live, test := "", ""
		if u.Mode == "live" {
			live = u.StripePriceID
		} else {
			test = u.StripePriceID
		}
		if err := q.EarnPriceInsert(ctx, store.EarnPriceInsertParams{ID: pid, WorkspaceID: ws.ID, TierID: tid, Period: period, AmountCents: u.AmountCents, Currency: u.Currency,
			Status: status, StripeLiveID: live, StripeTestID: test, Origin: "mapped", At: now}); err != nil {
			return err
		}
		if status == "legacy" {
			if _, err := tx.Exec(ctx, `UPDATE earn_prices SET retired_at = $2 WHERE id = $1`, pid, now); err != nil {
				return err
			}
		}
		if err := q.EarnSubscriptionsSetPrice(ctx, store.EarnSubscriptionsSetPriceParams{WorkspaceID: ws.ID, Mode: u.Mode, StripePriceID: u.StripePriceID, PriceID: &pid, TierID: &tid, At: now}); err != nil {
			return err
		}
		if err := q.EarnUnmappedPriceMapped(ctx, store.EarnUnmappedPriceMappedParams{ID: u.ID, At: &now, MappedPriceID: &pid, MappedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.price_mapped", TargetType: "earn_tier", TargetID: tid.String(), TargetName: t.Name,
			After: map[string]any{"price": u.StripePriceID, "amount": priceText(u.AmountCents, u.Currency, period), "legacy": in.Legacy, "subscriptions_moved": true}})
	})
}

var _ = i18n.Args{}
