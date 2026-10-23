package accounts

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/passwordreuse"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// ChangePassword replaces the caller's password. A temporary password from
// `reputecast install` is replaced the same way (current = the temporary one).
func (s *Service) ChangePassword(ctx context.Context, current, next string) error {
	u := api.UserFrom(ctx)
	pol, err := s.policy(ctx)
	if err != nil {
		return err
	}
	if u.PasswordHash != nil {
		ok, err := authn.VerifyPassword(current, *u.PasswordHash)
		if err != nil || !ok {
			return api.Invalid("body.current", say(ctx, msgs.M("problem.currentPassword")))
		}
	}
	if p := authn.PasswordProblem(next, int(pol.PasswordMinLength)); p != "" {
		return api.Invalid("body.password", p)
	}
	if current == next {
		return api.Invalid("body.password", say(ctx, msgs.M("problem.samePassword")))
	}
	if used, err := passwordreuse.Used(ctx, s.Q, u.ID, u.PasswordHash, next, pol.PasswordReuse); err != nil {
		return err
	} else if used {
		return api.Invalid("body.password", passwordreuse.Message(pol.PasswordReuse))
	}
	if pol.BreachedCheck {
		if hit, err := s.Breach.Breached(ctx, next); err == nil && hit {
			return api.Invalid("body.password", say(ctx, mBreached))
		}
	}
	hash, err := authn.HashPassword(next)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: u.ID, PasswordHash: &hash, MustChangePassword: false}); err != nil {
			return err
		}
		// Every other session is signed out when the password changes.
		if sess := api.SessionFrom(ctx); sess != nil {
			if _, err := q.RevokeOtherSessions(ctx, store.RevokeOtherSessionsParams{UserID: u.ID, KeepID: sess.ID}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.password_change", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Metadata: map[string]any{"temporary_replaced": u.MustChangePassword}})
	})
}

// TOTPSetup is the secret shown while setting up an authenticator app.
type TOTPSetup struct {
	Secret    string `json:"secret" doc:"Base32 secret grouped by four for manual entry"`
	URL       string `json:"url" doc:"otpauth:// URL for the QR code"`
	Challenge string `json:"challenge" doc:"Pass back with the first code to confirm"`
	Account   string `json:"account"`
	Issuer    string `json:"issuer"`
}

// BeginTOTP creates a pending authenticator secret (not active until confirmed).
func (s *Service) BeginTOTP(ctx context.Context) (TOTPSetup, error) {
	u := api.UserFrom(ctx)
	st, err := s.policy(ctx)
	if err != nil {
		return TOTPSetup{}, err
	}
	sec, err := authn.NewTOTP(st.Name, u.Email)
	if err != nil {
		return TOTPSetup{}, err
	}
	enc, err := s.Box.SealString(sec.Secret, "totp-pending:"+u.ID.String())
	if err != nil {
		return TOTPSetup{}, err
	}
	token, hash := authn.NewToken()
	uid := u.ID
	if _, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
		ID: uuid.Must(uuid.NewV7()), TokenHash: hash, Kind: "totp_setup", UserID: &uid,
		Data: mustJSON(map[string]any{"secret": enc}), ExpiresAt: s.now().Add(20 * time.Minute),
	}); err != nil {
		return TOTPSetup{}, err
	}
	return TOTPSetup{Secret: authn.GroupSecret(sec.Secret), URL: sec.URL, Challenge: token, Account: u.Email, Issuer: st.Name}, nil
}

// RecoveryCodes are shown exactly once.
type RecoveryCodes struct {
	Codes []string `json:"codes"`
}

