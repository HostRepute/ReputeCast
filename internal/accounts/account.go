package accounts

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// AccountOverview is the Account & security screen (A5 §5.6).
type AccountOverview struct {
	Profile       AccountProfile       `json:"profile"`
	Password      AccountPassword      `json:"password"`
	Passkeys      []AccountPasskey     `json:"passkeys"`
	TwoFactor     AccountTwoFactor     `json:"twoFactor"`
	Identities    []AccountIdentity    `json:"identities"`
	Workspaces    []AccountWorkspace   `json:"workspaces"`
	Notifications notify.Summary       `json:"notifications"`
	SessionPolicy AccountSessionPolicy `json:"sessionPolicy"`
}

// AccountProfile is how the person appears to the team.
type AccountProfile struct {
	Name            string        `json:"name"`
	Email           string        `json:"email"`
	Timezone        string        `json:"timezone"`
	AvatarColor     string        `json:"avatarColor"`
	AvatarURL       string        `json:"avatarUrl,omitempty"`
	IsInstanceAdmin bool          `json:"isInstanceAdmin"`
	PendingEmail    *PendingEmail `json:"pendingEmail,omitempty"`
}

// PendingEmail is an email change waiting for its verification link.
type PendingEmail struct {
	Email     string    `json:"email"`
	SentAt    time.Time `json:"sentAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AccountPassword is the Password section. It is hidden for people who
// only sign in through SSO.
type AccountPassword struct {
	Set           bool       `json:"set"`
	ChangedAt     *time.Time `json:"changedAt,omitempty"`
	MinLength     int        `json:"minLength"`
	BreachedCheck bool       `json:"breachedCheck"`
	// SSOOnly is true when there is no ReputeCast password and an SSO
	// identity signs the person in; ManagedBy names the provider.
	SSOOnly   bool   `json:"ssoOnly"`
	ManagedBy string `json:"managedBy,omitempty"`
}

// AccountPasskey is one passkey.
type AccountPasskey struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind: platform (built into a device or password manager) or security_key.
	Kind        string     `json:"kind" enum:"platform,security_key,unknown"`
	Provider    string     `json:"provider,omitempty" example:"iCloud Keychain"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	ThisSession bool       `json:"thisSession" doc:"This session signed in with the passkey"`
}

// AccountTwoFactor is the Authenticator app section and recovery codes.
type AccountTwoFactor struct {
	TOTP      bool       `json:"totp"`
	TOTPSetAt *time.Time `json:"totpSetAt,omitempty"`
	TOTPLabel string     `json:"totpLabel,omitempty" example:"1Password"`
	// Required is true when the instance policy requires a second factor.
	Required       bool       `json:"required"`
	RequiredReason string     `json:"requiredReason,omitempty"`
	GraceUntil     *time.Time `json:"graceUntil,omitempty"`
	// ViaIdP is true for SSO-only accounts: the identity provider enforces it.
	ViaIdP             bool       `json:"viaIdp"`
	RecoveryRemaining  int        `json:"recoveryRemaining"`
	RecoveryTotal      int        `json:"recoveryTotal"`
	RecoveryLastUsedAt *time.Time `json:"recoveryLastUsedAt,omitempty"`
}

// AccountIdentity is one row of Connected sign-ins.
type AccountIdentity struct {
	Kind   string `json:"kind" enum:"google,microsoft,github,sso"`
	Name   string `json:"name" example:"GitHub"`
	Preset string `json:"preset,omitempty" doc:"SSO preset, for the icon"`
	Linked bool   `json:"linked"`
	// ID identifies a linked identity (for unlinking).
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty" example:"linh-nguyen"`
	Email    string `json:"email,omitempty"`
	// Notes follow the account: "org acme-eng", "acme.io", "break-glass password kept".
	Notes []string `json:"notes"`
	// LinkURL starts linking (social sign-in).
	LinkURL   string `json:"linkUrl,omitempty"`
	CanUnlink bool   `json:"canUnlink"`
	// Managed: SSO is enforced for the domain; the organization manages it.
	Managed bool `json:"managed"`
	// OnlyMethod: this is the last way to sign in.
	OnlyMethod   bool   `json:"onlyMethod"`
	UnlinkReason string `json:"unlinkReason,omitempty"`
}

