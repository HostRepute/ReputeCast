package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// The send-time recheck (review SR-3, contract C3). When a person schedules
// a send they confirm a run: its warnings, its unknown checks (when they
// chose to continue without them) and its recipient count, for the campaign
// as it was. Before the send starts, Send Readiness runs again and the send
// is held for a person when the fresh run blocks, finds a warning or an
// unknown check nobody confirmed, counts an audience more than 10% larger
// or smaller, or when the content, the sender or the route changed after
// scheduling. A held send waits for a person until HeldExpiry after its
// time, then goes back to Draft.

// Why a send is held (SendHold.Kind).
const (
	HoldBlocked    = "blocked"         // a check blocks
	HoldNewWarning = "new_warning"     // a warning nobody confirmed
	HoldUnknown    = "unknown"         // a check couldn't run, and nobody chose to send without it
	HoldAudience   = "audience_change" // the audience changed by more than AudienceChangeLimit
	HoldEdited     = "edited"          // the content, sender or route changed after scheduling
)

// AudienceChangeLimit is how much the audience may grow or shrink after a
// person confirmed it before the send is held.
const AudienceChangeLimit = 0.10

// HeldExpiry is how long after its time a held send waits for a person;
// then it expires back to Draft (schedule and A/B test kept).
const HeldExpiry = 3 * time.Hour

// Digests identify the campaign a person confirmed: its content (with its
// language versions), its sender and its route (the campaign's override
// and the Broadcasts rules).
type Digests struct {
	Content string `json:"content"`
	Sender  string `json:"sender"`
	Route   string `json:"route"`
}

// Digests are the campaign's as it stands.
func (s *Service) Digests(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow) Digests {
	content := c.ContentHash
	if s.Languages != nil {
		content = WithLanguages(content, s.Languages.CampaignDigest(ctx, s.Q, ws.ID, c.ID))
	}
	sender := ""
	if c.SenderID != nil {
		sender = c.SenderID.String()
	}
	h := sha256.New()
	h.Write(c.RouteOverride)
	h.Write([]byte{0})
	h.Write(RoutingKey(ctx, s.Q, ws.ID))
	return Digests{Content: content, Sender: sender, Route: hex.EncodeToString(h.Sum(nil))[:32]}
}

// Confirmation is what a person confirmed when they scheduled a send.
type Confirmation struct {
	RunID       string
	ConfirmedBy *uuid.UUID
	ConfirmedAt time.Time
	ScheduledAt *time.Time
	Recipients  int64
	// Acknowledged are the warnings confirmed; Unknown the checks that
	// were unknown when they chose to continue without them.
	Acknowledged []string
	Unknown      []string
	Digests
}

// Confirm keeps what a person confirmed as they schedule (or start) a
// send: the run they reviewed and the warnings they confirmed, for the
// campaign as it stands. q writes in the caller's transaction.
func (s *Service) Confirm(ctx context.Context, q *store.Queries, ws *store.Workspace, c store.ReadinessCampaignRow, run ReadinessRun, by uuid.UUID,
	acknowledged []string, scheduledAt *time.Time, now time.Time) error {
	unknown := []string{}
	if run.UnknownAcceptedAt != nil {
		for _, ch := range run.Checks {
			if ch.Status == Unknown {
				unknown = append(unknown, ch.Key)
			}
		}
	}
	acks := []string{}
	for _, k := range acknowledged {
		if k != "" && !slices.Contains(acks, k) {
			acks = append(acks, k)
		}
	}
	d := s.Digests(ctx, ws, c)
	var runID *uuid.UUID
	if id, err := uuid.Parse(run.ID); err == nil {
		runID = &id
	}
	var sender *uuid.UUID
	if c.SenderID != nil {
		sender = c.SenderID
	}
	return q.ReadinessConfirmSend(ctx, store.ReadinessConfirmSendParams{CampaignID: c.ID, WorkspaceID: ws.ID, RunID: runID, ConfirmedBy: &by, ConfirmedAt: now,
		ScheduledAt: scheduledAt, Recipients: run.Estimates.Audience.Final, Acknowledged: acks, Unknown: unknown, ContentDigest: d.Content, SenderID: sender,
		RouteDigest: d.Route})
}

