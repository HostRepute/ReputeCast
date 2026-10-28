package engine

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// LiveSnapshot is the live monitor's view of a send (spec 7.6): progress
// and ETA, counts, the per-mailbox-provider and per-sending-provider
// breakdowns, thresholds against their limits, the open incident and the
// newest events.
type LiveSnapshot struct {
	CampaignID string `json:"campaignId"`
	Name       string `json:"name"`
	Status     string `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	// State is what the monitor shows: idle (not started), scheduled,
	// preparing (building the queue), sending, partial (a mailbox provider
	// or route on hold while the rest sends), paused, finished, cancelled
	// or failed.
	State       string     `json:"state" enum:"idle,scheduled,preparing,sending,partial,paused,finished,cancelled,failed"`
	At          time.Time  `json:"at"`
	Version     int64      `json:"version" doc:"Milliseconds since the epoch; the SSE event id"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	PreparedAt  *time.Time `json:"preparedAt,omitempty"`
	StartedBy   string     `json:"startedBy"`
	ApprovedBy  string     `json:"approvedBy"`
	ApprovedAt  *time.Time `json:"approvedAt,omitempty"`
	ScheduledBy string     `json:"scheduledBy"`
	PausedBy    string     `json:"pausedBy"`
	PausedAt    *time.Time `json:"pausedAt,omitempty"`
	PauseReason string     `json:"pauseReason"`
	// Recipients is the audience after exclusions.
	Recipients int64 `json:"recipients"`
	Sent       int64 `json:"sent"`
	// Queued is still to send now; Held waits for a decision (a mailbox
	// provider or route on hold); Waiting waits for the A/B winner; Unsent
	// will never be sent (cancelled, rejected, failed).
	Queued  int64 `json:"queued"`
	Held    int64 `json:"held"`
	Waiting int64 `json:"waiting"`
	// Retrying are the messages a mailbox provider deferred that the
	// sending provider is still retrying (review GS-14).
	Retrying       int64           `json:"retrying"`
	Unsent         int64           `json:"unsent"`
	Percent        float64         `json:"percent"`
	Rate           float64         `json:"rate" doc:"Messages per second now"`
	ETASeconds     *int64          `json:"etaSeconds,omitempty"`
	FinishesAt     *time.Time      `json:"finishesAt,omitempty"`
	Totals         LiveTotals      `json:"totals"`
	Mailboxes      []LiveMailbox   `json:"mailboxes"`
	Providers      []LiveProvider  `json:"providers"`
	Thresholds     []LiveThreshold `json:"thresholds"`
	ThresholdScope string          `json:"thresholdScope" enum:"route,campaign"`
	// Limits are the send's thresholds, as "Edit for this send" changes
	// them (review GS-2).
	Limits LiveLimits `json:"limits"`
	// Watch is the complaint watch after the send (review GS-6).
	Watch *LiveWatch `json:"watch,omitempty"`
	// DeferralWindow is the minutes the mailbox providers' deferral rates
	// count (the first deferral level's window; 0: none).
	DeferralWindow int `json:"deferralWindow,omitempty"`
	// Canary is the send's canary (review GS-10), when it has one.
	Canary *LiveCanary `json:"canary,omitempty"`
	// Postmaster is Gmail's latest daily spam rate for the sending domain.
	Postmaster   *LivePostmaster `json:"postmaster,omitempty"`
	Details      LiveDetails     `json:"details"`
	DeliveredMix []LiveShare     `json:"deliveredMix"`
	AB           *LiveAB         `json:"ab,omitempty"`
	Incident     *LiveIncident   `json:"incident,omitempty"`
	Incidents    []LiveIncident  `json:"incidents"`
	Events       []LiveEvent     `json:"events"`
	EventsVia    string          `json:"eventsVia"`
	LastEventAt  *time.Time      `json:"lastEventAt,omitempty"`
	Failovers    []LiveFailover  `json:"failovers"`
	// SharedStreams are the other streams (Automations, System) that route
	// to a sending provider whose share is held.
	SharedStreams  []string         `json:"sharedStreams"`
	Excluded       LiveExcluded     `json:"excluded"`
	Error          string           `json:"error"`
	Resumed        bool             `json:"resumed"`
	Duration       *int64           `json:"durationSeconds,omitempty"`
	Cost           LiveCostByRoute  `json:"cost"`
	ProvidersHeld  []string         `json:"providersHeld"`
	MailboxesHeld  []string         `json:"mailboxesHeld"`
	Continuing     []string         `json:"continuing"`
	PauseScopeHint string           `json:"pauseScopeHint"`
	Throttles      []LiveThrottle   `json:"throttles"`
	Options        LiveOptions      `json:"options"`
	Routes         []LiveRouteShare `json:"routes"`
	// InDoubt are the messages nobody knows whether their provider took
	// (review GS-1): they wait for a person, never sent again on a guess.
	InDoubt *LiveInDoubt `json:"inDoubt,omitempty"`
	// Helpers names who can pause and who can resume, for the reason on a
	// disabled button. Only the GET fills it; the stream leaves it out.
	Helpers *LiveHelpers `json:"helpers,omitempty"`

	// text is what In needs to write the snapshot in another language.
	text *liveText
}

// LiveInDoubt is what is in doubt, per sending provider.
type LiveInDoubt struct {
	Count     int64          `json:"count"`
	Since     time.Time      `json:"since"`
	Providers []InDoubtShare `json:"providers"`
	// Safe: every provider involved still recognizes the messages it took
	// (inside its idempotency window), so sending them again now can't
	// deliver anything twice.
	Safe bool `json:"safe"`
	// Waiting: nothing else is left to send, so the send waits for this
	// decision (resuming can't send anything; the decision is what goes
	// on).
	Waiting bool `json:"waiting"`
	// IncidentID is the open in-doubt incident with the ways to continue.
	IncidentID string `json:"incidentId,omitempty"`
}

// LiveHelpers are up to two members who hold each action's permission.
type LiveHelpers struct {
	Pause []string `json:"pause"`
	Send  []string `json:"send"`
	// Approve can continue a canary early (review GS-10).
	Approve []string `json:"approve"`
}

// LiveTotals are the campaign's counts.
type LiveTotals struct {
	Delivered   int64 `json:"delivered"`
	Bounced     int64 `json:"bounced"`
	HardBounced int64 `json:"hardBounced"`
	SoftBounced int64 `json:"softBounced"`
	// BlockBounced and UnknownBounced split the hard bounces: block/policy
	// rejections, and unknown users with the rest (review GS-11).
	BlockBounced   int64 `json:"blockBounced"`
	UnknownBounced int64 `json:"unknownBounced"`
	Complained     int64 `json:"complained"`
	Deferred       int64 `json:"deferred"`
	Rejected       int64 `json:"rejected"`
	Opened         int64 `json:"opened"`
	OpenedHuman    int64 `json:"openedHuman"`
	OpenedMachine  int64 `json:"openedMachine"`
	Clicked        int64 `json:"clicked"`
	Unsubscribed   int64 `json:"unsubscribed"`
	// ComplaintBase is what was delivered where mailbox providers report
	// complaints: the complaint rate's base (review F-2).
	ComplaintBase int64 `json:"complaintBase"`
	// NotReported names the mailbox providers this send reached that
	// don't report complaints ("Gmail", "iCloud").
	NotReported []string `json:"notReported"`
}

