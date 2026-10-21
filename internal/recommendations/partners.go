package recommendations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	pp "github.com/hostrepute/reputecast/internal/recommendations/partnerproto"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// secretAD binds a partner link's sealed secret to its row.
func secretAD(link uuid.UUID, field string) string {
	return "partner_link:" + link.String() + ":" + field
}

// liveSecrets are a link's secrets that sign and verify messages now: the
// current one, and the previous one for 24 hours after a rotation.
func (s *Service) liveSecrets(l store.PartnerLink, now time.Time) ([]string, error) {
	cur, err := s.Box.OpenString(l.Secret, secretAD(l.ID, "secret"))
	if err != nil {
		return nil, err
	}
	out := []string{cur}
	if len(l.PrevSecret) > 0 && l.PrevSecretUntil != nil && now.Before(*l.PrevSecretUntil) {
		if prev, err := s.Box.OpenString(l.PrevSecret, secretAD(l.ID, "prev_secret")); err == nil {
			out = append(out, prev)
		}
	}
	return out, nil
}

// --- Calling a partner -----------------------------------------------------------

// callError is a call that didn't get a usable answer.
type callError struct {
	// Kind: unreachable (network, timeout, 5xx), private (the SSRF guard),
	// unsupported (no partner endpoint), problem (an error answer),
	// signature (an answer whose signature doesn't match).
	Kind    string
	Status  int
	Problem pp.Problem
	Err     error
}

func (e *callError) Error() string {
	if e.Problem.Error != "" {
		return "partner answered " + strconv.Itoa(e.Status) + ": " + e.Problem.Error
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Kind
}

// checkBase validates a partner's base URL: HTTPS (plain HTTP only where
// private networks are allowed, in development) through the SSRF guard.
func (s *Service) checkBase(base string) (*url.URL, *callError) {
	u, err := s.guard().ValidateURL(base)
	if err != nil {
		if netguard.IsBlocked(err) {
			return nil, &callError{Kind: "private", Err: err}
		}
		return nil, &callError{Kind: "unreachable", Err: err}
	}
	if u.Scheme != "https" && !s.allowPrivate() {
		return nil, &callError{Kind: "unreachable", Err: errors.New("partner instances answer over HTTPS")}
	}
	return u, nil
}

// call sends one protocol message. With secrets the request is signed and
// the answer's signature checked. path starts with /partner/v1.
func (s *Service) call(ctx context.Context, base, method, path string, body any, secrets []string, out any, timeout time.Duration) *callError {
	if _, ce := s.checkBase(base); ce != nil {
		return ce
	}
	var raw []byte
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			return &callError{Kind: "unreachable", Err: err}
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return &callError{Kind: "unreachable", Err: err}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast/"+version.Version+" (partner link)")
	req.Header.Set(pp.HeaderProtocol, strconv.Itoa(pp.Version))
	if len(secrets) > 0 {
		ts := s.now().Unix()
		req.Header.Set(pp.HeaderTimestamp, strconv.FormatInt(ts, 10))
		req.Header.Set(pp.HeaderSignature, pp.SignatureHeader(secrets, ts, method, path, raw))
	}
	resp, err := s.client(timeout).Do(req)
	if err != nil {
		if netguard.IsBlocked(err) {
			return &callError{Kind: "private", Err: err}
		}
		return &callError{Kind: "unreachable", Err: err}
	}
	defer resp.Body.Close() //nolint:errcheck
	answer, err := io.ReadAll(io.LimitReader(resp.Body, pp.MaxBody))
	if err != nil {
		return &callError{Kind: "unreachable", Status: resp.StatusCode, Err: err}
	}
	if resp.StatusCode >= 500 {
		return &callError{Kind: "unreachable", Status: resp.StatusCode, Err: fmt.Errorf("answered %d", resp.StatusCode)}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var p pp.Problem
		if json.Unmarshal(answer, &p) != nil || p.Error == "" {
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
				return &callError{Kind: "unsupported", Status: resp.StatusCode}
			}
			return &callError{Kind: "unreachable", Status: resp.StatusCode, Err: fmt.Errorf("answered %d", resp.StatusCode)}
		}
		return &callError{Kind: "problem", Status: resp.StatusCode, Problem: p}
	}
	if len(secrets) > 0 {
		if err := pp.Verify(secrets, resp.Header.Get(pp.HeaderTimestamp), resp.Header.Get(pp.HeaderSignature), pp.ResponseMethod(resp.StatusCode), path,
			answer, s.now()); err != nil {
			return &callError{Kind: "signature", Status: resp.StatusCode, Err: err}
		}
	}
	if out != nil {
		if err := json.Unmarshal(answer, out); err != nil {
			return &callError{Kind: "unsupported", Status: resp.StatusCode, Err: err}
		}
	}
	return nil
}

