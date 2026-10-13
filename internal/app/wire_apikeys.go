package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apikeys"
	"github.com/hostrepute/reputecast/internal/audit"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		apikeys.Register(h, &apikeys.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit,
			PublicURL: func(context.Context) string { return a.URLs.Base() },
		})
	})
	// Calls with a revoked or expired key are counted, so the keys screen can
	// show what still uses them (the MCP endpoint records its own).
	startHooks = append(startHooks, func(a *App) error {
		a.Auth.OnKeyRejected = func(ctx context.Context, id uuid.UUID, reason string) {
			req := audit.RequestFrom(ctx)
			apikeys.RecordRejection(ctx, a.Q, id, req.IP, req.UserAgent, reason)
		}
		return nil
	})
}
