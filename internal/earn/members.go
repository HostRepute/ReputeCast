package earn

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Member filters (the status chips).
const (
	FilterWithAccess    = "with_access"
	FilterPaying        = "paying"
	FilterTrialing      = "trialing"
	FilterPastDue       = "past_due"
	FilterCancelsSoon   = "cancels_soon"
	FilterComplimentary = "complimentary"
	FilterGift          = "gift"
	FilterEnded         = "ended"
)

// EndedWindow is how far back the Ended chip looks.
const EndedWindow = 30 * 24 * time.Hour

// accessSQL is "has paid access": past due keeps it unless the access rule
// ended it at once.
const accessSQL = `(s.ended_at IS NULL AND (s.status IN ('trialing','active','complimentary','gift') OR (s.status = 'past_due' AND NOT s.access_paused)))`

// EarnMemberFilter narrows the paid members list.
type EarnMemberFilter struct {
	Status string `query:"status" enum:"with_access,paying,trialing,past_due,cancels_soon,complimentary,gift,ended" default:"with_access"`
	Q      string `query:"q" maxLength:"200" doc:"Name, email, billing email or Stripe customer ID"`
	Tier   string `query:"tier" doc:"Tier id"`
	Period string `query:"period" enum:"month,year,"`
	Source string `query:"source" enum:"campaign,automation,page,offer,website,import,gift,grant,direct,"`
	Offer  string `query:"offer" maxLength:"60" doc:"An offer's label (FALL26)"`
	Since  string `query:"since" enum:"7d,30d,90d,1y,"`
	// Campaign is the email a paid member came from (Weekly Digest #141).
	Campaign string `query:"campaign" doc:"Campaign id: paid members credited to it"`
	// Before is "Before the move": paid members since before a date.
	Before string `query:"before" format:"date" doc:"Paid members since before this date"`
	// Test shows test paid members too (Show test data).
	Test bool `query:"test"`
	// AllStatuses searches Ended too ("Search Ended too").
	AllStatuses bool `query:"all"`
	Page        int  `query:"page" minimum:"1" default:"1"`
	PageSize    int  `query:"pageSize" minimum:"1" maximum:"100" default:"8"`
}

// EarnMemberDiscount is a paid member's discount.
type EarnMemberDiscount struct {
	Label       string     `json:"label"`
	AmountCents int64      `json:"amountCents"`
	Percent     float64    `json:"percent,omitempty"`
	Duration    string     `json:"duration,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
}

// EarnMemberSource is how a paid member got here.
type EarnMemberSource struct {
	Kind       string `json:"kind" enum:"campaign,automation,page,offer,website,import,gift,grant,direct,"`
	Label      string `json:"label"`
	CampaignID string `json:"campaignId,omitempty"`
	// GrantedBy and Reason name complimentary access.
	GrantedBy string `json:"grantedBy,omitempty"`
	Reason    string `json:"reason,omitempty"`
	// Platform is the old platform of an import (substack), ImportID the
	// import that brought the paid member in.
	Platform string `json:"platform,omitempty"`
	ImportID string `json:"importId,omitempty"`
}

// EarnMemberRow is one paid member in the list.
type EarnMemberRow struct {
	ID        string `json:"id"`
	ContactID string `json:"contactId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	// Hidden: the viewer lacks subscribers:view, so names and addresses aren't shown.
	Hidden      bool                `json:"hidden"`
	Test        bool                `json:"test"`
	Kind        string              `json:"kind" enum:"stripe,complimentary,gift"`
	Tier        string              `json:"tier"`
	TierName    string              `json:"tierName"`
	Period      string              `json:"period" enum:"month,year,"`
	AmountCents int64               `json:"amountCents"`
	Currency    string              `json:"currency"`
	Discount    *EarnMemberDiscount `json:"discount,omitempty"`
	Legacy      bool                `json:"legacy"`
	Status      string              `json:"status" enum:"trialing,active,past_due,canceled,complimentary,gift"`
	CancelAt    *time.Time          `json:"cancelAt,omitempty"`
	Since       time.Time           `json:"since"`
	// Next is what comes next: renews, ends, retries (past due),
	// trial_ends, no_end (complimentary without an end), ended.
	Next     string           `json:"next" enum:"renews,ends,retries,trial_ends,no_end,ended"`
	NextAt   *time.Time       `json:"nextAt,omitempty"`
	LTVCents int64            `json:"ltvCents"`
	Refunded int64            `json:"refundedCents"`
	Source   EarnMemberSource `json:"source"`
	// EndReason and CancelReason explain an ended row.
	EndReason    string `json:"endReason,omitempty"`
	CancelReason string `json:"cancelReason,omitempty"`
	AccessPaused bool   `json:"accessPaused"`
	// GiftFrom names a gift's giver; GiftPeriod its length (a year).
	GiftFrom   string `json:"giftFrom,omitempty"`
	GiftMonths int    `json:"giftMonths,omitempty"`
	// PeriodShown: the tier sells monthly and yearly, so the period goes
	// with its name ("Insider yearly", but "Founding member").
	PeriodShown bool   `json:"periodShown"`
	TierID      string `json:"tierId,omitempty"`
}

