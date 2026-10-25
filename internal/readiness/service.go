package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/languages"
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
	// Languages gives the facts of the Languages group and checks every
	// language version; nil leaves them out.
	Languages *languages.Service
	// Accessibility runs the Accessibility group and its fixes; nil leaves
	// the group out.
	Accessibility AccessibilityChecker
	// Conditional gives the Conditional content group and the poll checks;
	// nil leaves them out.
	Conditional ConditionalChecker
	// Verification gives the address checks in Audience quality; nil
	// leaves them out.
	Verification VerificationChecker
	// Website gives the web checks of a campaign that publishes to the
	// website; nil leaves them out.
	Website WebsiteChecker
	// Sponsor gives the Sponsor block's checks and fixes; nil leaves them
	// out.
	Sponsor SponsorChecker
	// UpgradeOffers finds Upgrade blocks whose offer ended; nil leaves the
	// check out.
	UpgradeOffers UpgradeOfferChecker
	// Grow checks the Referral and Recommendations blocks (Content rows);
	// nil leaves them out.
	Grow GrowChecker
	// Mailbox classifies company domains from their MX records before the
	// audience is counted, so throttles and estimates use the mailbox
	// provider the send will use (internal/mailbox); nil skips it.
	Mailbox MailboxClassifier
	// DNS looks up again what Deliverability couldn't (a DNS error is
	// Unknown, never a wrong record); nil keeps the stored results.
	DNS dnscheck.Resolver
	// Links looks domains up live on the URI blocklists (the link domains'
	// lookups, shared with the composer): the return-path and tracking
	// domains Deliverability doesn't watch; nil leaves them out.
	Links contentcheck.BlocklistChecker
	// Jobs queues runs (Start); a worker runs them (RunWorker).
	Jobs func() JobInserter
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

// Campaign loads a campaign as a run sees it.
func (s *Service) Campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.ReadinessCampaignRow, error) {
	c, err := s.Q.ReadinessCampaign(ctx, store.ReadinessCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return c, api.MapError(err, "Campaign")
	}
	return c, nil
}

// CurrentFingerprint is the fingerprint of the campaign as it stands,
// language versions included.
func (s *Service) CurrentFingerprint(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow) string {
	fp := Fingerprint(c, RoutingKey(ctx, s.Q, ws.ID))
	if s.Languages != nil {
		fp = WithLanguages(fp, s.Languages.CampaignDigest(ctx, s.Q, ws.ID, c.ID))
	}
	return fp
}

// WithLanguages adds a campaign's language versions (languages.CampaignDigest)
// to its fingerprint; a campaign without versions keeps its fingerprint.
func WithLanguages(fingerprint, digest string) string {
	if digest == "" {
		return fingerprint
	}
	sum := sha256.Sum256([]byte(fingerprint + "\x00" + digest))
	return hex.EncodeToString(sum[:])
}

// Run runs every check on a campaign now and stores the result. It returns
// ErrRunning when a run is already waiting or in progress. Jobs use it (the
// recheck before a scheduled send); requests queue a run with Start.
func (s *Service) Run(ctx context.Context, ws *store.Workspace, id uuid.UUID, o RunOptions) (ReadinessRun, error) {
	if o.Trigger == "" {
		o.Trigger = TriggerManual
	}
	if _, err := s.Campaign(ctx, ws, id); err != nil {
		return ReadinessRun{}, err
	}
	runID := uuid.Must(uuid.NewV7())
	if _, err := s.claim(ctx, s.Q, id, runID, o); err != nil {
		return ReadinessRun{}, err
	}
	return s.execute(ctx, ws, id, runID, o)
}

