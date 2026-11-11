package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Back in stock and price drops (M62, G-30). Storing a product reads what
// it was before (CommerceProductUpsert): a product that comes back, or
// whose price falls, queues commerce.product_changed in the same
// transaction, when a live flow of the workspace starts on it. The job
// finds the contacts of each such flow on the product's store, a page at a
// time, and queues one automation event each (automations/storeflows.go):
//
//   - back in stock: the contacts waiting for it (store_stock_requests,
//     from the site tracker's "notifyWhenAvailable" or the API), and with
//     "Also people who viewed it" those who viewed it in the last N days;
//     the requests are answered (notified_at);
//   - price drop: those who viewed it or added it to their cart in the last
//     N days, when the price fell by the trigger's share or more.
//
// Contacts who aren't active are skipped, and a flow takes the same
// contact again only after 7 days (tracker_trigger_entries).

// ProductChangedArgs is one product's change.
type ProductChangedArgs struct {
	ProductID string `json:"product_id"`
	// Change is automations.TriggerBackInStock or TriggerPriceDrop.
	Change string `json:"kind"`
	// DropPercent is how much a price fell, in whole percent; OldPrice the
	// price before, in minor units.
	DropPercent int   `json:"drop_percent,omitempty"`
	OldPrice    int64 `json:"old_price,omitempty"`
}

// Kind implements river.JobArgs.
func (ProductChangedArgs) Kind() string { return "commerce.product_changed" }

// InsertOpts: the events queue; the same change of a product once.
func (ProductChangedArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 10 * time.Minute}}
}

