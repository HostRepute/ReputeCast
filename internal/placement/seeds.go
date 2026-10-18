package placement

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/store"
)

// CallbackPath is where Google and Microsoft send the browser back after a
// seed's consent. It's fixed per instance, so one redirect URI is added to
// each sign-in app; the state names the workspace.
const CallbackPath = "/api/v1/placement/oauth/callback"

// Endpoints are the OAuth endpoints of the two sign-in apps.
type Endpoints struct {
	GoogleAuth, GoogleToken string
	// MicrosoftBase is the identity platform ("https://login.microsoftonline.com").
	MicrosoftBase string
}

//nolint:gosec // G101: Google's and Microsoft's public OAuth endpoints, not credentials
var defaultEndpoints = Endpoints{
	GoogleAuth: "https://accounts.google.com/o/oauth2/v2/auth", GoogleToken: "https://oauth2.googleapis.com/token",
	MicrosoftBase: "https://login.microsoftonline.com",
}

func (s *Service) endpoints() Endpoints {
	if s.OAuthEndpoints != nil {
		return *s.OAuthEndpoints
	}
	return defaultEndpoints
}

// GoogleScopes are what a Gmail or Google Workspace seed is connected
// with: IMAP with XOAUTH2 needs https://mail.google.com/ (Google's
// restricted scope; see docs/integrations/seed-mailboxes.md).
var GoogleScopes = []string{"https://mail.google.com/", "openid", "email"}

// seedCredentials are a seed's sealed secret.
type seedCredentials struct {
	Password string `json:"password,omitempty"`
	// OAuth tokens, refreshed as they expire.
	Kind         string    `json:"kind,omitempty"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry"`
	// Demo seeds belong to the demo data: their copies are simulated.
	Demo bool `json:"demo,omitempty"`
}

func credentialsAD(seedID uuid.UUID) string {
	return "placement:seed:" + seedID.String() + ":credentials"
}

// SealCredentials seals a seed's password or tokens (the demo seeds with it).
func SealCredentials(s interface {
	Seal([]byte, string) ([]byte, error)
}, seedID uuid.UUID, password string, demo bool) ([]byte, error) {
	b, err := json.Marshal(seedCredentials{Password: password, Demo: demo}) //nolint:gosec // G117: sealed right away, never returned
	if err != nil {
		return nil, err
	}
	return s.Seal(b, credentialsAD(seedID))
}

func (s *Service) seal(seedID uuid.UUID, c seedCredentials) ([]byte, error) {
	b, err := json.Marshal(c) //nolint:gosec // G117: sealed right away, never returned
	if err != nil {
		return nil, err
	}
	return s.Box.Seal(b, credentialsAD(seedID))
}

func (s *Service) credentials(seed store.PlacementSeed) (seedCredentials, error) {
	var c seedCredentials
	b, err := s.Box.Open(seed.CredentialsEnc, credentialsAD(seed.ID))
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

// oauthConfig is a sign-in app's OAuth configuration for seeds.
func (s *Service) oauthConfig(app MailApp, redirect string) *oauth2.Config {
	e := s.endpoints()
	c := &oauth2.Config{ClientID: app.ClientID, ClientSecret: app.ClientSecret, RedirectURL: redirect}
	switch app.Kind {
	case "google":
		c.Endpoint = oauth2.Endpoint{AuthURL: e.GoogleAuth, TokenURL: e.GoogleToken, AuthStyle: oauth2.AuthStyleInParams}
		c.Scopes = GoogleScopes
	case "microsoft":
		tenant := app.Tenant
		if tenant == "" {
			tenant = "common"
		}
		base := strings.TrimRight(e.MicrosoftBase, "/") + "/" + tenant + "/oauth2/v2.0"
		c.Endpoint = oauth2.Endpoint{AuthURL: base + "/authorize", TokenURL: base + "/token", AuthStyle: oauth2.AuthStyleInParams}
		c.Scopes = seedbox.GraphScopes
	}
	return c
}

func (s *Service) oauthCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, s.httpClient())
}

// savingSource refreshes a seed's token and seals the new one.
type savingSource struct {
	s    *Service
	seed store.PlacementSeed
	c    seedCredentials
	src  oauth2.TokenSource
	last string
}

