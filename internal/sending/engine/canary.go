package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// The canary (review GS-10). A send starts with a canary when Review &
// send asks for one, or by itself (Traffic rules) when the sending domain
// or a dedicated sending IP is new, some recipients never got mail from
// the domain, or the audience is over twice the usual: the most engaged
// few percent, plus the workspace's seed mailboxes, get the email first
// while everyone else waits. After the last canary message the canary is
// watched for a while; then every gate with enough evidence must pass
// (bounces, block/policy bounces, deferrals, complaints per mailbox
// provider, seeds in spam) and the send continues by itself. A gate that
// fails holds the send and asks. People
// who can send stop it or watch longer; people who can approve continue
// early. The most engaged readers can't say how new imports will react:
// their complaints are watched during the main send and after it.

// Canary choices (StartOptions.Canary, the send body's "canary"; C1).
const (
	CanaryAuto = "auto"
	CanaryOn   = "on"
	CanaryOff  = "off"
)

// Why a canary runs by itself. A new sending IP counts as a new domain
// does (Traffic rules › "The sending domain or IP is new").
const (
	reasonNewDomain   = "new_domain"
	reasonNewIP       = "new_ip"
	reasonNeverMailed = "never_mailed"
	reasonOverUsual   = "over_usual"
)

// CanaryIP is a sending IP of a send's routes that never sent for the
// workspace, and the provider it belongs to (one type, so Review & send
// and the live monitor read the same API schema).
type CanaryIP = sending.CanaryIP

// Canary phases (campaign_sends.canary's phase).
const (
	canarySending   = "sending"
	canaryWatching  = "watching"
	canaryHeld      = "held"
	canaryPassed    = "passed"
	canaryContinued = "continued"
	canaryStopped   = "stopped"
)

// canaryExtend is how much longer "Watch 10 more minutes" watches.
const canaryExtend = 10 * time.Minute

// seedEvery is how often the loop reads the seed placement.
const seedEvery = 30 * time.Second

// minCanaryAudience is the smallest audience an automatic canary runs on:
// a few percent of fewer recipients can't show anything. A person can
// still ask for one in Review & send.
const minCanaryAudience = 1000

// CanaryPlan is what a send's canary would be (C1): whether it runs by
// itself, why, and how many recipients go first. Review & send shows it
// with the choice; the plan is worked out again when the send starts,
// on the audience as built.
type CanaryPlan struct {
	Auto    bool     `json:"auto"`
	Reasons []string `json:"reasons"`
	Size    int64    `json:"size"`
	// The workspace's canary settings, and the numbers behind the reasons.
	Enabled      bool    `json:"enabled"`
	Percent      float64 `json:"percent"`
	Seeds        int     `json:"seeds"`
	WatchMinutes int     `json:"watchMinutes"`
	Audience     int64   `json:"audience"`
	// NeverMailed are the recipients that never got mail from the domain
	// (before the audience is built: contacts added since its last send).
	NeverMailed int64  `json:"neverMailed"`
	Usual       int64  `json:"usual"`
	Domain      string `json:"domain"`
	// DomainSince is when the workspace first sent from the domain.
	DomainSince *time.Time `json:"domainSince,omitempty"`
	// NewIPs are the dedicated sending IPs of the send's routes that never
	// sent for the workspace (review GS-10).
	NewIPs []CanaryIP `json:"newIps"`
	// MinAudience is the smallest audience an automatic canary runs on.
	MinAudience int64 `json:"minAudience"`
}

