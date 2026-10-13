package deliverability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// CheckArgs is the River job of the local checks: DNS authentication of the
// sending domains, reverse DNS of the dedicated IPs and, when Blocklists is
// set, the built-in DNSBL check.
type CheckArgs struct {
	WorkspaceID uuid.UUID  `json:"workspace_id" river:"unique"`
	RunID       *uuid.UUID `json:"run_id,omitempty" river:"unique"`
	Blocklists  bool       `json:"blocklists" river:"unique"`
}

// Kind implements river.JobArgs.
func (CheckArgs) Kind() string { return "deliverability.check" }

// InsertOpts keeps one pending check per workspace and kind of check.
func (CheckArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: pendingStates}}
}

var pendingStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}

// CheckWorker runs CheckArgs.
type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	S *Service
}

// Timeout bounds a run of DNS lookups.
func (w *CheckWorker) Timeout(*river.Job[CheckArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *CheckWorker) Work(ctx context.Context, job *river.Job[CheckArgs]) error {
	return w.S.RunChecks(ctx, job.Args)
}

// ScheduleArgs is the job that queues the daily local checks.
type ScheduleArgs struct{}

// Kind implements river.JobArgs.
func (ScheduleArgs) Kind() string { return "deliverability.schedule" }

// InsertOpts runs the scheduler once per period.
func (ScheduleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 3 * time.Hour}}
}

// ScheduleWorker queues the checks of workspaces not checked for 20 hours:
// the built-in blocklist check too when HostRepute isn't covering them.
type ScheduleWorker struct {
	river.WorkerDefaults[ScheduleArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ScheduleWorker) Work(ctx context.Context, _ *river.Job[ScheduleArgs]) error {
	if _, err := w.S.Q.FailStaleDeliverabilityCheckRuns(ctx, time.Now().Add(-runTTL)); err != nil {
		return err
	}
	ids, err := w.S.Q.DeliverabilityWorkspaces(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		last, err := w.S.Q.LastDeliverabilityAssetCheckAt(ctx, id)
		if err != nil {
			return err
		}
		if time.Since(last) < 20*time.Hour {
			continue
		}
		covered := false
		if c, err := w.S.Q.GetHostReputeConnection(ctx, id); err == nil {
			covered = c.Status == "connected" && c.LastSyncError == ""
		}
		err = db.Tx(ctx, w.S.Pool, func(tx pgx.Tx) error {
			if covered {
				_, err := w.S.Jobs.InsertTx(ctx, tx, CheckArgs{WorkspaceID: id}, nil)
				return err
			}
			_, err := w.S.queueBuiltin(ctx, tx, id, nil, "schedule")
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// queueBuiltin records a built-in check run and queues it.
func (s *Service) queueBuiltin(ctx context.Context, tx pgx.Tx, ws uuid.UUID, by *uuid.UUID, trigger string) (store.DeliverabilityCheckRun, error) {
	q := store.New(tx)
	assets, err := s.builtinAssets(ctx, q, ws)
	if err != nil {
		return store.DeliverabilityCheckRun{}, err
	}
	names := make([]string, 0, len(assets))
	for _, a := range assets {
		names = append(names, a.Name)
	}
	run, err := q.CreateDeliverabilityCheckRun(ctx, store.CreateDeliverabilityCheckRunParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Via: "builtin", Trigger: trigger, Assets: names,
		Total: int32(len(dnscheck.Pairs(s.lists(), assets))), RequestedBy: by, StartedAt: s.now(),
	})
	if err != nil {
		return run, err
	}
	_, err = s.Jobs.InsertTx(ctx, tx, CheckArgs{WorkspaceID: ws, RunID: &run.ID, Blocklists: true}, nil)
	return run, err
}

// Fallback queues the built-in checks for HostRepute (disconnect, rejected
// token, outage). It has the signature hostrepute.Service.Fallback expects.
func (s *Service) Fallback(ctx context.Context, tx pgx.Tx, ws uuid.UUID, trigger string) error {
	_, err := s.queueBuiltin(ctx, tx, ws, nil, trigger)
	return err
}

// builtinAssets are what the built-in check looks up: sending domains and
// dedicated IPs.
func (s *Service) builtinAssets(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]dnscheck.Asset, error) {
	cands, err := hostrepute.Candidates(ctx, q, ws)
	if err != nil {
		return nil, err
	}
	var out []dnscheck.Asset
	for _, c := range cands {
		switch c.Role {
		case hostrepute.RoleSendingDomain:
			out = append(out, dnscheck.Asset{Name: c.Asset, Kind: dnscheck.KindDomain})
		case hostrepute.RoleDedicatedIP:
			out = append(out, dnscheck.Asset{Name: c.Asset, Kind: dnscheck.KindIP})
		}
	}
	return out, nil
}

// manualCheckGap is the minimum time between two manual runs.
const manualCheckGap = time.Minute

// RunNow starts "Run all checks now": a HostRepute re-check when HostRepute
// is connected (plus the local DNS checks), else the built-in checks. A run
// already in progress, or one started less than a minute ago, is returned
// instead of starting another.
func (s *Service) RunNow(ctx context.Context, ws *store.Workspace, builtinOnly bool) (CheckRun, error) {
	now := s.now()
	if run, err := s.Q.RunningDeliverabilityCheckRun(ctx, store.RunningDeliverabilityCheckRunParams{WorkspaceID: ws.ID, AliveAfter: now.Add(-runTTL)}); err == nil {
		return runView(run), nil
	}
	if run, err := s.Q.LatestDeliverabilityCheckRun(ctx, ws.ID); err == nil && run.Trigger == "manual" && now.Sub(run.StartedAt) < manualCheckGap &&
		(!builtinOnly || run.Via == "builtin") {
		return runView(run), nil
	}
	var by *uuid.UUID
	if p := api.PrincipalFrom(ctx); p != nil && p.Type == authz.ActorUser {
		id := p.UserID
		by = &id
	}
	useHostRepute := false
	if !builtinOnly {
		if c, err := s.Q.GetHostReputeConnection(ctx, ws.ID); err == nil && (c.Status == "connected" || c.Status == "limit_reached") {
			useHostRepute = true
		}
	}
	var out CheckRun
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		via := "builtin"
		if useHostRepute {
			via = "hostrepute"
			if err := s.HostRepute.QueueRecheck(ctx, tx, ws.ID, by); err != nil {
				return err
			}
			if _, err := s.Jobs.InsertTx(ctx, tx, CheckArgs{WorkspaceID: ws.ID}, nil); err != nil {
				return err
			}
		} else if _, err := s.queueBuiltin(ctx, tx, ws.ID, by, "manual"); err != nil {
			return err
		}
		run, err := q.LatestDeliverabilityCheckRun(ctx, ws.ID)
		if err != nil {
			return err
		}
		out = runView(run)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "deliverability.check_run", TargetType: "check_run",
			TargetID: run.ID.String(), TargetName: "Deliverability checks", Metadata: map[string]any{"via": via}})
	})
	return out, err
}

