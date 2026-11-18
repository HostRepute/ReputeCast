package mcpclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// authTransport adds a server's credential to each request and remembers
// the last status, to tell a refused sign-in from an unreachable server.
// It sends only to the configured server's origin (scheme, host and port):
// a request elsewhere, such as an HTTP with SSE endpoint event naming
// another host or plain http for an https server, isn't sent at all.
type authTransport struct {
	base          http.RoundTripper
	serverOrigin  string
	header, value string
	mu            sync.Mutex
	status        int
	elsewhere     string
}

// errElsewhere is a request refused for going to another origin.
var errElsewhere = errors.New("mcp: request to another origin than the server's refused")

// canonicalOrigin is a URL's scheme, host and port, the default port spelled out.
func canonicalOrigin(u *url.URL) string {
	scheme, port := strings.ToLower(u.Scheme), u.Port()
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	return scheme + "://" + net.JoinHostPort(strings.ToLower(u.Hostname()), port)
}

func (t *authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if o := canonicalOrigin(r.URL); o != t.serverOrigin {
		if r.Body != nil {
			_ = r.Body.Close()
		}
		t.mu.Lock()
		t.elsewhere = o
		t.mu.Unlock()
		return nil, errElsewhere
	}
	r = r.Clone(r.Context())
	if t.header != "" {
		r.Header.Set(t.header, t.value)
	}
	res, err := t.base.RoundTrip(r)
	if err == nil {
		t.mu.Lock()
		t.status = res.StatusCode
		t.mu.Unlock()
	}
	return res, err
}

func (t *authTransport) unauthorized() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status == http.StatusUnauthorized || t.status == http.StatusForbidden
}

// refused is the origin of a request refused for going elsewhere, if any.
func (t *authTransport) refused() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.elsewhere
}

// session is a connection to a server as one person.
type session struct {
	cs     *mcp.ClientSession
	rt     *authTransport
	cancel context.CancelFunc
}

func (s *session) close() {
	_ = s.cs.Close()
	s.cancel()
	if t, ok := s.rt.base.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}

// connect opens a session to a server: with its shared header, or as the
// person's own OAuth sign-in. Servers can't ask ReputeCast for sampling
// (no handler is offered) or elicitation.
func (s *Service) connect(ctx context.Context, srv store.McpServer, user uuid.UUID, transport string) (*session, error) {
	// The call's own context bounds it; the transport's header timeout is
	// only a backstop.
	u, err := url.Parse(srv.Url)
	if err != nil {
		return nil, unreachable(ctx, srv.Name)
	}
	rt := &authTransport{base: s.policy().Client(s.callTimeout() + 10*time.Second).Transport, serverOrigin: canonicalOrigin(u)}
	switch srv.Auth {
	case "header":
		v, err := s.Box.OpenString(srv.HeaderSealed, headerAD(srv.ID))
		if err != nil {
			return nil, err
		}
		rt.header, rt.value = srv.HeaderName, v
	case "oauth":
		token, err := s.accessToken(ctx, srv, user)
		if err != nil {
			return nil, err
		}
		rt.header, rt.value = "Authorization", "Bearer "+token
	}
	hc := &http.Client{Transport: rt, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var t mcp.Transport = &mcp.StreamableClientTransport{Endpoint: srv.Url, HTTPClient: hc, MaxRetries: -1, DisableStandaloneSSE: true}
	if transport == "sse" {
		t = &mcp.SSEClientTransport{Endpoint: srv.Url, HTTPClient: hc}
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "ReputeCast", Version: "1"}, &mcp.ClientOptions{})
	// The session's context outlives Connect (HTTP with SSE keeps its
	// stream on it); Connect itself has the call's timeout.
	sctx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(s.callTimeout(), cancel)
	cs, err := client.Connect(sctx, t, nil)
	late := !timer.Stop()
	if err != nil {
		cancel()
		if late {
			return nil, problem(ctx, http.StatusGatewayTimeout, CodeTimeout, "problem.timeout", i18n.Args{"server": srv.Name})
		}
		return nil, s.connectProblem(ctx, srv, rt, err)
	}
	return &session{cs: cs, rt: rt, cancel: cancel}, nil
}

