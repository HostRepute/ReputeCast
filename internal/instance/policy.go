package instance

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/store"
)

// InstanceMethods are the instance-wide sign-in method switches.
type InstanceMethods struct {
	Password bool `json:"password"`
	Passkeys bool `json:"passkeys"`
	SSO      bool `json:"sso"`
}

// InstanceSocialToggles turn each social provider on or off.
type InstanceSocialToggles struct {
	Google    bool `json:"google"`
	Microsoft bool `json:"microsoft"`
	GitHub    bool `json:"github"`
}

// InstanceGraceMember is someone in a two-factor grace period.
type InstanceGraceMember struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Until time.Time `json:"until"`
}

// InstanceDomainRejection is the latest sign-in rejected for its domain.
type InstanceDomainRejection struct {
	Email string    `json:"email"`
	At    time.Time `json:"at"`
}

// InstancePolicy is Instance admin › Authentication.
type InstancePolicy struct {
	Methods InstanceMethods         `json:"methods"`
	Social  []sso.SSOSocialProvider `json:"social"`
	SSO     []sso.SSOProviderItem   `json:"sso"`
	// Password policy.
	PasswordMinLength int  `json:"passwordMinLength"`
	PasswordReuse     int  `json:"passwordReuse" doc:"Block the last N passwords; 0 allows re-use"`
	BreachedCheck     bool `json:"breachedCheck"`
	// Two-factor enforcement: everyone, sensitive (holders of a ⚑ permission) or off.
	TwoFactor string `json:"twoFactor" enum:"everyone,sensitive,off"`
	GraceDays int    `json:"graceDays"`
	// Sessions and lockout.
	SessionLifetimeHours int `json:"sessionLifetimeHours"`
	SessionIdleMinutes   int `json:"sessionIdleMinutes"`
	LockoutAttempts      int `json:"lockoutAttempts"`
	LockoutMinutes       int `json:"lockoutMinutes"`
	// JITDomains is the email domain allowlist for automatic accounts.
	JITDomains []string `json:"jitDomains"`
	// State the page explains.
	InGrace       []InstanceGraceMember    `json:"inGrace"`
	LockedNow     int                      `json:"lockedNow"`
	LastRejection *InstanceDomainRejection `json:"lastRejection,omitempty"`
	UpdatedAt     time.Time                `json:"updatedAt"`
}

// Policy returns the authentication policy and what it affects right now.
func (s *Service) Policy(ctx context.Context) (InstancePolicy, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstancePolicy{}, err
	}
	allowed := func(m string) bool { return len(st.AllowedMethods) == 0 || slices.Contains(st.AllowedMethods, m) }
	p := InstancePolicy{
		Methods:           InstanceMethods{Password: allowed("password"), Passkeys: allowed("passkey"), SSO: allowed("sso")},
		PasswordMinLength: int(st.PasswordMinLength), PasswordReuse: int(st.PasswordReuse), BreachedCheck: st.BreachedCheck,
		TwoFactor: st.TwofaEnforcement, GraceDays: int(st.TwofaGraceDays),
		SessionLifetimeHours: int(st.SessionLifetime / time.Hour), SessionIdleMinutes: int(st.SessionIdle / time.Minute),
		LockoutAttempts: int(st.LockoutAttempts), LockoutMinutes: int(st.LockoutMinutes),
		JITDomains: nonNil(st.JitDomainAllowlist), InGrace: []InstanceGraceMember{}, UpdatedAt: st.UpdatedAt,
	}
	if p.Social, err = s.SSO.SocialProviders(ctx); err != nil {
		return p, err
	}
	if !allowed("social") {
		for i := range p.Social {
			p.Social[i].Enabled = false
		}
	}
	list, err := s.SSO.List(ctx)
	if err != nil {
		return p, err
	}
	p.SSO = list.Items
	grace, err := s.Q.InstanceAdminUsersInGrace(ctx)
	if err != nil {
		return p, err
	}
	for _, g := range grace {
		if g.TwofaGraceUntil != nil {
			p.InGrace = append(p.InGrace, InstanceGraceMember{Name: displayName(g.Name, g.Email), Email: g.Email, Until: *g.TwofaGraceUntil})
		}
	}
	if locked, err := s.Q.InstanceAdminLockedCount(ctx); err == nil {
		p.LockedNow = int(max(locked.Users, locked.Emails))
	}
	switch last, err := s.Q.InstanceAdminLastDomainRejection(ctx); {
	case err == nil:
		p.LastRejection = &InstanceDomainRejection{Email: last.TargetName, At: last.OccurredAt}
	case !isNoRows(err):
		return p, err
	}
	return p, nil
}