// linkPath is a partner's endpoint for one of its links.
func linkPath(remoteLink, action string) string {
	return pp.Prefix + "/links/" + url.PathEscape(remoteLink) + "/" + action
}

// signedCall sends a signed message about a link to the other side.
func (s *Service) signedCall(ctx context.Context, l store.PartnerLink, action string, body, out any) *callError {
	secrets, err := s.liveSecrets(l, s.now())
	if err != nil {
		return &callError{Kind: "unreachable", Err: err}
	}
	return s.call(ctx, l.RemoteUrl, http.MethodPost, linkPath(l.RemoteLinkID, action), body, secrets, out, 10*time.Second)
}

// --- Partner codes (the recommended side) ----------------------------------------

// PartnerCodeInput creates a partner code ("Let a newsletter recommend you").
type PartnerCodeInput struct {
	Label      string `json:"label" maxLength:"120"`
	ListID     string `json:"listId" format:"uuid"`
	DailyLimit int    `json:"dailyLimit" minimum:"1" maximum:"100000"`
}

// PartnerCodeCreated is a new partner code, shown once.
type PartnerCodeCreated struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	URL        string    `json:"url"`
	Label      string    `json:"label"`
	ListName   string    `json:"listName"`
	DailyLimit int       `json:"dailyLimit"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

var (
	mLabelRequired = msgs.M("code.labelRequired")
	mNoInstance    = msgs.M("code.noInstance")
)

// CreateCode creates a single-use partner code valid 7 days. Only its hash
// is kept: the code is shown this once.
func (s *Service) CreateCode(ctx context.Context, ws *store.Workspace, in PartnerCodeInput) (PartnerCodeCreated, error) {
	in.Label = strings.TrimSpace(in.Label)
	if in.Label == "" {
		return PartnerCodeCreated{}, api.Invalid("body.label", s.text(ctx, mLabelRequired, nil))
	}
	list := uuidOrNil(in.ListID)
	if list == nil {
		return PartnerCodeCreated{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	l, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: *list})
	if err != nil {
		return PartnerCodeCreated{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	if in.DailyLimit <= 0 {
		in.DailyLimit = 50
	}
	if s.instance() == "" {
		return PartnerCodeCreated{}, api.Err(http.StatusConflict, "no_public_url", s.text(ctx, mNoInstance, nil))
	}
	code := pp.NewCode()
	now := s.now()
	out := PartnerCodeCreated{ID: uuid.Must(uuid.NewV7()).String(), Code: code, URL: pp.CodeURL(s.instance(), code), Label: in.Label, ListName: l.Name,
		DailyLimit: in.DailyLimit, CreatedAt: now, ExpiresAt: now.Add(pp.CodeTTL)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerCodeInsert(ctx, store.PartnerCodeInsertParams{ID: uuid.MustParse(out.ID), WorkspaceID: ws.ID, CodeHash: pp.HashCode(code), Label: in.Label,
			ListID: *list, DailyLimit: int32(in.DailyLimit), CreatedBy: actor(ctx), Now: now, ExpiresAt: out.ExpiresAt}); err != nil {
			return err
		}
		// The code itself is never logged.
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.code_created", TargetType: "partner_code", TargetID: out.ID,
			TargetName: in.Label, After: map[string]any{"label": in.Label, "list": l.Name, "dailyLimit": in.DailyLimit, "expiresAt": out.ExpiresAt}})
	})
	return out, err
}

// RevokeCode revokes a partner code nobody has used yet.
func (s *Service) RevokeCode(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	c, err := s.Q.PartnerCodeLock(ctx, id)
	if err != nil || c.WorkspaceID != ws.ID {
		return api.NotFound("Partner code")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerCodeRevoke(ctx, store.PartnerCodeRevokeParams{ID: id, Now: s.now(), By: actor(ctx)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.code_revoked", TargetType: "partner_code", TargetID: id.String(),
			TargetName: c.Label})
	})
}

// --- Connecting (the recommending side) ------------------------------------------

// PartnerPreview is a partner's profile before connecting.
type PartnerPreview struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Initials    string `json:"initials"`
	LogoURL     string `json:"logoUrl"`
	List        string `json:"list"`
	DailyLimit  int    `json:"dailyLimit"`
	DoubleOptIn bool   `json:"doubleOptIn"`
	Instance    string `json:"instance"`
}

// codeProblem turns a failed check of a partner code into the dialog's
// states (RecommendationStates): expired, used, not a code, unreachable,
// not supported, a private address.
func (s *Service) codeProblem(ctx context.Context, ws *store.Workspace, base, code string, ce *callError) error {
	host := hostOf(base, false)
	l := i18n.LangFrom(ctx)
	switch ce.Kind {
	case "private":
		addr := host
		if h, _, err := splitHost(host); err == nil {
			addr = h
		}
		return api.Err(http.StatusUnprocessableEntity, "partner_private", msgs.M("partner.private").Text(l, i18n.Args{"address": addr})).WithMeta("host", host)
	case "unsupported":
		return api.Err(http.StatusUnprocessableEntity, "partner_unsupported", msgs.M("partner.unsupported").Text(l, i18n.Args{"host": host})).WithMeta("host", host)
	case "problem":
		at := ""
		if ce.Problem.At != nil {
			at = ce.Problem.At.Format("2 Jan")
		}
		switch ce.Problem.Error {
		case pp.ErrCodeExpired:
			return api.Err(http.StatusUnprocessableEntity, "code_expired", msgs.M("partner.codeExpired").Text(l, i18n.Args{"date": at})).WithMeta("at", at)
		case pp.ErrCodeUsed:
			if row, err := s.Q.PartnerLinkByCodeHash(ctx, store.PartnerLinkByCodeHashParams{WorkspaceID: ws.ID, CodeHash: pp.HashCode(code)}); err == nil {
				return api.Err(http.StatusUnprocessableEntity, "code_used", msgs.M("partner.codeUsedBy").Text(l, i18n.Args{"name": row.ConnectedByName,
					"date": row.ConnectedAt.Format("2 Jan"), "partner": row.RemoteName})).WithMeta("at", at)
			}
			return api.Err(http.StatusUnprocessableEntity, "code_used", msgs.M("partner.codeUsed").Text(l, i18n.Args{"date": at})).WithMeta("at", at)
		case pp.ErrCodeRevoked:
			return api.Err(http.StatusUnprocessableEntity, "code_revoked", msgs.M("partner.codeRevoked").Text(l, nil))
		case pp.ErrCodeUnknown:
			return api.Err(http.StatusUnprocessableEntity, "code_unknown", msgs.M("partner.codeUnknown").Text(l, i18n.Args{"host": host}))
		case pp.ErrUnsupportedVersion:
			return api.Err(http.StatusUnprocessableEntity, "partner_unsupported", msgs.M("partner.unsupported").Text(l, i18n.Args{"host": host})).WithMeta("host", host)
		}
	}
	return api.Err(http.StatusUnprocessableEntity, "partner_unreachable", msgs.M("partner.unreachable").Text(l, i18n.Args{"host": host})).WithMeta("host", host)
}

func splitHost(hostport string) (string, string, error) {
	if i := strings.LastIndex(hostport, ":"); i > 0 && !strings.Contains(hostport[i:], "]") {
		return strings.Trim(hostport[:i], "[]"), hostport[i+1:], nil
	}
	return strings.Trim(hostport, "[]"), "", nil
}

// readCode reads a pasted partner code, or says what it is instead.
func (s *Service) readCode(ctx context.Context, raw string) (base, code string, err error) {
	base, code, err = pp.ParseCodeURL(raw)
	if err == nil {
		return base, code, nil
	}
	l := i18n.LangFrom(ctx)
	if u, perr := url.Parse(strings.TrimSpace(raw)); perr == nil && u.Host != "" && (u.Scheme == "https" || u.Scheme == "http") {
		return "", "", api.Err(http.StatusUnprocessableEntity, "not_a_code", msgs.M("partner.notACodeLink").Text(l, nil))
	}
	return "", "", api.Err(http.StatusUnprocessableEntity, "not_a_code", msgs.M("partner.notACode").Text(l, nil))
}

// hello checks that a partner instance speaks the protocol.
func (s *Service) hello(ctx context.Context, base string) *callError {
	var h pp.Hello
	if ce := s.call(ctx, base, http.MethodGet, pp.Prefix, nil, nil, &h, 10*time.Second); ce != nil {
		return ce
	}
	for _, v := range h.Protocols {
		if v == pp.Version {
			return nil
		}
	}
	return &callError{Kind: "unsupported"}
}

// PreviewPartner asks the partner's instance what a pasted code connects to.
func (s *Service) PreviewPartner(ctx context.Context, ws *store.Workspace, raw string) (PartnerPreview, error) {
	base, code, err := s.readCode(ctx, raw)
	if err != nil {
		return PartnerPreview{}, err
	}
	if ce := s.hello(ctx, base); ce != nil {
		return PartnerPreview{}, s.codeProblem(ctx, ws, base, code, ce)
	}
	var pr pp.PreviewResponse
	if ce := s.call(ctx, base, http.MethodPost, pp.Prefix+"/preview", pp.PreviewRequest{Protocol: pp.Version, Code: code}, nil, &pr, 10*time.Second); ce != nil {
		return PartnerPreview{}, s.codeProblem(ctx, ws, base, code, ce)
	}
	return PartnerPreview{Code: pp.CodeURL(base, code), Name: pr.Profile.Name, Description: pr.Profile.Description, Initials: initials(pr.Profile.Name),
		LogoURL: pr.Profile.LogoURL, List: pr.Profile.List, DailyLimit: pr.DailyLimit, DoubleOptIn: pr.DoubleOptIn, Instance: hostOf(base, false)}, nil
}

// PartnerConnectInput connects a partner from its code.
type PartnerConnectInput struct {
	Code        string `json:"code" maxLength:"500"`
	ShowConfirm bool   `json:"showConfirm"`
	ShowBlock   bool   `json:"showBlock"`
}

// ownProfile is the profile partners get from this workspace.
func (s *Service) ownProfile(ctx context.Context, ws *store.Workspace) (pp.Profile, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return pp.Profile{}, err
	}
	p, err := s.profile(ctx, ws, set)
	if err != nil {
		return pp.Profile{}, err
	}
	out := pp.Profile{Name: p.Name, Description: p.Description, List: p.ListName}
	if p.Logo != nil && s.instance() != "" {
		out.LogoURL = s.instance() + p.Logo.URL
	}
	return out, nil
}

// ConnectPartner uses a partner code: the two instances agree a signing
// secret over HTTPS, and the partner joins "You recommend".
func (s *Service) ConnectPartner(ctx context.Context, ws *store.Workspace, in PartnerConnectInput) (RecommendationItem, error) {
	if !in.ShowConfirm && !in.ShowBlock {
		return RecommendationItem{}, api.Invalid("body.showConfirm", s.text(ctx, mWhere, nil))
	}
	base, code, err := s.readCode(ctx, in.Code)
	if err != nil {
		return RecommendationItem{}, err
	}
	if s.instance() == "" {
		return RecommendationItem{}, api.Err(http.StatusConflict, "no_public_url", s.text(ctx, mNoInstance, nil))
	}
	if ce := s.hello(ctx, base); ce != nil {
		return RecommendationItem{}, s.codeProblem(ctx, ws, base, code, ce)
	}
	me, err := s.ownProfile(ctx, ws)
	if err != nil {
		return RecommendationItem{}, err
	}
	keys, err := pp.NewKeyPair()
	if err != nil {
		return RecommendationItem{}, err
	}
	linkID := uuid.Must(uuid.NewV7())
	req := pp.ConnectRequest{Protocol: pp.Version, Code: code, Link: linkID.String(), Instance: s.instance(), Recommender: me, Key: keys.Public()}
	raw, _ := json.Marshal(req)
	// The answer is signed with the new secret: derive it from the
	// partner's key once it arrives, then check the signature.
	status, header, answer, ce := s.connectCall(ctx, base, raw)
	if ce != nil {
		return RecommendationItem{}, s.codeProblem(ctx, ws, base, code, ce)
	}
	var cr pp.ConnectResponse
	if err := json.Unmarshal(answer, &cr); err != nil || cr.Link == "" || cr.Key == "" {
		return RecommendationItem{}, s.codeProblem(ctx, ws, base, code, &callError{Kind: "unsupported"})
	}
	secret, err := keys.Secret(cr.Key, pp.ConnectSalt(code), linkID.String(), cr.Link)
	if err != nil {
		return RecommendationItem{}, s.codeProblem(ctx, ws, base, code, &callError{Kind: "unsupported", Err: err})
	}
	if err := pp.Verify([]string{secret}, header.Get(pp.HeaderTimestamp), header.Get(pp.HeaderSignature), pp.ResponseMethod(status),
		pp.Prefix+"/connect", answer, s.now()); err != nil {
		return RecommendationItem{}, s.codeProblem(ctx, ws, base, code, &callError{Kind: "signature", Err: err})
	}
	sealed, err := s.Box.SealString(secret, secretAD(linkID, "secret"))
	if err != nil {
		return RecommendationItem{}, err
	}
	now := s.now()
	recID := uuid.Must(uuid.NewV7())
	name := strings.TrimSpace(cr.Profile.Name)
	if name == "" {
		name = hostOf(base, false)
	}
	desc := cr.Profile.Description
	if r := []rune(desc); len(r) > 120 {
		desc = string(r[:119]) + "…"
	}
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkInsert(ctx, store.PartnerLinkInsertParams{ID: linkID, WorkspaceID: ws.ID, Direction: DirOut, RemoteUrl: base, RemoteLinkID: cr.Link,
			ProtocolVersion: pp.Version, RemoteName: name, RemoteDescription: desc, RemoteLogoUrl: cr.Profile.LogoURL, RemoteListName: cr.Profile.List,
			DailyLimit: int32(max(cr.DailyLimit, 0)), CodeHash: pp.HashCode(code), Secret: sealed, Now: now, ConnectedBy: actor(ctx)}); err != nil {
			return err
		}
		if err := q.RecInsert(ctx, store.RecInsertParams{ID: recID, WorkspaceID: ws.ID, Kind: KindPartner, Name: name, Description: desc, PartnerLinkID: &linkID,
			ShowConfirm: in.ShowConfirm, ShowBlock: in.ShowBlock, CreatedBy: actor(ctx), Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_connected", TargetType: "partner_link", TargetID: linkID.String(),
			TargetName: name, After: map[string]any{"direction": "you recommend", "instance": hostOf(base, false), "dailyLimit": cr.DailyLimit,
				"showConfirm": in.ShowConfirm, "showBlock": in.ShowBlock}, Metadata: map[string]any{"related": recID.String()}})
	})
	if err != nil {
		return RecommendationItem{}, err
	}
	s.fetchPartnerLogo(ctx, ws, recID, cr.Profile.LogoURL)
	return s.Item(ctx, ws, recID)
}

// connectCall posts the connect request and keeps the raw answer (its
// signature is checked with the secret derived from it).
func (s *Service) connectCall(ctx context.Context, base string, raw []byte) (int, http.Header, []byte, *callError) {
	if _, ce := s.checkBase(base); ce != nil {
		return 0, nil, nil, ce
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+pp.Prefix+"/connect", bytes.NewReader(raw))
	if err != nil {
		return 0, nil, nil, &callError{Kind: "unreachable", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ReputeCast/"+version.Version+" (partner link)")
	req.Header.Set(pp.HeaderProtocol, strconv.Itoa(pp.Version))
	resp, err := s.client(15 * time.Second).Do(req)
	if err != nil {
		if netguard.IsBlocked(err) {
			return 0, nil, nil, &callError{Kind: "private", Err: err}
		}
		return 0, nil, nil, &callError{Kind: "unreachable", Err: err}
	}
	defer resp.Body.Close() //nolint:errcheck
	answer, _ := io.ReadAll(io.LimitReader(resp.Body, pp.MaxBody))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var p pp.Problem
		if json.Unmarshal(answer, &p) == nil && p.Error != "" {
			return 0, nil, nil, &callError{Kind: "problem", Status: resp.StatusCode, Problem: p}
		}
		if resp.StatusCode == http.StatusNotFound {
			return 0, nil, nil, &callError{Kind: "unsupported", Status: resp.StatusCode}
		}
		return 0, nil, nil, &callError{Kind: "unreachable", Status: resp.StatusCode}
	}
	return resp.StatusCode, resp.Header, answer, nil
}

// fetchPartnerLogo keeps the partner's logo in the media library, so
// readers' browsers never load it from the partner. A logo that can't be
// fetched leaves the initials.
func (s *Service) fetchPartnerLogo(ctx context.Context, ws *store.Workspace, recID uuid.UUID, logoURL string) {
	if logoURL == "" || s.Media == nil {
		return
	}
	id, ok := s.fetchImage(ctx, ws, logoURL)
	if !ok {
		return
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE recommendations SET logo_id = $2 WHERE id = $1`, recID, id); err != nil {
		s.log().WarnContext(ctx, "partner logo", "err", err)
	}
}

