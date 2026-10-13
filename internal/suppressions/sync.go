// Package suppressions keeps the shared suppression list in sync with each
// sending provider: addresses suppressed at a provider are pulled into the
// shared list (a bounce at one provider stops mail everywhere) and the
// shared list is pushed to providers with a suppression API. The list
// itself (screen, manual adds, imports) belongs to the audience domain.
package suppressions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service syncs suppressions and serves their sync status.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Factory *manage.Factory
	Jobs    func() *jobs.Client
	Log     *slog.Logger
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// SyncArgs syncs one provider, or every provider of a workspace (or of the
// instance when both are empty).
type SyncArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id,omitempty"`
	ProviderID  uuid.UUID `json:"provider_id,omitempty"`
}

// Kind implements river.JobArgs.
func (SyncArgs) Kind() string { return "suppressions.sync" }

// InsertOpts dedupes requests made within a minute (a burst of bounces
// triggers one push).
func (SyncArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Minute}}
}

// SyncWorker runs SyncArgs.
type SyncWorker struct {
	river.WorkerDefaults[SyncArgs]
	S *Service
}

// Timeout bounds one sync.
func (w *SyncWorker) Timeout(*river.Job[SyncArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *SyncWorker) Work(ctx context.Context, job *river.Job[SyncArgs]) error {
	provs, err := w.S.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range provs {
		if job.Args.ProviderID != uuid.Nil && p.ID != job.Args.ProviderID || job.Args.WorkspaceID != uuid.Nil && p.WorkspaceID != job.Args.WorkspaceID {
			continue
		}
		if manage.IsDemo(p) {
			continue
		}
		if err := w.S.SyncProvider(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
		}
	}
	return errors.Join(errs...)
}

// Enqueue asks for a sync soon (after new suppressions).
func (s *Service) Enqueue(ctx context.Context, workspaceID uuid.UUID) {
	if s.Jobs == nil {
		return
	}
	if c := s.Jobs(); c != nil {
		if _, err := c.Insert(ctx, SyncArgs{WorkspaceID: workspaceID}, nil); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "queue suppression sync", "err", err)
		}
	}
}

// SyncProvider pulls a provider's suppressions into the shared list and
// pushes the shared list's new addresses to it.
func (s *Service) SyncProvider(ctx context.Context, p store.Provider) error {
	cur, err := s.Q.SuppSyncGet(ctx, p.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		cur = store.ProviderSuppressionSync{ProviderID: p.ID, Mode: "two_way", Status: "ok"}
	}
	if cur.Mode == "off" {
		return nil
	}
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		return err
	}
	syncer, ok := adapter.(providers.SuppressionSyncer)
	if !ok || !state.Caps(p).SuppressionSync {
		// Local only: ReputeCast enforces the shared list before sending.
		return nil
	}
	now := s.now()
	next := cur
	next.Status, next.StatusDetail = "ok", ""
	fail := func(stage string, err error) error {
		next.Status, next.StatusDetail = "error", stage+": "+err.Error()
		_ = s.save(ctx, next)
		return err
	}
	if cur.Mode == "two_way" || cur.Mode == "pull" {
		since := time.Time{}
		if cur.LastPullAt != nil {
			since = cur.LastPullAt.Add(-time.Hour)
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		list, err := syncer.ListSuppressions(cctx, since)
		cancel()
		if err != nil {
			return fail("pull", err)
		}
		pulled := int64(0)
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, sp := range list {
				email := strings.ToLower(strings.TrimSpace(sp.Email))
				if email == "" || !strings.Contains(email, "@") {
					continue
				}
				reason := sp.Reason
				switch reason {
				case "hard_bounce", "complaint", "unsubscribe", "manual":
				default:
					reason = "manual"
				}
				created := sp.CreatedAt
				if created.IsZero() {
					created = now
				}
				_, err := q.IngestAddSuppression(ctx, store.IngestAddSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: p.WorkspaceID, Email: email, Reason: reason,
					Source: "provider_sync", SourceProviderID: &p.ID, OriginalCode: sp.Code, Detail: "Suppressed at " + p.Name, CreatedAt: created})
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				pulled++
			}
			return nil
		})
		if err != nil {
			return fail("pull", err)
		}
		next.Pulled += pulled
		next.LastPullAt = &now
	}
	if cur.Mode == "two_way" || cur.Mode == "push" {
		since := time.Unix(0, 0)
		if cur.LastPushAt != nil {
			since = *cur.LastPushAt
		}
		latest := since
		for {
			rows, err := s.Q.SuppToPush(ctx, store.SuppToPushParams{WorkspaceID: p.WorkspaceID, Since: since, ProviderID: &p.ID, Lim: 500})
			if err != nil {
				return fail("push", err)
			}
			if len(rows) == 0 {
				break
			}
			emails := make([]string, 0, len(rows))
			for _, r := range rows {
				emails = append(emails, r.Email)
			}
			cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			err = syncer.AddSuppressions(cctx, emails)
			cancel()
			if err != nil {
				return fail("push", err)
			}
			next.Pushed += int64(len(rows))
			if len(rows) < 500 {
				break
			}
			// SuppToPush orders by created_at; advance past this page.
			var last time.Time
			if err := s.Pool.QueryRow(ctx, `SELECT max(created_at) FROM suppressions WHERE workspace_id = $1 AND email = ANY($2)`, p.WorkspaceID, emails).Scan(&last); err != nil {
				return fail("push", err)
			}
			since = last
		}
		if l, err := s.Q.SuppLatestCreated(ctx, p.WorkspaceID); err == nil && l.After(latest) {
			latest = l
		}
		next.LastPushAt = &now
		_ = latest
	}
	pending, err := s.Q.SuppPendingCount(ctx, store.SuppPendingCountParams{WorkspaceID: p.WorkspaceID, Since: now, ProviderID: &p.ID})
	if err == nil {
		next.Pending = pending
	}
	return s.save(ctx, next)
}

