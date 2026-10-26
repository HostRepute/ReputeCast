package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
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
	// Engagement counts the opens and clicks a provider reports when the
	// workspace keeps the provider's tracking (review AN-4); without it
	// they are only logged.
	Engagement Engagement
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
	err = sending.Tx(ctx, n.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		added, err := n.ApplyAll(ctx, tx, p, &raw.ID, raw.Source, raw.ReceivedAt, events)
		if err != nil {
			return err
		}
		suppressed = added
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
// state, campaign counters, provider outcome stats and, for hard bounces,
// complaints and provider unsubscribes, the shared suppression list. It
// returns how many addresses were newly suppressed. A caller applying
// several events in one transaction uses ApplyAll, so the rows they share
// are locked once, in key order.
func (n *Normalizer) Apply(ctx context.Context, tx pgx.Tx, p store.Provider, rawID *uuid.UUID, source string, receivedAt time.Time, ev providers.Event) (int, error) {
	q := store.New(tx)
	msg, err := n.message(ctx, q, p, ev)
	if err != nil {
		return 0, err
	}
	var e effects
	if err := n.apply(ctx, tx, &e, p, rawID, source, receivedAt, ev, msg); err != nil {
		return 0, err
	}
	return e.flush(ctx, q)
}

// ApplyAll records a payload's events in one transaction: the messages
// they name in id order (the order every writer locks messages in), then
// what they change beyond them (counters, provider stats, the suppression
// list), each in key order.
func (n *Normalizer) ApplyAll(ctx context.Context, tx pgx.Tx, p store.Provider, rawID *uuid.UUID, source string, receivedAt time.Time, evs []providers.Event) (int, error) {
	items := make([]rawItem, 0, len(evs))
	for _, ev := range evs {
		var raw uuid.UUID
		if rawID != nil {
			raw = *rawID
		}
		items = append(items, rawItem{raw: raw, ev: ev})
	}
	return n.applyItems(ctx, tx, p, source, receivedAt, items)
}

// rawItem is an event with the stored payload it came from.
type rawItem struct {
	raw uuid.UUID
	ev  providers.Event
}

func (n *Normalizer) applyItems(ctx context.Context, tx pgx.Tx, p store.Provider, source string, receivedAt time.Time, items []rawItem) (int, error) {
	q := store.New(tx)
	type item struct {
		rawItem
		msg *store.IngestMessageByIDRow
	}
	list := make([]item, 0, len(items))
	for _, it := range items {
		msg, err := n.message(ctx, q, p, it.ev)
		if err != nil {
			return 0, err
		}
		list = append(list, item{it, msg})
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i].msg, list[j].msg
		if a == nil || b == nil {
			return a != nil && b == nil
		}
		return bytes.Compare(a.ID[:], b.ID[:]) < 0
	})
	var e effects
	for _, it := range list {
		var raw *uuid.UUID
		if it.raw != uuid.Nil {
			r := it.raw
			raw = &r
		}
		if err := n.apply(ctx, tx, &e, p, raw, source, receivedAt, it.ev, it.msg); err != nil {
			return 0, err
		}
	}
	return e.flush(ctx, q)
}

// effects are what a batch of events changes beyond its messages. They
// are applied once the batch's messages are locked, in key order: campaign
// counters, the providers' daily outcomes, then the suppression list and
// the contacts' status by address, then the contacts' opens and clicks.
type effects struct {
	ledger     counters.Ledger
	outcomes   map[outcomeKey]*store.ProvRecordOutcomeParams
	supp       []suppression
	engagement EngagementBatch
}

// opens is the batch's provider opens and clicks, started on the first.
func (e *effects) opens(rec Engagement) EngagementBatch {
	if e.engagement == nil {
		e.engagement = rec.Begin(&e.ledger)
	}
	return e.engagement
}

type outcomeKey struct {
	provider uuid.UUID
	day      string
}

type suppression struct {
	add    store.IngestAddSuppressionParams
	status string
}

func (e *effects) outcome(provider uuid.UUID, at time.Time, delivered, bounced, complained int64) {
	if delivered+bounced+complained == 0 {
		return
	}
	if e.outcomes == nil {
		e.outcomes = map[outcomeKey]*store.ProvRecordOutcomeParams{}
	}
	day := at.UTC()
	k := outcomeKey{provider, day.Format(time.DateOnly)}
	o := e.outcomes[k]
	if o == nil {
		o = &store.ProvRecordOutcomeParams{ProviderID: provider, Day: day}
		e.outcomes[k] = o
	}
	o.Delivered += delivered
	o.Bounced += bounced
	o.Complained += complained
}

