package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// The collect writer: calls are answered as soon as their events are
// queued, and a batch of them is written every 200 ms or 500 events
// (group commit, as internal/tracking records opens and clicks), in one
// transaction that takes its locks in one order: the batch's visitors by
// (workspace, hash), the contacts' and visitors' products by (contact or
// visitor, product). Events go in with COPY. A batch that fails is tried
// again with a backoff, without holding up the calls: the queue is
// bounded, and what doesn't fit while the database is away is dropped and
// counted.

// collected is one call's events, waiting for the writer.
type collected struct {
	site  collectSite
	hash  []byte
	claim *claim
	items []item
	at    time.Time
}

// item is one event, checked.
type item struct {
	kind int16
	// notify is reputecast('notifyWhenAvailable'): a stock request, not an
	// event.
	notify          bool
	extID           string
	qty             int32
	path, ref, name string
	props           []byte
	at              time.Time
}

const (
	batchEvery = 200 * time.Millisecond
	batchMax   = 500
	// queueMax is how many calls may wait for the writer.
	queueMax = 4096
	// retryMax is how often a batch is tried before it is dropped (about
	// 25 minutes with the backoff); one failing for another reason than the
	// database being away is tried twice.
	retryMax = 50
)

type writer struct {
	s       *Service
	in      chan *collected
	batches chan []*collected
	// queued counts the calls not written yet.
	queued  atomic.Int64
	written atomic.Int64
	dropped atomic.Int64
	// refusing is set while refusals are being stored.
	refusing atomic.Bool
	lastLog  atomic.Int64
}

// WriterStats are the events the writer wrote and dropped since the
// process started.
type WriterStats struct {
	Written, Dropped int64
}

// WriterStats reports the writer's counts.
func (s *Service) WriterStats() WriterStats {
	w := s.start()
	return WriterStats{Written: w.written.Load(), Dropped: w.dropped.Load()}
}

func (s *Service) start() *writer {
	s.once.Do(func() {
		n := s.Writers
		if n <= 0 {
			n = 2
		}
		w := &writer{s: s, in: make(chan *collected, queueMax), batches: make(chan []*collected, n)}
		s.w = w
		go w.collect()
		for range n {
			go w.work()
		}
	})
	return s.w
}

// enqueue hands a call's events to the writer, never waiting: when the
// queue is full they are dropped and counted.
func (s *Service) enqueue(c *collected) {
	w := s.start()
	w.queued.Add(1)
	select {
	case w.in <- c:
	default:
		w.queued.Add(-1)
		w.drop(int64(len(c.items)), errors.New("the queue is full"))
	}
}

func (w *writer) drop(n int64, cause error) {
	w.dropped.Add(n)
	now := time.Now().Unix()
	if last := w.lastLog.Load(); now-last >= 10 && w.lastLog.CompareAndSwap(last, now) {
		w.s.log().Error("tracker events dropped", "events", n, "dropped_since_start", w.dropped.Load(), "err", cause)
	}
}

// flush waits until every call queued so far is written.
func (w *writer) flush(ctx context.Context) error {
	for w.queued.Load() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	w.s.flushRefusals(ctx)
	return nil
}

func (w *writer) collect() {
	tick := time.NewTicker(batchEvery)
	defer tick.Stop()
	var batch []*collected
	n := 0
	send := func() {
		if len(batch) > 0 {
			w.batches <- batch
			batch, n = nil, 0
		}
	}
	for {
		select {
		case c := <-w.in:
			batch = append(batch, c)
			n += max(len(c.items), 1)
			if n >= batchMax {
				send()
			}
		case <-tick.C:
			send()
			if w.refusing.CompareAndSwap(false, true) {
				go func() {
					defer w.refusing.Store(false)
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					w.s.flushRefusals(ctx)
				}()
			}
		}
	}
}

func (w *writer) work() {
	for b := range w.batches {
		w.write(b)
	}
}

func events(b []*collected) int64 {
	var n int64
	for _, c := range b {
		n += int64(len(c.items))
	}
	return n
}

