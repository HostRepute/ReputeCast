package hostrepute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Snapshot is everything one sync reads from HostRepute.
type Snapshot struct {
	Account    Account
	Assets     []Asset
	Blocklists Blocklists
	// Postmaster by domain, SNDS by IP.
	Postmaster map[string][]PostmasterDay
	SNDS       map[string][]SNDSDay
	DMARC      []DMARCReport
	Alerts     []Alert
}

// Days of history a sync reads.
const (
	PostmasterDays = 30
	SNDSDays       = 30
	DMARCDays      = 7
)

// Fetch reads a snapshot. Calls that need a scope the token lacks are
// skipped; any other error fails the fetch.
func Fetch(ctx context.Context, c Client, now, alertsSince time.Time) (Snapshot, error) {
	snap := Snapshot{Postmaster: map[string][]PostmasterDay{}, SNDS: map[string][]SNDSDay{}}
	var err error
	if snap.Account, err = c.Account(ctx); err != nil {
		return snap, err
	}
	if snap.Assets, err = c.Assets(ctx); err != nil {
		return snap, err
	}
	can := func(scope string) bool {
		return len(snap.Account.Scopes) == 0 || slices.Contains(snap.Account.Scopes, scope)
	}
	skip := func(err error) bool { return errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) }
	if can(ScopeMonitorRead) {
		if snap.Blocklists, err = c.Blocklists(ctx); err != nil && !skip(err) {
			return snap, err
		}
		for _, a := range snap.Assets {
			switch a.Type {
			case AssetDomain:
				days, err := c.Postmaster(ctx, a.Asset, now.AddDate(0, 0, -PostmasterDays), now)
				if err != nil && !skip(err) {
					return snap, err
				}
				if len(days) > 0 {
					snap.Postmaster[a.Asset] = days
				}
				reps, err := c.DMARCReports(ctx, a.Asset, now.AddDate(0, 0, -DMARCDays), now)
				if err != nil && !skip(err) {
					return snap, err
				}
				snap.DMARC = append(snap.DMARC, reps...)
			case AssetIP:
				days, err := c.SNDS(ctx, a.Asset, now.AddDate(0, 0, -SNDSDays), now)
				if err != nil && !skip(err) {
					return snap, err
				}
				if len(days) > 0 {
					snap.SNDS[a.Asset] = days
				}
			}
		}
	}
	if can(ScopeAlertsRead) {
		if snap.Alerts, err = c.Alerts(ctx, alertsSince); err != nil && !skip(err) {
			return snap, err
		}
	}
	return snap, nil
}

// incidentKind maps a HostRepute alert kind to an incident kind.
func incidentKind(k string) string {
	switch k {
	case "listed":
		return "blocklist_listed"
	case "delisted":
		return "blocklist_delisted"
	case "":
		return "hostrepute_alert"
	}
	return k
}

