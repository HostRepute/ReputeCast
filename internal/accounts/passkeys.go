package accounts

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Passkeys (WebAuthn) sign people in on their own (discoverable credentials
// with user verification) and count as a second factor after a password.
// The relying party is the host of the public URL, so passkeys keep working
// as long as the public URL's host doesn't change.

const ceremonyTTL = 5 * time.Minute

// PasskeyOptions starts a WebAuthn ceremony in the browser.
type PasskeyOptions struct {
	// Challenge identifies the ceremony; pass it back with the browser's answer.
	Challenge string `json:"challenge"`
	// Options are PublicKeyCredentialCreationOptionsJSON (registration) or
	// PublicKeyCredentialRequestOptionsJSON (sign-in) for navigator.credentials.
	Options json.RawMessage `json:"options"`
}

// webAuthn builds the relying party from the instance's public URL.
func (s *Service) webAuthn(ctx context.Context) (*webauthn.WebAuthn, error) {
	st, err := s.policy(ctx)
	if err != nil {
		return nil, err
	}
	pub := s.publicURL(ctx)
	u, err := url.Parse(pub)
	if err != nil || u.Host == "" {
		return nil, api.Err(http.StatusConflict, "public_url_invalid", "Passkeys need the instance's public URL. Set it in Instance admin › Instance settings.")
	}
	name := st.Name
	if name == "" {
		name = "ReputeCast"
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: name,
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTTL, TimeoutUVD: ceremonyTTL},
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTTL, TimeoutUVD: ceremonyTTL},
		},
	})
	if err != nil {
		return nil, api.Err(http.StatusConflict, "public_url_invalid", "Passkeys need a public URL with a domain name (not an IP address). Set it in Instance admin › Instance settings.")
	}
	return wa, nil
}

// waUser adapts a user and their passkeys to go-webauthn.
type waUser struct {
	u     store.User
	keys  []store.UserPasskey
	creds []webauthn.Credential
}

func (w *waUser) WebAuthnID() []byte   { return w.u.WebauthnID }
func (w *waUser) WebAuthnName() string { return w.u.Email }
func (w *waUser) WebAuthnDisplayName() string {
	if w.u.Name != "" {
		return w.u.Name
	}
	return w.u.Email
}
func (w *waUser) WebAuthnCredentials() []webauthn.Credential { return w.creds }

// key returns the stored passkey of a credential id.
func (w *waUser) key(credID []byte) (store.UserPasskey, bool) {
	for _, k := range w.keys {
		if string(k.CredentialID) == string(credID) {
			return k, true
		}
	}
	return store.UserPasskey{}, false
}

func (s *Service) loadWAUser(ctx context.Context, q *store.Queries, u store.User) (*waUser, error) {
	keys, err := q.ListPasskeys(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	w := &waUser{u: u, keys: keys}
	for _, k := range keys {
		var c webauthn.Credential
		if len(k.Credential) > 2 {
			if err := json.Unmarshal(k.Credential, &c); err != nil {
				s.Log.WarnContext(ctx, "unreadable passkey record", "passkey", k.ID, "err", err)
				continue
			}
		}
		if len(c.ID) == 0 {
			// Records created before full credential storage: rebuild from columns.
			c = webauthn.Credential{ID: k.CredentialID, PublicKey: k.PublicKey, AttestationType: k.AttestationType,
				Flags:         webauthn.CredentialFlags{UserPresent: true, UserVerified: true, BackupEligible: k.BackupEligible, BackupState: k.BackupState},
				Authenticator: webauthn.Authenticator{AAGUID: k.Aaguid, SignCount: uint32(k.SignCount)}}
		}
		w.creds = append(w.creds, c)
	}
	return w, nil
}

// ensureWebAuthnID gives the user a random WebAuthn user handle.
func (s *Service) ensureWebAuthnID(ctx context.Context, u store.User) (store.User, error) {
	if len(u.WebauthnID) > 0 {
		return u, nil
	}
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		return u, err
	}
	if err := s.Q.SetUserWebAuthnID(ctx, store.SetUserWebAuthnIDParams{ID: u.ID, WebauthnID: id}); err != nil {
		return u, err
	}
	return s.Q.GetUser(ctx, u.ID)
}

