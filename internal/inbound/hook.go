package inbound

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
)

// The inbound hook (POST /hooks/inbound/{token}): a provider posts what it
// received for a route. The request is checked at once (VerifyInbound),
// stored with a job to parse it, and answered; nothing is parsed here.

// MaxHookBody is the largest request the hook accepts: a message with
// attachments, as a provider posts it (often base64 inside JSON).
const MaxHookBody = 25 << 20

// hookSlots bounds the requests the hook reads at once, process-wide: each
// may hold up to MaxHookBody in memory.
const hookSlots = 16

// unverifiedKeep is how much of a request that failed its check is kept,
// for diagnosis only.
const unverifiedKeep = 2048

// Hook serves the inbound hook.
type Hook struct {
	S     *Service
	slots chan struct{}
}

// NewHook returns the hook of s.
func NewHook(s *Service) *Hook { return &Hook{S: s, slots: make(chan struct{}, hookSlots)} }

// Routes mounts the hook. Its static "inbound" segment wins over the
// provider events' /hooks/{adapter}/{token}.
func (h *Hook) Routes(r chi.Router) {
	r.Post("/hooks/inbound/{token}", h.ServeHTTP)
	r.Post("/hooks/inbound/{token}/*", h.ServeHTTP)
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	// The body is one of the fixed JSON answers below or an adapter's
	// handshake answer, never request data rendered as HTML.
	_, _ = io.WriteString(w, body) //nolint:gosec // see above
}

// retryLater answers a transient failure with 429 and Retry-After: the
// providers retry it (some drop mail on a 5xx).
func retryLater(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "30")
	reply(w, http.StatusTooManyRequests, `{"error":"try again"}`)
}

// ServeHTTP checks, stores and acknowledges one request.
func (h *Hook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	if len(token) < 16 || len(token) > 128 {
		reply(w, http.StatusNotFound, `{"error":"unknown route"}`)
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		retryLater(w)
		return
	}
	s := h.S
	route, err := s.Q.InboundRouteByTokenHash(ctx, authn.HashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			reply(w, http.StatusNotFound, `{"error":"unknown route"}`)
			return
		}
		s.Log.ErrorContext(ctx, "inbound hook lookup", "err", err)
		retryLater(w)
		return
	}
	if route.Kind != "provider" {
		reply(w, http.StatusNotFound, `{"error":"unknown route"}`)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxHookBody))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			reply(w, http.StatusRequestEntityTooLarge, `{"error":"message too large"}`)
			return
		}
		reply(w, http.StatusBadRequest, `{"error":"unreadable body"}`)
		return
	}
	rc, _, err := s.receiver(ctx, route)
	if err != nil {
		if errors.Is(err, errUnsupported) || errors.Is(err, errNoProvider) || errors.Is(err, pgx.ErrNoRows) {
			reply(w, http.StatusNotFound, `{"error":"unknown route"}`)
			return
		}
		s.Log.ErrorContext(ctx, "inbound hook adapter", "route", route.ID, "err", err)
		retryLater(w)
		return
	}
	ar, err := s.adapterRoute(route)
	if err != nil {
		s.Log.ErrorContext(ctx, "inbound hook route", "route", route.ID, "err", err)
		retryLater(w)
		return
	}
	if hs, ok := rc.(providers.InboundHandshaker); ok {
		handled, resp, err := hs.HandshakeInbound(ctx, ar, r, body)
		if handled {
			switch {
			case errors.Is(err, providers.ErrBadSignature):
				reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
			case err != nil:
				s.Log.WarnContext(ctx, "inbound handshake", "route", route.ID, "err", err)
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
	verr := rc.VerifyInbound(ctx, ar, r, body)
	if verr != nil && !errors.Is(verr, providers.ErrBadSignature) {
		// The check itself failed (the provider's API to read a signing
		// secret didn't answer): ask for the request again.
		s.Log.WarnContext(ctx, "inbound verification unavailable", "route", route.ID, "err", verr)
		retryLater(w)
		return
	}
	ok := verr == nil
	var secretHeaders []string
	if sh, has := rc.(providers.WebhookSecrets); has {
		secretHeaders = sh.WebhookSecretHeaders()
	}
	stored := body
	if !ok && len(stored) > unverifiedKeep {
		stored = stored[:unverifiedKeep]
	}
	headers := keepHeaders(r.Header, chi.URLParam(r, "*"), secretHeaders...)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.storeRaw(ctx, tx, route, "hook", r.Header.Get("Content-Type"), headers, stored, ok)
	})
	if err != nil {
		s.Log.ErrorContext(ctx, "store inbound", "route", route.ID, "err", err)
		retryLater(w)
		return
	}
	if !ok {
		reply(w, http.StatusUnauthorized, `{"error":"signature verification failed"}`)
		return
	}
	reply(w, http.StatusOK, `{"ok":true}`)
}

// keepHeaders stores the request headers parsing may need and the path
// after the token, never credentials: neither the standard ones nor the
// adapter's secret headers named in drop.
func keepHeaders(h http.Header, rest string, drop ...string) json.RawMessage {
	out := map[string]string{}
	size := 0
	for k, v := range h {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "cookie" || lk == "proxy-authorization" || lk == strings.ToLower(pathHeader) || len(v) == 0 {
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
	if rest = strings.Trim(rest, "/"); rest != "" && len(rest) <= 200 {
		out[pathHeader] = rest
	}
	b, _ := json.Marshal(out)
	return b
}