// EarnMemberCounts are the status chips' numbers (live only).
type EarnMemberCounts struct {
	WithAccess    int64 `json:"withAccess"`
	Paying        int64 `json:"paying"`
	Trialing      int64 `json:"trialing"`
	PastDue       int64 `json:"pastDue"`
	CancelsSoon   int64 `json:"cancelsSoon"`
	Complimentary int64 `json:"complimentary"`
	Gift          int64 `json:"gift"`
	Ended         int64 `json:"ended"`
}

// EarnMemberList is Earn › Paid members.
type EarnMemberList struct {
	Items  []EarnMemberRow  `json:"items"`
	Total  int64            `json:"total"`
	Counts EarnMemberCounts `json:"counts"`
	// The summary line: paying, MRR, past due.
	MRRCents int64  `json:"mrrCents"`
	Currency string `json:"currency"`
	// TestMembers counts the test paid members (Show test data).
	TestMembers   int64         `json:"testMembers"`
	TestShown     int64         `json:"testShown"`
	Tiers         []EarnListRef `json:"tiers"`
	Offers        []string      `json:"offers"`
	UpgradeURL    string        `json:"upgradeUrl"`
	UpgradeHost   string        `json:"upgradeHost"`
	CanManage     bool          `json:"canManage"`
	CanExport     bool          `json:"canExport"`
	CanImport     bool          `json:"canImport"`
	CanSeeNames   bool          `json:"canSeeNames"`
	Helpers       []string      `json:"helpers"`
	ExportHelpers []string      `json:"exportHelpers"`
	AccessRule    string        `json:"accessRule" enum:"keep,end"`
	Connected     bool          `json:"connected"`
	// Campaigns are the emails paid members are credited to, newest first
	// (the Source filter's choices); Platforms the imports' old platforms.
	Campaigns []EarnListRef `json:"campaigns"`
	Platforms []string      `json:"platforms"`
	// MovedAt is the first import's day ("Before the move (18 Sep)").
	MovedAt *time.Time `json:"movedAt,omitempty"`
}

type sqlBuild struct {
	where []string
	args  []any
}

func (b *sqlBuild) add(cond string, args ...any) {
	for _, a := range args {
		b.args = append(b.args, a)
		cond = strings.Replace(cond, "?", "$"+strconv.Itoa(len(b.args)), 1)
	}
	b.where = append(b.where, cond)
}

func (s *Service) memberWhere(ws *store.Workspace, f EarnMemberFilter, b *sqlBuild) {
	b.add("s.workspace_id = ?", ws.ID)
	b.add("s.status <> 'incomplete'")
	if f.Test {
		b.add("true")
	} else {
		b.add("s.mode = 'live'")
	}
	q := strings.TrimSpace(f.Q)
	status := f.Status
	if status == "" {
		status = FilterWithAccess
	}
	if q != "" && f.AllStatuses {
		status = "all"
	}
	switch status {
	case FilterWithAccess:
		b.add(accessSQL)
	case FilterPaying:
		b.add("s.ended_at IS NULL AND s.kind = 'stripe' AND s.status IN ('active','past_due')")
	case FilterTrialing:
		b.add("s.ended_at IS NULL AND s.status = 'trialing'")
	case FilterPastDue:
		b.add("s.ended_at IS NULL AND s.status = 'past_due'")
	case FilterCancelsSoon:
		b.add("s.ended_at IS NULL AND s.status = 'active' AND s.cancel_at IS NOT NULL")
	case FilterComplimentary:
		b.add("s.ended_at IS NULL AND s.status = 'complimentary'")
	case FilterGift:
		b.add("s.ended_at IS NULL AND s.status = 'gift'")
	case FilterEnded:
		b.add("s.ended_at IS NOT NULL AND s.ended_at >= ? AND NOT "+unpaidTrialSQL, s.now().Add(-EndedWindow))
	}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		b.add("(lower(c.email) LIKE ? OR lower(c.first_name || ' ' || c.last_name) LIKE ? OR lower(s.billing_email) LIKE ? OR lower(s.stripe_customer_id) LIKE ?)", like, like, like, like)
	}
	if id, err := uuid.Parse(f.Tier); err == nil {
		b.add("s.tier_id = ?", id)
	}
	if f.Period != "" {
		b.add("s.period = ?", f.Period)
	}
	if f.Source != "" {
		b.add("s.source_kind = ?", f.Source)
	}
	if f.Offer != "" {
		b.add("s.discount->>'label' = ?", f.Offer)
	}
	if d := sinceDuration(f.Since); d > 0 {
		b.add("s.started_at >= ?", s.now().Add(-d))
	}
	if id, err := uuid.Parse(f.Campaign); err == nil {
		b.add("s.source_campaign_id = ?", id)
	}
	if t, err := time.Parse("2006-01-02", f.Before); err == nil {
		b.add("s.started_at < ?", t)
	}
}

