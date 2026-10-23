package accounts

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// AccountSession is one signed-in browser or device.
type AccountSession struct {
	ID      string `json:"id"`
	Device  string `json:"device" example:"MacBook Pro"`
	OS      string `json:"os" example:"macOS 26"`
	Browser string `json:"browser" example:"Chrome 141"`
	IP      string `json:"ip,omitempty"`
	// Location is approximate and only known when the instance has a local
	// geolocation source; IPs are never sent to third parties.
	Location     string    `json:"location,omitempty" example:"New York, US"`
	Method       string    `json:"method" example:"password+totp"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	Current      bool      `json:"current"`
}

// AccountSessionList is GET /me/sessions.
type AccountSessionList struct {
	Items []AccountSession `json:"items"`
}

// Sessions lists the person's active sessions, the current one first.
func (s *Service) Sessions(ctx context.Context) (AccountSessionList, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountSessionList{}, err
	}
	rows, err := s.Q.ListUserSessions(ctx, u.ID)
	if err != nil {
		return AccountSessionList{}, err
	}
	var current uuid.UUID
	if sess := api.SessionFrom(ctx); sess != nil {
		current = sess.ID
	}
	var idle time.Duration
	if st, err := s.policy(ctx); err == nil {
		idle = st.SessionIdle
	}
	out := AccountSessionList{Items: make([]AccountSession, 0, len(rows))}
	for _, r := range rows {
		// Sessions idle longer than the policy can't be used any more.
		if r.ID != current && idle > 0 && s.now().Sub(r.LastSeenAt) > idle {
			continue
		}
		item := sessionView(r, r.ID == current)
		if item.Current {
			out.Items = append([]AccountSession{item}, out.Items...)
		} else {
			out.Items = append(out.Items, item)
		}
	}
	return out, nil
}

func sessionView(r store.Session, current bool) AccountSession {
	d := ParseUserAgent(r.UserAgent)
	if r.Device != "" {
		d.Device = r.Device
	}
	if r.Os != "" {
		d.OS = r.Os
	}
	if r.Browser != "" {
		d.Browser = r.Browser
	}
	if d.Device == "" {
		d.Device = "Unknown device"
	}
	v := AccountSession{ID: r.ID.String(), Device: d.Device, OS: d.OS, Browser: d.Browser, Location: r.Location, Method: r.AuthMethod,
		CreatedAt: r.CreatedAt, LastActiveAt: r.LastSeenAt, Current: current}
	if r.Ip != nil {
		v.IP = r.Ip.Unmap().String()
	}
	return v
}

// RevokeSession signs one other session out.
func (s *Service) RevokeSession(ctx context.Context, id uuid.UUID) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	if sess := api.SessionFrom(ctx); sess != nil && sess.ID == id {
		return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, msgs.M("problem.thisSession")))
	}
	r, err := s.Q.GetUserSession(ctx, store.GetUserSessionParams{ID: id, UserID: u.ID})
	if err != nil {
		return api.MapError(err, "Session")
	}
	v := sessionView(r, false)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.RevokeSession(ctx, store.RevokeSessionParams{ID: id, UserID: u.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.session_revoke", TargetType: "session", TargetID: id.String(), TargetName: v.Device + " · " + v.Browser,
			Metadata: map[string]any{"ip": v.IP}})
	})
}

// AccountSessionsRevoked says how many sessions were signed out.
type AccountSessionsRevoked struct {
	Revoked int64 `json:"revoked"`
}

// RevokeOtherSessions signs out every session except this one.
func (s *Service) RevokeOtherSessions(ctx context.Context) (AccountSessionsRevoked, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountSessionsRevoked{}, err
	}
	sess := api.SessionFrom(ctx)
	if sess == nil {
		return AccountSessionsRevoked{}, api.Err(http.StatusUnauthorized, api.CodeUnauthenticated, "Sign in to continue.")
	}
	var n int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		n, err = q.RevokeOtherSessions(ctx, store.RevokeOtherSessionsParams{UserID: u.ID, KeepID: sess.ID})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.sessions_revoke_others", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Metadata: map[string]any{"revoked": n}})
	})
	return AccountSessionsRevoked{Revoked: n}, err
}
