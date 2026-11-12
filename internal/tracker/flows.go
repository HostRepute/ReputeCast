package tracker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Browse abandonment and added to cart (M62, G-29; spec publish-sell
// §3.6a). Every minute, while a live flow starts on "Viewed a product and
// didn't buy" or "Added to cart and didn't check out", tracker.tick finds
// for each such flow the contacts whose last view (or add-to-cart) of the
// store's products happened AfterHours ago, in a window of an hour so a
// late or skipped minute misses no one: products in the trigger's
// collections, viewed MinViews times or more, nothing since on the store's
// products, no order and no checkout in the store since, the contact
// active and not erased, and not taken by the flow in the last 7 days
// (tracker_trigger_entries). Each one gets an automation event naming the
// flow (automations/storeflows.go), with the products they looked at in the
// day before: the store block "Viewed products" shows them, and
// {{ browse.products }} lists them (title, url, image, price).
//
// Flows is independent of the collect side (Service): it reads only
// tracker_product_views, the per-contact interest the collect writer keeps.
type Flows struct {
	Pool *pgxpool.Pool
	Q    *store.Queries
	Now  func() time.Time
	// Kick wakes the automations engine once events were queued.
	Kick func(ctx context.Context)
	Log  *slog.Logger
}

// BrowseBatch is how many contacts one transaction takes.
const BrowseBatch = 500

// browseDayBefore is how far before the window's end the products a
// contact looked at are listed.
const browseDayBefore = 24 * time.Hour

func (f *Flows) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func (f *Flows) logger() *slog.Logger {
	if f.Log != nil {
		return f.Log
	}
	return slog.Default()
}

// TickArgs runs every minute.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "tracker.tick" }

// InsertOpts: one tick at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker finds who enters the browse and cart flows.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	F *Flows
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error {
	_, err := w.F.Tick(ctx)
	return err
}

// RegisterFlowJobs adds tracker.tick, every minute.
func RegisterFlowJobs(reg *jobs.Registry, f *Flows) {
	river.AddWorker(reg.Workers, &TickWorker{F: f})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, false)
}

// Tick is one minute's work: it returns how many contacts entered a flow.
// Nothing is read unless a live flow listens.
func (f *Flows) Tick(ctx context.Context) (int, error) {
	listening, err := f.Q.TrackerFlowsListening(ctx)
	if err != nil || !listening {
		return 0, err
	}
	flows, err := f.Q.TrackerBrowseFlows(ctx)
	if err != nil {
		return 0, err
	}
	now := f.now()
	entered := 0
	var errs []error
	for _, fl := range flows {
		var t automations.AutomationTrigger
		if json.Unmarshal(fl.Trigger, &t) != nil || t.Browse == nil {
			continue
		}
		n, err := f.browseFlow(ctx, fl, t, now)
		entered += n
		if err != nil {
			// One flow's trouble doesn't hold up the others.
			f.logger().WarnContext(ctx, "tracker: browse flow", "automation", fl.ID, "err", err)
			errs = append(errs, err)
		}
	}
	if entered > 0 && f.Kick != nil {
		f.Kick(ctx)
	}
	return entered, errors.Join(errs...)
}

// browseFlow takes one flow's contacts, a page per transaction.
func (f *Flows) browseFlow(ctx context.Context, fl store.TrackerBrowseFlowsRow, t automations.AutomationTrigger, now time.Time) (int, error) {
	b := t.Browse
	end := now.Add(-time.Duration(b.Hours()) * time.Hour)
	cols := b.CollectionIDs
	if cols == nil {
		cols = []string{}
	}
	reentry := now.AddDate(0, 0, -automations.ReentryDays)
	entered := 0
	after := uuid.Nil
	for {
		rows, err := f.Q.TrackerBrowseCandidates(ctx, store.TrackerBrowseCandidatesParams{Carts: t.Kind == automations.TriggerAddedToCart, StoreID: fl.StoreID,
			Collections: cols, MinViews: int32(b.Views()), ProductsSince: end.Add(-browseDayBefore), WindowEnd: end, AutomationID: fl.ID, Kind: t.Kind,
			ReentrySince: reentry, WorkspaceID: fl.WorkspaceID, After: after, WindowStart: end.Add(-time.Hour), Lim: BrowseBatch})
		if err != nil || len(rows) == 0 {
			return entered, err
		}
		after = rows[len(rows)-1].ContactID
		n, err := f.browseEnter(ctx, fl, t.Kind, rows, now, reentry)
		entered += n
		if err != nil || len(rows) < BrowseBatch {
			return entered, err
		}
	}
}