// LiveMailbox is one mailbox provider's row.
type LiveMailbox struct {
	Key           string   `json:"key" enum:"gmail,outlook,yahoo,icloud,other"`
	Name          string   `json:"name"`
	Sent          int64    `json:"sent"`
	Remaining     int64    `json:"remaining"`
	Delivered     int64    `json:"delivered"`
	Bounced       int64    `json:"bounced"`
	Complained    int64    `json:"complained"`
	Opened        int64    `json:"opened"`
	DeliveredRate *float64 `json:"deliveredRate,omitempty"`
	BounceRate    *float64 `json:"bounceRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	// ComplaintReason says why ComplaintRate is null: Gmail and iCloud
	// don't report complaints per message (review F-2).
	ComplaintReason string   `json:"complaintReason,omitempty" enum:"not_reported"`
	OpenRate        *float64 `json:"openRate,omitempty"`
	// DeferredRate is the share of what went to it within the deferral
	// window that it asked to retry (review GS-14), in percent.
	DeferredRate *float64 `json:"deferredRate,omitempty"`
	// Deferred are its messages deferred once or more; Retrying those still
	// being retried (the canary's table; review GS-14).
	Deferred int64    `json:"deferred"`
	Retrying int64    `json:"retrying"`
	Throttle *float64 `json:"throttle,omitempty" doc:"Messages per second; absent = the provider's full rate"`
	// DoneAt is when its mail still to send has gone, at current rates
	// with its limit (review GS-8); absent while it waits.
	DoneAt    *time.Time `json:"doneAt,omitempty"`
	Held      bool       `json:"held"`
	HeldSince *time.Time `json:"heldSince,omitempty"`
	// HeldUntil is when a timed hold ends by itself (a reply rule's).
	HeldUntil *time.Time `json:"heldUntil,omitempty"`
	Status    string     `json:"status" enum:"ok,sending,paused,done"`
}

// LiveProvider is one sending provider's row.
type LiveProvider struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Short         string     `json:"short"`
	Adapter       string     `json:"adapter"`
	Color         string     `json:"color" enum:"ses,postal,postmark,other"`
	Sent          int64      `json:"sent"`
	Planned       int64      `json:"planned"`
	Remaining     int64      `json:"remaining"`
	Share         float64    `json:"share"`
	Rate          float64    `json:"rate"`
	DeliveredRate *float64   `json:"deliveredRate,omitempty"`
	BounceRate    *float64   `json:"bounceRate,omitempty"`
	ComplaintRate *float64   `json:"complaintRate,omitempty"`
	APIErrorRate  *float64   `json:"apiErrorRate,omitempty"`
	Status        string     `json:"status"`
	RouteStatus   string     `json:"routeStatus" enum:"pending,sending,held,paused,done,cancelled"`
	DownSince     *time.Time `json:"downSince,omitempty"`
	Host          string     `json:"host"`
	IP            string     `json:"ip"`
	QuotaFree     *int64     `json:"quotaFree,omitempty"`
	QuotaMax      *int64     `json:"quotaMax,omitempty"`
	WarmupDay     *int       `json:"warmupDay,omitempty"`
	WarmupDays    *int       `json:"warmupDays,omitempty"`
	WarmupCap     *int64     `json:"warmupCap,omitempty"`
	WarmupSent    *int64     `json:"warmupSent,omitempty"`
	// WarmupHold is set while today's warm-up cap (or a paused plan) holds
	// the route's share (review PR-8).
	WarmupHold  *LiveWarmupHold `json:"warmupHold,omitempty"`
	Cost        float64         `json:"cost"`
	PlannedCost float64         `json:"plannedCost"`
	Retrying    bool            `json:"retrying"`
	RetryUntil  *time.Time      `json:"retryUntil,omitempty"`
	// Probing: the share is held for an outage nobody decided on yet and
	// tests its provider on its own (review GS-3); a person's decision
	// still wins.
	Probing bool `json:"probing"`
	// While a share is retried or probing: the test messages that went
	// through in a row (it resumes at ProbesNeeded), when the next one goes
	// and whether they go to a seed mailbox.
	Probes       int        `json:"probes"`
	ProbesNeeded int        `json:"probesNeeded"`
	NextProbe    *time.Time `json:"nextProbe,omitempty"`
	ProbeSeed    bool       `json:"probeSeed"`
	// InfersDelivery: the provider reports no deliveries (plain SMTP), so
	// its delivered count is inferred from the server's acceptance (review
	// GS-7: "delivered (inferred)").
	InfersDelivery bool `json:"infersDelivery"`
	// Idempotent: the provider recognizes a message it already took, so a
	// retry after a lost answer never delivers it twice (review GS-1), for
	// IdempotencySeconds after the first try.
	Idempotent         bool  `json:"idempotent"`
	IdempotencySeconds int64 `json:"idempotencySeconds,omitempty"`
	// DoneAt is when its share is done at current rates, including the
	// per-mailbox limits (review GS-8: "Done about"); absent while it
	// waits.
	DoneAt *time.Time `json:"doneAt,omitempty"`
}

// LiveLimits are a send's auto-pause thresholds, in percent.
type LiveLimits struct {
	Bounce           float64 `json:"bounce"`
	Complaint        float64 `json:"complaint"`
	MailboxComplaint float64 `json:"mailboxComplaint"`
	Deferral         float64 `json:"deferral"`
	APIErrors        float64 `json:"apiErrors"`
}

// LiveWarmupHold is a route's share that today's warm-up cap (or a paused
// plan) holds (review PR-8, warmup_send.go). It goes on by itself with
// tomorrow's cap, or once the plan resumes, unless someone moves it to a
// warm provider or cancels the rest (POST warmup/{planId}/held/{campaignId}:
// waiting and moving need campaigns:send, cancelling campaigns:pause).
type LiveWarmupHold struct {
	PlanID string `json:"planId"`
	// Held is how many recipients wait.
	Held int64 `json:"held"`
	// Since is when the cap stopped the share.
	Since *time.Time `json:"since,omitempty"`
	// Until is when tomorrow's cap starts and the share goes on; absent
	// while a paused plan holds it (Paused).
	Until  *time.Time `json:"until,omitempty"`
	Paused bool       `json:"paused"`
	// Decision is "wait" once someone chose to wait for the cap.
	Decision string `json:"decision,omitempty" enum:"wait"`
	// Cap is today's cap.
	Cap int64 `json:"cap"`
	// Targets are the providers the share could move to, vetted as a move
	// is (state.Vet); those that can take it come first.
	Targets []LiveWarmupTarget `json:"targets"`
}

// LiveWarmupTarget is a provider a held share could move to.
type LiveWarmupTarget struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Short      string `json:"short"`
	Allowed    bool   `json:"allowed"`
	// Reason is why it can't take the share: its first failed check.
	Reason string `json:"reason,omitempty"`

	reason i18n.Text
}

// warmETA is a warming route's day for the ETAs (review PR-8): what
// today's cap leaves, a day's cap and when tomorrow's starts, in the
// workspace's time zone.
type warmETA struct {
	// held: the cap holds the share now; paused: a paused plan does (no
	// estimate until someone resumes it).
	held, paused bool
	left, cap    int64
	next         time.Time
	loc          *time.Location
}

// after is how long until n more messages have gone once today's cap is
// used: they go from tomorrow's cap on, a day's cap at a time (tomorrow's
// is at least today's), at rate.
func (w warmETA) after(n int64, rate float64, now time.Time) float64 {
	start := w.next
	if w.cap > 0 {
		days := 0
		for n > w.cap {
			n -= w.cap
			days++
		}
		if days > 0 {
			loc := w.loc
			if loc == nil {
				loc = time.UTC
			}
			// Calendar days, so a change of clocks doesn't move midnight.
			start = start.In(loc).AddDate(0, 0, days)
		}
	}
	return math.Max(start.Sub(now).Seconds(), 0) + float64(n)/rate
}

// liveWarmupHold is a route's warm-up hold: what waits, until when, and
// where it could move.
func (e *Engine) liveWarmupHold(ctx context.Context, cc store.SendingCampaignContextRow, p store.Provider, pr providerRate, raw string,
	queue map[queueKey]queueCell, loc *time.Location) *LiveWarmupHold {
	var d struct {
		Until    *time.Time `json:"until"`
		Paused   bool       `json:"paused"`
		Decision string     `json:"decision"`
	}
	_ = json.Unmarshal([]byte(raw), &d)
	h := &LiveWarmupHold{Until: d.Until, Paused: d.Paused, Cap: pr.Cap, Targets: []LiveWarmupTarget{}}
	if pr.PlanID != uuid.Nil {
		h.PlanID = pr.PlanID.String()
	}
	if d.Paused {
		h.Until = nil
	}
	if d.Decision == "wait" {
		h.Decision = d.Decision
	}
	for k, c := range queue {
		if k.provider == p.ID {
			h.Held += c.queued
		}
	}
	provs, err := e.Q.ProvListActive(ctx, cc.WorkspaceID)
	if err != nil {
		return h
	}
	sender := senderVerification(ctx, e.Q, cc.WorkspaceID, cc.SenderID)
	for _, o := range provs {
		if o.ID == p.ID {
			continue
		}
		v := e.vetTarget(ctx, e.Q, sender, o, max(h.Held, 1), loc)
		failed, bad := v.Failed()
		// A provider that can't take Broadcasts at all isn't a choice.
		if bad && (failed.Key == state.VetAvailable || failed.Key == state.VetBulkTerms) {
			continue
		}
		short := state.Short(o.Name, o.Adapter)
		t := LiveWarmupTarget{ProviderID: o.ID.String(), Name: o.Name, Short: short, Allowed: !bad}
		if bad {
			t.reason = failed.Keep(short)
			t.Reason = t.reason.In(i18n.Default)
		}
		h.Targets = append(h.Targets, t)
	}
	sort.SliceStable(h.Targets, func(i, j int) bool { return h.Targets[i].Allowed && !h.Targets[j].Allowed })
	return h
}

// In is the hold in a reader's language: why a provider can't take it.
func (h *LiveWarmupHold) In(l i18n.Lang) *LiveWarmupHold {
	if h == nil {
		return nil
	}
	out := *h
	out.Targets = slices.Clone(h.Targets)
	for i := range out.Targets {
		if t := out.Targets[i].reason; !t.IsZero() {
			out.Targets[i].Reason = t.In(l)
		}
	}
	return &out
}

// infersDelivery reports whether a provider's deliveries are inferred
// from its acceptance (plain SMTP).
func infersDelivery(p store.Provider) bool {
	d, ok := providers.Lookup(p.Adapter)
	return ok && d.Capabilities.InfersDelivery
}

// liveETAs sets when each route and mailbox provider is done at current
// rates (review GS-8). A throttled mailbox provider takes its remaining
// mail over its limit; a route takes its own over its rate, and no less
// than the throttled mailbox providers it carries; the others share what
// the throttled ones leave of their routes' rates. A warming route's mail
// past today's cap, and a share the cap holds, go from tomorrow's cap on
// (review PR-8). It returns when the last one is done (0 without an
// estimate).
func liveETAs(s *LiveSnapshot, queue map[queueKey]queueCell, holds map[string]hold, heldProviders map[uuid.UUID]bool, throttles map[string]float64,
	rates map[uuid.UUID]float64, warm map[uuid.UUID]warmETA, now time.Time) time.Duration {
	left := map[string]int64{}
	routeLeft := map[uuid.UUID]int64{}
	cell := map[queueKey]int64{}
	// What a warm-up cap holds until tomorrow's cap.
	waits := map[queueKey]int64{}
	for k, c := range queue {
		n := c.queued - c.waiting
		if n <= 0 || holdBlocks(holds, k.mailbox, k.provider) || warm[k.provider].paused {
			continue
		}
		if heldProviders[k.provider] {
			if warm[k.provider].held {
				waits[k] = n
			}
			continue
		}
		left[k.mailbox] += n
		routeLeft[k.provider] += n
		cell[k] = n
	}
	secs := func(n int64, rate float64) float64 { return float64(n) / rate }
	mbTime := map[string]float64{}
	for k, n := range left {
		if t := throttles[k]; t > 0 {
			mbTime[k] = secs(n, t)
		}
	}
	// Each route's time, and what its unthrottled mail sends at.
	routeTime := map[uuid.UUID]float64{}
	freeTime := map[uuid.UUID]float64{}
	// Routes whose mail runs into tomorrow's warm-up cap.
	nextDay := map[uuid.UUID]bool{}
	for pid, n := range routeLeft {
		rate := rates[pid]
		if rate <= 0 {
			continue
		}
		t := secs(n, rate)
		var throttled float64 // what the throttled mailbox providers take of its rate
		var free int64
		for k, c := range cell {
			if k.provider != pid {
				continue
			}
			if tr := throttles[k.mailbox]; tr > 0 {
				t = math.Max(t, mbTime[k.mailbox])
				throttled += tr * float64(c) / float64(left[k.mailbox])
			} else {
				free += c
			}
		}
		if w, ok := warm[pid]; ok && n > w.left {
			t, nextDay[pid] = math.Max(t, w.after(n-w.left, rate, now)), true
		}
		routeTime[pid] = t
		freeTime[pid] = secs(free, math.Max(rate-throttled, rate*0.1))
	}
	waitLeft := map[uuid.UUID]int64{}
	for k, n := range waits {
		waitLeft[k.provider] += n
	}
	for pid, n := range waitLeft {
		if rate := rates[pid]; rate > 0 {
			routeTime[pid], nextDay[pid] = warm[pid].after(n, rate, now), true
		}
	}
	at := func(sec float64) *time.Time {
		v := now.Add(time.Duration(math.Ceil(sec)) * time.Second)
		return &v
	}
	var latest float64
	for i := range s.Providers {
		pid, err := uuid.Parse(s.Providers[i].ID)
		if t, ok := routeTime[pid]; err == nil && ok {
			s.Providers[i].DoneAt = at(t)
			latest = math.Max(latest, t)
		}
	}
	for i := range s.Mailboxes {
		k := s.Mailboxes[i].Key
		var t float64
		ok := false
		if left[k] > 0 {
			t, ok = mbTime[k]
			if !ok {
				for c := range cell {
					if c.mailbox == k {
						if ft, has := freeTime[c.provider]; has {
							t, ok = math.Max(t, ft), true
						}
					}
				}
			}
		}
		// Its mail on a route that waits for tomorrow's cap goes then.
		for c := range queue {
			rt, has := routeTime[c.provider]
			if c.mailbox != k || !has || !nextDay[c.provider] || (cell[c] == 0 && waits[c] == 0) {
				continue
			}
			t, ok = math.Max(t, rt), true
		}
		if ok {
			s.Mailboxes[i].DoneAt = at(t)
		}
	}
	return time.Duration(math.Ceil(latest)) * time.Second
}

// LiveThreshold is a rate against its auto-pause limit, with the evidence
// behind it (review GS-6): the events and the base it counts, and what the
// guard needs before it can trip. Without data the value is null with a
// reason (F-3).
type LiveThreshold struct {
	Key   string   `json:"key" enum:"bounce,block_bounce,deferral,complaint,mailbox_complaint,reply_rules,api_errors"`
	Value *float64 `json:"value,omitempty"`
	Limit float64  `json:"limit"`
	// Mailbox is the mailbox provider a per-mailbox-provider row shows (the
	// one closest to its limit).
	Mailbox   string `json:"mailbox,omitempty"`
	Events    int64  `json:"events"`
	Base      int64  `json:"base"`
	MinEvents int64  `json:"minEvents"`
	MinBase   int64  `json:"minBase"`
	// State: ok (watching), near (at 90% of the limit or more), collecting
	// (at or near the limit without the evidence to trip yet), tripped, na
	// (nothing to count: see Reason).
	State  string `json:"state" enum:"ok,near,collecting,tripped,na"`
	Reason string `json:"reason,omitempty" enum:"no_data,not_reported"`
	// Window is the minutes a windowed rate counts (deferrals); Action what
	// the threshold does at its limit (pause_mailbox, slow, pause_send,
	// warn).
	Window int    `json:"window,omitempty"`
	Action string `json:"action,omitempty"`
	// Duration is how long a reply rule's action lasts, in minutes (0:
	// until someone resumes).
	Duration int `json:"duration,omitempty"`
	// Parts break the row down: the API error rate per sending provider,
	// the mailbox providers a whole-send complaint rate counts.
	Parts []LiveThresholdPart `json:"parts,omitempty"`
}

// LiveThresholdPart is one part of a threshold's row.
type LiveThresholdPart struct {
	Name  string   `json:"name"`
	Key   string   `json:"key,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// LiveDetails is the "Send details" card.
type LiveDetails struct {
	From          string  `json:"from"`
	Audience      string  `json:"audience"`
	AudienceCount int64   `json:"audienceCount"`
	Route         string  `json:"route"`
	Cost          float64 `json:"cost"`
	CostPlanned   float64 `json:"costPlanned"`
	ScheduledBy   string  `json:"scheduledBy"`
	// WatchHours is how long complaints are watched after the send
	// (Traffic rules; 0: not watched).
	WatchHours int `json:"watchHours"`
}

// LiveShare is one slice of a breakdown.
type LiveShare struct {
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
}

// LiveAB is the A/B test's progress.
type LiveAB struct {
	Kind     string          `json:"kind"`
	Metric   string          `json:"metric"`
	Phase    string          `json:"phase" enum:"testing,decided"`
	Sample   int             `json:"sample"`
	Variants []LiveABVariant `json:"variants"`
	Leader   string          `json:"leader"`
	DecideAt *time.Time      `json:"decideAt,omitempty"`
	Winner   string          `json:"winner"`
}

// LiveABVariant is one variant's result so far.
type LiveABVariant struct {
	Key  string  `json:"key"`
	Rate float64 `json:"rate"`
	Sent int64   `json:"sent"`
}

// LiveIncident is an auto-pause, outage or blocked recheck.
type LiveIncident struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind" enum:"threshold,provider_outage,throttled,recheck_blocked,reply_rule,canary,render_error,complaint_watch,in_doubt"`
	Scope         string         `json:"scope" enum:"campaign,route,mailbox_provider"`
	Mailbox       string         `json:"mailbox"`
	MailboxName   string         `json:"mailboxName"`
	ProviderID    string         `json:"providerId"`
	Provider      string         `json:"provider"`
	ProviderShort string         `json:"providerShort"`
	Metric        string         `json:"metric"`
	Value         *float64       `json:"value,omitempty"`
	Limit         *float64       `json:"limit,omitempty"`
	Held          int64          `json:"held"`
	TrippedAt     time.Time      `json:"trippedAt"`
	ResolvedAt    *time.Time     `json:"resolvedAt,omitempty"`
	Resolution    string         `json:"resolution"`
	Detail        IncidentDetail `json:"detail"`
	// Since is the tripped scope's rate on what was delivered (or sent)
	// after the trip, for the resume dialog (review GS-2).
	Since *IncidentRate `json:"since,omitempty"`
}

