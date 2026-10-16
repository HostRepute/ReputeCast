package accounts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// MeUser is the signed-in person.
type MeUser struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	AvatarColor     string     `json:"avatarColor"`
	AvatarURL       string     `json:"avatarUrl,omitempty"`
	Timezone        string     `json:"timezone"`
	IsInstanceAdmin bool       `json:"isInstanceAdmin"`
	BreakGlass      bool       `json:"breakGlass"`
	HasPassword     bool       `json:"hasPassword"`
	TOTP            bool       `json:"totp"`
	Passkeys        int        `json:"passkeys"`
	RecoveryCodes   int        `json:"recoveryCodes" doc:"Unused recovery codes"`
	TwoFAGraceUntil *time.Time `json:"twoFAGraceUntil,omitempty"`
	// Preferences are the app language and region format the person chose
	// (Account & security › Language & region).
	Preferences applang.MemberLanguage `json:"preferences,omitempty"`
	// LanguageSuggestionAnswered: the member answered "Your browser prefers …",
	// so it isn't offered again.
	LanguageSuggestionAnswered bool `json:"languageSuggestionAnswered,omitempty"`
}

// MeWorkspace is one workspace the person belongs to.
type MeWorkspace struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	RoleKey     string `json:"roleKey"`
	RoleName    string `json:"roleName"`
	MemberCount int64  `json:"memberCount"`
	Subscribers int64  `json:"subscribers"`
}

// MeInstance is what the shell shows about the installation.
type MeInstance struct {
	Name          string `json:"name"`
	PublicURL     string `json:"publicUrl"`
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion,omitempty"`
	SetupComplete bool   `json:"setupComplete"`
	// DefaultLanguage and DefaultRegionFormat apply to members who haven't chosen.
	DefaultLanguage     string `json:"defaultLanguage,omitempty" example:"en"`
	DefaultRegionFormat string `json:"defaultRegionFormat,omitempty" example:"standard"`
	// Languages are the app languages the server answers in (the pseudo-locale when it's on).
	Languages []applang.LanguageOption `json:"languages,omitempty"`
}

// Me is GET /me.
type Me struct {
	User       MeUser        `json:"user"`
	Workspaces []MeWorkspace `json:"workspaces"`
	Instance   MeInstance    `json:"instance"`
	// Pending is set when the person must finish something before using the
	// app: password_change, two_factor_setup or instance_setup.
	Pending string `json:"pending,omitempty" enum:"password_change,two_factor_setup,instance_setup"`
}

