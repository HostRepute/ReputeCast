package agency

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/team"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Deps are the domain services the agency drives. Each is optional: a
// missing one turns its feature off with an explanation rather than
// failing the whole area (the OpenAPI command builds every service with
// nothing behind it).
type Deps struct {
	// Health scores a workspace's sending: the Overview's sender health.
	Health func(ctx context.Context, ws *store.Workspace) (HealthScore, error)
	// Invite sends invitations in a workspace (Settings › Team's invite).
	Invite func(ctx context.Context, ws *store.Workspace, p *authz.Principal, in team.TeamInviteInput) (team.TeamInviteResult, error)
	// DeleteWorkspace is the existing workspace delete: typed name, sends
	// stop at once, restorable for 7 days. It returns the restore deadline.
	DeleteWorkspace func(ctx context.Context, ws *store.Workspace, confirm string) (time.Time, error)
	// Templates saves sections and campaign templates in a client.
	Templates *templates.Service
	// Transactional creates transactional templates in a client.
	Transactional *transactional.Service
	// Channels read the usage of SMS, push and WhatsApp (X21, X22): none
	// until those channels exist, and their columns read zero.
	Channels []ChannelUsage
}

// HealthScore is a workspace's sender health as the Overview computes it.
type HealthScore struct {
	// State is ready, or pending while the workspace has no sending domain.
	State string
	Score int
	Tone  string
}

// ChannelUsage reads one channel's usage for a workspace and day: units
// (SMS segments, notifications, messages) and cost, per provider and, for
// SMS, per country.
type ChannelUsage interface {
	Channel() string
	Day(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, day time.Time) ([]UsageRow, error)
}

// UsageRow is one rolled-up usage figure.
type UsageRow struct {
	Dimension  string // provider, country or stream
	Key        string
	Label      string
	Units      int64
	CostMicros int64
}
