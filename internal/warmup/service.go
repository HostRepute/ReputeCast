package warmup

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service serves the warm-up planner.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// WarmupSettings are a plan's editable settings.
type WarmupSettings struct {
	// Days is the plan's length in counted days.
	Days           int     `json:"days" enum:"14,21,30"`
	EngagedFirst   bool    `json:"engagedFirst"`
	GuardBounce    float64 `json:"guardBounce" minimum:"0.5" maximum:"10"`
	GuardComplaint float64 `json:"guardComplaint" minimum:"0.01" maximum:"1"`
	GuardAction    string  `json:"guardAction" enum:"hold,step_back,pause"`
	SpreadMailbox  bool    `json:"spreadMailbox"`
	// Overflow sends what's over today's cap through a warm provider of
	// the same route instead of waiting for tomorrow's cap.
	Overflow *bool `json:"overflow,omitempty"`
}

// WarmupCampaignUse is a scheduled campaign's share on a planned day.
type WarmupCampaignUse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// WarmupRow is one date of the plan table: a day that was judged, today,
// or a planned day (dates ahead assume every day counts).
type WarmupRow struct {
	Day  int       `json:"day"`
	Date time.Time `json:"date"`
	Cap  int64     `json:"cap"`
	// PlannedCap is the schedule's cap when a slowdown holds a lower one.
	PlannedCap int64    `json:"plannedCap,omitempty"`
	Sent       *int64   `json:"sent,omitempty"`
	Bounce     *float64 `json:"bounce,omitempty"`
	Complaint  *float64 `json:"complaint,omitempty"`
	Tiers      []string `json:"tiers"`
	Share      int      `json:"share"`
	// ShareChange is true on the day the Broadcast share steps up.
	ShareChange bool                `json:"shareChange"`
	Uses        []WarmupCampaignUse `json:"uses"`
	// Status: counted, not_counted, today, planned.
	Status string `json:"status" enum:"counted,not_counted,today,planned"`
	// Why a day didn't count: short (under 80% of its cap went out), guard
	// (a guard tripped; Metric says which) or held (a healthy day under a
	// slowdown's hold: it proves the held cap without moving the plan on).
	Reason string `json:"reason,omitempty" enum:"short,guard,held"`
	Metric string `json:"metric,omitempty" enum:"bounce,complaint"`
	// Judged: the guards had enough messages to judge the day.
	Judged bool `json:"judged"`
	// Bounced, Complained and Delivered are the day's counts (complaints
	// and deliveries at mailbox providers that report complaints), so
	// several days add up exactly.
	Bounced    int64 `json:"bounced"`
	Complained int64 `json:"complained"`
	Delivered  int64 `json:"delivered"`
	// Need is how many messages the day needs to count.
	Need int64 `json:"need"`
	// Held: a slowdown held the day's cap; Healthy is the healthy day it
	// was under the hold (1 of HealthyDays…).
	Held    bool `json:"held"`
	Healthy int  `json:"healthy,omitempty"`
	// Repeat marks a planned day that repeats a day that didn't count.
	Repeat bool `json:"repeat"`
	// Eligible marks the last day: the provider then becomes a Broadcast
	// failover target.
	Eligible bool `json:"eligible"`
}

// WarmupRampDay is one day of the volume ramp (the chart).
type WarmupRampDay struct {
	Day        int    `json:"day"`
	PlannedCap int64  `json:"plannedCap"`
	Sent       *int64 `json:"sent,omitempty"`
	Slowed     bool   `json:"slowed"`
	Today      bool   `json:"today"`
}

// WarmupMailboxCap is today's cap and sends of one mailbox provider.
type WarmupMailboxCap struct {
	Mailbox string `json:"mailbox"`
	Percent int    `json:"percent"`
	Cap     int64  `json:"cap"`
	Sent    int64  `json:"sent"`
}

// WarmupLogEntry is one line of the warm-up log.
type WarmupLogEntry struct {
	At    time.Time `json:"at"`
	Kind  string    `json:"kind"`
	Title string    `json:"title"`
	Body  string    `json:"body"`
}

// WarmupStep is a planned change ("rises to 4 /s on day 12").
type WarmupStep struct {
	Day   int     `json:"day"`
	Value float64 `json:"value"`
}

// WarmupSlowdown summarizes the last slowdown ("day 6, bounces 2.3%").
type WarmupSlowdown struct {
	Day    int      `json:"day"`
	Metric string   `json:"metric" enum:"bounce,complaint"`
	Rate   *float64 `json:"rate,omitempty"`
}

