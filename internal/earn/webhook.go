package earn

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxWebhookBody is the largest payment event accepted.
const MaxWebhookBody = 1 << 20

// ApplyArgs applies one stored payment event.
type ApplyArgs struct {
	EventID string `json:"event_id"`
}

// Kind implements river.JobArgs.
func (ApplyArgs) Kind() string { return "earn.apply_event" }

// InsertOpts runs on the events queue.
func (ApplyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 10}
}

// Routes mounts the payment webhooks: POST /hooks/{provider}/{token}.
// Static paths win over the sending providers' /hooks/{adapter}/{token}.
func (s *Service) Routes(r chi.Router) {
	r.Post("/hooks/stripe/{token}", s.serveWebhook)
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body) //nolint:gosec // fixed JSON answers, never request data
}

// serveWebhook verifies, stores and acknowledges one webhook request; a
// River job applies it. A duplicate or replayed event is stored once and
// answered 200; a bad signature (or one older than five minutes) is kept
// for diagnosis, never applied, and answered 400.
func (s *Service) serveWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	if len(token) < 16 {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	c, err := s.Q.PayConnectionByToken(ctx, &token)
	if errors.Is(err, pgx.ErrNoRows) {
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
	now := s.now()
	// The reachability check only proves the address answers from outside.
	if v := r.Header.Get(manage.SelfTestHeader); v != "" {
		if manage.VerifySelfTest(token, v, now) {
			reply(w, http.StatusOK, `{"ok":true,"selfTest":true}`)
		} else {
			reply(w, http.StatusUnauthorized, `{"error":"bad self-test signature"}`)
		}
		return
	}
	ev, verr := s.verify(c, r.Header, body, now)
	if verr != nil {
		s.storeRejected(ctx, c, body, now)
		reply(w, http.StatusBadRequest, `{"error":"signature verification failed"}`)
		return
	}
	dup, err := s.store(ctx, c, ev, body, "webhook", now)
	if err != nil {
		s.log().ErrorContext(ctx, "store payment event", "connection", c.ID, "err", err)
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

func (s *Service) verify(c store.PaymentConnection, h http.Header, body []byte, now time.Time) (payments.Event, error) {
	if len(c.WebhookSecretSealed) == 0 {
		return payments.Event{}, payments.ErrBadSignature
	}
	secret, err := s.Box.Open(c.WebhookSecretSealed, secretAD(c.ID))
	if err != nil {
		return payments.Event{}, payments.ErrBadSignature
	}
	v, ok := payments.ParserFor(c.Provider)
	if !ok {
		return payments.Event{}, payments.ErrBadSignature
	}
	ev, err := v.ParseWebhook(h, body, string(secret), now)
	if err != nil {
		return payments.Event{}, err
	}
	// A test event never reaches a live connection's state, nor the reverse.
	if ev.Livemode != (c.Mode == "live") {
		return payments.Event{}, payments.ErrBadSignature
	}
	return ev, nil
}

// storeRejected keeps a payload whose signature failed, cut short, so the
// webhook card can count rejected signatures.
func (s *Service) storeRejected(ctx context.Context, c store.PaymentConnection, body []byte, now time.Time) {
	if len(body) > 2048 {
		body = body[:2048]
	}
	payload, _ := json.Marshal(map[string]string{"unverified": string(body)})
	cid := c.ID
	_, err := s.Q.PayEventInsert(ctx, store.PayEventInsertParams{ID: newID(), WorkspaceID: c.WorkspaceID, ConnectionID: &cid, Mode: c.Mode,
		ProviderEventID: "", Type: "", Source: "webhook", OccurredAt: now, ReceivedAt: now, Payload: payload, SignatureOk: false, Status: "rejected"})
	if err != nil {
		s.log().WarnContext(ctx, "store rejected payment event", "connection", c.ID, "err", err)
	}
}

// store keeps a verified event once and queues its application in the
// same transaction. dup is true when the event was already stored.
func (s *Service) store(ctx context.Context, c store.PaymentConnection, ev payments.Event, body []byte, source string, now time.Time) (dup bool, err error) {
	cid := c.ID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id, err := q.PayEventInsert(ctx, store.PayEventInsertParams{ID: newID(), WorkspaceID: c.WorkspaceID, ConnectionID: &cid, Mode: c.Mode,
			ProviderEventID: ev.ID, Type: ev.Type, Source: source, OccurredAt: ev.Created, ReceivedAt: now, Payload: body, SignatureOk: true, Status: "pending"})
		if errors.Is(err, pgx.ErrNoRows) {
			dup = true
			return nil
		}
		if err != nil {
			return err
		}
		if source == "webhook" {
			if err := q.PayConnectionTouchEvent(ctx, store.PayConnectionTouchEventParams{ID: c.ID, ReceivedAt: &now, Type: ev.Type}); err != nil {
				return err
			}
		}
		if s.Jobs != nil {
			if jc := s.Jobs(); jc != nil {
				if _, err := jc.InsertTx(ctx, tx, ApplyArgs{EventID: id.String()}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return dup, err
}

// ApplyWorker applies stored payment events.
type ApplyWorker struct {
	river.WorkerDefaults[ApplyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ApplyWorker) Work(ctx context.Context, job *river.Job[ApplyArgs]) error {
	id, err := parseUUID(job.Args.EventID)
	if err != nil {
		return river.JobCancel(err)
	}
	return w.S.Apply(ctx, id)
}
