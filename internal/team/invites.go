package team

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// DefaultInviteDays is how long an invitation link works unless the inviter
// picks another expiry.
const DefaultInviteDays = 7

// CodeNoSystemProvider marks invite emails that couldn't be sent because no
// provider is assigned to the System stream; the response carries the links.
const CodeNoSystemProvider = "no_system_provider"

// TeamInvitation is a pending invitation.
type TeamInvitation struct {
	ID        string              `json:"id"`
	Email     string              `json:"email"`
	Role      TeamRoleRef         `json:"role"`
	Scope     access.ScopeSummary `json:"scope"`
	Message   string              `json:"message"`
	InvitedBy string              `json:"invitedBy"`
	// SentVia is the provider that delivered the email ("Postmark"); empty
	// when the invite was only shared as a link.
	SentVia    string     `json:"sentVia"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSentAt *time.Time `json:"lastSentAt,omitempty"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	Expired    bool       `json:"expired"`
}

// TeamInvitationList is the pending invitations card.
type TeamInvitationList struct {
	Items []TeamInvitation `json:"items"`
	// DefaultDays is the default expiry ("Links work for 7 days by default").
	DefaultDays int `json:"defaultDays"`
	// JIT describes SSO domains whose people can join on their own.
	JIT []TeamJITHint `json:"jit"`
	// CanInvite reflects team:invite (resend, copy link and revoke are hidden without it).
	CanInvite bool `json:"canInvite"`
}

// TeamJITHint is "acme.io users can also join automatically with Google Workspace".
type TeamJITHint struct {
	Domain   string `json:"domain"`
	Provider string `json:"provider"`
	Preset   string `json:"preset"`
	RoleName string `json:"roleName"`
}

// Invitations lists pending invitations.
func (s *Service) Invitations(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TeamInvitationList, error) {
	rows, err := s.Q.TeamListInvitations(ctx, ws.ID)
	if err != nil {
		return TeamInvitationList{}, err
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return TeamInvitationList{}, err
	}
	out := TeamInvitationList{Items: []TeamInvitation{}, DefaultDays: DefaultInviteDays, CanInvite: p.Can(authz.TeamInvite)}
	for _, r := range rows {
		inv := TeamInvitation{
			ID: r.ID.String(), Email: r.Email, Role: TeamRoleRef{ID: r.RoleID.String(), Key: r.RoleKey, Name: r.RoleName, Builtin: r.RoleBuiltin},
			Scope: access.View(access.ParseScope(r.Scope), lists, true), Message: r.Message, InvitedBy: r.InvitedByName,
			CreatedAt: r.CreatedAt, LastSentAt: r.LastSentAt, ExpiresAt: r.ExpiresAt, Expired: !r.ExpiresAt.After(s.now()),
		}
		if r.SentVia != nil {
			inv.SentVia = *r.SentVia
		}
		out.Items = append(out.Items, inv)
	}
	if out.JIT, err = s.jitHints(ctx, ws.ID); err != nil {
		return TeamInvitationList{}, err
	}
	return out, nil
}

func (s *Service) jitHints(ctx context.Context, wsID uuid.UUID) ([]TeamJITHint, error) {
	rows, err := s.Q.TeamJITProviders(ctx, &wsID)
	if err != nil {
		return nil, err
	}
	out := []TeamJITHint{}
	for _, r := range rows {
		role := "Viewer"
		if b, ok := authz.Builtin(r.JitRoleKey); ok {
			role = b.Name
		} else if r.JitRoleKey != "" {
			if rr, err := s.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: wsID, Key: r.JitRoleKey}); err == nil {
				role = rr.Name
			}
		}
		for _, d := range r.AllowedDomains {
			out = append(out, TeamJITHint{Domain: d, Provider: r.Name, Preset: r.Preset, RoleName: role})
		}
	}
	return out, nil
}

