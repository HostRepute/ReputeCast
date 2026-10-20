package recommendations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	pp "github.com/hostrepute/reputecast/internal/recommendations/partnerproto"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// The partner endpoints other instances call (docs/grow/partners.md):
//
//	GET  /partner/v1                       the versions this instance speaks
//	POST /partner/v1/preview               what a partner code connects to
//	POST /partner/v1/connect               use a partner code
//	POST /partner/v1/links/{link}/{action} signed messages about a link:
//	     subscribe, confirmed, state, profile, rotate
//
// Errors are JSON problems ({"protocol":1,"error":"code_expired",...});
// answers to signed messages are signed too.

// Routes mounts the partner endpoints on the public router.
func (s *Service) Routes(r chi.Router) {
	r.Route(pp.Prefix, func(r chi.Router) {
		r.Use(s.limitPartners)
		r.Get("/", s.serveHello)
		r.Post("/preview", s.servePreview)
		r.Post("/connect", s.serveConnect)
		r.Post("/links/{link}/{action}", s.serveLink)
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeProblem(w, http.StatusNotFound, pp.ErrBadRequest, "No such partner endpoint.", nil)
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
			writeProblem(w, http.StatusMethodNotAllowed, pp.ErrBadRequest, "Method not allowed.", nil)
		})
	})
}

// --- Rate limits ---------------------------------------------------------------------

// partnerLimiter bounds requests per address: codes are hard to guess,
// and the limit keeps it so.
type partnerLimiter struct {
	mu      sync.Mutex
	window  time.Time
	counts  map[string]int
	strict  map[string]int
	maxAll  int
	maxCode int
}

var limiter = &partnerLimiter{counts: map[string]int{}, strict: map[string]int{}, maxAll: 600, maxCode: 20}

func (l *partnerLimiter) allow(ip string, code bool, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.window) > time.Minute {
		l.window, l.counts, l.strict = now, map[string]int{}, map[string]int{}
	}
	l.counts[ip]++
	if l.counts[ip] > l.maxAll {
		return false
	}
	if code {
		l.strict[ip]++
		return l.strict[ip] <= l.maxCode
	}
	return true
}

func (s *Service) limitPartners(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := strings.HasSuffix(r.URL.Path, "/preview") || strings.HasSuffix(r.URL.Path, "/connect")
		if !limiter.allow(clientIP(r), code, time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, http.StatusTooManyRequests, pp.ErrRateLimited, "Too many requests. Try again in a minute.", nil)
			return
		}
		if v := r.Header.Get(pp.HeaderProtocol); v != "" && v != strconv.Itoa(pp.Version) {
			writeProblem(w, http.StatusBadRequest, pp.ErrUnsupportedVersion, "This instance speaks protocol version 1.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Writing answers ---------------------------------------------------------------

func writeProblem(w http.ResponseWriter, status int, code, detail string, at *time.Time) {
	body, _ := json.Marshal(pp.Problem{Protocol: pp.Version, Error: code, Detail: detail, At: at})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(pp.HeaderProtocol, strconv.Itoa(pp.Version))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeSigned answers a signed message, signed with the link's secrets.
func (s *Service) writeSigned(w http.ResponseWriter, path string, v any, secrets []string) {
	body, _ := json.Marshal(v)
	ts := s.now().Unix()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(pp.HeaderProtocol, strconv.Itoa(pp.Version))
	w.Header().Set(pp.HeaderTimestamp, strconv.FormatInt(ts, 10))
	w.Header().Set(pp.HeaderSignature, pp.SignatureHeader(secrets, ts, pp.ResponseMethod(http.StatusOK), path, body))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// canonPath is the path a signature covers: from /partner/v1 on, so an
// instance behind a proxy that adds a prefix still checks it.
func canonPath(r *http.Request) string {
	p := r.URL.EscapedPath()
	if i := strings.Index(p, pp.Prefix); i >= 0 {
		return p[i:]
	}
	return p
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, pp.MaxBody))
	if err != nil {
		writeProblem(w, http.StatusRequestEntityTooLarge, pp.ErrBadRequest, "The message is too large.", nil)
		return nil, false
	}
	return body, true
}

// --- Hello, preview, connect ---------------------------------------------------------

func (s *Service) serveHello(w http.ResponseWriter, _ *http.Request) {
	body, _ := json.Marshal(pp.Hello{Protocols: []int{pp.Version}, Instance: s.instance(), Software: "ReputeCast " + version.Version})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(pp.HeaderProtocol, strconv.Itoa(pp.Version))
	_, _ = w.Write(body)
}

// codeRow finds a partner code and says why it can't be used.
func (s *Service) codeRow(ctx context.Context, q *store.Queries, code string, lock bool) (store.PartnerCode, int, string, *time.Time) {
	norm, ok := pp.NormalizeCode(code)
	if !ok {
		return store.PartnerCode{}, http.StatusNotFound, pp.ErrCodeUnknown, nil
	}
	c, err := q.PartnerCodeByHash(ctx, pp.HashCode(norm))
	if err != nil {
		return c, http.StatusNotFound, pp.ErrCodeUnknown, nil
	}
	if lock {
		if c, err = q.PartnerCodeLock(ctx, c.ID); err != nil {
			return c, http.StatusNotFound, pp.ErrCodeUnknown, nil
		}
	}
	switch {
	case c.RevokedAt != nil:
		return c, http.StatusGone, pp.ErrCodeRevoked, c.RevokedAt
	case c.UsedAt != nil:
		return c, http.StatusConflict, pp.ErrCodeUsed, c.UsedAt
	case !s.now().Before(c.ExpiresAt):
		return c, http.StatusGone, pp.ErrCodeExpired, &c.ExpiresAt
	}
	return c, 0, "", nil
}

// partnerProfile is this workspace's profile as a code offers it (the
// code's list).
func (s *Service) partnerProfile(ctx context.Context, wsID, listID uuid.UUID) (*store.Workspace, pp.Profile, error) {
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, pp.Profile{}, err
	}
	p, err := s.ownProfile(ctx, &ws)
	if err != nil {
		return nil, pp.Profile{}, err
	}
	_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, listID).Scan(&p.List)
	return &ws, p, nil
}

