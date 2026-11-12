package coupons

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Timings of creating codes in a store.
const (
	// statusPoll is how often a store's background job is asked about.
	statusPoll = 5 * time.Second
	// stepBudget is how long one run of the job works before it snoozes.
	stepBudget = 2 * time.Minute
)

// CouponGenerateInput asks a store pool for more codes.
type CouponGenerateInput struct {
	Count int `json:"count" minimum:"1" maximum:"50000"`
}

// GenerateArgs creates a batch's codes in its store (coupons.generate).
type GenerateArgs struct {
	BatchID string `json:"batch_id"`
}

// Kind implements river.JobArgs.
func (GenerateArgs) Kind() string { return "coupons.generate" }

// InsertOpts: one job per batch.
func (GenerateArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 10, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// GenerateWorker runs GenerateArgs.
type GenerateWorker struct {
	river.WorkerDefaults[GenerateArgs]
	S *Service
}

// Timeout leaves the step budget room.
func (w *GenerateWorker) Timeout(*river.Job[GenerateArgs]) time.Duration {
	return stepBudget + 3*time.Minute
}

// Work implements river.Worker: the store's throttle and its background
// job snooze it; the last failed attempt fails the batch.
func (w *GenerateWorker) Work(ctx context.Context, job *river.Job[GenerateArgs]) error {
	id, err := uuid.Parse(job.Args.BatchID)
	if err != nil {
		return river.JobCancel(err)
	}
	again, err := w.S.Generate(ctx, id)
	if wait, ok := stores.IsRateLimited(err); ok {
		// The pool page says the store asked to slow down until then.
		until := w.S.now().Add(wait + time.Second)
		if err := w.S.Q.CouponBatchWaiting(ctx, store.CouponBatchWaitingParams{ID: id, Until: until}); err != nil {
			w.S.log().WarnContext(ctx, "coupons: note the store's throttle", "batch", id, "err", err)
		}
		return river.JobSnooze(wait + time.Second)
	}
	if err != nil {
		if job.Attempt >= job.MaxAttempts {
			w.S.failBatch(context.WithoutCancel(ctx), id, FailUnreachable)
		} else {
			// The pool page says the store isn't answering and when
			// ReputeCast tries again.
			at := (&river.DefaultClientRetryPolicy{}).NextRetry(job.JobRow)
			if err := w.S.Q.CouponBatchRetrying(context.WithoutCancel(ctx), store.CouponBatchRetryingParams{ID: id, Error: FailUnreachable, Until: at}); err != nil {
				w.S.log().WarnContext(ctx, "coupons: note the retry", "batch", id, "err", err)
			}
		}
		return err
	}
	if again > 0 {
		return river.JobSnooze(again)
	}
	return nil
}

// RemoveArgs removes a store pool's removed codes from the store too
// (coupons.remove).
type RemoveArgs struct {
	PoolID string `json:"pool_id"`
}

// Kind implements river.JobArgs.
func (RemoveArgs) Kind() string { return "coupons.remove" }

// InsertOpts: one job per pool.
func (RemoveArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 10, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// RemoveWorker runs RemoveArgs.
type RemoveWorker struct {
	river.WorkerDefaults[RemoveArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RemoveWorker) Work(ctx context.Context, job *river.Job[RemoveArgs]) error {
	id, err := uuid.Parse(job.Args.PoolID)
	if err != nil {
		return river.JobCancel(err)
	}
	err = w.S.RemoveAtStore(ctx, id)
	if wait, ok := stores.IsRateLimited(err); ok {
		return river.JobSnooze(wait + time.Second)
	}
	return err
}

// RefillArgs is the hourly job (coupons.refill): store pools that live
// automations send keep max(low mark, 1.2 × the last 7 days' use) free
// codes; pools under their low mark tell coupons:manage holders once;
// batches and removals left unfinished go on.
type RefillArgs struct{}

// Kind implements river.JobArgs.
func (RefillArgs) Kind() string { return "coupons.refill" }

// InsertOpts: one an hour.
func (RefillArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// RefillWorker runs RefillArgs.
type RefillWorker struct {
	river.WorkerDefaults[RefillArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RefillWorker) Work(ctx context.Context, _ *river.Job[RefillArgs]) error {
	return w.S.Refill(ctx)
}

// RegisterJobs adds the workers and the hourly schedule.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &GenerateWorker{S: s})
	river.AddWorker(reg.Workers, &RemoveWorker{S: s})
	river.AddWorker(reg.Workers, &RefillWorker{S: s})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RefillArgs{}, nil }, false)
}

// RequestCodes asks a store pool for count more codes: a batch and its
// job, audited (coupon_pool.codes_requested).
func (s *Service) RequestCodes(ctx context.Context, ws *store.Workspace, id uuid.UUID, count int) (CouponBatch, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponBatch{}, err
	}
	if count < 1 || count > MaxGenerate {
		return CouponBatch{}, invalid(ctx, "body.count", mCodesCount, nil)
	}
	switch {
	case p.Status != StatusActive:
		return CouponBatch{}, conflict(ctx, "pool_archived", mPoolArchived, i18n.Args{"name": p.Name})
	case p.Source == SourceUpload:
		return CouponBatch{}, conflict(ctx, "upload_pool", mCodesUploadPool, i18n.Args{"name": p.Name})
	case p.StoreID == nil:
		return CouponBatch{}, conflict(ctx, "store_gone", mCodesStoreGone, i18n.Args{"name": p.Name})
	}
	src, err := s.source(ctx, ws, *p.StoreID)
	if err != nil {
		return CouponBatch{}, err
	}
	if !src.Can {
		return CouponBatch{}, api.Conflict(src.Detail).WithMeta("reason", src.Reason)
	}
	reason := "request"
	if first, err := s.Q.CouponPoolHasBatch(ctx, p.ID); err != nil {
		return CouponBatch{}, err
	} else if !first {
		reason = "pool"
	}
	by := actorUser(api.PrincipalFrom(ctx))
	b, err := s.request(ctx, p, count, reason, nil, by, nil)
	if errors.Is(err, errRunning) {
		r, _ := s.Q.CouponBatchRunning(ctx, p.ID)
		return CouponBatch{}, conflict(ctx, "creating", mCodesRunning, i18n.Args{"name": p.Name, "done": r.Created, "requested": r.Requested})
	}
	if err != nil {
		return CouponBatch{}, err
	}
	name := ""
	if by != nil {
		if u, err := s.Q.GetUser(ctx, *by); err == nil {
			name = u.Name
		}
	}
	return *batchView(b, name, ""), nil
}

var errRunning = errors.New("coupons: a batch is running")

// request adds a running batch and its job, audited: reason is pool,
// request or refill (for automation); actor overrides the request's (the
// hourly refill).
func (s *Service) request(ctx context.Context, p store.CouponPoolsRow, count int, reason string, automation *uuid.UUID, by *uuid.UUID, actor *audit.Actor) (store.CouponBatch, error) {
	var b store.CouponBatch
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id, err := q.CouponBatchInsert(ctx, store.CouponBatchInsertParams{ID: newID(), WorkspaceID: p.WorkspaceID, PoolID: p.ID, Kind: "generate",
			Reason: reason, AutomationID: automation, Requested: int32(count), Status: "running", CreatedBy: by, Now: s.now()})
		if isUnique(err) {
			return errRunning
		}
		if err != nil {
			return err
		}
		if b, err = q.CouponBatch(ctx, id); err != nil {
			return err
		}
		if c := s.jobs(); c != nil {
			if _, err := c.InsertTx(ctx, tx, GenerateArgs{BatchID: id.String()}, nil); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &p.WorkspaceID, Action: "coupon_pool.codes_requested", TargetType: "coupon_pool",
			TargetID: p.ID.String(), TargetName: p.Name, Actor: actor,
			Metadata: map[string]any{"summary": strconv.Itoa(count) + " codes requested in " + p.StoreName, "count": count, "batch": id.String(), "reason": reason}})
	})
	return b, err
}

