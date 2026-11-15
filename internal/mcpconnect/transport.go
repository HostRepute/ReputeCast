package mcpconnect

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

type connectionCall struct {
	Grant     store.McpConnectGrant
	Principal *authz.Principal
	TokenHash []byte
	Request   audit.Request
}

func (s *Service) verify(ctx context.Context, raw string, r *http.Request) (*auth.TokenInfo, error) {
	if !strings.HasPrefix(raw, accessPrefix) || len(r.Header.Values("Authorization")) != 1 || r.URL.RawQuery != "" {
		return nil, auth.ErrInvalidToken
	}
	var out *auth.TokenInfo
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '1s'; SET LOCAL statement_timeout = '5s'"); err != nil {
			return err
		}
		q := store.New(tx)
		t, err := q.MCPConnectToken(ctx, authn.HashToken(raw))
		if err != nil || t.Kind != "access" || !t.ExpiresAt.After(s.now()) {
			return auth.ErrInvalidToken
		}
		g, err := q.MCPConnectGrantRead(ctx, t.GrantID)
		if err != nil {
			return auth.ErrInvalidToken
		}
		p, err := s.grantPrincipal(ctx, q, g, audit.RequestFrom(ctx).IP)
		if err != nil {
			return auth.ErrInvalidToken
		}
		out = &auth.TokenInfo{UserID: g.ID.String(), Scopes: g.Scopes, Expiration: t.ExpiresAt, Extra: map[string]any{"connection": &connectionCall{Grant: g, Principal: p, TokenHash: t.Hash, Request: audit.RequestFrom(ctx)}}}
		return nil
	})
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	return out, nil
}

func (s *Service) Handler() http.Handler {
	// Stateless transport does not let a cached session hold old authorization.
	// No prompts, tool arguments or SDK errors are sent to application logs.
	protocol := mcp.NewServer(&mcp.Implementation{Name: "reputecast-connect", Version: version.Get().Version}, &mcp.ServerOptions{
		Instructions: "This external connection uses only its granted editorial capabilities. Draft proposals require a person to review in ReputeCast. No tool sends, publishes or uses the workspace's AI model.",
	})
	mcp.AddTool(protocol, &mcp.Tool{Name: "connection_status", Description: "Read this connection's currently usable capabilities and absolute expiry. Returns no workspace content."}, func(ctx context.Context, r *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		out := struct {
			Scopes    []string  `json:"scopes"`
			ExpiresAt time.Time `json:"expiresAt"`
		}{Scopes: []string{}}
		err := s.call(ctx, r.Extra, "", func(_ context.Context, _ *store.Queries, c *connectionCall) error {
			out.ExpiresAt = c.Grant.ExpiresAt
			for _, scope := range c.Grant.Scopes {
				if canScope(scope, c.Principal.Permissions) {
					out.Scopes = append(out.Scopes, scope)
				}
			}
			return nil
		})
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Connection access changed. Reconnect or review permissions in ReputeCast."}}}, nil, nil //nolint:nilerr // The MCP tool result carries the failure without disclosing authorization details.
		}
		return nil, out, nil
	})
	s.registerTools(protocol)
	stream := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return protocol }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	admit := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		info := auth.TokenInfoFromContext(r.Context())
		c := info.Extra["connection"].(*connectionCall)
		_, err := s.Q.MCPConnectLimit(r.Context(), store.MCPConnectLimitParams{Key: authn.HashToken("mcp:" + c.Grant.ID.String()), Bucket: s.now().Truncate(time.Minute), Ceiling: 120})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				w.Header().Set("Retry-After", "60")
				writeJSON(w, 429, map[string]string{"error": "temporarily_unavailable"})
			} else {
				oauthError(w, err)
			}
			return
		}
		lease := uuid.Must(uuid.NewV7())
		now := s.now()
		_, err = s.Q.MCPConnectClaimSlot(r.Context(), store.MCPConnectClaimSlotParams{GrantID: c.Grant.ID, LeaseID: lease, Now: now, ExpiresAt: now.Add(30 * time.Second)})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				w.Header().Set("Retry-After", "1")
				writeJSON(w, 429, map[string]string{"error": "temporarily_unavailable"})
			} else {
				oauthError(w, err)
			}
			return
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
			defer cancel()
			_ = s.Q.MCPConnectReleaseSlot(ctx, store.MCPConnectReleaseSlotParams{GrantID: c.Grant.ID, LeaseID: lease})
		}()
		// No SSE or server-initiated calls: the complete bounded JSON response
		// is checked before any of its bytes reach the external client.
		out := &boundedResponse{header: w.Header().Clone()}
		stream.ServeHTTP(out, r)
		if out.overflow {
			oauthError(w, errors.New("response_limit"))
			return
		}
		for key, values := range out.header {
			w.Header()[key] = values
		}
		if out.status == 0 {
			out.status = http.StatusOK
		}
		w.WriteHeader(out.status)
		_, _ = w.Write(out.body.Bytes())
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		origin, err := s.origin()
		if err != nil {
			oauthError(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		// A context deadline does not unblock a socket stalled mid-body.
		control := http.NewResponseController(w)
		_ = control.SetReadDeadline(time.Now().Add(20 * time.Second))
		defer func() { _ = control.SetReadDeadline(time.Time{}) }()
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
		auth.RequireBearerToken(s.verify, &auth.RequireBearerTokenOptions{ResourceMetadataURL: origin + "/.well-known/oauth-protected-resource/mcp/connect"})(admit).ServeHTTP(w, r)
	})
}

// Each tool rechecks access under the grant and authority locks and does all
// database work in this transaction. It cannot borrow the human's identity.
func (s *Service) call(ctx context.Context, extra *mcp.RequestExtra, scope string, fn func(context.Context, *store.Queries, *connectionCall) error) error {
	if extra == nil || extra.TokenInfo == nil {
		return errGrant
	}
	c, ok := extra.TokenInfo.Extra["connection"].(*connectionCall)
	if !ok {
		return errGrant
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '1s'; SET LOCAL statement_timeout = '5s'"); err != nil {
			return err
		}
		q := store.New(tx)
		g, err := q.MCPConnectGrantLock(ctx, c.Grant.ID)
		if err != nil {
			return errGrant
		}
		t, err := q.MCPConnectToken(ctx, c.TokenHash)
		if err != nil || !t.ExpiresAt.After(s.now()) {
			return errGrant
		}
		p, err := s.grantPrincipal(ctx, q, g, c.Request.IP)
		if err != nil {
			return err
		}
		if scope != "" && (!canScope(scope, p.Permissions) || !containsScope(g.Scopes, scope)) {
			return errors.New("insufficient_scope")
		}
		ctx = api.WithPrincipal(audit.WithRequest(ctx, c.Request), p)
		if err := fn(ctx, q, &connectionCall{Grant: g, Principal: p, TokenHash: c.TokenHash, Request: c.Request}); err != nil {
			return err
		}
		if !t.ExpiresAt.After(s.now()) || !g.ExpiresAt.After(s.now()) {
			return errGrant
		}
		return nil
	})
}

func containsScope(scopes []string, value string) bool {
	for _, scope := range scopes {
		if scope == value {
			return true
		}
	}
	return false
}

type boundedResponse struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	overflow bool
}

func (w *boundedResponse) Header() http.Header { return w.header }
func (w *boundedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *boundedResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.overflow || w.body.Len()+len(b) > 1<<20 {
		w.overflow = true
		return 0, errors.New("response_limit")
	}
	return w.body.Write(b)
}