// Run returns a check run (progress polling).
func (s *Service) Run(ctx context.Context, ws uuid.UUID, id uuid.UUID) (CheckRun, error) {
	run, err := s.Q.GetDeliverabilityCheckRun(ctx, store.GetDeliverabilityCheckRunParams{ID: id, WorkspaceID: ws})
	if err != nil {
		return CheckRun{}, api.MapError(err, "Check run")
	}
	return runView(run), nil
}

// RunChecks runs the local checks of a workspace.
func (s *Service) RunChecks(ctx context.Context, args CheckArgs) error {
	q := s.Q
	ws := args.WorkspaceID
	if err := s.authChecks(ctx, ws); err != nil {
		s.Log.WarnContext(ctx, "deliverability: DNS checks failed", "workspace", ws, "err", err)
	}
	if !args.Blocklists {
		return nil
	}
	assets, err := s.builtinAssets(ctx, q, ws)
	if err != nil {
		return err
	}
	lists := s.lists()
	var mu sync.Mutex
	last := time.Now()
	results := dnscheck.CheckAll(ctx, s.DNS, lists, assets, 8, 5*time.Second, func(done, total int) {
		if args.RunID == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if done < total && time.Since(last) < 300*time.Millisecond {
			return
		}
		last = time.Now()
		_ = q.UpdateDeliverabilityCheckRunProgress(ctx, store.UpdateDeliverabilityCheckRunProgressParams{ID: *args.RunID, Total: int32(total), Answered: int32(done)})
	})
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		tq := store.New(tx)
		previous, err := tq.ListDeliverabilityBlocklistResults(ctx, ws)
		if err != nil {
			return err
		}
		wasListed := map[string]store.BlocklistResult{}
		for _, b := range previous {
			if b.Via == "builtin" && b.Listed {
				wasListed[b.Asset+"|"+b.Source] = b
			}
		}
		if err := tq.DeleteDeliverabilityBlocklistResults(ctx, store.DeleteDeliverabilityBlocklistResultsParams{WorkspaceID: ws, Via: "builtin"}); err != nil {
			return err
		}
		listed := 0
		var errs int
		for _, r := range results {
			if r.Err != nil {
				errs++
				continue // unknown, not clean: no row
			}
			p := store.InsertDeliverabilityBlocklistResultParams{
				ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Via: "builtin", AssetType: r.List.Kind, Asset: r.Asset, Source: r.List.Name,
				Impact: r.List.Impact, Rank: int32(r.List.Rank), Listed: r.Listed, DelistUrl: r.List.DelistURL, Detail: r.Reason, CheckedAt: now,
			}
			if r.Listed {
				listed++
				since := now
				if prev, ok := wasListed[r.Asset+"|"+r.List.Name]; ok && prev.ListedSince != nil {
					since = *prev.ListedSince
				} else if err := s.newListing(ctx, tx, ws, r, now); err != nil {
					return err
				}
				p.ListedSince = &since
			}
			if err := tq.InsertDeliverabilityBlocklistResult(ctx, p); err != nil {
				return err
			}
		}
		if args.RunID == nil {
			return nil
		}
		msg := ""
		if errs > 0 {
			msg = fmt.Sprintf("%d of %d lookups didn't answer; they show as unknown.", errs, len(results))
		}
		return tq.FinishDeliverabilityCheckRun(ctx, store.FinishDeliverabilityCheckRunParams{ID: *args.RunID, Status: "done", Answered: int32(len(results)), Total: int32(len(results)),
			Listed: int32(listed), Error: msg, FinishedAt: &now})
	})
}