// spec is a store pool's discount.
func spec(p store.CouponPoolsRow) stores.DiscountSpec {
	set := settingsOf(p.Settings)
	return stores.DiscountSpec{Title: p.Name, Kind: set.Kind, Value: set.Value, Currency: p.StoreCurrency, MinimumCents: set.MinimumCents,
		EndsAt: endsAt(p, set), CollectionIDs: set.CollectionIDs}
}

// Generate works on a batch for up to stepBudget: it creates the store's
// discount once, then adds the codes MaxCodesPerCall at a time, following
// the store's background job. Only codes the store confirms become free.
// It returns how long to wait before going on (0: done).
func (s *Service) Generate(ctx context.Context, batchID uuid.UUID) (time.Duration, error) {
	b, err := s.Q.CouponBatch(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if b.Status != "running" || b.Kind != "generate" {
		return 0, nil
	}
	rows, err := s.pools(ctx, b.WorkspaceID, &b.PoolID, nil)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 || rows[0].StoreID == nil {
		s.failBatch(ctx, b.ID, FailStoreGone)
		return 0, nil
	}
	p := rows[0]
	if p.Status != StatusActive {
		s.failBatch(ctx, b.ID, FailArchived)
		return 0, nil
	}
	sp := spec(p)
	if sp.EndsAt != nil && !s.now().Before(*sp.EndsAt) {
		s.failBatch(ctx, b.ID, FailEnded)
		return 0, nil
	}
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return 0, err
	}
	if s.Discounts == nil {
		s.failBatch(ctx, b.ID, FailUnsupported)
		return 0, nil
	}
	dc, err := s.Discounts(ctx, &ws, *p.StoreID)
	if err != nil {
		return s.stepError(ctx, b.ID, err)
	}
	created, failed := int(b.Created), int(b.Failed)
	discountID := p.ExternalDiscountID
	if b.ExternalJobID != "" {
		pending, err := s.openPending(b)
		if err != nil {
			return 0, err
		}
		res, err := dc.CodesStatus(ctx, storeDiscount(discountID), b.ExternalJobID, pending)
		if err != nil {
			return s.stepError(ctx, b.ID, err)
		}
		if !res.Done {
			return statusPoll, nil
		}
		if created, failed, err = s.confirm(ctx, p, b.ID, pending, res, created, failed); err != nil {
			return 0, err
		}
	}
	deadline := s.now().Add(stepBudget)
	for created+failed < int(b.Requested) && s.now().Before(deadline) {
		if discountID == "" {
			first := newCodes(p.Prefix, 1)
			id, firstID, err := dc.CreateDiscount(ctx, sp, first[0])
			if err != nil {
				return s.stepError(ctx, b.ID, err)
			}
			// A platform without a parent discount (WooCommerce, every code
			// its own coupon) is noted too, so the discount is made once.
			if id == "" {
				id = noParent
			}
			discountID = id
			if err := s.Q.CouponPoolSetDiscount(ctx, store.CouponPoolSetDiscountParams{ID: p.ID, ExternalDiscountID: id, Now: s.now()}); err != nil {
				return 0, err
			}
			res := stores.DiscountCodes{Done: true, IDs: map[string]string{first[0]: firstID}}
			if created, failed, err = s.confirm(ctx, p, b.ID, first, res, created, failed); err != nil {
				return 0, err
			}
			continue
		}
		codes := newCodes(p.Prefix, min(max(dc.MaxCodesPerCall(), 1), int(b.Requested)-created-failed))
		res, err := dc.AddCodes(ctx, storeDiscount(discountID), sp, codes)
		if err != nil {
			return s.stepError(ctx, b.ID, err)
		}
		if !res.Done {
			sealed, err := s.sealPending(b.ID, codes)
			if err != nil {
				return 0, err
			}
			if err := s.Q.CouponBatchProgress(ctx, store.CouponBatchProgressParams{ID: b.ID, Created: int32(created), Failed: int32(failed),
				ExternalJobID: res.JobID, PendingSealed: sealed}); err != nil {
				return 0, err
			}
			return statusPoll, nil
		}
		if created, failed, err = s.confirm(ctx, p, b.ID, codes, res, created, failed); err != nil {
			return 0, err
		}
	}
	if created+failed < int(b.Requested) {
		return time.Second, nil
	}
	if err := s.Q.CouponBatchFinish(ctx, store.CouponBatchFinishParams{ID: b.ID, Status: "done", Created: int32(created), Failed: int32(failed), At: s.now()}); err != nil {
		return 0, err
	}
	if free, err := s.Q.CouponPoolFree(ctx, p.ID); err == nil {
		s.recheckLow(ctx, p, free)
	}
	return 0, nil
}