// Start queues a run of a campaign's checks (SR-8): the campaign's progress
// row is claimed at once, so a second start is refused (ErrRunning) while a
// run waits or runs, and a worker runs the checks in the background. Nothing
// else is done inside the request.
func (s *Service) Start(ctx context.Context, ws *store.Workspace, id uuid.UUID, o RunOptions) (*ReadinessProgress, error) {
	if o.Trigger == "" {
		o.Trigger = TriggerManual
	}
	if _, err := s.Campaign(ctx, ws, id); err != nil {
		return nil, err
	}
	if s.Jobs == nil {
		return nil, errors.New("send readiness: no job queue in this process")
	}
	runID := uuid.Must(uuid.NewV7())
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := s.claim(ctx, store.New(tx), id, runID, o); err != nil {
			return err
		}
		_, err := s.Jobs().InsertTx(ctx, tx, RunArgs{CampaignID: id, WorkspaceID: ws.ID, RunID: runID, Trigger: o.Trigger, By: o.By}, nil)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.Progress(ctx, id)
}

// claim takes the campaign's progress row for a run, its groups queued.
func (s *Service) claim(ctx context.Context, q *store.Queries, id, runID uuid.UUID, o RunOptions) ([]ReadinessProgressGroup, error) {
	started := s.now()
	groups := s.queuedGroups()
	gb, _ := json.Marshal(groups)
	if _, err := q.ReadinessStartProgress(ctx, store.ReadinessStartProgressParams{CampaignID: id, RunID: runID, Trigger: o.Trigger, StartedBy: o.By,
		Now: started, Groups: gb, Total: 0, StaleBefore: started.Add(-staleAfter)}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRunning
		}
		return nil, err
	}
	return groups, nil
}

// queuedGroups are the groups of a run before it starts, in run order.
func (s *Service) queuedGroups() []ReadinessProgressGroup {
	groups := make([]ReadinessProgressGroup, 0, len(runOrder))
	for _, g := range runOrder {
		if g == GroupAccessibility && s.Accessibility == nil {
			continue
		}
		if g == GroupConditional && s.Conditional == nil {
			continue
		}
		t := &progressTexts{Title: groupTitle(g), Text: runningText(g)}
		groups = append(groups, ReadinessProgressGroup{Key: g, Title: t.Title.en(), Status: "queued", Text: t.Text.en(), texts: t})
	}
	return groups
}

// errTakenOver means another run took the campaign's progress row: this
// one stops without storing anything.
var errTakenOver = errors.New("send readiness: another run took over")

// execute runs the checks of a claimed run and stores the result. The
// progress row goes when the run is stored, and says why when it fails.
func (s *Service) execute(ctx context.Context, ws *store.Workspace, id, runID uuid.UUID, o RunOptions) (ReadinessRun, error) {
	started := s.now()
	if _, err := s.Q.ReadinessClaimProgress(ctx, store.ReadinessClaimProgressParams{Now: started, CampaignID: id, RunID: runID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReadinessRun{}, errTakenOver
		}
		return ReadinessRun{}, err
	}
	run, err := s.runClaimed(ctx, ws, id, runID, o, started)
	// A detached context, so a cancelled request or job still settles the row.
	settle := context.WithoutCancel(ctx)
	if err != nil {
		now := s.now()
		_ = s.Q.ReadinessFailProgress(settle, store.ReadinessFailProgressParams{Now: &now, Error: failureText(err), CampaignID: id, RunID: runID})
		return ReadinessRun{}, err
	}
	_ = s.Q.ReadinessEndProgress(settle, store.ReadinessEndProgressParams{CampaignID: id, RunID: runID})
	return run, nil
}

// failureText is what a failed run says, in a sentence (its English is
// shown as it is: it comes from the code, not from a message).
func failureText(err error) string {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return sentence(capitalize(msg))
}

