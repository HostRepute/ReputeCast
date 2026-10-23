// Package gifts is Earn's gift subscriptions (design expansion phase 6,
// X17): gift options (a tier for a fixed period at a one-time price), the
// hosted gift page and its one payment through the provider, delivery on
// the date the buyer picked, redemption without a card, the reminders,
// and Earn › Gifts. A gift is never revenue in MRR: its sale is income of
// the day it's paid, and its access is like complimentary access.
package gifts

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

//go:embed messages/*.json
var messagesFS embed.FS

// msgs holds the gifts' words: problems, the gift and redeem pages, and
// the lines the gift emails carry. Members read the app's in their app
// language; readers read the pages in the page's language and the emails
// in the email's.
var msgs = i18n.MustLoad(messagesFS, "gifts")

// text is a problem or a line of the app, in the member's language.
func text(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// page is a public page's line in the reader's language: the page's own,
// which internal/publicpages puts in the request's context (English
// without one).
func page(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// english is a line in English: the audit log's names, what goes to the
// payment provider, and the text stored for later readers.
func english(key string, args i18n.Args) string { return msgs.M(key).Text(earn.PageLang, args) }

// A gift's status: checkout (not paid yet), paid (waiting for its
// delivery date), delivered (not redeemed yet), redeemed, expired (the
// link ran out), refunded, abandoned (checkout never finished).
const (
	StatusCheckout  = "checkout"
	StatusPaid      = "paid"
	StatusDelivered = "delivered"
	StatusRedeemed  = "redeemed"
	StatusExpired   = "expired"
	StatusRefunded  = "refunded"
	StatusAbandoned = "abandoned"
)

// Reminder: "Your gift" goes again to a recipient who hasn't redeemed a
// week after delivery; "Gift ending" a week before the access ends;
// messages are kept until the gift ends plus 30 days.
const (
	RemindAfter   = 7 * 24 * time.Hour
	EndingBefore  = 7 * 24 * time.Hour
	MessageKeep   = 30 * 24 * time.Hour
	MessageMax    = 300
	AbandonAfter  = 24 * time.Hour
	MaxDeliverOut = 366 * 24 * time.Hour
)

// LinkMonths are the redemption windows the Gifts tab offers.
var LinkMonths = []int{3, 6, 12}

// Signup joins a gift's recipient to the tier's lists, with the redeem
// page's text as the consent evidence.
type Signup interface {
	Join(ctx context.Context, tx pgx.Tx, ws *store.Workspace, j Join) (contact uuid.UUID, pending bool, err error)
}

// Join is a recipient redeeming a gift.
type Join struct {
	Email, FirstName, LastName string
	Lists                      []uuid.UUID
	// ConsentText is the redeem page's text they agreed to; Note names
	// the gift ("Gift from Priya Natarajan").
	ConsentText, Note string
	IP                *netip.Addr
	UserAgent         string
	At                time.Time
}

// Service is the gifts domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Earn  *earn.Service
	Log   *slog.Logger
	Now   func() time.Time
	// Signup joins recipients to the tier's lists (nil: they join no list).
	Signup Signup
	// Confirm sends the double opt-in email when joining needs it (a
	// recipient who had unsubscribed).
	Confirm func(ctx context.Context, ws *store.Workspace, contact uuid.UUID) error
}

// now is the gifts' clock: their own, else Earn's (tests set it).
func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	if s.Earn != nil && s.Earn.Now != nil {
		return s.Earn.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// codeAlphabet leaves out the letters and digits people mix up (0/O,
// 1/I/L) so a code read aloud or printed is typed right.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// newCode is a gift's redeem code: three groups of four ("G7K2-M9QX-4TBH",
// about 59 bits, so codes can't be guessed).
func newCode() string {
	var b strings.Builder
	n := big.NewInt(int64(len(codeAlphabet)))
	for i := range 12 {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		r, err := rand.Int(rand.Reader, n)
		if err != nil {
			panic(err)
		}
		b.WriteByte(codeAlphabet[r.Int64()])
	}
	return b.String()
}

// NormalizeCode reads a code as typed or linked: upper case, the groups
// joined by dashes.
func NormalizeCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) > 40 {
		return ""
	}
	var b strings.Builder
	for _, r := range code {
		if strings.ContainsRune(codeAlphabet, r) || r == '-' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// period is a gift's length as readers read it: "a year", "3 months".
func period(l i18n.Lang, months int) string {
	return msgs.M("period").Text(l, i18n.Args{"months": months})
}

// periodTitle is the gift card's title: "A year", "3 months".
func periodTitle(l i18n.Lang, months int) string {
	return msgs.M("periodTitle").Text(l, i18n.Args{"months": months})
}

// location is the workspace's time zone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// zoneOr is a gift's time zone (the buyer's), else the workspace's.
func zoneOr(tz string, ws *store.Workspace) *time.Location {
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return location(ws)
}

// event records a gift's history in the caller's transaction.
func event(ctx context.Context, q *store.Queries, gift uuid.UUID, kind string, at time.Time, actor *uuid.UUID, msg *uuid.UUID, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	b, _ := json.Marshal(data)
	return q.GiftEventInsert(ctx, store.GiftEventInsertParams{ID: newID(), GiftID: gift, Kind: kind, At: at, ActorID: actor, MessageID: msg, Data: b})
}

// emit queues a webhook event for the workspace's endpoints that listen.
func (s *Service) emit(ctx context.Context, q *store.Queries, ws *store.Workspace, kind string, data any) error {
	hooks, err := q.HooksMatching(ctx, store.HooksMatchingParams{WorkspaceID: ws.ID, Event: kind})
	if err != nil || len(hooks) == 0 {
		return err
	}
	eventID := newID()
	body, err := json.Marshal(webhooks.Envelope{ID: eventID.String(), Type: kind, CreatedAt: s.now().UTC(), Workspace: ws.Slug, Data: data})
	if err != nil {
		return err
	}
	now := s.now()
	for _, h := range hooks {
		state := "pending"
		if !h.Enabled {
			state = "held"
		}
		if err := q.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{ID: newID(), WebhookID: h.ID, Event: kind, EventID: &eventID, Payload: body,
			State: state, NextAttemptAt: &now, MaxAttempts: webhooks.MaxAttempts}); err != nil {
			return err
		}
	}
	return nil
}

// GiftEvent is the data of the gift.purchased and gift.redeemed webhooks.
type GiftEvent struct {
	GiftID      string     `json:"gift_id"`
	Mode        string     `json:"mode"`
	Tier        string     `json:"tier,omitempty"`
	Months      int        `json:"months"`
	AmountCents int64      `json:"amount_cents"`
	Currency    string     `json:"currency"`
	BuyerEmail  string     `json:"buyer_email"`
	InPerson    bool       `json:"in_person"`
	DeliverAt   *time.Time `json:"deliver_at,omitempty"`
	// Redeemed: who, and until when.
	ContactID     string     `json:"contact_id,omitempty"`
	Email         string     `json:"email,omitempty"`
	AccessEndsAt  *time.Time `json:"access_ends_at,omitempty"`
	OccurredAt    time.Time  `json:"occurred_at"`
	RecipientName string     `json:"recipient_name,omitempty"`
}

// tierShort is a tier's name without its list's ("Insider").
func (s *Service) tierShort(ctx context.Context, name string, lists []uuid.UUID) string {
	names, _ := s.Q.EarnListNames(ctx, lists)
	return earn.ShortTierName(name, names)
}

// firstName is the first word of a name.
func firstName(name string) string {
	f, _, _ := strings.Cut(strings.TrimSpace(name), " ")
	return f
}

func splitName(name string) (string, string) {
	name = strings.TrimSpace(name)
	first, last, _ := strings.Cut(name, " ")
	return first, strings.TrimSpace(last)
}

// TextIn is one of the gift pages' words in a page's language
// (internal/publicpages).
func TextIn(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }
