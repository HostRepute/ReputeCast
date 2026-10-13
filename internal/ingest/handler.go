// Package ingest receives provider events: the webhook endpoint
// (POST /hooks/{adapter}/{token}) verifies the signature, stores the raw
// payload and answers immediately; a River job normalizes it into
// delivery_events, moves messages and campaign counters forward, updates
// the provider's webhook freshness and adds hard bounces and complaints to
// the shared suppression list. SMTP bounces arrive through the bounce
// mailbox poller and take the same path. The provider event log is served
// from here too.
package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxBody is the largest webhook payload accepted.
const MaxBody = 5 << 20

// Handler serves the webhook endpoint.
type Handler struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Factory *manage.Factory
	Jobs    func() *jobs.Client
	Log     *slog.Logger
	Now     func() time.Time
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// Routes mounts the endpoint.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/hooks/{adapter}/{token}", h.ServeHTTP)
	r.Options("/hooks/{adapter}/{token}", h.Validate)
}

// Validate answers the CloudEvents webhook validation handshake (an OPTIONS
// request with WebHook-Request-Origin, as Azure Event Grid sends before it
// delivers CloudEvents): only a known webhook URL is confirmed.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("WebHook-Request-Origin")
	adapter, token := chi.URLParam(r, "adapter"), chi.URLParam(r, "token")
	if !validOrigin(origin) || len(token) < 16 || adapter == "" {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	if _, err := h.Q.ProvGetByWebhookToken(r.Context(), store.ProvGetByWebhookTokenParams{WebhookToken: token, Adapter: adapter}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
			return
		}
		retryLater(w)
		return
	}
	w.Header().Set("WebHook-Allowed-Origin", origin)
	w.Header().Set("WebHook-Allowed-Rate", "*")
	w.Header().Set("Allow", "POST, OPTIONS")
	reply(w, http.StatusOK, `{"ok":true}`)
}

// validOrigin accepts a host name as WebHook-Request-Origin.
func validOrigin(o string) bool {
	if o == "" || len(o) > 253 {
		return false
	}
	for _, c := range o {
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		digit := c >= '0' && c <= '9'
		if !letter && !digit && c != '.' && c != '-' {
			return false
		}
	}
	return true
}

// retryLater answers a transient failure with 429 and Retry-After rather
// than 503: providers retry 429, and some (Brevo) drop events on a 5xx.
func retryLater(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "30")
	reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	// The body is one of the fixed JSON answers above or an adapter's
	// handshake answer, sent as JSON that browsers won't sniff as HTML.
	_, _ = io.WriteString(w, body) //nolint:gosec // see above: never request data rendered as HTML
}

// ServeHTTP verifies, stores and acknowledges one webhook request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adapter, token := chi.URLParam(r, "adapter"), chi.URLParam(r, "token")
	if len(token) < 16 || adapter == "" {
		reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
		return
	}
	p, err := h.Q.ProvGetByWebhookToken(ctx, store.ProvGetByWebhookTokenParams{WebhookToken: token, Adapter: adapter})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			reply(w, http.StatusNotFound, `{"error":"unknown webhook"}`)
			return
		}
		h.Log.ErrorContext(ctx, "webhook lookup", "err", err)
		retryLater(w)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBody+1))
	if err != nil {
		reply(w, http.StatusBadRequest, `{"error":"unreadable body"}`)
		return
	}
	if len(body) > MaxBody {
		reply(w, http.StatusRequestEntityTooLarge, `{"error":"payload too large"}`)
		return
	}
	// The endpoint self-test only proves the URL is reachable from outside.
	if v := r.Header.Get(manage.SelfTestHeader); v != "" {
		if manage.VerifySelfTest(p.WebhookToken, v, h.now()) {
			reply(w, http.StatusOK, `{"ok":true,"selfTest":true}`)
		} else {
			reply(w, http.StatusUnauthorized, `{"error":"bad self-test signature"}`)
		}
		return
	}
	prov, err := h.Factory.Open(ctx, p)
	if err != nil {
		h.Log.ErrorContext(ctx, "webhook adapter", "provider", p.ID, "err", err)
		retryLater(w)
		return
	}
	if hs, ok := prov.(providers.WebhookHandshaker); ok {
		handled, resp, err := hs.Handshake(ctx, r, body)
		if handled {
			switch {
			case errors.Is(err, providers.ErrBadSignature):
				reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
			case err != nil:
				h.Log.WarnContext(ctx, "webhook handshake", "provider", p.ID, "err", err)
				reply(w, http.StatusBadGateway, `{"error":"handshake failed"}`)
			default:
				if len(resp) == 0 {
					resp = []byte(`{"ok":true}`)
				}
				reply(w, http.StatusOK, string(resp))
			}
			return
		}
	}
	var verr error
	if v, ok := prov.(providers.WebhookVerifier); ok {
		verr = v.VerifyWebhook(ctx, r, body)
	} else {
		_, verr = prov.ParseWebhook(ctx, r, body)
	}
	if verr != nil && !errors.Is(verr, providers.ErrBadSignature) && errors.Is(verr, providers.ErrUnavailable) {
		// The signature couldn't be checked (a community adapter's process
		// isn't answering): ask the provider to retry instead of storing an
		// unverified payload as verified.
		h.Log.WarnContext(ctx, "webhook verification unavailable", "provider", p.ID, "err", verr)
		reply(w, http.StatusServiceUnavailable, `{"error":"try again"}`)
		return
	}
	sigOK := !errors.Is(verr, providers.ErrBadSignature)
	var secretHeaders []string
	if s, ok := prov.(providers.WebhookSecrets); ok {
		secretHeaders = s.WebhookSecretHeaders()
	}
	stored := body
	if !sigOK && len(stored) > 2048 {
		stored = stored[:2048] // unverified payloads are kept only for diagnosis
	}
	id := newID()
	err = db.Tx(ctx, h.Pool, func(tx pgx.Tx) error {
		if err := store.New(tx).IngestInsertRaw(ctx, store.IngestInsertRawParams{
			ID: id, WorkspaceID: p.WorkspaceID, ProviderID: p.ID, ReceivedAt: h.now(), ContentType: r.Header.Get("Content-Type"),
			Body: stored, Headers: keepHeaders(r.Header, secretHeaders...), SignatureOk: sigOK, Source: "webhook",
		}); err != nil {
			return err
		}
		if sigOK && h.Jobs != nil {
			if c := h.Jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, NormalizeArgs{RawEventID: id}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "store webhook", "provider", p.ID, "err", err)
		retryLater(w)
		return
	}
	if !sigOK {
		reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
		return
	}
	reply(w, http.StatusOK, `{"ok":true}`)
}

// keepHeaders stores the request headers parsing may need, never
// credentials: neither the standard ones nor the adapter's secret headers
// named in drop.
func keepHeaders(h http.Header, drop ...string) json.RawMessage {
	out := map[string]string{}
	size := 0
	for k, v := range h {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "cookie" || lk == "proxy-authorization" || len(v) == 0 {
			continue
		}
		if slices.ContainsFunc(drop, func(d string) bool { return strings.EqualFold(d, k) }) {
			continue
		}
		val := v[0]
		if size += len(k) + len(val); size > 16<<10 {
			break
		}
		out[k] = val
	}
	b, _ := json.Marshal(out)
	return b
}

// rebuild recreates the request ParseWebhook reads from stored headers.
func rebuild(raw store.ProviderRawEvent) *http.Request {
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://ingest.internal/", strings.NewReader(string(raw.Body)))
	var hs map[string]string
	_ = json.Unmarshal(raw.Headers, &hs)
	for k, v := range hs {
		r.Header.Set(k, v)
	}
	if raw.ContentType != "" {
		r.Header.Set("Content-Type", raw.ContentType)
	}
	return r
}