// unpaidTrialSQL is a trial that ended without a payment: it was never a
// paid member, so Ended leaves it out (the trials' numbers keep it).
const unpaidTrialSQL = "(s.kind = 'stripe' AND s.paid_cents = 0 AND s.end_reason = 'trial_ended')"

func sinceDuration(v string) time.Duration {
	switch v {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	case "1y":
		return 365 * 24 * time.Hour
	}
	return 0
}

// memberOrder: the most recently changed first; past due by the newest
// failure, ended by when access ended.
func memberOrder(status string) string {
	switch status {
	case FilterPastDue:
		return "s.failed_at DESC NULLS LAST, s.id DESC"
	case FilterEnded:
		return "s.ended_at DESC, s.id DESC"
	}
	return "s.updated_at DESC, s.id DESC"
}

const memberColumns = `s.id, s.contact_id, coalesce(c.email, ''), coalesce(c.first_name, ''), coalesce(c.last_name, ''), s.mode, s.kind, s.tier_id,
	coalesce(t.name, ''), s.period, s.amount_cents, s.currency, s.discount, coalesce(p.status, 'current'), s.status, s.cancel_at, s.started_at,
	s.trial_end, s.current_period_end, s.ended_at, s.next_retry_at, s.grant_until, s.paid_cents, s.refunded_cents, s.source_kind, s.source_label,
	s.source_campaign_id, s.source_detail, s.grant_reason, coalesce(u.name, ''), s.end_reason, s.cancel_reason, s.access_paused, s.billing_email`

const memberFrom = `FROM paid_subscriptions s
	JOIN contacts c ON c.id = s.contact_id
	LEFT JOIN earn_tiers t ON t.id = s.tier_id
	LEFT JOIN earn_prices p ON p.id = s.price_id
	LEFT JOIN users u ON u.id = s.granted_by`

