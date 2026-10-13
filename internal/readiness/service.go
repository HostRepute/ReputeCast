package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service runs Send Readiness and applies its fixes.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// Campaigns runs the composer's content and link checks and saves the
	// content changes fixes make.
	Campaigns *campaigns.Service
	// Deliverability reads blocklists, authentication and provider health.
	Deliverability Deliverability
	// Segments resolves segment members; nil until the audience domain wires it.
	Segments SegmentResolver
	// URLs builds the one-click unsubscribe link.
	URLs publicurl.Builder
	// AfterChange is told when a fix or a send setting changed a campaign.
	AfterChange AfterChange
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// ErrRunning means a run of the campaign is already in progress.
var ErrRunning = errors.New("send readiness is already running for this campaign")

// staleAfter is how long a run in progress may go without an update before
// another run takes over (the process running it died).
const staleAfter = 2 * time.Minute

// RunOptions describe a run.
type RunOptions struct {
	Trigger string
	// By is the person who asked (nil for jobs).
	By *uuid.UUID
}

// Fingerprint identifies what a run checks: content, sender, audience,
// route override, the Broadcasts routing rules and the schedule. Any change
// to them makes the latest run stale.
func Fingerprint(c store.ReadinessCampaignRow, routing json.RawMessage) string {
	h := sha256.New()
	sender := ""
	if c.SenderID != nil {
		sender = c.SenderID.String()
	}
	for _, part := range []string{c.ContentHash, sender, string(c.Audience), string(c.RouteOverride), string(routing), string(c.Schedule)} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// routingKey is the Broadcasts routing as it stands, for fingerprints.
func (s *Service) routingKey(ctx context.Context, q *store.Queries, wsID uuid.UUID) json.RawMessage {
	rows, err := q.RoutingListRules(ctx, wsID)
	if err != nil {
		return nil
	}
	type rule struct {
		Conditions json.RawMessage `json:"c"`
		Splits     json.RawMessage `json:"s"`
		Enabled    bool            `json:"e"`
	}
	var rules []rule
	for _, r := range rows {
		if r.Stream == "broadcast" {
			rules = append(rules, rule{r.Conditions, r.Splits, r.Enabled})
		}
	}
	b, _ := json.Marshal(rules)
	return b
}

// Campaign loads a campaign as a run sees it.
func (s *Service) Campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.ReadinessCampaignRow, error) {
	c, err := s.Q.ReadinessCampaign(ctx, store.ReadinessCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return c, api.MapError(err, "Campaign")
	}
	return c, nil
}

// CurrentFingerprint is the fingerprint of the campaign as it stands.
func (s *Service) CurrentFingerprint(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow) string {
	return Fingerprint(c, s.routingKey(ctx, s.Q, ws.ID))
}

// Run runs every check on a campaign and stores the result. It returns
// ErrRunning when a run is already in progress.
func (s *Service) Run(ctx context.Context, ws *store.Workspace, id uuid.UUID, o RunOptions) (ReadinessRun, error) {
	c, err := s.Campaign(ctx, ws, id)
	if err != nil {
		return ReadinessRun{}, err
	}
	if o.Trigger == "" {
		o.Trigger = TriggerManual
	}
	started := s.now()
	runID := uuid.Must(uuid.NewV7())
	groups := make([]ReadinessProgressGroup, 0, len(runOrder))
	for _, g := range runOrder {
		groups = append(groups, ReadinessProgressGroup{Key: g, Title: groupTitles[g], Status: "queued", Text: RunningText(g)})
	}
	groups[0].Status = "running"
	gb, _ := json.Marshal(groups)
	if _, err := s.Q.ReadinessStartProgress(ctx, store.ReadinessStartProgressParams{CampaignID: id, RunID: runID, Trigger: o.Trigger, StartedBy: o.By,
		Now: started, Groups: gb, Total: 25, StaleBefore: started.Add(-staleAfter)}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReadinessRun{}, ErrRunning
		}
		return ReadinessRun{}, err
	}
	// The progress row goes whatever happens; a detached context so a
	// cancelled request still clears it.
	defer func() {
		_ = s.Q.ReadinessEndProgress(context.WithoutCancel(ctx), store.ReadinessEndProgressParams{CampaignID: id, RunID: runID})
	}()
	fingerprint := s.CurrentFingerprint(ctx, ws, c)
	facts, err := s.gather(ctx, ws, c, func(st stage, f Facts) { s.progress(ctx, id, runID, groups, st, f) })
	if err != nil {
		return ReadinessRun{}, err
	}
	res := Assemble(facts)
	run := ReadinessRun{ID: runID.String(), CampaignID: id.String(), Trigger: o.Trigger, RanAt: s.now(), Version: int(c.Version), ReadinessResult: res, Fingerprint: fingerprint}
	run.DurationMs = int(run.RanAt.Sub(started) / time.Millisecond)
	if err := s.store(ctx, ws, c, run, o.By); err != nil {
		return ReadinessRun{}, err
	}
	if o.By != nil {
		if u, err := s.Q.GetUser(ctx, *o.By); err == nil {
			run.RanBy = u.Name
		}
	}
	return run, nil
}