// AccountWorkspace is one row of My workspaces.
type AccountWorkspace struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	RoleKey     string `json:"roleKey"`
	RoleName    string `json:"roleName"`
	Members     int64  `json:"members"`
	CanLeave    bool   `json:"canLeave"`
	LeaveReason string `json:"leaveReason,omitempty"`
}

// AccountSessionPolicy is the instance session policy, for the caption.
type AccountSessionPolicy struct {
	Lifetime string `json:"lifetime" example:"14 days"`
	Idle     string `json:"idle" example:"12 hours"`
}

var socialNames = map[string]string{"google": "Google", "microsoft": "Microsoft", "github": "GitHub"}

// Overview returns everything the Account & security screen shows.
func (s *Service) Overview(ctx context.Context) (AccountOverview, error) {
	cu, err := currentUser(ctx)
	if err != nil {
		return AccountOverview{}, err
	}
	u := *cu
	st, err := s.policy(ctx)
	if err != nil {
		return AccountOverview{}, err
	}
	out := AccountOverview{
		Profile: AccountProfile{Name: u.Name, Email: u.Email, Timezone: u.Timezone, AvatarColor: u.AvatarColor, AvatarURL: avatarURL(u), IsInstanceAdmin: u.IsInstanceAdmin},
		Password: AccountPassword{Set: u.PasswordHash != nil, ChangedAt: u.PasswordSetAt, MinLength: int(st.PasswordMinLength), BreachedCheck: st.BreachedCheck},
		Passkeys:   []AccountPasskey{},
		Identities: []AccountIdentity{},
		Workspaces: []AccountWorkspace{},
		SessionPolicy: AccountSessionPolicy{Lifetime: humanDuration(st.SessionLifetime), Idle: humanDuration(st.SessionIdle)},
	}
	if pe, err := s.Q.GetPendingEmailChange(ctx, u.ID); err == nil {
		out.Profile.PendingEmail = &PendingEmail{Email: pe.NewEmail, SentAt: pe.CreatedAt, ExpiresAt: pe.ExpiresAt}
	}

	var sessPasskey *uuid.UUID
	if sess := api.SessionFrom(ctx); sess != nil {
		sessPasskey = sess.PasskeyID
	}
	keys, err := s.Q.ListPasskeys(ctx, u.ID)
	if err != nil {
		return AccountOverview{}, err
	}
	for _, k := range keys {
		out.Passkeys = append(out.Passkeys, passkeyView(k, sessPasskey))
	}

	ids, err := s.Q.ListIdentitiesWithProvider(ctx, u.ID)
	if err != nil {
		return AccountOverview{}, err
	}
	methods := signInMethods{password: u.PasswordHash != nil, passkeys: len(keys), identities: len(ids), totp: u.TotpEnabledAt != nil}
	only := methods.total() <= 1
	domain := emailDomain(u.Email)
	enforcer, _ := s.enforcingProvider(ctx, u.Email)
	socialAllowed := enforcer == nil || u.BreakGlass
	var enabledSocial map[string]struct {
		Enabled bool `json:"enabled"`
	}
	_ = jsonUnmarshal(st.SocialProviders, &enabledSocial)
	for _, kind := range []string{"google", "microsoft", "github"} {
		var linked *store.ListIdentitiesWithProviderRow
		for i := range ids {
			if ids[i].Kind == kind {
				linked = &ids[i]
				break
			}
		}
		if linked == nil && (!enabledSocial[kind].Enabled || !socialAllowed || !methodAllowed(st, "social")) {
			continue
		}
		row := AccountIdentity{Kind: kind, Name: socialNames[kind], Notes: []string{}}
		if linked == nil {
			row.LinkURL = "/api/v1/auth/social/" + kind + "/start?intent=link"
		} else {
			row.Linked, row.ID, row.Username, row.Email = true, linked.ID.String(), linked.Username, linked.Email
			row.Notes = identityNotes(linked.Details)
			row.CanUnlink, row.OnlyMethod = !only, only
			if only {
				row.UnlinkReason = "Add a passkey or password first"
			}
		}
		out.Identities = append(out.Identities, row)
	}
	ssoName := ""
	for _, i := range ids {
		if i.Kind != "sso" {
			continue
		}
		name := i.ProviderName
		if name == "" {
			name = "Single sign-on"
		}
		ssoName = name
		row := AccountIdentity{Kind: "sso", Name: name + " SSO", Preset: i.ProviderPreset, Linked: true, ID: i.ID.String(), Email: i.Email, Notes: []string{}}
		if row.Email == "" {
			row.Email = u.Email
		}
		row.Notes = append(row.Notes, emailDomain(row.Email))
		managed := slices.Contains(i.EnforceDomains, domain) && i.ProviderStatus == "enabled"
		if managed && u.BreakGlass && u.PasswordHash != nil {
			row.Notes = append(row.Notes, "break-glass password kept")
		}
		switch {
		case only:
			row.OnlyMethod = true
			row.UnlinkReason = "Add a passkey or password first"
		case managed:
			row.Managed = true
			row.UnlinkReason = "SSO is enforced for " + domain + "; the identity is managed by your organization"
		default:
			row.CanUnlink = true
		}
		out.Identities = append(out.Identities, row)
	}
	if u.PasswordHash == nil && ssoName != "" {
		out.Password.SSOOnly, out.Password.ManagedBy = true, ssoName
	}

	required, reason := s.twoFactorRequired(ctx, u, st)
	out.TwoFactor = AccountTwoFactor{TOTP: u.TotpEnabledAt != nil, TOTPSetAt: u.TotpEnabledAt, TOTPLabel: u.TotpLabel, Required: required, RequiredReason: reason,
		ViaIdP: out.Password.SSOOnly}
	if required && u.TotpEnabledAt == nil && len(keys) == 0 {
		out.TwoFactor.GraceUntil = u.TwofaGraceUntil
	}
	if rc, err := s.Q.RecoveryCodeStats(ctx, u.ID); err == nil {
		out.TwoFactor.RecoveryRemaining, out.TwoFactor.RecoveryTotal = int(rc.Remaining), int(rc.Total)
	}
	if t, err := s.Q.LastRecoveryCodeUse(ctx, u.ID); err == nil {
		out.TwoFactor.RecoveryLastUsedAt = &t
	}

	ms, err := s.Q.ListMembershipsOfUser(ctx, u.ID)
	if err != nil {
		return AccountOverview{}, err
	}
	for _, m := range ms {
		w := AccountWorkspace{ID: m.WorkspaceID.String(), Slug: m.Slug, Name: m.Name, RoleKey: m.RoleKey, RoleName: m.RoleName, Members: m.MemberCount, CanLeave: true}
		if m.RoleKey == authz.RoleOwner && m.OwnerCount <= 1 {
			w.CanLeave, w.LeaveReason = false, "Transfer ownership before leaving: the last Owner can't leave"
		}
		out.Workspaces = append(out.Workspaces, w)
	}

	prefs, err := (&notify.Service{Q: s.Q}).Preferences(ctx, u, nil)
	if err != nil {
		return AccountOverview{}, err
	}
	out.Notifications = notify.Summarize(prefs)
	return out, nil
}