// ProductChangedWorker runs the stock flows for one change.
type ProductChangedWorker struct {
	river.WorkerDefaults[ProductChangedArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ProductChangedWorker) Work(ctx context.Context, job *river.Job[ProductChangedArgs]) error {
	if _, err := uuid.Parse(job.Args.ProductID); err != nil {
		return river.JobCancel(err)
	}
	_, err := w.S.ProductChanged(ctx, job.Args)
	return err
}

// stockPage is how many contacts one transaction takes.
const stockPage = 500

// productChanged queues commerce.product_changed in the product's
// transaction when the product (stored with status and currency) came back
// in stock or got cheaper, and a live flow of the workspace starts on that.
func (s *Service) productChanged(ctx context.Context, tx pgx.Tx, q *store.Queries, st store.Store, row store.CommerceProductUpsertRow, p stores.Product, status, currency string) error {
	if !row.Existed || status != "active" || tx == nil {
		return nil
	}
	jc := s.jobs()
	if jc == nil {
		return nil
	}
	var changes []ProductChangedArgs
	if !row.OldAvailable && p.Available {
		changes = append(changes, ProductChangedArgs{ProductID: row.ID.String(), Change: automations.TriggerBackInStock})
	}
	if row.OldPriceCents > 0 && p.Price >= 0 && p.Price < row.OldPriceCents && strings.EqualFold(row.OldCurrency, currency) {
		if drop := int((row.OldPriceCents - p.Price) * 100 / row.OldPriceCents); drop >= 1 {
			changes = append(changes, ProductChangedArgs{ProductID: row.ID.String(), Change: automations.TriggerPriceDrop, DropPercent: drop, OldPrice: row.OldPriceCents})
		}
	}
	for _, c := range changes {
		listens, err := q.CommerceFlowListens(ctx, store.CommerceFlowListensParams{WorkspaceID: st.WorkspaceID, Kind: c.Change})
		if err != nil {
			return err
		}
		if !listens {
			continue
		}
		if _, err := jc.InsertTx(ctx, tx, c, nil); err != nil {
			return err
		}
	}
	return nil
}

// ProductChanged runs the stock flows for one change: it returns how many
// contacts entered.
func (s *Service) ProductChanged(ctx context.Context, ch ProductChangedArgs) (int, error) {
	id, err := uuid.Parse(ch.ProductID)
	if err != nil {
		return 0, err
	}
	prods, err := s.Q.CommerceProductsByIDs(ctx, []uuid.UUID{id})
	if err != nil || len(prods) == 0 {
		return 0, err
	}
	p := prods[0]
	if p.RemovedAt != nil || p.Status != "active" || (ch.Change == automations.TriggerBackInStock && !p.Available) {
		return 0, nil
	}
	st, err := s.Q.CommerceStoreByID(ctx, p.StoreID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && st.Status != StatusConnected) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	flows, err := s.Q.CommerceStockFlows(ctx, store.CommerceStockFlowsParams{WorkspaceID: st.WorkspaceID, Kind: ch.Change, Store: st.ID.String()})
	if err != nil {
		return 0, err
	}
	started := s.now()
	entered := 0
	for _, f := range flows {
		var t automations.AutomationTrigger
		if json.Unmarshal(f.Trigger, &t) != nil {
			continue
		}
		if ch.Change == automations.TriggerPriceDrop && ch.DropPercent < t.Stock.DropPercent() {
			continue
		}
		n, err := s.stockFlow(ctx, st, p, f.ID, t.Stock, ch, started)
		entered += n
		if err != nil {
			return entered, err
		}
	}
	if entered > 0 && s.Kick != nil {
		s.Kick(ctx)
	}
	return entered, nil
}

// stockFlow takes one flow's contacts for a product change, a page per
// transaction.
func (s *Service) stockFlow(ctx context.Context, st store.Store, p store.StoreProduct, flow uuid.UUID, t *automations.AutomationStockTrigger, ch ProductChangedArgs, started time.Time) (int, error) {
	back := ch.Change == automations.TriggerBackInStock
	viewers := !back || t.IncludeViewers
	since := started.AddDate(0, 0, -t.LookbackDays())
	reentry := started.AddDate(0, 0, -automations.ReentryDays)
	day := started.UTC().Format(time.DateOnly)
	ref := "stock:" + p.ID.String() + ":" + day
	product := map[string]any{"title": p.Title, "url": p.Url, "image": p.ImageUrl, "price": Money(p.PriceCents, p.Currency, false)}
	liquid := map[string]any{"product": product}
	if !back {
		liquid["previous_price"] = Money(ch.OldPrice, p.Currency, false)
		liquid["drop_percent"] = ch.DropPercent
	}
	payload := mustJSON(map[string]any{"automation": flow.String(), "store": st.ID.String(), "product": p.ID.String(), "kind": ch.Change,
		"drop_percent": ch.DropPercent, "ctx": map[string]string{"store": st.ID.String(), "products": p.ID.String()},
		"liquid": map[string]any{"stock": liquid}})
	entered := 0
	after := uuid.Nil
	for {
		rows, err := s.Q.CommerceStockCandidates(ctx, store.CommerceStockCandidatesParams{AutomationID: flow, Kind: ch.Change, ReentrySince: reentry, Lim: stockPage,
			Requests: back, ProductID: p.ID, After: after, Started: started, Viewers: viewers, Carters: !back, Since: since})
		if err != nil {
			return entered, err
		}
		if len(rows) == 0 {
			return entered, nil
		}
		after = rows[len(rows)-1].ContactID
		var who []uuid.UUID
		for _, r := range rows {
			if r.Eligible {
				who = append(who, r.ContactID)
			}
		}
		if len(who) > 0 {
			err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				q := store.New(tx)
				in, err := q.TrackerFlowsEnter(ctx, store.TrackerFlowsEnterParams{AutomationID: flow, Kind: ch.Change, At: started, Contacts: who, ReentrySince: reentry})
				if err != nil || len(in) == 0 {
					return err
				}
				refs := make([]string, len(in))
				payloads := make([]json.RawMessage, len(in))
				for i := range in {
					refs[i], payloads[i] = ref, payload
				}
				if err := q.TrackerFlowsEmit(ctx, store.TrackerFlowsEmitParams{WorkspaceID: st.WorkspaceID, Kind: ch.Change, Refs: refs, Payloads: payloads, At: started, Contacts: in}); err != nil {
					return err
				}
				if back {
					if err := q.CommerceStockNotified(ctx, store.CommerceStockNotifiedParams{At: started, ProductID: p.ID, Contacts: in}); err != nil {
						return err
					}
				}
				entered += len(in)
				return nil
			})
			if err != nil {
				return entered, err
			}
		}
		if len(rows) < stockPage {
			return entered, nil
		}
	}
}

