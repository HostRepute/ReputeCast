package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// importProgress is the first import's state (stores.import_progress): the
// phase, its page cursor and the numbers so far, so a stopped import
// resumes where it was.
type importProgress struct {
	Phase            string           `json:"phase"`
	Months           int              `json:"months"`
	By               string           `json:"by,omitempty"`
	Products         CommerceProgress `json:"products"`
	Collections      int              `json:"collections"`
	Customers        CommerceProgress `json:"customers"`
	Accepted         int              `json:"accepted"`
	Known            int              `json:"known"`
	NotAdded         int              `json:"notAdded"`
	CustomersSkipped bool             `json:"customersSkipped"`
	Orders           CommerceProgress `json:"orders"`
	Matching         CommerceProgress `json:"matching"`
	Cursor           string           `json:"cursor,omitempty"`
	MatchAfter       string           `json:"matchAfter,omitempty"`
	OrdersStarted    *time.Time       `json:"ordersStarted,omitempty"`
	Since            time.Time        `json:"since"`
}

func progressOf(raw json.RawMessage) importProgress {
	var p importProgress
	_ = json.Unmarshal(raw, &p)
	if p.Phase == "" {
		p.Phase = "products"
	}
	return p
}

// ImportArgs runs (or resumes) a store's first import.
type ImportArgs struct {
	StoreID string `json:"store_id"`
}

// Kind implements river.JobArgs.
func (ImportArgs) Kind() string { return "commerce.import" }

// InsertOpts: one import per store at a time.
func (ImportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 25, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// ImportWorker runs first imports.
type ImportWorker struct {
	river.WorkerDefaults[ImportArgs]
	S *Service
}

// Timeout lets an import run long; it snoozes itself before the limit.
func (w *ImportWorker) Timeout(*river.Job[ImportArgs]) time.Duration { return 15 * time.Minute }

// Work implements river.Worker.
func (w *ImportWorker) Work(ctx context.Context, job *river.Job[ImportArgs]) error {
	id, err := uuid.Parse(job.Args.StoreID)
	if err != nil {
		return river.JobCancel(err)
	}
	again, err := w.S.Import(ctx, id, 10*time.Minute)
	if err != nil {
		return err
	}
	if again > 0 {
		return river.JobSnooze(again)
	}
	return nil
}

// Import runs a store's first import for up to budget: products and
// collections, customers who accepted marketing (Shopify), orders of the
// last 3, 6 or 12 months, then past orders matched to past clicks. It
// returns how long to wait before running again (0 when done).
func (s *Service) Import(ctx context.Context, id uuid.UUID, budget time.Duration) (time.Duration, error) {
	st, err := s.Q.CommerceStoreByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if st.Status != StatusImporting {
		return 0, nil
	}
	c, err := s.client(st)
	if err != nil {
		return 0, err
	}
	p := progressOf(st.ImportProgress)
	if p.Since.IsZero() {
		p.Since = s.now().AddDate(0, -int(st.ImportMonths), 0)
	}
	deadline := s.now().Add(budget)
	save := func() error {
		return s.Q.CommerceStoreProgress(ctx, store.CommerceStoreProgressParams{ID: st.ID, ImportProgress: mustJSON(p), At: s.now()})
	}
	for p.Phase != "done" {
		if s.now().After(deadline) {
			return time.Second, save()
		}
		var err error
		switch p.Phase {
		case "products":
			err = s.importProducts(ctx, c, st, &p)
		case "customers":
			err = s.importCustomers(ctx, c, st, &p)
		case "orders":
			err = s.importOrders(ctx, c, st, &p)
		case "matching":
			err = s.importMatching(ctx, st, &p)
		default:
			p.Phase = "done"
		}
		if err != nil {
			if wait, ok := stores.IsRateLimited(err); ok {
				_ = s.rateLimited(ctx, st, wait)
				if saveErr := save(); saveErr != nil {
					return 0, saveErr
				}
				if wait < 30*time.Second {
					select {
					case <-ctx.Done():
						return time.Second, nil
					case <-time.After(wait):
					}
					continue
				}
				return wait, nil
			}
			if errors.Is(err, stores.ErrUnauthorized) {
				return 0, s.revoke(ctx, st, "revoked", s.now())
			}
			_ = save()
			return 0, err
		}
		if err := save(); err != nil {
			return 0, err
		}
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommerceStoreImportDone(ctx, store.CommerceStoreImportDoneParams{ID: st.ID, At: now, ImportProgress: mustJSON(p)}); err != nil {
			return err
		}
		minutes := 0
		if st.ImportStartedAt != nil {
			minutes = int(now.Sub(*st.ImportStartedAt).Minutes())
		}
		return s.activity(ctx, q, st, activity{Kind: "import_finished", Tone: "ok", Products: p.Products.Done, Missed: p.Orders.Done, Hours: minutes})
	})
	return 0, err
}