func passkeyView(k store.UserPasskey, sessionPasskey *uuid.UUID) AccountPasskey {
	kind := "unknown"
	switch k.Attachment {
	case "platform":
		kind = "platform"
	case "cross-platform":
		kind = "security_key"
	}
	return AccountPasskey{ID: k.ID.String(), Name: k.Name, Kind: kind, Provider: k.Provider, CreatedAt: k.CreatedAt, LastUsedAt: k.LastUsedAt,
		ThisSession: sessionPasskey != nil && *sessionPasskey == k.ID}
}

func identityNotes(details []byte) []string {
	var d struct {
		Org string `json:"org"`
	}
	notes := []string{}
	if jsonUnmarshal(details, &d) == nil && d.Org != "" {
		notes = append(notes, "org "+d.Org)
	}
	return notes
}

func avatarURL(u store.User) string {
	if u.AvatarUpdatedAt == nil {
		return ""
	}
	return fmt.Sprintf("/api/v1/users/%s/avatar?v=%d", u.ID, u.AvatarUpdatedAt.Unix())
}

// humanDuration writes "14 days", "12 hours", "45 minutes".
func humanDuration(d time.Duration) string {
	switch {
	case d >= 24*time.Hour && d%(24*time.Hour) == 0:
		n := int(d / (24 * time.Hour))
		return fmt.Sprintf("%d %s", n, plural(n, "day", "days"))
	case d >= time.Hour && d%time.Hour == 0:
		n := int(d / time.Hour)
		return fmt.Sprintf("%d %s", n, plural(n, "hour", "hours"))
	default:
		n := int(d / time.Minute)
		return fmt.Sprintf("%d %s", n, plural(n, "minute", "minutes"))
	}
}