func (t *savingSource) Token() (*oauth2.Token, error) {
	tok, err := t.src.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != t.last {
		t.last = tok.AccessToken
		t.c.AccessToken, t.c.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			t.c.RefreshToken = tok.RefreshToken
		}
		if sealed, err := t.s.seal(t.seed.ID, t.c); err == nil {
			_ = t.s.Q.PlacementSeedSetCredentials(context.Background(), store.PlacementSeedSetCredentialsParams{ID: t.seed.ID, CredentialsEnc: sealed})
		}
	}
	return tok, nil
}

// boxConfig is how a seed is opened.
func (s *Service) boxConfig(ctx context.Context, seed store.PlacementSeed, c seedCredentials) (seedbox.Config, error) {
	cfg := seedbox.Config{Provider: seed.Provider, Email: seed.Email, Auth: seed.Auth, Host: seed.ImapHost, Port: int(seed.ImapPort),
		Username: seed.Username, Password: c.Password, Dialer: s.Guard.Dialer(30 * time.Second), HTTP: s.httpClient(),
		GraphURL: s.GraphURL, InsecureIMAP: s.InsecureIMAP, Timeout: 30 * time.Second}
	if seed.Auth == seedbox.AuthOAuth && s.IMAPHost != "" {
		cfg.Host, cfg.Port = s.IMAPHost, s.IMAPPort
	}
	if seed.Auth == seedbox.AuthOAuth {
		kind := c.Kind
		if kind == "" {
			kind = oauthKind(seed.Provider)
		}
		var app MailApp
		if s.Apps != nil {
			a, ok, err := s.Apps.MailApp(ctx, kind)
			if err != nil {
				return cfg, err
			}
			if ok {
				app = a
			}
		}
		app.Kind = kind
		conf := s.oauthConfig(app, "")
		tok := &oauth2.Token{AccessToken: c.AccessToken, RefreshToken: c.RefreshToken, Expiry: c.Expiry, TokenType: "Bearer"}
		cfg.Token = &savingSource{s: s, seed: seed, c: c, src: conf.TokenSource(s.oauthCtx(context.WithoutCancel(ctx)), tok), last: c.AccessToken}
	}
	return cfg, nil
}

func oauthKind(provider string) string {
	if seedbox.IsMicrosoft(provider) {
		return "microsoft"
	}
	return "google"
}

// openSeed signs in to a seed (a demo seed answers from the demo data).
func (s *Service) openSeed(ctx context.Context, seed store.PlacementSeed) (seedbox.Box, error) {
	c, err := s.credentials(seed)
	if err != nil {
		return nil, err
	}
	if c.Demo {
		return &demoBox{s: s, seed: seed}, nil
	}
	cfg, err := s.boxConfig(ctx, seed, c)
	if err != nil {
		return nil, err
	}
	return seedbox.Open(ctx, cfg)
}

// PlacementFolder is a folder found in a seed.
type PlacementFolder struct {
	Role  string `json:"role" enum:"inbox,tab,spam"`
	Name  string `json:"name"`
	Label string `json:"label"`
}

// PlacementSeedView is one seed mailbox. Its password or tokens are never returned.
type PlacementSeedView struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Provider     string `json:"provider" enum:"gmail,google_workspace,outlook,microsoft_365,yahoo,aol,icloud,gmx,webde"`
	ProviderName string `json:"providerName"`
	// Company runs the mailbox (Apple); Password is what it calls the
	// IMAP password (App-specific password).
	Company       string            `json:"company"`
	PasswordLabel string            `json:"passwordLabel"`
	Auth          string            `json:"auth" enum:"password,oauth"`
	Status        string            `json:"status" enum:"connected,expired,failed,unchecked"`
	StatusAt      *time.Time        `json:"statusAt,omitempty"`
	CheckedAt     *time.Time        `json:"checkedAt,omitempty"`
	Folders       []PlacementFolder `json:"folders"`
	LastUsedAt    *time.Time        `json:"lastUsedAt,omitempty"`
	LastTestID    string            `json:"lastTestId,omitempty"`
	LastTest      string            `json:"lastTest,omitempty"`
	AddedBy       string            `json:"addedBy"`
	AddedAt       time.Time         `json:"addedAt"`
	// The IMAP server and sign-in, filled in again when reconnecting.
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
}