func (s *Service) storeCeremony(ctx context.Context, kind string, userID *uuid.UUID, data any) (string, error) {
	token, hash := authn.NewToken()
	_, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
		ID: uuid.Must(uuid.NewV7()), TokenHash: hash, Kind: kind, UserID: userID, Data: mustJSON(data), ExpiresAt: s.now().Add(ceremonyTTL),
	})
	return token, err
}

func passkeyExpired() *api.Problem {
	return api.Err(http.StatusBadRequest, "challenge_expired", "The passkey request expired. Try again.")
}

// BeginPasskeyRegistration starts adding a passkey to the signed-in account.
func (s *Service) BeginPasskeyRegistration(ctx context.Context) (PasskeyOptions, error) {
	cu, err := currentUser(ctx)
	if err != nil {
		return PasskeyOptions{}, err
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return PasskeyOptions{}, err
	}
	u, err := s.ensureWebAuthnID(ctx, *cu)
	if err != nil {
		return PasskeyOptions{}, err
	}
	w, err := s.loadWAUser(ctx, s.Q, u)
	if err != nil {
		return PasskeyOptions{}, err
	}
	exclude := make([]protocol.CredentialDescriptor, 0, len(w.creds))
	for _, c := range w.creds {
		exclude = append(exclude, c.Descriptor())
	}
	required := true
	creation, session, err := wa.BeginRegistration(w,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: &required,
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithExclusions(exclude),
	)
	if err != nil {
		return PasskeyOptions{}, err
	}
	uid := u.ID
	token, err := s.storeCeremony(ctx, "webauthn_register", &uid, session)
	if err != nil {
		return PasskeyOptions{}, err
	}
	return PasskeyOptions{Challenge: token, Options: mustJSON(creation.Response)}, nil
}

// FinishPasskeyRegistration verifies the browser's answer and stores the passkey.
func (s *Service) FinishPasskeyRegistration(ctx context.Context, challenge string, credential json.RawMessage, name string) (AccountPasskey, error) {
	cu, err := currentUser(ctx)
	if err != nil {
		return AccountPasskey{}, err
	}
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "webauthn_register"})
	if err != nil || ch.UserID == nil || *ch.UserID != cu.ID {
		return AccountPasskey{}, passkeyExpired()
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	var session webauthn.SessionData
	if err := json.Unmarshal(ch.Data, &session); err != nil {
		return AccountPasskey{}, passkeyExpired()
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return AccountPasskey{}, err
	}
	u, err := s.Q.GetUser(ctx, cu.ID)
	if err != nil {
		return AccountPasskey{}, err
	}
	w, err := s.loadWAUser(ctx, s.Q, u)
	if err != nil {
		return AccountPasskey{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(credential)
	if err != nil {
		return AccountPasskey{}, api.Invalid("body.credential", "Your browser's answer couldn't be read. Try again.")
	}
	cred, err := wa.CreateCredential(w, session, parsed)
	if err != nil {
		s.Log.InfoContext(ctx, "passkey registration rejected", "user", u.ID, "err", err)
		return AccountPasskey{}, api.Err(http.StatusUnprocessableEntity, "passkey_rejected", "The passkey couldn't be verified. Try again, or use another device or security key.")
	}
	attachment := string(cred.Authenticator.Attachment)
	if attachment != "platform" && attachment != "cross-platform" {
		attachment = ""
	}
	provider := providerName(cred.Authenticator.AAGUID)
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultPasskeyName(audit.RequestFrom(ctx).UserAgent, attachment, provider)
	}
	if len([]rune(name)) > 80 {
		name = string([]rune(name)[:80])
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	var pk store.UserPasskey
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		pk, err = q.CreatePasskeyRecord(ctx, store.CreatePasskeyRecordParams{
			ID: uuid.Must(uuid.NewV7()), UserID: u.ID, Name: name, CredentialID: cred.ID, PublicKey: cred.PublicKey,
			AttestationType: cred.AttestationType, Aaguid: cred.Authenticator.AAGUID, SignCount: int64(cred.Authenticator.SignCount),
			Transports: transports, BackupEligible: cred.Flags.BackupEligible, BackupState: cred.Flags.BackupState,
			Credential: mustJSON(cred), Attachment: attachment, Provider: provider,
		})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.passkey_add", TargetType: "passkey", TargetID: pk.ID.String(), TargetName: name,
			Metadata: map[string]any{"attachment": attachment, "provider": provider}})
	})
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return AccountPasskey{}, api.Conflict("This passkey is already registered.")
		}
		return AccountPasskey{}, err
	}
	return passkeyView(pk, nil), nil
}

