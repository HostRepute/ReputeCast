package apptest

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// What the membership paths remove with a person (assistant brief D4,
// §3.7): their assistant threads in a workspace and their sign-ins to its
// MCP servers. Placeholder rows for the paths' tests, without content.

// SeedAssistantThread adds an empty assistant thread of a member and
// returns its id.
func (e *Env) SeedAssistantThread(workspace, user uuid.UUID) uuid.UUID {
	e.T.Helper()
	id, now := uuid.Must(uuid.NewV7()), time.Now()
	if _, err := e.App.Pool.Exec(context.Background(), `INSERT INTO ai_assistant_threads (id, workspace_id, user_id, tool_limit, token_limit, cost_limit_micros,
		created_at, updated_at, last_activity_at, expires_at) VALUES ($1, $2, $3, 40, 300000, 2000000, $4, $4, $4, $5)`, id, workspace, user, now, now.Add(30*24*time.Hour)); err != nil {
		e.T.Fatal(err)
	}
	return id
}

// SeedMCPSignIn adds a remote MCP server to a workspace (one per workspace)
// and a member's sign-in to it, its tokens placeholders.
func (e *Env) SeedMCPSignIn(workspace, user uuid.UUID) {
	e.T.Helper()
	ctx := context.Background()
	if _, err := e.App.Pool.Exec(ctx, `INSERT INTO mcp_servers (id, workspace_id, key, name, url, auth, created_at, updated_at)
		VALUES ($1, $2, 'seeded', 'Seeded', 'https://mcp.example/mcp', 'oauth', now(), now()) ON CONFLICT (workspace_id, key) DO NOTHING`, uuid.Must(uuid.NewV7()), workspace); err != nil {
		e.T.Fatal(err)
	}
	if _, err := e.App.Pool.Exec(ctx, `INSERT INTO mcp_tokens (server_id, user_id, workspace_id, access_sealed, created_at, updated_at)
		SELECT id, $2, $1, '\x00'::bytea, now(), now() FROM mcp_servers WHERE workspace_id = $1 AND key = 'seeded'`, workspace, user); err != nil {
		e.T.Fatal(err)
	}
}

// MemberLeftovers counts a person's assistant threads and MCP sign-ins in
// a workspace.
func (e *Env) MemberLeftovers(workspace, user uuid.UUID) (threads, signIns int) {
	e.T.Helper()
	if err := e.App.Pool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM ai_assistant_threads WHERE workspace_id = $1 AND user_id = $2),
		(SELECT count(*) FROM mcp_tokens WHERE workspace_id = $1 AND user_id = $2)`, workspace, user).Scan(&threads, &signIns); err != nil {
		e.T.Fatal(err)
	}
	return threads, signIns
}