// LiveEvent is one line of the live event feed. Addresses are masked.
type LiveEvent struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Label   string    `json:"label"`
	Short   string    `json:"short"`
	Tone    string    `json:"tone" enum:"ok,warn,bad,info,neutral"`
	Address string    `json:"address"`
	Detail  string    `json:"detail"`

	// parts are what the line's words are made of (In).
	parts *eventParts
}

// LiveFailover is another stream that moved off a provider on its own.
type LiveFailover struct {
	Stream string `json:"stream"`
	To     string `json:"to"`
}

// LiveExcluded counts who the audience build left out.
type LiveExcluded struct {
	Suppressed   int64 `json:"suppressed"`
	Unsubscribed int64 `json:"unsubscribed"`
	Bounced      int64 `json:"bounced"`
	Complained   int64 `json:"complained"`
	Paused       int64 `json:"paused"`
	Role         int64 `json:"role"`
	// Duplicates are aliases of another contact of the audience (the same mailbox).
	Duplicates int64 `json:"duplicates"`
	Sunset     int64 `json:"sunset"`
	Excluded   int64 `json:"excluded"`
	Tiers      int64 `json:"tiers"`
}

// LiveCostByRoute is the cost so far and planned.
type LiveCostByRoute struct {
	Total   float64 `json:"total"`
	Planned float64 `json:"planned"`
}