func scanMember(rows pgx.Rows, listNames func(*uuid.UUID) []string) (EarnMemberRow, error) {
	var r EarnMemberRow
	var id, contact uuid.UUID
	var first, last, mode, priceStatus, grantedBy, billing string
	var tier, campaign *uuid.UUID
	var discount, detail []byte
	var trialEnd, periodEnd, ended, retry, grantUntil *time.Time
	var paid, refunded int64
	if err := rows.Scan(&id, &contact, &r.Email, &first, &last, &mode, &r.Kind, &tier, &r.TierName, &r.Period, &r.AmountCents, &r.Currency, &discount,
		&priceStatus, &r.Status, &r.CancelAt, &r.Since, &trialEnd, &periodEnd, &ended, &retry, &grantUntil, &paid, &refunded, &r.Source.Kind,
		&r.Source.Label, &campaign, &detail, &r.Source.Reason, &grantedBy, &r.EndReason, &r.CancelReason, &r.AccessPaused, &billing); err != nil {
		return r, err
	}
	r.ID, r.ContactID = id.String(), contact.String()
	r.Name = strings.TrimSpace(first + " " + last)
	r.Test = mode == "test"
	r.Legacy = priceStatus == "legacy"
	r.LTVCents, r.Refunded = paid-refunded, refunded
	r.Tier = shortTier(r.TierName, listNames(tier))
	if tier != nil {
		r.TierID = tier.String()
	}
	if campaign != nil {
		r.Source.CampaignID = campaign.String()
	}
	r.Source.GrantedBy = grantedBy
	var d struct {
		Platform string `json:"platform"`
		ImportID string `json:"import_id"`
		From     string `json:"from"`
		Months   int    `json:"months"`
	}
	_ = json.Unmarshal(detail, &d)
	r.Source.Platform, r.Source.ImportID = d.Platform, d.ImportID
	if r.Kind == "gift" {
		r.GiftFrom, r.GiftMonths = d.From, d.Months
	}
	if len(discount) > 0 {
		var dd struct {
			Label    string  `json:"label"`
			Amount   int64   `json:"amountCents"`
			Percent  float64 `json:"percent"`
			Duration string  `json:"duration"`
			EndsAt   string  `json:"endsAt"`
		}
		if json.Unmarshal(discount, &dd) == nil && dd.Label != "" {
			r.Discount = &EarnMemberDiscount{Label: dd.Label, AmountCents: dd.Amount, Percent: dd.Percent, Duration: dd.Duration}
			if t, err := time.Parse(time.RFC3339, dd.EndsAt); err == nil {
				r.Discount.EndsAt = &t
			}
		}
	}
	switch {
	case ended != nil:
		r.Next, r.NextAt = "ended", ended
	case r.Status == StatusPastDue:
		r.Next, r.NextAt = "retries", retry
	case r.Status == StatusTrialing && trialEnd != nil:
		r.Next, r.NextAt = "trial_ends", trialEnd
	case r.Status == StatusComplimentary && grantUntil == nil:
		r.Next = "no_end"
	case r.Status == StatusComplimentary:
		r.Next, r.NextAt = "ends", grantUntil
	case r.CancelAt != nil:
		r.Next, r.NextAt = "ends", r.CancelAt
	case r.Kind == "gift":
		r.Next, r.NextAt = "ends", periodEnd
	default:
		r.Next, r.NextAt = "renews", periodEnd
	}
	return r, nil
}

// tierPeriods tells which tiers sell both periods, by tier id: their
// period goes with their name.
func (s *Service) tierPeriods(ctx context.Context, wsID uuid.UUID) map[string]bool {
	out := map[string]bool{}
	prices, err := s.Q.EarnPrices(ctx, wsID)
	if err != nil {
		return out
	}
	seen := map[string]map[string]bool{}
	for _, p := range prices {
		if p.Status != "current" {
			continue
		}
		k := p.TierID.String()
		if seen[k] == nil {
			seen[k] = map[string]bool{}
		}
		seen[k][p.Period] = true
	}
	for k, v := range seen {
		out[k] = v["month"] && v["year"]
	}
	return out
}

// listNamer names tiers' lists (for the short tier names), cached.
func (s *Service) listNamer(ctx context.Context, wsID uuid.UUID) func(*uuid.UUID) []string {
	cache := map[uuid.UUID][]string{}
	tiers, _ := s.Q.EarnTiers(ctx, wsID)
	byTier := map[uuid.UUID][]uuid.UUID{}
	for _, t := range tiers {
		byTier[t.ID] = t.ListIds
	}
	return func(id *uuid.UUID) []string {
		if id == nil {
			return nil
		}
		if v, ok := cache[*id]; ok {
			return v
		}
		names, _ := s.Q.EarnListNames(ctx, byTier[*id])
		cache[*id] = names
		return names
	}
}