// BeginPasskeySignIn starts "Sign in with a passkey", which needs no email.
func (s *Service) BeginPasskeySignIn(ctx context.Context) (PasskeyOptions, error) {
	st, err := s.policy(ctx)
	if err != nil {
		return PasskeyOptions{}, err
	}
	if !methodAllowed(st, "passkey") {
		return PasskeyOptions{}, api.Err(http.StatusForbidden, "method_disabled", "Passkey sign-in is turned off on this instance.")
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return PasskeyOptions{}, err
	}
	assertion, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return PasskeyOptions{}, err
	}
	token, err := s.storeCeremony(ctx, "webauthn_login", nil, session)
	if err != nil {
		return PasskeyOptions{}, err
	}
	return PasskeyOptions{Challenge: token, Options: mustJSON(assertion.Response)}, nil
}

func passkeyFailed() *api.Problem {
	return api.Err(http.StatusUnauthorized, "passkey_failed", "That passkey didn't work here. Try again, or sign in with your email and password.")
}

// PasskeySignIn verifies a discoverable passkey and signs the person in.
func (s *Service) PasskeySignIn(ctx context.Context, challenge string, credential json.RawMessage, remember bool) (SignInResult, *http.Cookie, error) {
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "webauthn_login"})
	if err != nil {
		return SignInResult{}, nil, passkeyExpired()
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	var session webauthn.SessionData
	if err := json.Unmarshal(ch.Data, &session); err != nil {
		return SignInResult{}, nil, passkeyExpired()
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return SignInResult{}, nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credential)
	if err != nil {
		return SignInResult{}, nil, api.Invalid("body.credential", "Your browser's answer couldn't be read. Try again.")
	}
	var found *waUser
	handler := func(_, userHandle []byte) (webauthn.User, error) {
		u, err := s.Q.GetUserByWebAuthnID(ctx, userHandle)
		if err != nil {
			return nil, err
		}
		w, err := s.loadWAUser(ctx, s.Q, u)
		if err != nil {
			return nil, err
		}
		found = w
		return w, nil
	}
	_, cred, err := wa.ValidatePasskeyLogin(handler, session, parsed)
	if err != nil {
		ev := audit.Event{Action: "auth.sign_in", Result: audit.Denied, Reason: "passkey verification failed", TargetType: "user", Actor: &audit.Actor{Type: "user", Name: "unknown passkey"},
			Metadata: map[string]any{"method": "passkey"}}
		if found != nil {
			ev.Actor, ev.TargetID, ev.TargetName = actorFor(found.u), found.u.ID.String(), found.u.Email
		}
		_ = s.Audit.Log(ctx, ev)
		return SignInResult{}, nil, passkeyFailed()
	}
	u := found.u
	if u.Status == "deactivated" {
		return SignInResult{}, nil, api.Err(http.StatusForbidden, "account_deactivated", "This account was deactivated. Ask your instance admin to reactivate it; nothing was deleted.").
			WithMeta("deactivatedAt", u.DeactivatedAt)
	}
	// Domain enforcement turns passkeys off too, except for break-glass Owners.
	if p, _ := s.enforcingProvider(ctx, u.Email); p != nil && !u.BreakGlass {
		return SignInResult{}, nil, api.Err(http.StatusForbidden, "sso_required", emailDomain(u.Email)+" signs in with "+p.Name+". Passkey sign-in is turned off for this domain.").
			WithMeta("providerId", p.ID.String(), "provider", p.Name, "preset", p.Preset)
	}
	pk, ok := found.key(cred.ID)
	if !ok {
		return SignInResult{}, nil, passkeyFailed()
	}
	if err := s.Q.UpdatePasskeyAfterUse(ctx, store.UpdatePasskeyAfterUseParams{ID: pk.ID, Credential: mustJSON(cred), SignCount: int64(cred.Authenticator.SignCount), BackupState: cred.Flags.BackupState}); err != nil {
		return SignInResult{}, nil, err
	}
	_ = s.Q.ClearSignInThrottle(ctx, u.Email)
	pid := pk.ID
	cookie, err := s.completeSignIn(ctx, u, "passkey", true, remember, &pid)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u)}, &cookie, nil
}