// newListing records a listing the built-in check found and notifies the
// members who can see Deliverability.
func (s *Service) newListing(ctx context.Context, tx pgx.Tx, ws uuid.UUID, r dnscheck.Listing, now time.Time) error {
	q := store.New(tx)
	var provider *uuid.UUID
	if ip, err := netip.ParseAddr(r.Asset); err == nil {
		if p, err := q.HostReputeProviderForIP(ctx, store.HostReputeProviderForIPParams{WorkspaceID: ws, Ip: ip.String()}); err == nil {
			provider = &p.ID
		}
	}
	sev := "warn"
	if r.List.Impact == "high" {
		sev = "bad"
	}
	title := fmt.Sprintf("%s listed on %s", r.Asset, r.List.Name)
	detail := fmt.Sprintf("Found by the built-in daily check (%s impact).", r.List.Impact)
	if _, err := q.InsertDeliverabilityIncident(ctx, store.InsertDeliverabilityIncidentParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, OccurredAt: now, Kind: "blocklist_listed", Severity: sev, Title: title,
		Detail: detail, ProviderID: provider, Source: "builtin", Asset: r.Asset,
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	slug, err := q.HostReputeWorkspaceSlug(ctx, ws)
	if err != nil {
		return err
	}
	return notify.ToMembersWith(ctx, tx, ws, authz.DeliverabilityView, notify.Notification{
		Kind: notify.KindHostReputeAlert, Severity: sev, Title: title, Body: detail + " Delisting page: " + r.List.DelistURL,
		Link: "/w/" + slug + "/deliverability", Data: map[string]any{"asset": r.Asset, "source": r.List.Name, "delistUrl": r.List.DelistURL},
	})
}

// authChecks runs the DNS authentication checks of every sending domain and
// its organization domain, and reverse DNS of every dedicated IP.
func (s *Service) authChecks(ctx context.Context, ws uuid.UUID) error {
	domains, err := s.Q.HostReputeAssetSources(ctx, ws)
	if err != nil {
		return err
	}
	cands, err := hostrepute.Candidates(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	now := s.now()
	save := func(asset, typ string, result any) error {
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return s.Q.UpsertDeliverabilityAssetCheck(ctx, store.UpsertDeliverabilityAssetCheckParams{WorkspaceID: ws, Asset: asset, AssetType: typ, Result: b, CheckedAt: now})
	}
	for _, d := range domains {
		auth := dnscheck.CheckDomain(ctx, s.DNS, d.Domain, Requirements(d.DnsRecords))
		if err := save(d.Domain, "domain", map[string]any{"auth": auth}); err != nil {
			return err
		}
	}
	for _, c := range cands {
		switch c.Role {
		case hostrepute.RoleOrganizationDomain:
			if slices.ContainsFunc(domains, func(d store.HostReputeAssetSourcesRow) bool { return d.Domain == c.Asset }) {
				continue
			}
			if err := save(c.Asset, "domain", map[string]any{"dmarc": dnscheck.CheckDMARC(ctx, s.DNS, c.Asset)}); err != nil {
				return err
			}
		case hostrepute.RoleDedicatedIP:
			ip, err := netip.ParseAddr(c.Asset)
			if err != nil {
				continue
			}
			if err := save(c.Asset, "ip", map[string]any{"rdns": dnscheck.CheckReverseDNS(ctx, s.DNS, ip)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Requirements reads the provider records stored on a sending domain.
func Requirements(raw json.RawMessage) []dnscheck.Requirement {
	var recs []struct {
		ProviderID string `json:"provider_id"`
		Purpose    string `json:"purpose"`
		Type       string `json:"type"`
		Name       string `json:"name"`
		Value      string `json:"value"`
	}
	_ = json.Unmarshal(raw, &recs)
	out := make([]dnscheck.Requirement, 0, len(recs))
	for _, r := range recs {
		if r.Name == "" || r.Type == "" {
			continue
		}
		out = append(out, dnscheck.Requirement{ProviderID: r.ProviderID, Purpose: strings.ToLower(r.Purpose), Type: strings.ToUpper(r.Type), Name: r.Name, Value: r.Value})
	}
	return out
}