// write writes a batch, trying again with a backoff while the database is
// away.
func (w *writer) write(b []*collected) {
	defer w.queued.Add(-int64(len(b)))
	n := events(b)
	delay := batchEvery
	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := w.s.writeBatch(ctx, b)
		cancel()
		if err == nil {
			w.written.Add(n)
			return
		}
		if attempt >= retryMax || (!transient(err) && attempt >= 2) {
			w.drop(n, err)
			return
		}
		// A product may have gone from the store meanwhile.
		w.s.products.drop(func(string) bool { return true })
		w.s.log().Warn("tracker events not written yet, trying again", "events", n, "attempt", attempt, "err", err)
		time.Sleep(delay)
		delay = min(delay*2, 30*time.Second)
	}
}

// transient reports an error a later try may not meet: the database away,
// busy or restarting, a deadlock, a lock timeout.
func transient(err error) bool {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code[:2] {
		case "08", "40", "53", "57", "58":
			return true
		}
		return pg.Code == "55P03"
	}
	return true
}

// productHit is a store's product id, or none.
type productHit struct {
	id    uuid.UUID
	found bool
}

const (
	productFor = 5 * time.Minute
	missFor    = time.Minute
)

func productKey(storeID uuid.UUID, ext string) string { return storeID.String() + "/" + ext }

// resolveProducts finds the linked stores' products the batch's events
// name (cached): an id the store doesn't know stays without one.
func (s *Service) resolveProducts(ctx context.Context, b []*collected) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	now := time.Now()
	missing := map[uuid.UUID][]string{}
	for _, c := range b {
		if c.site.Store == nil {
			continue
		}
		for _, it := range c.items {
			if it.extID == "" {
				continue
			}
			k := productKey(*c.site.Store, it.extID)
			if _, ok := out[k]; ok {
				continue
			}
			if h, ok := s.products.get(k, now); ok {
				if h.found {
					out[k] = h.id
				}
				continue
			}
			if !slices.Contains(missing[*c.site.Store], it.extID) {
				missing[*c.site.Store] = append(missing[*c.site.Store], it.extID)
			}
		}
	}
	for st, ids := range missing {
		rows, err := s.Q.TrackerProductsByExternal(ctx, store.TrackerProductsByExternalParams{StoreID: st, ExternalIds: ids})
		if err != nil {
			return nil, err
		}
		found := map[string]bool{}
		for _, r := range rows {
			k := productKey(st, r.ExternalID)
			out[k] = r.ID
			found[r.ExternalID] = true
			s.products.put(k, productHit{id: r.ID, found: true}, now.Add(productFor))
		}
		for _, id := range ids {
			if !found[id] {
				s.products.put(productKey(st, id), productHit{}, now.Add(missFor))
			}
		}
	}
	return out, nil
}

// visitor is a batch's visitor once upserted.
type visitor struct {
	id      uuid.UUID
	contact *uuid.UUID
}

func visitorKey(ws uuid.UUID, hash []byte) string { return string(ws[:]) + string(hash) }

// pair is two ids (a contact or visitor, then a product) as a sort key.
type pair [32]byte

func pairOf(a, b uuid.UUID) pair {
	var p pair
	copy(p[:16], a[:])
	copy(p[16:], b[:])
	return p
}

// interest is a batch's views and add-to-carts of one product by one
// contact or visitor.
type interest struct {
	ws             uuid.UUID
	views, carts   int32
	viewed, carted time.Time
}

// epoch stands for "none" in the time arrays.
var epoch = time.Unix(0, 0).UTC()