// canaryState is campaign_sends.canary.
type canaryState struct {
	// Choice is how the send was started: auto, on or off.
	Choice  string   `json:"choice,omitempty"`
	Phase   string   `json:"phase,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
	Percent float64  `json:"percent,omitempty"`
	Size    int64    `json:"size,omitempty"`
	Seeds   int      `json:"seeds,omitempty"`
	// SeedCheck is the seed placement test's id (internal/placement);
	// SeedResult its latest result, read at most every 30 s.
	SeedCheck   string         `json:"seedCheck,omitempty"`
	SeedResult  *SeedPlacement `json:"seedResult,omitempty"`
	SeedAt      *time.Time     `json:"seedAt,omitempty"`
	NeverMailed int64          `json:"neverMailed,omitempty"`
	Usual       int64          `json:"usual,omitempty"`
	Domain      string         `json:"domain,omitempty"`
	NewIPs      []CanaryIP     `json:"newIps,omitempty"`
	Started     *time.Time     `json:"started,omitempty"`
	SentAt      *time.Time     `json:"sentAt,omitempty"`
	Until       *time.Time     `json:"until,omitempty"`
	// Extended is the minutes people added to the watch.
	Extended  int        `json:"extended,omitempty"`
	Decided   *time.Time `json:"decided,omitempty"`
	DecidedBy string     `json:"decidedBy,omitempty"`
	Incident  string     `json:"incident,omitempty"`
	// Watch is the watch's length in minutes; Gates what it must stay under.
	Watch int               `json:"watch,omitempty"`
	Gates rules.CanaryGates `json:"gates"`
}

func parseCanary(raw json.RawMessage) canaryState {
	var c canaryState
	_ = json.Unmarshal(raw, &c)
	return c
}

func (c canaryState) active() bool {
	return c.Phase == canarySending || c.Phase == canaryWatching || c.Phase == canaryHeld
}

// SeedChecker sends a campaign's content to the workspace's seed mailboxes
// and reads where the copies landed (the app wires internal/placement): the
// canary's seed gate.
type SeedChecker interface {
	StartSeeds(ctx context.Context, ws, campaignID uuid.UUID) (check uuid.UUID, seeds int, err error)
	SeedPlacement(ctx context.Context, ws, check uuid.UUID) (SeedPlacement, error)
}

// SeedPlacement is where the canary's seed copies landed.
type SeedPlacement struct {
	Inbox      int `json:"inbox"`
	Promotions int `json:"promotions"`
	Spam       int `json:"spam"`
	Pending    int `json:"pending"`
	// Missing copies never arrived before the test ended.
	Missing   int                 `json:"missing"`
	ByMailbox []SeedMailboxResult `json:"byMailbox"`
}

// SeedMailboxResult is one mailbox provider's seeds.
type SeedMailboxResult struct {
	Mailbox    string `json:"mailbox" enum:"gmail,outlook,yahoo,icloud,other"`
	Seeds      int    `json:"seeds"`
	Inbox      int    `json:"inbox"`
	Promotions int    `json:"promotions"`
	Spam       int    `json:"spam"`
}

// SeedMailboxOrder sorts mailbox provider keys as the monitor lists them.
func SeedMailboxOrder(keys []string) []string {
	out := slices.Clone(keys)
	sort.SliceStable(out, func(i, j int) bool {
		return slices.Index(routing.MailboxProviders, out[i]) < slices.Index(routing.MailboxProviders, out[j])
	})
	return out
}

// CanaryPlanFor is a campaign's canary plan (C1): Review & send shows it
// and the choice.
func (e *Engine) CanaryPlanFor(ctx context.Context, campaignID uuid.UUID) (CanaryPlan, error) {
	cc, err := e.Q.SendingCampaignContext(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CanaryPlan{}, ErrNotFound
	}
	if err != nil {
		return CanaryPlan{}, err
	}
	audience := int64(0)
	if cc.AudienceCount != nil {
		audience = *cc.AudienceCount
	}
	return e.canaryPlan(ctx, e.Q, cc, audience, false)
}

// canaryPlan works out a send's canary for an audience; built: the send's
// recipients exist, so the ones never mailed are counted exactly.
func (e *Engine) canaryPlan(ctx context.Context, q *store.Queries, cc store.SendingCampaignContextRow, audience int64, built bool) (CanaryPlan, error) {
	settings, _, err := rules.Load(ctx, q, cc.WorkspaceID)
	if err != nil {
		return CanaryPlan{}, err
	}
	p := CanaryPlan{Reasons: []string{}, Enabled: settings.Enabled, Percent: settings.Percent, WatchMinutes: settings.WatchMinutes, Audience: audience,
		Domain: domainOf(cc.SenderEmail), NewIPs: []CanaryIP{}, MinAudience: minCanaryAudience}
	if settings.Seeds {
		p.Seeds = e.seedCount(ctx, cc.WorkspaceID)
	}
	p.Size = int64(math.Ceil(float64(audience) * settings.Percent / 100))
	if audience > 0 && p.Size < 1 {
		p.Size = 1
	}
	now := e.now()
	var last time.Time
	if p.Domain != "" {
		first, err := q.SendingDomainFirstSend(ctx, store.SendingDomainFirstSendParams{WorkspaceID: cc.WorkspaceID, CampaignID: cc.ID, Domain: p.Domain})
		if err != nil {
			return p, err
		}
		if first.Year() > 1971 {
			p.DomainSince = &first
		}
		if settings.NewDomain && (p.DomainSince == nil || now.Sub(first) < 30*24*time.Hour) {
			p.Reasons = append(p.Reasons, reasonNewDomain)
		}
		if last, err = q.SendingDomainLastSend(ctx, store.SendingDomainLastSendParams{WorkspaceID: cc.WorkspaceID, CampaignID: cc.ID, Domain: p.Domain}); err != nil {
			return p, err
		}
	}
	if settings.NewDomain {
		if err := canaryIPs(ctx, q, cc, &p); err != nil {
			return p, err
		}
	}
	switch {
	case p.Domain == "":
	case built:
		if p.NeverMailed, err = q.SendingCanaryNeverMailed(ctx, store.SendingCanaryNeverMailedParams{CampaignID: &cc.ID, Domain: p.Domain}); err != nil {
			return p, err
		}
	case last.Year() > 1971:
		if p.NeverMailed, err = q.SendingContactsAddedSince(ctx, store.SendingContactsAddedSinceParams{WorkspaceID: cc.WorkspaceID, Since: last}); err != nil {
			return p, err
		}
	}
	// A new domain says it already: everyone is new to it.
	if settings.NeverMailed && p.NeverMailed > 0 && !slices.Contains(p.Reasons, reasonNewDomain) {
		p.Reasons = append(p.Reasons, reasonNeverMailed)
	}
	sizes, err := q.SendingRecentAudiences(ctx, store.SendingRecentAudiencesParams{WorkspaceID: cc.WorkspaceID, CampaignID: cc.ID})
	if err != nil {
		return p, err
	}
	p.Usual = median(sizes)
	if settings.OverUsual && p.Usual > 0 && audience > 2*p.Usual {
		p.Reasons = append(p.Reasons, reasonOverUsual)
	}
	p.Auto = settings.Enabled && len(p.Reasons) > 0 && audience >= minCanaryAudience
	return p, nil
}

// canaryIPs finds the sending IPs of a send's routes that never sent for
// the workspace (review GS-10): a new IP has no reputation yet, as a new
// domain hasn't. Only a dedicated provider's IPs are the workspace's own:
// the ones its provider lists (Providers shows them), recorded as mail
// goes out (workspace_sending_ips). A shared pool's IPs are the
// provider's, kept warm by everyone's mail, and change from message to
// message, so they aren't checked. The routes are the send's split as
// Send Readiness last planned it.
func canaryIPs(ctx context.Context, q *store.Queries, cc store.SendingCampaignContextRow, p *CanaryPlan) error {
	routes, err := q.SendingRouteIPs(ctx, cc.ID)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.IpMode != "dedicated" || len(r.Ips) == 0 {
			continue
		}
		short := state.Short(r.Name, r.Adapter)
		known, err := q.SendingKnownIPs(ctx, store.SendingKnownIPsParams{WorkspaceID: cc.WorkspaceID, Ips: r.Ips})
		if err != nil {
			return err
		}
		for _, ip := range r.Ips {
			if !slices.Contains(known, ip) {
				p.NewIPs = append(p.NewIPs, CanaryIP{IP: ip, Provider: short})
			}
		}
	}
	if len(p.NewIPs) > 0 {
		p.Reasons = append(p.Reasons, reasonNewIP)
	}
	return nil
}

// noteIPs records the dedicated IPs a route sent from (once per run), so
// that only an IP a workspace never sent from starts a canary (canaryIPs).
func (e *Engine) noteIPs(ctx context.Context, r *runRoute, now time.Time) {
	if r.ipsNoted {
		return
	}
	r.ipsNoted = true
	p := r.provider
	if p.IpMode != "dedicated" || len(p.Ips) == 0 {
		return
	}
	id := p.ID
	if err := e.Q.SendingNoteIPs(ctx, store.SendingNoteIPsParams{WorkspaceID: p.WorkspaceID, ProviderID: &id, At: now, Ips: p.Ips}); err != nil {
		e.log().WarnContext(ctx, "note sending IPs", "provider", p.ID, "err", err)
	}
}

// startCanary decides, once the send's audience is built, whether a canary
// runs and which recipients it takes: the most engaged first. The others
// wait (held, hold_reason 'canary'). It runs in the transaction that ends
// the audience build; seeds start after it (startSeeds).
func (e *Engine) startCanary(ctx context.Context, q *store.Queries, cc store.SendingCampaignContextRow, snd store.CampaignSend, now time.Time) (bool, error) {
	cs := parseCanary(snd.Canary)
	if cs.Choice == CanaryOff || snd.Recipients == 0 {
		return false, nil
	}
	plan, err := e.canaryPlan(ctx, q, cc, snd.Recipients, true)
	if err != nil {
		return false, err
	}
	if cs.Choice != CanaryOn && !plan.Auto {
		return false, nil
	}
	settings, _, err := rules.Load(ctx, q, cc.WorkspaceID)
	if err != nil {
		return false, err
	}
	n, err := q.SendingMarkCanary(ctx, store.SendingMarkCanaryParams{CampaignID: &cc.ID, N: int32(max(plan.Size, 1))})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if _, err := q.SendingHoldForCanary(ctx, &cc.ID); err != nil {
		return false, err
	}
	cs.Phase, cs.Reasons, cs.Percent, cs.Size, cs.Started = canarySending, plan.Reasons, plan.Percent, n, &now
	if cs.Choice == CanaryOn {
		// Chosen in Review & send: the reasons are what Traffic rules
		// would have said.
		cs.Reasons = plan.Reasons
	}
	cs.NeverMailed, cs.Usual, cs.Domain, cs.Gates, cs.Watch = plan.NeverMailed, plan.Usual, plan.Domain, settings.Gates, settings.WatchMinutes
	cs.NewIPs = plan.NewIPs
	if settings.Seeds {
		cs.Seeds = plan.Seeds
	}
	return true, saveCanary(ctx, q, cc.ID, cs, now)
}

// startSeeds sends the canary's copies to the seed mailboxes. Without seeds
// (none connected, or the test can't start) the seed gate is left out.
func (e *Engine) startSeeds(ctx context.Context, ws, campaignID uuid.UUID) {
	if e.Seeds == nil {
		return
	}
	snd, err := e.Q.SendingGetSend(ctx, campaignID)
	if err != nil {
		return
	}
	cs := parseCanary(snd.Canary)
	if cs.Phase != canarySending || cs.Seeds == 0 || cs.SeedCheck != "" {
		return
	}
	check, seeds, err := e.Seeds.StartSeeds(ctx, ws, campaignID)
	if err != nil {
		e.log().WarnContext(ctx, "canary seed mailboxes", "campaign", campaignID, "err", err)
		cs.Seeds = 0
	} else {
		cs.SeedCheck, cs.Seeds = check.String(), seeds
	}
	if err := saveCanary(ctx, e.Q, campaignID, cs, e.now()); err != nil {
		e.log().WarnContext(ctx, "save canary", "campaign", campaignID, "err", err)
	}
}

func saveCanary(ctx context.Context, q *store.Queries, campaignID uuid.UUID, cs canaryState, now time.Time) error {
	b, _ := json.Marshal(cs)
	return q.SendingSetCanary(ctx, store.SendingSetCanaryParams{CampaignID: campaignID, Canary: b, At: now})
}

// CanaryGate is one gate's state.
type CanaryGate struct {
	Key    string   `json:"key" enum:"bounce,block_bounce,deferral,complaint,seeds"`
	Value  *float64 `json:"value,omitempty"`
	Limit  float64  `json:"limit"`
	Events int64    `json:"events"`
	Base   int64    `json:"base"`
	// MinBase is the base a gate needs before it can fail.
	MinBase int64  `json:"minBase"`
	State   string `json:"state" enum:"passing,failing,collecting"`
	// Mailbox is the mailbox provider a per-mailbox-provider gate shows.
	Mailbox string `json:"mailbox,omitempty"`
	// Reported and NotReported are the mailbox providers whose complaints
	// count, and the ones that don't report them (the complaint gate).
	Reported    []string `json:"reported,omitempty"`
	NotReported []string `json:"notReported,omitempty"`
	// Seeds is the seed placement (the seeds gate).
	Seeds *SeedPlacement `json:"seeds,omitempty"`
}

// canaryGates judges the canary on what it did so far.
func canaryGates(cs canaryState, ts tallies, ty typed) []CanaryGate {
	g := cs.Gates
	t := ts.total
	gate := func(key string, events, base int64, limit float64, minBase int64) CanaryGate {
		out := CanaryGate{Key: key, Limit: limit, Events: events, Base: base, MinBase: minBase, State: "passing"}
		if base > 0 {
			v := round2(rate(events, base))
			out.Value = &v
		}
		switch {
		case base >= minBase && over(events, base, limit, minEvents):
			out.State = "failing"
		case base < minBase && minBase > 1:
			out.State = "collecting"
		case base > 0 && rate(events, base) >= limit:
			// Over the limit without enough evidence yet.
			out.State = "collecting"
		}
		return out
	}
	var block int64
	for _, n := range ty.block {
		block += n
	}
	out := []CanaryGate{
		gate("bounce", t.bounced(), t.sent, g.Bounce, 1),
		gate("block_bounce", block, t.sent, g.BlockBounce, 1),
		gate("deferral", t.deferred, t.sent, g.Deferral, 1),
	}
	// Complaints per mailbox provider where they're reported: the one
	// closest to the limit; each needs 500 delivered to fail.
	worst := CanaryGate{Key: "complaint", Limit: g.Complaint, MinBase: minSampleScoped, State: "collecting"}
	ratio := -1.0
	var reported, unreported []string
	for _, k := range routing.MailboxProviders {
		m := ts.mailbox[k]
		if m == nil || m.delivered == 0 {
			continue
		}
		if !mailbox.ReportsComplaints(k) {
			unreported = append(unreported, k)
			continue
		}
		reported = append(reported, k)
		c := gate("complaint", m.complained, m.reported, g.Complaint, minSampleScoped)
		c.Mailbox = k
		r := 0.0
		if c.Value != nil && g.Complaint > 0 {
			r = *c.Value / g.Complaint
		}
		// The failing one, else the one closest to the limit, else the
		// one closest to having the evidence.
		if c.State == "failing" && worst.State != "failing" || worst.State != "failing" && (r > ratio || r == ratio && c.Base > worst.Base) {
			worst, ratio = c, r
		}
	}
	worst.Reported, worst.NotReported = reported, unreported
	out = append(out, worst)
	if cs.SeedCheck != "" {
		sg := CanaryGate{Key: "seeds", Limit: float64(g.SeedSpam), Base: int64(cs.Seeds), State: "collecting"}
		if p := cs.SeedResult; p != nil {
			sg.Seeds, sg.Events = p, int64(p.Spam)
			switch {
			case p.Spam > g.SeedSpam:
				sg.State = "failing"
			case p.Pending == 0 && p.Inbox+p.Promotions+p.Spam > 0:
				// Every copy found or given up on, and some arrived. When
				// none did, placement says nothing: it keeps collecting.
				sg.State = "passing"
			}
		}
		out = append(out, sg)
	}
	return out
}

// checkCanary moves the canary along: once its last message went, the
// watch starts; when the watch ends the gates decide. It returns how long
// the loop can wait, and whether the send was held for a decision.
func (e *Engine) checkCanary(ctx context.Context, rs *runState, ts tallies, now time.Time) (time.Duration, bool, error) {
	cs := parseCanary(rs.send.Canary)
	if cs.Phase != canarySending && cs.Phase != canaryWatching {
		return 0, false, nil
	}
	// The seed placement, at most every 30 s.
	if cs.SeedCheck != "" && e.Seeds != nil && (cs.SeedAt == nil || now.Sub(*cs.SeedAt) >= seedEvery) {
		if id, err := uuid.Parse(cs.SeedCheck); err == nil {
			if p, err := e.Seeds.SeedPlacement(ctx, rs.campaign.WorkspaceID, id); err == nil {
				cs.SeedResult = &p
			} else {
				e.log().WarnContext(ctx, "canary seed placement", "campaign", rs.campaign.ID, "err", err)
			}
		}
		cs.SeedAt = &now
		if err := e.keepCanary(ctx, rs, cs, now); err != nil {
			return 0, false, err
		}
	}
	var queued int64
	for _, c := range rs.queue {
		queued += c.queued
	}
	if cs.Phase == canarySending {
		if queued > 0 {
			return 0, false, nil
		}
		until := now.Add(time.Duration(max(cs.Watch, 1))*time.Minute + time.Duration(cs.Extended)*time.Minute)
		cs.Phase, cs.SentAt, cs.Until = canaryWatching, &now, &until
		if err := e.keepCanary(ctx, rs, cs, now); err != nil {
			return 0, false, err
		}
		return min(until.Sub(now), 30*time.Second), false, nil
	}
	if cs.Until != nil && now.Before(*cs.Until) {
		return min(cs.Until.Sub(now), 30*time.Second), false, nil
	}
	gates := canaryGates(cs, ts, rs.typed)
	for _, g := range gates {
		if g.State == "failing" {
			return 0, true, e.holdCanary(ctx, rs, cs, gates, now)
		}
	}
	if err := e.releaseCanary(ctx, rs.campaign.ID, uuid.Nil, canaryPassed, now); err != nil {
		return 0, false, err
	}
	// The loop's copy follows, so the next check doesn't release it again.
	cs.Phase = canaryPassed
	rs.send.Canary, _ = json.Marshal(cs)
	return 0, false, nil
}

// keepCanary saves the canary's state and keeps the loop's copy.
func (e *Engine) keepCanary(ctx context.Context, rs *runState, cs canaryState, now time.Time) error {
	if err := saveCanary(ctx, e.Q, rs.campaign.ID, cs, now); err != nil {
		return err
	}
	rs.send.Canary, _ = json.Marshal(cs)
	return nil
}

// releaseCanary sends everyone the canary held: it passed, or a person
// continued early. The A/B test starts its clock with the main send.
func (e *Engine) releaseCanary(ctx context.Context, campaignID, by uuid.UUID, phase string, now time.Time) error {
	var wsID uuid.UUID
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		wsID = c.WorkspaceID
		snd, err := q.SendingGetSendForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return stateErr("state.noCanary", campaignArgs(c.Name, c.Status))
		}
		if err != nil {
			return err
		}
		cs := parseCanary(snd.Canary)
		if !cs.active() {
			if by == uuid.Nil {
				// A person decided since the loop last read the canary.
				return nil
			}
			return stateErr("state.noCanary", campaignArgs(c.Name, c.Status))
		}
		ab := parseAB(snd.Ab)
		holdouts := ab.Phase == "testing" || ab.Phase == "decided"
		cid := c.ID
		n, err := q.SendingReleaseCanary(ctx, store.SendingReleaseCanaryParams{CampaignID: &cid, AbHoldouts: holdouts})
		if err != nil {
			return err
		}
		switch {
		case ab.Phase == "decided" && ab.Winner != "":
			// The winner was picked during the canary: its holdouts get it,
			// as the A/B test releases them (a send-time test keeps each
			// recipient's time).
			cc, err := q.SendingCampaignContext(ctx, c.ID)
			if err != nil {
				return err
			}
			if _, err := e.releaseHeld(ctx, q, cc, ab.Winner, now); err != nil {
				return err
			}
		case ab.Phase == "testing" && ab.DecideAt != nil && cs.Started != nil:
			at := ab.DecideAt.Add(now.Sub(*cs.Started))
			ab.DecideAt = &at
			b, _ := json.Marshal(ab)
			if err := q.SendingSetAB(ctx, store.SendingSetABParams{CampaignID: c.ID, Ab: b, At: now}); err != nil {
				return err
			}
		}
		ref, name := e.actorRef(ctx, q, by)
		if by == uuid.Nil {
			ref, name = nil, ""
		}
		cs.Phase, cs.Decided, cs.DecidedBy = phase, &now, name
		if err := saveCanary(ctx, q, c.ID, cs, now); err != nil {
			return err
		}
		if cs.Incident != "" {
			if iid, err := uuid.Parse(cs.Incident); err == nil {
				if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: iid, At: &now, Resolution: "continued", ResolvedBy: ref}); err != nil {
					return err
				}
			}
		}
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		actor := e.actorFor(ctx, ref, name)
		if by == uuid.Nil {
			actor = e.systemFor(ctx, c.ID)
		}
		action := map[string]string{canaryPassed: "campaign.canary_passed", canaryContinued: "campaign.canary_continued"}[phase]
		summary := fmt.Sprintf("Canary passed · %s recipients released", countText(n))
		if phase == canaryContinued {
			summary = fmt.Sprintf("Canary continued early · %s recipients released", countText(n))
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: actor, Metadata: map[string]any{"released": n, "canary": cs.Size, "summary": summary}})
	})
	if err != nil {
		return err
	}
	_ = wsID
	e.kick(ctx, campaignID)
	return nil
}

// holdCanary holds the send when a gate failed: the people who can send
// are asked to continue or stop.
func (e *Engine) holdCanary(ctx context.Context, rs *runState, cs canaryState, gates []CanaryGate, now time.Time) error {
	c := rs.campaign
	var held int64
	for _, cell := range rs.queue {
		held += cell.canary
	}
	var failing []string
	for _, g := range gates {
		if g.State == "failing" {
			failing = append(failing, g.Key)
		}
	}
	gate := firstNonEmpty(append(failing, "bounce")...)
	words := i18n.Args{"name": quoted(c.Name), "held": held, "gate": gate, "canary": cs.Size, "mailbox": "", "events": int64(0), "base": int64(0),
		"valuePct": 0.0, "limitPct": 0.0, "spam": int64(0), "seeds": int64(0), "allowed": int64(0)}
	for _, g := range gates {
		if g.Key != gate {
			continue
		}
		if g.Key == "seeds" {
			words["spam"], words["seeds"], words["allowed"] = g.Events, g.Base, int64(g.Limit)
			continue
		}
		words["events"], words["base"], words["limitPct"], words["mailbox"] = g.Events, g.Base, g.Limit, g.Mailbox
		if g.Value != nil {
			words["valuePct"] = *g.Value
		}
	}
	d := IncidentDetail{Events: held}
	d.addCause(keep("cause.canaryLead", words), keep("cause.canary", words))
	d.addRecommendation(keep("rec.canary", words))
	d.setOptions([]option{
		{key: "canary_stop", recommended: true, title: keep("opt.canaryStopTitle", nil), body: keep("opt.canaryStop", words), action: keep("opt.canaryStopAction", nil)},
		{key: "canary_continue", title: keep("opt.canaryContinueTitle", nil), body: keep("opt.canaryContinue", words), action: keep("opt.canaryContinueAction", nil)},
	})
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id := newID()
		var value *float64
		var limit float64
		for _, g := range gates {
			if g.Key == gate {
				value, limit = g.Value, g.Limit
			}
		}
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "canary", Scope: "campaign", Metric: gate,
			Value: value, Threshold: &limit, Held: held, Detail: d.stored(), TrippedAt: now}); err != nil {
			return err
		}
		cs.Phase, cs.Incident = canaryHeld, id.String()
		if err := saveCanary(ctx, q, c.ID, cs, now); err != nil {
			return err
		}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindSendPaused, Severity: "bad", TitleText: keep("notify.canaryTitle", words), BodyText: keep("notify.canaryBody", words),
			Link: liveLink(rs.campaign.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String()},
		}); err != nil {
			return err
		}
		if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "paused", At: now}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.canary_held", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: e.systemFor(ctx, c.ID), Metadata: map[string]any{"gates": failing, "held": held,
				"summary": "Canary gate failed (" + strings.Join(failing, ", ") + ") · " + countText(held) + " recipients held"}})
	})
}

// ContinueCanary sends everyone now, before the watch ends or after a
// gate failed (people who can approve sends).
func (e *Engine) ContinueCanary(ctx context.Context, campaignID, by uuid.UUID) error {
	return e.releaseCanary(ctx, campaignID, by, canaryContinued, e.now())
}

// ExtendCanary watches the canary 10 more minutes.
func (e *Engine) ExtendCanary(ctx context.Context, campaignID, by uuid.UUID) error {
	now := e.now()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		snd, err := q.SendingGetSendForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return stateErr("state.noCanaryWatch", campaignArgs(c.Name, c.Status))
		}
		if err != nil {
			return err
		}
		cs := parseCanary(snd.Canary)
		if cs.Phase != canaryWatching && cs.Phase != canarySending {
			return stateErr("state.noCanaryWatch", campaignArgs(c.Name, c.Status))
		}
		cs.Extended += int(canaryExtend / time.Minute)
		if cs.Phase == canaryWatching && cs.Until != nil {
			base := *cs.Until
			if base.Before(now) {
				base = now
			}
			until := base.Add(canaryExtend)
			cs.Until = &until
		}
		if err := saveCanary(ctx, q, c.ID, cs, now); err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		meta := map[string]any{"extended": cs.Extended, "summary": "Canary watched 10 more minutes"}
		if cs.Until != nil {
			meta["until"] = cs.Until.UTC()
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.canary_extended", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: e.actorFor(ctx, ref, name), Metadata: meta})
	})
}

// StopCanary stops the send: the recipients the canary held don't get it;
// the canary's results stay in the report.
func (e *Engine) StopCanary(ctx context.Context, campaignID, by uuid.UUID) error {
	now := e.now()
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		snd, err := q.SendingGetSendForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return stateErr("state.noCanary", campaignArgs(c.Name, c.Status))
		}
		if err != nil {
			return err
		}
		cs := parseCanary(snd.Canary)
		if !cs.active() {
			return stateErr("state.noCanary", campaignArgs(c.Name, c.Status))
		}
		ref, name := e.actorRef(ctx, q, by)
		cs.Phase, cs.Decided, cs.DecidedBy = canaryStopped, &now, name
		if err := saveCanary(ctx, q, c.ID, cs, now); err != nil {
			return err
		}
		if cs.Incident != "" {
			if iid, err := uuid.Parse(cs.Incident); err == nil {
				if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: iid, At: &now, Resolution: "stopped", ResolvedBy: ref}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// The rest of the send is cancelled as a person's cancel is, audited.
	return e.Cancel(ctx, campaignID, by)
}