// LiveThrottle is a mailbox provider's rate limit in this send.
type LiveThrottle struct {
	Mailbox string  `json:"mailbox"`
	Rate    float64 `json:"rate"`
}

// LiveOptions are flags the screen needs.
type LiveOptions struct {
	// CanResume: there is something to resume (paused or held).
	CanResume bool `json:"canResume"`
	CanPause  bool `json:"canPause"`
	CanCancel bool `json:"canCancel"`
}

// LiveRouteShare is a provider's planned share (the route line).
type LiveRouteShare struct {
	Short   string  `json:"short"`
	Percent float64 `json:"percent"`
}

func ptrRate(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*100*100) / 100
	return &v
}

// Snapshot builds the live monitor's view of a campaign.
func (e *Engine) Snapshot(ctx context.Context, wsID, campaignID uuid.UUID) (LiveSnapshot, error) {
	now := e.now()
	cc, err := e.Q.SendingCampaignContext(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && cc.WorkspaceID != wsID) {
		return LiveSnapshot{}, ErrNotFound
	}
	if err != nil {
		return LiveSnapshot{}, err
	}
	s := LiveSnapshot{CampaignID: cc.ID.String(), Name: cc.Name, Status: cc.Status, At: now, Version: now.UnixMilli(),
		StartedAt: cc.StartedAt, ScheduledAt: cc.ScheduledAt, CompletedAt: cc.CompletedAt, ThresholdScope: "route",
		Mailboxes: []LiveMailbox{}, Providers: []LiveProvider{}, Thresholds: []LiveThreshold{}, DeliveredMix: []LiveShare{}, Incidents: []LiveIncident{},
		Events: []LiveEvent{}, Failovers: []LiveFailover{}, SharedStreams: []string{}, ProvidersHeld: []string{}, MailboxesHeld: []string{}, Continuing: []string{}, Throttles: []LiveThrottle{},
		Routes: []LiveRouteShare{}, text: &liveText{}}
	loc := state.Location(cc.WorkspaceTimezone)
	snd, sendErr := e.Q.SendingGetSend(ctx, campaignID)
	hasSend := sendErr == nil
	if sendErr != nil && !errors.Is(sendErr, pgx.ErrNoRows) {
		return s, sendErr
	}
	th := ParseThresholds(cc.Thresholds, cc.WorkspaceThresholds)
	holds := map[string]hold{}
	throttles := map[string]float64{}
	content := sendContent{FromName: cc.SenderName, FromEmail: cc.SenderEmail}
	if hasSend {
		_ = json.Unmarshal(snd.Thresholds, &th)
		holds = parseHolds(snd.Holds)
		_ = json.Unmarshal(snd.Throttles, &throttles)
		_ = json.Unmarshal(snd.Content, &content)
		s.Recipients = snd.Recipients
		s.PreparedAt = snd.PreparedAt
		s.Error = snd.Error
		s.PausedAt, s.PausedBy, s.PauseReason = snd.PausedAt, snd.PausedByName, snd.PauseReason
		var ex LiveExcluded
		_ = json.Unmarshal(snd.Excluded, &ex)
		s.Excluded = ex
		s.StartedBy = snd.StartedByName
		names := e.names(ctx, snd.StartedBy, snd.ApprovedBy, snd.ScheduledBy)
		if s.StartedBy == "" && snd.StartedBy != nil {
			s.StartedBy = names[*snd.StartedBy]
		}
		if snd.ApprovedBy != nil {
			s.ApprovedBy, s.ApprovedAt = names[*snd.ApprovedBy], snd.ApprovedAt
		}
		if snd.ScheduledBy != nil {
			s.ScheduledBy = names[*snd.ScheduledBy]
		}
	} else {
		names := e.names(ctx, cc.StartedBy, cc.CreatedBy)
		if cc.StartedBy != nil {
			s.StartedBy = names[*cc.StartedBy]
		}
		if ap, err := e.Q.SendingApproval(ctx, &cc.ID); err == nil {
			s.ApprovedBy, s.ApprovedAt = ap.ApproverName, &ap.ApprovedAt
		}
		if cc.CreatedBy != nil {
			s.ScheduledBy = names[*cc.CreatedBy]
		}
		if cc.AudienceCount != nil {
			s.Recipients = *cc.AudienceCount
		}
	}
	s.ThresholdScope = th.Scope
	s.Limits = LiveLimits{Bounce: th.Bounce, Complaint: th.Complaint, MailboxComplaint: th.MailboxComplaint, Deferral: th.Deferral, APIErrors: th.APIErrors}
	for _, k := range sortedKeys(throttles) {
		s.Throttles = append(s.Throttles, LiveThrottle{Mailbox: k, Rate: throttles[k]})
	}
	switch {
	case cc.Status == "scheduled":
		s.State = "scheduled"
	case cc.Status == "sent":
		s.State = "finished"
	case cc.Status == "cancelled":
		s.State = "cancelled"
	case cc.Status == "failed":
		s.State = "failed"
	case cc.Status == "paused":
		s.State = "paused"
	case cc.Status == "sending" && hasSend && snd.Phase == "preparing":
		s.State = "preparing"
	case cc.Status == "sending":
		s.State = "sending"
	default:
		s.State = "idle"
	}
	rows, err := e.Q.SendingCounters(ctx, campaignID)
	if err != nil {
		return s, err
	}
	ts := tallyRows(rows)
	t := ts.total
	s.Sent = t.sent
	s.Totals = LiveTotals{Delivered: t.delivered, Bounced: t.bounced(), HardBounced: t.hard, SoftBounced: t.soft, Complained: t.complained, Deferred: t.deferred,
		Rejected: t.rejected, Opened: t.human + t.machine, OpenedHuman: t.human, OpenedMachine: t.machine, Clicked: t.clicked, Unsubscribed: t.unsubscribed,
		ComplaintBase: t.reported, NotReported: []string{}}
	for _, k := range routing.MailboxProviders {
		if m := ts.mailbox[k]; m != nil && m.delivered > 0 && !mailbox.ReportsComplaints(k) {
			s.Totals.NotReported = append(s.Totals.NotReported, mailboxName(k))
			s.text.notReported = append(s.text.notReported, k)
		}
	}
	cid := cc.ID
	queue := map[queueKey]queueCell{}
	// The latest a recipient is due (a send at each recipient's best time,
	// a send-time A/B test).
	var lastDue *time.Time
	if hasSend {
		qrows, err := e.Q.SendingQueueState(ctx, store.SendingQueueStateParams{CampaignID: &cid, Now: now})
		if err != nil {
			return s, err
		}
		for _, r := range qrows {
			if r.ProviderID == nil {
				continue
			}
			if r.Waiting > 0 && r.LastAt.After(now) && (lastDue == nil || r.LastAt.After(*lastDue)) {
				t := r.LastAt
				lastDue = &t
			}
			k := queueKey{*r.ProviderID, r.MailboxProvider}
			c := queue[k]
			c.add(r)
			queue[k] = c
		}
		urows, err := e.Q.SendingUnsentCounts(ctx, &cid)
		if err != nil {
			return s, err
		}
		for _, r := range urows {
			s.Unsent += r.N
		}
	}
	routes, err := e.Q.SendingRoutes(ctx, campaignID)
	if err != nil {
		return s, err
	}
	routeStatus := map[uuid.UUID]store.SendingRoutesRow{}
	var plannedTotal int64
	for _, r := range routes {
		routeStatus[r.ProviderID] = r
		plannedTotal += r.Planned
	}
	recent, err := e.Q.SendingSentSince(ctx, store.SendingSentSinceParams{CampaignID: &cid, Since: ptrTime(now.Add(-30 * time.Second))})
	if err != nil {
		return s, err
	}
	rateBy := map[uuid.UUID]float64{}
	for _, r := range recent {
		if r.ProviderID != nil {
			rateBy[*r.ProviderID] = float64(r.N) / 30
		}
	}
	ids := make([]uuid.UUID, 0, len(routes))
	for _, r := range routes {
		ids = append(ids, r.ProviderID)
	}
	wins, _ := e.Q.ProvCallWindowTotals(ctx, store.ProvCallWindowTotalsParams{ProviderIds: ids, Since: now.Add(-3 * time.Minute)})
	apiErr := map[uuid.UUID]*float64{}
	for _, w := range wins {
		apiErr[w.ProviderID] = ptrRate(w.Errors, w.Calls)
	}
	heldProviders := map[uuid.UUID]bool{}
	routeRate := map[uuid.UUID]float64{}
	warm := map[uuid.UUID]warmETA{}
	var apiRates []LiveThresholdPart
	via := []viaPart{}
	routeLabel := []string{}
	for _, r := range routes {
		p, err := e.Q.ProvGetAny(ctx, r.ProviderID)
		if err != nil {
			continue
		}
		pt := ts.provider[p.ID]
		if pt == nil {
			pt = &tally{}
		}
		short := state.Short(p.Name, p.Adapter)
		lp := LiveProvider{ID: p.ID.String(), Name: p.Name, Short: short, Adapter: p.Adapter, Color: state.Color(p.Adapter), Sent: pt.sent, Planned: r.Planned,
			Rate: math.Round(rateBy[p.ID]*10) / 10, DeliveredRate: ptrRate(pt.delivered, pt.sent), BounceRate: ptrRate(pt.bounced(), pt.sent),
			ComplaintRate: ptrRate(pt.complained, pt.reported), APIErrorRate: apiErr[p.ID], Status: p.Status, RouteStatus: r.Status,
			Host: hostOf(p), Cost: float64(r.CostMicros) / 1e6, Idempotent: dedupes(p), IdempotencySeconds: int64(window(p) / time.Second),
			InfersDelivery: infersDelivery(p)}
		if len(p.Ips) > 0 {
			lp.IP = p.Ips[0]
		}
		for k, c := range queue {
			if k.provider == p.ID {
				lp.Remaining += c.queued + c.held
			}
		}
		if plannedTotal > 0 {
			lp.Share = math.Round(float64(r.Planned)/float64(plannedTotal)*1000) / 10
		}
		if pr := state.PricingOf(p); pr.Model == "metered" {
			lp.PlannedCost = math.Round(float64(r.Planned)*pr.Per1000/1000*100) / 100
		}
		h := state.HealthOf(p)
		if h.Down != nil {
			t := h.Down.Since
			lp.DownSince = &t
		}
		d := parseRouteDetail(r.StatusDetail)
		if r.Status == "held" || r.Status == "paused" {
			heldProviders[p.ID] = true
			s.ProvidersHeld = append(s.ProvidersHeld, short)
			if lp.DownSince == nil {
				if in := e.firstFailure(cc.ID, p.ID); in != nil {
					lp.DownSince = in
				}
			}
		}
		switch {
		case r.Status != "held":
		case d.Decision == "retry":
			lp.Retrying, lp.RetryUntil = true, d.Until
			lp.Probes, lp.NextProbe, lp.ProbeSeed = d.Probes, d.NextProbe, d.Seed
		case d.Decision == "" && d.Reason != CapHoldReason && d.NextProbe != nil:
			lp.Probing = true
			lp.Probes, lp.NextProbe, lp.ProbeSeed = d.Probes, d.NextProbe, d.Seed
		}
		lp.ProbesNeeded = probesToResume
		if h.Quota != nil && h.Quota.Max > 0 {
			free := max(h.Quota.Max-h.Quota.Used-lp.Remaining, 0)
			mx := h.Quota.Max
			lp.QuotaFree, lp.QuotaMax = &free, &mx
		}
		pr := e.rateOf(ctx, e.Q, p, loc, now)
		routeRate[p.ID] = rateBy[p.ID]
		if routeRate[p.ID] <= 0 {
			routeRate[p.ID] = pr.PerSecond
		}
		if pr.WarmingUp {
			day, days, cp, sent := pr.WarmupDay, pr.WarmupDays, pr.Cap, pr.SentToday
			lp.WarmupDay, lp.WarmupDays, lp.WarmupCap, lp.WarmupSent = &day, &days, &cp, &sent
			warm[p.ID] = warmETA{paused: pr.Paused, left: max(pr.Cap-pr.SentToday, 0), cap: pr.Cap, next: pr.NextDay, loc: loc}
		}
		// Today's cap (or a paused plan) holds the share (review PR-8): it
		// goes on with tomorrow's cap unless someone moves it.
		if r.Status == "held" && d.Reason == CapHoldReason {
			lp.WarmupHold = e.liveWarmupHold(ctx, cc, p, pr, r.StatusDetail, queue, loc)
			w := warm[p.ID]
			w.held, w.paused, w.loc = true, lp.WarmupHold.Paused, loc
			if lp.WarmupHold.Until != nil {
				w.next = *lp.WarmupHold.Until
			}
			warm[p.ID] = w
		}
		// The API error threshold watches the routes this send still uses (a
		// share moved away or finished doesn't count).
		if r.Status == "pending" || r.Status == "sending" || r.Status == "held" {
			apiRates = append(apiRates, LiveThresholdPart{Name: short, Value: lp.APIErrorRate})
		}
		via = append(via, viaOf(p))
		s.Providers = append(s.Providers, lp)
		s.Details.CostPlanned += lp.PlannedCost
		s.Details.Cost += lp.Cost
		if plannedTotal > 0 {
			s.Routes = append(s.Routes, LiveRouteShare{Short: short, Percent: math.Round(float64(r.Planned) / float64(plannedTotal) * 100)})
			routeLabel = append(routeLabel, short+" "+strconvFloat(math.Round(float64(r.Planned)/float64(plannedTotal)*100))+"%")
		}
		s.Rate += rateBy[p.ID]
	}
	s.Rate = math.Round(s.Rate*10) / 10
	s.Details.Cost = math.Round(s.Details.Cost*100) / 100
	s.Details.CostPlanned = math.Round(s.Details.CostPlanned*100) / 100
	s.Cost = LiveCostByRoute{Total: s.Details.Cost, Planned: s.Details.CostPlanned}
	s.text.via = uniqVia(via)
	viaWords := make([]string, len(s.text.via))
	for i, v := range s.text.via {
		viaWords[i] = v.in(i18n.Default)
	}
	s.EventsVia = joinAnd(viaWords)
	for _, k := range routing.MailboxProviders {
		m := ts.mailbox[k]
		if m == nil {
			m = &tally{}
		}
		lm := LiveMailbox{Key: k, Name: mailboxName(k), Sent: m.sent, Delivered: m.delivered, Bounced: m.bounced(), Complained: m.complained, Opened: m.human + m.machine, Deferred: m.deferred,
			DeliveredRate: ptrRate(m.delivered, m.sent), BounceRate: ptrRate(m.bounced(), m.sent), ComplaintRate: ptrRate(m.complained, m.reported), OpenRate: ptrRate(m.human+m.machine, m.delivered)}
		if !mailbox.ReportsComplaints(k) {
			lm.ComplaintRate, lm.ComplaintReason = nil, mailbox.NotReported
		}
		for qk, c := range queue {
			if qk.mailbox == k {
				lm.Remaining += c.queued + c.held
			}
		}
		if t, ok := throttles[k]; ok && t > 0 {
			v := t
			lm.Throttle = &v
		}
		if h, ok := holds[k]; ok && !h.Resumed {
			lm.Held, lm.Status = true, "paused"
			since := h.Since
			lm.HeldSince, lm.HeldUntil = &since, h.Until
			s.MailboxesHeld = append(s.MailboxesHeld, lm.Name)
			s.text.held = append(s.text.held, k)
		} else if lm.Remaining > 0 && (s.State == "sending" || s.State == "partial") {
			lm.Status = "sending"
		} else if lm.Remaining == 0 && lm.Sent > 0 && s.State != "finished" {
			lm.Status = "done"
		} else {
			lm.Status = "ok"
		}
		if m.sent == 0 && lm.Remaining == 0 {
			continue
		}
		s.Mailboxes = append(s.Mailboxes, lm)
	}
	if t.delivered > 0 {
		for _, k := range routing.MailboxProviders {
			if m := ts.mailbox[k]; m != nil && m.delivered > 0 {
				s.DeliveredMix = append(s.DeliveredMix, LiveShare{Key: k, Name: mailboxName(k), Percent: math.Round(float64(m.delivered)/float64(t.delivered)*1000) / 10})
			}
		}
	}
	var ty typed
	if hasSend {
		if ty, err = readTypedAt(ctx, e.Q, campaignID, th, now); err != nil {
			return s, err
		}
		for _, n := range ty.block {
			s.Totals.BlockBounced += n
		}
		for _, n := range ty.unknown {
			s.Totals.UnknownBounced += n
		}
	}
	s.Thresholds = liveThresholds(th, ts, guardOf(snd, hasSend), apiRates, ty)
	// What the providers are still retrying after a deferral.
	if hasSend {
		cid := campaignID
		rows, err := e.Q.SendingRetrying(ctx, &cid)
		if err != nil {
			return s, err
		}
		for _, r := range rows {
			s.Retrying += r.N
			for i := range s.Mailboxes {
				if s.Mailboxes[i].Key == r.MailboxProvider {
					s.Mailboxes[i].Retrying += r.N
				}
			}
		}
	}
	// Each mailbox provider's deferrals over the first level's window.
	for _, w := range []int{th.DeferralSlowWindow, th.DeferralWindow} {
		if w <= 0 || ty.sent[w] == nil {
			continue
		}
		s.DeferralWindow = w
		for i := range s.Mailboxes {
			if n := ty.sent[w][s.Mailboxes[i].Key]; n > 0 {
				v := round2(rate(ty.deferred[w][s.Mailboxes[i].Key], n))
				s.Mailboxes[i].DeferredRate = &v
			}
		}
		break
	}
	if hasSend {
		if cs := parseCanary(snd.Canary); cs.Phase != "" {
			cid := campaignID
			ct, err := e.Q.SendingCanaryTotals(ctx, &cid)
			if err != nil {
				return s, err
			}
			s.Canary = liveCanary(cs, ts, ty, ct.Sent, queue, now)
			if cs.active() {
				// When everyone has it once the main send starts: what the
				// canary holds, at the routes' and mailbox providers' rates.
				main := map[queueKey]queueCell{}
				for k, c := range queue {
					c.queued += c.canary
					main[k] = c
				}
				scratch := s
				scratch.Providers, scratch.Mailboxes = slices.Clone(s.Providers), slices.Clone(s.Mailboxes)
				s.Canary.MainSeconds = int64(liveETAs(&scratch, main, holds, heldProviders, throttles, routeRate, warm, now) / time.Second)
			}
		}
	}
	if row, ok := e.replyRow(ctx, cc.WorkspaceID, guardOf(snd, hasSend)); ok {
		s.Thresholds = append(s.Thresholds[:len(s.Thresholds)-1], row, s.Thresholds[len(s.Thresholds)-1])
	}
	if hasSend {
		s.Watch = e.liveWatch(ctx, cc.WorkspaceID, snd, ts, th, now)
	}
	// Gmail's spam rate from Postmaster Tools shows only while Traffic rules
	// read it (After the send › Postmaster Tools), as the watch does.
	_, watchRules, watchErr := rules.Load(ctx, e.Q, cc.WorkspaceID)
	if watchErr == nil && watchRules.Postmaster {
		s.Postmaster = e.livePostmaster(ctx, cc.WorkspaceID, cc.SenderEmail, now)
	}
	for k, c := range queue {
		heldMailbox := holdBlocks(holds, k.mailbox, k.provider)
		switch {
		case heldMailbox || heldProviders[k.provider] || s.State == "paused":
			s.Held += c.queued
		default:
			s.Queued += c.queued
		}
		s.Waiting += c.held
	}
	if s.Recipients > 0 {
		s.Percent = math.Round(float64(s.Sent)/float64(s.Recipients)*1000) / 10
	}
	if s.Rate > 0 && (s.Queued > 0 || s.Waiting > 0) && (s.State == "sending" || s.State == "partial") {
		// What sends now, then what waits for the A/B winner: it goes out
		// once the test is decided, at the same rate.
		secs := float64(s.Queued) / s.Rate
		if s.Waiting > 0 && hasSend {
			if ab := parseAB(snd.Ab); ab.Phase == "testing" && ab.DecideAt != nil {
				secs = math.Max(secs, ab.DecideAt.Sub(now).Seconds())
			}
			secs += float64(s.Waiting) / s.Rate
		}
		eta := int64(math.Ceil(secs))
		s.ETASeconds = &eta
		f := now.Add(time.Duration(eta) * time.Second)
		s.FinishesAt = &f
	}
	if s.State == "sending" || s.State == "partial" {
		if latest := liveETAs(&s, queue, holds, heldProviders, throttles, routeRate, warm, now); latest > 0 && (s.FinishesAt == nil || now.Add(latest).After(*s.FinishesAt)) {
			eta := int64(math.Ceil(latest.Seconds()))
			f := now.Add(time.Duration(eta) * time.Second)
			s.ETASeconds, s.FinishesAt = &eta, &f
		}
	}
	if lastDue != nil && (s.State == "sending" || s.State == "partial") && (s.FinishesAt == nil || lastDue.After(*s.FinishesAt)) {
		eta := int64(math.Ceil(lastDue.Sub(now).Seconds()))
		s.ETASeconds, s.FinishesAt = &eta, lastDue
	}
	if s.State == "sending" && (len(s.MailboxesHeld) > 0 || len(s.ProvidersHeld) > 0) {
		s.State = "partial"
	}
	for _, lm := range s.Mailboxes {
		if !lm.Held && lm.Remaining > 0 {
			s.Continuing = append(s.Continuing, lm.Name)
			s.text.continuing = append(s.text.continuing, lm.Key)
		}
	}
	if cc.StartedAt != nil && cc.CompletedAt != nil {
		d := int64(cc.CompletedAt.Sub(*cc.StartedAt).Seconds())
		s.Duration = &d
	}
	s.Details.From = content.FromEmail
	if s.Details.From == "" {
		s.Details.From = cc.SenderEmail
	}
	s.text.audience = e.audienceLabel(ctx, cc.WorkspaceID, cc.Audience)
	s.Details.Audience = s.text.audience.in(i18n.Default)
	s.Details.AudienceCount = s.Recipients
	if watchErr == nil && watchRules.Enabled {
		s.Details.WatchHours = watchRules.Hours
	}
	s.Details.Route = strings.Join(routeLabel, " · ")
	s.Details.ScheduledBy = s.ScheduledBy
	if hasSend {
		ab := parseAB(snd.Ab)
		if ab.Phase != "" && len(ab.Variants) > 0 {
			la := &LiveAB{Kind: ab.Kind, Metric: ab.Metric, Phase: ab.Phase, Sample: ab.TestPercent, DecideAt: ab.DecideAt, Winner: ab.Winner, Variants: []LiveABVariant{}}
			_, rates := abWinner(ab, ts)
			if ab.Phase == "decided" && len(ab.Rates) > 0 {
				// The rates the decision was based on: the sample's opens keep
				// coming in, but the line reports what decided it.
				rates = ab.Rates
			}
			best := -1.0
			for _, v := range ab.Variants {
				vt := ts.variant[v]
				var sent int64
				if vt != nil {
					sent = vt.sent
				}
				la.Variants = append(la.Variants, LiveABVariant{Key: v, Rate: rates[v], Sent: sent})
				if rates[v] > best {
					best, la.Leader = rates[v], v
				}
			}
			if ab.Phase == "decided" && ab.Winner != "" {
				la.Leader = ab.Winner
			}
			s.AB = la
		}
	}
	// Messages in doubt (review GS-1): whether sending them again is safe
	// changes with time, so the incident's ways to continue are worked out
	// again for this snapshot.
	var doubtDetail *IncidentDetail
	if hasSend {
		shares, total, err := e.inDoubtShares(ctx, e.Q, campaignID, now)
		if err != nil {
			return s, err
		}
		if total > 0 {
			d := &LiveInDoubt{Count: total, Providers: shares, Safe: inDoubtSafety(shares) == "yes"}
			for _, sh := range shares {
				if d.Since.IsZero() || sh.Since.Before(d.Since) {
					d.Since = sh.Since
				}
			}
			s.InDoubt = d
			dd := inDoubtDetail(shares, total)
			doubtDetail = &dd
		}
	}
	incs, err := e.Q.SendingIncidents(ctx, campaignID)
	if err != nil {
		return s, err
	}
	provNames := map[uuid.UUID]store.Provider{}
	for _, in := range incs {
		li := LiveIncident{ID: in.ID.String(), Kind: in.Kind, Scope: in.Scope, Mailbox: in.MailboxProvider, Metric: in.Metric, Value: in.Value, Limit: in.Threshold,
			Held: in.Held, TrippedAt: in.TrippedAt, ResolvedAt: in.ResolvedAt, Resolution: in.Resolution}
		if in.MailboxProvider != "" {
			li.MailboxName = mailboxName(in.MailboxProvider)
		}
		if in.ProviderID != nil {
			p, ok := provNames[*in.ProviderID]
			if !ok {
				if p, err = e.Q.ProvGetAny(ctx, *in.ProviderID); err == nil {
					provNames[p.ID] = p
				}
			}
			li.ProviderID, li.Provider, li.ProviderShort = in.ProviderID.String(), p.Name, state.Short(p.Name, p.Adapter)
		}
		li.Detail = parseDetail(in.Detail)
		if in.Kind == "threshold" && in.ResolvedAt == nil {
			li.Since = sinceTrip(ts, in, li.Detail)
		}
		if in.Kind == "in_doubt" && in.ResolvedAt == nil && doubtDetail != nil {
			li.Detail, li.Held = *doubtDetail, s.InDoubt.Count
			s.InDoubt.IncidentID = li.ID
		}
		s.Incidents = append(s.Incidents, li)
		if in.ResolvedAt == nil && s.Incident == nil && (in.Kind == "threshold" || in.Kind == "provider_outage" || in.Kind == "reply_rule" || in.Kind == "canary") && !isNotice(in) {
			c := li
			s.Incident = &c
		}
	}
	// A held route's "down since" and error rate come from its outage
	// incident when this process didn't see the failures itself (another
	// worker sent, or the calls are older than the error window).
	for i := range s.Providers {
		lp := &s.Providers[i]
		if lp.RouteStatus != "held" {
			continue
		}
		for _, in := range s.Incidents {
			if in.ResolvedAt != nil || in.Kind != "provider_outage" || in.ProviderID != lp.ID {
				continue
			}
			if lp.DownSince == nil {
				since := in.TrippedAt
				if in.Detail.From != nil {
					since = *in.Detail.From
				}
				lp.DownSince = &since
			}
			if lp.APIErrorRate == nil && in.Detail.Attempts > 0 {
				v := round2(rate(int64(in.Detail.Attempts-in.Detail.Succeeded), int64(in.Detail.Attempts)))
				lp.APIErrorRate = &v
			}
			break
		}
	}
	// When the warm-up cap stopped a held share: its open incident.
	for i := range s.Providers {
		h := s.Providers[i].WarmupHold
		if h == nil {
			continue
		}
		for _, in := range s.Incidents {
			if in.ResolvedAt == nil && in.Kind == "throttled" && in.Metric == CapHoldReason && in.ProviderID == s.Providers[i].ID {
				since := in.TrippedAt
				h.Since = &since
				break
			}
		}
	}
	// Other streams that use a held provider, and those that failed over
	// on their own.
	if len(heldProviders) > 0 {
		if set, err := routing.Load(ctx, e.Q, cc.WorkspaceID); err == nil {
			for _, st := range []providers.Stream{providers.StreamAutomation, providers.StreamSystem} {
				for _, r := range set.StreamRules(st) {
					uses := r.Fallback != nil && heldProviders[*r.Fallback]
					for _, sp := range r.Splits {
						uses = uses || heldProviders[sp.ProviderID]
					}
					if uses {
						s.SharedStreams = append(s.SharedStreams, streamName(string(st)))
						s.text.shared = append(s.text.shared, string(st))
						break
					}
				}
			}
		}
	}
	if len(heldProviders) > 0 {
		if fos, err := e.Q.ProvActiveFailovers(ctx, cc.WorkspaceID); err == nil {
			for _, f := range fos {
				if !heldProviders[f.ProviderID] || f.Mode != "automatic" || f.Stream == "broadcast" || f.FallbackProviderID == nil {
					continue
				}
				fb, err := e.Q.ProvGetAny(ctx, *f.FallbackProviderID)
				if err != nil {
					continue
				}
				s.Failovers = append(s.Failovers, LiveFailover{Stream: streamName(f.Stream), To: state.Short(fb.Name, fb.Adapter)})
				s.text.failovers = append(s.text.failovers, f.Stream)
			}
		}
	}
	if hasSend || cc.StartedAt != nil {
		feed, err := e.Q.SendingFeed(ctx, store.SendingFeedParams{CampaignID: &cid, Since: now.Add(-72 * time.Hour), Lim: 20})
		if err != nil {
			return s, err
		}
		for _, f := range feed {
			s.Events = append(s.Events, e.liveEvent(f, provNames))
		}
	}
	// "last 1 s ago": this send's newest event (a provider's own last event
	// may belong to another send).
	if len(s.Events) > 0 {
		t := s.Events[0].At
		s.LastEventAt = &t
	}
	// A share the warm-up cap holds has its own choices; resuming the send
	// leaves it waiting for the cap (releaseAll).
	routesHeld := 0
	for _, lp := range s.Providers {
		if (lp.RouteStatus == "held" || lp.RouteStatus == "paused") && lp.WarmupHold == nil {
			routesHeld++
		}
	}
	// Only messages in doubt are left and nobody paused the send: it waits
	// for the decision on them, and resuming couldn't send anything.
	if s.InDoubt != nil && cc.Status == "paused" && s.PausedAt == nil && s.Held == 0 && s.Waiting == 0 && s.Queued == 0 {
		s.InDoubt.Waiting = true
	}
	s.Options = LiveOptions{
		CanPause:  cc.Status == "sending",
		CanResume: (cc.Status == "paused" && (s.InDoubt == nil || !s.InDoubt.Waiting)) || len(s.MailboxesHeld) > 0 || routesHeld > 0,
		CanCancel: cc.Status == "sending" || cc.Status == "paused",
	}
	return s, nil
}