// ConfirmTOTP activates the authenticator after a correct code and returns
// fresh recovery codes.
func (s *Service) ConfirmTOTP(ctx context.Context, challenge, code string) (RecoveryCodes, error) {
	u := api.UserFrom(ctx)
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "totp_setup"})
	if err != nil || ch.UserID == nil || *ch.UserID != u.ID {
		return RecoveryCodes{}, api.Err(http.StatusBadRequest, "challenge_expired", say(ctx, msgs.M("problem.setupExpired")))
	}
	var data struct {
		Secret []byte `json:"secret"`
	}
	if err := jsonUnmarshal(ch.Data, &data); err != nil {
		return RecoveryCodes{}, err
	}
	secret, err := s.Box.OpenString(data.Secret, "totp-pending:"+u.ID.String())
	if err != nil {
		return RecoveryCodes{}, err
	}
	if !authn.VerifyTOTP(secret, code, s.now()) {
		return RecoveryCodes{}, api.Invalid("body.code", say(ctx, msgs.M("problem.wrongSetupCode")))
	}
	enc, err := s.Box.SealString(secret, totpAD(u.ID))
	if err != nil {
		return RecoveryCodes{}, err
	}
	codes := authn.NewRecoveryCodes(10)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetUserTOTP(ctx, store.SetUserTOTPParams{ID: u.ID, TotpSecretEnc: enc, TotpEnabledAt: &now}); err != nil {
			return err
		}
		if err := replaceRecoveryCodes(ctx, q, u.ID, codes); err != nil {
			return err
		}
		if err := q.DeleteChallenge(ctx, ch.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.totp_enable", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email})
	})
	return RecoveryCodes{Codes: codes}, err
}

// RegenerateRecoveryCodes replaces every recovery code.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context) (RecoveryCodes, error) {
	u := api.UserFrom(ctx)
	if u.TotpEnabledAt == nil {
		if pk, _ := s.Q.ListPasskeys(ctx, u.ID); len(pk) == 0 {
			return RecoveryCodes{}, api.Conflict(say(ctx, msgs.M("problem.noSecondFactor")))
		}
	}
	codes := authn.NewRecoveryCodes(10)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := replaceRecoveryCodes(ctx, q, u.ID, codes); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.recovery_codes_regenerate", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email})
	})
	return RecoveryCodes{Codes: codes}, err
}

func replaceRecoveryCodes(ctx context.Context, q *store.Queries, userID uuid.UUID, codes []string) error {
	if err := q.DeleteRecoveryCodes(ctx, userID); err != nil {
		return err
	}
	for _, c := range codes {
		if err := q.CreateRecoveryCode(ctx, store.CreateRecoveryCodeParams{ID: uuid.Must(uuid.NewV7()), UserID: userID, CodeHash: hashRecovery(c)}); err != nil {
			return err
		}
	}
	return nil
}

// InstanceBasics is step 4 of the first sign-in.
type InstanceBasics struct {
	Name        string `json:"name" minLength:"1" maxLength:"80"`
	PublicURL   string `json:"publicUrl" format:"uri" maxLength:"300"`
	Timezone    string `json:"timezone" maxLength:"64"`
	UpdateCheck bool   `json:"updateCheck"`
	// DefaultLanguage is the instance's default app language; the region
	// format follows it. Unchanged when omitted.
	DefaultLanguage string `json:"defaultLanguage,omitempty" enum:"en,vi"`
}

// SaveInstanceBasics stores the instance name, public URL, timezone and
// update check, and marks the setup step done.
func (s *Service) SaveInstanceBasics(ctx context.Context, in InstanceBasics) error {
	u, err := url.Parse(strings.TrimRight(in.PublicURL, "/"))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return api.Invalid("body.publicUrl", say(ctx, msgs.M("problem.instanceURL")))
	}
	if strings.TrimSpace(in.Name) == "" {
		return api.Invalid("body.name", say(ctx, msgs.M("problem.instanceName")))
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return api.Invalid("body.timezone", say(ctx, msgs.M("problem.timezone")))
	}
	before, _ := s.policy(ctx)
	lang, region := before.DefaultLanguage, before.DefaultRegionFormat
	if in.DefaultLanguage != "" && in.DefaultLanguage != lang {
		l, ok := i18n.ParseLocale(in.DefaultLanguage)
		if !ok || l == i18n.Pseudo {
			return api.Invalid("body.defaultLanguage", say(ctx, msgs.M("problem.language")))
		}
		lang, region = string(l), string(l.DefaultRegion())
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.UpdateInstanceBasics(ctx, store.UpdateInstanceBasicsParams{
			Name: strings.TrimSpace(in.Name), LogoSvg: before.LogoSvg, PublicUrl: u.String(), DefaultTimezone: in.Timezone, UpdateCheck: in.UpdateCheck,
		}); err != nil {
			return err
		}
		if err := q.AppLangSetInstanceDefaults(ctx, store.AppLangSetInstanceDefaultsParams{DefaultLanguage: lang, DefaultRegionFormat: region}); err != nil {
			return err
		}
		if err := q.ConfirmInstanceBasics(ctx); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "instance.settings_update", TargetType: "instance", TargetName: in.Name,
			Before: map[string]any{"name": before.Name, "public_url": before.PublicUrl, "timezone": before.DefaultTimezone, "update_check": before.UpdateCheck,
				"default_language": before.DefaultLanguage, "default_region_format": before.DefaultRegionFormat},
			After: map[string]any{"name": in.Name, "public_url": u.String(), "timezone": in.Timezone, "update_check": in.UpdateCheck,
				"default_language": lang, "default_region_format": region}})
	})
	if err == nil && s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
	if err == nil && s.OnPublicURL != nil && before.PublicUrl != u.String() {
		s.OnPublicURL(ctx, before.PublicUrl, u.String())
	}
	return err
}

