package leads

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/store"
)

// Meta's webhook for a connection is /hooks/meta-leads/{token}: a GET
// handshake when the app subscribes (hub.verify_token must be the
// connection's), then POSTs signed with the app secret. Like every other
// inbound hook, the token is 256 random bits, kept sealed for the callback
// URL next to its SHA-256, which a request is checked against before
// anything else. Static paths win over the sending providers'
// /hooks/{adapter}/{token}.

// Routes mounts Meta's lead ads webhook.
func (s *Service) Routes(r chi.Router) {
	r.Get("/hooks/meta-leads/{token}", s.serveVerify)
	r.Post("/hooks/meta-leads/{token}", s.serveDelivery)
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body) //nolint:gosec // fixed JSON answers, never request data
}

// hookConnection is the connection a callback URL's token belongs to;
// ok is false for a token that isn't one (or no longer is: disconnected).
func (s *Service) hookConnection(ctx context.Context, token string) (store.LeadConnection, bool, error) {
	if len(token) < 32 {
		return store.LeadConnection{}, false, nil
	}
	c, err := s.Q.LeadConnectionByHash(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, false, nil
	}
	if err != nil {
		return c, false, err
	}
	// The lookup is by hash; compare again in constant time.
	if !authn.EqualHash(c.CallbackTokenHash, authn.HashToken(token)) || c.Status == StatusDisconnected {
		return c, false, nil
	}
	return c, true, nil
}

// serveVerify answers Meta's verification request: hub.challenge when
// hub.verify_token is the connection's.
func (s *Service) serveVerify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, ok, err := s.hookConnection(ctx, chi.URLParam(r, "token"))
	if err != nil {
		w.Header().Set("Retry-After", "30")
		reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
		return
	}
	if !ok {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	vt, err := s.Box.OpenString(c.VerifyTokenSealed, verifyAD(c.ID))
	if err != nil {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	challenge, ok := meta.VerifyChallenge(r.URL.Query(), vt)
	if !ok {
		reply(w, http.StatusForbidden, `{"error":"verification failed"}`)
		return
	}
	if err := s.Q.LeadConnectionVerified(ctx, store.LeadConnectionVerifiedParams{ID: c.ID, At: s.now()}); err != nil {
		s.log().WarnContext(ctx, "lead ads: keep the verification", "connection", c.ID, "err", err)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, challenge) //nolint:gosec // Meta's challenge, at most 200 bytes, as text/plain with nosniff
}

// serveDelivery verifies, stores and acknowledges one delivery; a River
// job applies it. A bad signature is kept (2 KB) for diagnosis, never
// applied, and answered 401; a database that can't take it answers 429 so
// Meta retries.
func (s *Service) serveDelivery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, ok, err := s.hookConnection(ctx, chi.URLParam(r, "token"))
	if err != nil {
		w.Header().Set("Retry-After", "30")
		reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
		return
	}
	if !ok {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil || len(body) > MaxBody {
		reply(w, http.StatusRequestEntityTooLarge, `{"error":"payload too large"}`)
		return
	}
	now := s.now()
	sig := r.Header.Get("X-Hub-Signature-256")
	creds, err := s.credentials(c)
	if err != nil || !meta.SignatureShape(sig) || !meta.VerifySignature(creds.AppSecret, body, sig) {
		kept := body
		if len(kept) > MaxRejected {
			kept = kept[:MaxRejected]
		}
		if _, err := s.Q.LeadEventInsert(ctx, store.LeadEventInsertParams{ID: newID(), WorkspaceID: c.WorkspaceID, ConnectionID: c.ID, ReceivedAt: now,
			Body: kept, SignatureOk: false, Status: "rejected"}); err != nil {
			s.log().WarnContext(ctx, "lead ads: keep a refused delivery", "connection", c.ID, "err", err)
		}
		reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
		return
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id, err := q.LeadEventInsert(ctx, store.LeadEventInsertParams{ID: newID(), WorkspaceID: c.WorkspaceID, ConnectionID: c.ID, ReceivedAt: now,
			Body: body, SignatureOk: true, Status: "pending"})
		if err != nil {
			return err
		}
		if err := q.LeadConnectionTouchEvent(ctx, store.LeadConnectionTouchEventParams{ID: c.ID, At: now}); err != nil {
			return err
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ApplyArgs{EventID: id.String()}, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.log().ErrorContext(ctx, "lead ads: store a delivery", "connection", c.ID, "err", err)
		w.Header().Set("Retry-After", "30")
		reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
		return
	}
	reply(w, http.StatusOK, `{"ok":true}`)
}

// ApplyArgs applies one stored delivery.
type ApplyArgs struct {
	EventID string `json:"event_id"`
}

// Kind implements river.JobArgs.
func (ApplyArgs) Kind() string { return "leads.apply" }

// InsertOpts runs on the events queue.
func (ApplyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 10}
}