// ProfileInput changes the name and timezone.
type ProfileInput struct {
	Name     string `json:"name" minLength:"1" maxLength:"80"`
	Timezone string `json:"timezone" minLength:"1" maxLength:"64"`
}

// UpdateProfile changes the name and timezone.
func (s *Service) UpdateProfile(ctx context.Context, in ProfileInput) (AccountProfile, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountProfile{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AccountProfile{}, api.Invalid("body.name", "Enter your name as the team should see it.")
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return AccountProfile{}, api.Invalid("body.timezone", "Unknown timezone.")
	}
	var nu store.User
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		nu, err = q.UpdateUserProfile(ctx, store.UpdateUserProfileParams{ID: u.ID, Name: name, Timezone: in.Timezone, AvatarColor: u.AvatarColor})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.profile_update", TargetType: "user", TargetID: u.ID.String(), TargetName: name,
			Before: map[string]any{"name": u.Name, "timezone": u.Timezone}, After: map[string]any{"name": name, "timezone": in.Timezone}})
	})
	if err != nil {
		return AccountProfile{}, err
	}
	return AccountProfile{Name: nu.Name, Email: nu.Email, Timezone: nu.Timezone, AvatarColor: nu.AvatarColor, AvatarURL: avatarURL(nu), IsInstanceAdmin: nu.IsInstanceAdmin}, nil
}

// EmailChangeInput asks to move the account to a new address.
type EmailChangeInput struct {
	Email string `json:"email" format:"email" maxLength:"254"`
	// Password confirms the change for accounts that have one.
	Password string `json:"password,omitempty" maxLength:"1024"`
}

// EmailChangeStarted says where the verification link went.
type EmailChangeStarted struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const emailChangeTTL = 24 * time.Hour

