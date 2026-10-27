// Package demoworkspace loads the Acme sample into a separate workspace
// from the first-run Overview: "Explore with demo data" (review PR-13).
package demoworkspace

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "demoworkspace")

// say writes a message in the request's language.
func say(ctx context.Context, key string) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), nil)
}

// Service creates sample workspaces.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Now   func() time.Time
}

// DemoWorkspace is the sample workspace to open.
type DemoWorkspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	// Created is false when the caller already had one (it opens that one).
	Created bool `json:"created"`
}

// Name is the sample workspace's name; one per person.
const Name = "Acme Newsletter (demo)"

// Create makes the caller a sample workspace, or returns the one they made
// before. Its connections are demo connections: nothing is sent.
func (s *Service) Create(ctx context.Context) (DemoWorkspace, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return DemoWorkspace{}, api.Err(http.StatusUnauthorized, "unauthorized", say(ctx, "problem.signIn"))
	}
	// Like the first workspace, the demo waits for the first sign-in setup
	// (a second factor, the instance basics): it isn't a way around it.
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil && u.IsInstanceAdmin && st.SetupCompletedAt == nil {
		return DemoWorkspace{}, api.Err(http.StatusConflict, "setup_incomplete", say(ctx, "problem.setupIncomplete"))
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	var out DemoWorkspace
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var existing store.Workspace
		err := tx.QueryRow(ctx, `SELECT w.id, w.slug, w.name FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id
			WHERE m.user_id = $1 AND w.name = $2 AND w.deleted_at IS NULL ORDER BY w.created_at LIMIT 1`, u.ID, Name).Scan(&existing.ID, &existing.Slug, &existing.Name)
		if err == nil {
			out = DemoWorkspace{ID: existing.ID.String(), Slug: existing.Slug, Name: existing.Name}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		ws, err := workspaces.Create(ctx, tx, workspaces.CreateParams{Name: Name, Timezone: "America/New_York", Owner: u.ID, Source: "self"})
		if err != nil {
			return err
		}
		owner, err := q.GetUser(ctx, u.ID)
		if err != nil {
			return err
		}
		if err := demo.SampleWorkspace(ctx, tx, s.Box, ws, owner, now); err != nil {
			return err
		}
		wsID := ws.ID
		out = DemoWorkspace{ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name, Created: true}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "workspace.create", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			After: map[string]any{"name": ws.Name, "slug": ws.Slug, "sample": true}})
	})
	return out, err
}

// Register adds the endpoint.
func Register(h huma.API, s *Service) {
	api.Post(h, api.Op{ID: "create-demo-workspace", Path: "/demo-workspace", Summary: "Load the Acme sample into a separate demo workspace you own; nothing is sent",
		Tags: []string{"Workspaces"}, Access: api.Access{Kind: api.AccessUser, HumansOnly: true}},
		func(ctx context.Context, _ *struct{}) (*api.Body[DemoWorkspace], error) {
			out, err := s.Create(ctx)
			return api.OK(out), err
		})
}