// noParent is a pool's external_discount_id when its platform has no
// parent discount (the discount was made: its first code exists).
const noParent = "-"

// storeDiscount is the discount id a store knows ("" for a platform
// without one).
func storeDiscount(id string) string {
	if id == noParent {
		return ""
	}
	return id
}

// confirm stores the codes the store created (res.IDs, or every code not
// in res.Failed when the store gives no ids) and the batch's progress, in
// one transaction.
func (s *Service) confirm(ctx context.Context, p store.CouponPoolsRow, batch uuid.UUID, codes []string, res stores.DiscountCodes, created, failed int) (int, int, error) {
	refused := map[string]bool{}
	for _, c := range res.Failed {
		refused[c] = true
	}
	var ok, ids []string
	for _, c := range codes {
		id, has := res.IDs[c]
		if refused[c] || (!has && len(res.IDs) > 0) {
			continue
		}
		ok, ids = append(ok, c), append(ids, id)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		added, err := s.insertCodes(ctx, q, p, batch, ok, ids)
		if err != nil {
			return err
		}
		created += int(added)
		failed += len(codes) - int(added)
		return q.CouponBatchProgress(ctx, store.CouponBatchProgressParams{ID: batch, Created: int32(created), Failed: int32(failed)})
	})
	return created, failed, err
}