// RequestEmailChange sends a verification link to the new address; the
// change applies when the person opens it.
func (s *Service) RequestEmailChange(ctx context.Context, in EmailChangeInput) (EmailChangeStarted, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return EmailChangeStarted{}, err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == u.Email {
		return EmailChangeStarted{}, api.Invalid("body.email", "That's already your email.")
	}
	if u.PasswordHash != nil {
		ok, err := authn.VerifyPassword(in.Password, *u.PasswordHash)
		if err != nil || !ok {
			return EmailChangeStarted{}, api.Invalid("body.password", "Your current password isn't right.")
		}
	}
	if taken, err := s.Q.EmailTaken(ctx, store.EmailTakenParams{Email: email, UserID: u.ID}); err != nil {
		return EmailChangeStarted{}, err
	} else if taken {
		return EmailChangeStarted{}, api.Invalid("body.email", "Another account uses this email.")
	}
	unsent := api.Err(http.StatusServiceUnavailable, api.CodeProviderUnavailable,
		"The verification email for "+email+" couldn't be sent: the System stream has no healthy provider right now. Your current email stays active.").WithMeta("email", email)
	if !s.mail().SystemAvailable(ctx, nil) {
		return EmailChangeStarted{}, unsent
	}
	token, hash := authn.NewToken()
	expires := s.now().Add(emailChangeTTL)
	link := s.publicURL(ctx) + s.accountPath(ctx, u.ID) + "?verifyEmail=" + token
	st, _ := s.policy(ctx)
	msg := mailer.Message{
		To: email, ToName: u.Name, Kind: "email_change",
		Subject: "Confirm your new email for " + st.Name,
		Text: fmt.Sprintf("%s asked to sign in to %s with this address from now on.\n\nConfirm the change: %s\n\nThe link works for 24 hours. If you didn't ask for this, ignore this email.\n", u.Email, st.Name, link),
		HTML: fmt.Sprintf(`<p style="font:14px/1.5 sans-serif">%s asked to sign in to %s with this address from now on.</p><p style="font:14px sans-serif"><a href="%s">Confirm the change</a></p><p style="font:13px/1.5 sans-serif;color:#5B6472">The link works for 24 hours. If you didn't ask for this, ignore this email.</p>`,
			html.EscapeString(u.Email), html.EscapeString(st.Name), html.EscapeString(link)),
	}
	if _, err := s.mail().SendSystem(ctx, msg); err != nil {
		s.Log.WarnContext(ctx, "email change verification not sent", "user", u.ID, "err", err)
		return EmailChangeStarted{}, unsent
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.DeletePendingEmailChanges(ctx, u.ID); err != nil {
			return err
		}
		if _, err := q.CreateEmailChange(ctx, store.CreateEmailChangeParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, NewEmail: email, TokenHash: hash, ExpiresAt: expires}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.email_change_request", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			After: map[string]any{"new_email": email}})
	})
	if err != nil {
		return EmailChangeStarted{}, err
	}
	return EmailChangeStarted{Email: email, ExpiresAt: expires}, nil
}

// accountPath is the Account & security path of the person's first workspace.
func (s *Service) accountPath(ctx context.Context, userID uuid.UUID) string {
	if wss, err := s.Q.ListUserWorkspaces(ctx, userID); err == nil && len(wss) > 0 {
		return "/w/" + wss[0].Slug + "/account"
	}
	return "/"
}

// CancelEmailChange drops a pending email change.
func (s *Service) CancelEmailChange(ctx context.Context) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.DeletePendingEmailChanges(ctx, u.ID)
		if err != nil || n == 0 {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.email_change_cancel", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email})
	})
}

// EmailChanged is returned after the new address is confirmed.
type EmailChanged struct {
	Email string `json:"email"`
}

