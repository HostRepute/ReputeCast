package referrals

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Leaderboard periods.
const (
	PeriodMonth = "month"
	PeriodAll   = "all"
)

// RefreshLeaderboard recomputes a program's leaderboard: every referrer
// with a counted referral, this month and all time, by counted referrals;
// ties share a rank. The pages show the top rows and a referrer's own rank.
func (s *Service) RefreshLeaderboard(ctx context.Context, programID uuid.UUID) error {
	p, err := s.Q.ReferralProgramByID(ctx, programID)
	if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	now := s.now()
	loc := workspaceLocation(&ws)
	l := now.In(loc)
	monthStart := time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, loc)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralLeaderboardClear(ctx, programID); err != nil {
			return err
		}
		for _, period := range []string{PeriodMonth, PeriodAll} {
			since := time.Time{}
			if period == PeriodMonth {
				since = monthStart
			}
			rows, err := tx.Query(ctx, `SELECT link_id, count(*)::int AS counted FROM referrals
				WHERE program_id = $1 AND status = 'counted' AND link_id IS NOT NULL AND counted_at >= $2
				GROUP BY link_id ORDER BY counted DESC, min(counted_at)`, programID, since)
			if err != nil {
				return err
			}
			type row struct {
				link    uuid.UUID
				counted int32
			}
			var list []row
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.link, &r.counted); err != nil {
					rows.Close()
					return err
				}
				list = append(list, r)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			rank := 0
			for i, r := range list {
				if i == 0 || r.counted != list[i-1].counted {
					rank = i + 1
				}
				if err := q.ReferralLeaderboardInsert(ctx, store.ReferralLeaderboardInsertParams{ProgramID: programID, Period: period, LinkID: r.link,
					Rank: int32(rank), Counted: r.counted, ComputedAt: now}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// RefreshLeaderboards recomputes the leaderboard of every program that has
// it on (the hourly job).
func (s *Service) RefreshLeaderboards(ctx context.Context) error {
	progs, err := s.Q.ReferralProgramsActive(ctx)
	if err != nil {
		return err
	}
	for _, p := range progs {
		if !p.LeaderboardOn {
			continue
		}
		if err := s.RefreshLeaderboard(ctx, p.ID); err != nil {
			s.log().WarnContext(ctx, "referrals: refresh the leaderboard", "program", p.ID, "err", err)
		}
	}
	return nil
}

// LeaderboardRow is one row of the public leaderboard.
type LeaderboardRow struct {
	Rank    int
	Name    string // "Omar H.", or "" for "A reader"
	Counted int
	Me      bool
}

// LeaderboardView is a period of the leaderboard as the public page shows it.
type LeaderboardView struct {
	Period     string
	Rows       []LeaderboardRow
	More       int // readers beyond the top rows with the last row's count
	MoreCount  int
	Me         *LeaderboardRow
	ComputedAt *time.Time
}

// Leaderboard reads a period of a program's leaderboard; me (a hub's link)
// is the reader's own row, highlighted even outside the top rows.
func (s *Service) Leaderboard(ctx context.Context, p store.ReferralProgram, period string, me *uuid.UUID) (LeaderboardView, error) {
	out := LeaderboardView{Period: period, Rows: []LeaderboardRow{}}
	rows, err := s.Pool.Query(ctx, `SELECT b.link_id, b.rank, b.counted, b.computed_at, rl.show_leaderboard, c.first_name, c.last_name
		FROM referral_leaderboard b JOIN referral_links rl ON rl.id = b.link_id JOIN contacts c ON c.id = rl.contact_id
		WHERE b.program_id = $1 AND b.period = $2 ORDER BY b.rank, b.counted DESC, b.link_id`, p.ID, period)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	limit := int(p.LeaderboardRows)
	if limit <= 0 {
		limit = 10
	}
	var lastCounted int32
	for rows.Next() {
		var link uuid.UUID
		var rank, counted int32
		var at time.Time
		var show bool
		var first, last string
		if err := rows.Scan(&link, &rank, &counted, &at, &show, &first, &last); err != nil {
			return out, err
		}
		if out.ComputedAt == nil {
			t := at
			out.ComputedAt = &t
		}
		r := LeaderboardRow{Rank: int(rank), Counted: int(counted), Me: me != nil && *me == link}
		if show {
			r.Name = shortName(first, last)
		}
		switch {
		case len(out.Rows) < limit:
			out.Rows = append(out.Rows, r)
			lastCounted = counted
		case counted == lastCounted:
			out.More++
			out.MoreCount = int(counted)
		}
		if r.Me {
			mine := r
			out.Me = &mine
		}
	}
	return out, rows.Err()
}