// TeamInviteRole is a role offered in the invite dialog.
type TeamInviteRole struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Builtin   bool   `json:"builtin"`
	Sensitive bool   `json:"sensitive" doc:"Holders need two-factor"`
	// Grantable is false when the inviter doesn't hold every permission of
	// the role (or it is the Owner role, which only a transfer gives).
	Grantable bool   `json:"grantable"`
	Reason    string `json:"reason,omitempty"`
	// Hint is "Editors can send; two-factor becomes required (7-day grace)."
	Hint string `json:"hint"`
}

// TeamInviteOptions feeds the invite dialog.
type TeamInviteOptions struct {
	Workspace string           `json:"workspace"`
	Roles     []TeamInviteRole `json:"roles"`
	// DefaultRoleID is Editor when grantable, otherwise the broadest grantable role.
	DefaultRoleID string              `json:"defaultRoleId"`
	Lists         []access.ScopedList `json:"lists"`
	SenderDomains []string            `json:"senderDomains"`
	JIT           []TeamJITHint       `json:"jit"`
	// SystemProvider names the provider of the System stream ("Postmark").
	// Empty when invitations can't be emailed yet; the dialog then offers links.
	SystemProvider string `json:"systemProvider"`
	CanEmail       bool   `json:"canEmail"`
	DefaultDays    int    `json:"defaultDays"`
	ExpiryOptions  []int  `json:"expiryOptions"`
	// SignInSummary is "Invitees pick a password or continue with Google, Microsoft or GitHub."
	SignInSummary string `json:"signInSummary"`
}