// Members is Earn › Paid members.
func (s *Service) Members(ctx context.Context, ws *store.Workspace, p *authz.Principal, f EarnMemberFilter) (EarnMemberList, error) {
	if f.PageSize == 0 {
		f.PageSize = 8
	}
	if f.Page == 0 {
		f.Page = 1
	}
	out := EarnMemberList{Items: []EarnMemberRow{}, Tiers: []EarnListRef{}, Offers: []string{},
		CanManage: p.Can(authz.EarnManage), CanExport: p.Can(authz.SubscribersExport), CanImport: p.Can(authz.EarnManage) && p.Can(authz.SubscribersImport),
		CanSeeNames: p.Can(authz.SubscribersView), Helpers: s.helpers(ctx, ws.ID, authz.EarnManage, p.UserID),
		ExportHelpers: s.helpers(ctx, ws.ID, authz.SubscribersExport, p.UserID),
		UpgradeURL:    s.pageURL(ctx, ws, "upgrade"), UpgradeHost: s.pageHost(ctx, ws, "upgrade")}
	b := &sqlBuild{}
	s.memberWhere(ws, f, b)
	where := strings.Join(b.where, " AND ")
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) "+memberFrom+" WHERE "+where, b.args...).Scan(&out.Total); err != nil {
		return out, err
	}
	args := append(append([]any{}, b.args...), f.PageSize, (f.Page-1)*f.PageSize)
	status := f.Status
	if status == "" {
		status = FilterWithAccess
	}
	order := memberOrder(status)
	if f.Test {
		// Test rows first, as the board shows them.
		order = "(s.mode = 'test') DESC, " + order
	}
	rows, err := s.Pool.Query(ctx, fmt.Sprintf("SELECT %s %s WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d", memberColumns, memberFrom, where, order, len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	names := s.listNamer(ctx, ws.ID)
	periods := s.tierPeriods(ctx, ws.ID)
	for rows.Next() {
		r, err := scanMember(rows, names)
		if err != nil {
			rows.Close()
			return out, err
		}
		r.PeriodShown = periods[r.TierID]
		if !out.CanSeeNames {
			r.Name, r.Email, r.Hidden = "", "", true
		}
		if r.Test {
			out.TestShown++
		}
		out.Items = append(out.Items, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	counts, mrr, err := s.countMembers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Counts, out.MRRCents = counts, mrr
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'test' AND status <> 'incomplete'`, ws.ID).Scan(&out.TestMembers)
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range tiers {
		out.Tiers = append(out.Tiers, EarnListRef{ID: t.ID.String(), Name: shortTier(t.Name, names(&t.ID))})
	}
	offers, err := s.Pool.Query(ctx, `SELECT DISTINCT discount->>'label' FROM paid_subscriptions WHERE workspace_id = $1 AND discount IS NOT NULL AND discount->>'label' <> '' ORDER BY 1`, ws.ID)
	if err == nil {
		for offers.Next() {
			var o string
			if offers.Scan(&o) == nil {
				out.Offers = append(out.Offers, o)
			}
		}
		offers.Close()
	}
	cur, _ := s.tierCurrency(ctx, ws)
	out.Currency = cur
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err == nil {
		out.AccessRule = st.OnPaymentFailure
	}
	conns, _ := s.connections(ctx, ws.ID)
	_, out.Connected = active(conns)
	out.Campaigns, out.Platforms = []EarnListRef{}, []string{}
	if rows, err := s.Pool.Query(ctx, `SELECT c.id, c.name FROM campaigns c WHERE c.workspace_id = $1 AND c.id IN
		(SELECT DISTINCT source_campaign_id FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live' AND source_campaign_id IS NOT NULL)
		ORDER BY coalesce(c.started_at, c.scheduled_at, c.created_at) DESC LIMIT 6`, ws.ID); err == nil {
		for rows.Next() {
			var r EarnListRef
			var id uuid.UUID
			if rows.Scan(&id, &r.Name) == nil {
				r.ID = id.String()
				out.Campaigns = append(out.Campaigns, r)
			}
		}
		rows.Close()
	}
	if rows, err := s.Pool.Query(ctx, `SELECT DISTINCT source_detail->>'platform' FROM paid_subscriptions WHERE workspace_id = $1 AND source_kind = 'import'
		AND coalesce(source_detail->>'platform', '') <> '' ORDER BY 1`, ws.ID); err == nil {
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil {
				out.Platforms = append(out.Platforms, p)
			}
		}
		rows.Close()
	}
	_ = s.Pool.QueryRow(ctx, `SELECT min(created_at) FROM earn_imports WHERE workspace_id = $1 AND mode = 'live' AND status = 'done'`, ws.ID).Scan(&out.MovedAt)
	return out, nil
}

// EarnTimelineItem is one entry of a paid member's timeline.
type EarnTimelineItem struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	At          time.Time      `json:"at"`
	AmountCents *int64         `json:"amountCents,omitempty"`
	Currency    string         `json:"currency,omitempty"`
	StripeEvent string         `json:"stripeEvent,omitempty"`
	Actor       string         `json:"actor,omitempty"`
	Test        bool           `json:"test"`
	Data        map[string]any `json:"data"`
	// MessageStatus is the member email's delivery (delivered, bounced…) and
	// Provider who carried it.
	MessageStatus string `json:"messageStatus,omitempty"`
	Provider      string `json:"provider,omitempty"`
	MessageID     string `json:"messageId,omitempty"`
}

// EarnSyncWarning is a second active subscription for one person.
type EarnSyncWarning struct {
	Subscriptions []EarnMemberRow `json:"subscriptions"`
}

// EarnMember is the member panel.
type EarnMember struct {
	EarnMemberRow
	// Stripe holds the provider's ids and dashboard links.
	CustomerID      string             `json:"customerId,omitempty"`
	SubscriptionID  string             `json:"subscriptionId,omitempty"`
	CustomerURL     string             `json:"customerUrl,omitempty"`
	SubscriptionURL string             `json:"subscriptionUrl,omitempty"`
	BillingEmail    string             `json:"billingEmail,omitempty"`
	BillingDiffers  bool               `json:"billingDiffers"`
	FailureReason   string             `json:"failureReason,omitempty"`
	FailedAt        *time.Time         `json:"failedAt,omitempty"`
	RenewalReminder *time.Time         `json:"renewalReminder,omitempty"`
	PaidCents       int64              `json:"paidCents"`
	ClickedAt       *time.Time         `json:"clickedAt,omitempty"`
	ImportedAt      *time.Time         `json:"importedAt,omitempty"`
	MatchedBy       string             `json:"matchedBy,omitempty"`
	GrantUntil      *time.Time         `json:"grantUntil,omitempty"`
	GrantNote       string             `json:"grantNote,omitempty"`
	CanceledBy      string             `json:"canceledBy,omitempty"`
	CanceledAt      *time.Time         `json:"canceledAt,omitempty"`
	Warning         *EarnSyncWarning   `json:"warning,omitempty"`
	Timeline        []EarnTimelineItem `json:"timeline"`
	// What the viewer may do, and who can help when they can't.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	// Emails lists the member emails that can be resent to this paid member.
	Emails []string `json:"emails"`
	Link   string   `json:"link"`
	// Lists are the tier's lists ("already on Weekly Digest"); TaxAdded says
	// amounts are before tax, which checkout adds.
	Lists    []string `json:"lists"`
	TaxAdded bool     `json:"taxAdded"`
}

// Member is the member panel of a paid subscription.
func (s *Service) Member(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnMember, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return EarnMember{}, api.NotFound("Paid member")
	}
	rows, err := s.Pool.Query(ctx, "SELECT "+memberColumns+" "+memberFrom+" WHERE s.id = $1 AND s.workspace_id = $2", sid, ws.ID)
	if err != nil {
		return EarnMember{}, err
	}
	names := s.listNamer(ctx, ws.ID)
	var row EarnMemberRow
	found := false
	for rows.Next() {
		if row, err = scanMember(rows, names); err != nil {
			rows.Close()
			return EarnMember{}, err
		}
		found = true
	}
	row.PeriodShown = s.tierPeriods(ctx, ws.ID)[row.TierID]
	rows.Close()
	if !found {
		return EarnMember{}, api.NotFound("Paid member")
	}
	sub, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnMember{}, api.MapError(err, "Paid member")
	}
	out := EarnMember{EarnMemberRow: row, CanManage: p.Can(authz.EarnManage), Helpers: s.helpers(ctx, ws.ID, authz.EarnManage, p.UserID),
		Timeline: []EarnTimelineItem{}, Emails: []string{}, PaidCents: sub.PaidCents, FailureReason: sub.FailureReason, FailedAt: sub.FailedAt,
		GrantUntil: sub.GrantUntil, GrantNote: sub.GrantNote, CanceledAt: sub.CanceledAt}
	if !p.Can(authz.SubscribersView) {
		out.Name, out.Email, out.Hidden = "", "", true
	}
	mode := payments.Mode(sub.Mode)
	if sub.StripeCustomerID != "" {
		out.CustomerID, out.CustomerURL = sub.StripeCustomerID, dashboard(DefaultProvider, mode, "customers", sub.StripeCustomerID)
	}
	if sub.StripeSubscriptionID != "" {
		out.SubscriptionID, out.SubscriptionURL = sub.StripeSubscriptionID, dashboard(DefaultProvider, mode, "subscriptions", sub.StripeSubscriptionID)
	}
	if !out.Hidden {
		out.BillingEmail = sub.BillingEmail
		out.BillingDiffers = sub.BillingEmail != "" && !strings.EqualFold(sub.BillingEmail, row.Email)
	}
	if sub.CanceledBy != nil {
		out.CanceledBy, _ = s.Q.EarnUserName(ctx, *sub.CanceledBy)
	}
	if sub.Period == "year" && sub.CurrentPeriodEnd != nil && sub.CancelAt == nil && sub.EndedAt == nil {
		t := sub.CurrentPeriodEnd.AddDate(0, 0, -7)
		out.RenewalReminder = &t
	}
	var d struct {
		ClickedAt string `json:"clicked_at"`
		MatchedBy string `json:"matched_by"`
		Imported  string `json:"imported_at"`
	}
	_ = json.Unmarshal(sub.SourceDetail, &d)
	if t, err := time.Parse(time.RFC3339, d.ClickedAt); err == nil {
		out.ClickedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, d.Imported); err == nil {
		out.ImportedAt = &t
	}
	out.MatchedBy = d.MatchedBy
	// A second active subscription of the same person (the sync warning).
	others, err := s.Q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: sub.ContactID, WorkspaceID: ws.ID})
	if err == nil {
		var w []EarnMemberRow
		for _, o := range others {
			x := o.PaidSubscription
			if x.Kind == "stripe" && x.EndedAt == nil && x.Mode == sub.Mode && x.Status != StatusIncomplete {
				w = append(w, EarnMemberRow{ID: x.ID.String(), Tier: shortTier(o.TierName, names(x.TierID)), TierName: o.TierName, Period: x.Period,
					AmountCents: x.AmountCents, Currency: x.Currency, Status: x.Status, Since: x.StartedAt})
			}
		}
		if len(w) > 1 {
			out.Warning = &EarnSyncWarning{Subscriptions: w}
		}
	}
	items, err := s.timelineOf(ctx, ws, sub.ContactID, sub.Mode)
	if err != nil {
		return out, err
	}
	out.Timeline = items
	switch {
	case sub.Kind == "complimentary":
		out.Emails = []string{EmailWelcome}
	case sub.EndedAt != nil:
		out.Emails = []string{EmailCanceled}
	case sub.Status == StatusPastDue:
		out.Emails = []string{EmailPaymentFailed, EmailWelcome}
	case sub.Status == StatusTrialing:
		out.Emails = []string{EmailWelcome, EmailTrialEnding}
	default:
		out.Emails = []string{EmailWelcome}
		if sub.Period == "year" {
			out.Emails = append(out.Emails, EmailRenewalReminder)
		}
		if sub.CancelAt != nil {
			out.Emails = append(out.Emails, EmailCanceled)
		}
	}
	out.Link = "/w/" + ws.Slug + "/earn/members?member=" + sub.ID.String()
	out.Lists = names(sub.TierID)
	if out.Lists == nil {
		out.Lists = []string{}
	}
	if st, err := s.settingsRow(ctx, s.Q, ws.ID); err == nil {
		out.TaxAdded = st.TaxAtCheckout && !st.PricesIncludeTax
	}
	return out, nil
}

// timelineOf is a contact's paid timeline with each member email's delivery.
func (s *Service) timelineOf(ctx context.Context, ws *store.Workspace, contact uuid.UUID, mode string) ([]EarnTimelineItem, error) {
	rows, err := s.Q.EarnTimeline(ctx, store.EarnTimelineParams{ContactID: contact, WorkspaceID: ws.ID, Mode: mode, Lim: 100})
	if err != nil {
		return nil, err
	}
	out := []EarnTimelineItem{}
	var msgIDs []uuid.UUID
	for _, r := range rows {
		it := EarnTimelineItem{ID: r.ID.String(), Kind: r.Kind, At: r.OccurredAt, AmountCents: r.AmountCents, Currency: r.Currency, StripeEvent: r.StripeEvent,
			Actor: r.ActorName, Test: r.Mode == "test", Data: map[string]any{}}
		_ = json.Unmarshal(r.Data, &it.Data)
		if r.MessageID != nil {
			it.MessageID = r.MessageID.String()
			msgIDs = append(msgIDs, *r.MessageID)
		}
		out = append(out, it)
	}
	if len(msgIDs) > 0 {
		q, err := s.Pool.Query(ctx, `SELECT m.id, m.status, coalesce(p.name, '') FROM messages m LEFT JOIN providers p ON p.id = m.provider_id WHERE m.id = ANY($1) AND m.workspace_id = $2`, msgIDs, ws.ID)
		if err == nil {
			st := map[string][2]string{}
			for q.Next() {
				var id uuid.UUID
				var status, provider string
				if q.Scan(&id, &status, &provider) == nil {
					st[id.String()] = [2]string{status, provider}
				}
			}
			q.Close()
			for i := range out {
				if v, ok := st[out[i].MessageID]; ok {
					out[i].MessageStatus, out[i].Provider = v[0], v[1]
				}
			}
		}
	}
	return out, nil
}

// ContactPaid is a contact's paid subscription for the contact profile's
// card (the one with access, else the latest).
func (s *Service) ContactPaid(ctx context.Context, ws *store.Workspace, p *authz.Principal, contact string) (*EarnMember, error) {
	cid, err := uuid.Parse(contact)
	if err != nil {
		return nil, api.NotFound("Contact")
	}
	rows, err := s.Q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: cid, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	m, err := s.Member(ctx, ws, p, rows[0].PaidSubscription.ID.String())
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Export writes the paid members of a filter as CSV (subscribers:export,
// audited). Amounts are in the currency's major unit, before tax.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, p *authz.Principal, f EarnMemberFilter) ([]byte, int, error) {
	b := &sqlBuild{}
	f.AllStatuses = true
	s.memberWhere(ws, f, b)
	where := strings.Join(b.where, " AND ")
	rows, err := s.Pool.Query(ctx, "SELECT "+memberColumns+" "+memberFrom+" WHERE "+where+" ORDER BY "+memberOrder(f.Status)+" LIMIT 100000", b.args...)
	if err != nil {
		return nil, 0, err
	}
	names := s.listNamer(ctx, ws.ID)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"email", "name", "tier", "period", "price", "currency", "status", "cancels_on", "member_since", "renews_or_ends", "lifetime_value", "source", "test"})
	n := 0
	for rows.Next() {
		r, err := scanMember(rows, names)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		at := func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.UTC().Format("2006-01-02")
		}
		amount := r.AmountCents
		if r.Discount != nil && r.Discount.AmountCents > 0 {
			amount = r.Discount.AmountCents
		}
		_ = w.Write([]string{r.Email, r.Name, r.TierName, r.Period, strconv.FormatFloat(float64(amount)/100, 'f', 2, 64), strings.ToUpper(r.Currency), r.Status,
			at(r.CancelAt), r.Since.UTC().Format("2006-01-02"), at(r.NextAt), strconv.FormatFloat(float64(r.LTVCents)/100, 'f', 2, 64), firstNonEmpty(r.Source.Label, r.Source.Kind),
			strconv.FormatBool(r.Test)})
		n++
	}
	rows.Close()
	w.Flush()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.exported", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Metadata: map[string]any{"rows": n, "status": f.Status, "tier": f.Tier, "test": f.Test, "q": f.Q}})
	})
	return buf.Bytes(), n, err
}

// countMembers counts the live paid members by status, with the MRR: yearly
// prices count a twelfth, after any discount; trials, complimentary access
// and gifts count nothing (ROADMAP defaults).
func (s *Service) countMembers(ctx context.Context, wsID uuid.UUID) (EarnMemberCounts, int64, error) {
	var c EarnMemberCounts
	var mrr int64
	err := s.Pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE `+accessSQL+`),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.kind = 'stripe' AND s.status IN ('active','past_due')),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.status = 'trialing'),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.status = 'past_due'),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.status = 'active' AND s.cancel_at IS NOT NULL),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.status = 'complimentary'),
		count(*) FILTER (WHERE s.ended_at IS NULL AND s.status = 'gift'),
		count(*) FILTER (WHERE s.ended_at IS NOT NULL AND s.ended_at >= $2 AND NOT `+unpaidTrialSQL+`),
		round(coalesce(sum(CASE WHEN s.ended_at IS NULL AND s.kind = 'stripe' AND s.status IN ('active','past_due') THEN
			CASE WHEN s.period = 'year' THEN coalesce((s.discount->>'amountCents')::bigint, s.amount_cents) / 12.0
			ELSE coalesce((s.discount->>'amountCents')::bigint, s.amount_cents) END ELSE 0 END), 0))::bigint
		FROM paid_subscriptions s WHERE s.workspace_id = $1 AND s.mode = 'live' AND s.status <> 'incomplete'`, wsID, s.now().Add(-EndedWindow)).Scan(
		&c.WithAccess, &c.Paying, &c.Trialing, &c.PastDue, &c.CancelsSoon, &c.Complimentary,
		&c.Gift, &c.Ended, &mrr)
	return c, mrr, err
}