// PlacementSeedProvider is a mailbox provider in the add dialog.
type PlacementSeedProvider struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Company       string   `json:"company" doc:"Who runs the mailboxes (Apple for iCloud)"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	OAuth         string   `json:"oauth,omitempty" enum:"google,microsoft,"`
	Password      string   `json:"password"`
	UsernameLabel string   `json:"usernameLabel"`
	PasswordHelp  string   `json:"passwordHelp,omitempty"`
	IMAPHelp      string   `json:"imapHelp,omitempty"`
	InboxName     string   `json:"inboxName"`
	SpamName      string   `json:"spamName"`
	Tabs          []string `json:"tabs"`
}

// PlacementSeedList is Placement tests › Seed mailboxes.
type PlacementSeedList struct {
	Seeds       []PlacementSeedView     `json:"seeds"`
	Attention   int                     `json:"attention" doc:"Seeds whose sign-in failed or expired"`
	Connected   int                     `json:"connected"`
	DeleteAfter string                  `json:"deleteAfter" enum:"24h,finished,never"`
	Providers   []PlacementSeedProvider `json:"providers"`
	// OAuth: which of the instance's sign-in apps can connect seeds.
	OAuthGoogle    bool     `json:"oauthGoogle"`
	OAuthMicrosoft bool     `json:"oauthMicrosoft"`
	RedirectURI    string   `json:"redirectUri"`
	CanManage      bool     `json:"canManage"`
	ManageHelpers  []string `json:"manageHelpers"`
	CanRun         bool     `json:"canRun"`
	RunHelpers     []string `json:"runHelpers"`
}

func seedView(r store.PlacementSeed, addedBy string) PlacementSeedView {
	v := PlacementSeedView{ID: r.ID.String(), Email: r.Email, Provider: r.Provider, Auth: r.Auth, Status: r.Status, StatusAt: r.StatusAt,
		CheckedAt: r.CheckedAt, Folders: []PlacementFolder{}, LastUsedAt: r.LastUsedAt, LastTest: r.LastTestLabel, AddedBy: addedBy, AddedAt: r.AddedAt,
		Host: r.ImapHost, Port: int(r.ImapPort), Username: r.Username}
	if p, ok := seedbox.ProviderOf(r.Provider); ok {
		v.ProviderName, v.Company, v.PasswordLabel = p.Name, p.Company, p.Password
	}
	if r.LastTestID != nil {
		v.LastTestID = r.LastTestID.String()
	}
	_ = json.Unmarshal(r.Folders, &v.Folders)
	if v.Folders == nil {
		v.Folders = []PlacementFolder{}
	}
	return v
}

// Seeds lists the seed mailboxes.
func (s *Service) Seeds(ctx context.Context, ws *store.Workspace) (PlacementSeedList, error) {
	p := api.PrincipalFrom(ctx)
	out := PlacementSeedList{Seeds: []PlacementSeedView{}, Providers: []PlacementSeedProvider{},
		CanManage: p.Can(authz.PlacementManage), ManageHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementManage),
		CanRun: p.Can(authz.PlacementRun), RunHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementRun),
		RedirectURI: strings.TrimRight(s.URLs.Base(), "/") + CallbackPath}
	rows, err := s.Q.PlacementSeeds(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		v := seedView(r.PlacementSeed, r.AddedByName)
		out.Seeds = append(out.Seeds, v)
		switch v.Status {
		case "failed", "expired":
			out.Attention++
		case "connected":
			out.Connected++
		}
	}
	if out.DeleteAfter, err = s.Q.PlacementSettings(ctx, ws.ID); err != nil {
		return out, err
	}
	for _, sp := range seedbox.Providers {
		tabs := sp.Tabs
		if tabs == nil {
			tabs = []string{}
		}
		out.Providers = append(out.Providers, PlacementSeedProvider{Key: sp.Key, Name: sp.Name, Company: sp.Company, Host: sp.Host, Port: sp.Port, OAuth: sp.OAuth,
			Password: sp.Password, UsernameLabel: sp.UsernameLabel, PasswordHelp: sp.PasswordHelp, IMAPHelp: sp.IMAPHelp,
			InboxName: sp.InboxName, SpamName: sp.SpamName, Tabs: tabs})
	}
	if s.Apps != nil {
		_, out.OAuthGoogle, _ = s.Apps.MailApp(ctx, "google")
		_, out.OAuthMicrosoft, _ = s.Apps.MailApp(ctx, "microsoft")
	}
	return out, nil
}

// PlacementSeedInput adds or reconnects a seed with an IMAP password.
type PlacementSeedInput struct {
	Provider string `json:"provider" enum:"gmail,google_workspace,outlook,microsoft_365,yahoo,aol,icloud,gmx,webde"`
	Email    string `json:"email" format:"email" maxLength:"254"`
	Host     string `json:"host,omitempty" maxLength:"253"`
	Port     int    `json:"port,omitempty" minimum:"1" maximum:"65535"`
	Username string `json:"username,omitempty" maxLength:"254"`
	Password string `json:"password" minLength:"1" maxLength:"500"`
}

// PlacementSeedCheck is "Check passed · seed added": signed in, and the
// folders found.
type PlacementSeedCheck struct {
	Seed    PlacementSeedView `json:"seed"`
	Folders []PlacementFolder `json:"folders"`
}

// signInProblem words a failed sign-in for the add and reconnect dialogs.
func signInProblem(ctx context.Context, p seedbox.Provider, host string, port int, err error) error {
	l := i18n.LangFrom(ctx)
	switch {
	case netguard.IsBlocked(err):
		return api.Invalid("body.host", mSeedBlocked.Text(l, i18n.Args{"host": host}))
	case errors.Is(err, seedbox.ErrAuthFailed):
		msg := mSeedRefused.Text(l, i18n.Args{"provider": p.Name, "password": strings.ToLower(p.Password)})
		switch {
		case p.Key == seedbox.GMX || p.Key == seedbox.WebDE:
			msg = mSeedRefusedIMAP.Text(l, i18n.Args{"provider": p.Name})
		case seedbox.IsMicrosoft(p.Key):
			msg = mSeedRefusedMicrosoft.Text(l, nil)
		}
		return api.Invalid("body.password", msg).WithMeta("kind", "refused")
	case errors.Is(err, seedbox.ErrUnreachable):
		return api.Invalid("body.host", mSeedUnreachable.Text(l, i18n.Args{"host": host, "port": port})).WithMeta("kind", "unreachable")
	}
	return err
}

func foldersOf(fs []seedbox.Folder) []PlacementFolder {
	out := make([]PlacementFolder, len(fs))
	for i, f := range fs {
		out[i] = PlacementFolder(f)
	}
	return out
}

// imapPassword is the password as the server wants it: app passwords are
// shown in groups ("abcd efgh ijkl mnop"); GMX and Web.de use the
// account's own password, spaces and all.
func imapPassword(p seedbox.Provider, pw string) string {
	if p.Key == seedbox.GMX || p.Key == seedbox.WebDE {
		return pw
	}
	return strings.ReplaceAll(pw, " ", "")
}

// checkSignIn signs in to a seed with a candidate configuration and lists its folders.
func (s *Service) checkSignIn(ctx context.Context, cfg seedbox.Config) ([]PlacementFolder, error) {
	b, err := seedbox.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.Close() }()
	fs, err := b.Folders(ctx)
	if err != nil {
		return nil, err
	}
	return foldersOf(fs), nil
}

// AddSeed checks an IMAP sign-in and adds the seed (audited).
func (s *Service) AddSeed(ctx context.Context, ws *store.Workspace, in PlacementSeedInput) (PlacementSeedCheck, error) {
	l := i18n.LangFrom(ctx)
	p, ok := seedbox.ProviderOf(in.Provider)
	if !ok {
		return PlacementSeedCheck{}, api.Invalid("body.provider", mSeedUnknownProvider.Text(l, nil))
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := s.Q.PlacementSeedByEmail(ctx, store.PlacementSeedByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		return PlacementSeedCheck{}, api.Conflict(mSeedDuplicate.Text(l, i18n.Args{"email": email}))
	}
	if strings.TrimSpace(in.Password) == "" {
		return PlacementSeedCheck{}, api.Invalid("body.password", mSeedPasswordNeeded.Text(l, i18n.Args{"password": strings.ToLower(p.Password)}))
	}
	host, port := strings.TrimSpace(in.Host), in.Port
	if host == "" {
		host = p.Host
	}
	if port == 0 {
		port = p.Port
	}
	user := strings.TrimSpace(in.Username)
	cfg := seedbox.Config{Provider: p.Key, Email: email, Auth: seedbox.AuthPassword, Host: host, Port: port, Username: user,
		Password: imapPassword(p, in.Password), Dialer: s.Guard.Dialer(30 * time.Second), InsecureIMAP: s.InsecureIMAP, Timeout: 30 * time.Second}
	folders, err := s.checkSignIn(ctx, cfg)
	if err != nil {
		return PlacementSeedCheck{}, signInProblem(ctx, p, host, port, err)
	}
	id := uuid.Must(uuid.NewV7())
	sealed, err := s.seal(id, seedCredentials{Password: cfg.Password})
	if err != nil {
		return PlacementSeedCheck{}, err
	}
	fb, _ := json.Marshal(folders)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementSeedInsert(ctx, store.PlacementSeedInsertParams{ID: id, WorkspaceID: ws.ID, Email: email, Provider: p.Key, Auth: seedbox.AuthPassword,
			ImapHost: host, ImapPort: int32(port), Username: user, CredentialsEnc: sealed, Status: "connected", StatusAt: &now, Folders: fb,
			CheckedAt: &now, AddedBy: actor(ctx), AddedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.seed_add", TargetType: "seed_mailbox", TargetID: id.String(),
			TargetName: email, Metadata: map[string]any{"provider": p.Name, "auth": "IMAP password", "folders": folderLabels(folders)}})
	})
	if err != nil {
		return PlacementSeedCheck{}, err
	}
	seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PlacementSeedCheck{}, err
	}
	return PlacementSeedCheck{Seed: seedView(seed, ""), Folders: folders}, nil
}

func folderLabels(fs []PlacementFolder) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Label
	}
	return out
}

// PlacementReconnectInput reconnects a seed with a new password.
type PlacementReconnectInput struct {
	Host     string `json:"host,omitempty" maxLength:"253"`
	Port     int    `json:"port,omitempty" minimum:"1" maximum:"65535"`
	Username string `json:"username,omitempty" maxLength:"254"`
	Password string `json:"password" minLength:"1" maxLength:"500"`
}

// Reconnect checks a new password and saves it (audited).
func (s *Service) Reconnect(ctx context.Context, ws *store.Workspace, id uuid.UUID, in PlacementReconnectInput) (PlacementSeedCheck, error) {
	seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PlacementSeedCheck{}, api.MapError(err, "Seed mailbox")
	}
	p, _ := seedbox.ProviderOf(seed.Provider)
	host, port, user := firstNonEmpty(strings.TrimSpace(in.Host), seed.ImapHost, p.Host), in.Port, firstNonEmpty(strings.TrimSpace(in.Username), seed.Username)
	if port == 0 {
		port = int(seed.ImapPort)
	}
	pw := imapPassword(p, in.Password)
	cfg := seedbox.Config{Provider: p.Key, Email: seed.Email, Auth: seedbox.AuthPassword, Host: host, Port: port, Username: user, Password: pw,
		Dialer: s.Guard.Dialer(30 * time.Second), InsecureIMAP: s.InsecureIMAP, Timeout: 30 * time.Second}
	folders, err := s.checkSignIn(ctx, cfg)
	if err != nil {
		return PlacementSeedCheck{}, signInProblem(ctx, p, host, port, err)
	}
	sealed, err := s.seal(seed.ID, seedCredentials{Password: pw})
	if err != nil {
		return PlacementSeedCheck{}, err
	}
	fb, _ := json.Marshal(folders)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementSeedReconnect(ctx, store.PlacementSeedReconnectParams{ID: seed.ID, Auth: seedbox.AuthPassword, ImapHost: host, ImapPort: int32(port),
			Username: user, CredentialsEnc: sealed, Status: "connected", StatusAt: &now, Folders: fb, CheckedAt: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.seed_reconnect", TargetType: "seed_mailbox",
			TargetID: seed.ID.String(), TargetName: seed.Email, Before: map[string]any{"status": seed.Status}, After: map[string]any{"status": "connected"},
			Metadata: map[string]any{"provider": p.Name, "auth": "IMAP password"}})
	})
	if err != nil {
		return PlacementSeedCheck{}, err
	}
	seed, _ = s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID})
	return PlacementSeedCheck{Seed: seedView(seed, ""), Folders: folders}, nil
}

// RemoveSeed removes a seed and its password or tokens. Copies of past
// tests stay in the mailbox; results keep the seed's row.
func (s *Service) RemoveSeed(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Seed mailbox")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.PlacementSeedDelete(ctx, store.PlacementSeedDeleteParams{ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.seed_remove", TargetType: "seed_mailbox",
			TargetID: id.String(), TargetName: seed.Email, Before: map[string]any{"provider": seed.Provider, "auth": seed.Auth, "status": seed.Status}})
	})
}

// PlacementSettingsInput sets when copies are deleted from the seeds.
type PlacementSettingsInput struct {
	DeleteAfter string `json:"deleteAfter" enum:"24h,finished,never"`
}

// SetSettings saves when copies are deleted (audited).
func (s *Service) SetSettings(ctx context.Context, ws *store.Workspace, in PlacementSettingsInput) (PlacementSettingsInput, error) {
	before, err := s.Q.PlacementSettings(ctx, ws.ID)
	if err != nil {
		return in, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementSettingsSet(ctx, store.PlacementSettingsSetParams{WorkspaceID: ws.ID, DeleteAfter: in.DeleteAfter, UpdatedBy: actor(ctx), UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.settings_update", TargetType: "placement_settings",
			TargetName: "Copies in the seed mailboxes", Before: map[string]any{"deleteAfter": before}, After: map[string]any{"deleteAfter": in.DeleteAfter}})
	})
	return in, err
}

// PlacementOAuthInput starts a seed's OAuth consent.
type PlacementOAuthInput struct {
	Provider string `json:"provider" enum:"gmail,google_workspace,outlook,microsoft_365"`
	// SeedID reconnects an existing seed.
	SeedID string `json:"seedId,omitempty"`
	// Email is the seed's address, as a hint for the account picker.
	Email string `json:"email,omitempty" maxLength:"254"`
}

// PlacementOAuthStart is where to send the browser.
type PlacementOAuthStart struct {
	URL string `json:"url"`
}

// StartOAuth begins Google's or Microsoft's consent for a seed, with PKCE
// and a single-use state bound to the person and the workspace.
func (s *Service) StartOAuth(ctx context.Context, ws *store.Workspace, in PlacementOAuthInput) (PlacementOAuthStart, error) {
	l := i18n.LangFrom(ctx)
	p, ok := seedbox.ProviderOf(in.Provider)
	if !ok || p.OAuth == "" {
		return PlacementOAuthStart{}, api.Invalid("body.provider", mSeedOAuthOnly.Text(l, i18n.Args{"provider": in.Provider}))
	}
	app, ready, err := s.app(ctx, p.OAuth)
	if err != nil {
		return PlacementOAuthStart{}, err
	}
	if !ready {
		return PlacementOAuthStart{}, api.Err(http.StatusConflict, "oauth_not_set_up", mSeedOAuthNotSetUp.Text(l, i18n.Args{"app": appName(p.OAuth)}))
	}
	var seedID *uuid.UUID
	if in.SeedID != "" {
		id, err := uuid.Parse(in.SeedID)
		if err != nil {
			return PlacementOAuthStart{}, api.NotFound("Seed mailbox")
		}
		if _, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID}); err != nil {
			return PlacementOAuthStart{}, api.MapError(err, "Seed mailbox")
		}
		seedID = &id
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	state := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(state))
	verifier := oauth2.GenerateVerifier()
	vb, err := s.Box.Seal([]byte(verifier), "placement:oauth:"+hex.EncodeToString(sum[:]))
	if err != nil {
		return PlacementOAuthStart{}, err
	}
	u := actor(ctx)
	if u == nil {
		return PlacementOAuthStart{}, api.Err(http.StatusForbidden, api.CodeHumansOnly, "Only people connect seed mailboxes.")
	}
	now := s.now()
	_ = s.Q.PlacementOAuthStatesExpire(ctx, now)
	if err := s.Q.PlacementOAuthStateInsert(ctx, store.PlacementOAuthStateInsertParams{StateHash: sum[:], WorkspaceID: ws.ID, UserID: *u,
		Kind: p.OAuth, Provider: p.Key, SeedID: seedID, VerifierEnc: vb, CreatedAt: now, ExpiresAt: now.Add(10 * time.Minute)}); err != nil {
		return PlacementOAuthStart{}, err
	}
	conf := s.oauthConfig(app, s.redirectURI())
	opts := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier)}
	if p.OAuth == "google" {
		opts = append(opts, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	} else {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "select_account"))
	}
	if in.Email != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", strings.TrimSpace(in.Email)))
	}
	return PlacementOAuthStart{URL: conf.AuthCodeURL(state, opts...)}, nil
}

func (s *Service) redirectURI() string { return strings.TrimRight(s.URLs.Base(), "/") + CallbackPath }

func (s *Service) app(ctx context.Context, kind string) (MailApp, bool, error) {
	if s.Apps == nil {
		return MailApp{}, false, nil
	}
	a, ok, err := s.Apps.MailApp(ctx, kind)
	a.Kind = kind
	return a, ok, err
}

func appName(kind string) string {
	if kind == "microsoft" {
		return "Microsoft"
	}
	return "Google"
}

// OAuthCallback finishes a consent: it trades the code for tokens, checks
// the mailbox with them and adds (or reconnects) the seed. It returns the
// app path to send the browser to. The signed-in person must be the one
// who started it and still hold placement:manage in the workspace.
func (s *Service) OAuthCallback(ctx context.Context, user *store.User, self, state, code, oauthErr, oauthErrDesc string) string {
	if user == nil {
		return "/sign-in?next=" + url.QueryEscape(self)
	}
	sum := sha256.Sum256([]byte(state))
	st, err := s.Q.PlacementOAuthStateTake(ctx, sum[:])
	if err != nil || st.ExpiresAt.Before(s.now()) || st.UserID != user.ID {
		return "/?placement_oauth=expired"
	}
	ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
	if err != nil {
		return "/"
	}
	m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: user.ID})
	if err != nil || m.Status != "active" {
		return "/"
	}
	perms := authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)
	page := "/w/" + ws.Slug + "/deliverability/placement/seeds"
	back := func(kv ...string) string {
		q := url.Values{}
		for i := 0; i+1 < len(kv); i += 2 {
			q.Set(kv[i], kv[i+1])
		}
		q.Set("provider", st.Provider)
		return page + "?" + q.Encode()
	}
	if !perms.Has(authz.PlacementManage) {
		return back("oauth", "forbidden")
	}
	if oauthErr != "" {
		if oauthErr == "access_denied" {
			return back("oauth", "cancelled")
		}
		return back("oauth", "failed", "reason", firstNonEmpty(oauthErrDesc, oauthErr))
	}
	vb, err := s.Box.Open(st.VerifierEnc, "placement:oauth:"+hex.EncodeToString(sum[:]))
	if err != nil {
		return back("oauth", "failed", "reason", "the sign-in state couldn't be read")
	}
	app, ready, err := s.app(ctx, st.Kind)
	if err != nil || !ready {
		return back("oauth", "failed", "reason", "the sign-in app isn't set up")
	}
	conf := s.oauthConfig(app, s.redirectURI())
	tok, err := conf.Exchange(s.oauthCtx(ctx), code, oauth2.VerifierOption(string(vb)))
	if err != nil {
		return back("oauth", "failed", "reason", err.Error())
	}
	email := idTokenEmail(tok)
	pctx := api.WithPrincipal(ctx, &authz.Principal{Type: authz.ActorUser, ID: user.ID, Name: user.Name, UserID: user.ID, Email: user.Email,
		WorkspaceID: ws.ID, RoleKey: m.RoleKey, RoleName: m.RoleName, Permissions: perms})
	view, perr := s.saveOAuthSeed(pctx, &ws, st, email, tok)
	if perr != nil {
		return back("oauth", "failed", "reason", perr.Error())
	}
	return back("added", view.ID)
}

// idTokenEmail reads the account's address from the ID token the token
// endpoint returned (over TLS, from the provider itself).
func idTokenEmail(tok *oauth2.Token) string {
	raw, _ := tok.Extra("id_token").(string)
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var c struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	_ = json.Unmarshal(b, &c)
	return strings.ToLower(firstNonEmpty(c.Email, c.PreferredUsername))
}

func (s *Service) saveOAuthSeed(ctx context.Context, ws *store.Workspace, st store.PlacementOauthState, email string, tok *oauth2.Token) (PlacementSeedView, error) {
	l := i18n.LangFrom(ctx)
	p, _ := seedbox.ProviderOf(st.Provider)
	var existing *store.PlacementSeed
	if st.SeedID != nil {
		if seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: *st.SeedID, WorkspaceID: ws.ID}); err == nil {
			existing = &seed
			if email != "" && email != seed.Email {
				return PlacementSeedView{}, errors.New(mSeedOAuthWrong.Text(l, i18n.Args{"app": appName(st.Kind), "email": email, "expected": seed.Email}))
			}
			email = seed.Email
		}
	}
	if email == "" {
		return PlacementSeedView{}, errors.New(mSeedOAuthFailed.Text(l, i18n.Args{"app": appName(st.Kind), "reason": "no email address came with the sign-in"}))
	}
	if existing == nil {
		if _, err := s.Q.PlacementSeedByEmail(ctx, store.PlacementSeedByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
			return PlacementSeedView{}, errors.New(mSeedDuplicate.Text(l, i18n.Args{"email": email}))
		}
	}
	id := uuid.Must(uuid.NewV7())
	if existing != nil {
		id = existing.ID
	}
	c := seedCredentials{Kind: st.Kind, AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, Expiry: tok.Expiry}
	seed := store.PlacementSeed{ID: id, WorkspaceID: ws.ID, Email: email, Provider: p.Key, Auth: seedbox.AuthOAuth, ImapHost: p.Host, ImapPort: int32(p.Port)}
	cfg, err := s.boxConfig(ctx, seed, c)
	if err != nil {
		return PlacementSeedView{}, err
	}
	folders, err := s.checkSignIn(ctx, cfg)
	if err != nil {
		return PlacementSeedView{}, err
	}
	// The check may have refreshed the token: seal the newest one.
	if ss, ok := cfg.Token.(*savingSource); ok {
		c = ss.c
	}
	sealed, err := s.seal(id, c)
	if err != nil {
		return PlacementSeedView{}, err
	}
	fb, _ := json.Marshal(folders)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		action := "placement.seed_add"
		if existing != nil {
			action = "placement.seed_reconnect"
			if err := q.PlacementSeedReconnect(ctx, store.PlacementSeedReconnectParams{ID: id, Auth: seedbox.AuthOAuth, ImapHost: p.Host, ImapPort: int32(p.Port),
				CredentialsEnc: sealed, Status: "connected", StatusAt: &now, Folders: fb, CheckedAt: &now}); err != nil {
				return err
			}
		} else if err := q.PlacementSeedInsert(ctx, store.PlacementSeedInsertParams{ID: id, WorkspaceID: ws.ID, Email: email, Provider: p.Key, Auth: seedbox.AuthOAuth,
			ImapHost: p.Host, ImapPort: int32(p.Port), CredentialsEnc: sealed, Status: "connected", StatusAt: &now, Folders: fb, CheckedAt: &now,
			AddedBy: actor(ctx), AddedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "seed_mailbox", TargetID: id.String(), TargetName: email,
			Metadata: map[string]any{"provider": p.Name, "auth": "OAuth · " + appName(st.Kind), "folders": folderLabels(folders)}})
	})
	if err != nil {
		return PlacementSeedView{}, err
	}
	row, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PlacementSeedView{}, err
	}
	return seedView(row, ""), nil
}

// seedFailed records a seed whose sign-in stopped working during a test
// and tells members with placement:manage once.
func (s *Service) seedFailed(ctx context.Context, ws *store.Workspace, seed store.PlacementSeed, err error) {
	status := "failed"
	if errors.Is(err, seedbox.ErrAuthExpired) {
		status = "expired"
	}
	at := s.now()
	if e := s.Q.PlacementSeedStatus(ctx, store.PlacementSeedStatusParams{ID: seed.ID, Status: status, StatusDetail: clip(err.Error(), 400), At: &at}); e != nil {
		return
	}
	if seed.Status != status && s.Notify != nil {
		seed.Status = status
		s.Notify.SeedFailed(ctx, ws, seed)
	}
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
