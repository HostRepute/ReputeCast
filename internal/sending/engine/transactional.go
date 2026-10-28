package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// TransactionalRetryWindow is how long a provider retries a transactional
// message after temporary errors before it goes to the route's fallback.
const TransactionalRetryWindow = 10 * time.Minute

// TransactionalFailure is a message every provider of its route failed.
type TransactionalFailure struct {
	ID     uuid.UUID
	Reason string
}

// TransactionalFailover is a message moved to another provider of its route.
type TransactionalFailover struct {
	ID       uuid.UUID
	From, To uuid.UUID
}

// TransactionalOutcome is what one pass of the transactional outbox did for
// one workspace: the transactional domain emits its webhook events from it.
type TransactionalOutcome struct {
	WorkspaceID uuid.UUID
	Sent        []uuid.UUID
	Failed      []TransactionalFailure
	FailedOver  []TransactionalFailover
	At          time.Time
}

// TransactionalHooks is told about each outcome inside the transaction
// that records it.
type TransactionalHooks interface {
	TransactionalOutcome(ctx context.Context, tx pgx.Tx, o TransactionalOutcome) error
}

// TransactionalOutbox delivers transactional messages: per provider, within
// the provider's per-second rate shared with campaigns and automations
// (ahead of Broadcasts, see transactionalHeadroom), with retries and back-off
// for up to ten minutes, then the route's fallback; a provider marked down
// sends its queue to the fallback at once.
type TransactionalOutbox struct {
	E *Engine
	// Open decodes a message's sealed payload (the transactional domain).
	Open  func(ctx context.Context, id uuid.UUID, sealed []byte) (providers.Message, providers.MessageExtras, error)
	Hooks TransactionalHooks
	// DevSMTP, in development, delivers the demo providers' transactional
	// mail to the mail catcher (Mailpit) and records it as delivered.
	DevSMTP string
}

// transactionalHeadroom is how many of a provider's sends this second
// belong to transactional mail before campaigns and automations take any.
func (e *Engine) transactionalHeadroom(ctx context.Context, provider uuid.UUID, now time.Time) int {
	n, err := e.Q.TxnOutboxHeadroom(ctx, store.TxnOutboxHeadroomParams{ProviderID: provider, Now: now})
	if err != nil || n <= 0 {
		return 0
	}
	return int(min(n, math.MaxInt32))
}

// reserveAfterTransactional is reserve for Broadcasts and Automations: it
// leaves the second's transactional mail its share first.
func (e *Engine) reserveAfterTransactional(ctx context.Context, provider uuid.UUID, sec int64, want, capN int, now time.Time) (int, error) {
	if h := e.transactionalHeadroom(ctx, provider, now); h > 0 {
		capN -= h
		if capN <= 0 {
			return 0, nil
		}
	}
	return e.reserve(ctx, provider, sec, want, capN)
}

// txRoute is one provider during a pass.
type txRoute struct {
	provider  store.Provider
	transport providers.Provider
	sim       *simulator.Provider
	dev       bool // delivered to the development mail catcher
	openErr   error
	rate      providerRate
	batch     int
	cooldown  time.Time
}

// TransactionalOutboxArgs wakes the transactional outbox.
type TransactionalOutboxArgs struct{}

// Kind implements river.JobArgs.
func (TransactionalOutboxArgs) Kind() string { return "sending.transactional" }

// InsertOpts implements river.JobArgsWithInsertOpts: one pass waits or runs
// at a time; it keeps running while mail is due.
func (TransactionalOutboxArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSending, UniqueOpts: river.UniqueOpts{ByState: activeStates}}
}

// TransactionalOutboxWorker runs the transactional outbox.
type TransactionalOutboxWorker struct {
	river.WorkerDefaults[TransactionalOutboxArgs]
	O *TransactionalOutbox
}

// Timeout bounds one run.
func (w *TransactionalOutboxWorker) Timeout(*river.Job[TransactionalOutboxArgs]) time.Duration {
	return 2 * time.Minute
}

