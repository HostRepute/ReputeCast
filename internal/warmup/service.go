package warmup

import (
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
	"github.com/hostrepute/reputecast/internal/providers/state"
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
	Days           int     `json:"days" enum:"14,21,30"`
	EngagedFirst   bool    `json:"engagedFirst"`
	GuardBounce    float64 `json:"guardBounce" minimum:"0.5" maximum:"10"`
	GuardComplaint float64 `json:"guardComplaint" minimum:"0.01" maximum:"1"`
	GuardAction    string  `json:"guardAction" enum:"hold,step_back,pause"`
	SpreadMailbox  bool    `json:"spreadMailbox"`
}

// WarmupCampaignUse is a scheduled campaign's share on a planned day.
type WarmupCampaignUse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// WarmupDay is one day of the ramp.
type WarmupDay struct {
	Day        int       `json:"day"`
	Date       time.Time `json:"date"`
	Cap        int64     `json:"cap"`
	PlannedCap int64     `json:"plannedCap"`
	Held       bool      `json:"held"`
	Sent       *int64    `json:"sent,omitempty"`
	Bounce     *float64  `json:"bounce,omitempty"`
	Complaint  *float64  `json:"complaint,omitempty"`
	Tiers      []string  `json:"tiers"`
	Share      int       `json:"share"`
	// ShareChange is true on the day the Broadcast share steps up.
	ShareChange bool                `json:"shareChange"`
	Uses        []WarmupCampaignUse `json:"uses"`
	// Status: done, slowdown, resumed, today, planned.
	Status string `json:"status" enum:"done,slowdown,resumed,today,planned"`
	// Eligible marks the day the provider becomes a Broadcast failover target.
	Eligible bool `json:"eligible"`
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

// WarmupSlowdown summarizes the last slowdown ("day 6 held, bounces 2.3%").
type WarmupSlowdown struct {
	Day    int     `json:"day"`
	Bounce float64 `json:"bounce"`
}

// WarmupPlanView is a warm-up plan with its progress.
type WarmupPlanView struct {
	ID            string           `json:"id"`
	ProviderID    string           `json:"providerId"`
	Provider      string           `json:"provider"`
	Short         string           `json:"short"`
	IP            string           `json:"ip"`
	Status        string           `json:"status" enum:"active,slowed,paused,done"`
	StatusDetail  string           `json:"statusDetail"`
	Day           int              `json:"day"`
	Days          int              `json:"days"`
	StartDate     time.Time        `json:"startDate"`
	EndDate       time.Time        `json:"endDate"`
	Progress      float64          `json:"progress" doc:"Percent of the plan's days"`
	TodayCap      int64            `json:"todayCap"`
	SentToday     int64            `json:"sentToday"`
	SentTotal     int64            `json:"sentTotal"`
	PlannedToDate int64            `json:"plannedToDate"`
	BounceRate    *float64         `json:"bounceRate,omitempty"`
	ComplaintRate *float64         `json:"complaintRate,omitempty"`
	Slowdowns     int              `json:"slowdowns"`
	LastSlowdown  *WarmupSlowdown  `json:"lastSlowdown,omitempty"`
	RateToday     float64          `json:"rateToday"`
	NextRate      *WarmupStep      `json:"nextRate,omitempty"`
	Schedule      []WarmupDay      `json:"schedule"`
	Settings      WarmupSettings   `json:"settings"`
	Log           []WarmupLogEntry `json:"log"`
	CreatedBy     string           `json:"createdBy"`
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

func (s *Service) view(ctx context.Context, ws *store.Workspace, r store.WarmupListPlansRow) (WarmupPlanView, error) {
	loc := state.Location(ws.Timezone)
	now := s.now()
	plan := ParsePlan(r.Schedule, r.Progress)
	start := time.Date(r.StartDate.Year(), r.StartDate.Month(), r.StartDate.Day(), 0, 0, 0, 0, loc)
	day := state.WarmupDay(r.StartDate, now, loc)
	days := int(r.Days)
	v := WarmupPlanView{ID: r.ID.String(), ProviderID: r.ProviderID.String(), Provider: r.ProviderName, Short: state.Short(r.ProviderName, r.Adapter), IP: r.Ip,
		Status: r.Status, StatusDetail: r.StatusDetail, Day: min(max(day, 1), days), Days: days, StartDate: start, EndDate: start.AddDate(0, 0, days-1),
		Schedule: []WarmupDay{}, Log: []WarmupLogEntry{}, CreatedBy: r.CreatedByName,
		Settings: WarmupSettings{Days: days, EngagedFirst: r.EngagedFirst, GuardBounce: r.SlowdownBounce, GuardComplaint: r.SlowdownComplaint, GuardAction: r.GuardAction, SpreadMailbox: r.SpreadMailbox}}
	if r.Status == "done" {
		v.Day = days
	}
	v.Progress = math.Round(float64(v.Day)/float64(days)*1000) / 10
	today := dateIn(now, loc)
	stats, err := s.Q.WarmupDailyStats(ctx, store.WarmupDailyStatsParams{ProviderID: r.ProviderID, Since: dayUTC(start), Until: dayUTC(today)})
	if err != nil {
		return v, err
	}
	byDate := map[string]store.WarmupDailyStatsRow{}
	for _, st := range stats {
		byDate[st.Day.Format("2006-01-02")] = st
	}
	endAt := start.AddDate(0, 0, days)
	uses, err := s.Q.WarmupPlannedCampaignUse(ctx, store.WarmupPlannedCampaignUseParams{ProviderID: r.ProviderID, Since: &today, Until: &endAt})
	if err != nil {
		return v, err
	}
	var weekSent, weekBounced, weekDelivered, weekComplained int64
	prevShare := 0
	for d := 1; d <= days; d++ {
		date := start.AddDate(0, 0, d-1)
		wd := WarmupDay{Day: d, Date: date, Cap: plan.Cap(d), PlannedCap: plan.PlannedCap(d), Tiers: plan.TiersOn(d), Share: plan.Share(d), Uses: []WarmupCampaignUse{}}
		wd.ShareChange = d > 1 && wd.Share != prevShare
		prevShare = wd.Share
		wd.Eligible = d == days
		res, has := plan.Result(d)
		wd.Held = has && res.Held
		switch {
		case date.Before(today) && has && res.Sent > 0 || date.Before(today) && has && !res.Held:
			sent, b, c := res.Sent, res.Bounce, res.Complaint
			wd.Sent, wd.Bounce, wd.Complaint = &sent, &b, &c
			wd.Status = "done"
			if res.Slowed {
				wd.Status = "slowdown"
			} else if res.Held {
				wd.Status = "resumed"
			}
		case date.Before(today):
			st := byDate[date.Format("2006-01-02")]
			sent := st.Sent
			wd.Sent = &sent
			if st.Sent > 0 {
				b := round1(float64(st.Bounced) / float64(st.Sent) * 100)
				wd.Bounce = &b
			}
			if st.Delivered > 0 {
				c := round2(float64(st.Complained) / float64(st.Delivered) * 100)
				wd.Complaint = &c
			}
			wd.Status = "done"
		case date.Equal(today):
			st := byDate[date.Format("2006-01-02")]
			sent := st.Sent
			wd.Sent = &sent
			wd.Status = "today"
			v.TodayCap, v.SentToday = wd.Cap, st.Sent
		default:
			wd.Status = "planned"
		}
		if wd.Status == "slowdown" {
			v.Slowdowns++
			v.LastSlowdown = &WarmupSlowdown{Day: d, Bounce: *wd.Bounce}
		}
		if date.Before(today) {
			if wd.Sent != nil {
				v.SentTotal += *wd.Sent
			}
			v.PlannedToDate += wd.Cap
		}
		// The last seven days, today included, as on Sending providers.
		if !date.Before(today.AddDate(0, 0, -6)) && !date.After(today) {
			st := byDate[date.Format("2006-01-02")]
			weekSent += st.Sent
			weekBounced += st.Bounced
			weekDelivered += st.Delivered
			weekComplained += st.Complained
		}
		for _, u := range uses {
			if u.ScheduledAt != nil && dateIn(*u.ScheduledAt, loc).Equal(date) {
				wd.Uses = append(wd.Uses, WarmupCampaignUse{Name: u.Name, Count: u.Planned})
			}
		}
		v.Schedule = append(v.Schedule, wd)
	}
	if weekSent > 0 {
		b := round1(float64(weekBounced) / float64(weekSent) * 100)
		v.BounceRate = &b
	}
	if weekDelivered > 0 {
		c := round2(float64(weekComplained) / float64(weekDelivered) * 100)
		v.ComplaintRate = &c
	}
	v.RateToday = plan.Rate(v.Day)
	for d := v.Day + 1; d <= days; d++ {
		if r := plan.Rate(d); r > v.RateToday {
			v.NextRate = &WarmupStep{Day: d, Value: r}
			break
		}
	}
	log, err := s.Q.WarmupLog(ctx, store.WarmupLogParams{PlanID: r.ID, Lim: 20})
	if err != nil {
		return v, err
	}
	for _, e := range log {
		v.Log = append(v.Log, WarmupLogEntry{At: e.OccurredAt, Kind: e.Kind, Title: e.Title, Body: e.Body})
	}
	return v, nil
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

// UpdateSettings changes a plan's length, ordering and guards. A new length
// rebuilds the schedule from the same first and last volumes; days already
// sent keep their results.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, id string, in WarmupSettings) (WarmupPlanView, error) {
	r, err := s.plan(ctx, ws, id)
	if err != nil {
		return WarmupPlanView{}, err
	}
	if r.Status == "done" {
		return WarmupPlanView{}, api.Conflict("This warm-up is complete. Start a new plan to warm up again.")
	}
	plan := ParsePlan(r.Schedule, r.Progress)
	sched := plan.Schedule
	if in.Days != int(r.Days) {
		first := int64(200)
		if len(sched) > 0 {
			first = sched[0].Volume
		}
		sched = Ramp(in.Days, first, lastVolume(sched))
	}
	sb, _ := json.Marshal(sched)
	before := WarmupSettings{Days: int(r.Days), EngagedFirst: r.EngagedFirst, GuardBounce: r.SlowdownBounce, GuardComplaint: r.SlowdownComplaint, GuardAction: r.GuardAction, SpreadMailbox: r.SpreadMailbox}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WarmupUpdateSettings(ctx, store.WarmupUpdateSettingsParams{ID: r.ID, Days: int32(in.Days), Schedule: sb, EngagedFirst: in.EngagedFirst,
			SlowdownBounce: in.GuardBounce, SlowdownComplaint: in.GuardComplaint, GuardAction: in.GuardAction, SpreadMailbox: in.SpreadMailbox}); err != nil {
			return err
		}
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: s.now(), Kind: "settings",
			Body: "Plan settings changed by " + actorName(ctx) + "."}); err != nil {
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

func actorName(ctx context.Context) string {
	if p := api.PrincipalFrom(ctx); p != nil && p.Name != "" {
		return p.Name
	}
	return "someone"
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// SetPaused pauses or resumes a plan.
func (s *Service) SetPaused(ctx context.Context, ws *store.Workspace, id string, paused bool) (WarmupPlanView, error) {
	r, err := s.plan(ctx, ws, id)
	if err != nil {
		return WarmupPlanView{}, err
	}
	if r.Status == "done" {
		return WarmupPlanView{}, api.Conflict("This warm-up is complete.")
	}
	status, detail, kind, body := "active", "", "resumed", "Warm-up resumed by "+actorName(ctx)+"."
	var at *time.Time
	if paused {
		now := s.now()
		status, detail, kind, body, at = "paused", "Paused by "+actorName(ctx), "paused", "Warm-up paused by "+actorName(ctx)+". No mail goes to "+r.ProviderName+" until it resumes.", &now
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WarmupSetStatus(ctx, store.WarmupSetStatusParams{ID: r.ID, Status: status, StatusDetail: detail, PausedAt: at}); err != nil {
			return err
		}
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: s.now(), Kind: kind, Body: body}); err != nil {
			return err
		}
		wsID := ws.ID
		action := "warmup.resume"
		if paused {
			action = "warmup.pause"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: action, TargetType: "warmup_plan", TargetID: r.ID.String(), TargetName: r.ProviderName,
			Before: map[string]string{"status": r.Status}, After: map[string]string{"status": status}})
	})
	if err != nil {
		return WarmupPlanView{}, err
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
		return WarmupPlanView{}, api.Invalid("body.providerId", "Pick a provider.")
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: pid})
	if err != nil {
		return WarmupPlanView{}, api.Invalid("body.providerId", "That provider is not connected to this workspace.")
	}
	if _, err := s.Q.WarmupActiveForProvider(ctx, p.ID); err == nil {
		return WarmupPlanView{}, api.Conflict(p.Name + " already has a warm-up plan. Change its settings instead.")
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
		if err := q.WarmupInsertPlan(ctx, store.WarmupInsertPlanParams{ID: id, ProviderID: p.ID, Ip: ip, StartDate: start, Days: int32(in.Days), Schedule: sb,
			EngagedFirst: in.EngagedFirst, SlowdownBounce: in.GuardBounce, SlowdownComplaint: in.GuardComplaint, GuardAction: in.GuardAction,
			SpreadMailbox: in.SpreadMailbox, CreatedBy: by, CreatedAt: s.now()}); err != nil {
			return err
		}
		order := "engaged-first"
		if !in.EngagedFirst {
			order = "random sample"
		}
		if err := q.WarmupAddEvent(ctx, store.WarmupAddEventParams{ID: newID(), PlanID: id, OccurredAt: s.now(), Kind: "started",
			Body: fmt.Sprintf("Plan started by %s: %d days, %s.", actorName(ctx), in.Days, order)}); err != nil {
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

// Allowance is what a warming provider may send today. The sending
// pipeline sends no more than Cap-Sent messages at Rate per second, to the
// listed engagement tiers, spread across mailbox providers when Spread.
type Allowance struct {
	Cap    int64
	Sent   int64
	Rate   float64
	Tiers  []string
	Spread bool
	Paused bool
	Day    int
	Days   int
}

// AllowanceFor returns a provider's warm-up allowance; ok is false when the
// provider has no running plan.
func AllowanceFor(ctx context.Context, q *store.Queries, providerID uuid.UUID, now time.Time, loc *time.Location) (Allowance, bool, error) {
	r, err := q.WarmupActiveForProvider(ctx, providerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Allowance{}, false, nil
	}
	if err != nil {
		return Allowance{}, false, err
	}
	plan := ParsePlan(r.Schedule, r.Progress)
	day := min(max(state.WarmupDay(r.StartDate, now, loc), 1), int(r.Days))
	a := Allowance{Cap: plan.Cap(day), Rate: plan.Rate(day), Tiers: plan.TiersOn(day), Spread: r.SpreadMailbox, Paused: r.Status == "paused", Day: day, Days: int(r.Days)}
	if !r.EngagedFirst {
		a.Tiers = Tiers
	}
	today := dayUTC(dateIn(now, loc))
	stats, err := q.WarmupDailyStats(ctx, store.WarmupDailyStatsParams{ProviderID: providerID, Since: today, Until: today})
	if err == nil && len(stats) == 1 {
		a.Sent = stats[0].Sent
	}
	return a, true, nil
}