func (s *Service) servePreview(w http.ResponseWriter, r *http.Request) {
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	var in pp.PreviewRequest
	if json.Unmarshal(body, &in) != nil {
		writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "The message isn't readable.", nil)
		return
	}
	c, status, code, at := s.codeRow(r.Context(), s.Q, in.Code, false)
	if status != 0 {
		writeProblem(w, status, code, "", at)
		return
	}
	_, p, err := s.partnerProfile(r.Context(), c.WorkspaceID, c.ListID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
		return
	}
	out, _ := json.Marshal(pp.PreviewResponse{Protocol: pp.Version, Profile: p, DailyLimit: int(c.DailyLimit), DoubleOptIn: true, Instance: s.instance()})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(out)
}

func (s *Service) serveConnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	var in pp.ConnectRequest
	if json.Unmarshal(body, &in) != nil || in.Key == "" || in.Link == "" || len(in.Link) > 100 {
		writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "The message isn't readable.", nil)
		return
	}
	u, err := url.Parse(strings.TrimSpace(in.Instance))
	if err != nil || u.Host == "" || (u.Scheme != "https" && (u.Scheme != "http" || !s.allowPrivate())) {
		writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "The recommending instance must answer over HTTPS.", nil)
		return
	}
	if _, ce := s.checkBase(in.Instance); ce != nil {
		writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "The recommending instance must answer at a public address.", nil)
		return
	}
	keys, err := pp.NewKeyPair()
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
		return
	}
	linkID := uuid.Must(uuid.NewV7())
	norm, ok := pp.NormalizeCode(in.Code)
	if !ok {
		writeProblem(w, http.StatusNotFound, pp.ErrCodeUnknown, "", nil)
		return
	}
	secret, err := keys.Secret(in.Key, pp.ConnectSalt(norm), in.Link, linkID.String())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "The key isn't an X25519 key.", nil)
		return
	}
	var answer pp.ConnectResponse
	var failStatus int
	var failCode string
	var failAt *time.Time
	name := strings.TrimSpace(in.Recommender.Name)
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	if name == "" {
		name = u.Host
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, status, code, at := s.codeRow(ctx, q, in.Code, true)
		if status != 0 {
			failStatus, failCode, failAt = status, code, at
			return nil
		}
		sealed, err := s.Box.SealString(secret, secretAD(linkID, "secret"))
		if err != nil {
			return err
		}
		now := s.now()
		desc := in.Recommender.Description
		if r := []rune(desc); len(r) > 300 {
			desc = string(r[:300])
		}
		if err := q.PartnerLinkInsert(ctx, store.PartnerLinkInsertParams{ID: linkID, WorkspaceID: c.WorkspaceID, Direction: DirIn, RemoteUrl: strings.TrimRight(in.Instance, "/"),
			RemoteLinkID: in.Link, ProtocolVersion: pp.Version, RemoteName: name, RemoteDescription: desc, RemoteListName: in.Recommender.List,
			DailyLimit: c.DailyLimit, ListID: &c.ListID, Label: c.Label, CodeID: &c.ID, Secret: sealed, Now: now, ConnectedBy: c.CreatedBy}); err != nil {
			return err
		}
		if err := q.PartnerCodeUse(ctx, store.PartnerCodeUseParams{ID: c.ID, Now: now, LinkID: &linkID}); err != nil {
			return err
		}
		ws, p, err := s.partnerProfile(ctx, c.WorkspaceID, c.ListID)
		if err != nil {
			return err
		}
		answer = pp.ConnectResponse{Protocol: pp.Version, Link: linkID.String(), Key: keys.Public(), Profile: p, DailyLimit: int(c.DailyLimit)}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_connected", TargetType: "partner_link", TargetID: linkID.String(),
			TargetName: name, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "Partner · " + name},
			After:    map[string]any{"direction": "recommending you", "instance": u.Host, "dailyLimit": c.DailyLimit, "label": c.Label},
			Metadata: map[string]any{"code": c.ID.String(), "summary": "Connected with the partner code for " + c.Label}})
	})
	if err != nil {
		s.log().ErrorContext(ctx, "partner connect", "err", err)
		writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
		return
	}
	if failStatus != 0 {
		writeProblem(w, failStatus, failCode, "", failAt)
		return
	}
	s.writeSigned(w, pp.Prefix+"/connect", answer, []string{secret})
}