// browseEnter takes a page's eligible contacts into the flow and queues
// their events, in one transaction.
func (f *Flows) browseEnter(ctx context.Context, fl store.TrackerBrowseFlowsRow, kind string, rows []store.TrackerBrowseCandidatesRow, now, reentry time.Time) (int, error) {
	who := make([]uuid.UUID, 0, len(rows))
	viewed := make(map[uuid.UUID][]uuid.UUID, len(rows))
	var ids []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		if !r.Eligible || len(r.Products) == 0 {
			continue
		}
		who = append(who, r.ContactID)
		viewed[r.ContactID] = r.Products
		for _, id := range r.Products {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(who) == 0 {
		return 0, nil
	}
	prods, err := f.Q.TrackerFlowsProducts(ctx, ids)
	if err != nil {
		return 0, err
	}
	byID := make(map[uuid.UUID]store.TrackerFlowsProductsRow, len(prods))
	for _, p := range prods {
		byID[p.ID] = p
	}
	day := now.UTC().Format(time.DateOnly)
	n := 0
	err = db.Tx(ctx, f.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		in, err := q.TrackerFlowsEnter(ctx, store.TrackerFlowsEnterParams{AutomationID: fl.ID, Kind: kind, At: now, Contacts: who, ReentrySince: reentry})
		if err != nil || len(in) == 0 {
			return err
		}
		refs := make([]string, len(in))
		payloads := make([]json.RawMessage, len(in))
		for i, c := range in {
			refs[i] = "browse:" + c.String() + ":" + day
			if payloads[i], err = browsePayload(fl, viewed[c], byID); err != nil {
				return err
			}
		}
		if err := q.TrackerFlowsEmit(ctx, store.TrackerFlowsEmitParams{WorkspaceID: fl.WorkspaceID, Kind: kind, Refs: refs, Payloads: payloads, At: now, Contacts: in}); err != nil {
			return err
		}
		n = len(in)
		return nil
	})
	return n, err
}

// browseProduct is one product of {{ browse.products }}.
type browseProduct struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Image string `json:"image"`
	Price string `json:"price"`
}

// browsePayload is a browse or cart event: the flow and store it's for,
// the products for the Viewed products block (the run's context) and for
// Liquid ({{ browse.products }}), newest first.
func browsePayload(fl store.TrackerBrowseFlowsRow, products []uuid.UUID, byID map[uuid.UUID]store.TrackerFlowsProductsRow) (json.RawMessage, error) {
	list := make([]browseProduct, 0, len(products))
	ctxIDs := make([]string, 0, len(products))
	for _, id := range products {
		p, ok := byID[id]
		if !ok {
			continue
		}
		ctxIDs = append(ctxIDs, id.String())
		list = append(list, browseProduct{Title: p.Title, URL: p.Url, Image: p.ImageUrl, Price: commerce.Money(p.PriceCents, p.Currency, false)})
	}
	return json.Marshal(map[string]any{
		"automation": fl.ID.String(), "store": fl.StoreID.String(),
		"ctx":    map[string]string{"store": fl.StoreID.String(), "products": strings.Join(ctxIDs, ",")},
		"liquid": map[string]any{"browse": map[string]any{"products": list}},
	})
}
