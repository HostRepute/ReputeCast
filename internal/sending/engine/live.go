package engine

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
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
	Queued     int64      `json:"queued"`
	Held       int64      `json:"held"`
	Waiting    int64      `json:"waiting"`
	Unsent     int64      `json:"unsent"`
	Percent    float64    `json:"percent"`
	Rate       float64    `json:"rate" doc:"Messages per second now"`
	ETASeconds *int64     `json:"etaSeconds,omitempty"`
	FinishesAt *time.Time `json:"finishesAt,omitempty"`
	Totals     LiveTotals `json:"totals"`
	// HeldMailboxes are the mailbox providers on hold, HeldProviders the
	// sending providers.
	Mailboxes      []LiveMailbox   `json:"mailboxes"`
	Providers      []LiveProvider  `json:"providers"`
	Thresholds     []LiveThreshold `json:"thresholds"`
	ThresholdScope string          `json:"thresholdScope" enum:"route,campaign"`
	Details        LiveDetails     `json:"details"`
	DeliveredMix   []LiveShare     `json:"deliveredMix"`
	AB             *LiveAB         `json:"ab,omitempty"`
	Incident       *LiveIncident   `json:"incident,omitempty"`
	Incidents      []LiveIncident  `json:"incidents"`
	Events         []LiveEvent     `json:"events"`
	EventsVia      string          `json:"eventsVia"`
	LastEventAt    *time.Time      `json:"lastEventAt,omitempty"`
	Failovers      []LiveFailover  `json:"failovers"`
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
	// Helpers names who can pause and who can resume, for the reason on a
	// disabled button. Only the GET fills it; the stream leaves it out.
	Helpers *LiveHelpers `json:"helpers,omitempty"`
}

// LiveHelpers are up to two members who hold each action's permission.
type LiveHelpers struct {
	Pause []string `json:"pause"`
	Send  []string `json:"send"`
}

// LiveTotals are the campaign's counts.
type LiveTotals struct {
	Delivered     int64 `json:"delivered"`
	Bounced       int64 `json:"bounced"`
	HardBounced   int64 `json:"hardBounced"`
	SoftBounced   int64 `json:"softBounced"`
	Complained    int64 `json:"complained"`
	Deferred      int64 `json:"deferred"`
	Rejected      int64 `json:"rejected"`
	Opened        int64 `json:"opened"`
	OpenedHuman   int64 `json:"openedHuman"`
	OpenedMachine int64 `json:"openedMachine"`
	Clicked       int64 `json:"clicked"`
	Unsubscribed  int64 `json:"unsubscribed"`
}

// LiveMailbox is one mailbox provider's row.
type LiveMailbox struct {
	Key           string     `json:"key" enum:"gmail,outlook,yahoo,icloud,other"`
	Name          string     `json:"name"`
	Sent          int64      `json:"sent"`
	Remaining     int64      `json:"remaining"`
	Delivered     int64      `json:"delivered"`
	Bounced       int64      `json:"bounced"`
	Complained    int64      `json:"complained"`
	Opened        int64      `json:"opened"`
	DeliveredRate *float64   `json:"deliveredRate,omitempty"`
	BounceRate    *float64   `json:"bounceRate,omitempty"`
	ComplaintRate *float64   `json:"complaintRate,omitempty"`
	OpenRate      *float64   `json:"openRate,omitempty"`
	Throttle      *float64   `json:"throttle,omitempty" doc:"Messages per second; absent = the provider's full rate"`
	Held          bool       `json:"held"`
	HeldSince     *time.Time `json:"heldSince,omitempty"`
	Status        string     `json:"status" enum:"ok,sending,paused,done"`
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
	Cost          float64    `json:"cost"`
	PlannedCost   float64    `json:"plannedCost"`
	Retrying      bool       `json:"retrying"`
	RetryUntil    *time.Time `json:"retryUntil,omitempty"`
}