type twoFactorPasskeyCeremony struct {
	Session   webauthn.SessionData `json:"session"`
	TwoFactor uuid.UUID            `json:"twoFactor"`
}

// BeginTwoFactorPasskey starts confirming a password sign-in with a passkey.
func (s *Service) BeginTwoFactorPasskey(ctx context.Context, challenge string) (PasskeyOptions, error) {
	ch, u, _, err := s.loadTwoFactor(ctx, challenge)
	if err != nil {
		return PasskeyOptions{}, err
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return PasskeyOptions{}, err
	}
	w, err := s.loadWAUser(ctx, s.Q, u)
	if err != nil {
		return PasskeyOptions{}, err
	}
	if len(w.creds) == 0 || len(u.WebauthnID) == 0 {
		return PasskeyOptions{}, api.Conflict("This account has no passkey. Use your authenticator app or a recovery code.")
	}
	assertion, session, err := wa.BeginLogin(w, webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return PasskeyOptions{}, err
	}
	uid := u.ID
	token, err := s.storeCeremony(ctx, "webauthn_2fa", &uid, twoFactorPasskeyCeremony{Session: *session, TwoFactor: ch.ID})
	if err != nil {
		return PasskeyOptions{}, err
	}
	return PasskeyOptions{Challenge: token, Options: mustJSON(assertion.Response)}, nil
}

// TwoFactorPasskey completes a password sign-in with a passkey.
func (s *Service) TwoFactorPasskey(ctx context.Context, challenge, passkeyChallenge string, credential json.RawMessage) (SignInResult, *http.Cookie, error) {
	ch, u, data, err := s.loadTwoFactor(ctx, challenge)
	if err != nil {
		return SignInResult{}, nil, err
	}
	pch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(passkeyChallenge), Kind: "webauthn_2fa"})
	if err != nil || pch.UserID == nil || *pch.UserID != u.ID {
		return SignInResult{}, nil, passkeyExpired()
	}
	_ = s.Q.DeleteChallenge(ctx, pch.ID)
	var cer twoFactorPasskeyCeremony
	if err := json.Unmarshal(pch.Data, &cer); err != nil || cer.TwoFactor != ch.ID {
		return SignInResult{}, nil, passkeyExpired()
	}
	wa, err := s.webAuthn(ctx)
	if err != nil {
		return SignInResult{}, nil, err
	}
	w, err := s.loadWAUser(ctx, s.Q, u)
	if err != nil {
		return SignInResult{}, nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credential)
	if err != nil {
		return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "Your browser's answer couldn't be read.")
	}
	cred, err := wa.ValidateLogin(w, cer.Session, parsed)
	if err != nil {
		return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "That passkey didn't work.")
	}
	pk, ok := w.key(cred.ID)
	if !ok {
		return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "That passkey didn't work.")
	}
	if err := s.Q.UpdatePasskeyAfterUse(ctx, store.UpdatePasskeyAfterUseParams{ID: pk.ID, Credential: mustJSON(cred), SignCount: int64(cred.Authenticator.SignCount), BackupState: cred.Flags.BackupState}); err != nil {
		return SignInResult{}, nil, err
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	pid := pk.ID
	cookie, err := s.completeSignInWith(ctx, u, data.Method+"+passkey", true, data.Remember, &pid, data.Extra)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u)}, &cookie, nil
}

// RenamePasskey changes a passkey's name.
func (s *Service) RenamePasskey(ctx context.Context, id uuid.UUID, name string) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return api.Invalid("body.name", "Give the passkey a name, like “MacBook Pro · Touch ID”.")
	}
	pk, err := s.Q.GetUserPasskey(ctx, store.GetUserPasskeyParams{ID: id, UserID: u.ID})
	if err != nil {
		return api.MapError(err, "Passkey")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RenamePasskey(ctx, store.RenamePasskeyParams{ID: id, UserID: u.ID, Name: name}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.passkey_rename", TargetType: "passkey", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"name": pk.Name}, After: map[string]any{"name": name}})
	})
}

