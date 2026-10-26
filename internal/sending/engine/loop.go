package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// errStopped means the campaign isn't sending any more (paused, cancelled,
// finished): the loop ends without rescheduling.
var errStopped = errors.New("send stopped")

// hold is a mailbox provider on hold (campaign_sends.holds).
type hold struct {
	Incident string    `json:"incident"`
	Since    time.Time `json:"since"`
	// Rearm is the complaint (or bounce) count when a person resumed it as
	// is: the threshold trips again at the next one.
	Rearm map[string]int64 `json:"rearm,omitempty"`
	// Resumed is true once a person resumed it (it stays here only to
	// remember Rearm).
	Resumed bool `json:"resumed,omitempty"`
}

func parseHolds(raw json.RawMessage) map[string]hold {
	out := map[string]hold{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// routeDetail is campaign_routes.status_detail for the engine's decisions.
type routeDetail struct {
	Decision string     `json:"decision,omitempty"` // retry, wait, move, resume
	Reason   string     `json:"reason,omitempty"`   // provider_down, outage, auth, identity, quota, throttled, api_errors, threshold
	Until    *time.Time `json:"until,omitempty"`
	Incident string     `json:"incident,omitempty"`
	To       string     `json:"to,omitempty"`
	// Retry (outages.go): test messages that went through in a row, when
	// the next one goes and whether they go to a seed inbox.
	Probes    int        `json:"probes,omitempty"`
	NextProbe *time.Time `json:"nextProbe,omitempty"`
	Seed      bool       `json:"seed,omitempty"`
}

func parseRouteDetail(s string) routeDetail {
	var d routeDetail
	_ = json.Unmarshal([]byte(s), &d)
	return d
}

// runRoute is one sending provider's share during a run.
type runRoute struct {
	provider  store.Provider
	planned   int64
	status    string
	detail    routeDetail
	transport providers.Provider
	sim       *simulator.Provider
	openErr   error
	rate      providerRate
	batch     int
	tracked   bool
	micros    int64 // cost per message
	carry     float64
	cooldown  time.Time
	lastProbe time.Time
	// slow is the share of its rate a throttled route sends at (0 = full),
	// calmFrom since when it has been answered normally (routeerrors.go).
	slow     float64
	calmFrom time.Time
	// apiCheckedAt is when the API error threshold was last checked;
	// judgedFrom is when the route last started sending again after a hold
	// (calls before it don't count against it).
	apiCheckedAt, judgedFrom time.Time
}

// active reports whether the route may send now.
func (r *runRoute) active(now time.Time) bool {
	if r.openErr != nil || r.transport == nil {
		return false
	}
	switch r.status {
	case "pending", "sending":
	default:
		return false
	}
	return state.Available(r.provider) || r.provider.Status == state.Degraded
}

type queueKey struct {
	provider uuid.UUID
	mailbox  string
}

type queueCell struct {
	queued, held, waiting int64
	// canary is what is held until the canary passes (also in held); doubt
	// is in doubt (doubt.go).
	canary, doubt int64
	nextAt        *time.Time
}

// runState is a campaign during one run of the send loop.
type runState struct {
	campaign   store.SendingCampaignContextRow
	send       store.CampaignSend
	content    sendContent
	templates  map[string]*Template
	routes     map[uuid.UUID]*runRoute
	order      []uuid.UUID
	holds      map[string]hold
	throttles  map[string]float64
	tokens     map[string]float64
	tokensAt   time.Time
	queue      map[queueKey]queueCell
	queueAt    time.Time
	loadedAt   time.Time
	guardedAt  time.Time
	thresholds Thresholds
	loc        *time.Location
	ab         abState
	started    time.Time
	// settledAt is when stale claims were last settled (doubt.go).
	settledAt time.Time
	// links are the per-recipient links of the email's polls and surveys.
	links []RecipientLink
}

// cachedTemplate keeps a send's parsed templates in the process.
type cachedTemplate struct {
	templates map[string]*Template
}

func (e *Engine) forgetTemplates(id uuid.UUID) {
	e.templMu.Lock()
	delete(e.template, id.String())
	e.templMu.Unlock()
}

func (e *Engine) templatesFor(id uuid.UUID, c sendContent) (map[string]*Template, error) {
	e.templMu.Lock()
	defer e.templMu.Unlock()
	if e.template == nil {
		e.template = map[string]*cachedTemplate{}
	}
	if t, ok := e.template[id.String()]; ok {
		return t.templates, nil
	}
	out := map[string]*Template{}
	keys := []string{""}
	for _, v := range c.Variants {
		keys = append(keys, v.Key)
	}
	for _, lang := range c.readyLanguages() {
		n := len(keys)
		for _, k := range keys[:n] {
			if _, v := splitKey(k); k == v {
				keys = append(keys, langKey(lang, k))
			}
		}
	}
	for _, k := range keys {
		t, err := Prepare(c.forVariant(k), c.Tracking)
		if err != nil {
			return nil, err
		}
		out[k] = t
	}
	e.template[id.String()] = &cachedTemplate{templates: out}
	return out, nil
}

// load reads everything a run needs. It returns errStopped when the
// campaign isn't sending.
func (e *Engine) load(ctx context.Context, id uuid.UUID, now time.Time) (*runState, error) {
	return e.loadState(ctx, id, now, false)
}

// loadState is load; paused also takes a paused send (a decision about one
// of its shares, an outage the health job found).
func (e *Engine) loadState(ctx context.Context, id uuid.UUID, now time.Time, paused bool) (*runState, error) {
	cc, err := e.Q.SendingCampaignContext(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errStopped
	}
	if err != nil {
		return nil, err
	}
	snd, err := e.Q.SendingGetSend(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errStopped
	}
	if err != nil {
		return nil, err
	}
	if (cc.Status != "sending" && (!paused || cc.Status != "paused")) || snd.Phase != "sending" {
		return nil, errStopped
	}
	rs := &runState{campaign: cc, send: snd, routes: map[uuid.UUID]*runRoute{}, tokens: map[string]float64{}, queue: map[queueKey]queueCell{},
		loc: state.Location(cc.WorkspaceTimezone), thresholds: ParseThresholds(snd.Thresholds, nil), ab: parseAB(snd.Ab), loadedAt: now}
	if cc.StartedAt != nil {
		rs.started = *cc.StartedAt
	}
	if err := json.Unmarshal(snd.Content, &rs.content); err != nil {
		return nil, fmt.Errorf("send content: %w", err)
	}
	if rs.content.Links == nil {
		rs.content.Links = map[string]int{}
	}
	if len(snd.Thresholds) > 0 {
		_ = json.Unmarshal(snd.Thresholds, &rs.thresholds)
	}
	rs.throttles = map[string]float64{}
	_ = json.Unmarshal(snd.Throttles, &rs.throttles)
	rs.holds = parseHolds(snd.Holds)
	if rs.templates, err = e.templatesFor(id, rs.content); err != nil {
		return nil, err
	}
	if e.Conditional != nil {
		if rs.links, err = e.Conditional.Links(ctx, cc.WorkspaceID, id); err != nil {
			return nil, err
		}
	}
	if err := e.loadRoutes(ctx, rs, now); err != nil {
		return nil, err
	}
	return rs, nil
}

func (e *Engine) loadRoutes(ctx context.Context, rs *runState, now time.Time) error {
	rows, err := e.Q.SendingRoutes(ctx, rs.campaign.ID)
	if err != nil {
		return err
	}
	rs.order = rs.order[:0]
	for _, row := range rows {
		p, err := e.Q.ProvGetAny(ctx, row.ProviderID)
		if err != nil {
			return err
		}
		r := rs.routes[row.ProviderID]
		if r == nil {
			r = &runRoute{}
			rs.routes[row.ProviderID] = r
			r.transport, _, r.openErr = e.transport(ctx, p, rs.send.Scenario, rs.started)
			if s, ok := r.transport.(*simulator.Provider); ok {
				r.sim = s
			}
		}
		if (r.status == "held" || r.status == "paused") && (row.Status == "pending" || row.Status == "sending") {
			r.judgedFrom = now
		}
		r.provider, r.planned, r.status, r.detail = p, row.Planned, row.Status, parseRouteDetail(row.StatusDetail)
		r.rate = e.rateOf(ctx, e.Q, p, rs.loc, now)
		r.batch = batchSize(p)
		r.tracked = state.TrackingOf(p).Mode != "provider"
		if pr := state.PricingOf(p); pr.Model == "metered" {
			r.micros = int64(math.Round(pr.Per1000 * 1000))
		}
		rs.order = append(rs.order, row.ProviderID)
	}
	return nil
}

// refresh rereads the campaign's status, holds and routes (people act from
// other processes while the loop runs).
func (e *Engine) refresh(ctx context.Context, rs *runState, now time.Time) error {
	c, err := e.Q.SendingCampaign(ctx, rs.campaign.ID)
	if err != nil {
		return err
	}
	snd, err := e.Q.SendingGetSend(ctx, rs.campaign.ID)
	if err != nil {
		return err
	}
	if c.Status != "sending" || snd.Phase != "sending" {
		return errStopped
	}
	rs.send = snd
	rs.holds = parseHolds(snd.Holds)
	rs.ab = parseAB(snd.Ab)
	rs.throttles = map[string]float64{}
	_ = json.Unmarshal(snd.Throttles, &rs.throttles)
	rs.loadedAt = now
	return e.loadRoutes(ctx, rs, now)
}

// refreshQueue rereads what is left per route and mailbox provider.
func (e *Engine) refreshQueue(ctx context.Context, rs *runState, now time.Time) error {
	cid := rs.campaign.ID
	rows, err := e.Q.SendingQueueState(ctx, store.SendingQueueStateParams{CampaignID: &cid, Now: now})
	if err != nil {
		return err
	}
	q := map[queueKey]queueCell{}
	for _, r := range rows {
		if r.ProviderID == nil {
			continue
		}
		k := queueKey{*r.ProviderID, r.MailboxProvider}
		c := q[k]
		c.add(r)
		q[k] = c
	}
	rs.queue, rs.queueAt = q, now
	return nil
}

// add counts one row of SendingQueueState.
func (c *queueCell) add(r store.SendingQueueStateRow) {
	switch r.Status {
	case "held":
		c.held += r.N
		if r.HoldReason == holdCanary {
			c.canary += r.N
		}
	case "in_doubt":
		c.doubt += r.N
	default:
		c.queued += r.N
		c.waiting += r.Waiting
		if r.NextAt.Year() > 2000 && (c.nextAt == nil || r.NextAt.Before(*c.nextAt)) {
			t := r.NextAt
			c.nextAt = &t
		}
	}
}

// holdCanary is messages.hold_reason of the recipients held until the
// canary passes (canary.go).
const holdCanary = "canary"

// Run sends a campaign for up to budget, then returns how soon it should
// run again (done when there is nothing left to do now).
func (e *Engine) Run(ctx context.Context, id uuid.UUID, budget time.Duration) (again time.Duration, done bool, err error) {
	start := e.now()
	unlock, err := e.lockSend(ctx, id)
	if errors.Is(err, errBusy) {
		return 5 * time.Second, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	defer unlock()
	rs, err := e.load(ctx, id, start)
	if errors.Is(err, errStopped) {
		return 0, true, nil
	}
	if err != nil {
		return 0, false, err
	}
	for {
		now := e.now()
		if now.Sub(start) >= budget {
			return 0, false, nil
		}
		idle, err := e.Step(ctx, rs, now)
		if errors.Is(err, errStopped) {
			return 0, true, nil
		}
		if err != nil {
			return 0, false, err
		}
		wait := time.Until(now.Truncate(time.Second).Add(time.Second))
		if idle > 0 {
			// Nothing to send for a while (A/B window, retries, holds):
			// come back later instead of spinning.
			if idle > 5*time.Second {
				return min(idle, 30*time.Second), false, nil
			}
			wait = max(wait, idle)
		}
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Step sends one second's worth across the campaign's routes and every two
// seconds checks thresholds, outages, the A/B test and completion. It
// returns how long nothing can be sent (0 when it sent or will send next
// second). now is explicit so tests and fast-forwarded demo sends drive it.
func (e *Engine) Step(ctx context.Context, rs *runState, now time.Time) (time.Duration, error) {
	if now.Sub(rs.loadedAt) >= 3*time.Second {
		if err := e.refresh(ctx, rs, now); err != nil {
			return 0, err
		}
	}
	if now.Sub(rs.queueAt) >= 3*time.Second || rs.queue == nil {
		if err := e.refreshQueue(ctx, rs, now); err != nil {
			return 0, err
		}
	}
	e.refill(rs, now)
	sent := 0
	for _, pid := range rs.order {
		r := rs.routes[pid]
		if !r.active(now) || r.cooldown.After(now) {
			continue
		}
		n, err := e.tick(ctx, rs, r, now)
		if err != nil {
			return 0, err
		}
		sent += n
	}
	if sent > 0 {
		rs.queueAt = time.Time{} // recount soon
	}
	if now.Sub(rs.guardedAt) >= 2*time.Second || sent == 0 {
		rs.guardedAt = now
		idle, err := e.guard(ctx, rs, now)
		if err != nil {
			return 0, err
		}
		if sent == 0 {
			return idle, nil
		}
	}
	return 0, nil
}

// refill adds per-mailbox-provider throttle tokens for the time passed.
func (e *Engine) refill(rs *runState, now time.Time) {
	dt := 1.0
	if !rs.tokensAt.IsZero() {
		dt = math.Min(now.Sub(rs.tokensAt).Seconds(), 2)
	}
	rs.tokensAt = now
	for mbp, rate := range rs.throttles {
		if rate <= 0 {
			continue
		}
		rs.tokens[mbp] = math.Min(rate, rs.tokens[mbp]+rate*dt)
	}
}

// tick sends one second's allowance through one route.
func (e *Engine) tick(ctx context.Context, rs *runState, r *runRoute, now time.Time) (int, error) {
	rate := r.rate.PerSecond * r.slowFactor()
	if r.rate.WarmingUp && r.rate.Cap > 0 {
		left := r.rate.Cap - r.rate.SentToday
		if left <= 0 {
			return 0, nil
		}
	}
	r.carry += rate
	want := int(math.Floor(r.carry))
	if want <= 0 {
		return 0, nil
	}
	alloc := e.allocate(rs, r, want)
	total := 0
	for _, n := range alloc {
		total += n
	}
	if total == 0 {
		r.carry = math.Min(r.carry, rate+1)
		return 0, nil
	}
	granted, err := e.reserveAfterTransactional(ctx, r.provider.ID, now.Unix(), total, int(math.Ceil(rate)), now)
	if err != nil {
		return 0, err
	}
	if granted <= 0 {
		// Another send used this second's window; don't bank the allowance.
		r.carry = math.Min(r.carry, rate+1)
		return 0, nil
	}
	if granted < total {
		alloc = scaleAlloc(alloc, granted)
	}
	var claimed []store.SendingClaimRow
	cid := rs.campaign.ID
	pid := r.provider.ID
	for _, mbp := range sortedKeys(alloc) {
		n := alloc[mbp]
		if n <= 0 {
			continue
		}
		rows, err := e.Q.SendingClaim(ctx, store.SendingClaimParams{CampaignID: &cid, ProviderID: &pid, MailboxProvider: mbp, Now: now, Lim: int32(n)})
		if err != nil {
			// What this second claimed already was never handed to the
			// provider: it goes back to the queue, not into doubt.
			e.releaseClaims(ctx, claimed)
			return 0, err
		}
		claimed = append(claimed, rows...)
		if t, ok := rs.throttles[mbp]; ok && t > 0 {
			rs.tokens[mbp] -= float64(len(rows))
		}
		k := queueKey{pid, mbp}
		c := rs.queue[k]
		c.queued = max(c.queued-int64(len(rows)), 0)
		rs.queue[k] = c
	}
	r.carry -= float64(len(claimed))
	if r.carry < 0 {
		r.carry = 0
	}
	if len(claimed) == 0 {
		r.carry = math.Min(r.carry, rate+1)
		return 0, nil
	}
	return len(claimed), e.deliver(ctx, rs, r, claimed, now)
}

// allocate splits a route's allowance across mailbox providers in
// proportion to what each has left, within each mailbox provider's
// throttle, skipping mailbox providers on hold.
func (e *Engine) allocate(rs *runState, r *runRoute, want int) map[string]int {
	type cand struct {
		mbp  string
		left int64
		cap  float64
	}
	var cands []cand
	var total int64
	for k, c := range rs.queue {
		if k.provider != r.provider.ID || c.queued-c.waiting <= 0 {
			continue
		}
		if h, ok := rs.holds[k.mailbox]; ok && !h.Resumed {
			continue
		}
		capN := math.Inf(1)
		if t, ok := rs.throttles[k.mailbox]; ok && t > 0 {
			capN = math.Floor(rs.tokens[k.mailbox])
			if capN <= 0 {
				continue
			}
		}
		left := c.queued - c.waiting
		cands = append(cands, cand{k.mailbox, left, capN})
		total += left
	}
	out := map[string]int{}
	if total == 0 {
		return out
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].left > cands[j].left })
	remaining := want
	for _, c := range cands {
		share := int(math.Round(float64(want) * float64(c.left) / float64(total)))
		share = min(share, int(math.Min(c.cap, float64(c.left))), remaining)
		if share > 0 {
			out[c.mbp] = share
			remaining -= share
		}
	}
	// Hand what rounding or throttles left over to whoever can take it.
	for remaining > 0 {
		gave := false
		for _, c := range cands {
			if remaining == 0 {
				break
			}
			if float64(out[c.mbp]) < math.Min(c.cap, float64(c.left)) {
				out[c.mbp]++
				remaining--
				gave = true
			}
		}
		if !gave {
			break
		}
	}
	return out
}

func scaleAlloc(alloc map[string]int, n int) map[string]int {
	total := 0
	for _, v := range alloc {
		total += v
	}
	out := map[string]int{}
	left := n
	for _, k := range sortedKeys(alloc) {
		v := alloc[k] * n / max(total, 1)
		out[k] = v
		left -= v
	}
	for _, k := range sortedKeys(alloc) {
		if left == 0 {
			break
		}
		if out[k] < alloc[k] {
			out[k]++
			left--
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// reserve takes up to want sends from the provider's shared one-second
// window (every worker and campaign shares it).
func (e *Engine) reserve(ctx context.Context, provider uuid.UUID, sec int64, want, capN int) (int, error) {
	if capN <= 0 {
		capN = 1
	}
	if err := e.Q.SendingOpenWindow(ctx, store.SendingOpenWindowParams{ProviderID: provider, Sec: sec}); err != nil {
		return 0, err
	}
	g, err := e.Q.SendingReserve(ctx, store.SendingReserveParams{Want: int32(want), Cap: int32(capN), ProviderID: provider, Sec: sec})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return int(g), err
}

// outcome collects what happened to a batch.
type outcome struct {
	sentIDs  []uuid.UUID
	sentPMID []string
	retryIDs []uuid.UUID
	retryAt  []time.Time
	retryErr []string
	unsentID []uuid.UUID
	unsentSt []string
	unsentEr []string
	release  []uuid.UUID
	cells    map[cellKey]*cellCount
	sims     [][]any
	cost     int64
}

type cellKey struct {
	mailbox, variant string
}

type cellCount struct{ sent, rejected int64 }

func (o *outcome) cell(mbp, variant string) *cellCount {
	if o.cells == nil {
		o.cells = map[cellKey]*cellCount{}
	}
	k := cellKey{mbp, variant}
	c := o.cells[k]
	if c == nil {
		c = &cellCount{}
		o.cells[k] = c
	}
	return c
}

func (o *outcome) retry(id uuid.UUID, attempts int, now time.Time, msg string) {
	if attempts >= maxAttempts {
		o.unsentID = append(o.unsentID, id)
		o.unsentSt = append(o.unsentSt, "failed")
		o.unsentEr = append(o.unsentEr, clip("Gave up after "+itoa(attempts)+" attempts: "+msg, 500))
		return
	}
	o.retryIDs = append(o.retryIDs, id)
	o.retryAt = append(o.retryAt, now.Add(backoff(attempts)))
	o.retryErr = append(o.retryErr, clip(msg, 500))
}

// doubt puts a message whose call may have reached the provider in doubt
// (doubt.go): never retried on a guess.
func (o *outcome) doubt(id uuid.UUID) {
	o.unsentID = append(o.unsentID, id)
	o.unsentSt = append(o.unsentSt, "in_doubt")
	o.unsentEr = append(o.unsentEr, inDoubtReason)
}

func (o *outcome) drop(id uuid.UUID, status, msg string) {
	o.unsentID = append(o.unsentID, id)
	o.unsentSt = append(o.unsentSt, status)
	o.unsentEr = append(o.unsentEr, clip(msg, 500))
}

// deliver sends claimed messages (send) and acts on what the provider said
// about the route: a hold, an outage, a slowdown.
func (e *Engine) deliver(ctx context.Context, rs *runState, r *runRoute, claimed []store.SendingClaimRow, now time.Time) error {
	d, err := e.send(ctx, rs, r, claimed, now)
	if err != nil {
		return err
	}
	switch {
	case d.hold == "quota":
		// An outage of the route: held, or moved when the Broadcast policy
		// allows it and a vetted target can take the rest.
		return e.outage(ctx, rs, r, d.hold, now, false)
	case d.hold != "":
		return e.holdRoute(ctx, rs, r, d.hold, now)
	case d.slowed:
		r.throttled(now)
	}
	return nil
}

// delivery is what a batch's provider calls said about the route: a hold
// (rejected credentials, an unverified sender, a used-up quota), whether the
// provider paced the account, and how many messages it took.
type delivery struct {
	hold   string
	slowed bool
	sent   int
}

// send renders claimed messages, hands them to the provider in batches and
// records every outcome in one transaction. Once messages are claimed it
// runs to the end even when the job is told to stop: rendering, the
// provider calls and the record all happen, so nothing is left claimed.
func (e *Engine) send(ctx context.Context, rs *runState, r *runRoute, claimed []store.SendingClaimRow, now time.Time) (delivery, error) {
	ctx = context.WithoutCancel(ctx)
	var out outcome
	type pending struct {
		row store.SendingClaimRow
		msg providers.Message
	}
	var batch []pending
	extra := e.readerVars(ctx, rs, claimed)
	for _, m := range claimed {
		switch {
		case m.Gone:
			out.drop(m.ID, "cancelled", "The contact was erased, unsubscribed or suppressed before the send reached them.")
			continue
		case m.Suppressed:
			out.drop(m.ID, "cancelled", "On the shared suppression list since the send started.")
			continue
		}
		pm, err := e.compose(rs, r, m, extra)
		if err != nil {
			out.drop(m.ID, "failed", err.Error())
			continue
		}
		batch = append(batch, pending{m, pm})
	}
	key := routeKey{rs.campaign.ID, r.provider.ID}
	// hold is an answer about the route (rejected credentials, an unverified
	// sender, a used-up quota), acted on once the batch is recorded; slowed
	// means the provider paced the account (routeerrors.go).
	hold, slowed := "", false
	for len(batch) > 0 {
		n := min(len(batch), max(r.batch, 1))
		chunk := batch[:n]
		batch = batch[n:]
		msgs := make([]providers.Message, len(chunk))
		for i, p := range chunk {
			msgs[i] = p.msg
		}
		cctx, cancel := e.callContext(ctx, now)
		res, callErr := r.transport.Send(cctx, msgs)
		cancel()
		if callErr == nil && len(res) != len(msgs) {
			callErr = &providers.CallError{Op: "Send", Message: fmt.Sprintf("the provider answered for %d of %d messages", len(res), len(msgs)), Temporary: true}
		}
		e.recordCall(key, now, callErr)
		if err := manage.RecordSend(context.WithoutCancel(ctx), e.Pool, r.provider, res, callErr, now); err != nil {
			e.log().WarnContext(ctx, "record provider call", "provider", r.provider.ID, "err", err)
		}
		if callErr != nil {
			// The batch goes back to the queue without using an attempt when
			// the answer is about the route or the provider paces the account.
			if hold, slowed = routeHold(callErr), throttledErr(callErr); hold != "" || slowed {
				for _, p := range chunk {
					out.release = append(out.release, p.row.ID)
				}
				break
			}
			// A call that may have reached the provider (a timeout, a cut
			// connection) leaves its messages in doubt, unless the provider
			// dedupes them by message id; one the provider answered or never
			// got is retried.
			doubt := outcomeUnknown(callErr) && idempotency(r.transport) == 0
			for _, p := range chunk {
				if doubt {
					out.doubt(p.row.ID)
					continue
				}
				out.retry(p.row.ID, int(p.row.Attempts), now, callText(callErr))
			}
			// Back off the whole route briefly; the rest of this second's
			// batch goes back to the queue.
			r.cooldown = now.Add(backoffMin)
			break
		}
		r.answered(now)
		for i, res := range res {
			p := chunk[i]
			switch {
			case res.Err == nil:
				out.sentIDs = append(out.sentIDs, p.row.ID)
				out.sentPMID = append(out.sentPMID, res.ProviderMessageID)
				out.cell(p.row.MailboxProvider, p.row.Variant).sent++
				out.cost += r.micros
				if r.sim != nil && !e.QuietDemo {
					out.sims = append(out.sims, simRows(rs.campaign.WorkspaceID, &rs.campaign.ID, p.row.ID, r.provider.ID, now, r.sim.Plan(p.msg))...)
				}
			case sendErrIdentity(res.Err):
				out.release, hold = append(out.release, p.row.ID), "identity"
			case sendErrQuota(res.Err):
				out.release, hold = append(out.release, p.row.ID), "quota"
			case res.Err.Throttled:
				out.release, slowed = append(out.release, p.row.ID), true
			case sendErrUnknown(res.Err) && idempotency(r.transport) == 0:
				out.doubt(p.row.ID)
			case res.Err.Temporary:
				out.retry(p.row.ID, int(p.row.Attempts), now, res.Err.Error())
			default:
				out.drop(p.row.ID, "rejected", res.Err.Error())
				out.cell(p.row.MailboxProvider, p.row.Variant).rejected++
			}
		}
		if hold != "" || slowed {
			break
		}
	}
	// What this second didn't hand over goes back to the queue.
	for _, p := range batch {
		out.release = append(out.release, p.row.ID)
	}
	// The provider has the batch now: the outcome is recorded even when the
	// job is told to stop. Messages it couldn't record stay claimed and end
	// in doubt (doubt.go), never sent again on a guess.
	err := e.recordOutcome(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return e.record(ctx, store.New(tx), tx, rs.campaign.ID, r.provider.ID, &out, now)
	})
	if err != nil {
		return delivery{}, err
	}
	if len(out.sims) > 0 {
		e.insert(ctx, SimArgs{}, nil)
	}
	return delivery{hold: hold, slowed: slowed, sent: len(out.sentIDs)}, nil
}

// record writes a batch's outcomes.
func (e *Engine) record(ctx context.Context, q *store.Queries, tx pgx.Tx, campaign, provider uuid.UUID, out *outcome, now time.Time) error {
	if len(out.sentIDs) > 0 {
		if err := q.SendingMarkSent(ctx, store.SendingMarkSentParams{At: &now, Ids: out.sentIDs, ProviderMessageIds: out.sentPMID}); err != nil {
			return err
		}
	}
	if len(out.retryIDs) > 0 {
		if err := q.SendingRetryLater(ctx, store.SendingRetryLaterParams{Ids: out.retryIDs, Ats: out.retryAt, Errors: out.retryErr}); err != nil {
			return err
		}
	}
	if len(out.unsentID) > 0 {
		if err := q.SendingMarkUnsent(ctx, store.SendingMarkUnsentParams{Ids: out.unsentID, Statuses: out.unsentSt, Errors: out.unsentEr}); err != nil {
			return err
		}
	}
	if len(out.release) > 0 {
		if err := q.SendingRelease(ctx, out.release); err != nil {
			return err
		}
	}
	if len(out.cells) > 0 && campaign != uuid.Nil {
		// Counter rows are locked in one order (mailbox provider, variant),
		// as the delivery-event pump does, so batches don't deadlock.
		keys := make([]cellKey, 0, len(out.cells))
		for k := range out.cells {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].mailbox != keys[j].mailbox {
				return keys[i].mailbox < keys[j].mailbox
			}
			return keys[i].variant < keys[j].variant
		})
		p := store.SendingBumpCountersParams{CampaignID: campaign}
		for _, k := range keys {
			c := out.cells[k]
			p.ProviderIds = append(p.ProviderIds, provider)
			p.Mailboxes = append(p.Mailboxes, k.mailbox)
			p.Variants = append(p.Variants, k.variant)
			p.Sent = append(p.Sent, c.sent)
			p.Rejected = append(p.Rejected, c.rejected)
		}
		if err := q.SendingBumpCounters(ctx, p); err != nil {
			return err
		}
	}
	if out.cost > 0 && campaign != uuid.Nil {
		if err := q.SendingAddRouteCost(ctx, store.SendingAddRouteCostParams{CampaignID: campaign, ProviderID: provider, CostMicros: out.cost}); err != nil {
			return err
		}
	} else if len(out.sentIDs) > 0 && campaign != uuid.Nil {
		if err := q.SendingAddRouteCost(ctx, store.SendingAddRouteCostParams{CampaignID: campaign, ProviderID: provider, CostMicros: 0}); err != nil {
			return err
		}
	}
	if len(out.sims) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sending_sim_events"}, []string{"workspace_id", "campaign_id", "message_id", "provider_id", "due_at", "kind", "data"},
			pgx.CopyFromRows(out.sims)); err != nil {
			return err
		}
	}
	return nil
}

// simRows turns the demo transport's plan into rows to apply when due.
func simRows(ws uuid.UUID, campaign *uuid.UUID, msg, provider uuid.UUID, now time.Time, plan []simulator.Event) [][]any {
	rows := make([][]any, 0, len(plan))
	for _, ev := range plan {
		data, _ := json.Marshal(map[string]any{"type": ev.OriginalType, "code": ev.Code, "diagnostic": ev.Diagnostic, "class": ev.BounceClass,
			"ua": ev.UserAgent, "ip": ev.IP, "link": ev.Link})
		rows = append(rows, []any{ws, campaign, msg, provider, now.Add(ev.After), ev.Kind, data})
	}
	return rows
}

// compose builds one recipient's message: personalization, per-recipient
// tracking and system links, and the bulk-mail headers.
func (e *Engine) compose(rs *runState, r *runRoute, m store.SendingClaimRow, extra map[uuid.UUID]map[string]any) (providers.Message, error) {
	key := renderKey(rs.content, m.Language, m.Variant)
	if rs.templates[key] == nil {
		key = renderKey(rs.content, m.Language, "")
	}
	if rs.templates[key] == nil {
		key = ""
	}
	// The recipient's version: their hidden conditional blocks cut out.
	tpl, err := e.versionTemplate(rs, key, m.HiddenBlocks)
	if err != nil {
		return providers.Message{}, err
	}
	ws, slug := rs.campaign.WorkspaceID, rs.campaign.WorkspaceSlug
	var contact uuid.UUID
	if m.ContactID != nil {
		contact = *m.ContactID
	}
	sign := func(k linktoken.Kind, link uint32) string {
		return e.Links.Sign(linktoken.Claims{Kind: k, WorkspaceID: ws, MessageID: m.ID, ContactID: contact, Link: link})
	}
	unsub := sign(linktoken.Unsubscribe, 0)
	links := map[string]string{
		"unsubscribe_url":     e.URLs.Unsubscribe(slug, unsub),
		"preferences_url":     e.URLs.Preferences(slug, sign(linktoken.Preferences, 0)),
		"view_in_browser_url": e.URLs.ViewOnline(slug, sign(linktoken.ViewOnline, 0)),
	}
	for _, l := range rs.links {
		links[l.Var] = l.URL(m.ID, contact)
	}
	tracked := r.tracked && (tpl.Opens || tpl.Clicks)
	if tracked {
		if tpl.Opens {
			links[OpenVar] = e.URLs.Open(sign(linktoken.Open, 0))
		}
		off := rs.content.Links[languageContentKey(rs.content, m.Language, m.Variant)]
		for _, l := range tpl.Links {
			links[LinkVar(l.Position)] = e.URLs.Click(sign(linktoken.Click, uint32(off+l.Position)))
		}
	}
	var attrs map[string]any
	if len(m.Attributes) > 0 {
		_ = json.Unmarshal(m.Attributes, &attrs)
	}
	vars := liquid.Vars(m.Email, m.FirstName, m.LastName, attrs, links)
	if m.ContactID != nil {
		vars = liquid.Merge(vars, extra[*m.ContactID])
	}
	out, err := tpl.Render(vars, tracked)
	if err != nil {
		return providers.Message{}, err
	}
	msg := providers.Message{
		ID: m.ID, Stream: providers.StreamBroadcast,
		From:    mail.Address{Name: rs.content.FromName, Address: rs.content.FromEmail},
		To:      mail.Address{Name: strings.TrimSpace(m.FirstName + " " + m.LastName), Address: m.Email},
		Subject: out.Subject, HTML: out.HTML, Text: out.Text,
		Headers: bulkHeaders(links["unsubscribe_url"], unsub, firstNonEmpty(rs.content.ReplyTo, rs.content.FromEmail), feedbackID(rs.campaign.ID.String(), slug, "broadcast")),
		Tags:    map[string]string{"campaign": rs.campaign.ID.String(), "workspace": slug},
	}
	if rs.content.ReplyTo != "" {
		if a, err := mail.ParseAddress(rs.content.ReplyTo); err == nil {
			msg.ReplyTo = a
		}
	}
	if r.sim != nil {
		msg.Tags[simulator.TagMailbox] = m.MailboxProvider
		msg.Tags[simulator.TagTier] = m.Tier
		msg.Tags[simulator.TagSource] = m.Source
		msg.Tags[simulator.TagEngaged] = fmt.Sprint(m.Engaged)
	}
	return msg, nil
}

// readerVars are the per-reader variables (Engine.Recipients) the send's
// templates use, for a page of claimed messages, by contact.
func (e *Engine) readerVars(ctx context.Context, rs *runState, claimed []store.SendingClaimRow) map[uuid.UUID]map[string]any {
	if e.Recipients == nil || len(claimed) == 0 {
		return nil
	}
	providers := e.Recipients()
	if len(providers) == 0 {
		return nil
	}
	var src strings.Builder
	for _, t := range rs.templates {
		src.WriteString(t.Source())
	}
	recipients := make([]liquid.Recipient, 0, len(claimed))
	for _, m := range claimed {
		if m.ContactID != nil && !m.Gone && !m.Suppressed {
			recipients = append(recipients, liquid.Recipient{ContactID: *m.ContactID, MessageID: m.ID})
		}
	}
	return liquid.FillBatch(ctx, providers, rs.campaign.WorkspaceID, recipients, src.String())
}

// contentKey is the variant whose content (and links) a message renders.
func contentKey(c sendContent, variant string) string {
	for _, v := range c.Variants {
		if v.Key == variant && v.HTML != "" && v.Key != "A" {
			return v.Key
		}
	}
	return ""
}

// bulkHeaders are the headers every bulk message carries: one-click
// unsubscribe (RFC 8058) with a mailto alternative, and Feedback-ID for
// Gmail's feedback loop. The Message-ID comes from the message id.
func bulkHeaders(unsubURL, token, mailto, feedback string) map[string]string {
	h := map[string]string{
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		"Feedback-ID":           feedback,
	}
	lu := "<" + unsubURL + ">"
	if mailto != "" {
		lu += ", <mailto:" + mailto + "?subject=" + url.PathEscape("unsubscribe "+token) + ">"
	}
	h["List-Unsubscribe"] = lu
	return h
}

// feedbackID is Gmail's Feedback-ID: campaign, customer (workspace), mail
// type and a constant sender id.
func feedbackID(campaign, workspace, stream string) string {
	if len(campaign) > 8 {
		campaign = campaign[:8]
	}
	return campaign + ":" + workspace + ":" + stream + ":reputecast"
}

// mailboxName is a mailbox provider's display name.
func mailboxName(k string) string { return routing.MailboxName(k) }
