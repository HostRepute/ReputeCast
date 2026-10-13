// Package sessionext tells the web app when the current browser session will
// end and lets the person extend it, so the app can warn two minutes before
// an idle timeout ("Session ending in 2 minutes · Extend 12 hours") instead
// of losing unsaved work.
//
// A session ends at the earlier of two times: its idle deadline (last
// activity plus the instance's idle timeout, 12 hours by default) and its
// absolute lifetime (14 days by default). Extending restarts the idle clock;
// the lifetime never moves, so an instance's policy can't be bypassed.
package sessionext

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service reads and extends the caller's session.
type Service struct {
	Pool  *pgxpool.Pool
	Q     store.Querier
	Audit *audit.Logger
	// Policy returns the instance authentication policy (the idle timeout).
	Policy func(context.Context) (api.Policy, error)
	Now    func() time.Time
}

// SessionTiming says when the caller's session ends and what extending does.
type SessionTiming struct {
	SignedInAt time.Time `json:"signedInAt" doc:"When this session was created"`
	LastSeenAt time.Time `json:"lastSeenAt" doc:"Last activity the server recorded; the idle clock runs from here"`
	// IdleTimeoutSeconds is 0 when the instance has no idle timeout.
	IdleTimeoutSeconds int64      `json:"idleTimeoutSeconds" doc:"The instance's idle timeout, in seconds"`
	IdleExpiresAt      *time.Time `json:"idleExpiresAt,omitempty" doc:"When the session ends without further activity"`
	ExpiresAt          time.Time  `json:"expiresAt" doc:"The session's absolute lifetime; extending never moves it"`
	EndsAt             time.Time  `json:"endsAt" doc:"The earlier of idleExpiresAt and expiresAt"`
	EndsBy             string     `json:"endsBy" enum:"idle,lifetime" doc:"Which limit ends the session first"`
	Extendable         bool       `json:"extendable" doc:"Whether extending now would end the session later"`
	ExtendsTo          *time.Time `json:"extendsTo,omitempty" doc:"When the session would end after extending now"`
	ServerTime         time.Time  `json:"serverTime" doc:"The server's clock, to correct for skew in the browser"`
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) idle(ctx context.Context) time.Duration {
	if s.Policy == nil {
		return 0
	}
	pol, err := s.Policy(ctx)
	if err != nil {
		return 0
	}
	return pol.SessionIdle
}

// timing computes the end of a session from its row and the idle timeout.
func timing(signedIn, lastSeen, expires time.Time, idle time.Duration, now time.Time) SessionTiming {
	t := SessionTiming{
		SignedInAt:         signedIn,
		LastSeenAt:         lastSeen,
		IdleTimeoutSeconds: int64(idle / time.Second),
		ExpiresAt:          expires,
		EndsAt:             expires,
		EndsBy:             "lifetime",
		ServerTime:         now,
	}
	if idle <= 0 {
		return t
	}
	idleEnd := lastSeen.Add(idle)
	t.IdleExpiresAt = &idleEnd
	if idleEnd.Before(expires) {
		t.EndsAt = idleEnd
		t.EndsBy = "idle"
	}
	to := now.Add(idle)
	if to.After(expires) {
		to = expires
	}
	t.ExtendsTo = &to
	// Worth offering only when it moves the end by at least a minute.
	t.Extendable = to.Sub(t.EndsAt) >= time.Minute
	return t
}

func current(ctx context.Context) (*store.Session, error) {
	sess := api.SessionFrom(ctx)
	if sess == nil {
		return nil, api.Err(http.StatusUnauthorized, api.CodeUnauthenticated, "Sign in to continue.")
	}
	return sess, nil
}

// Timing returns when the caller's session ends.
func (s *Service) Timing(ctx context.Context) (SessionTiming, error) {
	sess, err := current(ctx)
	if err != nil {
		return SessionTiming{}, err
	}
	// Re-read: the middleware may have just recorded this request as activity.
	row, err := s.Q.SessionExtTiming(ctx, sess.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionTiming{}, api.Err(http.StatusUnauthorized, api.CodeSessionExpired, "Your session ended. Sign in again; unsaved work in this tab is kept.")
	}
	if err != nil {
		return SessionTiming{}, err
	}
	return timing(row.CreatedAt, row.LastSeenAt, row.ExpiresAt, s.idle(ctx), s.now()), nil
}

// Extend restarts the idle clock of the caller's session. The change and its
// audit entry commit together.
func (s *Service) Extend(ctx context.Context) (SessionTiming, error) {
	sess, err := current(ctx)
	if err != nil {
		return SessionTiming{}, err
	}
	idle := s.idle(ctx)
	var t SessionTiming
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.SessionExtRestartIdle(ctx, store.SessionExtRestartIdleParams{ID: sess.ID, UserID: sess.UserID})
		if err != nil {
			return err
		}
		t = timing(row.CreatedAt, row.LastSeenAt, row.ExpiresAt, idle, s.now())
		return s.Audit.LogWith(ctx, q, audit.Event{
			Action:     "session.extend",
			TargetType: "session",
			TargetID:   sess.ID.String(),
			Metadata:   map[string]any{"endsAt": t.EndsAt.UTC().Format(time.RFC3339), "endsBy": t.EndsBy},
		})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionTiming{}, api.Err(http.StatusUnauthorized, api.CodeSessionExpired, "Your session ended. Sign in again; unsaved work in this tab is kept.")
	}
	return t, err
}