func (s *Service) connectProblem(ctx context.Context, srv store.McpServer, rt *authTransport, err error) error {
	var p *api.Problem
	switch {
	case errors.As(err, &p):
		return err
	case rt.refused() != "":
		s.log().WarnContext(ctx, "mcp: a request to another origin was refused", "server", srv.ID, "origin", rt.refused())
		return problem(ctx, http.StatusBadGateway, CodeUnreachable, "problem.elsewhere", i18n.Args{"server": srv.Name})
	case rt.unauthorized():
		if srv.Auth == "oauth" {
			return signIn(ctx, srv)
		}
		return problem(ctx, http.StatusBadGateway, CodeUnreachable, "problem.refused", i18n.Args{"server": srv.Name})
	case netguard.IsBlocked(err):
		return problem(ctx, http.StatusUnprocessableEntity, codeURL, "problem.url", i18n.Args{"reason": "blocked address"})
	case errors.Is(err, context.DeadlineExceeded):
		return problem(ctx, http.StatusGatewayTimeout, CodeTimeout, "problem.timeout", i18n.Args{"server": srv.Name})
	}
	return unreachable(ctx, srv.Name)
}

// Sync lists a server's tools into its catalog, as the manager who asks
// (an OAuth server needs their own sign-in). A new tool waits for a
// manager; one whose description, schema or annotations changed since it
// was decided on is off, and managers are told.
func (s *Service) Sync(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (RemoteMCPServer, error) {
	fresh, err := manager(ctx, s.Q, ws, p, authz.IntegrationsManage)
	if err != nil {
		return RemoteMCPServer{}, err
	}
	srv, err := s.Q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return RemoteMCPServer{}, api.MapError(err, "MCP server")
	}
	listed, transport, err := s.list(ctx, srv, fresh.UserID)
	if err != nil {
		status := "unreachable"
		var pr *api.Problem
		if errors.As(err, &pr) && pr.Code == CodeSignIn {
			status = "sign_in"
		}
		_ = s.Q.MCPServerStatus(ctx, store.MCPServerStatusParams{Status: status, StatusDetail: "", Now: s.now(), ID: srv.ID})
		return RemoteMCPServer{}, err
	}
	var out RemoteMCPServer
	changed := []string{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		srv, err := q.MCPServerLock(ctx, store.MCPServerLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "MCP server")
		}
		have, err := q.MCPTools(ctx, srv.ID)
		if err != nil {
			return err
		}
		byName := map[string]store.McpTool{}
		names := []string{}
		for _, t := range have {
			byName[t.Name] = t
			names = append(names, t.ModelName)
		}
		seen := map[string]bool{}
		added := 0
		for _, t := range listed {
			seen[t.name] = true
			hash := toolHash(t.description, t.schema, t.annotations)
			cur, ok := byName[t.name]
			if !ok {
				model := modelName(srv.Key, t.name, names)
				names = append(names, model)
				if err := q.MCPToolInsert(ctx, store.MCPToolInsertParams{ID: uuid.Must(uuid.NewV7()), ServerID: srv.ID, Name: t.name, ModelName: model,
					Description: t.description, InputSchema: t.schema, Annotations: t.annotations, Hash: hash, Now: s.now()}); err != nil {
					return err
				}
				added++
				continue
			}
			if cur.Decision != "pending" && !bytes.Equal(cur.ApprovedHash, hash) && bytes.Equal(cur.ApprovedHash, cur.Hash) {
				changed = append(changed, t.name)
			}
			if err := q.MCPToolSeen(ctx, store.MCPToolSeenParams{ID: cur.ID, Description: t.description, InputSchema: t.schema, Annotations: t.annotations, Hash: hash}); err != nil {
				return err
			}
		}
		removed := 0
		for _, t := range have {
			if !seen[t.Name] && t.RemovedAt == nil {
				removed++
				if err := q.MCPToolGone(ctx, store.MCPToolGoneParams{ID: t.ID, Now: s.now()}); err != nil {
					return err
				}
			}
		}
		now := s.now()
		if err := q.MCPServerStatus(ctx, store.MCPServerStatusParams{Status: "ok", StatusDetail: "", SyncedAt: &now, Now: now, ID: srv.ID}); err != nil {
			return err
		}
		if transport != srv.Transport {
			if _, err := tx.Exec(ctx, `UPDATE mcp_servers SET transport = $2 WHERE id = $1`, srv.ID, transport); err != nil {
				return err
			}
		}
		if len(changed) > 0 {
			// Off until a manager decides again: they are told.
			l := i18n.Default
			if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.IntegrationsManage, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindIntegrationAttention,
				Severity: "warn", Title: msgs.M("notify.changedTitle").Text(l, i18n.Args{"server": srv.Name}),
				Body: msgs.M("notify.changedBody").Text(l, i18n.Args{"server": srv.Name, "count": len(changed)}),
				Link: "/w/" + ws.Slug + "/settings/integrations/mcp-servers"}); err != nil {
				return err
			}
		}
		if srv, err = q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		if out, err = s.view(ctx, q, srv, fresh.UserID, true); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_server.updated", TargetType: "mcp_server", TargetID: id.String(), TargetName: srv.Name,
			Metadata: map[string]any{"sync": true, "added": added, "changed": changed, "removed": removed, "transport": transport}})
	})
	return out, err
}