func (s *Service) runClaimed(ctx context.Context, ws *store.Workspace, id, runID uuid.UUID, o RunOptions, started time.Time) (ReadinessRun, error) {
	c, err := s.Campaign(ctx, ws, id)
	if err != nil {
		return ReadinessRun{}, err
	}
	groups := s.queuedGroups()
	groups[0].Status = "running"
	pw := &progressWriter{s: s, ctx: ctx, campaign: id, run: runID, groups: groups}
	pw.write(true)
	fingerprint := s.CurrentFingerprint(ctx, ws, c)
	// A run is stored for everyone: what other domains tell it is in
	// English, whoever started it; its texts keep their messages. Link
	// probing and lookups tell the progress how far they got.
	gctx := withCounting(contentcheck.WithProgress(i18n.WithLang(ctx, i18n.Default), pw.step), pw.counting)
	facts, err := s.gather(gctx, ws, c, pw.stage)
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

// progressWriter keeps a run's progress row up to date: groups tick in as
// their stage finishes, and the long steps (counting the audience,
// requesting links, looking link domains up) say how far they got. Steps
// report from several goroutines; writes are at most one every 400 ms,
// except when a stage finishes.
type progressWriter struct {
	s        *Service
	ctx      context.Context
	campaign uuid.UUID
	run      uuid.UUID

	mu      sync.Mutex
	groups  []ReadinessProgressGroup
	done    int
	total   int
	written time.Time
}

func (p *progressWriter) group(key string) *ReadinessProgressGroup {
	for i := range p.groups {
		if p.groups[i].Key == key {
			return &p.groups[i]
		}
	}
	return nil
}

// write stores the groups (when forced, or 400 ms after the last write).
func (p *progressWriter) write(force bool) {
	now := p.s.now()
	if !force && now.Sub(p.written) < 400*time.Millisecond {
		return
	}
	p.written = now
	shown := make([]ReadinessProgressGroup, 0, len(p.groups))
	for _, g := range p.groups {
		if !g.hidden {
			shown = append(shown, g)
		}
	}
	gb, _ := json.Marshal(shown)
	_ = p.s.Q.ReadinessUpdateProgress(p.ctx, store.ReadinessUpdateProgressParams{Groups: gb, Done: int32(p.done), Total: int32(max(p.total, p.done)), Now: now,
		CampaignID: p.campaign, RunID: p.run})
}

// step records how far a long step got (contentcheck.Progress).
func (p *progressWriter) step(st string, done, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	g := p.group(GroupLinks)
	if g == nil || g.Status != "running" {
		return
	}
	switch st {
	case contentcheck.StageLinks:
		g.Unit, g.Count, g.Of = "links", int64(done), int64(total)
		g.setText(tx("running.linksFetching", nil).en(), tx("running.linksFetching", nil))
	case contentcheck.StageLinkDomains:
		g.Unit, g.Count, g.Of = "domains", int64(done), int64(total)
		g.setText(tx("running.linkDomains", A{"lists": ArgAnd(URIBlocklists)}).en(), tx("running.linkDomains", A{"lists": ArgAnd(URIBlocklists)}))
	default:
		return
	}
	p.write(false)
}

// counting records how many contacts the audience count went through.
func (p *progressWriter) counting(done, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	g := p.group(GroupAudience)
	if g == nil {
		return
	}
	g.Status, g.Unit, g.Count, g.Of = "running", "contacts", done, total
	g.setText(tx("running.audienceCounting", nil).en(), tx("running.audienceCounting", nil))
	p.write(done >= total)
}

// stage records which groups finished after a stage.
func (p *progressWriter) stage(st stage, f Facts) {
	res := Assemble(f)
	p.mu.Lock()
	defer p.mu.Unlock()
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
	for i := range p.groups {
		g := &p.groups[i]
		rg, ok := byKey[g.Key]
		if ok {
			g.Total = rg.Total
			total += rg.Total
		}
		switch {
		case doneGroups[g.Key] && ok:
			g.Status, g.Done, g.Unit, g.Count, g.Of = rg.Status, rg.Total, "", 0, 0
			gt := rg.texts
			if gt == nil {
				gt = &groupTexts{}
			}
			g.setText(rg.Summary, gt.Summary)
			if strings.TrimSpace(rg.Summary) == "" {
				g.setText(rg.Caption, gt.Caption)
			}
			done += rg.Total
		case doneGroups[g.Key] && st != stagePrepare:
			// A group with nothing to check (Languages for an email in one
			// language) leaves the list once its stage ran.
			g.hidden = true
		case !running:
			g.Status, running = "running", true
			if ok {
				g.setText(rg.Running, runningOf(rg))
			}
		default:
			g.Status, g.Unit, g.Count, g.Of = "queued", "", 0, 0
			if ok {
				g.setText(rg.Running, runningOf(rg))
			}
		}
	}
	p.done, p.total = done, total
	p.write(true)
}

// store saves a finished run, refreshes the campaign's planned route and
// audience count, and keeps the history short.
func (s *Service) store(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run ReadinessRun, by *uuid.UUID) error {
	checks, est, err := EncodeResult(run.ReadinessResult)
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
		switch {
		case by == nil:
			actor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}
		case audit.PrincipalFrom(ctx) == nil:
			// A run a person asked for, which a worker ran: it is theirs.
			name := ""
			if u, err := q.GetUser(ctx, *by); err == nil {
				name = u.Name
			}
			actor = &audit.Actor{Type: authz.ActorUser, ID: by, Name: name}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name, Actor: actor,
			After:    map[string]any{"verdict": run.Verdict, "passed": run.Counts.Passed, "warnings": run.Counts.Warnings, "blocks": run.Counts.Blocks, "unknown": run.Counts.Unknown},
			Metadata: map[string]any{"run_id": run.ID, "trigger": run.Trigger, "version": c.Version}})
	})
}