func orEpoch(t time.Time) time.Time {
	if t.IsZero() {
		return epoch
	}
	return t
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

var eventColumns = []string{"id", "workspace_id", "site_id", "visitor_id", "contact_id", "kind", "product_id", "quantity", "path", "referrer_domain", "name", "props", "occurred_at"}

// writeBatch writes one batch.
func (s *Service) writeBatch(ctx context.Context, b []*collected) error {
	products, err := s.resolveProducts(ctx, b)
	if err != nil {
		return err
	}
	touched := map[uuid.UUID]time.Time{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		vis, err := upsertVisitors(ctx, q, b)
		if err != nil {
			return err
		}
		if err := s.identify(ctx, q, b, vis); err != nil {
			return err
		}
		rows := make([][]any, 0, events(b))
		contacts, visitors, stock := map[pair]*interest{}, map[pair]*interest{}, map[pair]*interest{}
		add := func(m map[pair]*interest, k pair, ws uuid.UUID, it item) {
			x := m[k]
			if x == nil {
				x = &interest{ws: ws}
				m[k] = x
			}
			switch {
			case it.notify:
				x.viewed = later(x.viewed, it.at)
			case it.kind == KindProduct:
				x.views++
				x.viewed = later(x.viewed, it.at)
			case it.kind == KindCart:
				x.carts++
				x.carted = later(x.carted, it.at)
			}
		}
		for _, c := range b {
			v := vis[visitorKey(c.site.Workspace, c.hash)]
			for _, it := range c.items {
				var product *uuid.UUID
				if c.site.Store != nil && it.extID != "" {
					if id, ok := products[productKey(*c.site.Store, it.extID)]; ok {
						product = &id
					}
				}
				if it.notify {
					if product != nil && v.contact != nil {
						add(stock, pairOf(*v.contact, *product), c.site.Workspace, it)
					}
					continue
				}
				touched[c.site.ID] = later(touched[c.site.ID], it.at)
				var props any
				if len(it.props) > 0 {
					props = json.RawMessage(it.props)
				}
				rows = append(rows, []any{uuid.Must(uuid.NewV7()), c.site.Workspace, c.site.ID, v.id, v.contact, it.kind, product, it.qty, it.path, it.ref,
					it.name, props, it.at})
				if product == nil || (it.kind != KindProduct && it.kind != KindCart) {
					continue
				}
				if v.contact != nil {
					add(contacts, pairOf(*v.contact, *product), c.site.Workspace, it)
				} else {
					add(visitors, pairOf(v.id, *product), c.site.Workspace, it)
				}
			}
		}
		if len(rows) > 0 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"tracker_events"}, eventColumns, pgx.CopyFromRows(rows)); err != nil {
				return err
			}
		}
		if len(contacts) > 0 {
			p := store.TrackerProductViewsAddParams{}
			for _, k := range sortedPairs(contacts) {
				x := contacts[k]
				p.WorkspaceIds = append(p.WorkspaceIds, x.ws)
				p.ContactIds = append(p.ContactIds, uuid.UUID(k[:16]))
				p.ProductIds = append(p.ProductIds, uuid.UUID(k[16:]))
				p.Views, p.Carts = append(p.Views, x.views), append(p.Carts, x.carts)
				p.Viewed, p.Carted = append(p.Viewed, orEpoch(x.viewed)), append(p.Carted, orEpoch(x.carted))
			}
			if err := q.TrackerProductViewsAdd(ctx, p); err != nil {
				return err
			}
		}
		if len(visitors) > 0 {
			p := store.TrackerVisitorProductsAddParams{}
			for _, k := range sortedPairs(visitors) {
				x := visitors[k]
				p.VisitorIds = append(p.VisitorIds, uuid.UUID(k[:16]))
				p.ProductIds = append(p.ProductIds, uuid.UUID(k[16:]))
				p.Views, p.Carts = append(p.Views, x.views), append(p.Carts, x.carts)
				p.Viewed, p.Carted = append(p.Viewed, orEpoch(x.viewed)), append(p.Carted, orEpoch(x.carted))
			}
			if err := q.TrackerVisitorProductsAdd(ctx, p); err != nil {
				return err
			}
		}
		if len(stock) > 0 {
			p := store.TrackerStockRequestsAddParams{}
			for _, k := range sortedPairs(stock) {
				x := stock[k]
				p.WorkspaceIds = append(p.WorkspaceIds, x.ws)
				p.ContactIds = append(p.ContactIds, uuid.UUID(k[:16]))
				p.ProductIds = append(p.ProductIds, uuid.UUID(k[16:]))
				p.At = append(p.At, x.viewed)
			}
			if err := q.TrackerStockRequestsAdd(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.touchSites(ctx, touched)
	return nil
}

func sortedPairs(m map[pair]*interest) []pair {
	out := make([]pair, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b pair) int { return bytes.Compare(a[:], b[:]) })
	return out
}

