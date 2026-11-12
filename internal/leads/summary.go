package leads

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Summary is the catalog card's facts about the workspace's connection.
type Summary struct {
	ID       uuid.UUID
	Name     string
	Pages    int
	Leads30d int64
	Polling  bool
	Problem  string
}

// Summary is the live connection's card facts; ok is false without one.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace) (Summary, bool, error) {
	c, err := s.Q.LeadConnectionLive(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, false, nil
	}
	if err != nil {
		return Summary{}, false, err
	}
	out := Summary{ID: c.ID, Name: c.Name, Polling: c.PollingReason != ""}
	if c.Status == StatusAttention {
		out.Problem = c.StatusDetail
	}
	pages, err := s.Q.LeadPagesOfConnection(ctx, c.ID)
	if err != nil {
		return out, true, err
	}
	for _, p := range pages {
		if p.Chosen {
			out.Pages++
		}
	}
	out.Leads30d, err = s.Q.LeadLeads30d(ctx, store.LeadLeads30dParams{ConnectionID: c.ID, Since: s.now().Add(-30 * 24 * time.Hour)})
	return out, true, err
}
