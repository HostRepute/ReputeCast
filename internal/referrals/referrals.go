// Package referrals implements Grow › Referrals (design expansion phase 5,
// M25; brief reputecast-design-addon-grow.md §3.1–3.7, §3.10–3.13): one
// referral program per list, personal links, the counting and suspicion
// rules, milestones and their rewards, the review queue, results and
// referrers, the public hub and leaderboard data, the Referral and Reward
// blocks and the referral tokens.
//
// A referral is recorded when someone signs up through a link (the invite
// page, or a form or the hosted page opened with ?ref=). It starts its hold
// when its subscriber confirms (double opt-in is always on for referrals)
// and counts when the hold ends with the subscriber still on the list.
// Obvious abuse is excluded automatically and anything suspicious waits for
// a person. Automatic events stay in the program's referral log
// (referral_events), not the audit log; everything a person changes is
// audited.
package referrals

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "referrals")

// Statuses of a referral.
const (
	StatusAwaiting   = "awaiting"    // signed up, hasn't confirmed yet
	StatusHold       = "hold"        // confirmed, waiting for the hold to end
	StatusReview     = "review"      // held for review: a person decides
	StatusCounted    = "counted"     // final; counts toward milestones
	StatusNotCounted = "not_counted" // excluded, with its reason
)

// Reasons a referral isn't counted.
const (
	ReasonSelf           = "self"
	ReasonSameIP         = "same_ip"
	ReasonDisposable     = "disposable"
	ReasonLeft           = "left"
	ReasonAlready        = "already"
	ReasonNeverConfirmed = "never_confirmed"
	ReasonReviewer       = "reviewer"
)

// Suspicion signals that hold a referral for review.
const (
	SignalBurst    = "burst"
	SignalSharedIP = "shared_ip"
	SignalNumbered = "numbered"
)

// Reward types.
const (
	RewardDownload = "download"
	RewardCodes    = "codes"
	RewardManual   = "manual"
	RewardCoupon   = "coupon"
	RewardWebhook  = "webhook"
)

// Reward statuses.
const (
	RewardPending      = "pending"
	RewardSent         = "sent"
	RewardWaitingCodes = "waiting_codes"
	RewardToFulfill    = "to_fulfill"
	RewardFulfilled    = "fulfilled"
	RewardFailed       = "failed"
	RewardExpired      = "expired"
)

// Timing rules of the brief.
const (
	// NeverConfirmedAfter: no confirmation within 7 days.
	NeverConfirmedAfter = 7 * 24 * time.Hour
	// SameIPWindow: the addresses a referrer used in the last 30 days.
	SameIPWindow = 30 * 24 * time.Hour
	// IPKeptFor: addresses are kept 90 days for review, then reduced to
	// their network.
	IPKeptFor = 90 * 24 * time.Hour
	// AddressKeptFor: shipping addresses are deleted 30 days after the
	// reward is marked fulfilled.
	AddressKeptFor = 30 * 24 * time.Hour
	// DownloadValid: links to uploaded files work 7 days.
	DownloadValid = 7 * 24 * time.Hour
	// HubValid: a hub link opens the hub for 60 days.
	HubValid = 60 * 24 * time.Hour
	// UndoFor: a review decision can be undone for 10 seconds.
	UndoFor = 10 * time.Second
	// WebhookRetryFor: webhook rewards are retried for 6 hours.
	WebhookRetryFor = 6 * time.Hour
)

// Inserter enqueues River jobs (the job client, read at call time).
type Inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service is the referrals domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	Now   func() time.Time
	// Links signs the personal links of the hub, downloads and the
	// recommendations page.
	Links *linktoken.Signer
	// Pages are a workspace's public page addresses (its custom domain when
	// verified).
	Pages func(ctx context.Context, ws *store.Workspace) (forms.PageLinks, error)
	// Mail sends system mail: the hub's "Email me my link" (App.Mail, read
	// at call time).
	Mail func() mailer.Sender
	// Emails queues milestone emails on the Automations stream.
	Emails func() MilestoneSender
	// Coupons issues Stripe promotion codes once payments exist (phase 6);
	// nil means Stripe isn't connected.
	Coupons func() Coupons
	// HTTP posts webhook rewards through the SSRF guard.
	HTTP func() *http.Client
	// AllowPrivate lets webhook rewards reach private addresses (tests and
	// development, webhooks.allow_private_networks).
	AllowPrivate func() bool
	// Jobs enqueues River jobs.
	Jobs func() Inserter
	// Base is the instance's public URL (images in emails).
	Base func() string
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) pages(ctx context.Context, ws *store.Workspace) (forms.PageLinks, error) {
	if s.Pages == nil {
		return forms.LinksOn("http://localhost", ws.Slug, false), nil
	}
	return s.Pages(ctx, ws)
}

