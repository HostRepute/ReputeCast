package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// NormalizeArgs normalizes one stored webhook payload.
type NormalizeArgs struct {
	RawEventID uuid.UUID `json:"raw_event_id"`
}

// Kind implements river.JobArgs.
func (NormalizeArgs) Kind() string { return "ingest.normalize" }

// InsertOpts puts normalization on the events queue.
func (NormalizeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 8}
}

// Normalizer turns raw provider events into ReputeCast's model.
type Normalizer struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Factory *manage.Factory
	Log     *slog.Logger
	Now     func() time.Time
	// OnSuppressed is called after new addresses joined the shared
	// suppression list, so other providers get them soon (sync job).
	OnSuppressed func(ctx context.Context, workspaceID uuid.UUID)
	// Hooks are told about each event applied to a known message, inside
	// the transaction that applied it (transactional mail's webhooks).
	Hooks []MessageHook
}

// MessageHook is told about an event applied to a message of a stream.
type MessageHook func(ctx context.Context, tx pgx.Tx, p store.Provider, messageID uuid.UUID, stream string, ev providers.Event) error

func (n *Normalizer) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

// NormalizeWorker is the River worker for NormalizeArgs.
type NormalizeWorker struct {
	river.WorkerDefaults[NormalizeArgs]
	N *Normalizer
}

// Work implements river.Worker.
func (w *NormalizeWorker) Work(ctx context.Context, job *river.Job[NormalizeArgs]) error {
	return w.N.Normalize(ctx, job.Args.RawEventID)
}

var postalUA = regexp.MustCompile(`^Postal/([0-9][0-9A-Za-z.\-]*)`)

// Normalize processes a stored payload once; running it again is a no-op.
func (n *Normalizer) Normalize(ctx context.Context, rawID uuid.UUID) error {
	raw, err := n.Q.IngestGetRaw(ctx, rawID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if raw.NormalizedAt != nil {
		return nil
	}
	now := n.now()
	fail := func(msg string) error {
		return n.Q.IngestMarkNormalized(ctx, store.IngestMarkNormalizedParams{ID: raw.ID, NormalizedAt: &now, NormalizeError: &msg})
	}
	if !raw.SignatureOk {
		return fail("signature verification failed")
	}
	p, err := n.Q.ProvGetAny(ctx, raw.ProviderID)
	if err != nil {
		return err
	}
	prov, err := n.Factory.Open(ctx, p)
	if err != nil {
		return err // credentials can't be read: retry later
	}
	req := rebuild(raw)
	events, err := prov.ParseWebhook(providers.WithSignatureVerified(ctx), req, raw.Body)
	if err != nil {
		n.Log.WarnContext(ctx, "normalize webhook", "provider", p.ID, "raw", raw.ID, "err", err)
		if errors.Is(err, providers.ErrUnavailable) {
			return err // the parser can't be reached now (community adapter): retry later
		}
		return fail(err.Error())
	}
	suppressed := 0
	err = db.Tx(ctx, n.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, ev := range events {
			added, err := n.Apply(ctx, tx, p, &raw.ID, raw.Source, raw.ReceivedAt, ev)
			if err != nil {
				return err
			}
			suppressed += added
		}
		if raw.Source == "webhook" {
			if err := q.ProvTouchLastEvent(ctx, store.ProvTouchLastEventParams{ID: p.ID, At: raw.ReceivedAt}); err != nil {
				return err
			}
			if m := postalUA.FindStringSubmatch(headerValue(raw, "User-Agent")); m != nil && p.Adapter == "postal" {
				if _, err := tx.Exec(ctx, `UPDATE providers SET health = jsonb_set(coalesce(health, '{}'::jsonb), '{version}', to_jsonb($2::text)) WHERE id = $1`, p.ID, m[1]); err != nil {
					return err
				}
			}
		}
		return q.IngestMarkNormalized(ctx, store.IngestMarkNormalizedParams{ID: raw.ID, NormalizedAt: &now, Events: int32(len(events))})
	})
	if err != nil {
		return err
	}
	if suppressed > 0 && n.OnSuppressed != nil {
		n.OnSuppressed(ctx, p.WorkspaceID)
	}
	return nil
}

