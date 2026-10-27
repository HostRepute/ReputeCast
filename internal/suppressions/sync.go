// Package suppressions keeps the shared suppression list in sync with each
// sending provider: addresses suppressed at a provider are pulled into the
// shared list (a bounce at one provider stops mail everywhere) and the
// shared list is pushed to providers with a suppression API. The list
// itself (screen, manual adds, imports) belongs to the audience domain.
package suppressions

import (
	"context"
	"encoding/json"
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
	"github.com/hostrepute/reputecast/internal/i18n"
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

// pushBatch is how many addresses one push or removal round sends.
const pushBatch = 500

// listScope is where a provider's suppression list lives and whether
// ReputeCast may write to it (providers.SuppressionScoper); a provider that
// doesn't say has a writable account-level list.
func listScope(adapter providers.Provider) providers.SuppressionScope {
	if sc, ok := adapter.(providers.SuppressionScoper); ok {
		return sc.SuppressionScope()
	}
	return providers.SuppressionScope{Scope: providers.SuppressionScopeAccount, Writable: true}
}

// SyncProvider syncs one provider's suppression list with the shared list:
//
//   - pull: what the provider suppressed since the last pull joins the
//     shared list (a bounce at one provider stops mail everywhere);
//   - push: hard bounces and complaints that apply to all mail, which the
//     provider's list doesn't hold yet, in the shared list's insert order
//     (late bounces and imported history included). Unsubscribes and
//     marketing-only entries are never pushed: a provider list stops every
//     message, a password reset included;
//   - removals: what left the shared list leaves the provider's list.
//
// A list the provider shares with mail ReputeCast doesn't send (an Amazon SES
// account-level list without a tenant) is only read. A failed pull doesn't
// skip the push. The pending count is what remains to push and remove.
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
	scope := listScope(adapter)
	now := s.now()
	next := cur
	next.Status, next.StatusDetail, next.Scope = "ok", "", scope.Scope
	var errs []error
	failed := func(stage string, err error) {
		next.Status = "error"
		if next.StatusDetail != "" {
			next.StatusDetail += "; "
		}
		next.StatusDetail += stage + ": " + err.Error()
		errs = append(errs, fmt.Errorf("%s: %w", stage, err))
	}
	if cur.Mode == "two_way" || cur.Mode == "pull" {
		if n, err := s.pull(ctx, p, syncer, cur, scope, now); err != nil {
			failed("pull", err)
		} else {
			next.Pulled += n
			next.LastPullAt = &now
		}
	}
	push := (cur.Mode == "two_way" || cur.Mode == "push") && scope.Writable
	if push {
		pushed, top, mode, err := s.push(ctx, p, adapter, syncer, cur.Mode, now)
		next.Pushed += pushed
		next.PushSeq = max(next.PushSeq, top)
		if mode != "" {
			next.Mode = mode
		}
		if err != nil {
			failed("push", err)
		} else {
			next.LastPushAt = &now
		}
		if next.Mode != "off" && next.Mode != "pull" {
			removed, err := s.removeGone(ctx, p, adapter)
			next.Removed += removed
			if err != nil {
				failed("remove", err)
			}
		}
	}
	next.Pending = 0
	if push && next.Mode != "pull" && next.Mode != "off" {
		if pending, err := s.Q.SuppPendingCount(ctx, store.SuppPendingCountParams{WorkspaceID: p.WorkspaceID, ProviderID: &p.ID}); err == nil {
			next.Pending = pending
		} else {
			errs = append(errs, err)
		}
	}
	if err := s.save(ctx, next); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// pull copies what the provider suppressed since the last pull into the
// shared list. On a list ReputeCast writes to, it also records the entries,
// so removing one from the shared list removes it at the provider.
func (s *Service) pull(ctx context.Context, p store.Provider, syncer providers.SuppressionSyncer, cur store.ProviderSuppressionSync, scope providers.SuppressionScope, now time.Time) (int64, error) {
	since := time.Time{}
	if cur.LastPullAt != nil {
		since = cur.LastPullAt.Add(-time.Hour)
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	list, err := syncer.ListSuppressions(cctx, since)
	cancel()
	if err != nil {
		return 0, err
	}
	pulled := int64(0)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		emails := make([]string, 0, len(list))
		for _, sp := range list {
			emails = append(emails, strings.ToLower(strings.TrimSpace(sp.Email)))
		}
		// What the provider holds because of ReputeCast (pushed, or pulled
		// before) is no news: in particular an address someone removed from
		// the shared list stays out until its removal reaches the provider.
		known := map[string]bool{}
		if scope.Writable {
			held, err := q.SuppEntriesKnown(ctx, store.SuppEntriesKnownParams{ProviderID: p.ID, Emails: emails})
			if err != nil {
				return err
			}
			for _, e := range held {
				known[e] = true
			}
		}
		seen := make([]string, 0, len(list))
		for i, sp := range list {
			email := emails[i]
			if email == "" || !strings.Contains(email, "@") || known[email] {
				continue
			}
			seen = append(seen, email)
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
		if scope.Writable && len(seen) > 0 {
			return q.SuppEntriesAdd(ctx, store.SuppEntriesAddParams{ProviderID: p.ID, Emails: seen, Origin: "pulled", Seqs: make([]int64, len(seen)), SyncedAt: now})
		}
		return nil
	})
	return pulled, err
}

// push adds the qualifying suppressions the provider's list doesn't hold
// yet, page by page in insert order, and records each page once the
// provider took it. mode is the sync's new mode when the provider turned
// out to take no pushes ("pull", or "off" for a push-only sync).
func (s *Service) push(ctx context.Context, p store.Provider, adapter providers.Provider, syncer providers.SuppressionSyncer, curMode string, now time.Time) (pushed, top int64, mode string, err error) {
	withReason, _ := adapter.(providers.SuppressionReasonAdder)
	for {
		rows, err := s.Q.SuppToPush(ctx, store.SuppToPushParams{WorkspaceID: p.WorkspaceID, ProviderID: &p.ID, Lim: pushBatch})
		if err != nil {
			return pushed, top, "", err
		}
		if len(rows) == 0 {
			return pushed, top, "", nil
		}
		emails := make([]string, 0, len(rows))
		seqs := make([]int64, 0, len(rows))
		list := make([]providers.Suppression, 0, len(rows))
		for _, r := range rows {
			emails = append(emails, r.Email)
			seqs = append(seqs, r.Seq)
			list = append(list, providers.Suppression{Email: r.Email, Reason: r.Reason})
			top = max(top, r.Seq)
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		if withReason != nil {
			err = withReason.AddSuppressionsWithReason(cctx, list)
		} else {
			err = syncer.AddSuppressions(cctx, emails)
		}
		cancel()
		if errors.Is(err, providers.ErrNotSupported) {
			// The provider lists its suppressions but takes none over its
			// API (Brevo): a two-way sync pulls only from now on, and a
			// push-only one has nothing left to do.
			if curMode == "push" {
				return pushed, top, "off", nil
			}
			return pushed, top, "pull", nil
		}
		if err != nil {
			return pushed, top, "", err
		}
		if err := s.Q.SuppEntriesAdd(ctx, store.SuppEntriesAddParams{ProviderID: p.ID, Emails: emails, Origin: "pushed", Seqs: seqs, SyncedAt: now}); err != nil {
			return pushed, top, "", err
		}
		pushed += int64(len(rows))
		if len(rows) < pushBatch {
			return pushed, top, "", nil
		}
	}
}

// removeGone takes off the provider's list what left the shared list. A
// provider whose API can't remove an address only forgets the entry.
func (s *Service) removeGone(ctx context.Context, p store.Provider, adapter providers.Provider) (int64, error) {
	remover, _ := adapter.(providers.SuppressionRemover)
	removed := int64(0)
	for {
		rows, err := s.Q.SuppToRemove(ctx, store.SuppToRemoveParams{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Lim: pushBatch})
		if err != nil || len(rows) == 0 {
			return removed, err
		}
		emails := make([]string, 0, len(rows))
		for _, r := range rows {
			emails = append(emails, r.Email)
		}
		if remover != nil {
			cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			err := remover.RemoveSuppressions(cctx, emails)
			cancel()
			if err != nil && !errors.Is(err, providers.ErrNotSupported) {
				return removed, err
			}
			if err == nil {
				removed += int64(len(rows))
			}
		}
		if err := s.Q.SuppEntriesDelete(ctx, store.SuppEntriesDeleteParams{ProviderID: p.ID, Emails: emails}); err != nil {
			return removed, err
		}
		if len(rows) < pushBatch {
			return removed, nil
		}
	}
}

func (s *Service) save(ctx context.Context, r store.ProviderSuppressionSync) error {
	return s.Q.SuppSyncUpsert(ctx, store.SuppSyncUpsertParams(r))
}

// SuppressionSyncRow is one provider in "Shared suppression list sync".
type SuppressionSyncRow struct {
	ProviderID string `json:"providerId,omitempty"`
	Provider   string `json:"provider"`
	// Scope says which list it syncs with ("tenant list · acme-newsletter",
	// "per stream", "account-level list · read only").
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
	// ReadOnly: the provider's list is shared with mail ReputeCast doesn't
	// send (an Amazon SES account without a tenant), so it is only read.
	ReadOnly bool `json:"readOnly"`
	// Tenant is the Amazon SES tenant whose own list the sync uses.
	Tenant string `json:"tenant,omitempty"`
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
	// Pushed are the suppressions providers get, by reason: hard bounces
	// and complaints that apply to all mail. Kept stay in ReputeCast only,
	// by reason: marketing-only ones (unsubscribes, role addresses), so
	// password resets and receipts still arrive, and the other all-mail
	// ones (manual entries), which ReputeCast still blocks for all mail.
	Pushed        []SuppressionReasonCount `json:"pushed"`
	Kept          []SuppressionReasonCount `json:"kept"`
	PushedTotal   int64                    `json:"pushedTotal"`
	MarketingOnly int64                    `json:"marketingOnly"`
	KeptAllMail   int64                    `json:"keptAllMail"`
	Rows          []SuppressionSyncRow     `json:"rows"`
}

// sesTenant is the Amazon SES tenant a provider's settings name.
func sesTenant(p store.Provider) string {
	var s struct {
		Tenant string `json:"tenant"`
	}
	_ = json.Unmarshal(p.Config, &s)
	return s.Tenant
}

// rowScope describes the provider-side list a provider syncs with, in a
// reader's language, and whether ReputeCast only reads it.
func rowScope(l i18n.Lang, p store.Provider, stored string) (string, bool) {
	if p.Adapter == "ses" {
		if t := sesTenant(p); t != "" {
			return text(l, "scope.tenant", i18n.Args{"tenant": t}), false
		}
		return text(l, "scope.accountReadOnly", nil), true
	}
	switch stored {
	case providers.SuppressionScopeStream:
		return text(l, "scope.stream", nil), false
	case providers.SuppressionScopeDomain:
		return text(l, "scope.domain", nil), false
	}
	switch p.Adapter {
	case "postmark":
		return text(l, "scope.stream", nil), false
	case "mailgun":
		return text(l, "scope.domain", nil), false
	case "smtp", "postal":
		return text(l, "scope.enforced", nil), false
	}
	return text(l, "scope.list", nil), false
}

// Status returns the sync status of every provider.
func (s *Service) Status(ctx context.Context, ws *store.Workspace) (SuppressionSyncStatus, error) {
	l := i18n.LangFrom(ctx)
	out := SuppressionSyncStatus{Reasons: []SuppressionReasonCount{}, Pushed: []SuppressionReasonCount{}, Kept: []SuppressionReasonCount{}, Rows: []SuppressionSyncRow{}}
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
	summary, err := s.Q.SuppPushSummary(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	kept := map[string]int64{}
	var keptOrder []string
	for _, r := range summary {
		switch {
		case r.Pushed:
			out.Pushed = append(out.Pushed, SuppressionReasonCount{Reason: r.Reason, Count: r.N})
			out.PushedTotal += r.N
			continue
		case r.Scope == "marketing":
			out.MarketingOnly += r.N
		default:
			out.KeptAllMail += r.N
		}
		if _, ok := kept[r.Reason]; !ok {
			keptOrder = append(keptOrder, r.Reason)
		}
		kept[r.Reason] += r.N
	}
	for _, r := range keptOrder {
		out.Kept = append(out.Kept, SuppressionReasonCount{Reason: r, Count: kept[r]})
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
		r, ok := byID[p.ID]
		scopeText, readOnly := rowScope(l, p, r.Scope)
		row := SuppressionSyncRow{ProviderID: p.ID.String(), Provider: p.Name, Scope: scopeText, ReadOnly: readOnly}
		if p.Adapter == "ses" {
			row.Tenant = sesTenant(p)
		}
		if !state.Caps(p).SuppressionSync {
			// No suppression API: its bounce and complaint events still add
			// to the shared list, which is enforced before every send.
			row.Direction, row.Status, row.Detail = "local", "local", text(l, "detail.local", i18n.Args{"provider": p.Name})
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
			if readOnly {
				row.Direction = "pull"
			} else {
				pushed := int64(0)
				row.Pushed = &pushed
			}
			out.Rows = append(out.Rows, row)
			continue
		}
		row.Direction, row.Pulled, row.Pending = r.Mode, r.Pulled, r.Pending
		if readOnly && r.Mode != "off" {
			row.Direction, row.Pending = "pull", 0
			row.Detail = text(l, "detail.accountReadOnly", i18n.Args{"provider": p.Name})
		} else {
			pushed := r.Pushed
			row.Pushed = &pushed
		}
		row.LastSync = r.LastPushAt
		if r.LastPullAt != nil && (row.LastSync == nil || r.LastPullAt.After(*row.LastSync)) {
			row.LastSync = r.LastPullAt
		}
		switch {
		case r.Mode == "off":
			row.Status = "off"
		case r.Status != "ok":
			row.Status, row.Detail = "error", r.StatusDetail
		case row.Pending > 0:
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
		out.Rows = append(out.Rows, SuppressionSyncRow{Provider: text(l, "provider.import", i18n.Args{"name": name}), Scope: l.Region.DayMonth(at), Direction: "import", Pulled: im.N,
			LastSync: &at, Status: "done"})
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