// progress records which groups finished after a stage.
func (s *Service) progress(ctx context.Context, id, runID uuid.UUID, groups []ReadinessProgressGroup, st stage, f Facts) {
	res := Assemble(f)
	byKey := map[string]ReadinessGroup{}
	for _, g := range res.Groups {
		byKey[g.Key] = g
	}
	doneGroups := map[string]bool{}
	for _, g := range runOrder {
		doneGroups[g] = true
		if g == string(st) {
			break
		}
	}
	if st == stagePrepare {
		doneGroups = map[string]bool{}
	}
	done, total := 0, 0
	running := false
	for i := range groups {
		g := &groups[i]
		rg, ok := byKey[g.Key]
		if ok {
			g.Total = rg.Total
			total += rg.Total
		}
		switch {
		case doneGroups[g.Key] && ok:
			g.Status, g.Text, g.Done = rg.Status, firstNonEmpty(rg.Summary, rg.Caption), rg.Total
			done += rg.Total
		case !running:
			g.Status, running = "running", true
			if ok {
				g.Text = rg.Running
			}
		default:
			g.Status = "queued"
			if ok {
				g.Text = rg.Running
			}
		}
	}
	gb, _ := json.Marshal(groups)
	_ = s.Q.ReadinessUpdateProgress(ctx, store.ReadinessUpdateProgressParams{Groups: gb, Done: int32(done), Total: int32(max(total, done)), Now: s.now(), CampaignID: id, RunID: runID})
}

// store saves a finished run, refreshes the campaign's planned route and
// audience count, and keeps the history short.
func (s *Service) store(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run ReadinessRun, by *uuid.UUID) error {
	checks, err := json.Marshal(struct {
		Verdict    string           `json:"verdict"`
		Counts     ReadinessCounts  `json:"counts"`
		Groups     []ReadinessGroup `json:"groups"`
		Checks     []ReadinessCheck `json:"checks"`
		Incomplete bool             `json:"incomplete"`
		Problem    string           `json:"problem,omitempty"`
	}{run.Verdict, run.Counts, run.Groups, run.Checks, run.Incomplete, run.Problem})
	if err != nil {
		return err
	}
	est, err := json.Marshal(run.Estimates)
	if err != nil {
		return err
	}
	id := uuid.MustParse(run.ID)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessInsertRun(ctx, store.ReadinessInsertRunParams{ID: id, CampaignID: c.ID, Trigger: run.Trigger, Verdict: run.Verdict, Checks: checks,
			Estimates: est, RanAt: run.RanAt, DurationMs: int32(run.DurationMs), RanBy: by, CampaignVersion: c.Version, Fingerprint: run.Fingerprint}); err != nil {
			return err
		}
		if editable(c.Status) {
			if err := q.ReadinessReplaceRoutes(ctx, c.ID); err != nil {
				return err
			}
			for _, sh := range run.Estimates.Route.Shares {
				pid, err := uuid.Parse(sh.ProviderID)
				if err != nil || sh.Recipients == 0 {
					continue
				}
				if err := q.ReadinessInsertRoute(ctx, store.ReadinessInsertRouteParams{CampaignID: c.ID, ProviderID: pid, Planned: sh.Recipients,
					CostMicros: int64(sh.Cost * 1_000_000)}); err != nil {
					return err
				}
			}
			if !ParseAudience(c.Audience).Empty() {
				final := run.Estimates.Audience.Final
				if err := q.ReadinessSetAudienceCount(ctx, store.ReadinessSetAudienceCountParams{ID: c.ID, AudienceCount: &final}); err != nil {
					return err
				}
			}
		}
		if err := q.ReadinessPruneRuns(ctx, c.ID); err != nil {
			return err
		}
		if run.Trigger != TriggerManual && run.Trigger != TriggerSendTime {
			return nil
		}
		action := "readiness.run"
		if run.Trigger == TriggerSendTime {
			action = "readiness.send_time_check"
		}
		var actor *audit.Actor
		if by == nil {
			actor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name, Actor: actor,
			After: map[string]any{"verdict": run.Verdict, "passed": run.Counts.Passed, "warnings": run.Counts.Warnings, "blocks": run.Counts.Blocks, "unknown": run.Counts.Unknown},
			Metadata: map[string]any{"run_id": run.ID, "trigger": run.Trigger, "version": c.Version}})
	})
}

// editable statuses are those before the send starts.
func editable(status string) bool {
	switch status {
	case "draft", "approval", "approved", "scheduled":
		return true
	}
	return false
}