// URLCheck reports whether the public URL resolves, has valid TLS and answers.
type URLCheck struct {
	CheckedAt time.Time `json:"checkedAt"`
	DNS       CheckLine `json:"dns"`
	TLS       CheckLine `json:"tls"`
	Reachable CheckLine `json:"reachable"`
}

// In writes the check's lines in a reader's language, for the API: Instance
// admin and the integrations hub store the check and read it later.
func (c URLCheck) In(l i18n.Lang) URLCheck {
	c.DNS, c.TLS, c.Reachable = c.DNS.In(l), c.TLS.In(l), c.Reachable.In(l)
	return c
}

// CheckLine is one row of the public URL check. Its detail is kept as a
// message too, stored with the check as "i18n" (not part of the API).
type CheckLine struct {
	Status string `json:"status" enum:"ok,warn,bad,skipped"`
	Detail string `json:"detail"`
	text   i18n.Text
}

// checkLine is a line in English, with its message kept.
func checkLine(status, key string, args i18n.Args) CheckLine {
	t := msgs.M(key).Keep(args)
	return CheckLine{Status: status, Detail: t.In(i18n.Default), text: t}
}

// In writes the line in a reader's language (lines stored before, without
// a message, keep their English) and drops the kept message.
func (c CheckLine) In(l i18n.Lang) CheckLine {
	if s := c.text.In(l); s != "" {
		c.Detail = s
	}
	c.text = i18n.Text{}
	return c
}

// MarshalJSON adds the kept message ("i18n") to the line.
func (c CheckLine) MarshalJSON() ([]byte, error) {
	type plain CheckLine
	var t *i18n.Text
	if !c.text.IsZero() {
		t = &c.text
	}
	return json.Marshal(struct {
		plain
		I18n *i18n.Text `json:"i18n,omitempty"`
	}{plain(c), t})
}

