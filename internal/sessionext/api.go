package sessionext

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
)

// Browser sessions only: API keys and agents have no session to extend.
// Pending users (temporary password, required two-factor) may call these too.
var sessionAccess = api.Access{Kind: api.AccessUser, AllowPending: true, HumansOnly: true}

// Register adds the session timing operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Account"}
	api.Get(h, api.Op{
		ID: "get-session-timing", Path: "/me/session", Tags: tags, Access: sessionAccess,
		Summary:     "When your session ends",
		Description: "The idle deadline, the absolute lifetime and what extending would do. Like every request, reading it counts as activity.",
	}, func(ctx context.Context, _ *struct{}) (*api.Body[SessionTiming], error) {
		t, err := s.Timing(ctx)
		return api.OK(t), err
	})

	api.Post(h, api.Op{
		ID: "extend-session", Path: "/me/session/extend", Tags: tags, Access: sessionAccess,
		Summary:     "Extend your session",
		Description: "Restarts the idle timeout now. The session's absolute lifetime never moves. Recorded in the audit log as session.extend.",
	}, func(ctx context.Context, _ *struct{}) (*api.Body[SessionTiming], error) {
		t, err := s.Extend(ctx)
		return api.OK(t), err
	})
}