// upsertVisitors adds the batch's new visitors and sees the others again,
// in (workspace, hash) order.
func upsertVisitors(ctx context.Context, q *store.Queries, b []*collected) (map[string]visitor, error) {
	seen := map[string]time.Time{}
	keys := make([]string, 0, len(b))
	for _, c := range b {
		k := visitorKey(c.site.Workspace, c.hash)
		if _, ok := seen[k]; !ok {
			keys = append(keys, k)
		}
		seen[k] = later(seen[k], c.at)
	}
	slices.Sort(keys)
	p := store.TrackerVisitorsUpsertParams{}
	for _, k := range keys {
		p.Ids = append(p.Ids, uuid.Must(uuid.NewV7()))
		p.WorkspaceIds = append(p.WorkspaceIds, uuid.UUID([]byte(k[:16])))
		p.Hashes = append(p.Hashes, []byte(k[16:]))
		p.Seen = append(p.Seen, seen[k])
	}
	rows, err := q.TrackerVisitorsUpsert(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make(map[string]visitor, len(rows))
	for _, r := range rows {
		out[visitorKey(r.WorkspaceID, r.VidHash)] = visitor{id: r.ID, contact: r.ContactID}
	}
	return out, nil
}

// identify links the batch's visitors whose calls say who they are: to an
// existing contact of the workspace who hasn't been erased. The visitor's
// anonymous events of the last 30 days and its product interest join the
// contact; a visitor linked to someone else before is theirs from now on,
// and what was the other's stays the other's.
func (s *Service) identify(ctx context.Context, q *store.Queries, b []*collected, vis map[string]visitor) error {
	claims := map[string]*collected{}
	for _, c := range b {
		if c.claim != nil {
			claims[visitorKey(c.site.Workspace, c.hash)] = c
		}
	}
	if len(claims) == 0 {
		return nil
	}
	keys := make([]string, 0, len(claims))
	for k := range claims {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		x, y := vis[a].id, vis[b].id
		return bytes.Compare(x[:], y[:])
	})
	now := s.now()
	for _, k := range keys {
		c := claims[k]
		v, ok := vis[k]
		if !ok {
			continue
		}
		var contact uuid.UUID
		var err error
		if c.claim.email != "" {
			contact, err = q.TrackerContactByEmail(ctx, store.TrackerContactByEmailParams{WorkspaceID: c.site.Workspace, Email: c.claim.email})
		} else {
			contact, err = q.TrackerContactLive(ctx, store.TrackerContactLiveParams{ID: c.claim.contact, WorkspaceID: c.site.Workspace})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if v.contact != nil && *v.contact == contact {
			continue
		}
		if err := q.TrackerVisitorIdentify(ctx, store.TrackerVisitorIdentifyParams{ID: v.id, ContactID: &contact, IdentifiedBy: c.claim.by, At: &now}); err != nil {
			return err
		}
		if _, err := q.TrackerBackfillEvents(ctx, store.TrackerBackfillEventsParams{ContactID: &contact, VisitorID: v.id, Since: now.AddDate(0, 0, -BackfillDays)}); err != nil {
			return err
		}
		if err := q.TrackerMergeVisitorProducts(ctx, store.TrackerMergeVisitorProductsParams{WorkspaceID: c.site.Workspace, ContactID: contact, VisitorID: v.id}); err != nil {
			return err
		}
		if err := q.TrackerDropVisitorProducts(ctx, v.id); err != nil {
			return err
		}
		v.contact = &contact
		vis[k] = v
	}
	return nil
}

// touchSites sets the sites' last event (after the batch: no lock on a
// site is held while it is written).
func (s *Service) touchSites(ctx context.Context, touched map[uuid.UUID]time.Time) {
	if len(touched) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(touched))
	for id := range touched {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	p := store.TrackerSitesTouchParams{Ids: ids}
	for _, id := range ids {
		p.At = append(p.At, touched[id])
	}
	if err := s.Q.TrackerSitesTouch(ctx, p); err != nil {
		s.log().WarnContext(ctx, "tracker sites' last event", "err", err)
	}
}