// firstFailure is when the engine first saw a route failing (this process).
func (e *Engine) firstFailure(campaign, provider uuid.UUID) *time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	if l, ok := e.calls[routeKey{campaign, provider}]; ok && !l.firstFailure.IsZero() {
		t := l.firstFailure
		return &t
	}
	return nil
}

func (e *Engine) names(ctx context.Context, ids ...*uuid.UUID) map[uuid.UUID]string {
	var in []uuid.UUID
	for _, id := range ids {
		if id != nil {
			in = append(in, *id)
		}
	}
	out := map[uuid.UUID]string{}
	if len(in) == 0 {
		return out
	}
	rows, err := e.Q.SendingUserNames(ctx, in)
	if err != nil {
		return out
	}
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out
}

// audienceLabel is "Weekly Digest" or "Weekly Digest + 1 segment".
func (e *Engine) audienceLabel(ctx context.Context, ws uuid.UUID, raw json.RawMessage) audienceParts {
	def := parseAudience(raw)
	lists, _ := e.Q.SendingListNames(ctx, ws)
	names := map[uuid.UUID]string{}
	var public []uuid.UUID
	for _, l := range lists {
		names[l.ID] = l.Name
		// A store's list (X20) isn't one readers sign up for either.
		if l.Public && l.Origin != "store" {
			public = append(public, l.ID)
		}
	}
	var out audienceParts
	// "All subscribers": every public list (a private one, such as an
	// event's registrants, isn't one readers sign up for).
	if len(def.Segments) == 0 && len(public) > 1 && !slices.ContainsFunc(public, func(id uuid.UUID) bool { return !slices.Contains(def.Lists, id) }) {
		out.all = true
	} else {
		for _, id := range def.Lists {
			if n, ok := names[id]; ok {
				out.names = append(out.names, n)
			}
		}
	}
	if len(def.Segments) > 0 {
		segs, _ := e.Q.SendingSegmentNames(ctx, ws)
		segNames := map[uuid.UUID]string{}
		for _, sg := range segs {
			segNames[sg.ID] = sg.Name
		}
		for _, id := range def.Segments {
			if n, ok := segNames[id]; ok {
				out.names = append(out.names, n)
			}
		}
	}
	return out
}

