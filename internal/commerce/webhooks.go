package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// MaxWebhookBody is the largest delivery accepted.
const MaxWebhookBody = 2 << 20

// lifecyclePosition orders webhooks that aren't data webhooks (Shopify's
// app/uninstalled) after the table's, which leaves them out.
const lifecyclePosition = 100

// topicsOf are the webhooks a store registers: its platform's data topics,
// then Shopify's lifecycle topic.
func topicsOf(platform string) []string {
	t := slices.Clone(descriptor(platform).Topics)
	if platform == Shopify {
		t = append(t, shopify.LifecycleTopic)
	}
	return t
}

// registerWebhooks registers the store's webhooks that aren't there (all
// of them, or only those not registered when onlyMissing). A refused key
// stops it; a webhook the store won't take is marked failed.
func (s *Service) registerWebhooks(ctx context.Context, st store.Store, by *uuid.UUID, onlyMissing bool) error {
	c, err := s.client(st)
	if err != nil {
		return err
	}
	existing, err := c.Webhooks(ctx)
	if err != nil {
		return err
	}
	rows, err := s.Q.CommerceWebhooks(ctx, st.ID)
	if err != nil {
		return err
	}
	known := map[string]store.StoreWebhook{}
	for _, r := range rows {
		known[r.Topic] = r
	}
	addr := s.WebhookURL(st)
	secret := ""
	if st.Platform == WooCommerce {
		if secret, err = s.webhookSecret(st); err != nil {
			return err
		}
	}
	now := s.now()
	for i, topic := range topicsOf(st.Platform) {
		pos := int32(i)
		if topic == shopify.LifecycleTopic {
			pos = lifecyclePosition
		}
		ix := slices.IndexFunc(existing, func(w stores.Webhook) bool { return w.Topic == topic && w.Address == addr })
		if onlyMissing && known[topic].Status == "registered" && ix >= 0 {
			continue
		}
		if ix >= 0 {
			if err := s.Q.CommerceWebhookUpsert(ctx, store.CommerceWebhookUpsertParams{StoreID: st.ID, Topic: topic, ExternalID: existing[ix].ID, Status: "registered",
				RegisteredAt: &now, RegisteredBy: by, Position: pos}); err != nil {
				return err
			}
			continue
		}
		w, err := c.Subscribe(ctx, topic, addr, secret)
		if errors.Is(err, stores.ErrUnauthorized) || errors.Is(err, stores.ErrPermission) {
			return err
		}
		if err != nil {
			if err := s.Q.CommerceWebhookUpsert(ctx, store.CommerceWebhookUpsertParams{StoreID: st.ID, Topic: topic, Status: "failed",
				Error: err.Error(), Position: pos}); err != nil {
				return err
			}
			continue
		}
		if err := s.Q.CommerceWebhookUpsert(ctx, store.CommerceWebhookUpsertParams{StoreID: st.ID, Topic: topic, ExternalID: w.ID, Status: "registered",
			RegisteredAt: &now, RegisteredBy: by, Position: pos}); err != nil {
			return err
		}
	}
	return nil
}

// Routes mounts the stores' webhooks: POST /hooks/{platform}/{token}.
// Static paths win over the sending providers' /hooks/{adapter}/{token}.
func (s *Service) Routes(r chi.Router) {
	r.Post("/hooks/shopify/{token}", s.serveWebhook(Shopify))
	r.Post("/hooks/woocommerce/{token}", s.serveWebhook(WooCommerce))
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body) //nolint:gosec // fixed JSON answers, never request data
}

// keptHeaders are the headers parsing needs later.
var keptHeaders = []string{
	"X-Shopify-Topic", "X-Shopify-Event-Id", "X-Shopify-Webhook-Id", "X-Shopify-Triggered-At", "X-Shopify-Shop-Domain", "X-Shopify-API-Version",
	"X-WC-Webhook-Topic", "X-WC-Webhook-Delivery-ID", "X-WC-Webhook-ID", "X-WC-Webhook-Resource", "X-WC-Webhook-Event",
}