// --- Signed messages about a link --------------------------------------------------------

func (s *Service) serveLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "link"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, pp.ErrUnknownLink, "", nil)
		return
	}
	l, err := s.Q.PartnerLinkByID(ctx, id)
	if err != nil {
		writeProblem(w, http.StatusNotFound, pp.ErrUnknownLink, "", nil)
		return
	}
	now := s.now()
	secrets, err := s.liveSecrets(l, now)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
		return
	}
	path := canonPath(r)
	if err := pp.Verify(secrets, r.Header.Get(pp.HeaderTimestamp), r.Header.Get(pp.HeaderSignature), r.Method, path, body, now); err != nil {
		writeProblem(w, http.StatusUnauthorized, pp.ErrSignatureRejected, err.Error(), nil)
		return
	}
	if l.Status == LinkEnded {
		writeProblem(w, http.StatusGone, pp.ErrLinkEnded, "The partnership ended.", l.EndedAt)
		return
	}
	ws, err := s.Q.GetWorkspace(ctx, l.WorkspaceID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, pp.ErrUnknownLink, "", nil)
		return
	}
	switch chi.URLParam(r, "action") {
	case "subscribe":
		var in pp.SubscribeRequest
		if l.Direction != DirIn || json.Unmarshal(body, &in) != nil || in.Request == "" || len(in.Request) > 100 {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		out, err := s.receiveSubscribe(ctx, &ws, l, in)
		if err != nil {
			s.log().ErrorContext(ctx, "partner subscribe", "link", l.ID, "err", err)
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		s.writeSigned(w, path, out, secrets)
	case "confirmed":
		var in pp.ConfirmedNotice
		if l.Direction != DirOut || json.Unmarshal(body, &in) != nil {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		if err := s.receiveConfirmed(ctx, &ws, l, in); err != nil {
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		s.writeSigned(w, path, pp.SubscribeResponse{Protocol: pp.Version, Status: pp.StatusOK}, secrets)
	case "state":
		var in pp.StateNotice
		if json.Unmarshal(body, &in) != nil {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		if err := s.receiveState(ctx, &ws, l, in.State); err != nil {
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		s.writeSigned(w, path, pp.SubscribeResponse{Protocol: pp.Version, Status: pp.StatusOK}, secrets)
	case "profile":
		var in pp.ProfileNotice
		if l.Direction != DirOut || json.Unmarshal(body, &in) != nil {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		if err := s.receiveProfile(ctx, l, in); err != nil {
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		s.writeSigned(w, path, pp.SubscribeResponse{Protocol: pp.Version, Status: pp.StatusOK}, secrets)
	case "rotate":
		var in pp.RotateRequest
		if json.Unmarshal(body, &in) != nil || in.Rotation == "" || in.Key == "" {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		if in.Rotation == l.LastRotation {
			writeProblem(w, http.StatusConflict, pp.ErrBadRequest, "This rotation was already applied. Start a new one.", nil)
			return
		}
		keys, err := pp.NewKeyPair()
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		rec, partner := recommenderAndPartner(l)
		secret, err := keys.Secret(in.Key, pp.RotateSalt(in.Rotation), rec, partner)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, pp.ErrBadRequest, "", nil)
			return
		}
		if err := s.applyRotation(ctx, &ws, l, in.Rotation, secret, false); err != nil {
			writeProblem(w, http.StatusInternalServerError, pp.ErrBadRequest, "", nil)
			return
		}
		// Signed with the secrets the message came with: the other side
		// switches when it has our key.
		s.writeSigned(w, path, pp.RotateResponse{Protocol: pp.Version, Key: keys.Public()}, secrets)
	default:
		writeProblem(w, http.StatusNotFound, pp.ErrBadRequest, "No such message.", nil)
	}
}

// receiveState applies the partner's pause, resume or end.
func (s *Service) receiveState(ctx context.Context, ws *store.Workspace, l store.PartnerLink, state string) error {
	now := s.now()
	actor := &audit.Actor{Type: authz.ActorSystem, Name: "Partner · " + l.RemoteName}
	switch state {
	case pp.StatePaused:
		if l.Status != LinkConnected {
			return nil
		}
		return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.PartnerLinkSetStatus(ctx, store.PartnerLinkSetStatusParams{ID: l.ID, Status: LinkPaused, PausedSide: "remote", PausedAt: &now, Now: now}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_paused", TargetType: "partner_link", TargetID: l.ID.String(),
				TargetName: l.RemoteName, Actor: actor, After: map[string]any{"status": LinkPaused, "by": "the partner"}})
		})
	case pp.StateActive:
		if l.Status != LinkPaused || l.PausedSide != "remote" {
			return nil
		}
		return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.PartnerLinkSetStatus(ctx, store.PartnerLinkSetStatusParams{ID: l.ID, Status: LinkConnected, Now: now}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_resumed", TargetType: "partner_link", TargetID: l.ID.String(),
				TargetName: l.RemoteName, Actor: actor, After: map[string]any{"status": LinkConnected, "by": "the partner"}})
		})
	case pp.StateEnded:
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if l.Direction == DirOut {
				// The partner ended it: our recommendation turns invalid
				// and hides; a new partner code fixes it.
				if err := q.PartnerLinkInvalid(ctx, store.PartnerLinkInvalidParams{ID: l.ID, Reason: "ended", Now: now}); err != nil {
					return err
				}
			} else if err := q.PartnerLinkEnd(ctx, store.PartnerLinkEndParams{ID: l.ID, Now: now, Side: "remote"}); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE partner_requests_out SET status = 'failed', last_error = 'The partner ended the partnership.', email = NULL,
				next_attempt_at = NULL WHERE link_id = $1 AND status = 'queued'`, l.ID); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_ended", TargetType: "partner_link", TargetID: l.ID.String(),
				TargetName: l.RemoteName, Actor: actor, After: map[string]any{"status": LinkEnded, "by": "the partner"}})
		})
		if err == nil && l.Direction == DirOut {
			s.notifyProblem(ctx, ws, l.ID)
		}
		return err
	}
	return nil
}

// receiveProfile updates what we show of a partner we recommend.
func (s *Service) receiveProfile(ctx context.Context, l store.PartnerLink, in pp.ProfileNotice) error {
	name := strings.TrimSpace(in.Profile.Name)
	if name == "" {
		name = l.RemoteName
	}
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	desc := in.Profile.Description
	if r := []rune(desc); len(r) > 120 {
		desc = string(r[:119]) + "…"
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkSetProfile(ctx, store.PartnerLinkSetProfileParams{ID: l.ID, RemoteName: name, RemoteDescription: desc, RemoteLogoUrl: in.Profile.LogoURL,
			RemoteListName: in.Profile.List, DailyLimit: int32(max(in.DailyLimit, 0)), Now: now}); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE recommendations SET name = $2, description = $3, updated_at = $4 WHERE partner_link_id = $1 AND removed_at IS NULL`,
			l.ID, name, desc, now)
		return err
	})
}

var errNoLink = errors.New("recommendations: no such partner link")