func parseDay(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// Apply writes a snapshot into the workspace's tables in tx: blocklist
// results (replacing the previous HostRepute ones), Postmaster, SNDS and
// DMARC data, alerts as incidents (each once), and the connection's account,
// monitored assets and sync time. It returns the alerts that were new.
func Apply(ctx context.Context, tx pgx.Tx, ws uuid.UUID, snap Snapshot, now time.Time) ([]Alert, error) {
	q := store.New(tx)
	if snap.Blocklists.Results != nil {
		if err := q.DeleteDeliverabilityBlocklistResults(ctx, store.DeleteDeliverabilityBlocklistResultsParams{WorkspaceID: ws, Via: "hostrepute"}); err != nil {
			return nil, err
		}
		for _, r := range snap.Blocklists.Results {
			p := store.InsertDeliverabilityBlocklistResultParams{
				ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Via: "hostrepute", AssetType: r.AssetType, Asset: strings.ToLower(r.Asset),
				Source: r.Source, Impact: impact(r.Impact), Rank: int32(r.Rank), Listed: r.Listed, ListedSince: r.ListedSince,
				DelistUrl: r.DelistURL, Detail: r.Detail, CheckedAt: r.CheckedAt,
			}
			if p.AssetType != "ip" {
				p.AssetType = "domain"
			}
			if p.CheckedAt.IsZero() {
				p.CheckedAt = snap.Blocklists.CheckedAt
			}
			if r.Delisting != nil && r.Listed {
				p.Delisting = delisting(r.Delisting.Status)
				t := r.Delisting.RequestedAt
				p.DelistingRequestedAt = &t
			}
			if err := q.InsertDeliverabilityBlocklistResult(ctx, p); err != nil {
				return nil, fmt.Errorf("blocklist %s on %s: %w", r.Asset, r.Source, err)
			}
		}
	}
	for domain, days := range snap.Postmaster {
		for _, d := range days {
			day, err := parseDay(d.Day)
			if err != nil {
				continue
			}
			var fbl *int32
			if d.FBLFlagged != nil {
				f := int32(*d.FBLFlagged)
				fbl = &f
			}
			if err := q.UpsertDeliverabilityPostmasterDay(ctx, store.UpsertDeliverabilityPostmasterDayParams{
				WorkspaceID: ws, Domain: strings.ToLower(domain), Day: day, Via: "hostrepute", SpamRate: d.SpamRate,
				DomainReputation: strings.ToLower(d.DomainReputation), IpReputation: strings.ToLower(d.IPReputation),
				SpfRate: d.SPFRate, DkimRate: d.DKIMRate, DmarcRate: d.DMARCRate, TlsRate: d.TLSRate, DeliveryErrors: d.DeliveryErrors, FblFlagged: fbl,
			}); err != nil {
				return nil, err
			}
		}
	}
	for ip, days := range snap.SNDS {
		for _, d := range days {
			day, err := parseDay(d.Day)
			if err != nil {
				continue
			}
			if err := q.UpsertDeliverabilitySNDSDay(ctx, store.UpsertDeliverabilitySNDSDayParams{
				WorkspaceID: ws, Ip: ip, Day: day, Via: "hostrepute", FilterResult: filterResult(d.FilterResult),
				ComplaintRate: d.ComplaintRate, TrapHits: int32(d.TrapHits), Messages: d.Messages, JmrpReports: int32(d.JMRPReports),
			}); err != nil {
				return nil, err
			}
		}
	}
	for _, r := range snap.DMARC {
		policy, _ := json.Marshal(r.Policy)
		records, _ := json.Marshal(nonNil(r.Records))
		if err := q.UpsertDeliverabilityDMARCReport(ctx, store.UpsertDeliverabilityDMARCReportParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Domain: strings.ToLower(r.Domain), OrgName: r.OrgName, ReportID: r.ReportID,
			DateBegin: r.DateBegin, DateEnd: r.DateEnd, Policy: policy, Records: records, Via: "hostrepute", ReceivedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	var fresh []Alert
	alertsSince := time.Time{}
	for _, a := range snap.Alerts {
		var provider *uuid.UUID
		if ip, err := netip.ParseAddr(a.Asset); err == nil {
			if p, err := q.HostReputeProviderForIP(ctx, store.HostReputeProviderForIPParams{WorkspaceID: ws, Ip: ip.String()}); err == nil {
				provider = &p.ID
			}
		}
		ext := a.ID
		_, err := q.InsertDeliverabilityIncident(ctx, store.InsertDeliverabilityIncidentParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, OccurredAt: a.CreatedAt, Kind: incidentKind(a.Kind), Severity: severity(a.Severity),
			Title: a.Title, Detail: a.Detail, ProviderID: provider, Source: "hostrepute", Asset: strings.ToLower(a.Asset), ExternalID: &ext,
		})
		switch {
		case err == nil:
			fresh = append(fresh, a)
		case errors.Is(err, pgx.ErrNoRows):
			// Already stored by an earlier sync.
		default:
			return nil, err
		}
		if a.CreatedAt.After(alertsSince) {
			alertsSince = a.CreatedAt
		}
	}
	conn, err := q.GetHostReputeConnectionForUpdate(ctx, ws)
	if err != nil {
		return nil, err
	}
	since := conn.AlertsSince
	if !alertsSince.IsZero() && (since == nil || alertsSince.After(*since)) {
		since = &alertsSince
	}
	var limit *int32
	if snap.Account.Team.MonitorLimit > 0 {
		l := int32(snap.Account.Team.MonitorLimit)
		limit = &l
	}
	var lastCheck *time.Time
	if !snap.Blocklists.CheckedAt.IsZero() {
		t := snap.Blocklists.CheckedAt
		lastCheck = &t
	} else {
		lastCheck = conn.LastCheckAt
	}
	team := snap.Account.Team.Name
	if team == "" {
		team = conn.TeamName
	}
	scopes := snap.Account.Scopes
	if len(scopes) == 0 {
		scopes = conn.Scopes
	}
	if err := q.MarkHostReputeSynced(ctx, store.MarkHostReputeSyncedParams{
		WorkspaceID: ws, AccountEmail: firstNonEmpty(snap.Account.Email, conn.AccountEmail), TeamName: team,
		TeamID: firstNonEmpty(snap.Account.Team.ID, conn.TeamID), Plan: firstNonEmpty(snap.Account.Team.Plan, conn.Plan), Scopes: nonNil(scopes),
		Monitored: monitoredJSON(snap.Assets, now), MonitorLimit: limit, Sources: int32(max(snap.Account.Sources, snap.Blocklists.Sources)),
		CheckIntervalMinutes: int32(max(snap.Account.CheckIntervalMinutes, 1)), DmarcRua: firstNonEmpty(snap.Account.DMARCRua, conn.DmarcRua),
		LastCheckAt: lastCheck, AlertsSince: since, SyncedAt: &now,
	}); err != nil {
		return nil, err
	}
	return fresh, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func impact(s string) string {
	switch s {
	case "high", "medium", "low":
		return s
	}
	return "low"
}

func severity(s string) string {
	switch s {
	case "info", "ok", "warn", "bad":
		return s
	case "critical", "high", "error":
		return "bad"
	case "medium", "warning":
		return "warn"
	}
	return "info"
}

func delisting(s string) string {
	switch s {
	case "requested", "pending", "delisted", "denied":
		return s
	}
	return "requested"
}

func filterResult(s string) string {
	switch strings.ToLower(s) {
	case "green", "yellow", "red":
		return strings.ToLower(s)
	}
	return ""
}

// SyncArgs is the River job that syncs one workspace with HostRepute.
type SyncArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id" river:"unique"`
	// RunID is the check run to keep up to date (manual and connect syncs).
	RunID *uuid.UUID `json:"run_id,omitempty" river:"unique"`
	// Recheck asks HostRepute to check every source now before reading.
	Recheck bool `json:"recheck,omitempty"`
	// Initial is the first sync after connecting: no notifications for
	// alerts from before the connection.
	Initial bool `json:"initial,omitempty"`
}

// Kind implements river.JobArgs.
func (SyncArgs) Kind() string { return "hostrepute.sync" }

// InsertOpts keeps one pending sync per workspace (and run).
func (SyncArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: pendingStates}}
}

var pendingStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}

// SyncWorker runs SyncArgs.
type SyncWorker struct {
	river.WorkerDefaults[SyncArgs]
	S *Service
}

// Timeout leaves room for a re-check of 80+ sources.
func (w *SyncWorker) Timeout(*river.Job[SyncArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *SyncWorker) Work(ctx context.Context, job *river.Job[SyncArgs]) error {
	return w.S.Sync(ctx, job.Args, job.Attempt >= job.MaxAttempts)
}

// ScheduleArgs is the hourly job that queues a sync for every connected workspace.
type ScheduleArgs struct{}

// Kind implements river.JobArgs.
func (ScheduleArgs) Kind() string { return "hostrepute.schedule" }

// InsertOpts runs the scheduler once per period.
func (ScheduleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// ScheduleWorker runs ScheduleArgs.
type ScheduleWorker struct {
	river.WorkerDefaults[ScheduleArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ScheduleWorker) Work(ctx context.Context, _ *river.Job[ScheduleArgs]) error {
	if _, err := w.S.Q.DeleteExpiredHostReputeOAuthStates(ctx); err != nil {
		return err
	}
	ids, err := w.S.Q.ListConnectedHostRepute(ctx)
	if err != nil {
		return err
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	for _, id := range ids {
		if _, err := client.Insert(ctx, SyncArgs{WorkspaceID: id}, nil); err != nil {
			return err
		}
	}
	return nil
}

// recheckTimeout bounds how long a sync waits for HostRepute's re-check.
const recheckTimeout = 3 * time.Minute

// Sync reads HostRepute and writes the workspace's data. A rejected token
// marks the connection revoked (or expired) and notifies who can reconnect;
// an unreachable HostRepute is recorded for the Deliverability error state
// and, on the last attempt, the built-in checks take over.
func (s *Service) Sync(ctx context.Context, args SyncArgs, lastAttempt bool) error {
	conn, err := s.Q.GetHostReputeConnection(ctx, args.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.finishRun(ctx, args.RunID, "failed", 0, 0, 0, "HostRepute isn't connected.")
		return nil
	}
	if err != nil {
		return err
	}
	if conn.Status != "connected" && conn.Status != "limit_reached" {
		s.finishRun(ctx, args.RunID, "failed", 0, 0, 0, "HostRepute rejected the token. Reconnect to resume.")
		return nil
	}
	c, watch, err := s.clientFor(ctx, conn)
	if err != nil {
		return err
	}
	defer s.saveRefreshedToken(context.WithoutCancel(ctx), conn, watch)
	now := s.now()
	if args.Recheck {
		if err := s.recheck(ctx, c, conn, args.RunID); err != nil {
			return s.syncFailed(ctx, conn, args, err, lastAttempt)
		}
		now = s.now()
	}
	since := now.AddDate(0, 0, -PostmasterDays)
	if conn.AlertsSince != nil {
		since = *conn.AlertsSince
	}
	if args.Initial {
		if cands, err := Candidates(ctx, s.Q, conn.WorkspaceID); err == nil {
			s.ensureShared(ctx, c, cands)
		}
	}
	snap, err := Fetch(ctx, c, now, since)
	if err != nil {
		return s.syncFailed(ctx, conn, args, err, lastAttempt)
	}
	var fresh []Alert
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		if fresh, err = Apply(ctx, tx, conn.WorkspaceID, snap, now); err != nil {
			return err
		}
		if conn.AlertsToNotifications && !args.Initial {
			if err := s.notifyAlerts(ctx, tx, conn.WorkspaceID, fresh, now); err != nil {
				return err
			}
		}
		listed := 0
		for _, r := range snap.Blocklists.Results {
			if r.Listed {
				listed++
			}
		}
		total := max(snap.Blocklists.Sources, snap.Account.Sources)
		return s.finishRunTx(ctx, tx, args.RunID, "done", total, total, listed, "")
	})
	return err
}

// ensureShared adds the shared candidates HostRepute doesn't monitor yet
// (the first sync after connecting), up to the plan's limit.
func (s *Service) ensureShared(ctx context.Context, c Client, cands []Candidate) {
	assets, err := c.Assets(ctx)
	if err != nil {
		return
	}
	for _, cand := range cands {
		if !cand.Shared || slices.ContainsFunc(assets, func(a Asset) bool { return strings.EqualFold(a.Asset, cand.Asset) }) {
			continue
		}
		var le *LimitError
		if _, err := c.AddAsset(ctx, cand.Asset, cand.Type, cand.Role); errors.As(err, &le) {
			return
		}
	}
}

func (s *Service) recheck(ctx context.Context, c Client, conn store.HostreputeConnection, runID *uuid.UUID) error {
	chk, err := c.StartCheck(ctx, monitoredNames(conn.Monitored))
	if err != nil {
		return err
	}
	deadline := time.Now().Add(recheckTimeout)
	for {
		if runID != nil {
			_ = s.Q.UpdateDeliverabilityCheckRunProgress(ctx, store.UpdateDeliverabilityCheckRunProgressParams{ID: *runID, Total: int32(chk.Total), Answered: int32(chk.Answered)})
		}
		if chk.Status != "running" || time.Now().After(deadline) {
			return nil
		}
		if !sleep(ctx, 1500*time.Millisecond) {
			return ctx.Err()
		}
		if chk, err = c.Check(ctx, chk.ID); err != nil {
			return err
		}
	}
}

func (s *Service) syncFailed(ctx context.Context, conn store.HostreputeConnection, args SyncArgs, err error, lastAttempt bool) error {
	now := s.now()
	switch {
	case errors.Is(err, ErrUnauthorized):
		status := "revoked"
		if conn.TokenExpiresAt != nil && conn.TokenExpiresAt.Before(now) {
			status = "expired"
		}
		detail := "HostRepute rejected the token (revoked from the HostRepute side or expired)."
		txErr := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			n, err := q.MarkHostReputeTokenRejected(ctx, store.MarkHostReputeTokenRejectedParams{WorkspaceID: conn.WorkspaceID, Status: status, StatusDetail: detail, RejectedAt: &now})
			if err != nil || n == 0 {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{
				WorkspaceID: &conn.WorkspaceID, Action: "hostrepute.token_rejected", TargetType: "integration", TargetID: "hostrepute",
				TargetName: "HostRepute · " + conn.TeamName, Before: map[string]any{"status": conn.Status}, After: map[string]any{"status": status},
			}); err != nil {
				return err
			}
			slug, _ := q.HostReputeWorkspaceSlug(ctx, conn.WorkspaceID)
			if err := notify.ToMembersWith(ctx, tx, conn.WorkspaceID, authz.IntegrationsHostRepute, notify.Notification{
				Kind: notify.KindHostReputeAlert, Severity: "warn", Title: "HostRepute needs reconnecting",
				Body: "HostRepute rejected the token. Blocklist, Postmaster, SNDS and DMARC data is stale until someone reconnects; Send Readiness marks the reputation check as unknown.",
				Link: "/w/" + slug + "/settings/integrations",
			}); err != nil {
				return err
			}
			if s.Fallback != nil {
				return s.Fallback(ctx, tx, conn.WorkspaceID, "fallback")
			}
			return nil
		})
		s.finishRun(ctx, args.RunID, "failed", 0, 0, 0, detail)
		return txErr
	case errors.Is(err, ErrUnavailable):
		msg := err.Error()
		var ue *UnavailableError
		if !errors.As(err, &ue) {
			msg = "Couldn't reach HostRepute"
		}
		if rerr := s.Q.RecordHostReputeSyncError(ctx, store.RecordHostReputeSyncErrorParams{WorkspaceID: conn.WorkspaceID, AttemptedAt: &now, Error: msg}); rerr != nil {
			return rerr
		}
		if !lastAttempt {
			return err
		}
		s.finishRun(ctx, args.RunID, "failed", 0, 0, 0, msg)
		if s.Fallback != nil {
			_ = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error { return s.Fallback(ctx, tx, conn.WorkspaceID, "fallback") })
		}
		return nil
	}
	_ = s.Q.RecordHostReputeSyncError(ctx, store.RecordHostReputeSyncErrorParams{WorkspaceID: conn.WorkspaceID, AttemptedAt: &now, Error: "HostRepute sync failed: " + err.Error()})
	if lastAttempt {
		s.finishRun(ctx, args.RunID, "failed", 0, 0, 0, err.Error())
	}
	return err
}