// resultTexts are the messages of a result's own texts.
type resultTexts struct {
	Problem Text `json:"problem,omitempty"`
}

// EncodeResult is the stored form of a result: readiness_runs.checks
// (verdict, counts, groups and checks) and readiness_runs.estimates, with
// the messages their texts are written from.
func EncodeResult(res ReadinessResult) (checks, estimates json.RawMessage, err error) {
	var texts *resultTexts
	if !res.problem.Empty() {
		texts = &resultTexts{Problem: res.problem}
	}
	checks, err = json.Marshal(struct {
		Verdict    string           `json:"verdict"`
		Counts     ReadinessCounts  `json:"counts"`
		Groups     []ReadinessGroup `json:"groups"`
		Checks     []ReadinessCheck `json:"checks"`
		Incomplete bool             `json:"incomplete"`
		Problem    string           `json:"problem,omitempty"`
		I18n       *resultTexts     `json:"i18n,omitempty"`
	}{res.Verdict, res.Counts, res.Groups, res.Checks, res.Incomplete, res.Problem, texts})
	if err != nil {
		return nil, nil, err
	}
	estimates, err = json.Marshal(res.Estimates)
	return checks, estimates, err
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
	var texts struct {
		I18n *resultTexts `json:"i18n"`
	}
	if json.Unmarshal(checks, &texts) == nil && texts.I18n != nil {
		run.problem = texts.I18n.Problem
	}
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
	for i, sh := range e.Route.Shares {
		if sh.Pricing == "" {
			// Runs stored before the model was: their price label says it,
			// in English.
			switch sh.PriceLabel {
			case "self-hosted":
				e.Route.Shares[i].Pricing = "self_hosted"
			case "plan":
				e.Route.Shares[i].Pricing = "plan"
			default:
				e.Route.Shares[i].Pricing = "metered"
			}
		}
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

// Progress returns the run waiting or in progress, nil when none is.
func (s *Service) Progress(ctx context.Context, campaignID uuid.UUID) (*ReadinessProgress, error) {
	p, _, err := s.progressRow(ctx, campaignID)
	return p, err
}

// State is the run waiting or in progress, or the run that failed last.
func (s *Service) State(ctx context.Context, campaignID uuid.UUID) (*ReadinessProgress, *ReadinessRunFailure, error) {
	return s.progressRow(ctx, campaignID)
}

// progressRow reads the campaign's progress row: the run waiting or in
// progress, or the run that failed last.
func (s *Service) progressRow(ctx context.Context, campaignID uuid.UUID) (*ReadinessProgress, *ReadinessRunFailure, error) {
	p, err := s.Q.ReadinessGetProgress(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if p.FailedAt != nil {
		return nil, &ReadinessRunFailure{RunID: p.RunID.String(), At: *p.FailedAt, Error: p.Error}, nil
	}
	if p.UpdatedAt.Before(s.now().Add(-staleAfter)) {
		return nil, nil, nil
	}
	out := &ReadinessProgress{RunID: p.RunID.String(), Trigger: p.Trigger, StartedAt: p.StartedAt, StartedBy: p.StartedByName, Done: int(p.Done), Total: int(p.Total),
		Groups: []ReadinessProgressGroup{}}
	_ = json.Unmarshal(p.Groups, &out.Groups)
	out.Queued = true
	for _, g := range out.Groups {
		if g.Status != "queued" {
			out.Queued = false
		}
	}
	return out, nil, nil
}

// ReadinessPoll is what the review screen polls while checks run: the run
// waiting or in progress, the latest stored run and a run that failed.
type ReadinessPoll struct {
	Running *ReadinessProgress   `json:"running,omitempty"`
	Latest  *ReadinessRunRef     `json:"latest,omitempty"`
	Failed  *ReadinessRunFailure `json:"failed,omitempty"`
}

// Poll reads the progress row and the latest run's id: two indexed reads.
func (s *Service) Poll(ctx context.Context, campaignID uuid.UUID) (ReadinessPoll, error) {
	var out ReadinessPoll
	var err error
	if out.Running, out.Failed, err = s.progressRow(ctx, campaignID); err != nil {
		return out, err
	}
	r, err := s.Q.ReadinessLatestRunRef(ctx, campaignID)
	switch {
	case err == nil:
		out.Latest = &ReadinessRunRef{ID: r.ID.String(), RanAt: r.RanAt, Verdict: r.Verdict}
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}
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
		return ReadinessRun{}, api.Conflict(say(ctx, "problem.ranAgain", nil))
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
func runningProblem(ctx context.Context) error {
	return api.Err(http.StatusConflict, "readiness_running", say(ctx, "problem.running", nil))
}

// runningOf is the message of what a group shows while it runs.
func runningOf(g ReadinessGroup) Text {
	if g.texts == nil {
		return nil
	}
	return g.texts.Running
}

// setText sets a group in progress's text and the message it's written from.
func (g *ReadinessProgressGroup) setText(s string, t Text) {
	g.Text = s
	if g.texts == nil {
		g.texts = &progressTexts{}
	}
	g.texts.Text = t
}

// JobInserter queues jobs in a transaction (River's client).
type JobInserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// RunArgs is a queued run of a campaign's checks (Start).
type RunArgs struct {
	CampaignID  uuid.UUID  `json:"campaign_id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	RunID       uuid.UUID  `json:"run_id"`
	Trigger     string     `json:"trigger"`
	By          *uuid.UUID `json:"by,omitempty"`
}

// Kind implements river.JobArgs.
func (RunArgs) Kind() string { return "readiness.run" }

// InsertOpts implements river.JobArgsWithInsertOpts. A run isn't retried:
// a failure is recorded and shown, and a person or the schedule starts the
// next one.
func (RunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1}
}

// RunWorker runs queued Send Readiness runs.
type RunWorker struct {
	river.WorkerDefaults[RunArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RunWorker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	return w.S.Work(ctx, job.Args)
}

// Timeout implements river.Worker: requesting every link of a large email
// can take a while.
func (w *RunWorker) Timeout(*river.Job[RunArgs]) time.Duration { return 5 * time.Minute }

// Work runs a queued run. A run another one took over does nothing; a run
// that fails is recorded on the progress row (not retried).
func (s *Service) Work(ctx context.Context, a RunArgs) error {
	ws, err := s.Q.GetWorkspace(ctx, a.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.execute(ctx, &ws, a.CampaignID, a.RunID, RunOptions{Trigger: a.Trigger, By: a.By})
	switch {
	case errors.Is(err, errTakenOver):
		return nil
	case err != nil && s.Log != nil:
		s.Log.WarnContext(ctx, "send readiness run failed", "campaign", a.CampaignID, "run", a.RunID, "err", err)
	}
	return nil
}

// MailboxClassifier is the part of mailbox.Classifier Send Readiness uses.
type MailboxClassifier interface {
	ClassifyPending(ctx context.Context, workspaceID uuid.UUID, budget time.Duration) (int64, error)
}