// Work implements river.Worker.
func (w *TransactionalOutboxWorker) Work(ctx context.Context, _ *river.Job[TransactionalOutboxArgs]) error {
	again, done, err := w.O.Run(ctx, runBudget)
	if err != nil || done {
		return interrupted(ctx, err)
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
}

// NextRetry implements river.Worker.
func (w *TransactionalOutboxWorker) NextRetry(job *river.Job[TransactionalOutboxArgs]) time.Time {
	return quickRetry(job.Attempt)
}

// TransactionalWatchArgs is the safety net: it wakes the outbox for
// scheduled sends and retries that came due, and fails over queues of
// providers marked down.
type TransactionalWatchArgs struct{}

// Kind implements river.JobArgs.
func (TransactionalWatchArgs) Kind() string { return "sending.transactional_watch" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TransactionalWatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Second}}
}

// TransactionalWatchWorker runs TransactionalWatchArgs.
type TransactionalWatchWorker struct {
	river.WorkerDefaults[TransactionalWatchArgs]
	O *TransactionalOutbox
}

// Work implements river.Worker.
func (w *TransactionalWatchWorker) Work(ctx context.Context, _ *river.Job[TransactionalWatchArgs]) error {
	now := w.O.E.now()
	if err := w.O.reroute(ctx, now); err != nil {
		return err
	}
	if ids, err := w.O.E.Q.TxnOutboxProviders(ctx, now); err == nil && len(ids) > 0 {
		w.O.E.insert(ctx, TransactionalOutboxArgs{}, nil)
	}
	return nil
}

// RegisterTransactional adds the transactional outbox's workers.
func RegisterTransactional(reg *jobs.Registry, o *TransactionalOutbox) {
	river.AddWorker(reg.Workers, &TransactionalOutboxWorker{O: o})
	river.AddWorker(reg.Workers, &TransactionalWatchWorker{O: o})
	reg.Every(15*time.Second, func() (river.JobArgs, *river.InsertOpts) { return TransactionalWatchArgs{}, nil }, true)
}

// Wake queues a pass of the outbox in tx (the API's transaction).
func (o *TransactionalOutbox) Wake(ctx context.Context, tx pgx.Tx) error {
	return o.E.insertTx(ctx, tx, TransactionalOutboxArgs{}, nil)
}