func headerValue(raw store.ProviderRawEvent, key string) string {
	r := rebuild(raw)
	return r.Header.Get(key)
}

// Apply records one normalized event: the delivery event, the message's
// status, campaign counters, provider outcome stats and, for hard bounces,
// complaints and provider unsubscribes, the shared suppression list. It
// returns how many addresses were newly suppressed.
func (n *Normalizer) Apply(ctx context.Context, tx pgx.Tx, p store.Provider, rawID *uuid.UUID, source string, receivedAt time.Time, ev providers.Event) (int, error) {
	q := store.New(tx)
	// Provider-side opens and clicks only count when the workspace chose to
	// keep the provider's tracking; otherwise ReputeCast's own tracking
	// already counted them.
	if (ev.Type == providers.EventOpen || ev.Type == providers.EventClick) && state.TrackingOf(p).Mode != "provider" {
		return 0, nil
	}
	var msg *store.IngestMessageByIDRow
	if ev.MessageID != nil {
		if m, err := q.IngestMessageByID(ctx, store.IngestMessageByIDParams{ID: *ev.MessageID, WorkspaceID: p.WorkspaceID}); err == nil {
			msg = &m
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
	}
	if msg == nil && ev.ProviderMessageID != "" {
		pid := ev.ProviderMessageID
		if m, err := q.IngestMessageByProviderID(ctx, store.IngestMessageByProviderIDParams{ProviderID: &p.ID, ProviderMessageID: &pid}); err == nil {
			mm := store.IngestMessageByIDRow(m)
			msg = &mm
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
	}
	recipient := strings.ToLower(strings.TrimSpace(ev.Recipient))
	if recipient == "" && msg != nil {
		recipient = msg.Email
	}
	// Unsigned mailbox reports only count for a message we sent, to its recipient.
	trusted := true
	if source == "bounce_mailbox" {
		if msg != nil && !strings.EqualFold(recipient, msg.Email) {
			msg = nil
		}
		trusted = msg != nil
	}
	mailbox := routing.MailboxProvider(recipient)
	label := ""
	var messageID, campaignID, contactID *uuid.UUID
	if msg != nil {
		mailbox = msg.MailboxProvider
		label = sourceLabel(msg.CampaignName, msg.AutomationName, msg.AutomationNode, msg.Kind)
		mid := msg.ID
		messageID, campaignID, contactID = &mid, msg.CampaignID, msg.ContactID
	}
	occurred := ev.OccurredAt
	if occurred.IsZero() {
		occurred = receivedAt
	}
	dedupe := dedupeKey(p.ID, ev, recipient, occurred)
	id, err := q.IngestInsertEvent(ctx, store.IngestInsertEventParams{
		ID: newID(), WorkspaceID: p.WorkspaceID, MessageID: messageID, CampaignID: campaignID, ContactID: contactID, ProviderID: &p.ID, RawEventID: rawID,
		Type: string(ev.Type), OccurredAt: occurred, MailboxProvider: mailbox, OriginalType: clip(ev.OriginalType, 200), OriginalCode: clip(ev.OriginalCode, 100),
		Diagnostic: clip(ev.Diagnostic, 1000), BounceClass: ev.BounceClass, Url: clip(ev.URL, 2048), UserAgent: clip(ev.UserAgent, 500),
		Recipient: recipient, ProviderMessageID: clip(ev.ProviderMessageID, 300), Source: label, ReceivedAt: receivedAt, DedupeKey: dedupe,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // a redelivery of an event we already have
	}
	if err != nil {
		return 0, err
	}
	_ = id
	status := map[providers.EventType]string{
		providers.EventDelivered: "delivered", providers.EventDeferred: "deferred", providers.EventHardBounce: "bounced",
		providers.EventSoftBounce: "deferred", providers.EventComplaint: "complained", providers.EventRejected: "rejected",
	}[ev.Type]
	if msg != nil && status != "" {
		errText := ""
		if ev.Type != providers.EventDelivered {
			errText = strings.TrimSpace(ev.OriginalCode + " " + ev.Diagnostic)
		}
		if err := q.IngestUpdateMessage(ctx, store.IngestUpdateMessageParams{ID: msg.ID, Status: status, At: occurred, Error: clip(errText, 500)}); err != nil {
			return 0, err
		}
	}
	if msg != nil {
		for _, h := range n.Hooks {
			if err := h(ctx, tx, p, msg.ID, msg.Stream, ev); err != nil {
				return 0, err
			}
		}
	}
	if msg != nil && msg.CampaignID != nil {
		c := store.IngestBumpCountersParams{CampaignID: *msg.CampaignID, ProviderID: p.ID, MailboxProvider: mailbox, Variant: msg.Variant}
		switch ev.Type {
		case providers.EventDelivered:
			c.Delivered = 1
		case providers.EventDeferred:
			c.Deferred = 1
		case providers.EventHardBounce:
			c.HardBounced = 1
		case providers.EventSoftBounce:
			c.SoftBounced = 1
		case providers.EventComplaint:
			c.Complained = 1
		case providers.EventRejected:
			c.Rejected = 1
		}
		if c.Delivered+c.Deferred+c.HardBounced+c.SoftBounced+c.Complained+c.Rejected > 0 {
			if err := q.IngestBumpCounters(ctx, c); err != nil {
				return 0, err
			}
		}
	}
	out := store.ProvRecordOutcomeParams{ProviderID: p.ID, Day: occurred.UTC()}
	switch ev.Type {
	case providers.EventDelivered:
		out.Delivered = 1
	case providers.EventHardBounce:
		out.Bounced = 1
	case providers.EventComplaint:
		out.Complained = 1
	}
	if trusted && out.Delivered+out.Bounced+out.Complained > 0 {
		if err := q.ProvRecordOutcome(ctx, out); err != nil {
			return 0, err
		}
	}
	// A placement test copy went to a seed mailbox the workspace owns: its
	// bounces and complaints never suppress the seed (internal/placement).
	if msg != nil && msg.Kind == "placement" {
		return 0, nil
	}
	reason, contactStatus := "", ""
	switch ev.Type {
	case providers.EventHardBounce:
		reason, contactStatus = "hard_bounce", "bounced"
	case providers.EventComplaint:
		reason, contactStatus = "complaint", "complained"
	case providers.EventUnsubscribe:
		reason = "unsubscribe"
	}
	if !trusted || reason == "" || recipient == "" {
		return 0, nil
	}
	src := "webhook"
	if source == "bounce_mailbox" {
		src = "bounce_mailbox"
	}
	if _, err := q.IngestAddSuppression(ctx, store.IngestAddSuppressionParams{
		ID: newID(), WorkspaceID: p.WorkspaceID, Email: recipient, Reason: reason, Source: src, SourceProviderID: &p.ID,
		OriginalCode: clip(ev.OriginalCode, 100), Detail: clip(strings.TrimSpace(ev.OriginalType+" · "+ev.Diagnostic), 500), CreatedAt: occurred,
	}); errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // already suppressed
	} else if err != nil {
		return 0, err
	}
	if contactStatus != "" {
		if err := q.IngestSetContactStatus(ctx, store.IngestSetContactStatusParams{WorkspaceID: p.WorkspaceID, Email: recipient, Status: contactStatus}); err != nil {
			return 0, err
		}
	}
	return 1, nil
}

// dedupeKey identifies an event so provider redeliveries are dropped.
func dedupeKey(provider uuid.UUID, ev providers.Event, recipient string, occurred time.Time) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%s|%d|%s|%s", provider, ev.Type, ev.ProviderMessageID, recipient, occurred.UnixMilli(), ev.OriginalCode, ev.OriginalType)
	return hex.EncodeToString(h.Sum(nil))[:40]
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// sourceLabel names what sent a message, for the event log.
func sourceLabel(campaign, automation string, node *string, kind string) string {
	switch {
	case campaign != "":
		return campaign
	case automation != "":
		if node != nil && *node != "" {
			n := *node
			if _, err := strconv.Atoi(n); err == nil {
				n = "step " + n
			}
			return automation + " · " + n
		}
		return automation
	}
	switch kind {
	case "double_opt_in":
		return "Double opt-in confirmation"
	case "preference_link":
		return "Preference-center link"
	case "invite":
		return "Team invite"
	case "password_reset":
		return "Password reset"
	case "api":
		return "API · transactional"
	case "placement":
		return "Placement test copy"
	}
	return ""
}