// serveWebhook verifies, stores and acknowledges one delivery; a River job
// applies it. A retried delivery is stored once and answered 200; a bad
// signature is kept for diagnosis, never applied, and answered 401.
func (s *Service) serveWebhook(platform string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := chi.URLParam(r, "token")
		if len(token) < 16 {
			reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
			return
		}
		st, err := s.Q.CommerceStoreByToken(ctx, &token)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && st.Platform != platform) {
			reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
			return
		}
		if err != nil {
			w.Header().Set("Retry-After", "30")
			reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxWebhookBody+1))
		if err != nil || len(body) > MaxWebhookBody {
			reply(w, http.StatusRequestEntityTooLarge, `{"error":"payload too large"}`)
			return
		}
		d := descriptor(platform)
		topic, delivery := d.Webhooks.Topic(r.Header)
		// WooCommerce checks a new webhook's address with a ping.
		if platform == WooCommerce && bytes.HasPrefix(bytes.TrimSpace(body), []byte("webhook_id=")) {
			reply(w, http.StatusOK, `{"ok":true}`)
			return
		}
		now := s.now()
		secret, err := s.webhookSecret(st)
		if err != nil || d.Webhooks.Verify(r.Header, body, secret) != nil {
			s.storeRejected(ctx, st, topic, body, now)
			reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
			return
		}
		h := map[string]string{}
		for _, k := range keptHeaders {
			if v := r.Header.Get(k); v != "" {
				h[k] = v
			}
		}
		var triggered *time.Time
		if v := r.Header.Get("X-Shopify-Triggered-At"); v != "" {
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				triggered = &t
			}
		}
		dup, err := s.storeEvent(ctx, st, topic, delivery, triggered, h, body, now)
		if err != nil {
			s.log().ErrorContext(ctx, "store webhook", "store", st.ID, "err", err)
			w.Header().Set("Retry-After", "30")
			reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
			return
		}
		if dup {
			reply(w, http.StatusOK, `{"ok":true,"duplicate":true}`)
			return
		}
		reply(w, http.StatusOK, `{"ok":true}`)
	}
}

func (s *Service) storeRejected(ctx context.Context, st store.Store, topic string, body []byte, now time.Time) {
	if len(body) > 2048 {
		body = body[:2048]
	}
	_, err := s.Q.CommerceEventInsert(ctx, store.CommerceEventInsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, Topic: topic,
		DeliveryID: "", ReceivedAt: now, Headers: json.RawMessage(`{}`), Body: body, SignatureOk: false, Status: "rejected"})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.log().WarnContext(ctx, "keep rejected store webhook", "store", st.ID, "err", err)
	}
}

// storeEvent keeps a verified delivery and queues its application, in one
// transaction. dup reports a delivery stored before (a retry).
func (s *Service) storeEvent(ctx context.Context, st store.Store, topic, delivery string, triggered *time.Time, h map[string]string, body []byte, now time.Time) (bool, error) {
	dup := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		status := "pending"
		if st.Status == StatusDisconnected && topic != shopify.LifecycleTopic {
			// A delivery after a disconnect (its webhook couldn't be removed):
			// kept, not applied.
			status = "ignored"
		}
		id, err := q.CommerceEventInsert(ctx, store.CommerceEventInsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, Topic: topic,
			DeliveryID: delivery, TriggeredAt: triggered, ReceivedAt: now, Headers: mustJSON(h), Body: body, SignatureOk: true, Status: status})
		if errors.Is(err, pgx.ErrNoRows) {
			dup = true
			return nil
		}
		if err != nil {
			return err
		}
		if err := q.CommerceStoreTouchEvent(ctx, store.CommerceStoreTouchEventParams{ID: st.ID, At: now}); err != nil {
			return err
		}
		if err := q.CommerceWebhookEvent(ctx, store.CommerceWebhookEventParams{StoreID: st.ID, Topic: topic, At: now}); err != nil {
			return err
		}
		if status != "pending" {
			return nil
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ApplyArgs{EventID: id.String()}, nil); err != nil {
				return err
			}
		}
		return nil
	})
	return dup, err
}

// ApplyArgs applies one stored delivery.
type ApplyArgs struct {
	EventID string `json:"event_id"`
}

// Kind implements river.JobArgs.
func (ApplyArgs) Kind() string { return "commerce.apply" }

// InsertOpts runs on the events queue.
func (ApplyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 10}
}