// liveEvent turns a delivery event into a feed line.
func (e *Engine) liveEvent(f store.SendingFeedRow, provs map[uuid.UUID]store.Provider) LiveEvent {
	ev := LiveEvent{ID: f.ID.String(), At: f.ReceivedAt, Kind: f.Type, Address: MaskAddress(f.Recipient)}
	short := ""
	if f.ProviderID != nil {
		p, ok := provs[*f.ProviderID]
		if !ok {
			if pp, err := e.Q.ProvGetAny(context.Background(), *f.ProviderID); err == nil {
				p = pp
				provs[pp.ID] = pp
			}
		}
		short = state.Short(p.Name, p.Adapter)
	}
	p := &eventParts{short: short, code: strings.TrimSpace(f.OriginalCode), diag: f.Diagnostic, url: f.Url, vary: f.Variant, suppresses: f.Suppresses}
	switch providers.EventType(f.Type) {
	case providers.EventDelivered:
		p.kind, ev.Tone = "delivered", "ok"
	case providers.EventHardBounce:
		p.kind, ev.Tone = "hardBounce", "bad"
	case providers.EventSoftBounce:
		p.kind, ev.Tone = "softBounce", "warn"
	case providers.EventDeferred:
		p.kind, ev.Tone = "deferred", "warn"
	case providers.EventComplaint:
		p.kind, ev.Tone = "complaint", "bad"
	case providers.EventRejected:
		p.kind, ev.Tone = "rejected", "bad"
	case providers.EventOpen:
		p.kind, ev.Tone = "openedHuman", "info"
		if f.Machine {
			p.kind, ev.Tone = "openedMachine", "neutral"
		}
	case providers.EventClick:
		p.kind, ev.Tone = "clicked", "ok"
		if f.Machine {
			p.kind, ev.Tone = "clickedScanner", "neutral"
		}
	case providers.EventUnsubscribe:
		p.kind, ev.Tone = "unsubscribed", "neutral"
	default:
		p.kind, ev.Tone = "other", "neutral"
	}
	ev.Label, ev.Short = f.Type, f.Type
	if w, ok := eventWords[p.kind]; ok {
		ev.Label, ev.Short = w[0], w[1]
	}
	ev.Detail, ev.parts = p.detail(i18n.Default), p
	return ev
}