func (s *Service) save(ctx context.Context, r store.ProviderSuppressionSync) error {
	return s.Q.SuppSyncUpsert(ctx, store.SuppSyncUpsertParams{ProviderID: r.ProviderID, Mode: r.Mode, LastPushAt: r.LastPushAt, LastPullAt: r.LastPullAt,
		Pushed: r.Pushed, Pulled: r.Pulled, Pending: r.Pending, Status: r.Status, StatusDetail: r.StatusDetail})
}

// SuppressionSyncRow is one provider in "Shared suppression list sync".
type SuppressionSyncRow struct {
	ProviderID string `json:"providerId,omitempty"`
	Provider   string `json:"provider"`
	// Scope says which list it syncs with ("account-level list", "per stream").
	Scope string `json:"scope"`
	// Direction: two_way, push, pull, off, local (no suppression API),
	// import (one-time pull from a migration).
	Direction string     `json:"direction" enum:"two_way,push,pull,off,local,import"`
	Pulled    int64      `json:"pulled"`
	Pushed    *int64     `json:"pushed,omitempty"`
	Pending   int64      `json:"pending"`
	LastSync  *time.Time `json:"lastSync,omitempty"`
	Status    string     `json:"status" enum:"in_sync,syncing,error,local,done,off"`
	Detail    string     `json:"detail,omitempty"`
}

// SuppressionReasonCount counts suppressions by reason.
type SuppressionReasonCount struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// SuppressionSyncStatus is the sync card of the Events & sync tab.
type SuppressionSyncStatus struct {
	Total   int64                    `json:"total"`
	Reasons []SuppressionReasonCount `json:"reasons"`
	Rows    []SuppressionSyncRow     `json:"rows"`
}

// scope describes the provider-side list an adapter syncs with.
func scope(adapter string) string {
	switch adapter {
	case "ses":
		return "account-level list"
	case "postmark":
		return "per stream"
	case "smtp", "postal":
		return "enforced before sending"
	}
	return "suppression list"
}