// --- Link actions ------------------------------------------------------------------

func (s *Service) link(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.PartnerLink, error) {
	l, err := s.Q.PartnerLinkGet(ctx, store.PartnerLinkGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return l, api.MapError(err, "Partner link")
	}
	return l, nil
}

var mEnded = msgs.M("link.ended")

// PauseLink pauses or resumes a partner link from this side: requests stop
// (and the recommendation hides) until it resumes. The partner is told over
// the signed channel; if it can't be reached now, the change still holds
// here.
func (s *Service) PauseLink(ctx context.Context, ws *store.Workspace, id uuid.UUID, pause bool) error {
	l, err := s.link(ctx, ws, id)
	if err != nil {
		return err
	}
	if l.Status == LinkEnded || l.Status == LinkInvalid {
		return api.Err(http.StatusConflict, "link_"+l.Status, s.text(ctx, mEnded, nil))
	}
	now := s.now()
	status, side, action, state := LinkConnected, "", "partner.link_resumed", pp.StateActive
	var pausedAt *time.Time
	var by *uuid.UUID
	if pause {
		status, side, action, state, pausedAt, by = LinkPaused, "local", "partner.link_paused", pp.StatePaused, &now, actor(ctx)
	} else if l.Status == LinkPaused && l.PausedSide == "remote" {
		// The partner paused it: only they can resume.
		return api.Err(http.StatusConflict, "paused_by_partner", s.text(ctx, msgs.M("link.pausedByPartner"), i18n.Args{"partner": l.RemoteName}))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkSetStatus(ctx, store.PartnerLinkSetStatusParams{ID: id, Status: status, PausedSide: side, PausedAt: pausedAt, PausedBy: by, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "partner_link", TargetID: id.String(), TargetName: l.RemoteName,
			Before: map[string]any{"status": l.Status}, After: map[string]any{"status": status}, Metadata: map[string]any{"direction": l.Direction}})
	})
	if err != nil {
		return err
	}
	if ce := s.signedCall(ctx, l, "state", pp.StateNotice{Protocol: pp.Version, State: state}, nil); ce != nil {
		s.log().WarnContext(ctx, "partner link: tell the partner", "link", id, "state", state, "err", ce)
	}
	return nil
}