// flush applies the batch's effects and returns how many addresses were
// newly suppressed.
func (e *effects) flush(ctx context.Context, q *store.Queries) (int, error) {
	if err := e.ledger.Flush(ctx, q); err != nil {
		return 0, err
	}
	keys := make([]outcomeKey, 0, len(e.outcomes))
	for k := range e.outcomes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if c := bytes.Compare(keys[i].provider[:], keys[j].provider[:]); c != 0 {
			return c < 0
		}
		return keys[i].day < keys[j].day
	})
	for _, k := range keys {
		if err := q.ProvRecordOutcome(ctx, *e.outcomes[k]); err != nil {
			return 0, err
		}
	}
	sort.SliceStable(e.supp, func(i, j int) bool {
		a, b := e.supp[i].add, e.supp[j].add
		if a.WorkspaceID != b.WorkspaceID {
			return bytes.Compare(a.WorkspaceID[:], b.WorkspaceID[:]) < 0
		}
		return a.Email < b.Email
	})
	added := 0
	for _, sp := range e.supp {
		if _, err := q.IngestAddSuppression(ctx, sp.add); errors.Is(err, pgx.ErrNoRows) {
			continue // already suppressed
		} else if err != nil {
			return 0, err
		}
		added++
		if sp.status != "" {
			if err := q.IngestSetContactStatus(ctx, store.IngestSetContactStatusParams{WorkspaceID: sp.add.WorkspaceID, Email: sp.add.Email, Status: sp.status}); err != nil {
				return 0, err
			}
		}
	}
	if e.engagement != nil {
		if err := e.engagement.Flush(ctx, q); err != nil {
			return 0, err
		}
	}
	e.outcomes, e.supp, e.engagement = nil, nil, nil
	return added, nil
}