// Latest returns the latest run of a campaign, nil when it never ran.
func (s *Service) Latest(ctx context.Context, campaignID uuid.UUID) (*ReadinessRun, error) {
	r, err := s.Q.ReadinessLatestRun(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	run := decodeRun(r.ID, r.CampaignID, r.Trigger, r.Verdict, r.Checks, r.Estimates, r.RanAt, r.DurationMs, r.CampaignVersion, r.Fingerprint, r.UnknownAcceptedAt, r.RanByName, r.AcceptedByName)
	return &run, nil
}

// Get returns one run of a campaign.
func (s *Service) Get(ctx context.Context, campaignID, runID uuid.UUID) (*ReadinessRun, error) {
	r, err := s.Q.ReadinessGetRun(ctx, store.ReadinessGetRunParams{ID: runID, CampaignID: campaignID})
	if err != nil {
		return nil, api.MapError(err, "Readiness run")
	}
	run := decodeRun(r.ID, r.CampaignID, r.Trigger, r.Verdict, r.Checks, r.Estimates, r.RanAt, r.DurationMs, r.CampaignVersion, r.Fingerprint, r.UnknownAcceptedAt, r.RanByName, r.AcceptedByName)
	return &run, nil
}

// DecodeRun turns stored columns into a run (the demo stores runs too).
func decodeRun(id, campaign uuid.UUID, trigger, verdict string, checks, estimates json.RawMessage, ranAt time.Time, ms, version int32,
	fingerprint string, accepted *time.Time, ranBy, acceptedBy string) ReadinessRun {
	run := ReadinessRun{ID: id.String(), CampaignID: campaign.String(), Trigger: trigger, RanAt: ranAt, DurationMs: int(ms), Version: int(version),
		Fingerprint: fingerprint, UnknownAcceptedAt: accepted, RanBy: ranBy, UnknownAcceptedBy: acceptedBy}
	_ = json.Unmarshal(checks, &run.ReadinessResult)
	_ = json.Unmarshal(estimates, &run.Estimates)
	run.Verdict = verdict
	if run.Checks == nil {
		run.Checks = []ReadinessCheck{}
	}
	if run.Groups == nil {
		run.Groups = []ReadinessGroup{}
	}
	for i := range run.Checks {
		if run.Checks[i].Fixes == nil {
			run.Checks[i].Fixes = []ReadinessFix{}
		}
	}
	normalizeEstimates(&run.Estimates)
	return run
}

func normalizeEstimates(e *ReadinessEstimates) {
	if e.Audience.Rows == nil {
		e.Audience.Rows = []ReadinessAudienceRow{}
	}
	if e.Audience.Lists == nil {
		e.Audience.Lists = []string{}
	}
	if e.Route.Shares == nil {
		e.Route.Shares = []ReadinessRouteShare{}
	}
	if e.Throttling == nil {
		e.Throttling = []ReadinessThrottle{}
	}
	if e.Route.Stream == "" {
		e.Route.Stream = "broadcast"
	}
	if e.Route.Policy == "" {
		e.Route.Policy = "pause_and_ask"
	}
}

// Progress returns the run in progress, nil when none is.
func (s *Service) Progress(ctx context.Context, campaignID uuid.UUID) (*ReadinessProgress, error) {
	p, err := s.Q.ReadinessGetProgress(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.UpdatedAt.Before(s.now().Add(-staleAfter)) {
		return nil, nil
	}
	out := &ReadinessProgress{RunID: p.RunID.String(), Trigger: p.Trigger, StartedAt: p.StartedAt, StartedBy: p.StartedByName, Done: int(p.Done), Total: int(p.Total),
		Groups: []ReadinessProgressGroup{}}
	_ = json.Unmarshal(p.Groups, &out.Groups)
	return out, nil
}

// AcceptUnknown records "Continue with provider data only" on the latest run.
func (s *Service) AcceptUnknown(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, runID uuid.UUID) (ReadinessRun, error) {
	c, err := s.Campaign(ctx, ws, campaignID)
	if err != nil {
		return ReadinessRun{}, err
	}
	latest, err := s.Latest(ctx, campaignID)
	if err != nil {
		return ReadinessRun{}, err
	}
	if latest == nil || latest.ID != runID.String() {
		return ReadinessRun{}, api.Conflict("The checks ran again since you looked. Review the new results.")
	}
	if !latest.Incomplete {
		return *latest, nil
	}
	now := s.now()
	unknown := []string{}
	for _, ch := range latest.Checks {
		if ch.Status == Unknown {
			unknown = append(unknown, ch.Key)
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessAcceptUnknown(ctx, store.ReadinessAcceptUnknownParams{AcceptedAt: &now, AcceptedBy: &p.UserID, ID: runID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "readiness.unknown_accepted", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Reason: "Continue with provider data only", After: map[string]any{"skipped": unknown}, Metadata: map[string]any{"run_id": runID.String(), "problem": latest.Problem}})
	})
	if err != nil {
		return ReadinessRun{}, err
	}
	return s.mustLatest(ctx, campaignID)
}

func (s *Service) mustLatest(ctx context.Context, id uuid.UUID) (ReadinessRun, error) {
	r, err := s.Latest(ctx, id)
	if err != nil {
		return ReadinessRun{}, err
	}
	if r == nil {
		return ReadinessRun{}, api.NotFound("Readiness run")
	}
	return *r, nil
}

// runningProblem is the 409 for a second run while one is in progress.
func runningProblem() error {
	return api.Err(http.StatusConflict, "readiness_running", "Send Readiness is already running for this campaign. The results appear here when it finishes.")
}