// Options returns what the invite dialog needs.
func (s *Service) Options(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TeamInviteOptions, error) {
	out := TeamInviteOptions{Workspace: ws.Name, Roles: []TeamInviteRole{}, Lists: []access.ScopedList{}, SenderDomains: []string{}, DefaultDays: DefaultInviteDays, ExpiryOptions: []int{1, 3, 7, 14, 30}}
	roles, err := s.Q.ListRoles(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	pol := s.policy(ctx)
	for _, r := range roles {
		perms := authz.Effective(r.Key, r.Builtin, r.Permissions)
		ir := TeamInviteRole{ID: r.ID.String(), Key: r.Key, Name: r.Name, Builtin: r.Builtin, Sensitive: perms.HasSensitive()}
		ir.Grantable, ir.Reason = canGrantRole(p.Permissions, r.Key, r.Builtin, perms)
		ir.Hint = roleHint(r.Name, perms, pol)
		out.Roles = append(out.Roles, ir)
		if r.Key == authz.RoleEditor && ir.Grantable {
			out.DefaultRoleID = ir.ID
		}
	}
	if out.DefaultRoleID == "" {
		for _, r := range out.Roles {
			if r.Grantable {
				out.DefaultRoleID = r.ID
				break
			}
		}
	}
	lists, err := s.Q.TeamWorkspaceLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		if p.Scope.AllowsList(l.ID) {
			out.Lists = append(out.Lists, access.ScopedList{ID: l.ID.String(), Name: l.Name})
		}
	}
	domains, err := s.Q.TeamWorkspaceSenderDomains(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, d := range domains {
		if p.Scope.AllowsSenderDomain(d) {
			out.SenderDomains = append(out.SenderDomains, d)
		}
	}
	if out.JIT, err = s.jitHints(ctx, ws.ID); err != nil {
		return out, err
	}
	out.CanEmail = s.Mail != nil && s.Mail.SystemAvailable(ctx, &ws.ID)
	if out.CanEmail {
		if name, err := s.Q.TeamSystemProviderName(ctx, ws.ID); err == nil {
			out.SystemProvider = name
		}
	}
	out.SignInSummary = signInSummary(pol)
	return out, nil
}

// canGrantRole applies "nobody can grant what they don't hold" to a role.
func canGrantRole(granter authz.Set, key string, builtin bool, perms authz.Set) (bool, string) {
	if builtin && key == authz.RoleOwner {
		return false, "Only a transfer of ownership makes someone the Owner."
	}
	if missing := granter.Missing(perms); len(missing) > 0 {
		keys := make([]string, len(missing))
		for i, m := range missing {
			keys[i] = string(m)
		}
		return false, fmt.Sprintf("You can't grant this role because you don't hold %s.", strings.Join(keys, ", "))
	}
	return true, ""
}

// roleHint is the one-line explanation under the role select.
func roleHint(name string, perms authz.Set, pol store.InstanceSetting) string {
	var parts []string
	switch {
	case perms.Has(authz.WorkspaceDelete):
		parts = append(parts, name+"s can do everything")
	case perms.Has(authz.TeamManage):
		parts = append(parts, name+"s can manage the team and every setting")
	case perms.Has(authz.CampaignsSend):
		parts = append(parts, name+"s can send")
	case perms.Has(authz.CampaignsEdit):
		parts = append(parts, name+"s can edit campaigns and request approval")
	default:
		parts = append(parts, name+"s can view everything, never secrets")
	}
	required := pol.TwofaEnforcement == "everyone" || (pol.TwofaEnforcement == "sensitive" && perms.HasSensitive())
	if required {
		grace := ""
		if pol.TwofaGraceDays > 0 {
			grace = fmt.Sprintf(" (%d-day grace)", pol.TwofaGraceDays)
		}
		parts = append(parts, "two-factor becomes required"+grace)
	}
	return strings.Join(parts, "; ") + "."
}

func signInSummary(pol store.InstanceSetting) string {
	var social []string
	var soc map[string]struct {
		Enabled bool `json:"enabled"`
	}
	_ = jsonUnmarshal(pol.SocialProviders, &soc)
	for _, k := range []struct{ key, name string }{{"google", "Google"}, {"microsoft", "Microsoft"}, {"github", "GitHub"}} {
		if soc[k.key].Enabled {
			social = append(social, k.name)
		}
	}
	if len(social) == 0 {
		return "Invitees pick a password and can add a passkey."
	}
	return "Invitees pick a password or continue with " + access.OrList(social) + "."
}

// TeamInviteInput creates invitations.
type TeamInviteInput struct {
	Emails        []string             `json:"emails" minItems:"1" maxItems:"50"`
	RoleID        string               `json:"roleId" format:"uuid"`
	Scope         *access.ScopeRequest `json:"scope,omitempty"`
	ExpiresInDays int                  `json:"expiresInDays,omitempty" minimum:"1" maximum:"30" doc:"Defaults to 7"`
	Message       string               `json:"message,omitempty" maxLength:"1000"`
	// SendEmail false only creates the links (the "copy links" fallback).
	SendEmail *bool `json:"sendEmail,omitempty"`
}

// TeamCreatedInvitation is one created invitation with its link.
type TeamCreatedInvitation struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	// Link is the invitation URL. It is shown to the inviter so it can be
	// shared when email isn't available.
	Link    string `json:"link"`
	Emailed bool   `json:"emailed"`
}

// TeamInviteResult reports what was created and whether email went out.
type TeamInviteResult struct {
	Invitations []TeamCreatedInvitation `json:"invitations"`
	Emailed     bool                    `json:"emailed"`
	// Fallback is "no_system_provider" when email couldn't be sent.
	Fallback string `json:"fallback,omitempty"`
	SentVia  string `json:"sentVia,omitempty"`
}

var errNoMail = errors.New("no mailer")