func (s *Service) finishRun(ctx context.Context, id *uuid.UUID, status string, answered, total, listed int, msg string) {
	if id == nil {
		return
	}
	_ = s.Q.FinishDeliverabilityCheckRun(ctx, store.FinishDeliverabilityCheckRunParams{ID: *id, Status: status, Answered: int32(answered), Total: int32(total), Listed: int32(listed), Error: msg, FinishedAt: ptr(s.now())})
}

func (s *Service) finishRunTx(ctx context.Context, tx pgx.Tx, id *uuid.UUID, status string, answered, total, listed int, msg string) error {
	if id == nil {
		return nil
	}
	return store.New(tx).FinishDeliverabilityCheckRun(ctx, store.FinishDeliverabilityCheckRunParams{ID: *id, Status: status, Answered: int32(answered), Total: int32(total), Listed: int32(listed), Error: msg, FinishedAt: ptr(s.now())})
}

func ptr[T any](v T) *T { return &v }

// notifyAlerts turns fresh HostRepute alerts into notifications for members
// who can see Deliverability. Alerts older than a day (history caught up
// after downtime) are stored without a notification.
func (s *Service) notifyAlerts(ctx context.Context, tx pgx.Tx, ws uuid.UUID, alerts []Alert, now time.Time) error {
	if len(alerts) == 0 {
		return nil
	}
	q := store.New(tx)
	slug, err := q.HostReputeWorkspaceSlug(ctx, ws)
	if err != nil {
		return err
	}
	for _, a := range alerts {
		if now.Sub(a.CreatedAt) > 24*time.Hour {
			continue
		}
		data := map[string]any{"asset": a.Asset, "source": a.Source, "alertId": a.ID}
		if a.DelistURL != "" {
			data["delistUrl"] = a.DelistURL
		}
		if err := notify.ToMembersWith(ctx, tx, ws, authz.DeliverabilityView, notify.Notification{
			Kind: notify.KindHostReputeAlert, Severity: severity(a.Severity), Title: a.Title, Body: a.Detail,
			Link: "/w/" + slug + "/deliverability", Data: data,
		}); err != nil {
			return err
		}
	}
	return nil
}