// Run sends due transactional mail for up to budget.
func (o *TransactionalOutbox) Run(ctx context.Context, budget time.Duration) (time.Duration, bool, error) {
	start := o.E.now()
	routes := map[uuid.UUID]*txRoute{}
	for {
		now := o.E.now()
		if now.Sub(start) >= budget {
			return 0, false, nil
		}
		n, pending, err := o.pass(ctx, routes, now)
		if err != nil {
			return 0, false, err
		}
		if !pending {
			return 0, true, nil
		}
		wait := time.Until(now.Truncate(time.Second).Add(time.Second))
		if n == 0 {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Step sends one second's transactional mail (tests and fast-forward).
func (o *TransactionalOutbox) Step(ctx context.Context, now time.Time) (int, error) {
	n, _, err := o.pass(ctx, map[uuid.UUID]*txRoute{}, now)
	return n, err
}

func (o *TransactionalOutbox) pass(ctx context.Context, routes map[uuid.UUID]*txRoute, now time.Time) (int, bool, error) {
	if err := o.reroute(ctx, now); err != nil {
		return 0, false, err
	}
	ids, err := o.E.Q.TxnOutboxProviders(ctx, now)
	if err != nil {
		return 0, false, err
	}
	if len(ids) == 0 {
		return 0, false, nil
	}
	sent := 0
	for _, pid := range ids {
		n, err := o.tick(ctx, pid, routes, now)
		if err != nil {
			return sent, true, err
		}
		sent += n
	}
	return sent, true, nil
}

// attempt is one provider call for a message, kept in its log.
type attempt struct {
	ProviderID string    `json:"provider_id"`
	Provider   string    `json:"provider"`
	At         time.Time `json:"at"`
	OK         bool      `json:"ok"`
	Error      string    `json:"error,omitempty"`
	Temporary  bool      `json:"temporary,omitempty"`
	// GaveUp marks the attempt after which the message left this provider.
	GaveUp bool `json:"gave_up,omitempty"`
	// Down is set when the message left a provider marked down.
	Down bool   `json:"down,omitempty"`
	PMID string `json:"provider_message_id,omitempty"`
}

// txRouteSnap is the route decision stored with a message.
type txRouteSnap struct {
	Primary  uuid.UUID  `json:"primary"`
	Fallback *uuid.UUID `json:"fallback,omitempty"`
	Policy   string     `json:"policy"`
}

// next is the provider a message goes to after leaving current: the other
// provider of its route, when it hasn't given up on it yet and it can send.
func (o *TransactionalOutbox) next(ctx context.Context, q *store.Queries, current uuid.UUID, routeRaw, triedRaw json.RawMessage) (*store.Provider, error) {
	var rs txRouteSnap
	_ = json.Unmarshal(routeRaw, &rs)
	if rs.Policy == string(routing.PauseAndAsk) {
		return nil, nil
	}
	var tried []attempt
	_ = json.Unmarshal(triedRaw, &tried)
	gaveUp := map[string]bool{current.String(): true}
	for _, a := range tried {
		if a.GaveUp {
			gaveUp[a.ProviderID] = true
		}
	}
	cands := []uuid.UUID{rs.Primary}
	if rs.Fallback != nil {
		cands = append(cands, *rs.Fallback)
	}
	for i, id := range cands {
		if id == uuid.Nil || gaveUp[id.String()] {
			continue
		}
		p, err := q.ProvGetAny(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if p.DisconnectedAt != nil || !state.Available(p) {
			continue
		}
		// Moving to the fallback is vetted as every move is (review GS-5):
		// a sandboxed account or a used-up quota can't take the message.
		// A warming provider's cap is the outbox's to apply when it sends
		// (warmTransactional): product mail never waits for a warm-up.
		if i > 0 && !state.Vet(p, state.Load{Messages: 1}).Allowed() {
			continue
		}
		return &p, nil
	}
	return nil, nil
}

// reroute moves the queues of providers marked down (or removed) to the
// next provider of each message's route; messages with nowhere to go wait
// out the retry window, then fail.
func (o *TransactionalOutbox) reroute(ctx context.Context, now time.Time) error {
	ids, err := o.E.Q.TxnOutboxProviders(ctx, now.Add(TransactionalRetryWindow))
	if err != nil {
		return err
	}
	for _, pid := range ids {
		p, perr := o.E.Q.ProvGetAny(ctx, pid)
		if perr != nil && !errors.Is(perr, pgx.ErrNoRows) {
			return perr
		}
		if perr == nil && p.DisconnectedAt == nil && state.Available(p) {
			continue
		}
		name := "The provider"
		if perr == nil {
			name = p.Name
		}
		rows, err := o.E.Q.TxnOutboxQueuedOn(ctx, store.TxnOutboxQueuedOnParams{ProviderID: &pid, Stale: now.Add(-staleClaim)})
		if err != nil {
			return err
		}
		outcomes := map[uuid.UUID]*TransactionalOutcome{}
		err = sending.Tx(ctx, o.E.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, r := range rows {
				oc := outcomes[r.WorkspaceID]
				if oc == nil {
					oc = &TransactionalOutcome{WorkspaceID: r.WorkspaceID, At: now}
					outcomes[r.WorkspaceID] = oc
				}
				reason := routing.DeliveryError("down", i18n.Args{"provider": name})
				nx, err := o.next(ctx, q, pid, r.Route, r.Tried)
				if err != nil {
					return err
				}
				gave := attempt{ProviderID: pid.String(), Provider: name, At: now, Error: reason, GaveUp: true, Down: true}
				if nx != nil {
					if err := o.move(ctx, q, r.ID, pid, nx.ID, gave, now); err != nil {
						return err
					}
					oc.FailedOver = append(oc.FailedOver, TransactionalFailover{ID: r.ID, From: pid, To: nx.ID})
					continue
				}
				if r.ProviderSince != nil && now.Sub(*r.ProviderSince) < TransactionalRetryWindow {
					continue // waits for the provider to come back
				}
				if err := o.fail(ctx, q, r.ID, gave, routing.DeliveryError("downNoOther", i18n.Args{"provider": name})); err != nil {
					return err
				}
				oc.Failed = append(oc.Failed, TransactionalFailure{ID: r.ID, Reason: reason})
			}
			return o.hooks(ctx, tx, outcomes)
		})
		if err != nil {
			return err
		}
	}
	orphans, err := o.E.Q.TxnOutboxOrphans(ctx)
	if err != nil || len(orphans) == 0 {
		return err
	}
	outcomes := map[uuid.UUID]*TransactionalOutcome{}
	return sending.Tx(ctx, o.E.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, r := range orphans {
			oc := outcomes[r.WorkspaceID]
			if oc == nil {
				oc = &TransactionalOutcome{WorkspaceID: r.WorkspaceID, At: now}
				outcomes[r.WorkspaceID] = oc
			}
			nx, err := o.next(ctx, q, uuid.Nil, r.Route, r.Tried)
			if err != nil {
				return err
			}
			if nx != nil {
				if err := o.move(ctx, q, r.ID, uuid.Nil, nx.ID, attempt{}, now); err != nil {
					return err
				}
				continue
			}
			if err := o.fail(ctx, q, r.ID, attempt{}, routing.DeliveryError("providersRemoved", nil)); err != nil {
				return err
			}
			oc.Failed = append(oc.Failed, TransactionalFailure{ID: r.ID, Reason: "the route's providers were removed"})
		}
		return o.hooks(ctx, tx, outcomes)
	})
}

func (o *TransactionalOutbox) hooks(ctx context.Context, tx pgx.Tx, outcomes map[uuid.UUID]*TransactionalOutcome) error {
	if o.Hooks == nil {
		return nil
	}
	for _, oc := range outcomes {
		if len(oc.Sent)+len(oc.Failed)+len(oc.FailedOver) == 0 {
			continue
		}
		if err := o.Hooks.TransactionalOutcome(ctx, tx, *oc); err != nil {
			return err
		}
	}
	return nil
}

func (o *TransactionalOutbox) move(ctx context.Context, q *store.Queries, id, from, to uuid.UUID, gave attempt, now time.Time) error {
	if gave.ProviderID != "" {
		b, _ := json.Marshal([]attempt{gave})
		if err := q.TxnAppendAttempts(ctx, store.TxnAppendAttemptsParams{Ids: []uuid.UUID{id}, Attempts: []json.RawMessage{b}}); err != nil {
			return err
		}
	}
	if err := q.TxnMoveToProvider(ctx, store.TxnMoveToProviderParams{ID: id, ProviderID: &to}); err != nil {
		return err
	}
	var fromPtr *uuid.UUID
	if from != uuid.Nil {
		fromPtr = &from
	}
	return q.TxnMarkFailedOver(ctx, store.TxnMarkFailedOverParams{MessageID: id, At: now, FromProvider: fromPtr})
}

func (o *TransactionalOutbox) fail(ctx context.Context, q *store.Queries, id uuid.UUID, gave attempt, reason string) error {
	if gave.ProviderID != "" {
		b, _ := json.Marshal([]attempt{gave})
		if err := q.TxnAppendAttempts(ctx, store.TxnAppendAttemptsParams{Ids: []uuid.UUID{id}, Attempts: []json.RawMessage{b}}); err != nil {
			return err
		}
	}
	if err := q.TxnMarkFailed(ctx, store.TxnMarkFailedParams{ID: id, Error: clip(reason, 500)}); err != nil {
		return err
	}
	return q.TxnOutboxDone(ctx, []uuid.UUID{id})
}

// devTransport is the development mail catcher standing in for a demo
// provider (the System stream does the same, internal/systemmail).
func (o *TransactionalOutbox) devTransport(p store.Provider) (providers.Provider, error) {
	desc, ok := providers.Lookup("smtp")
	if !ok || desc.New == nil {
		return nil, errors.New("the SMTP adapter isn't in this build, so demo mail can't reach the mail catcher")
	}
	host, port, err := net.SplitHostPort(o.DevSMTP)
	if err != nil {
		return nil, fmt.Errorf("mail.dev_smtp %q: %w", o.DevSMTP, err)
	}
	n, _ := strconv.Atoi(port)
	settings, _ := json.Marshal(map[string]any{"host": host, "port": n, "tls": "none", "auth": "none", "helo": "reputecast.local"})
	return desc.New(providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: settings, Credentials: json.RawMessage(`{}`)})
}

func (o *TransactionalOutbox) route(ctx context.Context, pid uuid.UUID, routes map[uuid.UUID]*txRoute, now time.Time) (*txRoute, error) {
	if r := routes[pid]; r != nil {
		return r, nil
	}
	p, err := o.E.Q.ProvGetAny(ctx, pid)
	if err != nil {
		return nil, err
	}
	r := &txRoute{provider: p, batch: batchSize(p)}
	if manage.IsDemo(p) && o.DevSMTP != "" {
		r.transport, r.openErr = o.devTransport(p)
		r.dev = true
	} else {
		r.transport, _, r.openErr = o.E.transport(ctx, p, "", time.Time{})
	}
	if r.openErr != nil {
		o.E.log().WarnContext(ctx, "open a provider for transactional mail", "provider", p.ID, "err", r.openErr)
	}
	if s, ok := r.transport.(*simulator.Provider); ok {
		r.sim = s
	}
	r.rate = o.E.rateOf(ctx, o.E.Q, p, time.UTC, now)
	routes[pid] = r
	return r, nil
}

func (o *TransactionalOutbox) tick(ctx context.Context, pid uuid.UUID, routes map[uuid.UUID]*txRoute, now time.Time) (int, error) {
	r, err := o.route(ctx, pid, routes, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil // removed: reroute moves its mail
	}
	if err != nil {
		return 0, err
	}
	if !state.Available(r.provider) || r.transport == nil || r.cooldown.After(now) {
		return 0, nil
	}
	// Reserve what is due, up to the second's rate: the rest of the
	// window stays for campaigns and automations.
	due, err := o.E.Q.TxnOutboxDue(ctx, store.TxnOutboxDueParams{ProviderID: &pid, Now: now})
	if err != nil || due == 0 {
		return 0, err
	}
	want := int(min(due, int64(max(int(math.Floor(r.rate.PerSecond)), 1))))
	granted, err := o.E.reserve(ctx, pid, now.Unix(), want, int(math.Ceil(r.rate.PerSecond)))
	if err != nil || granted <= 0 {
		return 0, err
	}
	rows, err := o.E.Q.TxnOutboxClaim(ctx, store.TxnOutboxClaimParams{ProviderID: &pid, Stale: now.Add(-staleClaim), Now: now, Lim: int32(granted)})
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	// A warming provider sends within today's cap; over it a warm provider
	// of the route takes the message, else it still goes out, counted as
	// over the cap: transactional mail never waits (review R3.L6b).
	claimed := rows
	rows, warm, err := o.warmTransactional(ctx, r.provider, rows, now)
	if err != nil {
		ids := make([]uuid.UUID, 0, len(claimed))
		for _, row := range claimed {
			ids = append(ids, row.ID)
		}
		_ = o.E.Q.SendingRelease(context.WithoutCancel(ctx), ids)
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	type pending struct {
		row store.TxnOutboxClaimRow
		msg providers.Message
		x   providers.MessageExtras
	}
	var batch []pending
	var release []uuid.UUID
	results := map[uuid.UUID]*txResult{}
	for _, row := range rows {
		if len(row.Sealed) == 0 {
			// Never expected: the payload is written with the message.
			results[row.ID] = &txResult{row: row, permanent: true, final: true, err: routing.DeliveryError("contentGone", nil)}
			continue
		}
		m, x, err := o.Open(ctx, row.ID, row.Sealed)
		if err != nil {
			results[row.ID] = &txResult{row: row, permanent: true, final: true, err: routing.DeliveryError("unreadable", i18n.Args{"error": err.Error()})}
			continue
		}
		if r.sim != nil {
			if m.Tags == nil {
				m.Tags = map[string]string{}
			}
			m.Tags[simulator.TagMailbox] = row.MailboxProvider
			m.Tags[simulator.TagTier] = "active"
			m.Tags[simulator.TagEngaged] = "true"
		}
		batch = append(batch, pending{row, m, x})
	}
	key := routeKey{uuid.Nil, pid}
	var sims [][]any
	for len(batch) > 0 {
		n := min(len(batch), max(r.batch, 1))
		chunk := batch[:n]
		batch = batch[n:]
		msgs := make([]providers.Message, len(chunk))
		xs := make([]providers.MessageExtras, len(chunk))
		for i, p := range chunk {
			msgs[i], xs[i] = p.msg, p.x
		}
		cctx, cancel := o.E.callContext(ctx, now)
		res, callErr := providers.SendWithExtras(cctx, r.transport, msgs, xs)
		cancel()
		if callErr == nil && len(res) != len(msgs) {
			callErr = &providers.CallError{Op: "Send", Message: "the provider didn't answer for every message", Temporary: true}
		}
		o.E.recordCall(key, now, callErr)
		if err := manage.RecordSend(context.WithoutCancel(ctx), o.E.Pool, r.provider, res, callErr, now); err != nil {
			o.E.log().WarnContext(ctx, "record provider call", "provider", pid, "err", err)
		}
		if callErr != nil {
			temp := true
			var ce *providers.CallError
			if errors.As(callErr, &ce) && !ce.Temporary && !errors.Is(callErr, providers.ErrUnavailable) {
				temp = false
			}
			for _, p := range chunk {
				results[p.row.ID] = &txResult{row: p.row, err: callText(callErr), permanent: !temp}
			}
			for _, p := range batch {
				release = append(release, p.row.ID)
			}
			batch = nil
			r.cooldown = now.Add(backoffMin)
			break
		}
		for i, rr := range res {
			p := chunk[i]
			if rr.Err == nil {
				results[p.row.ID] = &txResult{row: p.row, ok: true, pmid: rr.ProviderMessageID}
				if r.transport != nil && r.transport.Capabilities().InfersDelivery {
					// Plain SMTP reports no deliveries: the server's 250 is it
					// (review GS-7).
					sims = append(sims, inferredRow(p.row.WorkspaceID, nil, p.row.ID, pid, now))
				}
				if r.sim != nil {
					for _, ev := range r.sim.Plan(p.msg) {
						if (ev.Kind == "open" && !p.row.TrackOpens) || (ev.Kind == "click" && !p.row.TrackClicks) {
							continue
						}
						sims = append(sims, simRows(p.row.WorkspaceID, nil, p.row.ID, pid, now, []simulator.Event{ev})...)
					}
				}
				continue
			}
			results[p.row.ID] = &txResult{row: p.row, err: rr.Err.Error(), permanent: !rr.Err.Temporary}
		}
	}
	sent := 0
	outcomes := map[uuid.UUID]*TransactionalOutcome{}
	// Recorded even when the job is told to stop: the providers have these
	// messages now (review GS-1).
	ctx = context.WithoutCancel(ctx)
	err = sending.Tx(ctx, o.E.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for k := range outcomes {
			delete(outcomes, k)
		}
		sent = 0
		var sentIDs []uuid.UUID
		var pmids []string
		var attemptIDs []uuid.UUID
		var attempts []json.RawMessage
		var retryIDs []uuid.UUID
		var retryAt []time.Time
		var retryErr []string
		for _, row := range rows {
			res := results[row.ID]
			if res == nil {
				continue
			}
			oc := outcomes[row.WorkspaceID]
			if oc == nil {
				oc = &TransactionalOutcome{WorkspaceID: row.WorkspaceID, At: now}
				outcomes[row.WorkspaceID] = oc
			}
			a := attempt{ProviderID: pid.String(), Provider: r.provider.Name, At: now, OK: res.ok, Error: clip(res.err, 300), Temporary: !res.ok && !res.permanent, PMID: res.pmid}
			if res.ok {
				sentIDs = append(sentIDs, row.ID)
				pmids = append(pmids, res.pmid)
				b, _ := json.Marshal([]attempt{a})
				attemptIDs, attempts = append(attemptIDs, row.ID), append(attempts, b)
				oc.Sent = append(oc.Sent, row.ID)
				continue
			}
			since := row.AcceptedAt
			if row.ProviderSince != nil {
				since = *row.ProviderSince
			}
			if !res.permanent && now.Sub(since) < TransactionalRetryWindow {
				b, _ := json.Marshal([]attempt{a})
				attemptIDs, attempts = append(attemptIDs, row.ID), append(attempts, b)
				retryIDs = append(retryIDs, row.ID)
				retryAt = append(retryAt, now.Add(backoff(int(row.Attempts))))
				retryErr = append(retryErr, clip(res.err, 500))
				continue
			}
			a.GaveUp = true
			var nx *store.Provider
			if !res.final {
				var err error
				if nx, err = o.next(ctx, q, pid, row.Route, row.Tried); err != nil {
					return err
				}
			}
			if nx != nil {
				if err := o.move(ctx, q, row.ID, pid, nx.ID, a, now); err != nil {
					return err
				}
				oc.FailedOver = append(oc.FailedOver, TransactionalFailover{ID: row.ID, From: pid, To: nx.ID})
				continue
			}
			reason := routing.DeliveryError("allFailed", i18n.Args{"error": res.err})
			if res.final {
				reason = res.err
			}
			if err := o.fail(ctx, q, row.ID, a, reason); err != nil {
				return err
			}
			oc.Failed = append(oc.Failed, TransactionalFailure{ID: row.ID, Reason: res.err})
		}
		if len(attemptIDs) > 0 {
			if err := q.TxnAppendAttempts(ctx, store.TxnAppendAttemptsParams{Ids: attemptIDs, Attempts: attempts}); err != nil {
				return err
			}
		}
		if len(sentIDs) > 0 {
			if err := q.SendingMarkSent(ctx, store.SendingMarkSentParams{At: &now, Ids: sentIDs, ProviderMessageIds: pmids}); err != nil {
				return err
			}
			if err := q.TxnOutboxDone(ctx, sentIDs); err != nil {
				return err
			}
			sent = len(sentIDs)
			// Sent over a warming provider's cap: the warm-up page and the
			// provider card say how many (R3.L6b).
			if n := warm.overSent(sentIDs); n > 0 {
				if err := q.WarmupAddOverCap(ctx, store.WarmupAddOverCapParams{ID: warm.w.a.PlanID, Day: warm.w.today(), N: n}); err != nil {
					return err
				}
			}
		}
		if len(retryIDs) > 0 {
			if err := q.SendingRetryLater(ctx, store.SendingRetryLaterParams{Ids: retryIDs, Ats: retryAt, Errors: retryErr}); err != nil {
				return err
			}
		}
		if len(release) > 0 {
			if err := q.SendingRelease(ctx, release); err != nil {
				return err
			}
		}
		if len(sims) > 0 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sending_sim_events"}, []string{"workspace_id", "campaign_id", "message_id", "provider_id", "due_at", "kind", "data"},
				pgx.CopyFromRows(sims)); err != nil {
				return err
			}
		}
		// The development mail catcher has no webhooks: its 250 reply is the
		// delivery, recorded through the same normalization as provider events.
		if r.dev && o.E.Normalizer != nil {
			for i, id := range sentIDs {
				mid := id
				row := rowByID(rows, id)
				ev := providers.Event{Type: providers.EventDelivered, ProviderMessageID: pmids[i], MessageID: &mid, Recipient: row.Email, OccurredAt: now,
					OriginalType: "Delivered to Mailpit (development)", OriginalCode: "250", Diagnostic: "2.0.0 Ok: queued · smtpResponse from " + o.DevSMTP}
				if _, err := o.E.Normalizer.Apply(ctx, tx, r.provider, nil, "api", now, ev); err != nil {
					return err
				}
			}
		}
		return o.hooks(ctx, tx, outcomes)
	})
	if err == nil && len(sims) > 0 {
		o.E.insert(ctx, SimArgs{}, nil)
	}
	return sent, err
}

type txResult struct {
	row       store.TxnOutboxClaimRow
	ok        bool
	pmid      string
	err       string
	permanent bool
	// final: no other provider could do better (the payload can't be read).
	final bool
}

func rowByID(rows []store.TxnOutboxClaimRow, id uuid.UUID) store.TxnOutboxClaimRow {
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	return store.TxnOutboxClaimRow{}
}