// Coupons is the payment provider's side of the Stripe coupon reward (phase
// 6): whether Stripe is connected, the offers of the kind "Referral reward"
// and a single-use promotion code with its claim link per reward.
type Coupons interface {
	Status(ctx context.Context, ws *store.Workspace) (CouponStatus, error)
	Offers(ctx context.Context, ws *store.Workspace) ([]CouponOffer, error)
	Issue(ctx context.Context, ws *store.Workspace, offerID string, contact uuid.UUID, validDays int) (IssuedCoupon, error)
	// ClaimURL is an issued code's claim link as it stands: its referrer's
	// personal upgrade link with the code.
	ClaimURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID, code string) string
}

// CouponStatus says whether a payment provider can issue coupons.
type CouponStatus struct {
	Connected bool
	// Account is the connected account's name ("Acme Inc").
	Account string
}

// CouponOffer is an offer of the kind "Referral reward".
type CouponOffer struct {
	ID          string
	Name        string
	Discount    string
	AppliesTo   string
	Issued      int64
	Redeemed    int64
	PaidMembers bool
}

// IssuedCoupon is a single-use promotion code and its claim link.
type IssuedCoupon struct {
	Code      string
	ClaimURL  string
	ExpiresAt time.Time
}

// MilestoneSender queues a milestone email for a referrer on the Automations
// stream (the sending engine's automation outbox), inside the transaction
// that gave the reward.
type MilestoneSender interface {
	QueueMilestoneEmail(ctx context.Context, tx pgx.Tx, ws uuid.UUID, emailID, contactID uuid.UUID, to string) (uuid.UUID, error)
}

// GrowHelpers names who can help with a Grow permission: up to three active
// members holding it, the people who run the program first (Editors and
// custom roles, then Admins, then Owners), as the boards write them ("Ask
// Sofia Rossi, Marcus Chen or Linh Nguyen").
func GrowHelpers(ctx context.Context, q *store.Queries, wsID uuid.UUID, perm authz.Permission) []string {
	rows, err := q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return []string{}
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleOwner:
			return 2
		case authz.RoleAdmin:
			return 1
		}
		return 0
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	names := []string{}
	for _, r := range rows {
		if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			continue
		}
		names = append(names, r.Name)
		if len(names) == 3 {
			break
		}
	}
	return names
}

// GrowHelperSets are who can help with each Grow permission, for the
// read-only lines and disabled controls.
type GrowHelperSets struct {
	Edit    []string `json:"edit"`
	Review  []string `json:"review"`
	Connect []string `json:"connect"`
	Export  []string `json:"export"`
	Earn    []string `json:"earn"`
}

func (s *Service) helperSets(ctx context.Context, wsID uuid.UUID) GrowHelperSets {
	return GrowHelperSets{
		Edit:    GrowHelpers(ctx, s.Q, wsID, authz.GrowEdit),
		Review:  GrowHelpers(ctx, s.Q, wsID, authz.GrowReview),
		Connect: GrowHelpers(ctx, s.Q, wsID, authz.GrowConnect),
		Export:  GrowHelpers(ctx, s.Q, wsID, authz.SubscribersExport),
		Earn:    GrowHelpers(ctx, s.Q, wsID, authz.EarnEdit),
	}
}

// displayName is a contact's name, or their address when they have none.
func displayName(first, last, email string) string {
	n := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
	if n == "" {
		return email
	}
	return n
}

// shortName is how a reader shows on the leaderboard when they allow it:
// "Sam R.".
func shortName(first, last string) string {
	first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	if first == "" {
		return ""
	}
	if last == "" {
		return first
	}
	r := []rune(last)
	return first + " " + strings.ToUpper(string(r[0])) + "."
}

// workspaceLocation is the workspace's time zone (days and weeks).
func workspaceLocation(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

func dayOf(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