// listedTool is a tool as a server listed it.
type listedTool struct {
	name, description   string
	schema, annotations json.RawMessage
}

// list connects and lists the tools, falling back to HTTP with SSE when
// the server doesn't take Streamable HTTP.
func (s *Service) list(ctx context.Context, srv store.McpServer, user uuid.UUID) ([]listedTool, string, error) {
	transport := srv.Transport
	sess, err := s.connect(ctx, srv, user, transport)
	if err != nil && transport == "streamable" {
		var p *api.Problem
		if errors.As(err, &p) && p.Code == CodeUnreachable {
			if alt, altErr := s.connect(ctx, srv, user, "sse"); altErr == nil {
				sess, err, transport = alt, nil, "sse"
			} else if errors.As(altErr, &p) && p.Code == CodeUnreachable {
				// HTTP with SSE got as far: its problem can say more.
				err = altErr
			}
		}
	}
	if err != nil {
		return nil, transport, err
	}
	defer sess.close()
	lctx, cancel := context.WithTimeout(ctx, s.callTimeout())
	defer cancel()
	out := []listedTool{}
	for t, err := range sess.cs.Tools(lctx, nil) {
		if err != nil {
			return nil, transport, s.connectProblem(ctx, srv, sess.rt, err)
		}
		if len(out) == maxTools {
			break
		}
		schema, _ := json.Marshal(t.InputSchema)
		ann := []byte("{}")
		if t.Annotations != nil {
			ann, _ = json.Marshal(t.Annotations)
		}
		out = append(out, listedTool{name: clip(t.Name, 128), description: clip(t.Description, 4000), schema: schema, annotations: ann})
	}
	return out, transport, nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// modelName is a tool's name for the model, mcp__<key>__<tool>, within the
// 64 characters model APIs take; a clash gets a number.
func modelName(key, name string, taken []string) string {
	t := unsafeName.ReplaceAllString(name, "_")
	if len(t) > 36 {
		t = t[:36]
	}
	if t == "" {
		t = "tool"
	}
	out := "mcp__" + key + "__" + t
	for i := 2; slices.Contains(taken, out); i++ {
		out = "mcp__" + key + "__" + t + "_" + strconv.Itoa(i)
	}
	return out
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// RemoteMCPToolDecision decides on a tool, at the hash the manager reviewed.
type RemoteMCPToolDecision struct {
	Decision string `json:"decision" enum:"read_only,writes,blocked"`
	Hash     string `json:"hash" doc:"The hash of the description, schema and annotations reviewed"`
}

// Decide marks a tool read-only, writes or blocked, pinning the hash the
// manager reviewed; a tool that changed since gets a conflict.
func (s *Service) Decide(ctx context.Context, ws *store.Workspace, p *authz.Principal, server, toolID uuid.UUID, in RemoteMCPToolDecision) (RemoteMCPServer, error) {
	var out RemoteMCPServer
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := manager(ctx, q, ws, p, authz.IntegrationsManage)
		if err != nil {
			return err
		}
		srv, err := q.MCPServerLock(ctx, store.MCPServerLockParams{WorkspaceID: ws.ID, ID: server})
		if err != nil {
			return api.MapError(err, "MCP server")
		}
		t, err := q.MCPTool(ctx, store.MCPToolParams{WorkspaceID: ws.ID, ID: toolID})
		if err != nil || t.ServerID != server {
			return api.NotFound("MCP tool")
		}
		if hexOf(t.Hash) != in.Hash || t.RemovedAt != nil {
			return problem(ctx, http.StatusConflict, codeReviewed, "problem.reviewed", i18n.Args{"tool": t.Name})
		}
		if err := q.MCPToolDecide(ctx, store.MCPToolDecideParams{ID: t.ID, Decision: in.Decision, ApprovedHash: t.Hash, DecidedBy: &fresh.UserID, Now: s.now()}); err != nil {
			return err
		}
		action := "mcp_tool.approved"
		if in.Decision == "blocked" {
			action = "mcp_tool.revoked"
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "mcp_tool", TargetID: t.ID.String(), TargetName: t.Name,
			Before: map[string]any{"decision": t.Decision}, After: map[string]any{"decision": in.Decision, "hash": in.Hash},
			Metadata: map[string]any{"server_id": server.String(), "annotations": t.Annotations}}); err != nil {
			return err
		}
		out, err = s.view(ctx, q, srv, fresh.UserID, true)
		return err
	})
	return out, err
}