// Invite creates one invitation per email, then emails them through the
// System stream. Without a System provider the links are returned instead.
func (s *Service) Invite(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TeamInviteInput) (TeamInviteResult, error) {
	roleID, err := uuid.Parse(in.RoleID)
	if err != nil {
		return TeamInviteResult{}, api.Invalid("body.roleId", "Pick a role.")
	}
	role, err := s.Q.GetRole(ctx, roleID)
	if err != nil || role.WorkspaceID != ws.ID {
		return TeamInviteResult{}, api.Invalid("body.roleId", "That role doesn't exist in this workspace.")
	}
	perms := authz.Effective(role.Key, role.Builtin, role.Permissions)
	if ok, reason := canGrantRole(p.Permissions, role.Key, role.Builtin, perms); !ok {
		pr := api.Err(http.StatusForbidden, api.CodeEscalation, reason)
		if m := p.Permissions.Missing(perms); len(m) > 0 {
			pr.Permission = string(m[0])
		}
		return TeamInviteResult{}, pr
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return TeamInviteResult{}, err
	}
	domains, err := s.Q.TeamWorkspaceSenderDomains(ctx, ws.ID)
	if err != nil {
		return TeamInviteResult{}, err
	}
	scope, err := in.Scope.Resolve(lists, domains)
	if err != nil {
		return TeamInviteResult{}, api.Invalid("body.scope", capitalize(err.Error())+".")
	}
	if !scope.Within(p.Scope) {
		return TeamInviteResult{}, api.Err(http.StatusForbidden, api.CodeEscalation, "You can only invite people within your own scope.")
	}
	emails, err := normalizeEmails(in.Emails)
	if err != nil {
		return TeamInviteResult{}, err
	}
	members, err := s.Q.TeamMemberEmails(ctx, store.TeamMemberEmailsParams{WorkspaceID: ws.ID, Emails: emails})
	if err != nil {
		return TeamInviteResult{}, err
	}
	if len(members) > 0 {
		return TeamInviteResult{}, api.Invalid("body.emails", fmt.Sprintf("%s already %s a member of %s.", access.AndList(members), plural(len(members), "is", "are"), ws.Name))
	}
	pending, err := s.Q.TeamPendingInviteEmails(ctx, store.TeamPendingInviteEmailsParams{WorkspaceID: ws.ID, Emails: emails})
	if err != nil {
		return TeamInviteResult{}, err
	}
	if len(pending) > 0 {
		return TeamInviteResult{}, api.Invalid("body.emails", fmt.Sprintf("%s already %s a pending invitation. Resend it instead.", access.AndList(pending), plural(len(pending), "has", "have")))
	}
	days := in.ExpiresInDays
	if days == 0 {
		days = DefaultInviteDays
	}
	sendEmail := in.SendEmail == nil || *in.SendEmail
	now := s.now()
	expires := now.Add(time.Duration(days) * 24 * time.Hour)
	type created struct {
		id    uuid.UUID
		email string
		token string
	}
	var made []created
	inviter := p.UserID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, e := range emails {
			token, hash := authn.NewToken()
			id := uuid.Must(uuid.NewV7())
			enc, err := s.Box.SealString(token, tokenAD(id))
			if err != nil {
				return err
			}
			if _, err := q.TeamCreateInvitation(ctx, store.TeamCreateInvitationParams{
				ID: id, WorkspaceID: ws.ID, Email: e, RoleID: role.ID, Scope: access.ScopeJSON(scope), Message: strings.TrimSpace(in.Message),
				TokenHash: hash, TokenEnc: enc, InvitedBy: &inviter, ExpiresAt: expires, ExpiresDays: int32(days),
			}); err != nil {
				return err
			}
			made = append(made, created{id, e, token})
			if err := s.Audit.LogWith(ctx, q, audit.Event{
				WorkspaceID: &ws.ID, Action: "team.invited", TargetType: "invitation", TargetID: id.String(), TargetName: e,
				After: map[string]any{"email": e, "role": role.Name, "scope": access.View(scope, lists, true).Label, "expires_at": expires},
				Metadata: map[string]any{
					"summary":    inviteSummary(role.Name, access.View(scope, lists, false).Label, expires),
					"permission": string(authz.TeamInvite),
				},
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return TeamInviteResult{}, err
	}
	res := TeamInviteResult{Invitations: []TeamCreatedInvitation{}, Emailed: sendEmail}
	base := s.base(ctx)
	stop := !sendEmail
	for _, c := range made {
		ci := TeamCreatedInvitation{ID: c.id.String(), Email: c.email, Link: base + "/invite/" + c.token}
		if !stop {
			provider, err := s.sendInvite(ctx, ws, p, c.email, c.token, role.Name, in.Message, expires)
			if err == nil {
				ci.Emailed = true
				res.SentVia = provider
				_ = s.Q.TeamMarkInvitationSent(ctx, store.TeamMarkInvitationSentParams{ID: c.id, WorkspaceID: ws.ID, SentVia: &provider, ExpiresAt: expires})
			} else if errors.Is(err, mailer.ErrNoSystemProvider) || errors.Is(err, errNoMail) {
				// Nothing can be emailed until a provider serves the System stream.
				stop = true
			}
		}
		if !ci.Emailed {
			res.Emailed = false
		}
		res.Invitations = append(res.Invitations, ci)
	}
	if sendEmail && !res.Emailed {
		res.Fallback = CodeNoSystemProvider
	}
	return res, nil
}

func inviteSummary(role, scope string, expires time.Time) string {
	s := "role " + role
	if scope != "" {
		s += " · scope " + strings.TrimSuffix(scope, " only")
	}
	return s + " · expires " + expires.Format("Jan 2")
}

func (s *Service) sendInvite(ctx context.Context, ws *store.Workspace, p *authz.Principal, email, token, role, message string, expires time.Time) (string, error) {
	if s.Mail == nil {
		return "", errNoMail
	}
	link := s.base(ctx) + "/invite/" + token
	text := fmt.Sprintf("%s invited you to %s on ReputeCast as %s.\n\n", p.Name, ws.Name, role)
	if strings.TrimSpace(message) != "" {
		text += strings.TrimSpace(message) + "\n\n"
	}
	text += fmt.Sprintf("Accept the invitation: %s\n\nThe link works until %s.\n", link, expires.Format("Jan 2, 2006"))
	wsID := ws.ID
	provider, err := s.Mail.SendSystem(ctx, mailer.Message{
		WorkspaceID: &wsID, To: email, Subject: fmt.Sprintf("%s invited you to %s", p.Name, ws.Name), Text: text, Kind: "invite",
	})
	if err != nil {
		return "", err
	}
	return provider, nil
}

func (s *Service) base(ctx context.Context) string {
	if s.PublicURL != nil {
		return strings.TrimRight(s.PublicURL(ctx), "/")
	}
	return ""
}

func tokenAD(id uuid.UUID) string { return "invitation:" + id.String() + ":token" }

// TeamInviteLink is a copyable invitation link.
type TeamInviteLink struct {
	Link      string    `json:"link"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Link returns the link of a pending invitation (the emailed link stays valid).
func (s *Service) Link(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TeamInviteLink, error) {
	inv, err := s.Q.TeamGetInvitation(ctx, store.TeamGetInvitationParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return TeamInviteLink{}, api.MapError(err, "Invitation")
	}
	if len(inv.TokenEnc) == 0 {
		return TeamInviteLink{}, api.Conflict("This invitation was created before links could be copied. Resend it to get a new link.")
	}
	token, err := s.Box.OpenString(inv.TokenEnc, tokenAD(inv.ID))
	if err != nil {
		return TeamInviteLink{}, err
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "team.invite_link_copied", TargetType: "invitation", TargetID: inv.ID.String(), TargetName: inv.Email}); err != nil {
		return TeamInviteLink{}, err
	}
	return TeamInviteLink{Link: s.base(ctx) + "/invite/" + token, ExpiresAt: inv.ExpiresAt}, nil
}

// TeamResendResult reports a resend.
type TeamResendResult struct {
	Emailed   bool      `json:"emailed"`
	SentVia   string    `json:"sentVia,omitempty"`
	Link      string    `json:"link"`
	ExpiresAt time.Time `json:"expiresAt"`
	Fallback  string    `json:"fallback,omitempty"`
}

// Resend emails a pending invitation again and restarts its expiry.
func (s *Service) Resend(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (TeamResendResult, error) {
	inv, err := s.Q.TeamGetInvitation(ctx, store.TeamGetInvitationParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return TeamResendResult{}, api.MapError(err, "Invitation")
	}
	perms := authz.Effective(inv.RoleKey, inv.RoleBuiltin, inv.RolePermissions)
	if ok, reason := canGrantRole(p.Permissions, inv.RoleKey, inv.RoleBuiltin, perms); !ok {
		return TeamResendResult{}, api.Err(http.StatusForbidden, api.CodeEscalation, reason)
	}
	var token string
	if len(inv.TokenEnc) > 0 {
		if token, err = s.Box.OpenString(inv.TokenEnc, tokenAD(inv.ID)); err != nil {
			return TeamResendResult{}, err
		}
	} else {
		// Older invitations without a sealed token get a new one.
		var hash []byte
		token, hash = authn.NewToken()
		enc, err := s.Box.SealString(token, tokenAD(inv.ID))
		if err != nil {
			return TeamResendResult{}, err
		}
		if _, err := s.Pool.Exec(ctx, `UPDATE invitations SET token_hash = $2, token_enc = $3 WHERE id = $1`, inv.ID, hash, enc); err != nil {
			return TeamResendResult{}, err
		}
	}
	days := int(inv.ExpiresDays)
	if days <= 0 {
		days = DefaultInviteDays
	}
	expires := s.now().Add(time.Duration(days) * 24 * time.Hour)
	res := TeamResendResult{Link: s.base(ctx) + "/invite/" + token, ExpiresAt: expires}
	provider, mailErr := s.sendInvite(ctx, ws, p, inv.Email, token, inv.RoleName, inv.Message, expires)
	var via *string
	if mailErr == nil {
		res.Emailed, res.SentVia = true, provider
		via = &provider
	} else {
		res.Fallback = CodeNoSystemProvider
		via = inv.SentVia
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TeamMarkInvitationSent(ctx, store.TeamMarkInvitationSentParams{ID: inv.ID, WorkspaceID: ws.ID, SentVia: via, ExpiresAt: expires}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "team.invite_resent", TargetType: "invitation", TargetID: inv.ID.String(), TargetName: inv.Email,
			Before: map[string]any{"expires_at": inv.ExpiresAt}, After: map[string]any{"expires_at": expires, "emailed": res.Emailed}})
	})
	return res, err
}

// Revoke cancels a pending invitation; its link stops working.
func (s *Service) Revoke(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	inv, err := s.Q.TeamGetInvitation(ctx, store.TeamGetInvitationParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Invitation")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.TeamRevokeInvitation(ctx, store.TeamRevokeInvitationParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Invitation")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "team.invite_revoked", TargetType: "invitation", TargetID: id.String(), TargetName: inv.Email,
			Before: map[string]any{"email": inv.Email, "role": inv.RoleName}})
	})
}

func normalizeEmails(in []string) ([]string, error) {
	var out []string
	for _, raw := range in {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' }) {
			e := strings.ToLower(strings.TrimSpace(part))
			if e == "" {
				continue
			}
			a, err := mail.ParseAddress(e)
			if err != nil || a.Address != e || !strings.Contains(e[strings.LastIndexByte(e, '@')+1:], ".") {
				return nil, api.Invalid("body.emails", fmt.Sprintf("“%s” isn't an email address.", part))
			}
			if !slices.Contains(out, e) {
				out = append(out, e)
			}
		}
	}
	if len(out) == 0 {
		return nil, api.Invalid("body.emails", "Add at least one email address.")
	}
	if len(out) > 50 {
		return nil, api.Invalid("body.emails", "Invite up to 50 people at a time.")
	}
	return out, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
