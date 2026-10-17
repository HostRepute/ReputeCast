package digests

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// minLead is how far ahead an issue's send time has to be for it to be the
// next one: a digest turned on at 07:59 doesn't send at 08:00.
const minLead = 15 * time.Minute

// nextTimes are the send, preparation and late-limit times of the next
// issue after t. An item-count schedule has none until it prepares.
func (s *Service) nextTimes(_ context.Context, d store.Digest, t time.Time) (sendAt, prepareAt, deadline *time.Time) {
	sched := ParseSchedule(d.Schedule, "UTC")
	if sched.Frequency == FreqItems {
		return nil, nil, nil
	}
	after := t
	if now := s.now().Add(minLead); now.After(after) {
		after = now
	}
	send := sched.NextSend(after)
	if send.IsZero() {
		return nil, nil, nil
	}
	prep := send.Add(-time.Duration(d.PrepareHours) * time.Hour)
	late := send.Add(time.Duration(d.LateHours) * time.Hour)
	return &send, &prep, &late
}

// ensureCollecting starts collecting the next issue of a digest that is on
// and has none in flight. Its times follow the schedule after the last
// issue's send time (or now).
func (s *Service) ensureCollecting(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	d, err := s.Q.DigestByID(ctx, id)
	if err != nil {
		return err
	}
	if d.State != "on" {
		return nil
	}
	if _, err := s.Q.DigestIssueOpen(ctx, id); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	after := s.now()
	if last, err := s.lastSettled(ctx, id); err == nil && last != nil && last.SendAt != nil && last.SendAt.After(after) {
		after = *last.SendAt
	}
	sendAt, prepareAt, deadline := s.nextTimes(ctx, d, after)
	err = s.Q.DigestIssueInsert(ctx, store.DigestIssueInsertParams{ID: uuid.Must(uuid.NewV7()), DigestID: id, WorkspaceID: d.WorkspaceID, Mode: d.Mode,
		SendAt: sendAt, PrepareAt: prepareAt, Deadline: deadline, Now: s.now()})
	if err != nil && isUnique(err) {
		return nil // another process started it
	}
	return err
}

// lastSettled is the newest issue that went out or was settled.
func (s *Service) lastSettled(ctx context.Context, id uuid.UUID) (*store.DigestIssuesRow, error) {
	rows, err := s.Q.DigestIssues(ctx, store.DigestIssuesParams{DigestID: id, Lim: 5})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		switch rows[i].Status {
		case "sending", "sent", "skipped", "not_sent":
			return &rows[i], nil
		}
	}
	return nil, nil
}