// MaskAddress hides most of an address for the live feed:
// "ana.kowalski@gmail.com" → "a.k···@gmail.com".
func MaskAddress(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 {
		return "···"
	}
	local, domain := email[:at], email[at:]
	r := []rune(local)
	out := string(r[0])
	for i := 1; i < len(r)-1; i++ {
		if r[i] == '.' || r[i] == '_' || r[i] == '-' || r[i] == '+' {
			out += string(r[i]) + string(r[i+1])
			break
		}
	}
	return out + "···" + domain
}

// linkText is a URL without its scheme and query ("docs.acme.io/changelog").
func linkText(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return strings.TrimSuffix(u.Host+u.Path, "/")
}

// viaOf is how a provider reports events ("SNS", "Postal webhooks").
func viaOf(p store.Provider) viaPart {
	c := state.Caps(p)
	switch c.Webhooks {
	case "SNS":
		return viaPart{kind: "sns"}
	case "Bounce mailbox":
		return viaPart{kind: "mailbox"}
	}
	return viaPart{kind: "webhooks", name: state.Short(p.Name, p.Adapter)}
}

func streamName(s string) string {
	switch s {
	case "automation":
		return "Automations"
	case "system":
		return "System"
	}
	return "Broadcasts"
}

func joinAnd(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " and " + xs[len(xs)-1]
}

// uniqVia lists each way once, SNS first.
func uniqVia(xs []viaPart) []viaPart {
	seen := map[viaPart]bool{}
	out := []viaPart{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].kind == "sns" && out[j].kind != "sns" })
	return out
}

func ptrTime(t time.Time) *time.Time { return &t }

func strconvFloat(v float64) string { return rateText(v) }