func (s *Service) sealPending(batch uuid.UUID, codes []string) ([]byte, error) {
	raw, err := json.Marshal(codes)
	if err != nil {
		return nil, err
	}
	return s.Box.Seal(raw, pendingAD(batch))
}

func (s *Service) openPending(b store.CouponBatch) ([]string, error) {
	if len(b.PendingSealed) == 0 {
		return nil, nil
	}
	raw, err := s.Box.Open(b.PendingSealed, pendingAD(b.ID))
	if err != nil {
		return nil, err
	}
	var codes []string
	return codes, json.Unmarshal(raw, &codes)
}

// stepError ends a batch the store refuses (a permission it lacks, a store
// not connected: the problem's words) and passes other errors on (a
// throttle snoozes the job; the rest retry).
func (s *Service) stepError(ctx context.Context, batch uuid.UUID, err error) (time.Duration, error) {
	if _, ok := stores.IsRateLimited(err); ok {
		return 0, err
	}
	var p *api.Problem
	if errors.As(err, &p) {
		why := FailUnsupported
		if r, ok := p.Meta["reason"].(string); ok && r != "" {
			why = r
		}
		s.failBatch(ctx, batch, why)
		return 0, nil
	}
	if errors.Is(err, stores.ErrPermission) {
		s.failBatch(ctx, batch, FailNotGranted)
		return 0, nil
	}
	return 0, err
}