// ThreadTool is a tool a thread may offer the model.
type ThreadTool struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	ServerName  string
	Name        string
	ModelName   string
	Description string
	InputSchema map[string]any
	// Writes: each call waits for the person's confirmation.
	Writes bool
	Hash   []byte
}

// ThreadTools are the tools a thread may offer: on the servers its owner
// chose that they may still use, enabled, decided on at their current
// hash, with the owner's own sign-in where the server needs one.
func (s *Service) ThreadTools(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID) ([]ThreadTool, error) {
	rows, err := q.MCPThreadTools(ctx, store.MCPThreadToolsParams{ThreadID: thread, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	out := []ThreadTool{}
	ok := map[uuid.UUID]bool{}
	for _, r := range rows {
		allowed, known := ok[r.ServerID]
		if !known {
			if allowed, err = MayUse(ctx, q, ws.ID, owner.UserID, r.ServerRoles); err != nil {
				return nil, err
			}
			if allowed && r.ServerAuth == "oauth" {
				_, err := q.MCPToken(ctx, store.MCPTokenParams{ServerID: r.ServerID, UserID: owner.UserID})
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return nil, err
				}
				allowed = err == nil
			}
			ok[r.ServerID] = allowed
		}
		if !allowed {
			continue
		}
		var schema map[string]any
		if json.Unmarshal(r.InputSchema, &schema) != nil || schema == nil {
			schema = map[string]any{"type": "object"}
		}
		out = append(out, ThreadTool{ID: r.ID, ServerID: r.ServerID, ServerName: r.ServerName, Name: r.Name, ModelName: r.ModelName,
			Description: r.Description, InputSchema: schema, Writes: r.Decision == "writes", Hash: r.Hash})
	}
	return out, nil
}

// Current is a tool as it is now, for a call the person decides on: a
// changed, blocked or removed tool, or a server they may no longer use,
// is a problem.
func (s *Service) Current(ctx context.Context, q *store.Queries, ws *store.Workspace, owner *authz.Principal, toolID uuid.UUID, hash []byte) (store.McpTool, store.McpServer, error) {
	t, err := q.MCPTool(ctx, store.MCPToolParams{WorkspaceID: ws.ID, ID: toolID})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, store.McpServer{}, toolChanged(ctx, "")
	}
	if err != nil {
		return t, store.McpServer{}, err
	}
	srv, err := q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: t.ServerID})
	if err != nil {
		return t, srv, err
	}
	switch {
	case t.Decision == "blocked":
		return t, srv, toolBlocked(ctx, t.Name)
	case t.RemovedAt != nil || !bytes.Equal(t.Hash, hash) || !bytes.Equal(t.ApprovedHash, t.Hash) || t.Decision == "pending" || !srv.Enabled:
		return t, srv, toolChanged(ctx, t.Name)
	}
	allowed, err := MayUse(ctx, q, ws.ID, owner.UserID, srv.AccessRoles)
	if err != nil {
		return t, srv, err
	}
	if !allowed {
		return t, srv, api.Forbidden(authz.AIUse, nil)
	}
	return t, srv, nil
}

// Result is a call's result as the model may read it: text only, images
// and HTML stripped, at most MaxResult bytes.
type Result struct {
	Text    string
	IsError bool
	Ms      int64
	Digest  []byte
}

// CallDigest identifies a call in the audit log without its content: its
// arguments, salted with the call's id (which the log doesn't keep).
func CallDigest(callID string, toolID uuid.UUID, args json.RawMessage) []byte {
	h := sha256.New()
	h.Write([]byte("mcp_call:" + callID + ":" + toolID.String() + ":"))
	h.Write(args)
	return h.Sum(nil)
}