// LiveThreshold is a rate against its auto-pause limit.
type LiveThreshold struct {
	Key     string  `json:"key" enum:"bounce,complaint,mailbox_complaint,api_errors"`
	Value   float64 `json:"value"`
	Limit   float64 `json:"limit"`
	Mailbox string  `json:"mailbox,omitempty"`
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
	Kind          string         `json:"kind" enum:"threshold,provider_outage,throttled,recheck_blocked"`
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
	Sunset       int64 `json:"sunset"`
	Excluded     int64 `json:"excluded"`
	Tiers        int64 `json:"tiers"`
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
		Routes: []LiveRouteShare{}}
	loc := state.Location(cc.WorkspaceTimezone)
	snd, sendErr := e.Q.SendingGetSend(ctx, campaignID)
	hasSend := sendErr == nil
	if sendErr != nil && !errors.Is(sendErr, pgx.ErrNoRows) {
		return s, sendErr
	}
	th := ParseThresholds(cc.Thresholds)
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
	for _, k := range sortedKeys(throttles) {
		s.Throttles = append(s.Throttles, LiveThrottle{Mailbox: k, Rate: throttles[k]})
	}
	// State.
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
	// Counters.
	rows, err := e.Q.SendingCounters(ctx, campaignID)
	if err != nil {
		return s, err
	}
	ts := tallyRows(rows)
	t := ts.total
	s.Sent = t.sent
	s.Totals = LiveTotals{Delivered: t.delivered, Bounced: t.bounced(), HardBounced: t.hard, SoftBounced: t.soft, Complained: t.complained, Deferred: t.deferred,
		Rejected: t.rejected, Opened: t.human + t.machine, OpenedHuman: t.human, OpenedMachine: t.machine, Clicked: t.clicked, Unsubscribed: t.unsubscribed}
	// Queue.
	cid := cc.ID
	queue := map[queueKey]queueCell{}
	if hasSend {
		qrows, err := e.Q.SendingQueueState(ctx, store.SendingQueueStateParams{CampaignID: &cid, Now: now})
		if err != nil {
			return s, err
		}
		for _, r := range qrows {
			if r.ProviderID == nil {
				continue
			}
			k := queueKey{*r.ProviderID, r.MailboxProvider}
			c := queue[k]
			if r.Status == "held" {
				c.held += r.N
			} else {
				c.queued += r.N
			}
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
	// Routes and providers.
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
	var maxAPI *float64
	via := []string{}
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
			ComplaintRate: ptrRate(pt.complained, pt.delivered), APIErrorRate: apiErr[p.ID], Status: p.Status, RouteStatus: r.Status,
			Host: hostOf(p), Cost: float64(r.CostMicros) / 1e6}
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
		if d.Decision == "retry" {
			lp.Retrying, lp.RetryUntil = true, d.Until
		}
		if h.Quota != nil && h.Quota.Max > 0 {
			free := max(h.Quota.Max-h.Quota.Used-lp.Remaining, 0)
			mx := h.Quota.Max
			lp.QuotaFree, lp.QuotaMax = &free, &mx
		}
		if pr := e.rateOf(ctx, e.Q, p, loc, now); pr.WarmingUp {
			day, days, cp, sent := pr.WarmupDay, pr.WarmupDays, pr.Cap, pr.SentToday
			lp.WarmupDay, lp.WarmupDays, lp.WarmupCap, lp.WarmupSent = &day, &days, &cp, &sent
		}
		// The API error threshold watches the routes this send still uses (a
		// share moved away or finished doesn't count).
		if lp.APIErrorRate != nil && (r.Status == "pending" || r.Status == "sending" || r.Status == "held") && (maxAPI == nil || *lp.APIErrorRate > *maxAPI) {
			v := *lp.APIErrorRate
			maxAPI = &v
		}
		via = append(via, webhooksVia(p))
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
	s.EventsVia = joinAnd(uniqStrings(via))
	// Mailbox providers.
	var bestMailbox string
	var bestRate float64 = -1
	for _, k := range routing.MailboxProviders {
		m := ts.mailbox[k]
		if m == nil {
			m = &tally{}
		}
		lm := LiveMailbox{Key: k, Name: mailboxName(k), Sent: m.sent, Delivered: m.delivered, Bounced: m.bounced(), Complained: m.complained, Opened: m.human + m.machine,
			DeliveredRate: ptrRate(m.delivered, m.sent), BounceRate: ptrRate(m.bounced(), m.sent), ComplaintRate: ptrRate(m.complained, m.delivered), OpenRate: ptrRate(m.human+m.machine, m.delivered)}
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
			lm.HeldSince = &since
			s.MailboxesHeld = append(s.MailboxesHeld, lm.Name)
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
		if lm.ComplaintRate != nil && *lm.ComplaintRate > bestRate {
			bestRate, bestMailbox = *lm.ComplaintRate, k
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
	// Thresholds.
	s.Thresholds = append(s.Thresholds,
		LiveThreshold{Key: "bounce", Value: round2(rate(t.bounced(), t.sent)), Limit: th.Bounce},
		LiveThreshold{Key: "complaint", Value: round2(rate(t.complained, t.delivered)), Limit: th.Complaint})
	if bestMailbox != "" {
		s.Thresholds = append(s.Thresholds, LiveThreshold{Key: "mailbox_complaint", Value: round2(bestRate), Limit: th.MailboxComplaint, Mailbox: bestMailbox})
	}
	api := 0.0
	if maxAPI != nil {
		api = *maxAPI
	}
	s.Thresholds = append(s.Thresholds, LiveThreshold{Key: "api_errors", Value: round2(api), Limit: th.APIErrors})
	// Queue totals and the held split.
	for k, c := range queue {
		_, onHold := holds[k.mailbox]
		heldMailbox := onHold && !holds[k.mailbox].Resumed
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
	if s.State == "sending" && (len(s.MailboxesHeld) > 0 || len(s.ProvidersHeld) > 0) {
		s.State = "partial"
	}
	for _, lm := range s.Mailboxes {
		if !lm.Held && lm.Remaining > 0 {
			s.Continuing = append(s.Continuing, lm.Name)
		}
	}
	if cc.StartedAt != nil && cc.CompletedAt != nil {
		d := int64(cc.CompletedAt.Sub(*cc.StartedAt).Seconds())
		s.Duration = &d
	}
	// Details.
	s.Details.From = content.FromEmail
	if s.Details.From == "" {
		s.Details.From = cc.SenderEmail
	}
	s.Details.Audience = e.audienceLabel(ctx, cc.WorkspaceID, cc.Audience)
	s.Details.AudienceCount = s.Recipients
	s.Details.Route = strings.Join(routeLabel, " · ")
	s.Details.ScheduledBy = s.ScheduledBy
	// A/B.
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
	// Incidents.
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
		_ = json.Unmarshal(in.Detail, &li.Detail)
		s.Incidents = append(s.Incidents, li)
		if in.ResolvedAt == nil && s.Incident == nil && (in.Kind == "threshold" || in.Kind == "provider_outage") {
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
			}
		}
	}
	// Feed.
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
	s.Options = LiveOptions{
		CanPause:  cc.Status == "sending",
		CanResume: cc.Status == "paused" || len(s.MailboxesHeld) > 0 || len(s.ProvidersHeld) > 0,
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
func (e *Engine) audienceLabel(ctx context.Context, ws uuid.UUID, raw json.RawMessage) string {
	def := parseAudience(raw)
	lists, _ := e.Q.SendingListNames(ctx, ws)
	names := map[uuid.UUID]string{}
	for _, l := range lists {
		names[l.ID] = l.Name
	}
	var parts []string
	if len(def.Segments) == 0 && len(def.Lists) == len(names) && len(names) > 1 {
		parts = append(parts, "All subscribers")
	} else {
		for _, id := range def.Lists {
			if n, ok := names[id]; ok {
				parts = append(parts, n)
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
				parts = append(parts, n)
			}
		}
	}
	return strings.Join(parts, " + ")
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
	code := strings.TrimSpace(f.OriginalCode)
	join := func(parts ...string) string {
		out := []string{}
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				out = append(out, p)
			}
		}
		return strings.Join(out, " · ")
	}
	switch providers.EventType(f.Type) {
	case providers.EventDelivered:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "delivered", "delivered", "ok", join(short, code)
	case providers.EventHardBounce:
		d := join(short, strings.TrimSpace(code+" "+f.Diagnostic))
		if f.Suppresses {
			d = join(d, "added to suppression list")
		}
		ev.Label, ev.Short, ev.Tone, ev.Detail = "hard bounce", "bounce", "bad", d
	case providers.EventSoftBounce:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "soft bounce", "bounce", "warn", join(short, strings.TrimSpace(code+" "+f.Diagnostic))
	case providers.EventDeferred:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "deferred", "deferred", "warn", join(short, code, f.Diagnostic)
	case providers.EventComplaint:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "complaint", "complaint", "bad", join(short, "feedback loop report", "added to suppression list")
	case providers.EventRejected:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "rejected", "rejected", "bad", join(short, strings.TrimSpace(code+" "+f.Diagnostic))
	case providers.EventOpen:
		if f.Machine {
			ev.Label, ev.Short, ev.Tone, ev.Detail = "opened · machine", "open", "neutral", f.Diagnostic
		} else {
			d := f.Diagnostic
			if f.Variant != "" {
				d = join(d, "variant "+f.Variant)
			}
			ev.Label, ev.Short, ev.Tone, ev.Detail = "opened · human", "open", "info", d
		}
	case providers.EventClick:
		if f.Machine {
			ev.Label, ev.Short, ev.Tone, ev.Detail = "clicked · scanner", "scanner", "neutral", join(linkText(f.Url), f.Diagnostic)
		} else {
			ev.Label, ev.Short, ev.Tone, ev.Detail = "clicked", "click", "ok", linkText(f.Url)
		}
	case providers.EventUnsubscribe:
		ev.Label, ev.Short, ev.Tone, ev.Detail = "unsubscribed", "unsubscribed", "neutral", short
	default:
		ev.Label, ev.Short, ev.Tone, ev.Detail = f.Type, f.Type, "neutral", short
	}
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

// webhooksVia names how a provider reports events ("SNS", "Postal webhooks").
func webhooksVia(p store.Provider) string {
	c := state.Caps(p)
	switch c.Webhooks {
	case "SNS":
		return "SNS"
	case "Bounce mailbox":
		return "the bounce mailbox"
	}
	return state.Short(p.Name, p.Adapter) + " webhooks"
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

func uniqStrings(xs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i] == "SNS" && out[j] != "SNS" })
	return out
}

func ptrTime(t time.Time) *time.Time { return &t }

func strconvFloat(v float64) string { return rateText(v) }