// rateLimited records that the store throttled ReputeCast: the sync slows
// down and the store's page says so.
func (s *Service) rateLimited(ctx context.Context, st store.Store, wait time.Duration) error {
	until := s.now().Add(wait)
	return s.Q.CommerceStoreRateLimited(ctx, store.CommerceStoreRateLimitedParams{ID: st.ID, Until: &until, At: s.now()})
}

func (s *Service) importProducts(ctx context.Context, c stores.Client, st store.Store, p *importProgress) error {
	if !st.SyncProducts {
		p.Phase, p.Cursor = s.afterProducts(st), ""
		return nil
	}
	if p.Cursor == "" && p.Products.Total == 0 {
		// Collections first: products name theirs.
		if err := s.importCollections(ctx, c, st, p); err != nil {
			return err
		}
		n, err := c.Count(ctx, stores.CountProducts, stores.Range{})
		if err != nil && !errors.Is(err, stores.ErrUnsupported) {
			return err
		}
		p.Products.Total = n
	}
	page, err := c.Products(ctx, p.Cursor, stores.Range{})
	if err != nil {
		return err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, pr := range page.Items {
			if _, err := s.applyProduct(ctx, q, st, pr, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.Products.Done += len(page.Items)
	p.Cursor = page.Next
	if page.Next == "" {
		if p.Products.Total < p.Products.Done {
			p.Products.Total = p.Products.Done
		}
		p.Phase, p.Cursor = s.afterProducts(st), ""
	}
	return nil
}

func (s *Service) afterProducts(st store.Store) string {
	if st.Platform == Shopify && st.SyncCustomers {
		return "customers"
	}
	if st.SyncOrders {
		return "orders"
	}
	return "matching"
}

func (s *Service) importCollections(ctx context.Context, c stores.Client, st store.Store, p *importProgress) error {
	cursor := ""
	pos := int32(0)
	for {
		page, err := c.Collections(ctx, cursor)
		if err != nil {
			return err
		}
		for _, col := range page.Items {
			if err := s.Q.CommerceCollectionUpsert(ctx, store.CommerceCollectionUpsertParams{StoreID: st.ID, ExternalID: col.ID, Title: col.Title,
				Handle: col.Handle, Products: int32(col.Products), Position: pos}); err != nil {
				return err
			}
			pos++
		}
		p.Collections += len(page.Items)
		if page.Next == "" {
			return nil
		}
		cursor = page.Next
	}
}

func (s *Service) importCustomers(ctx context.Context, c stores.Client, st store.Store, p *importProgress) error {
	if p.Cursor == "" && p.Customers.Total == 0 {
		n, err := c.Count(ctx, stores.CountCustomers, stores.Range{})
		if errors.Is(err, stores.ErrUnsupported) {
			p.CustomersSkipped = true
			p.Phase = "orders"
			return nil
		}
		if err != nil {
			return err
		}
		p.Customers.Total = n
	}
	page, err := c.Customers(ctx, p.Cursor, stores.Range{})
	if errors.Is(err, stores.ErrUnsupported) {
		p.CustomersSkipped, p.Phase, p.Cursor = true, "orders", ""
		return nil
	}
	if err != nil {
		return err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, cu := range page.Items {
			if !cu.Consent.Accepted() {
				p.NotAdded++
				if ct, ok := s.contactOf(ctx, q, st.WorkspaceID, cu.Email); ok {
					if err := s.link(ctx, q, st, cu, ct.ID); err != nil {
						return err
					}
				}
				continue
			}
			j, err := s.consent(ctx, q, st, cu, s.now())
			if err != nil {
				return err
			}
			if j.contactID != nil {
				p.Accepted++
				if !j.created {
					p.Known++
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.Customers.Done += len(page.Items)
	p.Cursor = page.Next
	if page.Next == "" {
		if p.Customers.Total < p.Customers.Done {
			p.Customers.Total = p.Customers.Done
		}
		p.Phase, p.Cursor = "orders", ""
		if !st.SyncOrders {
			p.Phase = "matching"
		}
	}
	return nil
}

func (s *Service) importOrders(ctx context.Context, c stores.Client, st store.Store, p *importProgress) error {
	r := stores.Range{CreatedSince: p.Since}
	if p.Cursor == "" && p.Orders.Total == 0 {
		n, err := c.Count(ctx, stores.CountOrders, r)
		if err != nil && !errors.Is(err, stores.ErrUnsupported) {
			return err
		}
		p.Orders.Total = n
		now := s.now()
		p.OrdersStarted = &now
	}
	page, err := c.Orders(ctx, p.Cursor, r)
	if err != nil {
		return err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, o := range page.Items {
			res, err := s.applyOrder(ctx, q, st, o, sourceImport, "")
			if err != nil {
				return err
			}
			// A click token in the landing page credits the order now; the
			// last phase matches the rest by email.
			if res.newOrder && o.ClickToken != "" && !o.Test {
				row, err := q.CommerceOrder(ctx, store.CommerceOrderParams{StoreID: st.ID, ExternalID: o.ID})
				if err == nil {
					if cr := s.match(ctx, q, st, o, nil); !cr.none() {
						if err := s.record(ctx, q, row.ID, cr); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.Orders.Done += len(page.Items)
	p.Cursor = page.Next
	if page.Next == "" {
		if p.Orders.Total < p.Orders.Done {
			p.Orders.Total = p.Orders.Done
		}
		p.Phase, p.Cursor = "matching", ""
		if st.Platform == WooCommerce {
			// Consent came with the orders.
			if stats, err := s.Q.CommerceCustomerStats(ctx, st.ID); err == nil {
				p.Accepted = int(stats.Joined)
			}
		}
	}
	return nil
}

// importMatching credits past orders to past clicks, by email.
func (s *Service) importMatching(ctx context.Context, st store.Store, p *importProgress) error {
	after := uuid.Nil
	if p.MatchAfter != "" {
		if id, err := uuid.Parse(p.MatchAfter); err == nil {
			after = id
		}
	}
	rows, err := s.Q.CommerceOrdersToMatch(ctx, store.CommerceOrdersToMatchParams{StoreID: st.ID, After: after, Lim: 500})
	if err != nil {
		return err
	}
	if p.Matching.Total == 0 && p.MatchAfter == "" {
		p.Matching.Total = len(rows)
		if len(rows) == 500 {
			p.Matching.Total = p.Orders.Done
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, r := range rows {
			cr := s.match(ctx, q, st, stores.Order{PlacedAt: r.PlacedAt}, r.ContactID)
			if err := s.record(ctx, q, r.ID, cr); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.Matching.Done += len(rows)
	if len(rows) > 0 {
		p.MatchAfter = rows[len(rows)-1].ID.String()
	}
	if len(rows) < 500 {
		if p.Matching.Total < p.Matching.Done {
			p.Matching.Total = p.Matching.Done
		}
		p.Phase = "done"
	}
	return nil
}

// ReconcileArgs reconciles a store: what changed since a time is fetched
// again, and every product checked.
type ReconcileArgs struct {
	StoreID string    `json:"store_id"`
	Since   time.Time `json:"since,omitzero"`
	// By is the person who asked ("Sync now").
	By string `json:"by,omitempty"`
}

// Kind implements river.JobArgs.
func (ReconcileArgs) Kind() string { return "commerce.reconcile" }

// InsertOpts: one reconciliation per store at a time.
func (ReconcileArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Minute}}
}

// ReconcileWorker runs reconciliations.
type ReconcileWorker struct {
	river.WorkerDefaults[ReconcileArgs]
	S *Service
}

// Timeout gives a big store's catalog time.
func (w *ReconcileWorker) Timeout(*river.Job[ReconcileArgs]) time.Duration { return 15 * time.Minute }

// Work implements river.Worker.
func (w *ReconcileWorker) Work(ctx context.Context, job *river.Job[ReconcileArgs]) error {
	id, err := uuid.Parse(job.Args.StoreID)
	if err != nil {
		return river.JobCancel(err)
	}
	var by *uuid.UUID
	if u, err := uuid.Parse(job.Args.By); err == nil {
		by = &u
	}
	err = w.S.Reconcile(ctx, id, job.Args.Since, by)
	if wait, ok := stores.IsRateLimited(err); ok {
		return river.JobSnooze(wait + time.Second)
	}
	return err
}

// Reconcile fetches what changed in a store since a time (the last
// reconciliation, an hour of overlap) and checks every product: orders
// that hadn't arrived count as missed; products the store no longer has
// are flagged removed.
func (s *Service) Reconcile(ctx context.Context, id uuid.UUID, since time.Time, by *uuid.UUID) error {
	st, err := s.Q.CommerceStoreByID(ctx, id)
	if err != nil {
		return err
	}
	if st.Status != StatusConnected {
		return nil
	}
	if since.IsZero() {
		since = s.now().Add(-25 * time.Hour)
		if st.ReconciledAt != nil {
			since = st.ReconciledAt.Add(-time.Hour)
		}
	}
	missed, checked := 0, 0
	if st.Demo {
		// The demo's sample store is never polled: its catalog and orders
		// are the demo's, and the check finds nothing missed.
		n, err := s.Q.CommerceProductCount(ctx, st.ID)
		if err != nil {
			return err
		}
		return s.reconciled(ctx, st, by, 0, int(n))
	}
	c, err := s.client(st)
	if err != nil {
		return err
	}
	r := stores.Range{UpdatedSince: since}
	fail := func(err error) error {
		if wait, ok := stores.IsRateLimited(err); ok {
			_ = s.rateLimited(ctx, st, wait)
			return err
		}
		if errors.Is(err, stores.ErrUnauthorized) {
			return s.revoke(ctx, st, "revoked", s.now())
		}
		return err
	}
	if st.SyncProducts {
		seen := []string{}
		cursor := ""
		for {
			page, err := c.Products(ctx, cursor, stores.Range{})
			if err != nil {
				return fail(err)
			}
			if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				q := store.New(tx)
				for _, p := range page.Items {
					seen = append(seen, p.ID)
					if _, err := s.applyProduct(ctx, q, st, p, false); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return err
			}
			if page.Next == "" {
				break
			}
			cursor = page.Next
		}
		checked = len(seen)
		if _, err := s.Q.CommerceProductsRemoveMissing(ctx, store.CommerceProductsRemoveMissingParams{StoreID: st.ID, At: s.now(), Seen: seen}); err != nil {
			return err
		}
	}
	if st.SyncOrders {
		n, err := s.pullOrders(ctx, c, st, r, sourceReconcile)
		if err != nil {
			return fail(err)
		}
		missed = n
	}
	if st.SyncCustomers && st.Platform == Shopify {
		if err := s.pullCustomers(ctx, c, st, r, sourceReconcile); err != nil {
			return fail(err)
		}
	}
	if st.SyncCheckouts {
		if err := s.pullCheckouts(ctx, c, st, r, sourceReconcile); err != nil {
			return fail(err)
		}
	}
	return s.reconciled(ctx, st, by, missed, checked)
}

// reconciled records a reconciliation: on the store, and in its activity.
func (s *Service) reconciled(ctx context.Context, st store.Store, by *uuid.UUID, missed, checked int) error {
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommerceStoreReconciled(ctx, store.CommerceStoreReconciledParams{ID: st.ID, At: now, Missed: int32(missed), Products: int32(checked)}); err != nil {
			return err
		}
		return s.activity(ctx, q, st, activity{Kind: "reconciliation", Tone: "ok", Missed: missed, Products: checked, By: s.userName(ctx, by)})
	})
}

// pullOrders applies the orders a range lists; it returns how many weren't
// stored yet.
func (s *Service) pullOrders(ctx context.Context, c stores.Client, st store.Store, r stores.Range, source string) (int, error) {
	n := 0
	emitted := false
	cursor := ""
	for {
		page, err := c.Orders(ctx, cursor, r)
		if err != nil {
			return n, err
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, o := range page.Items {
				res, err := s.applyOrder(ctx, q, st, o, source, "")
				if err != nil {
					return err
				}
				if res.newOrder {
					n++
				}
				emitted = emitted || res.emitted
			}
			return nil
		}); err != nil {
			return n, err
		}
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}
	if emitted && s.Kick != nil {
		s.Kick(ctx)
	}
	return n, nil
}

func (s *Service) pullCustomers(ctx context.Context, c stores.Client, st store.Store, r stores.Range, source string) error {
	cursor := ""
	for {
		page, err := c.Customers(ctx, cursor, r)
		if errors.Is(err, stores.ErrUnsupported) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, cu := range page.Items {
				if _, err := s.applyCustomer(ctx, q, st, cu, source); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if page.Next == "" {
			return nil
		}
		cursor = page.Next
	}
}

func (s *Service) pullCheckouts(ctx context.Context, c stores.Client, st store.Store, r stores.Range, source string) error {
	cursor := ""
	for {
		page, err := c.Checkouts(ctx, cursor, r)
		if errors.Is(err, stores.ErrUnsupported) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, ch := range page.Items {
				if _, err := s.applyCheckout(ctx, q, st, ch, source); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if page.Next == "" {
			return nil
		}
		cursor = page.Next
	}
}

// topicResource is what a webhook topic is about, to poll while it's missing.
func topicResource(topic string) string {
	switch {
	case slices.Contains([]string{"orders/create", "orders/paid", "orders/updated", "refunds/create", "order.created", "order.updated", "order.deleted"}, topic):
		return "orders"
	case slices.Contains([]string{"checkouts/create", "checkouts/update"}, topic):
		return "checkouts"
	case slices.Contains([]string{"customers/create", "customers/update"}, topic):
		return "customers"
	case slices.Contains([]string{"products/create", "products/update", "products/delete", "product.created", "product.updated", "product.deleted"}, topic):
		return "products"
	}
	return ""
}