// ConfirmEmailChange applies a pending change with the link from the email.
// The person must be signed in to the same account.
func (s *Service) ConfirmEmailChange(ctx context.Context, token string) (EmailChanged, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return EmailChanged{}, err
	}
	var ch store.EmailChange
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		ch, err = q.ConsumeEmailChange(ctx, store.ConsumeEmailChangeParams{TokenHash: authn.HashToken(token), UserID: u.ID})
		if err != nil {
			return api.Err(http.StatusGone, "email_change_expired", "This confirmation link expired, was already used, or belongs to another account. Change your email again from Account & security.")
		}
		if taken, err := q.EmailTaken(ctx, store.EmailTakenParams{Email: ch.NewEmail, UserID: u.ID}); err != nil {
			return err
		} else if taken {
			return api.Conflict("Another account uses " + ch.NewEmail + " now. Your email wasn't changed.")
		}
		if err := q.SetUserEmail(ctx, store.SetUserEmailParams{ID: u.ID, Email: ch.NewEmail}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.email_change", TargetType: "user", TargetID: u.ID.String(), TargetName: ch.NewEmail,
			Before: map[string]any{"email": u.Email}, After: map[string]any{"email": ch.NewEmail}})
	})
	if err != nil {
		return EmailChanged{}, err
	}
	// Tell the old address, so a takeover doesn't go unnoticed.
	st, _ := s.policy(ctx)
	_, _ = s.mail().SendSystem(ctx, mailer.Message{To: u.Email, ToName: u.Name, Kind: "email_changed",
		Subject: "Your " + st.Name + " email was changed",
		Text:    fmt.Sprintf("The account that used %s on %s now signs in with %s. If you didn't do this, contact your instance admin right away.\n", u.Email, st.Name, ch.NewEmail)})
	return EmailChanged{Email: ch.NewEmail}, nil
}

// UnlinkIdentity removes a social or SSO sign-in, unless it is the last way
// to sign in or the organization manages it.
func (s *Service) UnlinkIdentity(ctx context.Context, id uuid.UUID) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	ids, err := s.Q.ListIdentitiesWithProvider(ctx, u.ID)
	if err != nil {
		return err
	}
	var target *store.ListIdentitiesWithProviderRow
	for i := range ids {
		if ids[i].ID == id {
			target = &ids[i]
		}
	}
	if target == nil {
		return api.NotFound("Connected sign-in")
	}
	if s.methodsOf(ctx, *u).total() <= 1 {
		return api.Err(http.StatusConflict, "last_method", "This is your only sign-in method. Add a passkey or password before unlinking it.")
	}
	if target.Kind == "sso" && target.ProviderStatus == "enabled" && slices.Contains(target.EnforceDomains, emailDomain(u.Email)) {
		return api.Err(http.StatusConflict, "identity_managed", "SSO is enforced for "+emailDomain(u.Email)+"; the identity is managed by your organization.")
	}
	name := socialNames[target.Kind]
	if target.Kind == "sso" {
		name = target.ProviderName + " SSO"
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.DeleteIdentity(ctx, store.DeleteIdentityParams{ID: id, UserID: u.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.identity_unlink", TargetType: "identity", TargetID: id.String(), TargetName: name,
			Metadata: map[string]any{"kind": target.Kind}})
	})
}

// LeaveWorkspace removes the caller from the current workspace. The last
// Owner can't leave. API keys and agents the person created there are
// revoked, since a key never outlives its creator's access.
func (s *Service) LeaveWorkspace(ctx context.Context) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	ws := api.WorkspaceFrom(ctx)
	m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: u.ID})
	if err != nil {
		return api.MapError(err, "Membership")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if m.RoleKey == authz.RoleOwner {
			n, err := q.CountActiveOwners(ctx, ws.ID)
			if err != nil {
				return err
			}
			if n <= 1 {
				return api.Err(http.StatusConflict, api.CodeLastOwner, "You're the last Owner of "+ws.Name+". Transfer ownership before leaving: the last Owner can't leave.")
			}
		}
		if _, err := q.DeleteMember(ctx, store.DeleteMemberParams{ID: m.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		keys, err := q.RevokeAPIKeysCreatedBy(ctx, store.RevokeAPIKeysCreatedByParams{WorkspaceID: ws.ID, CreatedBy: &u.ID})
		if err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "member.leave", TargetType: "member", TargetID: m.ID.String(), TargetName: u.Email,
			Before: map[string]any{"role": m.RoleKey}, Metadata: map[string]any{"api_keys_revoked": keys}})
	})
}