// EndLink ends a partner link for good: requests stop, both sides drop it
// from their lists, and reconnecting needs a new partner code.
func (s *Service) EndLink(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	l, err := s.link(ctx, ws, id)
	if err != nil {
		return err
	}
	if l.Status == LinkEnded {
		return nil
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkEnd(ctx, store.PartnerLinkEndParams{ID: id, Now: now, By: actor(ctx), Side: "local"}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE partner_requests_out SET status = 'failed', last_error = 'The partnership ended.', email = NULL, next_attempt_at = NULL
			WHERE link_id = $1 AND status = 'queued'`, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.link_ended", TargetType: "partner_link", TargetID: id.String(), TargetName: l.RemoteName,
			Before: map[string]any{"status": l.Status}, After: map[string]any{"status": LinkEnded}, Metadata: map[string]any{"direction": l.Direction}})
	})
	if err != nil {
		return err
	}
	if l.Status != LinkInvalid {
		if ce := s.signedCall(ctx, l, "state", pp.StateNotice{Protocol: pp.Version, State: pp.StateEnded}, nil); ce != nil {
			s.log().WarnContext(ctx, "partner link: tell the partner it ended", "link", id, "err", ce)
		}
	}
	return nil
}

// PartnerLimitInput changes the daily limit a partner has for its readers.
type PartnerLimitInput struct {
	DailyLimit int `json:"dailyLimit" minimum:"1" maximum:"100000"`
}

// SetLimit changes the daily limit of a partner recommending this
// workspace, and tells the partner.
func (s *Service) SetLimit(ctx context.Context, ws *store.Workspace, id uuid.UUID, limit int) error {
	l, err := s.link(ctx, ws, id)
	if err != nil {
		return err
	}
	if l.Direction != DirIn {
		return api.Err(http.StatusConflict, "not_recommending_you", s.text(ctx, msgs.M("link.limitOut"), nil))
	}
	if l.Status == LinkEnded {
		return api.Err(http.StatusConflict, "link_ended", s.text(ctx, mEnded, nil))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkSetLimit(ctx, store.PartnerLinkSetLimitParams{ID: id, DailyLimit: int32(limit), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "partner.limit_changed", TargetType: "partner_link", TargetID: id.String(),
			TargetName: l.RemoteName, Before: map[string]any{"dailyLimit": l.DailyLimit}, After: map[string]any{"dailyLimit": limit}})
	})
	if err != nil {
		return err
	}
	l.DailyLimit = int32(limit)
	s.sendProfile(ctx, ws, l)
	return nil
}

// sendProfile tells a partner recommending this workspace its profile and
// daily limit.
func (s *Service) sendProfile(ctx context.Context, ws *store.Workspace, l store.PartnerLink) {
	if l.Status == LinkEnded || l.Status == LinkInvalid {
		return
	}
	me, err := s.ownProfile(ctx, ws)
	if err != nil {
		return
	}
	if l.ListID != nil {
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, *l.ListID).Scan(&me.List)
	}
	if ce := s.signedCall(ctx, l, "profile", pp.ProfileNotice{Protocol: pp.Version, Profile: me, DailyLimit: int(l.DailyLimit)}, nil); ce != nil {
		s.log().WarnContext(ctx, "partner link: send the profile", "link", l.ID, "err", ce)
	}
}

// announceProfile sends the new profile to every partner recommending
// this workspace.
func (s *Service) announceProfile(ctx context.Context, ws *store.Workspace) {
	links, err := s.Q.PartnerLinks(ctx, store.PartnerLinksParams{WorkspaceID: ws.ID, Direction: DirIn})
	if err != nil {
		return
	}
	for _, r := range links {
		l, err := s.Q.PartnerLinkByID(ctx, r.ID)
		if err == nil {
			s.sendProfile(ctx, ws, l)
		}
	}
}

// RotateSecret agrees a new signing secret with the partner over the signed
// channel. Both secrets work for 24 hours.
func (s *Service) RotateSecret(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	l, err := s.link(ctx, ws, id)
	if err != nil {
		return err
	}
	if l.Status == LinkEnded || l.Status == LinkInvalid {
		return api.Err(http.StatusConflict, "link_"+l.Status, s.text(ctx, mEnded, nil))
	}
	keys, err := pp.NewKeyPair()
	if err != nil {
		return err
	}
	rotation := uuid.Must(uuid.NewV7()).String()
	var rr pp.RotateResponse
	if ce := s.signedCall(ctx, l, "rotate", pp.RotateRequest{Protocol: pp.Version, Rotation: rotation, Key: keys.Public()}, &rr); ce != nil {
		s.linkTrouble(ctx, ws, l, ce)
		return api.Err(http.StatusBadGateway, "partner_unreachable", s.text(ctx, msgs.M("partner.unreachable"), i18n.Args{"host": hostOf(l.RemoteUrl, false)}))
	}
	rec, partner := recommenderAndPartner(l)
	secret, err := keys.Secret(rr.Key, pp.RotateSalt(rotation), rec, partner)
	if err != nil {
		return err
	}
	return s.applyRotation(ctx, ws, l, rotation, secret, true)
}

// recommenderAndPartner are the two link ids in the order the key
// agreement binds them.
func recommenderAndPartner(l store.PartnerLink) (recommender, partner string) {
	if l.Direction == DirOut {
		return l.ID.String(), l.RemoteLinkID
	}
	return l.RemoteLinkID, l.ID.String()
}

func (s *Service) applyRotation(ctx context.Context, ws *store.Workspace, l store.PartnerLink, rotation, secret string, byUs bool) error {
	now := s.now()
	sealed, err := s.Box.SealString(secret, secretAD(l.ID, "secret"))
	if err != nil {
		return err
	}
	cur, err := s.Box.OpenString(l.Secret, secretAD(l.ID, "secret"))
	if err != nil {
		return err
	}
	prev, err := s.Box.SealString(cur, secretAD(l.ID, "prev_secret"))
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PartnerLinkRotate(ctx, store.PartnerLinkRotateParams{ID: l.ID, PrevSecret: prev, Until: now.Add(pp.PreviousSecretGrace), Secret: sealed, Now: now,
			Rotation: rotation}); err != nil {
			return err
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "partner.secret_rotated", TargetType: "partner_link", TargetID: l.ID.String(), TargetName: l.RemoteName,
			Metadata: map[string]any{"by": map[bool]string{true: "this side", false: "the partner"}[byUs]}}
		if !byUs {
			ev.Actor = &audit.Actor{Type: authz.ActorSystem, Name: "Partner · " + l.RemoteName}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
}

// linkTrouble records what a failed call says about a link: the partner
// rejected our signature or ended the partnership (the link turns invalid,
// members with grow:connect are told), or it couldn't be reached.
func (s *Service) linkTrouble(ctx context.Context, ws *store.Workspace, l store.PartnerLink, ce *callError) {
	now := s.now()
	reason := ""
	if ce.Kind == "problem" {
		switch ce.Problem.Error {
		case pp.ErrSignatureRejected:
			reason = "signature_rejected"
		case pp.ErrUnknownLink:
			reason = "unknown_link"
		case pp.ErrLinkEnded:
			reason = "ended"
		}
	}
	if reason == "" {
		if l.Direction == DirOut {
			_ = s.Q.PartnerLinkFailing(ctx, store.PartnerLinkFailingParams{ID: l.ID, Failing: true, Now: now})
		}
		return
	}
	if err := s.Q.PartnerLinkInvalid(ctx, store.PartnerLinkInvalidParams{ID: l.ID, Reason: reason, Now: now}); err != nil {
		s.log().WarnContext(ctx, "partner link invalid", "err", err)
		return
	}
	s.notifyProblem(ctx, ws, l.ID)
}

// notifyProblem tells members with grow:connect once that a partner link is
// invalid or unreachable.
func (s *Service) notifyProblem(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	l, err := s.Q.PartnerLinkByID(ctx, id)
	if err != nil || l.NotifiedAt != nil {
		return
	}
	lang := i18n.For(i18n.English)
	n := notify.Notification{Kind: notify.KindPartnerLinkProblem, Severity: "bad", Link: "/w/" + ws.Slug + "/grow/recommendations",
		Data: map[string]any{"link": id.String(), "partner": l.RemoteName}}
	switch {
	case l.Status == LinkInvalid:
		n.Title = msgs.M("notify.invalidTitle").Text(lang, i18n.Args{"partner": l.RemoteName})
		n.Body = msgs.M("notify.invalidBody").Text(lang, nil)
	case l.FailingSince != nil:
		n.Severity = "warn"
		n.Title = msgs.M("notify.unreachableTitle").Text(lang, i18n.Args{"partner": l.RemoteName})
		n.Body = msgs.M("notify.unreachableBody").Text(lang, nil)
	default:
		return
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.GrowConnect, n); err != nil {
			return err
		}
		return store.New(tx).PartnerLinkNotified(ctx, store.PartnerLinkNotifiedParams{ID: id, Now: s.now()})
	})
	if err != nil {
		s.log().WarnContext(ctx, "partner link: notify", "err", err)
	}
}

// CheckAgain retries a partner's waiting requests now ("Check again").
func (s *Service) CheckAgain(ctx context.Context, ws *store.Workspace, id uuid.UUID) (RecommendationItem, error) {
	l, err := s.link(ctx, ws, id)
	if err != nil {
		return RecommendationItem{}, err
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE partner_requests_out SET next_attempt_at = $2 WHERE link_id = $1 AND status = 'queued'`, id, s.now()); err != nil {
		return RecommendationItem{}, err
	}
	if _, err := s.SendDue(ctx); err != nil {
		return RecommendationItem{}, err
	}
	if ce := s.hello(ctx, l.RemoteUrl); ce == nil {
		_ = s.Q.PartnerLinkFailing(ctx, store.PartnerLinkFailingParams{ID: id, Failing: false, Now: s.now()})
	}
	rec, err := s.Q.RecOfLink(ctx, &id)
	if err != nil {
		return RecommendationItem{}, api.NotFound("Recommendation")
	}
	return s.Item(ctx, ws, rec.ID)
}

// clientIP is where a request came from (rate limits of the partner
// endpoints).
func clientIP(r *http.Request) string {
	if ip, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		return ip.Addr().String()
	}
	return r.RemoteAddr
}
