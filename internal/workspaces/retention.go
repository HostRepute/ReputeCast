package workspaces

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// WorkspaceRetention is Settings › General › Data retention (review
// 2026-11 DP-4, DP-6, decision D3).
type WorkspaceRetention struct {
	// UnconfirmedEraseDays erases double opt-in signups never confirmed
	// after this many days; 0 never does.
	UnconfirmedEraseDays int32 `json:"unconfirmedEraseDays" enum:"0,14,30,60,90"`
	// EventDetailDays clears the IP address and user agent of opens and
	// clicks in campaigns and automations after this many days; 0 keeps
	// them. Counts, scores and reports stay.
	EventDetailDays int32 `json:"eventDetailDays" enum:"0,30,90,180,365,730"`
}

var (
	unconfirmedChoices = []int32{0, 14, 30, 60, 90}
	eventDetailChoices = []int32{0, 30, 90, 180, 365, 730}
)

// retentionOf is a workspace's retention settings.
func retentionOf(ws *store.Workspace) WorkspaceRetention {
	return WorkspaceRetention{UnconfirmedEraseDays: ws.UnconfirmedEraseDays, EventDetailDays: ws.EventDetailDays}
}

// UpdateRetention saves the data retention settings, audited.
func (s *Service) UpdateRetention(ctx context.Context, ws *store.Workspace, in WorkspaceRetention) error {
	if !slices.Contains(unconfirmedChoices, in.UnconfirmedEraseDays) {
		return api.Invalid("body.unconfirmedEraseDays", say(ctx, "problem.retention", nil))
	}
	if !slices.Contains(eventDetailChoices, in.EventDetailDays) {
		return api.Invalid("body.eventDetailDays", say(ctx, "problem.retention", nil))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WsUpdateRetention(ctx, store.WsUpdateRetentionParams{ID: ws.ID, UnconfirmedEraseDays: in.UnconfirmedEraseDays, EventDetailDays: in.EventDetailDays}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"retention": retentionOf(ws)}, After: map[string]any{"retention": in},
			Metadata: map[string]any{"permission": string(authz.WorkspaceManage)}})
	})
}