// WarmupPlanView is a warm-up plan with its progress (ProvidersWarmup).
type WarmupPlanView struct {
	ID           string `json:"id"`
	ProviderID   string `json:"providerId"`
	Provider     string `json:"provider"`
	Short        string `json:"short"`
	IP           string `json:"ip"`
	Status       string `json:"status" enum:"active,slowed,paused,done"`
	StatusDetail string `json:"statusDetail"`
	// Day is today's plan day; Counted the days that counted so far, of
	// Days; CalendarDays the days since the plan started (today excluded).
	Day          int `json:"day"`
	Days         int `json:"days"`
	Counted      int `json:"counted"`
	CalendarDays int `json:"calendarDays"`
	// StartedOn is the day the plan began; FinishesOn the earliest day it
	// can finish (every day from today counting).
	StartedOn  time.Time `json:"startedOn"`
	FinishesOn time.Time `json:"finishesOn"`
	Progress   float64   `json:"progress" doc:"Percent of the plan's days counted"`
	TodayCap   int64     `json:"todayCap"`
	SentToday  int64     `json:"sentToday"`
	// NeedToday is what today needs to count; RepeatsTomorrow is true when
	// today can't count any more (nothing scheduled and too little sent).
	NeedToday       int64 `json:"needToday"`
	RepeatsTomorrow bool  `json:"repeatsTomorrow"`
	// ScheduledToday is how many campaigns use the provider today.
	ScheduledToday int   `json:"scheduledToday"`
	SentTotal      int64 `json:"sentTotal"`
	// BounceRate and ComplaintRate cover the last seven days; nil when
	// nothing was measured (complaints count only mailbox providers that
	// report them).
	BounceRate    *float64        `json:"bounceRate,omitempty"`
	ComplaintRate *float64        `json:"complaintRate,omitempty"`
	Slowdowns     int             `json:"slowdowns"`
	LastSlowdown  *WarmupSlowdown `json:"lastSlowdown,omitempty"`
	RateToday     float64         `json:"rateToday"`
	NextRate      *WarmupStep     `json:"nextRate,omitempty"`
	// Held is the cap a slowdown holds and HealthyStreak the healthy days so far.
	Held          *int64             `json:"held,omitempty"`
	HealthyStreak int                `json:"healthyStreak"`
	HealthyDays   int                `json:"healthyDays"`
	MinSample     int                `json:"minSample"`
	CountShare    float64            `json:"countShare"`
	Rows          []WarmupRow        `json:"rows"`
	Ramp          []WarmupRampDay    `json:"ramp"`
	MailboxCaps   []WarmupMailboxCap `json:"mailboxCaps"`
	// OverflowTo is the warm provider over-cap mail goes through (the
	// largest share of the Broadcasts split that isn't warming up), empty
	// when there is none; OverflowToday what went through it today.
	OverflowTo    string `json:"overflowTo,omitempty"`
	OverflowToday int64  `json:"overflowToday"`
	// TransactionalOverCap is how many transactional messages went out over
	// today's cap (or while the plan was paused) because no warm provider of
	// their route could take them: transactional mail never waits, and it
	// counts toward the cap (review R3.L6b).
	TransactionalOverCap int64          `json:"transactionalOverCap"`
	Settings             WarmupSettings `json:"settings"`
	// Holds are campaigns' shares waiting at today's cap, with choices.
	Holds     []WarmupCapHold  `json:"holds"`
	Log       []WarmupLogEntry `json:"log"`
	CreatedBy string           `json:"createdBy"`
}

// WarmupCandidate is a provider that could start a warm-up plan.
type WarmupCandidate struct {
	ProviderID string   `json:"providerId"`
	Name       string   `json:"name"`
	IPs        []string `json:"ips"`
}

// WarmupOverview is the warm-up tab.
type WarmupOverview struct {
	Plans      []WarmupPlanView  `json:"plans"`
	Candidates []WarmupCandidate `json:"candidates"`
}

