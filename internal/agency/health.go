package agency

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// HostRepute's view of a client: clean when it's connected, its data is
// fresh and no monitored asset is listed.
func hostReputeOf(ctx context.Context, q *store.Queries, wsID uuid.UUID) (string, int) {
	c, err := q.GetHostReputeConnection(ctx, wsID)
	if err != nil {
		return "", 0
	}
	if (c.Status != "connected" && c.Status != "limit_reached") || c.LastSyncError != "" || c.TokenRejectedAt != nil {
		return "stale", 0
	}
	var assets []struct {
		Listings int `json:"listings"`
	}
	_ = json.Unmarshal(c.Monitored, &assets)
	listed := 0
	for _, a := range assets {
		listed += a.Listings
	}
	if listed > 0 {
		return "listed", listed
	}
	return "clean", 0
}

// refreshHealth scores a workspace now, as its Overview does, and keeps
// today's snapshot. The demo's snapshots are kept as they are.
func (s *Service) refreshHealth(ctx context.Context, wsID uuid.UUID) (*AgencyHealth, error) {
	d := s.deps()
	if d.Health == nil {
		return nil, nil
	}
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	h, err := d.Health(ctx, &ws)
	if err != nil {
		return nil, err
	}
	hr, listed := hostReputeOf(ctx, s.Q, wsID)
	now := s.now()
	if err := s.Q.AgencyHealthPut(ctx, store.AgencyHealthPutParams{WorkspaceID: wsID, Day: dateOnly(now), State: h.State, Score: int32(h.Score), Tone: h.Tone,
		Hostrepute: hr, Listings: int32(listed), ComputedAt: now}); err != nil {
		return nil, err
	}
	out := &AgencyHealth{State: h.State, Score: h.Score, Tone: h.Tone, Trend: []int{}, HostRepute: hr, Listings: listed, CheckedAt: now}
	if h.State == "ready" {
		out.Trend = append(out.Trend, h.Score)
	}
	return out, nil
}

// RefreshAll scores every workspace of the agency (the hourly job), and
// rolls up their usage for yesterday and today.
func (s *Service) RefreshAll(ctx context.Context) error {
	st, err := s.settings(ctx, s.Q)
	if err != nil || !st.Enabled {
		return err
	}
	rows, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, r := range rows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err := s.refreshHealth(ctx, r.ID); err != nil {
			return err
		}
		for _, day := range []time.Time{dateOnly(now).AddDate(0, 0, -1), dateOnly(now)} {
			if err := s.rollupDay(ctx, r.ID, day); err != nil {
				return err
			}
		}
	}
	return nil
}
