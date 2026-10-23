package accounts

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/membermail"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// AccountOverview is the Account & security screen (A5 §5.6).
type AccountOverview struct {
	Profile       AccountProfile             `json:"profile"`
	Password      AccountPassword            `json:"password"`
	Passkeys      []AccountPasskey           `json:"passkeys"`
	TwoFactor     AccountTwoFactor           `json:"twoFactor"`
	Identities    []AccountIdentity          `json:"identities"`
	Workspaces    []AccountWorkspace         `json:"workspaces"`
	Notifications notify.NotificationSummary `json:"notifications"`
	SessionPolicy AccountSessionPolicy       `json:"sessionPolicy"`
}

// AccountProfile is how the person appears to the team.
type AccountProfile struct {
	Name            string               `json:"name"`
	Email           string               `json:"email"`
	Timezone        string               `json:"timezone"`
	AvatarColor     string               `json:"avatarColor"`
	AvatarURL       string               `json:"avatarUrl,omitempty"`
	IsInstanceAdmin bool                 `json:"isInstanceAdmin"`
	PendingEmail    *AccountPendingEmail `json:"pendingEmail,omitempty"`
}

// AccountPendingEmail is an email change waiting for its verification link.
type AccountPendingEmail struct {
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
	// Required is true when the instance policy requires a second factor:
	// for everyone, or because the person holds a sensitive permission.
	Required       bool       `json:"required"`
	RequiredBy     string     `json:"requiredBy,omitempty" enum:"everyone,sensitive"`
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
		Profile:       AccountProfile{Name: u.Name, Email: u.Email, Timezone: u.Timezone, AvatarColor: u.AvatarColor, AvatarURL: avatarURL(u), IsInstanceAdmin: u.IsInstanceAdmin},
		Password:      AccountPassword{Set: u.PasswordHash != nil, ChangedAt: u.PasswordSetAt, MinLength: int(st.PasswordMinLength), BreachedCheck: st.BreachedCheck},
		Passkeys:      []AccountPasskey{},
		Identities:    []AccountIdentity{},
		Workspaces:    []AccountWorkspace{},
		SessionPolicy: AccountSessionPolicy{Lifetime: humanDuration(langOf(ctx), st.SessionLifetime), Idle: humanDuration(langOf(ctx), st.SessionIdle)},
	}
	if pe, err := s.Q.GetPendingEmailChange(ctx, u.ID); err == nil {
		out.Profile.PendingEmail = &AccountPendingEmail{Email: pe.NewEmail, SentAt: pe.CreatedAt, ExpiresAt: pe.ExpiresAt}
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
			row.Notes = identityNotes(ctx, linked.Details)
			row.CanUnlink, row.OnlyMethod = !only, only
			if only {
				row.UnlinkReason = say(ctx, mUnlinkFirst)
			}
		}
		out.Identities = append(out.Identities, row)
	}
	ssoName := ""
	for _, i := range ids {
		if i.Kind != "sso" {
			continue
		}
		name, label := i.ProviderName, ""
		if name == "" {
			name = say(ctx, mSSOUnnamed)
			label = name
		} else {
			label = say(ctx, mSSOName, "provider", name)
		}
		ssoName = name
		row := AccountIdentity{Kind: "sso", Name: label, Preset: i.ProviderPreset, Linked: true, ID: i.ID.String(), Email: i.Email, Notes: []string{}}
		if row.Email == "" {
			row.Email = u.Email
		}
		row.Notes = append(row.Notes, emailDomain(row.Email))
		managed := slices.Contains(i.EnforceDomains, domain) && i.ProviderStatus == "enabled"
		if managed && u.BreakGlass && u.PasswordHash != nil {
			row.Notes = append(row.Notes, say(ctx, mNoteBreakGlass))
		}
		switch {
		case only:
			row.OnlyMethod = true
			row.UnlinkReason = say(ctx, mUnlinkFirst)
		case managed:
			row.Managed = true
			row.UnlinkReason = say(ctx, mSSOManaged, "domain", domain)
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
	if required {
		out.TwoFactor.RequiredBy = "sensitive"
		if st.TwofaEnforcement == "everyone" {
			out.TwoFactor.RequiredBy = "everyone"
		}
		if u.TotpEnabledAt == nil && len(keys) == 0 {
			out.TwoFactor.GraceUntil = u.TwofaGraceUntil
		}
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
			w.CanLeave, w.LeaveReason = false, say(ctx, mLastOwner)
		}
		out.Workspaces = append(out.Workspaces, w)
	}

	prefs, err := (&notify.Service{Q: s.Q}).Preferences(ctx, u, nil)
	if err != nil {
		return AccountOverview{}, err
	}
	out.Notifications = notify.SummarizeIn(langOf(ctx), prefs)
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

func identityNotes(ctx context.Context, details []byte) []string {
	var d struct {
		Org string `json:"org"`
	}
	notes := []string{}
	if jsonUnmarshal(details, &d) == nil && d.Org != "" {
		notes = append(notes, say(ctx, mNoteOrg, "org", d.Org))
	}
	return notes
}

func avatarURL(u store.User) string {
	if u.AvatarUpdatedAt == nil {
		return ""
	}
	return fmt.Sprintf("/api/v1/users/%s/avatar?v=%d", u.ID, u.AvatarUpdatedAt.Unix())
}

// humanDuration writes "14 days", "12 hours", "45 minutes" in a language.
func humanDuration(l i18n.Lang, d time.Duration) string {
	switch {
	case d >= 24*time.Hour && d%(24*time.Hour) == 0:
		return mDays.Text(l, i18n.Args{"n": int(d / (24 * time.Hour))})
	case d >= time.Hour && d%time.Hour == 0:
		return mHours.Text(l, i18n.Args{"n": int(d / time.Hour)})
	default:
		return mMinutes.Text(l, i18n.Args{"n": int(d / time.Minute)})
	}
}

// AccountProfileInput changes the name and timezone.
type AccountProfileInput struct {
	Name     string `json:"name" minLength:"1" maxLength:"80"`
	Timezone string `json:"timezone" minLength:"1" maxLength:"64"`
}

// UpdateProfile changes the name and timezone.
func (s *Service) UpdateProfile(ctx context.Context, in AccountProfileInput) (AccountProfile, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountProfile{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AccountProfile{}, api.Invalid("body.name", say(ctx, msgs.M("problem.profileName")))
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return AccountProfile{}, api.Invalid("body.timezone", say(ctx, msgs.M("problem.timezone")))
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

// AccountEmailChangeInput asks to move the account to a new address.
type AccountEmailChangeInput struct {
	Email string `json:"email" format:"email" maxLength:"254"`
	// Password confirms the change for accounts that have one.
	Password string `json:"password,omitempty" maxLength:"1024"`
}

// AccountEmailChangeStarted says where the verification link went.
type AccountEmailChangeStarted struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const emailChangeTTL = 24 * time.Hour

// RequestEmailChange sends a verification link to the new address; the
// change applies when the person opens it.
func (s *Service) RequestEmailChange(ctx context.Context, in AccountEmailChangeInput) (AccountEmailChangeStarted, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountEmailChangeStarted{}, err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == u.Email {
		return AccountEmailChangeStarted{}, api.Invalid("body.email", say(ctx, msgs.M("problem.sameEmail")))
	}
	if u.PasswordHash != nil {
		ok, err := authn.VerifyPassword(in.Password, *u.PasswordHash)
		if err != nil || !ok {
			return AccountEmailChangeStarted{}, api.Invalid("body.password", say(ctx, msgs.M("problem.currentPassword")))
		}
	}
	if taken, err := s.Q.EmailTaken(ctx, store.EmailTakenParams{Email: email, UserID: u.ID}); err != nil {
		return AccountEmailChangeStarted{}, err
	} else if taken {
		return AccountEmailChangeStarted{}, api.Invalid("body.email", say(ctx, msgs.M("problem.emailTaken")))
	}
	unsent := api.Err(http.StatusServiceUnavailable, api.CodeProviderUnavailable,
		say(ctx, msgs.M("problem.verificationUnsent"), "email", email)).WithMeta("email", email)
	if !s.mail().SystemAvailable(ctx, nil) {
		return AccountEmailChangeStarted{}, unsent
	}
	token, hash := authn.NewToken()
	expires := s.now().Add(emailChangeTTL)
	link := s.publicURL(ctx) + s.accountPath(ctx, u.ID) + "?verifyEmail=" + token
	st, _ := s.policy(ctx)
	msg := membermail.EmailChange(applang.ForUser(ctx, s.Q, *u), st.Name, u.Email, link)
	msg.To, msg.ToName = email, u.Name
	if _, err := s.mail().SendSystem(ctx, msg); err != nil {
		s.Log.WarnContext(ctx, "email change verification not sent", "user", u.ID, "err", err)
		return AccountEmailChangeStarted{}, unsent
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
		return AccountEmailChangeStarted{}, err
	}
	return AccountEmailChangeStarted{Email: email, ExpiresAt: expires}, nil
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

// AccountEmailChanged is returned after the new address is confirmed.
type AccountEmailChanged struct {
	Email string `json:"email"`
}

// ConfirmEmailChange applies a pending change with the link from the email.
// The person must be signed in to the same account.
func (s *Service) ConfirmEmailChange(ctx context.Context, token string) (AccountEmailChanged, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return AccountEmailChanged{}, err
	}
	var ch store.EmailChange
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		ch, err = q.ConsumeEmailChange(ctx, store.ConsumeEmailChangeParams{TokenHash: authn.HashToken(token), UserID: u.ID})
		if err != nil {
			return api.Err(http.StatusGone, "email_change_expired", say(ctx, msgs.M("problem.emailLinkExpired")))
		}
		if taken, err := q.EmailTaken(ctx, store.EmailTakenParams{Email: ch.NewEmail, UserID: u.ID}); err != nil {
			return err
		} else if taken {
			return api.Conflict(say(ctx, msgs.M("problem.emailTakenNow"), "email", ch.NewEmail))
		}
		if err := q.SetUserEmail(ctx, store.SetUserEmailParams{ID: u.ID, Email: ch.NewEmail}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.email_change", TargetType: "user", TargetID: u.ID.String(), TargetName: ch.NewEmail,
			Before: map[string]any{"email": u.Email}, After: map[string]any{"email": ch.NewEmail}})
	})
	if err != nil {
		return AccountEmailChanged{}, err
	}
	// Tell the old address, so a takeover doesn't go unnoticed.
	st, _ := s.policy(ctx)
	notice := membermail.EmailChanged(applang.ForUser(ctx, s.Q, *u), st.Name, u.Email, ch.NewEmail)
	notice.To, notice.ToName = u.Email, u.Name
	_, _ = s.mail().SendSystem(ctx, notice)
	return AccountEmailChanged{Email: ch.NewEmail}, nil
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
		return api.Err(http.StatusConflict, "last_method", say(ctx, msgs.M("problem.onlyMethod")))
	}
	if target.Kind == "sso" && target.ProviderStatus == "enabled" && slices.Contains(target.EnforceDomains, emailDomain(u.Email)) {
		return api.Err(http.StatusConflict, "identity_managed", say(ctx, msgs.M("problem.identityManaged"), "domain", emailDomain(u.Email)))
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
				return api.Err(http.StatusConflict, api.CodeLastOwner, say(ctx, msgs.M("problem.lastOwnerLeave"), "workspace", ws.Name))
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