// Call calls a tool as the owner, at the hash it was offered with, within
// the call timeout. The call is audited with the digest of its arguments
// and its outcome, never their content.
func (s *Service) Call(ctx context.Context, ws *store.Workspace, owner *authz.Principal, thread uuid.UUID, callID string, toolID uuid.UUID, hash []byte, args json.RawMessage) (Result, error) {
	digest := CallDigest(callID, toolID, args)
	r := Result{Digest: digest}
	fresh, err := brandkit.VoiceActor(ctx, s.Q, ws, owner)
	if err != nil {
		return r, err
	}
	t, srv, err := s.Current(ctx, s.Q, ws, fresh, toolID, hash)
	if err != nil {
		return r, err
	}
	start := s.now()
	outcome := "ok"
	defer func() {
		r.Ms = s.now().Sub(start).Milliseconds()
		_ = s.Audit.Log(context.WithoutCancel(ctx), audit.Event{WorkspaceID: &ws.ID, Actor: &audit.Actor{Type: authz.ActorUser, ID: &fresh.UserID, Name: fresh.Name},
			Action: "mcp_tool.called", TargetType: "mcp_tool", TargetID: t.ID.String(), TargetName: t.Name,
			Metadata: map[string]any{"via": "ai_assistant", "thread_id": thread.String(), "server_id": srv.ID.String(), "arguments_digest": hexOf(digest),
				"outcome": outcome, "ms": r.Ms}})
	}()
	sess, err := s.connect(ctx, srv, fresh.UserID, srv.Transport)
	if err != nil {
		outcome = problemCode(err)
		if outcome == CodeSignIn && srv.Auth == "oauth" {
			s.SignedOut(ctx, srv.ID, fresh.UserID)
		}
		return r, err
	}
	defer sess.close()
	cctx, cancel := context.WithTimeout(ctx, s.callTimeout())
	defer cancel()
	var arguments any = map[string]any{}
	if len(args) > 0 {
		arguments = args
	}
	res, err := sess.cs.CallTool(cctx, &mcp.CallToolParams{Name: t.Name, Arguments: arguments})
	if err != nil {
		if errors.Is(cctx.Err(), context.DeadlineExceeded) {
			outcome = "timeout"
			return r, problem(ctx, http.StatusGatewayTimeout, CodeTimeout, "problem.timeout", i18n.Args{"server": srv.Name})
		}
		err = s.connectProblem(ctx, srv, sess.rt, err)
		outcome = problemCode(err)
		if outcome == CodeSignIn {
			s.SignedOut(ctx, srv.ID, fresh.UserID)
		}
		return r, err
	}
	text, ok := resultText(res)
	if !ok {
		outcome = "too_large"
		return r, problem(ctx, http.StatusBadGateway, CodeTooLarge, "problem.tooLarge", i18n.Args{"server": srv.Name, "kb": MaxResult >> 10})
	}
	r.Text, r.IsError = text, res.IsError
	if res.IsError {
		outcome = "tool_error"
	}
	return r, nil
}

func problemCode(err error) string {
	var p *api.Problem
	if errors.As(err, &p) {
		return p.Code
	}
	return "error"
}

// resultText is a result's text: images, audio and embedded resources are
// replaced by a note, HTML by its text. Over MaxResult, it isn't read.
func resultText(res *mcp.CallToolResult) (string, bool) {
	var b strings.Builder
	for _, c := range res.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			b.WriteString(stripHTML(v.Text))
		case *mcp.ImageContent:
			b.WriteString("[image removed]")
		case *mcp.AudioContent:
			b.WriteString("[audio removed]")
		default:
			b.WriteString("[content removed]")
		}
		b.WriteString("\n")
		if b.Len() > MaxResult {
			return "", false
		}
	}
	if b.Len() == 0 && res.StructuredContent != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		b.Write(raw)
	}
	if b.Len() > MaxResult {
		return "", false
	}
	return strings.TrimSpace(b.String()), true
}

var htmlLike = regexp.MustCompile(`(?i)<\s*/?\s*(html|head|body|div|p|span|a|img|script|style|table|tr|td|br|iframe|form|input|h[1-6]|ul|li)\b`)

// stripHTML keeps the text of HTML, without scripts and styles.
func stripHTML(s string) string {
	if !htmlLike.MatchString(s) {
		return s
	}
	z := html.NewTokenizer(strings.NewReader(s))
	var b strings.Builder
	skip := 0
	for {
		switch z.Next() {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				return strings.TrimSpace(b.String())
			}
			return strings.TrimSpace(b.String())
		case html.StartTagToken:
			if n, _ := z.TagName(); string(n) == "script" || string(n) == "style" {
				skip++
			}
		case html.EndTagToken:
			if n, _ := z.TagName(); (string(n) == "script" || string(n) == "style") && skip > 0 {
				skip--
			}
		case html.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		}
	}
}
