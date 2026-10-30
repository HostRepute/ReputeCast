// Package agency implements agency accounts (design expansion phase 8,
// M32; channels and agencies brief §3.10): an agency runs one instance for
// its clients, each in an ordinary workspace of its own.
//
//   - Agency mode, turned on in Instance admin › Instance & updates (one
//     agency per instance).
//   - Staff at instance level: agency managers (every client) and account
//     managers (the clients they're assigned to). Instance admins are agency
//     managers.
//   - Assignments are ordinary workspace memberships (source "agency"),
//     behind the escalation guard: you can assign only roles whose
//     permissions you hold in that workspace.
//   - Clients, their account manager, rebill rates and usage per month.
//   - Workspace templates and the shared library, copied into clients.
//   - The agency overview and search across clients.
//
// Nothing here moves data between clients: each keeps its own providers,
// domains, contacts, suppressions, consent, keys and audit log, and copies
// from the library or a template are new rows in the client's workspace.
package agency

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "agency")

// Service is the agency's API and jobs.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
	// Deps are the domain services the agency drives (invitations, copies
	// into clients, sender health, deleting a client). Built lazily by the
	// wiring so the services that reference each other can be constructed.
	Deps func() Deps
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) deps() Deps {
	if s.Deps == nil {
		return Deps{}
	}
	return s.Deps()
}

// Staff is the signed-in member of the agency's staff.
type Staff struct {
	UserID        uuid.UUID
	Name          string
	Email         string
	Level         string
	Title         string
	InstanceAdmin bool
	Permissions   authz.Set
}

// Can reports whether the staff member holds an agency permission.
func (st *Staff) Can(p authz.Permission) bool { return st != nil && st.Permissions.Has(p) }

// Manager reports whether they work across every client.
func (st *Staff) Manager() bool { return st != nil && st.Level == authz.AgencyManagerLevel }

// settings is agency mode as stored.
type settings struct {
	Enabled           bool
	Name              string
	DefaultTemplateID *uuid.UUID
	EnabledBy         *uuid.UUID
	EnabledByName     string
	EnabledAt         *time.Time
	InstanceName      string
	PublicURL         string
}

func (s *Service) settings(ctx context.Context, q *store.Queries) (settings, error) {
	r, err := q.AgencySettings(ctx)
	if err != nil {
		return settings{}, err
	}
	st := settings{Enabled: r.Enabled, Name: strings.TrimSpace(r.Name), DefaultTemplateID: r.DefaultTemplateID, EnabledBy: r.EnabledBy,
		EnabledByName: r.EnabledByName, EnabledAt: r.EnabledAt, InstanceName: r.InstanceName, PublicURL: r.InstancePublicUrl}
	if st.Name == "" {
		st.Name = r.InstanceName
	}
	return st, nil
}

// host is the instance's address without the scheme ("app.northwind.studio").
func host(publicURL string) string {
	h := strings.TrimPrefix(strings.TrimPrefix(publicurl.Effective(publicURL, ""), "https://"), "http://")
	return strings.TrimRight(h, "/")
}

// staffOf resolves a person's agency level while agency mode is on. It
// binds a row waiting for their email (invited by email) to their account.
func (s *Service) staffOf(ctx context.Context, u *store.User) (*Staff, error) {
	row, err := s.Q.AgencyStaffOf(ctx, store.AgencyStaffOfParams{UserID: &u.ID, Email: strings.ToLower(u.Email)})
	switch {
	case err == nil:
		if row.UserID == nil {
			if err := s.Q.AgencyBindStaff(ctx, store.AgencyBindStaffParams{UserID: &u.ID, Email: row.Email}); err != nil {
				return nil, err
			}
		}
	case isNoRows(err):
		if !u.IsInstanceAdmin {
			return nil, nil
		}
	default:
		return nil, err
	}
	st := &Staff{UserID: u.ID, Name: displayName(u.Name, u.Email), Email: u.Email, InstanceAdmin: u.IsInstanceAdmin}
	if err == nil {
		st.Level, st.Title = row.Level, row.Title
	}
	if u.IsInstanceAdmin {
		st.Level = authz.AgencyManagerLevel
	}
	r, _ := authz.Agency(st.Level)
	st.Permissions = r.Permissions
	// A narrowed row holds fewer of its level's permissions (never more,
	// never fewer than the overview); instance admins hold them all.
	if err == nil && row.Permissions != nil && !u.IsInstanceAdmin {
		keep := []authz.Permission{authz.AgencyView}
		for _, p := range row.Permissions {
			if r.Permissions.Has(authz.Permission(p)) {
				keep = append(keep, authz.Permission(p))
			}
		}
		st.Permissions = authz.AgencySet(keep...)
	}
	return st, nil
}

// staff returns the caller as agency staff, or the problem that keeps them
// out: agency mode off, or not on the agency's staff.
func (s *Service) staff(ctx context.Context) (*Staff, settings, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return nil, settings{}, api.Err(http.StatusUnauthorized, api.CodeUnauthenticated, "Sign in to continue.")
	}
	st, err := s.settings(ctx, s.Q)
	if err != nil {
		return nil, st, err
	}
	l := i18n.LangFrom(ctx)
	if !st.Enabled {
		return nil, st, api.Err(http.StatusForbidden, CodeAgencyOff, mAgencyOff.Text(l, nil))
	}
	me, err := s.staffOf(ctx, u)
	if err != nil {
		return nil, st, err
	}
	if me == nil {
		return nil, st, api.Err(http.StatusForbidden, api.CodePermissionDenied, mNotStaff.Text(l, i18n.Args{"agency": st.Name}))
	}
	return me, st, nil
}