// InstancePolicyInput is PUT /admin/policy (the "Save policies" button).
type InstancePolicyInput struct {
	Methods              InstanceMethods       `json:"methods"`
	Social               InstanceSocialToggles `json:"social"`
	PasswordMinLength    int                   `json:"passwordMinLength" minimum:"8" maximum:"128"`
	PasswordReuse        int                   `json:"passwordReuse" enum:"0,5,10"`
	BreachedCheck        bool                  `json:"breachedCheck"`
	TwoFactor            string                `json:"twoFactor" enum:"everyone,sensitive,off"`
	GraceDays            int                   `json:"graceDays" minimum:"0" maximum:"90"`
	SessionLifetimeHours int                   `json:"sessionLifetimeHours" minimum:"1" maximum:"2160"`
	SessionIdleMinutes   int                   `json:"sessionIdleMinutes" minimum:"5" maximum:"4320"`
	LockoutAttempts      int                   `json:"lockoutAttempts" minimum:"3" maximum:"100"`
	LockoutMinutes       int                   `json:"lockoutMinutes" minimum:"1" maximum:"1440"`
	JITDomains           []string              `json:"jitDomains" maxItems:"50"`
}

// SavePolicy stores the authentication policy. It refuses changes that
// would leave nobody a way to sign in, and drops the cached policy so the
// next request uses the new one.
func (s *Service) SavePolicy(ctx context.Context, in InstancePolicyInput) (InstancePolicy, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstancePolicy{}, err
	}
	before, err := s.Policy(ctx)
	if err != nil {
		return InstancePolicy{}, err
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstancePolicy{}, err
	}
	domains := []string{}
	for _, d := range in.JITDomains {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d), "@")))
		if d == "" || slices.Contains(domains, d) {
			continue
		}
		if !strings.Contains(d, ".") || strings.ContainsAny(d, " /@:") || len(d) > 253 {
			return InstancePolicy{}, api.Invalid("body.jitDomains", say(ctx, "problem.domain", i18n.Args{"domain": d}))
		}
		domains = append(domains, d)
	}
	if time.Duration(in.SessionIdleMinutes)*time.Minute > time.Duration(in.SessionLifetimeHours)*time.Hour {
		return InstancePolicy{}, api.Invalid("body.sessionIdleMinutes", say(ctx, "problem.idle", nil))
	}
	// Social providers can only be on with a client ID and secret.
	toggles := map[string]bool{"google": in.Social.Google, "microsoft": in.Social.Microsoft, "github": in.Social.GitHub}
	for _, k := range sso.SocialKinds {
		if toggles[k] && !socialOn(before.Social, k) {
			if why := s.SSO.SocialReady(st, k); why != "" {
				return InstancePolicy{}, api.Invalid("body.social."+k, why)
			}
		}
	}
	anySocial := in.Social.Google || in.Social.Microsoft || in.Social.GitHub
	enabledSSO := slices.ContainsFunc(before.SSO, func(p sso.SSOProviderItem) bool { return p.Status == "enabled" })
	if !in.Methods.SSO {
		for _, p := range before.SSO {
			if p.Status == "enabled" && len(p.Enforced) > 0 {
				return InstancePolicy{}, api.Invalid("body.methods.sso", say(ctx, "problem.ssoEnforces", i18n.Args{"provider": p.Name, "domains": strings.Join(p.Enforced, ", ")}))
			}
		}
	}
	if !in.Methods.Password && !in.Methods.Passkeys && (!in.Methods.SSO || !enabledSSO) && !anySocial {
		return InstancePolicy{}, api.Invalid("body.methods", say(ctx, "problem.noMethod", nil))
	}
	if err := s.keepsAWayIn(ctx, me, in, enabledSSO); err != nil {
		return InstancePolicy{}, err
	}
	methods := []string{}
	if in.Methods.Password {
		methods = append(methods, "password")
	}
	if in.Methods.Passkeys {
		methods = append(methods, "passkey")
	}
	if anySocial {
		methods = append(methods, "social")
	}
	if in.Methods.SSO {
		methods = append(methods, "sso")
	}
	cfgs := sso.SocialConfigs(st)
	for _, k := range sso.SocialKinds {
		c := cfgs[k]
		c.Enabled = toggles[k]
		cfgs[k] = c
	}
	socialJSON, err := json.Marshal(cfgs)
	if err != nil {
		return InstancePolicy{}, err
	}
	params := store.InstanceAdminSavePolicyParams{
		AllowedMethods: methods, PasswordMinLength: int32(in.PasswordMinLength), PasswordReuse: int32(in.PasswordReuse), BreachedCheck: in.BreachedCheck,
		TwofaEnforcement: in.TwoFactor, TwofaGraceDays: int32(in.GraceDays),
		SessionLifetime: time.Duration(in.SessionLifetimeHours) * time.Hour, SessionIdle: time.Duration(in.SessionIdleMinutes) * time.Minute,
		LockoutAttempts: int32(in.LockoutAttempts), LockoutMinutes: int32(in.LockoutMinutes), JitDomainAllowlist: domains, SocialProviders: socialJSON,
	}
	target, changes := policyChanges(before, in, domains)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.InstanceAdminSavePolicy(ctx, params); err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "policy.updated", TargetType: "instance_policy", TargetName: target.In(i18n.Default),
			Before: policyAuditView(before), After: policyAuditView(inputView(in, domains)), Metadata: audit.KeepText(audit.KeepSummary(nil, changes...), "target", target)})
	})
	if err != nil {
		return InstancePolicy{}, err
	}
	s.policyChanged()
	return s.Policy(ctx)
}