// Why a batch failed (coupon_batches.error; the pool page says it in the
// reader's language): the store's refusals (commerce's reasons), the
// store not answering, or the pool changing under it.
const (
	FailUnreachable = "unreachable"
	FailNotGranted  = "not_granted"
	FailStoreGone   = "store_gone"
	FailArchived    = "archived"
	FailEnded       = "ended"
	FailUnsupported = "unsupported"
)

// failBatch ends a batch with why (a Fail* code or one of commerce's
// refusal reasons).
func (s *Service) failBatch(ctx context.Context, batch uuid.UUID, why string) {
	b, err := s.Q.CouponBatch(ctx, batch)
	if err != nil {
		return
	}
	if err := s.Q.CouponBatchFinish(ctx, store.CouponBatchFinishParams{ID: batch, Status: "failed", Error: clip(why, 500), Created: b.Created, Failed: b.Failed,
		At: s.now()}); err != nil {
		s.log().WarnContext(ctx, "coupons: fail a batch", "batch", batch, "err", err)
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// RemoveAtStore removes a store pool's removed codes in the store, a
// call's worth at a time; each code removed there loses its store id.
func (s *Service) RemoveAtStore(ctx context.Context, poolID uuid.UUID) error {
	p0, err := s.Q.CouponPoolByID(ctx, poolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	p, err := s.pool(ctx, p0.WorkspaceID, poolID)
	if err != nil {
		return err
	}
	if p.StoreID == nil || s.Discounts == nil {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	dc, err := s.Discounts(ctx, &ws, *p.StoreID)
	var prob *api.Problem
	if errors.As(err, &prob) {
		return nil // a store that can't: the hourly job tries again once it can
	}
	if err != nil {
		return err
	}
	for {
		rows, err := s.Q.CouponCodesUnstored(ctx, store.CouponCodesUnstoredParams{PoolID: p.ID, Lim: int32(max(dc.MaxCodesPerCall(), 1))})
		if err != nil || len(rows) == 0 {
			return err
		}
		codes, ids, done := make([]string, 0, len(rows)), make([]string, 0, len(rows)), make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			c, err := s.Box.OpenString(r.CodeSealed, codeAD(p.ID))
			if err != nil {
				return err
			}
			codes, ids, done = append(codes, c), append(ids, r.ExternalID), append(done, r.ID)
		}
		if err := dc.RemoveCodes(ctx, storeDiscount(p.ExternalDiscountID), codes, ids); err != nil {
			return err
		}
		if err := s.Q.CouponCodesStoreRemoved(ctx, done); err != nil {
			return err
		}
	}
}

// Refill is one pass of the hourly job.
func (s *Service) Refill(ctx context.Context) error {
	pools, err := s.Q.CouponPoolsActive(ctx)
	if err != nil {
		return err
	}
	live := map[uuid.UUID][]store.CouponLiveAutomationEmailsRow{}
	for _, p := range pools {
		if p.Source != SourceUpload && p.StoreStatus == "connected" {
			emails, ok := live[p.WorkspaceID]
			if !ok {
				if emails, err = s.Q.CouponLiveAutomationEmails(ctx, p.WorkspaceID); err != nil {
					return err
				}
				live[p.WorkspaceID] = emails
			}
			if flow, ok := usedByLive(emails, p.Slug); ok {
				if err := s.refill(ctx, p, flow); err != nil {
					s.log().WarnContext(ctx, "coupons: refill a pool", "pool", p.ID, "err", err)
				}
			}
		}
		row := poolOf(p)
		if p.Free < int64(p.LowAt) {
			s.alertLow(ctx, row, p.Free)
		} else if p.NotifiedLowAt != nil {
			s.recheckLow(ctx, row, p.Free)
		}
	}
	// Batches and removals whose job went missing go on.
	if c := s.jobs(); c != nil {
		running, err := s.Q.CouponBatchesRunningAll(ctx)
		if err != nil {
			return err
		}
		for _, id := range running {
			if _, err := c.Insert(ctx, GenerateArgs{BatchID: id.String()}, nil); err != nil {
				return err
			}
		}
		unstored, err := s.Q.CouponPoolsUnstored(ctx)
		if err != nil {
			return err
		}
		for _, id := range unstored {
			if _, err := c.Insert(ctx, RemoveArgs{PoolID: id.String()}, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// usedByLive is the first live automation whose emails show the pool's
// codes.
func usedByLive(emails []store.CouponLiveAutomationEmailsRow, slug string) (uuid.UUID, bool) {
	for _, e := range emails {
		if names(e.Content, slug) {
			return e.ID, true
		}
	}
	return uuid.Nil, false
}

// refill asks a store pool for codes up to max(low mark, 1.2 × the last 7
// days' use), unless a batch is running.
func (s *Service) refill(ctx context.Context, p store.CouponPoolsActiveRow, flow uuid.UUID) error {
	if err := s.Q.CouponPoolRefillChecked(ctx, store.CouponPoolRefillCheckedParams{ID: p.ID, At: s.now()}); err != nil {
		return err
	}
	used, err := s.Q.CouponAssignedSince(ctx, store.CouponAssignedSinceParams{PoolID: p.ID, Since: s.now().Add(-7 * 24 * time.Hour)})
	if err != nil {
		return err
	}
	target := max(int64(p.LowAt), int64(math.Ceil(1.2*float64(used))))
	if p.Free >= target {
		return nil
	}
	rows, err := s.pools(ctx, p.WorkspaceID, &p.ID, nil)
	if err != nil || len(rows) == 0 {
		return err
	}
	_, err = s.request(ctx, rows[0], int(min(target-p.Free, MaxGenerate)), "refill", &flow, nil, &audit.Actor{Type: authz.ActorSystem, Name: "System · hourly refill"})
	if errors.Is(err, errRunning) {
		return nil
	}
	return err
}

// poolOf is an active pool's row as the rest of the package reads it.
func poolOf(p store.CouponPoolsActiveRow) store.CouponPoolsRow {
	return store.CouponPoolsRow{ID: p.ID, WorkspaceID: p.WorkspaceID, Name: p.Name, Slug: p.Slug, Source: p.Source, StoreID: p.StoreID, Settings: p.Settings,
		Prefix: p.Prefix, OnePerContact: p.OnePerContact, LowAt: p.LowAt, Status: p.Status, ExternalDiscountID: p.ExternalDiscountID,
		NotifiedLowAt: p.NotifiedLowAt, CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, StoreName: p.StoreName, StoreStatus: p.StoreStatus}
}

// alertLow tells the members with coupons:manage once that a pool is
// under its low mark (or out of codes); the alert comes back only after
// the pool was above the mark again.
func (s *Service) alertLow(ctx context.Context, p store.CouponPoolsRow, free int64) {
	ctx = context.WithoutCancel(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.CouponPoolMarkLow(ctx, store.CouponPoolMarkLowParams{ID: p.ID, At: s.now()}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		ws, err := q.GetWorkspace(ctx, p.WorkspaceID)
		if err != nil {
			return err
		}
		n := notify.Notification{Kind: notify.KindCouponCodesLow, Severity: "warn", Link: "/w/" + ws.Slug + "/campaigns/codes/" + p.ID.String(),
			TitleText: mNotifyLowTitle.Keep(i18n.Args{"name": p.Name}), BodyText: mNotifyLowBody.Keep(i18n.Args{"free": free, "low": p.LowAt}),
			Data: map[string]any{"pool": p.ID.String(), "free": free}}
		if free <= 0 {
			n.Severity, n.TitleText, n.BodyText = "bad", mNotifyOutTitle.Keep(i18n.Args{"name": p.Name}), mNotifyOutBody.Keep(nil)
		}
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.CouponsManage, n)
	})
	if err != nil {
		s.log().WarnContext(ctx, "coupons: the low alert", "pool", p.ID, "err", err)
	}
}