// Confirmed is what was confirmed for a campaign's send; nil when nothing
// was (a send scheduled through the API before confirmations were kept).
func (s *Service) Confirmed(ctx context.Context, campaignID uuid.UUID) (*Confirmation, error) {
	r, err := s.Q.ReadinessSendConfirmation(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &Confirmation{ConfirmedBy: r.ConfirmedBy, ConfirmedAt: r.ConfirmedAt, ScheduledAt: r.ScheduledAt, Recipients: r.Recipients,
		Acknowledged: r.Acknowledged, Unknown: r.Unknown, Digests: Digests{Content: r.ContentDigest, Route: r.RouteDigest}}
	if r.RunID != nil {
		out.RunID = r.RunID.String()
	}
	if r.SenderID != nil {
		out.Sender = r.SenderID.String()
	}
	return out, nil
}

// Forget drops what was confirmed (the send was unscheduled or expired).
func (s *Service) Forget(ctx context.Context, q *store.Queries, campaignID uuid.UUID) error {
	return q.ReadinessForgetConfirmation(ctx, campaignID)
}

// SendTimeCheck is what the send-time recheck found: why the send waits
// for a person (no hold: it may start), and the fresh count, which a person
// resuming the send confirms (the typed confirmation uses it).
type SendTimeCheck struct {
	Holds []SendHold `json:"holds"`
	// Audience is the fresh run's final count; Confirmed the count the
	// person confirmed (0 when nothing was confirmed).
	Audience  int64 `json:"audience"`
	Confirmed int64 `json:"confirmed"`
}

// Held reports whether the send must wait for a person.
func (c SendTimeCheck) Held() bool { return len(c.Holds) > 0 }

// SendHold is one reason a send waits.
type SendHold struct {
	Kind string `json:"kind" enum:"blocked,new_warning,unknown,audience_change,edited"`
	// Keys are the checks behind it (blocks, new warnings, unknown checks)
	// or what changed (content, sender, route).
	Keys []string `json:"keys"`
	// Text is the hold's sentence ("The audience grew to 44,010 (+41.1%
	// since you reviewed)"), kept so it reads in each reader's language.
	Text Text `json:"text"`
}

// English is the hold's sentence in English (notifications, the audit
// trail and the sending engine's hold keep it).
func (h SendHold) English() string { return h.Text.en() }

// CheckSendTime decides whether a send may start, comparing the fresh run
// and the campaign as it stands (now) with what was confirmed (conf). Without
// a confirmation, only blocks hold the send.
func CheckSendTime(run ReadinessRun, conf *Confirmation, now Digests) SendTimeCheck {
	out := SendTimeCheck{Holds: []SendHold{}, Audience: run.Estimates.Audience.Final}
	var blocks, warnings, unknown []ReadinessCheck
	for _, ch := range run.Checks {
		switch ch.Status {
		case Block:
			blocks = append(blocks, ch)
		case Warn:
			if conf != nil && !slices.Contains(conf.Acknowledged, ch.Key) {
				warnings = append(warnings, ch)
			}
		case Unknown:
			if conf != nil && !slices.Contains(conf.Unknown, ch.Key) {
				unknown = append(unknown, ch)
			}
		}
	}
	add := func(kind string, checks []ReadinessCheck, key string) {
		if len(checks) == 0 {
			return
		}
		h := SendHold{Kind: kind, Keys: []string{}, Text: tx(key, A{"title": checkTitle(checks[0]), "more": len(checks) - 1})}
		for _, c := range checks {
			h.Keys = append(h.Keys, c.Key)
		}
		out.Holds = append(out.Holds, h)
	}
	add(HoldBlocked, blocks, "sendHold.blocked")
	if conf == nil {
		return out
	}
	out.Confirmed = conf.Recipients
	add(HoldNewWarning, warnings, "sendHold.newWarning")
	add(HoldUnknown, unknown, "sendHold.unknown")
	if conf.Recipients > 0 {
		change := float64(out.Audience-conf.Recipients) / float64(conf.Recipients)
		if math.Abs(change) > AudienceChangeLimit {
			out.Holds = append(out.Holds, SendHold{Kind: HoldAudience, Keys: []string{},
				Text: tx("sendHold.audience", A{"count": out.Audience, "grew": yes(change > 0), "pct": ArgPercent(math.Round(math.Abs(change) * 100))})})
		}
	}
	edited := []string{}
	if now.Content != conf.Content {
		edited = append(edited, "content")
	}
	if now.Sender != conf.Sender {
		edited = append(edited, "sender")
	}
	if now.Route != conf.Route {
		edited = append(edited, "route")
	}
	if len(edited) > 0 {
		parts := []any{}
		for _, e := range edited {
			parts = append(parts, tx("sendHold.part."+e, nil))
		}
		out.Holds = append(out.Holds, SendHold{Kind: HoldEdited, Keys: edited, Text: tx("sendHold.edited", A{"what": argCapital(ArgAnd(parts)), "count": len(edited)})})
	}
	return out
}

// checkTitle is a check's flat title as a text (each reader's language).
func checkTitle(c ReadinessCheck) Text {
	if c.texts != nil && !c.texts.FlatTitle.Empty() {
		return c.texts.FlatTitle
	}
	if c.texts != nil && !c.texts.Title.Empty() {
		return c.texts.Title
	}
	return RawText(firstNonEmpty(c.FlatTitle, c.Title))
}