func socialOn(list []sso.SSOSocialProvider, kind string) bool {
	for _, p := range list {
		if p.Kind == kind {
			return p.Enabled
		}
	}
	return false
}

// keepsAWayIn refuses a policy that would lock out the admin saving it.
func (s *Service) keepsAWayIn(ctx context.Context, me *store.User, in InstancePolicyInput, enabledSSO bool) error {
	if in.Methods.Password && me.PasswordHash != nil || me.BreakGlass && me.PasswordHash != nil {
		return nil
	}
	if in.Methods.Passkeys {
		if pk, err := s.Q.ListPasskeys(ctx, me.ID); err == nil && len(pk) > 0 {
			return nil
		}
	}
	ids, err := s.Q.ListIdentities(ctx, me.ID)
	if err != nil {
		return err
	}
	for _, i := range ids {
		switch {
		case i.Kind == "sso" && in.Methods.SSO && enabledSSO:
			return nil
		case i.Kind == "google" && in.Social.Google, i.Kind == "microsoft" && in.Social.Microsoft, i.Kind == "github" && in.Social.GitHub:
			return nil
		}
	}
	return api.Err(http.StatusConflict, "lockout_risk", say(ctx, "problem.lockout", nil))
}

// policyView is the comparable part of a policy (for the audit log).
type policyView struct {
	Methods           InstanceMethods `json:"methods"`
	Social            map[string]bool `json:"social"`
	PasswordMinLength int             `json:"password_min_length"`
	PasswordReuse     int             `json:"password_reuse"`
	BreachedCheck     bool            `json:"breached_check"`
	TwoFactor         string          `json:"two_factor"`
	GraceDays         int             `json:"grace_days"`
	LifetimeHours     int             `json:"session_lifetime_hours"`
	IdleMinutes       int             `json:"session_idle_minutes"`
	LockoutAttempts   int             `json:"lockout_attempts"`
	LockoutMinutes    int             `json:"lockout_minutes"`
	JITDomains        []string        `json:"jit_domains"`
}

func policyAuditView(p InstancePolicy) policyView {
	social := map[string]bool{}
	for _, x := range p.Social {
		social[x.Kind] = x.Enabled
	}
	return policyView{Methods: p.Methods, Social: social, PasswordMinLength: p.PasswordMinLength, PasswordReuse: p.PasswordReuse, BreachedCheck: p.BreachedCheck,
		TwoFactor: p.TwoFactor, GraceDays: p.GraceDays, LifetimeHours: p.SessionLifetimeHours, IdleMinutes: p.SessionIdleMinutes,
		LockoutAttempts: p.LockoutAttempts, LockoutMinutes: p.LockoutMinutes, JITDomains: p.JITDomains}
}

func inputView(in InstancePolicyInput, domains []string) InstancePolicy {
	return InstancePolicy{Methods: in.Methods, Social: []sso.SSOSocialProvider{
		{Kind: "google", Enabled: in.Social.Google}, {Kind: "microsoft", Enabled: in.Social.Microsoft}, {Kind: "github", Enabled: in.Social.GitHub}},
		PasswordMinLength: in.PasswordMinLength, PasswordReuse: in.PasswordReuse, BreachedCheck: in.BreachedCheck, TwoFactor: in.TwoFactor, GraceDays: in.GraceDays,
		SessionLifetimeHours: in.SessionLifetimeHours, SessionIdleMinutes: in.SessionIdleMinutes, LockoutAttempts: in.LockoutAttempts,
		LockoutMinutes: in.LockoutMinutes, JITDomains: domains}
}