// message finds the message an event is about: by the id the provider
// echoes, else by the provider's message id.
func (n *Normalizer) message(ctx context.Context, q *store.Queries, p store.Provider, ev providers.Event) (*store.IngestMessageByIDRow, error) {
	if ev.MessageID != nil {
		if m, err := q.IngestMessageByID(ctx, store.IngestMessageByIDParams{ID: *ev.MessageID, WorkspaceID: p.WorkspaceID}); err == nil {
			return &m, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if ev.ProviderMessageID != "" {
		pid := ev.ProviderMessageID
		if m, err := q.IngestMessageByProviderID(ctx, store.IngestMessageByProviderIDParams{ProviderID: &p.ID, ProviderMessageID: &pid}); err == nil {
			mm := store.IngestMessageByIDRow(m)
			return &mm, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	return nil, nil
}

func (n *Normalizer) apply(ctx context.Context, tx pgx.Tx, e *effects, p store.Provider, rawID *uuid.UUID, source string, receivedAt time.Time, ev providers.Event,
	msg *store.IngestMessageByIDRow) error {
	q := store.New(tx)
	// Provider-side opens and clicks only count when the workspace chose to
	// keep the provider's tracking; otherwise ReputeCast's own tracking
	// already counted them.
	engagement := ev.Type == providers.EventOpen || ev.Type == providers.EventClick
	if engagement && state.TrackingOf(p).Mode != "provider" {
		return nil
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
	if msg != nil {
		// The message first, before its event refers to it (lock order).
		if err := q.CountersLockMessage(ctx, msg.ID); err != nil {
			return err
		}
	}
	dedupe := dedupeKey(p.ID, ev, recipient, occurred)
	id, err := q.IngestInsertEvent(ctx, store.IngestInsertEventParams{
		ID: newID(), WorkspaceID: p.WorkspaceID, MessageID: messageID, CampaignID: campaignID, ContactID: contactID, ProviderID: &p.ID, RawEventID: rawID,
		Type: string(ev.Type), OccurredAt: occurred, MailboxProvider: mailbox, OriginalType: clip(ev.OriginalType, 200), OriginalCode: clip(ev.OriginalCode, 100),
		Diagnostic: clip(ev.Diagnostic, 1000), BounceClass: ev.BounceClass, Url: clip(ev.URL, 2048), UserAgent: clip(ev.UserAgent, 500),
		Recipient: recipient, ProviderMessageID: clip(ev.ProviderMessageID, 300), Source: label, ReceivedAt: receivedAt, DedupeKey: dedupe,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // a redelivery of an event we already have
	}
	if err != nil {
		return err
	}
	if engagement {
		// The tracking service classifies and counts it like one of its
		// own; it tells transactional mail's webhooks itself, once.
		if msg == nil || !trusted || msg.Kind == "placement" {
			return nil
		}
		if n.Engagement == nil {
			return n.hooks(ctx, tx, p, msg, ev)
		}
		return e.opens(n.Engagement).Record(ctx, tx, ProviderEngagement{EventID: id, MessageID: msg.ID, Click: ev.Type == providers.EventClick,
			URL: ev.URL, UserAgent: ev.UserAgent, IP: ev.IP, At: occurred})
	}
	// The message moves on (delivered, deferred, bounced…); its campaign's
	// counters and its provider's outcomes follow the move, so a redelivered
	// event or a soft bounce followed by a delivery never counts twice.
	var outcome counters.Delta
	switch ev.Type {
	case providers.EventDelivered:
		outcome.Delivered = 1
	case providers.EventHardBounce:
		outcome.HardBounced = 1
	case providers.EventComplaint:
		outcome.Complained = 1
	}
	if msg != nil && moves[ev.Type] {
		errText := ""
		if ev.Type != providers.EventDelivered && ev.Type != providers.EventUnsubscribe {
			errText = strings.TrimSpace(ev.OriginalCode + " " + ev.Diagnostic)
		}
		mv, err := q.CountersMoveDelivery(ctx, store.CountersMoveDeliveryParams{ID: msg.ID, Kind: string(ev.Type), At: occurred, Error: clip(errText, 500)})
		if err != nil {
			return err
		}
		moved := counters.FromDelivery(mv)
		e.ledger.Add(moved)
		// The provider's daily outcomes count what this event added: a
		// message delivered twice is one delivery.
		d := counters.Diff(moved.Before, moved.After)
		outcome = counters.Delta{Delivered: max(d.Delivered, 0), HardBounced: max(d.HardBounced, 0), Complained: max(d.Complained, 0)}
	}
	if msg != nil {
		if err := n.hooks(ctx, tx, p, msg, ev); err != nil {
			return err
		}
	}
	if trusted {
		e.outcome(p.ID, occurred, outcome.Delivered, outcome.HardBounced, outcome.Complained)
	}
	// A placement test copy went to a seed mailbox the workspace owns: its
	// bounces and complaints never suppress the seed (internal/placement).
	if msg != nil && msg.Kind == "placement" {
		return nil
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
		return nil
	}
	src := "webhook"
	if source == "bounce_mailbox" {
		src = "bounce_mailbox"
	}
	e.supp = append(e.supp, suppression{status: contactStatus, add: store.IngestAddSuppressionParams{
		ID: newID(), WorkspaceID: p.WorkspaceID, Email: recipient, Reason: reason, Source: src, SourceProviderID: &p.ID,
		OriginalCode: clip(ev.OriginalCode, 100), Detail: clip(strings.TrimSpace(ev.OriginalType+" · "+ev.Diagnostic), 500), CreatedAt: occurred,
	}})
	return nil
}

// hooks tells the message hooks about an event.
func (n *Normalizer) hooks(ctx context.Context, tx pgx.Tx, p store.Provider, msg *store.IngestMessageByIDRow, ev providers.Event) error {
	for _, h := range n.Hooks {
		if err := h(ctx, tx, p, msg.ID, msg.Stream, ev); err != nil {
			return err
		}
	}
	return nil
}

// moves are the event types that move a message's state.
var moves = map[providers.EventType]bool{
	providers.EventDelivered: true, providers.EventDeferred: true, providers.EventHardBounce: true, providers.EventSoftBounce: true,
	providers.EventComplaint: true, providers.EventRejected: true, providers.EventUnsubscribe: true,
}

// dedupeKey identifies an event so provider redeliveries are dropped; a
// click's link is part of it (two links clicked in the same millisecond
// are two clicks).
func dedupeKey(provider uuid.UUID, ev providers.Event, recipient string, occurred time.Time) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%s|%d|%s|%s", provider, ev.Type, ev.ProviderMessageID, recipient, occurred.UnixMilli(), ev.OriginalCode, ev.OriginalType)
	if ev.Type == providers.EventClick {
		_, _ = fmt.Fprintf(h, "|%s", ev.URL)
	}
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