func dateIn(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// Overview returns every plan of the workspace, current plans first.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (WarmupOverview, error) {
	out := WarmupOverview{Plans: []WarmupPlanView{}, Candidates: []WarmupCandidate{}}
	rows, err := s.Q.WarmupListPlans(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	busy := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.Status != "done" {
			busy[r.ProviderID] = true
		}
		v, err := s.view(ctx, ws, r)
		if err != nil {
			return out, err
		}
		out.Plans = append(out.Plans, v)
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, p := range provs {
		if busy[p.ID] {
			continue
		}
		ips := p.Ips
		if ips == nil {
			ips = []string{}
		}
		out.Candidates = append(out.Candidates, WarmupCandidate{ProviderID: p.ID.String(), Name: p.Name, IPs: ips})
	}
	return out, nil
}

func settingsOf(r store.WarmupListPlansRow) WarmupSettings {
	overflow := r.Overflow
	return WarmupSettings{Days: int(r.Days), EngagedFirst: r.EngagedFirst, GuardBounce: r.SlowdownBounce, GuardComplaint: r.SlowdownComplaint,
		GuardAction: r.GuardAction, SpreadMailbox: r.SpreadMailbox, Overflow: &overflow}
}

func (s *Service) view(ctx context.Context, ws *store.Workspace, r store.WarmupListPlansRow) (WarmupPlanView, error) {
	loc := state.Location(ws.Timezone)
	now := s.now()
	today := dateIn(now, loc)
	plan := ParsePlan(r.Schedule, r.Progress)
	started := localDate(r.StartedOn, loc)
	days := int(r.Days)
	// A paused plan stays on the day it stopped.
	day := min(max(DayOn(r.StartDate, pausedAt(r.Status, r.PausedAt), now, loc), 1), days)
	l := i18n.LangFrom(ctx)
	v := WarmupPlanView{ID: r.ID.String(), ProviderID: r.ProviderID.String(), Provider: r.ProviderName, Short: state.Short(r.ProviderName, r.Adapter), IP: r.Ip,
		Status: r.Status, StatusDetail: textIn(l, r.Texts, "status_detail", r.StatusDetail), Day: day, Days: days, Counted: plan.Counted(),
		StartedOn: started, Rows: []WarmupRow{}, Ramp: []WarmupRampDay{}, MailboxCaps: []WarmupMailboxCap{}, Holds: []WarmupCapHold{}, Log: []WarmupLogEntry{}, CreatedBy: r.CreatedByName,
		Settings: settingsOf(r), HealthyStreak: int(r.HealthyStreak), HealthyDays: HealthyDays, MinSample: int(r.MinSample), CountShare: CountShare, Held: r.HeldCap}
	if r.Status == "done" {
		v.Day, v.Counted = days, max(v.Counted, days)
	}
	v.CalendarDays = max(int(today.Sub(started).Hours()/24+0.5), 0)
	v.Progress = math.Round(float64(min(v.Counted, days))/float64(days)*1000) / 10
	v.FinishesOn = today.AddDate(0, 0, days-day)
	hold := Hold{Cap: r.HeldCap}
	v.TodayCap = hold.CapFor(plan.PlannedCap(day))
	v.NeedToday = int64(math.Ceil(float64(v.TodayCap) * CountShare))

	// Today's sends per mailbox provider (every stream counts).
	sent, err := s.Q.WarmupSentSince(ctx, store.WarmupSentSinceParams{ProviderID: &r.ProviderID, Since: today})
	if err != nil {
		return v, err
	}
	byMailbox := map[string]int64{}
	for _, x := range sent {
		byMailbox[MailboxOf(x.MailboxProvider)] += x.N
		v.SentToday += x.N
	}
	caps := MailboxCaps(v.TodayCap)
	for _, sh := range MailboxShares {
		v.MailboxCaps = append(v.MailboxCaps, WarmupMailboxCap{Mailbox: sh.Mailbox, Percent: sh.Percent, Cap: caps[sh.Mailbox], Sent: byMailbox[sh.Mailbox]})
	}
	if r.OverflowedOn != nil && localDate(*r.OverflowedOn, loc).Equal(today) {
		v.OverflowToday = r.Overflowed
	}
	v.TransactionalOverCap = OverCapToday(r.OverCap, r.OverCapOn, today, loc)
	v.OverflowTo = s.overflowTarget(ctx, ws, r.ProviderID)
	if v.Holds, err = s.capHolds(ctx, ws, r.ProviderID); err != nil {
		return v, err
	}

	// Campaigns that use the provider on the days ahead.
	endAt := today.AddDate(0, 0, days-day+2)
	uses, err := s.Q.WarmupPlannedCampaignUse(ctx, store.WarmupPlannedCampaignUseParams{ProviderID: r.ProviderID, Since: &today, Until: &endAt})
	if err != nil {
		return v, err
	}
	usesOn := func(date time.Time) []WarmupCampaignUse {
		out := []WarmupCampaignUse{}
		for _, u := range uses {
			if u.ScheduledAt != nil && dateIn(*u.ScheduledAt, loc).Equal(date) {
				out = append(out, WarmupCampaignUse{Name: u.Name, Count: u.Planned})
			}
		}
		return out
	}

	// Judged days, as they happened; the last seven days' rates add up
	// their counts and today's (complaints count only mailbox providers that
	// report them).
	var week DayStats
	for _, res := range plan.Progress {
		date, err := time.ParseInLocation("2006-01-02", res.Date, loc)
		if err != nil || !date.Before(today) {
			continue
		}
		sentN := res.Sent
		row := WarmupRow{Day: res.Day, Date: date, Cap: res.Cap, PlannedCap: res.PlannedCap, Sent: &sentN, Bounce: res.Bounce, Complaint: res.Complaint,
			Tiers: plan.TiersOn(res.Day), Share: plan.Share(res.Day), Uses: []WarmupCampaignUse{}, Status: "counted", Judged: res.Judged,
			Need: int64(math.Ceil(float64(res.Cap) * CountShare)), Held: res.Held, Healthy: res.Healthy, Bounced: res.Bounced, Complained: res.Complained, Delivered: res.Delivered}
		if !res.Counted {
			row.Status = "not_counted"
			switch {
			case res.Slowed:
				row.Reason, row.Metric = "guard", res.Metric
			case res.Held && res.Healthy > 0:
				row.Reason = "held"
			default:
				row.Reason = "short"
			}
		}
		if res.Slowed {
			v.Slowdowns++
			rate := res.Bounce
			if res.Metric == "complaint" {
				rate = res.Complaint
			}
			v.LastSlowdown = &WarmupSlowdown{Day: res.Day, Metric: cmp.Or(res.Metric, "bounce"), Rate: rate}
		}
		v.SentTotal += res.Sent
		if !date.Before(today.AddDate(0, 0, -6)) {
			week.Sent += res.Sent
			week.Bounced += res.Bounced
			week.Complained += res.Complained
			week.Delivered += res.Delivered
		}
		v.Rows = append(v.Rows, row)
	}
	v.SentTotal += v.SentToday
	todayStats, err := dayStats(ctx, s.Q, r.ProviderID, today, today.AddDate(0, 0, 1))
	if err != nil {
		return v, err
	}
	week.Sent += todayStats.Sent
	week.Bounced += todayStats.Bounced
	week.Complained += todayStats.Complained
	week.Delivered += todayStats.Delivered
	if week.Sent > 0 {
		b := round1(float64(week.Bounced) / float64(week.Sent) * 100)
		v.BounceRate = &b
	}
	if week.Delivered > 0 {
		c := round2(float64(week.Complained) / float64(week.Delivered) * 100)
		v.ComplaintRate = &c
	}
	if r.Status == "done" {
		return s.finishView(ctx, r, v, l)
	}

	// Today, then the days ahead (every one of them counting).
	todayUses := usesOn(today)
	v.ScheduledToday = len(todayUses)
	v.RepeatsTomorrow = v.SentToday < v.NeedToday && len(todayUses) == 0 && r.Status != "paused"
	sentToday := v.SentToday
	// Today so far, judged as the daily job will once enough went out.
	tv := Judge(todayStats, v.TodayCap, Guards{Bounce: r.SlowdownBounce, Complaint: r.SlowdownComplaint, MinSample: int64(r.MinSample)})
	todayRow := WarmupRow{Day: day, Date: today, Cap: v.TodayCap, Sent: &sentToday, Tiers: plan.TiersOn(day), Share: plan.Share(day), Uses: todayUses, Status: "today",
		Need: v.NeedToday, Held: r.HeldCap != nil, Eligible: day == days, Judged: tv.Judged, Bounce: tv.Bounce, Complaint: tv.Complaint,
		Bounced: todayStats.Bounced, Complained: todayStats.Complained, Delivered: todayStats.Delivered}
	if r.HeldCap != nil && plan.PlannedCap(day) != v.TodayCap {
		todayRow.PlannedCap = plan.PlannedCap(day)
	}
	v.Rows = append(v.Rows, todayRow)
	date := today
	next := day
	if v.RepeatsTomorrow {
		// Today can't count any more: tomorrow repeats today's day.
		date = date.AddDate(0, 0, 1)
		v.Rows = append(v.Rows, WarmupRow{Day: day, Date: date, Cap: v.TodayCap, Tiers: plan.TiersOn(day), Share: plan.Share(day), Uses: usesOn(date), Status: "planned",
			Need: v.NeedToday, Repeat: true, Eligible: day == days})
	}
	prevShare := plan.Share(day)
	for d := next + 1; d <= days; d++ {
		date = date.AddDate(0, 0, 1)
		row := WarmupRow{Day: d, Date: date, Cap: plan.PlannedCap(d), Tiers: plan.TiersOn(d), Share: plan.Share(d), Uses: usesOn(date), Status: "planned",
			Need: int64(math.Ceil(float64(plan.PlannedCap(d)) * CountShare)), Eligible: d == days}
		row.ShareChange = row.Share != prevShare
		prevShare = row.Share
		v.Rows = append(v.Rows, row)
	}
	v.FinishesOn = date

	// The ramp: each plan day's planned cap and what its counted (or last)
	// date sent.
	for d := 1; d <= days; d++ {
		rd := WarmupRampDay{Day: d, PlannedCap: plan.PlannedCap(d), Today: d == day}
		for _, res := range plan.Progress {
			if res.Day != d {
				continue
			}
			sentN := res.Sent
			rd.Sent = &sentN
			rd.Slowed = rd.Slowed || res.Slowed
		}
		if d == day {
			rd.Sent = &sentToday
		}
		v.Ramp = append(v.Ramp, rd)
	}
	v.RateToday = RateFor(v.TodayCap)
	if r.HeldCap == nil {
		v.RateToday = plan.Rate(day)
	}
	for d := day + 1; d <= days; d++ {
		if rt := plan.Rate(d); rt > v.RateToday {
			v.NextRate = &WarmupStep{Day: d, Value: rt}
			break
		}
	}
	return s.finishView(ctx, r, v, l)
}

// finishView adds the log: today's line while nothing more is scheduled
// and today can't count, then what the plan recorded.
func (s *Service) finishView(ctx context.Context, r store.WarmupListPlansRow, v WarmupPlanView, l i18n.Lang) (WarmupPlanView, error) {
	log, err := s.Q.WarmupLog(ctx, store.WarmupLogParams{PlanID: r.ID, Lim: 20})
	if err != nil {
		return v, err
	}
	if v.RepeatsTomorrow {
		v.Log = append(v.Log, WarmupLogEntry{At: s.now(), Kind: "today",
			Body: msgs.M("log.todayRepeats").Text(l, i18n.Args{"day": v.Day, "sent": v.SentToday, "need": v.NeedToday})})
	}
	for _, e := range log {
		v.Log = append(v.Log, WarmupLogEntry{At: e.OccurredAt, Kind: e.Kind, Title: textIn(l, e.Texts, "title", e.Title), Body: textIn(l, e.Texts, "body", e.Body)})
	}
	return v, nil
}

// overflowTarget names the warm provider over-cap Broadcasts go through:
// the largest share of the Broadcasts split that isn't warming up.
func (s *Service) overflowTarget(ctx context.Context, ws *store.Workspace, providerID uuid.UUID) string {
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return ""
	}
	def, ok := set.Default(providers.StreamBroadcast)
	if !ok {
		return ""
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return ""
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	best, bestW := "", -1
	for _, sp := range def.Splits {
		p, ok := byID[sp.ProviderID]
		if !ok || sp.ProviderID == providerID || p.Status == state.WarmingUp || !state.Caps(p).BulkAllowed() {
			continue
		}
		if sp.Weight > bestW {
			best, bestW = p.Name, sp.Weight
		}
	}
	return best
}

// OverCapToday is the plan's count of transactional messages sent over the
// cap, when it counts today (the workspace's day) and 0 otherwise.
func OverCapToday(n int64, on *time.Time, today time.Time, loc *time.Location) int64 {
	if on == nil || !localDate(*on, loc).Equal(dateIn(today, loc)) {
		return 0
	}
	return n
}

func dayUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func (s *Service) plan(ctx context.Context, ws *store.Workspace, id string) (store.WarmupGetPlanRow, error) {
	pid, err := uuid.Parse(id)
	if err != nil {
		return store.WarmupGetPlanRow{}, api.NotFound("Warm-up plan")
	}
	r, err := s.Q.WarmupGetPlan(ctx, store.WarmupGetPlanParams{ID: pid, WorkspaceID: ws.ID})
	return r, api.MapError(err, "Warm-up plan")
}

func (s *Service) planView(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WarmupPlanView, error) {
	rows, err := s.Q.WarmupListPlans(ctx, ws.ID)
	if err != nil {
		return WarmupPlanView{}, err
	}
	for _, r := range rows {
		if r.ID == id {
			return s.view(ctx, ws, r)
		}
	}
	return WarmupPlanView{}, api.NotFound("Warm-up plan")
}

func lastVolume(sched []Day) int64 {
	if len(sched) == 0 {
		return 48000
	}
	return sched[len(sched)-1].Volume
}

// UpdateSettings changes a plan's length, ordering, guards, mailbox caps
// and overflow. A new length rebuilds the days ahead from today's cap to the
// same last volume, so shortening a plan never makes today's cap jump
// (PR-8); a plan can't get shorter than the days it already counted.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, id string, in WarmupSettings) (WarmupPlanView, error) {
	r, err := s.plan(ctx, ws, id)
	if err != nil {
		return WarmupPlanView{}, err
	}
	if r.Status == "done" {
		return WarmupPlanView{}, api.Conflict(say(ctx, "problem.complete", nil))
	}
	loc := state.Location(ws.Timezone)
	plan := ParsePlan(r.Schedule, r.Progress)
	sched := plan.Schedule
	day := max(DayOn(r.StartDate, pausedAt(r.Status, r.PausedAt), s.now(), loc), 1)
	if in.Days != int(r.Days) {
		todayCap := Hold{Cap: r.HeldCap}.CapFor(plan.PlannedCap(day))
		next, err := Reschedule(sched, day, in.Days, todayCap)
		if errors.Is(err, ErrTooShort) {
			return WarmupPlanView{}, api.Invalid("body.days", say(ctx, "problem.tooShort", i18n.Args{"counted": plan.Counted(), "day": day}))
		}
		if err != nil {
			return WarmupPlanView{}, err
		}
		sched = next
	}
	overflow := r.Overflow
	if in.Overflow != nil {
		overflow = *in.Overflow
	}
	in.Overflow = &overflow
	sb, _ := json.Marshal(sched)
	beforeOverflow := r.Overflow
	before := WarmupSettings{Days: int(r.Days), EngagedFirst: r.EngagedFirst, GuardBounce: r.SlowdownBounce, GuardComplaint: r.SlowdownComplaint, GuardAction: r.GuardAction,
		SpreadMailbox: r.SpreadMailbox, Overflow: &beforeOverflow}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WarmupUpdateSettings(ctx, store.WarmupUpdateSettingsParams{ID: r.ID, Days: int32(in.Days), Schedule: sb, EngagedFirst: in.EngagedFirst,
			SlowdownBounce: in.GuardBounce, SlowdownComplaint: in.GuardComplaint, GuardAction: in.GuardAction, SpreadMailbox: in.SpreadMailbox, Overflow: overflow}); err != nil {
			return err
		}
		body := keep("log.settings", actor(ctx))
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: s.now(), Kind: "settings",
			Body: body.English, Texts: textsJSON(map[string]i18n.Text{"body": body.Text})}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "warmup.update", TargetType: "warmup_plan", TargetID: r.ID.String(), TargetName: r.ProviderName, Before: before, After: in})
	})
	if err != nil {
		return WarmupPlanView{}, err
	}
	return s.planView(ctx, ws, r.ID)
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// SetPaused pauses or resumes a plan. A pause freezes the plan (PR-8):
// no mail goes to the provider through it, its days aren't judged, and it
// resumes on the day it stopped.
func (s *Service) SetPaused(ctx context.Context, ws *store.Workspace, id string, paused bool) (WarmupPlanView, error) {
	r, err := s.plan(ctx, ws, id)
	if err != nil {
		return WarmupPlanView{}, err
	}
	if r.Status == "done" {
		return WarmupPlanView{}, api.Conflict(say(ctx, "problem.completeShort", nil))
	}
	if (r.Status == "paused") == paused {
		return s.planView(ctx, ws, r.ID)
	}
	who := actor(ctx)
	now := s.now()
	loc := state.Location(ws.Timezone)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var kind string
		var body kept
		if paused {
			detail := keep("status.pausedBy", who)
			kind, body = "paused", keep("log.paused", with(who, "provider", r.ProviderName))
			if err := q.WarmupSetStatus(ctx, store.WarmupSetStatusParams{ID: r.ID, Status: "paused", StatusDetail: detail.English, PausedAt: &now,
				Texts: textsJSON(map[string]i18n.Text{"status_detail": detail.Text})}); err != nil {
				return err
			}
		} else {
			// The days spent paused don't count and aren't judged: the
			// plan's day 1 moves on by them, so today is the day it stopped.
			today := dateIn(now, loc)
			pausedOn := today
			if r.PausedAt != nil {
				pausedOn = dateIn(*r.PausedAt, loc)
			}
			skipped := max(int(today.Sub(pausedOn).Hours()/24+0.5), 0)
			start := localDate(r.StartDate, loc).AddDate(0, 0, skipped)
			through := r.EvaluatedThrough
			if skipped > 0 {
				t := dayUTC(today.AddDate(0, 0, -1))
				through = &t
			}
			status := "active"
			if r.HeldCap != nil {
				status = "slowed"
			}
			kind, body = "resumed", keep("log.resumedOn", with(who, "day", dayNumber(start, today), "skipped", skipped))
			if err := q.WarmupResume(ctx, store.WarmupResumeParams{ID: r.ID, Status: status, StatusDetail: "", StartDate: dayUTC(start), EvaluatedThrough: through}); err != nil {
				return err
			}
		}
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: now, Kind: kind, Body: body.English,
			Texts: textsJSON(map[string]i18n.Text{"body": body.Text})}); err != nil {
			return err
		}
		wsID := ws.ID
		action, status := "warmup.resume", "active"
		if paused {
			action, status = "warmup.pause", "paused"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: action, TargetType: "warmup_plan", TargetID: r.ID.String(), TargetName: r.ProviderName,
			Before: map[string]string{"status": r.Status}, After: map[string]string{"status": status}})
	})
	if err != nil {
		return WarmupPlanView{}, err
	}
	if !paused {
		// Shares the pause held go on now; the hold job would resume them
		// within five minutes anyway.
		_ = s.resumeCapHolds(ctx, r.ProviderID)
	}
	return s.planView(ctx, ws, r.ID)
}

