// Package appearance holds how a member sees the app: the light or dark
// theme they chose in Account & security › Appearance (M54). It is saved on
// the account so it follows them to every browser they sign in on; the web
// app keeps a copy per browser so the first paint is already right.
package appearance

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// The themes a member can choose. System is stored as NULL.
const (
	System = "system"
	Light  = "light"
	Dark   = "dark"
)

// Choice is what a member chose in Appearance.
type Choice struct {
	Theme string `json:"theme" enum:"system,light,dark" doc:"system follows the device's light or dark setting"`
}

// Service saves members' appearance choices.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
}

// ThemeOf returns the theme a member chose; System when they haven't.
func ThemeOf(u store.User) string {
	if u.Theme != nil && (*u.Theme == Light || *u.Theme == Dark) {
		return *u.Theme
	}
	return System
}

// Save stores a member's theme. A change is audited as
// account.appearance_update; saving the same theme again changes nothing.
func (s *Service) Save(ctx context.Context, in Choice) (Choice, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return Choice{}, api.Err(401, api.CodeUnauthenticated, "Sign in to continue.")
	}
	var theme *string
	switch in.Theme {
	case System:
	case Light, Dark:
		theme = &in.Theme
	default:
		// Huma's enum check refuses anything else first; this keeps Save safe on its own.
		return Choice{}, api.Invalid("body.theme", say(ctx, "problem.theme"))
	}
	before := ThemeOf(*u)
	if before == in.Theme {
		return Choice{Theme: before}, nil
	}
	var saved store.User
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if saved, err = q.AppearanceSetTheme(ctx, store.AppearanceSetThemeParams{ID: u.ID, Theme: theme}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.appearance_update", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Name,
			Before: map[string]any{"theme": before}, After: map[string]any{"theme": ThemeOf(saved)}})
	})
	if err != nil {
		return Choice{}, err
	}
	return Choice{Theme: ThemeOf(saved)}, nil
}