// policyChanges names the section that changed (or "Authentication
// policy") and describes each change like the audit log does, kept for
// later readers: "required for sensitive permissions · grace 7 days (was
// 14)".
func policyChanges(b InstancePolicy, in InstancePolicyInput, domains []string) (i18n.Text, []any) {
	var changes []any
	sections := map[string]bool{}
	add := func(section, key string, args i18n.Args) {
		sections[section] = true
		changes = append(changes, keep(key, args))
	}
	if b.Methods.Password != in.Methods.Password {
		add("methods", "policy.passwordSignIn", i18n.Args{"on": onOff(in.Methods.Password)})
	}
	if b.Methods.Passkeys != in.Methods.Passkeys {
		add("methods", "policy.passkeys", i18n.Args{"on": onOff(in.Methods.Passkeys)})
	}
	if b.Methods.SSO != in.Methods.SSO {
		add("methods", "policy.sso", i18n.Args{"on": onOff(in.Methods.SSO)})
	}
	for _, k := range sso.SocialKinds {
		want := map[string]bool{"google": in.Social.Google, "microsoft": in.Social.Microsoft, "github": in.Social.GitHub}[k]
		if socialOn(b.Social, k) != want {
			add("methods", "policy.social", i18n.Args{"provider": map[string]string{"google": "Google", "microsoft": "Microsoft", "github": "GitHub"}[k], "on": onOff(want)})
		}
	}
	if b.PasswordMinLength != in.PasswordMinLength {
		add("password", "policy.minLength", i18n.Args{"now": in.PasswordMinLength, "was": b.PasswordMinLength})
	}
	if b.PasswordReuse != in.PasswordReuse {
		kind := func(n int) string {
			if n == 0 {
				return "allowed"
			}
			return "blocked"
		}
		add("password", "policy.reuse", i18n.Args{"nowKind": kind(in.PasswordReuse), "now": in.PasswordReuse, "wasKind": kind(b.PasswordReuse), "was": b.PasswordReuse})
	}
	if b.BreachedCheck != in.BreachedCheck {
		add("password", "policy.breached", i18n.Args{"on": onOff(in.BreachedCheck)})
	}
	if b.TwoFactor != in.TwoFactor || b.GraceDays != in.GraceDays {
		changed := "no"
		if b.TwoFactor != in.TwoFactor {
			changed = "yes"
		}
		add("twoFactor", "policy.twoFactor", i18n.Args{"now": in.TwoFactor, "was": b.TwoFactor, "changed": changed})
		if b.GraceDays != in.GraceDays {
			add("twoFactor", "policy.grace", i18n.Args{"now": in.GraceDays, "was": b.GraceDays})
		}
	}
	if b.SessionLifetimeHours != in.SessionLifetimeHours {
		add("sessions", "policy.lifetime", durationArgs(in.SessionLifetimeHours*60, b.SessionLifetimeHours*60))
	}
	if b.SessionIdleMinutes != in.SessionIdleMinutes {
		add("sessions", "policy.idle", durationArgs(in.SessionIdleMinutes, b.SessionIdleMinutes))
	}
	if b.LockoutAttempts != in.LockoutAttempts || b.LockoutMinutes != in.LockoutMinutes {
		args := durationArgs(in.LockoutMinutes, b.LockoutMinutes)
		args["attempts"], args["wasAttempts"] = in.LockoutAttempts, b.LockoutAttempts
		add("sessions", "policy.lockout", args)
	}
	if !slices.Equal(b.JITDomains, domains) {
		var parts []string
		for _, d := range domains {
			if !slices.Contains(b.JITDomains, d) {
				parts = append(parts, "+ "+d)
			}
		}
		for _, d := range b.JITDomains {
			if !slices.Contains(domains, d) {
				parts = append(parts, "− "+d)
			}
		}
		if len(parts) > 0 {
			add("allowlist", "policy.allowlist", i18n.Args{"changes": strings.Join(parts, ", ")})
		}
	}
	if len(sections) == 1 {
		for k := range sections {
			return keep("section."+k, nil), changes
		}
	}
	return keep("section.all", nil), changes
}