// UnmarshalJSON reads a line and its kept message.
func (c *CheckLine) UnmarshalJSON(b []byte) error {
	type plain CheckLine
	var v struct {
		plain
		I18n *i18n.Text `json:"i18n"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*c = CheckLine(v.plain)
	if v.I18n != nil {
		c.text = *v.I18n
	}
	return nil
}

// CheckPublicURL resolves the host, checks the certificate and fetches
// /healthz. The lines are in English with their messages kept; callers
// answer with URLCheck.In.
func (s *Service) CheckPublicURL(ctx context.Context, raw string) (URLCheck, error) {
	res := URLCheck{CheckedAt: s.now()}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return res, api.Invalid("body.url", say(ctx, msgs.M("problem.checkURL")))
	}
	host := u.Hostname()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		res.DNS = checkLine("bad", "check.dnsBad", i18n.Args{"host": host})
	} else {
		res.DNS = checkLine("ok", "check.dnsOk", i18n.Args{"host": host, "addresses": strings.Join(addrs, ", ")})
	}
	if u.Scheme == "https" {
		d := &tls.Dialer{Config: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}
		port := u.Port()
		if port == "" {
			port = "443"
		}
		conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
		if err != nil {
			res.TLS = checkLine("bad", "check.tlsBad", i18n.Args{"error": err.Error()})
		} else {
			cs := conn.(*tls.Conn).ConnectionState()
			_ = conn.Close()
			if len(cs.PeerCertificates) > 0 {
				c := cs.PeerCertificates[0]
				args := i18n.Args{"known": "no", "issuer": "", "at": c.NotAfter}
				if issuer := c.Issuer.Organization; len(issuer) > 0 {
					args["known"], args["issuer"] = "yes", issuer[0]
				}
				res.TLS = checkLine("ok", "check.tlsOk", args)
				if time.Until(c.NotAfter) < 14*24*time.Hour {
					res.TLS.Status = "warn"
				}
			}
		}
	} else {
		res.TLS = checkLine("warn", "check.plainHTTP", nil)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(u.String(), "/")+"/healthz", nil)
	start := time.Now()
	resp, err := (&http.Client{Timeout: 6 * time.Second}).Do(req)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			res.Reachable = checkLine("bad", "check.noAnswerDNS", i18n.Args{"host": u.Host})
		} else {
			res.Reachable = checkLine("bad", "check.noAnswer", i18n.Args{"host": u.Host, "error": shortErr(err)})
		}
	} else {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			res.Reachable = checkLine("ok", "check.answered", i18n.Args{"ms": time.Since(start).Milliseconds()})
		} else {
			res.Reachable = checkLine("warn", "check.answeredStatus", i18n.Args{"status": strconv.Itoa(resp.StatusCode)})
		}
	}
	return res, nil
}

// shortErr is a request's error, as Go writes it, cut to 120 characters.
func shortErr(err error) string {
	msg := err.Error()
	if len(msg) > 120 {
		msg = msg[:120] + "…"
	}
	return msg
}

// CompleteSetup marks the instance as set up (after the first workspace exists).
func (s *Service) CompleteSetup(ctx context.Context) error {
	st, err := s.policy(ctx)
	if err != nil {
		return err
	}
	if st.SetupCompletedAt != nil {
		return nil
	}
	_, err = s.Q.CompleteInstanceSetup(ctx, store.CompleteInstanceSetupParams{Name: st.Name, PublicUrl: st.PublicUrl, DefaultTimezone: st.DefaultTimezone, UpdateCheck: st.UpdateCheck})
	return err
}

// CreatedWorkspace is returned after creating a workspace.
type CreatedWorkspace struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// CreateWorkspace makes a new workspace with the caller as Owner.
func (s *Service) CreateWorkspace(ctx context.Context, name, timezone string) (CreatedWorkspace, error) {
	u := api.UserFrom(ctx)
	name = strings.TrimSpace(name)
	if name == "" {
		return CreatedWorkspace{}, api.Invalid("body.name", say(ctx, msgs.M("problem.workspaceName")))
	}
	// The first Owner finishes the earlier setup steps before creating the
	// first workspace: two-factor can't be skipped (A5 §5.5).
	if st, err := s.policy(ctx); err == nil && u.IsInstanceAdmin && st.SetupCompletedAt == nil {
		m := s.methodsOf(ctx, *u)
		if u.MustChangePassword || (!m.totp && m.passkeys == 0) || st.BasicsConfirmedAt == nil {
			return CreatedWorkspace{}, api.Err(http.StatusConflict, "setup_incomplete", say(ctx, msgs.M("problem.setupIncomplete")))
		}
	}
	if timezone == "" {
		st, _ := s.policy(ctx)
		timezone = st.DefaultTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return CreatedWorkspace{}, api.Invalid("body.timezone", say(ctx, msgs.M("problem.timezone")))
	}
	var ws store.Workspace
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		ws, err = workspaces.Create(ctx, tx, workspaces.CreateParams{Name: name, Timezone: timezone, Owner: u.ID, Source: "self"})
		if err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "workspace.create", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			After: map[string]any{"name": ws.Name, "slug": ws.Slug, "timezone": ws.Timezone}})
	})
	if err != nil {
		return CreatedWorkspace{}, err
	}
	if u.IsInstanceAdmin {
		_ = s.CompleteSetup(ctx)
	}
	return CreatedWorkspace{ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name}, nil
}