// NewPlanInput starts a warm-up plan.
type NewPlanInput struct {
	ProviderID string `json:"providerId" format:"uuid"`
	IP         string `json:"ip,omitempty" maxLength:"45"`
	WarmupSettings
	// TargetVolume is the last day's volume; it defaults to the workspace's
	// largest broadcast.
	TargetVolume int64 `json:"targetVolume,omitempty" minimum:"0" maximum:"10000000"`
}

// Create starts a plan for a provider without one.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in NewPlanInput) (WarmupPlanView, error) {
	pid, err := uuid.Parse(in.ProviderID)
	if err != nil {
		return WarmupPlanView{}, api.Invalid("body.providerId", say(ctx, "problem.pickProvider", nil))
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: pid})
	if err != nil {
		return WarmupPlanView{}, api.Invalid("body.providerId", say(ctx, "problem.providerNotConnected", nil))
	}
	if _, err := s.Q.WarmupActiveForProvider(ctx, p.ID); err == nil {
		return WarmupPlanView{}, api.Conflict(say(ctx, "problem.planExists", i18n.Args{"provider": p.Name}))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return WarmupPlanView{}, err
	}
	target := in.TargetVolume
	if target <= 0 {
		target = 48000
		var largest int64
		if err := s.Pool.QueryRow(ctx, `SELECT coalesce(max(audience_count), 0) FROM campaigns WHERE workspace_id = $1 AND deleted_at IS NULL AND origin = ''`, ws.ID).Scan(&largest); err == nil && largest > 0 {
			target = largest
		}
	}
	ip := strings.TrimSpace(in.IP)
	if ip == "" && len(p.Ips) > 0 {
		ip = p.Ips[0]
	}
	sched := Ramp(in.Days, 200, target)
	sb, _ := json.Marshal(sched)
	loc := state.Location(ws.Timezone)
	start := dayUTC(dateIn(s.now(), loc))
	var by *uuid.UUID
	if pr := api.PrincipalFrom(ctx); pr != nil && pr.UserID != uuid.Nil {
		u := pr.UserID
		by = &u
	}
	id := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		overflow := in.Overflow == nil || *in.Overflow
		if err := q.WarmupInsertPlan(ctx, store.WarmupInsertPlanParams{ID: id, ProviderID: p.ID, Ip: ip, StartDate: start, Days: int32(in.Days), Schedule: sb,
			EngagedFirst: in.EngagedFirst, SlowdownBounce: in.GuardBounce, SlowdownComplaint: in.GuardComplaint, GuardAction: in.GuardAction,
			SpreadMailbox: in.SpreadMailbox, Overflow: overflow, CreatedBy: by, CreatedAt: s.now()}); err != nil {
			return err
		}
		// The provider's Broadcast share starts where day 1 puts it (10%),
		// as the plan promises, when it already shares the route.
		if share := sched[0].Share; share > 0 {
			if _, err := s.raiseShare(ctx, tx, *ws, p.ID, share); err != nil {
				return err
			}
		}
		order := "engaged"
		if !in.EngagedFirst {
			order = "random"
		}
		body := keep("log.started", with(actor(ctx), "days", in.Days, "order", order))
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: id, OccurredAt: s.now(), Kind: "started",
			Body: body.English, Texts: textsJSON(map[string]i18n.Text{"body": body.Text})}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE providers SET status = 'warming_up', status_detail = $2 WHERE id = $1 AND status IN ('healthy', 'sandbox')`, p.ID, fmt.Sprintf("Warming up · day 1 of %d", in.Days)); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "warmup.create", TargetType: "warmup_plan", TargetID: id.String(), TargetName: p.Name,
			After: map[string]any{"days": in.Days, "ip": ip, "target": target, "settings": in.WarmupSettings}})
	})
	if err != nil {
		return WarmupPlanView{}, err
	}
	return s.planView(ctx, ws, id)
}

// Allowance is what a warming provider may send today: the workspace's
// day, whatever clock the caller keeps (PR-8). The sending pipeline sends
// no more than Cap-Sent messages (per mailbox provider when Spread) at Rate
// per second, to the listed engagement tiers, and nothing while Paused.
// Every stream counts toward the cap.
type Allowance struct {
	PlanID uuid.UUID
	Cap    int64
	Sent   int64
	Rate   float64
	Tiers  []string
	Spread bool
	// MailboxCaps and MailboxSent are today's cap and sends per mailbox
	// provider (gmail, outlook, yahoo, icloud, other) when Spread.
	MailboxCaps map[string]int64
	MailboxSent map[string]int64
	// Overflow: what's over the cap goes through a warm provider.
	Overflow     bool
	EngagedFirst bool
	Paused       bool
	Held         bool
	Day          int
	Days         int
	// NextDay is when the workspace's next day starts (a share held at
	// the cap resumes then).
	NextDay time.Time
}

// Left is what the provider may still send today, overall.
func (a Allowance) Left() int64 {
	if a.Paused {
		return 0
	}
	return max(a.Cap-a.Sent, 0)
}

// MailboxLeft is what the provider may still send today to one mailbox
// provider (the overall room when the plan doesn't spread the cap).
func (a Allowance) MailboxLeft(mbp string) int64 {
	left := a.Left()
	if !a.Spread || a.MailboxCaps == nil {
		return left
	}
	k := MailboxOf(mbp)
	return min(left, max(a.MailboxCaps[k]-a.MailboxSent[k], 0))
}

// Today is a provider's warm-up allowance today for vetting it as a
// failover or move target (state.Vet), nil when no plan is running.
func Today(ctx context.Context, q *store.Queries, providerID uuid.UUID, now time.Time, loc *time.Location) *state.WarmupToday {
	a, ok, err := AllowanceFor(ctx, q, providerID, now, loc)
	if err != nil || !ok {
		return nil
	}
	return &state.WarmupToday{Cap: a.Cap, Sent: a.Sent, Paused: a.Paused}
}

// AllowanceFor returns a provider's warm-up allowance; ok is false when the
// provider has no running plan. The day is the workspace's: loc is ignored
// and kept for callers that pass theirs.
func AllowanceFor(ctx context.Context, q *store.Queries, providerID uuid.UUID, now time.Time, _ *time.Location) (Allowance, bool, error) {
	r, err := q.WarmupAllowancePlan(ctx, providerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Allowance{}, false, nil
	}
	if err != nil {
		return Allowance{}, false, err
	}
	loc := state.Location(r.WorkspaceTimezone)
	today := dateIn(now, loc)
	plan := ParsePlan(r.Schedule, r.Progress)
	day := min(max(DayOn(r.StartDate, pausedAt(r.Status, r.PausedAt), now, loc), 1), int(r.Days))
	capToday := Hold{Cap: r.HeldCap}.CapFor(plan.PlannedCap(day))
	a := Allowance{PlanID: r.ID, Cap: capToday, Rate: plan.Rate(day), Tiers: plan.TiersOn(day), Spread: r.SpreadMailbox, Overflow: r.Overflow,
		EngagedFirst: r.EngagedFirst, Paused: r.Status == "paused", Held: r.HeldCap != nil, Day: day, Days: int(r.Days), NextDay: today.AddDate(0, 0, 1)}
	if r.HeldCap != nil {
		a.Rate = RateFor(capToday)
	}
	if !r.EngagedFirst {
		a.Tiers = Tiers
	}
	sent, err := q.WarmupSentSince(ctx, store.WarmupSentSinceParams{ProviderID: &providerID, Since: today})
	if err != nil {
		return a, true, err
	}
	a.MailboxSent = map[string]int64{}
	for _, x := range sent {
		a.MailboxSent[MailboxOf(x.MailboxProvider)] += x.N
		a.Sent += x.N
	}
	if a.Spread {
		a.MailboxCaps = MailboxCaps(capToday)
	}
	return a, true, nil
}