// CommerceStockRequestInput asks to email a contact when a sold-out
// product is back.
type CommerceStockRequestInput struct {
	Email string `json:"email" format:"email" maxLength:"254" doc:"An existing contact's address"`
	// ProductID is ReputeCast's id of the product (uuid) or the store's own.
	ProductID string `json:"productId" minLength:"1" maxLength:"64"`
	// StoreID names the store when two connected stores have a product with
	// the store's own id.
	StoreID string `json:"storeId,omitempty" maxLength:"40"`
}

// CommerceStockRequest is a contact waiting for a product.
type CommerceStockRequest struct {
	ProductID string    `json:"productId" doc:"ReputeCast's id of the product"`
	StoreID   string    `json:"storeId"`
	Title     string    `json:"title" example:"Deploy Friday cap"`
	Available bool      `json:"available" doc:"The product is in stock now: back-in-stock flows start on its next return"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	mStockContactUnknown   = msgs.M("stock.contactUnknown")
	mStockProductUnknown   = msgs.M("stock.productUnknown")
	mStockProductAmbiguous = msgs.M("stock.productAmbiguous")
)

// RequestStock records that a contact wants to hear when a product is back
// ("Email me when it's back"): asking again clears an earlier answer.
// Only an existing contact of the workspace, and a product of one of its
// connected stores. Audited.
func (s *Service) RequestStock(ctx context.Context, ws *store.Workspace, in CommerceStockRequestInput) (CommerceStockRequest, error) {
	l := i18n.LangFrom(ctx)
	c, err := s.Q.CommerceContactByEmail(ctx, store.CommerceContactByEmailParams{WorkspaceID: ws.ID, Email: strings.TrimSpace(in.Email)})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && c.ErasedAt != nil) {
		return CommerceStockRequest{}, api.Err(http.StatusNotFound, api.CodeNotFound, mStockContactUnknown.Text(l, nil))
	}
	if err != nil {
		return CommerceStockRequest{}, err
	}
	var storeID *uuid.UUID
	if in.StoreID != "" {
		id, err := uuid.Parse(in.StoreID)
		if err != nil {
			return CommerceStockRequest{}, api.Invalid("body.storeId", mBlockStore.Text(l, nil))
		}
		storeID = &id
	}
	var prod store.CommerceProductInWorkspaceRow
	found := false
	if id, err := uuid.Parse(in.ProductID); err == nil {
		row, err := s.Q.CommerceProductInWorkspace(ctx, store.CommerceProductInWorkspaceParams{ID: id, WorkspaceID: ws.ID})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return CommerceStockRequest{}, err
		}
		if err == nil && (storeID == nil || row.StoreID == *storeID) {
			prod, found = row, true
		}
	}
	if !found {
		rows, err := s.Q.CommerceProductsByStoreID(ctx, store.CommerceProductsByStoreIDParams{ExternalID: strings.TrimSpace(in.ProductID), WorkspaceID: ws.ID, StoreID: storeID})
		if err != nil {
			return CommerceStockRequest{}, err
		}
		switch len(rows) {
		case 0:
			return CommerceStockRequest{}, api.Err(http.StatusNotFound, api.CodeNotFound, mStockProductUnknown.Text(l, nil))
		case 1:
			prod = store.CommerceProductInWorkspaceRow(rows[0])
		default:
			return CommerceStockRequest{}, api.Invalid("body.storeId", mStockProductAmbiguous.Text(l, nil))
		}
	}
	var out CommerceStockRequest
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		at, err := q.CommerceStockRequestUpsert(ctx, store.CommerceStockRequestUpsertParams{WorkspaceID: ws.ID, ContactID: c.ID, ProductID: prod.ID, At: s.now()})
		if err != nil {
			return err
		}
		out = CommerceStockRequest{ProductID: prod.ID.String(), StoreID: prod.StoreID.String(), Title: prod.Title, Available: prod.Available, CreatedAt: at}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store_stock_request.created", TargetType: "store_product", TargetID: prod.ID.String(),
			TargetName: prod.Title, Metadata: map[string]any{"contact": c.ID.String(), "store": prod.StoreID.String()}})
	})
	return out, err
}
