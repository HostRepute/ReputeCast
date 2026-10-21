package texting

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxHookBody is the largest SMS webhook accepted.
const MaxHookBody = 1 << 20

// HookRoutes mounts the SMS webhook endpoint: receipts and inbound texts
// from every SMS provider, one address per connected provider.
func (s *Service) HookRoutes(r chi.Router) {
	r.Post("/hooks/sms/{adapter}/{token}", s.serveHook)
}

func (s *Service) serveHook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxHookBody+1))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(body) > MaxHookBody {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	status := s.Ingest(r.Context(), chi.URLParam(r, "adapter"), chi.URLParam(r, "token"), r, body)
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "30")
	}
	// Providers want an empty 200: Twilio reads a body as a reply to send.
	w.WriteHeader(status)
}

// Ingest verifies one webhook, stores it raw and queues its normalization,
// answering the status code for the provider: 200 at once, 401 when the
// signature fails (the payload is kept, cut short, for diagnosis), 404 for
// an unknown address, 429 when it should be retried.
func (s *Service) Ingest(ctx context.Context, adapter, token string, r *http.Request, body []byte) int {
	if len(token) < 16 || adapter == "" {
		return http.StatusNotFound
	}
	p, err := s.Q.SmsProviderByToken(ctx, store.SmsProviderByTokenParams{WebhookToken: token, Adapter: adapter})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return http.StatusNotFound
		}
		s.log().ErrorContext(ctx, "sms webhook lookup", "err", err)
		return http.StatusTooManyRequests
	}
	// Providers sign the address they called: the public one.
	if u, err := url.Parse(s.hookURL(p)); err == nil {
		r = r.Clone(ctx)
		r.URL, r.Host = u, u.Host
	}
	ad, err := s.open(p)
	if err != nil {
		s.log().ErrorContext(ctx, "sms webhook adapter", "provider", p.ID, "err", err)
		return http.StatusTooManyRequests
	}
	if hs, ok := ad.(sms.WebhookHandshaker); ok {
		handled, err := hs.Handshake(ctx, r, body)
		if handled {
			switch {
			case errors.Is(err, sms.ErrBadSignature):
				return http.StatusUnauthorized
			case err != nil:
				s.log().WarnContext(ctx, "sms webhook handshake", "provider", p.ID, "err", err)
				return http.StatusBadGateway
			}
			return http.StatusOK
		}
	}
	verr := ad.VerifyWebhook(ctx, r, body)
	sigOK := verr == nil
	stored := body
	if !sigOK && len(stored) > 2048 {
		stored = stored[:2048]
	}
	id := uuid.Must(uuid.NewV7())
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.SmsRawInsert(ctx, store.SmsRawInsertParams{ID: id, WorkspaceID: p.WorkspaceID, ProviderID: &p.ID, Kind: "other", ReceivedAt: s.now(),
			ContentType: r.Header.Get("Content-Type"), Headers: hookHeaders(r.Header), Body: stored, SignatureOk: sigOK}); err != nil {
			return err
		}
		if sigOK {
			if c := s.jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, NormalizeArgs{RawEventID: id}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		s.log().ErrorContext(ctx, "store sms webhook", "provider", p.ID, "err", err)
		return http.StatusTooManyRequests
	}
	if !sigOK {
		s.log().WarnContext(ctx, "sms webhook signature", "provider", p.ID, "err", verr)
		return http.StatusUnauthorized
	}
	return http.StatusOK
}

// hookHeaders keeps the headers diagnosis may need, never credentials.
func hookHeaders(h http.Header) json.RawMessage {
	out := map[string]string{}
	size := 0
	for k, v := range h {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "cookie" || lk == "proxy-authorization" || len(v) == 0 {
			continue
		}
		if size += len(k) + len(v[0]); size > 16<<10 {
			break
		}
		out[k] = v[0]
	}
	return jsonOf(out)
}