// ApplyWorker applies stored deliveries.
type ApplyWorker struct {
	river.WorkerDefaults[ApplyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ApplyWorker) Work(ctx context.Context, job *river.Job[ApplyArgs]) error {
	id, err := uuid.Parse(job.Args.EventID)
	if err != nil {
		return river.JobCancel(err)
	}
	return w.S.Apply(ctx, id)
}

// Apply applies one stored delivery: normalized by its platform's parser,
// then applied in one transaction. A delivery that can't be parsed fails
// for good; one older than what's stored is stale.
func (s *Service) Apply(ctx context.Context, id uuid.UUID) error {
	ev, err := s.Q.CommerceEvent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if ev.Status != "pending" {
		return nil
	}
	st, err := s.Q.CommerceStoreByID(ctx, ev.StoreID)
	if err != nil {
		return err
	}
	var hm map[string]string
	_ = json.Unmarshal(ev.Headers, &hm)
	h := http.Header{}
	for k, v := range hm {
		h.Set(k, v)
	}
	h.Set(stores.CurrencyHeader, st.Currency)
	parsed, err := descriptor(st.Platform).Webhooks.Parse(h, ev.Body)
	now := s.now()
	if err != nil {
		return s.Q.CommerceEventDone(ctx, store.CommerceEventDoneParams{ID: ev.ID, Status: "failed", Outcome: err.Error(), At: now})
	}
	if parsed.Kind == stores.EventUninstalled {
		if err := s.revoke(ctx, st, "uninstalled", parsed.TriggeredAt); err != nil {
			return err
		}
		return s.Q.CommerceEventDone(ctx, store.CommerceEventDoneParams{ID: ev.ID, Status: "applied", Outcome: "uninstalled", At: now})
	}
	var emitted bool
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		res, err := s.applyEvent(ctx, q, st, parsed, sourceWebhook)
		if err != nil {
			return err
		}
		emitted = res.emitted
		return q.CommerceEventDone(ctx, store.CommerceEventDoneParams{ID: ev.ID, Status: res.status, Outcome: res.outcome, At: now})
	})
	if err != nil {
		return err
	}
	if emitted && s.Kick != nil {
		s.Kick(ctx)
	}
	return nil
}

// Sources of data (store_orders.source).
const (
	sourceImport    = "import"
	sourceWebhook   = "webhook"
	sourcePoll      = "poll"
	sourceReconcile = "reconcile"
	sourcePlugin    = "plugin"
)

type applied struct {
	status  string
	outcome string
	emitted bool
	// newOrder: the order wasn't stored before (reconciliations count them as missed).
	newOrder bool
}

// applyEvent applies one normalized event.
func (s *Service) applyEvent(ctx context.Context, q *store.Queries, st store.Store, ev stores.Event, source string) (applied, error) {
	switch ev.Kind {
	case stores.EventOrder:
		if !st.SyncOrders {
			return applied{status: "ignored", outcome: "orders don't sync"}, nil
		}
		return s.applyOrder(ctx, q, st, *ev.Order, source, ev.Topic)
	case stores.EventOrderDeleted:
		if err := q.CommerceOrderDelete(ctx, store.CommerceOrderDeleteParams{StoreID: st.ID, ExternalID: ev.DeletedID}); err != nil {
			return applied{}, err
		}
		return applied{status: "applied", outcome: "order deleted"}, nil
	case stores.EventRefund:
		return s.applyRefund(ctx, q, st, *ev.Refund)
	case stores.EventCustomer:
		return s.applyCustomer(ctx, q, st, *ev.Customer, source)
	case stores.EventCheckout:
		if !st.SyncCheckouts {
			return applied{status: "ignored", outcome: "checkouts don't sync"}, nil
		}
		return s.applyCheckout(ctx, q, st, *ev.Checkout, source)
	case stores.EventProduct:
		if !st.SyncProducts {
			return applied{status: "ignored", outcome: "products don't sync"}, nil
		}
		return s.applyProduct(ctx, q, st, *ev.Product, true)
	case stores.EventProductDeleted:
		n, err := q.CommerceProductRemoved(ctx, store.CommerceProductRemovedParams{StoreID: st.ID, ExternalID: ev.DeletedID, At: s.now()})
		if err != nil {
			return applied{}, err
		}
		if n == 0 {
			return applied{status: "ignored", outcome: "unknown product"}, nil
		}
		return applied{status: "applied", outcome: "product removed"}, nil
	case stores.EventPing:
		return applied{status: "ignored", outcome: "ping"}, nil
	}
	return applied{status: "ignored", outcome: "unknown event"}, nil
}