// Me returns the signed-in person, their workspaces and instance details.
func (s *Service) Me(ctx context.Context) (Me, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return Me{}, api.Err(401, api.CodeUnauthenticated, "Sign in to continue.")
	}
	passkeys, _ := s.Q.ListPasskeys(ctx, u.ID)
	codes, _ := s.Q.CountUnusedRecoveryCodes(ctx, u.ID)
	me := Me{
		User: MeUser{
			ID: u.ID.String(), Email: u.Email, Name: u.Name, AvatarColor: u.AvatarColor, AvatarURL: avatarURL(*u), Timezone: u.Timezone,
			IsInstanceAdmin: u.IsInstanceAdmin, BreakGlass: u.BreakGlass, HasPassword: u.PasswordHash != nil,
			TOTP: u.TotpEnabledAt != nil, Passkeys: len(passkeys), RecoveryCodes: int(codes), TwoFAGraceUntil: u.TwofaGraceUntil,
		},
		Workspaces: []MeWorkspace{},
	}
	wss, err := s.Q.ListUserWorkspaces(ctx, u.ID)
	if err != nil {
		return Me{}, err
	}
	for _, w := range wss {
		var subs int64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL`, w.ID).Scan(&subs)
		me.Workspaces = append(me.Workspaces, MeWorkspace{ID: w.ID.String(), Slug: w.Slug, Name: w.Name, RoleKey: w.RoleKey, RoleName: w.RoleName, MemberCount: w.MemberCount, Subscribers: subs})
	}
	st, err := s.policy(ctx)
	if err != nil {
		return Me{}, err
	}
	me.Instance = MeInstance{Name: st.Name, PublicURL: st.PublicUrl, Version: version.Get().Version, SetupComplete: st.SetupCompletedAt != nil,
		DefaultLanguage: st.DefaultLanguage, DefaultRegionFormat: st.DefaultRegionFormat, Languages: applang.Languages()}
	me.User.Preferences = applang.Choices(*u)
	me.User.LanguageSuggestionAnswered = u.LanguagePromptedAt != nil
	if len(st.LatestRelease) > 0 {
		var rel struct {
			Tag string `json:"tag"`
		}
		if json.Unmarshal(st.LatestRelease, &rel) == nil && rel.Tag != "" && rel.Tag != me.Instance.Version {
			me.Instance.LatestVersion = rel.Tag
		}
	}
	switch {
	case u.MustChangePassword:
		me.Pending = "password_change"
	case u.IsInstanceAdmin && st.SetupCompletedAt == nil:
		me.Pending = "instance_setup"
	case s.needsTwoFactor(ctx, *u, st):
		me.Pending = "two_factor_setup"
	}
	return me, nil
}

func (s *Service) needsTwoFactor(ctx context.Context, u store.User, st store.InstanceSetting) bool {
	if u.TotpEnabledAt != nil || st.TwofaEnforcement == "off" {
		return false
	}
	if sess := api.SessionFrom(ctx); sess != nil && sess.MfaVerified {
		return false
	}
	if pk, _ := s.Q.ListPasskeys(ctx, u.ID); len(pk) > 0 {
		return false
	}
	if u.TwofaGraceUntil != nil && u.TwofaGraceUntil.After(s.now()) {
		return false
	}
	if st.TwofaEnforcement == "everyone" {
		return true
	}
	wss, _ := s.Q.ListUserWorkspaces(ctx, u.ID)
	for _, w := range wss {
		m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: w.ID, UserID: u.ID})
		if err == nil && authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions).HasSensitive() {
			return true
		}
	}
	return false
}

// WorkspaceMe is GET /workspaces/{workspace}/me: the caller's role and
// permissions there, and the bits of workspace state the app shell shows.
type WorkspaceMe struct {
	Workspace   MeWorkspace `json:"workspace"`
	Timezone    string      `json:"timezone"`
	ActorType   string      `json:"actorType" enum:"user,api_key,mcp_agent"`
	Permissions []string    `json:"permissions"`
	Scope       authz.Scope `json:"scope"`
	// Unread is the number of unread notifications in this workspace.
	Unread int64 `json:"unread"`
	// HostRepute is the sidebar status line ("HostRepute connected · Acme Inc · synced 4 min ago").
	HostRepute *ShellHostRepute `json:"hostRepute,omitempty"`
	// HiddenSections are the sections the workspace hides from the sidebar,
	// the rail, the tab bar and the More sheet (Settings › General).
	HiddenSections []string `json:"hiddenSections" enum:"transactional,grow,earn"`
}

// ShellHostRepute summarizes the HostRepute connection for the sidebar.
type ShellHostRepute struct {
	Status     string     `json:"status" enum:"connected,revoked,expired,limit_reached,error"`
	TeamName   string     `json:"teamName"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
}

// WorkspaceMe returns the caller's view of the current workspace.
func (s *Service) WorkspaceMe(ctx context.Context) (WorkspaceMe, error) {
	p := api.PrincipalFrom(ctx)
	ws := api.WorkspaceFrom(ctx)
	out := WorkspaceMe{
		Workspace:   MeWorkspace{ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name, RoleKey: p.RoleKey, RoleName: p.RoleName},
		Timezone:    ws.Timezone,
		ActorType:   string(p.Type),
		Permissions: p.Permissions.Strings(),
		Scope:       p.Scope,
	}
	out.HiddenSections = workspaces.HiddenSections(ws)
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM workspace_members WHERE workspace_id = $1 AND status = 'active'`, ws.ID).Scan(&out.Workspace.MemberCount)
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL`, ws.ID).Scan(&out.Workspace.Subscribers)
	if p.Type == "user" {
		wsID := ws.ID
		out.Unread, _ = s.Q.CountUnreadNotifications(ctx, store.CountUnreadNotificationsParams{UserID: p.UserID, WorkspaceID: &wsID})
	}
	var hr ShellHostRepute
	if err := s.Pool.QueryRow(ctx, `SELECT status, team_name, last_sync_at FROM hostrepute_connections WHERE workspace_id = $1`, ws.ID).Scan(&hr.Status, &hr.TeamName, &hr.LastSyncAt); err == nil {
		out.HostRepute = &hr
	}
	return out, nil
}