// require returns the caller as staff holding perm.
func (s *Service) require(ctx context.Context, perm authz.Permission) (*Staff, settings, error) {
	me, st, err := s.staff(ctx)
	if err != nil {
		return nil, st, err
	}
	if !me.Can(perm) {
		return me, st, s.denied(ctx, perm)
	}
	return me, st, nil
}

// CodeAgencyOff is the problem code while agency mode is off.
const CodeAgencyOff = "agency_off"

// denied says, in plain words, what a missing agency permission keeps the
// reader from doing, and names the agency managers who hold it.
func (s *Service) denied(ctx context.Context, perm authz.Permission) *api.Problem {
	helpers := s.managerNames(ctx, 2)
	l := i18n.LangFrom(ctx)
	args := i18n.Args{"what": deniedWhat(l, perm)}
	msg := mDenied
	if len(helpers) > 0 {
		args["helpers"] = i18n.OrList(l.Locale, helpers)
		msg = mDeniedAsk
	}
	p := api.Err(http.StatusForbidden, api.CodePermissionDenied, msg.Text(l, args))
	p.Permission, p.Helpers = string(perm), helpers
	return p
}

func deniedWhat(l i18n.Lang, perm authz.Permission) string {
	switch perm {
	case authz.AgencyUsage:
		return mWhatUsage.Text(l, nil)
	case authz.AgencyLibrary:
		return mWhatLibrary.Text(l, nil)
	case authz.AgencyManage:
		return mWhatManage.Text(l, nil)
	}
	return mWhatView.Text(l, nil)
}

// managerNames are the active agency managers, instance admins first ("Ask
// Tomás Herrera or Nadia Okafor": the board names the newest first).
func (s *Service) managerNames(ctx context.Context, n int) []string {
	people, err := s.people(ctx)
	if err != nil {
		return nil
	}
	var out []string
	for i := len(people) - 1; i >= 0; i-- {
		p := people[i]
		if p.Level == authz.AgencyManagerLevel && p.Active && p.UserID != nil {
			out = append(out, p.Name)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

// workspaceSet are the agency's workspaces a staff member works in: every
// one for agency managers, the ones they're assigned to for account
// managers.
type workspaceSet struct {
	all     []store.AgencyWorkspacesRow
	mine    map[uuid.UUID]store.AgencyMembershipsRow
	visible []store.AgencyWorkspacesRow
}

func (w workspaceSet) ids() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(w.visible))
	for _, r := range w.visible {
		out = append(out, r.ID)
	}
	return out
}

func (w workspaceSet) find(id uuid.UUID) (store.AgencyWorkspacesRow, bool) {
	for _, r := range w.visible {
		if r.ID == id {
			return r, true
		}
	}
	return store.AgencyWorkspacesRow{}, false
}

// permsIn is what the staff member may do in one of the workspaces: the
// role they hold there.
func (w workspaceSet) permsIn(id uuid.UUID) authz.Set {
	m, ok := w.mine[id]
	if !ok {
		return authz.Set{}
	}
	return authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)
}

func (s *Service) workspacesOf(ctx context.Context, q *store.Queries, me *Staff) (workspaceSet, error) {
	all, err := q.AgencyWorkspaces(ctx)
	if err != nil {
		return workspaceSet{}, err
	}
	ms, err := q.AgencyMemberships(ctx, []uuid.UUID{me.UserID})
	if err != nil {
		return workspaceSet{}, err
	}
	set := workspaceSet{all: all, mine: map[uuid.UUID]store.AgencyMembershipsRow{}}
	for _, m := range ms {
		set.mine[m.WorkspaceID] = m
	}
	for _, r := range all {
		if _, ok := set.mine[r.ID]; ok || me.Manager() {
			set.visible = append(set.visible, r)
		}
	}
	return set, nil
}

// clientIDs are the client workspaces (not the agency's own) among ids.
func clientIDs(rows []store.AgencyWorkspacesRow) []uuid.UUID {
	out := []uuid.UUID{}
	for _, r := range rows {
		if r.Kind == "client" {
			out = append(out, r.ID)
		}
	}
	return out
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func displayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return email
}

// Initials of a workspace or a person: "Contoso Coffee" → "CC",
// "Wide World Importers" → "WW".
func Initials(name string) string {
	parts := strings.Fields(name)
	out := ""
	for _, p := range parts {
		if p == "" || strings.HasPrefix(p, "(") {
			continue
		}
		r := []rune(p)
		out += strings.ToUpper(string(r[0]))
		if len([]rune(out)) == 2 {
			break
		}
	}
	return out
}

// tileColors are the clients' initials tiles (the boards' solid tiles:
// green, brown, blue, teal, wine, purple, slate…). A client keeps the colour
// it was given when it joined; the agency's own tile is ink.
var tileColors = []string{"#166534", "#92400E", "#1D4ED8", "#0F766E", "#9D174D", "#5B21B6", "#475569", "#B45309", "#0E7490", "#4D7C0F"}

const agencyTile = "#0B1220"

// nextTileColor is the colour for the next client: the first one no
// client has yet, else the palette in turn.
func nextTileColor(rows []store.AgencyWorkspacesRow) string {
	used := map[string]bool{}
	for _, r := range rows {
		used[r.Color] = true
	}
	for _, c := range tileColors {
		if !used[c] {
			return c
		}
	}
	return tileColors[len(rows)%len(tileColors)]
}

// tileColor is a workspace's tile colour.
func tileColor(kind, color string) string {
	if kind == "agency" {
		return agencyTile
	}
	if color == "" {
		return tileColors[len(tileColors)-1]
	}
	return color
}