// Status returns the sync status of every provider.
func (s *Service) Status(ctx context.Context, ws *store.Workspace) (SuppressionSyncStatus, error) {
	out := SuppressionSyncStatus{Reasons: []SuppressionReasonCount{}, Rows: []SuppressionSyncRow{}}
	var err error
	if out.Total, err = s.Q.SuppCount(ctx, ws.ID); err != nil {
		return out, err
	}
	reasons, err := s.Q.SuppReasonCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range reasons {
		out.Reasons = append(out.Reasons, SuppressionReasonCount{Reason: r.Reason, Count: r.N})
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(provs))
	for _, p := range provs {
		ids = append(ids, p.ID)
	}
	syncs, err := s.Q.SuppSyncList(ctx, ids)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.ProviderSuppressionSync{}
	for _, r := range syncs {
		byID[r.ProviderID] = r
	}
	added, err := s.Q.SuppAddedByProvider(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	addedBy := map[uuid.UUID]store.SuppAddedByProviderRow{}
	for _, a := range added {
		addedBy[a.ProviderID] = a
	}
	for _, p := range provs {
		row := SuppressionSyncRow{ProviderID: p.ID.String(), Provider: p.Name, Scope: scope(p.Adapter)}
		r, ok := byID[p.ID]
		if !state.Caps(p).SuppressionSync {
			// No suppression API: its bounce and complaint events still add
			// to the shared list, which is enforced before every send.
			row.Direction, row.Status, row.Detail = "local", "local", "ReputeCast checks the shared list before handing mail to "+p.Name+"."
			if a, found := addedBy[p.ID]; found {
				row.Pulled = a.N
				at := a.LastAt
				row.LastSync = &at
			}
			if p.LastEventAt != nil && (row.LastSync == nil || p.LastEventAt.After(*row.LastSync)) {
				row.LastSync = p.LastEventAt
			}
			out.Rows = append(out.Rows, row)
			continue
		}
		if !ok {
			row.Direction, row.Status = "two_way", "syncing"
			pushed := int64(0)
			row.Pushed = &pushed
			out.Rows = append(out.Rows, row)
			continue
		}
		row.Direction, row.Pulled, row.Pending = r.Mode, r.Pulled, r.Pending
		pushed := r.Pushed
		row.Pushed = &pushed
		row.LastSync = r.LastPushAt
		if r.LastPullAt != nil && (row.LastSync == nil || r.LastPullAt.After(*row.LastSync)) {
			row.LastSync = r.LastPullAt
		}
		switch {
		case r.Mode == "off":
			row.Status = "off"
		case r.Status != "ok":
			row.Status, row.Detail = "error", r.StatusDetail
		case r.Pending > 0:
			row.Status = "syncing"
		default:
			row.Status = "in_sync"
		}
		out.Rows = append(out.Rows, row)
	}
	imports, err := s.Q.SuppImportSources(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, im := range imports {
		src := strings.TrimPrefix(im.Source, "import:")
		name := strings.ToUpper(src[:1]) + src[1:]
		if src == "csv" || src == "xlsx" {
			name = strings.ToUpper(src)
		}
		at := im.FirstAt
		out.Rows = append(out.Rows, SuppressionSyncRow{Provider: name + " (import)", Scope: at.Format("2 Jan"), Direction: "import", Pulled: im.N, LastSync: &at, Status: "done"})
	}
	return out, nil
}

// SuppressionSyncRun says what was queued.
type SuppressionSyncRun struct {
	Queued int `json:"queued"`
}

// Run queues a sync of every provider now ("Sync all now").
func (s *Service) Run(ctx context.Context, ws *store.Workspace) (SuppressionSyncRun, error) {
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return SuppressionSyncRun{}, err
	}
	n := 0
	for _, p := range provs {
		if state.Caps(p).SuppressionSync {
			n++
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if s.Jobs != nil {
			if c := s.Jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, SyncArgs{WorkspaceID: ws.ID}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 5 * time.Second}}); err != nil {
					return err
				}
			}
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "suppressions.sync", TargetType: "suppression_list", TargetID: ws.ID.String(), TargetName: ws.Name,
			Metadata: map[string]any{"providers": n}})
	})
	return SuppressionSyncRun{Queued: n}, err
}

type wsInput struct {
	api.WorkspaceInput
}

// Register adds the sync operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Sending providers"}
	api.Get(h, api.Op{ID: "get-suppression-sync", Path: "/workspaces/{workspace}/suppression-sync", Summary: "Sync of the shared suppression list with each provider",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[SuppressionSyncStatus], error) {
			out, err := s.Status(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "run-suppression-sync", Path: "/workspaces/{workspace}/suppression-sync/run", Summary: "Sync the shared suppression list with every provider now",
		Tags: tags, Access: api.Workspace(authz.ProvidersManage), Status: 202},
		func(ctx context.Context, _ *wsInput) (*api.Body[SuppressionSyncRun], error) {
			out, err := s.Run(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}