// RemovePasskey deletes a passkey unless it is the last way to sign in, or
// the last second factor while two-factor is required.
func (s *Service) RemovePasskey(ctx context.Context, id uuid.UUID) error {
	u, err := currentUser(ctx)
	if err != nil {
		return err
	}
	pk, err := s.Q.GetUserPasskey(ctx, store.GetUserPasskeyParams{ID: id, UserID: u.ID})
	if err != nil {
		return api.MapError(err, "Passkey")
	}
	m := s.methodsOf(ctx, *u)
	if m.total() <= 1 {
		return api.Err(http.StatusConflict, "last_method", "This passkey is your only way to sign in. Add another passkey or a password first.")
	}
	if m.passkeys == 1 && !m.totp && m.password {
		st, _ := s.policy(ctx)
		if req, _ := s.twoFactorRequired(ctx, *u, st); req {
			return api.Err(http.StatusConflict, "last_second_factor", "Two-factor is required for your account. Set up an authenticator app or add another passkey before removing this one.")
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.DeletePasskey(ctx, store.DeletePasskeyParams{ID: id, UserID: u.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.passkey_remove", TargetType: "passkey", TargetID: id.String(), TargetName: pk.Name})
	})
}

// knownAuthenticators names common passkey providers by AAGUID (from the
// community-maintained passkey AAGUID list).
var knownAuthenticators = map[string]string{
	"fbfc3007-154e-4ecc-8c0b-6e020557d7bd": "iCloud Keychain",
	"dd4ec289-e01d-41c9-bb89-70fa845d4bf2": "iCloud Keychain",
	"ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4": "Google Password Manager",
	"adce0002-35bc-c60a-648b-0b25f1f05503": "Chrome on Mac",
	"08987058-cadc-4b81-b6e1-30de50dcbe96": "Windows Hello",
	"9ddd1817-af5a-4672-a2b9-3e3dd95000a9": "Windows Hello",
	"6028b017-b1d4-4c02-b4b3-afcdafc96bb2": "Windows Hello",
	"bada5566-a7aa-401f-bd96-45619a55120d": "1Password",
	"d548826e-79b4-db40-a3d8-11116f7e8349": "Bitwarden",
	"531126d6-e717-415c-9320-3d9aa6981239": "Dashlane",
	"0ea242b4-43c4-4a1b-8b17-dd6d0b6baec6": "Keeper",
	"b84e4048-15dc-4dd0-8640-f4f60813c8af": "NordPass",
	"53414d53-554e-4700-0000-000000000000": "Samsung Pass",
	"cb69481e-8ff7-4039-93ec-0a2729a154a8": "YubiKey 5",
	"ee882879-721c-4913-9775-3dfcce97072a": "YubiKey 5",
	"fa2b99dc-9e39-4257-8f92-4a30d23c4118": "YubiKey 5",
	"2fc0579f-8113-47ea-b116-bb5a8db9202a": "YubiKey 5",
	"73bb0cd4-e502-49b8-9c6f-b59445bf720b": "YubiKey 5",
	"a4e9fc6d-4cbe-4758-b8ba-37598bb5bbaa": "Security key",
}

func providerName(aaguid []byte) string {
	id, err := uuid.FromBytes(aaguid)
	if err != nil || id == uuid.Nil {
		return ""
	}
	return knownAuthenticators[id.String()]
}

// defaultPasskeyName suggests a name from the device that created the passkey.
func defaultPasskeyName(userAgent, attachment, provider string) string {
	if attachment == "cross-platform" {
		if provider != "" && provider != "Security key" {
			return provider
		}
		return "Security key"
	}
	d := ParseUserAgent(userAgent)
	switch d.Device {
	case "Mac":
		return "Mac · Touch ID"
	case "iPhone":
		return "iPhone · Face ID"
	case "iPad":
		return "iPad"
	case "Windows PC":
		return "Windows Hello"
	case "":
		if provider != "" {
			return provider
		}
		return "Passkey"
	}
	return d.Device
}
